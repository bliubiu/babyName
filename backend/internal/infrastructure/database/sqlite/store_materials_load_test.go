package sqlite

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"name/internal/domain/fate"
	"name/internal/domain/hanzi"
	"name/internal/domain/yijing"
	"name/internal/domain/zodiac"
)

// 本文件守护「运行时统一只读 SQLite 数据源」（docs/31 ADR-008）：
// 库 → 内存 的装载路径必须与 JSON 路径产出等价的内存状态。

// Test运行时从库装载_等价于JSON路径 核心等价性断言：
// 同一份物料，分别经 import→库→内存 与 JSON→内存，两条路径的内存状态必须一致。
func Test运行时从库装载_等价于JSON路径(t *testing.T) {
	dir := writeMinimalMaterials(t)

	// 路径一：JSON → 内存（回落路径）
	if err := hanzi.LoadKangxiStrokesFromCSV(dir); err != nil {
		t.Fatalf("康熙 CSV 装载失败: %v", err)
	}
	if err := hanzi.LoadNamerFromJSON(dir); err != nil {
		t.Fatalf("namer JSON 装载失败: %v", err)
	}
	jsonKangxi := hanzi.GetKangxiStrokes("永")
	jsonHanzi, _ := hanzi.GetHanzi("永")
	jsonLevel := hanzi.GetNamerLevel("永")
	jsonNamer, jsonOk := hanzi.GetNamerChar("永")
	if !jsonOk {
		t.Fatal("JSON 路径应能读出「永」的 namer 数据")
	}

	// 路径二：JSON → import → 库 → 内存
	s := newTestStoreIn(t, dir)
	if _, err := s.Import(ImportOptions{DataDir: dir}); err != nil {
		t.Fatalf("导入失败: %v", err)
	}
	report := s.LoadMaterialsFromDBWithReport()
	// 本用例的最小物料集不含频率类与门禁扩展物料，属预期缺失
	for _, m := range report.Missing {
		switch m {
		case "name_frequency", "name_bigram", "allowlist", "compliance_redline":
		default:
			t.Errorf("核心物料不应缺失: %s", m)
		}
	}
	if report.Loaded == nil {
		t.Fatal("应报告已装载物料")
	}
	for _, want := range []string{"kangxi", "namer", "naming_gate", "forbidden_combos", "yijing", "zodiac"} {
		found := false
		for _, m := range report.Loaded {
			if m == want {
				found = true
			}
		}
		if !found {
			t.Errorf("物料 %s 应装载成功", want)
		}
	}

	dbKangxi := hanzi.GetKangxiStrokes("永")
	if dbKangxi != jsonKangxi {
		t.Errorf("康熙笔画两条路径不一致：JSON=%d 库=%d", jsonKangxi, dbKangxi)
	}

	// NamerCharMap 必须随库装载重建：GetNamerLevel 是等级闸门的唯一输入，
	// 库路径漏建会放行三级/表外字进推荐榜（e2e 门禁曾因此失败）
	if dbLevel := hanzi.GetNamerLevel("永"); jsonLevel != dbLevel {
		t.Errorf("「永」等级不一致：JSON=%d 库=%d", jsonLevel, dbLevel)
	}
	if got := hanzi.GetNamerLevel("和"); got != 2 {
		t.Errorf("「和」为二级字，实得等级 %d（库路径 NamerCharMap 未重建则恒为 0）", got)
	}
	dbNamer, dbOk := hanzi.GetNamerChar("永")
	if !dbOk {
		t.Fatal("库路径应能读出「永」的 namer 数据")
	}
	if dbNamer.Level != jsonNamer.Level || dbNamer.Gender != jsonNamer.Gender ||
		dbNamer.PositiveScore != jsonNamer.PositiveScore || dbNamer.IsPolyphonic != jsonNamer.IsPolyphonic {
		t.Errorf("namer 投影字段不一致：JSON={%d %s %d %t} 库={%d %s %d %t}",
			jsonNamer.Level, jsonNamer.Gender, jsonNamer.PositiveScore, jsonNamer.IsPolyphonic,
			dbNamer.Level, dbNamer.Gender, dbNamer.PositiveScore, dbNamer.IsPolyphonic)
	}

	dbHanzi, ok := hanzi.GetHanzi("永")
	if !ok {
		t.Fatal("库路径应能读出「永」")
	}
	if dbHanzi.Pinyin != jsonHanzi.Pinyin {
		t.Errorf("拼音不一致：JSON=%s 库=%s", jsonHanzi.Pinyin, dbHanzi.Pinyin)
	}
	if dbHanzi.KangxiStrokes != jsonHanzi.KangxiStrokes {
		t.Errorf("康熙笔画字段不一致：JSON=%d 库=%d", jsonHanzi.KangxiStrokes, dbHanzi.KangxiStrokes)
	}
	if dbHanzi.Level != jsonHanzi.Level {
		t.Errorf("字表等级不一致：JSON=%d 库=%d", jsonHanzi.Level, dbHanzi.Level)
	}
	if dbHanzi.Gender != jsonHanzi.Gender {
		t.Errorf("性别不一致：JSON=%s 库=%s", jsonHanzi.Gender, dbHanzi.Gender)
	}
	if dbHanzi.IsPolyphonic != jsonHanzi.IsPolyphonic {
		t.Errorf("多音字标记不一致：JSON=%t 库=%t", jsonHanzi.IsPolyphonic, dbHanzi.IsPolyphonic)
	}
	// 频率档：JSON 路径经 GetCharTier 合并，库路径由导入末态 syncNameFreqTier
	// 以 name_frequency 表为唯一真源归一化，两边必须等价
	if dbHanzi.NameFreqTier != jsonHanzi.NameFreqTier {
		t.Errorf("人名频率档不一致：JSON=%d 库=%d", jsonHanzi.NameFreqTier, dbHanzi.NameFreqTier)
	}
	if dbNamer.NameFreqTier != jsonNamer.NameFreqTier {
		t.Errorf("namer 频率档不一致：JSON=%d 库=%d", jsonNamer.NameFreqTier, dbNamer.NameFreqTier)
	}
}

