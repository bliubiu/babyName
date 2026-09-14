package hanzi

import (
	"testing"
)

// TestLoadNamerGroups 验证 namer.json 顶层 charGroups（精选偏旁分组）被正确加载，
// 取代独立的 standard_chars.json 独立文件。
func TestLoadNamerGroups(t *testing.T) {
	dataDir := "../../../data/"
	if err := LoadNamerFromJSON(dataDir); err != nil {
		t.Fatalf("LoadNamerFromJSON 失败: %v", err)
	}

	groups := GetNamerGroups()
	if len(groups) == 0 {
		t.Fatal("namer.json 应包含精选偏旁分组(charGroups)，实际为空")
	}

	// 每个分组应包含 radical 与非空 Chars
	for _, g := range groups {
		if g.Radical == "" {
			t.Errorf("分组缺少 radical: %+v", g)
		}
		if len(g.Chars) == 0 {
			t.Errorf("分组 %q 的 Chars 为空", g.Radical)
		}
	}
}

// TestGetCharRadicalFromNamer 验证从 namer 数据查询汉字的偏旁（取代盘上 standard_chars.json）。
func TestGetCharRadicalFromNamer(t *testing.T) {
	dataDir := "../../../data/"
	_ = LoadNamerFromJSON(dataDir)

	// 「林」字在标准字表内，radical 应为木
	if r := GetCharRadicalFromNamer("林"); r == "" {
		t.Errorf("「林」的偏旁应非空，实际为空")
	}
	// 表外/未收录字应返回空
	if r := GetCharRadicalFromNamer("𠀀"); r != "" {
		t.Errorf("未收录字应返回空偏旁，实际为 %q", r)
	}
}

// TestNormalizeGenderChineseEnums 验证 namer.json 中性别枚举被归一化：
// 中文（男/女）归一化为英文（male/female），fate filter 只识别英文枚举
// （filter.go 性别判定），历史数据混用导致标"男"/"女"的好字（如 风94/玄94）
// 在对应性别场景被误剔除、候选池被无谓砍掉约 155 个评分好字。
func TestNormalizeGenderChineseEnums(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"男", "male"},
		{"女", "female"},
		{"male", "male"},
		{"female", "female"},
		{"neutral", "neutral"},
		{"", ""},
	}
	for _, c := range cases {
		if got := normalizeGender(c.in); got != c.want {
			t.Errorf("normalizeGender(%q) = %q, 期望 %q", c.in, got, c.want)
		}
	}
}

// TestLoadNamerGenderNormalized 验证加载真实数据后：中文字符"男"/"女"枚举的
// 好字在内存中的 Gender 已变为 male/female（即与 filter 判定对齐）。
func TestLoadNamerGenderNormalized(t *testing.T) {
	dataDir := "../../../data/"
	if err := LoadNamerFromJSON(dataDir); err != nil {
		t.Fatalf("LoadNamerFromJSON 失败: %v", err)
	}
	mu.RLock()
	defer mu.RUnlock()
	// 「风」在 namer.json 中标注为"男"，归一化后应为 male
	if h, ok := HanziData["风"]; ok {
		if h.Gender != "male" {
			t.Errorf("「风」Gender = %q, 期望 male（namer.json 标注\"男\"应被归一化）", h.Gender)
		}
	} else {
		t.Error("「风」应在 HanziData 中")
	}
	// 「玉」在 namer.json 中标注为"女"，归一化后应为 female
	if h, ok := HanziData["玉"]; ok {
		if h.Gender != "female" {
			t.Errorf("「玉」Gender = %q, 期望 female（namer.json 标注\"女\"应被归一化）", h.Gender)
		}
	} else {
		t.Error("「玉」应在 HanziData 中")
	}
	// 存量英文枚举不受影响
	if h, ok := HanziData["中"]; ok && h.Gender == "男" {
		t.Errorf("「中」Gender 不应残留中文枚举")
	}
}
