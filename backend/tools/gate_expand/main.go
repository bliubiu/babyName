// 命令 gate_expand 扩容「名字用字质量门禁表」data/naming_quality.json
//
// 背景
//
//	门禁表历史上是纯人工策展资产：git 历史 commit aa024f4 的
//	internal/domain/fate/naming_quality.go 内嵌了 1094 字，并按「verify_fate
//	复测抓到的漏网字」逐轮追加，前后补了 40+ 轮仍未收敛。此后字表外置为
//	data/naming_quality.json，而 backend/data/ 不入版本库，文件丢失后只能凭
//	记忆重建 131 字核心表（见 commit 3c40d7e），质量防线因此出现大缺口。
//
// 本工具做两件只增不删的事
//
//  1. 基准表恢复（-recover）：从 git 历史中把内嵌表重新提取出来并入。
//  2. 命名准入规则扩容：把「在四个维度上都没有任何正面证据」的字自动收进门禁表。
//
// 命名准入规则（四个维度全部为「无」）
//
//	a) 真实人名用法：未出现在 95.7 万条人名语料中（NameFreqTier == 0）
//	b) 人工寓意评分：positiveScore == 0
//	c) 策展好名认可：fate.IsCuratedChar 为假
//	d) 典籍诗词出处：classics.HasPoetryChar 为假
//
// 即该字既没人拿来起过名、也没被人工评过寓意、也不在任何策展好名与诗词典籍中
// 出现过——可判定为「无命名价值」。规则完全由仓库内数据推导，可复现、可审计。
//
// 用法（在 backend 目录下）
//
//	go run ./tools/gate_expand -dry-run                  # 只报告，不写文件
//	go run ./tools/gate_expand -recover aa024f4 -dry-run # 先恢复历史表再评估
//	go run ./tools/gate_expand -recover aa024f4 -v       # 落盘并列出全部新增字
//
// 注意：本工具只增不删，文件里已有的条目一律保留（人工策展条目不允许被规则覆盖）。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"name/internal/application/services"
	"name/internal/domain/classics"
	"name/internal/domain/fate"
	"name/internal/domain/hanzi"
	"name/internal/infrastructure/cache"
	"name/internal/infrastructure/data"
)

// historicalGateRev 内嵌完整门禁表（1094 字）的最后一次提交。
const historicalGateRev = "aa024f4"

// historicalGatePath 该提交中承载内嵌表的文件路径。
const historicalGatePath = "backend/internal/domain/fate/naming_quality.go"

// guardChars 守护字表：这些字无论如何都不能进门槛表。
//
// 取两组来源：一是 naming_quality_test.go 中明确「必须放行」的清单（含大量
// 有虚词/数字属性但已形成稳定命名寓意的字），二是现代命名高频好字。
// 规则若命中其中任何一个，说明判据过宽，工具会直接失败而不是写出坏数据。
const guardChars = "若如然斯唯惟一三九百千万华毅辉涛轩书宇瑞明文云和子睿涵博卓雅琪琳雨雪梅兰" +
	"都路年微白江北河凡墨游鹤媚冉复" +
	"梓妍萱怡欣宁诗梦佳思俊杰翔鑫磊静雯露瑶珂玮琰琛皓懿麟骏熙曦瑾璐彧珺昶隽珩翊昕昱晟妤璟瑄玥涵泽清澄渊翰" +
	"瑄翀婳愔琮璩璠瓒瑜琤琚瑗琬頔"

