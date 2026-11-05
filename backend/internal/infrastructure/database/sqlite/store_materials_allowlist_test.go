package sqlite

import (
	"os"
	"path/filepath"
	"testing"

	"name/internal/domain/fate"
	"name/internal/materials/schema"
)

// store_materials_allowlist_test.go — 显式白名单物料（docs/33 门禁层 M1）：
// 校验 → 落库(entry,kind) → 运行时从库装载 → 内存白名单生效（豁免黑名单 + 祈福）。

// writeAllowlist 在 dir 下写入 allowlist.json
func writeAllowlist(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "allowlist.json"), []byte(body), 0o644); err != nil {
		t.Fatalf("写入白名单表失败: %v", err)
	}
}

// queryAllowlist 读出 allowlist(entry,kind) 映射
func queryAllowlist(t *testing.T, s *Store) map[string]string {
	t.Helper()
	rows, err := s.db.Query("SELECT entry, kind FROM allowlist")
	if err != nil {
		t.Fatalf("查询白名单表失败: %v", err)
	}
	defer func() { _ = rows.Close() }()

	out := map[string]string{}
	for rows.Next() {
		var entry, kind string
		if err := rows.Scan(&entry, &kind); err != nil {
			t.Fatalf("扫描白名单行失败: %v", err)
		}
		out[entry] = kind
	}
	return out
}

func TestImport_白名单物料落库与kind(t *testing.T) {
	dir := writeMinimalMaterials(t)
	writeAllowlist(t, dir, `{"version":"2.0","chars":["某"],"combos":["康泰"],"blessings":["去病"]}`)
	s := newTestStore(t)

	report := importMinimal(t, s, dir, ImportOptions{})
	r := resultOf(report, schema.AllowlistMaterial)
	if r.Err != "" {
		t.Fatalf("白名单物料应导入成功: %s", r.Err)
	}
	if r.Version != "2.0" {
		t.Errorf("版本号期望 2.0，实得 %q", r.Version)
	}
	if r.Skipped {
		t.Error("首次导入不应跳过")
	}

	got := queryAllowlist(t, s)
	want := map[string]string{"某": fate.AllowKindChar, "康泰": fate.AllowKindCombo, "去病": fate.AllowKindBlessing}
	if len(got) != len(want) {
		t.Fatalf("白名单条目数期望 %d，实得 %d：%v", len(want), len(got), got)
	}
	for entry, kind := range want {
		if got[entry] != kind {
			t.Errorf("条目 %s 类型期望 %s，实得 %s", entry, kind, got[entry])
		}
	}
}

func TestImport_白名单解析失败报错(t *testing.T) {
	dir := writeMinimalMaterials(t)
	writeAllowlist(t, dir, "{ 不是 JSON")
	s := newTestStore(t)

	report := importMinimal(t, s, dir, ImportOptions{})
	if r := resultOf(report, schema.AllowlistMaterial); r.Err == "" {
		t.Error("损坏的 JSON 应导致白名单导入失败")
	}
}

func TestImport_白名单重复条目去重(t *testing.T) {
	dir := writeMinimalMaterials(t)
	writeAllowlist(t, dir, `{"chars":["甲","甲"],"combos":[],"blessings":[]}`)
	s := newTestStore(t)

	importMinimal(t, s, dir, ImportOptions{})
	if got := queryAllowlist(t, s); len(got) != 1 {
		t.Errorf("重复条目应去重为 1 行，实得 %d：%v", len(got), got)
	}
}

func TestImport_白名单非法条目被拒且不写库(t *testing.T) {
	dir := writeMinimalMaterials(t)
	// combos 中出现三字组合，违反「必须两个汉字」规则
	writeAllowlist(t, dir, `{"chars":["甲"],"combos":["三字组合"],"blessings":[]}`)
	s := newTestStore(t)

	report := importMinimal(t, s, dir, ImportOptions{})
	if r := resultOf(report, schema.AllowlistMaterial); r.Err == "" {
		t.Fatal("非法组合应导致白名单导入失败")
	}
	if got := queryAllowlist(t, s); len(got) != 0 {
		t.Errorf("校验失败时不得写入任何白名单行，实得 %v", got)
	}
}

func Test运行时从库装载_白名单(t *testing.T) {
	dir := writeMinimalMaterials(t)
	writeAllowlist(t, dir, `{"version":"1.0","chars":["伙"],"combos":["康泰"],"blessings":["去病"]}`)
	s := newTestStoreIn(t, dir)
	if _, err := s.Import(ImportOptions{DataDir: dir}); err != nil {
		t.Fatalf("导入失败: %v", err)
	}
	s.LoadMaterialsFromDB()
	// 白名单为全局内存态，测试结束必须清空，避免污染同包后续门禁用例
	t.Cleanup(func() { fate.SetAllowlist(nil) })

	if !fate.IsAllowlistedChar("伙") {
		t.Error("库路径应装载字级白名单")
	}
	if fate.IsNonNamingChar("伙") {
		t.Error("白名单优先于黑名单：伙 应被豁免门禁")
	}
	if !fate.IsAllowlistedCombo("康", "泰") || !fate.IsAllowlistedCombo("泰", "康") {
		t.Error("库路径应装载组合级白名单（正反序均可命中）")
	}
	if !fate.IsBlessingCombo("去", "病") {
		t.Error("库路径应装载祈福组合")
	}
	if chars, combos, blessings := fate.AllowlistCounts(); chars != 1 || combos != 1 || blessings != 1 {
		t.Errorf("白名单计数期望 (1,1,1)，实得 (%d,%d,%d)", chars, combos, blessings)
	}
}
