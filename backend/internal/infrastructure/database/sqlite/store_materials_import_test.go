package sqlite

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

// 本文件守护 `materials import` 的端到端语义（docs/31 ADR-008 / ADR-011 / ADR-014）：
// 归一化 → 校验 → 单事务写入 → 指纹登记 → 二次导入短路。

// writeMinimalMaterials 在 dir 下生成一份最小可用的物料集
func writeMinimalMaterials(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	raw := filepath.Join(dir, "raw")
	if err := os.MkdirAll(raw, 0o755); err != nil {
		t.Fatalf("创建 raw 目录失败: %v", err)
	}

	// 康熙笔画 CSV：BOM + 4 行注释 + header + 数据
	kangxi := "\xEF\xBB\xBF" +
		"MIT License\nCopyright (c)\nhttps://example.com/kangxi\n\n" +
		"CodePoint,Value,Character,Strokes\n" +
		"20108,27700,永,5\n" +
		"27700,27700,水,4\n"
	if err := os.WriteFile(filepath.Join(raw, "kangxi-strokecount.csv"), []byte(kangxi), 0o644); err != nil {
		t.Fatalf("写入康熙 CSV 失败: %v", err)
	}

	namer := `{
  "version": "1.0",
  "generated": "2026-01-01",
  "totalChars": 3,
  "chars": [
    {"char":"永","pinyin":"yong3","strokes":5,"radical":"水","meaning":"水流长","wuxing":"水","level":1,"gender":"男","tone":3,"positiveScore":90,"namingCategories":["品德"],"isPolyphonic":true},
    {"char":"水","pinyin":"shui3","strokes":4,"radical":"水","meaning":"水体","wuxing":"水","level":1,"gender":"female"},
    {"char":"和","pinyin":"he2","strokes":8,"radical":"禾","wuxing":"","level":2}
  ],
  "charGroups": [
    {"radical":"水","name":"水字旁","meaning":"与水有关","chars":["永","水"]}
  ]
}`
	if err := os.WriteFile(filepath.Join(dir, "namer.json"), []byte(namer), 0o644); err != nil {
		t.Fatalf("写入 namer.json 失败: %v", err)
	}

	gate := `["伙","腕","之"]`
	if err := os.WriteFile(filepath.Join(dir, "naming_quality.json"), []byte(gate), 0o644); err != nil {
		t.Fatalf("写入门禁表失败: %v", err)
	}

	forbidden := `["作死","挽尊"]`
	if err := os.WriteFile(filepath.Join(dir, "forbidden_combos.json"), []byte(forbidden), 0o644); err != nil {
		t.Fatalf("写入禁忌表失败: %v", err)
	}

	yijing := `[{"id":1,"name":"乾","number":1,"symbol":"☰","upper_trigram":7,"lower_trigram":7,"gua_ci":"元亨利贞","xiang_ci":"天行健","yao_ci":["初九","九二","九三"],"interpretation":"刚健中正"}]`
	if err := os.WriteFile(filepath.Join(dir, "yijing.json"), []byte(yijing), 0o644); err != nil {
		t.Fatalf("写入易经失败: %v", err)
	}

	zodiac := `[{"id":1,"name":"鼠","wuxing":"水","compatible":["龙","猴"],"conflicting":["马"],"avoid_chars":"午","lucky_number":[1,13]}]`
	if err := os.WriteFile(filepath.Join(dir, "zodiac.json"), []byte(zodiac), 0o644); err != nil {
		t.Fatalf("写入生肖失败: %v", err)
	}

	// 单字/双字人名频率（JSON 路径的 NameFreqTier 合并依赖两个文件同时存在，
	// 等价性测试需要两边口径一致，故纳入最小物料集）
	freq := `{"meta":{"total_names":100,"built_at":"2026-01-01"},"frequency":[
      {"char":"永","count":500,"rank":1,"tier":5},
      {"char":"水","count":300,"rank":2,"tier":4},
      {"char":"和","count":10,"rank":9,"tier":1}
    ]}`
	if err := os.WriteFile(filepath.Join(dir, "name_frequency.json"), []byte(freq), 0o644); err != nil {
		t.Fatalf("写入单字频率失败: %v", err)
	}
	bigram := `{"meta":{"total_names":100,"built_at":"2026-01-01"},"frequency":[
      {"char1":"永","char2":"水","count":20,"rank":3,"tier":4}
    ]}`
	if err := os.WriteFile(filepath.Join(dir, "name_bigram_frequency.json"), []byte(bigram), 0o644); err != nil {
		t.Fatalf("写入双字频率失败: %v", err)
	}

	return dir
}

