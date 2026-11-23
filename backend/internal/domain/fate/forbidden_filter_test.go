package fate

import "testing"

// TestFilterForbiddenEntries 榜单入口先剔除合规红线名，保证后续截断不被占名额。
func TestFilterForbiddenEntries(t *testing.T) {
	entries := []ExcellentEntry{
		{Char1: "中", Char2: "国", Score: 90}, // 王中国 → 命中「中国」
		{Char1: "明", Char2: "月", Score: 85},
		{Char1: "国", Char2: "安", Score: 80},
		{Char1: "子", Char2: "涵", Score: 70},
	}

	got := filterForbiddenEntries("王", entries)
	if len(got) != 3 {
		t.Fatalf("过滤后条数 = %d, 期望 3（仅「王中国」应被剔除）", len(got))
	}
	for _, e := range got {
		if e.Char1 == "中" && e.Char2 == "国" {
			t.Fatalf("「王中国」未被剔除: %+v", e)
		}
	}
	// 顺序必须保持（过滤不重排，后续按分数截断依赖原顺序）
	wantOrder := []string{"明月", "国安", "子涵"}
	for i, want := range wantOrder {
		if got[i].Char1+got[i].Char2 != want {
			t.Errorf("过滤后第 %d 项 = %q, 期望 %q", i, got[i].Char1+got[i].Char2, want)
		}
	}
}

// TestFilterForbiddenEntries_AllAllowed 无红线命中时不得改变条数与顺序。
func TestFilterForbiddenEntries_AllAllowed(t *testing.T) {
	entries := []ExcellentEntry{
		{Char1: "明", Char2: "月"},
		{Char1: "子", Char2: "涵"},
	}
	got := filterForbiddenEntries("李", entries)
	if len(got) != len(entries) {
		t.Fatalf("条数 = %d, 期望 %d", len(got), len(entries))
	}
	for i := range entries {
		if got[i].Char1 != entries[i].Char1 || got[i].Char2 != entries[i].Char2 {
			t.Errorf("第 %d 项被改动: %+v", i, got[i])
		}
	}
}
