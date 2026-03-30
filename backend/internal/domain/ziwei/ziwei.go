package ziwei

import (
	"fmt"
	"sync"
)

var (
	ziweiMutex sync.Mutex
)

const (
	Lu  = "禄"
	Quan = "权"
	Ke  = "科"
	Ji  = "忌"
)

var TianganNames = []string{"甲", "乙", "丙", "丁", "戊", "己", "庚", "辛", "壬", "癸"}

var DizhiNames = []string{"子", "丑", "寅", "卯", "辰", "巳", "午", "未", "申", "酉", "戌", "亥"}

var Gongs = []string{"命宫", "兄弟宫", "夫妻宫", "子女宫", "财帛宫", "疾厄宫", "迁移宫", "交友宫", "官禄宫", "田宅宫", "福德宫", "父母宫"}

var ZhuXing14 = []string{"紫微", "天机", "太阳", "武曲", "天同", "廉贞", "天府", "太阴", "贪狼", "巨门", "天梁", "七杀", "破军", "紫微"}

var FuXingList = []string{
	"左辅", "右弼", "文昌", "文曲", "天魁", "天钺",
	"火星", "铃星", "擎羊", "陀罗",
	"天空", "地空", "截空", "旬空",
	"天伤", "天使",
}

var SuiXingList = []string{
	"天姚", "天哭", "天虚", "龙池", "凤阁",
	"红鸾", "天喜", "孤辰", "寡宿",
	"亡神", "劫煞", "大耗",
}

type ZiweiChart struct {
	Year       int      `json:"year"`
	Month      int      `json:"month"`
	Day        int      `json:"day"`
	Hour       int      `json:"hour"`
	Gender     string   `json:"gender"`

	NianZhu    string   `json:"nian_zhu"`
	YueZhu     string   `json:"yue_zhu"`
	RiZhu      string   `json:"ri_zhu"`
	ShiZhu     string   `json:"shi_zhu"`

	MingGong   string   `json:"ming_gong"`
	ShenGong   string   `json:"shen_gong"`

	ZhuXing    map[string]string `json:"zhu_xing"`
	FuXing     map[string][]string `json:"fu_xing"`
	SiHua      map[string]string `json:"si_hua"`

	Analysis   string   `json:"analysis"`
}

type ZiweiAnalysis struct {
	Tianzhu     string `json:"tianzhu"`
	Diwei       string `json:"diwei"`
	Lu         string `json:"lu"`
	Quan       string `json:"quan"`
	Ke         string `json:"ke"`
	Ji         string `json:"ji"`
	Zhushen    string `json:"zhushen"`
	Fuji       string `json:"fuji"`
	Xingyao    string `json:"xingyao"`
	Gongwei    string `json:"gongwei"`
	Analysis   string `json:"analysis"`
}

func AnalyzeZiwei(year, month, day, hour int) *ZiweiAnalysis {
	ziweiMutex.Lock()
	defer ziweiMutex.Unlock()

	chart := CalculateZiweiChart(year, month, day, hour, "")

	return &ZiweiAnalysis{
		Tianzhu:   chart.ZhuXing["命宫"],
		Diwei:     chart.ZhuXing["田宅宫"],
		Lu:        chart.SiHua["禄"],
		Quan:      chart.SiHua["权"],
		Ke:        chart.SiHua["科"],
		Ji:        chart.SiHua["忌"],
		Zhushen:   chart.ZhuXing["命宫"],
		Fuji:      getFuXingString(chart.FuXing["命宫"]),
		Xingyao:   "文昌、文曲、左辅、右弼",
		Gongwei:   chart.MingGong,
		Analysis:  chart.Analysis,
	}
}

