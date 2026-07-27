package bazi

import (
	"fmt"
	"sync"
	"time"

	"name/internal/domain/bazi/tyme"
)

type HolidayType string

const (
	HolidayTypeTraditional HolidayType = "traditional"
	HolidayTypePublic     HolidayType = "public"
	HolidayTypeAdjusted   HolidayType = "adjusted"
	HolidayTypeWeekend    HolidayType = "weekend"
)

type Holiday struct {
	Name        string      `json:"name"`
	Date        string      `json:"date"`
	HolidayType HolidayType `json:"holiday_type"`
	IsVacation  bool        `json:"is_vacation"`
	IsWorkDay   bool        `json:"is_work_day"`
	Description string      `json:"description"`
	DaysOff     int         `json:"days_off"`
}

type HolidayService struct {
	mu sync.RWMutex
}

var (
	holidayService *HolidayService
	once           sync.Once
)

func GetHolidayService() *HolidayService {
	once.Do(func() {
		holidayService = &HolidayService{}
	})
	return holidayService
}

func (hs *HolidayService) GetHoliday(year, month, day int) (*Holiday, error) {
	hs.mu.RLock()
	defer hs.mu.RUnlock()

	holiday := &Holiday{
		Date: fmt.Sprintf("%04d-%02d-%02d", year, month, day),
	}

	solarDay, err := tyme.SolarDay{}.FromYmd(year, month, day)
	if err != nil {
		return nil, err
	}

	lunarDay := solarDay.GetLunarDay()
	lunarMonth := lunarDay.GetLunarMonth()
	lunarDayNum := lunarDay.GetDay()
	isLeapMonth := lunarMonth.IsLeap()

	festivalName := hs.getTraditionalFestival(lunarMonth.GetMonth(), lunarDayNum, isLeapMonth, year)
	if festivalName != "" {
		holiday.Name = festivalName
		holiday.HolidayType = HolidayTypeTraditional
		holiday.IsVacation = hs.isTraditionalFestivalVacation(festivalName)
		holiday.Description = hs.getTraditionalFestivalDesc(festivalName)
		return holiday, nil
	}

	var legalHoliday *tyme.LegalHoliday
	legalHoliday, _ = tyme.LegalHoliday{}.FromYmd(year, month, day)
	if legalHoliday != nil {
		holiday.Name = legalHoliday.GetName()
		holiday.HolidayType = HolidayTypePublic
		holiday.IsVacation = !legalHoliday.IsWork()
		holiday.IsWorkDay = legalHoliday.IsWork()
		holiday.Description = hs.getPublicHolidayDesc(holiday.Name, holiday.IsVacation)
		holiday.DaysOff = hs.calculateDaysOff(year, month, day)
		return holiday, nil
	}

	if hs.isWeekend(year, month, day) {
		weekday := hs.getWeekday(year, month, day)
		holiday.HolidayType = HolidayTypeWeekend
		holiday.Description = fmt.Sprintf("周末（第%d天）", weekday)
		return holiday, nil
	}

	return holiday, nil
}

func (hs *HolidayService) GetHolidayInfo(year, month, day int) (map[string]interface{}, error) {
	holiday, err := hs.GetHoliday(year, month, day)
	if err != nil {
		return nil, err
	}

	solarDay, _ := tyme.SolarDay{}.FromYmd(year, month, day)
	lunarDay := solarDay.GetLunarDay()
	lunarMonth := lunarDay.GetLunarMonth()

	result := map[string]interface{}{
		"date":               holiday.Date,
		"is_holiday":         holiday.Name != "",
		"is_vacation":        holiday.IsVacation,
		"is_work_day":        holiday.IsWorkDay,
		"holiday_name":       holiday.Name,
		"holiday_type":       holiday.HolidayType,
		"description":        holiday.Description,
		"lunar_date":         fmt.Sprintf("农历%s月%s", hs.getLunarMonthName(lunarMonth.GetMonth()), hs.getLunarDayName(lunarDay.GetDay())),
		"lunar_festival":     hs.getLunarFestival(lunarMonth.GetMonth(), lunarDay.GetDay()),
		"solar_term":         hs.getSolarTerm(year, month, day),
		"days_until_holiday": hs.getDaysUntilNextHoliday(year, month, day),
		"next_holiday":       hs.getNextHoliday(year, month, day),
	}

	return result, nil
}

