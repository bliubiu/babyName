package bazi

import (
	"fmt"
	"sync"
	"testing"
)

// 本文件固化 docs/29 P3#22（全局互斥锁串行化纯计算）的处置验证：
// tyme 库无可变共享状态（唯一两处 lazy-init 的 LunarYearLeap /
// RabByungMonthDays 均由 sync.Once 保护，初始化后只读），
// 因此 CalculateBazi / AnalyzeBazi / CalculateZiweiChart 可完全并行，
// 无需 baziCalcMutex / ziweiMutex。

// refBaziInputs 固定输入集（覆盖闰月/晚子时/边界日期）
func refBaziInputs() [][5]int {
	return [][5]int{
		{2024, 8, 16, 4, 0},
		{2023, 1, 22, 10, 30},
		{2000, 2, 29, 23, 59}, // 闰日 + 晚子时
		{1985, 5, 13, 12, 0},
		{1950, 6, 1, 0, 1},
	}
}

// TestCalculateBaziConcurrentDeterminism 并发下同输入必须同输出（防共享缓冲串扰）
func TestCalculateBaziConcurrentDeterminism(t *testing.T) {
	inputs := refBaziInputs()

	// 单线程参考值
	refs := make([]string, len(inputs))
	for i, in := range inputs {
		b, err := CalculateBazi(in[0], in[1], in[2], in[3], in[4])
		if err != nil {
			t.Fatalf("参考计算失败 %v: %v", in, err)
		}
		refs[i] = fmt.Sprintf("%s|%s|%s|%s", b.YearGanzhi, b.MonthGanzhi, b.DayGanzhi, b.HourGanzhi)
	}

	const goroutines = 32
	var wg sync.WaitGroup
	errs := make(chan error, goroutines)
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i, in := range inputs {
				b, err := CalculateBazi(in[0], in[1], in[2], in[3], in[4])
				if err != nil {
					errs <- fmt.Errorf("g%d %v: %v", g, in, err)
					return
				}
				got := fmt.Sprintf("%s|%s|%s|%s", b.YearGanzhi, b.MonthGanzhi, b.DayGanzhi, b.HourGanzhi)
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

// BenchmarkCalculateBaziParallel16 16 goroutine 并发排盘吞吐观测点。
// 移除 baziCalcMutex 前该基准被全局锁串行化（耗时≈16×单次）；
// 移除后应接近 min(16×单次/NumCPU, 单次)（受核数限制）。
func BenchmarkCalculateBaziParallel16(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for g := 0; g < 16; g++ {
			wg.Add(1)
			go func(g int) {
				defer wg.Done()
				year := 1990 + g%30
				_, _ = CalculateBazi(year, g%12+1, g%28+1, g%24, 0)
			}(g)
		}
		wg.Wait()
	}
}
