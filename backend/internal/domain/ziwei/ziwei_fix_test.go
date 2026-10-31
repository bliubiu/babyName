package ziwei

import "testing"

// 本文件固化 docs/29 A5 及命宫主星 key 修复的回归：
// 禄存/天魁天钺查表、五行局纳音法、紫微借表法定位、
// 紫微系逆布/天府系顺布与宫名按地支定位。

// TestLuCunTable_Anchors 禄存口诀锚点：甲寅 乙卯 丙巳 丁午 戊巳 己午 庚申 辛酉 壬亥 癸子
func TestLuCunTable_Anchors(t *testing.T) {
	anchors := []struct {
		gan    string
		zhi    string
		zhiIdx int
	}{
		{"甲", "寅", 2}, {"乙", "卯", 3}, {"丙", "巳", 5}, {"丁", "午", 6},
		{"戊", "巳", 5}, {"己", "午", 6}, {"庚", "申", 8}, {"辛", "酉", 9},
		{"壬", "亥", 11}, {"癸", "子", 0},
	}
	for i, a := range anchors {
		if got := LuCunTable[i]; got != a.zhiIdx {
			t.Errorf("%s干禄存 = 地支%d(%s)，期望 %d(%s)", a.gan, got, DizhiNames[got], a.zhiIdx, a.zhi)
		}
	}
}

// TestTianKuiTianYueTable_Anchors 天魁天钺口诀锚点（甲戊庚牛羊、乙己鼠猴乡、
// 丙丁猪鸡位、壬癸兔蛇藏、六辛逢马虎）
func TestTianKuiTianYueTable_Anchors(t *testing.T) {
	anchors := [][3]string{
		{"甲", "丑", "未"}, {"乙", "子", "申"}, {"丙", "亥", "酉"}, {"丁", "亥", "酉"},
		{"戊", "丑", "未"}, {"己", "子", "申"}, {"庚", "丑", "未"}, {"辛", "午", "寅"},
		{"壬", "卯", "巳"}, {"癸", "卯", "巳"},
	}
	for i, a := range anchors {
		kui, yue := TianKuiTianYueTable[i][0], TianKuiTianYueTable[i][1]
		if DizhiNames[kui] != a[1] || DizhiNames[yue] != a[2] {
			t.Errorf("%s干天魁天钺 = %s/%s，期望 %s/%s", a[0], DizhiNames[kui], DizhiNames[yue], a[1], a[2])
		}
	}
}

// TestCalculateFiveElementsClass_Nayin 五行局由命宫干支纳音定局
func TestCalculateFiveElementsClass_Nayin(t *testing.T) {
	anchors := []struct {
		mingGong string
		want     string
	}{
		{"甲子", "金四局"}, // 海中金
		{"丙寅", "火六局"}, // 炉中火
		{"壬午", "木三局"}, // 杨柳木
		{"庚辰", "金四局"}, // 白蜡金
		{"戊子", "火六局"}, // 霹雳火
		{"丙戌", "土五局"}, // 屋上土
		{"辛酉", "木三局"}, // 石榴木
		{"甲寅", "水二局"}, // 大溪水
		{"辛丑", "土五局"}, // 壁上土
		{"癸未", "木三局"}, // 杨柳木
	}
	for _, a := range anchors {
		got, ok := calculateFiveElementsClass(a.mingGong)
		if !ok || got != a.want {
			t.Errorf("命宫 %s：五行局 got %s(ok=%v)，want %s", a.mingGong, got, ok, a.want)
		}
	}
}

