package sqlite

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	"name/internal/domain/classics"
	"name/internal/domain/hanzi"
	"name/internal/infrastructure/data"
	"name/internal/materials/schema"
	"name/internal/testutil"
)

// 本文件守护经典语料物料（docs/31 ADR-008）的物化与还原语义：
// 导入期物化 classics_extracts / classics_poem_entries，运行期可零 JSON 还原。

// writeMinimalClassics 在 dir 下补齐 classics 需要的经典源文件（诗经 + 论语）
//
// classics 物料的依赖链是 namer → classics；importClassics 需要字库在内存中
// 补全拼音/五行，因此必须同时提供 namer.json（writeMinimalMaterials 已含）。
func writeMinimalClassics(t *testing.T, dir string) {
	t.Helper()

	shijing := `[{"title":"关雎","chapter":"国风·周南","section":"关雎","content":["关关雎鸠","在河之洲"]},
      {"title":"蒹葭","chapter":"国风·秦风","section":"蒹葭","content":["蒹葭苍苍","白露为霜","所谓伊人","在水一方"]}]`
	if err := os.WriteFile(filepath.Join(dir, "shijing.json"), []byte(shijing), 0o644); err != nil {
		t.Fatalf("写入 shijing.json 失败: %v", err)
	}

	lunyu := `{"content":[
      {"chapter":"学而","paragraphs":["学而时习之","不亦说乎"]}
    ]}`
	if err := os.WriteFile(filepath.Join(dir, "lunyu.json"), []byte(lunyu), 0o644); err != nil {
		t.Fatalf("写入 lunyu.json 失败: %v", err)
	}

	// 诗经/楚辞是 LoadFromJSON 的硬依赖（缺失即返回错误），必须提供
	chuci := `[{"title":"离骚","section":"离骚","author":"屈原","content":["路漫漫其修远兮","吾将上下而求索"]}]`
	if err := os.WriteFile(filepath.Join(dir, "chuci.json"), []byte(chuci), 0o644); err != nil {
		t.Fatalf("写入 chuci.json 失败: %v", err)
	}
}

func TestImportClassics_骨架与物化表一并写入(t *testing.T) {
	dir := writeMinimalMaterials(t)
	writeMinimalClassics(t, dir)
	s := newTestStore(t)

	report := importMinimal(t, s, dir, ImportOptions{})
	r := resultOf(report, schema.ClassicsMaterial)
	if r.Err != "" {
		t.Fatalf("经典语料导入失败: %s", r.Err)
	}
	if r.Rows == 0 {
		t.Fatal("经典语料导入行数不应为 0")
	}

	var books, sections, extracts, poems int
	for _, q := range []struct {
		table string
		dst   *int
	}{
		{"classics_books", &books},
		{"classics_sections", &sections},
		{"classics_extracts", &extracts},
		{"classics_poem_entries", &poems},
	} {
		if err := s.db.QueryRow("SELECT COUNT(*) FROM " + q.table).Scan(q.dst); err != nil {
			t.Fatalf("查询 %s 失败: %v", q.table, err)
		}
	}
	// writeMinimalMaterials 已含 yijing.json（也在 classics 语料清单内），
	// 故此处为 shijing + chuci + lunyu + yijing 共 4 本
	if books != 4 {
		t.Errorf("classics_books 应为 4（shijing/chuci/lunyu/yijing），实得 %d", books)
	}
	if sections == 0 {
		t.Error("classics_sections 不应为空")
	}
	if extracts == 0 {
		t.Error("classics_extracts（物化字集）不应为空")
	}
	if poems == 0 {
		t.Error("classics_poem_entries（物化诗词索引）不应为空")
	}
}

