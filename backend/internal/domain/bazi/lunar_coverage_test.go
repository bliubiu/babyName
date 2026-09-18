package bazi

import (
	"testing"
)

// TestLunarCalendarCoverage tests the LunarCalendar struct
func TestLunarCalendarCoverage(t *testing.T) {
	lc := GetLunarCalendar(2024, 6, 15, 12, 0)
	if lc == nil {
		t.Error("GetLunarCalendar should not return nil")
	}
	if lc.Year != 2024 {
		t.Errorf("Year = %d, want 2024", lc.Year)
	}
	if lc.Month != 6 {
		t.Errorf("Month = %d, want 6", lc.Month)
	}
	if lc.Day != 15 {
		t.Errorf("Day = %d, want 15", lc.Day)
	}
	if lc.YearGan == "" {
		t.Error("YearGan should not be empty")
	}
	if lc.YearZhi == "" {
		t.Error("YearZhi should not be empty")
	}
	if lc.MonthGan == "" {
		t.Error("MonthGan should not be empty")
	}
	if lc.MonthZhi == "" {
		t.Error("MonthZhi should not be empty")
	}
	if lc.DayGan == "" {
		t.Error("DayGan should not be empty")
	}
	if lc.DayZhi == "" {
		t.Error("DayZhi should not be empty")
	}
	if lc.HourGan == "" {
		t.Error("HourGan should not be empty")
	}
	if lc.HourZhi == "" {
		t.Error("HourZhi should not be empty")
	}
	if lc.Nayin == "" {
		t.Error("Nayin should not be empty")
	}
	if lc.YearNayin == "" {
		t.Error("YearNayin should not be empty")
	}
	if lc.MonthNayin == "" {
		t.Error("MonthNayin should not be empty")
	}
	if lc.DayNayin == "" {
		t.Error("DayNayin should not be empty")
	}
	if lc.LunarYearName == "" {
		t.Error("LunarYearName should not be empty")
	}
	if lc.LunarMonthName == "" {
		t.Error("LunarMonthName should not be empty")
	}
	if lc.LunarDayName == "" {
		t.Error("LunarDayName should not be empty")
	}
	if lc.DateStr == "" {
		t.Error("DateStr should not be empty")
	}
	if lc.SolarTerm == "" {
		t.Error("SolarTerm should not be empty")
	}
}

// TestGetYearGanzhiCoverage tests GetYearGanzhi
func TestGetYearGanzhiCoverage(t *testing.T) {
	gan, zhi := GetYearGanzhi(2024)
	if gan == "" || zhi == "" {
		t.Errorf("GetYearGanzhi(2024) = (%q, %q), want non-empty", gan, zhi)
	}

	years := []int{2020, 2021, 2022, 2023, 2024, 2025, 1984, 1990, 2000}
	for _, year := range years {
		gan, zhi := GetYearGanzhi(year)
		if gan == "" || zhi == "" {
			t.Errorf("GetYearGanzhi(%d) = (%q, %q), want non-empty", year, gan, zhi)
		}
	}
}

// TestGetMonthGanzhiCoverage tests GetMonthGanzhi
func TestGetMonthGanzhiCoverage(t *testing.T) {
	gan, zhi := GetMonthGanzhi(2024, 6)
	if gan == "" || zhi == "" {
		t.Errorf("GetMonthGanzhi(2024, 6) = (%q, %q), want non-empty", gan, zhi)
	}

	for month := 1; month <= 12; month++ {
		gan, zhi := GetMonthGanzhi(2024, month)
		if gan == "" || zhi == "" {
			t.Errorf("GetMonthGanzhi(2024, %d) = (%q, %q), want non-empty", month, gan, zhi)
		}
	}
}

// TestGetDayGanzhiCoverage tests GetDayGanzhi
func TestGetDayGanzhiCoverage(t *testing.T) {
	gan, zhi := GetDayGanzhi(2024, 6, 15)
	if gan == "" || zhi == "" {
		t.Errorf("GetDayGanzhi(2024, 6, 15) = (%q, %q), want non-empty", gan, zhi)
	}

	dates := []struct{ year, month, day int }{
		{2024, 1, 1}, {2024, 6, 15}, {2024, 12, 31},
		{2023, 12, 31}, {2025, 1, 1},
	}
	for _, d := range dates {
		gan, zhi := GetDayGanzhi(d.year, d.month, d.day)
		if gan == "" || zhi == "" {
			t.Errorf("GetDayGanzhi(%d, %d, %d) = (%q, %q), want non-empty", d.year, d.month, d.day, gan, zhi)
		}
	}
}

