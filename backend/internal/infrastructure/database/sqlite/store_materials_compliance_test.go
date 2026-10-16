package sqlite

import (
	"os"
	"path/filepath"
	"testing"

	"name/internal/domain/fate"
	"name/internal/materials/schema"
)

// store_materials_compliance_test.go — L5 合规红线物料（docs/33 门禁层 M2）：
// 校验 → 落库(entry,kind) → 运行时从库装载 → 内存红线生效。

// writeCompliance 在 dir 下写入 compliance_redline.json
func writeCompliance(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "compliance_redline.json"), []byte(body), 0o644); err != nil {
		t.Fatalf("写入合规红线表失败: %v", err)
	}
}

// queryCompliance 读出 compliance_redline(entry,kind) 映射
func queryCompliance(t *testing.T, s *Store) map[string]string {
	t.Helper()
	rows, err := s.db.Query("SELECT entry, kind FROM compliance_redline")
	if err != nil {
		t.Fatalf("查询合规红线表失败: %v", err)
	}
	defer func() { _ = rows.Close() }()

	out := map[string]string{}
	for rows.Next() {
		var entry, kind string
		if err := rows.Scan(&entry, &kind); err != nil {
			t.Fatalf("扫描合规红线行失败: %v", err)
		}
		out[entry] = kind
	}
	return out
}

func TestImport_合规红线落库与kind(t *testing.T) {
	dir := writeMinimalMaterials(t)
	writeCompliance(t, dir, `{"version":"2.0","state_orgs":["省高院"],"places":["北京"],"persons":["秦桧"]}`)
	s := newTestStore(t)

	report := importMinimal(t, s, dir, ImportOptions{})
	r := resultOf(report, schema.ComplianceRedlineMaterial)
	if r.Err != "" {
		t.Fatalf("合规红线物料应导入成功: %s", r.Err)
	}
	if r.Version != "2.0" {
		t.Errorf("版本号期望 2.0，实得 %q", r.Version)
	}
	if r.Skipped {
		t.Error("首次导入不应跳过")
	}

	got := queryCompliance(t, s)
	want := map[string]string{
		"省高院": schema.RedlineKindStateOrg,
		"北京":  schema.RedlineKindPlace,
		"秦桧":  schema.RedlineKindPerson,
	}
	if len(got) != len(want) {
		t.Fatalf("合规红线条目数期望 %d，实得 %d：%v", len(want), len(got), got)
	}
	for entry, kind := range want {
		if got[entry] != kind {
			t.Errorf("条目 %s 类型期望 %s，实得 %s", entry, kind, got[entry])
		}
	}
}

func TestImport_合规红线解析失败报错(t *testing.T) {
	dir := writeMinimalMaterials(t)
	writeCompliance(t, dir, "{ 不是 JSON")
	s := newTestStore(t)

	report := importMinimal(t, s, dir, ImportOptions{})
	if r := resultOf(report, schema.ComplianceRedlineMaterial); r.Err == "" {
		t.Error("损坏的 JSON 应导致合规红线导入失败")
	}
}

func TestImport_合规红线重复条目去重(t *testing.T) {
	dir := writeMinimalMaterials(t)
	writeCompliance(t, dir, `{"state_orgs":["省高院","省高院"],"places":[],"persons":[]}`)
	s := newTestStore(t)

	importMinimal(t, s, dir, ImportOptions{})
	if got := queryCompliance(t, s); len(got) != 1 {
		t.Errorf("重复条目应去重为 1 行，实得 %d：%v", len(got), got)
	}
}

func TestImport_合规红线非法条目被拒且不写库(t *testing.T) {
	dir := writeMinimalMaterials(t)
	// 单字条目违反「长度必须为 2~8 个汉字」规则
	writeCompliance(t, dir, `{"state_orgs":["京"],"places":[],"persons":[]}`)
	s := newTestStore(t)

	report := importMinimal(t, s, dir, ImportOptions{})
	if r := resultOf(report, schema.ComplianceRedlineMaterial); r.Err == "" {
		t.Fatal("非法条目（单字）应导致合规红线导入失败")
	}
	if got := queryCompliance(t, s); len(got) != 0 {
		t.Errorf("校验失败时不得写入任何合规红线行，实得 %v", got)
	}
}

func Test运行时从库装载_合规红线(t *testing.T) {
	dir := writeMinimalMaterials(t)
	writeCompliance(t, dir, `{"version":"1.0","state_orgs":["省高院"],"places":["北京"],"persons":["秦桧"]}`)
	s := newTestStoreIn(t, dir)
	if _, err := s.Import(ImportOptions{DataDir: dir}); err != nil {
		t.Fatalf("导入失败: %v", err)
	}
	s.LoadMaterialsFromDB()
	// 合规红线为全局内存态，测试结束必须清空，避免污染同包后续用例
	t.Cleanup(func() { fate.SetComplianceRedline(nil) })

	if !fate.IsComplianceRedline("省高院") {
		t.Error("库路径应装载机关单位红线")
	}
	if !fate.IsComplianceRedline("李北京") {
		t.Error("库路径应装载争议地名红线（包含匹配）")
	}
	if !fate.IsComplianceRedline("李秦桧") {
		t.Error("库路径应装载负面历史人物红线")
	}
	if stateOrg, place, person := fate.ComplianceRedlineCounts(); stateOrg != 1 || place != 1 || person != 1 {
		t.Errorf("合规红线计数期望 (1,1,1)，实得 (%d,%d,%d)", stateOrg, place, person)
	}
}
