package bazi

import (
	"fmt"
	"time"
	"name/internal/domain/bazi/tyme"
)

type LunarCalendar struct {
	Year    int    `json:"year"`
	Month   int    `json:"month"`
	Day     int    `json:"day"`
	IsLeap  bool   `json:"isLeap"`
	YearGan string `json:"yearGan"`
	YearZhi string `json:"yearZhi"`
	MonthGan string `json:"monthGan"`
	MonthZhi string `json:"monthZhi"`
	DayGan  string `json:"dayGan"`
	DayZhi  string `json:"dayZhi"`
	HourGan string `json:"hourGan"`
	HourZhi string `json:"hourZhi"`
	Nayin   string `json:"nayin"`
	YearNayin string `json:"yearNayin"`
	MonthNayin string `json:"monthNayin"`
	DayNayin string `json:"dayNayin"`
	// 农历显示字段 - 用于替代前端 chinese-lunar-calendar 库
	LunarYearName  string `json:"lunarYearName"`  // 如"甲辰年"
	LunarMonthName string `json:"lunarMonthName"` // 如"正月"
	LunarDayName   string `json:"lunarDayName"`   // 如"初一"
	DateStr        string `json:"dateStr"`         // 如"正月十五"
	LunarDate      string `json:"lunarDate"`       // 如"十五"，仅日期部分
	LunarMonth     string `json:"lunarMonth"`      // 如"正月"，仅月份部分
	SolarTerm      string `json:"solarTerm"`       // 如"清明"
}

func GetLunarCalendar(year, month, day, hour, minute int) *LunarCalendar {
	// 使用 tyme4go 库获取农历信息
	solarDay, _ := tyme.SolarDay{}.FromYmd(year, month, day)
	lunarDay := solarDay.GetLunarDay()
	lunarHour, _ := tyme.LunarHour{}.FromYmdHms(lunarDay.GetYear(), lunarDay.GetMonth(), lunarDay.GetDay(), hour, minute, 0)
	sixtyCycleDay := lunarDay.GetSixtyCycleDay()
	sixtyCycleHour := lunarHour.GetSixtyCycleHour()

	lc := &LunarCalendar{
		Year:    year,
		Month:   month,
		Day:     day,
		IsLeap:  lunarDay.GetLunarMonth().IsLeap(),
		YearGan: sixtyCycleDay.GetYear().GetHeavenStem().GetName(),
		YearZhi: sixtyCycleDay.GetYear().GetEarthBranch().GetName(),
		MonthGan: sixtyCycleDay.GetMonth().GetHeavenStem().GetName(),
		MonthZhi: sixtyCycleDay.GetMonth().GetEarthBranch().GetName(),
		DayGan:  sixtyCycleDay.GetSixtyCycle().GetHeavenStem().GetName(),
		DayZhi:  sixtyCycleDay.GetSixtyCycle().GetEarthBranch().GetName(),
		HourGan: sixtyCycleHour.GetSixtyCycle().GetHeavenStem().GetName(),
		HourZhi: sixtyCycleHour.GetSixtyCycle().GetEarthBranch().GetName(),
		YearNayin: sixtyCycleDay.GetYear().GetSound().GetName(),
		MonthNayin: sixtyCycleDay.GetMonth().GetSound().GetName(),
		DayNayin: sixtyCycleDay.GetSixtyCycle().GetSound().GetName(),
		Nayin: sixtyCycleDay.GetSixtyCycle().GetSound().GetName(),
		// 农历显示字段
		LunarYearName:  sixtyCycleDay.GetYear().GetHeavenStem().GetName() + sixtyCycleDay.GetYear().GetEarthBranch().GetName() + "年",
		LunarMonthName: lunarDay.GetLunarMonth().GetName(),
		LunarDayName:   lunarDay.GetName(),
		LunarMonth:     lunarDay.GetLunarMonth().GetName(),
		LunarDate:      lunarDay.GetName(),
		SolarTerm:      GetCurrentJieQi(year, month, day),
	}
	lc.DateStr = lc.LunarMonthName + lc.LunarDayName

	return lc
}

