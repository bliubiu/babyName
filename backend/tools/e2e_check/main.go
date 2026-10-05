// 命令 e2e_check 是起名链路的端到端巡检工具（纯 Go 实现，仅依赖标准库）。
//
// 覆盖八项检查：
//  1. 多场景耗时（双名/单名/复姓/经典来源/避讳/笔画区间/显式五行）
//  2. 推荐用字质量（一/二/三级字分布、门禁字命中、无命名依据字占比）
//  3. 参数生效性（避讳长辈、经典来源偏好）
//  4. 同步与异步一致性（/names/generate 与 /names/generate/async + /names/task/:id）
//  5. 同一请求可复现性
//  6. 并发压测（中位/p95/最大耗时与吞吐）
//  7. 测名 /names/evaluate（与生成同源同分）
//  8. 探索模式 /names/generate/explore（换一批与上一轮零交集）
//
// 路由口径（2026.09.18.1 起）：/names/generate/analysis 已下线，
// /names/generate 是唯一公开生成入口，本工具不再访问已下线路由。
//
// 前置：先启动服务（cd backend && ./namer-server.exe）。
//
// 用法（在 backend 目录下）：
//
//	go run ./tools/e2e_check                        # 默认 127.0.0.1:8080
//	go run ./tools/e2e_check -host localhost:8080   # 注意 Windows 上 localhost 会先试 ::1，可能多出约 2s
//	go run ./tools/e2e_check -concurrency 32 -strict
//
// 本工具只读：仅发起 HTTP 请求并读取 data 目录下的字库，不写入任何项目数据。
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// 现行路由常量：analysis 路由已于 2026.09.18.1 下线，巡检一律走 /names/generate。
const (
	pathGenerate = "/api/v1/names/generate"
	pathAsync    = "/api/v1/names/generate/async"
	pathExplore  = "/api/v1/names/generate/explore"
	pathEvaluate = "/api/v1/names/evaluate"
)

// 默认请求体：一个固定的男宝出生时间，便于跨轮次横向对比。
func baseBody() map[string]any {
	return map[string]any{
		"surname": "张", "gender": "male",
		"birth_year": 2024, "birth_month": 5, "birth_day": 20,
		"birth_hour": 10, "birth_minute": 0,
	}
}

func withOver(over map[string]any) map[string]any {
	b := baseBody()
	for k, v := range over {
		b[k] = v
	}
	return b
}

// ---------- 本地字库（只读） ----------

// charData 从 namer.json / naming_quality.json 等读出的字级属性索引。
type charData struct {
	level   map[string]int  // 通用规范汉字表等级（1/2/3；0=表外补充字）
	pos     map[string]int  // 人工寓意评分 positiveScore（0=无信号）
	corpus  map[string]bool // 95.7 万条真实人名语料中出现过的字（name_frequency.json）
	curated map[string]bool // 策展好名中出现过的字（curated_names.json）
	gate    map[string]bool
}

func (c *charData) levelOf(ch string) int { return c.level[ch] }
func (c *charData) posOf(ch string) int   { return c.pos[ch] }
func (c *charData) inGate(ch string) bool { return c.gate[ch] }

// hasEvidence 该字是否具备「命名依据」——与 fate.hasNamingEvidence 同一口径：
// 人工寓意评分 / 策展分类字 / 真实人名语料证据，三者任一。
//
// 用这个口径而非「positiveScore>0」判榜：positiveScore 只覆盖少量汉字，
// 大量正常名字用字（如 珀/赟/茗/诗）本就没有寓意评分却完全可用；
// 真正该为 0 的是「三无字」——那才是荒谬字霸榜的成因。
func (c *charData) hasEvidence(ch string) bool {
	return c.pos[ch] > 0 || c.curated[ch] || c.corpus[ch]
}

// resolveDataDir 依次尝试候选路径，返回第一个含 namer.json 的目录。
//
// 这样无论从 backend 还是 backend/tools/e2e_check 作为工作目录启动都能工作。
func resolveDataDir(preferred string) string {
	candidates := []string{preferred, "data", "./data", "../data", "../../data"}
	for _, dir := range candidates {
		if dir == "" {
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, "namer.json")); err == nil {
			return dir
		}
	}
	return preferred
}

