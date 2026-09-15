// 命令 gate_audit 审计「名字用字质量门禁表」（data/naming_quality.json）的实际覆盖面，
// 用于判断该表能否移除、以及还差哪些字需要补。
//
// 两部分分析：
//
//	A. 静态覆盖 —— 门禁表里的每一个字，有多少已经被「其他字级防线」挡掉：
//	   笔画区间、生僻字/表外字、性别暗示、硬负面字、消极字、IsRegular/CommonLevel。
//	   若一个字在门禁之外就已经进不了候选池，那它在门禁表里就是冗余条目。
//	B. 动态对照 —— 双名/单名分别跑两次完整生成（启用门禁 vs 门禁置空），
//	   对比候选池规模、已评分组合数、Top-N 差异，看移除门禁会不会真的放进荒谬字。
//
// 用法（在 backend 目录下）：
//
//	go run ./tools/gate_audit
//	go run ./tools/gate_audit -surname 李 -count 50
//	go run ./tools/gate_audit -static-only     # 跳过生成，仅静态分析
//
// 本工具只读 data 目录，不写入任何项目数据。
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"name/internal/application/services"
	"name/internal/domain/fate"
	"name/internal/domain/hanzi"
	"name/internal/infrastructure/cache"
	"name/internal/infrastructure/data"
)

type genResult struct {
	top       []string
	scores    map[string]float64
	total     int
	poolSize  int
	elapsed   time.Duration
	timedOut  bool
	topFirstN int
}

