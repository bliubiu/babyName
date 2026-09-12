package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"name/internal/application/response"
	"name/internal/domain/namestatistics"
	"name/internal/infrastructure/logger"
)

type NameStatisticsHandler struct {
	service namestatistics.NameStatisticsService
}

func NewNameStatisticsHandler(service namestatistics.NameStatisticsService) *NameStatisticsHandler {
	return &NameStatisticsHandler{service: service}
}

// GetSurnameStats 获取姓氏排行
// @Summary 获取姓氏排行
// @Description 获取姓氏使用人数排行榜（含性别比例）
// @Tags 姓名统计
// @Accept json
// @Produce json
// @Param limit query int false "限制数量，默认20，最大200"
// @Success 200 {array} database.SurnameStat "成功"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /namestats/surnames [get]
func (h *NameStatisticsHandler) GetSurnameStats(c *gin.Context) {
	limit := 20
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}

	logger.Info("GetSurnameStats called", zap.Int("limit", limit))
	stats, err := h.service.GetSurnameStats(limit)
	if err != nil {
		logger.Error("GetSurnameStats failed", zap.Error(err))
		response.ErrorJSON(c, 500, "获取姓氏统计失败")
		return
	}
	response.SuccessJSON(c, stats)
}

// GetSurnameStat 获取单个姓氏统计
// @Summary 获取单个姓氏统计
// @Description 获取指定姓氏的详细统计信息
// @Tags 姓名统计
// @Accept json
// @Produce json
// @Param surname path string true "姓氏"
// @Success 200 {object} database.SurnameStat "成功"
// @Failure 404 {object} response.Response "未找到"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /namestats/surnames/{surname} [get]
func (h *NameStatisticsHandler) GetSurnameStat(c *gin.Context) {
	surname := c.Param("surname")
	if surname == "" {
		response.ErrorJSON(c, 400, "姓氏不能为空")
		return
	}

	logger.Info("GetSurnameStat called", zap.String("surname", surname))
	stat, err := h.service.GetSurnameStat(surname)
	if err != nil {
		logger.Error("GetSurnameStat failed", zap.Error(err))
		response.ErrorJSON(c, 500, "获取姓氏统计失败")
		return
	}
	if stat == nil {
		response.ErrorJSON(c, 404, "未找到该姓氏统计")
		return
	}
	response.SuccessJSON(c, stat)
}

// GetGivenNameStats 获取指定姓氏下的名字排行
// @Summary 获取指定姓氏下的名字排行
// @Description 获取指定姓氏下名字部分的使用排行
// @Tags 姓名统计
// @Accept json
// @Produce json
// @Param surname path string true "姓氏"
// @Param limit query int false "限制数量，默认20，最大100"
// @Success 200 {array} database.GivenNameStat "成功"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /namestats/surnames/{surname}/given-names [get]
func (h *NameStatisticsHandler) GetGivenNameStats(c *gin.Context) {
	surname := c.Param("surname")
	if surname == "" {
		response.ErrorJSON(c, 400, "姓氏不能为空")
		return
	}

	limit := 20
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	logger.Info("GetGivenNameStats called", zap.String("surname", surname), zap.Int("limit", limit))
	stats, err := h.service.GetGivenNameStats(surname, limit)
	if err != nil {
		logger.Error("GetGivenNameStats failed", zap.Error(err))
		response.ErrorJSON(c, 500, "获取名字统计失败")
		return
	}
	response.SuccessJSON(c, stats)
}

// GetFullNameStats 获取指定姓氏下的全名排行
// @Summary 获取指定姓氏下的全名排行
// @Description 获取指定姓氏下完整姓名的使用排行（含性别分布）
// @Tags 姓名统计
// @Accept json
// @Produce json
// @Param surname path string true "姓氏"
// @Param limit query int false "限制数量，默认20，最大100"
// @Success 200 {array} database.FullNameStat "成功"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /namestats/surnames/{surname}/full-names [get]
func (h *NameStatisticsHandler) GetFullNameStats(c *gin.Context) {
	surname := c.Param("surname")
	if surname == "" {
		response.ErrorJSON(c, 400, "姓氏不能为空")
		return
	}

	limit := 20
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	logger.Info("GetFullNameStats called", zap.String("surname", surname), zap.Int("limit", limit))
	stats, err := h.service.GetFullNameStats(surname, limit)
	if err != nil {
		logger.Error("GetFullNameStats failed", zap.Error(err))
		response.ErrorJSON(c, 500, "获取全名统计失败")
		return
	}
	response.SuccessJSON(c, stats)
}

