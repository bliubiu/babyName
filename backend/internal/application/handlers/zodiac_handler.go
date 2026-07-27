package handlers

import (
	"github.com/gin-gonic/gin"
	"name/internal/application/response"
	"name/internal/application/services"
	"name/internal/infrastructure/logger"
	"go.uber.org/zap"
)

type ZodiacHandler struct {
	service services.ZodiacServiceInterface
}

func NewZodiacHandler(service services.ZodiacServiceInterface) *ZodiacHandler {
	return &ZodiacHandler{service: service}
}

// GetZodiac 获取生肖
// @Summary 获取生肖
// @Description 根据生肖名称获取生肖信息
// @Tags 生肖
// @Accept json
// @Produce json
// @Param animal path string true "生肖名称"
// @Success 200 {object} zodiac.Zodiac "成功"
// @Failure 404 {object} response.Response "生肖不存在"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /zodiac/{animal} [get]
func (h *ZodiacHandler) GetZodiac(c *gin.Context) {
	animal := c.Param("animal")

	logger.Debug("GetZodiac", zap.String("animal", animal))
	result, err := h.service.GetZodiac(c.Request.Context(), animal)
	if err != nil || result == nil {
		logger.Warn("GetZodiac: not found", zap.String("animal", animal))
		response.ErrorJSON(c, 404, "生肖不存在")
		return
	}

	response.SuccessJSON(c, result)
}

// GetAllZodiacs 获取所有生肖
// @Summary 获取所有生肖
// @Description 获取所有12生肖的列表
// @Tags 生肖
// @Accept json
// @Produce json
// @Success 200 {array} zodiac.Zodiac "成功"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /zodiac [get]
func (h *ZodiacHandler) GetAllZodiacs(c *gin.Context) {
	result, err := h.service.GetAllZodiacs(c.Request.Context())
	if err != nil {
		logger.Error("GetAllZodiacs: failed", zap.Error(err))
		response.ErrorJSON(c, 500, "获取生肖列表失败")
		return
	}

	response.SuccessJSON(c, result)
}
