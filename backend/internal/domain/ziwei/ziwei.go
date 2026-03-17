package ziwei

import (
	"fmt"
	"time"
)

type ZiweiChart struct {
	BirthTime     time.Time `json:"birth_time"`
	SolarYear     int       `json:"solar_year"`
	SolarMonth    int       `json:"solar_month"`
	SolarDay      int       `json:"solar_day"`
	SolarHour     int       `json:"solar_hour"`
	LunarYear     int       `json:"lunar_year"`
	LunarMonth    int       `json:"lunar_month"`
	LunarDay      int       `json:"lunar_day"`
	IsLeapMonth   bool      `json:"is_leap_month"`
	Gender        string    `json:"gender"`
	DestinyHouse  int       `json:"destiny_house"`
	BodyHouse     int       `json:"body_house"`
	Stars         map[int][]string `json:"stars"`
	Houses        []House   `json:"houses"`
}

type House struct {
	Index     int      `json:"index"`
	Name      string   `json:"name"`
	Stars     []string `json:"stars"`
	MainStar  string   `json:"main_star"`
	Origin    string   `json:"origin"`
}

var HouseNames = []string{
	"命宫", "兄弟宫", "夫妻宫", "子女宫",
	"财帛宫", "疾厄宫", "迁移宫", "奴仆宫",
	"官禄宫", "田宅宫", "福德宫", "父母宫",
}

var MajorStars = []string{
	"紫微", "天机", "太阳", "武曲", "天同", "廉贞",
	"天府", "太阴", "贪狼", "巨门", "天相", "天梁",
	"七杀", "破军",
}

var SecondaryStars = []string{
	"左辅", "右弼", "文昌", "文曲", "天魁", "天钺",
	"火星", "铃星", "擎羊", "陀罗", "禄存", "天马",
	"天空", "地劫", "化禄", "化权", "化科", "化忌",
}

var DecadeStars = []string{
	"紫微星", "天机星", "太阳星", "武曲星", "天同星", "廉贞星",
	"天府星", "太阴星", "贪狼星", "巨门星", "天相星", "天梁星",
	"七杀星", "破军星",
}

var HourBranchNames = []string{
	"子", "丑", "寅", "卯", "辰", "巳",
	"午", "未", "申", "酉", "戌", "亥",
}

func GetZiweiChart(year, month, day, hour int, gender string) *ZiweiChart {
	chart := &ZiweiChart{
		SolarYear:    year,
		SolarMonth:   month,
		SolarDay:     day,
		SolarHour:    hour,
		Gender:       gender,
		Stars:        make(map[int][]string),
		Houses:       make([]House, 12),
	}

	for i := 0; i < 12; i++ {
		chart.Houses[i] = House{
			Index: i,
			Name:  HouseNames[i],
		}
	}

	chart.calculateMingGong()
	chart.calculateShenGong()
	chart.placeMajorStars()
	chart.placeSecondaryStars()

	return chart
}

func (c *ZiweiChart) calculateMingGong() {
	stemIndex := (c.SolarYear - 4) % 10
	branchIndex := (c.SolarYear - 4) % 12

	base := (stemIndex*2 + branchIndex) % 12
	mingGong := (base + 12 - c.SolarHour/2) % 12

	c.DestinyHouse = mingGong
}

func (c *ZiweiChart) calculateShenGong() {
	mingGong := c.DestinyHouse

	stemIndex := (c.SolarYear - 4) % 10
	shenGong := (mingGong + 6 + stemIndex%3) % 12

	c.BodyHouse = shenGong
}

func (c *ZiweiChart) placeMajorStars() {
	destinyHouse := c.DestinyHouse

	ziweiPos := (destinyHouse + 1) % 12
	c.Stars[ziweiPos] = append(c.Stars[ziweiPos], "紫微")

	tianjiPos := (destinyHouse + 3) % 12
	c.Stars[tianjiPos] = append(c.Stars[tianjiPos], "天机")

	sunPos := (destinyHouse + 4) % 12
	c.Stars[sunPos] = append(c.Stars[sunPos], "太阳")

	wuquPos := (destinyHouse + 5) % 12
	c.Stars[wuquPos] = append(c.Stars[wuquPos], "武曲")

	tiantongPos := (destinyHouse + 6) % 12
	c.Stars[tiantongPos] = append(c.Stars[tiantongPos], "天同")

	lianzhenPos := (destinyHouse + 7) % 12
	c.Stars[lianzhenPos] = append(c.Stars[lianzhenPos], "廉贞")

	tianfuPos := (destinyHouse + 8) % 12
	c.Stars[tianfuPos] = append(c.Stars[tianfuPos], "天府")

	taiyinPos := (destinyHouse + 9) % 12
	c.Stars[taiyinPos] = append(c.Stars[taiyinPos], "太阴")

	tanlangPos := (destinyHouse + 10) % 12
	c.Stars[tanlangPos] = append(c.Stars[tanlangPos], "贪狼")

	jumenPos := (destinyHouse + 11) % 12
	c.Stars[jumenPos] = append(c.Stars[jumenPos], "巨门")

	tianxiangPos := destinyHouse
	c.Stars[tianxiangPos] = append(c.Stars[tianxiangPos], "天相")

	tianliangPos := (destinyHouse + 1) % 12
	c.Stars[tianliangPos] = append(c.Stars[tianliangPos], "天梁")

	qishahPos := (destinyHouse + 2) % 12
	c.Stars[qishahPos] = append(c.Stars[qishahPos], "七杀")

	pojunPos := (destinyHouse + 3) % 12
	c.Stars[pojunPos] = append(c.Stars[pojunPos], "破军")
}