func CalculateZiweiChart(year, month, day, hour int, gender string) *ZiweiChart {
	chart := &ZiweiChart{
		Year:     year,
		Month:    month,
		Day:      day,
		Hour:     hour,
		Gender:   gender,
		ZhuXing:  make(map[string]string),
		FuXing:   make(map[string][]string),
		SiHua:    make(map[string]string),
	}

	chart.NianZhu = calculateNianZhu(year)
	chart.YueZhu = calculateYueZhu(month)
	chart.RiZhu = calculateRiZhu(year, month, day)
	chart.ShiZhu = calculateShiZhu(hour)

	chart.MingGong = calculateMingGong(chart.YueZhu, chart.ShiZhu)
	chart.ShenGong = calculateShenGong(chart.YueZhu, chart.ShiZhu)

	distributeZhuXing(chart)
	distributeFuXing(chart, year)
	calculateSiHua(chart)

	chart.Analysis = generateAnalysis(chart)

	return chart
}

func calculateNianZhu(year int) string {
	yearIndex := (year - 4) % 10
	if yearIndex < 0 {
		yearIndex += 10
	}
	gan := TianganNames[yearIndex]

	monthIndex := (year - 4) % 12
	if monthIndex < 0 {
		monthIndex += 12
	}
	zhi := DizhiNames[monthIndex]

	return gan + zhi
}

func calculateYueZhu(month int) string {
	if month < 1 || month > 12 {
		month = 1
	}
	return DizhiNames[(month-1)%12]
}

func calculateRiZhu(year, month, day int) string {
	baseGan := (year - 4) % 10
	if baseGan < 0 {
		baseGan += 10
	}

	dayGanIndex := (baseGan + day - 1) % 10
	if dayGanIndex < 0 {
		dayGanIndex += 10
	}
	gan := TianganNames[dayGanIndex]

	baseZhi := (year - 4) % 12
	if baseZhi < 0 {
		baseZhi += 12
	}

	lunarMonthDays := getLunarMonthDays(month)
	dayZhiIndex := (baseZhi + day - 1) % 12
	if dayZhiIndex < 0 {
		dayZhiIndex += 12
	}
	if day > lunarMonthDays {
		dayZhiIndex = (dayZhiIndex + 1) % 12
	}
	zhi := DizhiNames[dayZhiIndex]

	return gan + zhi
}

func calculateShiZhu(hour int) string {
	if hour < 0 || hour > 23 {
		hour = 12
	}

	hourIndex := (hour + 1) / 2 % 12
	return DizhiNames[hourIndex]
}

func calculateMingGong(yueZhu, shiZhu string) string {
	yueIndex := getDizhiIndex(yueZhu)
	shiIndex := getDizhiIndex(shiZhu)

	gongIndex := (yueIndex + shiIndex) % 12
	return Gongs[gongIndex]
}

func calculateShenGong(yueZhu, shiZhu string) string {
	yueIndex := getDizhiIndex(yueZhu)
	shiIndex := getDizhiIndex(shiZhu)

	gongIndex := (yueIndex + shiIndex + 6) % 12
	return Gongs[gongIndex]
}

func distributeZhuXing(chart *ZiweiChart) {
	nianIndex := (chart.Year - 4) % 10
	if nianIndex < 0 {
		nianIndex += 10
	}

	startOffset := nianIndex
	if nianIndex >= 5 {
		startOffset = nianIndex - 5
	} else {
		startOffset = nianIndex + 5
	}

	startGongIndex := 0
	for i, g := range Gongs {
		if g == chart.MingGong {
			startGongIndex = i
			break
		}
	}

	for i := 0; i < 12; i++ {
		gongName := Gongs[(startGongIndex+i)%12]
		xingIndex := (startOffset + i) % 14
		if xingIndex >= 14 {
			xingIndex -= 14
		}
		chart.ZhuXing[gongName] = ZhuXing14[xingIndex]
	}

	if _, exists := chart.ZhuXing[chart.ShenGong]; !exists {
		chart.ZhuXing[chart.ShenGong] = "天府"
	}
}