func GetYearGanzhi(year int) (gan, zhi string) {
	ganIndex := (year - 4) % 10
	if ganIndex < 0 {
		ganIndex += 10
	}
	zhiIndex := (year - 4) % 12
	if zhiIndex < 0 {
		zhiIndex += 12
	}
	return Tiangan[ganIndex], Dizhi[zhiIndex]
}

// GetMonthGanzhi 计算月份干支
// 月干根据年干推算：甲己之年丙作首，乙庚之岁戊为头，丙辛之岁寻庚起，丁壬壬位顺行流，戊癸何方发，甲寅之上好追求
// 月支固定：正月建寅，二月建卯，以此类推
func GetMonthGanzhi(year, month int) (gan, zhi string) {
	yearGan, _ := GetYearGanzhi(year)

	// 确定月干起始（根据年干）
	var startGanIndex int
	switch yearGan {
	case "甲", "己":
		startGanIndex = 2 // 丙
	case "乙", "庚":
		startGanIndex = 4 // 戊
	case "丙", "辛":
		startGanIndex = 6 // 庚
	case "丁", "壬":
		startGanIndex = 8 // 壬
	case "戊", "癸":
		startGanIndex = 0 // 甲
	}

	// 月支：正月建寅（索引2），二月建卯（索引3），以此类推
	// 月份1-12对应地支索引：2,3,4,5,6,7,8,9,10,11,0,1
	zhiIndex := (month + 1) % 12

	// 月干：从起始干开始，按月份偏移
	ganIndex := (startGanIndex + month - 1) % 10

	return Tiangan[ganIndex], Dizhi[zhiIndex]
}

func GetDayGanzhi(year, month, day int) (gan, zhi string) {
	date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	baseDate := time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)
	days := int(date.Sub(baseDate).Hours() / 24)

	ganIndex := (days - 1) % 10
	zhiIndex := (days - 1) % 12

	return Tiangan[ganIndex], Dizhi[zhiIndex]
}

func GetHourGanzhi(dayGan string, hour, minute int) (gan, zhi string) {
	hourIndex := hour
	if minute >= 30 {
		hourIndex++
	}
	if hourIndex >= 24 {
		hourIndex = 0
	}

	zhiIndex := (hourIndex + 1) / 2
	if zhiIndex > 11 {
		zhiIndex = 0
	}

	dayGanIndex := 0
	for i, g := range Tiangan {
		if g == dayGan {
			dayGanIndex = i
			break
		}
	}

	ganIndex := (dayGanIndex*2 + zhiIndex) % 10

	return Tiangan[ganIndex], Dizhi[zhiIndex]
}

func GetNayin(ganzhi string) string {
	if nayin, ok := NayinMap[ganzhi]; ok {
		return nayin
	}
	return ""
}

func GetNayinWuxing(nayin string) string {
	if wuxing, ok := NayinWuxingMap[nayin]; ok {
		return wuxing
	}
	return ""
}

type HourYiJi struct {
	TimeRange   string   `json:"timeRange"`
	TianShen    string   `json:"tianShen"`
	HeiHuangDao string   `json:"heiHuangDao"`
	JiXiong     string   `json:"jiXiong"`
	Yi          []string `json:"yi"`
	Ji          []string `json:"ji"`
}

