package handlers

import (
	"github.com/gin-gonic/gin"
	"name/internal/application/response"
	"name/internal/application/services"
	"name/internal/application/validator"
	"name/internal/infrastructure/logger"
	"go.uber.org/zap"
)

type BaziHandler struct {
	service services.BaziServiceInterface
}

func NewBaziHandler(service services.BaziServiceInterface) *BaziHandler {
	return &BaziHandler{service: service}
}

// Analyze 分析八字
// @Summary 分析八字
// @Description 根据出生日期和时间分析八字
// @Tags 八字分析
// @Accept json
// @Produce json
// @Param request body object true "八字分析请求参数"
// @Success 200 {object} bazi.BaziAnalysis "成功"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /bazi/analyze [post]
func (h *BaziHandler) Analyze(c *gin.Context) {
	var req struct {
		Year   int `json:"year" binding:"required"`
		Month  int `json:"month" binding:"required"`
		Day    int `json:"day" binding:"required"`
		Hour   int `json:"hour" binding:"required"`
		Minute int `json:"minute"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("Bazi Analyze: invalid request", zap.Error(err))
		response.ErrorJSON(c, 400, "请求参数格式错误")
		return
	}

	if err := validator.ValidateDate(req.Year, req.Month, req.Day); err != nil {
		logger.Warn("Bazi Analyze: invalid date", zap.Error(err))
		response.ErrorJSON(c, 400, err.Error())
		return
	}

	if err := validator.ValidateTime(req.Hour, req.Minute); err != nil {
		logger.Warn("Bazi Analyze: invalid time", zap.Error(err))
		response.ErrorJSON(c, 400, err.Error())
		return
	}

	logger.Info("Bazi Analyze",
		zap.Int("year", req.Year),
		zap.Int("month", req.Month),
		zap.Int("day", req.Day),
		zap.Int("hour", req.Hour),
	)
	result, err := h.service.Analyze(c.Request.Context(), req.Year, req.Month, req.Day, req.Hour, req.Minute)
	if err != nil {
		logger.Error("Bazi Analyze: failed", zap.Error(err))
		response.ErrorJSON(c, 500, "八字分析失败，请稍后重试")
		return
	}

	response.SuccessJSON(c, result)
}
