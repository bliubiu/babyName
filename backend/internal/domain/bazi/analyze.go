package bazi

import (
	"fmt"
	"math"
	"time"

	"github.com/6tail/lunar-go/calendar"
	"github.com/namemaster/backend/internal/domain/yijing"
)

// SolarTermData 节气数据（简化版）
var SolarTermData = map[string]struct {
	Month   int
	Day     int
	Hour    int
	Minute  int
}{
	"小寒": {12, 5, 12, 0}, "大寒": {1, 20, 12, 0},
	"立春": {2, 4, 12, 0}, "雨水": {2, 19, 12, 0},
	"惊蛰": {3, 5, 12, 0}, "春分": {3, 20, 12, 0},
	"清明": {4, 5, 12, 0}, "谷雨": {4, 20, 12, 0},
	"立夏": {5, 5, 12, 0}, "小满": {5, 21, 12, 0},
	"芒种": {6, 5, 12, 0}, "夏至": {6, 21, 12, 0},
	"小暑": {7, 7, 12, 0}, "大暑": {7, 22, 12, 0},
	"立秋": {8, 7, 12, 0}, "处暑": {8, 23, 12, 0},
	"白露": {9, 7, 12, 0}, "秋分": {9, 23, 12, 0},
	"寒露": {10, 8, 12, 0}, "霜降": {10, 23, 12, 0},
	"立冬": {11, 7, 12, 0}, "小雪": {11, 22, 12, 0},
	"大雪": {12, 7, 12, 0}, "冬至": {12, 21, 12, 0},
}

// GetSolarTerm 获取节气
func GetSolarTerm(month, day int) string {
	terms := []string{
		"小寒", "大寒", "立春", "雨水", "惊蛰", "春分",
		"清明", "谷雨", "立夏", "小满", "芒种", "夏至",
		"小暑", "大暑", "立秋", "处暑", "白露", "秋分",
		"寒露", "霜降", "立冬", "小雪", "大雪", "冬至",
	}
	termDays := []int{5, 20, 4, 19, 5, 20, 5, 20, 5, 21, 5, 21, 7, 22, 7, 23, 7, 23, 8, 23, 8, 22, 7, 21}

	for i, td := range termDays {
		if month == (i/2)+1 && day == td {
			return terms[i]
		}
	}
	return ""
}

// SolarTermOffset 节气时间偏移
func SolarTermOffset(year, month, day, hour, minute int) bool {
	term := GetSolarTerm(month, day)
	if term == "" {
		return false
	}
	if data, ok := SolarTermData[term]; ok {
		if month == data.Month && day == data.Day {
			hour = hour*60 + minute
			targetHour := data.Hour*60 + data.Minute
			return hour >= targetHour
		}
	}
	return false
}

// WuxingStrengthResult 五行强弱分析结果
type WuxingStrengthResult struct {
	Jin  StrengthInfo `json:"jin"`
	Mu   StrengthInfo `json:"mu"`
	Shui StrengthInfo `json:"shui"`
	Huo  StrengthInfo `json:"huo"`
	Tu   StrengthInfo `json:"tu"`
}

// StrengthInfo 强弱信息
type StrengthInfo struct {
	Count     int     `json:"count"`
	Strength  string  `json:"strength"` // 弱/平衡/强
	Score     int     `json:"score"`    // 1-100
	Advice    string  `json:"advice"`   // 建议
}

