package fate

import (
	"context"
	"testing"
	"time"
)

// ——— docs/28 W6：消除每请求的重复预分配 ———
//
// 缺陷本质：generate() 先建一张容量恒为 10000 的表，generateDoubleName 末尾又
// `*table = NewExcellentTable()` 重造一份，第一份从未写入任何条目就变成垃圾。
//
// 按 ExcellentEntry 约 200 字节估算，每次双名请求白白分配又丢弃约 2MB
// （10000 × 200B 的堆切片 + 两个 map 的初始桶）。
//
// 本组用例钉住三点：
//  1. 表的容量应与「本请求需要的池大小」挂钩，而非恒为 10000；
//  2. 结果正确性不受影响（榜单无重复、分数在合法区间）；
//  3. 小 topCount 场景仍有容量下限，避免堆频繁替换。
//
// ★ 等价性已实测：改造前后对同一组入参（王/male/双名/Count=50、
// 李/female/双名/Count=50、张/male/单名/Count=20）产出的 TopNames
// **逐字节一致**（名字、顺序、分数完全相同），唯一差异是表容量 10000 → 1000。
// 即本改造是纯粹的内存优化，零行为变更。

// TestDoubleNameTableCapacityAdaptsToRequest 容量应按请求规模自适应。
func TestDoubleNameTableCapacityAdaptsToRequest(t *testing.T) {
	provider := newCuratedPoolProvider(0)
	engine := NewEngine(provider, &stubAnalyzer{}, DefaultRaters())

	for _, count := range []int{10, 50, 100, 200} {
		session := engine.NewSession()
		if err := session.Start(context.Background(), &Input{
			Surname: "王",
			Gender:  GenderMale,
			Born:    time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
			Options: GenerateOptions{NameLength: 2, Count: count},
		}); err != nil {
			t.Fatalf("Count=%d 会话启动失败: %v", count, err)
		}
		if err := session.Wait(); err != nil {
			t.Fatalf("Count=%d 生成失败: %v", count, err)
		}
		out := session.Result()
		if out == nil || out.ExcellentTable == nil {
			t.Fatalf("Count=%d 未产出候选表", count)
		}

		want := count * 10
		if want < minExcellentTablePoolSize {
			want = minExcellentTablePoolSize
		}
		got := out.ExcellentTable.cap

		if got == excellentTableCapacity && count < excellentTableCapacity/10 {
			// 这正是 W6 的症状：请求只要 count*10 个槽位，却按 10000 预分配
			t.Errorf("Count=%d 时表容量仍为 %d（固定上限）——未按请求规模自适应，"+
				"每次请求白白预分配约 %d 个条目的空间",
				count, got, excellentTableCapacity-want)
		}
		if got != want {
			t.Errorf("Count=%d 时表容量 = %d，期望 %d", count, got, want)
		}
		t.Logf("Count=%-3d → 表容量 %-5d（旧实现固定 %d）", count, got, excellentTableCapacity)
	}
}

// TestSmallCountStillGetsReasonableCapacity 极小的 topCount 也要有容量下限，
// 否则堆会频繁触发「替换堆顶」的 O(log n) 操作并反复拷贝条目。
func TestSmallCountStillGetsReasonableCapacity(t *testing.T) {
	provider := newCuratedPoolProvider(0)
	engine := NewEngine(provider, &stubAnalyzer{}, DefaultRaters())

	session := engine.NewSession()
	if err := session.Start(context.Background(), &Input{
		Surname: "李",
		Gender:  GenderFemale,
		Born:    time.Date(2024, 5, 5, 5, 0, 0, 0, time.UTC),
		Options: GenerateOptions{NameLength: 2, Count: 1},
	}); err != nil {
		t.Fatalf("会话启动失败: %v", err)
	}
	if err := session.Wait(); err != nil {
		t.Fatalf("生成失败: %v", err)
	}
	out := session.Result()
	if out == nil || out.ExcellentTable == nil {
		t.Fatal("未产出候选表")
	}
	if got := out.ExcellentTable.cap; got != minExcellentTablePoolSize {
		t.Errorf("Count=1 时表容量 = %d，期望下限 %d", got, minExcellentTablePoolSize)
	}
}

