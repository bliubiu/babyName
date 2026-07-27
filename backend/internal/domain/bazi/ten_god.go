package bazi

import (
	"name/internal/domain/bazi/tyme"
)

// TenGodResult 十神结果（四柱天干）
type TenGodResult struct {
	Year  string `json:"year"`  // 年干十神
	Month string `json:"month"` // 月干十神
	Day   string `json:"day"`   // 日干十神（自身）
	Hour  string `json:"hour"`  // 时干十神
}

// HideHeavenGod 地支藏干十神
type HideHeavenGod struct {
	Stem     string `json:"stem"`      // 藏干天干名
	TenGod   string `json:"ten_god"`   // 十神名
	HideType string `json:"hide_type"` // 本气/中气/余气
}

// PillarTenGods 完整十神（含地支藏干）
type PillarTenGods struct {
	YearStem   string           `json:"year_stem"`   // 年干
	MonthStem  string           `json:"month_stem"`  // 月干
	DayStem    string           `json:"day_stem"`    // 日干
	HourStem   string           `json:"hour_stem"`   // 时干
	YearGod    string           `json:"year_god"`    // 年干十神
	MonthGod   string           `json:"month_god"`   // 月干十神
	DayGod     string           `json:"day_god"`     // 日干十神
	HourGod    string           `json:"hour_god"`    // 时干十神
	HideHeaven []HideHeavenGod  `json:"hide_heaven"` // 地支藏干十神（四柱所有藏干）
}

// GetTenGod 计算一个天干相对于日干的十神
// dayStem: 日干名 (e.g. "戊")
// targetStem: 要计算的天干 (e.g. "甲")
func GetTenGod(dayStem, targetStem string) string {
	dayHS, err := tyme.HeavenStem{}.FromName(dayStem)
	if err != nil {
		return ""
	}
	targetHS, err := tyme.HeavenStem{}.FromName(targetStem)
	if err != nil {
		return ""
	}
	return dayHS.GetTenStar(*targetHS).GetName()
}

// CalculateTenGods 计算四柱天干的十神
func CalculateTenGods(dayStem, yearStem, monthStem, hourStem string) *TenGodResult {
	if dayStem == "" {
		return nil
	}
	return &TenGodResult{
		Year:  GetTenGod(dayStem, yearStem),
		Month: GetTenGod(dayStem, monthStem),
		Day:   GetTenGod(dayStem, dayStem),
		Hour:  GetTenGod(dayStem, hourStem),
	}
}

// CalculatePillarTenGods 计算四柱天干+地支藏干的完整十神
func CalculatePillarTenGods(bazi *Bazi, dayStem string) *PillarTenGods {
	if bazi == nil || dayStem == "" {
		return nil
	}

	// 从各柱干支提取天干（第一个字符）
	yearG := string([]rune(bazi.YearGanzhi)[0])
	monthG := string([]rune(bazi.MonthGanzhi)[0])
	dayG := string([]rune(bazi.DayGanzhi)[0])
	hourG := string([]rune(bazi.HourGanzhi)[0])

	result := &PillarTenGods{
		YearStem:  yearG,
		MonthStem: monthG,
		DayStem:   dayG,
		HourStem:  hourG,
		YearGod:   GetTenGod(dayStem, yearG),
		MonthGod:  GetTenGod(dayStem, monthG),
		DayGod:    GetTenGod(dayStem, dayG),
		HourGod:   GetTenGod(dayStem, hourG),
	}

	// 解析地支藏干十神
	allGanzhi := []struct {
		name string
		zhi  string
	}{
		{"年支", string([]rune(bazi.YearGanzhi)[1])},
		{"月支", string([]rune(bazi.MonthGanzhi)[1])},
		{"日支", string([]rune(bazi.DayGanzhi)[1])},
		{"时支", string([]rune(bazi.HourGanzhi)[1])},
	}

	for _, gz := range allGanzhi {
		eb, err := tyme.EarthBranch{}.FromName(gz.zhi)
		if err != nil {
			continue
		}
		hideStems := eb.GetHideHeavenStems()
		for _, hs := range hideStems {
			stemName := hs.GetHeavenStem().GetName()
			tenGod := GetTenGod(dayStem, stemName)
			hideType := hs.GetType().GetName()
			result.HideHeaven = append(result.HideHeaven, HideHeavenGod{
				Stem:     stemName,
				TenGod:   tenGod,
				HideType: hideType,
			})
		}
	}

	return result
}

// TenGodMeaning 十神含义速查
var TenGodMeaning = map[string]string{
	"比肩": "同辈竞争，自我意识强，独立自主",
	"劫财": "兄弟姐妹，朋友相助，慷慨大方",
	"食神": "才华福气，口福享受，温和善良",
	"伤官": "聪明伶俐，锋芒外露，艺术天赋",
	"偏财": "意外之财，投资眼光，慷慨大方",
	"正财": "稳定收入，踏实理财，勤俭持家",
	"七杀": "魄力果断，挑战压力，领导才能",
	"正官": "贵气权威，正直守规，有管理才能",
	"偏印": "独特思维，玄学灵感，特殊才华",
	"正印": "贵人相助，学业有成，福泽深厚",
}
