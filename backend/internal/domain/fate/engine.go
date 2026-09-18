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

// candidateWorkerLimit 双重枚举 worker 的全局并发上限（信号量）。
//
// 历史问题（P2-7）：generateDoubleName 每个请求都按 runtime.NumCPU() 固定分片，
// 并发请求数 × 16 个 worker 全部在跑，16 核机床上 8 并发就有 128 个 goroutine
// 抢占 CPU，单请求耗时线性恶化、吞吐在低并发就封顶。
//
// 现在把「worker 并发总量」下沉到进程级：无论同时在飞多少请求，所有请求的
// 双重枚举 worker 合计不超过 CPU 核数（channel 容量即令牌数）。先到先占，
// 请求之间按到达顺序共享这份 CPU，避免了 N 请求各自开 16 线程的超订放大。
//
// 变量设计为包级可替换（:=）而不是 const，是为了让测试能注入小容量信号量，
// 如实模拟并发放大场景断言「worker 总量受控」（见 engine_concurrency_test.go）。
var candidateWorkerLimit = make(chan struct{}, runtime.NumCPU())

// candidateWorkerActive 当前正在执行双重枚举的 worker 数（并发钳制观测值）。
var candidateWorkerActive atomic.Int64

// candidateWorkerPeak 双重枚举 worker 并发峰值（P2-7 回归观测：任何时刻都不应
// 超过 candidateWorkerLimit 容量）。仅用于测试断言，生产路径零额外开销。
var candidateWorkerPeak atomic.Int64

