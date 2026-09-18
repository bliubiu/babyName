package bazi

import (
	"testing"
)

// TestCalculateWuxingStrengthCoverage tests the WuxingStrength calculation
func TestCalculateWuxingStrengthCoverage(t *testing.T) {
	wuxing := &WuxingResult{Jin: 3, Mu: 2, Shui: 2, Huo: 1, Tu: 2}
	result := CalculateWuxingStrength(wuxing)

	if result.Jin.Count != 3 {
		t.Errorf("Jin.Count = %d, want 3", result.Jin.Count)
	}

	elements := []*StrengthInfo{&result.Jin, &result.Mu, &result.Shui, &result.Huo, &result.Tu}
	for _, e := range elements {
		if e.Strength == "" {
			t.Errorf("Element missing Strength: %+v", e)
		}
		if e.Score == 0 {
			t.Errorf("Element missing Score: %+v", e)
		}
		if e.Advice == "" {
			t.Errorf("Element missing Advice: %+v", e)
		}
	}

	zeroWuxing := &WuxingResult{}
	zeroResult := CalculateWuxingStrength(zeroWuxing)
	if zeroResult.Jin.Strength != "太弱" {
		t.Errorf("Zero total should give 太弱, got %s", zeroResult.Jin.Strength)
	}
}

// TestCalculateWuxingScoreCoverage tests the WuxingScore calculation
func TestCalculateWuxingScoreCoverage(t *testing.T) {
	balanced := &WuxingResult{Jin: 2, Mu: 2, Shui: 2, Huo: 2, Tu: 2}
	score := CalculateWuxingScore(balanced)
	if score < 50 || score > 80 {
		t.Errorf("Balanced Wuxing score = %d, expected 50-80", score)
	}

	zero := &WuxingResult{}
	zeroScore := CalculateWuxingScore(zero)
	if zeroScore != 20 {
		t.Errorf("Zero Wuxing score = %d, want 20", zeroScore)
	}

	unbalanced := &WuxingResult{Jin: 10, Mu: 0, Shui: 0, Huo: 0, Tu: 0}
	unbalancedScore := CalculateWuxingScore(unbalanced)
	if unbalancedScore < 20 || unbalancedScore > 50 {
		t.Errorf("Unbalanced Wuxing score = %d, expected 20-50", unbalancedScore)
	}
}

