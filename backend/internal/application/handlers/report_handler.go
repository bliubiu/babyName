package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"name/internal/application/response"
	"name/internal/application/services"
	"name/internal/infrastructure/logger"
	"go.uber.org/zap"
)

type ReportHandler struct {
	service services.ReportServiceInterface
}

func NewReportHandler(service services.ReportServiceInterface) *ReportHandler {
	return &ReportHandler{service: service}
}

// GeneratePDF 生成PDF报告
// @Summary 生成PDF报告
// @Description 根据数据生成PDF格式的报告
// @Tags 报告
// @Accept json
// @Produce application/pdf
// @Param request body object true "报告数据"
// @Success 200 {file} file "PDF文件"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /report/pdf [post]
func (h *ReportHandler) GeneratePDF(c *gin.Context) {
	var req struct {
		Data interface{} `json:"data"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("GeneratePDF: invalid request", zap.Error(err))
		response.ErrorJSON(c, 400, "请求参数格式错误")
		return
	}

	result, err := h.service.GeneratePDF(c.Request.Context(), req.Data)
	if err != nil {
		logger.Error("GeneratePDF: failed", zap.Error(err))
		response.ErrorJSON(c, 500, "生成PDF失败")
		return
	}

	c.Data(http.StatusOK, "application/pdf", result)
}

// GenerateHTML 生成HTML报告
// @Summary 生成HTML报告
// @Description 根据数据生成HTML格式的报告
// @Tags 报告
// @Accept json
// @Produce text/html
// @Param request body object true "报告数据"
// @Success 200 {string} string "HTML内容"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /report/html [post]
func (h *ReportHandler) GenerateHTML(c *gin.Context) {
	var req struct {
		Data interface{} `json:"data"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("GenerateHTML: invalid request", zap.Error(err))
		response.ErrorJSON(c, 400, "请求参数格式错误")
		return
	}

	result, err := h.service.GenerateHTML(c.Request.Context(), req.Data)
	if err != nil {
		logger.Error("GenerateHTML: failed", zap.Error(err))
		response.ErrorJSON(c, 500, "生成HTML失败")
		return
	}

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(result))
}