// importMinimal 执行一次全量导入并返回报告
func importMinimal(t *testing.T, s *Store, dir string, opt ImportOptions) *ImportReport {
	t.Helper()
	opt.DataDir = dir
	report, err := s.Import(opt)
	if err != nil {
		t.Fatalf("导入失败: %v", err)
	}
	return report
}

func resultOf(report *ImportReport, material string) ImportResult {
	for _, r := range report.Results {
		if r.Material == material {
			return r
		}
	}
	return ImportResult{Material: material, Err: "报告中缺少该物料"}
}

func TestImport_全量导入成功并登记指纹(t *testing.T) {
	dir := writeMinimalMaterials(t)
	s := newTestStore(t)

	report := importMinimal(t, s, dir, ImportOptions{})
	if report.Failed != 0 {
		t.Fatalf("不应有失败物料，实际: %+v", report.Results)
	}

	states, err := s.MaterialStates()
	if err != nil {
		t.Fatalf("读取物料状态失败: %v", err)
	}
	if len(states) < 6 {
		t.Fatalf("登记的物料数应 ≥6，实际 %d", len(states))
	}
	for _, st := range states {
		if st.Checksum == "" {
			t.Errorf("物料 %s 缺少指纹", st.Material)
		}
		if st.RowCount == 0 {
			t.Errorf("物料 %s 行数为 0", st.Material)
		}
	}

	if got := resultOf(report, "namer").Rows; got != 4 { // 3 字 + 1 分组
		t.Errorf("namer 导入行数期望 4（3 字 + 1 分组），实得 %d", got)
	}
}

func TestImport_汉字表已归一化入库(t *testing.T) {
	dir := writeMinimalMaterials(t)
	s := newTestStore(t)
	importMinimal(t, s, dir, ImportOptions{})

	// 性别归一：中文「男」应落 male
	var gender string
	if err := s.db.QueryRow("SELECT gender FROM hanzi_data WHERE char = '永'").Scan(&gender); err != nil {
		t.Fatalf("查询 gender 失败: %v", err)
	}
	if gender != "male" {
		t.Errorf("中文「男」应归一为 male，实得 %q", gender)
	}

	// 多音字标记必须落库（此前 NamerChar 丢弃该字段）
	var poly int
	if err := s.db.QueryRow("SELECT is_polyphonic FROM hanzi_data WHERE char = '永'").Scan(&poly); err != nil {
		t.Fatalf("查询 is_polyphonic 失败: %v", err)
	}
	if poly != 1 {
		t.Errorf("is_polyphonic 期望 1，实得 %d", poly)
	}

	// 康熙笔画合并（拓扑顺序保证 kangxi 先于 namer）
	var kangxi int
	if err := s.db.QueryRow("SELECT kangxi_strokes FROM hanzi_data WHERE char = '永'").Scan(&kangxi); err != nil {
		t.Fatalf("查询 kangxi_strokes 失败: %v", err)
	}
	if kangxi != 5 {
		t.Errorf("康熙笔画期望 5（由 CSV 合并），实得 %d", kangxi)
	}

	// 空性别应归一为 neutral
	var neutral string
	if err := s.db.QueryRow("SELECT gender FROM hanzi_data WHERE char = '和'").Scan(&neutral); err != nil {
		t.Fatalf("查询中性字 gender 失败: %v", err)
	}
	if neutral != "neutral" {
		t.Errorf("空性别应归一为 neutral，实得 %q", neutral)
	}
}

func TestImport_偏旁分组独立入库(t *testing.T) {
	dir := writeMinimalMaterials(t)
	s := newTestStore(t)
	importMinimal(t, s, dir, ImportOptions{})

	var n int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM char_groups WHERE radical = '水'").Scan(&n); err != nil {
		t.Fatalf("查询 char_groups 失败: %v", err)
	}
	if n != 1 {
		t.Fatalf("偏旁分组应入库 1 行，实得 %d", n)
	}
	var chars string
	if err := s.db.QueryRow("SELECT chars FROM char_groups WHERE radical = '水'").Scan(&chars); err != nil {
		t.Fatalf("查询分组字表失败: %v", err)
	}
	if chars != `["水","永"]` {
		t.Errorf("分组内字表应去重排序后为 [\"水\",\"永\"]，实得 %s", chars)
	}
}

func TestImport_门禁与禁忌独立入库(t *testing.T) {
	dir := writeMinimalMaterials(t)
	s := newTestStore(t)
	importMinimal(t, s, dir, ImportOptions{})

	var gate int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM naming_gate").Scan(&gate); err != nil {
		t.Fatalf("查询门禁表失败: %v", err)
	}
	if gate != 3 {
		t.Errorf("门禁字数期望 3，实得 %d", gate)
	}

	var bad int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM forbidden_combos").Scan(&bad); err != nil {
		t.Fatalf("查询禁忌表失败: %v", err)
	}
	if bad != 2 {
		t.Errorf("禁忌组合数期望 2，实得 %d", bad)
	}
}

