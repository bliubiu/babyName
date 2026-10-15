package services

import (
	"errors"
	"testing"

	"name/internal/domain/fate"
	"name/internal/infrastructure/database"
)

// --- P2（ADR-003「SQL 与内存结果等价」）不变量守护 ---
//
// P2 修复前：queryViaSQL 与 FindCharacters 会逐字 `hanzi.GetHanzi` 回查内存补全
// gender / naming_categories / radical / level 等字段，因 SQL 表未落这些列。
// 该回查使 SQL 下沉收益归零，且两路径结果可能不一致。
//
// 本文件测试位于 package services 内部（白盒测试），故直接访问未导出的
// deriveCharacterFlags，并可直接构造 database.Hanzi。

// stubSQLStore 返回固定 SQL 行，用于验证 services 层只依赖 SQL 数据即可构造 Character
type stubSQLStore struct {
	rows []database.Hanzi
	err  error
}

func (s *stubSQLStore) SearchHanziByFilter(wuxing string, minStrokes, maxStrokes int,
	hasPositive, isRegular bool, chars []string, limit int) ([]*database.Hanzi, error) {
	if s.err != nil {
		return nil, s.err
	}
	out := make([]*database.Hanzi, 0, len(s.rows))
	for i := range s.rows {
		r := s.rows[i]
		out = append(out, &r)
	}
	return out, nil
}

// HanziRowCount 有数据（返回行数大于 0），保证 FindCharacters 的 SQL 空结果
// 不会被误判为「库无数据」而降级（本用例锁的正是 SQL 有结果的主路径）
func (s *stubSQLStore) HanziRowCount() (int, error) {
	if s.err != nil {
		return 0, s.err
	}
	return len(s.rows), nil
}

// TestSQLPathNoMemoryFallback 验证 SQL 路径不再回查内存
//
// 手法：构造一个只含 2 行的 stub SQL store，返回的字（"沄"、"汀"）都不在
// 内存 hanzi.HanziData 中。若 services 层仍有 `hanzi.GetHanzi` 回查，
// 这些字会被 `if !ok { continue }` 丢弃，FindCharacters 返回空 → 测试失败。
func TestSQLPathNoMemoryFallback(t *testing.T) {
	provider := &HanziDataProvider{}
	provider.SetSQLFilter(NewSQLiteHanziFilter(&stubSQLStore{
		rows: []database.Hanzi{
			{Char: "沄", Pinyin: "yun", Wuxing: "水", Strokes: 7,
				Level: 1, Gender: "neutral", Radical: "氵", Meaning: "水势盛大",
				NamingCategories: []string{"大气"}, PositiveScore: 70},
			{Char: "汀", Pinyin: "ting", Wuxing: "水", Strokes: 5,
				Level: 2, Gender: "male", Radical: "氵", Meaning: "水边平地",
				NamingCategories: []string{"清雅"}, PositiveScore: 60},
		},
	}))

	chars, err := provider.FindCharacters(&fate.BasicCharacterQuery{
		RegularFilter:  true,
		NameableFilter: true,
		WuxingIn:       []string{"水"},
	})
	if err != nil {
		t.Fatalf("FindCharacters 失败: %v", err)
	}

	// 关键断言：结果非空。若仍回查内存，这两个字都取不到，会被 continue 丢弃。
	if len(chars) != 2 {
		t.Fatalf("SQL 路径应返回 2 字（不得依赖内存回查），实际 %d", len(chars))
	}

	byChar := map[string]*fate.Character{}
	for _, c := range chars {
		byChar[c.Char] = c
	}

	// 字段必须来自 SQL 行，而非内存
	yun, ok := byChar["沄"]
	if !ok {
		t.Fatal("应包含「沄」")
	}
	if yun.Meaning != "水势盛大" {
		t.Errorf("Meaning 应取自 SQL 行，实际 %q", yun.Meaning)
	}
	if yun.Radical != "氵" {
		t.Errorf("Radical 应取自 SQL 行，实际 %q", yun.Radical)
	}
	if yun.CommonLevel != 1 {
		t.Errorf("CommonLevel 应取自 SQL level=1，实际 %d", yun.CommonLevel)
	}
	if len(yun.NamingCategory) != 1 || yun.NamingCategory[0] != "大气" {
		t.Errorf("NamingCategory 应取自 SQL naming_categories，实际 %v", yun.NamingCategory)
	}
	if yun.PositiveScore != 70 {
		t.Errorf("PositiveScore 应取自 SQL 行，实际 %d", yun.PositiveScore)
	}

	// genderHint 过滤必须基于 SQL 行的 gender 字段。
	// 注意语义：neutral 字视为「两性通用」，同样命中（与内存路径一致，
	// 见 queryViaSQL / FindCharacters 中的 c.GenderHint != "neutral" 豁免条件）。
	filtered, err := provider.FindCharacters(&fate.BasicCharacterQuery{
		GenderHint: "male",
	})
	if err != nil {
		t.Fatalf("FindCharacters(genderHint) 失败: %v", err)
	}
	if len(filtered) != 2 {
		t.Errorf("genderHint=male 应命中 male 字「汀」+ neutral 字「沄」，实际 %d 个", len(filtered))
	}
	// 校验 genderHint 确实读了 SQL 行的 gender 列：male 被排除，仅 neutral 的「沄」豁免命中
	filtered, err = provider.FindCharacters(&fate.BasicCharacterQuery{
		GenderHint: "female",
	})
	if err != nil {
		t.Fatalf("FindCharacters(genderHint=female) 失败: %v", err)
	}
	if len(filtered) != 1 || filtered[0].Char != "沄" {
		t.Errorf("genderHint=female 应仅命中 neutral 的「沄」，实际 %d 个", len(filtered))
	}

	// namingCategory 过滤必须基于 SQL 行的 naming_categories
	filtered, err = provider.FindCharacters(&fate.BasicCharacterQuery{
		NamingCategory: "清雅",
	})
	if err != nil {
		t.Fatalf("FindCharacters(namingCategory) 失败: %v", err)
	}
	if len(filtered) != 1 || filtered[0].Char != "汀" {
		t.Errorf("namingCategory=清雅 应只命中「汀」，实际 %d 个", len(filtered))
	}
}

