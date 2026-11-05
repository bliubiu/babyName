package bazi

// xiyong_jichou_test.go — DeriveJiChou 扶抑法推导守护（docs/43 P1-3）

import "testing"

// TestDeriveJiChou_WeakNeedsSupport 身弱/身衰：忌官杀、仇财
func TestDeriveJiChou_WeakNeedsSupport(t *testing.T) {
	cases := []struct {
		riZhu    string
		wantJi   string // 克我者
		wantChou string // 我克者
	}{
		{"木", "金", "土"}, // 金克木；木克土
		{"火", "水", "金"}, // 水克火；火克金
		{"土", "木", "水"}, // 木克土；土克水
		{"金", "火", "木"}, // 火克金；金克木
		{"水", "土", "火"}, // 土克水；水克火
	}
	for _, c := range cases {
		for _, strength := range []string{"身弱", "身衰"} {
			ji, chou := DeriveJiChou(c.riZhu, strength)
			if ji != c.wantJi || chou != c.wantChou {
				t.Errorf("%s(%s): 忌=%s 仇=%s，期望 忌=%s 仇=%s",
					c.riZhu, strength, ji, chou, c.wantJi, c.wantChou)
			}
		}
	}
}

// TestDeriveJiChou_StrongAvoidsSupport 身旺：忌比劫、仇印（生我者）
func TestDeriveJiChou_StrongAvoidsSupport(t *testing.T) {
	cases := []struct {
		riZhu    string
		wantChou string // 生我者
	}{
		{"木", "水"}, {"火", "木"}, {"土", "火"}, {"金", "土"}, {"水", "金"},
	}
	for _, c := range cases {
		ji, chou := DeriveJiChou(c.riZhu, "身旺")
		if ji != c.riZhu {
			t.Errorf("%s(身旺): 忌=%s，期望比劫 %s", c.riZhu, ji, c.riZhu)
		}
		if chou != c.wantChou {
			t.Errorf("%s(身旺): 仇=%s，期望印 %s", c.riZhu, chou, c.wantChou)
		}
	}
}

// TestDeriveJiChou_Neutral 身中及未知值：中性无忌仇
func TestDeriveJiChou_Neutral(t *testing.T) {
	for _, strength := range []string{"身中", "", "未知"} {
		ji, chou := DeriveJiChou("木", strength)
		if ji != "" || chou != "" {
			t.Errorf("木(%q): 忌=%s 仇=%s，期望均为空", strength, ji, chou)
		}
	}
}
