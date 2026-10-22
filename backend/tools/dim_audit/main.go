// dim_audit 是「起名解析说明」的专项巡检 CLI。
//
// 它直接调用真实 domain 逻辑（bazi / zodiac / hanzi / classics），用一组
// 已知答案（known-answer）参考 fixtures + 结构性不变量（invariant）来验证
// 四个维度——八字、生肖、五行（单字）、诗词出典——的解析结果是否
// 合理、准确，而非依赖肉眼抽检或 HTTP 链路。
//
// 用法：
//   go -C backend run ./tools/dim_audit            # 自动探测数据目录
//   go -C backend run ./tools/dim_audit -data ../data -raw ../data/raw
//   go -C backend run ./tools/dim_audit -v         # 逐项打印
//
// 退出码：0 = 全部通过；1 = 任一项失败。便于接入 pre-push / CI。
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"name/internal/domain/bazi"
	"name/internal/domain/classics"
	"name/internal/domain/hanzi"
	"name/internal/domain/zodiac"
)

// ---------------------------------------------------------------------------
// 巡检报告器
// ---------------------------------------------------------------------------

type reporter struct {
	suite     string
	pass      int
	fail      int
	failures  []string
	verbose   bool
}

func newReporter(suite string, verbose bool) *reporter {
	return &reporter{suite: suite, verbose: verbose}
}

func (r *reporter) check(name string, ok bool, detailFmt string, args ...interface{}) {
	if ok {
		r.pass++
		if r.verbose {
			fmt.Printf("    [PASS] %s\n", name)
		}
		return
	}
	r.fail++
	detail := name
	if detailFmt != "" {
		detail = name + " :: " + fmt.Sprintf(detailFmt, args...)
	}
	r.failures = append(r.failures, detail)
	fmt.Printf("    [FAIL] %s\n", detail)
}

func (r *reporter) summary() {
	total := r.pass + r.fail
	fmt.Printf("  [%s] 通过 %d / 失败 %d / 共 %d\n", r.suite, r.pass, r.fail, total)
}

// ---------------------------------------------------------------------------
// 通用参照表
// ---------------------------------------------------------------------------

var validWuxing = map[string]bool{"金": true, "木": true, "水": true, "火": true, "土": true}

var validStrength = map[string]bool{"身旺": true, "身中": true, "身弱": true, "身衰": true}

// branchToZodiac 地支 → 生肖（用于八字年柱与生肖的跨维度一致性校验）
var branchToZodiac = map[string]string{
	"子": "鼠", "丑": "牛", "寅": "虎", "卯": "兔", "辰": "龙", "巳": "蛇",
	"午": "马", "未": "羊", "申": "猴", "酉": "鸡", "戌": "狗", "亥": "猪",
}

// validGan / validZhi 天干地支合法性集合
var validGan = map[string]bool{"甲": true, "乙": true, "丙": true, "丁": true, "戊": true, "己": true, "庚": true, "辛": true, "壬": true, "癸": true}
var validZhi = map[string]bool{"子": true, "丑": true, "寅": true, "卯": true, "辰": true, "巳": true, "午": true, "未": true, "申": true, "酉": true, "戌": true, "亥": true}

func isTwoCharGanZhi(s string) bool {
	rs := []rune(s)
	if len(rs) != 2 {
		return false
	}
	return validGan[string(rs[0])] && validZhi[string(rs[1])]
}