func loadCharData(dataDir string) (*charData, error) {
	cd := &charData{
		level:   map[string]int{},
		pos:     map[string]int{},
		corpus:  map[string]bool{},
		curated: map[string]bool{},
		gate:    map[string]bool{},
	}

	raw, err := os.ReadFile(filepath.Join(dataDir, "namer.json"))
	if err != nil {
		return nil, fmt.Errorf("读取 namer.json 失败: %w", err)
	}
	var namer struct {
		Chars []struct {
			Char          string `json:"char"`
			Level         int    `json:"level"`
			PositiveScore int    `json:"positiveScore"`
		} `json:"chars"`
	}
	if err := json.Unmarshal(raw, &namer); err != nil {
		return nil, fmt.Errorf("解析 namer.json 失败: %w", err)
	}
	for _, c := range namer.Chars {
		cd.level[c.Char] = c.Level
		cd.pos[c.Char] = c.PositiveScore
	}

	// 人名语料字集（缺失不阻断，仅失去该维度判据）
	if raw, err := os.ReadFile(filepath.Join(dataDir, "name_frequency.json")); err == nil {
		var freq struct {
			Frequency []struct {
				Char string `json:"char"`
			} `json:"frequency"`
		}
		if err := json.Unmarshal(raw, &freq); err == nil {
			for _, e := range freq.Frequency {
				cd.corpus[e.Char] = true
			}
		}
	}

	// 策展好名字集
	if raw, err := os.ReadFile(filepath.Join(dataDir, "curated_names.json")); err == nil {
		var entries []struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(raw, &entries); err == nil {
			for _, e := range entries {
				for _, ch := range e.Name {
					if ch >= 0x4E00 && ch <= 0x9FFF {
						cd.curated[string(ch)] = true
					}
				}
			}
		}
	}

	// 门禁表缺失不阻断（与引擎降级行为一致）
	if gateRaw, err := os.ReadFile(filepath.Join(dataDir, "naming_quality.json")); err == nil {
		var gate []string
		if err := json.Unmarshal(gateRaw, &gate); err != nil {
			return nil, fmt.Errorf("解析 naming_quality.json 失败: %w", err)
		}
		for _, ch := range gate {
			cd.gate[ch] = true
		}
	}
	return cd, nil
}

// ---------- HTTP ----------

// result 一次 API 调用的原始结果。
type result struct {
	raw      []byte
	httpCode int
	dur      time.Duration
	err      error
}

// view 对一次调用的解析视图（键集与类型都保留，便于做结构对比）。
type view struct {
	data  map[string]json.RawMessage
	names []map[string]json.RawMessage
	bazi  map[string]json.RawMessage
	size  float64 // 响应体 KB
}

func (v *view) givenNames(limit int) []string {
	out := make([]string, 0, limit)
	for _, n := range v.names {
		if limit > 0 && len(out) >= limit {
			break
		}
		out = append(out, strField(n, "given_name"))
	}
	return out
}

func (v *view) fullNames(limit int) []string {
	out := make([]string, 0, limit)
	for _, n := range v.names {
		if limit > 0 && len(out) >= limit {
			break
		}
		out = append(out, strField(n, "full_name"))
	}
	return out
}

// xiyongshen 兼容「字符串数组」与「字符串」两种序列化形态。
func (v *view) xiyongshen() []string {
	return stringsFromRaw(v.bazi["xiyongshen"])
}

func (v *view) baziKeys() []string { return sortedKeys(v.bazi) }

func (v *view) nameKeys() []string {
	if len(v.names) == 0 {
		return nil
	}
	return sortedKeys(v.names[0])
}

func strField(m map[string]json.RawMessage, key string) string {
	raw, ok := m[key]
	if !ok {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return ""
	}
	return s
}

