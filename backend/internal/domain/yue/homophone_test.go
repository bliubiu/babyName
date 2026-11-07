package yue

import "testing"

// TestCheckNameYueHomophones_Fail 明确点名「好字→不雅谐音」配对（fail 级）
func TestCheckNameYueHomophones_Fail(t *testing.T) {
	cases := []string{"朱", "柯", "苟", "史", "赖"}
	jp := map[string]string{"朱": "zyu1", "柯": "o1", "苟": "gau2", "史": "si2", "赖": "laai6"}
	for _, ch := range cases {
		hits := CheckNameYueHomophones(jp[ch], ch)
		if len(hits) == 0 {
			t.Errorf("%s(%s) 应命中粤语不吉谐音", ch, jp[ch])
			continue
		}
		if hits[0].Severity != "fail" {
			t.Errorf("%s 应为 fail 级, got %q (%s)", ch, hits[0].Severity, hits[0].Description)
		}
	}
}

// TestCheckNameYueHomophones_Warn 近音歧义（warn 级）：吴→唔
func TestCheckNameYueHomophones_Warn(t *testing.T) {
	hits := CheckNameYueHomophones("ng4", "吴")
	if len(hits) == 0 {
		t.Fatal("吴(ng4) 应命中懒音近音歧义（唔）")
	}
	if hits[0].Severity != "warn" {
		t.Errorf("吴 应为 warn 级, got %q", hits[0].Severity)
	}
}

// TestCheckNameYueHomophones_Benign 同音好字不被误伤（本字豁免 / 不波及同音好字）
//
// 珠(zyu1) 与朱(zyu1) 同音，但珠是珍珠好字：朱→猪 的配对只点名朱，
// 珠不应被判不吉（这是整个谐音库「不误伤同音好字」设计原则的回归防线）。
func TestCheckNameYueHomophones_Benign(t *testing.T) {
	benign := []struct{ jp, ch string }{
		{"zyu1", "珠"},  // 珍珠（与朱同音 zyu1，但不应命中）
		{"jyu5", "宇"},  // 气宇
		{"zaak6", "泽"}, // 恩泽
		{"lei5", "李"},  // 姓氏常用字
		{"jan1", "欣"},  // 欣喜
	}
	for _, b := range benign {
		hits := CheckNameYueHomophones(b.jp, b.ch)
		if len(hits) != 0 {
			t.Errorf("%s(%s) 不应命中粤语谐音, got %+v", b.ch, b.jp, hits)
		}
	}
}

// TestCheckNameYueHomophones_BadSelfChar 名字真用了不吉字则命中（本字==不吉字）
func TestCheckNameYueHomophones_BadSelfChar(t *testing.T) {
	// 屎 si2：通用不吉字表带本字豁免，本字即「屎」时命中
	hits := CheckNameYueHomophones("si2", "屎")
	if len(hits) == 0 {
		t.Fatal("屎(si2) 作为本字应命中通用不吉字")
	}
	if hits[0].Severity != "fail" {
		t.Errorf("屎 应为 fail 级, got %q", hits[0].Severity)
	}
	// 思 si1 与屎 si2 声调不同（无声调后都是 si），本字非屎 → 不命中
	if hits := CheckNameYueHomophones("si1", "思"); len(hits) != 0 {
		t.Errorf("思 不应命中屎, got %+v", hits)
	}
}

// TestCheckYueBadCombo 粤语连读不良词（含姓氏拼音，去声调滑窗匹配）
func TestCheckYueBadCombo(t *testing.T) {
	// 吴得 → 唔得（不行）
	if hit, desc := CheckYueBadCombo("ng4", "dak1"); !hit {
		t.Errorf("吴得→唔得 应命中连读不良词 (desc=%q)", desc)
	}
	// 王板 → 黄板
	if hit, _ := CheckYueBadCombo("wong4", "baan2"); !hit {
		t.Error("王板 应命中连读不良词")
	}
	// 良性组合：李(lei5)+宇(jyu5) → leijyu，无命中
	if hit, _ := CheckYueBadCombo("lei5", "jyu5"); hit {
		t.Error("李宇 不应命中连读不良词")
	}
	// 空输入不报错
	if hit, _ := CheckYueBadCombo(""); hit {
		t.Error("空输入不应命中")
	}
}

// TestLazyVariants 懒音变体生成（含自身去重；ng 前缀剥离后为空不产生垃圾）
func TestLazyVariants(t *testing.T) {
	// n→l：nei 应含 lei
	got := lazyVariants("nei")
	if !contains(got, "nei") || !contains(got, "lei") {
		t.Errorf("lazyVariants(nei)=%v, 期望含 nei 与 lei", got)
	}
	// ng 前缀剥离后为空串，不应把 "4"/空串 当作变体
	ngVar := lazyVariants("ng")
	for _, v := range ngVar {
		if v == "" || v == "4" {
			t.Errorf("lazyVariants(ng) 产生垃圾变体: %v", ngVar)
		}
	}
	// 尾 t/k 互换：zaak→zaat
	tk := lazyVariants("zaak")
	if !contains(tk, "zaat") {
		t.Errorf("lazyVariants(zaak)=%v, 期望含 zaat", tk)
	}
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
