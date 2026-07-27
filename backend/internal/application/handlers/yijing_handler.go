package handlers

import (
	"github.com/gin-gonic/gin"
	"name/internal/application/response"
	"name/internal/application/services"
	"name/internal/infrastructure/logger"
	"go.uber.org/zap"
)

type YijingHandler struct {
	service services.YijingServiceInterface
}

func NewYijingHandler(service services.YijingServiceInterface) *YijingHandler {
	return &YijingHandler{service: service}
}

// GetHexagram 获取卦象
// @Summary 获取卦象
// @Description 根据卦象编号获取卦象信息
// @Tags 易经
// @Accept json
// @Produce json
// @Param id path string true "卦象编号"
// @Success 200 {object} yijing.Hexagram "成功"
// @Failure 404 {object} response.Response "卦象不存在"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /yijing/hexagram/{id} [get]
func (h *YijingHandler) GetHexagram(c *gin.Context) {
	id := c.Param("id")
	var num int
	for _, ch := range id {
		if ch >= '0' && ch <= '9' {
			num = num*10 + int(ch-'0')
		}
	}

	logger.Debug("GetHexagram", zap.Int("number", num))
	result, err := h.service.GetHexagram(c.Request.Context(), num)
	if err != nil || result == nil {
		logger.Warn("GetHexagram: not found", zap.Int("number", num))
		response.ErrorJSON(c, 404, "卦象不存在")
		return
	}

	response.SuccessJSON(c, result)
}

// GetAllHexagrams 获取所有卦象
// @Summary 获取所有卦象
// @Description 获取所有64卦象的列表
// @Tags 易经
// @Accept json
// @Produce json
// @Success 200 {array} yijing.Hexagram "成功"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /yijing/hexagram [get]
func (h *YijingHandler) GetAllHexagrams(c *gin.Context) {
	result, err := h.service.GetAllHexagrams(c.Request.Context())
	if err != nil {
		logger.Error("GetAllHexagrams: failed", zap.Error(err))
		response.ErrorJSON(c, 500, "获取卦象列表失败")
		return
	}

	response.SuccessJSON(c, result)
}