// TestImportClassics_段落表已废弃 docs/34 阶段 E-B：classics_paragraphs
// 无任何运行期读者却占库体积 8%，且与 poem_entries 重复 42.2% 文本，已整体删除。
func TestImportClassics_段落表已废弃(t *testing.T) {
	dir := writeMinimalMaterials(t)
	writeMinimalClassics(t, dir)
	s := newTestStore(t)
	importMinimal(t, s, dir, ImportOptions{})

	if _, err := s.db.Query("SELECT COUNT(*) FROM classics_paragraphs"); err == nil {
		t.Error("classics_paragraphs 应已废弃，不应再存在")
	}
	for _, name := range MaterialTables() {
		if name == "classics_paragraphs" {
			t.Error("classics_paragraphs 不应出现在物料表清单中")
		}
	}
}

// TestImportClassics_字集不再落字典属性 docs/34 阶段 E-A：拼音/释义/五行/性别
// 是 hanzi_data 的列，18 组字集平均把每个汉字抄 3.3 遍（meaning 单列曾占全库 11.5%）。
func TestImportClassics_字集不再落字典属性(t *testing.T) {
	dir := writeMinimalMaterials(t)
	writeMinimalClassics(t, dir)
	s := newTestStore(t)
	importMinimal(t, s, dir, ImportOptions{})

	cols, err := tableColumns(s.writeDB, "classics_extracts")
	if err != nil {
		t.Fatalf("读取 classics_extracts 列失败: %v", err)
	}
	for _, legacy := range classicExtractsLegacyColumns {
		if cols[legacy] {
			t.Errorf("classics_extracts 不应再有冗余列 %s", legacy)
		}
	}
	for _, want := range []string{"material", "seq", "char", "work", "chapter", "sentence"} {
		if !cols[want] {
			t.Errorf("classics_extracts 缺少列 %s", want)
		}
	}
}

// TestLoadClassicsFromDB_字典属性由字库补全 去冗余列后，字集属性必须在装载期
// 从 hanzi_data 按 char 补回，且与 JSON 路径的 extractFromEntry 口径一致。
func TestLoadClassicsFromDB_字典属性由字库补全(t *testing.T) {
	dir := writeMinimalMaterials(t)
	writeMinimalClassics(t, dir)
	s := newTestStore(t)
	importMinimal(t, s, dir, ImportOptions{})

	// 前置条件：字库须先装载，否则 LoadClassicsFromDB 拒绝执行（静默补空值
	// 会让释义评分整体失效却无任何报错）。
	if !s.loadHanziFromDB() {
		t.Fatal("从库装载字库失败")
	}
	if err := s.LoadClassicsFromDB(); err != nil {
		t.Fatalf("从库装载经典语料失败: %v", err)
	}

	var pinyin, meaning, wuxing string
	err := s.db.QueryRow(`SELECT pinyin, meaning, wuxing FROM hanzi_data WHERE char = '水'`).
		Scan(&pinyin, &meaning, &wuxing)
	if err != nil {
		t.Fatalf("查询字库失败: %v", err)
	}

	var gotPinyin, gotMeaning, gotWuxing string
	found := false
	for _, pc := range classics.GetShijingPoetryChars() {
		if pc.Char != "水" {
			continue
		}
		found = true
		gotPinyin, gotMeaning, gotWuxing = pc.Pinyin, pc.Meaning, pc.Wuxing
	}
	if !found {
		t.Fatal("诗经字集中应含「水」（蒹葭「在水一方」）")
	}
	if gotPinyin != pinyin || gotMeaning != meaning || gotWuxing != wuxing {
		t.Errorf("字集属性应与 hanzi_data 一致：\n 字集 pinyin=%q meaning(%d rune) wuxing=%q\n 字库 pinyin=%q meaning(%d rune) wuxing=%q",
			gotPinyin, len([]rune(gotMeaning)), gotWuxing,
			pinyin, len([]rune(meaning)), wuxing)
	}
}

