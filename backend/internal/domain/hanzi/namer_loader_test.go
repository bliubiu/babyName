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
