package bazi

import (
	"testing"
)

// --- 天干常量测试 ---

func TestTiangan_Length(t *testing.T) {
	if len(Tiangan) != 10 {
		t.Errorf("Tiangan length = %d, want 10", len(Tiangan))
	}
}

func TestTiangan_Order(t *testing.T) {
	expected := []string{"甲", "乙", "丙", "丁", "戊", "己", "庚", "辛", "壬", "癸"}
	for i, v := range expected {
		if Tiangan[i] != v {
			t.Errorf("Tiangan[%d] = %q, want %q", i, Tiangan[i], v)
		}
	}
}

// --- 地支常量测试 ---

func TestDizhi_Length(t *testing.T) {
	if len(Dizhi) != 12 {
		t.Errorf("Dizhi length = %d, want 12", len(Dizhi))
	}
}

func TestDizhi_Order(t *testing.T) {
	expected := []string{"子", "丑", "寅", "卯", "辰", "巳", "午", "未", "申", "酉", "戌", "亥"}
	for i, v := range expected {
		if Dizhi[i] != v {
			t.Errorf("Dizhi[%d] = %q, want %q", i, Dizhi[i], v)
		}
	}
}

// --- 五行对应表测试 ---

func TestWuxingMap_AllTianganCovered(t *testing.T) {
	for _, tg := range Tiangan {
		if _, ok := WuxingMap[tg]; !ok {
			t.Errorf("WuxingMap missing entry for 天干 %q", tg)
		}
	}
}

func TestWuxingMap_CorrectValues(t *testing.T) {
	tests := []struct {
		gan   string
		wuxing string
	}{
		{"甲", "木"}, {"乙", "木"},
		{"丙", "火"}, {"丁", "火"},
		{"戊", "土"}, {"己", "土"},
		{"庚", "金"}, {"辛", "金"},
		{"壬", "水"}, {"癸", "水"},
	}
	for _, tc := range tests {
		if got := WuxingMap[tc.gan]; got != tc.wuxing {
			t.Errorf("WuxingMap[%q] = %q, want %q", tc.gan, got, tc.wuxing)
		}
	}
}

func TestDizhiWuxingMap_AllDizhiCovered(t *testing.T) {
	for _, dz := range Dizhi {
		if _, ok := DizhiWuxingMap[dz]; !ok {
			t.Errorf("DizhiWuxingMap missing entry for 地支 %q", dz)
		}
	}
}

func TestDizhiWuxingMap_CorrectValues(t *testing.T) {
	tests := []struct {
		zhi    string
		wuxing string
	}{
		{"子", "水"}, {"丑", "土"}, {"寅", "木"}, {"卯", "木"},
		{"辰", "土"}, {"巳", "火"}, {"午", "火"}, {"未", "土"},
		{"申", "金"}, {"酉", "金"}, {"戌", "土"}, {"亥", "水"},
	}
	for _, tc := range tests {
		if got := DizhiWuxingMap[tc.zhi]; got != tc.wuxing {
			t.Errorf("DizhiWuxingMap[%q] = %q, want %q", tc.zhi, got, tc.wuxing)
		}
	}
}

// --- 时辰对应表测试 ---

func TestShichenMap_Length(t *testing.T) {
	if len(ShichenMap) != 12 {
		t.Errorf("ShichenMap length = %d, want 12", len(ShichenMap))
	}
}

func TestShichenMap_TimeRanges(t *testing.T) {
	tests := []struct {
		name  string
		start int
		end   int
	}{
		{"子时", 23, 1},
		{"丑时", 1, 3},
		{"寅时", 3, 5},
		{"卯时", 5, 7},
		{"辰时", 7, 9},
		{"巳时", 9, 11},
		{"午时", 11, 13},
		{"未时", 13, 15},
		{"申时", 15, 17},
		{"酉时", 17, 19},
		{"戌时", 19, 21},
		{"亥时", 21, 23},
	}
	for _, tc := range tests {
		got, ok := ShichenMap[tc.name]
		if !ok {
			t.Errorf("ShichenMap missing entry for %q", tc.name)
			continue
		}
		if got[0] != tc.start || got[1] != tc.end {
			t.Errorf("ShichenMap[%q] = [%d,%d], want [%d,%d]", tc.name, got[0], got[1], tc.start, tc.end)
		}
	}
}

