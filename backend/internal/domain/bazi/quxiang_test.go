package bazi

import (
	"testing"
)

// --- GenerateQuxiang 基础测试 ---

func TestGenerateQuxiang_Nil(t *testing.T) {
	if got := GenerateQuxiang(nil); got != nil {
		t.Error("GenerateQuxiang(nil) should return nil")
	}
}

func TestGenerateQuxiang_WithAnalysis(t *testing.T) {
	analysis := &BaziAnalysis{
		Bazi: Bazi{
			YearGanzhi:  "甲子",
			MonthGanzhi: "丙寅",
			DayGanzhi:   "戊辰",
			HourGanzhi:  "庚午",
		},
		Rishou:           "戊",
		RishouWuxing:     "土",
		DayMaster:        "土",
		DayMasterStrength: "身中",
		Wuxing:           WuxingResult{Jin: 2, Mu: 1, Shui: 1, Huo: 2, Tu: 4},
		Xiyongshen:       []string{"金", "水"},
		Yiyongshen:       []string{"木", "火"},
		Nayin:            "海中金",
	}

	result := GenerateQuxiang(analysis)
	if result == nil {
		t.Fatal("GenerateQuxiang returned nil")
	}
	if result.WuxingImage == nil {
		t.Fatal("五行取象不应为nil")
	}
	if result.WuxingImage.Layer != LayerWuxing {
		t.Errorf("layer = %q, want %q", result.WuxingImage.Layer, LayerWuxing)
	}
	if result.WuxingImage.Title != "五行取象" {
		t.Errorf("title = %q, want 五行取象", result.WuxingImage.Title)
	}
	if result.WuxingImage.Content == "" {
		t.Error("content should not be empty")
	}
	if result.WuxingImage.Summary == "" {
		t.Error("summary should not be empty")
	}
	if result.FullProfile == "" {
		t.Error("full_profile should not be empty")
	}
}

// --- 五行取象层测试 ---

func TestGenerateWuxingImage_Content(t *testing.T) {
	analysis := &BaziAnalysis{
		Rishou:           "甲",
		RishouWuxing:     "木",
		DayMasterStrength: "身弱",
		Xiyongshen:       []string{"木", "水"},
		Wuxing:           WuxingResult{Jin: 1, Mu: 3, Shui: 2, Huo: 2, Tu: 2},
		Nayin:            "海中金",
	}
	img := generateWuxingImage(analysis)
	if img == nil {
		t.Fatal("generateWuxingImage returned nil")
	}
	if img.Content == "" {
		t.Error("五行取象内容不应为空")
	}
}

func TestGenerateWuxingImage_NoNayin(t *testing.T) {
	analysis := &BaziAnalysis{
		Rishou:           "庚",
		RishouWuxing:     "金",
		DayMasterStrength: "身旺",
		Xiyongshen:       []string{"火", "木"},
		Wuxing:           WuxingResult{Jin: 4, Mu: 1, Shui: 1, Huo: 1, Tu: 3},
	}
	img := generateWuxingImage(analysis)
	if img == nil {
		t.Fatal("generateWuxingImage returned nil")
	}
	// 不报错即可
}

// --- 十神取象层测试 ---

func TestBuildTenGodImage_Nil(t *testing.T) {
	if got := BuildTenGodImage(nil, nil); got != nil {
		t.Error("BuildTenGodImage(nil, nil) should return nil")
	}
	analysis := &BaziAnalysis{Rishou: "戊", RishouWuxing: "土"}
	if got := BuildTenGodImage(analysis, nil); got != nil {
		t.Error("BuildTenGodImage with nil tenGods should return nil")
	}
}

func TestBuildTenGodImage_Basic(t *testing.T) {
	analysis := &BaziAnalysis{Rishou: "戊", RishouWuxing: "土"}
	tenGods := &PillarTenGods{
		YearStem:  "甲",
		MonthStem: "丙",
		DayStem:   "戊",
		HourStem:  "庚",
		YearGod:   "七杀",
		MonthGod:  "偏印",
		DayGod:    "比肩",
		HourGod:   "食神",
	}
	img := BuildTenGodImage(analysis, tenGods)
	if img == nil {
		t.Fatal("BuildTenGodImage returned nil")
	}
	if img.Layer != LayerTenGod {
		t.Errorf("layer = %q, want %q", img.Layer, LayerTenGod)
	}
	if img.Content == "" {
		t.Error("内容不应为空")
	}
	if img.Summary == "" {
		t.Error("summary不应为空")
	}
}

