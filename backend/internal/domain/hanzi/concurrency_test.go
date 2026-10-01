package hanzi

// concurrency_test.go — 全局字库的并发读写（docs/29 B9）
//
// B9 的原始病灶：ApplyWuxingOverrides 与 mergeKangxiStrokes 在写全局
// HanziData 时完全不加锁 —— mergeKangxiStrokes 的注释甚至声称「调用方已持有
// mu 写锁」，但调用点在 mu.Unlock() 之后。写侧无锁、读侧（IsCommonChar /
// HetuWuxingOfChar / GetNamingCategories 等）也无锁，一旦热更新接通，
// 并发读写直接 fatal error: concurrent map read and map write。
//
// 本环境无 gcc，-race 不可用；好在 Go 运行时的 map 并发检测是内建的，
// 无锁并发读写必然被直接判为 fatal error，故并发压测足以守住这条不变量。

import (
	"sync"
	"testing"
)

// TestApplyWuxingOverrides_ConcurrentWithReaders 写侧加锁后与读侧并发不得崩溃
func TestApplyWuxingOverrides_ConcurrentWithReaders(t *testing.T) {
	if len(HanziData) == 0 {
		t.Skip("字库未加载，跳过并发测试")
	}

	// 采样一批真实存在的字，供读侧循环查询
	chars := make([]string, 0, 256)
	for c := range HanziData {
		chars = append(chars, c)
		if len(chars) >= 256 {
			break
		}
	}

	// ★ 关键前置：把「会被真正改写」的那些字的五行清空。
	// ApplyWuxingOverrides 只在值确实需要变化时才写 map；首次执行后所有值
	// 都已就位，后续调用是纯读 —— 那样即使去掉锁也测不出并发写，
	// 测试就成了摆设（已实测：去掉锁后用例依然 PASS）。
	// 清空覆盖表命中的字的五行，保证每次调用都产生真实 map 写入。
	mu.Lock()
	for _, c := range chars {
		if _, ok := CharacterWuxingOverride[c]; ok {
			h := HanziData[c]
			h.Wuxing = ""
			HanziData[c] = h
		}
	}
	mu.Unlock()

	var wg sync.WaitGroup

	// 读侧：全部走公开 getter（即修复后的加锁读路径）
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := 0; n < 40; n++ {
				for _, c := range chars {
					_ = IsCommonChar(c)
					_ = HetuWuxingOfChar(c)
					_ = GetNamingCategories(c)
					_, _ = GetHanzi(c)
				}
			}
		}()
	}

	// 写侧：五行覆盖（幂等，重复执行不会改变最终状态）
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := 0; n < 5; n++ {
				ApplyWuxingOverrides()
			}
		}()
	}

	wg.Wait()

	// 覆盖表优先级必须仍然生效（加锁不能破坏业务语义）
	for _, c := range chars {
		if want, ok := CharacterWuxingOverride[c]; ok {
			if got := GetCharacterWuxing(c); got != want {
				t.Fatalf("字 %q 覆盖表应为 %q，实际 %q（加锁改造破坏了覆盖优先级）", c, want, got)
			}
		}
	}
}

// TestMergeKangxiStrokes_ConcurrentWithReaders 康熙笔画合并与读侧并发不得崩溃
func TestMergeKangxiStrokes_ConcurrentWithReaders(t *testing.T) {
	if len(NamerCharMap) == 0 {
		t.Skip("namer 字库未加载，跳过并发测试")
	}
	if !IsKangxiStrokesLoaded() {
		t.Skip("康熙笔画数据未加载，跳过并发测试")
	}

	chars := make([]string, 0, 256)
	for c := range NamerCharMap {
		chars = append(chars, c)
		if len(chars) >= 256 {
			break
		}
	}

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := 0; n < 40; n++ {
				for _, c := range chars {
					_ = GetCharRadicalFromNamer(c)
					_ = GetNamerLevel(c)
					_, _ = GetHanzi(c)
				}
			}
		}()
	}

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := 0; n < 5; n++ {
				// 复刻生产调用方式：mergeKangxiStrokes 会写 NamerCharMap，
				// 锁约定要求调用方先持有 namerMu 写锁（生产里由 LoadNamerFromJSON 持有）。
				// 不持锁直接调用会违反契约、测试本身就成了并发缺陷。
				namerMu.Lock()
				mergeKangxiStrokes()
				namerMu.Unlock()
			}
		}()
	}
	wg.Wait()

	// 合并后仍应能读到康熙笔画（加锁不能吞掉注入结果）
	var verified int
	for _, c := range chars {
		if nc, ok := GetNamerChar(c); ok && nc.KangxiStrokes > 0 {
			if h, ok := GetHanzi(c); !ok || h.KangxiStrokes != nc.KangxiStrokes {
				t.Fatalf("字 %q 的康熙笔画未同步到 HanziData：namer=%d hanzi=%d",
					c, nc.KangxiStrokes, h.KangxiStrokes)
			}
			verified++
		}
	}
	if verified == 0 {
		t.Error("未验证到任何康熙笔画数据，测试前提不成立")
	}
}

// TestLoadFromJSON_FailureLeavesDataIntact 加载失败不得留下半截数据
//
// 旧实现先快照、失败后回滚，但只补回旧键、不删除本轮新增的键；
// 且快照/回滚全程无锁。修复后所有可能失败的步骤都在改动全局之前完成。
func TestLoadFromJSON_FailureLeavesDataIntact(t *testing.T) {
	if len(HanziData) == 0 {
		t.Skip("字库未加载，跳过")
	}
	before := len(HanziData)
	probe, ok := GetHanzi("王")
	if !ok {
		t.Skip("字库缺少探针字「王」")
	}

	// 目录不存在 → 读文件即失败
	if err := LoadFromJSON(t.TempDir() + "/nonexistent"); err == nil {
		t.Fatal("目录不存在时 LoadFromJSON 应返回错误")
	}

	if after := len(HanziData); after != before {
		t.Errorf("加载失败后字库规模变了：before=%d after=%d（失败路径污染了全局数据）", before, after)
	}
	if now, ok := GetHanzi("王"); !ok {
		t.Error("加载失败后探针字丢失")
	} else if now.Char != probe.Char || now.Strokes != probe.Strokes || now.Wuxing != probe.Wuxing {
		t.Error("加载失败后已有字数据被改动")
	}
}