func TestImport_二次导入按指纹短路(t *testing.T) {
	dir := writeMinimalMaterials(t)
	s := newTestStore(t)

	importMinimal(t, s, dir, ImportOptions{})
	report := importMinimal(t, s, dir, ImportOptions{})

	if report.Written != 0 {
		t.Errorf("二次导入应全部跳过，实际写入 %d 项", report.Written)
	}
	r := resultOf(report, "namer")
	if !r.Skipped {
		t.Errorf("namer 二次导入应被跳过，实得 %+v", r)
	}
	if r.Version == "" {
		t.Error("跳过的物料应回填库内记录的版本")
	}
}

func TestImport_源变更后重新导入(t *testing.T) {
	dir := writeMinimalMaterials(t)
	s := newTestStore(t)
	importMinimal(t, s, dir, ImportOptions{})

	// 修改门禁表：1 字 → 2 字
	if err := os.WriteFile(filepath.Join(dir, "naming_quality.json"), []byte(`["伙","腕","之","扇"]`), 0o644); err != nil {
		t.Fatal(err)
	}

	report := importMinimal(t, s, dir, ImportOptions{})
	if resultOf(report, "naming_gate").Skipped {
		t.Error("源变更后不应跳过重灌")
	}

	var n int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM naming_gate").Scan(&n); err != nil {
		t.Fatalf("查询门禁表失败: %v", err)
	}
	if n != 4 {
		t.Errorf("重灌后门禁字数期望 4，实得 %d", n)
	}
}

func TestImport_删字传导到库(t *testing.T) {
	dir := writeMinimalMaterials(t)
	s := newTestStore(t)
	importMinimal(t, s, dir, ImportOptions{})

	// 从字表移除「和」
	trimmed := `{"version":"1.0","generated":"2026-01-02","totalChars":2,
      "chars":[
        {"char":"永","pinyin":"yong3","strokes":5,"radical":"水","wuxing":"水","level":1,"gender":"男"},
        {"char":"水","pinyin":"shui3","strokes":4,"radical":"水","wuxing":"水","level":1,"gender":"female"}
      ],
      "charGroups":[{"radical":"水","name":"水字旁","meaning":"","chars":["永","水"]}]}`
	if err := os.WriteFile(filepath.Join(dir, "namer.json"), []byte(trimmed), 0o644); err != nil {
		t.Fatal(err)
	}

	importMinimal(t, s, dir, ImportOptions{})

	var n int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM hanzi_data WHERE char = '和'").Scan(&n); err != nil {
		t.Fatalf("查询已删字失败: %v", err)
	}
	if n != 0 {
		t.Error("物料中移除的字必须同步从库中删除")
	}
}

func TestImport_幂等(t *testing.T) {
	dir := writeMinimalMaterials(t)
	s := newTestStore(t)

	importMinimal(t, s, dir, ImportOptions{})
	first := snapshotMaterialTables(t, s)

	importMinimal(t, s, dir, ImportOptions{Lenient: true}) // lenient 绕过指纹短路，强制重灌
	second := snapshotMaterialTables(t, s)

	importMinimal(t, s, dir, ImportOptions{Lenient: true})
	third := snapshotMaterialTables(t, s)

	if first != second || second != third {
		t.Errorf("重复导入结果应完全一致：\n首次=%s\n二次=%s\n三次=%s", first, second, third)
	}
}

func TestImport_脏数据被拒且不写库(t *testing.T) {
	dir := writeMinimalMaterials(t)
	s := newTestStore(t)

	// 笔画为 0（违反 ≥1 规则）
	dirty := `{"version":"1.0","generated":"x","chars":[{"char":"永","pinyin":"yong3","strokes":0,"level":1}],"charGroups":[]}`
	if err := os.WriteFile(filepath.Join(dir, "namer.json"), []byte(dirty), 0o644); err != nil {
		t.Fatal(err)
	}

	report := importMinimal(t, s, dir, ImportOptions{})
	r := resultOf(report, "namer")
	if r.Err == "" {
		t.Fatal("脏数据应导致 namer 导入失败")
	}

	var n int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM hanzi_data").Scan(&n); err != nil {
		t.Fatalf("查询字表失败: %v", err)
	}
	if n != 0 {
		t.Errorf("校验失败时不得写入任何字表数据，实得 %d 行", n)
	}
}

func TestImport_未知物料报错(t *testing.T) {
	dir := writeMinimalMaterials(t)
	s := newTestStore(t)

	report, err := s.Import(ImportOptions{DataDir: dir, Materials: []string{"不存在的物料"}})
	if err == nil {
		t.Fatalf("未知物料应报错，报告=%+v", report)
	}
}