func getZhuXingOrder(year, month int) []string {
	yearGan := (year - 4) % 10
	if yearGan < 0 {
		yearGan += 10
	}

	switch yearGan {
	case 0, 5:
		return []string{"紫微", "天机", "太阳", "武曲", "天同", "廉贞", "天府", "太阴", "贪狼", "巨门", "天梁", "七杀", "破军"}
	case 1, 6:
		return []string{"紫微", "天府", "天机", "太阳", "武曲", "天同", "廉贞", "太阴", "贪狼", "巨门", "天梁", "七杀", "破军"}
	case 2, 7:
		return []string{"紫微", "太阴", "天机", "天府", "太阳", "天同", "武曲", "廉贞", "巨门", "贪狼", "七杀", "天梁", "破军"}
	case 3, 8:
		return []string{"紫微", "天府", "太阴", "天机", "天同", "太阳", "廉贞", "武曲", "巨门", "贪狼", "天梁", "七杀", "破军"}
	case 4, 9:
		return []string{"紫微", "天同", "天府", "天机", "太阴", "太阳", "廉贞", "武曲", "贪狼", "巨门", "天梁", "七杀", "破军"}
	default:
		return []string{"紫微", "贪狼", "巨门", "禄存", "文曲", "廉贞", "武曲", "破军", "天府", "太阴", "天梁", "七杀", "天机"}
	}
}

func distributeFuXing(chart *ZiweiChart, year int) {
	yearZhi := (chart.Year - 4) % 12
	if yearZhi < 0 {
		yearZhi += 12
	}

	fuPositions := map[string]int{
		"左辅": (yearZhi + 2) % 12,
		"右弼": (yearZhi + 10) % 12,
		"文昌": (yearZhi + 4) % 12,
		"文曲": (yearZhi + 10) % 12,
		"天魁": (yearZhi + 1) % 12,
		"天钺": (yearZhi + 11) % 12,
		"火星": (yearZhi + 5) % 12,
		"铃星": (yearZhi + 9) % 12,
		"擎羊": (yearZhi + 6) % 12,
		"陀罗": (yearZhi + 8) % 12,
	}

	for star, pos := range fuPositions {
		gongName := Gongs[pos]
		chart.FuXing[gongName] = append(chart.FuXing[gongName], star)
	}

	for i := 0; i < 12; i++ {
		gongName := Gongs[i]
		if _, exists := chart.FuXing[gongName]; !exists {
			chart.FuXing[gongName] = []string{}
		}
	}
}

func calculateSiHua(chart *ZiweiChart) {
	nianGan := (chart.Year - 4) % 10
	if nianGan < 0 {
		nianGan += 10
	}

	luGong := getSiHuaLu(nianGan)
	quanGong := getSiHuaQuan(nianGan)
	keGong := getSiHuaKe(nianGan)
	jiGong := getSiHuaJi(nianGan)

	chart.SiHua["禄"] = luGong
	chart.SiHua["权"] = quanGong
	chart.SiHua["科"] = keGong
	chart.SiHua["忌"] = jiGong
}

func getSiHuaLu(nianGan int) string {
	switch nianGan {
	case 0, 5:
		return "破军"
	case 1, 6:
		return "贪狼"
	case 2, 7:
		return "太阳"
	case 3, 8:
		return "天梁"
	case 4, 9:
		return "太阴"
	default:
		return "贪狼"
	}
}

func getSiHuaQuan(nianGan int) string {
	switch nianGan {
	case 0, 5:
		return "贪狼"
	case 1, 6:
		return "天府"
	case 2, 7:
		return "太阴"
	case 3, 8:
		return "天机"
	case 4, 9:
		return "天同"
	default:
		return "天府"
	}
}

func getSiHuaKe(nianGan int) string {
	switch nianGan {
	case 0, 5:
		return "巨门"
	case 1, 6:
		return "武曲"
	case 2, 7:
		return "天机"
	case 3, 8:
		return "文昌"
	case 4, 9:
		return "文曲"
	default:
		return "武曲"
	}
}

