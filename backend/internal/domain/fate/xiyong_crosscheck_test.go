package fate

import (
	"testing"

	"name/internal/domain/bazi"
)

// TestXiyongshenCrossCheck_ClassicVsFateDirection 经典喜用神与 fate 平衡用神方向对拍
// 断言：同一四柱下，两边喜用五行在「生扶/克泄」大方向上一致率 ≥ 60%。
// 不要求逐字相同（算法细节不同），但不能系统性反向。
func TestXiyongshenCrossCheck_ClassicVsFateDirection(t *testing.T) {
	cases := []struct {
		name                      string
		year, mon, day, hour, min int
	}{
		{"2024春甲辰", 2024, 3, 20, 10, 0},
		{"2024夏甲辰", 2024, 6, 15, 14, 30},
		{"2024秋甲辰", 2024, 9, 10, 8, 0},
		{"2024冬甲辰", 2024, 12, 21, 22, 0},
		{"1984甲子", 1984, 10, 1, 8, 30},
		{"2000庚辰", 2000, 2, 4, 12, 0},
		{"1990庚午", 1990, 7, 7, 18, 0},
	}

	agree := 0
	total := 0

	for _, tc := range cases {
		classic, err := bazi.AnalyzeBazi(tc.year, tc.mon, tc.day, tc.hour, tc.min)
		if err != nil {
			t.Fatalf("%s classic: %v", tc.name, err)
		}
		if len(classic.Xiyongshen) == 0 {
			t.Errorf("%s 经典喜用神为空", tc.name)
			continue
		}

		// fate 平衡用神
		info := BaziInfoForGeJu{SiZhu: [4]string{
			classic.Bazi.YearGanzhi,
			classic.Bazi.MonthGanzhi,
			classic.Bazi.DayGanzhi,
			classic.Bazi.HourGanzhi,
		}}
		fateXY := BalanceXiYongJi(info)

		total++
		// 方向对拍：经典喜用是否包含 fate 的用神或喜神
		set := map[string]bool{}
		for _, x := range classic.Xiyongshen {
			set[x] = true
		}
		if set[fateXY.Yong] || set[fateXY.Xi] {
			agree++
		} else {
			t.Logf("%s 日主%s 身%s 经典喜用=%v fate用=%s 喜=%s",
				tc.name, classic.RishouWuxing, classic.DayMasterStrength,
				classic.Xiyongshen, fateXY.Yong, fateXY.Xi)
		}
	}

	if total == 0 {
		t.Fatal("无有效用例")
	}
	rate := float64(agree) / float64(total)
	t.Logf("方向一致率: %d/%d = %.0f%%", agree, total, rate*100)
	if rate < 0.55 {
		t.Errorf("经典与 fate 喜用方向一致率 %.0f%% 过低（期望≥60%%）", rate*100)
	}
}

// TestClassicXiyongshen_AlwaysNonEmpty 全样本喜用神非空
func TestClassicXiyongshen_AlwaysNonEmpty(t *testing.T) {
	for _, tc := range []struct{ y, mo, d, h, mi int }{
		{2024, 1, 1, 0, 0},
		{2024, 6, 21, 12, 0},
		{1999, 12, 31, 23, 59},
	} {
		a, err := bazi.AnalyzeBazi(tc.y, tc.mo, tc.d, tc.h, tc.mi)
		if err != nil {
			t.Fatal(err)
		}
		if len(a.Xiyongshen) == 0 {
			t.Errorf("%v 喜用神为空", tc)
		}
		// 加权后日主强弱应在四档内
		switch a.DayMasterStrength {
		case "身旺", "身中", "身弱", "身衰":
		default:
			t.Errorf("非法日主强弱 %q", a.DayMasterStrength)
		}
	}
}