func main() {
	dataDirFlag := flag.String("data", "data", "数据目录")
	surname := flag.String("surname", "张", "姓氏")
	gender := flag.String("gender", "male", "性别 male/female")
	count := flag.Int("count", 50, "期望名字数")
	timeout := flag.Duration("timeout", 120*time.Second, "单次生成超时")
	staticOnly := flag.Bool("static-only", false, "只做静态覆盖分析，不跑生成对照")
	flag.Parse()

	dataDir := resolveDataDir(*dataDirFlag)
	if err := data.Init(dataDir); err != nil {
		fmt.Fprintf(os.Stderr, "数据装载失败: %v\n", err)
		os.Exit(2)
	}
	services.SyncNamingIndexFromHanzi()
	cache.Init()

	gate := readGate(dataDir)
	if len(gate) == 0 {
		fmt.Fprintf(os.Stderr, "门禁表为空或读取失败（%s/naming_quality.json）\n", dataDir)
		os.Exit(2)
	}
	if names := readCuratedNames(dataDir); len(names) > 0 {
		fate.SetCuratedNames(names)
	}

	fmt.Println(repeat("=", 96))
	fmt.Println("门禁表覆盖审计 ——  data/naming_quality.json")
	fmt.Println(repeat("=", 96))
	fmt.Printf("门禁表条目: %d 字\n", len(gate))

	// ---------- A. 静态覆盖 ----------
	filter := fate.NewFilterOption().WithGenderFilter(*gender).WithStrictness("moderate").Build()
	var q fate.CharacterQuery = fate.NewBasicCharacterQuery()
	q = filter.QueryFilter(q)
	provider := &services.HanziDataProvider{}
	pool, _ := provider.FindCharacters(q)
	poolSet := make(map[string]bool, len(pool))
	for _, c := range pool {
		poolSet[c.Char] = true
	}

	fmt.Println()
	fmt.Println(repeat("-", 96))
	fmt.Println("A. 静态覆盖：门禁字中，有多少已被其他字级防线排除")
	fmt.Println(repeat("-", 96))
	fmt.Printf("filter 链输出候选池: %d 字（在池内的门禁字才会被门禁表真正拦下）\n\n", len(pool))

	type row struct {
		char    string
		inHanzi bool
		level   int
		inPool  bool
		hard    bool
		neg     bool
		reason  string
	}
	rows := make([]row, 0, len(gate))
	for _, ch := range gate {
		h, ok := hanzi.HanziData[ch]
		r := row{char: ch, inHanzi: ok, inPool: poolSet[ch], hard: fate.IsHardNegativeChar(ch)}
		if ok {
			r.level = hanzi.GetNamerLevel(ch)
			r.neg = h.IsNegative
		}
		switch {
		case r.hard:
			r.reason = "已被硬负面字表覆盖"
		case !ok:
			r.reason = "不在 hanzi 字库中（无从入池）"
		case r.neg:
			r.reason = "已被 IsNegative 标记覆盖"
		case !r.inPool:
			r.reason = "已被 filter 链（笔画/生僻/性别/等级）挡下"
		default:
			r.reason = "★ 仅靠门禁表拦截"
		}
		rows = append(rows, r)
	}

	byReason := map[string][]string{}
	for _, r := range rows {
		byReason[r.reason] = append(byReason[r.reason], r.char)
	}
	reasons := make([]string, 0, len(byReason))
	for k := range byReason {
		reasons = append(reasons, k)
	}
	sort.Strings(reasons)
	onlyGate := 0
	for _, k := range reasons {
		chars := byReason[k]
		sort.Strings(chars)
		fmt.Printf("  %-42s %3d 字  %s\n", k, len(chars), truncate(strings.Join(chars, ""), 60))
		if k == "★ 仅靠门禁表拦截" {
			onlyGate = len(chars)
		}
	}
	fmt.Println()
	fmt.Printf("结论（静态）：门禁表 %d 字中，%d 字的拦截完全依赖该表，%d 字属冗余（其他防线已覆盖）。\n",
		len(gate), onlyGate, len(gate)-onlyGate)
	if onlyGate == 0 {
		fmt.Println("  → 在这一 filter 配置下，门禁表对「双名候选池」不产生任何额外过滤效果。")
	}

	// ---------- C. 替代方案（黑名单 → 正向信号准入）可行性 ----------
	fmt.Println()
	fmt.Println(repeat("-", 96))
	fmt.Println("C. 替代方案可行性：把「黑名单剔除」换成「正向信号准入」")
	fmt.Println(repeat("-", 96))
	sig := 0
	for _, c := range pool {
		if fate.IsCuratedChar(c.Char) || c.PositiveScore > 0 {
			sig++
		}
	}
	fmt.Printf("  当前候选池           : %d 字\n", len(pool))
	fmt.Printf("  其中带正向信号的字   : %d 字（策展字 或 positiveScore>0）\n", sig)
	fmt.Printf("  无任何正向信号的字   : %d 字（占 %.0f%%，评分上与优质字不可区分）\n",
		len(pool)-sig, 100*float64(len(pool)-sig)/float64(len(pool)))
	fmt.Printf("  双名组合规模         : 现池 %d 对  →  准入集 %d 对\n",
		len(pool)*len(pool), sig*sig)
	fmt.Println("  说明：准入集过小时会重新引发「候选收窄 → 名字高度近似」的老问题，")
	fmt.Println("        因此更现实的做法是「补信号」（扩大策展/正分覆盖）而非「删表」。")

	if *staticOnly {
		fmt.Println("\n（-static-only：已跳过动态对照）")
		return
	}

	// ---------- B. 动态对照 ----------
	fmt.Println()
	fmt.Println(repeat("-", 96))
	fmt.Println("B. 动态对照：启用门禁 vs 门禁置空，完整生成的差异")
	fmt.Println(repeat("-", 96))

	for _, nlen := range []int{2, 1} {
		label := fmt.Sprintf("%s 双名", *surname)
		if nlen == 1 {
			label = fmt.Sprintf("%s 单名", *surname)
		}
		fmt.Printf("\n【%s】\n", label)

		on, err := generate(provider, filter, *surname, *gender, nlen, *count, *timeout)
		if err != nil {
			fmt.Printf("  启用门禁：生成失败 %v\n", err)
			continue
		}
		// 门禁置空：指向一个不存在的目录即可让 LoadNamingQualityFromJSON 降级为空表
		if err := fate.LoadNamingQualityFromJSON(filepath.Join(dataDir, "__disable_gate__")); err != nil {
			fmt.Printf("  门禁置空失败: %v\n", err)
			_ = fate.LoadNamingQualityFromJSON(dataDir)
			continue
		}
		off, err := generate(provider, filter, *surname, *gender, nlen, *count, *timeout)
		// 立即恢复门禁状态，避免影响后续分析
		_ = fate.LoadNamingQualityFromJSON(dataDir)
		if err != nil {
			fmt.Printf("  门禁置空：生成失败 %v\n", err)
			continue
		}

		fmt.Printf("  %-14s %-9s %-14s %s\n", "配置", "耗时", "已评分组合数", "门禁字进入 Top榜")
		fmt.Printf("  %-14s %-9s %-14d %s\n", "启用门禁", on.elapsed.Round(time.Millisecond),
			on.total, gateCharsInTop(on.top, gate))
		fmt.Printf("  %-14s %-9s %-14d %s\n", "门禁置空", off.elapsed.Round(time.Millisecond),
			off.total, gateCharsInTop(off.top, gate))

		inter := intersectCount(on.top, off.top)
		fmt.Printf("  Top%d 交集: %d/%d（差异 %d 个）\n", *count, inter, len(on.top),
			len(on.top)-inter)
		if len(on.top) > 0 && len(off.top) > 0 {
			fmt.Printf("  启用门禁 Top10: %s\n", joinLimit(on.top, 10))
			fmt.Printf("  门禁置空 Top10: %s\n", joinLimit(off.top, 10))
		}
		if g := gateCharsInTop(off.top, gate); g == "无" {
			if inter == len(on.top) && len(on.top) > 0 {
				fmt.Println("  → 移除门禁表对该路径「零影响」（榜单完全相同）")
			} else {
				fmt.Println("  → 虽然榜单有差异，但没有任何门禁字挤进 Top 榜，说明门禁表在本路径上已被其他防线取代。")
			}
		}
	}
}

