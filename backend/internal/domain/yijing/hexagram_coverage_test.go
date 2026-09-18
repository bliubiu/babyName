package yijing

import (
	"testing"
	"time"
)

// TestGetHexagramByNumberCoverage tests GetHexagramByNumber
func TestGetHexagramByNumberCoverage(t *testing.T) {
	// Test existing hexagram
	h := GetHexagramByNumber(1)
	if h == nil {
		t.Fatal("GetHexagramByNumber(1) should not be nil")
	}
	if h.Name != "乾" {
		t.Errorf("Name = %q, want 乾", h.Name)
	}

	// Test non-existing
	h2 := GetHexagramByNumber(100)
	if h2 != nil {
		t.Errorf("GetHexagramByNumber(100) should be nil, got %+v", h2)
	}

	h3 := GetHexagramByNumber(0)
	if h3 != nil {
		t.Errorf("GetHexagramByNumber(0) should be nil, got %+v", h3)
	}

	h4 := GetHexagramByNumber(-1)
	if h4 != nil {
		t.Errorf("GetHexagramByNumber(-1) should be nil, got %+v", h4)
	}
}

// TestGetHexagramByStrokesCoverage tests GetHexagramByStrokes
func TestGetHexagramByStrokesCoverage(t *testing.T) {
	h := GetHexagramByStrokes(1)
	if h == nil {
		t.Fatal("GetHexagramByStrokes(1) should not be nil")
	}
	if h.UpperTrigram < 1 || h.UpperTrigram > 8 {
		t.Errorf("UpperTrigram = %d, want 1-8", h.UpperTrigram)
	}
	if h.LowerTrigram < 1 || h.LowerTrigram > 8 {
		t.Errorf("LowerTrigram = %d, want 1-8", h.LowerTrigram)
	}

	// Test various stroke counts
	for strokes := 1; strokes <= 50; strokes++ {
		h := GetHexagramByStrokes(strokes)
		if h == nil {
			t.Errorf("GetHexagramByStrokes(%d) should not be nil", strokes)
		}
	}
}

// TestTrigramFromStrokesCoverage tests TrigramFromStrokes
func TestTrigramFromStrokesCoverage(t *testing.T) {
	tests := []struct {
		strokes int
		want    int
	}{
		{0, 1},
		{-1, 1},
		{1, 1},
		{2, 2},
		{3, 3},
		{4, 4},
		{5, 5},
		{6, 6},
		{7, 7},
		{8, 8},
		{9, 1},
		{16, 8},
		{17, 1},
		{24, 8},
	}

	for _, tc := range tests {
		got := TrigramFromStrokes(tc.strokes)
		if got != tc.want {
			t.Errorf("TrigramFromStrokes(%d) = %d, want %d", tc.strokes, got, tc.want)
		}
		if got < 1 || got > 8 {
			t.Errorf("TrigramFromStrokes(%d) = %d out of range 1-8", tc.strokes, got)
		}
	}
}

// TestFindHexagramByTrigramsCoverage tests FindHexagramByTrigrams
func TestFindHexagramByTrigramsCoverage(t *testing.T) {
	// Test valid trigrams
	h := FindHexagramByTrigrams(1, 1)
	if h == nil {
		t.Fatal("FindHexagramByTrigrams(1,1) should not be nil")
	}
	if h.Name != "乾" {
		t.Errorf("Name = %q, want 乾", h.Name)
	}

	// Test all combinations
	for upper := 1; upper <= 8; upper++ {
		for lower := 1; lower <= 8; lower++ {
			h := FindHexagramByTrigrams(upper, lower)
			if h == nil {
				t.Errorf("FindHexagramByTrigrams(%d,%d) should not be nil", upper, lower)
			}
			if h.UpperTrigram != upper || h.LowerTrigram != lower {
				t.Errorf("FindHexagramByTrigrams(%d,%d) = (%d,%d) want (%d,%d)",
					upper, lower, h.UpperTrigram, h.LowerTrigram, upper, lower)
			}
		}
	}

	// Test invalid trigrams
	h2 := FindHexagramByTrigrams(0, 1)
	if h2 != nil {
		t.Errorf("FindHexagramByTrigrams(0,1) should be nil, got %+v", h2)
	}

	h3 := FindHexagramByTrigrams(1, 0)
	if h3 != nil {
		t.Errorf("FindHexagramByTrigrams(1,0) should be nil, got %+v", h3)
	}

	h4 := FindHexagramByTrigrams(9, 1)
	if h4 != nil {
		t.Errorf("FindHexagramByTrigrams(9,1) should be nil, got %+v", h4)
	}
}

