package services

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"name/internal/infrastructure/logger"

	"go.uber.org/zap"
)

// ——— 异步生成任务（长任务体验） ———
//
// 背景（docs/27 §3.3）：/names/generate 是同步长请求，前端 30s 硬超时，
// 超时后只有 503，中间无任何进度。参考 fate v4 的「异步任务 + task_id 轮询」：
//   - POST /names/generate/async 立即返回 task_id；
//   - GET /names/task/:id 轮询状态/阶段/百分比，完成后取完整结果；
//   - 同步路径保留不变（task_service 只是把 GenerateWithProgress 搬进后台）。
//
// 任务存内存（单实例部署定位），带 TTL 与容量上限；结果体较大，取走即弃可选。

// TaskStatus 异步任务状态
type TaskStatus string

const (
	TaskStatusPending TaskStatus = "pending"
	TaskStatusRunning TaskStatus = "running"
	TaskStatusSuccess TaskStatus = "success"
	TaskStatusFailed  TaskStatus = "failed"
)

const (
	taskTTL        = 30 * time.Minute // 任务结果保留时长
	taskMaxCount   = 200              // 任务表容量上限（超限按创建时间淘汰最旧）
	taskRunTimeout = 5 * time.Minute  // 后台生成的硬超时（远大于前端 30s）
	// taskMaxRunning 同时在跑的后台生成任务上限（docs/29 B7）
	//
	// 此前 Submit 无条件 `go ts.run(...)`，且 evictLocked 只删 map 不终止
	// 已运行的 goroutine：稳态下可堆积数千个「全量枚举 + 持有完整候选池」
	// 的任务，生产容器 512M 内存直接 OOM。这里用带缓冲 channel 做信号量，
	// 满载时 Submit 直接拒绝（返回错误让前端提示稍后重试），不做无限排队 ——
	// 排队只会把拒绝推迟成 OOM。
	taskMaxRunning = 4
)

// ErrTooManyTasks 提交被拒：同时在跑的任务已达上限
var ErrTooManyTasks = errors.New("当前生成任务过多，请稍后重试")

// ErrTaskCanceled 任务被取消（淘汰或服务关闭）
var ErrTaskCanceled = errors.New("生成任务已取消")

// GenerateTask 一次异步生成的任务
type GenerateTask struct {
	mu        sync.Mutex
	status    TaskStatus
	stage     string
	percent   float64
	errMsg    string
	result    *GenerateResponse
	createdAt time.Time

	// cancel 终止本任务后台 goroutine 用的取消函数。
	// 淘汰（TTL/超容）时必须调用，否则 map 里虽然删了任务，
	// goroutine 仍会跑满 5 分钟全量枚举 —— 这正是 B7 的 OOM 来源。
	// 由 Submit 在写入 map 前赋值，此后只读，不需要额外加锁
	// （happens-before 由 ts.mu 保证）。
	cancel context.CancelFunc
	// evicted 标记任务已被淘汰：run 结束时据此跳过结果回填，
	// 避免给一个已从 map 删除的任务白白构造并持有 GenerateResponse。
	evicted bool
}

// View 任务状态快照
func (t *GenerateTask) View() (status TaskStatus, stage string, percent float64, errMsg string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.status, t.stage, t.percent, t.errMsg
}

// TakeResult 取走结果（仅 success 后有值；取出后任务内不再保留，防内存堆积）
func (t *GenerateTask) TakeResult() *GenerateResponse {
	t.mu.Lock()
	defer t.mu.Unlock()
	r := t.result
	t.result = nil
	return r
}

// TaskService 异步生成任务管理器
type TaskService struct {
	svc   *NameService
	mu    sync.Mutex
	tasks map[string]*GenerateTask
	order []string // 创建顺序，用于 TTL/容量淘汰
	seq   int
	// running 后台生成任务的并发信号量（容量 taskMaxRunning）
	running chan struct{}
}

// NewTaskService 创建异步任务服务
func NewTaskService(svc *NameService) *TaskService {
	return &TaskService{
		svc:     svc,
		tasks:   make(map[string]*GenerateTask),
		running: make(chan struct{}, taskMaxRunning),
	}
}

// Submit 提交异步生成任务，立即返回 task_id
//
// 并发已满时返回 ErrTooManyTasks（此时 task_id 为空），由 handler 转 503。
// 选择「拒绝」而非「排队」：排队只是把内存耗尽推迟到某一刻，且被排队的
// 用户拿到的 task_id 要等很久才有响应，交互上更差。
func (ts *TaskService) Submit(req *GenerateRequest) (string, error) {
	// 先抢并发名额。放在 map 写入之前，失败时不留任何状态。
	select {
	case ts.running <- struct{}{}:
	default:
		logger.Warn("GenerateTask: rejected, too many running tasks",
			zap.Int("running", len(ts.running)),
			zap.Int("limit", taskMaxRunning),
		)
		return "", ErrTooManyTasks
	}

	id := fmt.Sprintf("t%d-%d", time.Now().UnixNano(), ts.nextSeq())

	// 每个任务独立可取消：超时 + 淘汰取消都通过它生效。
	// 不能用 context.Background() 直接跑，淘汰时就无从终止。
	ctx, cancel := context.WithCancel(context.Background())

	task := &GenerateTask{
		status:    TaskStatusRunning,
		stage:     "排队中",
		percent:   0,
		createdAt: time.Now(),
		cancel:    cancel,
	}

	ts.mu.Lock()
	ts.tasks[id] = task
	ts.order = append(ts.order, id)
	ts.evictLocked()
	ts.mu.Unlock()

	go ts.run(ctx, id, task, req)
	return id, nil
}

