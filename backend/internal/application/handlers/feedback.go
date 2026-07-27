package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"name/internal/application/response"
	"name/internal/application/services"
	"name/internal/infrastructure/database"
	"name/internal/infrastructure/logger"
)

// FeedbackHandler 名字反馈处理
type FeedbackHandler struct {
	feedbackService services.FeedbackServiceInterface
}

// NewFeedbackHandler 创建反馈处理器
func NewFeedbackHandler(feedbackService services.FeedbackServiceInterface) *FeedbackHandler {
	return &FeedbackHandler{feedbackService: feedbackService}
}

// SaveFeedback 保存用户对名字的反馈
func (h *FeedbackHandler) SaveFeedback(c *gin.Context) {
	var req services.NameFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorJSON(c, http.StatusBadRequest, "请求参数格式错误")
		return
	}

	if req.GivenName == "" {
		response.ErrorJSON(c, http.StatusBadRequest, "given_name 不能为空")
		return
	}

	id, err := h.feedbackService.SaveNameFeedback(c.Request.Context(), &req)
	if err != nil {
		logger.Error("保存名字反馈失败", logger.String("given_name", req.GivenName), logger.ErrField(err))
		response.ErrorJSON(c, http.StatusInternalServerError, "保存反馈失败")
		return
	}

	response.SuccessJSON(c, gin.H{"id": id})
}

// SaveRequest 记录名字生成请求
func (h *FeedbackHandler) SaveRequest(c *gin.Context) {
	var req services.NameRequestRecord
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorJSON(c, http.StatusBadRequest, "请求参数格式错误")
		return
	}

	id, err := h.feedbackService.SaveNameRequest(c.Request.Context(), &req)
	if err != nil {
		logger.Error("保存名字请求记录失败", logger.ErrField(err))
		response.ErrorJSON(c, http.StatusInternalServerError, "保存失败")
		return
	}

	response.SuccessJSON(c, gin.H{"id": id})
}

// GetAlgorithmPerformance 获取算法性能统计
func (h *FeedbackHandler) GetAlgorithmPerformance(c *gin.Context) {
	results, err := h.feedbackService.GetAlgorithmPerformance(c.Request.Context())
	if err != nil {
		logger.Error("获取算法性能统计失败", logger.ErrField(err))
		response.ErrorJSON(c, http.StatusInternalServerError, "获取统计失败")
		return
	}

	if results == nil {
		results = []*database.AlgorithmPerformance{}
	}

	response.SuccessJSON(c, results)
}
