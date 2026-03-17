package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/namemaster/backend/internal/application/response"
	"github.com/namemaster/backend/internal/application/services"
	"github.com/namemaster/backend/internal/application/validator"
	"github.com/namemaster/backend/internal/domain/namestat"
	"github.com/namemaster/backend/internal/infrastructure/logger"
	"go.uber.org/zap"
)

type NameStatHandler struct{}

func NewNameStatHandler() *NameStatHandler {
	return &NameStatHandler{}
}

func (h *NameStatHandler) GetNameStats(c *gin.Context) {
	name := c.Param("name")
	logger.Info("GetNameStats called", zap.String("name", name))
	stats := namestat.GetNameStats(name)
	response.SuccessJSON(c, stats)
}

func (h *NameStatHandler) GetTopNames(c *gin.Context) {
	limit := 20
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	logger.Info("GetTopNames called", zap.Int("limit", limit))
	stats := namestat.GetTopNames(limit)
	response.SuccessJSON(c, stats)
}

func (h *NameStatHandler) GetProvinceStats(c *gin.Context) {
	name := c.Param("name")
	province := c.DefaultQuery("province", "北京")
	logger.Info("GetProvinceStats called", zap.String("name", name), zap.String("province", province))
	stats := namestat.GetProvinceNameStats(name, province)
	response.SuccessJSON(c, stats)
}

type NameHandler struct {
	service *services.NameService
}

func NewNameHandler(service *services.NameService) *NameHandler {
	return &NameHandler{service: service}
}

func (h *NameHandler) Generate(c *gin.Context) {
	var req services.GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("Generate: invalid request", zap.Error(err))
		response.ErrorJSON(c, 400, "请求参数错误: "+err.Error())
		return
	}

	if err := validator.ValidateSurname(req.Surname); err != nil {
		logger.Warn("Generate: invalid surname", zap.Error(err))
		response.ErrorJSON(c, 400, err.Error())
		return
	}

	if err := validator.ValidateGender(req.Gender); err != nil {
		logger.Warn("Generate: invalid gender", zap.Error(err))
		response.ErrorJSON(c, 400, err.Error())
		return
	}

	if err := validator.ValidateDate(req.BirthYear, req.BirthMonth, req.BirthDay); err != nil {
		logger.Warn("Generate: invalid date", zap.Error(err))
		response.ErrorJSON(c, 400, err.Error())
		return
	}

	if err := validator.ValidateTime(req.BirthHour, req.BirthMinute); err != nil {
		logger.Warn("Generate: invalid time", zap.Error(err))
		response.ErrorJSON(c, 400, err.Error())
		return
	}

	logger.Info("Generate: generating names",
		zap.String("surname", req.Surname),
		zap.String("gender", req.Gender),
	)
	result, err := h.service.Generate(&req)
	if err != nil {
		logger.LogError("Generate: failed to generate names", zap.Error(err))
		response.ErrorJSON(c, 500, "生成名字失败，请稍后重试")
		return
	}

	response.SuccessJSON(c, result)
}

func (h *NameHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	response.SuccessJSON(c, gin.H{"id": id})
}

type BaziHandler struct {
	service *services.BaziService
}

func NewBaziHandler(service *services.BaziService) *BaziHandler {
	return &BaziHandler{service: service}
}

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
		response.ErrorJSON(c, 400, "请求参数错误: "+err.Error())
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
	result, err := h.service.Analyze(req.Year, req.Month, req.Day, req.Hour, req.Minute)
	if err != nil {
		logger.LogError("Bazi Analyze: failed", zap.Error(err))
		response.ErrorJSON(c, 500, "八字分析失败，请稍后重试")
		return
	}

	response.SuccessJSON(c, result)
}

type YijingHandler struct {
	service *services.YijingService
}

func NewYijingHandler(service *services.YijingService) *YijingHandler {
	return &YijingHandler{service: service}
}

func (h *YijingHandler) GetHexagram(c *gin.Context) {
	id := c.Param("id")
	var num int
	for _, ch := range id {
		if ch >= '0' && ch <= '9' {
			num = num*10 + int(ch-'0')
		}
	}

	logger.Debug("GetHexagram", zap.Int("number", num))
	result, err := h.service.GetHexagram(num)
	if err != nil || result == nil {
		logger.Warn("GetHexagram: not found", zap.Int("number", num))
		response.ErrorJSON(c, 404, "卦象不存在")
		return
	}

	response.SuccessJSON(c, result)
}

func (h *YijingHandler) GetAllHexagrams(c *gin.Context) {
	result, err := h.service.GetAllHexagrams()
	if err != nil {
		logger.LogError("GetAllHexagrams: failed", zap.Error(err))
		response.ErrorJSON(c, 500, "获取卦象列表失败: "+err.Error())
		return
	}

	response.SuccessJSON(c, result)
}

type ZodiacHandler struct {
	service *services.ZodiacService
}

func NewZodiacHandler(service *services.ZodiacService) *ZodiacHandler {
	return &ZodiacHandler{service: service}
}

func (h *ZodiacHandler) GetZodiac(c *gin.Context) {
	animal := c.Param("animal")

	logger.Debug("GetZodiac", zap.String("animal", animal))
	result, err := h.service.GetZodiac(animal)
	if err != nil || result == nil {
		logger.Warn("GetZodiac: not found", zap.String("animal", animal))
		response.ErrorJSON(c, 404, "生肖不存在")
		return
	}

	response.SuccessJSON(c, result)
}

