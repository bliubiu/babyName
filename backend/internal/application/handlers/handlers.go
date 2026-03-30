package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/bliubiu/babyName/internal/application/response"
	"github.com/bliubiu/babyName/internal/application/services"
	"github.com/bliubiu/babyName/internal/application/validator"
	"github.com/bliubiu/babyName/internal/domain/bazi"
	"github.com/bliubiu/babyName/internal/domain/namestat"
	"github.com/bliubiu/babyName/internal/infrastructure/logger"
	"go.uber.org/zap"
)

// @title 宝宝起名大师 API
// @version 1.0
// @description 宝宝起名大师后端API接口文档
// @BasePath /api/v1
// @Schemes http https
// @Accept json
// @Produce json
// @Security ApiKeyAuth


type NameStatHandler struct{}

func NewNameStatHandler() *NameStatHandler {
	return &NameStatHandler{}
}

// GetNameStats 获取名字统计信息
// @Summary 获取名字统计信息
// @Description 根据名字获取相关统计信息
// @Tags 名字统计
// @Accept json
// @Produce json
// @Param name path string true "名字"
// @Success 200 {object} namestat.NameStat "成功"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /namestat/{name} [get]
func (h *NameStatHandler) GetNameStats(c *gin.Context) {
	name := c.Param("name")
	logger.Info("GetNameStats called", zap.String("name", name))
	stats := namestat.GetNameStats(name)
	response.SuccessJSON(c, stats)
}

// GetTopNames 获取热门名字列表
// @Summary 获取热门名字列表
// @Description 获取热门名字排行榜
// @Tags 名字统计
// @Accept json
// @Produce json
// @Param limit query int false "限制数量，默认20，最大100"
// @Success 200 {array} namestat.NameStat "成功"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /namestat [get]
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

// GetProvinceStats 获取省份名字统计
// @Summary 获取省份名字统计
// @Description 根据名字和省份获取相关统计信息
// @Tags 名字统计
// @Accept json
// @Produce json
// @Param name path string true "名字"
// @Param province query string false "省份，默认北京"
// @Success 200 {object} namestat.NameStat "成功"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /namestat/{name}/province [get]
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

// Generate 生成名字
// @Summary 生成名字
// @Description 根据输入参数生成适合的名字
// @Tags 名字生成
// @Accept json
// @Produce json
// @Param request body services.GenerateRequest true "生成名字请求参数"
// @Success 200 {object} services.GenerateResponse "成功"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /names/generate [post]
func (h *NameHandler) Generate(c *gin.Context) {
	var req services.GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("Generate: invalid request", zap.Error(err))
		response.ErrorJSON(c, 400, "请求参数错误: "+err.Error())
		return
	}

	// 去除姓氏首尾空格
	req.Surname = strings.TrimSpace(req.Surname)
	logger.Info("Generate: received request", 
		zap.String("surname", req.Surname), 
		zap.Int("surname_length", len(req.Surname)),
		zap.String("generation", req.Generation),
		zap.String("generation_position", req.GenerationPosition),
		zap.String("name_type", req.NameType),
		zap.Int("name_length", req.NameLength),
	)
	if err := validator.ValidateSurname(req.Surname); err != nil {
		logger.Warn("Generate: invalid surname", zap.Error(err), zap.String("surname", req.Surname))
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

// GenerateWithAnalysis 生成带详细分析的名字
// @Summary 生成带详细分析的名字
// @Description 根据输入参数生成适合的名字，并提供详细的名字分析
// @Tags 名字生成
// @Accept json
// @Produce json
// @Param request body services.GenerateRequest true "生成名字请求参数"
// @Success 200 {object} services.GenerateWithAnalysisResponse "成功"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /names/generate/analysis [post]
func (h *NameHandler) GenerateWithAnalysis(c *gin.Context) {
	var req services.GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("GenerateWithAnalysis: invalid request", zap.Error(err))
		response.ErrorJSON(c, 400, "请求参数错误: "+err.Error())
		return
	}

	// 去除姓氏首尾空格
	req.Surname = strings.TrimSpace(req.Surname)
	logger.Info("GenerateWithAnalysis: received request",
		zap.String("surname", req.Surname),
		zap.Int("surname_length", len(req.Surname)),
		zap.String("generation", req.Generation),
		zap.String("generation_position", req.GenerationPosition),
		zap.String("name_type", req.NameType),
		zap.Int("name_length", req.NameLength),
	)
	if err := validator.ValidateSurname(req.Surname); err != nil {
		logger.Warn("GenerateWithAnalysis: invalid surname", zap.Error(err), zap.String("surname", req.Surname))
		response.ErrorJSON(c, 400, err.Error())
		return
	}

	if err := validator.ValidateGender(req.Gender); err != nil {
		logger.Warn("GenerateWithAnalysis: invalid gender", zap.Error(err))
		response.ErrorJSON(c, 400, err.Error())
		return
	}

	if err := validator.ValidateDate(req.BirthYear, req.BirthMonth, req.BirthDay); err != nil {
		logger.Warn("GenerateWithAnalysis: invalid date", zap.Error(err))
		response.ErrorJSON(c, 400, err.Error())
		return
	}

	if err := validator.ValidateTime(req.BirthHour, req.BirthMinute); err != nil {
		logger.Warn("GenerateWithAnalysis: invalid time", zap.Error(err))
		response.ErrorJSON(c, 400, err.Error())
		return
	}

	logger.Info("GenerateWithAnalysis: generating names with analysis",
		zap.String("surname", req.Surname),
		zap.String("gender", req.Gender),
	)
	result, err := h.service.GenerateWithAnalysis(&req)
	if err != nil {
		logger.LogError("GenerateWithAnalysis: failed to generate names", zap.Error(err))
		response.ErrorJSON(c, 500, "生成名字失败，请稍后重试")
		return
	}

	response.SuccessJSON(c, result)
}

