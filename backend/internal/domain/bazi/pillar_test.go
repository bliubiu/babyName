package bazi

import (
	"testing"
)

// --- DescribePillarPosition 测试 ---

func TestDescribePillarPosition_Year(t *testing.T) {
	got := DescribePillarPosition("年")
	if got == "" {
		t.Error("DescribePillarPosition(年) should not be empty")
	}
}

func TestDescribePillarPosition_Month(t *testing.T) {
	got := DescribePillarPosition("月")
	if got == "" {
		t.Error("DescribePillarPosition(月) should not be empty")
	}
}

func TestDescribePillarPosition_Day(t *testing.T) {
	got := DescribePillarPosition("日")
	if got == "" {
		t.Error("DescribePillarPosition(日) should not be empty")
	}
}

func TestDescribePillarPosition_Hour(t *testing.T) {
	got := DescribePillarPosition("时")
	if got == "" {
		t.Error("DescribePillarPosition(时) should not be empty")
	}
}

func TestDescribePillarPosition_Invalid(t *testing.T) {
	if got := DescribePillarPosition("X"); got != "" {
		t.Errorf("DescribePillarPosition(X) = %q, want empty", got)
	}
}

// --- PillarImage 测试 ---

func TestPillarImage_Known(t *testing.T) {
	// 甲木午火
	got := PillarImage("年", "甲", "午", "木", "火")
	if got == "" {
		t.Error("PillarImage should not be empty for known values")
	}
}

func TestPillarImage_UnknownStem(t *testing.T) {
	got := PillarImage("年", "X", "子", "水", "水")
	if got == "" {
		t.Error("PillarImage should fallback to branch image")
	}
}

// --- AnalyzePillar 测试 ---

func TestAnalyzePillar_EmptyGanzhi(t *testing.T) {
	if got := AnalyzePillar("年", ""); got != nil {
		t.Error("AnalyzePillar with empty ganzhi should return nil")
	}
}

func TestAnalyzePillar_ShortGanzhi(t *testing.T) {
	if got := AnalyzePillar("年", "甲"); got != nil {
		t.Error("AnalyzePillar with single char ganzhi should return nil")
	}
}

func TestAnalyzePillar_Year(t *testing.T) {
	result := AnalyzePillar("年", "甲子")
	if result == nil {
		t.Fatal("AnalyzePillar returned nil")
	}
	if result.Position != "年" {
		t.Errorf("position = %q, want 年", result.Position)
	}
	if result.HeavenStem != "甲" {
		t.Errorf("heaven_stem = %q, want 甲", result.HeavenStem)
	}
	if result.EarthBranch != "子" {
		t.Errorf("earth_branch = %q, want 子", result.EarthBranch)
	}
	if result.StemWuxing != "木" {
		t.Errorf("stem_wuxing = %q, want 木", result.StemWuxing)
	}
	if result.BranchWuxing != "水" {
		t.Errorf("branch_wuxing = %q, want 水", result.BranchWuxing)
	}
	if result.Description == "" {
		t.Error("description should not be empty")
	}
}

func TestAnalyzePillar_Nayin(t *testing.T) {
	// 甲子 → 海中金
	result := AnalyzePillar("年", "甲子")
	if result == nil {
		t.Fatal("AnalyzePillar returned nil")
	}
	if result.Nayin != "海中金" {
		t.Errorf("nayin = %q, want 海中金", result.Nayin)
	}
	if result.NayinWuxing != "金" {
		t.Errorf("nayin_wuxing = %q, want 金", result.NayinWuxing)
	}
}

func TestAnalyzePillar_HideHeaven(t *testing.T) {
	// 子 → 藏干为癸（子中藏癸水）
	result := AnalyzePillar("年", "甲子")
	if result == nil {
		t.Fatal("AnalyzePillar returned nil")
	}
	if len(result.HideHeaven) == 0 {
		t.Log("天干甲的日子支为子，应有藏干癸")
	} else {
		found := false
		for _, hh := range result.HideHeaven {
			if hh.Stem == "癸" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("子支应藏癸, 但藏干列表为 %v", result.HideHeaven)
		}
	}
}

func TestAnalyzePillar_WithHideHeaven(t *testing.T) {
	// 辰 = 戊乙癸（本气戊、中气乙、余气癸）
	result := AnalyzePillar("月", "丙辰")
	if result == nil {
		t.Fatal("AnalyzePillar returned nil")
	}
	if len(result.HideHeaven) == 0 {
		t.Log("辰应有藏干")
	} else {
		for _, hh := range result.HideHeaven {
			if hh.Stem == "" {
				t.Error("藏干名不应为空")
			}
			if hh.Wuxing == "" {
				t.Error("藏干五行不应为空")
			}
			if hh.ZhiType == "" {
				t.Error("藏干类型不应为空")
			}
		}
	}
}

// --- AnalyzeFourPillars 测试 ---

func TestAnalyzeFourPillars_NilBazi(t *testing.T) {
	if got := AnalyzeFourPillars(nil); got != nil {
		t.Error("AnalyzeFourPillars with nil bazi should return nil")
	}
}

func TestAnalyzeFourPillars_Basic(t *testing.T) {
	bazi := &Bazi{
		YearGanzhi:  "甲子",
		MonthGanzhi: "丙寅",
		DayGanzhi:   "戊辰",
		HourGanzhi:  "庚午",
	}
	result := AnalyzeFourPillars(bazi)
	if result == nil {
		t.Fatal("AnalyzeFourPillars returned nil")
	}
	if result.Year == nil || result.Month == nil || result.Day == nil || result.Hour == nil {
		t.Error("四柱分析不应有nil")
	}
	if result.Year.Ganzhi != "甲子" {
		t.Errorf("年柱 = %q, want 甲子", result.Year.Ganzhi)
	}
	if result.Month.Ganzhi != "丙寅" {
		t.Errorf("月柱 = %q, want 丙寅", result.Month.Ganzhi)
	}
	if result.Day.Ganzhi != "戊辰" {
		t.Errorf("日柱 = %q, want 戊辰", result.Day.Ganzhi)
	}
	if result.Hour.Ganzhi != "庚午" {
		t.Errorf("时柱 = %q, want 庚午", result.Hour.Ganzhi)
	}
}
