package bazi

import (
	"testing"
)

// --- GetTenGod 基础测试 ---

func TestGetTenGod_SameStem(t *testing.T) {
	// 日干"甲"对"甲" → 比肩
	got := GetTenGod("甲", "甲")
	if got != "比肩" {
		t.Errorf("GetTenGod(甲, 甲) = %q, want 比肩", got)
	}
}

func TestGetTenGod_YearStem(t *testing.T) {
	// 日干"戊"对年干"甲" → 七杀（木克土，同性）
	got := GetTenGod("戊", "甲")
	if got != "七杀" {
		t.Errorf("GetTenGod(戊, 甲) = %q, want 七杀", got)
	}
}

func TestGetTenGod_MonthStem(t *testing.T) {
	// 日干"丁"对月干"乙" → 偏印（木生火，同性）
	got := GetTenGod("丁", "乙")
	if got != "偏印" {
		t.Errorf("GetTenGod(丁, 乙) = %q, want 偏印", got)
	}
}

func TestGetTenGod_EmptyStem(t *testing.T) {
	if got := GetTenGod("", "甲"); got != "" {
		t.Errorf("GetTenGod(空, 甲) = %q, want 空字符串", got)
	}
	if got := GetTenGod("甲", ""); got != "" {
		t.Errorf("GetTenGod(甲, 空) = %q, want 空字符串", got)
	}
}

func TestGetTenGod_InvalidStem(t *testing.T) {
	if got := GetTenGod("甲", "X"); got != "" {
		t.Errorf("GetTenGod(甲, X) = %q, want 空字符串", got)
	}
}

// --- CalculateTenGods 测试 ---

func TestCalculateTenGods_Default(t *testing.T) {
	// 日干戊，年甲月丙日戊时壬
	result := CalculateTenGods("戊", "甲", "丙", "壬")
	if result == nil {
		t.Fatal("CalculateTenGods returned nil")
	}
	if result.Day != "比肩" {
		t.Errorf("日干十神 = %q, want 比肩 (自身)", result.Day)
	}
	if result.Year == "" {
		t.Error("年干十神不应为空")
	}
	if result.Month == "" {
		t.Error("月干十神不应为空")
	}
	if result.Hour == "" {
		t.Error("时干十神不应为空")
	}
}

func TestCalculateTenGods_EmptyDayStem(t *testing.T) {
	if got := CalculateTenGods("", "甲", "乙", "丙"); got != nil {
		t.Error("CalculateTenGods with empty dayStem should return nil")
	}
}

// --- TenGodMeaning 映射表测试 ---

func TestTenGodMeaning_AllTen(t *testing.T) {
	expectedGods := []string{"比肩", "劫财", "食神", "伤官", "偏财", "正财", "七杀", "正官", "偏印", "正印"}
	for _, god := range expectedGods {
		if _, ok := TenGodMeaning[god]; !ok {
			t.Errorf("TenGodMeaning missing entry for %q", god)
		}
	}
}

func TestTenGodMeaning_NonEmpty(t *testing.T) {
	for god, meaning := range TenGodMeaning {
		if meaning == "" {
			t.Errorf("TenGodMeaning[%q] is empty", god)
		}
	}
}

// --- CalculatePillarTenGods 测试 ---

func TestCalculatePillarTenGods_NilBazi(t *testing.T) {
	if got := CalculatePillarTenGods(nil, "甲"); got != nil {
		t.Error("CalculatePillarTenGods with nil bazi should return nil")
	}
}

func TestCalculatePillarTenGods_EmptyDayStem(t *testing.T) {
	bazi := &Bazi{
		YearGanzhi:  "甲子",
		MonthGanzhi: "丙寅",
		DayGanzhi:   "戊辰",
		HourGanzhi:  "庚午",
	}
	if got := CalculatePillarTenGods(bazi, ""); got != nil {
		t.Error("CalculatePillarTenGods with empty dayStem should return nil")
	}
}

func TestCalculatePillarTenGods_Basic(t *testing.T) {
	bazi := &Bazi{
		YearGanzhi:  "甲子",
		MonthGanzhi: "丙寅",
		DayGanzhi:   "戊辰",
		HourGanzhi:  "庚午",
	}
	result := CalculatePillarTenGods(bazi, "戊")
	if result == nil {
		t.Fatal("CalculatePillarTenGods returned nil")
	}
	if result.YearStem != "甲" {
		t.Errorf("年干 = %q, want 甲", result.YearStem)
	}
	if result.DayStem != "戊" {
		t.Errorf("日干 = %q, want 戊", result.DayStem)
	}
	if result.YearGod == "" || result.MonthGod == "" || result.DayGod == "" || result.HourGod == "" {
		t.Error("十神不应有空值")
	}
	// 应有地支藏干（有日支辰，应包含藏干）
	if len(result.HideHeaven) == 0 {
		t.Error("地支藏干不应为空")
	}
}