// workerCountHint 计算双重枚举的 worker 规划数：
// 仍按 CPU 核数切分候选池以获得均匀分片，但实际并发由 candidateWorkerLimit 钳制。
func workerCountHint(n int) int {
	workers := runtime.NumCPU()
	if workers < 1 {
		workers = 1
	}
	if workers > n {
		workers = n
	}
	if workers == 0 {
		workers = 1
	}
	return workers
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
//   - ch:            候选字 Character
//   - stroke:        笔画数（filter 口径，按 StrokeMode 决定）
//   - pinyin:        第一拼音（无声调）
//   - poetryFound:   是否关联到诗词
//   - poetryDesc:    诗词出处描述
//   - meaningProfile: 释义字符画像（供 NoveltyRater 的字义重叠 O(|A|+|B|) 归并）
//
// 避免双重循环内重复调用 GetCharacterStroke / firstPinyin / 诗词检索 / 释义解析。
type charInfo struct {
	ch             *Character
	stroke         int
	pinyin         string
	poetryFound    bool
	poetryDesc     string
	meaningProfile *meaningProfile
}

// sessionStage 会话进度阶段（供异步任务的进度查询）
const (
	sessionStageBazi     int32 = iota // 八字排盘
	sessionStageLoadChars             // 加载字库
	sessionStagePool                  // 构建候选池
	sessionStageScore                 // 评分筛选（笛卡尔积枚举）
	sessionStageRank                  // 汇总排序
)

// sessionStageNames 阶段中文名（对外展示用）
var sessionStageNames = map[int32]string{
	sessionStageBazi:     "八字排盘",
	sessionStageLoadChars: "加载字库",
	sessionStagePool:      "构建候选池",
	sessionStageScore:     "评分筛选",
	sessionStageRank:      "汇总排序",
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

	// 进度上报（异步任务轮询用）。
	// stage 为阶段下标（sessionStage*），percent 为百分比 ×100 的整数
	// （避免浮点原子操作）；两者只在 generate 流程内写入、查询侧原子读取。
	stage    atomic.Int32
	percentX atomic.Int64
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

	// 经典来源偏好（per-request 装配）：
	// engine.raters 构造期固定（一次性 DefaultRaters），而经典来源是请求级参数。
	// 此处把文化印象（WenHuaRater）替换为带来源字集的版本，与 resolveExtraChars
	// 的候选池注入联动，实现"选论语则结果偏论语"。未指定时保持默认，无来源加分。
	if src := input.Options.SourceClassic; src != "" {
		for i, r := range s.raters {
			if wr, ok := r.(*WenHuaRater); ok {
				s.raters[i] = NewWenHuaRaterWithSource(wr.weight, src)
				break
			}
		}
	}
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

// Progress 返回当前生成阶段名与完成百分比（0-100）。
// 实现 ProgressReporter 可选接口，供异步任务轮询（type 断言获取，不影响既有调用方）。
func (s *sessionImpl) Progress() (string, float64) {
	stage := sessionStageNames[s.stage.Load()]
	return stage, float64(s.percentX.Load()) / 100
}

// setStage 更新进度阶段与百分比（percent 单位 %）
func (s *sessionImpl) setStage(stage int32, percent float64) {
	s.stage.Store(stage)
	s.percentX.Store(int64(percent * 100))
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
	s.setStage(sessionStageBazi, 2)

	// 1. 八字分析
	fateData, err := s.engine.analyzer.Analyze(input.Born, input.Gender)
	if err != nil {
		return nil, fmt.Errorf("八字分析失败: %w", err)
	}
	s.setStage(sessionStageBazi, 8)

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

	// 6. 全放行《通用规范汉字表》字源
	//
	// 历史：曾用「人工寓意评分>=85」作为唯一推荐字源门槛，把候选池收窄到
	// 200-360 字，导致多次生成名字高度近似。且 namer.json 8105 字中仅 563 字
	// 被人工评分（93% 无评分），门槛信号极度稀缺；再叠加五行收窄后往往只剩
	// 几十字可组合。
	//
	// 现改为「全放行」：filter 各层（性别/笔画/禁用字/负面语义/生僻字/门禁表）
	// 是唯一字源防线，荒谬字由以下多层防线兜底，不再用评分门槛一刀切：
	//   - 字级：IsHardNegativeChar / IsNegative / CheckCharacter 已剔除硬负面与门禁字
	//   - 组合级：策展感知质量门禁（IsNonNamingCombo）、禁忌组合（IsBadCombo）
	//   - 评分级：WenHuaRater 仅对「策展∩评分>=90」加分，PositiveScore 空无文化分；
	//     非策展双名四维封顶 75，荒谬组合天然低分掉榜
	// 入选检查由 engine_curated_pool_test.go 改为在真实数据规模上断言推荐榜无荒谬字。
	//

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

	// 7.1 候选池按汉字去重
	//
	// 重复汉字会让 (i, j) 组合在「名字」层面重复——同一 Char1+Char2 被枚举多次。
	// 历史上由 ExcellentTable 的 seen map 兜底去重（首次写入获胜）；而枚举期的
	// worker 局部表为省下每次 push 的字符串拼接已改为不去重（NewExcellentTableUnique），
	// 因此必须在池层保证组合语义唯一。保留首次出现，与历史上「seen 首次写入获胜」
	// 的结果完全一致（同字不同下标构造出的候选字段相同、得分相同）。
	//
	// 生产数据中候选池本身按字唯一，此处为外部注入（ExtraChars）与上游数据异常的兜底；
	// 顺带避免对重复字做无谓的重复枚举。
	validChars = dedupCharsByName(validChars)

	// 6. 预计算每个候选字的固定属性（使用包级 charInfo 类型，避免双重循环内重复调用
	// GetCharacterStroke / firstPinyin / 诗词检索）

	infos := make([]charInfo, len(validChars))
	for i, c := range validChars {
		found, desc, _ := classics.FindPoetryByChars(c.Char)
		infos[i] = charInfo{
			ch:             c,
			stroke:         s.filter.GetCharacterStroke(c),
			pinyin:         firstPinyin(c.Pinyin),
			poetryFound:    found,
			poetryDesc:     desc,
			meaningProfile: meaningProfileOf(c.Meaning),
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
		s.setStage(sessionStageScore, 30)
		s.generateSingleName(ctx, infos, table, &totalCount, fateData, surnamePinyin)
	} else {
		// 双名：迭代 Char1 × Char2（全量枚举，按外层 Char1 分片并行）
		s.generateDoubleName(ctx, infos, &table, &totalCount, fateData, surnamePinyin, topCount, xiSet)
	}

	// 7. 构建 TopNames 输出（过滤禁止的国家机关单位名称）
	s.setStage(sessionStageRank, 90)
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

	// 6.1 惰性明细回算（评分与解释分离）
	//
	// 枚举阶段只算总分（RateNameScore），不构造各维度的 Items/Details 与
	// 解释文案；此处只对真正进入推荐榜的条目（≤ poolSize 条，约千分之一量级）
	// 用 RateName 回算一次完整明细，填回条目供下游透传。
	// 回算用独立候选（skipDetail=false），不影响枚举期的复用候选。
	s.fillEntryDetails(topEntries, infos, fateData, surnamePinyin)
	s.setStage(sessionStageRank, 97)

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

	s.setStage(sessionStageRank, 100)
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

// dedupCharsByName 按汉字去重候选池，保留首次出现。
//
// 保证 (i, j) 枚举出来的组合在「名字」层面唯一，使 worker 局部表可以
// 安全地不做 seen 去重（见 generateDoubleName 中的 NewExcellentTableUnique）。
func dedupCharsByName(chars []*Character) []*Character {
	if len(chars) < 2 {
		return chars
	}
	seen := make(map[string]struct{}, len(chars))
	out := make([]*Character, 0, len(chars))
	for _, c := range chars {
		if _, dup := seen[c.Char]; dup {
			continue
		}
		seen[c.Char] = struct{}{}
		out = append(out, c)
	}
	return out
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

// singleNameCandidate 组装单名的 NameCandidate
//
// 字段与历史上两处内联构造逐一对齐（单名不注入 Meaning1/Radical/GenderHint/
// bigramCache，以免无意改变 WenHuaRater 的「字义明确 +4」等分支结果）。
// 抽成函数是为了让「枚举期评分」与「入榜后回算明细」走同一条构造路径。
func singleNameCandidate(a charInfo, surnamePinyin string) *NameCandidate {
	return &NameCandidate{
		Char1:          a.ch.Char,
		Pinyin1:        a.pinyin,
		WuXing1:        a.ch.WuXing,
		Stroke1:        a.stroke,
		HasPoetry:      a.poetryFound,
		PoetryFrom:     a.poetryDesc,
		IsRegular:      a.ch.IsRegular,
		CommonLevel1:   a.ch.CommonLevel,
		NameFreqTier1:  a.ch.NameFreqTier,
		NamePenalty1:   a.ch.NamePenalty,
		IsCurated1:     a.ch.IsCurated,
		PositiveScore1: a.ch.PositiveScore,
		SurnamePinyin:  surnamePinyin,
		// 释义画像：单名无名2，仅注入字1画像（NoveltyRater 单名分支不比较重叠）
		MeaningProfile1: a.meaningProfile,
	}
}

// doubleNameCandidate 组装双名的 NameCandidate
//
// poetryFound/poetryDesc 由调用方按「甲字||乙字任一有出典」预先合并后传入
// （与引擎内层循环的既有语义一致）。
func doubleNameCandidate(
	a, b charInfo, poetryFound bool, poetryDesc, surnamePinyin string, bigramCache *SessionBigramCache,
) *NameCandidate {
	return &NameCandidate{
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
		// 释义画像：按候选字预计算，消除 N² 次释义字符串扫描
		MeaningProfile1: a.meaningProfile,
		MeaningProfile2: b.meaningProfile,
		// bigramCache per-session 缓存（避免 WenHuaRater/BigramRater
		// 在 N² 笛卡尔积中重复 50 万次 GetBigramScore RLock）
		bigramCache: bigramCache,
	}
}

// fillEntryDetails 为进入推荐榜的条目回算各维度得分与依据文案
//
// 评分与解释分离：枚举阶段只算总分（RateNameScore），Items/Details 留空；
// 本函数按条目携带的候选字下标（idx1/idx2）从 infos 还原 NameCandidate，
// 再调用完整 RateName 回算一次明细，填回条目供服务层透传。
//
// 代价可忽略：条目数 ≤ topCount*10（约 500~1000），相对 N² 枚举量级约千分之一。
// 回算结果的总分与枚举期一致（两条路径共用封顶判据与取整逻辑，
// 由 TestRateNameScoreMatchesRateName 断言保证）。
func (s *sessionImpl) fillEntryDetails(
	entries []ExcellentEntry, infos []charInfo, fateData *FateData, surnamePinyin string,
) {
	for k := range entries {
		e := &entries[k]
		if e.idx1 < 0 || e.idx1 >= len(infos) {
			continue
		}
		a := infos[e.idx1]
		var cand *NameCandidate
		if e.idx2 >= 0 && e.idx2 < len(infos) {
			b := infos[e.idx2]
			poetryFound := a.poetryFound || b.poetryFound
			poetryDesc := ""
			if a.poetryFound {
				poetryDesc = a.poetryDesc
			} else if b.poetryFound {
				poetryDesc = b.poetryDesc
			}
			cand = doubleNameCandidate(a, b, poetryFound, poetryDesc, surnamePinyin, s.bigramCache)
		} else {
			cand = singleNameCandidate(a, surnamePinyin)
		}
		ns := RateName(cand, fateData, s.raters)
		e.Items = ns.Items
		e.Details = ns.Details
	}
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
		// 进度：单名串行，30% 起步按外层推进到 85%
		s.percentX.Store(int64((30 + 55*float64(i)/float64(max(len(infos), 1))) * 100))
		a := infos[i]
		if !s.filter.CheckStrokePair(a.stroke, 0) {
			continue
		}
		// 名字用字质量门禁：单名若为门禁字（虚词/排行字/口语物名等）直接剔除。
		// 策展库无单名，故单名门禁不会误伤策展推荐。
		if IsNonNamingChar(a.ch.Char) {
			continue
		}

		// 枚举期只算总分：不构造 Items/Details 与各维度解释文案
		// （评分与解释分离，入榜后再按 idx1 回算完整明细）。
		total := RateNameScore(singleNameCandidate(a, surnamePinyin), fateData, s.raters)
		entry := ExcellentEntry{
			Char1:         a.ch.Char,
			Pinyin1:       a.pinyin,
			Meaning1:      a.ch.Meaning,
			Score:         total,
			Grade:         scoreToGrade(total),
			WuXing1:       a.ch.WuXing,
			Stroke1:       a.stroke,
			KangxiStroke1: a.ch.KangxiStroke,
			HasPoetry:     a.poetryFound,
			PoetryFrom:    a.poetryDesc,
			NameFreqTier1: a.ch.NameFreqTier,
			idx1:          i,
			idx2:          -1,
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
	workers := workerCountHint(len(infos))

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

	// 进度上报：所有 worker 共享一个外层完成计数器（0 → len(infos)），
	// 按推进比例把评分阶段映射到 30%–85%。只写 percentX，不改阶段。
	var outerDone atomic.Int64

	for w := 0; w < workers; w++ {
		start := w * chunkSize
		end := start + chunkSize
		if end > len(infos) {
			end = len(infos)
		}
		if start >= end {
			continue
		}

		// 获取全局 worker 令牌：总量 = CPU 核数，跨请求共享（P2-7）。
		// 高并发下后来的 worker 在此排队，最多等 CPU 核数个在跑，杜绝
		// 「并发请求数 × NumCPU」的超订放大。ctx 取消时直接放弃令牌占用，
		// 避免积累无谓的排队位。
		select {
		case candidateWorkerLimit <- struct{}{}:
		case <-ctx.Done():
			return
		}

		wg.Add(1)
		go func(start, end, w int) {
			defer wg.Done()
			defer func() { <-candidateWorkerLimit }()
			// 并发钳制观测：本 worker 进入枚举前记录活跃计数与峰值，
			// 退出时递减。生产为零开销的原子递增，仅峰值恒定供测试断言。
			cur := candidateWorkerActive.Add(1)
			defer candidateWorkerActive.Add(-1)
			for {
				peak := candidateWorkerPeak.Load()
				if cur <= peak || candidateWorkerPeak.CompareAndSwap(peak, cur) {
					break
				}
			}
			// worker 局部表：本分片的 (i, j) 组合天然唯一，无需 seen 去重
			// （省下每个条目一次「两个汉字拼接成字符串 + map 写入」的分配）。
			local := NewExcellentTableUnique(localCap)

			// 潜力分预计算：内层循环每轮都要用，改为按字预算成数组。
			// 原实现每次组合都重算 a、b 两个字的潜力分，其中 a 的部分与内层
			// 下标 j 无关，属于纯重复计算（N² 次里算了 N³ 量级的一半）。
			potential := make([]int, len(infos))
			for k := range infos {
				potential[k] = charPotentialScore(
					infos[k].ch.WuXing, infos[k].ch.IsCurated,
					infos[k].ch.PositiveScore, infos[k].poetryFound, xiSet)
			}

			for i := start; i < end; i++ {
				if cancelled(ctx) {
					return
				}
				a := infos[i]
				if !s.filter.CheckStrokePair(a.stroke, 0) {
					continue
				}
				pa := potential[i]

				for j := range infos {
					if cancelled(ctx) {
						return
					}
					b := infos[j]

					// 早停前置：表已满时，潜力分明显不足的组合不可能最终入榜，
					// 直接跳过后续「组合级门禁 + 八维评分」，避免为注定淘汰的组合
					// 反复做字符串判重与谐音检索。
					//
					// 语义等价：原顺序为「门禁 → 早停 → 评分」，被早停跳过的组合
					// 本就不会产生任何结果，互换不改变实际被评分的组合集合，
					// 也不改变 totalCount（它只统计完成 RateName 的组合）。
					if cutoff := local.earlyStopCutoff(); cutoff > 0 &&
						float64(pa+potential[j]) < cutoff {
						continue
					}

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

					poetryFound := a.poetryFound || b.poetryFound
					poetryDesc := ""
					if a.poetryFound {
						poetryDesc = a.poetryDesc
					} else if b.poetryFound {
						poetryDesc = b.poetryDesc
					}

					// 枚举期只算总分：不构造 Items/Details 与各维度解释文案
					// （评分与解释分离，入榜后再按 idx1/idx2 回算完整明细）。
					total := RateNameScore(doubleNameCandidate(a, b, poetryFound, poetryDesc, surnamePinyin, s.bigramCache), fateData, s.raters)
					entry := ExcellentEntry{
						Char1:         a.ch.Char,
						Char2:         b.ch.Char,
						Pinyin1:       a.pinyin,
						Pinyin2:       b.pinyin,
						Meaning1:      a.ch.Meaning,
						Meaning2:      b.ch.Meaning,
						Score:         total,
						Grade:         scoreToGrade(total),
						WuXing1:       a.ch.WuXing,
						WuXing2:       b.ch.WuXing,
						Stroke1:       a.stroke,
						Stroke2:       b.stroke,
						KangxiStroke1: a.ch.KangxiStroke,
						KangxiStroke2: b.ch.KangxiStroke,
						HasPoetry:     poetryFound,
						PoetryFrom:    poetryDesc,
						NameFreqTier1: a.ch.NameFreqTier,
						NameFreqTier2: b.ch.NameFreqTier,
						idx1:          i,
						idx2:          j,
					}
					local.TryPush(entry)
					totalCount.Add(1)
				}
				// 外层 i 完成一步：共享计数器推进评分阶段进度（30%–85%）
				done := outerDone.Add(1)
				s.percentX.Store(int64((30 + 55*float64(done)/float64(max(len(infos), 1))) * 100))
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
