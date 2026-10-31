package ziwei

import (
	"fmt"
	"strings"
	"testing"
)

// 本文件固化辅星缺口（火铃、地空地劫、天空、截空、旬空、天伤天使）与
// 杂曜（SuiXingList 全 12 颗）安布规则，以及 14 主星《紫微斗数全书》
// 派亮度表。全部锚点经 iztro 2.6.3 实排对拍验证（sample：
// 公历 2000-08-16 巳时 女 → 庚辰年七月十七巳时，命宫辛卯，土五局）。

// TestCalculateFiveElementsClass_InvalidExplicit 命宫干支非法/纳音末字无法识别时
// 必须显式返回 ("", false)，不得再静默降级为「水二局」（docs/39 不静默降级原则）。
func TestCalculateFiveElementsClass_InvalidExplicit(t *testing.T) {
	invalid := []string{"", "甲", "甲乙丙", "X", "子"}
	for _, mg := range invalid {
		if fe, ok := calculateFiveElementsClass(mg); ok || fe != "" {
			t.Errorf("calculateFiveElementsClass(%q) = (%q,%v)，期望 (\"\",false)", mg, fe, ok)
		}
	}
	// 合法命宫仍应命中真实五行局（确认正常路径未破坏）
	if fe, ok := calculateFiveElementsClass("甲子"); !ok || fe != "金四局" {
		t.Errorf("calculateFiveElementsClass(甲子) = (%q,%v)，期望 (金四局,true)", fe, ok)
	}
}

// newAuxChart 构造仅含定位所需字段的合成盘：mingZhiIdx 即命宫实际地支索引，
// 命宫天干按五虎遁自寅起推算（安布只用宫名定位，不参与断言）
func newAuxChart(mingZhiIdx, yearGanIdx, yearZhiIdx int) *ZiweiChart {
	return &ZiweiChart{
		FuXing:   make(map[string][]string),
		SuiXing:  make(map[string][]string),
		NianZhu:  getGanZhi(yearGanIdx, yearZhiIdx),
		MingGong: getGanZhi(TigerRule[yearGanIdx]+fixIndex(mingZhiIdx-2, 12), mingZhiIdx),
	}
}

// palaceAt 命宫地支 mingZhiIdx 时，地支 z 所在的宫名（十二宫自命宫逆行）
func palaceAt(mingZhiIdx, zhi int) string {
	return Gongs[fixIndex(mingZhiIdx-zhi, 12)]
}

// containsStar 断言 m[gongName] 包含 star，tag 支持格式化
func containsStar(t *testing.T, m map[string][]string, gongName, star, tag string, args ...interface{}) {
	t.Helper()
	for _, s := range m[gongName] {
		if s == star {
			return
		}
	}
	t.Errorf("%s：%s 应落在 %s，实际分布不满足（%v）",
		fmt.Sprintf(tag, args...), star, gongName, m[gongName])
}

// ============================================================
// 辅星缺口
// ============================================================

// TestDistributeFuXing_Iztro对拍 辅星新安星全量对拍（sample 盘）：
// 庚辰年七月十七巳时、命宫在卯。
func TestDistributeFuXing_Iztro对拍(t *testing.T) {
	chart := CalculateZiweiChart(2000, 8, 16, 10, "女")
	if chart == nil || chart.MingGong != "己卯" {
		t.Fatalf("sample 盘命宫期望 己卯，got %s", chart.MingGong)
	}
	mingZhiIdx := 3 // 卯
	// 申子辰组（辰年）巳时（t=5）：火寅+5=未、铃戌+5=卯
	containsStar(t, chart.FuXing, palaceAt(mingZhiIdx, 7), "火星", "火铃")
	containsStar(t, chart.FuXing, palaceAt(mingZhiIdx, 3), "铃星", "火铃")
	// 地空=亥−t=午、地劫=亥+t=辰
	containsStar(t, chart.FuXing, palaceAt(mingZhiIdx, 6), "地空", "空劫")
	containsStar(t, chart.FuXing, palaceAt(mingZhiIdx, 4), "地劫", "空劫")
	// 天空=年支+1=巳
	containsStar(t, chart.FuXing, palaceAt(mingZhiIdx, 5), "天空", "天空")
	// 截空：庚年辰年（阳支）取截路支=午
	containsStar(t, chart.FuXing, palaceAt(mingZhiIdx, 6), "截空", "截空")
	// 旬空：庚辰属甲戌旬空申酉，年支阳取申
	containsStar(t, chart.FuXing, palaceAt(mingZhiIdx, 8), "旬空", "旬空")
	// 天伤居交友宫、天使居疾厄宫
	containsStar(t, chart.FuXing, palaceAt(mingZhiIdx, 8), "天伤", "天伤天使")
	containsStar(t, chart.FuXing, palaceAt(mingZhiIdx, 10), "天使", "天伤天使")
}