type Huangli struct {
	Date        string      `json:"date"`
	Year        string      `json:"year"`
	Month       string      `json:"month"`
	Day         string      `json:"day"`
	Yi          []string    `json:"yi"`
	Ji          []string    `json:"ji"`
	JiShen      []string    `json:"jiShen"`
	XiongSha    []string    `json:"xiongSha"`
	BaiCang     string      `json:"baiCang"`
	PengZhu     string      `json:"pengZhu"`
	Fu          string      `json:"fu"`
	JieShen     string      `json:"jieShen"`
	Chong       string      `json:"chong"`
	Sha         string      `json:"sha"`
	ZhangSong   string      `json:"zhangSong"`
	YearWuxing  string      `json:"yearWuxing"`
	MonthWuxing string      `json:"monthWuxing"`
	DayWuxing   string      `json:"dayWuxing"`
	LiuYao      string      `json:"liuYao"`
	ZhiShen     string      `json:"zhiShen"`
	TaiShen     string      `json:"taiShen"`
	JianChu     string      `json:"jianChu"`
	XingXiu     string      `json:"xingXiu"`
	XiShen      string      `json:"xiShen"`
	FuShen      string      `json:"fuShen"`
	CaiShen     string      `json:"caiShen"`
	YangGui     string      `json:"yangGui"`
	YinGui      string      `json:"yinGui"`
	Hours       []HourYiJi  `json:"hours"`
}

var YiJiData = map[string]struct {
	Yi []string
	Ji []string
}{
	"子": {Yi: []string{"嫁娶", "纳采", "订盟", "祭祀", "祈福", "求嗣", "开光", "出行", "移徙", "入宅", "安床", "动土", "破土", "安葬"}, Ji: []string{"开市", "立券", "交易", "置产"}},
	"丑": {Yi: []string{"订盟", "纳采", "祭祀", "祈福", "修造", "起基", "定磉", "开光", "塑绘", "出货财", "安床", "造仓", "合脊"}, Ji: []string{"开市", "交易", "立券", "挂匾", "开仓", "出货财"}},
	"寅": {Yi: []string{"祭祀", "祈福", "求嗣", "开光", "出火", "出行", "移徙", "新造", "结网", "入宅", "补垣", "塞穴"}, Ji: []string{"开市", "立券", "交易", "置产", "分居", "安床", "伐木", "纳畜", "无畜市"}},
	"卯": {Yi: []string{"祭祀", "祈福", "求嗣", "开光", "塑绘", "出行", "移徙", "入宅", "安床", "扫舍", "求医", "治病"}, Ji: []string{"开市", "交易", "立券", "挂匾", "伐木", "架马", "开仓", "出货财"}},
	"辰": {Yi: []string{"祭祀", "祈福", "求嗣", "开光", "出行", "移徙", "入宅", "安床", "开仓", "出货财", "开市", "立券", "交易"}, Ji: []string{"动土", "破土", "安葬", "伐木", "纳畜", "无畜市"}},
	"巳": {Yi: []string{"嫁娶", "纳采", "订盟", "祭祀", "祈福", "求嗣", "开光", "出行", "移徙", "入宅", "竖柱", "上梁", "立券", "交易"}, Ji: []string{"开市", "立券", "交易", "挂匾", "开仓", "出货财", "伐木", "架马"}},
	"午": {Yi: []string{"祭祀", "祈福", "求嗣", "开光", "塑绘", "出行", "移徙", "入宅", "安床", "扫舍", "求医", "治病", "起基", "定磉"}, Ji: []string{"开市", "立券", "交易", "挂匾", "开仓", "出货财", "伐木", "架马"}},
	"未": {Yi: []string{"嫁娶", "纳采", "订盟", "祭祀", "祈福", "求嗣", "开光", "出行", "移徙", "入宅", "安床", "竖柱", "上梁", "立券"}, Ji: []string{"开市", "立券", "交易", "置产", "分居", "安床", "伐木", "纳畜", "无畜市"}},
	"申": {Yi: []string{"祭祀", "祈福", "求嗣", "开光", "出火", "出行", "移徙", "入宅", "安床", "开仓", "出货财", "开市", "立券", "交易"}, Ji: []string{"动土", "破土", "安葬", "伐木", "纳畜", "无畜市"}},
	"酉": {Yi: []string{"祭祀", "祈福", "求嗣", "开光", "塑绘", "出行", "移徙", "入宅", "安床", "扫舍", "求医", "治病"}, Ji: []string{"开市", "交易", "立券", "挂匾", "伐木", "架马", "开仓", "出货财"}},
	"戌": {Yi: []string{"嫁娶", "纳采", "订盟", "祭祀", "祈福", "求嗣", "开光", "出行", "移徙", "入宅", "安床", "竖柱", "上梁", "立券"}, Ji: []string{"开市", "立券", "交易", "挂匾", "开仓", "出货财", "动土", "破土"}},
	"亥": {Yi: []string{"祭祀", "祈福", "求嗣", "开光", "塑绘", "出行", "移徙", "入宅", "安床", "开仓", "出货财", "开市", "立券", "交易"}, Ji: []string{"动土", "破土", "安葬", "伐木", "架马", "开仓"}},
}

