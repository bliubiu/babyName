package zodiac

import (
	"os"
	"testing"
)

// Backup original ZodiacList for tests that modify it
var originalZodiacList = ZodiacList

func restoreZodiacList() {
	ZodiacList = originalZodiacList
}

// TestGetZodiacByNameCoverage tests GetZodiacByName
func TestGetZodiacByNameCoverage(t *testing.T) {
	// Test all 12 zodiacs by name
	for _, z := range ZodiacList {
		found := GetZodiacByName(z.Name)
		if found == nil {
			t.Errorf("GetZodiacByName(%q) should not be nil", z.Name)
		}
		if found.Name != z.Name {
			t.Errorf("GetZodiacByName(%q) = %q, want %q", z.Name, found.Name, z.Name)
		}
	}

	// Test non-existing
	if found := GetZodiacByName("不存在"); found != nil {
		t.Errorf("GetZodiacByName(不存在) should be nil, got %+v", found)
	}
	if found := GetZodiacByName(""); found != nil {
		t.Errorf("GetZodiacByName('') should be nil, got %+v", found)
	}
}

// TestGetZodiacByYearCoverage tests GetZodiacByYear
func TestGetZodiacByYearCoverage(t *testing.T) {
	// 1900 is 鼠 (ID 1)
	z := GetZodiacByYear(1900)
	if z == nil {
		t.Fatal("GetZodiacByYear(1900) should not be nil")
	}
	if z.Name != "鼠" {
		t.Errorf("GetZodiacByYear(1900) = %q, want 鼠", z.Name)
	}

	// Test cycle through years
	expected := []string{"鼠", "牛", "虎", "兔", "龙", "蛇", "马", "羊", "猴", "鸡", "狗", "猪"}
	for i := 0; i < 12; i++ {
		year := 1900 + i
		z := GetZodiacByYear(year)
		if z == nil {
			t.Errorf("GetZodiacByYear(%d) should not be nil", year)
			continue
		}
		if z.Name != expected[i] {
			t.Errorf("GetZodiacByYear(%d) = %q, want %q", year, z.Name, expected[i])
		}
	}

	// Test years beyond 1912 (cycle repeats)
	for i := 0; i < 12; i++ {
		year := 1912 + i
		z := GetZodiacByYear(year)
		if z == nil {
			t.Errorf("GetZodiacByYear(%d) should not be nil", year)
			continue
		}
		if z.Name != expected[i] {
			t.Errorf("GetZodiacByYear(%d) = %q, want %q", year, z.Name, expected[i])
		}
	}

	// Test negative year (before 1900)
	z2 := GetZodiacByYear(1899)
	if z2 == nil {
		t.Fatal("GetZodiacByYear(1899) should not be nil")
	}
	if z2.Name != "猪" { // 1899 is 猪 (year before 1900)
		t.Errorf("GetZodiacByYear(1899) = %q, want 猪", z2.Name)
	}
}

// TestGetZodiacByIDCoverage tests GetZodiacByID
func TestGetZodiacByIDCoverage(t *testing.T) {
	for i := 1; i <= 12; i++ {
		z := GetZodiacByID(i)
		if z == nil {
			t.Errorf("GetZodiacByID(%d) should not be nil", i)
			continue
		}
		if z.ID != i {
			t.Errorf("GetZodiacByID(%d).ID = %d, want %d", i, z.ID, i)
		}
	}

	// Test invalid IDs
	if z := GetZodiacByID(0); z != nil {
		t.Errorf("GetZodiacByID(0) should be nil, got %+v", z)
	}
	if z := GetZodiacByID(13); z != nil {
		t.Errorf("GetZodiacByID(13) should be nil, got %+v", z)
	}
	if z := GetZodiacByID(-1); z != nil {
		t.Errorf("GetZodiacByID(-1) should be nil, got %+v", z)
	}
}

// TestGetAllZodiacsCoverage tests GetAllZodiacs
func TestGetAllZodiacsCoverage(t *testing.T) {
	all := GetAllZodiacs()
	if len(all) != 12 {
		t.Errorf("GetAllZodiacs() = %d, want 12", len(all))
	}
	// Note: GetAllZodiacs returns the original slice, not a copy
	// This is the current implementation behavior
}

