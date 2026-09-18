package handlers

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"name/internal/application/response"
	"name/internal/application/services"
	"name/internal/infrastructure/logger"
)

// NameAsyncHandler 异步生成任务 + 探索模式（换一批）处理器
type NameAsyncHandler struct {
	name  *NameHandler // 复用生成请求的解析与校验
	tasks *services.TaskService
	names services.NameServiceInterface
}

// NewNameAsyncHandler 创建处理器
func NewNameAsyncHandler(name *NameHandler, tasks *services.TaskService, names services.NameServiceInterface) *NameAsyncHandler {
	return &NameAsyncHandler{name: name, tasks: tasks, names: names}
}

// GenerateAsync 提交异步生成任务，立即返回 task_id。
//
// 长任务体验改造（docs/27 §3.3）：同步生成受 30s 请求超时约束且无进度，
// 异步路径提交后由前端轮询 GET /names/task/:id 获取阶段与百分比。
func (h *NameAsyncHandler) GenerateAsync(c *gin.Context) {
	req, ok := h.name.validateGenerateRequest(c)
	if !ok {
		return
	}

	taskID := h.tasks.Submit(req)
	logger.Info("GenerateAsync: task submitted",
		zap.String("task_id", taskID),
		zap.String("surname", req.Surname),
	)
	response.SuccessJSON(c, gin.H{"task_id": taskID})
}

// GetTask 查询异步任务状态；成功时附带完整生成结果（取走即弃）
func (h *NameAsyncHandler) GetTask(c *gin.Context) {
	id := c.Param("id")
	task, ok := h.tasks.Get(id)
	if !ok {
		response.ErrorJSON(c, 404, "任务不存在或已过期，请重新提交")
		return
	}

	status, stage, percent, errMsg := task.View()
	resp := gin.H{
		"task_id": id,
		"status":  status,
		"stage":   stage,
		"percent": percent,
	}
	switch status {
	case services.TaskStatusFailed:
		resp["error"] = errMsg
	case services.TaskStatusSuccess:
		resp["result"] = task.TakeResult()
	}
	response.SuccessJSON(c, resp)
}

// Explore 探索模式：从指定生成会话换一批（与上次结果零交集）
func (h *NameAsyncHandler) Explore(c *gin.Context) {
	var req services.ExploreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorJSON(c, 400, "请求参数格式错误")
		return
	}
	result, err := h.names.ExploreNames(c.Request.Context(), &req)
	if err != nil {
		logger.Warn("Explore: failed", zap.Error(err), zap.String("generation_id", req.GenerationID))
		response.ErrorJSON(c, 400, err.Error())
		return
	}
	response.SuccessJSON(c, result)
}