// WuxingStrengthAnalysis 五行强弱分析
func WuxingStrengthAnalysis(wuxing *WuxingResult) *WuxingStrengthResult {
	result := &WuxingStrengthResult{}

	elements := map[string]*StrengthInfo{
		"金": &result.Jin, "木": &result.Mu, "水": &result.Shui, "火": &result.Huo, "土": &result.Tu,
	}

	counts := map[string]int{"金": wuxing.Jin, "木": wuxing.Mu, "水": wuxing.Shui, "火": wuxing.Huo, "土": wuxing.Tu}

	adviceMap := map[string]string{
		"金": "金过旺则折，需火炼土生；金太弱则缺，宜补金",
		"木": "木过旺则折，需金克火泄；木太弱则缺，宜补木",
		"水": "水过旺则泛滥，需土制木泄；水太弱则缺，宜补水",
		"火": "火过旺则炎上，需水克金泄；火太弱则缺，宜补火",
		"土": "土过旺则壅塞，需木克水泄；土太弱则缺，宜补土",
	}

	total := wuxing.Jin + wuxing.Mu + wuxing.Shui + wuxing.Huo + wuxing.Tu
	if total == 0 {
		total = 1
	}

	for elem, count := range counts {
		ratio := float64(count) / float64(total) * 100
		strength := "平衡"
		score := 50

		if ratio < 15 {
			strength = "太弱"
			score = 20 + int(ratio)
		} else if ratio < 25 {
			strength = "偏弱"
			score = 40 + int((ratio-15)*2)
		} else if ratio < 35 {
			strength = "平衡"
			score = 60 + int((ratio-25))
		} else if ratio < 45 {
			strength = "偏强"
			score = 75 + int((ratio-35))
		} else {
			strength = "过旺"
			score = 90 - int((ratio-45))
			if score > 95 {
				score = 95
			}
		}

		elements[elem].Count = count
		elements[elem].Strength = strength
		elements[elem].Score = score
		elements[elem].Advice = adviceMap[elem]
	}

	return result
}

// Bazi 八字结构
type Bazi struct {
	Year      string `json:"year"`      // 年柱
	Month     string `json:"month"`     // 月柱
	Day       string `json:"day"`       // 日柱
	Hour      string `json:"hour"`      // 时柱
	YearGanzhi string `json:"year_ganzhi"`
	MonthGanzhi string `json:"month_ganzhi"`
	DayGanzhi   string `json:"day_ganzhi"`
	HourGanzhi  string `json:"hour_ganzhi"`
}

// WuxingResult 五行分析结果
type WuxingResult struct {
	Jin  int `json:"jin"`  // 金
	Mu   int `json:"mu"`   // 木
	Shui int `json:"shui"` // 水
	Huo  int `json:"huo"`  // 火
	Tu   int `json:"tu"`   // 土
}

// BaziAnalysis 八字分析结果
type BaziAnalysis struct {
	Bazi              Bazi                  `json:"bazi"`
	Wuxing            WuxingResult          `json:"wuxing"`
	Xiyongshen       []string              `json:"xiyongshen"`
	Yiyongshen       []string              `json:"iyongshen"`
	TiaohouShen      []string              `json:"tiaohou_shen"`
	Rishou           string                `json:"rishou"`
	RishouWuxing     string                `json:"rishou_wuxing"`
	Nayin            string                `json:"nayin"`
	DayMaster        string                `json:"day_master"`
	DayMasterStrength string               `json:"day_master_strength"`
	SolarTerm        string                `json:"solar_term"`
	Season           string                `json:"season"`
	WuxingStrength   *WuxingStrengthResult `json:"wuxing_strength"`
	BestHexagram     string                `json:"best_hexagram"`
	HexagramWuxing  string                `json:"hexagram_wuxing"`
	BaziPattern      string                `json:"bazi_pattern"`
	YongshenScore    int                   `json:"yongshen_score"`
	DayanHexagram   string                `json:"dayan_hexagram"`
	DayanInterpretation string             `json:"dayan_interpretation"`
}

// CalculateBazi 计算八字（使用lunar-go高精度库）
func CalculateBazi(year, month, day, hour, minute int) *Bazi {
	solar := calendar.NewSolar(year, month, day, hour, minute, 0)
	lunar := calendar.NewLunarFromSolar(solar)

	yearGanzhi := lunar.GetYearInGanZhi()
	monthGanzhi := lunar.GetMonthInGanZhi()
	dayGanzhi := lunar.GetDayInGanZhi()
	hourGanzhi := lunar.GetTimeInGanZhi()

	return &Bazi{
		Year:        yearGanzhi,
		Month:       monthGanzhi,
		Day:         dayGanzhi,
		Hour:        hourGanzhi,
		YearGanzhi:  yearGanzhi,
		MonthGanzhi: monthGanzhi,
		DayGanzhi:   dayGanzhi,
		HourGanzhi:  hourGanzhi,
	}
}

