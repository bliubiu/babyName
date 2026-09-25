package fate

import (
	"context"
	"errors"
	"testing"
	"time"
)

// ——— docs/28 W3：取消/超时不得伪装成成功 ———
//
// 缺陷本质：各枚举循环在 `cancelled(ctx)` 为真时只是 `return`（静默退出该 worker），
// 主流程照常组装 TopNames 并返回 nil error。于是「会话被取消」这件事在领域层
// **完全不可见**：Wait() 返回 nil、State() 是 Finish、Result() 是一份被截断的榜单。
// 上层若不额外检查 ctx.Err()，就会把它当正常结果回 200，客户端拿到不完整名单
// 却毫不知情、也不会重试。
//
// 本组用例钉住三点：
//  1. ctx 已取消时 Wait() 必须返回错误，且可用 errors.Is 判定；
//  2. State() 必须是 Canceled（而非 Finish/Failed），便于上层区分「该重试」；
//  3. Result() 不得被当作有效结果使用（应为 nil）。
//  4. 正常路径不受影响（仍返回 nil error + Finish + 有名字）。

// TestSessionCanceledContextReportsError 取消路径的核心断言。
func TestSessionCanceledContextReportsError(t *testing.T) {
	provider := newCuratedPoolProvider(0)
	engine := NewEngine(provider, &stubAnalyzer{}, DefaultRaters())

	// 用一个已取消的 ctx 启动：引擎各分片会立刻从循环中退出，
	// 这正是「被截断的榜单」得以产生的入口条件。
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	session := engine.NewSession()
	err := session.Start(ctx, &Input{
		Surname: "王",
		Gender:  GenderMale,
		Born:    time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
		Options: GenerateOptions{NameLength: 2, Count: 50},
	})
	if err != nil {
		// Start 自身因 ctx 失败也是可接受的结果（此时已满足「不可见」的修复目标）
		t.Logf("Start 直接拒绝已取消的 ctx: %v", err)
		return
	}

	waitErr := session.Wait()
	if waitErr == nil {
		t.Fatalf("ctx 已取消却 Wait() 返回 nil —— 上层会拿到被截断的榜单当成成功")
	}
	if !errors.Is(waitErr, context.Canceled) {
		t.Errorf("Wait() 错误 = %v，期望可被 errors.Is(err, context.Canceled) 判定", waitErr)
	}

	if got := session.State(); got != SessionStateCanceled {
		t.Errorf("State() = %d，期望 SessionStateCanceled(%d)——"+
			"标记为 Failed 会让上层误判为需重试的生成故障", got, SessionStateCanceled)
	}

	if out := session.Result(); out != nil {
		t.Errorf("被取消的会话不应产出 Result（实际拿到 %d 个名字）", len(out.TopNames))
	}
}

// TestSessionDeadlineExceededReportsError 超时路径与取消同源，
// 但错误必须是 DeadlineExceeded（handler 据此返回 503 而非 500）。
func TestSessionDeadlineExceededReportsError(t *testing.T) {
	provider := newCuratedPoolProvider(0)
	engine := NewEngine(provider, &stubAnalyzer{}, DefaultRaters())

	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	time.Sleep(2 * time.Millisecond) // 确保 deadline 已过期

	session := engine.NewSession()
	if err := session.Start(ctx, &Input{
		Surname: "李",
		Gender:  GenderFemale,
		Born:    time.Date(2024, 6, 1, 8, 0, 0, 0, time.UTC),
		Options: GenerateOptions{NameLength: 2, Count: 50},
	}); err != nil {
		t.Logf("Start 直接拒绝已过期的 ctx: %v", err)
		return
	}

	waitErr := session.Wait()
	if waitErr == nil {
		t.Fatal("deadline 已过期却 Wait() 返回 nil —— 会返回被截断的榜单")
	}
	if !errors.Is(waitErr, context.DeadlineExceeded) {
		t.Errorf("Wait() 错误 = %v，期望可被 errors.Is(err, context.DeadlineExceeded) 判定", waitErr)
	}
	if got := session.State(); got != SessionStateCanceled {
		t.Errorf("State() = %d，期望 SessionStateCanceled(%d)", got, SessionStateCanceled)
	}
}

// TestSessionNormalPathUnaffected 确认 W3 的修复没有把正常路径也误判为取消。
func TestSessionNormalPathUnaffected(t *testing.T) {
	provider := newCuratedPoolProvider(0)
	engine := NewEngine(provider, &stubAnalyzer{}, DefaultRaters())

	session := engine.NewSession()
	ctx := context.Background()
	if err := session.Start(ctx, &Input{
		Surname: "张",
		Gender:  GenderMale,
		Born:    time.Date(2024, 3, 10, 14, 0, 0, 0, time.UTC),
		Options: GenerateOptions{NameLength: 2, Count: 50},
	}); err != nil {
		t.Fatalf("正常会话启动失败: %v", err)
	}
	if err := session.Wait(); err != nil {
		t.Fatalf("正常会话 Wait() 失败: %v", err)
	}
	if got := session.State(); got != SessionStateFinish {
		t.Errorf("State() = %d，期望 SessionStateFinish(%d)", got, SessionStateFinish)
	}
	out := session.Result()
	if out == nil || len(out.TopNames) == 0 {
		t.Fatal("正常路径应产出名字")
	}
}

// TestSessionStopReportsError 通过 Stop() 主动中止的路径也必须能被上层感知。
//
// Stop() 会 cancel 内部 context，若引擎不把取消上抛，调用方就无法区分
// 「我主动停了」与「它跑完了」。
func TestSessionStopReportsError(t *testing.T) {
	provider := newCuratedPoolProvider(0)
	engine := NewEngine(provider, &stubAnalyzer{}, DefaultRaters())

	session := engine.NewSession()
	if err := session.Start(context.Background(), &Input{
		Surname: "赵",
		Gender:  GenderMale,
		Born:    time.Date(2024, 9, 9, 9, 0, 0, 0, time.UTC),
		Options: GenerateOptions{NameLength: 2, Count: 100},
	}); err != nil {
		t.Fatalf("会话启动失败: %v", err)
	}

	// 尽量在生成进行中按停；即便它已经跑完，Stop() 也不应让状态回退成 Finish。
	if err := session.Stop(); err != nil {
		t.Fatalf("Stop() 失败: %v", err)
	}

	waitErr := session.Wait()
	state := session.State()

	// 两种可接受结局：
	//   - 已被我们成功中止：Wait 返回取消错误 + Canceled；
	//   - 生成太快已经跑完：Wait 返回 nil + Finish（不算缺陷）。
	switch state {
	case SessionStateCanceled:
		if waitErr == nil {
			t.Errorf("状态为 Canceled 但 Wait() 返回 nil —— 上层无法感知中止")
		} else if !errors.Is(waitErr, context.Canceled) {
			t.Errorf("Wait() 错误 = %v，期望可被 errors.Is(err, context.Canceled) 判定", waitErr)
		}
	case SessionStateFinish:
		t.Logf("生成在 Stop 前已完成，本轮未覆盖中止路径（属正常竞态）")
	default:
		t.Errorf("Stop 后状态 = %d，期望 Canceled 或 Finish", state)
	}
}