// TestDistributeFuXing_火铃四组 火星铃星起点表按年支三合组分派，
// 自起点顺数至生时（口诀：申子辰人寅戌扬，寅午戌人丑卯方，
// 巳酉丑人卯戌位，亥卯未人酉戌房）
func TestDistributeFuXing_火铃四组(t *testing.T) {
	cases := []struct {
		yearZhi int
		t       int
		huo     int
		ling    int
	}{
		// 申子辰 → 寅、戌；顺数 t
		{8, 0, 2, 10}, {4, 5, 7, 3},
		// 寅午戌 → 丑、卯
		{2, 2, 3, 5},
		// 巳酉丑 → 卯、戌
		{5, 1, 4, 11},
		// 亥卯未 → 酉、戌
		{11, 3, 0, 1},
		// 晚子时按子时同宫
		{8, 12, 2, 10},
	}
	for i, c := range cases {
		chart := newAuxChart(0, 0, c.yearZhi)
		distributeFuXing(chart, 1, c.t, 0, c.yearZhi)
		containsStar(t, chart.FuXing, palaceAt(0, c.huo), "火星", "火铃组%d", i)
		containsStar(t, chart.FuXing, palaceAt(0, c.ling), "铃星", "火铃组%d", i)
	}
}

// TestDistributeFuXing_空劫时序 地空=亥−时、地劫=亥+时，子时二者同宫于亥
func TestDistributeFuXing_空劫时序(t *testing.T) {
	cases := []struct{ t, kong, jie int }{
		{0, 11, 11}, {1, 10, 0}, {5, 6, 4}, {11, 0, 10}, {12, 11, 11},
	}
	for i, c := range cases {
		chart := newAuxChart(0, 0, 0)
		distributeFuXing(chart, 1, c.t, 0, 0)
		containsStar(t, chart.FuXing, palaceAt(0, c.kong), "地空", "空劫%d", i)
		containsStar(t, chart.FuXing, palaceAt(0, c.jie), "地劫", "空劫%d", i)
	}
}

// TestDistributeFuXing_截空阴阳性 截路空亡：甲己申酉、乙庚午未、丙辛辰巳、
// 丁壬寅卯、戊癸子丑；阳年支取截路支（前），阴年支取空亡支（后）
func TestDistributeFuXing_截空阴阳性(t *testing.T) {
	cases := []struct {
		gan, zhi, want int
	}{
		{0, 0, 8},  // 甲子（阳支）→ 申
		{0, 1, 9},  // 甲丑（阴支）→ 酉
		{2, 6, 4},  // 丙午（阳支）→ 辰
		{2, 5, 5},  // 丙巳（阴支）→ 巳
		{4, 10, 0}, // 戊戌（阳支）→ 子
		{9, 11, 1}, // 癸亥（阴支）→ 丑
	}
	for i, c := range cases {
		chart := newAuxChart(0, c.gan, c.zhi)
		distributeFuXing(chart, 1, 0, c.gan, c.zhi)
		containsStar(t, chart.FuXing, palaceAt(0, c.want), "截空", "截空%d", i)
	}
}

// TestDistributeFuXing_旬空阴阳性 旬空取年支所在旬的空亡两支中
// 与年支阴阳同性的支：甲子旬空戌亥、甲戌旬空申酉、甲申旬空午未
func TestDistributeFuXing_旬空阴阳性(t *testing.T) {
	cases := []struct {
		gan, zhi, want int
	}{
		{0, 0, 10}, // 甲子（甲子旬）→ 戌
		{1, 1, 11}, // 乙丑（甲子旬）→ 亥（阴支取后）
		{6, 4, 8},  // 庚辰（甲戌旬）→ 申（iztro 实测锚点）
		{9, 5, 7},  // 癸巳（甲申旬）→ 未（阴支取后）
		{0, 10, 8}, // 甲戌（甲戌旬）→ 申
	}
	for i, c := range cases {
		chart := newAuxChart(0, c.gan, c.zhi)
		distributeFuXing(chart, 1, 0, c.gan, c.zhi)
		containsStar(t, chart.FuXing, palaceAt(0, c.want), "旬空", "旬空%d", i)
	}
}

