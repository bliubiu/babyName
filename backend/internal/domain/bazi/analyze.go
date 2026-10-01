package bazi

import (
	"fmt"
	"math"
	"sync"
	"time"

	"name/internal/domain/bazi/tyme"
	"name/internal/domain/yijing"
)

const (
	WuxingRatioTaiRuo    = 15.0
	WuxingRatioPianRuo   = 25.0
	WuxingRatioPingheng  = 35.0
	WuxingRatioPianQiang = 45.0

	ScoreTaiRuoMax    = 35
	ScorePianRuoMax   = 55
	ScorePinghengMax  = 75
	ScorePianQiangMax = 90
	ScoreGuoWangMax   = 95
)

var (
	wuxingAdviceMap = map[string]string{
		"金": "金过旺则折，需火炼土生；金太弱则缺，宜补金",
		"木": "木过旺则折，需金克火泄；木太弱则缺，宜补木",
		"水": "水过旺则泛滥，需土制木泄；水太弱则缺，宜补水",
		"火": "火过旺则炎上，需水克金泄；火太弱则缺，宜补火",
		"土": "土过旺则壅塞，需木克水泄；土太弱则缺，宜补土",
	}
)

// GetSolarTerm 根据月日获取节气名（已废弃，精度低）
// 请使用 GetSolarTermByDate(year, month, day int) 替代
// Deprecated: 使用 tyme4go 天文精度版
func GetSolarTerm(month, day int) string {
	return GetSolarTermByDate(2000, month, day)
}

// GetSolarTermByDate 使用 tyme4go 天文计算获取指定日期的节气名
func GetSolarTermByDate(year, month, day int) string {
	solarDay, err := tyme.SolarDay{}.FromYmd(year, month, day)
	if err != nil {
		return ""
	}
	term := solarDay.GetTerm()
	if term.GetSolarDay().GetYear() == year &&
		term.GetSolarDay().GetMonth() == month &&
		term.GetSolarDay().GetDay() == day {
		return term.GetName()
	}
	return ""
}

func GetSolarTermFromTyme(year, month, day, hour, minute int) (string, bool, error) {
	solarTime, err := tyme.SolarTime{}.FromYmdHms(year, month, day, hour, minute, 0)
	if err != nil {
		return "", false, err
	}

	term := solarTime.GetTerm()
	lunarMonth := solarTime.GetSolarDay().GetLunarDay().GetLunarMonth()
	isLeapMonth := lunarMonth.IsLeap()

	return term.GetName(), isLeapMonth, nil
}

type WuxingStrengthResult struct {
	Jin  StrengthInfo `json:"jin"`
	Mu   StrengthInfo `json:"mu"`
	Shui StrengthInfo `json:"shui"`
	Huo  StrengthInfo `json:"huo"`
	Tu   StrengthInfo `json:"tu"`
}

type StrengthInfo struct {
	Count    int    `json:"count"`
	Strength string `json:"strength"`
	Score    int    `json:"score"`
	Advice   string `json:"advice"`
}

func CalculateWuxingStrength(wuxing *WuxingResult) *WuxingStrengthResult {
	result := &WuxingStrengthResult{}

	elements := map[string]*StrengthInfo{
		"金": &result.Jin, "木": &result.Mu, "水": &result.Shui, "火": &result.Huo, "土": &result.Tu,
	}

	counts := map[string]int{"金": wuxing.Jin, "木": wuxing.Mu, "水": wuxing.Shui, "火": wuxing.Huo, "土": wuxing.Tu}

	total := wuxing.Jin + wuxing.Mu + wuxing.Shui + wuxing.Huo + wuxing.Tu
	if total == 0 {
		total = 1
	}

	for elem, count := range counts {
		ratio := float64(count) / float64(total) * 100
		strength, score := calculateStrengthAndScore(ratio)

		elements[elem].Count = count
		elements[elem].Strength = strength
		elements[elem].Score = score
		elements[elem].Advice = wuxingAdviceMap[elem]
	}

	return result
}