// TestGetHourGanzhiCoverage tests GetHourGanzhi
func TestGetHourGanzhiCoverage(t *testing.T) {
	gan, zhi := GetHourGanzhi("甲", 12, 0)
	if gan == "" || zhi == "" {
		t.Errorf("GetHourGanzhi(甲, 12, 0) = (%q, %q), want non-empty", gan, zhi)
	}

	for hour := 0; hour < 24; hour += 2 {
		gan, zhi := GetHourGanzhi("甲", hour, 0)
		if gan == "" || zhi == "" {
			t.Errorf("GetHourGanzhi(甲, %d, 0) = (%q, %q), want non-empty", hour, gan, zhi)
		}
	}
}

// TestGetNayinCoverage tests GetNayin
func TestGetNayinCoverage(t *testing.T) {
	nayin := GetNayin("甲子")
	if nayin != "海中金" {
		t.Errorf("GetNayin(甲子) = %q, want 海中金", nayin)
	}

	for ganzhi, expected := range NayinMap {
		n := GetNayin(ganzhi)
		if n != expected {
			t.Errorf("GetNayin(%s) = %q, want %q", ganzhi, n, expected)
		}
	}

	unknown := GetNayin("未知")
	if unknown != "" {
		t.Errorf("GetNayin(未知) = %q, want empty", unknown)
	}
}

// TestGetNayinWuxingCoverage tests GetNayinWuxing
func TestGetNayinWuxingCoverage(t *testing.T) {
	wx := GetNayinWuxing("海中金")
	if wx != "金" {
		t.Errorf("GetNayinWuxing(海中金) = %q, want 金", wx)
	}

	// Test all nayin
	for _, nayin := range NayinMap {
		wx := GetNayinWuxing(nayin)
		validWx := map[string]bool{"金": true, "木": true, "水": true, "火": true, "土": true}
		if !validWx[wx] {
			t.Errorf("GetNayinWuxing(%s) = %q, not a valid 五行", nayin, wx)
		}
	}
}

// TestGetDayYiJiCoverage tests GetDayYiJi
func TestGetDayYiJiCoverage(t *testing.T) {
	yiJi := GetDayYiJi("子")
	if len(yiJi.Yi) == 0 && len(yiJi.Ji) == 0 {
		t.Error("GetDayYiJi should have Yi or Ji")
	}

	// Test all day zhi
	dizhis := []string{"子", "丑", "寅", "卯", "辰", "巳", "午", "未", "申", "酉", "戌", "亥"}
	for _, dz := range dizhis {
		yj := GetDayYiJi(dz)
		if len(yj.Yi) == 0 && len(yj.Ji) == 0 {
			t.Errorf("GetDayYiJi(%s) should have Yi or Ji", dz)
		}
	}
}

// TestGetChongZhiCoverage tests GetChongZhi
func TestGetChongZhiCoverage(t *testing.T) {
	chongZhi := GetChongZhi("子")
	if chongZhi != "午" {
		t.Errorf("GetChongZhi(子) = %q, want 午", chongZhi)
	}

	dizhis := []string{"子", "丑", "寅", "卯", "辰", "巳", "午", "未", "申", "酉", "戌", "亥"}
	expectedChong := map[string]string{
		"子": "午", "丑": "未", "寅": "申", "卯": "酉",
		"辰": "戌", "巳": "亥", "午": "子", "未": "丑",
		"申": "寅", "酉": "卯", "戌": "辰", "亥": "巳",
	}
	for _, dz := range dizhis {
		c := GetChongZhi(dz)
		if c != expectedChong[dz] {
			t.Errorf("GetChongZhi(%s) = %q, want %q", dz, c, expectedChong[dz])
		}
	}
}

