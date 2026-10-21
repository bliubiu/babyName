package sqlite

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"name/internal/materials/schema"
)

// 本文件守护 docs/40 A1/A2/A3 的指纹语义修复：
//
// T-A1 classics 目录级指纹 —— 目录内任一登记文件变更即短路失效；
// T-A2 解析失败文件指纹打点 —— 失败不登记成功指纹，修复后自愈重灌；
// T-A3 依赖解析硬失败 —— 康熙 CSV 破损中止整次导入，不写脏数据。

// T-A1：classics 指纹升级为目录级（docs/40 A1）。
// 旧实现只指纹 shijing.json，目录内其余文件（lunyu/chuci）变更被误短路——
// 死亡螺旋场景：文件损坏失败不登记指纹 → 指纹仍是旧值 → 永久短路不重试。
func TestImport_classics目录级指纹任一文件变更触发重灌(t *testing.T) {
	dir := writeMinimalMaterials(t)
	writeMinimalClassics(t, dir)
	s := newTestStore(t)
	defer func() { _ = s.Close() }()

	first := importMinimal(t, s, dir, ImportOptions{})
	r := resultOf(first, schema.ClassicsMaterial)
	if r.Err != "" || r.Skipped {
		t.Fatalf("首次导入应成功落库，实得 Err=%q Skipped=%v", r.Err, r.Skipped)
	}

	// 指纹未变更 → 短路（首次登记的已是目录级纯指纹，runImport 通用短路命中）
	second := importMinimal(t, s, dir, ImportOptions{})
	r2 := resultOf(second, schema.ClassicsMaterial)
	if !r2.Skipped || !strings.Contains(r2.Reason, "指纹未变更") {
		t.Fatalf("指纹未变更应短路跳过，实得 Skipped=%v Reason=%q", r2.Skipped, r2.Reason)
	}

	// 变更非哨兵文件（lunyu.json）：目录指纹应感知（旧实现此处仍短路，即 A1 修复点）
	lunyu := filepath.Join(dir, "lunyu.json")
	orig, err := os.ReadFile(lunyu)
	if err != nil {
		t.Fatalf("读取 lunyu.json 失败: %v", err)
	}
	if err := os.WriteFile(lunyu, append(orig, ' '), 0o644); err != nil {
		t.Fatalf("改写 lunyu.json 失败: %v", err)
	}

	third := importMinimal(t, s, dir, ImportOptions{})
	r3 := resultOf(third, schema.ClassicsMaterial)
	if r3.Err != "" || r3.Skipped {
		t.Fatalf("目录内非哨兵文件变更应触发重灌，实得 Err=%q Skipped=%v Reason=%q", r3.Err, r3.Skipped, r3.Reason)
	}
	if r3.Checksum == r2.Checksum {
		t.Fatal("重灌后登记的目录指纹应与短路前的指纹不同")
	}

	// 重灌后恢复短路
	fourth := importMinimal(t, s, dir, ImportOptions{})
	if r4 := resultOf(fourth, schema.ClassicsMaterial); !r4.Skipped {
		t.Fatalf("重灌登记新指纹后应恢复短路，实得 Reason=%q", r4.Reason)
	}
}

