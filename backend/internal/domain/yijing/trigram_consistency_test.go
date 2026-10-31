package yijing

import "testing"

// TestHexagram_TrigramsBijection 64 组先天上下卦组合与 HexagramList 双射：
// 每个 (upper, lower) 恰好命中一卦，且 64 卦互不重复（Symbol 同口径校验）
func TestHexagram_TrigramsBijection(t *testing.T) {
	seen := map[string]bool{}
	for u := 1; u <= 8; u++ {
		for l := 1; l <= 8; l++ {
			h := FindHexagramByTrigrams(u, l)
			if h == nil {
				t.Fatalf("组合 (%d,%d) 无卦命中", u, l)
			}
			if h.UpperTrigram != u || h.LowerTrigram != l {
				t.Fatalf("组合 (%d,%d) 命中卦 %s 的上下卦为 %d/%d，表数据错位",
					u, l, h.Name, h.UpperTrigram, h.LowerTrigram)
			}
			if seen[h.Name] {
				t.Fatalf("卦名 %s 重复命中，组合不构成双射", h.Name)
			}
			seen[h.Name] = true
			wantSym := TrigramMap[u] + TrigramMap[l]
			if h.Symbol != wantSym {
				t.Errorf("卦 %s Symbol=%q, 期望 %q", h.Name, h.Symbol, wantSym)
			}
		}
	}
	if len(seen) != 64 {
		t.Fatalf("覆盖卦名 %d 个，应为 64", len(seen))
	}
}

// TestGetHexagramByMeihuaName_UsesSameData 梅花起卦结果的 Upper/Lower 与笔画一致
func TestGetHexagramByMeihuaName_UsesSameData(t *testing.T) {
	for s := 1; s <= 8; s++ {
		for g := 1; g <= 8; g++ {
			h := GetHexagramByMeihuaName(s, g)
			if h == nil {
				t.Fatalf("Meihua(%d,%d)=nil", s, g)
			}
			wantU := TrigramFromStrokes(s)
			wantL := TrigramFromStrokes(g)
			if h.UpperTrigram != wantU || h.LowerTrigram != wantL {
				t.Errorf("Meihua(%d,%d)=%s 上下=%d/%d, 期望 %d/%d",
					s, g, h.Name, h.UpperTrigram, h.LowerTrigram, wantU, wantL)
			}
		}
	}
}