// TestGetZiweiPosition_Anchors 紫微借表法锚点（与《紫微斗数全书》安星表核对）
func TestGetZiweiPosition_Anchors(t *testing.T) {
	anchors := []struct {
		day     int
		jue     string
		wantZhi string
	}{
		// 水二局：初一丑、初二寅、初三寅、初四卯
		{1, "水二局", "丑"}, {2, "水二局", "寅"}, {3, "水二局", "寅"}, {4, "水二局", "卯"},
		// 木三局：初一辰、初二丑、初三寅、初四巳、初五寅、初六卯
		{1, "木三局", "辰"}, {2, "木三局", "丑"}, {3, "木三局", "寅"}, {4, "木三局", "巳"},
		{5, "木三局", "寅"}, {6, "木三局", "卯"},
		// 金四局：初一亥、初二辰、初三丑、初四寅、初五子、初六巳、初七寅、初八卯
		{1, "金四局", "亥"}, {2, "金四局", "辰"}, {3, "金四局", "丑"}, {4, "金四局", "寅"},
		{5, "金四局", "子"}, {6, "金四局", "巳"}, {7, "金四局", "寅"}, {8, "金四局", "卯"},
		// 土五局：初一午、初二亥、初三辰、初四丑、初五寅、初六未、初七子
		{1, "土五局", "午"}, {2, "土五局", "亥"}, {3, "土五局", "辰"}, {4, "土五局", "丑"},
		{5, "土五局", "寅"}, {6, "土五局", "未"}, {7, "土五局", "子"},
		// 火六局：初一酉、初二午、初三亥、初四辰、初五丑、初六寅、初七戌、初八未
		{1, "火六局", "酉"}, {2, "火六局", "午"}, {3, "火六局", "亥"}, {4, "火六局", "辰"},
		{5, "火六局", "丑"}, {6, "火六局", "寅"}, {7, "火六局", "戌"}, {8, "火六局", "未"},
	}
	for _, a := range anchors {
		got := getZiweiPosition(a.day, a.jue, 1)
		if DizhiNames[got] != a.wantZhi {
			t.Errorf("%s局生日%d：紫微 = %s，期望 %s", a.jue, a.day, DizhiNames[got], a.wantZhi)
		}
	}
}

// TestDistributeZhuXing_ZiweiZiConfig 紫微在子全盘：紫微独坐对照贪狼、
// 廉贞天府在辰、太阳天梁在酉、武曲天相在申、天同巨门在未——
// 同宫双星两颗都必须落位（历史缺陷：单星 map 覆盖丢失紫微系星）
func TestDistributeZhuXing_ZiweiZiConfig(t *testing.T) {
	chart := &ZiweiChart{
		ZhuXing:      make(map[string][]string),
		MingGong:     "壬午", // 命宫在午(6)
		FiveElements: "金四局",
	}
	// 金四局初五 → 紫微在子
	distributeZhuXing(chart, 5, 1)

	// 宫名按地支定位：命宫在午 → 地支 z 的宫名 = Gongs[(6-z) mod 12]
	palaceAt := func(zhi int) string { return Gongs[(6-zhi+12)%12] }

	// 14 主星地支 → 期望星曜（宫内紫微系在前、天府系在后）
	wantByZhi := map[int][]string{
		0: {"紫微"}, 11: {"天机"},
		4: {"廉贞", "天府"}, 5: {"太阴"}, 6: {"贪狼"},
		7: {"天同", "巨门"}, 8: {"武曲", "天相"}, 9: {"太阳", "天梁"},
		10: {"七杀"}, 2: {"破军"},
	}
	for zhi, want := range wantByZhi {
		got := chart.ZhuXing[palaceAt(zhi)]
		if !equalStringSlices(got, want) {
			t.Errorf("%s宫(%s)：got %v，want %v", palaceAt(zhi), DizhiNames[zhi], got, want)
		}
	}

	// 主星总数必须为 14（覆盖缺陷下单星 map 只剩 10 颗）
	total := 0
	for _, stars := range chart.ZhuXing {
		total += len(stars)
	}
	if total != 14 {
		t.Errorf("主星总数 = %d，期望 14（同宫双星不得覆盖）", total)
	}
}

