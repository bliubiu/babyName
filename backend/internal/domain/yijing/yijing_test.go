package yijing

import (
	"testing"
)

// --- 八卦映射表测试 ---

func TestTrigramMap_AllEight(t *testing.T) {
	if len(TrigramMap) != 8 {
		t.Errorf("TrigramMap length = %d, want 8", len(TrigramMap))
	}
}

func TestTrigramMap_Symbols(t *testing.T) {
	tests := []struct {
		id     int
		symbol string
	}{
		{1, "☰"}, {2, "☱"}, {3, "☲"}, {4, "☳"},
		{5, "☴"}, {6, "☵"}, {7, "☶"}, {8, "☷"},
	}
	for _, tc := range tests {
		if got := TrigramMap[tc.id]; got != tc.symbol {
			t.Errorf("TrigramMap[%d] = %q, want %q", tc.id, got, tc.symbol)
		}
	}
}

// --- 八卦名称表测试 ---

func TestTrigramNameMap_AllEight(t *testing.T) {
	if len(TrigramNameMap) != 8 {
		t.Errorf("TrigramNameMap length = %d, want 8", len(TrigramNameMap))
	}
}

func TestTrigramNameMap_Names(t *testing.T) {
	tests := []struct {
		id   int
		name string
	}{
		{1, "乾"}, {2, "兑"}, {3, "离"}, {4, "震"},
		{5, "巽"}, {6, "坎"}, {7, "艮"}, {8, "坤"},
	}
	for _, tc := range tests {
		if got := TrigramNameMap[tc.id]; got != tc.name {
			t.Errorf("TrigramNameMap[%d] = %q, want %q", tc.id, got, tc.name)
		}
	}
}

// --- 八卦五行表测试 ---

func TestTrigramWuxingMap_AllEight(t *testing.T) {
	if len(TrigramWuxingMap) != 8 {
		t.Errorf("TrigramWuxingMap length = %d, want 8", len(TrigramWuxingMap))
	}
}

func TestTrigramWuxingMap_ValidWuxing(t *testing.T) {
	validWuxing := map[string]bool{"金": true, "木": true, "水": true, "火": true, "土": true}
	for id, wx := range TrigramWuxingMap {
		if !validWuxing[wx] {
			t.Errorf("TrigramWuxingMap[%d] = %q, not a valid 五行", id, wx)
		}
	}
}

// --- 八卦象征意义表测试 ---

func TestTrigramMeaningMap_AllEight(t *testing.T) {
	if len(TrigramMeaningMap) != 8 {
		t.Errorf("TrigramMeaningMap length = %d, want 8", len(TrigramMeaningMap))
	}
}

// --- 爻结果映射表测试 ---

func TestYaoMap_AllFour(t *testing.T) {
	if len(YaoMap) != 4 {
		t.Errorf("YaoMap length = %d, want 4", len(YaoMap))
	}
}

func TestYaoMap_Values(t *testing.T) {
	tests := []struct {
		yao     int
		name    string
		yaoName string
	}{
		{6, "老阴", "六"},
		{7, "少阳", "七"},
		{8, "少阴", "八"},
		{9, "老阳", "九"},
	}
	for _, tc := range tests {
		got, ok := YaoMap[tc.yao]
		if !ok {
			t.Errorf("YaoMap missing entry for %d", tc.yao)
			continue
		}
		if got.Name != tc.name {
			t.Errorf("YaoMap[%d].Name = %q, want %q", tc.yao, got.Name, tc.name)
		}
		if got.YaoName != tc.yaoName {
			t.Errorf("YaoMap[%d].YaoName = %q, want %q", tc.yao, got.YaoName, tc.yaoName)
		}
	}
}

// --- 64卦数据完整性测试（通过 loader 加载的 HexagramList）---

func TestHexagramList_All64(t *testing.T) {
	// 测试通过访问全局 hexagramList 来验证 64 卦
	if len(HexagramList) != 64 {
		t.Errorf("HexagramList length = %d, want 64", len(HexagramList))
	}
}

func TestHexagramList_NoDuplicateIDs(t *testing.T) {
	seen := make(map[int]bool)
	for _, h := range HexagramList {
		if seen[h.ID] {
			t.Errorf("Duplicate hexagram ID: %d (%s)", h.ID, h.Name)
		}
		seen[h.ID] = true
	}
}

func TestHexagramList_NoDuplicateNumbers(t *testing.T) {
	seen := make(map[int]bool)
	for _, h := range HexagramList {
		if seen[h.Number] {
			t.Errorf("Duplicate hexagram number: %d (%s)", h.Number, h.Name)
		}
		seen[h.Number] = true
	}
}

func TestHexagramList_IDRange(t *testing.T) {
	for _, h := range HexagramList {
		if h.ID < 1 || h.ID > 64 {
			t.Errorf("Hexagram %q has ID %d, want 1-64", h.Name, h.ID)
		}
	}
}

func TestHexagramList_NumberRange(t *testing.T) {
	for _, h := range HexagramList {
		if h.Number < 1 || h.Number > 64 {
			t.Errorf("Hexagram %q has Number %d, want 1-64", h.Name, h.Number)
		}
	}
}

func TestHexagramList_HasRequiredFields(t *testing.T) {
	for _, h := range HexagramList {
		if h.Name == "" {
			t.Errorf("Hexagram ID %d has empty Name", h.ID)
		}
		if h.Symbol == "" {
			t.Errorf("Hexagram %q has empty Symbol", h.Name)
		}
		if h.GuaCi == "" {
			t.Errorf("Hexagram %q has empty GuaCi", h.Name)
		}
	}
}

func TestHexagramList_TrigramRange(t *testing.T) {
	for _, h := range HexagramList {
		if h.UpperTrigram < 1 || h.UpperTrigram > 8 {
			t.Errorf("Hexagram %q has upper_trigram %d, want 1-8", h.Name, h.UpperTrigram)
		}
		if h.LowerTrigram < 1 || h.LowerTrigram > 8 {
			t.Errorf("Hexagram %q has lower_trigram %d, want 1-8", h.Name, h.LowerTrigram)
		}
	}
}

func TestHexagramList_SpecificKnownHexagrams(t *testing.T) {
	// 根据 ID 查找特定卦象（loader 用 ID 做索引）
	known := map[int]string{
		1:  "乾",
		2:  "坤",
		3:  "屯",
		4:  "蒙",
		11: "泰",
		12: "否",
		63: "既济",
		64: "未济",
	}

	for id, expectedName := range known {
		found := false
		for _, h := range HexagramList {
			if h.ID == id {
				found = true
				if h.Name != expectedName {
					t.Errorf("Hexagram ID %d: Name = %q, want %q", id, h.Name, expectedName)
				}
				break
			}
		}
		if !found {
			t.Errorf("Hexagram ID %d not found in HexagramList", id)
		}
	}
}