// TestDistributeFuXing_天伤天使 天伤居交友宫、天使居疾厄宫（以命宫
// 起逆行宫序定位，与 iztro、通行排盘一致）
func TestDistributeFuXing_天伤天使(t *testing.T) {
	for _, mingZhi := range []int{0, 3, 6, 11} {
		chart := newAuxChart(mingZhi, 5, 5)
		distributeFuXing(chart, 1, 0, 5, 5)
		containsStar(t, chart.FuXing, palaceAt(mingZhi, fixIndex(mingZhi-7, 12)), "天伤", "天伤@交友")
		containsStar(t, chart.FuXing, palaceAt(mingZhi, fixIndex(mingZhi-5, 12)), "天使", "天使@疾厄")
	}
}

// ============================================================
// 杂曜
// ============================================================

// TestDistributeSuiXing_Iztro对拍 sample 盘全部 12 颗杂曜安布对拍
func TestDistributeSuiXing_Iztro对拍(t *testing.T) {
	chart := CalculateZiweiChart(2000, 8, 16, 10, "女")
	mingZhiIdx := 3
	wants := map[string]int{
		"红鸾": 11, "天喜": 5,
		"天哭": 2, "天虚": 10,
		"孤辰": 5, "寡宿": 1,
		"劫煞": 5, "亡神": 11, "大耗": 11,
		"天姚": 7,
		"龙池": 8, "凤阁": 6,
	}
	for star, zhi := range wants {
		containsStar(t, chart.SuiXing, palaceAt(mingZhiIdx, zhi), star, "杂曜对拍")
	}
	// 12 颗全部落位且不重复
	seen := map[string]int{}
	for _, stars := range chart.SuiXing {
		for _, s := range stars {
			seen[s]++
		}
	}
	for star, n := range seen {
		if n != 1 {
			t.Errorf("杂曜 %s 落位 %d 次，期望恰 1 次", star, n)
		}
	}
	if len(seen) != len(SuiXingList) {
		t.Errorf("落位杂曜 %d 种，期望 %d（全部安布）", len(seen), len(SuiXingList))
	}
}

// TestDistributeSuiXing_表驱动 年支系/月系杂曜公式锚点
func TestDistributeSuiXing_表驱动(t *testing.T) {
	cases := []struct {
		yearZhi, month int
		star           string
		want           int
	}{
		// 红鸾=卯−年支（逆数至生年），天喜=红鸾+6
		{0, 1, "红鸾", 3}, {7, 1, "红鸾", 8}, {11, 1, "红鸾", 4},
		{0, 1, "天喜", 9}, {7, 1, "天喜", 2},
		// 天哭=午−年支（逆行），天虚=午+年支（顺转）
		{0, 1, "天哭", 6}, {9, 1, "天哭", 9},
		{0, 1, "天虚", 6}, {9, 1, "天虚", 3},
		// 孤辰寡宿（寅卯辰→巳丑、巳午未→申辰、申酉戌→亥未、亥子丑→寅戌）
		{0, 1, "孤辰", 2}, {0, 1, "寡宿", 10},
		{3, 1, "孤辰", 5}, {3, 1, "寡宿", 1},
		{5, 1, "孤辰", 8}, {5, 1, "寡宿", 4},
		{10, 1, "孤辰", 11}, {10, 1, "寡宿", 7},
		// 劫煞三合（申子辰→巳、亥卯未→申、寅午戌→亥、巳酉丑→寅），亡神=劫煞+6
		{0, 1, "劫煞", 5}, {8, 1, "劫煞", 5},
		{1, 1, "劫煞", 2}, {3, 1, "劫煞", 8},
		{2, 1, "劫煞", 11}, {11, 1, "劫煞", 8},
		{0, 1, "亡神", 11}, {2, 1, "亡神", 5},
		// 大耗：年支对宫，阳支顺行一位、阴支逆行一位
		{0, 1, "大耗", 7}, {1, 1, "大耗", 6}, {5, 1, "大耗", 10}, {4, 1, "大耗", 11},
		// 龙池=辰+年支（顺），凤阁=戌−年支（逆）
		{0, 1, "龙池", 4}, {9, 1, "龙池", 1},
		{0, 1, "凤阁", 10}, {9, 1, "凤阁", 1},
		// 天姚=丑+月−1（顺数生月）：正月丑、七月未、腊月子
		{0, 1, "天姚", 1}, {0, 7, "天姚", 7}, {0, 12, "天姚", 0},
	}
	for i, c := range cases {
		chart := newAuxChart(0, 0, c.yearZhi)
		distributeSuiXing(chart, c.month, c.yearZhi)
		containsStar(t, chart.SuiXing, palaceAt(0, c.want), c.star, "杂曜表%d", i)
	}
}

