package ziwei

import (
	"fmt"
	"sync"
	"testing"
)

// 本文件固化 docs/29 P3#22 的处置验证：ziweiMutex 全程持锁串行化纯计算。
// tyme 无可变共享状态（lazy-init 均 sync.Once 保护），紫微排盘可完全并行。

// TestCalculateZiweiChartConcurrentDeterminism 并发下同输入必须同输出
func TestCalculateZiweiChartConcurrentDeterminism(t *testing.T) {
	type in struct {
		y, m, d, h int
		gender     string
	}
	inputs := []in{
		{2000, 8, 16, 4, "女"},
		{2023, 1, 22, 10, "男"},
		{1985, 5, 13, 12, "女"},
		{1995, 10, 12, 20, "男"},
	}

	refs := make([]string, len(inputs))
	for i, c := range inputs {
		chart := CalculateZiweiChart(c.y, c.m, c.d, c.h, c.gender)
		if chart == nil {
			t.Fatalf("参考排盘失败 %v", c)
		}
		refs[i] = fmt.Sprintf("%s|%s|%s|%s", chart.MingGong, chart.FiveElements, chart.Soul, chart.Body)
	}

	const goroutines = 32
	var wg sync.WaitGroup
	errs := make(chan error, goroutines)
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i, c := range inputs {
				chart := CalculateZiweiChart(c.y, c.m, c.d, c.h, c.gender)
				got := fmt.Sprintf("%s|%s|%s|%s", chart.MingGong, chart.FiveElements, chart.Soul, chart.Body)
				if got != refs[i] {
					errs <- fmt.Errorf("g%d 并发结果与参考不符: got %s, want %s", g, got, refs[i])
					return
				}
			}
		}(g)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

// BenchmarkCalculateZiweiChartParallel16 16 goroutine 并发排盘吞吐观测点
// （移除 ziweiMutex 前被全局锁串行化）。
func BenchmarkCalculateZiweiChartParallel16(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for g := 0; g < 16; g++ {
			wg.Add(1)
			go func(g int) {
				defer wg.Done()
				_ = CalculateZiweiChart(1990+g%30, g%12+1, g%28+1, g%24, "男")
			}(g)
		}
		wg.Wait()
	}
}
