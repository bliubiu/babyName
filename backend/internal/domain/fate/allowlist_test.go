package fate

import (
	"os"
	"path/filepath"
	"testing"

	"name/internal/materials/schema"
)

// allowlist_test.go 守护显式白名单（docs/33 门禁层 M1）：
// 白名单优先于黑名单（字级/组合级/历史人物）+ 祈福组合动态并入内置默认。
//
// 全局白名单由本文件独占：每个用例用 resetAllowlist 在结束时清空，
// 避免污染同包其他测试（门禁字表/禁忌组合表由 TestMain 从 data 加载，不动）。

// resetAllowlist 测试结束清空白名单（内置 blessingCombos 不受影响）
func resetAllowlist(t *testing.T) {
	t.Helper()
	t.Cleanup(func() { SetAllowlist(nil) })
}

func TestAllowlist_分类装载与计数(t *testing.T) {
	resetAllowlist(t)
	SetAllowlist([]AllowEntry{
		{Entry: "伙", Kind: schema.AllowKindChar},
		{Entry: "父母", Kind: schema.AllowKindCombo},
		{Entry: "康泰", Kind: schema.AllowKindBlessing},
	})

	chars, combos, blessings := AllowlistCounts()
	if chars != 1 || combos != 1 || blessings != 1 {
		t.Fatalf("计数不符：chars=%d combos=%d blessings=%d", chars, combos, blessings)
	}
	if !IsAllowlistedChar("伙") {
		t.Error("伙 应被字级白名单豁免")
	}
	if !IsAllowlistedCombo("父", "母") || !IsAllowlistedCombo("母", "父") {
		t.Error("父母 组合级白名单应正反序均命中")
	}
	if IsAllowlistedChar("父母") || IsAllowlistedCombo("伙", "") {
		t.Error("跨类/空参不应误命中")
	}
}

func TestAllowlist_空条目与未知kind被忽略(t *testing.T) {
	resetAllowlist(t)
	SetAllowlist([]AllowEntry{
		{Entry: "", Kind: schema.AllowKindChar},
		{Entry: "  ", Kind: schema.AllowKindCombo},
		{Entry: "某", Kind: "未知类型"},
	})
	chars, combos, blessings := AllowlistCounts()
	if chars != 0 || combos != 0 || blessings != 0 {
		t.Fatalf("空条目与未知 kind 应被忽略，实得 chars=%d combos=%d blessings=%d", chars, combos, blessings)
	}
}

// TestAllowlist_字级豁免优先于门禁字 白名单字必须压过 naming_gate
func TestAllowlist_字级豁免优先于门禁字(t *testing.T) {
	resetAllowlist(t)
	if !IsNonNamingChar("伙") {
		t.Skip("前提不成立：伙 不在已加载的门禁字表中")
	}
	SetAllowlist([]AllowEntry{{Entry: "伙", Kind: schema.AllowKindChar}})
	if IsNonNamingChar("伙") {
		t.Error("白名单优先于黑名单：伙 被豁免后不应再判为门禁字")
	}
	// 明确反向控制：清空白名单后恢复门禁
	SetAllowlist(nil)
	if !IsNonNamingChar("伙") {
		t.Error("清空白名单后 伙 应重新被判为门禁字")
	}
}

// TestAllowlist_组合级豁免优先于禁忌组合 白名单组合必须压过 forbidden_combos
func TestAllowlist_组合级豁免优先于禁忌组合(t *testing.T) {
	resetAllowlist(t)
	if !IsBadCombo("父", "母") {
		t.Skip("前提不成立：父母 不在硬编码禁忌组合中")
	}
	SetAllowlist([]AllowEntry{{Entry: "父母", Kind: schema.AllowKindCombo}})
	if IsBadCombo("父", "母") || IsBadCombo("母", "父") {
		t.Error("白名单优先于黑名单：父母 被豁免后不应再判为禁忌组合")
	}
}

// TestAllowlist_组合级豁免优先于历史人物 白名单组合必须压过历史人物专名
func TestAllowlist_组合级豁免优先于历史人物(t *testing.T) {
	resetAllowlist(t)
	if !IsHistoricalFigureCombo("仲", "尼") {
		t.Skip("前提不成立：仲尼 不在历史人物专名表中")
	}
	SetAllowlist([]AllowEntry{{Entry: "仲尼", Kind: schema.AllowKindCombo}})
	if IsHistoricalFigureCombo("仲", "尼") {
		t.Error("白名单优先于黑名单：仲尼 被豁免后不应再判为历史人物组合")
	}
}

