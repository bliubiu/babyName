package fate

import (
	"runtime"
	"sync/atomic"
	"time"
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
	sessionStageBazi      int32 = iota // 八字排盘
	sessionStageLoadChars              // 加载字库
	sessionStagePool                   // 构建候选池
	sessionStageScore                  // 评分筛选（笛卡尔积枚举）
	sessionStageRank                   // 汇总排序
)

// sessionStageNames 阶段中文名（对外展示用）
var sessionStageNames = map[int32]string{
	sessionStageBazi:      "八字排盘",
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