func TestBuildTenGodImage_WithHideHeaven(t *testing.T) {
	analysis := &BaziAnalysis{Rishou: "戊", RishouWuxing: "土"}
	tenGods := &PillarTenGods{
		YearStem: "甲", MonthStem: "丙", DayStem: "戊", HourStem: "庚",
		YearGod: "七杀", MonthGod: "偏印", DayGod: "比肩", HourGod: "食神",
		HideHeaven: []HideHeavenGod{
			{Stem: "癸", TenGod: "正财", HideType: "本气"},
		},
	}
	img := BuildTenGodImage(analysis, tenGods)
	if img == nil {
		t.Fatal("BuildTenGodImage returned nil")
	}
	if img.Content == "" {
		t.Error("内容不应为空")
	}
}

// --- 神煞取象层测试 ---

func TestBuildShenshaImage_Nil(t *testing.T) {
	if got := BuildShenshaImage(nil); got != nil {
		t.Error("BuildShenshaImage(nil) should return nil")
	}
}

func TestBuildShenshaImage_Basic(t *testing.T) {
	shensha := &PillarShensha{
		DayGods:   []string{"天乙贵人", "文昌"},
		MonthGods: []string{"天德"},
		YearGods:  []string{"华盖"},
		HourGods:  []string{"桃花"},
	}
	img := BuildShenshaImage(shensha)
	if img == nil {
		t.Fatal("BuildShenshaImage returned nil")
	}
	if img.Content == "" {
		t.Error("内容不应为空")
	}
	if img.Summary == "" {
		t.Error("summary不应为空")
	}
}

func TestBuildShenshaImage_EmptyGods(t *testing.T) {
	shensha := &PillarShensha{}
	img := BuildShenshaImage(shensha)
	if img == nil {
		t.Fatal("BuildShenshaImage should not return nil even with empty gods")
	}
	// 神煞为空也应有默认内容
}

// --- 柱位取象层测试 ---

func TestBuildPillarImage_Nil(t *testing.T) {
	if got := BuildPillarImage(nil); got != nil {
		t.Error("BuildPillarImage(nil) should return nil")
	}
}

func TestBuildPillarImage_Basic(t *testing.T) {
	pillars := &FourPillarAnalysis{
		Year:  &PillarAnalysis{Position: "年", Ganzhi: "甲子", HeavenStem: "甲", EarthBranch: "子", StemWuxing: "木", BranchWuxing: "水", Nayin: "海中金", NayinWuxing: "金", Description: "年柱代表祖上", Image: "参天大树"},
		Month: &PillarAnalysis{Position: "月", Ganzhi: "丙寅", HeavenStem: "丙", EarthBranch: "寅", StemWuxing: "火", BranchWuxing: "木"},
		Day:   &PillarAnalysis{Position: "日", Ganzhi: "戊辰", HeavenStem: "戊", EarthBranch: "辰", StemWuxing: "土", BranchWuxing: "土"},
		Hour:  &PillarAnalysis{Position: "时", Ganzhi: "庚午", HeavenStem: "庚", EarthBranch: "午", StemWuxing: "金", BranchWuxing: "火"},
	}
	img := BuildPillarImage(pillars)
	if img == nil {
		t.Fatal("BuildPillarImage returned nil")
	}
	if img.Content == "" {
		t.Error("内容不应为空")
	}
	if img.Summary == "" {
		t.Error("summary不应为空")
	}
}

func TestBuildPillarImage_Partial(t *testing.T) {
	// 只有日柱
	pillars := &FourPillarAnalysis{
		Day: &PillarAnalysis{Position: "日", Ganzhi: "甲子", HeavenStem: "甲", EarthBranch: "子", StemWuxing: "木", BranchWuxing: "水"},
	}
	img := BuildPillarImage(pillars)
	if img == nil {
		t.Fatal("BuildPillarImage returned nil")
	}
	if img.Content == "" {
		t.Error("内容不应为空")
	}
}

// --- Setter 测试 ---

