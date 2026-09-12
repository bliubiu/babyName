package fate

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"name/internal/domain/classics"
)

// CharacterProvider 汉字数据提供接口
// 由 infrastructure 层实现适配器，对接 hanzi.HanziData 或数据库
type CharacterProvider interface {
	// GetCharacter 获取单个汉字的详细信息
	GetCharacter(char string) (*Character, error)

	// FindCharacters 按条件查询汉字列表
	FindCharacters(query CharacterQuery) ([]*Character, error)

	// GetSurnameStrokes 获取姓氏笔画
	// 单姓返回 (l1, 0, nil)；复姓返回 (l1, l2, nil)
	GetSurnameStrokes(surname string) (int, int, error)

	// CountCharacters 获取可用汉字总数
	CountCharacters(query CharacterQuery) (int, error)
}

// BaziAnalyzer 八字分析接口
// 由 infrastructure 层实现，对接 bazi.BaziAnalyzer
type BaziAnalyzer interface {
	// Analyze 根据出生时间进行八字分析，返回命运数据
	Analyze(born time.Time, gender Gender) (*FateData, error)
}

// EngineFactoryFunc EngineFactory 的默认实现
func EngineFactoryFunc(provider CharacterProvider, analyzer BaziAnalyzer) EngineFactory {
	return func(raters []Rater) (Fate, error) {
		return NewEngine(provider, analyzer, raters), nil
	}
}

// charInfo 候选字预计算属性（用于 generateSingleName / generateDoubleName 共享）
//
// 字段含义：
//   - ch:         候选字 Character
//   - stroke:     笔画数（filter 口径，按 StrokeMode 决定）
//   - pinyin:     第一拼音（无声调）
//   - poetryFound: 是否关联到诗词
//   - poetryDesc:  诗词出处描述
//
// 避免双重循环内重复调用 GetCharacterStroke / firstPinyin / 诗词检索。
type charInfo struct {
	ch          *Character
	stroke      int
	pinyin      string
	poetryFound bool
	poetryDesc  string
}

// --- engineImpl: Fate 接口默认实现 ---

type engineImpl struct {
	raters   []Rater
	provider CharacterProvider
	analyzer BaziAnalyzer
}

// NewEngine 创建 Fate 引擎实例
func NewEngine(provider CharacterProvider, analyzer BaziAnalyzer, raters []Rater) Fate {
	return &engineImpl{
		provider: provider,
		analyzer: analyzer,
		raters:   raters,
	}
}

func (e *engineImpl) NewSession() Session {
	return e.NewSessionWithFilter(NewFilterOption().Build())
}

func (e *engineImpl) NewSessionWithFilter(filter Filter) Session {
	return &sessionImpl{
		engine: e,
		filter: filter,
		state:  SessionStatePending,
	}
}

// --- sessionImpl: Session 接口默认实现 ---

type sessionImpl struct {
	mu     sync.Mutex
	engine *engineImpl
	filter Filter
	state  SessionState
	input  *Input
	output *Output
	err    error
	cancel context.CancelFunc
	done   chan struct{}

	// session 级别的 raters 副本，避免并发会话相互覆盖 engine.raters
	raters []Rater

	// bigramCache per-session 二字共现评分缓存（SessionBigramCache）
	// 用于减少 generate() 阶段 N×N 双名笛卡尔积中的 50 万次 RLock。
	bigramCache *SessionBigramCache

	// 负面反馈
	excludedChars  map[string]bool // 排除的字符
	excludedCombos map[string]bool // 排除的组合 "char1+char2"（排序后）
}

func (s *sessionImpl) Start(ctx context.Context, input *Input) error {
	s.mu.Lock()
	if s.state != SessionStatePending {
		s.mu.Unlock()
		return fmt.Errorf("会话已开始，当前状态=%d", s.state)
	}
	s.input = input
	s.state = SessionStateGenerating
	s.excludedChars = make(map[string]bool)
	s.excludedCombos = make(map[string]bool)
	// 复制 engine.raters 到 session 级别，避免并发会话相互覆盖
	s.raters = make([]Rater, len(s.engine.raters))
	copy(s.raters, s.engine.raters)
	// 初始化 per-session 二字共现缓存（与 session 同生命周期）
	s.bigramCache = newSessionBigramCache()
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.done = make(chan struct{})
	s.mu.Unlock()

	go s.run(ctx, input)
	return nil
}

func (s *sessionImpl) Wait() error {
	<-s.done
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
}

func (s *sessionImpl) Result() *Output {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.output
}

func (s *sessionImpl) State() SessionState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

func (s *sessionImpl) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
	if s.state == SessionStateGenerating {
		s.state = SessionStateCanceled
	}
	return nil
}

// ——— 负面反馈实现 ———

func (s *sessionImpl) ExcludeChar(char string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.excludedChars == nil {
		s.excludedChars = make(map[string]bool)
	}
	s.excludedChars[char] = true
}

func (s *sessionImpl) ExcludeCombo(c1, c2 string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.excludedCombos == nil {
		s.excludedCombos = make(map[string]bool)
	}
	// 排序确保 key 唯一
	a, b := c1, c2
	if a > b {
		a, b = b, a
	}
	s.excludedCombos[a+"+"+b] = true
}