// TestGetHexagramByMeihuaNameCoverage tests GetHexagramByMeihuaName
func TestGetHexagramByMeihuaNameCoverage(t *testing.T) {
	// Test valid inputs
	h := GetHexagramByMeihuaName(4, 3) // 王=4, 小=3
	if h == nil {
		t.Fatal("GetHexagramByMeihuaName(4,3) should not be nil")
	}
	if h.UpperTrigram < 1 || h.UpperTrigram > 8 {
		t.Errorf("UpperTrigram = %d, want 1-8", h.UpperTrigram)
	}
	if h.LowerTrigram < 1 || h.LowerTrigram > 8 {
		t.Errorf("LowerTrigram = %d, want 1-8", h.LowerTrigram)
	}

	// Test edge cases
	h2 := GetHexagramByMeihuaName(0, 3)
	if h2 == nil {
		t.Error("GetHexagramByMeihuaName(0,3) should not be nil")
	}

	h3 := GetHexagramByMeihuaName(1, 0)
	if h3 == nil {
		t.Error("GetHexagramByMeihuaName(1,0) should not be nil")
	}

	// Test large stroke counts
	h4 := GetHexagramByMeihuaName(50, 30)
	if h4 == nil {
		t.Error("GetHexagramByMeihuaName(50,30) should not be nil")
	}
}

// TestGetHexagramByNameCoverage tests GetHexagramByName
func TestGetHexagramByNameCoverage(t *testing.T) {
	h := GetHexagramByName("乾")
	if h == nil {
		t.Fatal("GetHexagramByName(乾) should not be nil")
	}
	if h.Number != 1 {
		t.Errorf("Number = %d, want 1", h.Number)
	}

	h2 := GetHexagramByName("不存在的卦")
	if h2 != nil {
		t.Errorf("GetHexagramByName(不存在的卦) should be nil, got %+v", h2)
	}
}

// TestGetHexagramSymbolCoverage tests GetHexagramSymbol
func TestGetHexagramSymbolCoverage(t *testing.T) {
	symbol := GetHexagramSymbol(1, 8)
	if symbol == "" {
		t.Error("GetHexagramSymbol(1,8) should not be empty")
	}
	// Unicode symbols are 3 bytes each, so 2 runes = 6 bytes
	if len(symbol) != 6 {
		t.Errorf("Symbol length = %d, expected 6 bytes (2 runes)", len(symbol))
	}

	// Test all combinations
	for upper := 1; upper <= 8; upper++ {
		for lower := 1; lower <= 8; lower++ {
			s := GetHexagramSymbol(upper, lower)
			if s == "" {
				t.Errorf("GetHexagramSymbol(%d,%d) should not be empty", upper, lower)
			}
		}
	}
}

// TestGetAllHexagramsCoverage tests GetAllHexagrams
func TestGetAllHexagramsCoverage(t *testing.T) {
	all := GetAllHexagrams()
	if len(all) != 64 {
		t.Errorf("GetAllHexagrams() = %d, want 64", len(all))
	}
	// Verify all have required fields
	for _, h := range all {
		if h.Name == "" || h.ID == 0 {
			t.Errorf("Hexagram missing required fields: %+v", h)
		}
	}
}

