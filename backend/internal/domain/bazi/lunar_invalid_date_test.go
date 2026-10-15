package bazi

import "testing"

// 本文件固化 docs/29 A2 的修复回归：格式合法但不存在的日期
// （如 2024-02-30、2023-04-31）不得让领域层 nil 解引用 panic。
// 此前 GetLunarCalendar/GetHuangli 吞掉 tyme.SolarDay{}.FromYmd 的错误，
// 直接对 nil 调 GetLunarDay()，用户侧表现为 500。

func TestGetLunarCalendarInvalidDateReturnsNil(t *testing.T) {
	invalid := []struct {
		y, m, d int
		reason  string
	}{
		{2024, 2, 30, "2 月无 30 日"},
		{2023, 4, 31, "4 月小无 31 日"},
		{1900, 1, 0, "日为 0"},
		{2024, 13, 1, "月越界"},
	}
	for _, c := range invalid {
		t.Run(c.reason, func(t *testing.T) {
			if got := GetLunarCalendar(c.y, c.m, c.d, 12, 0); got != nil {
				t.Fatalf("GetLunarCalendar(%d-%d-%d) 应返回 nil，got %+v", c.y, c.m, c.d, got)
			}
		})
	}
}

func TestGetHuangliInvalidDateReturnsNil(t *testing.T) {
	if got := GetHuangli(2024, 2, 30); got != nil {
		t.Fatalf("GetHuangli(2024-02-30) 应返回 nil，got %+v", got)
	}
}

func TestGetLunarCalendarValidDateUnaffected(t *testing.T) {
	lc := GetLunarCalendar(2024, 2, 29, 12, 0) // 闰日合法
	if lc == nil {
		t.Fatal("合法闰日 2024-02-29 不应返回 nil")
	}
	if lc.LunarMonthNumber <= 0 || lc.LunarDayNumber <= 0 {
		t.Fatalf("农历换算结果异常: %+v", lc)
	}
}