func main() {
	dataDirFlag := flag.String("data", "data", "数据目录（内含 naming_quality.json / namer.json）")
	repoRoot := flag.String("repo", "..", "仓库根目录（-recover 需要，用于调用 git）")
	recoverRev := flag.String("recover", "", "从该 git 提交恢复历史内嵌表并并入基准表（留空则跳过）")
	dryRun := flag.Bool("dry-run", false, "只输出报告，不写入文件")
	verbose := flag.Bool("v", false, "列出全部新增字")
	flag.Parse()

	dataDir := resolveDataDir(*dataDirFlag)
	gatePath := filepath.Join(dataDir, "naming_quality.json")

	current := readGate(gatePath)
	currentSet := toSet(current)

	// ---------- 1. 基准表 ----------
	section("1. 基准表（人工策展）")
	fmt.Printf("  文件现有条目        : %d 字\n", len(current))

	base := currentSet
	if *recoverRev != "" {
		hist, err := recoverHistoricalGate(*repoRoot, *recoverRev)
		if err != nil {
			fail("从 %s 恢复历史门禁表失败: %v", *recoverRev, err)
		}
		added := difference(hist, currentSet)
		base = unionList(currentSet, hist)
		fmt.Printf("  自 %s 恢复        : %d 字（其中新增 %d 字）\n", *recoverRev, len(hist), len(added))
		if len(added) > 0 && *verbose {
			fmt.Printf("    新增: %s\n", joinSorted(added))
		}
	}

	// ---------- 2. 命名准入规则 ----------
	if err := data.Init(dataDir); err != nil {
		fail("数据装载失败: %v", err)
	}
	cache.Init()
	if names := readCuratedNames(filepath.Join(dataDir, "curated_names.json")); len(names) > 0 {
		fate.SetCuratedNames(names)
	}

	section("2. 命名准入规则扩容")
	rule := make([]string, 0, 2048)
	levelCount := map[int]int{}
	for ch, h := range hanzi.HanziData {
		if !isNamingIneligible(ch, h) {
			continue
		}
		rule = append(rule, ch)
		levelCount[hanzi.GetNamerLevel(ch)]++
	}
	sort.Strings(rule)
	fmt.Printf("  规则命中            : %d 字\n", len(rule))
	fmt.Printf("  等级分布            : 一级 %d / 二级 %d / 三级 %d\n",
		levelCount[1], levelCount[2], levelCount[3])

	newly := difference(rule, base)
	fmt.Printf("  其中相对基准表新增  : %d 字\n", len(newly))
	if len(newly) > 0 {
		if *verbose {
			fmt.Printf("    新增: %s\n", joinSorted(newly))
		} else {
			fmt.Printf("    样例（前 60）: %s\n", firstN(newly, 60))
		}
	}

	final := unionList(base, rule)

	// ---------- 3. 守护检查 ----------
	section("3. 守护字检查（优质字不得被误伤）")
	var hit []string
	for _, ch := range guardChars {
		if final[string(ch)] {
			hit = append(hit, string(ch))
		}
	}
	if len(hit) > 0 {
		fail("守护字被误伤 %d 个: %s\n规则过宽，已中止（未写入文件）", len(hit), strings.Join(hit, ""))
	}
	fmt.Printf("  ✓ %d 个守护字全部放行\n", len([]rune(guardChars)))

	// ---------- 4. 候选池影响 ----------
	section("4. 候选池影响（男宝 moderate 配置）")
	filter := fate.NewFilterOption().WithGenderFilter("male").WithStrictness("moderate").Build()
	var q fate.CharacterQuery = fate.NewBasicCharacterQuery()
	q = filter.QueryFilter(q)
	provider := &services.HanziDataProvider{}
	pool, _ := provider.FindCharacters(q)

	poolNew := 0
	for _, c := range pool {
		if final[c.Char] {
			poolNew++
		}
	}
	effective := len(pool) - poolNew
	fmt.Printf("  filter 链输出候选池 : %d 字\n", len(pool))
	fmt.Printf("  其中被门禁表剔除    : %d 字 → 双名可用 %d 字\n", poolNew, effective)
	fmt.Printf("  双名组合规模        : %.1f 万对 → %.1f 万对\n",
		float64(len(pool)*len(pool))/1e4, float64(effective*effective)/1e4)
	fmt.Println("  注：双名路径在候选池阶段不套门禁（组合内层循环才剔除），单名路径在池阶段即剔除；")
	fmt.Println("      单名池只有约 1100 字，受影响比例更高。")

	// ---------- 5. 落盘 ----------
	section("5. 落盘")
	out := make([]string, 0, len(final))
	for ch := range final {
		out = append(out, ch)
	}
	sort.Strings(out)

	if *dryRun {
		fmt.Printf("  -dry-run：未写入。合计将为 %d 字（现 %d 字，+%d）\n",
			len(out), len(current), len(out)-len(current))
		return
	}
	if err := writeGate(gatePath, out); err != nil {
		fail("写入 %s 失败: %v", gatePath, err)
	}
	fmt.Printf("  ✓ 已写入 %s：%d 字（原 %d 字，+%d）\n",
		gatePath, len(out), len(current), len(out)-len(current))
}

// exemptRadicals 吉祥部首豁免：这些部首承载明确的美好意象
// （美玉，「君子比德于玉」），即使现代人名语料罕见也具备命名价值。
// 仅当该字为三级（过于生僻）时仍执行规则。
var exemptRadicals = map[string]bool{"王": true, "玉": true}

// exemptChars 显式豁免字：经人工复核应从规则中排除的字。
// 收录标准与守护字表相同——出现即说明它确实是可用的名字用字。
var exemptChars = map[string]bool{
	"瑄": true, // 祭天玉璧，现代命名常用
	"翀": true, // 鸟向上直飞
	"婳": true, // 娴静美好
	"愔": true, // 安静和悦
}

