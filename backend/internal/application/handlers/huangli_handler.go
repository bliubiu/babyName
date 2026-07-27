package handlers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"name/internal/application/response"
	"name/internal/domain/bazi"
	"name/internal/infrastructure/logger"
	"go.uber.org/zap"
)

// HuangliHandler 黄历服务
type HuangliHandler struct{}

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
			logger.Error("GetHuangli panicked", zap.Any("recover", r))
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
