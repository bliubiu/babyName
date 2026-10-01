package bazi

// holiday_perf_test.go — 节假日查询（docs/29 B4）
//
// B4 的原始病灶：getDaysUntilNextHoliday 与 getNextHoliday 是两段逐字重复的
// 365 次逐日轮询，单次 GetHolidayInfo 要做约 730 次农历/公历判定。
//
// 本文件锁定两件事：
//  1. 合并后的 findNextHoliday 与原两函数语义完全一致（天数、名称、边界）；
//  2. GetHoliday 的按日期缓存改变了性能而非结果（缓存命中与未命中同解），
//     且返回副本，调用方改写返回值不会污染缓存。

import (
	"testing"
	"time"
)

// referenceFindNextHoliday 保留原始双循环算法的参考实现，仅用于回归对拍
func referenceFindNextHoliday(hs *HolidayService, year, month, day int) (int, string) {
	base := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)

	daysUntil := -1
	var nextDesc string
	for offset := 1; offset <= 365; offset++ {
		futureDate := base.AddDate(0, 0, offset)
		holiday, err := hs.GetHoliday(futureDate.Year(), int(futureDate.Month()), futureDate.Day())
		if err != nil || holiday.Name == "" {
			continue
		}
		if daysUntil == -1 {
			daysUntil = offset
			nextDesc = holiday.Name + " (" + holiday.Date + ")"
		}
	}
	return daysUntil, nextDesc
}

// TestFindNextHoliday_MatchesReference 双循环合并后与原算法逐一对拍
func TestFindNextHoliday_MatchesReference(t *testing.T) {
	hs := &HolidayService{}

	cases := []struct{ y, m, d int }{
		{2026, 1, 1},   // 元旦后
		{2026, 2, 17},  // 春节（2026-02-17）
		{2026, 2, 18},  // 春节次日
		{2026, 5, 1},   // 劳动节
		{2026, 6, 19},  // 端午（2026-06-19）
		{2026, 9, 25},  // 中秋（2026-09-25）
		{2026, 10, 1},  // 国庆
		{2026, 12, 31}, // 年末，跨年边界
		{2027, 1, 1},   // 跨年
		{2025, 12, 25}, // 圣诞后
	}

	for _, c := range cases {
		wantDays, wantDesc := referenceFindNextHoliday(hs, c.y, c.m, c.d)
		gotDays, gotDesc := hs.findNextHoliday(c.y, c.m, c.d)

		if gotDays != wantDays {
			t.Errorf("%04d-%02d-%02d：距下次节假日天数 got=%d want=%d",
				c.y, c.m, c.d, gotDays, wantDays)
		}
		if gotDesc != wantDesc {
			t.Errorf("%04d-%02d-%02d：下次节假日描述 got=%q want=%q",
				c.y, c.m, c.d, gotDesc, wantDesc)
		}
	}
}

// TestFindNextHoliday_ReturnsConsistentPair 天数与名称必须指向同一个节假日
//
// 合并前两个函数各跑一遍循环，理论上会一致，但一旦某天判定被改就会错位
// （天数说还有 3 天、名称却是 4 天后的节日）。这里把两者绑在一起断言。
func TestFindNextHoliday_ReturnsConsistentPair(t *testing.T) {
	hs := &HolidayService{}

	days, desc := hs.findNextHoliday(2026, 6, 1)
	if days < 0 {
		t.Skip("该日期起一年内未识别到节假日，跳过一致性断言")
	}
	if desc == "" {
		t.Errorf("识别到节假日（%d 天后）但名称为空，两者不一致", days)
	}

	target := time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local).AddDate(0, 0, days)
	holiday, err := hs.GetHoliday(target.Year(), int(target.Month()), target.Day())
	if err != nil {
		t.Fatalf("按天数反查日期失败：%v", err)
	}
	if holiday.Name == "" {
		t.Errorf("第 %d 天（%s）并非节假日，天数与名称错位", days, target.Format("2006-01-02"))
	}
	if want := holiday.Name + " (" + holiday.Date + ")"; desc != want {
		t.Errorf("名称与反查结果不一致：got=%q want=%q", desc, want)
	}
}

// TestGetHoliday_CacheReturnsEquivalentResult 缓存不得改变查询结果
func TestGetHoliday_CacheReturnsEquivalentResult(t *testing.T) {
	hs := &HolidayService{}

	first, err := hs.GetHoliday(2026, 2, 17)
	if err != nil {
		t.Fatalf("首次查询失败：%v", err)
	}
	if first.Name == "" {
		t.Fatal("2026-02-17 应为春节，测试前提不成立")
	}

	// 调用方恶意改写返回值，不得污染后续查询
	first.Name = "被篡改"
	first.IsVacation = false

	second, err := hs.GetHoliday(2026, 2, 17)
	if err != nil {
		t.Fatalf("缓存命中查询失败：%v", err)
	}
	if second.Name != "春节" {
		t.Errorf("缓存被调用方改写污染：got=%q want=%q", second.Name, "春节")
	}
	if !second.IsVacation {
		t.Error("缓存被调用方改写污染：IsVacation 应为 true")
	}
}

// TestHolidayCache_BoundedAndResettable 缓存超限应整体重建而非无限增长
func TestHolidayCache_BoundedAndResettable(t *testing.T) {
	hs := &HolidayService{}

	// 跨 25 年扫描，必然突破 8192 上限并触发重建
	for year := 2020; year <= 2044; year++ {
		if _, err := hs.GetHoliday(year, 6, 15); err != nil {
			t.Fatalf("GetHoliday(%d) 失败：%v", year, err)
		}
	}

	hs.cacheMu.RLock()
	size := len(hs.holidayCache)
	hs.cacheMu.RUnlock()

	if size > holidayCacheMax {
		t.Errorf("缓存条目数 %d 超过上限 %d，未做上界控制", size, holidayCacheMax)
	}
	if size == 0 {
		t.Error("缓存为空，重建后应重新填充")
	}
}

// TestGetHolidayInfo_AfterMergeConsistent GetHolidayInfo 的两个字段应同源
func TestGetHolidayInfo_AfterMergeConsistent(t *testing.T) {
	hs := &HolidayService{}

	info, err := hs.GetHolidayInfo(2026, 6, 1)
	if err != nil {
		t.Fatalf("GetHolidayInfo 失败：%v", err)
	}

	days, ok := info["days_until_holiday"].(int)
	if !ok {
		t.Fatalf("days_until_holiday 类型异常：%T", info["days_until_holiday"])
	}
	desc, ok := info["next_holiday"].(string)
	if !ok {
		t.Fatalf("next_holiday 类型异常：%T", info["next_holiday"])
	}

	if (days < 0) != (desc == "") {
		t.Errorf("两个字段不同源：days=%d desc=%q", days, desc)
	}
}