func calculateStrengthAndScore(ratio float64) (string, int) {
	var strength string
	var score int

	if ratio < WuxingRatioTaiRuo {
		strength = "太弱"
		score = 20 + int(ratio)
	} else if ratio < WuxingRatioPianRuo {
		strength = "偏弱"
		score = 40 + int((ratio-WuxingRatioTaiRuo)*2)
	} else if ratio < WuxingRatioPingheng {
		strength = "平衡"
		score = 60 + int((ratio - WuxingRatioPianRuo))
	} else if ratio < WuxingRatioPianQiang {
		strength = "偏强"
		score = 75 + int((ratio - WuxingRatioPingheng))
	} else {
		strength = "过旺"
		score = 90 - int((ratio - WuxingRatioPianQiang))
		if score > ScoreGuoWangMax {
			score = ScoreGuoWangMax
		}
	}

	return strength, score
}

func CalculateWuxingScore(wuxing *WuxingResult) int {
	strength := CalculateWuxingStrength(wuxing)
	total := 0
	count := 0

	elements := []*StrengthInfo{&strength.Jin, &strength.Mu, &strength.Shui, &strength.Huo, &strength.Tu}
	for _, e := range elements {
		total += e.Score
		count++
	}

	if count == 0 {
		return 50
	}
	return total / count
}

type Bazi struct {
	Year        string `json:"year"`
	Month       string `json:"month"`
	Day         string `json:"day"`
	Hour        string `json:"hour"`
	YearGanzhi  string `json:"year_ganzhi"`
	MonthGanzhi string `json:"month_ganzhi"`
	DayGanzhi   string `json:"day_ganzhi"`
	HourGanzhi  string `json:"hour_ganzhi"`
	IsLeapMonth bool   `json:"is_leap_month"`
	LunarMonth  int    `json:"lunar_month"`
}

type WuxingResult struct {
	Jin  int `json:"jin"`
	Mu   int `json:"mu"`
	Shui int `json:"shui"`
	Huo  int `json:"huo"`
	Tu   int `json:"tu"`
}

// NayinInfo 单柱纳音
type NayinInfo struct {
	Ganzhi string `json:"ganzhi"`
	Nayin  string `json:"nayin"`
	Wuxing string `json:"nayin_wuxing"`
}

// FourPillarNayin 四柱纳音
type FourPillarNayin struct {
	Year  NayinInfo `json:"year"`
	Month NayinInfo `json:"month"`
	Day   NayinInfo `json:"day"`
	Hour  NayinInfo `json:"hour"`
}

// BuildFourPillarNayin 由四柱干支构建纳音结构
func BuildFourPillarNayin(b *Bazi) *FourPillarNayin {
	if b == nil {
		return nil
	}
	one := func(gz string) NayinInfo {
		n := NayinMap[gz]
		return NayinInfo{Ganzhi: gz, Nayin: n, Wuxing: NayinWuxingMap[n]}
	}
	return &FourPillarNayin{
		Year:  one(b.YearGanzhi),
		Month: one(b.MonthGanzhi),
		Day:   one(b.DayGanzhi),
		Hour:  one(b.HourGanzhi),
	}
}