func TestImportClassics_诗词条目内容可还原(t *testing.T) {
	dir := writeMinimalMaterials(t)
	writeMinimalClassics(t, dir)
	s := newTestStore(t)

	importMinimal(t, s, dir, ImportOptions{})

	var content string
	if err := s.db.QueryRow(`SELECT content FROM classics_poem_entries
		WHERE title = '关雎'`).Scan(&content); err != nil {
		t.Fatalf("查询诗词条目失败: %v", err)
	}
	var got []string
	if err := json.Unmarshal([]byte(content), &got); err != nil {
		t.Fatalf("诗词内容不是合法 JSON 数组: %v", err)
	}
	if len(got) != 2 || got[0] != "关关雎鸠" {
		t.Errorf("诗词内容还原不正确: %v", got)
	}
}

// TestLoadClassicsFromDB_零JSON还原 装载后无需读任何 JSON 即可还原经典状态
func TestLoadClassicsFromDB_零JSON还原(t *testing.T) {
	dir := writeMinimalMaterials(t)
	writeMinimalClassics(t, dir)
	s := newTestStore(t)

	importMinimal(t, s, dir, ImportOptions{})

	// 先用空数据污染内存态，再从库还原
	if err := classics.LoadExtractSets(nil); err != nil {
		t.Fatalf("清空字集失败: %v", err)
	}
	if len(classics.GetShijingPoetryChars()) != 0 {
		t.Fatal("预置的诗经字集应为空")
	}

	if err := s.LoadClassicsFromDB(); err != nil {
		t.Fatalf("从库装载经典语料失败: %v", err)
	}

	chars := classics.GetShijingPoetryChars()
	if len(chars) == 0 {
		t.Fatal("从库装载后诗经字集不应为空")
	}
	// 码点升序必须保持（下游候选池注入顺序依赖）
	for i := 1; i < len(chars); i++ {
		if chars[i-1].Char > chars[i].Char {
			t.Fatalf("字集顺序未按码点升序: %q 在 %q 之后", chars[i-1].Char, chars[i].Char)
		}
	}

	idx := classics.GetGlobalPoemIndex()
	if idx == nil {
		t.Fatal("从库装载后应重建全局诗词索引")
	}
	if len(idx.FindBySource("诗经")) == 0 {
		t.Error("诗词索引按来源检索不到诗经条目")
	}
}

// TestLoadClassicsFromDB_空库报错 空库（未执行 materials import）必须 fail-fast
func TestLoadClassicsFromDB_空库报错(t *testing.T) {
	s := newTestStore(t)
	if err := s.LoadClassicsFromDB(); err == nil {
		t.Fatal("空库装载经典语料应报错以引导 import")
	}
}

// TestImportClassics_自增序列归零保证幂等 同输入重复导入，自增 id 不应递增
func TestImportClassics_自增序列归零保证幂等(t *testing.T) {
	dir := writeMinimalMaterials(t)
	writeMinimalClassics(t, dir)
	s := newTestStore(t)

	importMinimal(t, s, dir, ImportOptions{})
	first := snapshotMaterialTables(t, s)

	importMinimal(t, s, dir, ImportOptions{Lenient: true}) // 绕过指纹短路强制重灌
	second := snapshotMaterialTables(t, s)

	if first != second {
		t.Error("重复导入经典语料结果应完全一致（自增序列未归零？）")
	}
}

