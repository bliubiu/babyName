package yijing

import "testing"

// 本文件固化 docs/29 A3/A4 的修复回归：
// 1. 大衍筮法值域 {6,7,8,9} 且变爻可达；
// 2. 卦由六爻阴阳直接排定（组合码 ≠ 通行本卦序，不得互查）；
// 3. GetHexagramSymbol 上卦符号在前，与 HexagramList.Symbol 同口径。

// TestCastHexagramByDayan_HexagramMatchesLines 本卦必须与所摇六爻的阴阳一致
func TestCastHexagramByDayan_HexagramMatchesLines(t *testing.T) {
	for i := 0; i < 50; i++ {
		r := CastHexagramByDayan()
		if r == nil || r.Hexagram == nil {
			t.Fatal("CastHexagramByDayan 返回 nil")
		}
		wantLower := trigramFromYaoLines(r.YaoLines[0:3])
		wantUpper := trigramFromYaoLines(r.YaoLines[3:6])
		if r.Hexagram.UpperTrigram != wantUpper || r.Hexagram.LowerTrigram != wantLower {
			t.Fatalf("第 %d 次：卦 %s 上下卦 %d/%d 与六爻阴阳排定的 %d/%d 不符",
				i, r.Hexagram.Name, r.Hexagram.UpperTrigram, r.Hexagram.LowerTrigram, wantUpper, wantLower)
		}
	}
}

// TestCalculateDayanNumber_ChangeYaoReachable 老阳/老阴必须摇得出来（变爻可达）
func TestCalculateDayanNumber_ChangeYaoReachable(t *testing.T) {
	sawChange := false
	for i := 0; i < 200; i++ {
		r := CastHexagramByDayan()
		if r.ChangeYao > 0 {
			sawChange = true
			if yao := r.YaoLines[r.ChangeYao-1]; yao != 6 && yao != 9 {
				t.Fatalf("变爻位置 %d 的爻值=%d，应为 6/9", r.ChangeYao, yao)
			}
		}
	}
	if !sawChange {
		t.Fatal("200 次起卦无一变爻：老阳(9)/老阴(6) 不可达，筮法仍是错的")
	}
}

// TestGetHexagramByStrokes_MatchesTrigrams 姓名卦按先天上下卦查表
func TestGetHexagramByStrokes_MatchesTrigrams(t *testing.T) {
	for strokes := 1; strokes <= 64; strokes++ {
		h := GetHexagramByStrokes(strokes)
		if h == nil {
			t.Fatalf("GetHexagramByStrokes(%d)=nil", strokes)
		}
		wantU := TrigramFromStrokes(strokes / 8)
		wantL := TrigramFromStrokes(strokes)
		if h.UpperTrigram != wantU || h.LowerTrigram != wantL {
			t.Fatalf("笔画 %d：卦 %s 上下卦 %d/%d，期望 %d/%d（先天八卦）",
				strokes, h.Name, h.UpperTrigram, h.LowerTrigram, wantU, wantL)
		}
	}
}

// TestTrigramLookup_KingWenAnchors 组合码当卦序号的反例锚点
// 上兑下兑：组合码 (2-1)*8+2 = 10 是第 10 卦「履」（上乾下兑）；
// 正确按上下卦查表应得第 58 卦「兑」。其余三处同构。
func TestTrigramLookup_KingWenAnchors(t *testing.T) {
	h := FindHexagramByTrigrams(2, 2)
	if h == nil || h.Name != "兑" || h.Number != 58 {
		t.Fatalf("上兑下兑应得第 58 卦「兑」，got %+v", h)
	}
	wrong := GetHexagramByNumber(10)
	if wrong == nil || wrong.Name != "履" {
		t.Fatalf("第 10 卦应为「履」，got %+v（锚点失效，请核对卦表）", wrong)
	}
	// 上坎下乾 = 第 5 卦「需」（组合码 41 是「损」）
	need := FindHexagramByTrigrams(6, 1)
	if need == nil || need.Name != "需" || need.Number != 5 {
		t.Fatalf("上坎下乾应得第 5 卦「需」，got %+v", need)
	}
}

// TestGetHexagramSymbol_Order 符号顺序：上卦在前，与 HexagramList.Symbol 同口径
func TestGetHexagramSymbol_Order(t *testing.T) {
	for _, h := range HexagramList {
		if got := GetHexagramSymbol(h.UpperTrigram, h.LowerTrigram); got != h.Symbol {
			t.Fatalf("卦 %s：GetHexagramSymbol=%q，表内 Symbol=%q，口径不一致", h.Name, got, h.Symbol)
		}
	}
}