// Test导入频率档回填hanzi_data 守护导入末态的频率档归一化（docs/31 §9.6 步骤4）：
// namer.json 无 nameFreqTier 真值，必须由频率物料派生进 hanzi_data，
// 运行时读库即得规范态，与 JSON 路径的 GetCharTier 合并口径一致（仅 Tier ≥ 2）。
//
// 归一化收敛在 Import() 末尾的 syncNameFreqTier 单点完成（docs/34 阶段 A3），
// 不再由 importCharFreq 在自身事务内 UPDATE hanzi_data。
func Test导入频率档回填hanzi_data(t *testing.T) {
	dir := writeMinimalMaterials(t)
	s := newTestStoreIn(t, dir)
	if _, err := s.Import(ImportOptions{DataDir: dir}); err != nil {
		t.Fatalf("导入失败: %v", err)
	}

	cases := []struct {
		char string
		want int
	}{
		{"永", 5},
		{"水", 4},
		{"和", 0}, // Tier 1 低于回填口径，应保持 0
	}
	for _, c := range cases {
		var got int
		if err := s.db.QueryRow("SELECT name_freq_tier FROM hanzi_data WHERE char = ?", c.char).Scan(&got); err != nil {
			t.Fatalf("查询 %s 频率档失败: %v", c.char, err)
		}
		if got != c.want {
			t.Errorf("「%s」回填档位期望 %d，实得 %d", c.char, c.want, got)
		}
	}

	// 装载后内存态同样拿到回填值
	s.LoadMaterialsFromDB()
	h, ok := hanzi.GetHanzi("永")
	if !ok {
		t.Fatal("应能读出「永」")
	}
	if h.NameFreqTier != 5 {
		t.Errorf("装载后「永」频率档期望 5，实得 %d", h.NameFreqTier)
	}
	if n, ok := hanzi.GetNamerChar("永"); !ok || n.NameFreqTier != 5 {
		t.Errorf("装载后 namer 投影频率档期望 5，实得 %d（存在=%t）", n.NameFreqTier, ok)
	}
}

