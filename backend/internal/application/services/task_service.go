package services

import (
	"context"
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
)

// GenerateTask 一次异步生成的任务
type GenerateTask struct {
	mu        sync.Mutex
	status    TaskStatus
	stage     string
	percent   float64
	errMsg    string
	result    *GenerateResponse
	createdAt time.Time
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
}

// NewTaskService 创建异步任务服务
func NewTaskService(svc *NameService) *TaskService {
	return &TaskService{
		svc:   svc,
		tasks: make(map[string]*GenerateTask),
	}
}

// Submit 提交异步生成任务，立即返回 task_id
func (ts *TaskService) Submit(req *GenerateRequest) string {
	id := fmt.Sprintf("t%d-%d", time.Now().UnixNano(), ts.nextSeq())

	task := &GenerateTask{
		status:    TaskStatusRunning,
		stage:     "排队中",
		percent:   0,
		createdAt: time.Now(),
	}

	ts.mu.Lock()
	ts.tasks[id] = task
	ts.order = append(ts.order, id)
	ts.evictLocked()
	ts.mu.Unlock()

	go ts.run(id, task, req)
	return id
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
func (ts *TaskService) evictLocked() {
	now := time.Now()
	kept := ts.order[:0]
	for _, id := range ts.order {
		t, ok := ts.tasks[id]
		if !ok {
			continue
		}
		if now.Sub(t.createdAt) > taskTTL {
			delete(ts.tasks, id)
			continue
		}
		kept = append(kept, id)
	}
	ts.order = kept
	for len(ts.order) > taskMaxCount {
		oldest := ts.order[0]
		ts.order = ts.order[1:]
		delete(ts.tasks, oldest)
	}
}

// run 后台执行生成（独立于请求生命周期的 ctx，客户端断开不影响完成）
func (ts *TaskService) run(id string, task *GenerateTask, req *GenerateRequest) {
	defer func() {
		if r := recover(); r != nil {
			task.mu.Lock()
			task.status = TaskStatusFailed
			task.errMsg = "生成过程异常，请稍后重试"
			task.mu.Unlock()
			logger.Error("GenerateTask: panic", zap.String("task_id", id), zap.Any("panic", r))
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), taskRunTimeout)
	defer cancel()

	result, err := ts.svc.GenerateWithProgress(ctx, req, func(stage string, percent float64) {
		task.mu.Lock()
		task.stage = stage
		task.percent = percent
		task.mu.Unlock()
	})

	task.mu.Lock()
	defer task.mu.Unlock()
	if err != nil {
		task.status = TaskStatusFailed
		task.errMsg = "名字生成失败，请调整筛选条件或稍后重试"
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