// 根据日支获取宜忌
func GetDayYiJi(dayZhi string) struct {
	Yi []string
	Ji []string
} {
	if yiji, ok := YiJiData[dayZhi]; ok {
		return yiji
	}
	return YiJiData["子"]
}

// 获取冲煞的地支
func GetChongZhi(dayZhi string) string {
	chongMap := map[string]string{
		"子": "午", "丑": "未", "寅": "申", "卯": "酉",
		"辰": "戌", "巳": "亥", "午": "子", "未": "丑",
		"申": "寅", "酉": "卯", "戌": "辰", "亥": "巳",
	}
	if c, ok := chongMap[dayZhi]; ok {
		return c
	}
	return ""
}

// 获取煞的地支
func GetShaZhi(dayZhi string) string {
	shaMap := map[string]string{
		"子": "马", "丑": "羊", "寅": "猴", "卯": "鸡",
		"辰": "龙", "巳": "蛇", "午": "马", "未": "羊",
		"申": "猴", "酉": "鸡", "戌": "狗", "亥": "猪",
	}
	if s, ok := shaMap[dayZhi]; ok {
		return s
	}
	return ""
}

// 获取吉神宜趋
func GetJiShen(dayGan, dayZhi string) []string {
	// 简化版本：根据日支返回对应的吉神
	jiShenMap := map[string][]string{
		"子": {"天德", "月德", "三合", "天喜"},
		"丑": {"天恩", "母仓", "福生", "圣心"},
		"寅": {"天恩", "生气", "阴德", "解神"},
		"卯": {"阴德", "三合", "天喜", "明星"},
		"辰": {"月德", "三合", "天仓", "圣心"},
		"巳": {"天恩", "生气", "阴德", "解神"},
		"午": {"天德", "三合", "天喜", "明星"},
		"未": {"母仓", "三合", "吉期", "圣心"},
		"申": {"天恩", "生气", "三合", "福生"},
		"酉": {"阴德", "三合", "天喜", "明星"},
		"戌": {"月德", "三合", "吉期", "圣心"},
		"亥": {"天恩", "生气", "解神", "福生"},
	}
	if js, ok := jiShenMap[dayZhi]; ok {
		return js
	}
	return []string{"天恩", "天德"}
}

// 获取凶神宜忌
func GetXiongSha(dayGan, dayZhi string) []string {
	xiongShaMap := map[string][]string{
		"子": {"天牢", "玄武", "五虚"},
		"丑": {"墓门", "死符", "五虚"},
		"寅": {"天贼", "五虚", "丧门"},
		"卯": {"天牢", "玄武", "五虚"},
		"辰": {"墓门", "死符", "五虚"},
		"巳": {"天贼", "五虚", "丧门"},
		"午": {"天牢", "玄武", "五虚"},
		"未": {"墓门", "死符", "五虚"},
		"申": {"天贼", "五虚", "丧门"},
		"酉": {"天牢", "玄武", "五虚"},
		"戌": {"墓门", "死符", "五虚"},
		"亥": {"天贼", "五虚", "丧门"},
	}
	if xs, ok := xiongShaMap[dayZhi]; ok {
		return xs
	}
	return []string{"五虚", "天贼"}
}

