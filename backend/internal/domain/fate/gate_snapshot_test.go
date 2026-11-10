package fate

import (
	"sync"
	"testing"
)

// gate_snapshot_test.go 守护门禁层不可变快照（docs/46 P2 免锁改造）
//
// 既有 allowlist_test.go / compliance_redline_test.go 已覆盖查询语义等价性
// （白名单优先、正反序、历史人物豁免、祈福并入内置默认），本文件只补新快照
// 结构特有的三类风险：
//  ① COW 跨 setter 字段不丢失（浅拷贝只替换本次变更字段）；
//  ② 幂等重载与空集合回落；
//  ③ 并发读 + 覆盖式重载无数据竞争/崩溃。
//
// 注意：门禁字表/禁忌组合表由 TestMain 从 data 装载，同包其他用例依赖它，
// 因此不能调用 setter 清空来还原状态，必须整体保存并恢复快照指针。

// saveGateSnapshot 保存并在用例结束后原样恢复整个快照指针。
func saveGateSnapshot(t *testing.T) {
	t.Helper()
	old := gateSnap.Load()
	t.Cleanup(func() { gateSnap.Store(old) })
}

// 生僻字组合：不混入硬编码 forbiddenCombos / historicalFigureCombos /
// blessingCombos，保证断言只反映动态集合的行为。
const (
	testCharA = "龘"
	testCharB = "靐"
	testCharC = "齉"
	testCharD = "爩"
)

// TestGateSnapshot_COWKeepsOtherFields 依次调用三个 setter，先写入的字段不能丢。
func TestGateSnapshot_COWKeepsOtherFields(t *testing.T) {
	saveGateSnapshot(t)

	SetAllowlist([]AllowEntry{
		{Entry: testCharA, Kind: AllowKindChar},
		{Entry: testCharB + testCharC, Kind: AllowKindCombo},
		{Entry: testCharD + testCharA, Kind: AllowKindBlessing},
	})
	SetForbiddenCombos([]string{testCharA + testCharB})
	SetNonNamingChars([]string{testCharB})

	chars, combos, blessings := AllowlistCounts()
	if chars != 1 || combos != 1 || blessings != 1 {
		t.Fatalf("COW 后白名单计数漂移：chars=%d combos=%d blessings=%d", chars, combos, blessings)
	}
	if !IsAllowlistedChar(testCharA) {
		t.Error("COW 后字级白名单丢失")
	}
	if !IsAllowlistedCombo(testCharB, testCharC) {
		t.Error("COW 后组合级白名单丢失")
	}
	if !IsAllowlistedText(testCharB + testCharC) {
		t.Error("COW 后原串白名单丢失")
	}
	if !IsBlessingCombo(testCharD, testCharA) {
		t.Error("COW 后动态祈福组合丢失")
	}
	if !IsBadCombo(testCharA, testCharB) {
		t.Error("COW 后动态禁忌组合未命中（白名单只含 靐齉，不应豁免 龘靐）")
	}
	if !IsBadCombo(testCharB, testCharA) {
		t.Error("动态禁忌组合反序未命中")
	}
	if !IsNonNamingChar(testCharB) {
		t.Error("COW 后门禁字丢失")
	}
	if ForbiddenComboCount() != 1 || NonNamingCharCount() != 1 {
		t.Errorf("计数漂移：forbidden=%d nonNaming=%d", ForbiddenComboCount(), NonNamingCharCount())
	}
}

// TestGateSnapshot_ReloadIsIdempotent 重复重载同数据不改变结果；空集合覆盖
// 只清掉自己负责的字段，其余字段保持；全部清空后回落到空门禁。
func TestGateSnapshot_ReloadIsIdempotent(t *testing.T) {
	saveGateSnapshot(t)

	SetAllowlist([]AllowEntry{{Entry: testCharA, Kind: AllowKindChar}})
	SetForbiddenCombos([]string{testCharA + testCharB})
	SetNonNamingChars([]string{testCharB})

	// 幂等：同数据重复发布
	SetForbiddenCombos([]string{testCharA + testCharB})
	if ForbiddenComboCount() != 1 || !IsBadCombo(testCharA, testCharB) {
		t.Fatal("重复重载同数据后结果变化（COW 发布异常）")
	}

	// 空集合覆盖：只清 forbiddenExtra，白名单与门禁字仍在
	SetForbiddenCombos(nil)
	if ForbiddenComboCount() != 0 {
		t.Error("空集合应把动态禁忌计数清零")
	}
	if !IsAllowlistedChar(testCharA) {
		t.Error("清空禁忌集合误伤了白名单字段")
	}
	if !IsNonNamingChar(testCharB) {
		t.Error("清空禁忌集合误伤了门禁字字段")
	}

	// 全部清空回落
	SetAllowlist(nil)
	SetNonNamingChars(nil)
	if chars, combos, blessings := AllowlistCounts(); chars != 0 || combos != 0 || blessings != 0 {
		t.Errorf("清空后白名单计数应全为 0，实得 %d/%d/%d", chars, combos, blessings)
	}
	if IsAllowlistedChar(testCharA) || IsNonNamingChar(testCharB) {
		t.Error("清空后查询应回落为 false")
	}
	if NonNamingCharCount() != 0 {
		t.Error("清空后门禁字计数应为 0")
	}
}

// TestGateSnapshot_ConcurrentReadAndReload 并发读热路径 + 反复 COW 重载。
//
// 本机无法跑 -race，但 Go 运行时对 concurrent map read and map write 会抛
// 不可 recover 的 fatal error，因此本用例仍能捕获「读方与写方共享同一 map」
// 这一最严重回归；写方始终以相同数据重载，读方结果恒定、结束态自洽。
func TestGateSnapshot_ConcurrentReadAndReload(t *testing.T) {
	saveGateSnapshot(t)

	SetAllowlist([]AllowEntry{{Entry: testCharA, Kind: AllowKindChar}})
	SetForbiddenCombos([]string{testCharA + testCharB})
	SetNonNamingChars([]string{testCharB})

	const readers = 8
	const readIterations = 2000
	const writeIterations = 200

	var wg sync.WaitGroup
	wg.Add(readers + 1)

	for i := 0; i < readers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < readIterations; j++ {
				_ = IsBadCombo(testCharA, testCharB)
				_ = IsHistoricalFigureCombo("仲", "尼")
				_ = IsNonNamingChar(testCharB)
				_ = IsAllowlistedChar(testCharA)
				_ = IsAllowlistedText(testCharA)
				_ = ForbiddenComboCount()
				_ = NonNamingCharCount()
			}
		}()
	}

	go func() {
		defer wg.Done()
		for j := 0; j < writeIterations; j++ {
			SetForbiddenCombos([]string{testCharA + testCharB})
			SetAllowlist([]AllowEntry{{Entry: testCharA, Kind: AllowKindChar}})
			SetNonNamingChars([]string{testCharB})
		}
	}()

	wg.Wait()

	if !IsBadCombo(testCharA, testCharB) || !IsNonNamingChar(testCharB) || !IsAllowlistedChar(testCharA) {
		t.Error("并发重载结束后快照状态不自洽")
	}
	if ForbiddenComboCount() != 1 || NonNamingCharCount() != 1 {
		t.Errorf("并发重载结束后计数漂移：forbidden=%d nonNaming=%d",
			ForbiddenComboCount(), NonNamingCharCount())
	}
}