// AnalyzeBazi 分析八字
func AnalyzeBazi(year, month, day, hour, minute int) *BaziAnalysis {
	solar := calendar.NewSolar(year, month, day, hour, minute, 0)
	lunar := calendar.NewLunarFromSolar(solar)

	bazi := CalculateBazi(year, month, day, hour, minute)

	ganzhi := bazi.YearGanzhi + bazi.MonthGanzhi + bazi.DayGanzhi + bazi.HourGanzhi
	wuxing := calculateWuxing(ganzhi)
	dayGanzhi := bazi.DayGanzhi
	dayTiangan := string([]rune(dayGanzhi)[0])
	rishouWuxing := WuxingMap[dayTiangan]

	xiyongshen := calculateXiyongshen(wuxing, rishouWuxing)
	iyongshen := calculateYiyongshen(wuxing, rishouWuxing)
	tiaohouShen := calculateTiaohouShen(month)

	nayin := lunar.GetYearNaYin()
	if nayin == "" {
		nayin = "未知"
	}

	solarTerm := lunar.GetJieQi()
	if solarTerm == "" {
		solarTerm = GetSolarTerm(month, day)
	}

	season := getSeason(month, day)
	dayMasterStrength := calculateDayMasterStrength(wuxing, rishouWuxing)
	wuxingStrength := WuxingStrengthAnalysis(wuxing)

	hexagram := yijing.GetHexagramByStrokes(10)
	hexagramWuxing := ""
	bestHexagram := ""
	if hexagram != nil {
		bestHexagram = hexagram.Name
		hexagramWuxing = yijing.AnalyzeHexagramWuxing(hexagram)
	}

	baziPattern := calculateBaziPattern(dayMasterStrength, rishouWuxing, season)
	yongshenScore := calculateYongshenScore(xiyongshen, iyongshen, wuxing)

	dayanResult := yijing.CastHexagramByDayan()
	dayanHexagram := ""
	dayanInterpretation := ""
	if dayanResult != nil && dayanResult.Hexagram != nil {
		dayanHexagram = dayanResult.Hexagram.Name
		dayanInterpretation = dayanResult.Interpretation
	}

	return &BaziAnalysis{
		Bazi:              *bazi,
		Wuxing:            *wuxing,
		Xiyongshen:       xiyongshen,
		Yiyongshen:       iyongshen,
		TiaohouShen:      tiaohouShen,
		Rishou:           dayTiangan,
		RishouWuxing:     rishouWuxing,
		Nayin:            nayin,
		DayMaster:        rishouWuxing,
		DayMasterStrength: dayMasterStrength,
		SolarTerm:        solarTerm,
		Season:            season,
		WuxingStrength:    wuxingStrength,
		BestHexagram:     bestHexagram,
		HexagramWuxing:   hexagramWuxing,
		BaziPattern:      baziPattern,
		YongshenScore:    yongshenScore,
		DayanHexagram:   dayanHexagram,
		DayanInterpretation: dayanInterpretation,
	}
}

// getYearGanzhi 计算年柱
func getYearGanzhi(year int) string {
	tianganIndex := (year - 4) % 10
	if tianganIndex < 0 {
		tianganIndex += 10
	}
	dizhiIndex := (year - 4) % 12
	if dizhiIndex < 0 {
		dizhiIndex += 12
	}
	return Tiangan[tianganIndex] + Dizhi[dizhiIndex]
}

// getMonthGanzhi 计算月柱
func getMonthGanzhi(year, month int) string {
	yearTianganIndex := (year - 4) % 10
	if yearTianganIndex < 0 {
		yearTianganIndex += 10
	}

	monthTianganBase := []int{0, 2, 4, 6, 8}
	monthTianganIndex := (monthTianganBase[yearTianganIndex] + month - 1) % 10

	dizhiIndex := (month + 1) % 12
	if dizhiIndex == 0 {
		dizhiIndex = 12
	}

	return Tiangan[monthTianganIndex] + Dizhi[dizhiIndex-1]
}

// getDayGanzhi 计算日柱（使用蔡勒公式）
func getDayGanzhi(year, month, day int) string {
	y := year
	m := month
	if m < 3 {
		y--
		m += 12
	}

	c := y / 100
	y = y % 100

	d := day

	w := c/4 - 2*c + y + y/4 + (13*(m+1))/5 + d - 1
	w = ((w % 7) + 7) % 7

	tianganIndex := (d + 2) % 10
	dizhiIndex := w

	return Tiangan[tianganIndex] + Dizhi[dizhiIndex]
}

