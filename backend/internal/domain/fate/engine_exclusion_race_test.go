package fate

import (
	"context"
	"sync"
	"testing"
	"time"
)

// ——— 负面反馈排除集（docs/28 W4）的两条独立性质 ———
//
// 1) 竞态安全：读路径与写路径并发时不得产生数据竞争
// 2) 时序契约：ExcludeChar 必须在 Start 之前调用（或至少不晚于所关心的那次 generate）
//
// 两条性质的测试分开写，避免「修了竞态却以为语义也修了」这类误判。
//
// 读路径实现演进（与本文件断言无关，但影响性能结论）：
//   无锁读（有 bug）→ RWMutex 逐次 RLock（正确但热路径开销大）→
//   atomic.Pointer 不可变快照（当前实现，正确且无锁）。

// TestSessionExclusionConcurrentAccess 竞态安全：
// 在生成进行中并发调用排除接口，不应触发 Go 的
// "concurrent map read and map write"（该错误不可 recover，会直接终止进程）。
//
// 历史缺陷：ExcludeChar/ExcludeCombo/ClearExclusions 写路径有 s.mu 保护，
// 而 isCharExcluded/isComboExcluded 读路径完全不加锁（且注释谎称
// 「并发安全由 s.mu 守护」——generate() 实际全程不持有 s.mu）。
//
// 注：本机无 gcc，`go test -race` 不可用（-race 需 cgo），故本用例依赖
// 运行时自身的 map 并发检测 + 结果自洽断言。
func TestSessionExclusionConcurrentAccess(t *testing.T) {
	provider := newCuratedPoolProvider(0)
	engine := NewEngine(provider, &stubAnalyzer{}, DefaultRaters())

	session := engine.NewSession()
	// 刻意在 Start 之前写一次，确保生成期间读路径确实会命中非空排除集
	session.ExcludeChar("珀")

	if err := session.Start(context.Background(), &Input{
		Surname: "王",
		Gender:  GenderMale,
		Born:    time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
		Options: GenerateOptions{NameLength: 2, Count: 100},
	}); err != nil {
		t.Fatalf("会话启动失败: %v", err)
	}

	// 生成进行中：写路径与读路径并发
	var wg sync.WaitGroup
	stop := make(chan struct{})
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				session.ExcludeChar("沅")
				session.ExcludeCombo("沅", "浦")
				_ = session.ExcludedChars()
				if n%2 == 0 {
					session.ClearExclusions()
				}
				time.Sleep(time.Millisecond)
			}
		}(i)
	}

	if err := session.Wait(); err != nil {
		t.Logf("生成本身返回错误（不影响竞态断言）: %v", err)
	}
	close(stop)
	wg.Wait()

	// 并发写结束后，读路径应给出确定、自洽的结果
	session.ClearExclusions()
	session.ExcludeChar("测试字")
	got := session.ExcludedChars()
	if len(got) != 1 || got[0] != "测试字" {
		t.Fatalf("并发写后排除集状态不一致：期望 [测试字]，实际 %v", got)
	}
}

// TestSessionExclusionAppliedWhenSetBeforeStart 时序契约的正向用例：
// 按契约在 Start 之前调用 ExcludeChar，被排除的字不得出现在结果中。
func TestSessionExclusionAppliedWhenSetBeforeStart(t *testing.T) {
	provider := newCuratedPoolProvider(0)
	engine := NewEngine(provider, &stubAnalyzer{}, DefaultRaters())

	input := &Input{
		Surname: "王",
		Gender:  GenderMale,
		Born:    time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
		Options: GenerateOptions{NameLength: 2, Count: 50},
	}

	// 基线：取一个真实出现在榜单里的字作为排除对象
	base := engine.NewSession()
	if err := base.Start(context.Background(), input); err != nil {
		t.Fatalf("基线会话启动失败: %v", err)
	}
	if err := base.Wait(); err != nil {
		t.Fatalf("基线生成失败: %v", err)
	}
	out := base.Result()
	if out == nil || len(out.TopNames) == 0 {
		t.Fatal("基线应产出名字")
	}
	target := string([]rune(out.TopNames[0].GivenName)[0])
	t.Logf("基线首名 = %s，取首字 %q 作为排除对象", out.TopNames[0].GivenName, target)

	// 契约用法：先 ExcludeChar，再 Start
	session := engine.NewSession()
	session.ExcludeChar(target)
	if err := session.Start(context.Background(), input); err != nil {
		t.Fatalf("排除后会话启动失败: %v", err)
	}
	if err := session.Wait(); err != nil {
		t.Fatalf("排除后生成失败: %v", err)
	}
	res := session.Result()
	if res == nil || len(res.TopNames) == 0 {
		t.Fatal("排除后仍应产出名字（不应把池子清空）")
	}
	for _, n := range res.TopNames {
		for _, r := range n.GivenName {
			if string(r) == target {
				t.Fatalf("排除失效：结果 %s 仍包含被排除的字 %q", n.GivenName, target)
			}
		}
	}
	t.Logf("排除字 %q 生效，%d 个结果均不含该字", target, len(res.TopNames))

	// ClearExclusions 后排除集应清空
	session.ClearExclusions()
	if got := session.ExcludedChars(); len(got) != 0 {
		t.Fatalf("ClearExclusions 后应为空，实际 %v", got)
	}
}

