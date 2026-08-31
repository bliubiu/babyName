package ziwei

import (
	"fmt"
	"testing"

	"name/internal/domain/bazi/tyme"
)

// TestDebugRawPillars prints raw tyme library outputs
func TestDebugRawPillars(t *testing.T) {
	cases := []struct {
		name             string
		year, month, day int
		hour             int
		expectedMing     string
	}{
		{"case1", 2000, 8, 16, 4, "壬午"},
		{"case5", 1999, 6, 27, 6, "丁卯"},
		{"case7", 1990, 4, 6, 8, "辛酉"},
		{"case8", 2012, 5, 6, 2, "壬寅"},
		{"case9", 1995, 10, 12, 20, "戊子"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			solarDay, _ := tyme.SolarDay{}.FromYmd(tc.year, tc.month, tc.day)
			lunarDayObj := solarDay.GetLunarDay()
			lunarMonthObj := lunarDayObj.GetLunarMonth()
			lunarYearObj := lunarMonthObj.GetLunarYear()

			lunarMonth := lunarMonthObj.GetMonthWithLeap()

			// Current: year from节令
			sixtyCycleDay := lunarDayObj.GetSixtyCycleDay()
			nianZhuObj := sixtyCycleDay.GetYear()
			oldYearGan := nianZhuObj.GetHeavenStem().GetIndex()
			oldYearZhi := nianZhuObj.GetEarthBranch().GetIndex()

			// New: year from正月初一
			newYearSC := lunarYearObj.GetSixtyCycle()
			newYearGan := newYearSC.GetHeavenStem().GetIndex()
			newYearZhi := newYearSC.GetEarthBranch().GetIndex()

			timeIndex := hourToInt(tc.hour)

			fmt.Printf("[%s] solar=%d-%d-%d\n", tc.name, tc.year, tc.month, tc.day)
			fmt.Printf("  lunarMonth=%d timeIdx=%d\n", lunarMonth, timeIndex)
			fmt.Printf("  year(节令): gan=%d(%s) zhi=%d(%s)\n", oldYearGan, TianganNames[oldYearGan], oldYearZhi, DizhiNames[oldYearZhi])
			fmt.Printf("  year(正月): gan=%d(%s) zhi=%d(%s)\n", newYearGan, TianganNames[newYearGan], newYearZhi, DizhiNames[newYearZhi])
			fmt.Printf("  month(节令): %s\n", sixtyCycleDay.GetMonth().GetName())

			// Try all combos of year gan + formula
			fmt.Printf("  --- 命宫 combos ---\n")
			for _, yg := range []struct {
				label string
				ganIdx int
			}{
				{"节令年干", oldYearGan},
				{"正月初一年干", newYearGan},
			} {
				monthIndex := lunarMonth - 1
				if monthIndex < 0 {
					monthIndex = -monthIndex - 1
				}
				soulIndex := fixIndex(monthIndex-timeIndex%12, 12)
				ganIdx := TigerRule[yg.ganIdx] + soulIndex
				zhiIdx := fixIndex(soulIndex+2, 12)
				fmt.Printf("    %s(%d): TigerRule=%d soulIdx=%d → %s\n",
					yg.label, yg.ganIdx, TigerRule[yg.ganIdx], soulIndex,
					getGanZhi(ganIdx, zhiIdx))
			}
			fmt.Printf("  expected: %s\n\n", tc.expectedMing)
		})
	}
}