// hanziSnapshot 字库内存态的**逐字段逐字**快照
//
// ⚠️ Meaning 必须纳入：此前等价性守护只覆盖了 pinyin/笔画/五行等 8 个维度，
// 漏掉 Meaning 后，JSON 路径装载未截断原文（262~276 rune）而 hanzi_data 是
// 200 rune 截断版这一分歧被整体漏过（docs/34 阶段 E-C）。
//
// 切片字段统一格式化为 "" 或 "a|b"：JSON 路径缺字段时是 nil 切片，
// 库路径经 unmarshalTags 得到空切片，二者 json.Marshal 结果为 null 与 []，
// 字节不同但语义相同，按值比较而非按序列化字节比较。
func hanziSnapshot() map[string]map[string]string {
	pick := map[string]func(hanzi.Hanzi) any{
		"pinyin":          func(h hanzi.Hanzi) any { return h.Pinyin },
		"strokes":         func(h hanzi.Hanzi) any { return h.Strokes },
		"kangxi_strokes":  func(h hanzi.Hanzi) any { return h.KangxiStrokes },
		"radical":         func(h hanzi.Hanzi) any { return h.Radical },
		"wuxing":          func(h hanzi.Hanzi) any { return h.Wuxing },
		"level":           func(h hanzi.Hanzi) any { return h.Level },
		"name_freq_tier":  func(h hanzi.Hanzi) any { return h.NameFreqTier },
		"gender":          func(h hanzi.Hanzi) any { return h.Gender },
		"meaning":         func(h hanzi.Hanzi) any { return h.Meaning },
		"usage_level":     func(h hanzi.Hanzi) any { return h.UsageLevel },
		"is_polyphonic":   func(h hanzi.Hanzi) any { return h.IsPolyphonic },
		"is_rare":         func(h hanzi.Hanzi) any { return h.IsRare },
		"is_negative":     func(h hanzi.Hanzi) any { return h.IsNegative },
		"positive_score":  func(h hanzi.Hanzi) any { return h.PositiveScore },
		"name_penalty":    func(h hanzi.Hanzi) any { return h.NamePenalty },
		"gender_tags":     func(h hanzi.Hanzi) any { return h.GenderTags },
		"style_tags":      func(h hanzi.Hanzi) any { return h.StyleTags },
		"naming_cats":     func(h hanzi.Hanzi) any { return h.NamingCategories },
		"pair_blacklist":  func(h hanzi.Hanzi) any { return h.PairBlacklist },
		"phonetic_score":  func(h hanzi.Hanzi) any { return h.PhoneticScore },
		"modern_score":    func(h hanzi.Hanzi) any { return h.ModernScore },
		"classical_score": func(h hanzi.Hanzi) any { return h.ClassicalScore },
		"curation_level":  func(h hanzi.Hanzi) any { return h.CurationLevel },
	}

	out := make(map[string]map[string]string, len(pick))
	for field, get := range pick {
		col := make(map[string]string, len(hanzi.HanziData))
		for ch, h := range hanzi.HanziData {
			col[ch] = fmt.Sprint(get(h))
		}
		out[field] = col
	}
	return out
}

// assertHanziEquivalent 逐字段比对两条路径的字库内存态，并给出具体差异字
func assertHanziEquivalent(t *testing.T, want, got map[string]map[string]string) {
	t.Helper()
	for _, field := range sortedFields(want) {
		wf, gf := want[field], got[field]
		var diffs []string
		for ch, wv := range wf {
			if gv, ok := gf[ch]; !ok || gv != wv {
				if len(diffs) < 5 {
					diffs = append(diffs, fmt.Sprintf("%s: JSON=%q DB=%q", ch, wv, gv))
				}
			}
		}
		if len(diffs) > 0 {
			t.Errorf("字库字段 %s 不等价（最多列 5 例）：%s", field, strings.Join(diffs, "; "))
		}
	}
}