func stringsFromRaw(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var arr []string
	if err := json.Unmarshal(raw, &arr); err == nil {
		return arr
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil && s != "" {
		return []string{s}
	}
	return nil
}

func sortedKeys(m map[string]json.RawMessage) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

type client struct {
	base string
	hc   *http.Client
}

func newClient(base string, timeout time.Duration) *client {
	return &client{base: base, hc: &http.Client{Timeout: timeout}}
}

func (c *client) call(path string, body map[string]any) result {
	payload, err := json.Marshal(body)
	if err != nil {
		return result{err: fmt.Errorf("构造请求体失败: %w", err)}
	}
	t0 := time.Now()
	resp, err := c.hc.Post(c.base+path, "application/json", bytes.NewReader(payload))
	if err != nil {
		return result{err: err, dur: time.Since(t0)}
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	return result{raw: raw, httpCode: resp.StatusCode, dur: time.Since(t0), err: err}
}

// get 发起 GET 请求（轮询异步任务状态用）。
func (c *client) get(path string) result {
	t0 := time.Now()
	resp, err := c.hc.Get(c.base + path)
	if err != nil {
		return result{err: err, dur: time.Since(t0)}
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	return result{raw: raw, httpCode: resp.StatusCode, dur: time.Since(t0), err: err}
}

// strAt 从响应体顶层 data 中取字符串字段（如 generation_id）。
func strAt(v *view, key string) string { return strField(v.data, key) }

// parse 解析调用结果；任何结构问题都降级为空视图而不是报错，
// 以便巡检继续跑完并如实呈现「哪个字段缺失」。
func (r result) parse() *view { return r.parseAt("") }

// parseAt 在 parse 的基础上再下沉一层：异步任务把生成结果放在 data.result 里，
// 传 field="result" 即可用同一套视图逻辑解析。field 为空时与 parse 等价。
func (r result) parseAt(field string) *view {
	v := &view{size: float64(len(r.raw)) / 1024}
	if len(r.raw) == 0 {
		return v
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(r.raw, &root); err != nil {
		return v
	}
	if err := json.Unmarshal(root["data"], &v.data); err != nil {
		// data 缺失时退化为顶层（兼容部分接口直接返回业务对象）
		var flat map[string]json.RawMessage
		if err := json.Unmarshal(r.raw, &flat); err == nil {
			v.data = flat
		}
	}
	if field != "" {
		var nested map[string]json.RawMessage
		if err := json.Unmarshal(v.data[field], &nested); err == nil {
			v.data = nested
		}
	}
	_ = json.Unmarshal(v.data["names"], &v.names)
	_ = json.Unmarshal(v.data["bazi"], &v.bazi)
	return v
}

// numAt 从视图的 data 中取数字字段（评分、百分比等），缺失返回 NaN。
func numAt(v *view, key string) float64 {
	raw, ok := v.data[key]
	if !ok {
		return math.NaN()
	}
	var f float64
	if err := json.Unmarshal(raw, &f); err != nil {
		return math.NaN()
	}
	return f
}

// ---------- 巡检状态 ----------

type checker struct {
	total  int
	issues []string
}

func (c *checker) ok(format string, args ...any) {
	c.total++
	fmt.Printf("  ✓ %s\n", fmt.Sprintf(format, args...))
}

func (c *checker) bad(format string, args ...any) {
	c.total++
	msg := fmt.Sprintf(format, args...)
	c.issues = append(c.issues, msg)
	fmt.Printf("  ★ %s\n", msg)
}

// ---------- 异步任务 ----------

// asyncOutcome 一次异步任务的终态观察结果。
type asyncOutcome struct {
	view     *view         // 终态生成结果（解析 data.result）
	httpCode int           // 提交请求的 HTTP 状态
	dur      time.Duration // 从提交到拿到终态的总耗时
	stages   string        // 轮询过程中观察到的进度阶段序列
}

// runAsync 提交异步任务并轮询到终态。
//
// 异步链路的价值在于「提交即返回 + 可查进度」，因此除了终态结果，
// 这里还记录轮询途中观察到的阶段序列——若阶段一直不变或直接从 pending
// 跳到 success，前端进度条就形同虚设。
func runAsync(c *client, path string, body map[string]any, timeout time.Duration) *asyncOutcome {
	sub := c.call(path, body)
	if sub.err != nil {
		fmt.Printf("  ★ 异步提交失败: %v\n", sub.err)
		return nil
	}
	if sub.httpCode != 200 {
		fmt.Printf("  ★ 异步提交返回非 200: HTTP=%d %s\n", sub.httpCode, string(sub.raw))
		return nil
	}
	taskID := strAt(sub.parse(), "task_id")
	if taskID == "" {
		fmt.Printf("  ★ 异步提交响应缺少 task_id: %s\n", string(sub.raw))
		return nil
	}

	t0 := time.Now()
	deadline := time.Now().Add(timeout)
	var stages []string
	seen := map[string]bool{}
	for time.Now().Before(deadline) {
		r := c.get("/api/v1/names/task/" + taskID)
		v := r.parse()
		status := strField(v.data, "status")
		mark := fmt.Sprintf("%s:%s@%.0f%%", status, strField(v.data, "stage"), numAt(v, "percent"))
		if !seen[mark] {
			seen[mark] = true
			stages = append(stages, mark)
		}
		switch status {
		case "success":
			return &asyncOutcome{view: r.parseAt("result"), httpCode: sub.httpCode,
				dur: time.Since(t0), stages: strings.Join(stages, " → ")}
		case "failed":
			fmt.Printf("  ★ 异步任务失败: %s\n", strField(v.data, "error"))
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	fmt.Printf("  ★ 异步任务在 %.0fs 内未到达终态\n", timeout.Seconds())
	return nil
}

// exploreStoreCapacity 与 services.exploreStoreCapacity 对齐：
// 探索会话缓存是进程内 FIFO，容量固定，超过就淘汰最旧的会话。
const exploreStoreCapacity = 32

// clip 截断字符串，避免把整个响应体打进巡检日志。
func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// fmtNum 格式化数字，NaN 显示为空占位，避免日志里出现 "NaN%"。
func fmtNum(f float64) string {
	if math.IsNaN(f) {
		return "-"
	}
	return fmt.Sprintf("%.0f", f)
}

// ---------- 各段检查 ----------

func section(title string) {
	fmt.Println()
	fmt.Println(strings.Repeat("=", 96))
	fmt.Println(title)
	fmt.Println(strings.Repeat("=", 96))
}

// scanQuality 统计 Top-N 用字的等级分布 / 门禁命中 / 无正分信号字。
func scanQuality(names *view, limit int, cd *charData) (lv map[int]int, gateHits, noPos []string) {
	lv = map[int]int{1: 0, 2: 0, 3: 0}
	for i, n := range names.names {
		if i >= limit {
			break
		}
		full := strField(n, "full_name")
		for _, ch := range strField(n, "given_name") {
			lv[cd.levelOf(string(ch))]++
			if cd.inGate(string(ch)) {
				gateHits = append(gateHits, fmt.Sprintf("%s(%s)", full, string(ch)))
			}
			if cd.posOf(string(ch)) == 0 {
				noPos = append(noPos, string(ch))
			}
		}
	}
	sort.Strings(noPos)
	noPos = dedup(noPos)
	return lv, gateHits, noPos
}

func dedup(in []string) []string {
	if len(in) == 0 {
		return in
	}
	out := in[:1]
	for _, s := range in[1:] {
		if s != out[len(out)-1] {
			out = append(out, s)
		}
	}
	return out
}

// optionalSuffix 把命中字列表去重排序后拼成「（命中：xx）」后缀，为空返回空串。
func optionalSuffix(hits []string) string {
	if len(hits) == 0 {
		return ""
	}
	s := append([]string(nil), hits...)
	sort.Strings(s)
	return "（命中：" + strings.Join(dedup(s), "") + "）"
}

func median(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	n := len(s)
	if n%2 == 1 {
		return s[n/2]
	}
	return (s[n/2-1] + s[n/2]) / 2
}

func percentile(xs []float64, p float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	idx := int(math.Ceil(p*float64(len(s)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(s) {
		idx = len(s) - 1
	}
	return s[idx]
}

// ---------- main ----------

func main() {
	host := flag.String("host", "127.0.0.1:8080", "服务地址 host:port")
	conc := flag.Int("concurrency", 16, "并发压测并发数")
	rounds := flag.Int("rounds", 2, "并发压测轮数")
	dataDirFlag := flag.String("data", "data", "数据目录（内含 namer.json / naming_quality.json）")
	timeout := flag.Duration("timeout", 180*time.Second, "单请求超时")
	repeat := flag.Int("repeat", 6, "可复现性检查的重复请求次数")
	strict := flag.Bool("strict", false, "发现异常项时以非 0 退出码结束（便于接入 CI）")
	flag.Parse()

	dataDir := resolveDataDir(*dataDirFlag)
	cd, err := loadCharData(dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载本地字库失败: %v\n", err)
		os.Exit(2)
	}
	base := "http://" + *host
	c := newClient(base, *timeout)
	ck := &checker{}

	fmt.Println(strings.Repeat("=", 96))
	fmt.Printf("起名链路 E2E 巡检  ->  %s\n", base)
	fmt.Printf("本地字库: %s（%d 字有等级、%d 字有正分信号、门禁表 %d 字）\n",
		dataDir, len(cd.level), countPositive(cd.pos), len(cd.gate))
	fmt.Println(strings.Repeat("=", 96))

	// ---------- 0. 预热 ----------
	if r := c.call(pathGenerate, baseBody()); r.err != nil {
		fmt.Printf("预热失败，服务未就绪: %v\n", r.err)
		os.Exit(2)
	} else {
		fmt.Printf("预热: HTTP=%d 耗时=%.3fs\n", r.httpCode, r.dur.Seconds())
	}

	// ---------- 1. 场景计时 ----------
	type scenario struct {
		label string
		path  string
		body  map[string]any
	}
	scenarios := []scenario{
		{"双名 基线", pathGenerate, withOver(nil)},
		{"单名", pathGenerate, withOver(map[string]any{"name_length": 1})},
		{"女宝 李", pathGenerate, withOver(map[string]any{"gender": "female", "surname": "李"})},
		{"经典来源=论语", pathGenerate, withOver(map[string]any{"source_classic": "论语"})},
		{"显式五行=水", pathGenerate, withOver(map[string]any{"wuxing_match": []string{"水"}})},
		{"避讳 张伟/王秀英", pathGenerate, withOver(map[string]any{"avoid_elder_names": []string{"张伟", "王秀英"}})},
		{"复姓 欧阳", pathGenerate, withOver(map[string]any{"surname": "欧阳"})},
		{"笔画 5-20", pathGenerate, withOver(map[string]any{"min_strokes": 5, "max_strokes": 20})},
	}

	section("1. 场景耗时")
	fmt.Printf("%-28s %5s %9s %7s %9s %s\n", "场景", "HTTP", "耗时(s)", "名字数", "响应KB", "喜用神")
	fmt.Println(strings.Repeat("-", 96))
	views := map[string]*view{}
	durs := map[string]float64{}
	var slowest string
	var slowestDur time.Duration
	for _, sc := range scenarios {
		r := c.call(sc.path, sc.body)
		v := r.parse()
		views[sc.label] = v
		durs[sc.label] = r.dur.Seconds()
		xi := "-"
		if len(v.xiyongshen()) > 0 {
			xi = "[" + strings.Join(v.xiyongshen(), " ") + "]"
		}
		status := fmt.Sprintf("%d", r.httpCode)
		if r.err != nil {
			status = "ERR"
			ck.bad("场景 %q 请求失败: %v", sc.label, r.err)
		} else if r.httpCode != 200 {
			ck.bad("场景 %q 返回非 200: HTTP=%d", sc.label, r.httpCode)
		}
		if len(v.names) == 0 && sc.label != "/analysis 单名" {
			// 单名同样应该有名；这里统一要求非空
			ck.bad("场景 %q 未返回任何名字", sc.label)
		}
		if r.dur > slowestDur {
			slowest, slowestDur = sc.label, r.dur
		}
		fmt.Printf("%-28s %5s %9.3f %7d %9.1f %s\n",
			sc.label, status, r.dur.Seconds(), len(v.names), v.size, xi)
		time.Sleep(120 * time.Millisecond)
	}
	fmt.Printf("最慢场景: %s  %.3fs\n", slowest, slowestDur.Seconds())

	// ---------- 2. 用字质量 ----------
	section("2. 推荐用字质量")
	// 实测结论：单名 Top-N 用字一直正常，双名是历史问题区，两者都查。
	for _, label := range []string{"双名 基线", "单名"} {
		v := views[label]
		if v == nil || len(v.names) == 0 {
			continue
		}
		lv, gateHits, noPos := scanQuality(v, 50, cd)
		tot := lv[1] + lv[2] + lv[3]
		fmt.Printf("%-24s 用字 %d：一级=%d(%.0f%%) 二级=%d(%.0f%%) 三级/表外=%d(%.0f%%)\n",
			label, tot,
			lv[1], 100*float64(lv[1])/float64(max(tot, 1)),
			lv[2], 100*float64(lv[2])/float64(max(tot, 1)),
			lv[3], 100*float64(lv[3])/float64(max(tot, 1)))
		fmt.Printf("    Top10: %s\n", strings.Join(v.fullNames(10), " "))
		if len(noPos) > 0 {
			fmt.Printf("    ⚠ 无正分信号(positiveScore=0)的用字 %d 个: %s\n", len(noPos), strings.Join(noPos, ""))
		}
		if len(gateHits) > 0 {
			ck.bad("%s 推荐榜命中质量门禁字 %d 处: %s", label, len(gateHits), strings.Join(gateHits[:min(len(gateHits), 8)], "、"))
		} else {
			ck.ok("%s 推荐榜未命中质量门禁字", label)
		}
		if lv[3] > 0 {
			ck.bad("%s 推荐榜混入三级/表外生僻字 %d 个（AGENTS.md 要求谨慎使用三级字表）", label, lv[3])
		}

		// 双名特有的质量缺陷探测器：
		// 评分体系对「无命名依据的字」（既无寓意评分、非策展、也从未出现在 95.7 万条
		// 真实人名语料中）没有任何负反馈，它们与优质字完全同分。因此把「含无依据字的
		// 比例」作为双名榜的质量水位指标——这正是荒谬字霸榜的量化形态。
		if label == "双名 基线" {
			hit, tot := 0, 0
			var offenders []string
			for i, n := range v.names {
				if i >= 50 {
					break
				}
				tot++
				for _, ch := range strField(n, "given_name") {
					if !cd.hasEvidence(string(ch)) {
						hit++
						offenders = append(offenders, string(ch))
						break
					}
				}
			}
			if tot > 0 {
				ratio := float64(hit) / float64(tot)
				fmt.Printf("    含无命名依据字的比例: %d/%d（%.0f%%）%s\n", hit, tot, 100*ratio,
					optionalSuffix(offenders))
				if ratio > 0.5 {
					ck.bad("双名榜 %d/%d 含无命名依据字（无寓意评分/非策展/不在人名语料）：评分无法区分优劣，榜单被稀释", hit, tot)
				} else {
					ck.ok("双名榜无命名依据字比例 %.0f%%（阈值 50%%）", 100*ratio)
				}
			}
		}
	}

	// ---------- 3. 参数生效性 ----------
	section("3. 参数生效性")
	avoid := views["避讳 张伟/王秀英"]
	if avoid != nil {
		var bad []string
		for _, n := range avoid.names {
			for _, ch := range strField(n, "given_name") {
				if strings.ContainsRune("伟秀英", ch) {
					bad = append(bad, fmt.Sprintf("%s(%s)", strField(n, "full_name"), string(ch)))
				}
			}
		}
		if len(bad) > 0 {
			ck.bad("避讳长辈(张伟/王秀英) 命中避讳字: %s", strings.Join(bad, "、"))
		} else {
			ck.ok("避讳长辈(张伟/王秀英) Top%d 无避讳字", len(avoid.names))
		}
	}
	if b, l := views["双名 基线"], views["经典来源=论语"]; b != nil && l != nil {
		inter := intersectCount(b.givenNames(10), l.givenNames(10))
		inter50 := intersectCount(b.givenNames(50), l.givenNames(50))
		fmt.Printf("  经典来源=论语 与基线 Top10 交集: %d/10（越小说明来源偏好影响越大）；Top50 交集 %d/50\n", inter, inter50)
		if inter >= 10 && inter50 >= 50 {
			ck.bad("切换经典来源后 Top 榜完全无变化，source_classic 疑似未生效")
		} else {
			ck.ok("source_classic 生效（Top10 差异 %d 个）", 10-inter)
		}
	}

	// ---------- 4. 同步 vs 异步一致性 ----------
	section("4. 同步 vs 异步一致性（/names/generate vs /names/generate/async + /names/task/:id）")
	baseline := views["双名 基线"]
	if async := runAsync(c, pathAsync, baseBody(), *timeout); async != nil {
		fmt.Printf("  异步: HTTP=%d 轮询耗时=%.3fs 名字数=%d 响应KB=%.1f\n",
			async.httpCode, async.dur.Seconds(), len(async.view.names), async.view.size)
		fmt.Printf("  进度条: %s\n", async.stages)
		if len(async.view.names) == 0 {
			ck.bad("异步任务成功但未返回任何名字")
		} else {
			ck.ok("异步任务返回 %d 个名字", len(async.view.names))
		}
		if baseline != nil && len(baseline.names) > 0 {
			xs, xa := baseline.xiyongshen(), async.view.xiyongshen()
			fmt.Printf("  喜用神: 同步=%v  异步=%v\n", xs, xa)
			if !equalStrings(xs, xa) {
				ck.bad("同步/异步喜用神不一致：%v vs %v", xs, xa)
			} else {
				ck.ok("同步/异步喜用神一致")
			}
			// 异步与同步走同一套服务层逻辑，Top 榜应当一致；
			// 不一致说明任务侧参数装配或评分链路存在偏差。
			inter := intersectCount(baseline.givenNames(10), async.view.givenNames(10))
			fmt.Printf("  前10名交集: %d/10\n", inter)
			if inter < 10 {
				ck.bad("异步与同步 Top10 不一致，任务侧参数装配或评分链路存在偏差")
			} else {
				ck.ok("异步与同步 Top10 完全一致")
			}
			if d := diffKeys(baseline.baziKeys(), async.view.baziKeys()); len(d) > 0 {
				ck.bad("同步/异步 bazi 字段集差异: %s", strings.Join(d, "、"))
			} else {
				ck.ok("同步/异步 bazi 字段集一致（%d 个）", len(baseline.baziKeys()))
			}
		}
	} else {
		ck.bad("异步任务未到达终态，异步链路不可用")
	}

	// ---------- 5. 可复现性 ----------
	section(fmt.Sprintf("5. 同一请求可复现性（连发 %d 次）", *repeat))
	seen := map[string]int{}
	for i := 0; i < *repeat; i++ {
		r := c.call(pathGenerate, baseBody())
		key := strings.Join(r.parse().givenNames(10), " ")
		seen[key]++
	}
	fmt.Printf("  不同 Top10 结果数: %d / %d\n", len(seen), *repeat)
	if len(seen) > 1 {
		ck.bad("同一请求的 Top10 在进程内不稳定（%d 种结果），存在与请求无关的随机性", len(seen))
		for k, n := range seen {
			fmt.Printf("      x%d  %s\n", n, k)
		}
	} else {
		ck.ok("同一请求的 Top10 在进程内完全一致")
	}

	// ---------- 6. 并发压测 ----------
	section(fmt.Sprintf("6. 并发压测（%d 并发 × %d 轮）", *conc, *rounds))
	surnames := []string{"张", "王", "李", "赵", "陈", "刘", "杨", "黄"}
	bodies := make([]map[string]any, 0, len(surnames)*4)
	for i := 0; i < 4; i++ {
		for _, s := range surnames {
			bodies = append(bodies, withOver(map[string]any{"surname": s}))
		}
	}
	var allOK []float64
	var allTotal int
	var totalWall float64
	for round := 0; round < *rounds; round++ {
		codes := map[string]int{}
		durs := make([]float64, *conc)
		var wg sync.WaitGroup
		var mu sync.Mutex
		t0 := time.Now()
		for i := 0; i < *conc; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				r := c.call(pathGenerate, bodies[i%len(bodies)])
				mu.Lock()
				defer mu.Unlock()
				codes[fmt.Sprintf("%d", r.httpCode)]++
				durs[i] = r.dur.Seconds()
			}(i)
		}
		wg.Wait()
		wall := time.Since(t0).Seconds()
		totalWall += wall
		okDurs := filterPositive(durs)
		allOK = append(allOK, okDurs...)
		allTotal += *conc
		fmt.Printf("  轮次%d: wall=%.2fs 状态=%v 单请求 中位=%.3fs p95=%.3fs max=%.3fs 吞吐=%.2f req/s\n",
			round+1, wall, codes, median(okDurs), percentile(okDurs, 0.95), maxFloat(okDurs), float64(*conc)/wall)
		time.Sleep(300 * time.Millisecond)
	}
	if len(allOK) > 0 {
		fmt.Printf("  汇总: 成功 %d/%d  中位=%.3fs p95=%.3fs\n",
			len(allOK), allTotal, median(allOK), percentile(allOK, 0.95))

		// 并发健康度判定（docs/28 §8.5.1 / docs/30 §九 判据改造）：
		// 旧判据「加速比 < 1.5× 即失败」存在设计缺陷——由 Little 定律可推出
		// 加速比 ≈ NumCPU / f（f 为枚举段占单请求服务时长的比例），比值只取决于
		// 枚举段占比：串行路径被优化得越好，比值越低，判据越难过（假阳性）；
		// 而 worker 失控超订但串行段很慢时反而可能达标（假阴性）。
		// 现改为两条直击「不超订」初衷的判据：
		//   ① 并发吞吐不低于串行吞吐——并行至少不比串行慢；
		//   ② 并发 p95 ≤ 串行时长 × 12——防排队延迟爆炸。
		//      校准依据：worker 钳制 NumCPU + 低枚举占比下，p95 膨胀 ≈ (并发/NumCPU)×(1/f)，
//      本机三轮实测 8-9×（docs/28 §6 中位 6.9s / 串行 0.87s 同构），12× 为包络上限；
//      OOM/无界排队类爆炸通常是 30×+，仍会被抓住。
		// 加速比仅作信息展示，不再作为判定依据。
		if serial := durs["双名 基线"]; serial > 0 && totalWall > 0 {
			concThroughput := float64(allTotal) / totalWall
			serialThroughput := 1.0 / serial
			speedup := concThroughput * serial
			concP95 := percentile(allOK, 0.95)
			fmt.Printf("  并发加速比: %.2f×（并发吞吐 %.2f req/s / 串行吞吐 %.2f req/s）\n",
				speedup, concThroughput, serialThroughput)

			if concThroughput < serialThroughput {
				ck.bad("并发吞吐 %.2f req/s 低于串行 %.2f req/s——并发反而更慢，调度串行化或资源争用",
					concThroughput, serialThroughput)
			} else {
				ck.ok("并发吞吐 %.2f req/s ≥ 串行 %.2f req/s", concThroughput, serialThroughput)
			}
			if concP95 > serial*12 {
				ck.bad("并发 p95 %.3fs 超过串行时长 %.3fs 的 12 倍——排队延迟爆炸，存在超订或资源耗尽",
					concP95, serial)
			} else {
				ck.ok("并发 p95 %.3fs ≤ 串行 %.3fs × 12（排队延迟受控）", concP95, serial)
			}
		}
	}

	// ---------- 7. 测名 ----------
	section("7. 测名 /names/evaluate（与生成同源同分）")
	if baseline != nil && len(baseline.names) > 0 {
		top := baseline.names[0]
		given, full := strField(top, "given_name"), strField(top, "full_name")
		genScore := numAt(&view{data: top}, "total_score")
		body := baseBody()
		body["given_name"] = given
		r := c.call(pathEvaluate, body)
		// 测名把名字对象放在 data.name 下，下沉一层即可复用同一套字段读取逻辑
		evScore := numAt(r.parseAt("name"), "total_score")
		fmt.Printf("  测名 %s：HTTP=%d 耗时=%.3fs 生成链路分=%.2f 测名分=%.2f\n",
			full, r.httpCode, r.dur.Seconds(), genScore, evScore)
		if r.httpCode != 200 {
			ck.bad("测名 %s 返回非 200: HTTP=%d", full, r.httpCode)
		} else if math.IsNaN(evScore) {
			ck.bad("测名响应缺少 total_score，前端无法展示评分")
		} else if evScore < 0 || evScore > 100 {
			ck.bad("测名评分 %.2f 越界（应为 0-100）", evScore)
		} else {
			ck.ok("测名评分 %.2f 在合理区间", evScore)
		}
		if !math.IsNaN(genScore) && !math.IsNaN(evScore) && math.Abs(genScore-evScore) > 0.01 {
			ck.bad("同一名字生成链路 %.2f 与测名 %.2f 不同分，两条链路评分未收口", genScore, evScore)
		} else if !math.IsNaN(genScore) {
			ck.ok("测名与生成链路同分（%.2f）", genScore)
		}
		// 参数校验：缺 given_name 必须被拒
		if bad := c.call(pathEvaluate, baseBody()); bad.httpCode != 400 {
			ck.bad("测名缺少 given_name 时应返回 400，实际 HTTP=%d", bad.httpCode)
		} else {
			ck.ok("测名缺少 given_name 正确返回 400")
		}
	}

	// ---------- 8. 探索模式 ----------
	section("8. 探索模式 /names/generate/explore（换一批）")
	// 注意：探索会话存在进程内 FIFO 缓存（容量 32），前面压测已生成几十个会话，
	// 直接用第 1 节的基线 generation_id 必然已被淘汰。真实用户也是「生成→换一批」
	// 紧邻操作，因此这里就地重新生成一次再换一批。
	fresh := c.call(pathGenerate, baseBody())
	fv := fresh.parse()
	gid := strAt(fv, "generation_id")
	fmt.Printf("  新会话 generation_id=%q\n", gid)
	if gid == "" {
		ck.bad("/names/generate 响应缺少 generation_id，探索模式无法工作")
	} else {
		r := c.call(pathExplore, map[string]any{"generation_id": gid, "count": 10})
		v := r.parse()
		fmt.Printf("  换一批: HTTP=%d 耗时=%.3fs 名字数=%d 剩余=%s\n",
			r.httpCode, r.dur.Seconds(), len(v.names), fmtNum(numAt(v, "remaining")))
		if r.httpCode != 200 {
			ck.bad("探索模式返回非 200: HTTP=%d %s", r.httpCode, clip(string(r.raw), 200))
		} else if len(v.names) == 0 {
			ck.bad("探索模式未返回任何名字（候选池已取尽或会话失效）")
		} else {
			ck.ok("探索模式返回 %d 个名字", len(v.names))
			// 核心契约：换一批与上一轮零交集
			prev, next := fv.givenNames(50), v.givenNames(50)
			overlap := intersectCount(prev, next)
			fmt.Printf("  与本轮 Top50 交集: %d（契约要求 0）\n", overlap)
			if overlap > 0 {
				ck.bad("换一批与上一轮存在 %d 个重复名字，MarkShown 去重失效", overlap)
			} else {
				ck.ok("换一批与上一轮零交集")
			}
		}

		// 会话被 FIFO 淘汰后的行为：必须给出可恢复的明确错误，而不是 500 或空榜。
		// 用单名（约 0.2s）快速把缓存顶满，避免这一步拖慢巡检。
		for i := 0; i < exploreStoreCapacity+2; i++ {
			c.call(pathGenerate, withOver(map[string]any{"name_length": 1}))
		}
		e := c.call(pathExplore, map[string]any{"generation_id": gid, "count": 10})
		msg := strField(e.parse().data, "message")
		fmt.Printf("  会话被淘汰后: HTTP=%d message=%q\n", e.httpCode, msg)
		if e.httpCode != 400 {
			ck.bad("探索会话被淘汰后应返回 400，实际 HTTP=%d", e.httpCode)
		} else if !strings.Contains(msg, "重新生成") {
			ck.bad("会话淘汰的错误提示未引导用户重新生成: %q", msg)
		} else {
			ck.ok("会话淘汰后返回 400 且提示用户重新生成（缓存容量 %d，FIFO 淘汰）", exploreStoreCapacity)
		}
	}

	// ---------- 汇总 ----------
	section("汇总")
	if len(ck.issues) == 0 {
		fmt.Printf("  ✓ %d 项检查全部通过\n", ck.total)
	} else {
		fmt.Printf("  %d 项检查，发现 %d 个问题：\n", ck.total, len(ck.issues))
		for i, s := range ck.issues {
			fmt.Printf("    %d) %s\n", i+1, s)
		}
	}
	if *strict && len(ck.issues) > 0 {
		os.Exit(1)
	}
}

// ---------- 小工具 ----------

func countPositive(m map[string]int) int {
	n := 0
	for _, v := range m {
		if v > 0 {
			n++
		}
	}
	return n
}

func intersectCount(a, b []string) int {
	set := make(map[string]bool, len(b))
	for _, s := range b {
		set[s] = true
	}
	n := 0
	for _, s := range a {
		if set[s] {
			n++
		}
	}
	return n
}

// diffKeys 返回只在 a 或只在 b 中出现的键（带 +/- 前缀），用于定位结构差异。
func diffKeys(a, b []string) []string {
	sa := make(map[string]bool, len(a))
	sb := make(map[string]bool, len(b))
	for _, k := range a {
		sa[k] = true
	}
	for _, k := range b {
		sb[k] = true
	}
	var out []string
	for _, k := range a {
		if !sb[k] {
			out = append(out, "-"+k)
		}
	}
	for _, k := range b {
		if !sa[k] {
			out = append(out, "+"+k)
		}
	}
	sort.Strings(out)
	return out
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	sortedA := append([]string(nil), a...)
	sortedB := append([]string(nil), b...)
	sort.Strings(sortedA)
	sort.Strings(sortedB)
	for i := range sortedA {
		if sortedA[i] != sortedB[i] {
			return false
		}
	}
	return true
}

func filterPositive(xs []float64) []float64 {
	out := make([]float64, 0, len(xs))
	for _, x := range xs {
		if x > 0 {
			out = append(out, x)
		}
	}
	return out
}

func maxFloat(xs []float64) float64 {
	m := 0.0
	for _, x := range xs {
		if x > m {
			m = x
		}
	}
	return m
}