type BaziAnalysis struct {
	Bazi         Bazi         `json:"bazi"`
	Wuxing       WuxingResult `json:"wuxing"`
	Xiyongshen   []string     `json:"xiyongshen"`
	Yiyongshen   []string     `json:"yiyongshen"`
	TiaohouShen  []string     `json:"tiaohou_shen"`
	Rishou       string       `json:"rishou"`
	RishouWuxing string       `json:"rishou_wuxing"`
	// Nayin 年命纳音（兼容字段，等于 FourNayin.Year.Nayin）
	Nayin string `json:"nayin"`
	// FourNayin 四柱纳音（年/月/日/时）
	FourNayin           *FourPillarNayin      `json:"four_nayin,omitempty"`
	DayMaster           string                `json:"day_master"`
	DayMasterStrength   string                `json:"day_master_strength"`
	SolarTerm           string                `json:"solar_term"`
	Season              string                `json:"season"`
	WuxingStrength      *WuxingStrengthResult `json:"wuxing_strength"`
	BestHexagram        string                `json:"best_hexagram"`
	HexagramWuxing      string                `json:"hexagram_wuxing"`
	BaziPattern         string                `json:"bazi_pattern"`
	YongshenScore       int                   `json:"yongshen_score"`
	DayanHexagram       string                `json:"dayan_hexagram"`
	DayanInterpretation string                `json:"dayan_interpretation"`
	IsLeapMonth         bool                  `json:"is_leap_month"`
	LeapMonthAdvice     string                `json:"leap_month_advice,omitempty"`
}

var baziCalcMutex sync.Mutex

func CalculateBazi(year, month, day, hour, minute int) (*Bazi, error) {
	baziCalcMutex.Lock()
	defer baziCalcMutex.Unlock()

	solarTime, err := tyme.SolarTime{}.FromYmdHms(year, month, day, hour, minute, 0)
	if err != nil {
		return nil, fmt.Errorf("创建SolarTime失败: %w", err)
	}

	lunarHour := solarTime.GetLunarHour()
	eightChar := lunarHour.GetEightChar()

	lunarDay := solarTime.GetSolarDay().GetLunarDay()
	lunarMonth := lunarDay.GetLunarMonth()
	isLeapMonth := lunarMonth.IsLeap()
	lunarMonthNum := lunarMonth.GetMonth()

	yearGanzhi := eightChar.GetYear().GetName()
	monthGanzhi := eightChar.GetMonth().GetName()
	dayGanzhi := eightChar.GetDay().GetName()
	hourGanzhi := eightChar.GetHour().GetName()

	return &Bazi{
		Year:        yearGanzhi,
		Month:       monthGanzhi,
		Day:         dayGanzhi,
		Hour:        hourGanzhi,
		YearGanzhi:  yearGanzhi,
		MonthGanzhi: monthGanzhi,
		DayGanzhi:   dayGanzhi,
		HourGanzhi:  hourGanzhi,
		IsLeapMonth: isLeapMonth,
		LunarMonth:  lunarMonthNum,
	}, nil
}