// ============================================================
// 主星亮度（全书派）
// ============================================================

// TestZhuXingBrightness_全书派锚点 学理锚点与 iztro 对拍锚点
func TestZhuXingBrightness_全书派锚点(t *testing.T) {
	anchors := []struct {
		star, zhi, want string
	}{
		{"太阳", "子", "陷"}, {"太阳", "午", "旺"}, {"太阳", "亥", "陷"}, // 日月反背/丽中天/亥地（v4.0-13d QS 与 iztro 一致）
		{"太阴", "亥", "庙"}, {"太阴", "卯", "陷"},
		{"紫微", "寅", "旺"}, {"天机", "丑", "陷"},
		{"天府", "寅", "庙"}, {"天府", "亥", "得"}, {"廉贞", "亥", "陷"},
		{"破军", "子", "庙"}, {"贪狼", "辰", "庙"},
	}
	for _, a := range anchors {
		if got := brightnessOf(a.star, a.zhi); got != a.want {
			t.Errorf("%s@%s 亮度 got %s，want %s", a.star, a.zhi, got, a.want)
		}
	}
}

// TestZhuXingBrightness_盘上样本 sample 盘 14 主星亮度（iztro 对拍）
func TestZhuXingBrightness_盘上样本(t *testing.T) {
	chart := CalculateZiweiChart(2000, 8, 16, 10, "女")
	wants := map[string]string{
		"紫微": "旺", "天府": "庙", // 同宫于寅
		"天机": "陷", "太阳": "陷", "武曲": "庙", "天同": "平",
		"廉贞": "平", "太阴": "陷", "贪狼": "庙", "巨门": "旺",
		"天相": "庙", "天梁": "旺", "七杀": "庙", "破军": "庙",
	}
	if got := len(chart.Brightness); got != 14 {
		t.Errorf("Brightness 条目数 = %d，期望 14", got)
	}
	for star, want := range wants {
		if got := chart.Brightness[star]; got != want {
			t.Errorf("亮度[%s] got %s，want %s", star, got, want)
		}
	}
}

// TestGenerateAnalysis_新段落 分析文本须包含主星亮度与杂曜段落
func TestGenerateAnalysis_新段落(t *testing.T) {
	chart := CalculateZiweiChart(2000, 8, 16, 10, "女")
	analysis := generateAnalysis(chart)
	for _, sec := range []string{"【主星亮度】", "【杂曜】", "紫微（旺）", "天梁（旺）"} {
		if !strings.Contains(analysis, sec) {
			t.Errorf("分析文本缺少 %q", sec)
		}
	}
}

// TestGetAllXingYao_含命宫杂曜 Xingyao 汇总应并入命宫辅星与命宫杂曜
func TestGetAllXingYao_含命宫杂曜(t *testing.T) {
	chart := CalculateZiweiChart(2000, 8, 16, 10, "女")
	x := getAllXingYao(chart)
	// 命宫在卯：主星 太阴，辅星 铃星
	for _, star := range []string{"太阴", "铃星"} {
		if !strings.Contains(x, star) {
			t.Errorf("命宫 Xingyao 缺少主辅星 %s", star)
		}
	}
	if strings.Contains(x, "火星") {
		t.Errorf("命宫 Xingyao 不应混入他宫星曜（火星@官禄）")
	}
}