func TestShichenTianganMap_Length(t *testing.T) {
	if len(ShichenTianganMap) != 12 {
		t.Errorf("ShichenTianganMap length = %d, want 12", len(ShichenTianganMap))
	}
}

// --- 纳音五行表测试 ---

func TestNayinMap_Length(t *testing.T) {
	// 60甲子应该全部覆盖
	if len(NayinMap) != 60 {
		t.Errorf("NayinMap length = %d, want 60", len(NayinMap))
	}
}

func TestNayinMap_KeyFormat(t *testing.T) {
	for key := range NayinMap {
		if len([]rune(key)) != 2 {
			t.Errorf("NayinMap key %q should be 2 characters", key)
		}
	}
}

func TestNayinMap_SampleValues(t *testing.T) {
	tests := []struct {
		key    string
		want   string
	}{
		{"甲子", "海中金"},
		{"丙寅", "炉中火"},
		{"戊辰", "大林木"},
		{"壬申", "剑锋金"},
		{"庚子", "壁上土"},
		{"壬午", "杨柳木"},
		{"癸亥", "大海水"},
	}
	for _, tc := range tests {
		if got := NayinMap[tc.key]; got != tc.want {
			t.Errorf("NayinMap[%q] = %q, want %q", tc.key, got, tc.want)
		}
	}
}

func TestNayinWuxingMap_AllNayinCovered(t *testing.T) {
	nayinSet := make(map[string]bool)
	for _, v := range NayinMap {
		nayinSet[v] = true
	}
	for nayin := range nayinSet {
		if _, ok := NayinWuxingMap[nayin]; !ok {
			t.Errorf("NayinWuxingMap missing entry for 纳音 %q", nayin)
		}
	}
}

func TestNayinWuxingMap_ValidWuxing(t *testing.T) {
	validWuxing := map[string]bool{"金": true, "木": true, "水": true, "火": true, "土": true}
	for nayin, wx := range NayinWuxingMap {
		if !validWuxing[wx] {
			t.Errorf("NayinWuxingMap[%q] = %q, not a valid 五行", nayin, wx)
		}
	}
}

// --- 五行生克表测试 ---

func TestWuxingShengkeMap_AllWuxingCovered(t *testing.T) {
	allWuxing := []string{"金", "木", "水", "火", "土"}
	for _, wx := range allWuxing {
		relations, ok := WuxingShengkeMap[wx]
		if !ok {
			t.Errorf("WuxingShengkeMap missing entry for %q", wx)
			continue
		}
		if len(relations) != 2 {
			t.Errorf("WuxingShengkeMap[%q] has %d entries, want 2", wx, len(relations))
		}
	}
}

// --- 喜用神计算测试 ---

// assertElementsMatch 检查两个字符串切片元素相同（忽略顺序）
func assertElementsMatch(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("长度不匹配: got %v (len=%d), want %v (len=%d)", got, len(got), want, len(want))
		return
	}
	gotSet := make(map[string]int)
	wantSet := make(map[string]int)
	for _, v := range got {
		gotSet[v]++
	}
	for _, v := range want {
		wantSet[v]++
	}
	for k, c := range gotSet {
		if wantSet[k] != c {
			t.Errorf("元素 %q 出现次数不匹配: got %d, want %d", k, c, wantSet[k])
		}
	}
	for k, c := range wantSet {
		if gotSet[k] != c {
			t.Errorf("缺少元素 %q (期望 %d 次)", k, c)
		}
	}
}

func TestCalculateXiyongshen_Strong_Wood(t *testing.T) {
	// 日主木, 身旺(木占40%) → 克泄耗: [金(克木), 火(木生), 土(木克)]
	wuxing := &WuxingResult{Jin: 2, Mu: 4, Shui: 2, Huo: 1, Tu: 1}
	got := calculateXiyongshen(wuxing, "木", "身旺")
	want := []string{"金", "火", "土"}
	assertElementsMatch(t, got, want)
}