// GetFullNameStat 获取全名统计
// @Summary 获取全名统计
// @Description 获取指定全名的详细统计信息（含性别分布、排名）
// @Tags 姓名统计
// @Accept json
// @Produce json
// @Param full_name path string true "全名"
// @Success 200 {object} database.FullNameStat "成功"
// @Failure 404 {object} response.Response "未找到"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /namestats/full-names/{full_name} [get]
func (h *NameStatisticsHandler) GetFullNameStat(c *gin.Context) {
	fullName := c.Param("full_name")
	if fullName == "" {
		response.ErrorJSON(c, 400, "全名不能为空")
		return
	}

	logger.Info("GetFullNameStat called", zap.String("full_name", fullName))
	stat, err := h.service.GetFullNameStat(fullName)
	if err != nil {
		logger.Error("GetFullNameStat failed", zap.Error(err))
		response.ErrorJSON(c, 500, "获取全名统计失败")
		return
	}
	if stat == nil {
		response.ErrorJSON(c, 404, "未找到该全名统计")
		return
	}
	response.SuccessJSON(c, stat)
}

// GetNameGenderStats 获取名字性别分布
// @Summary 获取名字性别分布
// @Description 获取指定名字（去姓氏）的性别使用分布
// @Tags 姓名统计
// @Accept json
// @Produce json
// @Param name path string true "名字（去姓氏）"
// @Success 200 {object} database.NameGenderStat "成功"
// @Failure 404 {object} response.Response "未找到"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /namestats/names/{name}/gender [get]
func (h *NameStatisticsHandler) GetNameGenderStats(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		response.ErrorJSON(c, 400, "名字不能为空")
		return
	}

	logger.Info("GetNameGenderStats called", zap.String("name", name))
	stat, err := h.service.GetNameGenderStats(name)
	if err != nil {
		logger.Error("GetNameGenderStats failed", zap.Error(err))
		response.ErrorJSON(c, 500, "获取名字性别统计失败")
		return
	}
	if stat == nil {
		response.ErrorJSON(c, 404, "未找到该名字性别统计")
		return
	}
	response.SuccessJSON(c, stat)
}

// GetTopFullNames 获取热门全名排行
// @Summary 获取热门全名排行
// @Description 获取全国范围内热门全名排行榜
// @Tags 姓名统计
// @Accept json
// @Produce json
// @Param limit query int false "限制数量，默认20，最大100"
// @Success 200 {array} database.FullNameStat "成功"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /namestats/top [get]
func (h *NameStatisticsHandler) GetTopFullNames(c *gin.Context) {
	limit := 20
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}

	logger.Info("GetTopFullNames called", zap.Int("limit", limit))
	stats, err := h.service.GetTopFullNames(limit)
	if err != nil {
		logger.Error("GetTopFullNames failed", zap.Error(err))
		response.ErrorJSON(c, 500, "获取热门全名失败")
		return
	}
	response.SuccessJSON(c, stats)
}

// GetTotalNameCount 获取语料库总人名数
// @Summary 获取语料库总人名数
// @Description 获取Chinese-Names-Corpus语料库的总人名数量
// @Tags 姓名统计
// @Accept json
// @Produce json
// @Success 200 {object} map[string]int "成功"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /namestats/total [get]
func (h *NameStatisticsHandler) GetTotalNameCount(c *gin.Context) {
	count, err := h.service.GetTotalNameCount()
	if err != nil {
		logger.Error("GetTotalNameCount failed", zap.Error(err))
		response.ErrorJSON(c, 500, "获取总数失败")
		return
	}
	response.SuccessJSON(c, map[string]int{"total_names": count})
}
