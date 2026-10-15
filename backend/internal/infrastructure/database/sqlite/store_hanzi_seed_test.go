package sqlite

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"name/internal/domain/hanzi"
)

// 本文件守护 docs/31 ADR-011：hanzi_data 的持久化写入语义必须是
// 「同输入多次执行结果等价 + 物料字段修订可传导 + 删字可删除」。
//
// 修订前的实现是 `if COUNT(*) > 0 { return }` + `INSERT OR IGNORE`：
// 表非空即整体跳过、冲突行不更新，于是 JSON 侧改了字段或删了字，
// SQLite 侧永远停留在首次导入的旧值，SQL 与内存双读长期静默不一致。
//
// 注意：本文件不声明 t.Parallel()，测试内直接替换包级 hanzi.HanziData。
// go test 按包隔离进程，且本包现有测试同样依赖该全局态。

// newTestStore 建一个临时库 Store（不触发真实物料装载）
func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s, err := NewStore(filepath.Join(dir, "test.db"), dir)
	if err != nil {
		t.Fatalf("创建测试 Store 失败: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// setHanziData 整体替换内存字库（测试用），测试结束后恢复原状
func setHanziData(t *testing.T, data map[string]hanzi.Hanzi) {
	t.Helper()
	old := hanzi.HanziData
	hanzi.HanziData = data
	t.Cleanup(func() { hanzi.HanziData = old })
}

func rowCount(t *testing.T, s *Store) int {
	t.Helper()
	var n int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM hanzi_data").Scan(&n); err != nil {
		t.Fatalf("统计 hanzi_data 行数失败: %v", err)
	}
	return n
}

func positiveScore(t *testing.T, s *Store, char string) int {
	t.Helper()
	var v int
	if err := s.db.QueryRow("SELECT positive_score FROM hanzi_data WHERE char = ?", char).Scan(&v); err != nil {
		t.Fatalf("读取 %s 的 positive_score 失败: %v", char, err)
	}
	return v
}

// snapshotHanziData 读回整表并拼成可比较字符串，用于幂等断言
func snapshotHanziData(t *testing.T, s *Store) string {
	t.Helper()
	rows, err := s.db.Query(`SELECT char, pinyin, strokes, radical, meaning, wuxing, gender,
		tone, gender_tags, style_tags, usage_level,
		positive_score, phonetic_score, modern_score, classical_score,
		is_polyphonic, is_rare, is_negative,
		pair_blacklist, curation_level, name_penalty, naming_categories
		FROM hanzi_data ORDER BY char`)
	if err != nil {
		t.Fatalf("查询 hanzi_data 失败: %v", err)
	}
	defer func() { _ = rows.Close() }()

	cols, err := rows.Columns()
	if err != nil {
		t.Fatalf("读取列名失败: %v", err)
	}
	var b strings.Builder
	for rows.Next() {
		cells := make([]any, len(cols))
		for i := range cells {
			cells[i] = new(sql.NullString)
		}
		if err := rows.Scan(cells...); err != nil {
			t.Fatalf("扫描行失败: %v", err)
		}
		for i, c := range cells {
			fmt.Fprintf(&b, "%s=%s|", cols[i], c.(*sql.NullString).String)
		}
		b.WriteString("\n")
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("遍历行失败: %v", err)
	}
	return b.String()
}

// TestRebuildHanziData_字段修订可传导 是本次修订的核心回归：
// 修订前该用例必然失败（表非空直接 return，SQL 侧仍是旧值）
func TestRebuildHanziData_字段修订可传导(t *testing.T) {
	s := newTestStore(t)

	setHanziData(t, map[string]hanzi.Hanzi{
		"永": {Char: "永", Pinyin: "yong3", Strokes: 5, PositiveScore: 10},
	})
	s.seedHanziData()
	if got := positiveScore(t, s, "永"); got != 10 {
		t.Fatalf("首次写入期望 positive_score=10，实得 %d", got)
	}

	// 模拟 JSON 物料修订：同一个字字段值改变
	setHanziData(t, map[string]hanzi.Hanzi{
		"永": {Char: "永", Pinyin: "yong3", Strokes: 5, PositiveScore: 88},
	})
	s.ReloadHanziData()

	if got := positiveScore(t, s, "永"); got != 88 {
		t.Fatalf("物料修订应传导到 SQLite：期望 positive_score=88，实得 %d", got)
	}
}

// TestRebuildHanziData_删字可删除 守护 ADR-011 的删除语义
func TestRebuildHanziData_删字可删除(t *testing.T) {
	s := newTestStore(t)

	setHanziData(t, map[string]hanzi.Hanzi{
		"永": {Char: "永", PositiveScore: 10},
		"水": {Char: "水", PositiveScore: 20},
	})
	s.seedHanziData()
	if got := rowCount(t, s); got != 2 {
		t.Fatalf("首次写入期望 2 行，实得 %d", got)
	}

	// 物料中移除「水」
	setHanziData(t, map[string]hanzi.Hanzi{
		"永": {Char: "永", PositiveScore: 10},
	})
	s.ReloadHanziData()

	if got := rowCount(t, s); got != 1 {
		t.Fatalf("删字应同步删除：期望 1 行，实得 %d", got)
	}
	var n int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM hanzi_data WHERE char = '水'").Scan(&n); err != nil {
		t.Fatalf("查询已删字失败: %v", err)
	}
	if n != 0 {
		t.Fatal("已从物料移除的「水」仍残留在 hanzi_data 表中")
	}
}

