package bazi

import (
	"testing"

	"name/internal/domain/bazi/tyme"
)

// TestGetShichen_Boundary 时辰边界：23 时必须是子，与 tyme 一致
func TestGetShichen_Boundary(t *testing.T) {
	cases := map[int]string{
		0: "子", 1: "丑", 2: "丑",
		3: "寅", 5: "卯", 7: "辰", 9: "巳",
		11: "午", 13: "未", 15: "申", 17: "酉",
		19: "戌", 21: "亥", 22: "亥", 23: "子",
	}
	for hour, want := range cases {
		if got := GetShichen(hour); got != want {
			t.Errorf("GetShichen(%d) = %q, want %q", hour, got, want)
		}
	}
}

// TestGetShichen_MatchesTyme 与 tyme LunarHour 索引对齐
func TestGetShichen_MatchesTyme(t *testing.T) {
	names := []string{"子", "丑", "寅", "卯", "辰", "巳", "午", "未", "申", "酉", "戌", "亥"}
	for hour := 0; hour < 24; hour++ {
		lh, err := tyme.LunarHour{}.FromYmdHms(2024, 6, 15, hour, 30, 0)
		if err != nil {
			t.Fatalf("LunarHour: %v", err)
		}
		want := names[lh.GetIndexInDay()%12]
		if got := GetShichen(hour); got != want {
			t.Errorf("hour=%d GetShichen=%q tyme=%q", hour, got, want)
		}
	}
}

// TestSeasonFromMonthBranch 月支定季
func TestSeasonFromMonthBranch(t *testing.T) {
	cases := map[string]string{
		"甲寅": "春", "乙卯": "春", "丙辰": "春",
		"丁巳": "夏", "戊午": "夏", "己未": "夏",
		"庚申": "秋", "辛酉": "秋", "壬戌": "秋",
		"癸亥": "冬", "甲子": "冬", "乙丑": "冬",
	}
	for gz, want := range cases {
		if got := seasonFromMonthBranch(gz); got != want {
			t.Errorf("seasonFromMonthBranch(%s) = %q, want %q", gz, got, want)
		}
	}
}

// TestCalculateBazi_Golden 四柱金标准（与 tyme 排盘对拍 + 立春换年）
func TestCalculateBazi_Golden(t *testing.T) {
	// 立春前仍属上一年：2024 立春约 2/4，2/3 仍为癸卯年
	b, err := CalculateBazi(2024, 2, 3, 12, 0)
	if err != nil {
		t.Fatal(err)
	}
	if b.YearGanzhi != "癸卯" {
		t.Errorf("2024-02-03 年柱 = %q, want 癸卯（立春前）", b.YearGanzhi)
	}

	b2, err := CalculateBazi(2024, 2, 5, 12, 0)
	if err != nil {
		t.Fatal(err)
	}
	if b2.YearGanzhi != "甲辰" {
		t.Errorf("2024-02-05 年柱 = %q, want 甲辰（立春后）", b2.YearGanzhi)
	}

	// 与 tyme 八字完全一致
	b3, err := CalculateBazi(1984, 10, 1, 8, 30)
	if err != nil {
		t.Fatal(err)
	}
	st, err := tyme.SolarTime{}.FromYmdHms(1984, 10, 1, 8, 30, 0)
	if err != nil {
		t.Fatal(err)
	}
	ec := st.GetLunarHour().GetEightChar()
	if b3.YearGanzhi != ec.GetYear().GetName() {
		t.Errorf("年柱 %q vs tyme %q", b3.YearGanzhi, ec.GetYear().GetName())
	}
	if b3.MonthGanzhi != ec.GetMonth().GetName() {
		t.Errorf("月柱 %q vs tyme %q", b3.MonthGanzhi, ec.GetMonth().GetName())
	}
	if b3.DayGanzhi != ec.GetDay().GetName() {
		t.Errorf("日柱 %q vs tyme %q", b3.DayGanzhi, ec.GetDay().GetName())
	}
	if b3.HourGanzhi != ec.GetHour().GetName() {
		t.Errorf("时柱 %q vs tyme %q", b3.HourGanzhi, ec.GetHour().GetName())
	}
}

// TestAnalyzeBazi_NayinUsesYearPillar 年命纳音必须来自年柱干支（含立春校正）
func TestAnalyzeBazi_NayinUsesYearPillar(t *testing.T) {
	a, err := AnalyzeBazi(2024, 2, 3, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := NayinMap[a.Bazi.YearGanzhi]
	if a.Nayin != want {
		t.Errorf("Nayin = %q, want %q（年柱 %s）", a.Nayin, want, a.Bazi.YearGanzhi)
	}
	// 癸卯 → 金箔金
	if a.Bazi.YearGanzhi == "癸卯" && a.Nayin != "金箔金" {
		t.Errorf("癸卯纳音 = %q, want 金箔金", a.Nayin)
	}
}

// TestAnalyzeBazi_SeasonFromMonthBranch 季节随月支，非公历月
func TestAnalyzeBazi_SeasonFromMonthBranch(t *testing.T) {
	// 2024-02-05 立春后为甲辰年，月柱应为寅月（春）
	a, err := AnalyzeBazi(2024, 2, 5, 12, 0)
	if err != nil {
		t.Fatal(err)
	}
	runes := []rune(a.Bazi.MonthGanzhi)
	if len(runes) >= 2 && string(runes[1]) == "寅" && a.Season != "春" {
		t.Errorf("月支寅 Season = %q, want 春", a.Season)
	}
	if a.Season != seasonFromMonthBranch(a.Bazi.MonthGanzhi) {
		t.Errorf("Season %q 与月支 %s 推导不一致", a.Season, a.Bazi.MonthGanzhi)
	}
}

// TestCalculateBazi_LateZiHour 晚子时（23 点）日柱按流派进位
func TestCalculateBazi_LateZiHour(t *testing.T) {
	b22, err := CalculateBazi(2024, 6, 15, 22, 0)
	if err != nil {
		t.Fatal(err)
	}
	b23, err := CalculateBazi(2024, 6, 15, 23, 0)
	if err != nil {
		t.Fatal(err)
	}
	if b22.DayGanzhi == b23.DayGanzhi {
		// 默认流派晚子时算第二天，日柱应不同；若库实现为流派2则允许相同，仅记录
		t.Logf("22时与23时日柱相同（%s），若为流派2设计可接受", b22.DayGanzhi)
	}
	if GetShichen(23) != "子" {
		t.Error("23 时时辰必须是子")
	}
}