func GetHuangli(year, month, day int) *Huangli {
	// 使用 tyme4go 库获取准确的农历信息
	solarDay, _ := tyme.SolarDay{}.FromYmd(year, month, day)
	lunarDay := solarDay.GetLunarDay()
	sixtyCycleDay := lunarDay.GetSixtyCycleDay()

	// 获取干支
	yearGan := sixtyCycleDay.GetYear().GetHeavenStem().GetName()
	yearZhi := sixtyCycleDay.GetYear().GetEarthBranch().GetName()
	monthGan := sixtyCycleDay.GetMonth().GetHeavenStem().GetName()
	monthZhi := sixtyCycleDay.GetMonth().GetEarthBranch().GetName()
	dayGan := sixtyCycleDay.GetSixtyCycle().GetHeavenStem().GetName()
	dayZhi := sixtyCycleDay.GetSixtyCycle().GetEarthBranch().GetName()

	// 获取纳音五行
	dayNayin := sixtyCycleDay.GetSixtyCycle().GetSound().GetName()
	yearNayin := sixtyCycleDay.GetYear().GetSound().GetName()
	monthNayin := sixtyCycleDay.GetMonth().GetSound().GetName()

	// 获取冲煞
	chongZhi := GetChongZhi(dayZhi)
	shaZhi := GetShaZhi(dayZhi)

	// 根据日干获取宜忌
	dayYiJi := GetDayYiJi(dayZhi)

	// 获取吉神宜趋和凶神宜忌
	jiShen := GetJiShen(dayGan, dayZhi)
	xiongSha := GetXiongSha(dayGan, dayZhi)

	// 获取值神和白仓
	pengZhu := GetPengZhu(dayZhi)
	baiCang := GetBaCang(dayGan)
	fu := GetFu(dayZhi)
	jieShen := GetJieShen(dayZhi)
	zhangSong := GetZhangSong(dayZhi)
	
	// 获取其他黄历数据
	liuYao := GetLiuYao(dayGan, dayZhi)
	zhiShen := GetZhiShen(dayZhi)
	taiShen := GetTaiShen(dayGan, dayZhi)
	jianChu := GetJianChu(dayZhi)
	xingXiu := GetXingXiu(dayGan, dayZhi)
	xiShen := GetXiShen(dayGan)
	fuShen := GetFuShen(dayGan)
	caiShen := GetCaiShen(dayGan)
	yangGui := GetYangGui(dayGan)
	yinGui := GetYinGui(dayGan)

	// 创建黄历对象
	hl := &Huangli{
		Date:       fmt.Sprintf("%d-%02d-%02d", year, month, day),
		Year:       yearGan + yearZhi,
		Month:      monthGan + monthZhi,
		Day:        dayGan + dayZhi,
		Yi:         dayYiJi.Yi,
		Ji:         dayYiJi.Ji,
		JiShen:     jiShen,
		XiongSha:   xiongSha,
		BaiCang:    baiCang,
		PengZhu:    pengZhu,
		Fu:         fu,
		JieShen:    jieShen,
		Chong:      "冲" + chongZhi,
		Sha:        "煞" + shaZhi,
		ZhangSong:  zhangSong,
		YearWuxing: yearNayin,
		MonthWuxing: monthNayin,
		DayWuxing:  dayNayin,
		LiuYao:     liuYao,
		ZhiShen:    zhiShen,
		TaiShen:    taiShen,
		JianChu:    jianChu,
		XingXiu:    xingXiu,
		XiShen:     xiShen,
		FuShen:     fuShen,
		CaiShen:    caiShen,
		YangGui:    yangGui,
		YinGui:     yinGui,
		Hours:      generateHourYiJi(dayGan, dayZhi),
	}

	return hl
}

