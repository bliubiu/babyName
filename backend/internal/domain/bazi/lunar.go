package bazi

import (
	"time"
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
}

func GetLunarCalendar(year, month, day, hour, minute int) *LunarCalendar {
	lc := &LunarCalendar{
		Year:  year,
		Month: month,
		Day:   day,
	}

	lc.YearGan, lc.YearZhi = GetYearGanzhi(year)
	yearNayin := lc.YearGan + lc.YearZhi
	lc.YearNayin = GetNayin(yearNayin)

	lc.MonthGan, lc.MonthZhi = GetMonthGanzhi(year, month)
	monthNayin := lc.MonthGan + lc.MonthZhi
	lc.MonthNayin = GetNayin(monthNayin)

	lc.DayGan, lc.DayZhi = GetDayGanzhi(year, month, day)
	dayNayin := lc.DayGan + lc.DayZhi
	lc.DayNayin = GetNayin(dayNayin)
	lc.Nayin = lc.DayNayin

	lc.HourGan, lc.HourZhi = GetHourGanzhi(lc.DayGan, hour, minute)

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

func GetMonthGanzhi(year, month int) (gan, zhi string) {
	ganIndex := ((year%100)*3/4 + month/2 + 2) % 10
	if month <= 2 {
		ganIndex = (ganIndex + 8) % 10
	}
	zhiIndex := (month + 2) % 12
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

type Huangli struct {
	Date        string   `json:"date"`
	Year        string   `json:"year"`
	Month       string   `json:"month"`
	Day         string   `json:"day"`
	Yi          []string `json:"yi"`
	Ji          []string `json:"ji"`
	JiShen      []string `json:"jiShen"`
	XiongSha    []string `json:"xiongSha"`
	BaiCang     string   `json:"baiCang"`
	PengZhu     string   `json:"pengZhu"`
	Fu          string   `json:"fu"`
	JieShen     string   `json:"jieShen"`
	Chong       string   `json:"chong"`
	Sha         string   `json:"sha"`
	ZhangSong   string   `json:"zhangSong"`
	YearWuxing  string   `json:"yearWuxing"`
	MonthWuxing string   `json:"monthWuxing"`
	DayWuxing   string   `json:"dayWuxing"`
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

func GetHuangli(year, month, day int) *Huangli {
	dayGan, dayZhi := GetDayGanzhi(year, month, day)
	monthGan, monthZhi := GetMonthGanzhi(year, month)

	hl := &Huangli{
		Date:   "",
		Year:   "",
		Month:  "",
		Day:    dayGan + dayZhi,
		Yi:     []string{},
		Ji:     []string{},
		JiShen: []string{},
	}

	if data, ok := YiJiData[dayZhi]; ok {
		hl.Yi = data.Yi
		hl.Ji = data.Ji
	}

	yearGan, yearZhi := GetYearGanzhi(year)
	yearNayin := GetNayin(yearGan + yearZhi)
	monthNayin := GetNayin(monthGan + monthZhi)
	dayNayin := GetNayin(dayGan + dayZhi)

	hl.YearWuxing = GetNayinWuxing(yearNayin)
	hl.MonthWuxing = GetNayinWuxing(monthNayin)
	hl.DayWuxing = GetNayinWuxing(dayNayin)

	hl.BaiCang = GetBaCang(dayGan)
	hl.PengZhu = GetPengZhu(dayZhi)
	hl.JieShen = GetJieShen(dayZhi)
	hl.Chong, hl.Sha = GetChongSha(dayZhi, monthZhi)
	hl.ZhangSong = GetZhangSong(dayZhi)
	hl.Fu = GetFu(dayZhi)

	return hl
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

type JieQi struct {
	Name    string `json:"name"`
	Date    string `json:"date"`
	Time    int    `json:"time"`
}

var JieQiDates = map[string][2]int{
	"小寒": {1, 5}, "大寒": {1, 20}, "立春": {2, 4}, "雨水": {2, 19},
	"惊蛰": {3, 6}, "春分": {3, 21}, "清明": {4, 5}, "谷雨": {4, 20},
	"立夏": {5, 6}, "小满": {5, 21}, "芒种": {6, 6}, "夏至": {6, 21},
	"小暑": {7, 7}, "大暑": {7, 23}, "立秋": {8, 8}, "处暑": {8, 23},
	"白露": {9, 8}, "秋分": {9, 23}, "寒露": {10, 8}, "霜降": {10, 23},
	"立冬": {11, 7}, "小雪": {11, 22}, "大雪": {12, 7}, "冬至": {12, 22},
}

func GetJieQiName(month, day int) string {
	for name, data := range JieQiDates {
		if data[0] == month && data[1] == day {
			return name
		}
	}
	return ""
}

func GetCurrentJieQi(year, month, day int) string {
	monthJieQiMap := map[int][]string{
		1: {"小寒", "大寒"}, 2: {"立春", "雨水"}, 3: {"惊蛰", "春分"},
		4: {"清明", "谷雨"}, 5: {"立夏", "小满"}, 6: {"芒种", "夏至"},
		7: {"小暑", "大暑"}, 8: {"立秋", "处暑"}, 9: {"白露", "秋分"},
		10: {"寒露", "霜降"}, 11: {"立冬", "小雪"}, 12: {"大雪", "冬至"},
	}

	if mjq, ok := monthJieQiMap[month]; ok {
		firstDate := JieQiDates[mjq[0]]
		secondDate := JieQiDates[mjq[1]]

		if day < firstDate[1] {
			if month == 1 {
				return "冬至"
			}
			prevMonth := month - 1
			prevJieQi := monthJieQiMap[prevMonth]
			return prevJieQi[1]
		} else if day < secondDate[1] {
			return mjq[0]
		} else {
			return mjq[1]
		}
	}
	return ""
}