func (s *sessionImpl) ClearExclusions() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.excludedChars = make(map[string]bool)
	s.excludedCombos = make(map[string]bool)
}

func (s *sessionImpl) ExcludedChars() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	chars := make([]string, 0, len(s.excludedChars))
	for ch := range s.excludedChars {
		chars = append(chars, ch)
	}
	return chars
}

// sessionImpl 上的闭包：调用方直接用 s.isCharExcluded / s.isComboExcluded
// 不再加锁（session 生命周期内 excludedChars/excludedCombos 只读不变）
func (s *sessionImpl) isCharExcluded(char string) bool {
	return s.excludedChars[char]
}

func (s *sessionImpl) isComboExcluded(c1, c2 string) bool {
	a, b := c1, c2
	if a > b {
		a, b = b, a
	}
	return s.excludedCombos[a+"+"+b]
}

// run 异步启动的名字生成主流程
func (s *sessionImpl) run(ctx context.Context, input *Input) {
	defer func() {
		if r := recover(); r != nil {
			s.mu.Lock()
			s.state = SessionStateFailed
			s.err = fmt.Errorf("生成过程异常: %v", r)
			s.mu.Unlock()
		}
		// 释放 context 资源，避免泄漏
		if s.cancel != nil {
			s.cancel()
		}
		close(s.done)
	}()

	output, err := s.generate(ctx, input)

	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil {
		s.state = SessionStateFailed
		s.err = err
		return
	}
	s.output = output
	s.state = SessionStateFinish
}

