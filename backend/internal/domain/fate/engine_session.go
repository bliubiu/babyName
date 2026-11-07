package fate

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"name/internal/domain/yue"
)

// --- sessionImpl: Session 接口默认实现 ---

type sessionImpl struct {
	// mu 守护 state/input/output/err 以及负面反馈排除集的**可写副本**。
	//
	// 排除集的**读路径**（isCharExcluded/isComboExcluded）不走此锁，而是读取
	// atomic.Pointer 指向的不可变快照 exclusionSnap（见该字段说明），因此要
	// 修改 s.excludedChars/s.excludedCombos 就必须同时重建快照，切勿只改 map。
	mu     sync.RWMutex
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

	// 粤语评分（开启 EnableCantonese 时装配）
	enableCantonese bool    // 本会话是否开启粤语评分
	cantoneseRaters []Rater // 粤语评分器（DefaultCantoneseRaters）
	surnameJyutping string  // 姓氏粤拼（预计算，供候选组装注入）

	// 负面反馈
	//
	// excludedChars/excludedCombos 是**可写副本**，仅在持有 s.mu 写锁时被修改；
	// 每次修改都会同步重建一份不可变快照发布到 exclusionSnap。
	excludedChars  map[string]bool // 排除的字符
	excludedCombos map[string]bool // 排除的组合 "char1+char2"（排序后）

	// exclusionSnap 指向排除集的**不可变只读快照**（*exclusionSnapshot）。
	//
	// 为什么不直接对 map 加锁读：isComboExcluded 位于双名 N×N 笛卡尔积的最内层
	// 循环，真实数据下每次生成会被调用数百万次；即便 RWMutex 的读锁只是原子
	// 计数，量级足够大时仍会显现（实测串行吞吐损失约 20%）。
	//
	// 快照方案把读路径降为「一次原子指针加载 + 一次 map 查找」，无锁无竞争；
	// 写路径（ExcludeChar/ExcludeCombo/ClearExclusions，调用次数为个位数量级）
	// 在写锁内 copy-on-write 重建快照——排除集通常只有几个元素，重建成本可忽略。
	//
	// 语义：nil 快照等价于「空排除集」，因此 Start() 之前无需初始化。
	exclusionSnap atomic.Pointer[exclusionSnapshot]

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
	// 排除集**不在此重置**：接口契约（fate.Session 的 ExcludeChar 注释）要求
	// 「后续重新生成时所有包含该字的候选将被过滤」，即排除项需跨多次 Start 保持。
	//
	// ★ 历史缺陷（docs/28 W4 连带发现）：此处原本无条件
	//     s.excludedChars = make(map[string]bool)
	//     s.excludedCombos = make(map[string]bool)
	// 把调用方在 Start 之前设置的排除项全部抹掉，导致 ExcludeChar/ExcludeCombo
	// 在「先从界面排除、再重新生成」这一唯一合理用法下**完全失效**
	// （实测：ExcludeChar("渝") 后生成，TopNames 中仍出现 12 个含「渝」的名字）。
	//
	// 空 map 的初始化改由 ExcludeChar/ExcludeCombo 内部按需完成（两者已有
	// nil 判断），ClearExclusions 负责显式清空。
	//
	// 排除集为空时 exclusionSnap 保持 nil，读路径据此直接返回 false，
	// 因此无需在此为「无排除项」的常规请求付出任何快照开销。
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
	// 粤语评分装配（per-request）：开启 EnableCantonese 时注入粤语评分器与姓氏粤拼。
	// 粤拼标注（Jyutping1/2/SurnameJyutping）在候选组装阶段无条件预填（见
	// singleNameCandidate/doubleNameCandidate），但粤语评分仅在开启开关时参与聚合。
	if input.Options.EnableCantonese {
		s.enableCantonese = true
		s.cantoneseRaters = DefaultCantoneseRaters()
		s.surnameJyutping = yue.PrimaryJyutping(input.Surname)
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

// exclusionSnapshot 是排除集的不可变副本。
// 一旦由 rebuildExclusionSnapshot 发布，其内容不再被任何代码修改；
// 读方拿到指针后可安全地在无锁状态下遍历（只读 map 无数据竞争）。
type exclusionSnapshot struct {
	chars  map[string]bool
	combos map[string]bool
}

// rebuildExclusionSnapshotLocked 在持有 s.mu 写锁的前提下，
// 把当前可写 map 复制为一份新的不可变快照并原子发布。
//
// 必须「先复制、后发布」：若直接把 s.excludedChars 放进快照，写方后续对
// map 的写入就会命中读方正在遍历的同一份 map，重现 concurrent map read
// and map write。
func (s *sessionImpl) rebuildExclusionSnapshotLocked() {
	snap := &exclusionSnapshot{}
	if n := len(s.excludedChars); n > 0 {
		snap.chars = make(map[string]bool, n)
		for k, v := range s.excludedChars {
			snap.chars[k] = v
		}
	}
	if n := len(s.excludedCombos); n > 0 {
		snap.combos = make(map[string]bool, n)
		for k, v := range s.excludedCombos {
			snap.combos[k] = v
		}
	}
	s.exclusionSnap.Store(snap)
}

func (s *sessionImpl) ExcludeChar(char string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.excludedChars == nil {
		s.excludedChars = make(map[string]bool)
	}
	s.excludedChars[char] = true
	s.rebuildExclusionSnapshotLocked()
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
	s.rebuildExclusionSnapshotLocked()
}

func (s *sessionImpl) ClearExclusions() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.excludedChars = make(map[string]bool)
	s.excludedCombos = make(map[string]bool)
	s.rebuildExclusionSnapshotLocked()
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

// sessionImpl 上的负反馈查询：**全链路无锁**。
//
// ★ 历史缺陷（docs/28 W4）三连：
//
//	① 这两处曾不加锁直接读 map，并附注释声称「并发安全由 s.mu 守护
//	   （generate 与 ExcludeChar 互斥）」——该论断是错的：generate() 全程
//	   并不持有 s.mu（见 generate 入口），因此「生成进行中调用 ExcludeChar」
//	   会与这里的读形成 concurrent map read and map write，Go 运行时直接抛
//	   不可 recover 的 fatal error 终止整个进程。
//	② 首版修复改为逐次 RLock，正确但代价高：这两个函数位于 N×N 笛卡尔积的
//	   最内层循环，调用数百万次，读锁的原子计数开销实测拖慢串行吞吐约 20%。
//	③ 现改为读取不可变快照（atomic.Pointer），读路径无锁无竞争，
//	   写路径 copy-on-write 重建快照——既消除数据竞争，又比 RLock 更快。
//
// 时序语义：指针加载发生在**本次调用**，因此生成进行中调用 ExcludeChar 后，
// 尚未评估的组合会立刻看到新的排除集（比 RLock 版本更及时）。
func (s *sessionImpl) isCharExcluded(char string) bool {
	snap := s.exclusionSnap.Load()
	if snap == nil {
		return false
	}
	return snap.chars[char]
}

func (s *sessionImpl) isComboExcluded(c1, c2 string) bool {
	snap := s.exclusionSnap.Load()
	if snap == nil || len(snap.combos) == 0 {
		return false
	}
	a, b := c1, c2
	if a > b {
		a, b = b, a
	}
	return snap.combos[a+"+"+b]
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
		// 取消/超时与真正的生成错误在语义上不同：前者是调用方主动中止
		// （或超时兜底），会话应标记为 Canceled 而非 Failed，
		// 便于上层区分「该重试」与「该放弃/改参数」（docs/28 W3）。
		if ctx.Err() != nil {
			s.state = SessionStateCanceled
		} else {
			s.state = SessionStateFailed
		}
		s.err = err
		return
	}
	s.output = output
	s.state = SessionStateFinish
}