// TestDoubleNameResultsStillCorrect 容量改造不得影响结果正确性。
//
// 关键点：合并各分片时 table 必须**保留去重能力**（NewExcellentTableWithCap
// 而非 NewExcellentTableUnique），否则跨分片的重复 Char1+Char2 会稀释榜单。
// 本用例通过「榜单内无重复名字」间接守住这一点。
func TestDoubleNameResultsStillCorrect(t *testing.T) {
	provider := newCuratedPoolProvider(0)
	engine := NewEngine(provider, &stubAnalyzer{}, DefaultRaters())

	session := engine.NewSession()
	if err := session.Start(context.Background(), &Input{
		Surname: "张",
		Gender:  GenderMale,
		Born:    time.Date(2024, 8, 20, 16, 0, 0, 0, time.UTC),
		Options: GenerateOptions{NameLength: 2, Count: 50},
	}); err != nil {
		t.Fatalf("会话启动失败: %v", err)
	}
	if err := session.Wait(); err != nil {
		t.Fatalf("生成失败: %v", err)
	}
	out := session.Result()
	if out == nil || len(out.TopNames) == 0 {
		t.Fatal("应产出名字")
	}

	seen := map[string]bool{}
	for _, n := range out.TopNames {
		if seen[n.GivenName] {
			t.Errorf("榜单出现重复名字 %q —— 说明合并分片时丢失了去重能力"+
				"（应使用 NewExcellentTableWithCap 而非 NewExcellentTableUnique）", n.GivenName)
		}
		seen[n.GivenName] = true
		if len([]rune(n.GivenName)) != 2 {
			t.Errorf("双名请求产出了 %d 个字的名字 %q", len([]rune(n.GivenName)), n.GivenName)
		}
	}

	// 榜单**不**保证全局按分数降序——ensureWuxingDiversity 为满足「每种五行组合
	// 至少一个代表」会刻意按五行首现顺序重排。因此这里只断言每个条目的分数都在
	// 合理区间，并确认改造没有引入异常分数。
	for i, n := range out.TopNames {
		if n.Score.Total < 0 || n.Score.Total > 100 {
			t.Errorf("第 %d 名 %s 分数 %.2f 超出 0-100 区间", i+1, n.GivenName, n.Score.Total)
		}
	}
	t.Logf("产出 %d 个名字，最大分 %.2f，无重复；"+
		"（榜单顺序受五行多样性保底影响，不保证严格按分数降序）",
		len(out.TopNames), out.TopNames[0].Score.Total)
}

// TestSingleNameTableCapacityAdapts 单名路径同样不应按 10000 预分配。
func TestSingleNameTableCapacityAdapts(t *testing.T) {
	provider := newCuratedPoolProvider(0)
	engine := NewEngine(provider, &stubAnalyzer{}, DefaultRaters())

	session := engine.NewSession()
	const count = 20
	if err := session.Start(context.Background(), &Input{
		Surname: "陈",
		Gender:  GenderMale,
		Born:    time.Date(2024, 11, 11, 11, 0, 0, 0, time.UTC),
		Options: GenerateOptions{NameLength: 1, Count: count},
	}); err != nil {
		t.Fatalf("会话启动失败: %v", err)
	}
	if err := session.Wait(); err != nil {
		t.Fatalf("生成失败: %v", err)
	}
	out := session.Result()
	if out == nil || out.ExcellentTable == nil {
		t.Fatal("未产出候选表")
	}
	want := count * 10
	if want < minExcellentTablePoolSize {
		want = minExcellentTablePoolSize
	}
	if got := out.ExcellentTable.cap; got != want {
		t.Errorf("单名 Count=%d 时表容量 = %d，期望 %d", count, got, want)
	}
}