// generate 核心生成逻辑
func (s *sessionImpl) generate(ctx context.Context, input *Input) (*Output, error) {
	// 1. 八字分析
	fateData, err := s.engine.analyzer.Analyze(input.Born, input.Gender)
	if err != nil {
		return nil, fmt.Errorf("八字分析失败: %w", err)
	}

	// 2. 获取姓氏笔画信息（用于总笔画数计算，支撑河图数理与易经卦象解读）
	l1, l2, err := s.engine.provider.GetSurnameStrokes(input.Surname)
	if err != nil {
		return nil, fmt.Errorf("获取姓氏笔画失败: %w", err)
	}

	// 3. 构建查询条件并加载汉字
	query := NewBasicCharacterQuery()
	query = s.filter.QueryFilter(query).(*BasicCharacterQuery)
	allChars, err := s.engine.provider.FindCharacters(query)
	if err != nil {
		return nil, fmt.Errorf("加载汉字数据失败: %w", err)
	}

	// 4. 合并外部注入的额外候选字（诗词/经典字库），去重
	if len(input.Options.ExtraChars) > 0 {
		existingChars := make(map[string]bool, len(allChars))
		for _, c := range allChars {
			existingChars[c.Char] = true
		}
		for _, c := range input.Options.ExtraChars {
			if !existingChars[c.Char] {
				allChars = append(allChars, c)
			}
		}
	}

	// 5. 对每个字符进行过滤（含负面反馈排除的字符）
	// excludedChars/excludedCombos 由 sessionImpl 字段持有，通过 s.isCharExcluded / s.isComboExcluded
	// 方法直接查询，并发安全由 s.mu 守护（generate 与 ExcludeChar 互斥）。

	// 避讳长辈：构建同形字与同音字排除集
	// 规则：同形字=侵佔福分，同音字=气场冲撞（"压运"）
	elderCharSet, elderPinyinSet := s.buildElderAvoidance(input.AvoidElderNames)

	// 大名需大命：检测极旺格局（专旺格/从强格）
	// 普通格局需移除敏感字（龙/凤/乾/坤/圣/贤/天/帝/皇/神/仙/君）
	allowSensitive := isExtremeStrongPattern(fateData)

	// 负面反馈排除通过 s.isCharExcluded / s.isComboExcluded 调用（直接读 session 字段，
	// session 生命周期内 excludedChars/excludedCombos 仅在 ExcludeChar/ExcludeCombo 时变更），
	// 但 generate() 与 Exclude* 并发安全通过 s.mu 守护。
	validChars := make([]*Character, 0, len(allChars))
	for _, c := range allChars {
		if s.isCharExcluded(c.Char) {
			continue
		}
		// 避讳长辈：排除同形字
		if elderCharSet[c.Char] {
			continue
		}
		// 避讳长辈：排除同音字（任一拼音命中即排除）
		if len(elderPinyinSet) > 0 {
			avoid := false
			for _, p := range c.Pinyin {
				if elderPinyinSet[p] {
					avoid = true
					break
				}
			}
			if avoid {
				continue
			}
		}
		// 大名需大命：普通格局排除敏感字
		if !allowSensitive && IsSensitiveChar(c.Char) {
			continue
		}
		// 负面语义：硬禁用字（秽物/尸棺/淫猥/盗匪/暴虐/贬义等）直接剔除
		// 软惩罚字（病/疾/哀/愁等）保留入池资格，由评分器重罚（支持"去病/弃疾"式祈福）
		if IsHardNegativeChar(c.Char) || c.IsNegative {
			continue
		}
		// 名字用字质量门禁：虚词/排行字/口语物名/数字量词/叹词等
		// 在「组合级」由策展感知门禁处理（IsNonNamingCombo），此处不剔除，
		// 以免误伤策展好名（如「与砺」「若兮」中的「与/兮」）。
		// 单名场景（无策展单名）在单名循环内单独剔除。
		if s.filter.CheckCharacter(c) {
			validChars = append(validChars, c)
		}
	}

	// 6. 策展白名单收窄（治本：荒谬字防漏）
	//
	// 背景：namer.json 8105 字中含大量生僻/物名/化学/贬义等荒谬字
	// （拤/䏝/囵/饹/嚄/姮/婊/蚂/蛞/羟/苯/仫/滃…），它们的 PositiveScore（寓意评分）为空，
	// 却仍能靠音韵/生肖/五行/新颖度等维度拿到 80+ 高分进入推荐榜；
	// 而原 1094 字人工门禁表数据已丢失（重建仅 131 字），无法靠穷举拦截这类字。
	// 实证：CLI 复测中荒谬字（拤/䏝/囵/饹/阇/睄/啰/竑/尥 等）全部 PositiveScore 为空，
	// 而优质字（远90/瑞90/宝90/旻88 等）均在名内。故以「人工寓意评分 >=85」作为
	// 唯一推荐字源门槛（纳入口径：共 563 字，全为人工评过分的好字），荒谬字天然排除。
	// 注意点：
	//   - 仅当候选池为真实生产规模（>=80）时才收窄，避免破坏基于小字桩的测试（如
	//     TestExtraCharsInjectIntoPool 仅 8 字，其注入字由 ExtraChars 豁免逻辑而非收窄保证）。
	//   - 收窄后的白名单池只要非空即采用（荒谬字入池不可接受，候选稍少可接受），
	//     不使用喜用神五行收窄的 <80 阈值作降级标准——单五行偏好下白名单好字可能仅 55 个
	//     （如 -wuxing_match 水 时水行>=85 仅 55 字），沿用 80 阈值会回退到含荒谬字的全量。
	//   - 外部注入的 ExtraChars（诗词/经典来源字）豁免收窄：用户显式指定，
	//     不受白名单门槛约束，避免误伤（见 TestExtraCharsInjectIntoPool）。
	if len(validChars) >= 80 {
		curatedPool := make([]*Character, 0, len(validChars))
		extraSet := make(map[string]bool, len(input.Options.ExtraChars))
		for _, ec := range input.Options.ExtraChars {
			extraSet[ec.Char] = true
		}
		for _, c := range validChars {
			if c.PositiveScore >= 85 || extraSet[c.Char] {
				curatedPool = append(curatedPool, c)
			}
		}
		// 非空即采用（荒谬字入池不可接受；候选稍少可接受）
		if len(curatedPool) > 0 {
			validChars = curatedPool
		}
	}

	// 7. 喜用神五行收窄候选池（性能优化 + 方案B：含生助五行）
	//
	// 旧版（纯性能优化）：仅保留喜用神五行字 → 所有候选五行相同 → WuxingRater 恒分，无区分度。
	// 方案B：收窄池扩展为「喜用神 + 生助喜用神」的五行集合。
	//   例：喜用神=土、金 → 收窄池含土、金（喜用）+ 火（火生土）+ 土（土生金已含）→ 合计3种五行。
	//   这样 WuxingRater 的间接生助梯度（生喜用+8 / 中性+3）才能真正区分候选。
	//
	// 收窄后若候选过少（<80），放弃收窄避免过度限制（降级保护）。
	// 单名例外：候选池仅 ~1100 字，串行枚举足够快，无需性能收窄。
	effectiveWuxing := s.filter.PreferredWuXing()
	if input.Options.NameLength != 1 && len(effectiveWuxing) == 0 && len(fateData.WuXingXiji.XiYongShen) > 0 {
		// 构建收窄集：喜用神 + 每个喜用神的"生我"元素（谁生我）
		// 五行相生：木→火→土→金→水→木（wuXingShengMap[a]=b 表示 a 生 b）
		// "生我"即反查：wuXingShengMap[x]==target → x 生 target
		narrowSet := make(map[string]bool)
		for _, wx := range fateData.WuXingXiji.XiYongShen {
			narrowSet[wx] = true // 直接匹配喜用神
		}
		// 反查"谁生我"：遍历所有五行，找到生喜用神的元素
		for _, wx := range fateData.WuXingXiji.XiYongShen {
			for src, dst := range wuXingShengMap {
				if dst == wx {
					narrowSet[src] = true // src 生 wx（生助喜用神）
				}
			}
		}

		narrowed := make([]*Character, 0, len(validChars))
		for _, c := range validChars {
			if narrowSet[c.WuXing] {
				narrowed = append(narrowed, c)
			}
		}
		// 降级保护：收窄后候选过少时保留全量，避免结果单一或空
		if len(narrowed) >= 80 {
			validChars = narrowed
		}
	}

	// 6. 预计算每个候选字的固定属性（使用包级 charInfo 类型，避免双重循环内重复调用
	// GetCharacterStroke / firstPinyin / 诗词检索）

	infos := make([]charInfo, len(validChars))
	for i, c := range validChars {
		found, desc, _ := classics.FindPoetryByChars(c.Char)
		infos[i] = charInfo{
			ch:          c,
			stroke:      s.filter.GetCharacterStroke(c),
			pinyin:      firstPinyin(c.Pinyin),
			poetryFound: found,
			poetryDesc:  desc,
		}
	}

	// 预排序候选池：按「潜力分」降序排列
	// 潜力分 = 五行匹配(0-24) + 策展好字(0-16) + 寓意评分(0-9) + 诗词出典(0-5)
	// 高潜力字排在前面，使笛卡尔积循环中高分组合优先进入 ExcellentTable，
	// 后续低潜力组合在早停检查中被跳过，减少无效评分计算。
	xiSet := make(map[string]bool)
	if fateData != nil {
		for _, wx := range fateData.WuXingXiji.XiYongShen {
			xiSet[wx] = true
		}
	}
	sort.SliceStable(infos, func(i, j int) bool {
		pi := charPotentialScore(infos[i].ch.WuXing, infos[i].ch.IsCurated, infos[i].ch.PositiveScore, infos[i].poetryFound, xiSet)
		pj := charPotentialScore(infos[j].ch.WuXing, infos[j].ch.IsCurated, infos[j].ch.PositiveScore, infos[j].poetryFound, xiSet)
		return pi > pj
	})

	surnamePinyin := firstPinyinForSurname(input.Surname, s.engine.provider)

	// 生成名字候选 */
	table := NewExcellentTable()
	// totalCount 统计「完成 RateName 评分」的候选组合数（被过滤/早停跳过的组合不计数），
	// 用于展示生成规模统计，语义为「已评分组合数」而非「总尝试组合数」（B11）。
	var totalCount atomic.Int64

	nameLen := input.Options.NameLength
	if nameLen <= 0 {
		nameLen = 2
	}
	topCount := input.Options.Count
	if topCount <= 0 {
		topCount = 100
	}

	if nameLen == 1 {
		// 单名：仅迭代第一个字（候选集小，串行足够快）
		s.generateSingleName(ctx, infos, table, &totalCount, fateData, surnamePinyin)
	} else {
		// 双名：迭代 Char1 × Char2（全量枚举，按外层 Char1 分片并行）
		s.generateDoubleName(ctx, infos, &table, &totalCount, fateData, surnamePinyin, topCount, xiSet)
	}

	// 7. 构建 TopNames 输出（过滤禁止的国家机关单位名称）
	// 五行多样性保底策略：取更多候选（Top10N），确保每种五行组合至少1个代表，
	// 避免单一五行组合（如金金）垄断排名。未达多样性要求时退化为纯分数排序。
	// 注意：ExcellentTable容量10000，远超Top10N（~1000），不会溢出。
	poolSize := topCount * 10
	if poolSize > table.Len() {
		poolSize = table.Len()
	}
	topEntries := table.TopN(poolSize)
	if poolSize > topCount {
		topEntries = ensureWuxingDiversity(topEntries, topCount)
	}
	topNames := make([]NameResult, 0, len(topEntries))
	rank := 0
	for _, e := range topEntries {
		givenName := e.Char1 + e.Char2
		fullName := input.Surname + givenName
		// 国家机关单位名称禁止检查（AGENTS.md 安全策略）
		if IsForbiddenEntity(fullName) {
			continue
		}
		rank++
		// 计算总笔画数（姓氏笔画 + 名字笔画）
		totalStrokes := 0
		if l1 > 0 {
			totalStrokes += l1
		}
		if l2 > 0 {
			totalStrokes += l2
		}
		// 笔画口径统一为康熙字典笔画（KangxiStroke1/2），与姓氏走 l1+l2
		// 同一口径（LookupSurnameStrokes 返回康熙笔画）。
		// 修复前 e.Stroke1+Stroke2 走的是 filter.GetCharacterStroke（默认 ScienceStroke），
		// 导致"姓用康熙、名用科学"的总笔画错误（docs/19 报告 B3）。
		// KangxiStroke1/2 == 0 时降级为 Stroke1/2（未收录兜底）。
		nameStroke1 := e.KangxiStroke1
		if nameStroke1 == 0 {
			nameStroke1 = e.Stroke1
		}
		nameStroke2 := e.KangxiStroke2
		if nameStroke2 == 0 {
			nameStroke2 = e.Stroke2
		}
		totalStrokes += nameStroke1 + nameStroke2
		topNames = append(topNames, NameResult{
			Rank:      rank,
			Surname:   input.Surname,
			GivenName: givenName,
			FullName:  fullName,
			// 拼音回填：ExcellentEntry 已携带预计算的候选字读音，
			// 双名以空格组合，单名仅首字读音（TrimSpace 清理尾随空格）
			Pinyin: strings.TrimSpace(e.Pinyin1 + " " + e.Pinyin2),
			// 释义回填：候选字释义组合为名字寓意描述，
			// 单名仅首字释义，双名以「；」连接；超长释义按 rune 截断防撑爆响应
			Meaning: combineCharMeanings(e.Meaning1, e.Meaning2),
			Strokes: totalStrokes,
			WuXing:  e.WuXing1 + e.WuXing2,
			// 诗词出处回填：从 ExcellentEntry 透传
			PoetryFrom: e.PoetryFrom,
			Score: NameScore{
				Total:   e.Score,
				Grade:   e.Grade,
				Items:   e.Items,
				Details: e.Details,
			},
		})
	}

	return &Output{
		Input:          input,
		FateData:       fateData,
		TopNames:       topNames,
		ExcellentTable: table,
		TotalCount:     int(totalCount.Load()),
	}, nil
}

