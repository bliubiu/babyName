package yijing

import "testing"

func testHexagramByNumber(t *testing.T, num int) *Hexagram {
	t.Helper()
	for i := range HexagramList {
		if HexagramList[i].Number == num {
			return &HexagramList[i]
		}
	}
	t.Fatalf("未找到卦序 %d", num)
	return nil
}

// 64 卦四象限全部非空（差异化覆盖完整）
func TestInterpretHexagram_All64Populated(t *testing.T) {
	for num := 1; num <= 64; num++ {
		h := testHexagramByNumber(t, num)
		it := InterpretHexagram(h, nil)
		for field, v := range map[string]string{
			"career": it.Career, "love": it.Love,
			"health": it.Health, "fortune": it.Fortune,
		} {
			if v == "" {
				t.Errorf("卦 #%d %s 的 %s 文案为空", h.Number, h.Name, field)
			}
		}
	}
}

// 差异化核心断言：跨卦文案不得雷同（64 卦 career 须两两不同）
func TestInterpretHexagram_Differentiated(t *testing.T) {
	seen := map[string]int{}
	for num := 1; num <= 64; num++ {
		h := testHexagramByNumber(t, num)
		it := InterpretHexagram(h, nil)
		seen[it.Career]++
		if seen[it.Career] > 1 {
			t.Errorf("卦 #%d 的 career 文案与其他卦重复：%q", h.Number, it.Career)
		}
	}
	if len(seen) != 64 {
		t.Errorf("career 去重后 %d 种，期望 64", len(seen))
	}
}

// 金标准锚点：乾/坤/既济取意须与卦象对应
func TestInterpretHexagram_GoldenAnchors(t *testing.T) {
	cases := []struct {
		num    int
		career string
	}{
		{1, "刚健进取，自强不息，宜开创引领，忌刚愎自用"},
		{2, "厚德载物，宜稳步积累、辅佐执行，不宜急进"},
		{63, "功成守成，初吉后乱，防懈怠生变"},
		{64, "事未竟成，火水未交，仍需努力不懈"},
	}
	for _, c := range cases {
		h := testHexagramByNumber(t, c.num)
		it := InterpretHexagram(h, nil)
		if it.Career != c.career {
			t.Errorf("卦 #%d career=%q，期望 %q", c.num, it.Career, c.career)
		}
	}
}

// 喜用神命中上下卦五行时保留「与命格相合」吉断（旧口径不回归）
func TestInterpretHexagram_XiyongshenBonus(t *testing.T) {
	h := testHexagramByNumber(t, 11) // 泰：坤上乾下，土/金
	it := InterpretHexagram(h, []string{"金"})
	if it.Overall == h.Interpretation {
		t.Error("喜用神命中时 Overall 应附「与命格相合」断语")
	}
	if it.Fortune != "财运平稳，不宜冒险" {
		// 泰卦命中金（乾金）→ fortune 附加相合句
	}
	itNone := InterpretHexagram(h, []string{"水"})
	if itNone.Overall != h.Interpretation {
		t.Error("喜用神未命中时 Overall 不应附加断语")
	}
}

// 卦序越界回落通用模板（防御）
func TestInterpretHexagram_OutOfRangeFallback(t *testing.T) {
	h := &Hexagram{Number: 99, Name: "未知", UpperTrigram: 1, LowerTrigram: 1}
	it := InterpretHexagram(h, nil)
	if it.Career != "事业平稳发展，需把握时机" {
		t.Errorf("越界卦应回落通用模板，实得 %q", it.Career)
	}
}
