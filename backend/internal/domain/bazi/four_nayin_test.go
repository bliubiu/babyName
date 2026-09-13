package bazi

import "testing"

func TestBuildFourPillarNayin(t *testing.T) {
	b := &Bazi{
		YearGanzhi:  "甲子",
		MonthGanzhi: "丙寅",
		DayGanzhi:   "戊辰",
		HourGanzhi:  "庚午",
	}
	f := BuildFourPillarNayin(b)
	if f == nil {
		t.Fatal("FourNayin 为 nil")
	}
	cases := []struct {
		info NayinInfo
		want string
	}{
		{f.Year, "海中金"},
		{f.Month, "炉中火"},
		{f.Day, "大林木"},
		{f.Hour, "路旁土"},
	}
	for i, c := range cases {
		if c.info.Nayin != c.want {
			t.Errorf("第%d柱纳音 = %q, want %q", i+1, c.info.Nayin, c.want)
		}
		if c.info.Nayin == "" || c.info.Wuxing == "" || c.info.Ganzhi == "" {
			t.Errorf("第%d柱字段不完整: %+v", i+1, c.info)
		}
	}
	// 五行映射
	if f.Year.Wuxing != "金" || f.Month.Wuxing != "火" {
		t.Errorf("纳音五行: 年=%s 月=%s", f.Year.Wuxing, f.Month.Wuxing)
	}
}

func TestAnalyzeBazi_FourNayin(t *testing.T) {
	a, err := AnalyzeBazi(1984, 10, 1, 8, 30)
	if err != nil {
		t.Fatal(err)
	}
	if a.FourNayin == nil {
		t.Fatal("AnalyzeBazi 未填充 FourNayin")
	}
	// 年命纳音与四柱·年一致
	if a.Nayin != a.FourNayin.Year.Nayin {
		t.Errorf("Nayin=%q 与 four_nayin.year=%q 不一致", a.Nayin, a.FourNayin.Year.Nayin)
	}
	// 各柱干支与纳音表一致
	if got := NayinMap[a.Bazi.YearGanzhi]; got != a.FourNayin.Year.Nayin {
		t.Errorf("年柱 %s 纳音 %q 表内 %q", a.Bazi.YearGanzhi, a.FourNayin.Year.Nayin, got)
	}
	if got := NayinMap[a.Bazi.MonthGanzhi]; got != a.FourNayin.Month.Nayin {
		t.Errorf("月柱 %s 纳音不匹配", a.Bazi.MonthGanzhi)
	}
	if got := NayinMap[a.Bazi.DayGanzhi]; got != a.FourNayin.Day.Nayin {
		t.Errorf("日柱 %s 纳音不匹配", a.Bazi.DayGanzhi)
	}
	if got := NayinMap[a.Bazi.HourGanzhi]; got != a.FourNayin.Hour.Nayin {
		t.Errorf("时柱 %s 纳音不匹配", a.Bazi.HourGanzhi)
	}
}

func TestBuildFourPillarNayin_All60(t *testing.T) {
	// 任意合法干支都应有纳音
	for gz, n := range NayinMap {
		b := &Bazi{YearGanzhi: gz, MonthGanzhi: gz, DayGanzhi: gz, HourGanzhi: gz}
		f := BuildFourPillarNayin(b)
		if f.Year.Nayin != n {
			t.Errorf("%s 纳音 %q ≠ 表 %q", gz, f.Year.Nayin, n)
		}
	}
}
