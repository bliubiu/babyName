package handlers

import (
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"name/internal/application/response"
	"name/internal/application/services"
	"name/internal/application/validator"
	"name/internal/infrastructure/logger"
)

type NameHandler struct {
	service services.NameServiceInterface
}

func NewNameHandler(service services.NameServiceInterface) *NameHandler {
	return &NameHandler{service: service}
}

// validateGenerateRequest 通用请求解析与校验
// 解析 JSON 请求体，去除姓氏空格，校验姓氏/性别/日期/时间
// 校验失败时自动写入错误响应并返回 false
func (h *NameHandler) validateGenerateRequest(c *gin.Context) (*services.GenerateRequest, bool) {
	var req services.GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("NameHandler: invalid request", zap.Error(err))
		response.ErrorJSON(c, 400, "请求参数格式错误")
		return nil, false
	}

	// 去除姓氏首尾空格
	req.Surname = strings.TrimSpace(req.Surname)

	if err := validator.ValidateSurname(req.Surname); err != nil {
		logger.Warn("NameHandler: invalid surname", zap.Error(err), zap.String("surname", req.Surname))
		response.ErrorJSON(c, 400, err.Error())
		return nil, false
	}

	if err := validator.ValidateGender(req.Gender); err != nil {
		logger.Warn("NameHandler: invalid gender", zap.Error(err))
		response.ErrorJSON(c, 400, err.Error())
		return nil, false
	}

	if err := validator.ValidateDate(req.BirthYear, req.BirthMonth, req.BirthDay); err != nil {
		logger.Warn("NameHandler: invalid date", zap.Error(err))
		response.ErrorJSON(c, 400, err.Error())
		return nil, false
	}

	if err := validator.ValidateTime(req.BirthHour, req.BirthMinute); err != nil {
		logger.Warn("NameHandler: invalid time", zap.Error(err))
		response.ErrorJSON(c, 400, err.Error())
		return nil, false
	}

	return &req, true
}

// Generate 生成名字
func (h *NameHandler) Generate(c *gin.Context) {
	req, ok := h.validateGenerateRequest(c)
	if !ok {
		return
	}

	logger.Info("Generate: creating names",
		zap.String("surname", req.Surname),
		zap.String("gender", req.Gender),
	)
	result, err := h.service.Generate(c.Request.Context(), req)
	if err != nil {
		logger.Error("Generate: failed", zap.Error(err))
		response.ErrorJSON(c, 500, "生成名字失败，请稍后重试")
		return
	}

	response.SuccessJSON(c, result)
}

// GenerateWithAnalysis 生成带详细分析的名字
func (h *NameHandler) GenerateWithAnalysis(c *gin.Context) {
	req, ok := h.validateGenerateRequest(c)
	if !ok {
		return
	}

	logger.Info("GenerateWithAnalysis: creating names with analysis",
		zap.String("surname", req.Surname),
		zap.String("gender", req.Gender),
	)
	result, err := h.service.GenerateWithAnalysis(c.Request.Context(), req)
	if err != nil {
		logger.Error("GenerateWithAnalysis: failed", zap.Error(err))
		response.ErrorJSON(c, 500, "生成名字失败，请稍后重试")
		return
	}

	response.SuccessJSON(c, result)
}
