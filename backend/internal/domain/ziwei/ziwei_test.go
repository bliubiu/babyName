package ziwei

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// TestCase 定义紫微斗数测试用例
type TestCase struct {
	Name        string
	Year        int
	Month       int
	Day         int
	Hour        int // 24小时制，23代表晚子时
	Gender      string
	// 四柱
	ExpectedNianZhu string
	ExpectedYueZhu  string
	ExpectedRiZhu   string
	ExpectedShiZhu  string
	// 命身宫（干支）
	ExpectedMingGong string
	ExpectedShenGong string
	// 五行局
	ExpectedFiveElements string
	// 命主身主
	ExpectedSoul string // 命主
	ExpectedBody string // 身主
	// 四化（年干）
	ExpectedSiHua map[string]string // 禄权科忌 -> 星名
	// 大限
	ExpectedDaXian map[string]DaXianExpected
	// 宫位主星（部分宫位）
	ExpectedZhuXing map[string]string
	// 宫位辅星（部分宫位）
	ExpectedFuXing map[string][]string
}

// DaXianExpected 大限期望值
type DaXianExpected struct {
	Range       [2]int
	HeavenlyStem string
	EarthlyBranch string
}

func TestAnchors(t *testing.T) {
	testCases := getTestCases()

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			chart := CalculateZiweiChart(tc.Year, tc.Month, tc.Day, tc.Hour, tc.Gender)

			// 验证四柱
			if chart.NianZhu != tc.ExpectedNianZhu {
				t.Errorf("四柱年柱: got %s, want %s", chart.NianZhu, tc.ExpectedNianZhu)
			}
			if chart.YueZhu != tc.ExpectedYueZhu {
				t.Errorf("四柱月柱: got %s, want %s", chart.YueZhu, tc.ExpectedYueZhu)
			}
			if chart.RiZhu != tc.ExpectedRiZhu {
				t.Errorf("四柱日柱: got %s, want %s", chart.RiZhu, tc.ExpectedRiZhu)
			}
			if chart.ShiZhu != tc.ExpectedShiZhu {
				t.Errorf("四柱时柱: got %s, want %s", chart.ShiZhu, tc.ExpectedShiZhu)
			}

			// 验证命宫（干支）
			if chart.MingGong != tc.ExpectedMingGong {
				t.Errorf("命宫: got %s, want %s", chart.MingGong, tc.ExpectedMingGong)
			}
			// 验证身宫（干支）
			if chart.ShenGong != tc.ExpectedShenGong {
				t.Errorf("身宫: got %s, want %s", chart.ShenGong, tc.ExpectedShenGong)
			}

			// 验证五行局
			if chart.FiveElements != tc.ExpectedFiveElements {
				t.Errorf("五行局: got %s, want %s", chart.FiveElements, tc.ExpectedFiveElements)
			}

			// 验证命主
			if chart.Soul != tc.ExpectedSoul {
				t.Errorf("命主: got %s, want %s", chart.Soul, tc.ExpectedSoul)
			}
			// 验证身主
			if chart.Body != tc.ExpectedBody {
				t.Errorf("身主: got %s, want %s", chart.Body, tc.ExpectedBody)
			}

			// 验证四化
			for _, hua := range []string{"禄", "权", "科", "忌"} {
				if got, ok := chart.SiHua[hua]; !ok {
					t.Errorf("四化[%s] 缺失", hua)
				} else if want, ok := tc.ExpectedSiHua[hua]; !ok {
					// expected doesn't have this hua, skip
				} else if got != want {
					t.Errorf("四化[%s]: got %s, want %s", hua, got, want)
				}
			}

			// 验证主星分布
			for gong, expectedStar := range tc.ExpectedZhuXing {
				got, ok := chart.ZhuXing[gong]
				if !ok {
					t.Errorf("宫位[%s] 主星缺失，期望: %s", gong, expectedStar)
				} else if got != expectedStar {
					t.Errorf("宫位[%s] 主星: got %s, want %s", gong, got, expectedStar)
				}
			}

			// 验证辅星分布
			for gong, expectedStars := range tc.ExpectedFuXing {
				got, ok := chart.FuXing[gong]
				if !ok {
					t.Errorf("宫位[%s] 辅星缺失，期望: %v", gong, expectedStars)
				} else {
					// 排序后比较
					sort.Strings(got)
					sort.Strings(expectedStars)
					if !equalStringSlices(got, expectedStars) {
						t.Errorf("宫位[%s] 辅星: got %v, want %v", gong, got, expectedStars)
					}
				}
			}

			// 验证大限
			for gong, expectedDX := range tc.ExpectedDaXian {
				dx, ok := chart.DaXian[gong]
				if !ok {
					t.Errorf("大限宫位[%s] 缺失", gong)
				} else {
					if dx.Range != expectedDX.Range {
						t.Errorf("大限宫位[%s] Range: got %v, want %v", gong, dx.Range, expectedDX.Range)
					}
					if dx.HeavenlyStem != expectedDX.HeavenlyStem {
						t.Errorf("大限宫位[%s] HeavenlyStem: got %s, want %s", gong, dx.HeavenlyStem, expectedDX.HeavenlyStem)
					}
					if dx.EarthlyBranch != expectedDX.EarthlyBranch {
						t.Errorf("大限宫位[%s] EarthlyBranch: got %s, want %s", gong, dx.EarthlyBranch, expectedDX.EarthlyBranch)
					}
				}
			}
		})
	}
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// getTestCases 返回9个anchor测试用例
func getTestCases() []TestCase {
	return []TestCase{
		// Case 1: 2000-8-16 时4h 女 → 逆
		{
			Name: "case1_2000_08_16_04h_female",
			Year: 2000, Month: 8, Day: 16, Hour: 4, Gender: "女",
			ExpectedNianZhu: "庚辰",
			ExpectedYueZhu:  "甲申",
			ExpectedRiZhu:   "丙午",
			ExpectedShiZhu:  "庚寅",
			ExpectedMingGong: "壬午",
			ExpectedShenGong: "丙戌",
			ExpectedFiveElements: "木三局",
			ExpectedSoul: "破军",
			ExpectedBody: "文昌",
			ExpectedSiHua: map[string]string{
				"禄": "太阳",
				"权": "武曲",
				"科": "太阴",
				"忌": "天同",
			},
			ExpectedDaXian: map[string]DaXianExpected{
				"命宫": {Range: [2]int{3, 12}, HeavenlyStem: "壬", EarthlyBranch: "午"},
				"兄弟宫": {Range: [2]int{13, 22}, HeavenlyStem: "辛", EarthlyBranch: "巳"},
				"夫妻宫": {Range: [2]int{23, 32}, HeavenlyStem: "庚", EarthlyBranch: "辰"},
				"子女宫": {Range: [2]int{33, 42}, HeavenlyStem: "己", EarthlyBranch: "卯"},
				"财帛宫": {Range: [2]int{43, 52}, HeavenlyStem: "戊", EarthlyBranch: "寅"},
				"疾厄宫": {Range: [2]int{53, 62}, HeavenlyStem: "己", EarthlyBranch: "丑"},
			},
		},
		// Case 2: 2000-8-16 时4h 男 → 顺
		{
			Name: "case2_2000_08_16_04h_male",
			Year: 2000, Month: 8, Day: 16, Hour: 4, Gender: "男",
			ExpectedNianZhu: "庚辰",
			ExpectedYueZhu:  "甲申",
			ExpectedRiZhu:   "丙午",
			ExpectedShiZhu:  "庚寅",
			ExpectedMingGong: "壬午",
			ExpectedShenGong: "丙戌",
			ExpectedFiveElements: "木三局",
			ExpectedSoul: "破军",
			ExpectedBody: "文昌",
			ExpectedSiHua: map[string]string{
				"禄": "太阳",
				"权": "武曲",
				"科": "太阴",
				"忌": "天同",
			},
			ExpectedDaXian: map[string]DaXianExpected{
				"命宫": {Range: [2]int{3, 12}, HeavenlyStem: "壬", EarthlyBranch: "午"},
				"兄弟宫": {Range: [2]int{113, 122}, HeavenlyStem: "辛", EarthlyBranch: "巳"},
				"夫妻宫": {Range: [2]int{103, 112}, HeavenlyStem: "庚", EarthlyBranch: "辰"},
				"子女宫": {Range: [2]int{93, 102}, HeavenlyStem: "己", EarthlyBranch: "卯"},
				"财帛宫": {Range: [2]int{83, 92}, HeavenlyStem: "戊", EarthlyBranch: "寅"},
				"疾厄宫": {Range: [2]int{73, 82}, HeavenlyStem: "己", EarthlyBranch: "丑"},
			},
		},
		// Case 3: 2023-1-22 时10h 女
		{
			Name: "case3_2023_01_22_10h_female",
			Year: 2023, Month: 1, Day: 22, Hour: 10, Gender: "女",
			ExpectedNianZhu: "癸卯",
			ExpectedYueZhu:  "甲寅",
			ExpectedRiZhu:   "庚辰",
			ExpectedShiZhu:  "辛巳",
			ExpectedMingGong: "辛酉",
			ExpectedShenGong: "己未",
			ExpectedFiveElements: "木三局",
			ExpectedSoul: "文曲",
			ExpectedBody: "天同",
			ExpectedSiHua: map[string]string{
				"禄": "破军",
				"权": "巨门",
				"科": "太阴",
				"忌": "贪狼",
			},
		},
		// Case 4: 2023-2-19 时23h 女（晚子时=子时+日进位）
		{
			Name: "case4_2023_02_19_23h_female",
			Year: 2023, Month: 2, Day: 19, Hour: 23, Gender: "女",
			ExpectedNianZhu: "癸卯",
			ExpectedYueZhu:  "甲寅",
			ExpectedRiZhu:   "己酉",
			ExpectedShiZhu:  "甲子",
			ExpectedMingGong: "甲寅",
			ExpectedShenGong: "甲寅",
			ExpectedFiveElements: "水二局",
			ExpectedSoul: "禄存",
			ExpectedBody: "天同",
			ExpectedSiHua: map[string]string{
				"禄": "破军",
				"权": "巨门",
				"科": "太阴",
				"忌": "贪狼",
			},
		},
		// Case 5: 1999-6-27 时6h 男
		{
			Name: "case5_1999_06_27_06h_male",
			Year: 1999, Month: 6, Day: 27, Hour: 6, Gender: "男",
			ExpectedNianZhu: "己卯",
			ExpectedYueZhu:  "庚午",
			ExpectedRiZhu:   "庚戌",
			ExpectedShiZhu:  "己卯",
			ExpectedMingGong: "丁卯",
			ExpectedShenGong: "癸酉",
			ExpectedFiveElements: "火六局",
			ExpectedSoul: "文曲",
			ExpectedBody: "天同",
			ExpectedSiHua: map[string]string{
				"禄": "武曲",
				"权": "贪狼",
				"科": "天梁",
				"忌": "文曲",
			},
		},
		// Case 6: 1985-5-13 时12h 女 → 顺行（阴女）
		{
			Name: "case6_1985_05_13_12h_female",
			Year: 1985, Month: 5, Day: 13, Hour: 12, Gender: "女",
			ExpectedNianZhu: "乙丑",
			ExpectedYueZhu:  "庚辰",
			ExpectedRiZhu:   "壬子",
			ExpectedShiZhu:  "丙午",
			ExpectedMingGong: "丙戌",
			ExpectedShenGong: "丙戌",
			ExpectedFiveElements: "土五局",
			ExpectedSoul: "禄存",
			ExpectedBody: "天相",
			ExpectedSiHua: map[string]string{
				"禄": "天机",
				"权": "天梁",
				"科": "紫微",
				"忌": "太阴",
			},
		},
		// Case 7: 1990-4-6 时8h 男
		{
			Name: "case7_1990_04_06_08h_male",
			Year: 1990, Month: 4, Day: 6, Hour: 8, Gender: "男",
			ExpectedNianZhu: "庚午",
			ExpectedYueZhu:  "庚辰",
			ExpectedRiZhu:   "辛丑",
			ExpectedShiZhu:  "壬辰",
			ExpectedMingGong: "戊子",
			ExpectedShenGong: "甲申",
			ExpectedFiveElements: "火六局",
			ExpectedSoul: "贪狼",
			ExpectedBody: "火星",
			ExpectedSiHua: map[string]string{
				"禄": "太阳",
				"权": "武曲",
				"科": "太阴",
				"忌": "天同",
			},
		},
		// Case 8: 2012-5-6 时2h 女
		{
			Name: "case8_2012_05_06_02h_female",
			Year: 2012, Month: 5, Day: 6, Hour: 2, Gender: "女",
			ExpectedNianZhu: "壬辰",
			ExpectedYueZhu:  "乙巳",
			ExpectedRiZhu:   "丁卯",
			ExpectedShiZhu:  "辛丑",
			ExpectedMingGong: "甲辰",
			ExpectedShenGong: "丙午",
			ExpectedFiveElements: "火六局",
			ExpectedSoul: "廉贞",
			ExpectedBody: "文昌",
			ExpectedSiHua: map[string]string{
				"禄": "天梁",
				"权": "紫微",
				"科": "左辅",
				"忌": "武曲",
			},
		},
		// Case 9: 1995-10-12 时20h 男 → 逆行（阴男）
		{
			Name: "case9_1995_10_12_20h_male",
			Year: 1995, Month: 10, Day: 12, Hour: 20, Gender: "男",
			ExpectedNianZhu: "乙亥",
			ExpectedYueZhu:  "丙戌",
			ExpectedRiZhu:   "丙子",
			ExpectedShiZhu:  "戊戌",
			ExpectedMingGong: "戊子",
			ExpectedShenGong: "甲申",
			ExpectedFiveElements: "火六局",
			ExpectedSoul: "贪狼",
			ExpectedBody: "天机",
			ExpectedSiHua: map[string]string{
				"禄": "天机",
				"权": "天梁",
				"科": "紫微",
				"忌": "太阴",
			},
			ExpectedDaXian: map[string]DaXianExpected{
				"命宫": {Range: [2]int{6, 15}, HeavenlyStem: "戊", EarthlyBranch: "子"},
				"父母宫": {Range: [2]int{116, 125}, HeavenlyStem: "己", EarthlyBranch: "丑"},
				"福德宫": {Range: [2]int{106, 115}, HeavenlyStem: "戊", EarthlyBranch: "寅"},
				"田宅宫": {Range: [2]int{96, 105}, HeavenlyStem: "己", EarthlyBranch: "卯"},
				"官禄宫": {Range: [2]int{86, 95}, HeavenlyStem: "庚", EarthlyBranch: "辰"},
				"交友宫": {Range: [2]int{76, 85}, HeavenlyStem: "辛", EarthlyBranch: "巳"},
			},
		},
	}
}

