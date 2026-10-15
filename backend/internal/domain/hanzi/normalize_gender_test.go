package hanzi

import "testing"

// 守护 docs/31 ADR-014 / docs/32 P-1.4：性别枚举归一的表驱动用例。
//
// 关键行为变更：gender 为空的字（实测 8105 字中 6266 字，占 77%）归一为 neutral。
// 关闭 GenderUnspecifiedAsNeutral 后必须完全回到旧行为（返回空串）。

func TestNormalizeGender_枚举归一(t *testing.T) {
	cases := []struct{ in, want, note string }{
		{"male", "male", "英文枚举原样"},
		{"female", "female", "英文枚举原样"},
		{"neutral", "neutral", "英文枚举原样"},
		{"男", "male", "中文枚举归一"},
		{"女", "female", "中文枚举归一"},
		{"", "neutral", "空值归一为中性（行为变更）"},
	}

	for _, c := range cases {
		got := normalizeGender(c.in)
		if got != c.want {
			t.Errorf("normalizeGender(%q) 期望 %q（%s），实得 %q", c.in, c.want, c.note, got)
		}
	}
}

func TestNormalizeGender_开关关闭时回到旧行为(t *testing.T) {
	old := GenderUnspecifiedAsNeutral
	GenderUnspecifiedAsNeutral = false
	t.Cleanup(func() { GenderUnspecifiedAsNeutral = old })

	if got := normalizeGender(""); got != "" {
		t.Errorf("开关关闭时空值应原样返回空串，实得 %q", got)
	}
	// 其他枚举不受开关影响
	if got := normalizeGender("男"); got != "male" {
		t.Errorf("开关关闭时中文枚举仍应归一，实得 %q", got)
	}
}

func TestToHanzi_空性别归中性(t *testing.T) {
	old := GenderUnspecifiedAsNeutral
	GenderUnspecifiedAsNeutral = true
	t.Cleanup(func() { GenderUnspecifiedAsNeutral = old })

	if got := ToHanzi(NamerChar{Char: "和"}).Gender; got != "neutral" {
		t.Errorf("空性别经 mapper 后期望 neutral，实得 %q", got)
	}
}