// TestAllowlist_祈福组合动态并入内置 内置 blessing 与物料 blessing 取并集
func TestAllowlist_祈福组合动态并入内置(t *testing.T) {
	resetAllowlist(t)
	if !IsBlessingCombo("去", "病") {
		t.Error("内置祈福组合 去病 应始终命中（硬编码兜底）")
	}
	if IsBlessingCombo("康", "泰") {
		t.Error("未装载动态前 康泰 不应命中")
	}
	SetAllowlist([]AllowEntry{{Entry: "康泰", Kind: schema.AllowKindBlessing}})
	if !IsBlessingCombo("康", "泰") || !IsBlessingCombo("泰", "康") {
		t.Error("动态祈福组合应正反序命中")
	}
	if !IsBlessingCombo("去", "病") {
		t.Error("动态装载不应冲掉内置祈福组合")
	}
}

func TestLoadAllowlistFromJSON(t *testing.T) {
	resetAllowlist(t)
	dir := t.TempDir()
	body := `{"version":"1.0","chars":["伙"],"combos":["父母"],"blessings":["康泰"]}`
	if err := os.WriteFile(filepath.Join(dir, "allowlist.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := LoadAllowlistFromJSON(dir); err != nil {
		t.Fatalf("加载白名单失败: %v", err)
	}
	if !IsAllowlistedChar("伙") || !IsAllowlistedCombo("父", "母") || !IsBlessingCombo("康", "泰") {
		c, cb, b := AllowlistCounts()
		t.Fatalf("JSON 装载后应生效：chars=%d combos=%d blessings=%d", c, cb, b)
	}
}

// TestLoadAllowlistFromJSON_文件缺失降级内置JSON 缺失只清空动态集合并警告，不报错
func TestLoadAllowlistFromJSON_文件缺失降级内置JSON(t *testing.T) {
	resetAllowlist(t)
	SetAllowlist([]AllowEntry{{Entry: "伙", Kind: schema.AllowKindChar}})
	if err := LoadAllowlistFromJSON(t.TempDir()); err != nil {
		t.Fatalf("文件缺失不应报错（降级内置默认）: %v", err)
	}
	if c, cb, b := AllowlistCounts(); c != 0 || cb != 0 || b != 0 {
		t.Errorf("缺失文件应清空动态集合，实得 chars=%d combos=%d blessings=%d", c, cb, b)
	}
	if !IsBlessingCombo("去", "病") {
		t.Error("缺失文件后内置祈福组合仍应可用")
	}
}

func TestLoadAllowlistFromJSON_解析失败报错(t *testing.T) {
	resetAllowlist(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "allowlist.json"), []byte("{ 不是 JSON"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := LoadAllowlistFromJSON(dir); err == nil {
		t.Error("损坏的 JSON 应返回错误以暴露数据损坏")
	}
}

// TestAllowlist_空参不命中 空参守卫：任何空字/空组合都不得误命中
func TestAllowlist_空参不命中(t *testing.T) {
	resetAllowlist(t)
	if IsAllowlistedChar("") {
		t.Error("空字不应命中字级白名单")
	}
	if IsAllowlistedCombo("", "母") || IsAllowlistedCombo("父", "") {
		t.Error("含空参的组合不应命中组合级白名单")
	}
	if IsBlessingCombo("", "") {
		t.Error("空参不应判为祈福组合")
	}
	if IsHistoricalFigureCombo("", "") {
		t.Error("空参不应判为历史人物组合")
	}
	// 反序仍应命中历史人物（覆盖反向查表分支）
	if !IsHistoricalFigureCombo("仲", "尼") {
		t.Skip("前提不成立：仲尼 不在历史人物专名表中")
	}
	if !IsHistoricalFigureCombo("尼", "仲") {
		t.Error("历史人物组合应正反序均命中")
	}
}

// TestLoadAllowlistFromJSON_路径非法报错 读取失败（非「不存在」）必须返回错误
func TestLoadAllowlistFromJSON_路径非法报错(t *testing.T) {
	resetAllowlist(t)
	dir := t.TempDir()
	// 以目录占据 allowlist.json 路径：os.ReadFile 返回非 IsNotExist 错误
	if err := os.Mkdir(filepath.Join(dir, "allowlist.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := LoadAllowlistFromJSON(dir); err == nil {
		t.Error("读取失败（路径为目录）应返回错误")
	}
}

// TestAllowlist_整体文本豁免查询 供跨长度条目（合规红线等）豁免使用
func TestAllowlist_整体文本豁免查询(t *testing.T) {
	resetAllowlist(t)
	if IsAllowlistedText("") {
		t.Error("空文本不应命中")
	}
	if IsAllowlistedText("父母") {
		t.Error("未装载前不应命中")
	}
	SetAllowlist([]AllowEntry{
		{Entry: "父母", Kind: schema.AllowKindCombo},
		{Entry: "伙", Kind: schema.AllowKindChar},
	})
	if !IsAllowlistedText("父母") || !IsAllowlistedText("伙") {
		t.Error("装载后整体文本应命中（任意 kind）")
	}
	if IsAllowlistedText("母父") {
		t.Error("整体文本为原串精确匹配，反序不应命中")
	}
}
