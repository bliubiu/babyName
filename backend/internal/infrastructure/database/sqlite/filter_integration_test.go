package sqlite

import (
	"path/filepath"
	"runtime"
	"testing"

	"name/internal/application/services"
	"name/internal/domain/fate"
	"name/internal/infrastructure/data"
)

// 接口守卫：*Store 必须实现 services.SQLiteCharStore（SQLite 下沉装配契约）
var _ services.SQLiteCharStore = (*Store)(nil)

// mustSetupStore 装配数据 + SQLite store（数据缺失时跳过测试）
func mustSetupStore(t *testing.T) *Store {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	backendRoot := filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..")
	dataDir := filepath.Join(backendRoot, "data")

	if err := data.Init(dataDir); err != nil {
		t.Skipf("data.Init 失败（raw 数据缺失），跳过: %v", err)
	}
	services.SyncNamingIndexFromHanzi()

	store, err := NewStore(filepath.Join(t.TempDir(), "test.db"), dataDir)
	if err != nil {
		t.Fatalf("NewStore 失败: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

// charset 提取字符集合（做结果一致性比较）
func charset(chars []*fate.Character) map[string]bool {
	set := make(map[string]bool, len(chars))
	for _, c := range chars {
		set[c.Char] = true
	}
	return set
}

// findChars 用 HanziDataProvider 走指定后端（SQL 或 Go）查候选字
func findChars(t *testing.T, store *Store, useSQL bool, q fate.CharacterQuery) []*fate.Character {
	t.Helper()
	provider := &services.HanziDataProvider{}
	if useSQL {
		provider.SetSQLFilter(services.NewSQLiteHanziFilter(store))
	}
	chars, err := provider.FindCharacters(q)
	if err != nil {
		t.Fatalf("FindCharacters 失败(useSQL=%v): %v", useSQL, err)
	}
	return chars
}

// TestSQLiteHanziFilterIntegration 验证 SQLite 下沉与纯 Go 过滤结果一致
//
// 完整链路：data.Init → NewStore seed → HanziDataProvider 装配 SQLite 过滤
// → FindCharacters 实际查 SQL。
//
// 验证：相同过滤条件下，SQL 注入后的结果集与纯 Go 端过滤完全一致
// （覆盖 wuxingIn 多值 / genderHint / regular+nameable 组合）。
func TestSQLiteHanziFilterIntegration(t *testing.T) {
	store := mustSetupStore(t)

	t.Run("条件-性别 genderHint", func(t *testing.T) {
		// 从数据中探测实际生效的 genderHint 取值（不硬编码，兼容不同数据源口径）
		probe := &fate.BasicCharacterQuery{
			RegularFilter:  true,
			NameableFilter: true,
			StrokeGTE:      5,
			StrokeLTE:      20,
		}
		all := findChars(t, store, false, probe)
		genders := map[string]bool{}
		for _, c := range all {
			if c.GenderHint != "" && c.GenderHint != "neutral" {
				genders[c.GenderHint] = true
			}
		}
		if len(genders) == 0 {
			t.Skip("数据集中无性别标记，无法验证 genderHint 过滤")
		}
		var picked string
		for g := range genders {
			picked = g
			break
		}
		q := &fate.BasicCharacterQuery{
			RegularFilter:  true,
			NameableFilter: true,
			StrokeGTE:      5,
			StrokeLTE:      20,
			GenderHint:     picked,
		}
		goSet := charset(findChars(t, store, false, q))
		if len(goSet) == 0 {
			t.Skipf("genderHint=%q 过滤后无结果，无法断言", picked)
		}
		sqlSet := charset(findChars(t, store, true, q))
		if len(sqlSet) == 0 {
			t.Fatal("SQLite 下沉路径应返回非空结果（曾因 genderHint 误按拼音比较而全灭）")
		}
		assertSameSet(t, goSet, sqlSet)
	})

	t.Run("条件-忌用五行 wuxingNotIn", func(t *testing.T) {
		q := &fate.BasicCharacterQuery{
			RegularFilter:  true,
			NameableFilter: true,
			WuxingIn:       []string{"水", "木"},
			WuxingNotIn:    []string{"木"},
		}
		goSet := charset(findChars(t, store, false, q))
		sqlSet := charset(findChars(t, store, true, q))
		if len(goSet) != len(sqlSet) {
			t.Errorf("wuxingNotIn 结果集不一致: Go=%d SQL=%d", len(goSet), len(sqlSet))
		}
		assertSameSet(t, goSet, sqlSet)
	})

	t.Run("条件-精选分类 namingCategory", func(t *testing.T) {
		probe := &fate.BasicCharacterQuery{
			RegularFilter:  true,
			NameableFilter: true,
		}
		all := findChars(t, store, false, probe)
		cats := map[string]bool{}
		for _, c := range all {
			for _, cat := range c.NamingCategory {
				cats[cat] = true
			}
		}
		if len(cats) == 0 {
			t.Skip("数据集中无命名分类标记，无法验证 namingCategory 过滤")
		}
		var picked string
		for cat := range cats {
			picked = cat
			break
		}
		q := &fate.BasicCharacterQuery{
			RegularFilter:  true,
			NameableFilter: true,
			NamingCategory: picked,
		}
		goSet := charset(findChars(t, store, false, q))
		if len(goSet) == 0 {
			t.Skipf("namingCategory=%q 过滤后无结果，无法断言", picked)
		}
		sqlSet := charset(findChars(t, store, true, q))
		assertSameSet(t, goSet, sqlSet)
	})
}

// assertSameSet 断言两个字符集合完全一致
func assertSameSet(t *testing.T, goSet, sqlSet map[string]bool) {
	t.Helper()
	if len(goSet) != len(sqlSet) {
		t.Errorf("结果集大小不一致: Go=%d SQL=%d", len(goSet), len(sqlSet))
	}
	for c := range goSet {
		if !sqlSet[c] {
			t.Errorf("SQLite 路径遗漏字符 %q（Go 端有）", c)
		}
	}
	for c := range sqlSet {
		if !goSet[c] {
			t.Errorf("SQLite 路径多出字符 %q（Go 端无）", c)
		}
	}
}

// TestSQLiteHanziFilterBasicParams 直接测试 store 的 SearchHanziByFilter
//
// 单条件、组合条件、限字列表都覆盖。验证 SQL 路径返回结果与 namer.json 一致。
func TestSQLiteHanziFilterBasicParams(t *testing.T) {
	store := mustSetupStore(t)

	t.Run("单条件-五行", func(t *testing.T) {
		rows := store.SearchHanziByFilter("金", 0, 0, false, false, nil, 0)
		if len(rows) == 0 {
			t.Fatal("金行 SQL 应返回字，实际 0")
		}
		for _, h := range rows {
			if h.Wuxing != "金" {
				t.Errorf("字=%q 应属金行，实际 %q", h.Char, h.Wuxing)
			}
		}
	})

	t.Run("组合条件-五行+笔画+常用字", func(t *testing.T) {
		rows := store.SearchHanziByFilter("水", 5, 12, false, true, nil, 100)
		for _, h := range rows {
			if h.Wuxing != "水" {
				t.Errorf("字=%q 应属水行，实际 %q", h.Char, h.Wuxing)
			}
			if h.Strokes < 5 || h.Strokes > 12 {
				t.Errorf("字=%q strokes=%d 越界", h.Char, h.Strokes)
			}
			if h.UsageLevel != 1 && h.UsageLevel != 2 {
				t.Errorf("字=%q IsRegular=true 应 level=1/2，实际 %d", h.Char, h.UsageLevel)
			}
		}
	})

	t.Run("限制字列表", func(t *testing.T) {
		rows := store.SearchHanziByFilter("", 0, 0, false, false,
			[]string{"王", "李", "张"}, 0)
		if len(rows) != 3 {
			t.Errorf("限制字列表应返回 3 条，实际 %d", len(rows))
		}
		chars := map[string]bool{}
		for _, h := range rows {
			chars[h.Char] = true
		}
		for _, want := range []string{"王", "李", "张"} {
			if !chars[want] {
				t.Errorf("应包含 %q, 实际 %v", want, chars)
			}
		}
	})

	t.Run("positive_score 过滤", func(t *testing.T) {
		rows := store.SearchHanziByFilter("", 0, 0, true, false, nil, 0)
		if len(rows) == 0 {
			t.Fatal("positive_score > 0 应有字，实际 0")
		}
		t.Logf("HasPositive 命中 %d 字", len(rows))
	})

	t.Run("空条件返回全表（Limit 默认 10000）", func(t *testing.T) {
		rows := store.SearchHanziByFilter("", 0, 0, false, false, nil, 0)
		if len(rows) == 0 {
			t.Fatal("空条件应返回全表，实际 0")
		}
		if len(rows) > 10000 {
			t.Errorf("默认 Limit 应为 10000, 实际返回 %d", len(rows))
		}
	})
}