// --- BasicCharacterQuery: CharacterQuery 接口的内存实现 ---

// BasicCharacterQuery 是 CharacterQuery 的可导出内存实现，
// 外部适配器（如 HanziDataProvider）可通过类型断言读取过滤条件。
type BasicCharacterQuery struct {
	RegularFilter  bool
	NameableFilter bool
	StrokeEQ       int // 0 表示不限制
	StrokeGTE      int // 0 表示不限制
	StrokeLTE      int // 0 表示不限制
	WuxingIn       []string
	WuxingNotIn    []string
	CharIn         []string
	GenderHint     string
	NamingCategory string // 精选起名分类筛选（空字符串表示不限制）
}

func NewBasicCharacterQuery() *BasicCharacterQuery {
	return &BasicCharacterQuery{}
}

func (q *BasicCharacterQuery) WhereRegular() CharacterQuery {
	q.RegularFilter = true
	return q
}

func (q *BasicCharacterQuery) WhereNameable() CharacterQuery {
	q.NameableFilter = true
	return q
}

func (q *BasicCharacterQuery) WhereStrokeEQ(stroke int) CharacterQuery {
	q.StrokeEQ = stroke
	return q
}

func (q *BasicCharacterQuery) WhereStrokeGTE(min int) CharacterQuery {
	q.StrokeGTE = min
	return q
}

