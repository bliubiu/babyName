package zodiac

import "testing"

// TestGetZodiacByYearGanzhi_立春前后 生肖必须与年支一致：
// 2024-02-03 年柱癸卯（兔），2024-02-05 年柱甲辰（龙）。
func TestGetZodiacByYearGanzhi_立春前后(t *testing.T) {
	cases := []struct {
		ganzhi string
		want   string
	}{
		{"癸卯", "兔"},
		{"甲辰", "龙"},
		{"甲子", "鼠"},
		{"乙丑", "牛"},
		{"丙寅", "虎"},
		{"丁巳", "蛇"},
		{"戊午", "马"},
		{"己未", "羊"},
		{"庚申", "猴"},
		{"辛酉", "鸡"},
		{"壬戌", "狗"},
		{"癸亥", "猪"},
	}
	for _, c := range cases {
		z := GetZodiacByYearGanzhi(c.ganzhi)
		if z == nil {
			t.Errorf("%s：GetZodiacByYearGanzhi 返回 nil", c.ganzhi)
			continue
		}
		if z.Name != c.want {
			t.Errorf("%s：生肖 = %q，want %q", c.ganzhi, z.Name, c.want)
		}
	}
}

func TestGetZodiacByBranch_非法输入(t *testing.T) {
	if z := GetZodiacByBranch(""); z != nil {
		t.Errorf("空地支应返回 nil，实际 %q", z.Name)
	}
	if z := GetZodiacByBranch("A"); z != nil {
		t.Errorf("非地支应返回 nil，实际 %q", z.Name)
	}
	if z := GetZodiacByYearGanzhi("甲"); z != nil {
		t.Errorf("残缺年柱应返回 nil，实际 %q", z.Name)
	}
}