// TestDistributeZhuXing_十四星唯一性 回归：任意五行局 × 全生日，14 主星
// 必须恰好各落位一次（历史缺陷：同宫双星被 map 覆盖，每盘丢 3~4 颗）
func TestDistributeZhuXing_十四星唯一性(t *testing.T) {
	wantStars := []string{"紫微", "天机", "太阳", "武曲", "天同", "廉贞",
		"天府", "太阴", "贪狼", "巨门", "天相", "天梁", "七杀", "破军"}

	for _, jue := range []string{"水二局", "木三局", "金四局", "土五局", "火六局"} {
		for day := 1; day <= 30; day++ {
			chart := &ZiweiChart{
				ZhuXing:      make(map[string][]string),
				MingGong:     "甲寅",
				FiveElements: jue,
			}
			distributeZhuXing(chart, day, 1)

			seen := map[string]int{}
			for _, stars := range chart.ZhuXing {
				for _, star := range stars {
					seen[star]++
				}
			}
			if len(seen) != len(wantStars) {
				t.Errorf("%s局生日%d：落位主星 %d 种（期望 14）：%v", jue, day, len(seen), seen)
				continue
			}
			for _, star := range wantStars {
				if seen[star] != 1 {
					t.Errorf("%s局生日%d：主星 %s 落位 %d 次（期望恰 1 次）", jue, day, star, seen[star])
				}
			}
		}
	}
}

// TestDistributeFuXing_PalaceMapping 辅星按地支落宫：年干甲（禄存寅、
// 擎羊卯、陀罗丑、天魁丑、天钺未），命宫在午
func TestDistributeFuXing_PalaceMapping(t *testing.T) {
	chart := &ZiweiChart{FuXing: make(map[string][]string), MingGong: "壬午"}
	distributeFuXing(chart, 1, 17, 0, 0, 0) // 正月、子时、甲年、子年

	palaceAt := func(zhi int) string { return Gongs[(6-zhi+12)%12] }
	contains := func(gong, star string) bool {
		for _, s := range chart.FuXing[gong] {
			if s == star {
				return true
			}
		}
		return false
	}
	// 甲年：禄存寅(2)、擎羊卯(3)、陀罗丑(1)、天魁丑(1)、天钺未(7)
	if !contains(palaceAt(2), "禄存") {
		t.Errorf("禄存应在 %s（寅）", palaceAt(2))
	}
	if !contains(palaceAt(3), "擎羊") {
		t.Errorf("擎羊应在 %s（卯）", palaceAt(3))
	}
	if !contains(palaceAt(1), "陀罗") {
		t.Errorf("陀罗应在 %s（丑）", palaceAt(1))
	}
	if !contains(palaceAt(1), "天魁") {
		t.Errorf("甲年天魁应在 %s（丑）", palaceAt(1))
	}
	if !contains(palaceAt(7), "天钺") {
		t.Errorf("甲年天钺应在 %s（未）", palaceAt(7))
	}
	// 左辅：辰起正月顺行 → 正月在辰(4)；右弼：戌起正月逆行 → 正月在戌(10)
	if !contains(palaceAt(4), "左辅") {
		t.Errorf("正月左辅应在 %s（辰）", palaceAt(4))
	}
	if !contains(palaceAt(10), "右弼") {
		t.Errorf("正月右弼应在 %s（戌）", palaceAt(10))
	}
	// 文昌：戌起子时逆行 → 子时在戌(10)；文曲：辰起子时顺行 → 子时在辰(4)
	if !contains(palaceAt(10), "文昌") {
		t.Errorf("子时文昌应在 %s（戌）", palaceAt(10))
	}
	if !contains(palaceAt(4), "文曲") {
		t.Errorf("子时文曲应在 %s（辰）", palaceAt(4))
	}
}

// TestAnalyzeZiwei_MingGongStar 命宫主星不得恒为「无主星」
// （原实现用干支串查宫名键，恒落空）
func TestAnalyzeZiwei_MingGongStar(t *testing.T) {
	result := AnalyzeZiwei(2000, 8, 16, 4, "女")
	if result == nil {
		t.Fatal("AnalyzeZiwei 返回 nil")
	}
	if result.Tianzhu == "" {
		t.Fatal("命宫主星为空")
	}
	if result.Tianzhu == "无主星" {
		t.Fatal("命宫主星恒为「无主星」：ZhuXing 键口径断裂未修复")
	}
}