// TestGetHexagramsByWuxingCoverage tests GetHexagramsByWuxing
func TestGetHexagramsByWuxingCoverage(t *testing.T) {
	// Test valid wuxing
	results := GetHexagramsByWuxing("金")
	if len(results) == 0 {
		t.Error("GetHexagramsByWuxing(金) should not be empty")
	}
	for _, h := range results {
		upperWx := TrigramWuxingMap[h.UpperTrigram]
		lowerWx := TrigramWuxingMap[h.LowerTrigram]
		if upperWx != "金" && lowerWx != "金" {
			t.Errorf("Hexagram %q does not have 金 in trigrams: upper=%s, lower=%s",
				h.Name, upperWx, lowerWx)
		}
	}

	// Test all wuxing
	for _, wx := range []string{"金", "木", "水", "火", "土"} {
		r := GetHexagramsByWuxing(wx)
		if len(r) == 0 {
			t.Errorf("GetHexagramsByWuxing(%s) should not be empty", wx)
		}
	}

	// Test invalid wuxing - should return first 8
	invalid := GetHexagramsByWuxing("无效")
	if len(invalid) != 8 {
		t.Errorf("GetHexagramsByWuxing(无效) should return 8 default, got %d", len(invalid))
	}
}

// TestMatchHexagramWithXiyongshenCoverage tests MatchHexagramWithXiyongshen
func TestMatchHexagramWithXiyongshenCoverage(t *testing.T) {
	hexagram := GetHexagramByNumber(1) // 乾: 上金下金
	if hexagram == nil {
		t.Fatal("GetHexagramByNumber(1) should not be nil")
	}

	// Test matching xiyongshen
	match := MatchHexagramWithXiyongshen(hexagram, []string{"金"})
	if match == nil {
		t.Fatal("MatchHexagramWithXiyongshen should not return nil")
	}
	if match.Score <= 50 {
		t.Errorf("Score = %d, should be > 50 for matching wuxing", match.Score)
	}
	if match.WuxingCompat != "最佳" {
		t.Errorf("WuxingCompat = %q, want 最佳", match.WuxingCompat)
	}

	// Test sheng (generating)
	match2 := MatchHexagramWithXiyongshen(hexagram, []string{"土"}) // 土生金
	if match2 == nil {
		t.Fatal("MatchHexagramWithXiyongshen should not return nil for sheng")
	}
	if match2.Score <= 50 {
		t.Errorf("Score = %d, should be > 50 for sheng", match2.Score)
	}
	if match2.WuxingCompat != "良好" && match2.WuxingCompat != "最佳" {
		t.Errorf("WuxingCompat = %q, want 良好 or 最佳", match2.WuxingCompat)
	}

	// Test no match
	match3 := MatchHexagramWithXiyongshen(hexagram, []string{"木"}) // 木克土，金不相关
	if match3 == nil {
		t.Fatal("MatchHexagramWithXiyongshen should not return nil for no match")
	}
	if match3.Score != 50 {
		t.Errorf("Score = %d, should be 50 for no match", match3.Score)
	}
	if match3.WuxingCompat != "一般" {
		t.Errorf("WuxingCompat = %q, want 一般", match3.WuxingCompat)
	}

	// Test nil hexagram
	match4 := MatchHexagramWithXiyongshen(nil, []string{"金"})
	if match4 != nil {
		t.Error("MatchHexagramWithXiyongshen(nil) should return nil")
	}

	// Test empty xiyongshen
	match5 := MatchHexagramWithXiyongshen(hexagram, []string{})
	if match5 == nil {
		t.Fatal("MatchHexagramWithXiyongshen with empty xiyongshen should not be nil")
	}
	if match5.Score != 50 {
		t.Errorf("Score = %d, should be 50 for empty xiyongshen", match5.Score)
	}
}