func getSiHuaJi(nianGan int) string {
	switch nianGan {
	case 0, 5:
		return "廉贞"
	case 1, 6:
		return "七杀"
	case 2, 7:
		return "天同"
	case 3, 8:
		return "天梁"
	case 4, 9:
		return "太阳"
	default:
		return "七杀"
	}
}

func getDizhiIndex(dizhi string) int {
	for i, d := range DizhiNames {
		if d == dizhi {
			return i
		}
	}
	return 0
}

func getLunarMonthDays(month int) int {
	if month < 1 || month > 12 {
		return 30
	}
	daysMap := map[int]int{
		1: 30, 2: 29, 3: 30, 4: 29, 5: 30, 6: 29,
		7: 30, 8: 30, 9: 29, 10: 30, 11: 29, 12: 30,
	}
	if days, ok := daysMap[month]; ok {
		return days
	}
	return 30
}

func getFuXingString(fuXing []string) string {
	if len(fuXing) == 0 {
		return ""
	}
	result := fuXing[0]
	for i := 1; i < len(fuXing); i++ {
		result += "、" + fuXing[i]
	}
	return result
}

func generateAnalysis(chart *ZiweiChart) string {
	analysis := fmt.Sprintf("出生年份：%d年\n", chart.Year)
	analysis += fmt.Sprintf("四柱：%s %s %s %s\n", chart.NianZhu, chart.YueZhu, chart.RiZhu, chart.ShiZhu)
	analysis += fmt.Sprintf("命宫：%s\n", chart.MingGong)
	analysis += fmt.Sprintf("身宫：%s\n", chart.ShenGong)

	analysis += "\n【主星分布】\n"
	for _, gong := range Gongs {
		if zhuXing, ok := chart.ZhuXing[gong]; ok {
			analysis += fmt.Sprintf("%s：%s\n", gong, zhuXing)
		}
	}

	analysis += "\n【四化】\n"
	analysis += fmt.Sprintf("禄：%s\n", chart.SiHua["禄"])
	analysis += fmt.Sprintf("权：%s\n", chart.SiHua["权"])
	analysis += fmt.Sprintf("科：%s\n", chart.SiHua["科"])
	analysis += fmt.Sprintf("忌：%s\n", chart.SiHua["忌"])

	analysis += "\n【辅星】\n"
	hasFuXing := false
	for _, gong := range Gongs {
		if fuXing, ok := chart.FuXing[gong]; ok && len(fuXing) > 0 {
			analysis += fmt.Sprintf("%s：%s\n", gong, getFuXingString(fuXing))
			hasFuXing = true
		}
	}
	if !hasFuXing {
		analysis += "无特殊辅星\n"
	}

	mingGongZhuXing := chart.ZhuXing[chart.MingGong]
	analysis += "\n【命宫主星分析】\n"
	analysis += getZhuXingAnalysis(mingGongZhuXing)

	return analysis
}

func getZhuXingAnalysis(zhuXing string) string {
	analysisMap := map[string]string{
		"紫微": "紫微星君临，美满福禄，地位显赫，有领导才能，适合管理岗位。",
		"天机": "天机星临命，聪明机智，思维敏捷，适合策划、创意、技术类工作。",
		"太阳": "太阳星照耀，热情开朗，积极主动，适合教育、公共关系、销售类工作。",
		"武曲": "武曲星刚毅果断，行动力强，适合金融、技术、军事类工作。",
		"天同": "天同星温和仁慈，人缘好，适合服务、医疗、教育类工作。",
		"廉贞": "廉贞星情感丰富，善于表达，适合艺术、文学、演艺类工作。",
		"天府": "天府星稳重可靠，理财能力强，适合财务、管理、稳定类工作。",
		"太阴": "太阴星细腻温柔，想象力丰富，适合艺术、设计、文职类工作。",
		"贪狼": "贪狼星欲望强烈，适应力强，适合商业、娱乐、社交类工作。",
		"巨门": "巨门星口才出众，分析力强，适合法律、教育、媒体类工作。",
		"天梁": "天梁星成熟稳重，有责任感，适合教育、宗教、慈善类工作。",
		"七杀": "七杀星刚强果断，敢于冒险，适合执法、军事、创业类工作。",
		"破军": "破军星变革创新，不畏艰难，适合改革、技术、开创类工作。",
	}

	if analysis, ok := analysisMap[zhuXing]; ok {
		return analysis
	}
	return "命主性格温和，做事稳重，适合稳定的工作环境。"
}