type BaziHandler struct {
	service *services.BaziService
}

func NewBaziHandler(service *services.BaziService) *BaziHandler {
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
	result, err := h.service.GetHexagram(num)
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
	result, err := h.service.GetZodiac(animal)
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

// GetHistory 获取历史记录
// @Summary 获取历史记录
// @Description 获取所有历史记录
// @Tags 历史记录
// @Accept json
// @Produce json
// @Success 200 {array} services.HistoryRecord "成功"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /history [get]
func (h *HistoryHandler) GetHistory(c *gin.Context) {
	result, err := h.service.GetHistory()
	if err != nil {
		logger.LogError("GetHistory: failed", zap.Error(err))
		response.ErrorJSON(c, 500, "获取历史记录失败: "+err.Error())
		return
	}

	response.SuccessJSON(c, result)
}

// SaveHistory 保存历史记录
// @Summary 保存历史记录
// @Description 保存一条历史记录
// @Tags 历史记录
// @Accept json
// @Produce json
// @Param request body services.HistoryRecord true "历史记录请求参数"
// @Success 200 {object} object "成功，返回记录ID"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /history [post]
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

// DeleteHistory 删除历史记录
// @Summary 删除历史记录
// @Description 根据ID删除一条历史记录
// @Tags 历史记录
// @Accept json
// @Produce json
// @Param id path string true "历史记录ID"
// @Success 200 {object} object "成功"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /history/{id} [delete]
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
	result, err := h.service.GetFavorites()
	if err != nil {
		logger.LogError("GetFavorites: failed", zap.Error(err))
		response.ErrorJSON(c, 500, "获取收藏列表失败: "+err.Error())
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

	if err := h.service.DeleteFavorite(id); err != nil {
		logger.LogError("DeleteFavorite: failed", zap.Error(err))
		response.ErrorJSON(c, 500, "删除收藏失败: "+err.Error())
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

	isFavorite := h.service.CheckFavorite(surname, givenName)
	response.SuccessJSON(c, gin.H{"is_favorite": isFavorite})
}

// BatchSaveHistory 批量保存历史记录
// @Summary 批量保存历史记录
// @Description 批量保存多条历史记录
// @Tags 历史记录
// @Accept json
// @Produce json
// @Param request body []services.HistoryRecord true "历史记录请求参数列表"
// @Success 200 {object} object "成功，返回记录ID列表"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /history/batch [post]
func (h *HistoryHandler) BatchSaveHistory(c *gin.Context) {
	var reqs []services.HistoryRecord
	if err := c.ShouldBindJSON(&reqs); err != nil {
		logger.Warn("BatchSaveHistory: invalid request", zap.Error(err))
		response.ErrorJSON(c, 400, "请求参数错误: "+err.Error())
		return
	}

	// 转换为指针切片
	ptrReqs := make([]*services.HistoryRecord, len(reqs))
	for i := range reqs {
		ptrReqs[i] = &reqs[i]
	}

	ids, err := h.service.BatchSaveHistory(ptrReqs)
	if err != nil {
		logger.LogError("BatchSaveHistory: failed to save history", zap.Error(err))
		response.ErrorJSON(c, 500, "保存历史记录失败: "+err.Error())
		return
	}

	response.SuccessJSON(c, gin.H{"ids": ids})
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
		response.ErrorJSON(c, 400, "请求参数错误: "+err.Error())
		return
	}

	// 转换为指针切片
	ptrReqs := make([]*services.FavoriteRecord, len(reqs))
	for i := range reqs {
		ptrReqs[i] = &reqs[i]
	}

	ids, err := h.service.BatchSaveFavorite(ptrReqs)
	if err != nil {
		logger.LogError("BatchSaveFavorite: failed to save favorite", zap.Error(err))
		response.ErrorJSON(c, 500, "保存收藏失败: "+err.Error())
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
		response.ErrorJSON(c, 400, "请求参数错误: "+err.Error())
		return
	}

	if err := h.service.BatchDeleteFavorite(req.IDs); err != nil {
		logger.LogError("BatchDeleteFavorite: failed to delete favorite", zap.Error(err))
		response.ErrorJSON(c, 500, "删除收藏失败: "+err.Error())
		return
	}

	response.SuccessJSON(c, gin.H{"message": "批量删除成功"})
}

// HuangliHandler 黄历服务

type HuangliHandler struct {}

func NewHuangliHandler() *HuangliHandler {
	return &HuangliHandler{}
}

// GetHuangli 获取黄历信息
// @Summary 获取黄历信息
// @Description 根据日期获取黄历信息
// @Tags 黄历
// @Accept json
// @Produce json
// @Param year query int true "年份"
// @Param month query int true "月份"
// @Param day query int true "日期"
// @Success 200 {object} bazi.Huangli "成功"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /huangli [get]
func (h *HuangliHandler) GetHuangli(c *gin.Context) {
	yearStr := c.Query("year")
	monthStr := c.Query("month")
	dayStr := c.Query("day")

	year, err := strconv.Atoi(yearStr)
	if err != nil {
		response.ErrorJSON(c, 400, "年份参数错误")
		return
	}

	month, err := strconv.Atoi(monthStr)
	if err != nil || month < 1 || month > 12 {
		response.ErrorJSON(c, 400, "月份参数错误")
		return
	}

	day, err := strconv.Atoi(dayStr)
	if err != nil || day < 1 || day > 31 {
		response.ErrorJSON(c, 400, "日期参数错误")
		return
	}

	logger.Info("GetHuangli called", zap.Int("year", year), zap.Int("month", month), zap.Int("day", day))
	
	// 捕获可能的 panic
	defer func() {
		if r := recover(); r != nil {
			logger.LogError("GetHuangli panicked", zap.Any("recover", r))
			response.ErrorJSON(c, 500, "获取黄历信息失败，请稍后重试")
			return
		}
	}()
	
	huangli := bazi.GetHuangli(year, month, day)

	logger.Debug("Huangli data generated",
		zap.String("year", huangli.Year),
		zap.String("month", huangli.Month),
		zap.String("day", huangli.Day),
		zap.Strings("yi", huangli.Yi),
		zap.Strings("ji", huangli.Ji),
		zap.String("yearWuxing", huangli.YearWuxing),
		zap.String("monthWuxing", huangli.MonthWuxing),
		zap.String("dayWuxing", huangli.DayWuxing),
		zap.String("pengZhu", huangli.PengZhu),
		zap.String("jieShen", huangli.JieShen),
		zap.String("fu", huangli.Fu),
	)
	response.SuccessJSON(c, huangli)
}

// GetLunarCalendar 获取农历信息
// @Summary 获取农历信息
// @Description 根据日期获取农历信息
// @Tags 黄历
// @Accept json
// @Produce json
// @Param year query int true "年份"
// @Param month query int true "月份"
// @Param day query int true "日期"
// @Param hour query int false "小时"
// @Param minute query int false "分钟"
// @Success 200 {object} bazi.LunarCalendar "成功"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /lunar [get]
func (h *HuangliHandler) GetLunarCalendar(c *gin.Context) {
	yearStr := c.Query("year")
	monthStr := c.Query("month")
	dayStr := c.Query("day")
	hourStr := c.DefaultQuery("hour", "0")
	minuteStr := c.DefaultQuery("minute", "0")

	year, err := strconv.Atoi(yearStr)
	if err != nil {
		response.ErrorJSON(c, 400, "年份参数错误")
		return
	}

	month, err := strconv.Atoi(monthStr)
	if err != nil || month < 1 || month > 12 {
		response.ErrorJSON(c, 400, "月份参数错误")
		return
	}

	day, err := strconv.Atoi(dayStr)
	if err != nil || day < 1 || day > 31 {
		response.ErrorJSON(c, 400, "日期参数错误")
		return
	}

	hour, err := strconv.Atoi(hourStr)
	if err != nil || hour < 0 || hour > 23 {
		response.ErrorJSON(c, 400, "小时参数错误")
		return
	}

	minute, err := strconv.Atoi(minuteStr)
	if err != nil || minute < 0 || minute > 59 {
		response.ErrorJSON(c, 400, "分钟参数错误")
		return
	}

	logger.Info("GetLunarCalendar called", zap.Int("year", year), zap.Int("month", month), zap.Int("day", day), zap.Int("hour", hour), zap.Int("minute", minute))
	lunar := bazi.GetLunarCalendar(year, month, day, hour, minute)
	
	// 调试日志：输出农历数据详情
	logger.Debug("Lunar calendar data generated",
		zap.String("yearGan", lunar.YearGan),
		zap.String("yearZhi", lunar.YearZhi),
		zap.String("monthGan", lunar.MonthGan),
		zap.String("monthZhi", lunar.MonthZhi),
		zap.String("dayGan", lunar.DayGan),
		zap.String("dayZhi", lunar.DayZhi),
		zap.String("yearNayin", lunar.YearNayin),
		zap.String("monthNayin", lunar.MonthNayin),
		zap.String("dayNayin", lunar.DayNayin),
	)
	
	response.SuccessJSON(c, lunar)
}

type ReportHandler struct {
	service *services.ReportService
}

func NewReportHandler(service *services.ReportService) *ReportHandler {
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
