package bazi

import (
	"testing"
	"time"
)

// 扫描指定年份全部公历日，返回满足 predicate 的 (month,day) 列表
func scanYear(year int, predicate func(m, d int) bool) [][2]int {
	var hits [][2]int
	for m := 1; m <= 12; m++ {
		days := time.Date(year, time.Month(m)+1, 0, 0, 0, 0, 0, time.UTC).Day()
		for d := 1; d <= days; d++ {
			if predicate(m, d) {
				hits = append(hits, [2]int{m, d})
			}
		}
	}
	return hits
}

// 每年恰好 24 个节气日；且 GetSolarTermExact 仅在节气当日返回非空，非节气日必须为空。
func TestGetSolarTermExact_PerYearCount(t *testing.T) {
	const year = 2026
	termDays := scanYear(year, func(m, d int) bool {
		return GetSolarTermExact(year, m, d) != ""
	})
	if len(termDays) != 24 {
		t.Fatalf("2026 年节气日数量应为 24，实际 %d: %v", len(termDays), termDays)
	}

	// 校验每个返回的节气名都在标准 24 节气表中
	valid := map[string]bool{}
	for _, name := range []string{"冬至", "小寒", "大寒", "立春", "雨水", "惊蛰", "春分", "清明", "谷雨", "立夏", "小满", "芒种", "夏至", "小暑", "大暑", "立秋", "处暑", "白露", "秋分", "寒露", "霜降", "立冬", "小雪", "大雪"} {
		valid[name] = true
	}
	for _, md := range termDays {
		name := GetSolarTermExact(year, md[0], md[1])
		if !valid[name] {
			t.Errorf("未知节气名 %q @ %d-%02d-%02d", name, year, md[0], md[1])
		}
	}
}

// 非节气日必须返回空串（对照 GetCurrentJieQi 在非节气日会回退到上一个节气，本函数不应如此）。
func TestGetSolarTermExact_NonTermDayEmpty(t *testing.T) {
	// 2026-10-20 介于寒露与霜降之间，并非节气日
	if got := GetSolarTermExact(2026, 10, 20); got != "" {
		t.Errorf("非节气日不应返回节气名，实际返回 %q", got)
	}
}

// 重阳节（农历九月初九）在一年内应恰好出现一次，且 GetSolarTermExact 与之无关。
func TestGetLunarFestivalName_Chongyang(t *testing.T) {
	const year = 2026
	fest := scanYear(year, func(m, d int) bool {
		return GetLunarFestivalName(year, m, d) == "重阳节"
	})
	if len(fest) != 1 {
		t.Fatalf("重阳节在 %d 年应恰好出现 1 次，实际 %d 次: %v", year, len(fest), fest)
	}
}

// 春节（农历正月初一）在一年内应恰好出现一次。
func TestGetLunarFestivalName_Chunjie(t *testing.T) {
	const year = 2026
	fest := scanYear(year, func(m, d int) bool {
		return GetLunarFestivalName(year, m, d) == "春节"
	})
	if len(fest) != 1 {
		t.Fatalf("春节在 %d 年应恰好出现 1 次，实际 %d 次: %v", year, len(fest), fest)
	}
}

// GetLunarMonthCalendar 返回当月全部日期，且春节所在日标注正确。
func TestGetLunarMonthCalendar_ChunjieAnnotated(t *testing.T) {
	const year, month = 2026, 2
	cal := GetLunarMonthCalendar(year, month)
	if cal == nil {
		t.Fatal("GetLunarMonthCalendar 不应返回 nil")
	}
	if len(cal) != 28 {
		t.Fatalf("2026-02 应有 28 天，实际 %d", len(cal))
	}
	// 找到春节那一天的标注
	found := false
	for _, d := range cal {
		if d.LunarFestival == "春节" {
			found = true
			if d.SolarTerm != "" {
				t.Errorf("春节日不应同时是节气日，solarTerm=%q", d.SolarTerm)
			}
		}
	}
	if !found {
		t.Errorf("2026-02 的逐日标注中未找到春节")
	}
}

// 非法月份应返回 nil。
func TestGetLunarMonthCalendar_InvalidMonth(t *testing.T) {
	if GetLunarMonthCalendar(2026, 0) != nil {
		t.Error("月份为 0 应返回 nil")
	}
	if GetLunarMonthCalendar(2026, 13) != nil {
		t.Error("月份为 13 应返回 nil")
	}
}
