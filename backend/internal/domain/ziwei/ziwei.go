package ziwei

import (
	"fmt"
	"strings"

	"name/internal/domain/bazi/tyme"
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

// SuiXingList 杂曜清单（24 颗）：原 12 颗 + Excel 派扩展 12 颗
// （天刑/三台/八座/恩光/天贵/天才/天寿/台辅/封诰/天官/天福/天厨，
// docs/41 第 7 节遗留项），安布口径与 iztro 2.6.3 对拍（见 ziwei_excel_test.go）。
var SuiXingList = []string{
	"天姚", "天哭", "天虚", "龙池", "凤阁",
	"红鸾", "天喜", "孤辰", "寡宿",
	"亡神", "劫煞", "大耗",
	"天刑", "三台", "八座", "恩光", "天贵",
	"天才", "天寿", "台辅", "封诰",
	"天官", "天福", "天厨",
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

// LuCunTable 禄存查表：按年干索引 → 地支索引（子0丑1寅2…亥11）
// 口诀「甲禄到寅宫，乙禄居卯位，丙戊禄在巳，丁己禄居午，
// 庚禄居申位，辛禄到酉方，壬禄亥中藏，癸禄子中藏」。
// 原表为简单递增 2..11，自丙干起全部错位，擎羊陀罗随之全错。
var LuCunTable = [10]int{2, 3, 5, 6, 5, 6, 8, 9, 11, 0}

// TianKuiTianYueTable 天魁天钺查表：按年干索引 → [天魁地支, 天钺地支]
// 口诀「甲戊庚牛羊，乙己鼠猴乡，丙丁猪鸡位，壬癸兔蛇藏，六辛逢马虎」
var TianKuiTianYueTable = [10][2]int{
	{1, 7},  // 甲→丑(牛), 未(羊)
	{0, 8},  // 乙→子(鼠), 申(猴)
	{11, 9}, // 丙→亥(猪), 酉(鸡)
	{11, 9}, // 丁→亥(猪), 酉(鸡)
	{1, 7},  // 戊→丑, 未
	{0, 8},  // 己→子, 申
	{1, 7},  // 庚→丑, 未
	{6, 2},  // 辛→午(马), 寅(虎)
	{3, 5},  // 壬→卯(兔), 巳(蛇)
	{3, 5},  // 癸→卯, 巳
}

// HuoLingTable 火星铃星起点表：三合分组号 = 年支索引 % 4（0=申子辰、
// 1=巳酉丑、2=寅午戌、3=亥卯未）→ [火星起点支, 铃星起点支]（子起索引），
// 自起点顺数至生时。
// 口诀「申子辰人寅戌扬，寅午戌人丑卯方，巳酉丑人卯戌位，亥卯未人酉戌房」
var HuoLingTable = [4][2]int{
	{2, 10}, // 申子辰：火星寅起、铃星戌起
	{3, 10}, // 巳酉丑：火卯、铃戌
	{1, 3},  // 寅午戌：火丑、铃卯
	{9, 10}, // 亥卯未：火酉、铃戌
}

// JieKongTable 截路空亡查表：分组号 = 年干索引 % 5（0=甲己、1=乙庚、
// 2=丙辛、3=丁壬、4=戊癸）→ [截路支, 空亡支]（子起索引）。
// 口诀「甲己之年申酉方，乙庚之年午未当，丙辛辰巳，丁壬寅卯，戊癸子丑」
var JieKongTable = [5][2]int{{8, 9}, {6, 7}, {4, 5}, {2, 3}, {0, 1}}

// GuGuaTable 孤辰寡宿查表：分组号 = ((年支+1) mod 12) / 3（0=亥子丑、
// 1=寅卯辰、2=巳午未、3=申酉戌）→ [孤辰支, 寡宿支]（子起索引）。
// 口诀「寅卯辰年安巳丑，巳午未年安申辰，申酉戌年安亥未，亥子丑年安寅戌」
var GuGuaTable = [4][2]int{{2, 10}, {5, 1}, {8, 4}, {11, 7}}

// JieShaTable 劫煞查表：三合分组号 = 年支索引 % 4（组序同 HuoLingTable）
// → 劫煞支（子起索引）。「申子辰人蛇开口，亥卯未人猴速走，
// 寅午戌人猪面黑，巳酉丑人虎咆哮」；亡神居劫煞对宫，并入本表推算。
var JieShaTable = [4]int{5, 2, 11, 8}

// tianGuanTable 天官查表（年干系，子起索引）。
// 口诀「甲喜羊鸡乙龙猴，丙年蛇鼠一窝谋。丁虎擒猪戊玉兔，己鸡居然与虎俦。
// 庚猪马辛鸡蛇走，壬犬马癸马蛇游」前字为天官：
// 甲未 乙辰 丙巳 丁寅 戊卯 己酉 庚亥 辛酉 壬戌 癸午（与 iztro 2.6.3 一致）
var tianGuanTable = [10]int{7, 4, 5, 2, 3, 9, 11, 9, 10, 6}

// tianFuXingTable 天福查表（年干系，子起索引）。口诀后字为天福：
// 甲酉 乙申 丙子 丁亥 戊卯 己寅 庚午 辛巳 壬午 癸巳（与 iztro 2.6.3 一致）
var tianFuXingTable = [10]int{9, 8, 0, 11, 3, 2, 6, 5, 6, 5}

// tianChuTable 天厨查表（年干系，子起索引）。
// 口诀「甲丁食蛇口，乙戊辛马方。丙从鼠口得，己食于猴房。
// 庚食虎头上，壬鸡癸猪堂」：
// 甲巳 乙午 丙子 丁巳 戊午 己申 庚寅 辛午 壬酉 癸亥（与 iztro 2.6.3 一致）
var tianChuTable = [10]int{5, 6, 0, 5, 6, 8, 2, 6, 9, 11}

type ZiweiChart struct {
	Year   int    `json:"year"`
	Month  int    `json:"month"`
	Day    int    `json:"day"`
	Hour   int    `json:"hour"`
	Gender string `json:"gender"`

	NianZhu string `json:"nian_zhu"`
	YueZhu  string `json:"yue_zhu"`
	RiZhu   string `json:"ri_zhu"`
	ShiZhu  string `json:"shi_zhu"`

	MingGong string `json:"ming_gong"`
	ShenGong string `json:"shen_gong"`

	FiveElements string                 `json:"five_elements"` // 五行局，如"火六局"
	Soul         string                 `json:"soul"`          // 命主
	Body         string                 `json:"body"`          // 身主
	DaXian       map[string]DaXianEntry `json:"da_xian"`       // 大限

	ZhuXing map[string][]string `json:"zhu_xing"` // 宫名 → 主星列表（同宫双星全部保留，紫微系在前）
	FuXing  map[string][]string `json:"fu_xing"`
	SuiXing map[string][]string `json:"sui_xing"` // 宫名 → 杂曜列表
	SiHua   map[string]string   `json:"si_hua"`

	// Brightness 主星亮度（《紫微斗数全书》派：庙/旺/得/利/平/不/陷），
	// 键为主星名，与 ZhuXing 落宫一一对应
	Brightness map[string]string `json:"brightness"`

	Analysis string `json:"analysis"`
}

// DaXianEntry 大限条目
type DaXianEntry struct {
	Range         [2]int `json:"range"`          // 起运年龄范围
	HeavenlyStem  string `json:"heavenly_stem"`  // 天干
	EarthlyBranch string `json:"earthly_branch"` // 地支
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
// effectiveMonthIndex 闰月「按日切」口径（与 iztro fixLeap 默认一致）：
// 闰X月， lunarDayNum<=15 → 上个月(X月)，lunarDayNum>15 → 下个月(X+1月)；
// 非闰月直接返回 lunarMonth-1。返回值 0-based：0=正月 … 11=腊月。
// 月柱/命宫/辅星/杂曜共用此口径，避免同一闰月盘内部月份基准冲突。
func effectiveMonthIndex(lunarMonth, lunarDayNum int) int {
	if lunarMonth > 0 {
		return lunarMonth - 1
	}
	x := -lunarMonth // 闰的月份号 X
	if lunarDayNum <= 15 {
		return x - 1 // 上个月 = X月 → 0-based X-1
	}
	return x // 下个月 = X+1月 → 0-based X
}

func calculateFourPillars(year, month, day, hour int) (nianZhu, yueZhu, riZhu, shiZhu string, lunarYear, lunarMonth, lunarDayNum int, yearGanIndex int) {
	// 使用 tyme 库的 SolarDay 进行公历→农历转换。
	// 日期不合法（如 2024-02-30）时 FromYmd 返回 nil，旧代码吞错后
	// nil 解引用 panic（docs/29 A2），现返回零值由调用方得到退化排盘。
	solarDay, err := tyme.SolarDay{}.FromYmd(year, month, day)
	if err != nil || solarDay == nil {
		return
	}
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
	// 月柱口径：与命宫/辅星共用「按日切」闰月规则（iztro fixLeap 默认）
	monthIdx := effectiveMonthIndex(lunarMonth, lunarDayNum)
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
	lunarHour, hourErr := tyme.LunarHour{}.FromYmdHms(lunarYearObj.GetYear(), lunarMonthObj.GetMonthWithLeap(), lunarDayObj.GetDay(), hour, 0, 0)
	if hourErr != nil || lunarHour == nil {
		return
	}
	shiZhuObj := lunarHour.GetSixtyCycleHour().GetSixtyCycle()
	shiZhu = shiZhuObj.GetName()

	return
}

// ============================================================
// 命宫/身宫计算
// ============================================================

// calculateMingGong 计算命宫（干支格式）
func calculateMingGong(yearGanIndex, lunarMonth, lunarDayNum, timeIndex int) string {
	// monthIndex: 与月柱/辅星共用「按日切」闰月规则（iztro fixLeap 默认）
	monthIndex := effectiveMonthIndex(lunarMonth, lunarDayNum)

	// soulIndex = fixIndex(monthIndex - timeIndex%12)
	soulIndex := fixIndex(monthIndex-timeIndex%12, 12)

	// 命宫天干 = TigerRule[年干] + soulIndex
	ganIndex := TigerRule[yearGanIndex] + soulIndex

	// 命宫地支 = fixIndex(soulIndex + 2)
	zhiIndex := fixIndex(soulIndex+2, 12)

	return getGanZhi(ganIndex, zhiIndex)
}

// calculateShenGong 计算身宫（干支格式）
func calculateShenGong(yearGanIndex, lunarMonth, lunarDayNum, timeIndex int) string {
	// monthIndex: 与月柱/辅星共用「按日切」闰月规则（iztro fixLeap 默认）
	monthIndex := effectiveMonthIndex(lunarMonth, lunarDayNum)

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
// 由命宫干支的纳音五行定局：水二局、木三局、金四局、土五局、火六局。
// 原实现以「干数+支数取模」自造公式，与纳音无关，导致紫微落宫与
// 大限起运年龄整体错位（如命宫甲子纳音海中金应为金四局，旧式算出水二局）。
// 命宫干支非法或纳音末字无法识别五行时返回 ("", false) —— 不再静默降级为
// 「水二局」，由调用方（CalculateZiweiChart）标记为退化盘并提示，
// 与 docs/39「失败不再静默降级」原则一致。
func calculateFiveElementsClass(mingGong string) (string, bool) {
	runes := []rune(mingGong)
	if len(runes) < 2 {
		return "", false
	}
	sc, err := tyme.SixtyCycle{}.FromName(string(runes))
	if err != nil || sc == nil {
		// 命宫干支非法：显式返回失败，不再兜底「水二局」
		return "", false
	}
	sound := sc.GetSound().GetName()
	switch string([]rune(sound)[len([]rune(sound))-1]) {
	case "水":
		return "水二局", true
	case "木":
		return "木三局", true
	case "金":
		return "金四局", true
	case "土":
		return "土五局", true
	case "火":
		return "火六局", true
	}
	return "", false
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
	juShu := juShuOf(fiveElements)

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
	// 历史缺陷：isMale 只认 "男"，而 API 枚举是 "male"/"female"（见
	// ziwei/interfaces.go 声明），"male" 全被按女性逆行计算——紫微大限
	// 方向整片反置且无任何报错。现同时接受中文与英文枚举。
	isMale := gender == "男" || gender == "male"
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

// juShuOf 五行局 → 局数
func juShuOf(fiveElementsClass string) int {
	switch fiveElementsClass {
	case "水二局":
		return 2
	case "木三局":
		return 3
	case "金四局":
		return 4
	case "土五局":
		return 5
	case "火六局":
		return 6
	}
	return 0
}

// getZiweiPosition 计算紫微星位置（借表法还原公式，返回地支索引 0=子）
// 商 q = ⌈生日/局数⌉，加值 add = q×局数 - 生日；add 为偶数时自寅顺数
// q-1+add 位，为奇数时自寅顺数 q-1 位再逆退 add 位。
// 已对水二/木三/金四/土五/火六局逐日锚点与《紫微斗数全书》安星表核对
// （见 ziwei_position_test.go）。原实现只有 5 组偏移且商超出范围即失准。
func getZiweiPosition(lunarDayNum int, fiveElementsClass string, lunarMonth int) int {
	juShu := juShuOf(fiveElementsClass)
	if juShu <= 0 || lunarDayNum <= 0 {
		return 2
	}

	q := (lunarDayNum + juShu - 1) / juShu
	add := q*juShu - lunarDayNum

	if add%2 == 0 {
		return fixIndex(2+q-1+add, 12)
	}
	return fixIndex(2+q-1-add, 12)
}

// ============================================================
// 主星分布
// ============================================================

// distributeZhuXing 分布14主星
//
// 紫微系自紫微起逆布（地支递减）：天机-1、空、太阳-3、武曲-4、天同-5、空二宫、廉贞-8；
// 天府与紫微以寅申轴对称：天府位 = (4 - 紫微位) mod 12；
// 天府系自天府起顺布（地支递增）：太阴+1、贪狼+2、巨门+3、天相+4、天梁+5、七杀+6、空三宫、破军+10。
// 宫名按地支定位：命宫居命宫地支，十二宫自命宫逆行（兄弟=命宫地支-1，以此类推）。
// 同宫双星全部保留：宫内顺序紫微系在前、天府系在后，与经典并称（紫府、日月、
// 同巨、武贪、廉杀等）一致。历史缺陷：ZhuXing 曾为单星 map，后布的天府系
// 覆盖先布的紫微系，每盘静默丢失 3~4 颗主星。
// 原实现把紫微系天府系 14 星混入一个数组按宫序线性铺开，下标 12/13 永不可达
// （破军不可能出现在任何宫），且把宫序当宫位地支。
func distributeZhuXing(chart *ZiweiChart, lunarDayNum int, lunarMonth int) {
	ziweiPos := getZiweiPosition(lunarDayNum, chart.FiveElements, lunarMonth)

	mingRunes := []rune(chart.MingGong)
	if len(mingRunes) < 2 {
		return
	}
	mingZhiIdx := getZhiIndex(string(mingRunes[1]))
	palaceByZhi := func(zhi int) string {
		return Gongs[fixIndex(mingZhiIdx-zhi, 12)]
	}

	// 紫微系（逆布）
	ziweiSeries := []struct {
		offset int
		star   string
	}{
		{0, "紫微"}, {1, "天机"}, {3, "太阳"}, {4, "武曲"}, {5, "天同"}, {8, "廉贞"},
	}
	if chart.Brightness == nil {
		chart.Brightness = make(map[string]string)
	}
	for _, s := range ziweiSeries {
		zhi := fixIndex(ziweiPos-s.offset, 12)
		p := palaceByZhi(zhi)
		chart.ZhuXing[p] = append(chart.ZhuXing[p], s.star)
		chart.Brightness[s.star] = brightnessOf(s.star, DizhiNames[zhi])
	}

	// 天府系（顺布）
	tianfuPos := fixIndex(4-ziweiPos, 12)
	tianfuSeries := []struct {
		offset int
		star   string
	}{
		{0, "天府"}, {1, "太阴"}, {2, "贪狼"}, {3, "巨门"}, {4, "天相"}, {5, "天梁"}, {6, "七杀"}, {10, "破军"},
	}
	for _, s := range tianfuSeries {
		zhi := fixIndex(tianfuPos+s.offset, 12)
		p := palaceByZhi(zhi)
		chart.ZhuXing[p] = append(chart.ZhuXing[p], s.star)
		chart.Brightness[s.star] = brightnessOf(s.star, DizhiNames[zhi])
	}
}

// ============================================================
// 主星亮度（全书派）
// ============================================================

// brightnessNames 亮度数值 → 中文档位：1=庙 2=旺 3=得 4=利 5=平 6=不 7=陷
var brightnessNames = [8]string{"", "庙", "旺", "得", "利", "平", "不", "陷"}

// zhuXingBrightnessRows 14 主星亮度表（《紫微斗数全书》派，按地支自子起一）。
// 数据核对自 Renhuai123/ziwei-doushu v4.0-13d 的 lib/ziwei/wenmo-data.ts
// 中 WENMO_BRIGHTNESS_QS（MIT 许可，数据核对式使用，非代码移植）；
// 该版本 2026-06-20 已订正「子起一、1=庙…7=陷、0=留白」口径。
// 与 iztro 2.6.3 对拍：14 主星落宫亮度全部命中（含太阳在亥——QS 派与
// iztro 在 v4.0-13d 已统一记「陷」，不再存在历史文档所述的派差）。
// 注：天府在 v4.0-13d 恰为 12 个地支值（亥=得），旧版曾多一尾数，现按
// 上游 12 值原样采用，不再做「取前 12 值」截断。
var zhuXingBrightnessRows = []struct {
	star  string
	codes [12]uint8
}{
	{"紫微", [12]uint8{5, 1, 2, 2, 3, 2, 1, 1, 2, 2, 3, 2}},
	{"天机", [12]uint8{1, 7, 3, 2, 4, 5, 1, 7, 3, 2, 4, 5}},
	{"太阳", [12]uint8{7, 6, 2, 1, 2, 2, 2, 3, 3, 5, 6, 7}},
	{"武曲", [12]uint8{2, 1, 3, 4, 1, 5, 2, 1, 3, 4, 1, 5}},
	{"天同", [12]uint8{2, 6, 4, 5, 5, 1, 7, 6, 2, 5, 5, 1}},
	{"廉贞", [12]uint8{5, 4, 1, 5, 4, 7, 5, 4, 1, 5, 4, 7}},
	{"天府", [12]uint8{1, 1, 1, 3, 1, 3, 2, 1, 3, 2, 1, 3}},
	{"太阴", [12]uint8{1, 1, 2, 7, 7, 7, 6, 6, 4, 2, 2, 1}},
	{"贪狼", [12]uint8{2, 1, 5, 4, 1, 7, 2, 1, 5, 4, 1, 7}},
	{"巨门", [12]uint8{2, 6, 1, 1, 7, 2, 2, 6, 1, 1, 7, 2}},
	{"天相", [12]uint8{1, 1, 1, 7, 3, 3, 1, 3, 1, 7, 3, 3}},
	{"天梁", [12]uint8{1, 2, 1, 1, 1, 3, 1, 2, 7, 3, 1, 7}},
	{"七杀", [12]uint8{2, 1, 1, 2, 1, 5, 2, 1, 1, 2, 1, 5}},
	{"破军", [12]uint8{1, 2, 3, 7, 2, 5, 1, 2, 3, 7, 2, 5}},
}

// brightnessOf 查主星在某地支的亮度档位（中文），未收录星返回空串
func brightnessOf(star, zhi string) string {
	col := getZhiIndex(zhi)
	for _, r := range zhuXingBrightnessRows {
		if r.star == star {
			return brightnessNames[r.codes[col]]
		}
	}
	return ""
}

// ============================================================
// 辅星分布
// ============================================================

// distributeFuXing 分布辅星（地支定位 → 宫名；十二宫自命宫地支逆行）
// 月系：左辅辰(4)上起正月顺行，右弼戌(10)上起正月逆行；
// 时系：文昌戌(10)上起子时逆行，文曲辰(4)上起子时顺行；
// 年干系：天魁/天钺按口诀查表，禄存按口诀查表，擎羊居禄存下一宫、陀罗居上一宫；
// 时系+年支系：火星铃星按三合起点顺数生时，地空地劫自亥顺逆数生时；
// 年支系：天空居年支下一位；年干支系：截空取阴阳相应支、旬空取旬中空亡与
// 年支阴阳同支；天伤居交友宫、天使居疾厄宫。
func distributeFuXing(chart *ZiweiChart, lunarMonth, lunarDayNum, timeIndex, yearGanIndex, yearZhiIndex int) {
	for i := range Gongs {
		chart.FuXing[Gongs[i]] = []string{}
	}

	mingRunes := []rune(chart.MingGong)
	if len(mingRunes) < 2 {
		return
	}
	mingZhiIdx := getZhiIndex(string(mingRunes[1]))
	palaceByZhi := func(zhi int) string {
		return Gongs[fixIndex(mingZhiIdx-zhi, 12)]
	}
	place := func(zhi int, star string) {
		p := palaceByZhi(fixIndex(zhi, 12))
		chart.FuXing[p] = append(chart.FuXing[p], star)
	}

	// 月系：与月柱/命宫共用「按日切」闰月规则（iztro fixLeap 默认）
	m := effectiveMonthIndex(lunarMonth, lunarDayNum) + 1 // 1-based 生效月份

	// 月系
	place(4+m-1, "左辅")
	place(10-(m-1), "右弼")

	// 时系（晚子时 timeIndex=12 经 fixIndex 归位到子时同宫）
	place(10-timeIndex, "文昌")
	place(4+timeIndex, "文曲")

	// 年干系：天魁天钺
	kuiYue := TianKuiTianYueTable[yearGanIndex]
	place(kuiYue[0], "天魁")
	place(kuiYue[1], "天钺")

	// 年干系：禄存及擎羊陀罗
	luCunIdx := LuCunTable[yearGanIndex]
	place(luCunIdx, "禄存")
	place(luCunIdx+1, "擎羊")
	place(luCunIdx-1, "陀罗")

	// 时系+年支系：火星铃星（先据生年支口诀定起子时位，顺数至生时）
	huoLing := HuoLingTable[yearZhiIndex%4]
	place(huoLing[0]+timeIndex, "火星")
	place(huoLing[1]+timeIndex, "铃星")

	// 时系：地空地劫（亥上子时顺安劫，逆回便是地空亡）
	place(11-timeIndex, "地空")
	place(11+timeIndex, "地劫")

	// 年支系：天空（生年支顺数前一位）
	place(yearZhiIndex+1, "天空")

	// 年干支系：截路空亡（阳年支取截路支，阴年支取空亡支；
	// 主流排盘另有正副双星同置成对两宫的派法）
	jieKong := JieKongTable[yearGanIndex%5]
	if YinZhiSet[yearZhiIndex] {
		place(jieKong[0], "截空")
	} else {
		place(jieKong[1], "截空")
	}

	// 年干支系：旬空（年支所在旬的空亡两支中，取与年支阴阳性相同的支；
	// 旬首支 = 年支 − 年干 + 10，其下一支即成对空亡的另一支）
	xunKong := fixIndex(yearZhiIndex-yearGanIndex+10, 12)
	if xunKong%2 != yearZhiIndex%2 {
		xunKong = fixIndex(xunKong+1, 12)
	}
	place(xunKong, "旬空")

	// 天伤天使：天伤居交友宫、天使居疾厄宫（以命宫起逆行宫序定位）
	place(mingZhiIdx-7, "天伤")
	place(mingZhiIdx-5, "天使")
}

// ============================================================
// 杂曜分布
// ============================================================

// distributeSuiXing 分布杂曜（SuiXingList 24 颗）。
// 年支系：红鸾（卯起子年逆数）与天喜（红鸾对宫）、天哭（午起子年逆行）
// 与天虚（午起子年顺转）、孤辰寡宿（三合组查表）、劫煞（三合组查表）
// 与亡神（劫煞对宫）、大耗（年支对宫，阳支顺行一位、阴支逆行一位）、
// 龙池（辰起子年顺数）与凤阁（戌起子年逆数）、
// 天才（命宫起子年顺数至生年支）与天寿（身宫起子年顺数至生年支）；
// 月系：天姚（丑起正月顺数）、天刑（酉起正月顺数）；
// 日系：三台（左辅起初一顺数至生日）、八座（右弼起初一逆数至生日）、
// 恩光（文昌起初一顺数至生日再退一步）、天贵（文曲起初一顺数至生日再退一步），
// 晚子时（timeIndex>=12）日系偏移按次日计（iztro fixLunarDayIndex 口径）；
// 时系：台辅（午起子时顺数至生时）、封诰（寅起子时顺数至生时）；
// 年干系：天官/天福/天厨查表（tianGuanTable/tianFuXingTable/tianChuTable）。
// Excel 派扩展 12 颗与 iztro 2.6.3 实排 8 盘对拍（ziwei_excel_test.go，docs/41 遗留）。
func distributeSuiXing(chart *ZiweiChart, lunarMonth, lunarDayNum, yearZhiIndex, yearGanIndex, timeIndex int) {
	if chart.SuiXing == nil {
		chart.SuiXing = make(map[string][]string)
	}
	for i := range Gongs {
		chart.SuiXing[Gongs[i]] = []string{}
	}

	mingRunes := []rune(chart.MingGong)
	if len(mingRunes) < 2 {
		return
	}
	mingZhiIdx := getZhiIndex(string(mingRunes[1]))
	place := func(zhi int, star string) {
		p := Gongs[fixIndex(mingZhiIdx-zhi, 12)]
		chart.SuiXing[p] = append(chart.SuiXing[p], star)
	}

	// 闰月「按日切」（与月柱/命宫/左辅右弼口径一致）
	m := effectiveMonthIndex(lunarMonth, lunarDayNum) + 1 // 1-based 生效月份

	hongLuan := fixIndex(3-yearZhiIndex, 12)
	place(hongLuan, "红鸾")
	place(hongLuan+6, "天喜")

	place(6-yearZhiIndex, "天哭")
	place(6+yearZhiIndex, "天虚")

	guGua := GuGuaTable[((yearZhiIndex+1)%12)/3]
	place(guGua[0], "孤辰")
	place(guGua[1], "寡宿")

	jieSha := JieShaTable[yearZhiIndex%4]
	place(jieSha, "劫煞")
	place(jieSha+6, "亡神")

	daHaoOffset := 1
	if !YinZhiSet[yearZhiIndex] {
		daHaoOffset = -1
	}
	place(yearZhiIndex+6+daHaoOffset, "大耗")

	place(4+yearZhiIndex, "龙池")
	place(10-yearZhiIndex, "凤阁")

	place(1+m-1, "天姚")

	// ---- Excel 派扩展（docs/41 遗留，口径对齐 iztro 2.6.3）----

	// 月系：天刑（酉起正月顺数至生月）
	place(9+m-1, "天刑")

	// 日系：三台/八座（左辅右弼起初一，顺/逆数至生日）、
	// 恩光/天贵（文昌文曲起初一顺数至生日再退一步）；晚子时生日按次日计
	dayOffset := lunarDayNum - 1
	if timeIndex >= 12 {
		dayOffset = lunarDayNum
	}
	zuoFu := fixIndex(4 + m - 1, 12)
	youBi := fixIndex(10 - (m - 1), 12)
	wenChang := fixIndex(10-timeIndex, 12)
	wenQu := fixIndex(4+timeIndex, 12)
	place(zuoFu+dayOffset, "三台")
	place(youBi-dayOffset, "八座")
	place(wenChang+dayOffset-1, "恩光")
	place(wenQu+dayOffset-1, "天贵")

	// 年支系：天才（命宫起子年顺数至生年支）、天寿（身宫起子年顺数至生年支）
	shenRunes := []rune(chart.ShenGong)
	if len(shenRunes) >= 2 {
		shenZhiIdx := getZhiIndex(string(shenRunes[1]))
		place(mingZhiIdx+yearZhiIndex, "天才")
		place(shenZhiIdx+yearZhiIndex, "天寿")
	}

	// 时系：台辅（午起子时顺数至生时）、封诰（寅起子时顺数至生时）；
	// 晚子时 timeIndex=12 经 place 内取模归位子时（iztro fixIndex(12)=0 同口径）
	place(6+timeIndex, "台辅")
	place(2+timeIndex, "封诰")

	// 年干系：天官/天福/天厨查表
	place(tianGuanTable[yearGanIndex], "天官")
	place(tianFuXingTable[yearGanIndex], "天福")
	place(tianChuTable[yearGanIndex], "天厨")
}

// ============================================================
// 主接口
// ============================================================

// AnalyzeZiwei 紫微斗数分析（无锁：tyme lazy-init 均 sync.Once 保护、
// 排盘为纯计算，并发安全由 TestCalculateZiweiChartConcurrentDeterminism
// 压测固化——原 ziweiMutex 把并发排盘串行化，docs/29 P3#22）
// AnalyzeZiwei 紫微斗数排盘入口：返回完整 ZiweiChart。
//
// 此前本函数把完整盘面压缩为 9 字段摘要（ZiweiAnalysis），导致 12 宫落星、
// 主星亮度、大限、辅星/杂曜分布全部丢失，前端「算而不显」。现直接返回
// CalculateZiweiChart 的完整结果（含 Analysis 文本与 Brightness 等全部字段），
// 由前端按需渲染。
func AnalyzeZiwei(year, month, day, hour int, gender string) *ZiweiChart {
	return CalculateZiweiChart(year, month, day, hour, gender)
}

func CalculateZiweiChart(year, month, day, hour int, gender string) *ZiweiChart {
	chart := &ZiweiChart{
		Year:    year,
		Month:   month,
		Day:     day,
		Hour:    hour,
		Gender:  gender,
		ZhuXing: make(map[string][]string),
		FuXing:  make(map[string][]string),
		SuiXing: make(map[string][]string),
		SiHua:   make(map[string]string),
		DaXian:  make(map[string]DaXianEntry),
	}

	// 1. 计算四柱
	nianZhu, yueZhu, riZhu, shiZhu, _, lunarMonth, lunarDayNum, yearGanIndex := calculateFourPillars(year, month, day, hour)
	chart.NianZhu = nianZhu
	chart.YueZhu = yueZhu
	chart.RiZhu = riZhu
	chart.ShiZhu = shiZhu

	// 年支索引（火铃/天空/截空旬空/杂曜年支系安布依赖；四柱异常时按子起 0 兜底）
	nianZhiIndex := 0
	if runes := []rune(nianZhu); len(runes) >= 2 {
		nianZhiIndex = getZhiIndex(string(runes[1]))
	}

	// 2. 计算命宫/身宫（干支格式）
	timeIndex := hourToInt(hour)
	chart.MingGong = calculateMingGong(yearGanIndex, lunarMonth, lunarDayNum, timeIndex)
	chart.ShenGong = calculateShenGong(yearGanIndex, lunarMonth, lunarDayNum, timeIndex)

	// 3. 计算五行局
	fiveElements, fiveOK := calculateFiveElementsClass(chart.MingGong)
	chart.FiveElements = fiveElements

	// 4. 计算命主/身主
	chart.Soul = calculateSoul(chart.MingGong)
	chart.Body = calculateBody(chart.NianZhu)

	// 5. 计算四化
	chart.SiHua = calculateSiHua(yearGanIndex)

	// 6. 分布主星（依赖五行局；五行局未定时跳过，避免错盘）
	if fiveOK {
		distributeZhuXing(chart, lunarDayNum, lunarMonth)
	}

	// 7. 分布辅星与杂曜
	distributeFuXing(chart, lunarMonth, lunarDayNum, timeIndex, yearGanIndex, nianZhiIndex)
	distributeSuiXing(chart, lunarMonth, lunarDayNum, nianZhiIndex, yearGanIndex, timeIndex)

	// 8. 计算大限（依赖五行局；五行局未定时跳过）
	if fiveOK {
		chart.DaXian = calculateDaXian(chart.NianZhu, chart.MingGong, chart.FiveElements, gender)
	}

	// 9. 生成分析
	chart.Analysis = generateAnalysis(chart)
	if !fiveOK {
		// 五行局未定：不静默降级为「水二局」，明确提示盘已退化
		chart.Analysis = "【排盘异常】命宫干支非法或纳音无法识别，未能定五行局；" +
			"已跳过主星安布与大限，未以「水二局」静默兜底。\n\n" + chart.Analysis
	}

	return chart
}

// joinStars 星曜列表 → 「、」连接（主星/辅星通用，空列表返回空串）
func joinStars(stars []string) string {
	if len(stars) == 0 {
		return ""
	}
	result := stars[0]
	for i := 1; i < len(stars); i++ {
		result += "、" + stars[i]
	}
	return result
}

func getAllXingYao(chart *ZiweiChart) string {
	var allStars []string

	// 主星
	allStars = append(allStars, chart.ZhuXing["命宫"]...)

	// 辅星
	if fuXing, ok := chart.FuXing["命宫"]; ok {
		allStars = append(allStars, fuXing...)
	}

	// 杂曜
	if suiXing, ok := chart.SuiXing["命宫"]; ok {
		allStars = append(allStars, suiXing...)
	}

	if len(allStars) == 0 {
		return "无"
	}
	return joinStars(allStars)
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
		if stars := chart.ZhuXing[gong]; len(stars) > 0 {
			analysis += fmt.Sprintf("%s：%s\n", gong, joinStars(stars))
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
			analysis += fmt.Sprintf("%s：%s\n", gong, joinStars(fuXing))
			hasFuXing = true
		}
	}
	if !hasFuXing {
		analysis += "无特殊辅星\n"
	}

	analysis += "\n【杂曜】\n"
	hasSuiXing := false
	for _, gong := range Gongs {
		if suiXing, ok := chart.SuiXing[gong]; ok && len(suiXing) > 0 {
			analysis += fmt.Sprintf("%s：%s\n", gong, joinStars(suiXing))
			hasSuiXing = true
		}
	}
	if !hasSuiXing {
		analysis += "无杂曜\n"
	}

	analysis += "\n【主星亮度】\n"
	for _, gong := range Gongs {
		stars := chart.ZhuXing[gong]
		if len(stars) == 0 {
			continue
		}
		parts := make([]string, 0, len(stars))
		for _, s := range stars {
			if b := chart.Brightness[s]; b != "" {
				parts = append(parts, fmt.Sprintf("%s（%s）", s, b))
			} else {
				parts = append(parts, s)
			}
		}
		analysis += fmt.Sprintf("%s：%s\n", gong, strings.Join(parts, "、"))
	}

	analysis += "\n【命宫主星分析】\n"
	analysis += getZhuXingAnalysis(chart.ZhuXing["命宫"])

	return analysis
}

func getZhuXingAnalysis(zhuXing []string) string {
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

	var descs []string
	for _, star := range zhuXing {
		if desc, ok := analysisMap[star]; ok {
			descs = append(descs, desc)
		}
	}
	if len(descs) == 0 {
		return "命主性格温和，做事稳重，适合稳定的工作环境。"
	}
	return strings.Join(descs, "；")
}