// TestRebuildHanziData_幂等 同输入多次执行结果必须等价
func TestRebuildHanziData_幂等(t *testing.T) {
	s := newTestStore(t)

	setHanziData(t, map[string]hanzi.Hanzi{
		"永": {Char: "永", Pinyin: "yong3", Strokes: 5, Radical: "水", Meaning: "水流长",
			Wuxing: "水", Gender: "male", Tone: 3, UsageLevel: 4, PositiveScore: 77,
			IsPolyphonic: true, IsRare: false, IsNegative: false, CurationLevel: 3,
			NamePenalty: 2, StyleTags: []string{"大气"}, NamingCategories: []string{"品德"}},
	})
	s.seedHanziData()
	first := snapshotHanziData(t, s)

	s.ReloadHanziData()
	second := snapshotHanziData(t, s)
	s.ReloadHanziData()
	third := snapshotHanziData(t, s)

	if first != second || second != third {
		t.Fatalf("重复重建结果不一致：\n首次=%s\n二次=%s\n三次=%s", first, second, third)
	}
}

// TestRebuildHanziData_重复seed不重复计数 seed 与 reload 走同一条路径，不得累加行
func TestRebuildHanziData_重复seed不重复计数(t *testing.T) {
	s := newTestStore(t)
	setHanziData(t, map[string]hanzi.Hanzi{
		"永": {Char: "永"},
		"水": {Char: "水"},
	})

	s.seedHanziData()
	s.seedHanziData()
	s.seedHanziData()

	if got := rowCount(t, s); got != 2 {
		t.Fatalf("重复 seed 不应改变行数：期望 2，实得 %d", got)
	}
}

// TestRebuildHanziData_空字库不清表 防止「数据尚未装载就把既有表清空」
func TestRebuildHanziData_空字库不清表(t *testing.T) {
	s := newTestStore(t)

	setHanziData(t, map[string]hanzi.Hanzi{"永": {Char: "永"}})
	s.seedHanziData()
	if got := rowCount(t, s); got != 1 {
		t.Fatalf("前置写入失败，期望 1 行，实得 %d", got)
	}

	setHanziData(t, map[string]hanzi.Hanzi{})
	s.seedHanziData()
	s.ReloadHanziData()

	if got := rowCount(t, s); got != 1 {
		t.Fatalf("字库为空时不得清空既有表：期望仍有 1 行，实得 %d", got)
	}
}