// 生成时辰宜忌数据
func generateHourYiJi(dayGan, dayZhi string) []HourYiJi {
	hours := []HourYiJi{}

	timeRanges := []string{
		"23:00 ~ 00:59", "01:00 ~ 02:59", "03:00 ~ 04:59", "05:00 ~ 06:59",
		"07:00 ~ 08:59", "09:00 ~ 10:59", "11:00 ~ 12:59", "13:00 ~ 14:59",
		"15:00 ~ 16:59", "17:00 ~ 18:59", "19:00 ~ 20:59", "21:00 ~ 22:59",
	}

	// 根据日支确定天神顺序的起始位置
	tianShenOrder := []string{
		"青龙", "明堂", "天刑", "朱雀", "金匮", "天德", "白虎", "玉堂", "天牢", "玄武", "司命", "勾陈",
	}
	
	// 根据日支计算起始索引
	startIndex := 0
	switch dayZhi {
	case "子": startIndex = 0
	case "丑": startIndex = 2
	case "寅": startIndex = 4
	case "卯": startIndex = 6
	case "辰": startIndex = 8
	case "巳": startIndex = 10
	case "午": startIndex = 0
	case "未": startIndex = 2
	case "申": startIndex = 4
	case "酉": startIndex = 6
	case "戌": startIndex = 8
	case "亥": startIndex = 10
	}

	// 生成当天的天神顺序
	tianShen := make([]string, 12)
	for i := 0; i < 12; i++ {
		tianShen[i] = tianShenOrder[(startIndex+i)%12]
	}

	// 根据天神确定黑道黄道和吉凶
	heiHuangDao := make([]string, 12)
	jiXiong := make([]string, 12)
	for i, ts := range tianShen {
		switch ts {
		case "青龙", "明堂", "金匮", "天德", "玉堂", "司命":
			heiHuangDao[i] = "黄道"
			jiXiong[i] = "吉"
		default:
			heiHuangDao[i] = "黑道"
			jiXiong[i] = "凶"
		}
	}

	// 根据时辰地支确定宜忌
	timeZhi := []string{"子", "丑", "寅", "卯", "辰", "巳", "午", "未", "申", "酉", "戌", "亥"}
	yiItems := make([][]string, 12)
	jiItems := make([][]string, 12)
	
	for i, tz := range timeZhi {
		if yiJi, ok := YiJiData[tz]; ok {
			yiItems[i] = yiJi.Yi
			jiItems[i] = yiJi.Ji
		} else {
			// fallback to default
			yiItems[i] = []string{"祭祀", "祈福", "求嗣"}
			jiItems[i] = []string{"开市", "交易"}
		}
	}

	for i := 0; i < 12; i++ {
		hour := HourYiJi{
			TimeRange:   timeRanges[i],
			TianShen:    tianShen[i],
			HeiHuangDao: heiHuangDao[i],
			JiXiong:     jiXiong[i],
			Yi:          yiItems[i],
			Ji:          jiItems[i],
		}
		hours = append(hours, hour)
	}

	return hours
}

func GetBaCang(dayGan string) string {
	baCangMap := map[string]string{
		"甲": "仓库", "乙": "花仓", "丙": "光猛", "丁": "星星",
		"戊": "霞", "己": "云", "庚": "路", "辛": "秀",
		"壬": "湖", "癸": "泉",
	}
	if v, ok := baCangMap[dayGan]; ok {
		return v
	}
	return ""
}

func GetPengZhu(dayZhi string) string {
	pengZhuMap := map[string]string{
		"子": "司命", "丑": "勾陈", "寅": "青龙", "卯": "明堂",
		"辰": "天刑", "巳": "朱雀", "午": "金匮", "未": "天德",
		"申": "白虎", "酉": "玉堂", "戌": "天牢", "亥": "玄武",
	}
	if v, ok := pengZhuMap[dayZhi]; ok {
		return v
	}
	return ""
}

func GetJieShen(dayZhi string) string {
	jieShenMap := map[string]string{
		"子": "天喜", "丑": "天厨", "寅": "天福", "卯": "天德",
		"辰": "月德", "巳": "天息", "午": "天恩", "未": "三合",
		"申": "天仓", "酉": "天富", "戌": "天喜", "亥": "天庆",
	}
	if v, ok := jieShenMap[dayZhi]; ok {
		return v
	}
	return ""
}

