package bazi

import "testing"

func TestCalculateWeightedWuxing_MonthBoost(t *testing.T) {
	// 甲木日主，寅月（木旺），加权后木应明显偏高
	b := &Bazi{
		YearGanzhi:  "甲子",
		MonthGanzhi: "丙寅",
		DayGanzhi:   "甲子",
		HourGanzhi:  "甲子",
	}
	w := CalculateWeightedWuxing(b)
	if w.Mu <= w.Jin {
		t.Errorf("木分 %.2f 应高于金分 %.2f", w.Mu, w.Jin)
	}
	// 月支寅藏甲0.6+丙0.3+戊0.1，月令×1.5 → 木至少 1+1+1 + 0.6*1.5 = 3.9
	if w.Mu < 3.0 {
		t.Errorf("加权木分 %.2f 过低，月令应放大木", w.Mu)
	}
}

func TestCalculateWeightedWuxing_HiddenStems(t *testing.T) {
	// 寅藏甲丙戊，仅看月柱
	b := &Bazi{
		YearGanzhi:  "甲子",
		MonthGanzhi: "丙寅",
		DayGanzhi:   "甲子",
		HourGanzhi:  "甲子",
	}
	w := CalculateWeightedWuxing(b)
	// 丙干+寅中丙0.3*1.5
	if w.Huo < 1.0+0.3 {
		t.Errorf("火分 %.2f 应含丙干与寅中丙", w.Huo)
	}
	if w.Tu < 0.1 {
		t.Errorf("土分 %.2f 应含寅中戊", w.Tu)
	}
}

func TestDizhiHiddenStems_AllBranches(t *testing.T) {
	for _, zhi := range Dizhi {
		stems, ok := DizhiHiddenStems[zhi]
		if !ok || len(stems) == 0 {
			t.Errorf("地支 %s 缺藏干表", zhi)
			continue
		}
		sum := 0.0
		for _, s := range stems {
			if s.Stem == "" || s.Weight <= 0 {
				t.Errorf("地支 %s 藏干条目非法: %+v", zhi, s)
			}
			if _, ok := WuxingMap[s.Stem]; !ok {
				t.Errorf("地支 %s 藏干 %s 非天干", zhi, s.Stem)
			}
			sum += s.Weight
		}
		if sum < 0.9 || sum > 1.1 {
			t.Errorf("地支 %s 藏干权重合计 %.2f 应约等于 1.0", zhi, sum)
		}
	}
}

func TestAnalyzeBazi_WeightedStrength(t *testing.T) {
	// 甲寅月木旺，甲日主应偏旺或至少非身衰
	a, err := AnalyzeBazi(1984, 2, 15, 12, 0) // 约寅月
	if err != nil {
		t.Fatal(err)
	}
	if a.Xiyongshen == nil || len(a.Xiyongshen) == 0 {
		t.Error("喜用神不应为空")
	}
	// BestHexagram 不再硬编码填充
	if a.BestHexagram != "" {
		t.Errorf("AnalyzeBazi 不应硬编码 BestHexagram，实际 %q", a.BestHexagram)
	}
}

func TestCalculateXiyongshenWeighted_Biased(t *testing.T) {
	// 日主木极旺 → 喜金火土（克泄耗）
	w := &WeightedWuxing{Mu: 8, Huo: 1, Tu: 1, Jin: 0.5, Shui: 0.5}
	xy := calculateXiyongshenWeighted(w, "木", "身旺", nil)
	if len(xy) == 0 {
		t.Fatal("喜用神为空")
	}
	found := false
	for _, x := range xy {
		if x == "金" || x == "火" || x == "土" {
			found = true
		}
	}
	if !found {
		t.Errorf("身旺木应取克泄耗，实际 %v", xy)
	}

	// 日主木极弱 → 喜水木
	w2 := &WeightedWuxing{Mu: 0.2, Jin: 5, Huo: 3, Tu: 3, Shui: 0.3}
	xy2 := calculateXiyongshenWeighted(w2, "木", "身衰", nil)
	hasWaterOrWood := false
	for _, x := range xy2 {
		if x == "水" || x == "木" {
			hasWaterOrWood = true
		}
	}
	if !hasWaterOrWood {
		t.Errorf("身衰木应取生扶，实际 %v", xy2)
	}
}