// TestMatchNameWithZodiacCoverage tests MatchNameWithZodiac
func TestMatchNameWithZodiacCoverage(t *testing.T) {
	// Restore zodiac list to ensure clean state
	defer restoreZodiacList()

	// Test with good characters for 鼠
	match := MatchNameWithZodiac("米", "鼠")
	if match == nil {
		t.Fatal("MatchNameWithZodiac(米, 鼠) should not be nil")
	}
	if match.Zodiac.Name != "鼠" {
		t.Errorf("Zodiac = %q, want 鼠", match.Zodiac.Name)
	}
	if match.PianpangScore < 50 {
		t.Errorf("Score = %d, should be >= 50 for good chars", match.PianpangScore)
	}
	if match.PianpangDesc == "" {
		t.Error("Description should not be empty")
	}

	// Test with bad characters for 鼠
	match2 := MatchNameWithZodiac("刀", "鼠")
	if match2 == nil {
		t.Fatal("MatchNameWithZodiac(刀, 鼠) should not be nil")
	}
	if match2.PianpangScore > 50 {
		t.Errorf("Score = %d, should be <= 50 for bad chars", match2.PianpangScore)
	}

	// Test with mixed good and bad
	match3 := MatchNameWithZodiac("米刀", "鼠")
	if match3 == nil {
		t.Fatal("MatchNameWithZodiac(米刀, 鼠) should not be nil")
	}

	// Test non-existing zodiac
	match4 := MatchNameWithZodiac("测试", "不存在")
	if match4 != nil {
		t.Errorf("MatchNameWithZodiac for non-existing zodiac should be nil, got %+v", match4)
	}

	// Test empty name
	match5 := MatchNameWithZodiac("", "鼠")
	if match5 == nil {
		t.Fatal("MatchNameWithZodiac with empty name should not be nil")
	}
	if match5.PianpangScore != 50 {
		t.Errorf("Empty name should have base score 50, got %d", match5.PianpangScore)
	}

	// Test all zodiacs
	for _, z := range ZodiacList {
		m := MatchNameWithZodiac("测试", z.Name)
		if m == nil {
			t.Errorf("MatchNameWithZodiac for %s should not be nil", z.Name)
		}
		if m.Zodiac.Name != z.Name {
			t.Errorf("Zodiac mismatch for %s: got %s", z.Name, m.Zodiac.Name)
		}
	}
}

// TestParsePianpangCoverage tests parsePianpang
func TestParsePianpangCoverage(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"米、豆、鱼", []string{"米", "豆", "鱼"}},
		{"刀、刂、火、灬", []string{"刀", "刂", "火", "灬"}},
		{"单个", []string{"单个"}},
		{"", []string{}},
		{"、", []string{}},
		{"A、B、", []string{"A", "B"}},
	}

	for _, tc := range tests {
		result := parsePianpang(tc.input)
		if len(result) != len(tc.expected) {
			t.Errorf("parsePianpang(%q) = %v, want %v", tc.input, result, tc.expected)
			continue
		}
		for i, v := range result {
			if v != tc.expected[i] {
				t.Errorf("parsePianpang(%q)[%d] = %q, want %q", tc.input, i, v, tc.expected[i])
			}
		}
	}
}

// TestContainsRuneCoverage tests containsRune
func TestContainsRuneCoverage(t *testing.T) {
	tests := []struct {
		pianpang string
		char     rune
		expected bool
	}{
		{"米", '米', true},
		{"米", '豆', false},
		{"刀", '刀', true},
		{"", '米', false},
		{"火", '火', true},
	}

	for _, tc := range tests {
		result := containsRune(tc.pianpang, tc.char)
		if result != tc.expected {
			t.Errorf("containsRune(%q, %q) = %v, want %v", tc.pianpang, tc.char, result, tc.expected)
		}
	}
}