// getHourGanzhi 计算时柱
func getHourGanzhi(hour int, dayGanzhi string) string {
	dizhiIndex := hour / 2
	if dizhiIndex > 11 {
		dizhiIndex = 11
	}

	dayTiangan := string(dayGanzhi[0])
	dayTianganIndex := 0
	for i, t := range Tiangan {
		if t == dayTiangan {
			dayTianganIndex = i
			break
		}
	}

	tianganIndex := (dayTianganIndex*2 + dizhiIndex) % 10

	return Tiangan[tianganIndex] + Dizhi[dizhiIndex]
}

// calculateWuxing 计算五行分布
func calculateWuxing(ganzhi string) *WuxingResult {
	result := &WuxingResult{}

	for i := 0; i < len(ganzhi); i++ {
		char := string(ganzhi[i])

		if wuxing, ok := WuxingMap[char]; ok {
			switch wuxing {
			case "金":
				result.Jin++
			case "木":
				result.Mu++
			case "水":
				result.Shui++
			case "火":
				result.Huo++
			case "土":
				result.Tu++
			}
		} else if wuxing, ok := DizhiWuxingMap[char]; ok {
			switch wuxing {
			case "金":
				result.Jin++
			case "木":
				result.Mu++
			case "水":
				result.Shui++
			case "火":
				result.Huo++
			case "土":
				result.Tu++
			}
		}
	}

	return result
}

func getSeason(month int, day int) string {
	if month >= 3 && month <= 5 {
		if month == 3 {
			if day < 21 {
				return "冬"
			}
			return "春"
		}
		if month == 4 {
			return "春"
		}
		if month == 5 {
			if day < 21 {
				return "春"
			}
			return "夏"
		}
		return "春"
	} else if month >= 6 && month <= 8 {
		if month == 6 {
			if day < 21 {
				return "夏"
			}
			return "夏"
		}
		return "夏"
	} else if month >= 9 && month <= 11 {
		if month == 9 {
			if day < 23 {
				return "夏"
			}
			return "秋"
		}
		if month == 10 {
			return "秋"
		}
		if month == 11 {
			if day < 22 {
				return "秋"
			}
			return "冬"
		}
		return "秋"
	}
	return "冬"
}

func calculateBaziPattern(dayMasterStrength, rishouWuxing, season string) string {
	if dayMasterStrength == "身旺" || dayMasterStrength == "身中" {
		if rishouWuxing == season {
			return "正格-印比相生格"
		}
		return "正格-财官相生格"
	} else if dayMasterStrength == "身弱" || dayMasterStrength == "身衰" {
		if rishouWuxing == season {
			return "从弱格-从印格"
		}
		return "从弱格-从财格"
	}
	return "正格-普通格局"
}

func calculateYongshenScore(xiyongshen []string, yiyongshen []string, wuxing *WuxingResult) int {
	score := 60

	if len(xiyongshen) == 0 {
		return 50
	}

	for _, xy := range xiyongshen {
		switch xy {
		case "金":
			score += wuxing.Jin * 3
		case "木":
			score += wuxing.Mu * 3
		case "水":
			score += wuxing.Shui * 3
		case "火":
			score += wuxing.Huo * 3
		case "土":
			score += wuxing.Tu * 3
		}
	}

	shengKeScore := 0
	shengMap := map[string]string{
		"金": "水",
		"木": "火",
		"土": "火",
		"水": "木",
		"火": "土",
	}

	for _, xy := range xiyongshen {
		ke := shengMap[xy]
		switch ke {
		case "金":
			shengKeScore += wuxing.Jin
		case "木":
			shengKeScore += wuxing.Mu
		case "水":
			shengKeScore += wuxing.Shui
		case "火":
			shengKeScore += wuxing.Huo
		case "土":
			shengKeScore += wuxing.Tu
		}
	}

	score += shengKeScore * 2

	if len(yiyongshen) > 0 {
		for _, yy := range yiyongshen {
			switch yy {
			case "金":
				score -= wuxing.Jin
			case "木":
				score -= wuxing.Mu
			case "水":
				score -= wuxing.Shui
			case "火":
				score -= wuxing.Huo
			case "土":
				score -= wuxing.Tu
			}
		}
	}

	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}

	return score
}

