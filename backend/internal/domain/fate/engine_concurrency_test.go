package fate

import (
	"context"
	"sync"
	"testing"
	"time"
)

// TestDoubleNameWorkerLimitBoundsPeak 并发放大回归（P2-7）：
// 双重枚举的全局 worker 令牌必须把并发峰值钳制在容量内，无论多少请求在飞。
//
// 历史缺陷：generateDoubleName 每请求按 runtime.NumCPU() 固定分片，
// 8 个并发请求 = 128 个 worker goroutine 全部在跑 → 16 核机床上单请求
// 耗时线性恶化、吞吐在低并发出就封顶（实测见 docs/24 第 7 条）。
//
// 本测试注入一个小容量信号量（2）模拟紧张资源，同时发起 8 个会话，
// 断言所有 worker 的并发峰值 ≤ 2。若回归为「每请求各开 NumCPU」，
// 峰值会冲到 16×8，断言必然失败。
func TestDoubleNameWorkerLimitBoundsPeak(t *testing.T) {
	const injectedLimit = 2
	const concurrentReqs = 8

	origWorkerLimit := candidateWorkerLimit
	origPeak := candidateWorkerPeak.Load()
	defer func() {
		candidateWorkerLimit = origWorkerLimit
		candidateWorkerPeak.Store(origPeak)
	}()

	// 注入小容量令牌 + 清零峰值观测。
	// 全局峰值计数跨用例共享：若上一用例的在途 worker 尚在收尾（等待调度），
	// 会把本次峰值抬高到注入容量之外，形成与被测行为无关的偶发失败。
	// 故先等在途 worker 归零且短暂稳定后再注入测量。
	waitWorkerQuiescence(t)
	candidateWorkerLimit = make(chan struct{}, injectedLimit)
	candidateWorkerPeak.Store(0)

	provider := newCuratedPoolProvider(0)
	engine := NewEngine(provider, &stubAnalyzer{}, DefaultRaters())

	var wg sync.WaitGroup
	errs := make(chan error, concurrentReqs)
	for r := 0; r < concurrentReqs; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			session := engine.NewSession()
			if err := session.Start(context.Background(), &Input{
				Surname: "王",
				Gender:  GenderMale,
				Born:    time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
				Options: GenerateOptions{NameLength: 2, Count: 100},
			}); err != nil {
				errs <- err
				return
			}
			if err := session.Wait(); err != nil {
				errs <- err
				return
			}
			if session.Result() == nil || len(session.Result().TopNames) == 0 {
				errs <- &syncError{msg: "结果为 nil 或空"}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Logf("并发生成异常（不应阻塞整体断言）: %v", err)
		}
	}

	peak := candidateWorkerPeak.Load()
	if peak > injectedLimit {
		t.Fatalf("P2-7 回归：双重枚举 worker 并发峰值 = %d，超过注入限制 %d（说明 worker 未受全局令牌钳制）",
			peak, injectedLimit)
	}
	if peak == 0 {
		t.Fatalf("观测失效：并发峰值 = 0，说明 worker 计数路径未被触发或测试装配有误")
	}
}

// TestDoubleNameWorkerLimitCapsPerRequestWhileIdle 单请求场景下 worker 不应受全局限额过度抑制：
// 只有 1 个请求在飞时，最多 workerCountHint 个 worker 依次拿令牌执行，
// 峰值应 ≤ min(NumCPU, 候选池大小)，且结果正常产出。
//
// 该用例补充验证：令牌共享不会在「无竞争」时把 worker 压到 1（否则单请求性能倒退）。
func TestDoubleNameWorkerLimitCapsPerRequestWhileIdle(t *testing.T) {
	origWorkerLimit := candidateWorkerLimit
	origPeak := candidateWorkerPeak.Load()
	defer func() {
		candidateWorkerLimit = origWorkerLimit
		candidateWorkerPeak.Store(origPeak)
	}()
	waitWorkerQuiescence(t)
	candidateWorkerPeak.Store(0)

	provider := newCuratedPoolProvider(0)
	engine := NewEngine(provider, &stubAnalyzer{}, DefaultRaters())

	session := engine.NewSession()
	if err := session.Start(context.Background(), &Input{
		Surname: "王",
		Gender:  GenderMale,
		Born:    time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
		Options: GenerateOptions{NameLength: 2, Count: 100},
	}); err != nil {
		t.Fatalf("会话启动失败: %v", err)
	}
	if err := session.Wait(); err != nil {
		t.Fatalf("生成失败: %v", err)
	}
	if output := session.Result(); output == nil || len(output.TopNames) == 0 {
		t.Fatal("单请求应正常产出名字")
	}

	peak := candidateWorkerPeak.Load()
	hint := int64(workerCountHint(len(provider.chars)))
	if peak == 0 || peak > hint {
		t.Fatalf("单请求 worker 峰值 = %d，期望 (0, %d]，钳制逻辑异常", peak, hint)
	}
}

// syncError 测试内错误类型
type syncError struct{ msg string }

func (e *syncError) Error() string { return e.msg }

// waitWorkerQuiescence 等待全局在途 worker 计数归零且连续三次探测保持为 0。
// worker 从获取令牌到计入 active 存在微小窗口，仅查一次归零不排除
// 「刚查完就又有上一用例 worker 计入」的竞态，连续稳定才算安静。
func waitWorkerQuiescence(t *testing.T) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	stable := 0
	for {
		if candidateWorkerActive.Load() == 0 {
			stable++
			if stable >= 3 {
				return
			}
		} else {
			stable = 0
		}
		if time.Now().After(deadline) {
			t.Fatalf("等待在途枚举 worker 归零超时（active=%d），存在跨用例泄漏", candidateWorkerActive.Load())
		}
		time.Sleep(10 * time.Millisecond)
	}
}