// ---------- 生成 ----------

func generate(provider *services.HanziDataProvider, filter fate.Filter, surname, gender string,
	nameLength, count int, timeout time.Duration) (*genResult, error) {
	analyzer := services.NewBaziAnalyzerAdapter()
	born := time.Date(2024, 5, 20, 10, 0, 0, 0, time.UTC)
	fd, err := analyzer.Analyze(born, fate.Gender(gender))
	if err != nil {
		return nil, fmt.Errorf("八字分析失败: %w", err)
	}

	engine := fate.NewEngine(provider, analyzer, fate.DefaultRaters())
	sess := engine.NewSessionWithFilter(filter)
	input := fate.Input{
		Surname: surname,
		Born:    born,
		Gender:  fate.Gender(gender),
		Options: fate.GenerateOptions{Count: count, NameLength: nameLength},
	}
	start := time.Now()
	if err := sess.Start(context.Background(), &input); err != nil {
		return nil, fmt.Errorf("会话启动失败: %w", err)
	}
	_ = fd

	done := make(chan error, 1)
	go func() { done <- sess.Wait() }()

	select {
	case err := <-done:
		if err != nil {
			return nil, err
		}
		out := sess.Result()
		res := &genResult{total: out.TotalCount, elapsed: time.Since(start), scores: map[string]float64{}}
		for _, n := range out.TopNames {
			res.top = append(res.top, n.GivenName)
			res.scores[n.GivenName] = n.Score.Total
		}
		return res, nil
	case <-time.After(timeout):
		_ = sess.Stop()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
		}
		return nil, fmt.Errorf("生成超时（%v）", timeout)
	}
}

// ---------- 数据读取 ----------

func readGate(dataDir string) []string {
	raw, err := os.ReadFile(filepath.Join(dataDir, "naming_quality.json"))
	if err != nil {
		return nil
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}

func readCuratedNames(dataDir string) []string {
	raw, err := os.ReadFile(filepath.Join(dataDir, "curated_names.json"))
	if err != nil {
		return nil
	}
	var entries []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.Name != "" {
			out = append(out, e.Name)
		}
	}
	return out
}

func resolveDataDir(preferred string) string {
	for _, dir := range []string{preferred, "data", "../data", "../../data"} {
		if dir == "" {
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, "namer.json")); err == nil {
			return dir
		}
	}
	return preferred
}

// ---------- 小工具 ----------

func gateCharsInTop(top []string, gate []string) string {
	set := make(map[string]bool, len(gate))
	for _, ch := range gate {
		set[ch] = true
	}
	var hits []string
	for _, name := range top {
		for _, ch := range name {
			if set[string(ch)] {
				hits = append(hits, string(ch))
			}
		}
	}
	if len(hits) == 0 {
		return "无"
	}
	return strings.Join(dedupStrings(hits), "、")
}

func dedupStrings(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
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

func joinLimit(xs []string, n int) string {
	if len(xs) <= n {
		return strings.Join(xs, " ")
	}
	return strings.Join(xs[:n], " ")
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

func repeat(s string, n int) string {
	out := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		out = append(out, s...)
	}
	return string(out)
}