func TestSetTenGodImage(t *testing.T) {
	q := &QuxiangResult{}
	img := &LayerImage{Layer: LayerTenGod, Title: "十神取象", Content: "测试内容"}
	q.SetTenGodImage(img)
	if q.TenGodImage != img {
		t.Error("SetTenGodImage failed to set")
	}
	if q.FullProfile == "" {
		t.Error("after SetTenGodImage full_profile should be recomputed")
	}
}

func TestSetShenshaImage(t *testing.T) {
	q := &QuxiangResult{}
	img := &LayerImage{Layer: LayerShensha, Title: "神煞取象", Content: "测试神煞"}
	q.SetShenshaImage(img)
	if q.ShenshaImage != img {
		t.Error("SetShenshaImage failed to set")
	}
}

func TestSetPillarImage(t *testing.T) {
	q := &QuxiangResult{}
	img := &LayerImage{Layer: LayerPillar, Title: "柱位取象", Content: "测试柱位"}
	q.SetPillarImage(img)
	if q.PillarImage != img {
		t.Error("SetPillarImage failed to set")
	}
}

// --- 辅助函数测试 ---

func TestJoinStrings_Multiple(t *testing.T) {
	got := joinStrings([]string{"甲", "乙", "丙"}, "、")
	if got != "甲、乙、丙" {
		t.Errorf("joinStrings = %q, want 甲、乙、丙", got)
	}
}

func TestJoinStrings_Single(t *testing.T) {
	if got := joinStrings([]string{"甲"}, "、"); got != "甲" {
		t.Errorf("joinStrings single = %q, want 甲", got)
	}
}

func TestJoinStrings_Empty(t *testing.T) {
	if got := joinStrings([]string{}, "、"); got != "" {
		t.Errorf("joinStrings empty = %q, want empty", got)
	}
}

// --- GenerateFullProfile 集成测试 ---

func TestGenerateFullProfile_Basic(t *testing.T) {
	analysis := &BaziAnalysis{
		Bazi: Bazi{
			YearGanzhi: "甲子", MonthGanzhi: "丙寅",
			DayGanzhi: "戊辰", HourGanzhi: "庚午",
		},
		Rishou: "戊", RishouWuxing: "土",
		DayMaster: "土", DayMasterStrength: "身中",
		Wuxing:     WuxingResult{Jin: 2, Mu: 1, Shui: 1, Huo: 2, Tu: 4},
		Xiyongshen: []string{"金", "水"},
		Nayin:      "海中金",
	}
	tenGods := &PillarTenGods{
		YearStem: "甲", MonthStem: "丙", DayStem: "戊", HourStem: "庚",
		YearGod: "七杀", MonthGod: "偏印", DayGod: "比肩", HourGod: "食神",
	}
	shensha := &PillarShensha{
		DayGods: []string{"天乙贵人"},
	}
	pillars := &FourPillarAnalysis{
		Year:  &PillarAnalysis{Position: "年", Ganzhi: "甲子", HeavenStem: "甲", EarthBranch: "子", StemWuxing: "木", BranchWuxing: "水", Description: "年柱代表祖上"},
		Month: &PillarAnalysis{Position: "月", Ganzhi: "丙寅", HeavenStem: "丙", EarthBranch: "寅", StemWuxing: "火", BranchWuxing: "木"},
		Day:   &PillarAnalysis{Position: "日", Ganzhi: "戊辰", HeavenStem: "戊", EarthBranch: "辰", StemWuxing: "土", BranchWuxing: "土"},
		Hour:  &PillarAnalysis{Position: "时", Ganzhi: "庚午", HeavenStem: "庚", EarthBranch: "午", StemWuxing: "金", BranchWuxing: "火"},
	}

	profile := GenerateFullProfile(analysis, tenGods, shensha, pillars)
	if profile == "" {
		t.Error("GenerateFullProfile should not return empty")
	}
	if len(profile) < 50 {
		t.Errorf("profile too short (%d chars): %q", len(profile), profile)
	}
}

func TestGenerateFullProfile_Partial(t *testing.T) {
	// 只有八字分析，没有十神/神煞/柱位
	analysis := &BaziAnalysis{
		Bazi: Bazi{YearGanzhi: "甲子"},
		Rishou: "甲", RishouWuxing: "木",
		DayMasterStrength: "身弱",
		Wuxing: WuxingResult{Mu: 3},
	}
	profile := GenerateFullProfile(analysis, nil, nil, nil)
	if profile == "" {
		t.Error("GenerateFullProfile with nil sub-results should still return basic profile")
	}
}