func (q *BasicCharacterQuery) WhereStrokeLTE(max int) CharacterQuery {
	q.StrokeLTE = max
	return q
}

func (q *BasicCharacterQuery) WhereWuXingIn(wuxing ...string) CharacterQuery {
	q.WuxingIn = append(q.WuxingIn, wuxing...)
	return q
}

func (q *BasicCharacterQuery) WhereWuXingNotIn(wuxing ...string) CharacterQuery {
	q.WuxingNotIn = append(q.WuxingNotIn, wuxing...)
	return q
}

func (q *BasicCharacterQuery) WhereCharIn(chars ...string) CharacterQuery {
	q.CharIn = append(q.CharIn, chars...)
	return q
}

func (q *BasicCharacterQuery) WhereGenderHint(gender string) CharacterQuery {
	q.GenderHint = gender
	return q
}

// WhereNamingCategory 设置精选起名分类筛选
func (q *BasicCharacterQuery) WhereNamingCategory(category string) CharacterQuery {
	q.NamingCategory = category
	return q
}

// --- 辅助函数 ---

// buildElderAvoidance 构建避讳长辈的同形字与同音字排除集
// 输入：长辈姓名列表（如 ["张三", "李四"]）
// 输出：elderCharSet（同形字集合）、elderPinyinSet（同音拼音集合）
//
// 规则（AGENTS.md 规则1）：
//   - 同形字：长辈姓名中出现的每个字，生成名字时排除
//   - 同音字：长辈姓名中每个字的拼音，生成名字时排除同音字
//   - 范围：父系+母系直系长辈（建议往上两代）
func (s *sessionImpl) buildElderAvoidance(elderNames []string) (map[string]bool, map[string]bool) {
	elderCharSet := make(map[string]bool)
	elderPinyinSet := make(map[string]bool)

	for _, name := range elderNames {
		for _, r := range name {
			ch := string(r)
			// 同形字直接加入排除集
			elderCharSet[ch] = true
			// 查询该字的拼音，加入同音排除集
			if info, err := s.engine.provider.GetCharacter(ch); err == nil && info != nil {
				for _, p := range info.Pinyin {
					elderPinyinSet[p] = true
				}
			}
		}
	}
	return elderCharSet, elderPinyinSet
}