func (h *ZodiacHandler) GetAllZodiacs(c *gin.Context) {
	result, err := h.service.GetAllZodiacs()
	if err != nil {
		logger.LogError("GetAllZodiacs: failed", zap.Error(err))
		response.ErrorJSON(c, 500, "获取生肖列表失败: "+err.Error())
		return
	}

	response.SuccessJSON(c, result)
}

type HistoryHandler struct {
	service *services.HistoryService
}

func NewHistoryHandler(service *services.HistoryService) *HistoryHandler {
	return &HistoryHandler{service: service}
}

func (h *HistoryHandler) GetHistory(c *gin.Context) {
	result, err := h.service.GetHistory()
	if err != nil {
		logger.LogError("GetHistory: failed", zap.Error(err))
		response.ErrorJSON(c, 500, "获取历史记录失败: "+err.Error())
		return
	}

	response.SuccessJSON(c, result)
}

func (h *HistoryHandler) SaveHistory(c *gin.Context) {
	var req services.HistoryRecord
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("SaveHistory: invalid request", zap.Error(err))
		response.ErrorJSON(c, 400, "请求参数错误: "+err.Error())
		return
	}

	id, err := h.service.SaveHistory(&req)
	if err != nil {
		logger.LogError("SaveHistory: failed", zap.Error(err))
		response.ErrorJSON(c, 500, "保存历史记录失败: "+err.Error())
		return
	}

	response.SuccessJSON(c, gin.H{"id": id})
}

func (h *HistoryHandler) DeleteHistory(c *gin.Context) {
	id := c.Param("id")

	if err := h.service.DeleteHistory(id); err != nil {
		logger.LogError("DeleteHistory: failed", zap.Error(err))
		response.ErrorJSON(c, 500, "删除历史记录失败: "+err.Error())
		return
	}

	response.SuccessJSON(c, gin.H{"message": "删除成功"})
}

type FavoriteHandler struct {
	service *services.FavoriteService
}

func NewFavoriteHandler(service *services.FavoriteService) *FavoriteHandler {
	return &FavoriteHandler{service: service}
}

func (h *FavoriteHandler) GetFavorites(c *gin.Context) {
	result, err := h.service.GetFavorites()
	if err != nil {
		logger.LogError("GetFavorites: failed", zap.Error(err))
		response.ErrorJSON(c, 500, "获取收藏列表失败: "+err.Error())
		return
	}

	response.SuccessJSON(c, result)
}

func (h *FavoriteHandler) SaveFavorite(c *gin.Context) {
	var req services.FavoriteRecord
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("SaveFavorite: invalid request", zap.Error(err))
		response.ErrorJSON(c, 400, "请求参数错误: "+err.Error())
		return
	}

	id, err := h.service.SaveFavorite(&req)
	if err != nil {
		logger.LogError("SaveFavorite: failed", zap.Error(err))
		response.ErrorJSON(c, 500, "保存收藏失败: "+err.Error())
		return
	}

	response.SuccessJSON(c, gin.H{"id": id})
}

func (h *FavoriteHandler) DeleteFavorite(c *gin.Context) {
	id := c.Param("id")

	if err := h.service.DeleteFavorite(id); err != nil {
		logger.LogError("DeleteFavorite: failed", zap.Error(err))
		response.ErrorJSON(c, 500, "删除收藏失败: "+err.Error())
		return
	}

	response.SuccessJSON(c, gin.H{"message": "删除成功"})
}

func (h *FavoriteHandler) CheckFavorite(c *gin.Context) {
	surname := c.Query("surname")
	givenName := c.Query("given_name")

	if surname == "" || givenName == "" {
		response.ErrorJSON(c, 400, "缺少参数")
		return
	}

	isFavorite := h.service.CheckFavorite(surname, givenName)
	response.SuccessJSON(c, gin.H{"is_favorite": isFavorite})
}

type ReportHandler struct {
	service *services.ReportService
}

func NewReportHandler(service *services.ReportService) *ReportHandler {
	return &ReportHandler{service: service}
}

func (h *ReportHandler) GeneratePDF(c *gin.Context) {
	var req struct {
		Data interface{} `json:"data"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("GeneratePDF: invalid request", zap.Error(err))
		response.ErrorJSON(c, 400, "请求参数错误: "+err.Error())
		return
	}

	result, err := h.service.GeneratePDF(req.Data)
	if err != nil {
		logger.LogError("GeneratePDF: failed", zap.Error(err))
		response.ErrorJSON(c, 500, "生成PDF失败: "+err.Error())
		return
	}

	c.Data(http.StatusOK, "application/pdf", result)
}

func (h *ReportHandler) GenerateHTML(c *gin.Context) {
	var req struct {
		Data interface{} `json:"data"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("GenerateHTML: invalid request", zap.Error(err))
		response.ErrorJSON(c, 400, "请求参数错误: "+err.Error())
		return
	}

	result, err := h.service.GenerateHTML(req.Data)
	if err != nil {
		logger.LogError("GenerateHTML: failed", zap.Error(err))
		response.ErrorJSON(c, 500, "生成HTML失败: "+err.Error())
		return
	}

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(result))
}