// sortedFields 快照 map 的有序键，保证断言输出稳定
func sortedFields(m map[string]map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// TestImportClassics_真实数据下库路径与JSON路径等价 docs/31 ADR-003
// 「SQL 与内存结果等价」在 classics 物料上的守护：
// 物化（导入期提取 → 落库）与还原（运行期读回）必须逐字段等价，
// 否则「运行期改读 SQLite」会静默改变诗词取名结果。
//
// 用真实 data 目录跑（导入约 8s，含 6.7MB shici.json），故置于 -short 之后。
func TestImportClassics_真实数据下库路径与JSON路径等价(t *testing.T) {
	if testing.Short() {
		t.Skip("短跑模式跳过真实数据对拍")
	}
	_, thisFile, _, _ := runtime.Caller(0)
	realData := testutil.DataDir(thisFile)
	if realData == "" {
		t.Skip("未找到物料数据目录（向上探测 namer.json 失败），跳过对拍")
	}

	s := newTestStore(t)

	// 1. JSON 路径基线：用生产同款入口 data.Init（它按 康熙→namer→易经→语料
	//    →生肖→门禁→… 顺序装载）。直接调 hanzi.LoadNamerFromJSON 会漏掉康熙
	//    笔画合并（loader 在 namer 之后才 merge），基线就会带上 kangxi=0 的偏差。
	if err := data.Init(realData); err != nil {
		t.Fatalf("JSON 路径装载失败: %v", err)
	}
	wantHanzi := hanziSnapshot()
	wantSets := classics.ExportExtractSets()
	wantPoems := classics.ExportPoemEntries()

	// 2. 导入到库
	if _, err := s.Import(ImportOptions{DataDir: realData}); err != nil {
		t.Fatalf("真实数据导入失败: %v", err)
	}

	// 3. 清空内存态后从库还原
	if err := classics.LoadExtractSets(nil); err != nil {
		t.Fatalf("清空字集失败: %v", err)
	}
	classics.LoadPoemEntries(nil)
	if err := s.LoadMaterialsFromDB(); err == nil {
		t.Fatal("前置条件：此时不应能完整装载，LoadClassicsFromDB 尚未调用")
	}
	// 库路径的字库：读 hanzi_data
	if !s.loadHanziFromDB() {
		t.Fatal("从库装载字库失败")
	}
	if err := s.LoadClassicsFromDB(); err != nil {
		t.Fatalf("从库装载失败: %v", err)
	}

	// 4. 字库等价（逐字段逐字；E-C 的回归守护）
	assertHanziEquivalent(t, wantHanzi, hanziSnapshot())

	// 5. 逐字集、逐诗词条目比对
	gotSets := classics.ExportExtractSets()
	if len(gotSets) != len(wantSets) {
		t.Fatalf("字集物料数不一致: JSON %d vs DB %d", len(wantSets), len(gotSets))
	}
	total := 0
	for i := range wantSets {
		if wantSets[i].Material != gotSets[i].Material {
			t.Fatalf("第 %d 个字集物料不一致: %s vs %s", i, wantSets[i].Material, gotSets[i].Material)
		}
		if !reflect.DeepEqual(wantSets[i].Chars, gotSets[i].Chars) {
			t.Errorf("字集 %s 不等价: JSON %d 条 vs DB %d 条", wantSets[i].Material,
				len(wantSets[i].Chars), len(gotSets[i].Chars))
			for k := range wantSets[i].Chars {
				if k >= len(gotSets[i].Chars) {
					break
				}
				if !reflect.DeepEqual(wantSets[i].Chars[k], gotSets[i].Chars[k]) {
					t.Errorf("  首个差异 @%d:\n    JSON=%+v\n    DB  =%+v",
						k, wantSets[i].Chars[k], gotSets[i].Chars[k])
					break
				}
			}
		}
		total += len(wantSets[i].Chars)
	}
	t.Logf("对拍字集条目 %d 条", total)

	gotPoems := classics.ExportPoemEntries()
	if len(gotPoems) != len(wantPoems) {
		t.Fatalf("诗词条目数不一致: JSON %d vs DB %d", len(wantPoems), len(gotPoems))
	}
	for i := range wantPoems {
		if !reflect.DeepEqual(wantPoems[i], gotPoems[i]) {
			t.Fatalf("诗词第 %d 条不等价: JSON=%+v DB=%+v", i, wantPoems[i], gotPoems[i])
		}
	}
	t.Logf("对拍诗词条目 %d 条", len(wantPoems))
}
