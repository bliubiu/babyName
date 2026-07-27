package yijing

import (
	"testing"
)

// --- StrokesToTrigram 测试 ---

func TestStrokesToTrigram_One(t *testing.T) {
	// 1%8=1 → 乾(1)
	if got := StrokesToTrigram(1); got != 1 {
		t.Errorf("StrokesToTrigram(1) = %d, want 1(乾)", got)
	}
}

func TestStrokesToTrigram_Zero(t *testing.T) {
	// 0%8=0 → 坤(8)
	if got := StrokesToTrigram(0); got != 8 {
		t.Errorf("StrokesToTrigram(0) = %d, want 8(坤)", got)
	}
}

func TestStrokesToTrigram_Eight(t *testing.T) {
	// 8%8=0 → 坤(8)
	if got := StrokesToTrigram(8); got != 8 {
		t.Errorf("StrokesToTrigram(8) = %d, want 8(坤)", got)
	}
}

func TestStrokesToTrigram_AllValues(t *testing.T) {
	tests := []struct {
		strokes int
		want    int
	}{
		{1, 1}, {2, 2}, {3, 3}, {4, 4},
		{5, 5}, {6, 6}, {7, 7}, {8, 8},
		{9, 1}, {10, 2}, {11, 3}, {12, 4},
		{13, 5}, {14, 6}, {15, 7}, {16, 8},
	}
	for _, tc := range tests {
		if got := StrokesToTrigram(tc.strokes); got != tc.want {
			t.Errorf("StrokesToTrigram(%d) = %d, want %d", tc.strokes, got, tc.want)
		}
	}
}

// --- GenerateYaoLines 测试 ---

func TestGenerateYaoLines_QianKun(t *testing.T) {
	// 乾(1)上, 坤(8)下
	lines := GenerateYaoLines(1, 8)
	// 下坤: 8,8,8  上乾: 7,7,7
	expected := [6]int{8, 8, 8, 7, 7, 7}
	if lines != expected {
		t.Errorf("GenerateYaoLines(上乾下坤) = %v, want %v", lines, expected)
	}
}

func TestGenerateYaoLines_LiKan(t *testing.T) {
	// 离(3)上, 坎(6)下
	lines := GenerateYaoLines(3, 6)
	// 下坎: 8,7,8  上离: 7,8,7
	expected := [6]int{8, 7, 8, 7, 8, 7}
	if lines != expected {
		t.Errorf("GenerateYaoLines(上离下坎) = %v, want %v", lines, expected)
	}
}

func TestGenerateYaoLines_InvalidID(t *testing.T) {
	lines := GenerateYaoLines(0, 1)
	if lines != [6]int{} {
		t.Errorf("GenerateYaoLines with invalid ID should return zeros, got %v", lines)
	}
}

// --- CalcChangedHexagram 测试 ---

func TestCalcChangedHexagram_FirstYao(t *testing.T) {
	// 上乾(1)下坤(8)，动初爻(1)
	// 初爻由少阴(8)→少阳(7)，下卦爻变{7,8,8}=艮(7)
	upper, lower := CalcChangedHexagram(1, 8, 1)
	// yaoToTrigram[{7,8,8}] = 7(艮)
	if lower != 7 {
		t.Errorf("下卦 = %d, want 7(艮)", lower)
	}
	if upper != 1 {
		t.Errorf("上卦 = %d, want 1(乾)", upper)
	}
}

func TestCalcChangedHexagram_NoChange(t *testing.T) {
	// 动爻超出范围(0)，应返回原卦
	upper, lower := CalcChangedHexagram(1, 8, 0)
	if upper != 1 || lower != 8 {
		t.Errorf("CalcChangedHexagram with movingYao=0 should return original: got (%d,%d), want (1,8)", upper, lower)
	}
}

func TestCalcChangedHexagram_AllYao(t *testing.T) {
	// 验证每一爻变动都能正确计算
	for movingYao := 1; movingYao <= 6; movingYao++ {
		upper, lower := CalcChangedHexagram(1, 8, movingYao)
		if upper == 0 || lower == 0 {
			t.Errorf("CalcChangedHexagram(1,8,%d) returned invalid: (%d,%d)", movingYao, upper, lower)
		}
		// 变卦应与本卦不同（至少有一爻变）
		if upper == 1 && lower == 8 {
			t.Errorf("动爻%d未产生变化", movingYao)
		}
	}
}

// --- CalcInterHexagram 测试 ---

func TestCalcInterHexagram_Kun(t *testing.T) {
	// 坤(8)上坤(8)下，互卦应为坤(8)上坤(8)下
	// 爻线全为少阴(8)，互卦2-4、3-5也全阴
	upper, lower := CalcInterHexagram(8, 8)
	if upper != 8 || lower != 8 {
		t.Errorf("坤为地互卦 = (%d,%d), want (8,8)", upper, lower)
	}
}

