package services

import (
	"name/internal/domain/bazi"
	"name/internal/domain/zodiac"
)

// zodiacNameForBazi 统一生肖口径：优先由八字年支推导，分析缺失时才回退公历年。
//
// 生肖以立春为界，必须与年柱一致；公历年取模在 1 月 1 日至立春之间会错一个生肖。
// 该函数供生成、测名等所有响应装配链路复用，避免各写一套取法。
func zodiacNameForBazi(analysis *bazi.BaziAnalysis, finder zodiac.ZodiacFinder, fallbackYear int) string {
	if analysis != nil && analysis.Bazi.YearGanzhi != "" {
		if z := zodiac.GetZodiacByYearGanzhi(analysis.Bazi.YearGanzhi); z != nil {
			return z.Name
		}
	}
	if finder != nil {
		return finder.FindByYear(fallbackYear)
	}
	return ""
}