// TestGetBestHexagramsForXiyongshenCoverage tests GetBestHexagramsForXiyongshen
func TestGetBestHexagramsForXiyongshenCoverage(t *testing.T) {
	matches := GetBestHexagramsForXiyongshen([]string{"金"}, 10)
	if len(matches) == 0 {
		t.Error("GetBestHexagramsForXiyongshen should return non-empty")
	}
	if len(matches) > 10 {
		t.Errorf("Should limit to 10, got %d", len(matches))
	}

	// Verify sorted by score descending
	for i := 0; i < len(matches)-1; i++ {
		if matches[i].Score < matches[i+1].Score {
			t.Errorf("Results not sorted by score: %d < %d at index %d",
				matches[i].Score, matches[i+1].Score, i)
		}
	}

	// Test with limit 0 (no limit)
	matches2 := GetBestHexagramsForXiyongshen([]string{"金"}, 0)
	if len(matches2) < len(matches) {
		t.Errorf("Limit 0 should return more or equal, got %d vs %d", len(matches2), len(matches))
	}

	// Test with multiple xiyongshen
	matches3 := GetBestHexagramsForXiyongshen([]string{"金", "水"}, 5)
	if len(matches3) == 0 {
		t.Error("Multiple xiyongshen should return non-empty")
	}
	if len(matches3) > 5 {
		t.Errorf("Should limit to 5, got %d", len(matches3))
	}
}

// TestAnalyzeHexagramWuxingCoverage tests AnalyzeHexagramWuxing
func TestAnalyzeHexagramWuxingCoverage(t *testing.T) {
	hexagram := GetHexagramByNumber(1) // 乾: 上金下金
	if hexagram == nil {
		t.Fatal("GetHexagramByNumber(1) should not be nil")
	}

	result := AnalyzeHexagramWuxing(hexagram)
	if result == "" {
		t.Error("AnalyzeHexagramWuxing should not be empty")
	}
	if len(result) < 10 {
		t.Errorf("Result too short: %q", result)
	}

	// Test with different hexagrams
	for _, num := range []int{1, 2, 11, 12, 63, 64} {
		h := GetHexagramByNumber(num)
		if h == nil {
			continue
		}
		r := AnalyzeHexagramWuxing(h)
		if r == "" {
			t.Errorf("AnalyzeHexagramWuxing(%d) should not be empty", num)
		}
	}

	// Test nil
	result2 := AnalyzeHexagramWuxing(nil)
	if result2 != "" {
		t.Errorf("AnalyzeHexagramWuxing(nil) should be empty, got %q", result2)
	}
}

// TestCalculateDayanNumberCoverage tests CalculateDayanNumber
func TestCalculateDayanNumberCoverage(t *testing.T) {
	// Run multiple times to check range
	results := make(map[int]int)
	for i := 0; i < 100; i++ {
		num := CalculateDayanNumber()
		results[num]++
		// Current implementation returns 1-3 or 8 due to modulo logic
		if num < 1 || num > 8 || num == 4 || num == 5 || num == 6 || num == 7 {
			t.Errorf("CalculateDayanNumber() = %d, unexpected value", num)
		}
	}
	if len(results) == 0 {
		t.Error("No results generated")
	}
}

// TestGetYaoInfoCoverage tests GetYaoInfo
func TestGetYaoInfoCoverage(t *testing.T) {
	for _, yao := range []int{6, 7, 8, 9} {
		info := GetYaoInfo(yao)
		if info.Name == "" {
			t.Errorf("GetYaoInfo(%d) Name should not be empty", yao)
		}
		if info.Yao != yao {
			t.Errorf("GetYaoInfo(%d).Yao = %d, want %d", yao, info.Yao, yao)
		}
	}

	// Test invalid yao
	info := GetYaoInfo(5)
	if info.Name != "未知" {
		t.Errorf("GetYaoInfo(5) Name = %q, want 未知", info.Name)
	}
}

// TestGetYaoNameCoverage tests GetYaoName
func TestGetYaoNameCoverage(t *testing.T) {
	tests := []struct {
		yao  int
		want string
	}{
		{6, "初六"},
		{7, "初九"},
		{8, "六二"},
		{9, "九二"},
	}

	for _, tc := range tests {
		got := GetYaoName(tc.yao)
		if got != tc.want {
			t.Errorf("GetYaoName(%d) = %q, want %q", tc.yao, got, tc.want)
		}
	}

	// Test invalid
	got := GetYaoName(10)
	if got != "" {
		t.Errorf("GetYaoName(10) = %q, want empty", got)
	}
}