// TestCalculateDayMasterStrengthWeightedCoverage tests the weighted day master strength
func TestCalculateDayMasterStrengthWeightedCoverage(t *testing.T) {
	weighted := &WeightedWuxing{
		Jin:  10, Mu: 10, Shui: 10, Huo: 10, Tu: 10,
	}

	// Just test that the function runs and returns a valid strength
	strengths := []string{"身旺", "身中", "身弱", "身衰", "偏弱"}
	for _, rishou := range []string{"金", "木", "水", "火", "土"} {
		got := calculateDayMasterStrengthWeighted(weighted, rishou, "木")
		found := false
		for _, s := range strengths {
			if got == s {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Invalid strength %q for %s: %q", got, rishou, strengths)
		}
	}

	got := calculateDayMasterStrengthWeighted(weighted, "木", "")
	if got == "" {
		t.Error("Empty monthBranch should not give empty strength")
	}
}

// TestCalculateXiyongshenWeightedCoverage tests the weighted xiyongshen calculation
func TestCalculateXiyongshenWeightedCoverage(t *testing.T) {
	weighted := &WeightedWuxing{
		Jin:  10, Mu: 10, Shui: 10, Huo: 10, Tu: 10,
	}

	got := calculateXiyongshenWeighted(weighted, "木", "身旺", []string{"火", "土"})
	if len(got) == 0 {
		t.Error("calculateXiyongshenWeighted should return non-empty slice")
	}

	got2 := calculateXiyongshenWeighted(weighted, "木", "身弱", []string{"火", "土"})
	if len(got2) == 0 {
		t.Error("calculateXiyongshenWeighted for weak should return non-empty slice")
	}

	got3 := calculateXiyongshenWeighted(weighted, "土", "身中", []string{"金", "水"})
	if len(got3) == 0 {
		t.Error("calculateXiyongshenWeighted for balanced should return non-empty slice")
	}
}

// TestCalculateYiyongshenFromWeightedCoverage tests the weighted yiyongshen
func TestCalculateYiyongshenFromWeightedCoverage(t *testing.T) {
	weighted := &WeightedWuxing{
		Jin:  10, Mu: 5, Shui: 10, Huo: 10, Tu: 10,
	}

	got := calculateYiyongshenFromWeighted(weighted, "木")
	// Just verify it runs without panic
	_ = got

	// Test with zero total
	zeroWeighted := &WeightedWuxing{}
	got2 := calculateYiyongshenFromWeighted(zeroWeighted, "木")
	if got2 != nil {
		t.Errorf("Zero total should return nil, got %v", got2)
	}
}

// TestCalculateBaziPatternCoverage tests the bazi pattern classification
func TestCalculateBaziPatternCoverage(t *testing.T) {
	// Just test that the function runs and returns a non-empty pattern
	patterns := []struct {
		dayMasterStr string
		rishouWuxing string
		season       string
	}{
		{"身旺", "木", "春"},
		{"身旺", "木", "秋"},
		{"身中", "火", "夏"},
		{"身弱", "土", "长夏"},
		{"身弱", "土", "春"},
		{"身衰", "金", "秋"},
		{"身衰", "金", "夏"},
		{"未知", "水", "冬"},
	}

	for _, tc := range patterns {
		got := calculateBaziPattern(tc.dayMasterStr, tc.rishouWuxing, tc.season)
		if got == "" {
			t.Errorf("calculateBaziPattern returned empty for %+v", tc)
		}
	}
}

// TestCalculateYongshenScoreCoverage tests the yongshen score
func TestCalculateYongshenScoreCoverage(t *testing.T) {
	tests := []struct {
		name         string
		xiyongshen   []string
		yiyongshen   []string
		wuxing       *WuxingResult
	}{
		{"有喜用神", []string{"木", "水"}, []string{"金"}, &WuxingResult{Jin: 2, Mu: 4, Shui: 3, Huo: 1, Tu: 2}},
		{"无喜用神", []string{}, []string{"金"}, &WuxingResult{Jin: 2, Mu: 4, Shui: 3, Huo: 1, Tu: 2}},
		{"喜用神满分", []string{"木"}, []string{}, &WuxingResult{Jin: 0, Mu: 10, Shui: 0, Huo: 0, Tu: 0}},
		{"忌神扣分", []string{"木"}, []string{"木"}, &WuxingResult{Jin: 2, Mu: 4, Shui: 3, Huo: 1, Tu: 2}},
	}

	for _, tc := range tests {
		got := calculateYongshenScore(tc.xiyongshen, tc.yiyongshen, tc.wuxing)
		if got < 0 || got > 100 {
			t.Errorf("%s: got %d, want range [0, 100]", tc.name, got)
		}
	}
}

// TestCalculateShengKeScoreCoverage tests the sheng/ke score
func TestCalculateShengKeScoreCoverage(t *testing.T) {
	wuxing := &WuxingResult{Jin: 3, Mu: 2, Shui: 4, Huo: 1, Tu: 2}

	score := calculateShengKeScore([]string{"水"}, wuxing)
	if score < 0 {
		t.Errorf("ShengKeScore should not be negative: %d", score)
	}

	score2 := calculateShengKeScore([]string{"木"}, wuxing)
	if score2 < 0 {
		t.Errorf("ShengKeScore for 木 should not be negative: %d", score2)
	}

	score3 := calculateShengKeScore([]string{"水", "木"}, wuxing)
	if score3 < 0 {
		t.Errorf("ShengKeScore for [水,木] should not be negative: %d", score3)
	}

	score4 := calculateShengKeScore([]string{}, wuxing)
	if score4 != 0 {
		t.Errorf("ShengKeScore for empty got %d, want 0", score4)
	}
}

// TestGetLeapMonthAdviceCoverage tests the leap month advice
func TestGetLeapMonthAdviceCoverage(t *testing.T) {
	advice := getLeapMonthAdvice(6, "甲")
	if advice == "" {
		t.Error("getLeapMonthAdvice returned empty string")
	}
	if len(advice) < 10 {
		t.Errorf("Advice too short: %q", advice)
	}
}

// TestCalculateWuxingCoverage tests the basic wuxing calculation
func TestCalculateWuxingCoverage(t *testing.T) {
	result := calculateWuxing("甲子")
	// 甲=木, 子=水
	if result.Mu != 1 || result.Shui != 1 {
		t.Errorf("甲子: got %+v, want Mu=1, Shui=1", result)
	}

	result2 := calculateWuxing("丙午")
	// 丙=火, 午=火
	if result2.Huo != 2 {
		t.Errorf("丙午: got %+v, want Huo=2", result2)
	}

	result3 := calculateWuxing("未知")
	// 未知字符可能有默认五行
	_ = result3
}

// TestCalculateDayMasterStrengthCoverage tests the old day master strength
func TestCalculateDayMasterStrengthCoverage(t *testing.T) {
	tests := []struct {
		name         string
		wuxing       *WuxingResult
		rishouWuxing string
		want         string
	}{
		{"身旺", &WuxingResult{Jin: 1, Mu: 6, Shui: 1, Huo: 1, Tu: 1}, "木", "身旺"},
		{"身中", &WuxingResult{Jin: 2, Mu: 3, Shui: 2, Huo: 2, Tu: 1}, "木", "身中"},
		{"身弱", &WuxingResult{Jin: 2, Mu: 2, Shui: 3, Huo: 2, Tu: 1}, "木", "身弱"},
		{"身衰", &WuxingResult{Jin: 3, Mu: 1, Shui: 3, Huo: 2, Tu: 1}, "木", "身衰"},
		{"零总和", &WuxingResult{}, "木", "偏弱"},
	}

	for _, tc := range tests {
		got := calculateDayMasterStrength(tc.wuxing, tc.rishouWuxing)
		if got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

// TestCalculateYiyongshenCoverage tests the old yiyongshen
func TestCalculateYiyongshenCoverage(t *testing.T) {
	wuxing := &WuxingResult{Jin: 2, Mu: 1, Shui: 3, Huo: 2, Tu: 2}
	got := calculateYiyongshen(wuxing, "木")
	// Just verify it runs
	_ = got

	wuxing2 := &WuxingResult{Jin: 2, Mu: 2, Shui: 2, Huo: 2, Tu: 2}
	got2 := calculateYiyongshen(wuxing2, "木")
	if len(got2) != 1 {
		t.Errorf("Equal wuxing got %v, want 1 element", got2)
	}
}

// TestCalculateXiyongshenCoverage tests the old xiyongshen
func TestCalculateXiyongshenCoverage(t *testing.T) {
	wuxing := &WuxingResult{Jin: 1, Mu: 6, Shui: 1, Huo: 1, Tu: 1}

	got := calculateXiyongshen(wuxing, "木", "身旺")
	if len(got) != 3 {
		t.Errorf("身旺 got len %d, want 3", len(got))
	}

	got2 := calculateXiyongshen(wuxing, "木", "身弱")
	if len(got2) != 2 {
		t.Errorf("身弱 got len %d, want 2", len(got2))
	}
}

// TestCalculateTiaohouShenByBranchCoverage tests tiaohou shen by branch
func TestCalculateTiaohouShenByBranchCoverage(t *testing.T) {
	tests := []struct {
		monthGanzhi string
		want        []string
	}{
		{"甲子", []string{"火", "土"}},
		{"乙丑", []string{"火", "土"}},
		{"丙寅", []string{"火", "金"}},
		{"丁卯", []string{"火", "金"}},
		{"戊辰", []string{"金", "水"}},
		{"己巳", []string{"水", "金"}},
		{"庚午", []string{"水", "金"}},
		{"辛未", []string{"水", "金"}},
		{"壬申", []string{"火", "木"}},
		{"癸酉", []string{"火", "木"}},
		{"甲戌", []string{"火", "木"}},
		{"乙亥", []string{"火", "土"}},
		{"甲X", nil},
	}

	for _, tc := range tests {
		got := calculateTiaohouShenByBranch(tc.monthGanzhi)
		if tc.want == nil {
			if got != nil {
				t.Errorf("%s: got %v, want nil", tc.monthGanzhi, got)
			}
		} else {
			if len(got) != len(tc.want) {
				t.Errorf("%s: got len %d, want %d", tc.monthGanzhi, len(got), len(tc.want))
			}
			for i, w := range tc.want {
				if i < len(got) && got[i] != w {
					t.Errorf("%s: got[%d]=%q, want %q", tc.monthGanzhi, i, got[i], w)
				}
			}
		}
	}
}

// TestGetSolarTermByDateCoverage tests solar term by date
func TestGetSolarTermByDateCoverage(t *testing.T) {
	// Test a few known dates - June 15 is not a solar term date
	term := GetSolarTermByDate(2024, 6, 15)
	_ = term // May be empty for non-term dates

	// Test a known term date - June 5 is 芒种
	term2 := GetSolarTermByDate(2024, 6, 5)
	if term2 == "" {
		t.Error("GetSolarTermByDate(2024,6,5) should not be empty (芒种)")
	}

	// Test date with no term
	term3 := GetSolarTermByDate(2024, 6, 16)
	_ = term3 // Some dates don't have a term, that's OK
}

// TestGetSolarTermFromTymeCoverage tests tyme-based solar term
func TestGetSolarTermFromTymeCoverage(t *testing.T) {
	term, isLeap, err := GetSolarTermFromTyme(2024, 6, 15, 12, 0)
	if err != nil {
		t.Fatalf("GetSolarTermFromTyme error: %v", err)
	}
	if term == "" {
		t.Error("term should not be empty")
	}
	if isLeap {
		t.Errorf("June should not be leap month")
	}

	term2, isLeap2, err := GetSolarTermFromTyme(2023, 5, 20, 12, 0)
	if err != nil {
		t.Fatalf("GetSolarTermFromTyme error: %v", err)
	}
	if term2 == "" {
		t.Error("term2 should not be empty")
	}
	_ = isLeap2
}

// TestParseTimeToHourMinuteCoverage tests time parsing
func TestParseTimeToHourMinuteCoverage(t *testing.T) {
	tests := []struct {
		input    string
		wantHour int
		wantMin  int
		wantErr  bool
	}{
		{"12:30", 12, 30, false},
		{"00:00", 0, 0, false},
		{"23:59", 23, 59, false},
		{"08:05", 8, 5, false},
		{"12", 0, 0, true},
		{"abc", 0, 0, true},
		{"25:00", 0, 0, true},
		{"12:60", 0, 0, true},
	}

	for _, tc := range tests {
		h, m, err := ParseTimeToHourMinute(tc.input)
		if tc.wantErr {
			if err == nil {
				t.Errorf("ParseTimeToHourMinute(%q) should error, got %d:%d", tc.input, h, m)
			}
		} else {
			if err != nil {
				t.Errorf("ParseTimeToHourMinute(%q) unexpected error: %v", tc.input, err)
			}
			if h != tc.wantHour || m != tc.wantMin {
				t.Errorf("ParseTimeToHourMinute(%q) = %d:%d, want %d:%d", tc.input, h, m, tc.wantHour, tc.wantMin)
			}
		}
	}
}

// TestSeasonFromMonthBranchCoverage tests season calculation
func TestSeasonFromMonthBranchCoverage(t *testing.T) {
	tests := []struct {
		monthGanzhi string
		want        string
	}{
		{"甲寅", "春"}, {"乙卯", "春"}, {"丙辰", "春"},
		{"丁巳", "夏"}, {"戊午", "夏"}, {"己未", "夏"},
		{"庚申", "秋"}, {"辛酉", "秋"}, {"壬戌", "秋"},
		{"癸亥", "冬"}, {"甲子", "冬"}, {"乙丑", "冬"},
		{"甲X", ""},
	}

	for _, tc := range tests {
		got := seasonFromMonthBranch(tc.monthGanzhi)
		if got != tc.want {
			t.Errorf("seasonFromMonthBranch(%s) = %q, want %q", tc.monthGanzhi, got, tc.want)
		}
	}
}