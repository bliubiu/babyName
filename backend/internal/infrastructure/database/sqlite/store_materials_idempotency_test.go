package sqlite

import (
	"os"
	"path/filepath"
	"testing"
)

// 本文件守护 docs/34 阶段 D1 的 CI 端到端断言，与 store_materials_import_test.go
// 的单点测试互补，专注三条链路级不变量：
//
//  ① 全量导入 + 二次导入：走**正常指纹短路路径**（非 lenient 强制重灌），
//     物料内容快照必须逐字节等价（幂等）；
//  ② 增量门控：仅改动单物料源文件时，只有该物料重灌，其余全部短路；
//  ③ 辅表为空的物料不得被误判为「已变更」（docs/34 阶段 B3 的回归）。

// TestImport_二次全短路后内容等价 全量导入后原样再导一次：
// 每个物料都应走指纹短路，且短路路径本身不得改动任何物料内容。
func TestImport_二次全短路后内容等价(t *testing.T) {
	dir := writeMinimalMaterials(t)
	s := newTestStore(t)

	importMinimal(t, s, dir, ImportOptions{})
	before := snapshotMaterialTables(t, s)

	report := importMinimal(t, s, dir, ImportOptions{})
	if report.Written != 0 {
		t.Fatalf("二次导入应全部按指纹短路，实得写入 %d 项：%+v", report.Written, report.Results)
	}
	if report.Skipped == 0 || report.Failed != 0 {
		t.Fatalf("二次导入应全部跳过且无失败：skipped=%d failed=%d", report.Skipped, report.Failed)
	}
	if after := snapshotMaterialTables(t, s); after != before {
		t.Error("二次导入（全短路）后物料内容必须逐字节不变")
	}

	// 频率档派生列在短路路径后仍与 name_frequency 一致（A3 末态归一化）
	var mismatch int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM hanzi_data h WHERE h.name_freq_tier != COALESCE(
		(SELECT tier FROM name_frequency nf WHERE nf.char = h.char AND nf.tier >= 2), 0)`).Scan(&mismatch); err != nil {
		t.Fatalf("查询频率档一致性失败: %v", err)
	}
	if mismatch != 0 {
		t.Errorf("短路路径后频率档仍有 %d 字与 name_frequency 不一致", mismatch)
	}
}

// TestImport_增量门控只重灌变更项 只改一个源文件，重灌范围必须精确限于该项
func TestImport_增量门控只重灌变更项(t *testing.T) {
	dir := writeMinimalMaterials(t)
	s := newTestStore(t)
	importMinimal(t, s, dir, ImportOptions{})

	// 仅改动禁忌组合：2 条 → 3 条
	if err := os.WriteFile(filepath.Join(dir, "forbidden_combos.json"),
		[]byte(`["作死","挽尊","坏种"]`), 0o644); err != nil {
		t.Fatal(err)
	}

	report := importMinimal(t, s, dir, ImportOptions{})
	if report.Written != 1 || report.Failed != 0 {
		t.Fatalf("单物料变更应只重灌 1 项：written=%d failed=%d results=%+v",
			report.Written, report.Failed, report.Results)
	}
	for _, r := range report.Results {
		if r.Material == "forbidden_combos" {
			if r.Skipped {
				t.Error("已变更的 forbidden_combos 不应被短路")
			}
			continue
		}
		if !r.Skipped {
			t.Errorf("未变更物料 %s 应按指纹短路，实得 %+v", r.Material, r)
		}
	}

	var n int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM forbidden_combos").Scan(&n); err != nil {
		t.Fatalf("查询禁忌表失败: %v", err)
	}
	if n != 3 {
		t.Errorf("重灌后禁忌组合数期望 3，实得 %d", n)
	}
}

// TestImport_辅表为空不误判变更 守护 docs/34 阶段 B3：
// namer 的辅表 char_groups 在源无分组时合理为空，指纹未变不得重灌。
func TestImport_辅表为空不误判变更(t *testing.T) {
	dir := writeMinimalMaterials(t)

	// 无 charGroups 的 namer.json → char_groups 辅表为空、hanzi_data 主表非空
	noGroups := `{
  "version": "1.0",
  "generated": "2026-01-01",
  "totalChars": 2,
  "chars": [
    {"char":"永","pinyin":"yong3","strokes":5,"radical":"水","wuxing":"水","level":1,"gender":"男"},
    {"char":"水","pinyin":"shui3","strokes":4,"radical":"水","wuxing":"水","level":1,"gender":"female"}
  ]
}`
	if err := os.WriteFile(filepath.Join(dir, "namer.json"), []byte(noGroups), 0o644); err != nil {
		t.Fatal(err)
	}

	s := newTestStore(t)
	importMinimal(t, s, dir, ImportOptions{})

	// 主表有数据、辅表为空
	var groups int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM char_groups").Scan(&groups); err != nil {
		t.Fatalf("查询 char_groups 失败: %v", err)
	}
	if groups != 0 {
		t.Fatalf("前置条件：辅表应为空，实得 %d 行", groups)
	}

	report := importMinimal(t, s, dir, ImportOptions{})
	r := resultOf(report, "namer")
	if !r.Skipped {
		t.Errorf("辅表为空但指纹未变，namer 不应重灌（B3 回归），实得 %+v", r)
	}

	var chars int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM hanzi_data").Scan(&chars); err != nil {
		t.Fatalf("查询 hanzi_data 失败: %v", err)
	}
	if chars != 2 {
		t.Errorf("主表应保留 2 行，实得 %d", chars)
	}
}

// TestImport_可选物料源缺失时跳过 频率类源文件按定义 optional：
// 缺失即跳过（不计失败），其余物料照常导入。
func TestImport_可选物料源缺失时跳过(t *testing.T) {
	dir := writeMinimalMaterials(t)
	if err := os.Remove(filepath.Join(dir, "name_frequency.json")); err != nil {
		t.Fatal(err)
	}

	s := newTestStore(t)
	report := importMinimal(t, s, dir, ImportOptions{})
	if report.Failed != 0 {
		t.Fatalf("可选物料缺失不应导致失败：%+v", report.Results)
	}
	r := resultOf(report, "name_frequency")
	if !r.Skipped {
		t.Errorf("可选物料源缺失应跳过，实得 %+v", r)
	}

	// 必需物料不受影响
	if resultOf(report, "namer").Err != "" {
		t.Error("可选物料缺失不应影响必需物料导入")
	}
}