func AnalyzeBazi(year, month, day, hour, minute int) (*BaziAnalysis, error) {
	bazi, err := CalculateBazi(year, month, day, hour, minute)
	if err != nil {
		return nil, err
	}

	// 加权五行：藏干 + 月令
	weighted := CalculateWeightedWuxing(bazi)
	wuxing := weighted.ToCounts()
	dayGanzhi := bazi.DayGanzhi
	dayTiangan := string([]rune(dayGanzhi)[0])
	rishouWuxing := WuxingMap[dayTiangan]

	monthBranch := ""
	if mb := []rune(bazi.MonthGanzhi); len(mb) >= 2 {
		monthBranch = string(mb[1])
	}

	dayMasterStrength := calculateDayMasterStrengthWeighted(weighted, rishouWuxing, monthBranch)
	// 调候/季节按月支（节气月）判定
	tiaohouShen := calculateTiaohouShenByBranch(bazi.MonthGanzhi)
	season := seasonFromMonthBranch(bazi.MonthGanzhi)

	xiyongshen := calculateXiyongshenWeighted(weighted, rishouWuxing, dayMasterStrength, tiaohouShen)
	iyongshen := calculateYiyongshenFromWeighted(weighted, rishouWuxing)

	// 年命纳音 + 四柱纳音（tyme 已校正立春后的干支）
	fourNayin := BuildFourPillarNayin(bazi)
	nayin := ""
	if fourNayin != nil {
		nayin = fourNayin.Year.Nayin
	}

	solarTerm, isLeapMonth, err := GetSolarTermFromTyme(year, month, day, hour, minute)
	if err != nil {
		solarTerm = ""
	}

	wuxingStrength := CalculateWuxingStrength(wuxing)

	// 姓名卦需具体名字笔画，在起名服务层计算；此处不硬编码占位卦
	bestHexagram := ""
	hexagramWuxing := ""

	baziPattern := calculateBaziPattern(dayMasterStrength, rishouWuxing, season)
	yongshenScore := calculateYongshenScore(xiyongshen, iyongshen, wuxing)

	dayanResult := yijing.CastHexagramByDayan()
	dayanHexagram := ""
	dayanInterpretation := ""
	if dayanResult != nil && dayanResult.Hexagram != nil {
		dayanHexagram = dayanResult.Hexagram.Name
		dayanInterpretation = dayanResult.Interpretation
	}

	result := &BaziAnalysis{
		Bazi:                *bazi,
		Wuxing:              *wuxing,
		Xiyongshen:          xiyongshen,
		Yiyongshen:          iyongshen,
		TiaohouShen:         tiaohouShen,
		Rishou:              dayTiangan,
		RishouWuxing:        rishouWuxing,
		Nayin:               nayin,
		FourNayin:           fourNayin,
		DayMaster:           rishouWuxing,
		DayMasterStrength:   dayMasterStrength,
		SolarTerm:           solarTerm,
		Season:              season,
		WuxingStrength:      wuxingStrength,
		BestHexagram:        bestHexagram,
		HexagramWuxing:      hexagramWuxing,
		BaziPattern:         baziPattern,
		YongshenScore:       yongshenScore,
		DayanHexagram:       dayanHexagram,
		DayanInterpretation: dayanInterpretation,
		IsLeapMonth:         isLeapMonth || bazi.IsLeapMonth,
	}

	if result.IsLeapMonth {
		result.LeapMonthAdvice = getLeapMonthAdvice(bazi.LunarMonth, dayTiangan)
	}

	return result, nil
}

func getLeapMonthAdvice(lunarMonth int, dayTiangan string) string {
	advice := fmt.Sprintf("闰月出生的宝宝，月柱论断需特别考虑。闰月出生的孩子性格多变化，需因势利导。")
	return advice
}