func Test运行时从库装载_字库条数一致(t *testing.T) {
	dir := writeMinimalMaterials(t)
	s := newTestStoreIn(t, dir)
	if _, err := s.Import(ImportOptions{DataDir: dir}); err != nil {
		t.Fatalf("导入失败: %v", err)
	}
	s.LoadMaterialsFromDB()

	if got := hanzi.HanziCount(); got != 3 {
		t.Errorf("内存字库条数期望 3，实得 %d", got)
	}
}

func Test运行时从库装载_偏旁分组(t *testing.T) {
	dir := writeMinimalMaterials(t)
	s := newTestStoreIn(t, dir)
	if _, err := s.Import(ImportOptions{DataDir: dir}); err != nil {
		t.Fatalf("导入失败: %v", err)
	}
	s.LoadMaterialsFromDB()

	groups := hanzi.GetNamerGroups()
	if len(groups) != 1 {
		t.Fatalf("偏旁分组数期望 1，实得 %d", len(groups))
	}
	if groups[0].Radical != "水" {
		t.Errorf("部首期望 水，实得 %s", groups[0].Radical)
	}
	if len(groups[0].Chars) != 2 {
		t.Errorf("分组内字数期望 2，实得 %d", len(groups[0].Chars))
	}
}

func Test运行时从库装载_门禁与禁忌(t *testing.T) {
	dir := writeMinimalMaterials(t)
	s := newTestStoreIn(t, dir)
	if _, err := s.Import(ImportOptions{DataDir: dir}); err != nil {
		t.Fatalf("导入失败: %v", err)
	}
	s.LoadMaterialsFromDB()

	if !fate.IsNonNamingChar("伙") {
		t.Error("门禁字「伙」应被判为不可入名")
	}
	if fate.IsNonNamingChar("永") {
		t.Error("非门禁字不应被误判")
	}
	if got := fate.NonNamingCharCount(); got != 3 {
		t.Errorf("门禁字数期望 3，实得 %d", got)
	}
	if got := fate.ForbiddenComboCount(); got != 2 {
		t.Errorf("禁忌组合数期望 2，实得 %d", got)
	}
}

func Test运行时从库装载_易经与生肖(t *testing.T) {
	dir := writeMinimalMaterials(t)
	s := newTestStoreIn(t, dir)
	if _, err := s.Import(ImportOptions{DataDir: dir}); err != nil {
		t.Fatalf("导入失败: %v", err)
	}
	s.LoadMaterialsFromDB()

	h, ok := yijing.GetHexagram(1)
	if !ok {
		t.Fatal("应能从库读出卦象 1")
	}
	if h.Name != "乾" {
		t.Errorf("卦名期望 乾，实得 %s", h.Name)
	}
	if len(h.YaoCi) != 3 {
		t.Errorf("爻辞条数期望 3，实得 %d", len(h.YaoCi))
	}

	z, ok := zodiac.GetZodiac("鼠")
	if !ok {
		t.Fatal("应能从库读出生肖「鼠」")
	}
	if z.Wuxing != "水" {
		t.Errorf("生肖五行期望 水，实得 %s", z.Wuxing)
	}
	if len(z.LuckyNumbers) != 2 {
		t.Errorf("幸运数字条数期望 2，实得 %d", len(z.LuckyNumbers))
	}
}