// isNamingIneligible 命名准入判据：四个「证据维度」全部落空即判为无命名价值。
//
// 证据维度：
//
//	a) 真实人名用法：出现在 95.7 万条人名语料中（NameFreqTier > 0）
//	b) 人工寓意评分：positiveScore > 0
//	c) 策展好名认可：fate.IsCuratedChar 为真
//	d) 典籍诗词出处：classics.HasPoetryChar 为真
//
// 关于维度 d 的定位：它是**防误伤的逃生口，不是命名价值的证据**。实测全唐诗全宋词
// 等语料覆盖了《通用规范汉字表》的绝大多数汉字，沚/鲛/唣/僰/恃/荥/噬 这些霸榜荒谬字
// 全部都能查到诗词出处，所以「有出处」毫无区分力。反过来若把 d 作为必要条件（即
// 不查诗词就判无价值），规则会一次命中 6816/8105 = 84% 的字，连「彧」（荀彧）都会被
// 误伤——这已不是「无命名价值」而是「现代不常用」了。
//
// 结论：本规则用于「收录明显无命名价值的字」（宁缺毋滥），**不能**用来收敛
// 「无信号字与优质字同分」这一评分层缺陷——那需要 RateName 的封顶豁免判据修正。
func isNamingIneligible(ch string, h hanzi.Hanzi) bool {
	if exemptChars[ch] {
		return false
	}
	if exemptRadicals[h.Radical] && hanzi.GetNamerLevel(ch) <= 2 {
		return false
	}
	if h.PositiveScore > 0 { // b) 人工寓意评分
		return false
	}
	if h.NameFreqTier > 0 { // a) 95.7 万人名语料中出现过
		return false
	}
	if fate.IsCuratedChar(ch) { // c) 策展好名认可
		return false
	}
	if classics.HasPoetryChar(ch) { // d) 典籍诗词出处（防误伤逃生口）
		return false
	}
	return true
}

// recoverHistoricalGate 从 git 历史的内嵌 map 中提取门禁字。
//
// 仅在 -recover 显式指定时调用；失败即中止，避免用不完整的基准表写坏数据。
func recoverHistoricalGate(repoRoot, rev string) ([]string, error) {
	cmd := exec.Command("git", "-C", repoRoot, "show", rev+":"+historicalGatePath)
	raw, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	re := regexp.MustCompile(`"([^"]+)"\s*:\s*true`)
	seen := map[string]bool{}
	var out []string
	for _, m := range re.FindAllStringSubmatch(string(raw), -1) {
		ch := m[1]
		if ch == "" || seen[ch] {
			continue
		}
		seen[ch] = true
		out = append(out, ch)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("未在 %s:%s 中提取到任何条目", rev, historicalGatePath)
	}
	return out, nil
}

// ---------- 数据读写 ----------

func readGate(path string) []string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil {
		fail("解析 %s 失败: %v", path, err)
	}
	return out
}

// writeGate 以项目统一格式写回：4 空格缩进 + 按码点升序 + 末尾换行。
func writeGate(path string, chars []string) error {
	list := make([]string, 0, len(chars))
	seen := map[string]bool{}
	for _, ch := range chars {
		if ch == "" || seen[ch] {
			continue
		}
		seen[ch] = true
		list = append(list, ch)
	}
	sort.Strings(list)

	var b strings.Builder
	b.WriteString("[\n")
	for i, ch := range list {
		b.WriteString("    ")
		enc, _ := json.Marshal(ch)
		b.Write(enc)
		if i < len(list)-1 {
			b.WriteByte(',')
		}
		b.WriteByte('\n')
	}
	b.WriteString("]\n")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func readCuratedNames(path string) []string {
	raw, err := os.ReadFile(path)
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

func toSet(list []string) map[string]bool {
	m := make(map[string]bool, len(list))
	for _, s := range list {
		m[s] = true
	}
	return m
}

func union(a, b map[string]bool) map[string]bool {
	m := make(map[string]bool, len(a)+len(b))
	for k := range a {
		m[k] = true
	}
	for k := range b {
		m[k] = true
	}
	return m
}

// unionList 把字符串列表并入集合（就地修改并返回集合）。
func unionList(set map[string]bool, list []string) map[string]bool {
	for _, s := range list {
		set[s] = true
	}
	return set
}

func difference(list []string, exclude map[string]bool) []string {
	out := make([]string, 0, len(list))
	for _, s := range list {
		if !exclude[s] {
			out = append(out, s)
		}
	}
	return out
}

func joinSorted(list []string) string {
	s := append([]string(nil), list...)
	sort.Strings(s)
	return strings.Join(s, "")
}

func firstN(list []string, n int) string {
	if len(list) <= n {
		return strings.Join(list, "")
	}
	return strings.Join(list[:n], "") + "…"
}

func section(title string) {
	fmt.Println()
	fmt.Println(strings.Repeat("=", 88))
	fmt.Println(title)
	fmt.Println(strings.Repeat("=", 88))
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "错误: %s\n", fmt.Sprintf(format, args...))
	os.Exit(1)
}
