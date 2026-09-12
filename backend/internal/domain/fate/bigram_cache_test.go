package fate

import (
	"testing"
)

// TestSessionBigramCacheGetOrCompute 缓存命中/未命中
//
// 验证：
//   - 同 (a, b) 第二次调用走缓存（结果完全一致）
//   - 排序键无关性：(a, b) 与 (b, a) 命中同一键
//   - 空字符/同字返回零值（不走缓存）
func TestSessionBigramCacheGetOrCompute(t *testing.T) {
	cache := newSessionBigramCache()
	if cache.Size() != 0 {
		t.Errorf("新缓存 Size = %d, 期望 0", cache.Size())
	}

	// 第一次查询（未命中 → 写入缓存）
	r1 := cache.GetOrCompute("子", "轩")
	if r1.Score < 0 {
		t.Errorf("首次查询期望 Score >= 0, 实际 %d", r1.Score)
	}
	if cache.Size() != 1 {
		t.Errorf("查询后 Size = %d, 期望 1", cache.Size())
	}

	// 第二次查询（命中缓存）
	r2 := cache.GetOrCompute("子", "轩")
	if r2 != r1 {
		t.Errorf("缓存命中应返回相同结果：r1=%+v, r2=%+v", r1, r2)
	}
	if cache.Size() != 1 {
		t.Errorf("命中不应新增缓存：Size = %d", cache.Size())
	}

	// 顺序无关：(a, b) 与 (b, a) 命中同一键
	r3 := cache.GetOrCompute("轩", "子")
	if r3 != r1 {
		t.Errorf("反序应命中同一缓存：r1=%+v, r3=%+v", r1, r3)
	}
	if cache.Size() != 1 {
		t.Errorf("反序不应新增：Size = %d", cache.Size())
	}
}

// TestSessionBigramCacheEmptyAndSameChar 空输入与同字不入缓存
func TestSessionBigramCacheEmptyAndSameChar(t *testing.T) {
	cache := newSessionBigramCache()

	// 空字符
	if r := cache.GetOrCompute("", "子"); r.Found {
		t.Errorf("空字符串应返回 Found=false")
	}
	if r := cache.GetOrCompute("子", ""); r.Found {
		t.Errorf("空字符串应返回 Found=false")
	}
	if cache.Size() != 0 {
		t.Errorf("空输入不应写缓存：Size = %d", cache.Size())
	}

	// 同字
	if r := cache.GetOrCompute("子", "子"); r.Found {
		t.Errorf("同字应返回 Found=false")
	}
	if cache.Size() != 0 {
		t.Errorf("同字不应写缓存：Size = %d", cache.Size())
	}
}

// TestSessionBigramCacheClear 清空缓存
func TestSessionBigramCacheClear(t *testing.T) {
	cache := newSessionBigramCache()
	_ = cache.GetOrCompute("王", "李")
	_ = cache.GetOrCompute("张", "李")
	if cache.Size() != 2 {
		t.Errorf("查询后 Size = %d, 期望 2", cache.Size())
	}
	cache.Clear()
	if cache.Size() != 0 {
		t.Errorf("Clear 后 Size = %d, 期望 0", cache.Size())
	}
}

// TestSessionBigramCacheFoundFalseUnfoundCombo 缓存对未找到的组合仍写入
//
// 未命中（如随机生僻字组合）也写入 Found=false 的结果，
// 避免重复穿透到底层（仍走慢路径但省 RLock）。
func TestSessionBigramCacheFoundFalseUnfoundCombo(t *testing.T) {
	cache := newSessionBigramCache()
	r1 := cache.GetOrCompute("㐬", "䶮") // 极冷僻组合，不在 bigramIdx 中
	if r1.Found {
		t.Errorf("极冷僻组合应 Found=false, 实际 %+v", r1)
	}
	if cache.Size() != 1 {
		t.Errorf("未命中应仍写入缓存（避免重复穿透）：Size = %d", cache.Size())
	}
	// 二次查询应命中 Found=false 的缓存
	r2 := cache.GetOrCompute("㐬", "䶮")
	if r2.Found || r2.Score != r1.Score || r2.SourceDesc != r1.SourceDesc {
		t.Errorf("二次查询应命中 Found=false 缓存：r1=%+v, r2=%+v", r1, r2)
	}
}