func GetChongSha(dayZhi, monthZhi string) (chong, sha string) {
	chongMap := map[string]string{
		"子": "午", "丑": "未", "寅": "申", "卯": "酉",
		"辰": "戌", "巳": "亥", "午": "子", "未": "丑",
		"申": "寅", "酉": "卯", "戌": "辰", "亥": "巳",
	}
	shaMap := map[string]string{
		"子": "马", "丑": "羊", "寅": "猴", "卯": "鸡",
		"辰": "龙", "巳": "蛇", "午": "马", "未": "羊",
		"申": "猴", "酉": "鸡", "戌": "狗", "亥": "猪",
	}
	if c, ok := chongMap[dayZhi]; ok {
		chong = c
	}
	if s, ok := shaMap[dayZhi]; ok {
		sha = s
	}
	return
}

func GetZhangSong(dayZhi string) string {
	zhangSongMap := map[string]string{
		"子": "天贼", "丑": "天祝", "寅": "天贼", "卯": "天贼",
		"辰": "天贼", "巳": "天祝", "午": "天贼", "未": "天祝",
		"申": "天贼", "酉": "天祝", "戌": "天贼", "亥": "天祝",
	}
	if v, ok := zhangSongMap[dayZhi]; ok {
		return v
	}
	return ""
}

func GetFu(dayZhi string) string {
	fuMap := map[string]string{
		"子": "天富", "丑": "天恩", "寅": "天福", "卯": "天贵",
		"辰": "天富", "巳": "天恩", "午": "天福", "未": "天贵",
		"申": "天富", "酉": "天恩", "戌": "天福", "亥": "天贵",
	}
	if v, ok := fuMap[dayZhi]; ok {
		return v
	}
	return ""
}

// 获取六曜
func GetLiuYao(dayGan, dayZhi string) string {
	// 简化版本：根据日支返回六曜
	liuYaoMap := map[string]string{
		"子": "先胜", "丑": "友引", "寅": "先负", "卯": "佛灭",
		"辰": "大安", "巳": "赤口", "午": "先胜", "未": "友引",
		"申": "先负", "酉": "佛灭", "戌": "大安", "亥": "赤口",
	}
	if ly, ok := liuYaoMap[dayZhi]; ok {
		return ly
	}
	return "先胜"
}

// 获取值神
func GetZhiShen(dayZhi string) string {
	zhiShenMap := map[string]string{
		"子": "青龙", "丑": "明堂", "寅": "天刑", "卯": "朱雀",
		"辰": "金匮", "巳": "天德", "午": "白虎", "未": "玉堂",
		"申": "天牢", "酉": "玄武", "戌": "司命", "亥": "勾陈",
	}
	if zs, ok := zhiShenMap[dayZhi]; ok {
		return zs
	}
	return "青龙"
}

// 获取胎神占方
func GetTaiShen(dayGan, dayZhi string) string {
	// 简化版本：根据日支返回胎神占方
	taiShenMap := map[string]string{
		"子": "占门床房内南", "丑": "占厨灶炉外正南", "寅": "占门炉外东北", "卯": "占大门外东南",
		"辰": "占房床栖外东南", "巳": "占碓磨床外东南", "午": "占房床碓外正南", "未": "占厨灶厕外西南",
		"申": "占门炉外西南", "酉": "占大门外正南", "戌": "占房床栖外东北", "亥": "占碓磨床外西北",
	}
	if ts, ok := taiShenMap[dayZhi]; ok {
		return ts
	}
	return "占门床房内南"
}

// 获取建除十二神
func GetJianChu(dayZhi string) string {
	jianChuMap := map[string]string{
		"子": "建", "丑": "除", "寅": "满", "卯": "平",
		"辰": "定", "巳": "执", "午": "破", "未": "危",
		"申": "成", "酉": "收", "戌": "开", "亥": "闭",
	}
	if jc, ok := jianChuMap[dayZhi]; ok {
		return jc
	}
	return "建"
}

