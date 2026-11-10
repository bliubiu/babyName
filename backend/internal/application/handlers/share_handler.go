package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"name/internal/application/response"
	"name/internal/application/services"
	"name/internal/infrastructure/logger"
)

type ShareHandler struct {
	service *services.ShareService
}

func NewShareHandler(service *services.ShareService) *ShareHandler {
	return &ShareHandler{service: service}
}

// CreateShare 保存一份展示摘要并返回可分享的链接 id
// @Summary 创建分享快照
// @Description 保存结果页裁剪后的展示摘要，返回随机 id 与分享链接；无需登录
// @Tags 分享
// @Accept json
// @Produce json
// @Param request body object true "分享摘要"
// @Success 200 {object} response.Response "成功，data 含 id 与 url"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /share [post]
func (h *ShareHandler) CreateShare(c *gin.Context) {
	var req struct {
		Data json.RawMessage `json:"data"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("分享快照：请求解析失败", zap.Error(err))
		response.ErrorJSON(c, 400, "请求参数格式错误")
		return
	}
	id, err := h.service.Create(c.Request.Context(), req.Data)
	if err != nil {
		if errors.Is(err, services.ErrInvalidShare) {
			response.ErrorJSON(c, 400, err.Error())
			return
		}
		logger.Error("分享快照：保存失败", zap.Error(err))
		response.ErrorJSON(c, 500, "创建分享失败")
		return
	}
	response.SuccessJSON(c, gin.H{"id": id, "url": "/share?id=" + id})
}

// GetShare 读取分享快照
// @Summary 读取分享快照
// @Description 按随机 id 读取结果页展示摘要；id 非法或不存在返回 404
// @Tags 分享
// @Produce json
// @Param id path string true "分享标识"
// @Success 200 {object} response.Response "成功，data 为展示摘要"
// @Failure 404 {object} response.Response "分享不存在"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /share/{id} [get]
func (h *ShareHandler) GetShare(c *gin.Context) {
	payload, err := h.service.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, services.ErrShareNotFound) {
			response.ErrorJSON(c, http.StatusNotFound, "分享不存在或已失效")
			return
		}
		logger.Error("分享快照：读取失败", zap.Error(err))
		response.ErrorJSON(c, 500, "读取分享失败")
		return
	}
	response.SuccessJSON(c, payload)
}