// TestSessionExclusionOnlyAffectsNextGenerate 时序契约的反向用例：
// 在生成完成后再改排除集，不应回溯影响已产出的结果；
// 排除集的作用范围是「下一次 generate」，不是当前结果。
func TestSessionExclusionOnlyAffectsNextGenerate(t *testing.T) {
	provider := newCuratedPoolProvider(0)
	engine := NewEngine(provider, &stubAnalyzer{}, DefaultRaters())

	session := engine.NewSession()
	input := &Input{
		Surname: "王",
		Gender:  GenderMale,
		Born:    time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
		Options: GenerateOptions{NameLength: 2, Count: 50},
	}
	if err := session.Start(context.Background(), input); err != nil {
		t.Fatalf("会话启动失败: %v", err)
	}
	if err := session.Wait(); err != nil {
		t.Fatalf("生成失败: %v", err)
	}
	before := session.Result()
	if before == nil || len(before.TopNames) == 0 {
		t.Fatal("应产出名字")
	}
	snapshot := before.TopNames[0].GivenName

	// 生成结束后再排除：不得改变已产出的结果对象
	session.ExcludeChar(string([]rune(snapshot)[0]))
	after := session.Result()
	if after == nil || len(after.TopNames) == 0 {
		t.Fatal("Result() 不应因 ExcludeChar 而变空")
	}
	if after.TopNames[0].GivenName != snapshot {
		t.Fatalf("生成后调用 ExcludeChar 不应回溯改变已有结果：期望 %s，实际 %s",
			snapshot, after.TopNames[0].GivenName)
	}
}

// ——— 读路径微基准 ———
//
// isComboExcluded 位于双名 N×N 笛卡尔积的最内层循环，用真实数据规模（~3700 字
// 收窄后池）算，单次生成调用量在千万次量级；因此哪怕每次只多一次原子计数，
// 累积起来也会被观测到（这正是首版 RWMutex 修复导致串行吞吐下降约 20% 的原因）。
//
// 本基准以「无排除项」与「有 8 项排除」两种形态各跑一轮，用于：
//   - 回归护栏：若有人把实现改回逐次加锁，ns/op 会立刻抬升；
//   - 说明「空排除集」是绝大多数线上请求的真实形态（会话未做任何排除），
//     该路径应几乎零成本。

func BenchmarkIsComboExcludedEmpty(b *testing.B) {
	s := &sessionImpl{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if s.isComboExcluded("文", "轩") {
			b.Fatal("空排除集不应命中")
		}
	}
}

func BenchmarkIsComboExcludedPopulated(b *testing.B) {
	s := &sessionImpl{}
	for _, ch := range []string{"甲", "乙", "丙", "丁", "戊", "己", "庚", "辛"} {
		s.ExcludeChar(ch)
	}
	s.ExcludeCombo("壬", "癸")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if s.isComboExcluded("文", "轩") {
			b.Fatal("未排除的组合不应命中")
		}
	}
}

func BenchmarkIsComboExcludedHit(b *testing.B) {
	s := &sessionImpl{}
	s.ExcludeCombo("壬", "癸")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !s.isComboExcluded("癸", "壬") {
			b.Fatal("已排除组合（乱序传入）应命中")
		}
	}
}