func (hs *HolidayService) GetMonthHolidays(year, month int) ([]*Holiday, error) {
	hs.mu.RLock()
	defer hs.mu.RUnlock()

	var holidays []*Holiday
	daysInMonth := hs.getDaysInMonth(year, month)

	for day := 1; day <= daysInMonth; day++ {
		holiday, err := hs.GetHoliday(year, month, day)
		if err != nil {
			continue
		}
		if holiday.Name != "" || holiday.IsVacation {
			holidays = append(holidays, holiday)
		}
	}

	return holidays, nil
}

func (hs *HolidayService) GetYearHolidays(year int) ([]*Holiday, error) {
	hs.mu.RLock()
	defer hs.mu.RUnlock()

	var holidays []*Holiday

	for month := 1; month <= 12; month++ {
		monthHolidays, err := hs.GetMonthHolidays(year, month)
		if err != nil {
			continue
		}
		holidays = append(holidays, monthHolidays...)
	}

	return holidays, nil
}

func (hs *HolidayService) IsVacationDay(year, month, day int) bool {
	hs.mu.RLock()
	defer hs.mu.RUnlock()

	var legalHoliday *tyme.LegalHoliday
	legalHoliday, _ = tyme.LegalHoliday{}.FromYmd(year, month, day)
	if legalHoliday != nil {
		return !legalHoliday.IsWork()
	}

	lunarDay, _ := tyme.SolarDay{}.FromYmd(year, month, day)
	lunar := lunarDay.GetLunarDay()
	monthNum := lunar.GetLunarMonth().GetMonth()
	dayNum := lunar.GetDay()
	isLeapMonth := lunar.GetLunarMonth().IsLeap()

	festivalName := hs.getTraditionalFestival(monthNum, dayNum, isLeapMonth, year)
	if festivalName != "" {
		return hs.isTraditionalFestivalVacation(festivalName)
	}

	return hs.isWeekend(year, month, day)
}

func (hs *HolidayService) IsWorkDay(year, month, day int) bool {
	hs.mu.RLock()
	defer hs.mu.RUnlock()

	var legalHoliday *tyme.LegalHoliday
	legalHoliday, _ = tyme.LegalHoliday{}.FromYmd(year, month, day)
	if legalHoliday != nil {
		return legalHoliday.IsWork()
	}

	return false
}

func (hs *HolidayService) getTraditionalFestival(month, day int, isLeapMonth bool, year int) string {
	if isLeapMonth {
		return ""
	}

	festivals := map[string]struct {
		month int
		day   int
	}{
		"春节":   {1, 1},
		"元宵节": {1, 15},
		"龙头节": {2, 2},
		"上巳节": {3, 3},
		"端午节": {5, 5},
		"七夕节": {7, 7},
		"中元节": {7, 15},
		"中秋节": {8, 15},
		"重阳节": {9, 9},
		"腊八节": {12, 8},
		"除夕":   {12, 30},
	}

	for name, info := range festivals {
		if info.month == month && info.day == day {
			return name
		}
	}

	return ""
}

func (hs *HolidayService) isTraditionalFestivalVacation(name string) bool {
	vacationFestivals := map[string]bool{
		"春节":   true,
		"清明节":  true,
		"端午节":  true,
		"中秋节":  true,
		"国庆节":  true,
		"元旦":    true,
		"劳动节":  true,
		"除夕":   true,
	}
	return vacationFestivals[name]
}

func (hs *HolidayService) getTraditionalFestivalDesc(name string) string {
	descriptions := map[string]string{
		"春节":   "春节期间放假7天，是中华民族最重要的传统节日",
		"元宵节": "元宵节是春节之后的第一个重要节日，有赏花灯、吃汤圆的习俗",
		"龙头节": "龙抬头节，象征春回大地，万物复苏",
		"上巳节": "上巳节是传统的踏青节",
		"清明节": "清明时节，慎终追远，缅怀先人",
		"端午节": "端午节纪念屈原，有赛龙舟、吃粽子的习俗",
		"七夕节": "七夕节又称中国情人节，牛郎织女相会的日子",
		"中元节": "中元节是传统的祭祀节日",
		"中秋节": "中秋节是团圆节，有赏月、吃月饼的习俗",
		"重阳节": "重阳节是敬老节，有登高、赏菊的习俗",
		"腊八节": "腊八节有喝腊八粥的习俗",
		"除夕":   "除夕是农历年的最后一天，阖家团圆吃年夜饭",
	}

	if desc, ok := descriptions[name]; ok {
		return desc
	}
	return "传统节日"
}

