package ziwei

import (
	"fmt"
	"math"
	"name/internal/domain/bazi/tyme"
	"sync"
)

var (
	ziweiMutex sync.Mutex
)

const (
	Lu   = "禄"
	Quan = "权"
	Ke   = "科"
	Ji   = "忌"
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

// TigerRule 五虎遁：年干 → 正月天干起始索引
// 甲己→丙(2), 乙庚→戊(4), 丙辛→庚(6), 丁壬→壬(8), 戊癸→甲(0)
var TigerRule = [10]int{2, 4, 6, 8, 0, 2, 4, 6, 8, 0}

// SoulStarTable 命主查表（按命宫地支索引）
// 子→贪狼, 丑→巨门, 寅→禄存, 卯→文曲, 辰→廉贞, 巳→武曲, 午→破军, 未→武曲, 申→廉贞, 酉→文曲, 戌→禄存, 亥→巨门
var SoulStarTable = [12]string{"贪狼", "巨门", "禄存", "文曲", "廉贞", "武曲", "破军", "武曲", "廉贞", "文曲", "禄存", "巨门"}

// BodyStarTable 身主查表（按年支索引）
// 子→火星, 丑→天相, 寅→天梁, 卯→天同, 辰→文昌, 巳→天机, 午→火星, 未→天相, 申→天梁, 酉→天同, 戌→文昌, 亥→天机
var BodyStarTable = [12]string{"火星", "天相", "天梁", "天同", "文昌", "天机", "火星", "天相", "天梁", "天同", "文昌", "天机"}

// FiveElementsClassTable 五行局查表
var FiveElementsClassTable = [5]string{"木三局", "金四局", "水二局", "火六局", "土五局"}

// GenderTable 性别 → 阴阳
var GenderTable = map[string]string{"男": "阳", "女": "阴"}

// YinZhiSet 阳支集合 (子寅辰午申戌 → index%2==0)
var YinZhiSet = [12]bool{true, false, true, false, true, false, true, false, true, false, true, false} // true=阳

// SiHuaTable 四化表：按年干索引 → [禄, 权, 科, 忌]
var SiHuaTable = [10][4]string{
	{"廉贞", "破军", "武曲", "太阳"}, // 甲
	{"天机", "天梁", "紫微", "太阴"}, // 乙
	{"天同", "天机", "文昌", "廉贞"}, // 丙
	{"太阴", "天同", "天机", "巨门"}, // 丁
	{"贪狼", "太阴", "右弼", "天机"}, // 戊
	{"武曲", "贪狼", "天梁", "文曲"}, // 己
	{"太阳", "武曲", "太阴", "天同"}, // 庚
	{"巨门", "太阳", "文曲", "文昌"}, // 辛
	{"天梁", "紫微", "左辅", "武曲"}, // 壬
	{"破军", "巨门", "太阴", "贪狼"}, // 癸
}

// LuCunTable 禄存查表：按年干索引 → 地支索引
var LuCunTable = [10]int{2, 3, 4, 5, 6, 7, 8, 9, 10, 11} // 甲→寅(2), 乙→卯(3), ..., 癸→子(10/→0)

// TianKuiTianYueTable 天魁天钺查表：按年干索引 → [天魁地支, 天钺地支]
// 阳干顺行到辰戌，阴干逆行到卯酉
var TianKuiTianYueTable = [10][2]int{
	{8, 10}, // 甲→戌(10), 酉(9) → 实际:阳干→戌(10)/辰(4)? 需要精确
}

type ZiweiChart struct {
	Year       int    `json:"year"`
	Month      int    `json:"month"`
	Day        int    `json:"day"`
	Hour       int    `json:"hour"`
	Gender     string `json:"gender"`

	NianZhu string `json:"nian_zhu"`
	YueZhu  string `json:"yue_zhu"`
	RiZhu   string `json:"ri_zhu"`
	ShiZhu  string `json:"shi_zhu"`

	MingGong string `json:"ming_gong"`
	ShenGong string `json:"shen_gong"`

	FiveElements string                `json:"five_elements"` // 五行局，如"火六局"
	Soul         string                `json:"soul"`          // 命主
	Body         string                `json:"body"`          // 身主
	DaXian       map[string]DaXianEntry `json:"da_xian"`     // 大限

	ZhuXing map[string]string   `json:"zhu_xing"`
	FuXing  map[string][]string `json:"fu_xing"`
	SiHua   map[string]string   `json:"si_hua"`

	Analysis string `json:"analysis"`
}

// DaXianEntry 大限条目
type DaXianEntry struct {
	Range         [2]int  `json:"range"`          // 起运年龄范围
	HeavenlyStem  string  `json:"heavenly_stem"`  // 天干
	EarthlyBranch string  `json:"earthly_branch"` // 地支
}

type ZiweiAnalysis struct {
	Tianzhu  string `json:"tianzhu"`
	Diwei    string `json:"diwei"`
	Lu       string `json:"lu"`
	Quan     string `json:"quan"`
	Ke       string `json:"ke"`
	Ji       string `json:"ji"`
	Zhushen  string `json:"zhushen"`
	Fuji     string `json:"fuji"`
	Xingyao  string `json:"xingyao"`
	Gongwei  string `json:"gongwei"`
	Analysis string `json:"analysis"`
}

// ============================================================
// 工具函数
// ============================================================

// fixIndex 取模修正，确保结果在 [0, mod) 范围内
func fixIndex(x, mod int) int {
	return ((x%mod)%mod + mod) % mod
}

// getGanIndex 获取天干索引
func getGanIndex(gan string) int {
	for i, g := range TianganNames {
		if g == gan {
			return i
		}
	}
	return 0
}

// getZhiIndex 获取地支索引
func getZhiIndex(zhi string) int {
	for i, z := range DizhiNames {
		if z == zhi {
			return i
		}
	}
	return 0
}

// getGanZhi 干支索引 → 干支字符串
func getGanZhi(ganIndex, zhiIndex int) string {
	return TianganNames[fixIndex(ganIndex, 10)] + DizhiNames[fixIndex(zhiIndex, 12)]
}

// hourToInt 将小时转换为时辰索引 (0=子时, 1=丑时, ..., 11=亥时, 12=晚子时)
func hourToInt(hour int) int {
	if hour == 23 {
		return 12 // 晚子时
	}
	return (hour + 1) / 2
}

// ============================================================
// 四柱计算（使用 tyme 库，正月初一分界）
// ============================================================

// calculateFourPillars 使用 tyme 库计算四柱
func calculateFourPillars(year, month, day, hour int) (nianZhu, yueZhu, riZhu, shiZhu string, lunarYear, lunarMonth, lunarDayNum int, yearGanIndex int) {
	// 使用 tyme 库的 SolarDay 进行公历→农历转换
	solarDay, _ := tyme.SolarDay{}.FromYmd(year, month, day)
	lunarDayObj := solarDay.GetLunarDay()
	lunarMonthObj := lunarDayObj.GetLunarMonth()
	lunarYearObj := lunarMonthObj.GetLunarYear()

	lunarYear = lunarYearObj.GetYear()
	lunarMonth = lunarMonthObj.GetMonthWithLeap() // 闰月为负数
	lunarDayNum = lunarDayObj.GetDay()

	// 年柱（正月初一分界，与 iztro 配置 yearDivide:'normal' 一致）
	nianZhuObj := lunarYearObj.GetSixtyCycle()
	nianZhu = nianZhuObj.GetName()
	yearGanIndex = nianZhuObj.GetHeavenStem().GetIndex()

	// 获取六十甲子（使用节令分界，用于日柱）
	sixtyCycleDay := lunarDayObj.GetSixtyCycleDay()

	// 月柱（使用农历月 + 正月初一年干 + 五虎遁）
	// 农历月: 1=正月(寅), 2=二月(卯), ..., 12=腊月(丑)
	// 闰月: 用绝对值作为月份索引（闰八月→按九月计算），不减1
	var monthIdx int
	if lunarMonth > 0 {
		monthIdx = lunarMonth - 1 // 0-based: 0=正月, 1=二月, ..., 11=腊月
	} else {
		monthIdx = -lunarMonth // 闰月：直接使用绝对值
	}
	monthGanIdx := fixIndex(TigerRule[yearGanIndex]+monthIdx, 10)
	monthZhiIdx := fixIndex(monthIdx+2, 12) // 正月=寅(2), 二月=卯(3), ...
	yueZhu = TianganNames[monthGanIdx] + DizhiNames[monthZhiIdx]

	// 日柱（晚子时需日进位）
	var riZhuObj tyme.SixtyCycle
	if hour == 23 {
		riZhuObj = lunarDayObj.Next(1).GetSixtyCycleDay().GetSixtyCycle()
	} else {
		riZhuObj = sixtyCycleDay.GetSixtyCycle()
	}
	riZhu = riZhuObj.GetName()

	// 时柱（tyme 库内部已处理晚子时日进位）
	lunarHour, _ := tyme.LunarHour{}.FromYmdHms(lunarYearObj.GetYear(), lunarMonthObj.GetMonthWithLeap(), lunarDayObj.GetDay(), hour, 0, 0)
	shiZhuObj := lunarHour.GetSixtyCycleHour().GetSixtyCycle()
	shiZhu = shiZhuObj.GetName()

	return
}

// ============================================================
// 命宫/身宫计算
// ============================================================

// calculateMingGong 计算命宫（干支格式）
func calculateMingGong(yearGanIndex, lunarMonth, timeIndex int) string {
	// monthIndex: lunarMonth - 1（如果闰月且timeIndex==12且lunarDay>15，则需调整）
	monthIndex := lunarMonth - 1
	if monthIndex < 0 {
		monthIndex = -monthIndex - 1 // 闰月处理
	}

	// soulIndex = fixIndex(monthIndex - timeIndex%12)
	soulIndex := fixIndex(monthIndex-timeIndex%12, 12)

	// 命宫天干 = TigerRule[年干] + soulIndex
	ganIndex := TigerRule[yearGanIndex] + soulIndex

	// 命宫地支 = fixIndex(soulIndex + 2)
	zhiIndex := fixIndex(soulIndex+2, 12)

	return getGanZhi(ganIndex, zhiIndex)
}

// calculateShenGong 计算身宫（干支格式）
func calculateShenGong(yearGanIndex, lunarMonth, timeIndex int) string {
	monthIndex := lunarMonth - 1
	if monthIndex < 0 {
		monthIndex = -monthIndex - 1
	}

	// bodyIndex = fixIndex(monthIndex + timeIndex%12)
	bodyIndex := fixIndex(monthIndex+timeIndex%12, 12)

	// 身宫天干 = TigerRule[年干] + bodyIndex
	ganIndex := TigerRule[yearGanIndex] + bodyIndex

	// 身宫地支 = fixIndex(bodyIndex + 2)
	zhiIndex := fixIndex(bodyIndex+2, 12)

	return getGanZhi(ganIndex, zhiIndex)
}

// ============================================================
// 五行局计算
// ============================================================

// calculateFiveElementsClass 计算五行局
func calculateFiveElementsClass(mingGong string) string {
	ganStr := string([]rune(mingGong)[0])
	zhiStr := string([]rune(mingGong)[1])

	ganIdx := getGanIndex(ganStr)
	zhiIdx := getZhiIndex(zhiStr)

	// 干数 = floor(ganIdx/2) + 1
	ganNum := int(math.Floor(float64(ganIdx)/2)) + 1

	// 支数 = floor(fixIndex(zhiIdx, 6)/2) + 1
	zhiNum := int(math.Floor(float64(fixIndex(zhiIdx, 6))/2)) + 1

	sum := ganNum + zhiNum
	if sum > 5 {
		sum -= 5
	}

	return FiveElementsClassTable[sum-1]
}

// ============================================================
// 命主/身主
// ============================================================

// calculateSoul 计算命主（按命宫地支）
func calculateSoul(mingGong string) string {
	zhiStr := string([]rune(mingGong)[1])
	zhiIdx := getZhiIndex(zhiStr)
	return SoulStarTable[zhiIdx]
}

// calculateBody 计算身主（按年支）
func calculateBody(yearGanZhi string) string {
	zhiStr := string([]rune(yearGanZhi)[1])
	zhiIdx := getZhiIndex(zhiStr)
	return BodyStarTable[zhiIdx]
}

// ============================================================
// 四化计算
// ============================================================

// calculateSiHua 计算四化
func calculateSiHua(yearGanIndex int) map[string]string {
	siHua := SiHuaTable[yearGanIndex]
	return map[string]string{
		"禄": siHua[0],
		"权": siHua[1],
		"科": siHua[2],
		"忌": siHua[3],
	}
}

// ============================================================
// 大限计算
// ============================================================

// calculateDaXian 计算大限（与 iztro getHoroscope 保持一致）
// 阳男阴女顺行，阴男阳女逆行（以年支阴阳与性别匹配判断）
// 大限天干 = 年干五虎遁起正月干 + 宫位寅序索引
// 大限地支 = 宫位地支（寅=2+宫位索引）
func calculateDaXian(yearGanZhi, mingGong string, fiveElements string, gender string) map[string]DaXianEntry {
	// 五行局数
	var juShu int
	switch fiveElements {
	case "水二局":
		juShu = 2
	case "木三局":
		juShu = 3
	case "金四局":
		juShu = 4
	case "土五局":
		juShu = 5
	case "火六局":
		juShu = 6
	}

	// 年干、年支
	yearGan := string([]rune(yearGanZhi)[0])
	yearZhi := string([]rune(yearGanZhi)[1])
	yearGanIdx := getGanIndex(yearGan)
	yearZhiIdx := getZhiIndex(yearZhi)

	// 命宫地支 → 命宫寅序索引（0=寅）
	mingZhi := string([]rune(mingGong)[1])
	mingZhiIdx := getZhiIndex(mingZhi)
	soulIdx := fixIndex(mingZhiIdx-2, 12)

	// 顺行判断：性别与年支阴阳一致则顺行（阳男阴女顺行）
	isMale := gender == "男"
	isYearYang := YinZhiSet[yearZhiIdx] // true=阳支
	gongForward := (isMale && isYearYang) || (!isMale && !isYearYang)

	daXian := make(map[string]DaXianEntry)

	for i := 0; i < 12; i++ {
		var idx int
		if gongForward {
			idx = fixIndex(soulIdx+i, 12)
		} else {
			idx = fixIndex(soulIdx-i, 12)
		}

		// 宫名：逆行→顺时针（命宫,兄弟,夫妻,...）；顺行→逆时针（命宫,父母,福德,...）
		var gongName string
		if gongForward {
			gongName = Gongs[fixIndex(-i, 12)]
		} else {
			gongName = Gongs[i]
		}

		// 大限天干 = 年干五虎遁正月干 + 宫位寅序索引
		ganIdx := fixIndex(TigerRule[yearGanIdx]+idx, 10)

		// 大限地支 = 宫位地支（继承自地支序：寅=2+idx）
		zhiIdx := fixIndex(2+idx, 12)

		// 大限年龄 = [五行局数 + i*10, 五行局数 + i*10 + 9]
		startAge := juShu + i*10
		endAge := startAge + 9

		daXian[gongName] = DaXianEntry{
			Range:         [2]int{startAge, endAge},
			HeavenlyStem:  TianganNames[ganIdx],
			EarthlyBranch: DizhiNames[zhiIdx],
		}
	}

	return daXian
}

// ============================================================
// 紫微星定位
// ============================================================

// getZiweiPosition 计算紫微星位置
func getZiweiPosition(lunarDayNum int, fiveElementsClass string, lunarMonth int) int {
	// 五行局数
	var juShu int
	switch fiveElementsClass {
	case "水二局":
		juShu = 2
	case "木三局":
		juShu = 3
	case "金四局":
		juShu = 4
	case "土五局":
		juShu = 5
	case "火六局":
		juShu = 6
	}

	// 需要加值才能整除的最小数
	remainder := lunarDayNum % juShu
	var addValue int
	if remainder != 0 {
		addValue = juShu - remainder
	}

	// 商
	quotient := (lunarDayNum + addValue) / juShu

	// 偏移量计算
	qianduanPairs := [][2]int{
		{6, 9}, // 六→酉(9)
		{5, 6}, // 五→午(6)
		{4, 11}, // 四→亥(11)
		{3, 4}, // 三→辰(4)
		{2, 1}, // 二→丑(1)
	}

	var offset int
	for _, pair := range qianduanPairs {
		if quotient >= pair[0] {
			offset = pair[1]
			break
		}
	}

	// 紫微星位置
	ziweiIndex := fixIndex(offset+quotient-1, 12)

	return ziweiIndex
}

// ============================================================
// 主星分布
// ============================================================

// distributeZhuXing 分布14主星
func distributeZhuXing(chart *ZiweiChart, lunarDayNum int, lunarMonth int) {
	// 紫微星位置
	ziweiPos := getZiweiPosition(lunarDayNum, chart.FiveElements, lunarMonth)

	// 紫微系星（顺时针）
	ziweiStars := []string{"紫微", "天机", "太阳", "武曲", "天同", "廉贞", "空", "天府", "太阴", "贪狼", "巨门", "天梁", "七杀", "空"}

	// 天府系星（逆时针）
	tianfuStars := []string{"天府", "太阴", "贪狼", "巨门", "天梁", "七杀", "破军", "空", "紫微", "天机", "太阳", "武曲", "天同", "廉贞"}

	for i := 0; i < 12; i++ {
		gongName := Gongs[i]

		// 紫微系星
		starIdx := fixIndex(ziweiPos+i, 12)
		if starIdx < len(ziweiStars) && ziweiStars[starIdx] != "空" {
			chart.ZhuXing[gongName] = ziweiStars[starIdx]
		}

		// 天府系星
		tianfuIdx := fixIndex(12-ziweiPos+i, 12)
		if tianfuIdx < len(tianfuStars) && tianfuStars[tianfuIdx] != "空" {
			if chart.ZhuXing[gongName] == "" {
				chart.ZhuXing[gongName] = tianfuStars[tianfuIdx]
			}
		}
	}
}

// ============================================================
// 辅星分布
// ============================================================

// distributeFuXing 分布辅星
func distributeFuXing(chart *ZiweiChart, lunarMonth, timeIndex, yearGanIndex int) {
	// 初始化所有宫位辅星
	for i := 0; i < 12; i++ {
		chart.FuXing[Gongs[i]] = []string{}
	}

	// 左辅：monthIndex+1 顺数到地支序
	leftFuIdx := fixIndex(lunarMonth, 12)
	gongIdx := fixIndex(leftFuIdx-1, 12) // 转换为宫位序
	chart.FuXing[Gongs[gongIdx]] = append(chart.FuXing[Gongs[gongIdx]], "左辅")

	// 右弼：12-monthIndex 逆数到地支序
	rightBiIdx := fixIndex(12-lunarMonth, 12)
	gongIdx = fixIndex(rightBiIdx-1, 12)
	chart.FuXing[Gongs[gongIdx]] = append(chart.FuXing[Gongs[gongIdx]], "右弼")

	// 文昌：timeIndex 顺数到地支序
	wenChangIdx := fixIndex(timeIndex, 12)
	gongIdx = fixIndex(wenChangIdx-1, 12)
	chart.FuXing[Gongs[gongIdx]] = append(chart.FuXing[Gongs[gongIdx]], "文昌")

	// 文曲：11-timeIndex 逆数到地支序
	wenQuIdx := fixIndex(11-timeIndex, 12)
	gongIdx = fixIndex(wenQuIdx-1, 12)
	chart.FuXing[Gongs[gongIdx]] = append(chart.FuXing[Gongs[gongIdx]], "文曲")

	// 天魁天钺（简化实现）
	tianKuiIdx := fixIndex(yearGanIndex*2+8, 12) // 简化计算
	tianYueIdx := fixIndex(yearGanIndex*2+9, 12)
	gongIdx = fixIndex(tianKuiIdx-1, 12)
	chart.FuXing[Gongs[gongIdx]] = append(chart.FuXing[Gongs[gongIdx]], "天魁")
	gongIdx = fixIndex(tianYueIdx-1, 12)
	chart.FuXing[Gongs[gongIdx]] = append(chart.FuXing[Gongs[gongIdx]], "天钺")

	// 禄存
	luCunIdx := LuCunTable[yearGanIndex]
	gongIdx = fixIndex(luCunIdx-1, 12)
	chart.FuXing[Gongs[gongIdx]] = append(chart.FuXing[Gongs[gongIdx]], "禄存")

	// 擎羊（禄存+1）
	qingYangIdx := fixIndex(luCunIdx+1, 12)
	gongIdx = fixIndex(qingYangIdx-1, 12)
	chart.FuXing[Gongs[gongIdx]] = append(chart.FuXing[Gongs[gongIdx]], "擎羊")

	// 陀罗（禄存-1）
	tuoLuoIdx := fixIndex(luCunIdx-1, 12)
	gongIdx = fixIndex(tuoLuoIdx-1, 12)
	chart.FuXing[Gongs[gongIdx]] = append(chart.FuXing[Gongs[gongIdx]], "陀罗")
}

// ============================================================
// 主接口
// ============================================================

func AnalyzeZiwei(year, month, day, hour int, gender string) *ZiweiAnalysis {
	ziweiMutex.Lock()
	defer ziweiMutex.Unlock()

	chart := CalculateZiweiChart(year, month, day, hour, gender)

	// 命宫主星
	mingGongStar := chart.ZhuXing[chart.MingGong]
	if mingGongStar == "" {
		mingGongStar = "无主星"
	}

	return &ZiweiAnalysis{
		Tianzhu:  mingGongStar,
		Diwei:    chart.ZhuXing["田宅宫"],
		Lu:       chart.SiHua["禄"],
		Quan:     chart.SiHua["权"],
		Ke:       chart.SiHua["科"],
		Ji:       chart.SiHua["忌"],
		Zhushen:  chart.Soul,
		Fuji:     getFuXingString(chart.FuXing["命宫"]),
		Xingyao:  getAllXingYao(chart),
		Gongwei:  chart.MingGong,
		Analysis: chart.Analysis,
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
		DaXian:   make(map[string]DaXianEntry),
	}

	// 1. 计算四柱
	nianZhu, yueZhu, riZhu, shiZhu, _, lunarMonth, lunarDayNum, yearGanIndex := calculateFourPillars(year, month, day, hour)
	chart.NianZhu = nianZhu
	chart.YueZhu = yueZhu
	chart.RiZhu = riZhu
	chart.ShiZhu = shiZhu

	// 2. 计算命宫/身宫（干支格式）
	timeIndex := hourToInt(hour)
	chart.MingGong = calculateMingGong(yearGanIndex, lunarMonth, timeIndex)
	chart.ShenGong = calculateShenGong(yearGanIndex, lunarMonth, timeIndex)

	// 3. 计算五行局
	chart.FiveElements = calculateFiveElementsClass(chart.MingGong)

	// 4. 计算命主/身主
	chart.Soul = calculateSoul(chart.MingGong)
	chart.Body = calculateBody(chart.NianZhu)

	// 5. 计算四化
	chart.SiHua = calculateSiHua(yearGanIndex)

	// 6. 分布主星
	distributeZhuXing(chart, lunarDayNum, lunarMonth)

	// 7. 分布辅星
	distributeFuXing(chart, lunarMonth, timeIndex, yearGanIndex)

	// 8. 计算大限
	chart.DaXian = calculateDaXian(chart.NianZhu, chart.MingGong, chart.FiveElements, gender)

	// 9. 生成分析
	chart.Analysis = generateAnalysis(chart)

	return chart
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

func getAllXingYao(chart *ZiweiChart) string {
	var allStars []string

	// 主星
	if star, ok := chart.ZhuXing[chart.MingGong]; ok && star != "" {
		allStars = append(allStars, star)
	}

	// 辅星
	if fuXing, ok := chart.FuXing[chart.MingGong]; ok {
		allStars = append(allStars, fuXing...)
	}

	if len(allStars) == 0 {
		return "无"
	}
	return getFuXingString(allStars)
}

func generateAnalysis(chart *ZiweiChart) string {
	analysis := fmt.Sprintf("出生年份：%d年\n", chart.Year)
	analysis += fmt.Sprintf("四柱：%s %s %s %s\n", chart.NianZhu, chart.YueZhu, chart.RiZhu, chart.ShiZhu)
	analysis += fmt.Sprintf("命宫：%s\n", chart.MingGong)
	analysis += fmt.Sprintf("身宫：%s\n", chart.ShenGong)
	analysis += fmt.Sprintf("五行局：%s\n", chart.FiveElements)
	analysis += fmt.Sprintf("命主：%s\n", chart.Soul)
	analysis += fmt.Sprintf("身主：%s\n", chart.Body)

	analysis += "\n【主星分布】\n"
	for _, gong := range Gongs {
		if zhuXing, ok := chart.ZhuXing[gong]; ok && zhuXing != "" {
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