// TestRebuildHanziData_全字段投影 每个 Hanzi 字段都必须落到对应列，
// 防止新增字段只改结构体却忘了写库（docs/31 ADR-012）
func TestRebuildHanziData_全字段投影(t *testing.T) {
	s := newTestStore(t)
	setHanziData(t, map[string]hanzi.Hanzi{
		"行": {
			Char: "行", Pinyin: "xing2", Strokes: 6, KangxiStrokes: 16,
			Radical: "彳", Meaning: "道路", Wuxing: "水", Gender: "neutral",
			Level: 2, Tone: 2, NameFreqTier: 4,
			GenderTags: []string{"male"}, StyleTags: []string{"古典"},
			UsageLevel: 5, PositiveScore: 90, PhoneticScore: 80, ModernScore: 70,
			ClassicalScore: 95, IsPolyphonic: true, IsRare: true, IsNegative: true,
			PairBlacklist: []string{"组合甲"}, CurationLevel: 4, NamePenalty: 6,
			NamingCategories: []string{"品德", "志向"},
		},
	})
	s.seedHanziData()

	rows, err := s.db.Query(`SELECT char, pinyin, strokes, kangxi_strokes, radical, meaning,
		wuxing, gender, level, tone, name_freq_tier, gender_tags, style_tags, usage_level,
		positive_score, phonetic_score, modern_score, classical_score,
		is_polyphonic, is_rare, is_negative,
		pair_blacklist, curation_level, name_penalty, naming_categories
		FROM hanzi_data WHERE char = '行'`)
	if err != nil {
		t.Fatalf("查询投影结果失败: %v", err)
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		t.Fatal("hanzi_data 中查不到「行」")
	}
	var (
		char, pinyin, radical, meaning, wuxing, gender   string
		genderTags, styleTags, pairBlacklist, namingCats string
		strokes, kangxi, level, tone, freqTier, usage    int
		pos, pho, mod, cla, curation, penalty            int
		polyphonic, rare, negative                       int
	)
	if err := rows.Scan(&char, &pinyin, &strokes, &kangxi, &radical, &meaning,
		&wuxing, &gender, &level, &tone, &freqTier, &genderTags, &styleTags, &usage,
		&pos, &pho, &mod, &cla, &polyphonic, &rare, &negative,
		&pairBlacklist, &curation, &penalty, &namingCats); err != nil {
		t.Fatalf("扫描投影结果失败（列数与 SELECT 不匹配）: %v", err)
	}

	checks := []struct {
		name string
		got  any
		want any
	}{
		{"char", char, "行"},
		{"pinyin", pinyin, "xing2"}, {"strokes", strokes, 6},
		{"kangxi_strokes", kangxi, 16}, {"radical", radical, "彳"},
		{"meaning", meaning, "道路"}, {"wuxing", wuxing, "水"},
		{"gender", gender, "neutral"}, {"level", level, 2}, {"tone", tone, 2},
		{"name_freq_tier", freqTier, 4}, {"usage_level", usage, 5},
		{"positive_score", pos, 90},
		{"phonetic_score", pho, 80}, {"modern_score", mod, 70},
		{"classical_score", cla, 95}, {"is_polyphonic", polyphonic, 1},
		{"is_rare", rare, 1}, {"is_negative", negative, 1},
		{"curation_level", curation, 4}, {"name_penalty", penalty, 6},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("列 %s 期望 %v，实得 %v", c.name, c.want, c.got)
		}
	}

	for _, c := range []struct{ name, got, want string }{
		{"gender_tags", genderTags, `["male"]`},
		{"style_tags", styleTags, `["古典"]`},
		{"pair_blacklist", pairBlacklist, `["组合甲"]`},
		{"naming_categories", namingCats, `["品德","志向"]`},
	} {
		if c.got != c.want {
			t.Errorf("列 %s 期望 %s，实得 %s", c.name, c.want, c.got)
		}
	}
}

// TestRebuildHanziData_多音字真实落库 回归 namer.json 中 1279 条 isPolyphonic
// 曾因 NamerChar 未声明该字段而被静默丢弃的缺陷
func TestRebuildHanziData_多音字真实落库(t *testing.T) {
	s := newTestStore(t)
	setHanziData(t, map[string]hanzi.Hanzi{
		"和": {Char: "和", Pinyin: "he2", IsPolyphonic: true},
		"安": {Char: "安", Pinyin: "an1", IsPolyphonic: false},
	})
	s.seedHanziData()

	for _, c := range []struct {
		char string
		want int
	}{{"和", 1}, {"安", 0}} {
		var got int
		if err := s.db.QueryRow("SELECT is_polyphonic FROM hanzi_data WHERE char = ?", c.char).Scan(&got); err != nil {
			t.Fatalf("读取 %s 的 is_polyphonic 失败: %v", c.char, err)
		}
		if got != c.want {
			t.Errorf("%s 的 is_polyphonic 期望 %d，实得 %d", c.char, c.want, got)
		}
	}
}
