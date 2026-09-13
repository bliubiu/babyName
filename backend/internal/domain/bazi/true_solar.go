package bazi

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// CityLongitudes 常见出生地经度（东经为正，近似市中心）
var CityLongitudes = map[string]float64{
	"北京": 116.41, "上海": 121.47, "天津": 117.20, "重庆": 106.55,
	"广州": 113.26, "深圳": 114.06, "杭州": 120.15, "南京": 118.80,
	"成都": 104.07, "武汉": 114.31, "西安": 108.95, "沈阳": 123.43,
	"哈尔滨": 126.53, "长春": 125.32, "济南": 117.00, "青岛": 120.38,
	"郑州": 113.63, "长沙": 112.94, "福州": 119.30, "厦门": 118.09,
	"昆明": 102.83, "贵阳": 106.63, "兰州": 103.83, "乌鲁木齐": 87.62,
	"拉萨": 91.11, "呼和浩特": 111.75, "银川": 106.23, "西宁": 101.78,
	"石家庄": 114.51, "太原": 112.55, "合肥": 117.28, "南昌": 115.86,
	"南宁": 108.37, "海口": 110.33, "香港": 114.17, "澳门": 113.55,
	"台北": 121.52, "苏州": 120.58, "无锡": 120.30, "宁波": 121.55,
	"温州": 120.70, "佛山": 113.12, "东莞": 113.75, "天津滨海": 117.70,
}

// DefaultStandardMeridian 东八区标准经线
const DefaultStandardMeridian = 120.0

// LookupLongitude 按出生地名解析经度；未收录返回 (0, false)
func LookupLongitude(location string) (float64, bool) {
	loc := strings.TrimSpace(location)
	if loc == "" {
		return 0, false
	}
	if lon, ok := CityLongitudes[loc]; ok {
		return lon, true
	}
	// 前缀匹配：如「北京市」「成都·武侯」
	for city, lon := range CityLongitudes {
		if strings.HasPrefix(loc, city) {
			return lon, true
		}
	}
	return 0, false
}

// EquationOfTimeMinutes 近似均时差（分钟），Bourke 简化公式
// dayOfYear 为年内序号（1–366）
func EquationOfTimeMinutes(dayOfYear int) float64 {
	if dayOfYear < 1 {
		dayOfYear = 1
	}
	b := 2 * math.Pi * float64(dayOfYear-81) / 364.0
	return 9.87*math.Sin(2*b) - 7.53*math.Cos(b) - 1.5*math.Sin(b)
}

// CorrectTrueSolarTime 将钟表时间校正为真太阳时（东经）
// longitude 为出生地经度（度）；standardMeridian 标准经线，传 0 用 120°
// 返回校正后的年月日时分（可能跨日）
func CorrectTrueSolarTime(year, month, day, hour, minute int, longitude, standardMeridian float64) (int, int, int, int, int) {
	if standardMeridian == 0 {
		standardMeridian = DefaultStandardMeridian
	}
	// 经度差 → 分钟：每度 4 分钟
	offsetMin := (longitude - standardMeridian) * 4.0

	t := time.Date(year, time.Month(month), day, hour, minute, 0, 0, time.UTC)
	// 年内序号（粗略，足够均时差）
	dayOfYear := t.YearDay()
	offsetMin += EquationOfTimeMinutes(dayOfYear)

	corrected := t.Add(time.Duration(math.Round(offsetMin) * float64(time.Minute)))
	return corrected.Year(), int(corrected.Month()), corrected.Day(), corrected.Hour(), corrected.Minute()
}

// TrueSolarInfo 真太阳时校正结果
type TrueSolarInfo struct {
	Enabled       bool    `json:"enabled"`
	Longitude     float64 `json:"longitude"`
	OffsetMinutes float64 `json:"offset_minutes"`
	Original      string  `json:"original"`
	Corrected     string  `json:"corrected"`
	CrossDay      bool    `json:"cross_day"`
	CrossShichen  bool    `json:"cross_shichen"`
}

// ApplyTrueSolar 根据出生地（或显式经度）校正出生时刻
// location 用于查表；explicitLon > 0 时优先使用显式经度
func ApplyTrueSolar(year, month, day, hour, minute int, location string, explicitLon float64) (int, int, int, int, int, *TrueSolarInfo) {
	lon := explicitLon
	fromTable := false
	if lon == 0 {
		lon, fromTable = LookupLongitude(location)
		if !fromTable {
			info := &TrueSolarInfo{
				Enabled:   false,
				Original:  fmt.Sprintf("%04d-%02d-%02d %02d:%02d", year, month, day, hour, minute),
				Corrected: fmt.Sprintf("%04d-%02d-%02d %02d:%02d", year, month, day, hour, minute),
			}
			return year, month, day, hour, minute, info
		}
	}

	y2, m2, d2, h2, mi2 := CorrectTrueSolarTime(year, month, day, hour, minute, lon, 0)
	offset := (lon - DefaultStandardMeridian) * 4.0
	origShichen := GetShichen(hour)
	newShichen := GetShichen(h2)
	info := &TrueSolarInfo{
		Enabled:       true,
		Longitude:     lon,
		OffsetMinutes: math.Round(offset*100) / 100,
		Original:      fmt.Sprintf("%04d-%02d-%02d %02d:%02d", year, month, day, hour, minute),
		Corrected:     fmt.Sprintf("%04d-%02d-%02d %02d:%02d", y2, m2, d2, h2, mi2),
		CrossDay:      y2 != year || m2 != month || d2 != day,
		CrossShichen:  origShichen != newShichen,
	}
	return y2, m2, d2, h2, mi2, info
}
