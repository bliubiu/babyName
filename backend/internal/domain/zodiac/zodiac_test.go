package zodiac

import (
	"testing"
)

func TestZodiacList_All12(t *testing.T) {
	if len(ZodiacList) != 12 {
		t.Errorf("ZodiacList length = %d, want 12", len(ZodiacList))
	}
}

func TestZodiacList_NoDuplicateIDs(t *testing.T) {
	seen := make(map[int]bool)
	for _, z := range ZodiacList {
		if seen[z.ID] {
			t.Errorf("Duplicate zodiac ID: %d (%s)", z.ID, z.Name)
		}
		seen[z.ID] = true
	}
}

func TestZodiacList_NoDuplicateNames(t *testing.T) {
	seen := make(map[string]bool)
	for _, z := range ZodiacList {
		if seen[z.Name] {
			t.Errorf("Duplicate zodiac name: %s", z.Name)
		}
		seen[z.Name] = true
	}
}

func TestZodiacList_IDRange(t *testing.T) {
	for _, z := range ZodiacList {
		if z.ID < 1 || z.ID > 12 {
			t.Errorf("Zodiac %q has ID %d, want 1-12", z.Name, z.ID)
		}
	}
}

func TestZodiacList_Order(t *testing.T) {
	expected := []string{"鼠", "牛", "虎", "兔", "龙", "蛇", "马", "羊", "猴", "鸡", "狗", "猪"}
	for i, name := range expected {
		if ZodiacList[i].Name != name {
			t.Errorf("ZodiacList[%d].Name = %q, want %q", i, ZodiacList[i].Name, name)
		}
	}
}

func TestZodiacList_HasAllRequiredFields(t *testing.T) {
	for _, z := range ZodiacList {
		if z.Wuxing == "" {
			t.Errorf("Zodiac %q has empty Wuxing", z.Name)
		}
		if len(z.Compatible) == 0 {
			t.Errorf("Zodiac %q has empty Compatible", z.Name)
		}
		if len(z.Conflicting) == 0 {
			t.Errorf("Zodiac %q has empty Conflicting", z.Name)
		}
		if len(z.LuckyNumbers) == 0 {
			t.Errorf("Zodiac %q has empty LuckyNumbers", z.Name)
		}
		if z.LuckyColor == "" {
			t.Errorf("Zodiac %q has empty LuckyColor", z.Name)
		}
		if z.LuckyDir == "" {
			t.Errorf("Zodiac %q has empty LuckyDir", z.Name)
		}
	}
}

func TestZodiacList_WuxingIsValid(t *testing.T) {
	validWuxing := map[string]bool{"金": true, "木": true, "水": true, "火": true, "土": true}
	for _, z := range ZodiacList {
		if !validWuxing[z.Wuxing] {
			t.Errorf("Zodiac %q has invalid Wuxing %q", z.Name, z.Wuxing)
		}
	}
}

func TestZodiacList_SpecificValues(t *testing.T) {
	tests := []struct {
		id       int
		name     string
		wuxing   string
		compatible []string
	}{
		{1, "鼠", "水", []string{"牛", "龙", "猴"}},
		{2, "牛", "土", []string{"鼠", "蛇", "鸡"}},
		{3, "虎", "木", []string{"马", "狗", "猪"}},
		{4, "兔", "木", []string{"猪", "羊", "狗"}},
		{5, "龙", "土", []string{"鼠", "猴", "鸡"}},
		{6, "蛇", "火", []string{"牛", "鸡"}},
		{7, "马", "火", []string{"虎", "狗", "羊"}},
		{8, "羊", "土", []string{"猪", "兔", "马"}},
		{9, "猴", "金", []string{"鼠", "龙", "蛇"}},
		{10, "鸡", "金", []string{"牛", "龙", "蛇"}},
		{11, "狗", "土", []string{"虎", "兔", "马"}},
		{12, "猪", "水", []string{"虎", "兔", "羊"}},
	}

	for _, tc := range tests {
		// 按 ID 查找
		var found *Zodiac
		for i := range ZodiacList {
			if ZodiacList[i].ID == tc.id {
				found = &ZodiacList[i]
				break
			}
		}
		if found == nil {
			t.Errorf("Zodiac ID %d not found", tc.id)
			continue
		}
		if found.Name != tc.name {
			t.Errorf("Zodiac ID %d: Name = %q, want %q", tc.id, found.Name, tc.name)
		}
		if found.Wuxing != tc.wuxing {
			t.Errorf("Zodiac %q: Wuxing = %q, want %q", tc.name, found.Wuxing, tc.wuxing)
		}
		if len(found.Compatible) != len(tc.compatible) {
			t.Errorf("Zodiac %q: Compatible length = %d, want %d", tc.name, len(found.Compatible), len(tc.compatible))
		}
	}
}

func TestZodiacList_LuckyNumbers(t *testing.T) {
	for _, z := range ZodiacList {
		for _, n := range z.LuckyNumbers {
			if n < 0 || n > 100 {
				t.Errorf("Zodiac %q has unusual lucky number %d", z.Name, n)
			}
		}
	}
}