func calculateDayMasterStrength(wuxing *WuxingResult, rishouWuxing string) string {
	total := wuxing.Jin + wuxing.Mu + wuxing.Shui + wuxing.Huo + wuxing.Tu
	if total == 0 {
		return "偏弱"
	}

	dayMasterValue := 0
	switch rishouWuxing {
	case "金":
		dayMasterValue = wuxing.Jin
	case "木":
		dayMasterValue = wuxing.Mu
	case "水":
		dayMasterValue = wuxing.Shui
	case "火":
		dayMasterValue = wuxing.Huo
	case "土":
		dayMasterValue = wuxing.Tu
	}

	dayMasterRatio := float64(dayMasterValue) / float64(total) * 100

	if dayMasterRatio >= 35 {
		return "身旺"
	} else if dayMasterRatio >= 25 {
		return "身中"
	} else if dayMasterRatio >= 15 {
		return "身弱"
	} else {
		return "身衰"
	}
}

func calculateYiyongshen(wuxing *WuxingResult, rishouWuxing string) []string {
	var yiyongshen []string

	counts := map[string]int{
		"金": wuxing.Jin,
		"木": wuxing.Mu,
		"水": wuxing.Shui,
		"火": wuxing.Huo,
		"土": wuxing.Tu,
	}

	minCount := math.MaxInt
	minWuxing := ""
	for w, count := range counts {
		if count < minCount {
			minCount = count
			minWuxing = w
		}
	}

	ke := WuxingShengkeMap[minWuxing]
	if len(ke) > 1 {
		yiyongshen = append(yiyongshen, ke[1])
	}

	return yiyongshen
}

func calculateTiaohouShen(month int) []string {
	var tiaohou []string

	switch month {
	case 1, 2:
		tiaohou = append(tiaohou, "火", "土")
	case 3, 4, 5:
		tiaohou = append(tiaohou, "水", "金")
	case 6:
		tiaohou = append(tiaohou, "水", "金")
	case 7, 8:
		tiaohou = append(tiaohou, "水", "火")
	case 9, 10, 11:
		tiaohou = append(tiaohou, "火", "木")
	case 12:
		tiaohou = append(tiaohou, "火", "木")
	}

	return tiaohou
}

// calculateXiyongshen 计算喜用神
func calculateXiyongshen(wuxing *WuxingResult, rishouWuxing string) []string {
	minCount := math.MaxInt
	minWuxing := ""

	counts := map[string]int{
		"金": wuxing.Jin,
		"木": wuxing.Mu,
		"水": wuxing.Shui,
		"火": wuxing.Huo,
		"土": wuxing.Tu,
	}

	for w, count := range counts {
		if count < minCount {
			minCount = count
			minWuxing = w
		}
	}

	result := []string{minWuxing}

	sheng := WuxingShengkeMap[minWuxing]
	if len(sheng) > 0 {
		result = append(result, sheng[0])
	}

	return result
}

// GetZodiac 获取生肖
func GetZodiac(year int) string {
	zodiacs := []string{"鼠", "牛", "虎", "兔", "龙", "蛇", "马", "羊", "猴", "鸡", "狗", "猪"}
	index := (year - 1900) % 12
	if index < 0 {
		index += 12
	}
	return zodiacs[index]
}

// GetShichen 根据小时获取时辰
func GetShichen(hour int) string {
	hourMap := []string{"子", "丑", "寅", "卯", "辰", "巳", "午", "未", "申", "酉", "戌", "亥"}
	index := hour / 2
	if index > 11 {
		index = 11
	}
	return hourMap[index]
}

// ParseTimeToHourMinute 将时间字符串解析为小时和分钟
func ParseTimeToHourMinute(timeStr string) (hour, minute int, err error) {
	_, err = time.Parse("15:04", timeStr)
	if err != nil {
		return 0, 0, err
	}
	fmt.Sscanf(timeStr, "%d:%d", &hour, &minute)
	return hour, minute, nil
}
