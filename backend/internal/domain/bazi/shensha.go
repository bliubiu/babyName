package bazi

import (
	"name/internal/domain/bazi/tyme"
)

// PillarShensha 四柱神煞
type PillarShensha struct {
	YearGods  []string `json:"year_gods"`  // 年神煞
	MonthGods []string `json:"month_gods"` // 月神煞
	DayGods   []string `json:"day_gods"`   // 日神煞
	HourGods  []string `json:"hour_gods"`  // 时神煞
}

// GetDayShensha 获取日神煞列表
// 使用 tyme 库的日神煞查询，以月柱和日柱的六十甲子为参数
func GetDayShensha(monthGanzhi, dayGanzhi string) []string {
	if monthGanzhi == "" || dayGanzhi == "" {
		return nil
	}

	monthSC, err := tyme.SixtyCycle{}.FromName(monthGanzhi)
	if err != nil {
		return nil
	}
	daySC, err := tyme.SixtyCycle{}.FromName(dayGanzhi)
	if err != nil {
		return nil
	}

	gods, err := tyme.God{}.GetDayGods(*monthSC, *daySC)
	if err != nil {
		return nil
	}

	result := make([]string, 0, len(gods))
	for _, g := range gods {
		result = append(result, g.GetName())
	}
	return result
}

// GetPillarShensha 获取四柱神煞
// 目前日神煞使用 tyme 库查询，年月时神煞暂用简化方式（部分常用神煞）
func GetPillarShensha(yearGanzhi, monthGanzhi, dayGanzhi, hourGanzhi string) *PillarShensha {
	if dayGanzhi == "" {
		return nil
	}

	result := &PillarShensha{}

	// 日神煞 - 使用 tyme 库精确查询
	result.DayGods = GetDayShensha(monthGanzhi, dayGanzhi)

	// 常用神煞（基于地支的简化查询）
	// 年/月/时神煞通过简单的地支关系计算
	if yearGanzhi != "" {
		result.YearGods = getSimpleYearShensha(yearGanzhi)
	}
	if monthGanzhi != "" {
		result.MonthGods = getSimpleMonthShensha(monthGanzhi)
	}
	if hourGanzhi != "" {
		result.HourGods = getSimpleHourShensha(hourGanzhi)
	}

	return result
}

// getBranchFromGanzhi 从干支中提取地支名
func getBranchFromGanzhi(ganzhi string) string {
	runes := []rune(ganzhi)
	if len(runes) < 2 {
		return ""
	}
	return string(runes[1])
}

// getSimpleYearShensha 年神煞（基于年支的常用神煞）
func getSimpleYearShensha(yearGanzhi string) []string {
	branch := getBranchFromGanzhi(yearGanzhi)
	if branch == "" {
		return nil
	}

	eb, err := tyme.EarthBranch{}.FromName(branch)
	if err != nil {
		return nil
	}
	_ = eb // 后续可扩展

	var gods []string

	// 基于年支的简化神煞
	shenshaByYear := map[string][]string{
		"子": {"将星", "桃花"},
		"丑": {"华盖", "金舆"},
		"寅": {"驿马", "福星"},
		"卯": {"将星", "桃花"},
		"辰": {"华盖", "天罗"},
		"巳": {"驿马", "文昌"},
		"午": {"将星", "桃花"},
		"未": {"华盖", "金舆"},
		"申": {"驿马", "福星"},
		"酉": {"将星", "桃花"},
		"戌": {"华盖", "地网"},
		"亥": {"驿马", "文昌"},
	}

	if s, ok := shenshaByYear[branch]; ok {
		gods = append(gods, s...)
	}

	return gods
}

// getSimpleMonthShensha 月神煞
func getSimpleMonthShensha(monthGanzhi string) []string {
	branch := getBranchFromGanzhi(monthGanzhi)
	if branch == "" {
		return nil
	}

	shenshaByMonth := map[string][]string{
		"寅": {"天德", "月德"},
		"卯": {"天德", "月德"},
		"辰": {"天德"},
		"巳": {"月德"},
		"午": {"天德"},
		"未": {"月德"},
		"申": {"天德"},
		"酉": {"月德"},
		"戌": {"天德"},
		"亥": {"月德"},
		"子": {"天德"},
		"丑": {"月德"},
	}

	if s, ok := shenshaByMonth[branch]; ok {
		return s
	}
	return nil
}

// getSimpleHourShensha 时神煞
func getSimpleHourShensha(hourGanzhi string) []string {
	branch := getBranchFromGanzhi(hourGanzhi)
	if branch == "" {
		return nil
	}

	shenshaByHour := map[string][]string{
		"子": {"福星"},
		"丑": {"华盖"},
		"寅": {"文昌"},
		"卯": {"福星"},
		"辰": {"华盖"},
		"巳": {"文昌"},
		"午": {"福星"},
		"未": {"华盖"},
		"申": {"文昌"},
		"酉": {"福星"},
		"戌": {"华盖"},
		"亥": {"文昌"},
	}

	if s, ok := shenshaByHour[branch]; ok {
		return s
	}
	return nil
}
