package bazi

// holiday_ny_eve_test.go — 除夕识别（docs/29 B3）
//
// 原实现把除夕硬编码成 {腊月, 30}。但农历腊月是大小月交替，凡逢腊月小月
// （29 天）的年份，除夕当天会整年匹配不上任何节日：GetHoliday 落回
// 「非节假日」、IsWorkDay 落回「按周末判断」，排盘与择日全按普通工作日算。
// 正确规则是「腊月最后一天」，与月大小无关。

import (
	"testing"
)

// 除夕候选日：2023-01-21（腊月初一）到 2023-01-22（正月初一）
// 前者是腊月最后一天 → 除夕；后者是春节。
// 2023 年腊月为小月（29 天），正是原实现漏判的年份。
func TestChuxi_SmallTwelfthMonth(t *testing.T) {
	hs := GetHolidayService()

	tests := []struct {
		name         string
		year         int
		month        int
		day          int
		wantHoliday  string
		wantVacation bool
	}{
		{"小月腊月末日=除夕", 2023, 1, 21, "除夕", true},
		{"次日正月初一=春节", 2023, 1, 22, "春节", true},
		{"小月腊月28日非除夕", 2023, 1, 19, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, err := hs.GetHoliday(tt.year, tt.month, tt.day)
			if err != nil {
				t.Fatalf("GetHoliday(%d-%02d-%02d) 报错: %v", tt.year, tt.month, tt.day, err)
			}
			if h.Name != tt.wantHoliday {
				t.Errorf("Name = %q, want %q", h.Name, tt.wantHoliday)
			}
			if tt.wantHoliday != "" && h.IsVacation != tt.wantVacation {
				t.Errorf("IsVacation = %v, want %v", h.IsVacation, tt.wantVacation)
			}
		})
	}
}

// TestChuxi_DayBeforeSpringFestival 除夕的次日必然是春节
//
// 只测小月会漏掉「判定跟的是固定日号而非腊月最后一天」这类反向回归。
// 这里不硬编码各年除夕的公历日期，只断言这条不变式：只要除夕被认出来，
// 它就必须落在春节前一天。这同时覆盖大小月两种情形。
func TestChuxi_DayBeforeSpringFestival(t *testing.T) {
	hs := GetHolidayService()

	for _, y := range []int{2022, 2023, 2024, 2025, 2026} {
		m, d, ok := findChuxi(hs, y)
		if !ok {
			t.Fatalf("%d 年未找到除夕", y)
		}
		next, err := hs.GetHoliday(y, m, d+1)
		if err != nil {
			// 除夕落在月末（如 1 月 31 日）时次日跨月，顺延到下月 1 日
			next, err = hs.GetHoliday(y, m+1, 1)
		}
		if err != nil {
			t.Fatalf("%d 年除夕次日查询失败: %v", y, err)
		}
		if next.Name != "春节" {
			t.Errorf("%d 年除夕(%d-%02d-%02d)次日应为春节，实际 %q", y, y, m, d, next.Name)
		}
	}
}

// findChuxi 在 1~2 月窗口内定位除夕日
func findChuxi(hs *HolidayService, y int) (month, day int, ok bool) {
	for m := 1; m <= 2; m++ {
		// 上限放到 31：2022 年除夕是 1 月 31 日。非法日期（如 2 月 30 日）
		// 会在 FromYmd 处返回错误，直接跳过即可。
		for d := 1; d <= 31; d++ {
			h, err := hs.GetHoliday(y, m, d)
			if err != nil {
				continue
			}
			if h.Name == "除夕" {
				return m, d, true
			}
		}
	}
	return 0, 0, false
}

// TestChuxi_IsVacation 除夕必须是放假日
//
// 除夕不在 isTraditionalFestivalVacation 的名单里的话，工作日汇总会把
// 除夕算成上班，与现实不符。
func TestChuxi_IsVacation(t *testing.T) {
	hs := GetHolidayService()
	m, d, ok := findChuxi(hs, 2024)
	if !ok {
		t.Fatal("2024 年未找到除夕")
	}
	if hs.IsWorkDay(2024, m, d) {
		t.Errorf("2024-%02d-%02d 是除夕，不应判定为工作日", m, d)
	}
}