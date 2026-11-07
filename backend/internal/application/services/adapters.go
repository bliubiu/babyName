package services

import (
	"sync"

	"name/internal/domain/bazi"
	"name/internal/domain/fate"
	"name/internal/domain/hanzi"
	"name/internal/domain/name"
	"name/internal/domain/yijing"
	"name/internal/domain/ziwei"
	"name/internal/domain/zodiac"
	"name/internal/infrastructure/database"
)

var (
	namingSyncMu      sync.Mutex
	namingIndexSynced bool
)

// SyncNamingIndexFromHanzi 将 HanziData 中的起名分类同步到 fate 层命名索引
// 使得 GetCompatibleCharsByCategory 等函数能访问全量分类数据
// force=true 时强制重新同步（用于热更新后刷新数据）
func SyncNamingIndexFromHanzi() {
	namingSyncMu.Lock()
	defer namingSyncMu.Unlock()

	if namingIndexSynced {
		return
	}
	syncNamingIndex()
	namingIndexSynced = true
}

func syncNamingIndex() {
	for char, h := range hanzi.HanziData {
		if len(h.NamingCategories) > 0 {
			fate.AddNamingChar(char, h.NamingCategories, h.Gender)
		}
	}
}

// --- 领域接口适配器 ---

// BaziAdapter 八字分析适配器
type BaziAdapter struct{}

func (a *BaziAdapter) Analyze(year, month, day, hour, minute int) (*bazi.BaziAnalysis, error) {
	return bazi.AnalyzeBazi(year, month, day, hour, minute)
}

// HexagramAdapter 易经卦象适配器
type HexagramAdapter struct{}

func (a *HexagramAdapter) FindByMeihuaName(surnameStrokes, givenStrokes int) *yijing.Hexagram {
	return yijing.GetHexagramByMeihuaName(surnameStrokes, givenStrokes)
}

// FindMeihuaDetail 梅花易数姓名完整起卦：本卦/变卦/互卦/动爻与六爻（docs/42 P0-4）
func (a *HexagramAdapter) FindMeihuaDetail(surnameStrokes, givenStrokes int) *yijing.PlumBlossomResult {
	return yijing.GetMeihuaDetailByStrokes(surnameStrokes, givenStrokes)
}

func (a *HexagramAdapter) FindByNumber(id int) *yijing.Hexagram {
	return yijing.GetHexagramByNumber(id)
}

func (a *HexagramAdapter) FindAll() []yijing.Hexagram {
	return yijing.GetAllHexagrams()
}

// ZiweiAdapter 紫微斗数适配器
type ZiweiAdapter struct{}

func (a *ZiweiAdapter) Analyze(year, month, day, hour int, gender string) *ziwei.ZiweiChart {
	return ziwei.AnalyzeZiwei(year, month, day, hour, gender)
}

// ============================================================
// CuratedPersisterAdapter 基础设施 CuratedStore → 领域 CuratedPersister
// ============================================================

// CuratedPersisterAdapter 将 database.CuratedStore 适配为 name.CuratedPersister
type CuratedPersisterAdapter struct {
	store database.CuratedStore
}

// NewCuratedPersisterAdapter 创建适配器
func NewCuratedPersisterAdapter(store database.CuratedStore) *CuratedPersisterAdapter {
	return &CuratedPersisterAdapter{store: store}
}

func (a *CuratedPersisterAdapter) SaveCuratedName(name, pinyin, gender string, score float64, source string) error {
	return a.store.SaveCuratedName(name, pinyin, gender, score, source)
}

func (a *CuratedPersisterAdapter) LoadAllCuratedNames() ([]name.CuratedNameEntry, error) {
	entries, err := a.store.LoadAllCuratedNames()
	if err != nil {
		return nil, err
	}
	result := make([]name.CuratedNameEntry, len(entries))
	for i, e := range entries {
		result[i] = name.CuratedNameEntry{
			Name:    e.Name,
			Pinyin:  e.Pinyin,
			Gender:  e.Gender,
			Score:   e.Score,
			Source:  e.Source,
			Meaning: e.Meaning,
			Wuxing:  e.Wuxing,
			Styles:  e.Styles,
			Tags:    e.Tags,
		}
	}
	return result, nil
}

func (a *CuratedPersisterAdapter) IsCurated(name string) (bool, error) {
	return a.store.IsCurated(name)
}

// ZodiacAdapter 生肖适配器
type ZodiacAdapter struct{}

func (a *ZodiacAdapter) FindByYear(year int) string {
	return zodiac.GetZodiacByYear(year).Name
}

// --> fate 包适配器

//

// NewHanziDataProvider 汉字 CharacterProvider 的**唯一构造点**。
//

// 领域接口实现守卫
var _ bazi.BaziAnalyzer = (*BaziAdapter)(nil)
var _ yijing.HexagramFinder = (*HexagramAdapter)(nil)
var _ ziwei.ZiweiAnalyzer = (*ZiweiAdapter)(nil)
var _ zodiac.ZodiacFinder = (*ZodiacAdapter)(nil)

// fate 层接口实现守卫
var _ fate.CharacterProvider = (*HanziDataProvider)(nil)
var _ fate.BaziAnalyzer = (*BaziAnalyzerAdapter)(nil)

// SetSQLFilter 注入 SQLite 过滤后端（可选启用 SQLite 下沉）
//
// 装配方式：
//
//	provider := &HanziDataProvider{}
//	provider.SetSQLFilter(sqliteAdapter)
//	// sqliteAdapter 实现 SQLHanziFilter 接口，桥接 sqlite.Store
//
// 不注入则走默认 Go 端全表过滤（向后兼容）。
func (p *HanziDataProvider) SetSQLFilter(f SQLHanziFilter) {
	p.sqlFilter = f
}

// queryViaSQL 通过 SQL 后端查询汉字（SQLite 下沉入口）
//
// SQL 端只下推可精确表达的等值/区间条件（单值五行、笔画范围、字表，命中索引，
// 作为候选召回）。**本函数不再回查内存 hanzi.HanziData**：原先因 SQL 表缺
// gender / naming_categories / radical / level 等列，须逐字 `hanzi.GetHanzi`
// 补全字段后再按 pattern 二次过滤（adapters.go 原 560-601 行）。
// 现 hanzi_data 已落齐这些列（见 database.Hanzi 的 P2 字段说明），
// 直接由 databaseHanziToCharacter 从 SQL 行构造 Character 并按 pattern 过滤，
// 派生规则与内存路径共用 deriveCharacterFlags，保证两路结果等价（ADR-003）。
//