// T-A2：解析失败的文件纳入指纹打点（后缀 /partial），避免「失败不登记指纹
// → 指纹永驻旧值 → 每次重试都重复踩坏文件」的死亡螺旋（docs/40 A2）。
// 修复文件后指纹自然变化，重灌自愈。
func TestImport_classics解析失败指纹打点修复后自愈(t *testing.T) {
	dir := writeMinimalMaterials(t)
	writeMinimalClassics(t, dir)

	// 预埋一个解析失败的登记文件（lunyu.json 前部为垃圾字节）
	broken := []byte("{{{\"not json")
	if err := os.WriteFile(filepath.Join(dir, "lunyu.json"), broken, 0o644); err != nil {
		t.Fatalf("写入损坏 lunyu.json 失败: %v", err)
	}

	s := newTestStore(t)
	defer func() { _ = s.Close() }()

	bad := importMinimal(t, s, dir, ImportOptions{})
	rb := resultOf(bad, schema.ClassicsMaterial)
	// 无法识别文件不阻断整体（记录 skipped + Reason），但必须登记打点指纹
	if rb.Reason == "" {
		t.Fatalf("含损坏文件的导入应记录跳过原因，实得 Reason=%q Err=%q", rb.Reason, rb.Err)
	}
	if !strings.HasSuffix(rb.Checksum, partialChecksumSuffix) {
		t.Fatalf("解析失败的 classics 应登记 partial 打点指纹（后缀 %q），实得 %q", partialChecksumSuffix, rb.Checksum)
	}
	// 成功 SOS 校验：materials_state 库内登记的也是打点指纹，下次短路可复现
	var stored string
	if err := s.db.QueryRow(
		"SELECT checksum FROM materials_state WHERE material = ?", schema.ClassicsMaterial).Scan(&stored); err != nil {
		t.Fatalf("查询 materials_state 失败: %v", err)
	}
	if !strings.HasSuffix(stored, partialChecksumSuffix) {
		t.Fatalf("库内登记指纹应与结果一致含打点，实得 %q", stored)
	}

	// 修复文件：指纹变化 → 重灌 → 自愈为纯指纹
	fixed := `{"content":[{"chapter":"学而","paragraphs":["学而时习之","不亦说乎"]}]}`
	if err := os.WriteFile(filepath.Join(dir, "lunyu.json"), []byte(fixed), 0o644); err != nil {
		t.Fatalf("写入修复 lunyu.json 失败: %v", err)
	}

	good := importMinimal(t, s, dir, ImportOptions{})
	rg := resultOf(good, schema.ClassicsMaterial)
	if rg.Err != "" || rg.Skipped {
		t.Fatalf("修复后应重灌成功，实得 Err=%q Skipped=%v Reason=%q", rg.Err, rg.Skipped, rg.Reason)
	}
	if strings.HasSuffix(rg.Checksum, partialChecksumSuffix) {
		t.Fatalf("成功重灌后不应再保留打点后缀，实得 %q", rg.Checksum)
	}

	// 自愈后恢复短路（纯指纹兜底）
	again := importMinimal(t, s, dir, ImportOptions{})
	if ra := resultOf(again, schema.ClassicsMaterial); !ra.Skipped {
		t.Fatalf("自愈重灌后应恢复短路，实得 Reason=%q", ra.Reason)
	}
}

// T-A3 顺带守护 A4 语义：依赖解析失败为硬失败，后续物料不落库。
// 互动记录：修复前静默降级会让 namer 以 kangxi_strokes=0 落库、登记空剪枝，
// 之后无论修 CSV 还是重灌 kangxi，短路都会让 0 列永久存在。
func TestImport_康熙依赖解析失败硬停止且无脏数据(t *testing.T) {
	dir := writeMinimalMaterials(t)

	// case 1：依赖内的 kangxi CSV 破损 → Import 硬失败。
	// 注意 CSV 解析器前 4 行按 MIT 注释消耗，空流/短文件不会失败，
	// 这里构造「注释占满前 4 行 + 第 5 行字段数 != 4」触发 csv.Reader 报错
	broken := "# 注释1\n# 注释2\n# 注释3\n# 注释4\n只,有,三列\n"
	if err := os.WriteFile(
		filepath.Join(dir, "raw", "kangxi-strokecount.csv"), []byte(broken), 0o644); err != nil {
		t.Fatalf("写入损坏康熙 CSV 失败: %v", err)
	}
	s := newTestStore(t)
	if _, err := s.Import(ImportOptions{DataDir: dir}); err == nil {
		t.Fatal("依赖解析失败应中止整次导入（硬失败）")
	} else if !strings.Contains(err.Error(), "解析依赖源文件失败") {
		t.Fatalf("错误信息应指向依赖解析失败，实得 %v", err)
	}
	// 脏数据守护：hanzi_data 不应有任何行
	var n int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM hanzi_data").Scan(&n); err != nil {
		t.Fatalf("查询 hanzi_data 失败: %v", err)
	}
	if n != 0 {
		t.Fatalf("导入中止后 hanzi_data 不应有脏数据，实得 %d 行", n)
	}
	// 指纹不应登记（materials_state 为空）
	var registered int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM materials_state").Scan(&registered); err != nil {
		t.Fatalf("查询 materials_state 失败: %v", err)
	}
	if registered != 0 {
		t.Fatalf("导入中止后不应登记物料指纹，实得 %d 条", registered)
	}
	_ = s.Close()

	// case 2：order 中显式排除 kangxi（--only namer）时依赖留空不算失败
	dir2 := writeMinimalMaterials(t)
	s2 := newTestStore(t)
	defer func() { _ = s2.Close() }()
	report, err := s2.Import(ImportOptions{DataDir: dir2, Materials: []string{schema.NamerMaterial}})
	if err != nil {
		t.Fatalf("显式排除依赖物料时不应硬失败: %v", err)
	}
	if r := resultOf(report, schema.NamerMaterial); r.Err != "" || r.Skipped {
		t.Fatalf("namer 应正常落库，实得 Err=%q Skipped=%v", r.Err, r.Skipped)
	}
}
