package handlers

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"name/internal/application/response"
	"name/internal/application/services"
	"name/internal/application/validator"
	"name/internal/infrastructure/logger"
	"name/internal/infrastructure/middleware"
)

// validateFavoriteRecord 校验收藏记录核心字段，校验失败写入响应并返回 false
func validateFavoriteRecord(c *gin.Context, req *services.FavoriteRecord) bool {
	if err := validator.ValidateSurname(req.Surname); err != nil {
		logger.Warn("SaveFavorite: invalid surname", zap.String("surname", req.Surname))
		response.ErrorJSON(c, 400, err.Error())
		return false
	}
	if err := validator.ValidateGivenName(req.GivenName); err != nil {
		logger.Warn("SaveFavorite: invalid given_name", zap.String("given_name", req.GivenName))
		response.ErrorJSON(c, 400, err.Error())
		return false
	}
	return true
}

type FavoriteHandler struct {
	service services.FavoriteServiceInterface
}

func NewFavoriteHandler(service services.FavoriteServiceInterface) *FavoriteHandler {
	return &FavoriteHandler{service: service}
}

// GetFavorites 获取当前会话的收藏列表
// @Summary 获取收藏列表
// @Description 分页获取当前会话的收藏记录（limit 上限 100）
// @Tags 收藏
// @Accept json
// @Produce json
// @Param page query int false "页码，默认1"
// @Param limit query int false "每页数量，默认100，上限100"
// @Success 200 {object} object "成功，包含 data/records、total、page、limit"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /favorites [get]
func (h *FavoriteHandler) GetFavorites(c *gin.Context) {
	page, limit, ok := parsePagination(c, maxPageSize)
	if !ok {
		return
	}
	sid, err := middleware.EnsureSessionID(c)
	if err != nil {
		logger.Error("GetFavorites: 初始化会话失败", zap.Error(err))
		response.ErrorJSON(c, 500, "会话初始化失败，请稍后重试")
		return
	}

	records, total, err := h.service.GetFavoritesPage(c.Request.Context(), sid, page, limit)
	if err != nil {
		logger.Error("GetFavorites: failed", zap.Error(err))
		response.ErrorJSON(c, 500, "获取收藏列表失败")
		return
	}

	// data 字段保持为记录数组（与旧契约兼容），另附分页元信息
	response.SuccessJSON(c, gin.H{
		"data":    records,
		"records": records,
		"total":   total,
		"page":    page,
		"limit":   limit,
	})
}

// SaveFavorite 保存收藏
// @Summary 保存收藏
// @Description 保存一条归属于当前会话的收藏记录
// @Tags 收藏
// @Accept json
// @Produce json
// @Param X-Session-ID header string false "会话标识（无 cookie 的客户端使用）"
// @Param request body services.FavoriteRecord true "收藏记录请求参数"
// @Success 200 {object} object "成功，返回记录ID"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /favorites [post]
func (h *FavoriteHandler) SaveFavorite(c *gin.Context) {
	var req services.FavoriteRecord
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("SaveFavorite: invalid request", zap.Error(err))
		response.ErrorJSON(c, 400, "请求参数格式错误")
		return
	}

	if !validateFavoriteRecord(c, &req) {
		return
	}
	sid, err := middleware.EnsureSessionID(c)
	if err != nil {
		logger.Error("SaveFavorite: 初始化会话失败", zap.Error(err))
		response.ErrorJSON(c, 500, "会话初始化失败，请稍后重试")
		return
	}
	// 归属由服务端会话决定，客户端提交的 SessionID 字段被 json:"-" 屏蔽。
	req.SessionID = sid

	id, err := h.service.SaveFavorite(c.Request.Context(), &req)
	if err != nil {
		logger.Error("SaveFavorite: failed", zap.Error(err))
		response.ErrorJSON(c, 500, "保存收藏失败")
		return
	}

	response.SuccessJSON(c, gin.H{"id": id})
}

// DeleteFavorite 删除收藏
// @Summary 删除收藏
// @Description 根据ID删除当前会话的一条收藏记录
// @Tags 收藏
// @Accept json
// @Produce json
// @Param X-Session-ID header string false "会话标识（无 cookie 的客户端使用）"
// @Param id path string true "收藏记录ID"
// @Success 200 {object} object "成功"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /favorites/{id} [delete]
func (h *FavoriteHandler) DeleteFavorite(c *gin.Context) {
	id := c.Param("id")
	sid, err := middleware.EnsureSessionID(c)
	if err != nil {
		logger.Error("DeleteFavorite: 初始化会话失败", zap.Error(err))
		response.ErrorJSON(c, 500, "会话初始化失败，请稍后重试")
		return
	}

	if err := h.service.DeleteFavorite(c.Request.Context(), sid, id); err != nil {
		logger.Error("DeleteFavorite: failed", zap.Error(err))
		response.ErrorJSON(c, 500, "删除收藏失败")
		return
	}

	response.SuccessJSON(c, gin.H{"message": "删除成功"})
}
