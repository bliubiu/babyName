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
	"name/internal/infrastructure/logger"
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

func NewHanziDataProvider() *HanziDataProvider {
	return &HanziDataProvider{}
}

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
	// SearchHanziByFilter 按过滤条件返回 SQL 查询结果。
	//
	// 必须返回 error（docs/29 B8）：此前签名只有返回值，SQL 查询出错时
	// store 返回 nil，FindCharacters 判空后当成「SQL 正常但无匹配」，
	// 于是注释里承诺的「SQL 失败降级 Go 端全表过滤」路径永远不可达 ——
	// 基础设施故障被伪装成业务空结果（生成不出名字，却看不出是 DB 挂了）。
	// 返回 nil slice + nil error 表示「查询成功且确实无匹配」，两者必须可区分。
	SearchHanziByFilter(wuxing string, minStrokes, maxStrokes int,
		hasPositive, isRegular bool, chars []string, limit int) ([]database.Hanzi, error)
	// HasData 报告底层字库是否载有数据。
	//
	// B8 的孪生陷阱（本次修复，空库回落场景实测）：库未执行
	// `materials import`（或物料被清空）时，SQL 查询**成功且零行**，
	// 与「查询成功、确实无匹配」在返回值上完全一致。若一律按无匹配
	// 处理，回落场景（DB 缺料 → 内存已装 JSON 字库）会拿到空候选池，
	// 表现为全部生成请求失败，且看不出是库没导入。
	//
	// 因此 FindCharacters 在 SQL 返回空结果时调用本方法甄别：
	// 无数据 → 降级 Go 端全表过滤；有数据 → 如实返回空。
	HasData() bool
}