// TestGetShaZhiCoverage tests GetShaZhi
func TestGetShaZhiCoverage(t *testing.T) {
	shaZhi := GetShaZhi("子")
	if shaZhi != "马" {
		t.Errorf("GetShaZhi(子) = %q, want 马", shaZhi)
	}

	dizhis := []string{"子", "丑", "寅", "卯", "辰", "巳", "午", "未", "申", "酉", "戌", "亥"}
	expectedSha := map[string]string{
		"子": "马", "丑": "羊", "寅": "猴", "卯": "鸡",
		"辰": "龙", "巳": "蛇", "午": "马", "未": "羊",
		"申": "猴", "酉": "鸡", "戌": "狗", "亥": "猪",
	}
	for _, dz := range dizhis {
		s := GetShaZhi(dz)
		if s != expectedSha[dz] {
			t.Errorf("GetShaZhi(%s) = %q, want %q", dz, s, expectedSha[dz])
		}
	}
}

// TestGetJiShenCoverage tests GetJiShen
func TestGetJiShenCoverage(t *testing.T) {
	jiShen := GetJiShen("甲", "子")
	if len(jiShen) == 0 {
		t.Error("GetJiShen should not be empty")
	}

	tiangans := []string{"甲", "乙", "丙", "丁", "戊", "己", "庚", "辛", "壬", "癸"}
	dizhis := []string{"子", "丑", "寅", "卯", "辰", "巳", "午", "未", "申", "酉", "戌", "亥"}

	for _, tg := range tiangans {
		for _, dz := range dizhis {
			js := GetJiShen(tg, dz)
			_ = js
		}
	}
}

// TestGetXiongShaCoverage tests GetXiongSha
func TestGetXiongShaCoverage(t *testing.T) {
	xiongSha := GetXiongSha("甲", "子")
	if len(xiongSha) == 0 {
		t.Error("GetXiongSha should not be empty")
	}

	tiangans := []string{"甲", "乙", "丙", "丁", "戊", "己", "庚", "辛", "壬", "癸"}
	dizhis := []string{"子", "丑", "寅", "卯", "辰", "巳", "午", "未", "申", "酉", "戌", "亥"}

	for _, tg := range tiangans {
		for _, dz := range dizhis {
			xs := GetXiongSha(tg, dz)
			_ = xs
		}
	}
}

// TestGetHuangliCoverage tests GetHuangli
func TestGetHuangliCoverage(t *testing.T) {
	hl := GetHuangli(2024, 6, 15)
	if hl == nil {
		t.Fatal("GetHuangli should not return nil")
	}
	if hl.Date == "" {
		t.Error("Date should not be empty")
	}
	if hl.Year == "" {
		t.Error("Year should not be empty")
	}
	if hl.Month == "" {
		t.Error("Month should not be empty")
	}
	if hl.Day == "" {
		t.Error("Day should not be empty")
	}
	if hl.Yi == nil {
		t.Error("Yi should not be nil")
	}
	if hl.Ji == nil {
		t.Error("Ji should not be nil")
	}
	if hl.JiShen == nil {
		t.Error("JiShen should not be nil")
	}
	if hl.XiongSha == nil {
		t.Error("XiongSha should not be nil")
	}
	if hl.BaiCang == "" {
		t.Error("BaiCang should not be empty")
	}
	if hl.PengZhu == "" {
		t.Error("PengZhu should not be empty")
	}
	if hl.Fu == "" {
		t.Error("Fu should not be empty")
	}
	if hl.JieShen == "" {
		t.Error("JieShen should not be empty")
	}
	if hl.Chong == "" {
		t.Error("Chong should not be empty")
	}
	if hl.Sha == "" {
		t.Error("Sha should not be empty")
	}
	if hl.ZhangSong == "" {
		t.Error("ZhangSong should not be empty")
	}
	if hl.YearWuxing == "" {
		t.Error("YearWuxing should not be empty")
	}
	if hl.MonthWuxing == "" {
		t.Error("MonthWuxing should not be empty")
	}
	if hl.DayWuxing == "" {
		t.Error("DayWuxing should not be empty")
	}
	if hl.LiuYao == "" {
		t.Error("LiuYao should not be empty")
	}
	if hl.ZhiShen == "" {
		t.Error("ZhiShen should not be empty")
	}
	if hl.TaiShen == "" {
		t.Error("TaiShen should not be empty")
	}
	if hl.JianChu == "" {
		t.Error("JianChu should not be empty")
	}
	if hl.XingXiu == "" {
		t.Error("XingXiu should not be empty")
	}
	if hl.XiShen == "" {
		t.Error("XiShen should not be empty")
	}
	if hl.FuShen == "" {
		t.Error("FuShen should not be empty")
	}
	if hl.CaiShen == "" {
		t.Error("CaiShen should not be empty")
	}
	if hl.YangGui == "" {
		t.Error("YangGui should not be empty")
	}
	if hl.YinGui == "" {
		t.Error("YinGui should not be empty")
	}
	if hl.Hours == nil {
		t.Error("Hours should not be nil")
	}
	if len(hl.Hours) != 12 {
		t.Errorf("Hours length = %d, want 12", len(hl.Hours))
	}
}

