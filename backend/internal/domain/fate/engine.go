package fate

import (
	"context"
	"fmt"
	"sync"
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
		if s.filter.CheckCharacter(c) {
			validChars = append(validChars, c)
		}
	}

	// 6. 生成名字候选
	table := NewExcellentTable()
	totalCount := 0

	nameLen := input.Options.NameLength
	if nameLen <= 0 {
		nameLen = 2
	}
	topCount := input.Options.Count
	if topCount <= 0 {
		topCount = 100
	}

	if nameLen == 1 {
		// 单名：仅迭代第一个字
		for _, c1 := range validChars {
			if cancelled(ctx) {
				break
			}
			stroke1 := s.filter.GetCharacterStroke(c1)
			if !s.filter.CheckStrokePair(stroke1, 0) {
				continue
			}

			poetryFound, poetryDesc, _ := classics.FindPoetryByChars(c1.Char)
			candidate := &NameCandidate{
				Char1:      c1.Char,
				Pinyin1:    firstPinyin(c1.Pinyin),
				WuXing1:    c1.WuXing,
				Stroke1:    stroke1,
				WuXing2:    "",
				Stroke2:    0,
				HasPoetry:  poetryFound,
				PoetryFrom: poetryDesc,
				IsRegular:  c1.IsRegular,
				SurnamePinyin: firstPinyinForSurname(input.Surname, s.engine.provider),
			}
			ns := RateName(candidate, fateData, s.raters)
			entry := ExcellentEntry{
				Char1:     c1.Char,
				Score:     ns.Total,
				Grade:     ns.Grade,
				WuXing1:   c1.WuXing,
				Stroke1:   candidate.Stroke1,
				HasPoetry: poetryFound,
				Items:     ns.Items,
			}
			table.TryPush(entry)
			totalCount++
		}
	} else {
		// 双名：迭代 Char1 × Char2
		for _, c1 := range validChars {
			if cancelled(ctx) {
				break
			}
			stroke1 := s.filter.GetCharacterStroke(c1)
			if !s.filter.CheckStrokePair(stroke1, 0) {
				continue
			}

			for _, c2 := range validChars {
				if cancelled(ctx) {
					goto done
				}
				stroke2 := s.filter.GetCharacterStroke(c2)
				if !s.filter.CheckStrokePair(stroke1, stroke2) {
					continue
				}

				// 避免重复字（双名一般不用相同字）
				if c1.Char == c2.Char {
					continue
				}

				// 负面反馈：排除组合（使用快照，无需加锁）
				if isComboExcluded(c1.Char, c2.Char) {
					continue
				}

				poetryFound, poetryDesc, _ := classics.FindPoetryByChars(c1.Char, c2.Char)
				candidate := &NameCandidate{
					Char1:      c1.Char,
					Char2:      c2.Char,
					Pinyin1:    firstPinyin(c1.Pinyin),
					Pinyin2:    firstPinyin(c2.Pinyin),
					WuXing1:    c1.WuXing,
					WuXing2:    c2.WuXing,
					Stroke1:    stroke1,
					Stroke2:    stroke2,
					Meaning1:   c1.Meaning,
					Meaning2:   c2.Meaning,
					Radical1:   c1.Radical,
					Radical2:   c2.Radical,
					HasPoetry:  poetryFound,
					PoetryFrom: poetryDesc,
					IsRegular:  c1.IsRegular && c2.IsRegular,
					GenderHint: bestGenderHint(c1.GenderHint, c2.GenderHint),
					// 姓氏拼音取自 input，用于音韵评分器检测跨字谐音
					SurnamePinyin: firstPinyinForSurname(input.Surname, s.engine.provider),
				}

				ns := RateName(candidate, fateData, s.raters)
				entry := ExcellentEntry{
					Char1:     candidate.Char1,
					Char2:     candidate.Char2,
					Score:     ns.Total,
					Grade:     ns.Grade,
					WuXing1:   candidate.WuXing1,
					WuXing2:   candidate.WuXing2,
					Stroke1:   candidate.Stroke1,
					Stroke2:   candidate.Stroke2,
					HasPoetry: candidate.HasPoetry,
					Items:     ns.Items,
				}
				table.TryPush(entry)
				totalCount++
			}
		}
	}
done:
	table.Finalize()

	// 7. 构建 TopNames 输出（过滤禁止的国家机关单位名称）
	topEntries := table.TopN(topCount)
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
			Pinyin:    "", // 拼音由 pinyin 字段组合，ExcellentEntry 不存储，后续由 provider 补充
			Strokes:   totalStrokes,
			WuXing:    e.WuXing1 + e.WuXing2,
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
		TotalCount:     totalCount,
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


