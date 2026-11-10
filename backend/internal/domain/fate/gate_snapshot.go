package fate

// gate_snapshot.go 门禁层动态集合的不可变只读快照（docs/46 P2 性能）
//
// 背景：IsBadCombo / IsHistoricalFigureCombo / IsNonNamingChar 位于双名
// N×N 笛卡尔积的最内层循环，真实数据下每次生成被调用数百万次。改造前
// 每个组合要执行约 7 次 RWMutex.RLock/RUnlock；项目已在
// sessionImpl.exclusionSnap 实测「读锁在数百万次调用下拖慢串行吞吐约 20%」
// 并用 atomic.Pointer 快照消除，门禁层沿用同一模式。
//
// 设计：三组动态集合（白名单 / 动态禁忌组合 / 命名质量门禁字）都是
// 「启动期物料装载一次、写路径整体覆盖式重建、读路径极高频」——把它们的
// 只读副本集中进一个 gateSnapshot，用单个 atomic.Pointer 发布：
//   - 读路径：一次原子指针加载 + map 查找，无锁无竞争；
//   - 写路径：gateWriteMu 串行化，copy-on-write 浅拷贝快照后只替换本次
//     变更的 map 字段（其余字段复用旧快照的不可变 map 指针），原子发布。
//
// 不可变约定：发布后的 gateSnapshot 及其 map 不再被任何代码修改。setter
// 必须新建 map 后整体替换字段，禁止原地增删键值——否则读方会命中
// concurrent map read and map write。
//
// nil 快照等价于「全部集合为空」，因此物料装载前无需初始化；这也保证
// 物料缺失降级路径（各 LoadXxxFromJSON 的空集合）行为与改造前一致。

import (
	"sync"
	"sync/atomic"
)

// gateSnapshot 门禁层全部动态集合的不可变副本。
type gateSnapshot struct {
	// allowlist.go：白名单（字 / 组合 / 祈福 / 原串）
	allowChars    map[string]bool
	allowCombos   map[string]bool
	blessingExtra map[string]bool
	allowEntries  map[string]bool
	// semantic_filter.go：动态禁忌组合
	forbiddenExtra map[string]bool
	// naming_quality.go：命名质量门禁字
	nonNaming map[string]bool
}

var (
	gateSnap    atomic.Pointer[gateSnapshot]
	gateWriteMu sync.Mutex

	// emptyGateSnapshot 是 nil 快照的读侧替身，让读路径无需判空分支。
	emptyGateSnapshot = &gateSnapshot{}
)

// currentGateSnapshot 返回当前不可变快照；尚未装载物料时返回空快照。
//
// 读路径只做一次原子指针加载，禁止在此加任何锁：本函数位于 N² 最内层。
func currentGateSnapshot() *gateSnapshot {
	if s := gateSnap.Load(); s != nil {
		return s
	}
	return emptyGateSnapshot
}

// updateGateSnapshot 以 copy-on-write 方式发布新快照。
//
// 写路径调用次数为个位数（启动期物料装载 / SQLite 物料注入），加锁无性能
// 影响；mutate 只允许整体替换自己负责的 map 字段，不得原地修改旧 map。
func updateGateSnapshot(mutate func(*gateSnapshot)) {
	gateWriteMu.Lock()
	defer gateWriteMu.Unlock()

	old := currentGateSnapshot()
	next := *old // 浅拷贝：未变更字段仍指向旧快照的不可变 map
	mutate(&next)
	gateSnap.Store(&next)
}
