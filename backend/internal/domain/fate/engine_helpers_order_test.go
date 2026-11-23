package fate

import "testing"

// TestEnsureWuxingDiversity_SortsByScore 回归：多样性筛选只决定入选集合，
// 返回顺序必须恢复评分降序，不能出现低分代表排在补位高分之前。
func TestEnsureWuxingDiversity_SortsByScore(t *testing.T) {
	entries := []ExcellentEntry{
		{Char1: "甲", Char2: "子", WuXing1: "金", WuXing2: "水", Score: 90},
		{Char1: "乙", Char2: "丑", WuXing1: "金", WuXing2: "水", Score: 89},
		{Char1: "丙", Char2: "寅", WuXing1: "木", WuXing2: "水", Score: 88},
		{Char1: "丁", Char2: "卯", WuXing1: "火", WuXing2: "土", Score: 70},
		{Char1: "戊", Char2: "辰", WuXing1: "金", WuXing2: "水", Score: 60},
	}

	got := ensureWuxingDiversity(entries, 4)
	if len(got) != 4 {
		t.Fatalf("结果数量 = %d，want 4", len(got))
	}
	for i := 1; i < len(got); i++ {
		if betterEntry(got[i], got[i-1]) {
			t.Fatalf("第 %d 名 %.1f 分高于第 %d 名 %.1f 分，榜单未按评分降序",
				i+1, got[i].Score, i, got[i-1].Score)
		}
	}
	if got[0].Score != 90 || got[1].Score != 89 || got[2].Score != 88 || got[3].Score != 70 {
		t.Fatalf("排序结果分数 = %.1f,%.1f,%.1f,%.1f，want 90,89,88,70",
			got[0].Score, got[1].Score, got[2].Score, got[3].Score)
	}
}