// TestGenerateHourYiJiCoverage tests generateHourYiJi
func TestGenerateHourYiJiCoverage(t *testing.T) {
	yiJi := generateHourYiJi("甲", "子")
	if len(yiJi) == 0 {
		t.Error("generateHourYiJi should not be empty")
	}

	tiangans := []string{"甲", "乙", "丙", "丁", "戊", "己", "庚", "辛", "壬", "癸"}
	dizhis := []string{"子", "丑", "寅", "卯", "辰", "巳", "午", "未", "申", "酉", "戌", "亥"}

	for _, tg := range tiangans {
		for _, dz := range dizhis {
			yj := generateHourYiJi(tg, dz)
			if len(yj) == 0 {
				t.Errorf("generateHourYiJi(%s, %s) should not be empty", tg, dz)
			}
			if len(yj) != 12 {
				t.Errorf("generateHourYiJi(%s, %s) length = %d, want 12", tg, dz, len(yj))
			}
		}
	}
}

// TestGetBaCangCoverage tests GetBaCang
func TestGetBaCangCoverage(t *testing.T) {
	baCang := GetBaCang("甲")
	if baCang != "仓库" {
		t.Errorf("GetBaCang(甲) = %q, want 仓库", baCang)
	}

	tiangans := []string{"甲", "乙", "丙", "丁", "戊", "己", "庚", "辛", "壬", "癸"}
	expected := map[string]string{
		"甲": "仓库", "乙": "花仓", "丙": "光猛", "丁": "星星",
		"戊": "霞", "己": "云", "庚": "路", "辛": "秀",
		"壬": "湖", "癸": "泉",
	}
	for _, tg := range tiangans {
		bc := GetBaCang(tg)
		if bc != expected[tg] {
			t.Errorf("GetBaCang(%s) = %q, want %q", tg, bc, expected[tg])
		}
	}
}

// TestGetPengZhuCoverage tests GetPengZhu
func TestGetPengZhuCoverage(t *testing.T) {
	pengZhu := GetPengZhu("子")
	if pengZhu != "司命" {
		t.Errorf("GetPengZhu(子) = %q, want 司命", pengZhu)
	}

	dizhis := []string{"子", "丑", "寅", "卯", "辰", "巳", "午", "未", "申", "酉", "戌", "亥"}
	expected := map[string]string{
		"子": "司命", "丑": "勾陈", "寅": "青龙", "卯": "明堂",
		"辰": "天刑", "巳": "朱雀", "午": "金匮", "未": "天德",
		"申": "白虎", "酉": "玉堂", "戌": "天牢", "亥": "玄武",
	}
	for _, dz := range dizhis {
		pz := GetPengZhu(dz)
		if pz != expected[dz] {
			t.Errorf("GetPengZhu(%s) = %q, want %q", dz, pz, expected[dz])
		}
	}
}

// TestGetJieShenCoverage tests GetJieShen
func TestGetJieShenCoverage(t *testing.T) {
	jieShen := GetJieShen("子")
	if jieShen != "天喜" {
		t.Errorf("GetJieShen(子) = %q, want 天喜", jieShen)
	}

	dizhis := []string{"子", "丑", "寅", "卯", "辰", "巳", "午", "未", "申", "酉", "戌", "亥"}
	expected := map[string]string{
		"子": "天喜", "丑": "天厨", "寅": "天福", "卯": "天德",
		"辰": "月德", "巳": "天息", "午": "天恩", "未": "三合",
		"申": "天仓", "酉": "天富", "戌": "天喜", "亥": "天庆",
	}
	for _, dz := range dizhis {
		js := GetJieShen(dz)
		if js != expected[dz] {
			t.Errorf("GetJieShen(%s) = %q, want %q", dz, js, expected[dz])
		}
	}
}

