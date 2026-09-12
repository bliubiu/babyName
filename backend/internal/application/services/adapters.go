package services

import (
	"fmt"
	"sync"
	"time"

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

func (a *HexagramAdapter) FindByStrokes(strokes int) *yijing.Hexagram {
	return yijing.GetHexagramByStrokes(strokes)
}

func (a *HexagramAdapter) FindByNumber(id int) *yijing.Hexagram {
	return yijing.GetHexagramByNumber(id)
}

func (a *HexagramAdapter) FindAll() []yijing.Hexagram {
	return yijing.GetAllHexagrams()
}

func (a *HexagramAdapter) MatchXiyongshen(hexagram *yijing.Hexagram, xiyongshen []string) *yijing.HexagramMatch {
	return yijing.MatchHexagramWithXiyongshen(hexagram, xiyongshen)
}

// ZiweiAdapter 紫微斗数适配器
type ZiweiAdapter struct{}

func (a *ZiweiAdapter) Analyze(year, month, day, hour int, gender string) *ziwei.ZiweiAnalysis {
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
			Name:   e.Name,
			Pinyin: e.Pinyin,
			Gender: e.Gender,
			Score:  e.Score,
			Source: e.Source,
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

// HanziDataProvider 基于 hanzi.HanziData 的 CharacterProvider 实现
//
// FindCharacters 默认走 Go 端全表过滤；如配置了 WithSQLiteFilter，则优先走
// SQL 查询（docs/19 报告 P3 方案6 落地）。SQLite 不可用时降级回 Go 过滤。
type HanziDataProvider struct {
	// sqlFilter 可选：注入后 FindCharacters 走 SQL（与 in-memory 路径并存）
	sqlFilter SQLHanziFilter
}

// SQLHanziFilter SQL 下推过滤接口（避免直接依赖 sqlite 包造成循环）
//
// 实现方在 application/services 层负责 sqlite.store 与本接口的桥接；
// 典型实现是 sqliteCharacterProvider（见本文件末尾）。
//
// 返回结果为 []database.Hanzi（只承载粗筛维度），FindCharacters 收到候选后
// 以内存 hanzi.HanziData 回查补全完整属性。
type SQLHanziFilter interface {
	// SearchHanziByFilter 按过滤条件返回 SQL 查询结果
	SearchHanziByFilter(wuxing string, minStrokes, maxStrokes int,
		hasPositive, isRegular bool, chars []string, limit int) []database.Hanzi
}

func (p *HanziDataProvider) GetCharacter(char string) (*fate.Character, error) {
	h, ok := hanzi.HanziData[char]
	if !ok {
		return nil, nil
	}
	return hanziToCharacter(h), nil
}

func (p *HanziDataProvider) FindCharacters(query fate.CharacterQuery) ([]*fate.Character, error) {
	pattern := extractFilterPattern(query)

	// SQLite 下沉（P3 方案6 落地）：sqlFilter 注入时优先走 SQL + 索引召回，
	// 失败/未注入时降级回 Go 端全表过滤（向后兼容）。
	if p.sqlFilter != nil {
		hanzis, err := p.queryViaSQL(pattern)
		if err == nil {
			// 最终字符数据以内存 HanziData 全量为权威源：
			// SQL 表仅承载粗筛维度（char/pinyin/wuxing/strokes/usage_level/positive_score），
			// Radical/Meaning/Gender 等完整属性回查内存补全，避免 SQL 路径劣化评分质量。
			result := make([]*fate.Character, 0, len(hanzis))
			for i := range hanzis {
				full, ok := hanzi.HanziData[hanzis[i].Char]
				if !ok {
					continue
				}
				result = append(result, hanziToCharacter(full))
			}
			return result, nil
		}
		// SQL 失败 → 降级 Go 过滤（不阻断服务）
	}

	var result []*fate.Character
	// 如果 query 是 basicCharacterQuery，提取过滤条件
	for _, h := range hanzi.HanziData {
		c := hanziToCharacter(h)

		// 应用过滤条件
		if pattern.regularFilter && !c.IsRegular {
			continue
		}
		if pattern.nameableFilter && !c.IsNameable {
			continue
		}
		if pattern.strokeEQ > 0 && c.ScienceStroke != pattern.strokeEQ {
			continue
		}
		if pattern.strokeGTE > 0 && c.ScienceStroke < pattern.strokeGTE {
			continue
		}
		if pattern.strokeLTE > 0 && c.ScienceStroke > pattern.strokeLTE {
			continue
		}
		if len(pattern.wuxingIn) > 0 && !inSlice(c.WuXing, pattern.wuxingIn) {
			continue
		}
		if len(pattern.wuxingNotIn) > 0 && inSlice(c.WuXing, pattern.wuxingNotIn) {
			continue
		}
		if len(pattern.charIn) > 0 && !inSlice(c.Char, pattern.charIn) {
			continue
		}
		if pattern.genderHint != "" && c.GenderHint != pattern.genderHint && c.GenderHint != "neutral" {
			continue
		}
		if pattern.namingCategory != "" && !inSlice(pattern.namingCategory, c.NamingCategory) {
			continue
		}
		result = append(result, c)
	}
	return result, nil
}

func (p *HanziDataProvider) GetSurnameStrokes(surname string) (int, int, error) {
	// 优先查康熙笔画表（河图数理与易经卦象解读使用康熙笔画）
	if l1, l2 := fate.LookupSurnameStrokes(surname); l1 > 0 {
		return l1, l2, nil
	}
	// 未收录的姓氏降级使用简体笔画
	runes := []rune(surname)
	if len(runes) == 0 {
		return 0, 0, nil
	}
	l1 := getStroke(string(runes[0]))
	if len(runes) >= 2 {
		return l1, getStroke(string(runes[1])), nil
	}
	return l1, 0, nil
}

func (p *HanziDataProvider) CountCharacters(query fate.CharacterQuery) (int, error) {
	chars, err := p.FindCharacters(query)
	if err != nil {
		return 0, err
	}
	return len(chars), nil
}

// hanziToCharacter 将 hanzi.Hanzi 转换为 fate.Character
func hanziToCharacter(h hanzi.Hanzi) *fate.Character {
	// 以《通用规范汉字表》等级为准判定常用字（namer.json level：1=一级3500常用字）
	// level==0 表示不在规范表中（旧版精选字库补充字），保守沿用笔画+生僻字表判定
	level := hanzi.GetNamerLevel(h.Char)
	// 一级字(3500)+二级字(3000) 全量纳入，三级字(1605)暂不纳入需逐字审核
	// 表外字(level=0)按笔画+生僻字表保守判定
	isRegular := level == 1 || level == 2
	if level == 0 {
		isRegular = h.Strokes <= 25 && !hanzi.RareChars[h.Char]
	}
	isNameable := h.Strokes <= 30 && isRegular

	// 笔画口径统一：
	//   - SimplifiedStroke / TraditionalStroke: namer 简体笔画（h.Strokes）
	//   - KangxiStroke: 康熙字典笔画（h.KangxiStrokes，0 时降为简体）
	//   - ScienceStroke: 简体笔画（科学笔画口径，与 Kangxi 区分）
	//
	// 此前四个字段均填 h.Strokes，导致"姓用康熙、名用科学"的口径混乱
	// （参见 docs/19 报告 B3）。修复要点：
	//   1. KangxiStroke 走真实康熙字典笔画（namer_loader 已合并 kangxi-strokecount.csv）
	//   2. KangxiStroke == 0 时降级为简体（未收录兜底）
	kangxiStroke := h.KangxiStrokes
	if kangxiStroke == 0 {
		kangxiStroke = h.Strokes
	}

	c := &fate.Character{
		Char:              h.Char,
		Pinyin:            []string{h.Pinyin},
		WuXing:            h.Wuxing,
		SimplifiedStroke:  h.Strokes,
		TraditionalStroke: h.Strokes,
		KangxiStroke:      kangxiStroke,
		ScienceStroke:     h.Strokes,
		Radical:           h.Radical,
		Meaning:           h.Meaning,
		IsRegular:         isRegular,
		IsNameable:        isNameable,
		CommonLevel:       level,
		GenderHint:        h.Gender,
		HasPoetry:         false,

		// 数据层策展标注：直接透传 hanzi.json 的策展判断
		// （isNegative=不宜入名 / namePenalty=起名扣分 / pairBlacklist=搭配黑名单）
		IsNegative:    h.IsNegative,
		NamePenalty:   h.NamePenalty,
		PairBlacklist: h.PairBlacklist,

		// 策展覆盖表标记：人工精选起名好字（17 分类约 446 字）是荒谬字与好字的强区分信号，
		// WenHuaRater 据此给策展字文化加分，打破单名 Top5 五维同分的僵局。
		IsCurated: hanzi.IsCuratedNamingChar(h.Char),

		// 寓意评分：namer.json positiveScore（0-100），策展加分收窄为「策展字 ∩ positiveScore>=85」
		// 的精选好字专属，平庸字（软/际/映/耿 等分类字）虽然也在策展表内但不享受加分。
		PositiveScore: h.PositiveScore,

		// 人名频率档位（1-5），来自 Chinese-Names-Corpus 语料统计
		NameFreqTier: h.NameFreqTier,
	}

	// 标注起名分类（优先使用 HanziData 的全量分类数据，其次 fallback 到 fate 精选库）
	if cats := hanzi.GetNamingCategories(c.Char); len(cats) > 0 {
		c.NamingCategory = cats
	} else if info := fate.GetNamingCharInfo(c.Char); info != nil {
		c.NamingCategory = info.Category
	}

	return c
}

// extractFilterPattern 从 CharacterQuery 中提取过滤模式
// 如果 query 是 *basicCharacterQuery，直接读取字段
// 否则遍历所有过滤条件应用到统一结构
func extractFilterPattern(query fate.CharacterQuery) filterPattern {
	p := filterPattern{}
	if q, ok := query.(*fate.BasicCharacterQuery); ok {
		p.regularFilter = q.RegularFilter
		p.nameableFilter = q.NameableFilter
		p.strokeEQ = q.StrokeEQ
		p.strokeGTE = q.StrokeGTE
		p.strokeLTE = q.StrokeLTE
		p.wuxingIn = q.WuxingIn
		p.wuxingNotIn = q.WuxingNotIn
		p.charIn = q.CharIn
		p.genderHint = q.GenderHint
		p.namingCategory = q.NamingCategory
	}
	return p
}

// filterPattern 简单过滤模式（与 basicCharacterQuery 字段映射）
type filterPattern struct {
	regularFilter  bool
	nameableFilter bool
	strokeEQ       int
	strokeGTE      int
	strokeLTE      int
	wuxingIn       []string
	wuxingNotIn    []string
	charIn         []string
	genderHint     string
	namingCategory string // 精选起名分类
}

// getStroke 获取单字笔画
func getStroke(char string) int {
	if h, ok := hanzi.HanziData[char]; ok {
		return h.Strokes
	}
	return 0
}

// inSlice 检查字符串是否在切片中
func inSlice(s string, slice []string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// BaziAnalyzerAdapter 基于 domain/bazi 的 BaziAnalyzer 实现
type BaziAnalyzerAdapter struct{}

func NewBaziAnalyzerAdapter() *BaziAnalyzerAdapter {
	return &BaziAnalyzerAdapter{}
}

func (a *BaziAnalyzerAdapter) Analyze(born time.Time, gender fate.Gender) (*fate.FateData, error) {
	year, month, day := born.Date()
	hour := born.Hour()
	minute := born.Minute()

	baziResult, err := bazi.AnalyzeBazi(year, int(month), day, hour, minute)
	if err != nil {
		return nil, err
	}

	zodiacName := zodiac.GetZodiacByYear(year).Name
	fourPillars := baziGanzhiToArray(baziResult.Bazi)

	// 使用 fate 层的平衡用神法计算完整喜用忌仇四神
	baziInfo := fate.BaziInfoForGeJu{
		SiZhu: fourPillars,
	}
	xiyong := fate.BalanceXiYongJi(baziInfo)

	// 构建喜用神排序列表：用神 > 喜神
	xiYongShen := make([]string, 0, 2)
	if xiyong.Yong != "" {
		xiYongShen = append(xiYongShen, xiyong.Yong)
	}
	if xiyong.Xi != "" && xiyong.Xi != xiyong.Yong {
		xiYongShen = append(xiYongShen, xiyong.Xi)
	}
	// fallback: 若 balance 法未能产出，使用 bazi 层原始结果
	if len(xiYongShen) == 0 {
		xiYongShen = baziResult.Xiyongshen
	}

	// 构建 FateData
	fateData := &fate.FateData{
		BaziInfo: fate.BaziInfo{
			FourPillars: fourPillars,
			Zodiac:      zodiacName,
		},
		WuXingXiji: fate.WuXingXiji{
			XiYongShen:  xiYongShen,
			YongShen:    xiyong.Yong,
			Xi:          xiyong.Xi,
			Ji:          xiyong.Ji,
			Chou:        xiyong.Chou,
			RiZhu:       baziResult.Rishou,
			RiZhuWuXing: baziResult.RishouWuxing,
			QiangRuo:    baziResult.DayMasterStrength,
		},
		TiaoHouYongShen: joinStrings(baziResult.TiaohouShen, "、"),
	}

	// fallback: 若 balance 法未产出用神，用 bazi 原始映射
	if fateData.WuXingXiji.YongShen == "" && len(baziResult.Xiyongshen) > 0 {
		fateData.WuXingXiji.YongShen = baziResult.Xiyongshen[0]
		fateData.WuXingXiji.Xi = baziResult.Xiyongshen[0]
	}
	if fateData.WuXingXiji.Ji == "" && len(baziResult.Xiyongshen) > 1 {
		fateData.WuXingXiji.Ji = baziResult.Xiyongshen[1]
	}

	// 设置四柱五行和纳音
	// bazi.BaziAnalysis 没有直接提供四柱五行/纳音数组，
	// 通过四个天干推算
	tiangan := []rune(baziResult.Bazi.YearGanzhi + baziResult.Bazi.MonthGanzhi + baziResult.Bazi.DayGanzhi + baziResult.Bazi.HourGanzhi)
	for i, tg := range tiangan {
		if i%2 == 0 { // 天干
			if i/2 < len(fateData.BaziInfo.WuXing) {
				fateData.BaziInfo.WuXing[i/2] = bazi.WuxingMap[string(tg)]
			}
		}
	}

	fateData.BaziInfo.NaYin[0] = baziResult.Nayin

	return fateData, nil
}

// baziGanzhiToArray 将 Bazi 的四个柱转为 [4]string
func baziGanzhiToArray(b bazi.Bazi) [4]string {
	return [4]string{b.Year, b.Month, b.Day, b.Hour}
}

// joinStrings 拼接字符串切片
func joinStrings(s []string, sep string) string {
	if len(s) == 0 {
		return ""
	}
	result := s[0]
	for _, v := range s[1:] {
		result += sep + v
	}
	return result
}

// --- 接口守卫（编译时检查） ---

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
// 作为候选召回）。SQL 表仅承载粗筛维度，genderHint / namingCategory / wuxingNotIn /
// 多值 wuxingIn / regular（含表外字保守判定）/ nameable / strokeEQ 等条件由本函数
// 回查内存 hanzi.HanziData 全量数据，按与 FindCharacters 一致的语义二次过滤，
// 保证与纯 Go 路径结果一致。SQL 失败时返回非 nil error，由调用方降级到 Go 端全表过滤。
func (p *HanziDataProvider) queryViaSQL(pat filterPattern) ([]database.Hanzi, error) {
	if p.sqlFilter == nil {
		return nil, fmt.Errorf("sqlFilter 未注入")
	}

	// 多值 wuxingIn 时 SQL 端仅支持单值等值匹配，不下推，交由下方 Go 端二次过滤。
	wuxing := ""
	if len(pat.wuxingIn) == 1 {
		wuxing = pat.wuxingIn[0]
	}
	hanzis := p.sqlFilter.SearchHanziByFilter(
		wuxing, pat.strokeGTE, pat.strokeLTE,
		false, false, pat.charIn, 10000,
	)
	if len(hanzis) == 0 {
		return hanzis, nil
	}

	// Go 端二次过滤：与 FindCharacters 的 pattern 匹配逐项一致
	filtered := hanzis[:0]
	for i := range hanzis {
		h := &hanzis[i]
		// 字段以内存全量数据为准（SQL 表无 gender/categories/radical 等列）
		full, ok := hanzi.HanziData[h.Char]
		if !ok {
			continue
		}
		c := hanziToCharacter(full)

		if pat.regularFilter && !c.IsRegular {
			continue
		}
		if pat.nameableFilter && !c.IsNameable {
			continue
		}
		if pat.strokeEQ > 0 && c.ScienceStroke != pat.strokeEQ {
			continue
		}
		if pat.strokeGTE > 0 && c.ScienceStroke < pat.strokeGTE {
			continue
		}
		if pat.strokeLTE > 0 && c.ScienceStroke > pat.strokeLTE {
			continue
		}
		if len(pat.wuxingIn) > 0 && !inSlice(c.WuXing, pat.wuxingIn) {
			continue
		}
		if len(pat.wuxingNotIn) > 0 && inSlice(c.WuXing, pat.wuxingNotIn) {
			continue
		}
		if len(pat.charIn) > 0 && !inSlice(c.Char, pat.charIn) {
			continue
		}
		if pat.genderHint != "" && c.GenderHint != pat.genderHint && c.GenderHint != "neutral" {
			continue
		}
		if pat.namingCategory != "" && !inSlice(pat.namingCategory, c.NamingCategory) {
			continue
		}
		filtered = append(filtered, *h)
	}
	return filtered, nil
}

// SQLiteCharStore 最小接口（避免在 services 包 import sqlite 包造成循环依赖）
//
// sqlite.Store 已实现 SearchHanziByFilter(HanziFilter) []*database.Hanzi，
// 但参数类型 HanziFilter 在 sqlite 包内部定义。本接口用更宽松的签名
// 桥接，使 services 包只需 database.Hanzi 数据 + 基础过滤条件。
type SQLiteCharStore interface {
	SearchHanziByFilter(wuxing string, minStrokes, maxStrokes int,
		hasPositive, isRegular bool, chars []string, limit int) []*database.Hanzi
}

// sqliteHanziFilterAdapter 桥接实现：把 SQLHanziFilter 调用转给 sqlite.Store
//
// 为什么不直接 import sqlite 包：services 包已被 application/handlers 引用，
// 而 cmd/server 同时引用 services + sqlite，加 import 会让 services 反向依赖
// infrastructure/database/sqlite，破坏分层。
// 解法：services 定义接口 SQLiteCharStore（够用即可），由调用方注入实现。
type sqliteHanziFilterAdapter struct {
	store SQLiteCharStore
}

// NewSQLiteHanziFilter 创建基于 sqlite.Store 的过滤适配器
//
// 装配示例（cmd/server/main.go）：
//
//	filter := services.NewSQLiteHanziFilter(store)
//	provider := &services.HanziDataProvider{}
//	provider.SetSQLFilter(filter)
//
// 运行时：FindCharacters 优先走 SQL + 索引，失败降级到 Go 端全表过滤。
func NewSQLiteHanziFilter(store SQLiteCharStore) SQLHanziFilter {
	return &sqliteHanziFilterAdapter{store: store}
}

func (a *sqliteHanziFilterAdapter) SearchHanziByFilter(
	wuxing string, minStrokes, maxStrokes int,
	hasPositive, isRegular bool, chars []string, limit int,
) []database.Hanzi {
	if a.store == nil {
		return nil
	}
	// 调底层 store 拿到 []*database.Hanzi，转 []database.Hanzi 返回
	raw := a.store.SearchHanziByFilter(
		wuxing, minStrokes, maxStrokes,
		hasPositive, isRegular, chars, limit,
	)
	result := make([]database.Hanzi, 0, len(raw))
	for i := range raw {
		result = append(result, *raw[i])
	}
	return result
}

// 接口守卫：sqliteHanziFilterAdapter 必须实现 SQLHanziFilter
var _ SQLHanziFilter = (*sqliteHanziFilterAdapter)(nil)

// HasSQLiteCharStore 报告任意 store 是否实现了 SQLiteCharStore 接口（用于类型断言）
//
// 调用方在装配时不必关心 store 具体类型，通过此 helper 自动识别：
//
//	if services.HasSQLiteCharStore(store) {
//	    filter := services.NewSQLiteHanziFilter(store.(services.SQLiteCharStore))
//	    provider.SetSQLFilter(filter)
//	}
func HasSQLiteCharStore(store database.Store) bool {
	_, ok := store.(SQLiteCharStore)
	return ok
}

// AsSQLiteCharStore 桥接 database.Store 到 SQLiteCharStore（仅在 HasSQLiteCharStore 返回 true 时安全）
//
// 返回值供 NewSQLiteHanziFilter 消费。type assertion 失败时返回 nil。
func AsSQLiteCharStore(store database.Store) SQLiteCharStore {
	if s, ok := store.(SQLiteCharStore); ok {
		return s
	}
	return nil
}