// Get 查询任务；status=success 时 result 非nil（若尚未被取走）
func (ts *TaskService) Get(id string) (*GenerateTask, bool) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	t, ok := ts.tasks[id]
	return t, ok
}

// nextSeq 生成递增序号
func (ts *TaskService) nextSeq() int {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.seq++
	return ts.seq
}

// evictLocked 淘汰过期与超容任务（调用方须持 ts.mu）
//
// 淘汰不再只是「从 map 里删掉」：仍在运行的任务必须同时被 cancel，
// 否则后台 goroutine 会继续跑满 taskRunTimeout 的全量枚举，
// 而 map 里已经查不到它 —— 稳态下就是「几千个不可见的长任务」把内存打爆。
//
// 锁序说明：这里在持有 ts.mu 的情况下调用 context.CancelFunc。
// CancelFunc 本身只做「关闭 channel + 遍历子 ctx」，不回调任何会再取
// ts.mu 的代码，因此不会自死锁；被唤醒的 goroutine 自行阻塞在 channel 上
// 直至 run 收尾，与本函数无锁竞争。
func (ts *TaskService) evictLocked() {
	now := time.Now()
	kept := ts.order[:0]
	for _, id := range ts.order {
		t, ok := ts.tasks[id]
		if !ok {
			continue
		}
		if now.Sub(t.createdAt) > taskTTL {
			ts.evictTaskLocked(id, t)
			continue
		}
		kept = append(kept, id)
	}
	ts.order = kept
	for len(ts.order) > taskMaxCount {
		oldest := ts.order[0]
		ts.order = ts.order[1:]
		if t, ok := ts.tasks[oldest]; ok {
			ts.evictTaskLocked(oldest, t)
		}
	}
}

// evictTaskLocked 淘汰单个任务：从 map 摘除并取消其后台执行（调用方须持 ts.mu）
func (ts *TaskService) evictTaskLocked(id string, t *GenerateTask) {
	delete(ts.tasks, id)

	t.mu.Lock()
	t.evicted = true
	alreadyDone := t.status == TaskStatusSuccess || t.status == TaskStatusFailed
	t.mu.Unlock()

	// 已完成的任务 cancel 无意义，且会误伤「结果刚回填、正等人来取」的窗口
	if !alreadyDone && t.cancel != nil {
		t.cancel()
	}
}

// run 后台执行生成（独立于请求生命周期的 ctx，客户端断开不影响完成；
// 但被淘汰时 ctx 会被 cancel，此时应尽快退出）
func (ts *TaskService) run(ctx context.Context, id string, task *GenerateTask, req *GenerateRequest) {
	// 归还并发名额：无论成功、失败、panic 还是被取消，都必须释放，
	// 否则槽位会被逐步耗尽，Submit 永久拒绝。
	defer func() { <-ts.running }()

	defer func() {
		if r := recover(); r != nil {
			task.mu.Lock()
			task.status = TaskStatusFailed
			task.errMsg = "生成过程异常，请稍后重试"
			task.mu.Unlock()
			logger.Error("GenerateTask: panic", zap.String("task_id", id), zap.Any("panic", r))
		}
	}()

	// 超时与淘汰取消合并到同一个 ctx：父 ctx 被 cancel 即视为任务取消
	ctx, cancel := context.WithTimeout(ctx, taskRunTimeout)
	defer cancel()

	result, err := ts.svc.GenerateWithProgress(ctx, req, func(stage string, percent float64) {
		// 取消后不再回填进度：任务已经不该对任何人可见了
		if ctx.Err() != nil {
			return
		}
		task.mu.Lock()
		task.stage = stage
		task.percent = percent
		task.mu.Unlock()
	})

	task.mu.Lock()
	defer task.mu.Unlock()

	// 已被淘汰：不要回填结果。GenerateResponse 持有完整候选与解析结果，
	// 给一个查不到的任务留着它纯属浪费内存。
	if task.evicted {
		logger.Info("GenerateTask: canceled by eviction", zap.String("task_id", id))
		return
	}

	if err != nil {
		task.status = TaskStatusFailed
		if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
			task.errMsg = ErrTaskCanceled.Error()
		} else {
			task.errMsg = "名字生成失败，请调整筛选条件或稍后重试"
		}
		logger.Warn("GenerateTask: failed",
			zap.String("task_id", id),
			zap.String("surname", req.Surname),
			zap.Error(err),
		)
		return
	}
	task.status = TaskStatusSuccess
	task.stage = "完成"
	task.percent = 100
	task.result = result
	logger.Info("GenerateTask: success",
		zap.String("task_id", id),
		zap.String("surname", req.Surname),
		zap.Int("count", len(result.Names)),
	)
}