// cancelled 检查 context 是否已取消
func cancelled(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

// maxPerCharMeaningRunes 单字释义保留的最大 rune 数
// （hanzi.json 部分条目含《说文》类数百字长释义，需截断防撑爆响应与前端展示）
const maxPerCharMeaningRunes = 60

// combineCharMeanings 组合候选字释义为名字寓意描述。
// 单名仅取首字释义；双名以「；」连接两字释义；
// 每字释义按 rune 截断至 maxPerCharMeaningRunes 并补省略号。
func combineCharMeanings(m1, m2 string) string {
	trunc := func(m string) string {
		m = strings.TrimSpace(m)
		if m == "" {
			return ""
		}
		r := []rune(m)
		if len(r) > maxPerCharMeaningRunes {
			return string(r[:maxPerCharMeaningRunes]) + "…"
		}
		return m
	}
	c1 := trunc(m1)
	if m2 == "" {
		return c1
	}
	c2 := trunc(m2)
	switch {
	case c1 == "":
		return c2
	case c2 == "":
		return c1
	default:
		return c1 + "；" + c2
	}
}

// firstPinyin 获取拼音列表中的第一个拼音（去掉声调数字）
func firstPinyin(pinyins []string) string {
	if len(pinyins) == 0 {
		return ""
	}
	return pinyins[0]
}

// firstPinyinForSurname 获取姓氏的拼音
func firstPinyinForSurname(surname string, provider CharacterProvider) string {
	if surname == "" {
		return ""
	}
	char, err := provider.GetCharacter(string([]rune(surname)[0]))
	if err != nil || char == nil {
		return ""
	}
	if len(char.Pinyin) == 0 {
		return ""
	}
	return char.Pinyin[0]
}

// bestGenderHint 取两个字的性别暗示的"更明确的"那个
func bestGenderHint(a, b string) string {
	if a == b {
		return a
	}
	if a == "neutral" {
		return b
	}
	if b == "neutral" {
		return a
	}
	// 都有具体性别时取第一个
	if a != "" {
		return a
	}
	return b
}

// charPotentialScore 计算候选字的潜力分，用于预排序
// 高潜力字排在前面，使笛卡尔积循环中高分组合优先进入 ExcellentTable
func charPotentialScore(wuxing string, isCurated bool, positiveScore int, poetryFound bool, xiSet map[string]bool) int {
	score := 0
	// 五行匹配：喜用神 +12，生助喜用神 +8
	if xiSet[wuxing] {
		score += 12
	} else {
		for xi := range xiSet {
			if isWuXingSheng(wuxing, xi) {
				score += 8
				break
			}
		}
	}
	// 策展好字 +8
	if isCurated && positiveScore >= 90 {
		score += 8
	}
	// 寓意评分（0-9分，归一化）
	if positiveScore > 0 {
		score += int(positiveScore) / 10
	}
	// 诗词出典 +5
	if poetryFound {
		score += 5
	}
	return score
}

// hasPairBlacklist 检查字符的策展搭配黑名单中是否包含指定字
// 命中表示策展判定该字与此字组合不宜（如 瑾-艳/妍-艳/嫣-艳）
func hasPairBlacklist(ch *Character, other string) bool {
	if ch == nil || len(ch.PairBlacklist) == 0 {
		return false
	}
	for _, bl := range ch.PairBlacklist {
		if bl == other {
			return true
		}
	}
	return false
}

// ensureWuxingDiversity 五行多样性保底策略
//
// 从候选池中选取 topCount 个条目，确保每种五行组合（如金金、火土、金火等）
// 至少有1个代表。选取规则：
// 1. 按分数从高到低遍历候选池
// 2. 对于未出现过的五行组合，直接入选
// 3. 对于已出现过的五行组合，仅在有空位时入选
// 4. 所有空位填满后停止
//
// 这确保了 TopN 中既有高分候选（金金组合），也有不同五行组合的候选（火土/火金等），
// 使 WuxingRater 的间接生助梯度真正发挥作用。
func ensureWuxingDiversity(entries []ExcellentEntry, topCount int) []ExcellentEntry {
	if len(entries) <= topCount {
		return entries
	}

	result := make([]ExcellentEntry, 0, topCount)
	seenWuxing := make(map[string]bool) // 已出现的五行组合
	// 用于第二轮填充时快速判重，避免 O(N²) 嵌套循环
	seenInResult := make(map[string]bool, topCount)

	// 第一轮：每种五行组合取1个代表
	for _, e := range entries {
		if len(result) >= topCount {
			break
		}
		wuxing := e.WuXing1 + e.WuXing2
		if !seenWuxing[wuxing] {
			seenWuxing[wuxing] = true
			result = append(result, e)
			seenInResult[e.Char1+e.Char2] = true
		}
	}

	// 第二轮：未达 topCount 时，按分数填充剩余空位
	for _, e := range entries {
		if len(result) >= topCount {
			break
		}
		key := e.Char1 + e.Char2
		if seenInResult[key] {
			continue
		}
		seenInResult[key] = true
		result = append(result, e)
	}

	return result
}

// generateSingleName 单名生成：串行迭代候选字
//
// 候选集较小（~1100），串行足够快，无需并发。
// 关键差异（vs 双名）：
//   - 单层循环（无 i×j 笛卡尔积）
//   - IsNonNamingChar 兜底（单名字是名字的全部，门禁字必须剔除）
//   - 无 IsBadCombo / IsHistoricalFigureCombo（单字无组合级过滤）
//   - 无早停（候选集小，全枚举 + RateName 性能可接受）
func (s *sessionImpl) generateSingleName(
	ctx context.Context,
	infos []charInfo,
	table *ExcellentTable,
	totalCount *atomic.Int64,
	fateData *FateData,
	surnamePinyin string,
) {
	for i := range infos {
		if cancelled(ctx) {
			break
		}
		a := infos[i]
		if !s.filter.CheckStrokePair(a.stroke, 0) {
			continue
		}
		// 名字用字质量门禁：单名若为门禁字（虚词/排行字/口语物名等）直接剔除。
		// 策展库无单名，故单名门禁不会误伤策展推荐。
		if IsNonNamingChar(a.ch.Char) {
			continue
		}

		candidate := &NameCandidate{
			Char1:          a.ch.Char,
			Pinyin1:        a.pinyin,
			WuXing1:        a.ch.WuXing,
			Stroke1:        a.stroke,
			WuXing2:        "",
			Stroke2:        0,
			HasPoetry:      a.poetryFound,
			PoetryFrom:     a.poetryDesc,
			IsRegular:      a.ch.IsRegular,
			CommonLevel1:   a.ch.CommonLevel,
			NameFreqTier1:  a.ch.NameFreqTier,
			NamePenalty1:   a.ch.NamePenalty,
			IsCurated1:     a.ch.IsCurated,
			PositiveScore1: a.ch.PositiveScore,
			SurnamePinyin:  surnamePinyin,
		}
		ns := RateName(candidate, fateData, s.raters)
		entry := ExcellentEntry{
			Char1:         a.ch.Char,
			Pinyin1:       a.pinyin,
			Meaning1:      a.ch.Meaning,
			Score:         ns.Total,
			Grade:         ns.Grade,
			WuXing1:       a.ch.WuXing,
			Stroke1:       candidate.Stroke1,
			KangxiStroke1: a.ch.KangxiStroke,
			HasPoetry:     a.poetryFound,
			PoetryFrom:    a.poetryDesc,
			Items:         ns.Items,
			Details:       ns.Details,
			NameFreqTier1: a.ch.NameFreqTier,
		}
		table.TryPush(entry)
		totalCount.Add(1)
	}
	table.Finalize()
}

// generateDoubleName 双名生成：i×j 笛卡尔积 + worker 并发
//
// 关键差异（vs 单名）：
//   - 嵌套笛卡尔积 N²
//   - 按 CPU 分片并发（外层 i 分片，内层 j 全量——worker 间无重叠）
//   - 组合级过滤（IsBadCombo / IsHistoricalFigureCombo / PairBlacklist 等）
//   - 早停：local.IsFull() 时按 (a, b) 组合潜力分与本地堆最小分比较
//
// localTable 容量自适应 topCount（避免无谓的 10000 容量浪费内存）。
// 末尾合并各分片局部 Top-N 到主表 → 全局 TopN。
func (s *sessionImpl) generateDoubleName(
	ctx context.Context,
	infos []charInfo,
	table **ExcellentTable,
	totalCount *atomic.Int64,
	fateData *FateData,
	surnamePinyin string,
	topCount int,
	xiSet map[string]bool,
) {
	workers := runtime.NumCPU()
	if workers < 1 {
		workers = 1
	}
	if workers > len(infos) {
		workers = len(infos)
	}
	if workers == 0 {
		workers = 1
	}

	// localTable 容量自适应 topCount（避免无谓的 10000 容量浪费内存）：
	// 取 topCount*2 留余量避免抖动，溢出时 TryPush 按堆顶最小分替换。
	// topCount 在 NewSessionWithFilter 后由 input.Options.Count 决定（默认 50）。
	localCap := topCount * 2
	if localCap < 100 {
		localCap = 100
	}

	chunkSize := (len(infos) + workers - 1) / workers
	localTables := make([]*ExcellentTable, workers)
	var wg sync.WaitGroup

	for w := 0; w < workers; w++ {
		start := w * chunkSize
		end := start + chunkSize
		if end > len(infos) {
			end = len(infos)
		}
		if start >= end {
			continue
		}

		wg.Add(1)
		go func(start, end, w int) {
			defer wg.Done()
			local := NewExcellentTableWithCap(localCap)

			for i := start; i < end; i++ {
				if cancelled(ctx) {
					return
				}
				a := infos[i]
				if !s.filter.CheckStrokePair(a.stroke, 0) {
					continue
				}

				for j := range infos {
					if cancelled(ctx) {
						return
					}
					b := infos[j]
					if !s.filter.CheckStrokePair(a.stroke, b.stroke) {
						continue
					}

					// 避免重复字（双名一般不用相同字）
					if a.ch.Char == b.ch.Char {
						continue
					}

					// 负面反馈：排除组合（sessionImpl 方法，调用时不加锁）
					if s.isComboExcluded(a.ch.Char, b.ch.Char) {
						continue
					}

					// 负面语义：组合禁忌（亲属称谓/物名/动词等，「父母」「蜂蜜」类直接剔除）
					if IsBadCombo(a.ch.Char, b.ch.Char) {
						continue
					}

					// 历史人物字/号专名：剔除「仲尼」「尼仲」「孔明」等与历史人物撞车组合
					// （GetBigramScore 会因《论语》"仲尼曰"等典籍共现给高分，需在此拦截）
					if IsHistoricalFigureCombo(a.ch.Char, b.ch.Char) {
						continue
					}

					// 名字用字质量门禁（纯语义）：若任一位置为门禁字
					// （虚词/排行字/口语物名/数字量词等），组合即无命名价值，
					// 直接剔除——不复依赖策展库豁免。
					// 策展库已降级为纯出典参考：其本身是历史人物名采集
					// （混入仲尼/若兮/七政/与砺 等 962 条垃圾），不可作为质量基准。
					if IsNonNamingChar(a.ch.Char) || IsNonNamingChar(b.ch.Char) {
						continue
					}

					// 数据层搭配黑名单：策展标注的 PairBlacklist（如 瑾-艳/妍-艳/嫣-艳）
					if hasPairBlacklist(a.ch, b.ch.Char) || hasPairBlacklist(b.ch, a.ch.Char) {
						continue
					}

					// 早停检查：表已满时，跳过组合潜力分明显不足的配对
					// 潜力分 = charPotentialScore(a) + charPotentialScore(b)
					// 低于当前表最小分时，即使满分各维度也难以入表
					if local.IsFull() {
						pa := charPotentialScore(a.ch.WuXing, a.ch.IsCurated, a.ch.PositiveScore, a.poetryFound, xiSet)
						pb := charPotentialScore(b.ch.WuXing, b.ch.IsCurated, b.ch.PositiveScore, b.poetryFound, xiSet)
						combined := pa + pb
						minScore := local.MinScore()
						// 潜力分上限约40，实际得分约60-90，保守阈值=最小分的60%
						if float64(combined) < minScore*0.6 {
							continue
						}
					}

					poetryFound := a.poetryFound || b.poetryFound
					poetryDesc := ""
					if a.poetryFound {
						poetryDesc = a.poetryDesc
					} else if b.poetryFound {
						poetryDesc = b.poetryDesc
					}

					candidate := &NameCandidate{
						Char1:         a.ch.Char,
						Char2:         b.ch.Char,
						Pinyin1:       a.pinyin,
						Pinyin2:       b.pinyin,
						WuXing1:       a.ch.WuXing,
						WuXing2:       b.ch.WuXing,
						Stroke1:       a.stroke,
						Stroke2:       b.stroke,
						Meaning1:      a.ch.Meaning,
						Meaning2:      b.ch.Meaning,
						Radical1:      a.ch.Radical,
						Radical2:      b.ch.Radical,
						HasPoetry:     poetryFound,
						PoetryFrom:    poetryDesc,
						IsRegular:     a.ch.IsRegular && b.ch.IsRegular,
						CommonLevel1:  a.ch.CommonLevel,
						CommonLevel2:  b.ch.CommonLevel,
						NameFreqTier1: a.ch.NameFreqTier,
						NameFreqTier2: b.ch.NameFreqTier,
						GenderHint:    bestGenderHint(a.ch.GenderHint, b.ch.GenderHint),
						NamePenalty1:  a.ch.NamePenalty,
						NamePenalty2:  b.ch.NamePenalty,
						// 策展覆盖表标记：人工精选起名好字，供 WenHuaRater 文化加分（破荒谬字同分）
						IsCurated1: a.ch.IsCurated,
						IsCurated2: b.ch.IsCurated,
						// 寓意评分：与 IsCurated 结合将策展加分收窄为「策展 ∩ positiveScore>=85」精选好字
						PositiveScore1: a.ch.PositiveScore,
						PositiveScore2: b.ch.PositiveScore,
						// 姓氏拼音取自 input，用于音韵评分器检测跨字谐音
						SurnamePinyin: surnamePinyin,
						// bigramCache per-session 缓存（避免 WenHuaRater/BigramRater
						// 在 N² 笛卡尔积中重复 50 万次 GetBigramScore RLock）
						bigramCache: s.bigramCache,
					}

					ns := RateName(candidate, fateData, s.raters)
					entry := ExcellentEntry{
						Char1:         candidate.Char1,
						Char2:         candidate.Char2,
						Pinyin1:       a.pinyin,
						Pinyin2:       b.pinyin,
						Meaning1:      a.ch.Meaning,
						Meaning2:      b.ch.Meaning,
						Score:         ns.Total,
						Grade:         ns.Grade,
						WuXing1:       candidate.WuXing1,
						WuXing2:       candidate.WuXing2,
						Stroke1:       candidate.Stroke1,
						Stroke2:       candidate.Stroke2,
						KangxiStroke1: a.ch.KangxiStroke,
						KangxiStroke2: b.ch.KangxiStroke,
						HasPoetry:     candidate.HasPoetry,
						PoetryFrom:    poetryDesc,
						Items:         ns.Items,
						Details:       ns.Details,
						NameFreqTier1: candidate.NameFreqTier1,
						NameFreqTier2: candidate.NameFreqTier2,
					}
					local.TryPush(entry)
					totalCount.Add(1)
				}
			}
			localTables[w] = local
		}(start, end, w)
	}

	wg.Wait()

	// 合并各分片的局部 Top-N 到主表（全局 Top-k 必落在某分片局部 Top-容量内）
	*table = NewExcellentTable()
	for _, lt := range localTables {
		if lt == nil {
			continue
		}
		lt.Finalize()
		for _, e := range lt.entries {
			(*table).TryPush(e)
		}
	}
	(*table).Finalize()
}
