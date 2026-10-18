package sqlite

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"name/internal/domain/classics"
	"name/internal/domain/hanzi"
	"name/internal/materials/schema"
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

func TestImportClassics_三表与物化表一并写入(t *testing.T) {
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

	var books, sections, paragraphs, extracts, poems int
	for _, q := range []struct {
		table string
		dst   *int
	}{
		{"classics_books", &books},
		{"classics_sections", &sections},
		{"classics_paragraphs", &paragraphs},
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
	if paragraphs == 0 {
		t.Error("classics_paragraphs 不应为空")
	}
	if extracts == 0 {
		t.Error("classics_extracts（物化字集）不应为空")
	}
	if poems == 0 {
		t.Error("classics_poem_entries（物化诗词索引）不应为空")
	}
}

func TestImportClassics_字集属性已由字库补全(t *testing.T) {
	dir := writeMinimalMaterials(t)
	writeMinimalClassics(t, dir)
	s := newTestStore(t)

	importMinimal(t, s, dir, ImportOptions{})

	// 「鸠」不在测试字库中，应落默认值；「水」在字库中，应带上拼音与五行
	var pinyin, wuxing string
	err := s.db.QueryRow(`SELECT pinyin, wuxing FROM classics_extracts
		WHERE material = ? AND char = '水'`, classics.MaterialShijing).Scan(&pinyin, &wuxing)
	if err != nil {
		t.Fatalf("查询物化字集失败: %v", err)
	}
	if pinyin != "shui3" {
		t.Errorf("字集拼音应由字库补全为 shui3，实得 %q", pinyin)
	}
	if wuxing != "水" {
		t.Errorf("字集五行应由字库补全为 水，实得 %q", wuxing)
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
	realData := filepath.Join("..", "..", "..", "..", "data")
	if _, err := os.Stat(filepath.Join(realData, "namer.json")); err != nil {
		t.Skipf("未找到真实数据目录，跳过对拍: %v", err)
	}

	s := newTestStore(t)

	// 1. JSON 路径基线
	if err := hanzi.LoadNamerFromJSON(realData); err != nil {
		t.Fatalf("装载字库失败: %v", err)
	}
	if err := classics.LoadFromJSON(realData); err != nil {
		t.Fatalf("JSON 路径装载经典语料失败: %v", err)
	}
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
	if err := s.LoadClassicsFromDB(); err != nil {
		t.Fatalf("从库装载失败: %v", err)
	}

	// 4. 逐字集、逐诗词条目比对
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
			t.Errorf("字集 %s 不等价: JSON %d 条 vs DB %d 条",
				wantSets[i].Material, len(wantSets[i].Chars), len(gotSets[i].Chars))
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