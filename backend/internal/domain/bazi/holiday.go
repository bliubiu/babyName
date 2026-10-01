package bazi

import (
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"name/internal/domain/bazi/tyme"
	"name/internal/infrastructure/logger"
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

	// cacheMu 保护按日期缓存（docs/29 B4）。
	// 农历↔公历换算结果是纯函数（同一日期恒等），可安全长期缓存。
	cacheMu      sync.RWMutex
	holidayCache map[string]*Holiday
}

// holidayCacheMax 单日缓存条目上限。
// 日期键空间有界（365 天 × 20 年 ≈ 7300 条），超限后整体重建即可，
// 不需要引入 LRU 这种额外结构。
const holidayCacheMax = 8192

// dateKey 统一日期键格式，避免各处 Sprintf 口径不一致
func dateKey(year, month, day int) string {
	return fmt.Sprintf("%04d-%02d-%02d", year, month, day)
}

// cachedHoliday 读取单日缓存（返回副本，避免调用方改写缓存内容）
func (hs *HolidayService) cachedHoliday(key string) (*Holiday, bool) {
	hs.cacheMu.RLock()
	cached, ok := hs.holidayCache[key]
	hs.cacheMu.RUnlock()
	if !ok {
		return nil, false
	}
	clone := *cached
	return &clone, true
}

// storeHoliday 写入单日缓存
func (hs *HolidayService) storeHoliday(key string, holiday *Holiday) {
	hs.cacheMu.Lock()
	defer hs.cacheMu.Unlock()
	if hs.holidayCache == nil {
		hs.holidayCache = make(map[string]*Holiday)
	}
	if len(hs.holidayCache) >= holidayCacheMax {
		hs.holidayCache = make(map[string]*Holiday)
	}
	snapshot := *holiday
	hs.holidayCache[key] = &snapshot
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
	// 命中缓存直接返回：农历换算是本函数的主要开销（docs/29 B4）
	key := dateKey(year, month, day)
	if cached, ok := hs.cachedHoliday(key); ok {
		return cached, nil
	}

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

	festivalName := hs.getTraditionalFestival(lunarDay)
	if festivalName != "" {
		holiday.Name = festivalName
		holiday.HolidayType = HolidayTypeTraditional
		holiday.IsVacation = hs.isTraditionalFestivalVacation(festivalName)
		holiday.Description = hs.getTraditionalFestivalDesc(festivalName)
		hs.storeHoliday(key, holiday)
		return holiday, nil
	}

	var legalHoliday *tyme.LegalHoliday
	legalHoliday, err = tyme.LegalHoliday{}.FromYmd(year, month, day)
	if err != nil {
		logger.Warn("法定节假日解析失败，按非节假日处理",
			zap.String("date", fmt.Sprintf("%04d-%02d-%02d", year, month, day)), zap.Error(err))
		legalHoliday = nil
	}
	if legalHoliday != nil {
		holiday.Name = legalHoliday.GetName()
		holiday.HolidayType = HolidayTypePublic
		holiday.IsVacation = !legalHoliday.IsWork()
		holiday.IsWorkDay = legalHoliday.IsWork()
		holiday.Description = hs.getPublicHolidayDesc(holiday.Name, holiday.IsVacation)
		holiday.DaysOff = hs.calculateDaysOff(year, month, day)
		hs.storeHoliday(key, holiday)
		return holiday, nil
	}

	if hs.isWeekend(year, month, day) {
		weekday := hs.getWeekday(year, month, day)
		holiday.HolidayType = HolidayTypeWeekend
		holiday.Description = fmt.Sprintf("周末（第%d天）", weekday)
		hs.storeHoliday(key, holiday)
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

	// 单次轮询同时得到天数与名称（docs/29 B4）
	daysUntilHoliday, nextHoliday := hs.findNextHoliday(year, month, day)

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
		"days_until_holiday": daysUntilHoliday,
		"next_holiday":       nextHoliday,
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
	legalHoliday, err := tyme.LegalHoliday{}.FromYmd(year, month, day)
	if err != nil {
		logger.Warn("法定节假日解析失败，按非节假日处理",
			zap.String("date", fmt.Sprintf("%04d-%02d-%02d", year, month, day)), zap.Error(err))
		legalHoliday = nil
	}
	if legalHoliday != nil {
		return !legalHoliday.IsWork()
	}

	solarDay, _ := tyme.SolarDay{}.FromYmd(year, month, day)
	lunar := solarDay.GetLunarDay()

	festivalName := hs.getTraditionalFestival(lunar)
	if festivalName != "" {
		return hs.isTraditionalFestivalVacation(festivalName)
	}

	return hs.isWeekend(year, month, day)
}

func (hs *HolidayService) IsWorkDay(year, month, day int) bool {
	hs.mu.RLock()
	defer hs.mu.RUnlock()

	var legalHoliday *tyme.LegalHoliday
	legalHoliday, err := tyme.LegalHoliday{}.FromYmd(year, month, day)
	if err != nil {
		logger.Warn("法定节假日解析失败，按非节假日处理",
			zap.String("date", fmt.Sprintf("%04d-%02d-%02d", year, month, day)), zap.Error(err))
		legalHoliday = nil
	}
	if legalHoliday != nil {
		return legalHoliday.IsWork()
	}

	return false
}

// getTraditionalFestival 传统节日判定。
//
// 参数是具体的农历日，而非拆开的月/日：除夕判定需要「腊月共几天」，
// 而腊月可能是 29 天也可能是 30 天（docs/29 B3）。原来固定匹配 {12, 30}，
// 在腊月小月的年份里除夕会整年识别不到 —— 该年 12 月 29 日既不是腊八节
// 也不是任何节日，直接落回「非节假日」，排盘结果按普通工作日处理。
func (hs *HolidayService) getTraditionalFestival(lunarDay tyme.LunarDay) string {
	month := lunarDay.GetLunarMonth()
	monthNum := month.GetMonth()
	dayNum := lunarDay.GetDay()
	isLeapMonth := month.IsLeap()

	// 闰月不算传统节日（与原实现一致）
	if isLeapMonth {
		return ""
	}

	// 除夕 = 腊月最后一天，大小月都算。先于下表判定，避免被固定日期表误伤。
	if monthNum == 12 && dayNum == month.GetDayCount() {
		return "除夕"
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
	}

	for name, info := range festivals {
		if info.month == monthNum && info.day == dayNum {
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

// findNextHoliday 从指定日期起向后查找最近的一个节假日（docs/29 B4）
//
// 此前 getDaysUntilNextHoliday 与 getNextHoliday 是两段逐字重复的 365 次
// 逐日轮询，GetHolidayInfo 单次调用要跑满 730 次日期判定，而两者的答案
// 必然来自同一个日期 —— 纯重复计算。
//
// 现合并为一次轮询同时产出「距今天数」与「名称 (日期)」，循环次数减半；
// 叠加 GetHoliday 的按日期缓存后，重复查询退化为纯 map 查找。
//
// 返回值约定与原实现保持一致：一年内无节假日时返回 (-1, "")。
func (hs *HolidayService) findNextHoliday(year, month, day int) (int, string) {
	base := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
	for offset := 1; offset <= 365; offset++ {
		futureDate := base.AddDate(0, 0, offset)

		holiday, err := hs.GetHoliday(futureDate.Year(), int(futureDate.Month()), futureDate.Day())
		if err != nil {
			continue
		}
		if holiday.Name != "" {
			return offset, fmt.Sprintf("%s (%s)", holiday.Name, holiday.Date)
		}
	}
	return -1, ""
}