func TestCalculateXiyongshen_Weak_Wood(t *testing.T) {
	// 日主木, 身弱(木占20%) → 生扶: [木(本身), 水(生木)]
	wuxing := &WuxingResult{Jin: 2, Mu: 2, Shui: 3, Huo: 2, Tu: 1}
	got := calculateXiyongshen(wuxing, "木", "身弱")
	want := []string{"木", "水"}
	assertElementsMatch(t, got, want)
}

func TestCalculateXiyongshen_Decline_Metal(t *testing.T) {
	// 日主金, 身衰(金占10%) → 生扶: [金(本身), 土(生金)]
	wuxing := &WuxingResult{Jin: 1, Mu: 3, Shui: 2, Huo: 2, Tu: 2}
	got := calculateXiyongshen(wuxing, "金", "身衰")
	want := []string{"金", "土"}
	assertElementsMatch(t, got, want)
}

func TestCalculateXiyongshen_Balanced_Fire(t *testing.T) {
	// 日主火, 身中(火占30%) → 平衡: [火(本身)]
	wuxing := &WuxingResult{Jin: 2, Mu: 2, Shui: 1, Huo: 3, Tu: 2}
	got := calculateXiyongshen(wuxing, "火", "身中")
	want := []string{"火"}
	assertElementsMatch(t, got, want)
}

func TestCalculateXiyongshen_Strong_Earth(t *testing.T) {
	// 日主土, 身旺(土占40%) → 克泄耗: [木(克土), 金(土生), 水(土克)]
	wuxing := &WuxingResult{Jin: 2, Mu: 1, Shui: 2, Huo: 1, Tu: 4}
	got := calculateXiyongshen(wuxing, "土", "身旺")
	want := []string{"木", "金", "水"}
	assertElementsMatch(t, got, want)
}

func TestCalculateXiyongshen_Weak_Water(t *testing.T) {
	// 日主水, 身弱(水占20%) → 生扶: [水(本身), 金(生水)]
	wuxing := &WuxingResult{Jin: 3, Mu: 2, Shui: 2, Huo: 2, Tu: 1}
	got := calculateXiyongshen(wuxing, "水", "身弱")
	want := []string{"水", "金"}
	assertElementsMatch(t, got, want)
}

func TestCalculateXiyongshen_AllZero(t *testing.T) {
	// 五行全零 → 返回日主本身
	wuxing := &WuxingResult{}
	got := calculateXiyongshen(wuxing, "木", "身衰")
	want := []string{"木"}
	assertElementsMatch(t, got, want)
}

func TestCalculateXiyongshen_Strong_AllElementsPresent(t *testing.T) {
	// 日主金, 身旺(金占40%) → 克泄耗: [火(克金), 水(金生), 木(金克)]
	wuxing := &WuxingResult{Jin: 4, Mu: 2, Shui: 2, Huo: 1, Tu: 1}
	got := calculateXiyongshen(wuxing, "金", "身旺")
	want := []string{"火", "水", "木"}
	assertElementsMatch(t, got, want)
}

func TestCalculateXiyongshen_Weak_CheckOnlyTwo(t *testing.T) {
	// 身弱时只返回2个元素（日主本身 + 生日主），不会多返回
	wuxing := &WuxingResult{Jin: 2, Mu: 1, Shui: 1, Huo: 2, Tu: 4}
	got := calculateXiyongshen(wuxing, "火", "身弱")
	if len(got) != 2 {
		t.Errorf("身弱应返回2个元素, got %v (len=%d)", got, len(got))
	}
	assertElementsMatch(t, got, []string{"火", "木"}) // 木生火
}

func TestCalculateXiyongshen_Strong_CheckThree(t *testing.T) {
	// 身旺时返回3个元素
	wuxing := &WuxingResult{Jin: 1, Mu: 4, Shui: 1, Huo: 1, Tu: 3}
	got := calculateXiyongshen(wuxing, "木", "身旺")
	if len(got) != 3 {
		t.Errorf("身旺应返回3个元素, got %v (len=%d)", got, len(got))
	}
}
