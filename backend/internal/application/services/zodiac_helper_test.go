package services

import (
	"testing"
	"time"

	"name/internal/domain/bazi"
	"name/internal/domain/fate"
)

type fakeZodiacFinder struct{ name string }

func (f fakeZodiacFinder) FindByYear(int) string { return f.name }

// TestZodiacNameForBazi_优先年支 回归：立春前出生必须按年柱取生肖，
// 不能被公历年带偏（2024-01-15 年柱癸卯 → 兔，而非公历年的龙）。
func TestZodiacNameForBazi_优先年支(t *testing.T) {
	analysis := &bazi.BaziAnalysis{Bazi: bazi.Bazi{YearGanzhi: "癸卯"}}
	got := zodiacNameForBazi(analysis, fakeZodiacFinder{name: "龙"}, 2024)
	if got != "兔" {
		t.Fatalf("生肖 = %q，want 兔（年柱癸卯）", got)
	}

	analysis.Bazi.YearGanzhi = "甲辰"
	if got := zodiacNameForBazi(analysis, fakeZodiacFinder{name: "兔"}, 2024); got != "龙" {
		t.Fatalf("生肖 = %q，want 龙（年柱甲辰）", got)
	}
}

func TestZodiacNameForBazi_分析缺失回退(t *testing.T) {
	if got := zodiacNameForBazi(nil, fakeZodiacFinder{name: "龙"}, 2024); got != "龙" {
		t.Fatalf("无分析结果应回退公历年，got %q", got)
	}
	if got := zodiacNameForBazi(&bazi.BaziAnalysis{}, nil, 2024); got != "" {
		t.Fatalf("无分析且无 finder 应返回空，got %q", got)
	}
}

// TestBaziAnalyzerAdapter_立春前后生肖 覆盖真正进入引擎的 FateData 生肖，
// 该字段同时决定生肖维度评分。
func TestBaziAnalyzerAdapter_立春前后生肖(t *testing.T) {
	adapter := NewBaziAnalyzerAdapter()
	cases := []struct {
		born time.Time
		want string
	}{
		{time.Date(2024, 2, 3, 10, 0, 0, 0, time.UTC), "兔"},
		{time.Date(2024, 2, 5, 10, 0, 0, 0, time.UTC), "龙"},
		{time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC), "兔"},
	}
	for _, c := range cases {
		data, err := adapter.Analyze(c.born, fate.GenderMale)
		if err != nil {
			t.Fatalf("%s Analyze 失败: %v", c.born.Format("2006-01-02"), err)
		}
		if data.BaziInfo.Zodiac != c.want {
			t.Errorf("%s 生肖 = %q，want %q（年柱 %s）",
				c.born.Format("2006-01-02"), data.BaziInfo.Zodiac, c.want, data.BaziInfo.FourPillars[0])
		}
	}
}
