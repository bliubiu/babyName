package ziwei

import (
	"testing"

	"name/internal/domain/bazi/tyme"
)

// TestLeapMonth_ByDayConvention 闰月口径必须采用与 iztro fixLeap 默认一致的
// 「按日切」规则，且月柱/命宫/辅星共用同一生效月份：
//   - 闰X月， lunarDay<=15 → 上个月(X月)
//   - 闰X月， lunarDay >15 → 下个月(X+1月)
//
// 评审发现的缺陷：旧实现月柱/命宫按「归后月」（闰X月恒当X+1月），
// 而辅星/杂曜按「归本月」（闰X月恒当X月），同一闰月盘内部月份基准冲突；
// 且纯归后月与 iztro 对闰月上半月的口径也不一致。本测试锁死按日切。
func TestLeapMonth_ByDayConvention(t *testing.T) {
	cases := []struct {
		year, month, day, hour int
	}{
		{2023, 3, 23, 10},  // 2023 闰二月（上半月，应归二月）
		{2023, 4, 12, 10},  // 2023 闰二月（下半月的较晚日期，应归三月）
		{1995, 10, 12, 20}, // 1995 闰八月（下半月，应归九月 → 命宫戊子，iztro 锚点）
	}

	seenFirst, seenSecond := false, false
	for _, c := range cases {
		solarDay, err := tyme.SolarDay{}.FromYmd(c.year, c.month, c.day)
		if err != nil || solarDay == nil {
			t.Fatalf("公历日期非法 %d-%02d-%02d：%v", c.year, c.month, c.day, err)
		}
		lm := solarDay.GetLunarDay().GetLunarMonth().GetMonthWithLeap()
		if lm >= 0 {
			t.Fatalf("用例 %d-%02d-%02d 非闰月（lunarMonth=%d），闰月口径测试失效", c.year, c.month, c.day, lm)
		}
		mAbs := -lm // 闰X月 → X
		lunarDay := solarDay.GetLunarDay().GetDay()

		if lunarDay <= 15 {
			seenFirst = true
		} else {
			seenSecond = true
		}

		chart := CalculateZiweiChart(c.year, c.month, c.day, c.hour, "男")
		if len([]rune(chart.YueZhu)) < 2 || len([]rune(chart.MingGong)) < 2 {
			t.Fatalf("排盘异常：YueZhu=%q MingGong=%q", chart.YueZhu, chart.MingGong)
		}

		// 按日切期望的 0-based 生效月份索引
		wantMonthIdx := mAbs - 1 // 上半月 → 上个月(X月)
		if lunarDay > 15 {
			wantMonthIdx = mAbs // 下半月 → 下个月(X+1月)
		}

		// 1) 月柱地支须 = 生效月份地支
		wantYueZhi := fixIndex(wantMonthIdx+2, 12)
		gotYueZhi := getZhiIndex(string([]rune(chart.YueZhu)[1]))
		if gotYueZhi != wantYueZhi {
			t.Errorf("%d-%02d-%02d(闰%d月 农历%d日) 月柱地支 got %s，want 按日切=%s",
				c.year, c.month, c.day, mAbs, lunarDay, chart.YueZhu, DizhiNames[wantYueZhi])
		}

		// 2) 命宫地支须 = 生效月份下的命宫
		timeIndex := hourToInt(c.hour)
		wantMingZhi := fixIndex(fixIndex(wantMonthIdx-timeIndex%12, 12)+2, 12)
		gotMingZhi := getZhiIndex(string([]rune(chart.MingGong)[1]))
		if gotMingZhi != wantMingZhi {
			t.Errorf("%d-%02d-%02d(闰%d月 农历%d日) 命宫地支 got %s，want 按日切=%s",
				c.year, c.month, c.day, mAbs, lunarDay, chart.MingGong, DizhiNames[wantMingZhi])
		}
	}

	if !seenFirst || !seenSecond {
		t.Fatalf("测试未能同时覆盖闰月上半月与下半月（seenFirst=%v seenSecond=%v），按日切分支未锁全", seenFirst, seenSecond)
	}
}