// TestLoadFromJSONCoverage tests LoadFromJSON (requires file)
func TestLoadFromJSONCoverage(t *testing.T) {
	defer restoreZodiacList()

	// Create a temporary zodiac.json file
	tmpDir := t.TempDir()
	jsonContent := `[
		{
			"id": 1,
			"name": "鼠",
			"wuxing": "水",
			"compatible": ["牛", "龙", "猴"],
			"conflicting": ["马", "兔", "羊"],
			"avoid_chars": "午、马、兔、羊",
			"lucky_number": [2, 3],
			"lucky_color": "蓝色、金色、绿色",
			"lucky_direction": "东南、东北",
			"good_pianpang": "米、豆、鱼、虫",
			"bad_pianpang": "刀、刂、火、灬"
		}
	]`
	err := os.WriteFile(tmpDir+"/zodiac.json", []byte(jsonContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	// Load from JSON
	err = LoadFromJSON(tmpDir)
	if err != nil {
		t.Fatalf("LoadFromJSON failed: %v", err)
	}

	// Verify loaded
	if len(ZodiacList) != 1 {
		t.Errorf("Loaded ZodiacList length = %d, want 1", len(ZodiacList))
	}
	if ZodiacList[0].Name != "鼠" {
		t.Errorf("Loaded name = %q, want 鼠", ZodiacList[0].Name)
	}
}

// TestReloadFromJSONCoverage tests ReloadFromJSON
func TestReloadFromJSONCoverage(t *testing.T) {
	defer restoreZodiacList()

	// Use the same temp file setup
	tmpDir := t.TempDir()
	jsonContent := `[
		{"id": 2, "name": "牛", "wuxing": "土", "compatible": ["鼠"], "conflicting": [], "avoid_chars": "", "lucky_number": [1], "lucky_color": "黄", "lucky_direction": "东", "good_pianpang": "艹", "bad_pianpang": "石"}
	]`
	err := os.WriteFile(tmpDir+"/zodiac.json", []byte(jsonContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	// Load
	err = LoadFromJSON(tmpDir)
	if err != nil {
		t.Fatalf("LoadFromJSON failed: %v", err)
	}

	// Reload
	err = ReloadFromJSON(tmpDir)
	if err != nil {
		t.Fatalf("ReloadFromJSON failed: %v", err)
	}

	// Verify
	if len(ZodiacList) != 1 {
		t.Errorf("Reloaded ZodiacList length = %d, want 1", len(ZodiacList))
	}
	if ZodiacList[0].Name != "牛" {
		t.Errorf("Reloaded name = %q, want 牛", ZodiacList[0].Name)
	}
}

// TestGetZodiacCoverage tests GetZodiac (thread-safe getter)
func TestGetZodiacCoverage(t *testing.T) {
	defer restoreZodiacList()
	// Test existing
	z, ok := GetZodiac("鼠")
	if !ok {
		t.Error("GetZodiac(鼠) should return ok=true")
	}
	if z.Name != "鼠" {
		t.Errorf("GetZodiac(鼠) = %q, want 鼠", z.Name)
	}

	// Test non-existing
	z2, ok2 := GetZodiac("不存在")
	if ok2 {
		t.Error("GetZodiac(不存在) should return ok=false")
	}
	if z2.Name != "" {
		t.Errorf("GetZodiac(不存在) Name = %q, want empty", z2.Name)
	}
}

// TestZodiacMatchScoreBounds tests score bounds
func TestZodiacMatchScoreBounds(t *testing.T) {
	defer restoreZodiacList()
	// Test max score cap
	match := MatchNameWithZodiac("米米米米米米米米米米", "鼠") // 10 good chars
	if match == nil {
		t.Fatal("Match should not be nil")
	}
	if match.PianpangScore > 100 {
		t.Errorf("Score should be capped at 100, got %d", match.PianpangScore)
	}

	// Test min score floor
	match2 := MatchNameWithZodiac("刀刀刀刀刀刀刀刀刀刀", "鼠") // 10 bad chars
	if match2 == nil {
		t.Fatal("Match should not be nil")
	}
	if match2.PianpangScore < 0 {
		t.Errorf("Score should be floored at 0, got %d", match2.PianpangScore)
	}
}

// TestZodiacMatchDescriptions tests description generation
func TestZodiacMatchDescriptions(t *testing.T) {
	defer restoreZodiacList()
	// Only good
	m1 := MatchNameWithZodiac("米", "鼠")
	if m1 == nil {
		t.Fatal("Match should not be nil")
	}
	if m1.PianpangDesc == "" {
		t.Error("Description should not be empty")
	}
	if !contains(m1.PianpangDesc, "喜用偏旁") && !contains(m1.PianpangDesc, "吉利") {
		t.Errorf("Description for good only: %q", m1.PianpangDesc)
	}

	// Only bad
	m2 := MatchNameWithZodiac("刀", "鼠")
	if m2 == nil {
		t.Fatal("Match should not be nil")
	}
	if m2.PianpangDesc == "" {
		t.Error("Description should not be empty")
	}
	if !contains(m2.PianpangDesc, "忌用偏旁") && !contains(m2.PianpangDesc, "不利") {
		t.Errorf("Description for bad only: %q", m2.PianpangDesc)
	}

	// Mixed
	m3 := MatchNameWithZodiac("米刀", "鼠")
	if m3 == nil {
		t.Fatal("Match should not be nil")
	}
	if m3.PianpangDesc == "" {
		t.Error("Description should not be empty")
	}
	if !contains(m3.PianpangDesc, "喜用偏旁") || !contains(m3.PianpangDesc, "忌用偏旁") {
		t.Errorf("Description for mixed: %q", m3.PianpangDesc)
	}

	// Neither
	m4 := MatchNameWithZodiac("王", "鼠")
	if m4 == nil {
		t.Fatal("Match should not be nil")
	}
	if m4.PianpangDesc == "" {
		t.Error("Description should not be empty")
	}
	if !contains(m4.PianpangDesc, "无特殊关联") {
		t.Errorf("Description for neither: %q", m4.PianpangDesc)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}