// 获取二十八星宿
func GetXingXiu(dayGan, dayZhi string) string {
	xingXiu := []string{
		"角", "亢", "氐", "房", "心", "尾", "箕", "斗", "牛", "女", "虚", "危",
		"室", "壁", "奎", "娄", "胃", "昴", "毕", "觜", "参", "井", "鬼", "柳",
		"星", "张", "翼", "轸",
	}
	// 根据日支计算星宿索引
	indexMap := map[string]int{
		"子": 0, "丑": 2, "寅": 4, "卯": 6, "辰": 8, "巳": 10,
		"午": 12, "未": 14, "申": 16, "酉": 18, "戌": 20, "亥": 22,
	}
	if idx, ok := indexMap[dayZhi]; ok {
		return xingXiu[idx%28]
	}
	return "角"
}

// 获取喜神方位
func GetXiShen(dayGan string) string {
	xShenMap := map[string]string{
		"甲": "东北", "乙": "正东", "丙": "正南", "丁": "正南",
		"戊": "东北", "己": "正北", "庚": "西北", "辛": "正西",
		"壬": "正北", "癸": "正南",
	}
	if xs, ok := xShenMap[dayGan]; ok {
		return xs
	}
	return "东北"
}

// 获取福神方位
func GetFuShen(dayGan string) string {
	fuShenMap := map[string]string{
		"甲": "正北", "乙": "东南", "丙": "正东", "丁": "正西",
		"戊": "正北", "己": "东北", "庚": "正南", "辛": "西南",
		"壬": "正东", "癸": "正南",
	}
	if fs, ok := fuShenMap[dayGan]; ok {
		return fs
	}
	return "正北"
}

// 获取财神方位
func GetCaiShen(dayGan string) string {
	caiShenMap := map[string]string{
		"甲": "东北", "乙": "正东", "丙": "正南", "丁": "正南",
		"戊": "东北", "己": "正北", "庚": "西北", "辛": "正西",
		"壬": "正北", "癸": "正南",
	}
	if cs, ok := caiShenMap[dayGan]; ok {
		return cs
	}
	return "正北"
}

// 获取阳贵方位
func GetYangGui(dayGan string) string {
	yangGuiMap := map[string]string{
		"甲": "正北", "乙": "正东", "丙": "正南", "丁": "正西",
		"戊": "正北", "己": "东北", "庚": "正南", "辛": "西南",
		"壬": "正东", "癸": "正南",
	}
	if yg, ok := yangGuiMap[dayGan]; ok {
		return yg
	}
	return "正北"
}

// 获取阴贵方位
func GetYinGui(dayGan string) string {
	yinGuiMap := map[string]string{
		"甲": "西南", "乙": "正北", "丙": "正东", "丁": "正南",
		"戊": "西南", "己": "正西", "庚": "东北", "辛": "正北",
		"壬": "正南", "癸": "正东",
	}
	if yg, ok := yinGuiMap[dayGan]; ok {
		return yg
	}
	return "西南"
}

type JieQi struct {
	Name    string `json:"name"`
	Date    string `json:"date"`
	Time    int    `json:"time"`
}

// GetCurrentJieQi 使用 tyme4go 天文精度计算当前节气
func GetCurrentJieQi(year, month, day int) string {
	solarDay, err := tyme.SolarDay{}.FromYmd(year, month, day)
	if err != nil {
		return ""
	}
	term := solarDay.GetTerm()
	termDay := term.GetSolarDay()
	// 返回距离最近且与当天最接近的节气名
	if termDay.GetYear() == year && termDay.GetMonth() == month && termDay.GetDay() == day {
		return term.GetName()
	}
	// 获取前一个节气
	prevTerm := solarDay.GetTermDay().GetSolarTerm()
	return prevTerm.GetName()
}
