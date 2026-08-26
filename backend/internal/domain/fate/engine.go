package fate

import (
	"context"
	"fmt"
	"runtime"
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

	// 负面反馈
	excludedChars   map[string]bool       // 排除的字符
	excludedCombos  map[string]bool       // 排除的组合 "char1+char2"（排序后）
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

// isCharExcluded 检查字符是否被排除（内部方法，不加锁，调用者需持有锁）
func (s *sessionImpl) isCharExcluded(char string) bool {
	if s.excludedChars == nil {
		return false
	}
	return s.excludedChars[char]
}

// isComboExcluded 检查字符组合是否被排除（内部方法，不加锁）
func (s *sessionImpl) isComboExcluded(c1, c2 string) bool {
	if s.excludedCombos == nil {
		return false
	}
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
	// 仅在读取 excludedChars/excludedCombos 时短暂加锁，复制快照后立即释放，
	// 避免长时间持锁阻塞 State()/Result()/Stop()/ExcludeChar() 等方法。
	s.mu.Lock()
	excludedCharsSnapshot := make(map[string]bool, len(s.excludedChars))
	for k, v := range s.excludedChars {
		excludedCharsSnapshot[k] = v
	}
	excludedCombosSnapshot := make(map[string]bool, len(s.excludedCombos))
	for k, v := range s.excludedCombos {
		excludedCombosSnapshot[k] = v
	}
	s.mu.Unlock()

	// 避讳长辈：构建同形字与同音字排除集
	// 规则：同形字=侵佔福分，同音字=气场冲撞（"压运"）
	elderCharSet, elderPinyinSet := s.buildElderAvoidance(input.AvoidElderNames)

	// 大名需大命：检测极旺格局（专旺格/从强格）
	// 普通格局需移除敏感字（龙/凤/乾/坤/圣/贤/天/帝/皇/神/仙/君）
	allowSensitive := isExtremeStrongPattern(fateData)

	isCharExcluded := func(char string) bool {
		return excludedCharsSnapshot[char]
	}
	isComboExcluded := func(c1, c2 string) bool {
		a, b := c1, c2
		if a > b {
			a, b = b, a
		}
		return excludedCombosSnapshot[a+"+"+b]
	}

	validChars := make([]*Character, 0, len(allChars))
	for _, c := range allChars {
		if isCharExcluded(c.Char) {
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

	// 6. 喜用神五行收窄候选池（性能优化 + 方案B：含生助五行）
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

	// 6. 预计算每个候选字的固定属性
	// 避免双重循环内重复调用 GetCharacterStroke / firstPinyin / 诗词检索
	type charInfo struct {
		ch          *Character
		stroke      int
		pinyin      string
		poetryFound bool
		poetryDesc  string
	}

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

	surnamePinyin := firstPinyinForSurname(input.Surname, s.engine.provider)

	// 生成名字候选 */
	table := NewExcellentTable()
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
				Char1:      a.ch.Char,
				Pinyin1:    a.pinyin,
				WuXing1:    a.ch.WuXing,
				Stroke1:    a.stroke,
				WuXing2:    "",
				Stroke2:    0,
				HasPoetry:  a.poetryFound,
				PoetryFrom: a.poetryDesc,
				IsRegular:  a.ch.IsRegular,
				CommonLevel1: a.ch.CommonLevel,
				NamePenalty1: a.ch.NamePenalty,
				IsCurated1: a.ch.IsCurated,
				PositiveScore1: a.ch.PositiveScore,
				SurnamePinyin: surnamePinyin,
			}
			ns := RateName(candidate, fateData, s.raters)
			entry := ExcellentEntry{
				Char1:      a.ch.Char,
				Pinyin1:    a.pinyin,
				Meaning1:   a.ch.Meaning,
				Score:      ns.Total,
				Grade:      ns.Grade,
				WuXing1:    a.ch.WuXing,
				Stroke1:    candidate.Stroke1,
				HasPoetry:  a.poetryFound,
				PoetryFrom: a.poetryDesc,
				Items:      ns.Items,
			}
			table.TryPush(entry)
			totalCount.Add(1)
		}
		table.Finalize()
	} else {
		// 双名：迭代 Char1 × Char2（全量枚举，按外层 Char1 分片并行）
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
				local := NewExcellentTable()

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

						// 负面反馈：排除组合（使用快照，无需加锁）
						if isComboExcluded(a.ch.Char, b.ch.Char) {
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

						poetryFound := a.poetryFound || b.poetryFound
						poetryDesc := ""
						if a.poetryFound {
							poetryDesc = a.poetryDesc
						} else if b.poetryFound {
							poetryDesc = b.poetryDesc
						}

						candidate := &NameCandidate{
							Char1:      a.ch.Char,
							Char2:      b.ch.Char,
							Pinyin1:    a.pinyin,
							Pinyin2:    b.pinyin,
							WuXing1:    a.ch.WuXing,
							WuXing2:    b.ch.WuXing,
							Stroke1:    a.stroke,
							Stroke2:    b.stroke,
							Meaning1:   a.ch.Meaning,
							Meaning2:   b.ch.Meaning,
							Radical1:   a.ch.Radical,
							Radical2:   b.ch.Radical,
							HasPoetry:  poetryFound,
							PoetryFrom: poetryDesc,
							IsRegular:  a.ch.IsRegular && b.ch.IsRegular,
							CommonLevel1: a.ch.CommonLevel,
							CommonLevel2: b.ch.CommonLevel,
							GenderHint: bestGenderHint(a.ch.GenderHint, b.ch.GenderHint),
							NamePenalty1: a.ch.NamePenalty,
							NamePenalty2: b.ch.NamePenalty,
							// 策展覆盖表标记：人工精选起名好字，供 WenHuaRater 文化加分（破荒谬字同分）
							IsCurated1: a.ch.IsCurated,
							IsCurated2: b.ch.IsCurated,
							// 寓意评分：与 IsCurated 结合将策展加分收窄为「策展 ∩ positiveScore>=85」精选好字
							PositiveScore1: a.ch.PositiveScore,
							PositiveScore2: b.ch.PositiveScore,
							// 姓氏拼音取自 input，用于音韵评分器检测跨字谐音
							SurnamePinyin: surnamePinyin,
						}

						ns := RateName(candidate, fateData, s.raters)
						entry := ExcellentEntry{
							Char1:      candidate.Char1,
							Char2:      candidate.Char2,
							Pinyin1:    a.pinyin,
							Pinyin2:    b.pinyin,
							Meaning1:   a.ch.Meaning,
							Meaning2:   b.ch.Meaning,
							Score:      ns.Total,
							Grade:      ns.Grade,
							WuXing1:    candidate.WuXing1,
							WuXing2:    candidate.WuXing2,
							Stroke1:    candidate.Stroke1,
							Stroke2:    candidate.Stroke2,
							HasPoetry:  candidate.HasPoetry,
							PoetryFrom: poetryDesc,
							Items:      ns.Items,
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
		table = NewExcellentTable()
		for _, lt := range localTables {
			if lt == nil {
				continue
			}
			lt.Finalize()
			for _, e := range lt.entries {
				table.TryPush(e)
			}
		}
		table.Finalize()
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
		// 从 ExcellentEntry 中获取笔画（存储在排名的条目中）
		// 注意：单名时 Stroke2 为 0，不累加
		totalStrokes += e.Stroke1 + e.Stroke2
		topNames = append(topNames, NameResult{
			Rank:      rank,
			Surname:   input.Surname,
			GivenName: givenName,
			FullName:  fullName,
			// 拼音回填：ExcellentEntry 已携带预计算的候选字读音，
			// 双名以空格组合，单名仅首字读音（TrimSpace 清理尾随空格）
			Pinyin:    strings.TrimSpace(e.Pinyin1 + " " + e.Pinyin2),
			// 释义回填：候选字释义组合为名字寓意描述，
			// 单名仅首字释义，双名以「；」连接；超长释义按 rune 截断防撑爆响应
			Meaning:   combineCharMeanings(e.Meaning1, e.Meaning2),
			Strokes:   totalStrokes,
			WuXing:    e.WuXing1 + e.WuXing2,
			// 诗词出处回填：从 ExcellentEntry 透传
			PoetryFrom: e.PoetryFrom,
			Score: NameScore{
				Total: e.Score,
				Grade: e.Grade,
				Items: e.Items,
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
	RegularFilter   bool
	NameableFilter  bool
	StrokeEQ        int     // 0 表示不限制
	StrokeGTE       int     // 0 表示不限制
	StrokeLTE       int     // 0 表示不限制
	WuxingIn        []string
	WuxingNotIn     []string
	CharIn          []string
	GenderHint      string
	NamingCategory  string // 精选起名分类筛选（空字符串表示不限制）
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

	// 第一轮：每种五行组合取1个代表
	for _, e := range entries {
		if len(result) >= topCount {
			break
		}
		wuxing := e.WuXing1 + e.WuXing2
		if !seenWuxing[wuxing] {
			seenWuxing[wuxing] = true
			result = append(result, e)
		}
	}

	// 第二轮：未达 topCount 时，按分数填充剩余空位
	for _, e := range entries {
		if len(result) >= topCount {
			break
		}
		// 检查是否已在结果中（按 Char1+Char2 去重）
		dup := false
		for _, r := range result {
			if r.Char1 == e.Char1 && r.Char2 == e.Char2 {
				dup = true
				break
			}
		}
		if !dup {
			result = append(result, e)
		}
	}

	return result
}


