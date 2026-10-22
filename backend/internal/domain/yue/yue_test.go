package yue

import "testing"

// TestPrimaryJyutping 常用起名字粤拼（与 jyutping.json 三源合并产物核对）
func TestPrimaryJyutping(t *testing.T) {
	cases := map[string]string{
		"柯": "o1",
		"朱": "zyu1",
		"珠": "zyu1",
		"吴": "ng4",
		"苟": "gau2",
		"史": "si2",
		"泽": "zaak6",
		"李": "lei5",
		"王": "wong4",
		"陈": "can4",
		"黎": "lai4",
		"英": "jing1",
		"俊": "zeon3",
		"宇": "jyu5",
		"欣": "jan1",
	}
	for ch, want := range cases {
		if got := PrimaryJyutping(ch); got != want {
			t.Errorf("PrimaryJyutping(%q)=%q, want %q", ch, got, want)
		}
	}
	// 未收录返回空串（调用方据此降级）
	if got := PrimaryJyutping("𠀀"); got != "" {
		t.Errorf("未收录字应返回空串, got %q", got)
	}
}

// TestJyutpingOf_MultiReading 多音字返回全部读音
func TestJyutpingOf_MultiReading(t *testing.T) {
	vs := JyutpingOf("朱")
	if len(vs) == 0 || vs[0] != "zyu1" {
		t.Fatalf("JyutpingOf(朱)=%v, 期望包含 zyu1", vs)
	}
	if !HasJyutping("李") {
		t.Error("HasJyutping(李) 应为 true")
	}
	if HasJyutping("") {
		t.Error("空串应无粤拼")
	}
}

// TestStripJyutpingTone 去声调
func TestStripJyutpingTone(t *testing.T) {
	cases := map[string]string{
		"nei5":  "nei",
		"jyu5":  "jyu",
		"aa1":   "aa",
		"zaak6": "zaak",
		"gwong2": "gwong",
		"ng4":   "ng",
		"":      "",
	}
	for in, want := range cases {
		if got := StripJyutpingTone(in); got != want {
			t.Errorf("StripJyutpingTone(%q)=%q, want %q", in, got, want)
		}
	}
}

// TestJyutpingTone 提取声调（1-6）
func TestJyutpingTone(t *testing.T) {
	cases := map[string]int{"lei5": 5, "zaak6": 6, "jyu5": 5, "aa1": 1, "ng4": 4, "": 0}
	for in, want := range cases {
		if got := JyutpingTone(in); got != want {
			t.Errorf("JyutpingTone(%q)=%d, want %d", in, got, want)
		}
	}
}

// TestJyutpingInitial 声母提取（含 gw/kw/ng，零声母为空）
func TestJyutpingInitial(t *testing.T) {
	cases := map[string]string{
		"gwong2": "gw",
		"kwong2": "kw",
		"ng4":    "ng",
		"nei5":   "n",
		"jyu5":   "j",
		"aa1":    "", // 零声母
		"zaak6":  "z",
	}
	for in, want := range cases {
		if got := JyutpingInitial(in); got != want {
			t.Errorf("JyutpingInitial(%q)=%q, want %q", in, got, want)
		}
	}
}

// TestJyutpingFinal 韵母提取
func TestJyutpingFinal(t *testing.T) {
	cases := map[string]string{
		"gwong2": "ong",
		"nei5":   "ei",
		"aa1":    "aa",
		"zaak6":  "aak",
	}
	for in, want := range cases {
		if got := JyutpingFinal(in); got != want {
			t.Errorf("JyutpingFinal(%q)=%q, want %q", in, got, want)
		}
	}
}

// TestIsEnteringTone 入声判定（p/t/k 塞尾）
func TestIsEnteringTone(t *testing.T) {
	entering := []string{"zaak6", "sap1", "hot1", "gok3"}
	for _, jp := range entering {
		if !IsEnteringTone(jp) {
			t.Errorf("IsEnteringTone(%q) 应为 true", jp)
		}
	}
	nonEntering := []string{"nei5", "jyu5", "gwong2", "aa1"}
	for _, jp := range nonEntering {
		if IsEnteringTone(jp) {
			t.Errorf("IsEnteringTone(%q) 应为 false", jp)
		}
	}
}

// TestToneClass 平仄分类：平={1,2}、仄={3,4,5,6}
func TestToneClass(t *testing.T) {
	for _, tone := range []int{1, 2} {
		if ToneClass(tone) != 0 {
			t.Errorf("ToneClass(%d) 应为平(0)", tone)
		}
	}
	for _, tone := range []int{3, 4, 5, 6} {
		if ToneClass(tone) != 1 {
			t.Errorf("ToneClass(%d) 应为仄(1)", tone)
		}
	}
}

// TestShengMuGroup 声母发音部位分组
func TestShengMuGroup(t *testing.T) {
	cases := map[string]int{"b": 1, "m": 1, "f": 2, "d": 3, "l": 3, "g": 4, "ng": 4, "gw": 5, "w": 5, "z": 6, "j": 7}
	for sm, want := range cases {
		if got := ShengMuGroup(sm); got != want {
			t.Errorf("ShengMuGroup(%q)=%d, want %d", sm, got, want)
		}
	}
	if ShengMuGroupName(3) != "齿龈" {
		t.Errorf("ShengMuGroupName(3)=%q, want 齿龈", ShengMuGroupName(3))
	}
	if ShengMuGroupName(99) != "未知" {
		t.Errorf("未知组应返回未知, got %q", ShengMuGroupName(99))
	}
}

// TestAllChars 已收录字表非空（保证嵌入数据可用）
func TestAllChars(t *testing.T) {
	chars := AllChars()
	if len(chars) < 1000 {
		t.Fatalf("粤拼收录字过少: %d", len(chars))
	}
}
