package hanzi

// loader_wuxing_test.go — 五行判定口径统一守护（docs/43 P2-3）
//
// 背景：LoadFromJSON 此前内联「覆盖表 > 部首 > JSON 原值」的优先级，
// 与单一真源 ResolveWuxing 的「覆盖表 > 已有值（人工校正） > 部首」冲突：
// namer.json 中人工校正过的五行会被部首映射覆盖而丢失。
// 统一后 loader 内联改为调用 ResolveWuxing，本文件钉住口径：
//   - 覆盖表字 → 覆盖表值
//   - 非覆盖表字 + JSON 带人工校正五行 → 保留 JSON 值（此前被部首覆盖）
//   - 非覆盖表字 + JSON 无五行 → 部首映射兜底

import (
	"os"
	"path/filepath"
	"testing"
)

// writeLoadFixture 写入最小 namer.json 夹具并返回目录
func writeLoadFixture(t *testing.T, jsonContent string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "hanzi.json"), []byte(jsonContent), 0o644); err != nil {
		t.Fatalf("写夹具失败: %v", err)
	}
	return dir
}

// TestLoadFromJSON_WuxingPriority 五行判定三级优先级与 ResolveWuxing 一致
func TestLoadFromJSON_WuxingPriority(t *testing.T) {
	// 构造一个不在覆盖表中的字（生僻组合避免误中 CharacterWuxingOverride）
	const probe = "龒龒"
	if _, hit := CharacterWuxingOverride[probe]; hit {
		t.Skipf("探测字 %s 在覆盖表中，用例失效", probe)
	}
	dir := writeLoadFixture(t, `[{
		"char": "`+probe+`", "radical": "氵", "wuxing": "火",
		"strokes": 12, "level": 1, "gender": "neutral"
	}]`)

	if err := LoadFromJSON(dir); err != nil {
		t.Fatalf("加载失败: %v", err)
	}

	h, ok := GetHanzi(probe)
	if !ok {
		t.Fatal("探测字未入库")
	}

	// ① JSON 人工校正值必须保留（此前被部首「氵→水」覆盖丢失）
	if h.Wuxing != "火" {
		t.Errorf("人工校正五行丢失: got %q，期望保留 JSON 值 %q（部首映射不得覆盖）", h.Wuxing, "火")
	}

	// ② 与 ResolveWuxing 单一真源一致
	if want := ResolveWuxing(probe, "火", "氵"); h.Wuxing != want {
		t.Errorf("loader 口径 (%q) ≠ ResolveWuxing (%q)", h.Wuxing, want)
	}
}

// TestLoadFromJSON_RadicalFallback 无 JSON 五行时部首兜底仍生效
func TestLoadFromJSON_RadicalFallback(t *testing.T) {
	const probe = "龑龑"
	if _, hit := CharacterWuxingOverride[probe]; hit {
		t.Skipf("探测字 %s 在覆盖表中，用例失效", probe)
	}
	want := GetWuxingByRadical("氵")
	if want == "" {
		t.Skip("部首表无「氵」映射，用例失效")
	}
	dir := writeLoadFixture(t, `[{
		"char": "`+probe+`", "radical": "氵", "wuxing": "",
		"strokes": 12, "level": 1, "gender": "neutral"
	}]`)

	if err := LoadFromJSON(dir); err != nil {
		t.Fatalf("加载失败: %v", err)
	}
	h, ok := GetHanzi(probe)
	if !ok {
		t.Fatal("探测字未入库")
	}
	if h.Wuxing != want {
		t.Errorf("部首兜底失效: got %q，期望 %q", h.Wuxing, want)
	}
}
