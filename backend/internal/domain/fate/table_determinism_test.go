package fate

// table_determinism_test.go — Top-N 全序确定性守护（docs/32 §6.2 配套修复）
//
// 背景：golden 快照首次接入即实测复现「同进程重复生成不一致」——
// 同一请求两次生成，同分 74.1 的首位分别是「王金」「王春」。根因：
// Finalize 的稳定排序只保证「同分保持堆内既有次序」，而堆内次序由
// TryPush 的推入次序决定；双名路径是多 worker 并发推入，同分名次
// 随 goroutine 调度波动。
//
// 修复：引入全序 betterEntry（分数降序 + 同分按 Char1+Char2 字典序），
// 堆比较、容量替换判断、Finalize 排序三处统一走全序，使 Top-N 对
// 推入次序完全不敏感。

import (
	"reflect"
	"testing"
)

// sameScoreEntries 构造 n 个同分条目（key 依次为 甲乙丙……）
func sameScoreEntries(t *testing.T, keys []string, score float64) []ExcellentEntry {
	t.Helper()
	out := make([]ExcellentEntry, 0, len(keys))
	for _, k := range keys {
		runes := []rune(k)
		if len(runes) != 2 {
			t.Fatalf("测试条目 key 必须是两个汉字：%q", k)
		}
		out = append(out, ExcellentEntry{Char1: string(runes[0]), Char2: string(runes[1]), Score: score})
	}
	return out
}

// pushAll 按给定顺序推入全部条目
func pushAll(table *ExcellentTable, entries []ExcellentEntry) {
	for _, e := range entries {
		table.TryPush(e)
	}
}

// reverse 返回逆序副本（不改动入参）
func reverse(in []ExcellentEntry) []ExcellentEntry {
	out := make([]ExcellentEntry, len(in))
	for i, e := range in {
		out[len(in)-1-i] = e
	}
	return out
}

// TestExcellentTable_同分次序与推入顺序无关 同分条目无论以什么顺序进堆，
// Finalize 后的名次必须一致（字典序升序）
func TestExcellentTable_同分次序与推入顺序无关(t *testing.T) {
	keys := []string{"王金", "王春", "王秋", "王夏", "王冬"}
	entries := sameScoreEntries(t, keys, 74.1)

	forward := NewExcellentTableWithCap(100)
	pushAll(forward, entries)
	forward.Finalize()
	want := forward.TopN(5)

	backward := NewExcellentTableWithCap(100)
	pushAll(backward, reverse(entries))
	backward.Finalize()
	got := backward.TopN(5)

	for i := range want {
		if entryKey(want[i]) != entryKey(got[i]) {
			t.Fatalf("同分名次随推入顺序波动：\n正序推入=%v\n逆序推入=%v",
				entryKeys(want), entryKeys(got))
		}
	}
	// 字典序升序契约
	for i := 1; i < len(want); i++ {
		if entryKey(want[i-1]) >= entryKey(want[i]) {
			t.Fatalf("同分条目应按字典序升序，实得 %v", entryKeys(want))
		}
	}
}

// TestExcellentTable_同分边界入选与推入顺序无关 容量受限时，同分区间的
// 入选集合必须由全序决定（字典序最小者入选），与推入顺序无关
func TestExcellentTable_同分边界入选与推入顺序无关(t *testing.T) {
	keys := []string{"王金", "王春", "王秋", "王夏", "王冬"}
	entries := sameScoreEntries(t, keys, 80.0)

	tableA := NewExcellentTableWithCap(3)
	pushAll(tableA, entries)
	tableA.Finalize()

	tableB := NewExcellentTableWithCap(3)
	pushAll(tableB, reverse(entries))
	tableB.Finalize()

	if len(tableA.Entries(0, 10)) != 3 {
		t.Fatalf("容量 3 应保留 3 条，实得 %d", len(tableA.Entries(0, 10)))
	}
	if !reflect.DeepEqual(entryKeys(tableA.TopN(3)), entryKeys(tableB.TopN(3))) {
		t.Fatalf("同分边界入选随推入顺序波动：\n正序=%v\n逆序=%v",
			entryKeys(tableA.TopN(3)), entryKeys(tableB.TopN(3)))
	}
	// 入选的必须是字典序最小的三个
	want := entryKeys(tableA.TopN(3))
	sorted := append([]string{}, keys...)
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j-1] > sorted[j]; j-- {
			sorted[j-1], sorted[j] = sorted[j], sorted[j-1]
		}
	}
	for i, k := range sorted[:3] {
		if want[i] != k {
			t.Fatalf("同分边界应入选字典序最小三者：期望 %v，实得 %v", sorted[:3], want)
		}
	}
}

// TestExcellentTable_混合分数全序 分数不同走分数、同分走字典序
func TestExcellentTable_混合分数全序(t *testing.T) {
	table := NewExcellentTableWithCap(100)
	pushAll(table, []ExcellentEntry{
		{Char1: "张", Char2: "三", Score: 90},
		{Char1: "李", Char2: "四", Score: 90},
		{Char1: "王", Char2: "五", Score: 95},
		{Char1: "赵", Char2: "六", Score: 90},
	})
	table.Finalize()

	got := entryKeys(table.TopN(4))
	// 95 最高；90 同分按字典序：张(U+5F20) < 李(U+674E) < 赵(U+8D75)
	want := []string{"王五", "张三", "李四", "赵六"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("混合分数全序错误：期望 %v，实得 %v", want, got)
		}
	}
}

// TestExcellentTable_替换判断走全序 表满后同分但字典序更小的条目必须能进
func TestExcellentTable_替换判断走全序(t *testing.T) {
	table := NewExcellentTableWithCap(2)
	table.TryPush(ExcellentEntry{Char1: "王", Char2: "金", Score: 70})
	table.TryPush(ExcellentEntry{Char1: "王", Char2: "木", Score: 70})
	// 同分、字典序「王春」<「王金」，应挤掉王金
	table.TryPush(ExcellentEntry{Char1: "王", Char2: "春", Score: 70})
	table.Finalize()

	got := entryKeys(table.TopN(2))
	want := []string{"王春", "王木"} // 王金被字典序更小的王春替换
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("同分替换未走全序：期望 %v，实得 %v", want, got)
		}
	}
}

// TestExcellentTable_低分不得挤走同分高字典序 反向守护：同分且字典序更大
// 的条目不得替换堆顶（否则入选集合又随顺序波动）
func TestExcellentTable_低分不得挤走同分高字典序(t *testing.T) {
	table := NewExcellentTableWithCap(2)
	table.TryPush(ExcellentEntry{Char1: "王", Char2: "春", Score: 70})
	table.TryPush(ExcellentEntry{Char1: "王", Char2: "木", Score: 70})
	table.TryPush(ExcellentEntry{Char1: "王", Char2: "金", Score: 70}) // 同分字典序更大，不得进
	table.Finalize()

	got := entryKeys(table.TopN(2))
	want := []string{"王春", "王木"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("同分高字典序条目不应入选：期望 %v，实得 %v", want, got)
		}
	}
}

// entryKeys 取一组条目的 key 列表（错误报告用）
func entryKeys(entries []ExcellentEntry) []string {
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, entryKey(e))
	}
	return out
}
