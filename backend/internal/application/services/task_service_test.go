package services

// task_service_test.go — 异步任务并发上限与淘汰取消（docs/29 B7）
//
// B7 的原始病灶：Submit 无条件 `go ts.run(...)`，且 evictLocked 只从 map 删条目、
// 不终止已在运行的 goroutine。生产容器 512M 内存下可堆积数千个「全量枚举 +
// 持有完整候选池」的任务直接 OOM，且这些任务在 /task 轮询接口里已查不到，
// 运维完全看不见。这里锁死三条不变量。

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"
)

// TestSubmit_RejectsWhenRunningFull 槽位满时必须拒绝，且不留任何状态
func TestSubmit_RejectsWhenRunningFull(t *testing.T) {
	ts := NewTaskService(nil)

	// 直接占满信号量：run 里 defer 的归还逻辑此时尚未参与
	for i := 0; i < taskMaxRunning; i++ {
		ts.running <- struct{}{}
	}

	id, err := ts.Submit(&GenerateRequest{Surname: "王"})
	if !errors.Is(err, ErrTooManyTasks) {
		t.Fatalf("槽位满时应返回 ErrTooManyTasks，实际 err=%v", err)
	}
	if id != "" {
		t.Errorf("被拒绝时不应返回 task_id，实际 %q", id)
	}

	// 拒绝不能留下半成品状态，否则会污染轮询与淘汰计数
	ts.mu.Lock()
	taskCount, orderCount := len(ts.tasks), len(ts.order)
	ts.mu.Unlock()
	if taskCount != 0 || orderCount != 0 {
		t.Errorf("被拒绝的任务不应入表，实际 tasks=%d order=%d", taskCount, orderCount)
	}
}

// TestEvictLocked_CancelsRunningTask 淘汰必须终止仍在运行的任务（核心回归）
func TestEvictLocked_CancelsRunningTask(t *testing.T) {
	ts := NewTaskService(nil)

	_, cancel := context.WithCancel(context.Background())
	canceled := make(chan struct{})
	task := &GenerateTask{
		status:    TaskStatusRunning,
		stage:     "生成中",
		createdAt: time.Now(),
		cancel:    func() { cancel(); close(canceled) },
	}

	ts.mu.Lock()
	// 目标任务作为最旧的一个，另填 taskMaxCount 个占位把容量顶到上限 +1
	ts.tasks["t-old"] = task
	ts.order = append(ts.order, "t-old")
	for i := 0; i < taskMaxCount; i++ {
		id := "filler-" + strconv.Itoa(i)
		ts.tasks[id] = &GenerateTask{status: TaskStatusRunning, createdAt: time.Now()}
		ts.order = append(ts.order, id)
	}
	ts.evictLocked()
	_, stillThere := ts.tasks["t-old"]
	ts.mu.Unlock()

	if stillThere {
		t.Error("超容任务应从 map 移除")
	}
	select {
	case <-canceled:
	default:
		t.Fatal("淘汰仍在运行的任务时必须调用 cancel，否则 goroutine 继续跑满 5 分钟全量枚举")
	}
}

// TestEvictLocked_NotCancelFinishedTask 已完成任务不应被 cancel
//
// 结果回填与淘汰之间存在「刚写完还没人来取」的窗口，无脑 cancel 会让
// 一个本可交付的成功任务变成中途夭折。
func TestEvictLocked_NotCancelFinishedTask(t *testing.T) {
	ts := NewTaskService(nil)

	canceled := make(chan struct{})
	task := &GenerateTask{
		status:    TaskStatusSuccess,
		stage:     "完成",
		percent:   100,
		createdAt: time.Now(),
		cancel:    func() { close(canceled) },
	}

	ts.mu.Lock()
	ts.tasks["t-done"] = task
	ts.order = append(ts.order, "t-done")
	ts.evictLocked()
	ts.mu.Unlock()

	select {
	case <-canceled:
		t.Fatal("已完成任务被淘汰时不应 cancel")
	default:
	}
}

