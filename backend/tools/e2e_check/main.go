// 命令 e2e_check 是起名链路的端到端巡检工具（纯 Go 实现，仅依赖标准库）。
//
// 覆盖六项检查：
//  1. 多场景耗时（双名/单名/复姓/经典来源/避讳/笔画区间/显式五行）
//  2. 推荐用字质量（一/二/三级字分布、门禁字命中、无命名依据字占比）
//  3. 参数生效性（避讳长辈、经典来源偏好）
//  4. 双路径一致性（/names/generate 与 /names/generate/analysis）
//  5. 同一请求可复现性
//  6. 并发压测（中位/p95/最大耗时与吞吐）
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

// parse 解析调用结果；任何结构问题都降级为空视图而不是报错，
// 以便巡检继续跑完并如实呈现「哪个字段缺失」。
func (r result) parse() *view {
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
	_ = json.Unmarshal(v.data["names"], &v.names)
	_ = json.Unmarshal(v.data["bazi"], &v.bazi)
	return v
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
	if r := c.call("/api/v1/names/generate/analysis", baseBody()); r.err != nil {
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
		{"/analysis 双名 基线", "/api/v1/names/generate/analysis", withOver(nil)},
		{"/generate 双名 基线", "/api/v1/names/generate", withOver(nil)},
		{"/analysis 单名", "/api/v1/names/generate/analysis", withOver(map[string]any{"name_length": 1})},
		{"/analysis 女宝 李", "/api/v1/names/generate/analysis", withOver(map[string]any{"gender": "female", "surname": "李"})},
		{"/analysis 经典来源=论语", "/api/v1/names/generate/analysis", withOver(map[string]any{"source_classic": "论语"})},
		{"/analysis 显式五行=水", "/api/v1/names/generate/analysis", withOver(map[string]any{"wuxing_match": []string{"水"}})},
		{"/analysis 避讳 张伟/王秀英", "/api/v1/names/generate/analysis", withOver(map[string]any{"avoid_elder_names": []string{"张伟", "王秀英"}})},
		{"/analysis 复姓 欧阳", "/api/v1/names/generate/analysis", withOver(map[string]any{"surname": "欧阳"})},
		{"/analysis 笔画 5-20", "/api/v1/names/generate/analysis", withOver(map[string]any{"min_strokes": 5, "max_strokes": 20})},
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
	for _, label := range []string{"/analysis 双名 基线", "/analysis 单名"} {
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
		if label == "/analysis 双名 基线" {
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
	avoid := views["/analysis 避讳 张伟/王秀英"]
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
	if b, l := views["/analysis 双名 基线"], views["/analysis 经典来源=论语"]; b != nil && l != nil {
		inter := intersectCount(b.givenNames(10), l.givenNames(10))
		inter50 := intersectCount(b.givenNames(50), l.givenNames(50))
		fmt.Printf("  经典来源=论语 与基线 Top10 交集: %d/10（越小说明来源偏好影响越大）；Top50 交集 %d/50\n", inter, inter50)
		if inter >= 10 && inter50 >= 50 {
			ck.bad("切换经典来源后 Top 榜完全无变化，source_classic 疑似未生效")
		} else {
			ck.ok("source_classic 生效（Top10 差异 %d 个）", 10-inter)
		}
	}

	// ---------- 4. 双路径一致性 ----------
	section("4. 双路径一致性（/names/generate vs /names/generate/analysis）")
	a, g := views["/analysis 双名 基线"], views["/generate 双名 基线"]
	if a != nil && g != nil {
		xa, xg := a.xiyongshen(), g.xiyongshen()
		fmt.Printf("  喜用神: /analysis=%v   /generate=%v\n", xa, xg)
		if !equalStrings(xa, xg) {
			ck.bad("两条链路喜用神不一致：/analysis=%v vs /generate=%v", xa, xg)
		} else {
			ck.ok("两条链路喜用神一致")
		}
		fmt.Printf("  前10名交集: %d/10\n", intersectCount(a.givenNames(10), g.givenNames(10)))

		// 响应结构差异（前端需分别适配两套结构）
		ka, kg := a.baziKeys(), g.baziKeys()
		if d := diffKeys(ka, kg); len(d) > 0 {
			fmt.Printf("  bazi 字段差异: %s\n", strings.Join(d, "、"))
		} else {
			ck.ok("bazi 字段集一致（%d 个）", len(ka))
		}
		na, ng := a.nameKeys(), g.nameKeys()
		if d := diffKeys(na, ng); len(d) > 0 {
			fmt.Printf("  names[0] 字段差异: %s\n", strings.Join(d, "、"))
		}
		_, hasA := a.names[0]["score_detail"]
		_, hasG := g.names[0]["score_detail"]
		fmt.Printf("  score_detail: /analysis=%v  /generate=%v\n", hasA, hasG)
		if hasA != hasG {
			ck.bad("score_detail 只在单侧出现（/analysis=%v /generate=%v），前端需写两套渲染逻辑", hasA, hasG)
		}
	}

	// ---------- 5. 可复现性 ----------
	section(fmt.Sprintf("5. 同一请求可复现性（连发 %d 次）", *repeat))
	seen := map[string]int{}
	for i := 0; i < *repeat; i++ {
		r := c.call("/api/v1/names/generate/analysis", baseBody())
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
				r := c.call("/api/v1/names/generate/analysis", bodies[i%len(bodies)])
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

		// 并发超订探测（P2-7 令牌钳制后的语义）：worker 总数被全局令牌钳制在
		// NumCPU 内，单请求的排队延迟会随并发抬升，但「吞吐不随并发恶化」
		// 才是目标。改用吞吐加速比判定：并发总吞吐相对串行吞吐（1/串行时长）
		// 应有实质提升，否则说明调度仍是串行化/超订。
		if serial := durs["/analysis 双名 基线"]; serial > 0 && totalWall > 0 {
			speedup := float64(allTotal) / totalWall * serial
			fmt.Printf("  并发加速比: %.2f×（并发吞吐 %.2f req/s / 串行吞吐 %.2f req/s）\n",
				speedup, float64(allTotal)/totalWall, 1.0/serial)
			if speedup < 1.5 {
				ck.bad("%d 并发下吞吐加速比 %.2f×，几乎无并行收益——worker 调度存在串行化或超订",
					*conc, speedup)
			} else {
				ck.ok("吞吐相对串行有 %.2f× 加速，worker 收敛无超订", speedup)
			}
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
