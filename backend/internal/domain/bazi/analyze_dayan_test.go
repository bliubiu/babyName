package bazi

// analyze_dayan_test.go — 大衍起卦可复现性守护（docs/43 P2-5）
//
// 背景：八字分析此前无条件调用真随机的 yijing.CastHexagramByDayan，
// 同一生辰两次分析返回不同 DayanHexagram，破坏结果可复现与缓存前提。
// 修复：按生辰确定性播种（CastHexagramByDayanSeeded），同一生辰恒得同卦。

import "testing"

// TestAnalyzeBazi_DayanReproducible 同一生辰两次分析的大衍卦必须一致
func TestAnalyzeBazi_DayanReproducible(t *testing.T) {
	a, err := AnalyzeBazi(2024, 5, 20, 10, 0)
	if err != nil {
		t.Fatalf("首次分析失败: %v", err)
	}
	b, err := AnalyzeBazi(2024, 5, 20, 10, 0)
	if err != nil {
		t.Fatalf("二次分析失败: %v", err)
	}

	if a.DayanHexagram != b.DayanHexagram {
		t.Errorf("同一生辰大衍卦不一致: %q vs %q（起卦仍为真随机）", a.DayanHexagram, b.DayanHexagram)
	}
	if a.DayanInterpretation != b.DayanInterpretation {
		t.Error("同一生辰大衍解读不一致")
	}
}

// TestAnalyzeBazi_DayanVariesAcrossBirths 不同生辰应得到不同卦（避免全库一卦）
func TestAnalyzeBazi_DayanVariesAcrossBirths(t *testing.T) {
	seen := map[string]bool{}
	cases := [][5]int{
		{2024, 5, 20, 10, 0}, {2024, 5, 20, 11, 0}, {2023, 1, 15, 6, 30},
		{1990, 12, 3, 23, 0}, {2000, 2, 10, 12, 0}, {1985, 7, 8, 3, 0},
	}
	distinct := 0
	for _, c := range cases {
		r, err := AnalyzeBazi(c[0], c[1], c[2], c[3], c[4])
		if err != nil {
			t.Fatalf("分析 %v 失败: %v", c, err)
		}
		if r.DayanHexagram == "" {
			t.Fatalf("%v 大衍卦为空", c)
		}
		if !seen[r.DayanHexagram] {
			distinct++
		}
		seen[r.DayanHexagram] = true
	}
	if distinct < 2 {
		t.Errorf("6 组生辰仅出现 %d 种卦象，播种疑似退化", distinct)
	}
}