// TestEvictLocked_TTLExpired 超过保留时长的任务同样要取消
func TestEvictLocked_TTLExpired(t *testing.T) {
	ts := NewTaskService(nil)

	canceled := make(chan struct{})
	task := &GenerateTask{
		status:    TaskStatusRunning,
		createdAt: time.Now().Add(-taskTTL - time.Minute),
		cancel:    func() { close(canceled) },
	}

	ts.mu.Lock()
	ts.tasks["t-stale"] = task
	ts.order = append(ts.order, "t-stale")
	ts.evictLocked()
	ts.mu.Unlock()

	select {
	case <-canceled:
	default:
		t.Fatal("TTL 过期的运行中任务必须被取消")
	}
}

// TestEvictLocked_KeepsFreshTasks 未到期的任务不得被误淘汰
func TestEvictLocked_KeepsFreshTasks(t *testing.T) {
	ts := NewTaskService(nil)

	task := &GenerateTask{status: TaskStatusRunning, createdAt: time.Now()}
	ts.mu.Lock()
	ts.tasks["t-fresh"] = task
	ts.order = append(ts.order, "t-fresh")
	ts.evictLocked()
	_, ok := ts.tasks["t-fresh"]
	ts.mu.Unlock()

	if !ok {
		t.Error("未超期任务不应被淘汰")
	}
}

// TestRun_ReleasesSlot 任务跑完必须归还并发名额
//
// 槽位只借不还的话，跑满 4 次之后 Submit 会永久拒绝 —— 这是本实现最容易
// 引入的回归，必须有独立断言守住。
func TestRun_ReleasesSlot(t *testing.T) {
	svc := setupNameServiceE2E(t)
	ts := NewTaskService(svc)

	id, err := ts.Submit(&GenerateRequest{
		Surname:    "王",
		Gender:     "male",
		BirthYear:  2024,
		BirthMonth: 1,
		BirthDay:   15,
		BirthHour:  12,
		NameLength: 2,
	})
	if err != nil {
		t.Fatalf("提交失败: %v", err)
	}

	deadline := time.Now().Add(2 * time.Minute)
	for {
		task, ok := ts.Get(id)
		if !ok {
			t.Fatal("任务丢失")
		}
		status, _, _, _ := task.View()
		if status == TaskStatusSuccess || status == TaskStatusFailed {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("任务未在 2 分钟内结束，status=%s", status)
		}
		time.Sleep(20 * time.Millisecond)
	}

	// run 的 defer 已执行，槽位应回到 0
	deadline = time.Now().Add(2 * time.Second)
	for len(ts.running) > 0 {
		if time.Now().After(deadline) {
			t.Fatalf("任务结束后并发名额未归还，running=%d", len(ts.running))
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// TestRun_CanceledTaskDoesNotHoldResult 被取消的任务不应回填结果
//
// 淘汰后任务在 map 里已查不到，若仍构造并持有完整 GenerateResponse
// 就是纯浪费内存 —— 而这正是 B7 OOM 的形状。
func TestRun_CanceledTaskDoesNotHoldResult(t *testing.T) {
	ts := NewTaskService(nil)
	task := &GenerateTask{status: TaskStatusRunning, stage: "生成中"}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // 模拟淘汰已在 run 之前完成

	ts.running <- struct{}{}
	ts.run(ctx, "t-gone", task, &GenerateRequest{Surname: "王"})

	task.mu.Lock()
	defer task.mu.Unlock()
	if task.result != nil {
		t.Error("被取消的任务不应持有结果")
	}
}

// TestTaskStatusConstants 状态字面量是前端契约的一部分，改动会静默破坏轮询
func TestTaskStatusConstants(t *testing.T) {
	cases := map[TaskStatus]string{
		TaskStatusRunning: "running",
		TaskStatusSuccess: "success",
		TaskStatusFailed:  "failed",
	}
	for got, want := range cases {
		if string(got) != want {
			t.Errorf("状态常量漂移: got %q want %q", got, want)
		}
	}
}