func containsAllChars(s string, chars []string) bool {
	for _, c := range chars {
		if !strings.Contains(s, c) {
			return false
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// 主流程
// ---------------------------------------------------------------------------

func main() {
	var (
		dataDir string
		verbose bool
	)
	flag.StringVar(&dataDir, "data", "", "根数据目录（含 namer.json / shijing.json 等）；留空则自动探测")
	flag.BoolVar(&verbose, "v", false, "逐项打印 PASS")
	flag.Parse()

	if dataDir == "" {
		dataDir = detectDataDir()
	}
	if dataDir == "" {
		fmt.Fprintln(os.Stderr, "✗ 无法定位数据目录（需含 namer.json）；请用 -data 指定")
		os.Exit(2)
	}
	fmt.Printf("数据目录: %s\n\n", dataDir)

	// 加载真实数据：
	//   - 汉字五行：LoadNamerFromJSON 读 namer.json（运行时权威源，含五行自动校正）
	//     —— 注意不可误用 data/raw/hanzi.json（旧原料，无运行时调用方）
	//   - 诗词索引：BuildPoemIndexFromJSON 读 data 根目录的 shijing/chuci/shici/yuanqu.json
	if err := hanzi.LoadNamerFromJSON(dataDir); err != nil {
		fmt.Fprintf(os.Stderr, "✗ 加载汉字数据失败: %v\n", err)
		os.Exit(2)
	}
	if err := classics.BuildPoemIndexFromJSON(dataDir); err != nil {
		fmt.Fprintf(os.Stderr, "✗ 构建诗词索引失败: %v\n", err)
		os.Exit(2)
	}

	var (
		totalPass, totalFail int
		anyFail             bool
	)

	run := func(name string, fn func(*reporter)) {
		fmt.Printf("== %s ==\n", name)
		r := newReporter(name, verbose)
		fn(r)
		r.summary()
		totalPass += r.pass
		totalFail += r.fail
		if r.fail > 0 {
			anyFail = true
		}
		fmt.Println()
	}

	run("维度一 · 八字与五行（bazi）", auditBazi)
	run("维度二 · 生肖（zodiac）", auditZodiac)
	run("维度三 · 单字五行（hanzi wuxing）", auditWuxing)
	run("维度四 · 诗词出典准确性（classics）", auditPoetry)

	fmt.Printf("========================================\n")
	fmt.Printf("总览: 通过 %d / 失败 %d\n", totalPass, totalFail)
	if anyFail {
		fmt.Println("✗ 存在未通过项，请检查上方 [FAIL] 明细")
		os.Exit(1)
	}
	fmt.Println("✓ 全部专项巡检通过")
}

// detectDataDir 从 CWD 向上探测含 shijing.json 的目录
func detectDataDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	candidates := []string{
		cwd,
		filepath.Join(cwd, "data"),
		filepath.Join(cwd, "..", "data"),
		filepath.Join(cwd, "..", "..", "data"),
		filepath.Join(cwd, "backend", "data"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c, "shijing.json")); err == nil {
			abs, err := filepath.Abs(c)
			if err == nil {
				return abs
			}
			return c
		}
	}
	return ""
}

// ---------------------------------------------------------------------------
// 维度一：八字与五行
// ---------------------------------------------------------------------------

func auditBazi(r *reporter) {
	// --- 已知答案：年柱在「立春」换年（2000 为庚辰龙年） ---
	type yearCase struct {
		y, m, d, hh, mm int
		wantYearGZ      string
	}
	yearCases := []yearCase{
		{2000, 1, 1, 0, 30, "己卯"},  // 立春(约2/5)前 → 仍为己卯(兔)年
		{2000, 2, 10, 12, 0, "庚辰"}, // 立春后 → 庚辰(龙)年
		{1999, 12, 31, 12, 0, "己卯"}, // 1999 己卯兔年
	}
	for _, c := range yearCases {
		got, err := bazi.AnalyzeBazi(c.y, c.m, c.d, c.hh, c.mm)
		r.check(fmt.Sprintf("年柱边界 %04d-%02d-%02d", c.y, c.m, c.d),
			err == nil && got.Bazi.YearGanzhi == c.wantYearGZ,
			"期望 %s，实得 %q (err=%v)", c.wantYearGZ, got.Bazi.YearGanzhi, err)
	}

	// --- 跨维度一致性：八字年柱地支 → 生肖 应与 GetZodiacByName 匹配 ---
	for _, yr := range []int{2000, 1999, 1988, 2012, 2024} {
		// 取年中(7/1)确保年柱==公历年，便于与 GetZodiacByYear 对照
		got, err := bazi.AnalyzeBazi(yr, 7, 1, 12, 0)
		if err != nil {
			r.check(fmt.Sprintf("八字可解析 %d", yr), false, "err=%v", err)
			continue
		}
		rs := []rune(got.Bazi.YearGanzhi)
		branch := string(rs[1])
		wantZodiac, ok := branchToZodiac[branch]
		if !ok {
			r.check(fmt.Sprintf("年柱地支合法 %d", yr), false, "非法地支 %q", branch)
			continue
		}
		z := zodiac.GetZodiacByName(wantZodiac)
		r.check(fmt.Sprintf("八字年支→生肖 %d (%s→%s)", yr, branch, wantZodiac),
			z != nil && z.Name == wantZodiac,
			"GetZodiacByName(%s)=%v", wantZodiac, z)
		// 与 GetZodiacByYear 对照（公历年）
		zy := zodiac.GetZodiacByYear(yr)
		r.check(fmt.Sprintf("八字年支 vs GetZodiacByYear %d", yr),
			zy != nil && zy.Name == wantZodiac,
			"GetZodiacByYear=%v 期望 %s", zy, wantZodiac)
	}

	// --- 结构不变量 + 可复现性：多组样本 ---
	samples := [][5]int{
		{2000, 5, 1, 10, 0},
		{1995, 12, 25, 8, 30},
		{2010, 3, 15, 22, 0},
		{1988, 8, 8, 14, 20},
	}
	for _, s := range samples {
		got, err := bazi.AnalyzeBazi(s[0], s[1], s[2], s[3], s[4])
		if err != nil {
			r.check(fmt.Sprintf("八字可解析 %v", s), false, "err=%v", err)
			continue
		}
		tag := fmt.Sprintf("%04d-%02d-%02d", s[0], s[1], s[2])
		r.check(tag+" 四柱为合法干支",
			isTwoCharGanZhi(got.Bazi.YearGanzhi) && isTwoCharGanZhi(got.Bazi.MonthGanzhi) &&
				isTwoCharGanZhi(got.Bazi.DayGanzhi) && isTwoCharGanZhi(got.Bazi.HourGanzhi),
			"年=%s 月=%s 日=%s 时=%s", got.Bazi.YearGanzhi, got.Bazi.MonthGanzhi, got.Bazi.DayGanzhi, got.Bazi.HourGanzhi)
		r.check(tag+" 日主五行合法", validWuxing[got.DayMaster],
			"DayMaster=%q", got.DayMaster)
		r.check(tag+" 日主强弱合法", validStrength[got.DayMasterStrength],
			"DayMasterStrength=%q", got.DayMasterStrength)
		r.check(tag+" 喜用神非空且合法", len(got.Xiyongshen) > 0 && allValidWuxing(got.Xiyongshen),
			"Xiyongshen=%v", got.Xiyongshen)
		wuxSum := got.Wuxing.Jin + got.Wuxing.Mu + got.Wuxing.Shui + got.Wuxing.Huo + got.Wuxing.Tu
		r.check(tag+" 五行计数之和>0", wuxSum > 0, "sum=%d", wuxSum)
		r.check(tag+" 季节取值合法", got.Season == "" || got.Season == "春" || got.Season == "夏" || got.Season == "秋" || got.Season == "冬",
			"Season=%q", got.Season)

		// 可复现性：同输入两次结果一致
		again, _ := bazi.AnalyzeBazi(s[0], s[1], s[2], s[3], s[4])
		r.check(tag+" 可复现(年柱/喜用神一致)",
			again != nil && again.Bazi.YearGanzhi == got.Bazi.YearGanzhi &&
				strings.Join(again.Xiyongshen, ",") == strings.Join(got.Xiyongshen, ","),
			"首次=%s/%v 再次=%s/%v", got.Bazi.YearGanzhi, got.Xiyongshen, again.Bazi.YearGanzhi, again.Xiyongshen)
	}
}

func allValidWuxing(xs []string) bool {
	for _, x := range xs {
		if !validWuxing[x] {
			return false
		}
	}
	return true
}

// ---------------------------------------------------------------------------
// 维度二：生肖
// ---------------------------------------------------------------------------

func auditZodiac(r *reporter) {
	// 已知答案：基于 (year-1900)%12 的索引
	cases := map[int]string{
		2024: "龙", 2023: "兔", 2025: "蛇", 2000: "龙", 1999: "兔", 2012: "龙", 1988: "龙",
	}
	for yr, want := range cases {
		z := zodiac.GetZodiacByYear(yr)
		r.check(fmt.Sprintf("GetZodiacByYear(%d)", yr), z != nil && z.Name == want,
			"期望 %s，实得 %v", want, z)
	}

	// 已知答案：五行属性
	wuxingCases := map[string]string{
		"鼠": "水", "牛": "土", "虎": "木", "兔": "木", "龙": "土", "蛇": "火",
		"马": "火", "羊": "土", "猴": "金", "鸡": "金", "狗": "土", "猪": "水",
	}
	for name, want := range wuxingCases {
		z := zodiac.GetZodiacByName(name)
		r.check(fmt.Sprintf("生肖五行 %s", name), z != nil && z.Wuxing == want,
			"期望 %s，实得 %q", want, safeWuxing(z))
	}

	// 自洽：GetZodiacByID 与 GetZodiacByName 一致
	for i := 1; i <= 12; i++ {
		byID := zodiac.GetZodiacByID(i)
		if byID == nil {
			r.check(fmt.Sprintf("GetZodiacByID(%d) 非空", i), false, "返回 nil")
			continue
		}
		byName := zodiac.GetZodiacByName(byID.Name)
		r.check(fmt.Sprintf("ID/Name 一致 %d", i), byName != nil && byName.ID == i,
			"byID.Name=%s byName.ID=%d", byID.Name, safeID(byName))
	}

	// 12 生肖五行覆盖完整（每个都应是合法五行）
	all := zodiac.GetAllZodiacs()
	r.check("12 生肖齐全", len(all) == 12, "len=%d", len(all))
	allWuxingOK := true
	for _, z := range all {
		if !validWuxing[z.Wuxing] {
			allWuxingOK = false
		}
	}
	r.check("生肖五行全部合法", allWuxingOK, "")
}

func safeWuxing(z *zodiac.Zodiac) string {
	if z == nil {
		return "<nil>"
	}
	return z.Wuxing
}
func safeID(z *zodiac.Zodiac) int {
	if z == nil {
		return -1
	}
	return z.ID
}

// ---------------------------------------------------------------------------
// 维度三：单字五行
// ---------------------------------------------------------------------------

func auditWuxing(r *reporter) {
	// 已知答案：在 ResolveWuxing 口径（覆盖表 > namer.json 校正 > 部首推导）下，
	// 这些字的五行应如下。重点是验证「部首推导值 != 校正值时取校正」
	// （明=水、马=水、石=金、玉=木、不=水、冲=金 均为部首与校正冲突、收敛后取校正）。
	known := map[string]string{
		"江":   "水", // 氵
		"林":   "木", // 木（覆盖表特例）
		"炎":   "火", // 火
		"铁":   "金", // 钅
		"城":   "土", // 土（覆盖表特例）
		"海":   "水",
		"森":   "木", // 覆盖表特例
		"灿":   "火",
		"锋":   "金",
		"坤":   "土", // 覆盖表特例
		"金":   "金",
		"木":   "木",
		"水":   "水",
		"火":   "火",
		"土":   "土",
		// —— 部首推导与校正冲突，收敛后必须取校正值（核心回归点）——
		"明": "水", // 部首日→火，校正=水
		"马": "水", // 部首马→火，校正=水
		"石": "金", // 部首石→土，校正=金
		"玉": "木", // 部首玉→土，校正=木
		"不": "水", // 部首一→土，校正=水
		"冲": "金", // 部首冫→水，校正=金
	}
	for ch, want := range known {
		// (a) ResolveWuxing 口径：GetCharacterWuxing 必须等于该值
		got := hanzi.GetCharacterWuxing(ch)
		r.check(fmt.Sprintf("单字五行(ResolveWuxing口径) %s", ch), got == want,
			"期望 %s，实得 %q", want, got)
		// (b) 收敛不变式：GetCharacterWuxing 必须等价于 ResolveWuxing(校正, 部首)，
		//     杜绝两套五行判定口径再次分叉（docs/33 遗留双口径问题）。
		if h, ok := hanzi.GetHanzi(ch); ok {
			resolved := hanzi.ResolveWuxing(ch, h.Wuxing, h.Radical)
			r.check(fmt.Sprintf("收敛到 ResolveWuxing %s", ch), got == resolved,
				"GetCharacterWuxing=%q 但 ResolveWuxing=%q（双口径分叉）", got, resolved)
		} else {
			r.check(fmt.Sprintf("收敛到 ResolveWuxing %s", ch), false, "未在 HanziData 中加载")
		}
	}

	// 合法性：一组常用字五行必须非空且属于 {金木水火土}
	common := []string{"王", "李", "张", "刘", "陈", "杨", "黄", "赵", "周", "吴",
		"明", "静", "思", "远", "嘉", "宇", "浩", "欣", "然", "泽"}
	allOK := true
	for _, ch := range common {
		got := hanzi.GetCharacterWuxing(ch)
		if got == "" || !validWuxing[got] {
			allOK = false
			r.check(fmt.Sprintf("常用字五行合法 %s", ch), false, "实得 %q", got)
		}
	}
	if allOK {
		r.check("常用字五行全部非空且合法", true, "")
	}
}

// ---------------------------------------------------------------------------
// 维度四：诗词出典准确性
// ---------------------------------------------------------------------------

func auditPoetry(r *reporter) {
	idx := classics.GetGlobalPoemIndex()
	r.check("诗词索引已构建", idx != nil, "GetGlobalPoemIndex()=nil")

	// 已知答案：名字明显出自某来源族，且选了该族后出处应落在族内（不回退全库）
	type fixture struct {
		name   string
		source string
	}
	fixtures := []fixture{
		{"关雎", "诗经"},   // 诗经·国风·关雎
		{"蒹葭", "诗经"},   // 诗经·秦风·蒹葭
		{"离骚", "楚辞"},   // 楚辞·离骚
		{"九歌", "楚辞"},   // 楚辞·九歌
		{"静夜", "唐诗宋词"}, // 李白《静夜思》
		{"春眠", "唐诗宋词"}, // 孟浩然《春晓》
		{"明月", "唐诗宋词"}, // 唐诗常见意象
	}
	for _, f := range fixtures {
		res := classics.QueryNamePoetryBySource(f.name, f.source)
		if res == nil || res.BestMatch == nil || res.BestMatch.Poem == nil {
			r.check(fmt.Sprintf("出典命中 %s/%s", f.name, f.source), false, "无匹配结果")
			continue
		}
		bestSrc := res.BestMatch.Poem.Source
		inFamily := classics.MatchedSourceMembers(f.source, bestSrc)
		r.check(fmt.Sprintf("出典来源=所选族 %s/%s", f.name, f.source),
			inFamily,
			"BestMatch.Source=%q 不在族 %q 内（或发生非预期回退）", bestSrc, f.source)
		// 结构性准确性：命中字 ⊆ 名字；引文含命中字
		matched := res.BestMatch.MatchedChars
		charsOK := allMatchedInName(f.name, matched)
		r.check(fmt.Sprintf("命中字⊆名字 %s", f.name), charsOK,
			"matched=%v", matched)
		quote := res.BestMatch.Quote
		quoteOK := len(matched) > 0 && containsAllChars(quote, matched)
		r.check(fmt.Sprintf("引文含命中字 %s", f.name), quoteOK,
			"quote=%q matched=%v", quote, matched)
	}

	// 全库查询结构性不变量（不指定来源）
	for _, name := range []string{"关雎", "静夜", "思远", "梓涵"} {
		res := classics.QueryNamePoetry(name)
		if res == nil {
			continue
		}
		if res.BestMatch == nil {
			continue
		}
		matched := res.BestMatch.MatchedChars
		r.check(fmt.Sprintf("全库命中字⊆名字 %s", name), allMatchedInName(name, matched),
			"matched=%v", matched)
		if res.BestMatch.Poem != nil {
			r.check(fmt.Sprintf("全库引文含命中字 %s", name),
				containsAllChars(res.BestMatch.Quote, matched),
				"quote=%q matched=%v", res.BestMatch.Quote, matched)
		}
	}
}

func allMatchedInName(name string, matched []string) bool {
	chars := make(map[string]bool)
	for _, r := range name {
		chars[string(r)] = true
	}
	for _, m := range matched {
		if !chars[m] {
			return false
		}
	}
	return true
}
