package yijing

import "testing"

// TestHexagram_TrigramsMatchMeihuaTable 64 卦 Upper/Lower 与先天八卦姓名卦对照表一致
func TestHexagram_TrigramsMatchMeihuaTable(t *testing.T) {
	if len(meihuaGuaNames) != 64 {
		t.Fatalf("meihuaGuaNames len=%d", len(meihuaGuaNames))
	}
	byName := map[string]*Hexagram{}
	for i := range HexagramList {
		byName[HexagramList[i].Name] = &HexagramList[i]
	}
	for i, name := range meihuaGuaNames {
		wantU := i/8 + 1
		wantL := i%8 + 1
		h, ok := byName[name]
		if !ok {
			t.Errorf("对照表卦名 %s 不在 HexagramList", name)
			continue
		}
		if h.UpperTrigram != wantU || h.LowerTrigram != wantL {
			t.Errorf("卦 %s Upper/Lower=%d/%d, 期望 %d/%d（先天八卦）",
				name, h.UpperTrigram, h.LowerTrigram, wantU, wantL)
		}
		// Symbol 应为 上卦符号+下卦符号
		wantSym := TrigramMap[wantU] + TrigramMap[wantL]
		if h.Symbol != wantSym {
			t.Errorf("卦 %s Symbol=%q, 期望 %q", name, h.Symbol, wantSym)
		}
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