func (c *ZiweiChart) placeSecondaryStars() {
	destinyHouse := c.DestinyHouse
	hour := c.SolarHour

	zuofuPos := (destinyHouse + 2) % 12
	c.Stars[zuofuPos] = append(c.Stars[zuofuPos], "左辅")

	youbiPos := (destinyHouse + 10) % 12
	c.Stars[youbiPos] = append(c.Stars[youbiPos], "右弼")

	wenchangPos := (destinyHouse + 4) % 12
	c.Stars[wenchangPos] = append(c.Stars[wenchangPos], "文昌")

	wenquPos := (destinyHouse + 8) % 12
	c.Stars[wenquPos] = append(c.Stars[wenquPos], "文曲")

	tiankuiPos := (destinyHouse + 1) % 12
	c.Stars[tiankuiPos] = append(c.Stars[tiankuiPos], "天魁")

	tianyuePos := (destinyHouse + 9) % 12
	c.Stars[tianyuePos] = append(c.Stars[tianyuePos], "天钺")

	huoxingPos := (destinyHouse + 5) % 12
	c.Stars[huoxingPos] = append(c.Stars[huoxingPos], "火星")

	lingxingPos := (destinyHouse + 11) % 12
	c.Stars[lingxingPos] = append(c.Stars[lingxingPos], "铃星")

	qingyangPos := (destinyHouse + 6) % 12
	c.Stars[qingyangPos] = append(c.Stars[qingyangPos], "擎羊")

	tuoluoPos := (destinyHouse + 12 - ((destinyHouse + 6) % 12)) % 12
	if tuoluoPos == destinyHouse {
		tuoluoPos = (destinyHouse + 6) % 12
	}
	c.Stars[tuoluoPos] = append(c.Stars[tuoluoPos], "陀罗")

	lucunPos := (c.SolarYear - 4) % 12
	c.Stars[lucunPos] = append(c.Stars[lucunPos], "禄存")

	tianmaPos := (destinyHouse + 4 + hour/2) % 12
	c.Stars[tianmaPos] = append(c.Stars[tianmaPos], "天马")

	tiankongPos := (destinyHouse + 7) % 12
	c.Stars[tiankongPos] = append(c.Stars[tiankongPos], "天空")

	dijiePos := (destinyHouse + 1) % 12
	c.Stars[dijiePos] = append(c.Stars[dijiePos], "地劫")
}

func GetStarMeaning(star string) string {
	meanings := map[string]string{
		"紫微": "帝王星，尊贵、权力、领导力",
		"天机": "智慧星，思考、策划、机敏",
		"太阳": "光明星，事业、名望、父亲",
		"武曲": "财星，果断、坚毅、财富",
		"天同": "福星，享乐、温和、福气",
		"廉贞": "次桃花星，忠诚、廉洁、感情",
		"天府": "财库星，稳定、保守、财富",
		"太阴": "母星，温柔、阴柔、月亮",
		"贪狼": "桃花星，欲望、交际、野心",
		"巨门": "是非星，口才、神秘、纠纷",
		"天相": "官禄星，服务、辅佐、中立",
		"天梁": "荫星，慈善、长辈、稳定",
		"七杀": "将星，果断、冲动、变革",
		"破军": "耗星，破坏、创新、变动",
		"左辅": "辅佐星，帮助、支持、忠臣",
		"右弼": "辅佐星，帮助、支持、忠臣",
		"文昌": "文星，学术、考试、文化",
		"文曲": "文星，才艺、演艺、浪漫",
		"天魁": "贵人星，机会、帮助、地位",
		"天钺": "贵人星，机会、帮助、地位",
		"火星": "煞星，冲动、火灾、急性",
		"铃星": "煞星，冲动、火灾、演艺",
		"擎羊": "煞星，伤害、刀光、争执",
		"陀罗": "煞星，纠结、拖延、障碍",
		"禄存": "财星，财富、稳定、正财",
		"天马": "驿马星，变动、旅行、移动",
		"天空": "空亡星，想象、空灵、精神",
		"地劫": "劫财星，损失、破耗、抢夺",
	}

	if meaning, ok := meanings[star]; ok {
		return meaning
	}
	return "未知星曜"
}