func (p *HanziDataProvider) GetCharacter(char string) (*fate.Character, error) {
	h, ok := hanzi.GetHanzi(char)
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
			// 空库甄别（B8 孪生陷阱修复）：库未导入物料时 SQL「成功且零行」，
			// 与「有数据但确实无匹配」不可区分。HasData()=false 时内存已由
			// dataloader 回落装上 JSON 字库，必须降级 Go 端过滤，否则回落
			// 场景候选池恒空、生成全失败（空库实测：每次请求 400/500）。
			if len(hanzis) == 0 && !p.sqlFilter.HasData() {
				logger.Warn("SQLite 字库无数据（未执行 materials import？），降级 Go 端全表过滤")
			} else {
				// P2（ADR-003）：queryViaSQL 已完成召回 + 二次过滤，且 hanzi_data 已落齐
				// 构造 Character 所需的全部列，故直接由 SQL 行构造，不再逐字
				// `hanzi.GetHanzi` 回查内存（原先此处是第二处回查，且会丢弃 SQL 返回的
				// level/gender/radical/meaning 等字段，使 SQL 下沉收益归零）。
				result := make([]*fate.Character, 0, len(hanzis))
				for i := range hanzis {
					result = append(result, databaseHanziToCharacter(&hanzis[i]))
				}
				return result, nil
			}
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

// deriveCharacterFlags 由「通用规范汉字表等级 + 简体笔画 + 生僻标记」派生
// IsRegular / IsNameable / CommonLevel。
//
// 这是 SQL 路径与内存路径唯一的共有派生逻辑（ADR-003「结果等价」的关键）：
// 两个路径的差别只在**输入来源**（hanzi_data.level/is_rare vs GetNamerLevel/RareChars），
// 判定规则本身必须同源，否则同一字在两条路径下 IsRegular 可能不一致。
func deriveCharacterFlags(level, strokes int, isRare bool) (isRegular, isNameable bool, commonLevel int) {
	// 一级字(3500)+二级字(3000) 全量纳入，三级字(1605)暂不纳入需逐字审核
	// 表外字(level=0)按笔画+生僻字表保守判定
	isRegular = level == 1 || level == 2
	if level == 0 {
		isRegular = strokes <= 25 && !isRare
	}
	isNameable = strokes <= 30 && isRegular
	return isRegular, isNameable, level
}

// hanziToCharacter 将 hanzi.Hanzi 转换为 fate.Character
func hanziToCharacter(h hanzi.Hanzi) *fate.Character {
	// 以《通用规范汉字表》等级为准判定常用字（namer.json level：1=一级3500常用字）
	// level==0 表示不在规范表中（旧版精选字库补充字），保守沿用笔画+生僻字表判定
	//
	// 生僻字判定取**并集**：hanzi.Hanzi.IsRare（namer.json 落库的 99 字）与
	// hanzi.RareChars（编译期静态表，含 CJK 扩展区字）。两者实测有 102 字不一致，
	// 只取其一会使同一字在内存路径与 SQL 路径下 IsRegular 得到相反结论
	// （SQL 路径见 databaseHanziToCharacter，同样取并集）。取并集即取保守侧，
	// 与 docs/31「取名优选范围：避开生僻字」的安全策略一致。
	isRare := h.IsRare || hanzi.RareChars[h.Char]
	level := hanzi.GetNamerLevel(h.Char)
	isRegular, isNameable, commonLevel := deriveCharacterFlags(level, h.Strokes, isRare)

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
		CommonLevel:       commonLevel,
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

// databaseHanziToCharacter 将 database.Hanzi（SQLite 查询结果）转换为 fate.Character。
//
// P2（ADR-003）：与 hanziToCharacter 的区别仅在于**输入优先取自 SQL 返回的列**，
// 不再逐字 `hanzi.GetHanzi` 回查内存。派生规则共用 deriveCharacterFlags，
// 保证同一字在 SQL 路径与内存路径下的 IsRegular / IsNameable / CommonLevel 一致。
func databaseHanziToCharacter(h *database.Hanzi) *fate.Character {
	// 生僻字判定必须是两路径的**并集**，否则同一字在两条路径下 IsRegular 可能相反。
	//
	// 实测（filter_integration_test 守护）：`hanzi_data.is_rare` 来自 namer.json（99 字），
	// `hanzi.RareChars` 是编译期静态表（含 CJK 扩展区字），两者有 102 字不一致。
	// 若 SQL 路径只信 is_rare、内存路径只信 RareChars，同一字会得到相反的 IsRegular
	// ——「SQL 与内存结果等价」当场破功。故此处取并集：
	// 任一侧判为生僻即按生僻处理（取保守侧，与「取名优选范围」的安全策略一致）。
	isRare := h.IsRare || hanzi.RareChars[h.Char]
	isRegular, isNameable, commonLevel := deriveCharacterFlags(h.Level, h.Strokes, isRare)

	// KangxiStroke == 0 时降级为简体（未收录兜底），口径与内存路径一致
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
		CommonLevel:       commonLevel,
		GenderHint:        h.Gender,
		HasPoetry:         false,

		// 策展标注与评分：全部取自 SQL 行。
		// 这些字段不是「装饰」——is_negative / name_penalty / name_freq_tier 会实打实
		// 参与 fate 评分；一旦在 SQL 路径置零，生成链路（走 FindCharacters/SQL）与
		// 测名链路（走 GetCharacter/内存）就会算出不同总分，
		// 违反「测名与生成同源同分」契约（e2e_check 第 7 项 / evaluate_e2e_test.go）。
		IsNegative:  h.IsNegative,
		NamePenalty: h.NamePenalty,
		// pair_blacklist 上游无真值（内存侧实测 0 字有值），两路径同为「无禁忌搭配」
		PairBlacklist: nil,

		// 策展覆盖表：curated_names 属运行时自学习数据，未纳入统一 import，
		// 仍查内存静态表（无 I/O，不构成回查）。
		IsCurated: hanzi.IsCuratedNamingChar(h.Char),

		PositiveScore: h.PositiveScore,
		// 人名频率档位（1-5），由 name_frequency 语料合并而来，已随 import 落库
		NameFreqTier: h.NameFreqTier,
	}

	// 起名分类：SQL 已返回 naming_categories，优先使用；为空时 fallback 到 fate 精选库
	if len(h.NamingCategories) > 0 {
		c.NamingCategory = h.NamingCategories
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
	if h, ok := hanzi.GetHanzi(char); ok {
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

	// 喜用神单一真源（docs/43 P1-3）：以 bazi weighted 算法（calculateXiyongshenWeighted）
	// 为唯一权威——喜用神列表、强弱、忌仇全部同源于本次 AnalyzeBazi 结果。
	// 此前此处另调 fate.BalanceXiYongJi（total/2 简单多数判强弱）再与
	// bazi 的 DayMasterStrength 混装，同一响应内喜用神与强弱栏可能自相矛盾；
	// 测名路径（fate_name_service.classicXiyongShen）早已统一为经典口径，
	// 本适配器是最后一块，收敛后 /generate、/generate/analysis、测名三链路同源。
	xiYongShen := baziResult.Xiyongshen
	yongShen := ""
	xi := ""
	if len(xiYongShen) > 0 {
		yongShen = xiYongShen[0]
		xi = xiYongShen[0]
	}
	if len(xiYongShen) > 1 {
		xi = xiYongShen[1]
	}
	// 忌/仇按同一强弱结论推导（weighted 算法无忌仇概念，扶抑法补齐）
	ji, chou := bazi.DeriveJiChou(baziResult.RishouWuxing, baziResult.DayMasterStrength)

	// 构建 FateData
	fateData := &fate.FateData{
		BaziInfo: fate.BaziInfo{
			FourPillars: fourPillars,
			Zodiac:      zodiacName,
		},
		WuXingXiji: fate.WuXingXiji{
			XiYongShen:  xiYongShen,
			YongShen:    yongShen,
			Xi:          xi,
			Ji:          ji,
			Chou:        chou,
			RiZhu:       baziResult.Rishou,
			RiZhuWuXing: baziResult.RishouWuxing,
			QiangRuo:    baziResult.DayMasterStrength,
		},
		TiaoHouYongShen: joinStrings(baziResult.TiaohouShen, "、"),
		// 出生季节（月支节气月）：卦气旺衰依据（docs/42 P3-12 卦象评分）
		BirthSeason: baziResult.Season,
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

	// 四柱纳音
	if four := baziResult.FourNayin; four != nil {
		fateData.BaziInfo.NaYin = [4]string{
			four.Year.Nayin, four.Month.Nayin, four.Day.Nayin, four.Hour.Nayin,
		}
	} else {
		fateData.BaziInfo.NaYin[0] = baziResult.Nayin
	}

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
// 作为候选召回）。**本函数不再回查内存 hanzi.HanziData**：原先因 SQL 表缺
// gender / naming_categories / radical / level 等列，须逐字 `hanzi.GetHanzi`
// 补全字段后再按 pattern 二次过滤（adapters.go 原 560-601 行）。
// 现 hanzi_data 已落齐这些列（见 database.Hanzi 的 P2 字段说明），
// 直接由 databaseHanziToCharacter 从 SQL 行构造 Character 并按 pattern 过滤，
// 派生规则与内存路径共用 deriveCharacterFlags，保证两路结果等价（ADR-003）。
//
// SQL 失败时返回非 nil error，由调用方降级到 Go 端全表过滤。
func (p *HanziDataProvider) queryViaSQL(pat filterPattern) ([]database.Hanzi, error) {
	if p.sqlFilter == nil {
		return nil, fmt.Errorf("sqlFilter 未注入")
	}

	// 多值 wuxingIn 时 SQL 端仅支持单值等值匹配，不下推，交由下方 Go 端二次过滤。
	wuxing := ""
	if len(pat.wuxingIn) == 1 {
		wuxing = pat.wuxingIn[0]
	}
	hanzis, err := p.sqlFilter.SearchHanziByFilter(
		wuxing, pat.strokeGTE, pat.strokeLTE,
		false, false, pat.charIn, 10000,
	)
	if err != nil {
		return nil, fmt.Errorf("SQL 汉字筛选失败: %w", err)
	}
	if len(hanzis) == 0 {
		// 查询成功且确实无匹配 —— 与上面的失败路径必须可区分（docs/29 B8）
		return nil, nil
	}

	// 二次过滤：与 FindCharacters 的 pattern 匹配逐项一致，但字段全部取自 SQL 行。
	filtered := make([]database.Hanzi, 0, len(hanzis))
	for i := range hanzis {
		h := &hanzis[i]
		c := databaseHanziToCharacter(h)

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
		hasPositive, isRegular bool, chars []string, limit int) ([]*database.Hanzi, error)
	// HanziRowCount 返回 hanzi_data 行数，供 HasData 甄别「库无数据」
	// 与「有数据但无匹配」（空库回落降级的判据）
	HanziRowCount() (int, error)
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
) ([]database.Hanzi, error) {
	if a.store == nil {
		return nil, fmt.Errorf("底层 store 未注入")
	}
	// 调底层 store 拿到 []*database.Hanzi，转 []database.Hanzi 返回
	raw, err := a.store.SearchHanziByFilter(
		wuxing, minStrokes, maxStrokes,
		hasPositive, isRegular, chars, limit,
	)
	if err != nil {
		// 错误必须上抛，让 FindCharacters 走 Go 端降级路径（docs/29 B8）
		return nil, err
	}
	result := make([]database.Hanzi, 0, len(raw))
	for i := range raw {
		result = append(result, *raw[i])
	}
	return result, nil
}

// HasData 底层 hanzi_data 是否有行
//
// 探测失败（如库损坏）时保守返回 true：让调用方把空结果当作
// 「确实无匹配」如实返回，而不是借一次探测失败就降级 —— 探测本身
// 不应成为新的故障面。真正的查询失败仍由 SearchHanziByFilter 的
// error 路径负责降级（docs/29 B8）。
func (a *sqliteHanziFilterAdapter) HasData() bool {
	if a.store == nil {
		return false
	}
	n, err := a.store.HanziRowCount()
	if err != nil {
		return true
	}
	return n > 0
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