func TestCalcInterHexagram_Qian(t *testing.T) {
	// 乾(1)上乾(1)下，互卦应为乾(1)上乾(1)下
	upper, lower := CalcInterHexagram(1, 1)
	if upper != 1 || lower != 1 {
		t.Errorf("乾为天互卦 = (%d,%d), want (1,1)", upper, lower)
	}
}

func TestCalcInterHexagram_LiKan(t *testing.T) {
	// 上离(3)下坎(6)，即未济卦
	upper, lower := CalcInterHexagram(3, 6)
	if upper == 0 || lower == 0 {
		t.Errorf("CalcInterHexagram(3,6) returned invalid: (%d,%d)", upper, lower)
	}
}

// --- GetHexagramByPlumBlossom 集成测试 ---
// 注意: 此测试依赖 hanzi 包的 HanziData 数据

func TestGetHexagramByPlumBlossom_Invalid(t *testing.T) {
	// 姓为空 → 返回 nil
	result := GetHexagramByPlumBlossom("", "明", "")
	if result != nil {
		t.Error("GetHexagramByPlumBlossom with empty surname should return nil")
	}
}

func TestGetHexagramByPlumBlossom_SingleName(t *testing.T) {
	// "王"=4画 → 上卦4(震), "小"=3画 → 下卦3(离), 总=4+3+0=7 → 动爻7%6=1
	result := GetHexagramByPlumBlossom("王", "小", "")
	if result == nil {
		t.Fatal("GetHexagramByPlumBlossom returned nil")
	}
	if result.UpperTrigramID == 0 || result.LowerTrigramID == 0 {
		t.Error("上卦/下卦不应为0")
	}
	if result.MovingYao < 1 || result.MovingYao > 6 {
		t.Errorf("动爻应在1-6之间, got %d", result.MovingYao)
	}
	if result.OriginalHexagram == nil || result.ChangedHexagram == nil {
		t.Error("本卦/变卦不应为nil")
	}
	if result.Interpretation == "" {
		t.Error("综合解读不应为空")
	}
}

func TestGetHexagramByPlumBlossom_DoubleName(t *testing.T) {
	// "李"=7画, "小"=3画, "明"=8画
	// 上卦=7%8=7(艮), 下卦=3%8=3(离)
	result := GetHexagramByPlumBlossom("李", "小", "明")
	if result == nil {
		t.Fatal("GetHexagramByPlumBlossom returned nil")
	}
	if result.OriginalHexagram == nil {
		t.Error("本卦不应为nil")
	}
}

// --- GenerateHexagramInterpretation 测试 ---

func TestGenerateHexagramInterpretation_NotEmpty(t *testing.T) {
	original := GetHexagramByNumber(1) // 乾
	changed := GetHexagramByNumber(44) // 姤
	inter := GetHexagramByNumber(1)    // 乾

	text := GenerateHexagramInterpretation(original, changed, inter, 1, 1, 3, [6]int{7, 7, 7, 7, 7, 7})
	if text == "" {
		t.Error("GenerateHexagramInterpretation should not be empty")
	}
	if len(text) < 20 {
		t.Errorf("解读太短: %q", text)
	}
}

// --- 八卦爻映射完整性测试 ---

func TestTrigramYaoMaps_AllEight(t *testing.T) {
	if len(trigramYaoMaps) != 8 {
		t.Errorf("trigramYaoMaps length = %d, want 8", len(trigramYaoMaps))
	}
	for id, yao := range trigramYaoMaps {
		for i, v := range yao {
			if v != 7 && v != 8 {
				t.Errorf("trigramYaoMaps[%d][%d] = %d, want 7(少阳) or 8(少阴)", id, i, v)
			}
		}
	}
}

func TestYaoToTrigram_Bidirectional(t *testing.T) {
	// verify双向映射一致
	for id, yao := range trigramYaoMaps {
		mapped := yaoToTrigram[yao]
		if mapped != id {
			t.Errorf("trigramYaoMaps[%d] -> yaoToTrigraph -> %d, not bidirectional", id, mapped)
		}
	}
}

func TestShengKeMap_Complete(t *testing.T) {
	wuxing := []string{"木", "火", "土", "金", "水"}
	for _, wx := range wuxing {
		if _, ok := shengMap[wx]; !ok {
			t.Errorf("shengMap missing %q", wx)
		}
		if _, ok := keMap[wx]; !ok {
			t.Errorf("keMap missing %q", wx)
		}
	}
	// 验证五行循环：木→火→土→金→水→木
	if shengMap["木"] != "火" {
		t.Errorf("shengMap[木] = %q, want 火", shengMap["木"])
	}
	if keMap["木"] != "土" {
		t.Errorf("keMap[木] = %q, want 土", keMap["木"])
	}
}
