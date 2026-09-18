package classics

// shici_ready_test.go — 诗词异步加载「不会永久挂死」回归
//
// 背景（docs/24 P2-9）：classic_loader.go 的加载协程写成
//
//	go func() {
//	    if err := loadShiCiFromJSON(dataDir); err != nil { logger.Error(...) }
//	    close(shiciReady)   // ← 未 defer，且 goroutine 内无 recover
//	}()
//
// 而 ensureShiCiLoaded() 是无超时的 `<-shiciReady`。两个后果：
//   - 加载中 panic ⇒ 进程直接被未 recover 的 panic 打崩；
//   - 即便外层 recover，通道也永不关闭 ⇒ 所有依赖诗词的请求（诗词出处、
//     共现评分、候选池加字）全部挂死，表现为「服务假死」而非报错。
//
// 治理：加载协程改为 recover + defer close（runShiCiLoad），
// 等待侧改为有界等待（waitShiCiLoaded），超时即降级并告警。
//
// 说明：这里只测两个纯函数（入参注入通道/加载器），不碰包级 shiciReady——
// 它是 sync.Once 保护的全局单例，测试里无法安全重置，直接测会互相污染。

import (
	"errors"
	"testing"
	"time"
)

// TestWaitShiCiLoaded_ClosedChannel 通道已关闭 → 立即返回 true
func TestWaitShiCiLoaded_ClosedChannel(t *testing.T) {
	ready := make(chan struct{})
	close(ready)

	start := time.Now()
	if !waitShiCiLoaded(ready, time.Second) {
		t.Fatal("已关闭的通道应返回 true")
	}
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Errorf("已关闭通道不应等待，实际耗时 %v", elapsed)
	}
}

// TestWaitShiCiLoaded_Timeout 通道永不关闭 → 在超时点返回 false（而不是永久阻塞）
func TestWaitShiCiLoaded_Timeout(t *testing.T) {
	ready := make(chan struct{}) // 永不关闭：模拟加载协程异常退出

	const timeout = 30 * time.Millisecond
	start := time.Now()
	if waitShiCiLoaded(ready, timeout) {
		t.Fatal("未关闭的通道不应返回 true")
	}
	elapsed := time.Since(start)
	if elapsed < timeout {
		t.Errorf("提前返回: 耗时 %v < 超时 %v", elapsed, timeout)
	}
	if elapsed > timeout*10 {
		t.Errorf("超时未生效: 耗时 %v，期望约 %v", elapsed, timeout)
	}
}

// TestWaitShiCiLoaded_NonPositiveTimeoutWaitsForClose
// timeout <= 0 表示「不限时等待」，此时必须等到通道关闭才返回
func TestWaitShiCiLoaded_NonPositiveTimeoutWaitsForClose(t *testing.T) {
	ready := make(chan struct{})
	done := make(chan bool, 1)

	go func() { done <- waitShiCiLoaded(ready, 0) }()

	// 未关闭前不应返回
	select {
	case <-done:
		t.Fatal("通道未关闭且不限时，不应提前返回")
	case <-time.After(50 * time.Millisecond):
	}

	close(ready)
	select {
	case ok := <-done:
		if !ok {
			t.Error("通道关闭后应返回 true")
		}
	case <-time.After(time.Second):
		t.Fatal("通道关闭后仍未返回")
	}
}

// TestRunShiCiLoad_ClosesChannelOnPanic ★ 核心不变量
//
// 加载器 panic 时：通道仍须被关闭（否则 ensureShiCiLoaded 死等），
// 且 panic 不得逸出（goroutine 内未 recover 的 panic 会打崩整个进程）。
func TestRunShiCiLoad_ClosesChannelOnPanic(t *testing.T) {
	ready := make(chan struct{})

	runShiCiLoad(func() error { panic("模拟 shici.json 解析 panic") }, ready)

	select {
	case <-ready:
	default:
		t.Fatal("panic 后通道未被关闭 → ensureShiCiLoaded 会永久挂死")
	}
}

// TestRunShiCiLoad_ClosesChannelOnError 加载报错时通道同样关闭
func TestRunShiCiLoad_ClosesChannelOnError(t *testing.T) {
	ready := make(chan struct{})

	runShiCiLoad(func() error { return errors.New("读取 shici.json 失败") }, ready)

	select {
	case <-ready:
	default:
		t.Fatal("加载失败后通道未被关闭 → 等待方会永久挂死")
	}
}

// TestRunShiCiLoad_ClosesChannelOnSuccess 正常路径关闭通道
func TestRunShiCiLoad_ClosesChannelOnSuccess(t *testing.T) {
	ready := make(chan struct{})
	called := false

	runShiCiLoad(func() error { called = true; return nil }, ready)

	if !called {
		t.Error("加载器未被调用")
	}
	select {
	case <-ready:
	default:
		t.Fatal("加载成功后通道未被关闭")
	}
}

// TestRunShiCiLoad_PanicDoesNotEscape 与 waitShiCiLoaded 组合：panic 后等待方不挂死
func TestRunShiCiLoad_PanicDoesNotEscape(t *testing.T) {
	ready := make(chan struct{})
	runShiCiLoad(func() error { panic("boom") }, ready)

	// 模拟业务入口：必须在超时前拿到 true（通道已关闭）
	if !waitShiCiLoaded(ready, time.Second) {
		t.Fatal("加载协程 panic 后，业务入口仍被挂住")
	}
}
