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

// maxPageSize 单页最大条数（docs/29 B6）
//
// 历史记录每条内嵌**完整生成结果 JSON**（八字 + 卦象 + 紫微 + 推荐名列表），
// 单条可达数十 KB。此前 limit 只校验「正整数」无上限，`?limit=100000000`
// 合法，一次请求即可拖走全库数据并放大内存 —— 记录里含出生日期等 PII，
// 等于一次请求把整个语料库导出。
const maxPageSize = 100

// parsePagination 解析并校验 page/limit 查询参数。
//
// 校验策略：**非法值直接 400**，不做静默兜底。原先 `if err == nil && parsed > 0`
// 的写法会把 `limit=abc`、`limit=-1` 悄悄变成 100，客户端拼错参数却拿到
// 「成功」响应，问题被藏到数据不对时才暴露。
func parsePagination(c *gin.Context, defaultLimit int) (page int, limit int, ok bool) {
	page, limit = 1, defaultLimit

	if p := c.Query("page"); p != "" {
		parsed, err := strconv.Atoi(p)
		if err != nil || parsed <= 0 {
			response.ErrorJSON(c, 400, "page 必须是正整数")
			return 0, 0, false
		}
		page = parsed
	}
	if l := c.Query("limit"); l != "" {
		parsed, err := strconv.Atoi(l)
		if err != nil || parsed <= 0 {
			response.ErrorJSON(c, 400, "limit 必须是正整数")
			return 0, 0, false
		}
		if parsed > maxPageSize {
			// 超出上限明确告知真实生效值，而不是静默截断
			response.ErrorJSON(c, 400, "limit 不能超过 "+strconv.Itoa(maxPageSize))
			return 0, 0, false
		}
		limit = parsed
	}
	return page, limit, true
}

// GetHistory 获取历史记录
// @Summary 获取历史记录
// @Description 获取所有历史记录，支持分页（limit 上限 100）
// @Tags 历史记录
// @Accept json
// @Produce json
// @Param page query int false "页码，默认1"
// @Param limit query int false "每页数量，默认100，上限100"
// @Success 200 {object} object "成功，包含records和total"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /history [get]
func (h *HistoryHandler) GetHistory(c *gin.Context) {
	page, limit, ok := parsePagination(c, maxPageSize)
	if !ok {
		return
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