type StarAnalysis struct {
	Star       string   `json:"star"`
	House      string   `json:"house"`
	Meaning    string   `json:"meaning"`
	Strength   string   `json:"strength"`
	Advice     string   `json:"advice"`
}

func (c *ZiweiChart) AnalyzeStars() []StarAnalysis {
	var analyses []StarAnalysis

	for houseIdx, stars := range c.Stars {
		for _, star := range stars {
			analysis := StarAnalysis{
				Star:    star,
				House:   HouseNames[houseIdx],
				Meaning: GetStarMeaning(star),
			}

			if isMajorStar(star) {
				analysis.Strength = "主星"
				analysis.Advice = getMajorStarAdvice(star, houseIdx)
			} else if isSecondaryStar(star) {
				analysis.Strength = "副星"
				analysis.Advice = getSecondaryStarAdvice(star, houseIdx)
			} else {
				analysis.Strength = "煞星"
				analysis.Advice = "需注意化解"
			}

			analyses = append(analyses, analysis)
		}
	}

	return analyses
}

func isMajorStar(star string) bool {
	for _, s := range MajorStars {
		if s == star {
			return true
		}
	}
	return false
}

func isSecondaryStar(star string) bool {
	for _, s := range SecondaryStars {
		if s == star {
			return true
		}
	}
	return false
}

func getMajorStarAdvice(star string, houseIdx int) string {
	adviceMap := map[string]string{
		"紫微": "发挥领导才能，注意培养下属",
		"天机": "多动脑筋，保持灵活性",
		"太阳": "积极进取，注意父子关系",
		"武曲": "果断决策，财运亨通",
		"天同": "知足常乐，注意健康",
		"廉贞": "坚守原则，注意感情",
		"天府": "保守理财，积累财富",
		"太阴": "内敛温柔，注意母亲",
		"贪狼": "控制欲望，避免桃花劫",
		"巨门": "谨言慎行，避免口舌",
		"天相": "中庸之道，辅佐他人",
		"天梁": "慈悲为怀，长辈缘佳",
		"七杀": "果断勇敢，忌冲动",
		"破军": "破旧立新，变动较大",
	}

	if advice, ok := adviceMap[star]; ok {
		return advice
	}
	return "保持平衡发展"
}

func getSecondaryStarAdvice(star string, houseIdx int) string {
	adviceMap := map[string]string{
		"左辅": "多获帮助，人际关系佳",
		"右弼": "多获帮助，人际关系佳",
		"文昌": "学业进步，考试运佳",
		"文曲": "艺术才华，表达能力强",
		"天魁": "贵人相助，机会多多",
		"天钺": "贵人相助，机会多多",
		"火星": "注意防火，避免冲动",
		"铃星": "注意情绪，避免纠纷",
		"擎羊": "注意安全，避免血光",
		"陀罗": "注意拖延，耐心化解",
		"禄存": "正财稳定，理财有道",
		"天马": "适合变动，驿马运旺",
		"天空": "发挥想象，精神追求",
		"地劫": "注意破财，保管财物",
	}

	if advice, ok := adviceMap[star]; ok {
		return advice
	}
	return "正常发展"
}

func (c *ZiweiChart) GetHouseDescription(houseIdx int) string {
	descriptions := []string{
		"命宫：代表本人，性格、相貌、命格",
		"兄弟宫：代表兄弟姐妹、合作关系",
		"夫妻宫：代表婚姻、感情、配偶",
		"子女宫：代表子女、桃花、欲望",
		"财帛宫：代表财运、收入、理财",
		"疾厄宫：代表健康、疾病、灾祸",
		"迁移宫：代表外出、旅行、迁移",
		"奴仆宫：代表朋友、下属、合作关系",
		"官禄宫：代表事业、职位、权力",
		"田宅宫：代表房产、不动产、家运",
		"福德宫：代表福气、享受、兴趣",
		"父母宫：代表父母、长辈、上司",
	}

	if houseIdx >= 0 && houseIdx < 12 {
		return descriptions[houseIdx]
	}
	return ""
}

func (c *ZiweiChart) String() string {
	result := fmt.Sprintf("紫微命盘 - %d年%d月%d日 %s时\n", 
		c.SolarYear, c.SolarMonth, c.SolarDay, HourBranchNames[c.SolarHour/2])
	result += fmt.Sprintf("命宫: %s, 身宫: %s\n", 
		HouseNames[c.DestinyHouse], HouseNames[c.BodyHouse])
	result += "\n各宫星曜:\n"

	for i := 0; i < 12; i++ {
		stars := c.Stars[i]
		result += fmt.Sprintf("%s: ", HouseNames[i])
		for _, star := range stars {
			result += star + " "
		}
		result += "\n"
	}

	return result
}
