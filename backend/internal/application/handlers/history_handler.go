package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"name/internal/application/response"
	"name/internal/application/services"
	"name/internal/infrastructure/logger"
	"go.uber.org/zap"
)

type HistoryHandler struct {
	service services.HistoryServiceInterface
}

func NewHistoryHandler(service services.HistoryServiceInterface) *HistoryHandler {
	return &HistoryHandler{service: service}
}

// GetHistory 获取历史记录
// @Summary 获取历史记录
// @Description 获取所有历史记录，支持分页
// @Tags 历史记录
// @Accept json
// @Produce json
// @Param page query int false "页码，默认1"
// @Param limit query int false "每页数量，默认100"
// @Success 200 {object} object "成功，包含records和total"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /history [get]
func (h *HistoryHandler) GetHistory(c *gin.Context) {
	page := 1
	limit := 100

	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	result, total, err := h.service.GetHistoryPage(c.Request.Context(), page, limit)
	if err != nil {
		logger.Error("GetHistory: failed", zap.Error(err))
		response.ErrorJSON(c, 500, "获取历史记录失败")
		return
	}

	response.SuccessJSON(c, gin.H{
		"records": result,
		"total":   total,
		"page":    page,
		"limit":   limit,
	})
}

// SaveHistory 保存历史记录
// @Summary 保存历史记录
// @Description 保存一条历史记录
// @Tags 历史记录
// @Accept json
// @Produce json
// @Param request body services.HistoryRecord true "历史记录请求参数"
// @Success 200 {object} object "成功，返回记录ID"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /history [post]
func (h *HistoryHandler) SaveHistory(c *gin.Context) {
	var req services.HistoryRecord
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("SaveHistory: invalid request", zap.Error(err))
		response.ErrorJSON(c, 400, "请求参数格式错误")
		return
	}

	id, err := h.service.SaveHistory(c.Request.Context(), &req)
	if err != nil {
		logger.Error("SaveHistory: failed", zap.Error(err))
		response.ErrorJSON(c, 500, "保存历史记录失败")
		return
	}

	response.SuccessJSON(c, gin.H{"id": id})
}

// DeleteHistory 删除历史记录
// @Summary 删除历史记录
// @Description 根据ID删除一条历史记录
// @Tags 历史记录
// @Accept json
// @Produce json
// @Param id path string true "历史记录ID"
// @Success 200 {object} object "成功"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /history/{id} [delete]
func (h *HistoryHandler) DeleteHistory(c *gin.Context) {
	id := c.Param("id")

	if err := h.service.DeleteHistory(c.Request.Context(), id); err != nil {
		logger.Error("DeleteHistory: failed", zap.Error(err))
		response.ErrorJSON(c, 500, "删除历史记录失败")
		return
	}

	response.SuccessJSON(c, gin.H{"message": "删除成功"})
}

// BatchSaveHistory 批量保存历史记录
// @Summary 批量保存历史记录
// @Description 批量保存多条历史记录
// @Tags 历史记录
// @Accept json
// @Produce json
// @Param request body []services.HistoryRecord true "历史记录请求参数列表"
// @Success 200 {object} object "成功，返回记录ID列表"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /history/batch [post]
func (h *HistoryHandler) BatchSaveHistory(c *gin.Context) {
	var reqs []services.HistoryRecord
	if err := c.ShouldBindJSON(&reqs); err != nil {
		logger.Warn("BatchSaveHistory: invalid request", zap.Error(err))
		response.ErrorJSON(c, 400, "请求参数格式错误")
		return
	}

	// 转换为指针切片
	ptrReqs := make([]*services.HistoryRecord, len(reqs))
	for i := range reqs {
		ptrReqs[i] = &reqs[i]
	}

	ids, err := h.service.BatchSaveHistory(c.Request.Context(), ptrReqs)
	if err != nil {
		logger.Error("BatchSaveHistory: failed to save history", zap.Error(err))
		response.ErrorJSON(c, 500, "保存历史记录失败")
		return
	}

	response.SuccessJSON(c, gin.H{"ids": ids})
}