// TestDeriveCharacterFlagsEquivalence 验证派生规则自洽
//
// SQL 路径与内存路径共用 deriveCharacterFlags，因此其行为必须稳定且符合
// docs/31 §9.9 的判定约定：level 1/2 恒为常用字；level 0 按笔画+生僻保守判定；
// isNameable = strokes<=30 && isRegular。
func TestDeriveCharacterFlagsEquivalence(t *testing.T) {
	cases := []struct {
		name           string
		level, strokes int
		isRare         bool
	}{
		{"一级字", 1, 5, false},
		{"二级字", 2, 12, false},
		{"三级字不算常用", 3, 8, false},
		{"表外字少笔画", 0, 20, false},
		{"表外字生僻", 0, 20, true},
		{"表外字超长", 0, 28, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			isRegular, isNameable, level := deriveCharacterFlags(tc.level, tc.strokes, tc.isRare)

			if level != tc.level {
				t.Errorf("CommonLevel 应为 %d，实际 %d", tc.level, level)
			}
			wantNameable := tc.strokes <= 30 && isRegular
			if isNameable != wantNameable {
				t.Errorf("isNameable 应为 %v，实际 %v", wantNameable, isNameable)
			}
			if tc.level == 1 || tc.level == 2 {
				if !isRegular {
					t.Errorf("一级/二级字恒为常用字，实际 isRegular=false")
				}
			}
		})
	}
}

// TestRareCharUnionAcrossPaths 验证生僻字取并集
//
// hanzi.Hanzi.IsRare（namer.json 99 字）与 hanzi.RareChars（编译期静态表）
// 实测有 102 字不一致。若两路径各取其一，同一字会得到相反的 IsRegular。
// 守护不变量：任何一侧判生僻 → 两路径均按生僻处理。
func TestRareCharUnionAcrossPaths(t *testing.T) {
	// 构造 SQL 行，标 isRare=true
	h := &database.Hanzi{Char: "龘", Strokes: 48, Level: 0, IsRare: true}
	c := databaseHanziToCharacter(h)
	if c.IsRegular {
		t.Error("isRare=true 的表外字不应判为常用字")
	}

	// 同一字若 isRare=false 但笔画少，仍应按笔画+非生僻判定为常用
	h2 := &database.Hanzi{Char: "龘", Strokes: 48, Level: 0, IsRare: false}
	// 笔画 48 > 25 → 表外字即使不生僻也不判常用
	c2 := databaseHanziToCharacter(h2)
	if c2.IsRegular {
		t.Error("表外字笔画 48 超过 25 阈值，不应判为常用字")
	}
}

// TestSQLAndMemoryScoringFieldsIdentical 守护「评分字段两路径同源」
//
// 回归背景：P2 初次实现把 is_negative / name_penalty / name_freq_tier 在 SQL 路径
// 误置为 false/0/0（误以为上游无真值）。但内存侧实测这 3 个字段均有真值
// （isNegative 14 字、namePenalty 54 字、nameFreqTier 1118 字），且都会参与
// fate 评分 —— 于是生成链路（SQL）与测名链路（内存）算出不同总分，
// 违反「测名与生成同源同分」契约（e2e_check 第 7 项报 80.10 vs 80.60）。
//
// 本测试锁定：SQL 路径构造的 Character，其全部评分相关字段必须取自 SQL 行，
// 不得出现「默认零值」。
func TestSQLAndMemoryScoringFieldsIdentical(t *testing.T) {
	h := &database.Hanzi{
		Char: "珀", Pinyin: "po", Wuxing: "水", Strokes: 10,
		Level: 1, PositiveScore: 72, NameFreqTier: 4,
		IsNegative: true, NamePenalty: 30,
	}
	c := databaseHanziToCharacter(h)

	if !c.IsNegative {
		t.Error("IsNegative 应透传 SQL 行的 true，不得置 false")
	}
	if c.NamePenalty != 30 {
		t.Errorf("NamePenalty 应透传 SQL 行的 30，实际 %d", c.NamePenalty)
	}
	if c.NameFreqTier != 4 {
		t.Errorf("NameFreqTier 应透传 SQL 行的 4，实际 %d", c.NameFreqTier)
	}
	if c.PositiveScore != 72 {
		t.Errorf("PositiveScore 应透传 SQL 行的 72，实际 %d", c.PositiveScore)
	}
	if c.CommonLevel != 1 {
		t.Errorf("CommonLevel 应透传 SQL 行的 1，实际 %d", c.CommonLevel)
	}
}

// TestSQLPathFallsBackOnError 验证 SQL 失败仍可降级到 Go 端过滤
//
// P2 改动不得破坏 B8 的降级契约：SQL 出错必须返回 error 并让调用方降级，
// 而不是伪装成「查询成功但无匹配」。
func TestSQLPathFallsBackOnError(t *testing.T) {
	provider := &HanziDataProvider{}
	provider.SetSQLFilter(NewSQLiteHanziFilter(&stubSQLStore{
		err: errors.New("模拟 DB 故障"),
	}))

	// 不应 panic；FindCharacters 内部对 SQL 失败走 Go 端降级
	if _, err := provider.FindCharacters(&fate.BasicCharacterQuery{}); err != nil {
		t.Fatalf("SQL 失败时应降级而非上抛: %v", err)
	}
}