func (hs *HolidayService) getPublicHolidayDesc(name string, isVacation bool) string {
	descriptions := map[string]string{
		"元旦":      "元旦是新年的第一天",
		"春节":      "春节是中华民族最隆重的传统节日",
		"清明节":    "清明节是祭祖和扫墓的节日",
		"劳动节":    "劳动节是全世界劳动者的节日",
		"端午节":    "端午节是纪念屈原的传统节日",
		"中秋节":    "中秋节是团圆赏月的好时节",
		"国庆节":    "国庆节庆祝中华人民共和国成立",
		"国庆中秋":  "国庆节与中秋节假期安排",
		"抗战胜利日": "纪念抗日战争胜利",
	}

	if desc, ok := descriptions[name]; ok {
		if isVacation {
			return fmt.Sprintf("%s（放假）", desc)
		}
		return fmt.Sprintf("%s（调休上班）", desc)
	}
	return "法定节假日"
}

func (hs *HolidayService) calculateDaysOff(year, month, day int) int {
	count := 1

	for d := day - 1; d >= 1; d-- {
		if hs.IsVacationDay(year, month, d) {
			count++
		} else {
			break
		}
	}

	for d := day + 1; d <= hs.getDaysInMonth(year, month); d++ {
		if hs.IsVacationDay(year, month, d) {
			count++
		} else {
			break
		}
	}

	return count
}

func (hs *HolidayService) isWeekend(year, month, day int) bool {
	date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
	weekday := date.Weekday()
	return weekday == time.Saturday || weekday == time.Sunday
}

func (hs *HolidayService) getWeekday(year, month, day int) int {
	date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
	return int(date.Weekday())
}

func (hs *HolidayService) getDaysInMonth(year, month int) int {
	date := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	return date.AddDate(0, 1, -1).Day()
}

func (hs *HolidayService) getLunarMonthName(month int) string {
	names := []string{"正", "二", "三", "四", "五", "六", "七", "八", "九", "十", "冬", "腊"}
	if month >= 1 && month <= 12 {
		return names[month-1]
	}
	return ""
}

func (hs *HolidayService) getLunarDayName(day int) string {
	names := []string{
		"初一", "初二", "初三", "初四", "初五", "初六", "初七", "初八", "初九", "初十",
		"十一", "十二", "十三", "十四", "十五", "十六", "十七", "十八", "十九", "二十",
		"廿一", "廿二", "廿三", "廿四", "廿五", "廿六", "廿七", "廿八", "廿九", "三十",
	}
	if day >= 1 && day <= 30 {
		return names[day-1]
	}
	return ""
}

func (hs *HolidayService) getLunarFestival(month, day int) string {
	festivals := map[string]struct{ month, day int }{
		"春节":   {1, 1},
		"元宵节": {1, 15},
		"端午节": {5, 5},
		"七夕节": {7, 7},
		"中元节": {7, 15},
		"中秋节": {8, 15},
		"重阳节": {9, 9},
		"腊八节": {12, 8},
	}

	for name, info := range festivals {
		if info.month == month && info.day == day {
			return name
		}
	}
	return ""
}

func (hs *HolidayService) getSolarTerm(year, month, day int) string {
	solarTime, err := tyme.SolarTime{}.FromYmdHms(year, month, day, 0, 0, 0)
	if err != nil {
		return ""
	}
	return solarTime.GetTerm().GetName()
}

func (hs *HolidayService) getDaysUntilNextHoliday(year, month, day int) int {
	for offset := 1; offset <= 365; offset++ {
		futureDate := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local).AddDate(0, 0, offset)

		if holiday, err := hs.GetHoliday(futureDate.Year(), int(futureDate.Month()), futureDate.Day()); err == nil {
			if holiday.Name != "" {
				return offset
			}
		}
	}
	return -1
}

func (hs *HolidayService) getNextHoliday(year, month, day int) string {
	for offset := 1; offset <= 365; offset++ {
		futureDate := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local).AddDate(0, 0, offset)

		if holiday, err := hs.GetHoliday(futureDate.Year(), int(futureDate.Month()), futureDate.Day()); err == nil {
			if holiday.Name != "" {
				return fmt.Sprintf("%s (%s)", holiday.Name, holiday.Date)
			}
		}
	}
	return ""
}
