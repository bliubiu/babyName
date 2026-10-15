package handlers

import (
	"context"
	stderrors "errors"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	apperrors "name/internal/application/errors"
	"name/internal/application/response"
	"name/internal/application/services"
	"name/internal/application/validator"
	"name/internal/infrastructure/logger"
)

// writeGenerateError 生成失败时的统一错误响应
//
// 错误分三级，语义不可互相掩盖：
//   - 超时（context.DeadlineExceeded）：可重试 → 503；
//   - 业务错误（*errors.AppError）：透传 Code 与 Message。例如
//     「无法生成符合条件的名字，请调整筛选条件」是客户端可行动的错误，
//     必须返回 400 与原提示，让前端引导用户放宽条件；
//   - 其余未知错误：服务端缺陷 → 500 + 通用文案。
//
// 历史缺陷：原实现把非超时错误一律映射为 500 并替换成通用文案，业务层
// 明确的 400 被吞掉——前端拿不到可行动提示，监控上也多了一批假 500
// （空库回落场景实测：每次 /names/generate 都返回 500，掩盖了真实的
// 「物料缺失 → 候选池为空」业务语义）。
func writeGenerateError(c *gin.Context, err error, scope string) {
	logger.Error(scope+": failed", zap.Error(err))
	if stderrors.Is(err, context.DeadlineExceeded) {
		response.ErrorJSON(c, 503, "请求超时，请稍后重试")
		return
	}
	var appErr *apperrors.AppError
	if stderrors.As(err, &appErr) {
		response.ErrorJSON(c, int(appErr.Code), appErr.Message)
		return
	}
	response.ErrorJSON(c, 500, "生成名字失败，请稍后重试")
}

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
		writeGenerateError(c, err, "Generate")
		return
	}

	response.SuccessJSON(c, result)
}

// Evaluate 测名：输入姓名 + 生辰 → 完整评分报告 + 风险体检
func (h *NameHandler) Evaluate(c *gin.Context) {
	var req services.EvaluateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("Evaluate: invalid request", zap.Error(err))
		response.ErrorJSON(c, 400, "请求参数格式错误")
		return
	}

	req.Surname = strings.TrimSpace(req.Surname)
	req.GivenName = strings.TrimSpace(req.GivenName)

	if err := validator.ValidateSurname(req.Surname); err != nil {
		response.ErrorJSON(c, 400, err.Error())
		return
	}
	if err := validator.ValidateGender(req.Gender); err != nil {
		response.ErrorJSON(c, 400, err.Error())
		return
	}
	if err := validator.ValidateDate(req.BirthYear, req.BirthMonth, req.BirthDay); err != nil {
		response.ErrorJSON(c, 400, err.Error())
		return
	}
	if err := validator.ValidateTime(req.BirthHour, req.BirthMinute); err != nil {
		response.ErrorJSON(c, 400, err.Error())
		return
	}

	logger.Info("Evaluate: start",
		zap.String("full_name", req.Surname+req.GivenName),
		zap.String("gender", req.Gender),
	)
	result, err := h.service.Evaluate(c.Request.Context(), &req)
	if err != nil {
		writeGenerateError(c, err, "Evaluate")
		return
	}

	response.SuccessJSON(c, result)
}