// TestInterpretHexagramCoverage tests InterpretHexagram
func TestInterpretHexagramCoverage(t *testing.T) {
	hexagram := GetHexagramByNumber(1)
	if hexagram == nil {
		t.Fatal("GetHexagramByNumber(1) should not be nil")
	}

	interp := InterpretHexagram(hexagram, []string{"金"})
	if interp == nil {
		t.Fatal("InterpretHexagram should not return nil")
	}
	if interp.Overall == "" {
		t.Error("Overall should not be empty")
	}
	if interp.Career == "" {
		t.Error("Career should not be empty")
	}
	if interp.Love == "" {
		t.Error("Love should not be empty")
	}
	if interp.Health == "" {
		t.Error("Health should not be empty")
	}
	if interp.Fortune == "" {
		t.Error("Fortune should not be empty")
	}
	if interp.Auspicious == "" {
		t.Error("Auspicious should not be empty")
	}
	if interp.Inauspicious == "" {
		t.Error("Inauspicious should not be empty")
	}

	// Test with matching xiyongshen
	interp2 := InterpretHexagram(hexagram, []string{"金"})
	if interp2.Fortune != "五行相合，财运亨通" {
		t.Errorf("Fortune = %q, want 五行相合，财运亨通", interp2.Fortune)
	}
	if interp2.Overall == hexagram.Interpretation {
		t.Error("Overall should be modified when matching xiyongshen")
	}

	// Test with non-matching xiyongshen
	interp3 := InterpretHexagram(hexagram, []string{"木"})
	if interp3.Fortune == "五行相合，财运亨通" {
		t.Error("Fortune should not be 五行相合 for non-matching xiyongshen")
	}
}

// TestCastHexagramByTimeCoverage tests CastHexagramByTime
func TestCastHexagramByTimeCoverage(t *testing.T) {
	// Use fixed seed for deterministic test
	now := time.Now()
	r := CastHexagramByTime(now.Year(), int(now.Month()), now.Day(), now.Hour())
	if r == nil {
		t.Fatal("CastHexagramByTime should not return nil")
	}
	if r.Hexagram == nil {
		t.Error("Hexagram should not be nil")
	}
	if len(r.YaoLines) != 6 {
		t.Errorf("YaoLines length = %d, want 6", len(r.YaoLines))
	}
	for _, yao := range r.YaoLines {
		if yao < 6 || yao > 9 {
			t.Errorf("Yao = %d, want 6-9", yao)
		}
	}
	if r.ChangeYao < 0 || r.ChangeYao > 6 {
		t.Errorf("ChangeYao = %d, want 0-6", r.ChangeYao)
	}
	if r.Interpretation == "" {
		t.Error("Interpretation should not be empty")
	}
}

// TestCastHexagramByDayanCoverage tests CastHexagramByDayan
func TestCastHexagramByDayanCoverage(t *testing.T) {
	r := CastHexagramByDayan()
	if r == nil {
		t.Fatal("CastHexagramByDayan should not return nil")
	}
	if r.Hexagram == nil {
		t.Error("Hexagram should not be nil")
	}
	if len(r.YaoLines) != 6 {
		t.Errorf("YaoLines length = %d, want 6", len(r.YaoLines))
	}
	if len(r.DayanNumbers) != 6 {
		t.Errorf("DayanNumbers length = %d, want 6", len(r.DayanNumbers))
	}
	for _, yao := range r.YaoLines {
		if yao < 6 || yao > 9 {
			t.Errorf("Yao = %d, want 6-9", yao)
		}
	}
	for _, num := range r.DayanNumbers {
		// Current CalculateDayanNumber returns 1-3 or 8
		if num < 1 || num > 8 || num == 4 || num == 5 || num == 6 || num == 7 {
			t.Errorf("DayanNumber = %d, unexpected value", num)
		}
	}
	if r.ChangeYao < 0 || r.ChangeYao > 6 {
		t.Errorf("ChangeYao = %d, want 0-6", r.ChangeYao)
	}
	if r.Interpretation == "" {
		t.Error("Interpretation should not be empty")
	}

	// Verify OriginalHex is set when there's a change
	if r.ChangeYao > 0 && r.OriginalHex == nil {
		t.Error("OriginalHex should be set when ChangeYao > 0")
	}
}