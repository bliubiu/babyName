package bazi

import "testing"

func TestLookupLongitude_Cities(t *testing.T) {
	lon, ok := LookupLongitude("北京")
	if !ok || lon < 116 || lon > 117 {
		t.Errorf("北京经度 = %v ok=%v", lon, ok)
	}
	if _, ok := LookupLongitude("不存在的城市"); ok {
		t.Error("未收录城市不应命中")
	}
	if lon, ok := LookupLongitude("成都市"); !ok || lon < 103 {
		t.Errorf("前缀匹配成都市失败: %v %v", lon, ok)
	}
}

func TestCorrectTrueSolarTime_Beijing(t *testing.T) {
	// 北京约 116.41，相对 120° 约慢 14 分钟
	y, m, d, h, mi := CorrectTrueSolarTime(2024, 6, 15, 12, 0, 116.41, 0)
	// 12:00 - ~14min ≈ 11:46（含均时差可能 ±15 分内）
	if h != 11 && h != 12 {
		t.Errorf("北京 12:00 真太阳时 = %d:%02d，期望约 11:4x", h, mi)
	}
	_ = y
	_ = m
	_ = d
}

func TestCorrectTrueSolarTime_Urumqi(t *testing.T) {
	// 乌鲁木齐 87.62，差约 -130 分钟 → 12:00 ≈ 09:50
	_, _, _, h, mi := CorrectTrueSolarTime(2024, 6, 15, 12, 0, 87.62, 0)
	if h != 9 && h != 10 {
		t.Errorf("乌鲁木齐 12:00 真太阳时 = %d:%02d，期望约 09:5x", h, mi)
	}
}

func TestApplyTrueSolar_UnknownLocation(t *testing.T) {
	y, m, d, h, mi, info := ApplyTrueSolar(2024, 1, 1, 8, 0, "某不存在地", 0)
	if info.Enabled {
		t.Error("未收录地点不应启用真太阳时")
	}
	if y != 2024 || m != 1 || d != 1 || h != 8 || mi != 0 {
		t.Error("未校正时应原样返回")
	}
}

func TestApplyTrueSolar_ExplicitLon(t *testing.T) {
	_, _, _, h, _, info := ApplyTrueSolar(2024, 6, 15, 12, 0, "", 87.62)
	if !info.Enabled {
		t.Fatal("显式经度应启用")
	}
	if h != 9 && h != 10 {
		t.Errorf("显式经度校正后小时 = %d", h)
	}
}

func TestEquationOfTime_Bounds(t *testing.T) {
	for day := 1; day <= 365; day += 7 {
		eot := EquationOfTimeMinutes(day)
		if eot < -20 || eot > 20 {
			t.Errorf("day=%d EoT=%.2f 超出 ±20 分钟", day, eot)
		}
	}
}