func TestImport_只导入指定物料并自动补齐依赖(t *testing.T) {
	dir := writeMinimalMaterials(t)
	s := newTestStore(t)

	// naming_gate 依赖 namer，namer 依赖 kangxi → 应自动补齐为 3 项
	report := importMinimal(t, s, dir, ImportOptions{Materials: []string{"naming_gate"}})
	if report.Failed != 0 {
		t.Fatalf("不应失败：%+v", report.Results)
	}

	names := make(map[string]bool, len(report.Results))
	for _, r := range report.Results {
		names[r.Material] = true
	}
	for _, want := range []string{"naming_gate", "namer", "kangxi"} {
		if !names[want] {
			t.Errorf("应自动补齐依赖物料 %s", want)
		}
	}
	if len(report.Results) != 3 {
		t.Errorf("报告项数应为 3（目标 + 依赖闭包），实得 %d", len(report.Results))
	}

	// 未列入的物料不应被导入
	if names["zodiac"] {
		t.Error("未列入的物料不应被导入")
	}
	var n int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM zodiac").Scan(&n); err == nil && n > 0 {
		t.Errorf("未列入的物料不应有数据，实得 %d 行", n)
	}

	// 康熙笔画必须经「源文件预解析注入」合并进 hanzi_data（docs/34 阶段 A2）：
	// 部分导入同样成立，不依赖回读 kangxi_strokes 表的落库时序。
	var kangxi int
	if err := s.db.QueryRow("SELECT kangxi_strokes FROM hanzi_data WHERE char = '永'").Scan(&kangxi); err != nil {
		t.Fatalf("查询 kangxi_strokes 失败: %v", err)
	}
	if kangxi != 5 {
		t.Errorf("部分导入时康熙笔画期望 5（由源文件注入合并），实得 %d", kangxi)
	}
}

func TestImport_干运行不写库(t *testing.T) {
	dir := writeMinimalMaterials(t)
	s := newTestStore(t)

	report := importMinimal(t, s, dir, ImportOptions{DryRun: true})
	if report.Failed != 0 {
		t.Fatalf("干运行不应失败：%+v", report.Results)
	}

	var n int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM hanzi_data").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("干运行不得写库，实得 %d 行", n)
	}
	states, _ := s.MaterialStates()
	if len(states) != 0 {
		t.Error("干运行不得登记物料状态")
	}
}

// snapshotMaterialTables 读出全部物料表内容快照（用于幂等比对）
//
// 刻意**排除 materials_state**：它是导入元数据（含 imported_at 时间戳），
// 每次导入必然不同，纳入比对会让幂等断言恒假。
//
// 同时**排除 updated_at 列**：该列由数据库 `DEFAULT CURRENT_TIMESTAMP` 自维护
// （见 store.go 建表 DDL 与 store_schema_guard_test.go 同口径的注释），两次导入
// 只要跨越秒边界取值就不同——曾导致本测试随机失败（快照含该列时断言恒假）。
// 幂等语义针对物料内容，不含数据库自维护的时间戳。
func snapshotMaterialTables(t *testing.T, s *Store) string {
	t.Helper()
	out := ""
	for _, table := range MaterialTables() {
		if table == "materials_state" {
			continue
		}
		rows, err := s.db.Query("SELECT * FROM " + table)
		if err != nil {
			continue
		}
		cols, _ := rows.Columns()
		cols = excludeVolatileCols(cols)
		q := "SELECT " + joinCols(cols) + " FROM " + table + " ORDER BY " + cols[0]
		rows.Close()

		r2, err := s.db.Query(q)
		if err != nil {
			continue
		}
		out += "== " + table + " ==\n"
		for r2.Next() {
			cells := make([]any, len(cols))
			for i := range cells {
				cells[i] = new(sql.NullString)
			}
			if err := r2.Scan(cells...); err != nil {
				continue
			}
			for i, c := range cells {
				out += cols[i] + "=" + c.(*sql.NullString).String + "|"
			}
			out += "\n"
		}
		_ = r2.Close()
	}
	return out
}

// excludeVolatileCols 过滤数据库自维护的易变列（updated_at）
//
// 与 store.go 建表 DDL 的 `DEFAULT CURRENT_TIMESTAMP` 对应：该值由数据库在写入时
// 生成，同一输入跨秒重灌即不同，不参与「物料内容幂等」比对。
func excludeVolatileCols(cols []string) []string {
	out := make([]string, 0, len(cols))
	for _, c := range cols {
		if c == "updated_at" {
			continue
		}
		out = append(out, c)
	}
	return out
}

func joinCols(cols []string) string {
	s := ""
	for i, c := range cols {
		if i > 0 {
			s += ","
		}
		s += c
	}
	return s
}