// TestAnalyzeZiwei 测试 AnalyzeZiwei 接口返回的 ZiweiAnalysis
func TestAnalyzeZiwei(t *testing.T) {
	// Case 1 简单验证
	result := AnalyzeZiwei(2000, 8, 16, 4, "女")
	if result == nil {
		t.Fatal("AnalyzeZiwei returned nil")
	}

	// 命宫应该是干支格式 "壬午"，不是宫名 "命宫"
	if result.Gongwei == "命宫" {
		t.Errorf("Gongwei should be 干支 format like '壬午', got '%s'", result.Gongwei)
	}

	// 四化应该有值
	if result.Lu == "" {
		t.Error("四化禄 should not be empty")
	}
	if result.Quan == "" {
		t.Error("四化权 should not be empty")
	}
	if result.Ke == "" {
		t.Error("四化科 should not be empty")
	}
	if result.Ji == "" {
		t.Error("四化忌 should not be empty")
	}
}

// TestNewFields 新增字段在旧实现中不存在，编译即红
func TestNewFields(t *testing.T) {
	chart := CalculateZiweiChart(2000, 8, 16, 4, "女")

	// FiveElements 字段（旧实现没有）
	if chart.FiveElements == "" {
		t.Error("FiveElements field should exist and not be empty")
	}

	// Soul 字段（命主）
	if chart.Soul == "" {
		t.Error("Soul field (命主) should exist and not be empty")
	}

	// Body 字段（身主）
	if chart.Body == "" {
		t.Error("Body field (身主) should exist and not be empty")
	}

	// DaXian 字段（大限）
	if chart.DaXian == nil {
		t.Error("DaXian field should exist and not be nil")
	}

	runes := []rune(chart.MingGong)
	// MingGong 应该是干支格式（2个unicode字符）
	if len(runes) != 2 {
		t.Errorf("MingGong should be 2-char gan-zhi, got '%s'", chart.MingGong)
	}
	gan := string(runes[0])
	zhi := string(runes[1])
	validGan := false
	for _, g := range TianganNames {
		if g == gan {
			validGan = true
			break
		}
	}
	if !validGan {
		t.Errorf("MingGong first char should be heavenly stem, got '%s'", gan)
	}
	validZhi := false
	for _, z := range DizhiNames {
		if z == zhi {
			validZhi = true
			break
		}
	}
	if !validZhi {
		t.Errorf("MingGong second char should be earthly branch, got '%s'", zhi)
	}
}

// TestCaseStr 辅助：格式化测试结果用于调试
func formatChartDebug(chart *ZiweiChart) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("四柱: %s %s %s %s\n", chart.NianZhu, chart.YueZhu, chart.RiZhu, chart.ShiZhu))
	sb.WriteString(fmt.Sprintf("命宫: %s\n", chart.MingGong))
	sb.WriteString(fmt.Sprintf("身宫: %s\n", chart.ShenGong))
	sb.WriteString(fmt.Sprintf("五行局: %s\n", chart.FiveElements))
	sb.WriteString(fmt.Sprintf("命主: %s\n", chart.Soul))
	sb.WriteString(fmt.Sprintf("身主: %s\n", chart.Body))
	sb.WriteString("主星:\n")
	for gong, star := range chart.ZhuXing {
		sb.WriteString(fmt.Sprintf("  %s: %s\n", gong, star))
	}
	return sb.String()
}