func calculateWuxing(ganzhi string) *WuxingResult {
	result := &WuxingResult{}

	for _, r := range ganzhi {
		char := string(r)

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

// seasonFromMonthBranch 按月支（节气月）判定季节
// 寅卯辰→春，巳午未→夏，申酉戌→秋，亥子丑→冬
func seasonFromMonthBranch(monthGanzhi string) string {
	runes := []rune(monthGanzhi)
	if len(runes) < 2 {
		return ""
	}
	switch string(runes[1]) {
	case "寅", "卯", "辰":
		return "春"
	case "巳", "午", "未":
		return "夏"
	case "申", "酉", "戌":
		return "秋"
	case "亥", "子", "丑":
		return "冬"
	}
	return ""
}

// seasonWuxing 季节 → 当令五行（春木夏火秋金冬水；土旺四季，辰戌丑未月
// 未在此区分，土日主一律按不得季令处理）
var seasonWuxing = map[string]string{
	"春": "木",
	"夏": "火",
	"秋": "金",
	"冬": "水",
}

func calculateBaziPattern(dayMasterStrength, rishouWuxing, season string) string {
	// 日主五行与季节当令五行比较；原实现直接拿五行跟「春/夏/秋/冬」
	// 字符串比较，恒为 false，格局分支沦为死代码。
	seasonElement := seasonWuxing[season]
	if dayMasterStrength == "身旺" || dayMasterStrength == "身中" {
		if rishouWuxing == seasonElement {
			return "正格-印比相生格"
		}
		return "正格-财官相生格"
	} else if dayMasterStrength == "身弱" || dayMasterStrength == "身衰" {
		if rishouWuxing == seasonElement {
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

	shengKeScore := calculateShengKeScore(xiyongshen, wuxing)
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

func calculateShengKeScore(xiyongshen []string, wuxing *WuxingResult) int {
	shengKeScore := 0
	// 五行相生：金生水、水生木、木生火、火生土、土生金（键生值）。
	// 原表「土」误作生「火」，土生金才对，导致喜用神含土时加分算到火的计数上。
	shengMap := map[string]string{
		"金": "水",
		"木": "火",
		"土": "金",
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

	return shengKeScore
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

// calculateTiaohouShenByBranch 按月支给简化调候用神（冬火夏水、春火金、秋火木）
func calculateTiaohouShenByBranch(monthGanzhi string) []string {
	runes := []rune(monthGanzhi)
	if len(runes) < 2 {
		return nil
	}
	switch string(runes[1]) {
	case "子", "丑", "亥": // 冬令寒气重
		return []string{"火", "土"}
	case "寅", "卯": // 初春/木旺
		return []string{"火", "金"}
	case "辰":
		return []string{"金", "水"}
	case "巳", "午": // 夏令火旺
		return []string{"水", "金"}
	case "未":
		return []string{"水", "金"}
	case "申", "酉", "戌": // 秋令金旺
		return []string{"火", "木"}
	}
	return nil
}

func calculateXiyongshen(wuxing *WuxingResult, rishouWuxing string, dayMasterStrength string) []string {
	total := wuxing.Jin + wuxing.Mu + wuxing.Shui + wuxing.Huo + wuxing.Tu
	if total == 0 {
		// 无五行数据时，返回日主本身作为喜用神
		return []string{rishouWuxing}
	}

	// 构建反向映射表（运行时计算，与 WuxingShengkeMap 保持同步）
	// shengWoMap: 生我者（印绶） — WuxingShengkeMap[X][0] = X所生, 则谁生Y即查找谁的[0]==Y
	shengWoMap := map[string]string{}
	// keWoMap: 克我者（官杀） — WuxingShengkeMap[X][1] = X所克, 则谁克Y即查找谁的[1]==Y
	keWoMap := map[string]string{}
	for k, v := range WuxingShengkeMap {
		shengWoMap[v[0]] = k // k 生 v[0]
		keWoMap[v[1]] = k    // k 克 v[1]
	}

	switch dayMasterStrength {
	case "身旺":
		// 日主强 → 克泄耗：官杀(克我) + 食伤(我生) + 妻财(我克)
		result := make([]string, 0, 3)
		if ke, ok := keWoMap[rishouWuxing]; ok {
			result = append(result, ke)
		}
		if shengList, ok := WuxingShengkeMap[rishouWuxing]; ok {
			result = append(result, shengList[0]) // 我生（食伤）
			result = append(result, shengList[1]) // 我克（妻财）
		}
		return result

	case "身弱", "身衰":
		// 日主弱 → 生扶：比劫(本身) + 印绶(生我)
		result := make([]string, 0, 2)
		result = append(result, rishouWuxing)
		if sheng, ok := shengWoMap[rishouWuxing]; ok {
			result = append(result, sheng)
		}
		return result

	default: // "身中" 或未知值
		// 中和 → 以日主本身为喜用神
		return []string{rishouWuxing}
	}
}

// GetShichen 按小时取时辰地支，与 tyme `(hour+1)/2` 一致（23 时归子）
func GetShichen(hour int) string {
	if hour < 0 || hour > 23 {
		return ""
	}
	return Dizhi[((hour+1)/2)%12]
}

func ParseTimeToHourMinute(timeStr string) (hour, minute int, err error) {
	_, err = time.Parse("15:04", timeStr)
	if err != nil {
		return 0, 0, err
	}
	_, _ = fmt.Sscanf(timeStr, "%d:%d", &hour, &minute)
	return hour, minute, nil
}