// TestGetChongShaCoverage tests GetChongSha
func TestGetChongShaCoverage(t *testing.T) {
	chong, sha := GetChongSha("子", "丑")
	if chong != "午" {
		t.Errorf("GetChongSha chong = %q, want 午", chong)
	}
	if sha != "马" {
		t.Errorf("GetChongSha sha = %q, want 马", sha)
	}

	dizhis := []string{"子", "丑", "寅", "卯", "辰", "巳", "午", "未", "申", "酉", "戌", "亥"}
	for _, d1 := range dizhis {
		for _, d2 := range dizhis {
			c, s := GetChongSha(d1, d2)
			_ = c
			_ = s
		}
	}
}

// TestGetZhangSongCoverage tests GetZhangSong
func TestGetZhangSongCoverage(t *testing.T) {
	zhangSong := GetZhangSong("子")
	_ = zhangSong // May be empty
}

// TestGetFuCoverage tests GetFu
func TestGetFuCoverage(t *testing.T) {
	fu := GetFu("子")
	_ = fu // May be empty
}

// TestGetLiuYaoCoverage tests GetLiuYao
func TestGetLiuYaoCoverage(t *testing.T) {
	liuYao := GetLiuYao("甲", "子")
	if liuYao == "" {
		t.Error("GetLiuYao should not be empty")
	}

	tiangans := []string{"甲", "乙", "丙", "丁", "戊", "己", "庚", "辛", "壬", "癸"}
	dizhis := []string{"子", "丑", "寅", "卯", "辰", "巳", "午", "未", "申", "酉", "戌", "亥"}

	for _, tg := range tiangans {
		for _, dz := range dizhis {
			ly := GetLiuYao(tg, dz)
			_ = ly
		}
	}
}

// TestGetZhiShenCoverage tests GetZhiShen
func TestGetZhiShenCoverage(t *testing.T) {
	zhiShen := GetZhiShen("子")
	_ = zhiShen
}

// TestGetTaiShenCoverage tests GetTaiShen
func TestGetTaiShenCoverage(t *testing.T) {
	taiShen := GetTaiShen("甲", "子")
	_ = taiShen
}

// TestGetJianChuCoverage tests GetJianChu
func TestGetJianChuCoverage(t *testing.T) {
	jianChu := GetJianChu("子")
	_ = jianChu
}

// TestGetXingXiuCoverage tests GetXingXiu
func TestGetXingXiuCoverage(t *testing.T) {
	xingXiu := GetXingXiu("甲", "子")
	_ = xingXiu
}

// TestGetXiShenCoverage tests GetXiShen
func TestGetXiShenCoverage(t *testing.T) {
	xiShen := GetXiShen("甲")
	_ = xiShen
}

// TestGetFuShenCoverage tests GetFuShen
func TestGetFuShenCoverage(t *testing.T) {
	fuShen := GetFuShen("甲")
	_ = fuShen
}

// TestGetCaiShenCoverage tests GetCaiShen
func TestGetCaiShenCoverage(t *testing.T) {
	caiShen := GetCaiShen("甲")
	_ = caiShen
}

// TestGetYangGuiCoverage tests GetYangGui
func TestGetYangGuiCoverage(t *testing.T) {
	yangGui := GetYangGui("甲")
	_ = yangGui
}

// TestGetYinGuiCoverage tests GetYinGui
func TestGetYinGuiCoverage(t *testing.T) {
	yinGui := GetYinGui("甲")
	_ = yinGui
}

// TestGetCurrentJieQiCoverage tests GetCurrentJieQi
func TestGetCurrentJieQiCoverage(t *testing.T) {
	jieQi := GetCurrentJieQi(2024, 6, 15)
	if jieQi == "" {
		t.Error("GetCurrentJieQi should not be empty")
	}

	dates := []struct{ year, month, day int }{
		{2024, 1, 5}, {2024, 2, 4}, {2024, 3, 5}, {2024, 4, 4},
		{2024, 5, 5}, {2024, 6, 5}, {2024, 7, 6}, {2024, 8, 7},
		{2024, 9, 7}, {2024, 10, 8}, {2024, 11, 7}, {2024, 12, 6},
	}
	for _, d := range dates {
		jq := GetCurrentJieQi(d.year, d.month, d.day)
		if jq == "" {
			t.Errorf("GetCurrentJieQi(%d, %d, %d) should not be empty", d.year, d.month, d.day)
		}
	}
}