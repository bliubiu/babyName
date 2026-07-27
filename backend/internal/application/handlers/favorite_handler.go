package handlers

import (
	"github.com/gin-gonic/gin"
	"name/internal/application/response"
	"name/internal/application/services"
	"name/internal/application/validator"
	"name/internal/infrastructure/logger"
	"go.uber.org/zap"
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

// GetFavorites 获取收藏列表
// @Summary 获取收藏列表
// @Description 获取所有收藏记录
// @Tags 收藏
// @Accept json
// @Produce json
// @Success 200 {array} services.FavoriteRecord "成功"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /favorites [get]
func (h *FavoriteHandler) GetFavorites(c *gin.Context) {
	result, err := h.service.GetFavorites(c.Request.Context())
	if err != nil {
		logger.Error("GetFavorites: failed", zap.Error(err))
		response.ErrorJSON(c, 500, "获取收藏列表失败")
		return
	}

	response.SuccessJSON(c, result)
}

// SaveFavorite 保存收藏
// @Summary 保存收藏
// @Description 保存一条收藏记录
// @Tags 收藏
// @Accept json
// @Produce json
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
// @Description 根据ID删除一条收藏记录
// @Tags 收藏
// @Accept json
// @Produce json
// @Param id path string true "收藏记录ID"
// @Success 200 {object} object "成功"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /favorites/{id} [delete]
func (h *FavoriteHandler) DeleteFavorite(c *gin.Context) {
	id := c.Param("id")

	if err := h.service.DeleteFavorite(c.Request.Context(), id); err != nil {
		logger.Error("DeleteFavorite: failed", zap.Error(err))
		response.ErrorJSON(c, 500, "删除收藏失败")
		return
	}

	response.SuccessJSON(c, gin.H{"message": "删除成功"})
}

// CheckFavorite 检查是否收藏
// @Summary 检查是否收藏
// @Description 检查指定名字是否已收藏
// @Tags 收藏
// @Accept json
// @Produce json
// @Param surname query string true "姓氏"
// @Param given_name query string true "名字"
// @Success 200 {object} object "成功，返回是否收藏"
// @Failure 400 {object} response.Response "请求参数错误"
// @Router /favorites/check [get]
func (h *FavoriteHandler) CheckFavorite(c *gin.Context) {
	surname := c.Query("surname")
	givenName := c.Query("given_name")

	if surname == "" || givenName == "" {
		response.ErrorJSON(c, 400, "缺少参数")
		return
	}

	isFavorite := h.service.CheckFavorite(c.Request.Context(), surname, givenName)
	response.SuccessJSON(c, gin.H{"is_favorite": isFavorite})
}

// BatchSaveFavorite 批量保存收藏
// @Summary 批量保存收藏
// @Description 批量保存多条收藏记录
// @Tags 收藏
// @Accept json
// @Produce json
// @Param request body []services.FavoriteRecord true "收藏记录请求参数列表"
// @Success 200 {object} object "成功，返回记录ID列表"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /favorites/batch [post]
func (h *FavoriteHandler) BatchSaveFavorite(c *gin.Context) {
	var reqs []services.FavoriteRecord
	if err := c.ShouldBindJSON(&reqs); err != nil {
		logger.Warn("BatchSaveFavorite: invalid request", zap.Error(err))
		response.ErrorJSON(c, 400, "请求参数格式错误")
		return
	}

	// 转换为指针切片并逐条校验
	ptrReqs := make([]*services.FavoriteRecord, len(reqs))
	for i := range reqs {
		ptrReqs[i] = &reqs[i]
		if !validateFavoriteRecord(c, ptrReqs[i]) {
			return
		}
	}

	ids, err := h.service.BatchSaveFavorite(c.Request.Context(), ptrReqs)
	if err != nil {
		logger.Error("BatchSaveFavorite: failed to save favorite", zap.Error(err))
		response.ErrorJSON(c, 500, "保存收藏失败")
		return
	}

	response.SuccessJSON(c, gin.H{"ids": ids})
}

// BatchDeleteFavorite 批量删除收藏
// @Summary 批量删除收藏
// @Description 批量删除多条收藏记录
// @Tags 收藏
// @Accept json
// @Produce json
// @Param request body object true "删除收藏请求参数"
// @Success 200 {object} object "成功"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /favorites/batch [delete]
func (h *FavoriteHandler) BatchDeleteFavorite(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("BatchDeleteFavorite: invalid request", zap.Error(err))
		response.ErrorJSON(c, 400, "请求参数格式错误")
		return
	}

	if err := h.service.BatchDeleteFavorite(c.Request.Context(), req.IDs); err != nil {
		logger.Error("BatchDeleteFavorite: failed to delete favorite", zap.Error(err))
		response.ErrorJSON(c, 500, "删除收藏失败")
		return
	}

	response.SuccessJSON(c, gin.H{"message": "批量删除成功"})
}