// Test运行时从库装载_频率档 守护库路径的人名频率档装载：
// 仅 Tier ≥ 2 入内存，与 JSON 路径 LoadFrequencyDB 过滤口径一致。
func Test运行时从库装载_频率档(t *testing.T) {
	dir := writeMinimalMaterials(t)
	freq := `{"meta":{"total_names":100,"built_at":"2026-01-01"},"frequency":[
      {"char":"永","count":500,"rank":1,"tier":5},
      {"char":"水","count":300,"rank":2,"tier":4},
      {"char":"和","count":10,"rank":9,"tier":1}
    ]}`
	if err := os.WriteFile(filepath.Join(dir, "name_frequency.json"), []byte(freq), 0o644); err != nil {
		t.Fatal(err)
	}

	s := newTestStoreIn(t, dir)
	if _, err := s.Import(ImportOptions{DataDir: dir}); err != nil {
		t.Fatalf("导入失败: %v", err)
	}
	s.LoadMaterialsFromDB()

	// Tier 1 不入内存（与 JSON 路径同一过滤口径）
	if got := hanzi.GetCharTier("永"); got != 5 {
		t.Errorf("「永」频率档期望 5，实得 %d", got)
	}
	if got := hanzi.GetCharTier("水"); got != 4 {
		t.Errorf("「水」频率档期望 4，实得 %d", got)
	}
	if got := hanzi.GetCharTier("和"); got != 0 {
		t.Errorf("Tier<2 的字不应入内存，实得档位 %d", got)
	}
}

func Test运行时从库装载_库为空时报告缺失(t *testing.T) {
	dir := t.TempDir() // 空目录，无任何物料
	s := newTestStoreIn(t, dir)

	report := s.LoadMaterialsFromDBWithReport()
	if len(report.Loaded) != 0 {
		t.Errorf("空库不应装载出物料，实得 %v", report.Loaded)
	}
	if len(report.Missing) == 0 {
		t.Error("空库应报告缺失物料")
	}
	if err := s.VerifyMaterialsFromDB(); err == nil {
		t.Error("VerifyMaterialsFromDB 对不完整库应报错以引导 import")
	}
}

// TestMaterialNames_覆盖全部已接入物料 materialNames 必须覆盖
// LoadMaterialsFromDB 的全部装载分支（docs/34 阶段 B2：此前手抄清单漏列
// allowlist / compliance_redline，装载报告永远不报其缺失）。
//
// 期望列表同时充当「新增物料时的检查清单」：接入新物料必须同步补
// LoadMaterialsFromDB 装载分支，否则本测试失败。
func TestMaterialNames_覆盖全部已接入物料(t *testing.T) {
	want := []string{
		"kangxi", "namer", "name_frequency", "name_bigram",
		"naming_gate", "forbidden_combos", "allowlist", "compliance_redline",
		"yijing", "zodiac",
	}
	if got := materialNames(); !reflect.DeepEqual(got, want) {
		t.Errorf("materialNames 与装载分支集合不一致\n got: %v\nwant: %v", got, want)
	}

	// legacy 物料不得混入装载报告
	for _, name := range materialNames() {
		if legacyMaterials[name] {
			t.Errorf("legacy 物料 %s 不应参与运行时装载报告", name)
		}
	}
}

// newTestStoreIn 在给定 dataDir 上建 Store（物料导入/装载测试用）
func newTestStoreIn(t *testing.T, dataDir string) *Store {
	t.Helper()
	s, err := NewStore(filepath.Join(t.TempDir(), "test.db"), dataDir)
	if err != nil {
		t.Fatalf("创建 Store 失败: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// TestLoadMaterials_恢复频率语料总人数 回归：导入框架只落行数据不落 meta，
// 库装载后 GetTotalNames() 恒为 0 → 统计页占比全部显示 0.0000%。
// 装载侧须用构造恒等式恢复：total = Σ单字count − Σ双字count
// （单字表计每个人名的每个「名」字：Σ单字 = 单字名数 + 2×双字名数；
//   双字表计每个双字名一次：Σ双字 = 双字名数 ⇒ 差 = 语料总人数）。
// 夹具：单字 500+300+10=810，双字 20 ⇒ 期望 790。
func TestLoadMaterials_恢复频率语料总人数(t *testing.T) {
	dir := writeMinimalMaterials(t)
	s := newTestStoreIn(t, dir)
	if _, err := s.Import(ImportOptions{DataDir: dir}); err != nil {
		t.Fatalf("导入失败: %v", err)
	}
	if report := s.LoadMaterialsFromDBWithReport(); report.Loaded == nil {
		t.Fatal("装载失败")
	}
	if got := hanzi.GetTotalNames(); got != 790 {
		t.Fatalf("装载后 GetTotalNames() = %d，期望 790（Σ单字810 − Σ双字20）", got)
	}
}