func GetZiweiInfo(year, month, day, hour int) map[string]interface{} {
	chart := CalculateZiweiChart(year, month, day, hour, "")

	return map[string]interface{}{
		"nian_zhu":        chart.NianZhu,
		"yue_zhu":         chart.YueZhu,
		"ri_zhu":          chart.RiZhu,
		"shi_zhu":         chart.ShiZhu,
		"ming_gong":       chart.MingGong,
		"shen_gong":       chart.ShenGong,
		"zhu_xing":        chart.ZhuXing,
		"fu_xing":         chart.FuXing,
		"si_hua":          chart.SiHua,
		"analysis":        chart.Analysis,
		"lucky_direction": getLuckyDirection(year),
		"lucky_color":     getLuckyColor(year),
		"lucky_number":    getLuckyNumber(year),
		"career_suggestion": getCareerSuggestion(chart.ZhuXing[chart.MingGong]),
		"relationship":    getRelationshipAdvice(chart.ZhuXing[chart.MingGong]),
	}
}

func getLuckyDirection(year int) string {
	directions := []string{"东方", "南方", "西方", "北方", "东南方", "西南方", "东北方", "西北方"}
	index := year % len(directions)
	return directions[index]
}

func getLuckyColor(year int) string {
	colors := []string{"红色", "黄色", "蓝色", "绿色", "紫色", "白色", "黑色", "金色"}
	index := year % len(colors)
	return colors[index]
}

func getLuckyNumber(year int) int {
	return (year % 9) + 1
}

func getCareerSuggestion(zhuXing string) string {
	careerMap := map[string]string{
		"紫微": "适合从政、管理、领导岗位",
		"贪狼": "适合商业、艺术、娱乐行业",
		"巨门": "适合教育、法律、媒体行业",
		"武曲": "适合军事、金融、技术行业",
		"天机": "适合科技、策划、创意行业",
		"天同": "适合服务、医疗、教育行业",
		"太阴": "适合艺术、设计、文职行业",
		"太阳": "适合教育、公共服务、销售行业",
		"天梁": "适合教育、宗教、慈善行业",
		"七杀": "适合军事、执法、冒险行业",
		"破军": "适合创业、改革、技术行业",
		"天府": "适合管理、财务、稳定行业",
		"廉贞": "适合艺术、文学、演艺行业",
	}

	if career, ok := careerMap[zhuXing]; ok {
		return career
	}
	return "适合稳定的工作环境，注重团队合作"
}

func getRelationshipAdvice(zhuXing string) string {
	relationMap := map[string]string{
		"紫微": "感情中注重尊严，需要伴侣理解支持",
		"贪狼": "感情丰富，需要伴侣有共同兴趣",
		"巨门": "口才出众，需要伴侣善于沟通",
		"武曲": "性格刚毅，需要伴侣温柔体贴",
		"天机": "思维敏捷，需要伴侣聪明伶俐",
		"天同": "性格温和，需要伴侣真诚相待",
		"太阴": "感情细腻，需要伴侣细心呵护",
		"太阳": "热情开朗，需要伴侣积极乐观",
		"天梁": "传统保守，需要伴侣稳重可靠",
		"七杀": "个性强势，需要伴侣独立自信",
		"破军": "喜欢挑战，需要伴侣勇敢坚强",
		"天府": "注重实际，需要伴侣踏实肯干",
		"廉贞": "情感丰富，需要伴侣能理解艺术气质",
	}

	if advice, ok := relationMap[zhuXing]; ok {
		return advice
	}
	return "感情中需要相互理解，共同成长"
}
