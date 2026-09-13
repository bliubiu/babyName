package yijing

import "testing"

func TestTrigramFromStrokes(t *testing.T) {
	cases := map[int]int{
		1: 1, 2: 2, 7: 7, 8: 8, 9: 1, 16: 8, 0: 1, -1: 1,
	}
	for strokes, want := range cases {
		if got := TrigramFromStrokes(strokes); got != want {
			t.Errorf("TrigramFromStrokes(%d) = %d, want %d", strokes, got, want)
		}
	}
}

func TestGetHexagramByMeihuaName_SurnameUpper(t *testing.T) {
	// 姓 4 画 → 震(4)，名 6 画 → 坎(6) → 震上坎下 = 解
	h := GetHexagramByMeihuaName(4, 6)
	if h == nil {
		t.Fatal("卦象为空")
	}
	if h.Name != "解" {
		t.Errorf("卦名 = %q, want 解（震上坎下）", h.Name)
	}
}

func TestGetHexagramByMeihuaName_ZeroStrokes(t *testing.T) {
	h := GetHexagramByMeihuaName(0, 0)
	if h == nil {
		t.Fatal("卦象为空")
	}
	if h.Name != "乾" {
		t.Errorf("0画应对乾，实际 %q", h.Name)
	}
}

func TestGetHexagramByMeihuaName_Eight(t *testing.T) {
	h := GetHexagramByMeihuaName(8, 16)
	if h.Name != "坤" {
		t.Errorf("8/16 画应对坤，实际 %q", h.Name)
	}
}

func TestGetHexagramByMeihuaName_KnownPairs(t *testing.T) {
	cases := []struct {
		sur, given int
		want       string
	}{
		{1, 1, "乾"},
		{1, 8, "否"},
		{8, 1, "泰"},
		{3, 3, "离"},
		{6, 4, "屯"}, // 坎上震下
	}
	for _, tc := range cases {
		h := GetHexagramByMeihuaName(tc.sur, tc.given)
		if h == nil || h.Name != tc.want {
			t.Errorf("Meihua(%d,%d) = %v, want %s", tc.sur, tc.given, h, tc.want)
		}
	}
}

func TestGetHexagramByMeihuaName_AllPairsNonNil(t *testing.T) {
	for s := 0; s <= 16; s++ {
		for g := 0; g <= 16; g++ {
			if GetHexagramByMeihuaName(s, g) == nil {
				t.Fatalf("Meihua(%d,%d) 返回 nil", s, g)
			}
		}
	}
}
