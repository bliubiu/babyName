package cache

import (
	"testing"
	"time"
)

// newTestCache 创建测试用内存缓存，测试结束自动关闭清理协程
func newTestCache(t *testing.T) *MemoryCache {
	t.Helper()
	c := NewMemoryCache()
	t.Cleanup(c.Close)
	return c
}

// TestMemoryCacheSetGet 基本存取
func TestMemoryCacheSetGet(t *testing.T) {
	c := newTestCache(t)

	c.Set("k1", "v1", time.Minute)
	val, ok := c.Get("k1")
	if !ok {
		t.Fatal("Get(k1) 期望命中，实际未命中")
	}
	if val != "v1" {
		t.Errorf("Get(k1) 期望 v1，实际 %v", val)
	}

	// 未设置的 key
	if _, ok := c.Get("notexist"); ok {
		t.Error("Get(notexist) 期望未命中")
	}
}

// TestMemoryCacheOverwrite 覆盖已存在的 key
func TestMemoryCacheOverwrite(t *testing.T) {
	c := newTestCache(t)

	c.Set("k", 1, time.Minute)
	c.Set("k", 2, time.Minute)
	val, ok := c.Get("k")
	if !ok {
		t.Fatal("Get(k) 期望命中")
	}
	if val != 2 {
		t.Errorf("覆盖后期望 2，实际 %v", val)
	}
}

// TestMemoryCacheExpiration 过期项不命中（惰性删除）
func TestMemoryCacheExpiration(t *testing.T) {
	c := newTestCache(t)

	c.Set("short", "v", 50*time.Millisecond)
	if _, ok := c.Get("short"); !ok {
		t.Fatal("未过期前应命中")
	}
	time.Sleep(80 * time.Millisecond)
	if _, ok := c.Get("short"); ok {
		t.Error("过期后应未命中")
	}
}

// TestMemoryCacheZeroExpiration expiration=0 表示永不过期
func TestMemoryCacheZeroExpiration(t *testing.T) {
	c := newTestCache(t)

	c.Set("forever", "v", 0)
	time.Sleep(60 * time.Millisecond)
	if _, ok := c.Get("forever"); !ok {
		t.Error("expiration=0 应永不过期")
	}
}

// TestMemoryCacheDelete 删除 key
func TestMemoryCacheDelete(t *testing.T) {
	c := newTestCache(t)

	c.Set("k", "v", time.Minute)
	c.Delete("k")
	if _, ok := c.Get("k"); ok {
		t.Error("Delete 后应未命中")
	}

	// 删除不存在的 key 不应 panic
	c.Delete("notexist")
}

// TestMemoryCacheClear 清空所有缓存
func TestMemoryCacheClear(t *testing.T) {
	c := newTestCache(t)

	c.Set("k1", 1, time.Minute)
	c.Set("k2", 2, time.Minute)
	c.Clear()
	if _, ok := c.Get("k1"); ok {
		t.Error("Clear 后 k1 应未命中")
	}
	if _, ok := c.Get("k2"); ok {
		t.Error("Clear 后 k2 应未命中")
	}
}

// TestMemoryCacheNullMarker 空值标记防穿透
func TestMemoryCacheNullMarker(t *testing.T) {
	c := newTestCache(t)

	c.SetNull("nullkey", time.Minute)
	val, ok := c.Get("nullkey")
	if !ok {
		t.Fatal("SetNull 后应命中")
	}
	if !c.IsNull(val) {
		t.Error("IsNull 应识别 NullMarker")
	}

	// 普通值不应被识别为 NullMarker
	c.Set("normal", "v", time.Minute)
	v, _ := c.Get("normal")
	if c.IsNull(v) {
		t.Error("普通值不应被识别为 NullMarker")
	}
}

// TestMemoryCacheSetWithJitter 带抖动设置（验证不 panic 且命中）
func TestMemoryCacheSetWithJitter(t *testing.T) {
	c := newTestCache(t)

	c.SetWithJitter("k", "v", time.Minute, 0.2)
	if _, ok := c.Get("k"); !ok {
		t.Error("SetWithJitter 后应命中")
	}

	c.SetNullWithJitter("nk", time.Minute, 0.2)
	val, ok := c.Get("nk")
	if !ok || !c.IsNull(val) {
		t.Error("SetNullWithJitter 后应命中且为 NullMarker")
	}
}

// TestAddJitter 抖动范围合理
func TestAddJitter(t *testing.T) {
	base := time.Second
	// jitterPercent=0.2 时，抖动范围应在 ±20% 内
	for i := 0; i < 100; i++ {
		got := AddJitter(base, 0.2)
		if got < base*8/10 || got > base*12/10 {
			t.Errorf("AddJitter 超出 ±20%% 范围: %v", got)
		}
	}

	// 非法 jitterPercent 应回退到默认 0.2
	got := AddJitter(base, 0)
	if got == base {
		// 默认也有抖动，几乎不可能恰好等于 base；仅验证不 panic
		t.Log("AddJitter(base,0) 使用默认抖动")
	}
}

// TestGenerateCacheKey 缓存键生成
func TestGenerateCacheKey(t *testing.T) {
	k1 := GenerateCacheKey("prefix", "a", 1)
	k2 := GenerateCacheKey("prefix", "a", 1)
	k3 := GenerateCacheKey("prefix", "b", 1)

	if k1 != k2 {
		t.Error("相同输入应生成相同 key")
	}
	if k1 == k3 {
		t.Error("不同输入应生成不同 key")
	}
	if len(k1) == 0 {
		t.Error("key 不应为空")
	}
}

// TestGetTyped 类型安全获取
func TestGetTyped(t *testing.T) {
	c := newTestCache(t)

	c.Set("s", "hello", time.Minute)
	s, ok := GetTyped[string](c, "s")
	if !ok || s != "hello" {
		t.Errorf("GetTyped[string] 期望 hello ok=true，实际 %q %v", s, ok)
	}

	c.Set("n", 42, time.Minute)
	n, ok := GetTyped[int](c, "n")
	if !ok || n != 42 {
		t.Errorf("GetTyped[int] 期望 42 ok=true，实际 %d %v", n, ok)
	}

	// 类型不匹配
	_, ok = GetTyped[string](c, "n")
	if ok {
		t.Error("类型不匹配应返回 ok=false")
	}

	// 不存在
	_, ok = GetTyped[string](c, "notexist")
	if ok {
		t.Error("不存在的 key 应返回 ok=false")
	}
}

// TestGetObject 对象获取
func TestGetObject(t *testing.T) {
	c := newTestCache(t)

	type person struct{ Name string }
	p := &person{Name: "张三"}
	c.Set("p", p, time.Minute)

	got, ok := GetObject[person](c, "p")
	if !ok {
		t.Fatal("GetObject 期望命中")
	}
	if got.Name != "张三" {
		t.Errorf("Name 期望 张三，实际 %s", got.Name)
	}

	// 不存在
	_, ok = GetObject[person](c, "notexist")
	if ok {
		t.Error("不存在应返回 ok=false")
	}
}

// TestGetCacheSingleton GetCache 返回一致实例
func TestGetCacheSingleton(t *testing.T) {
	c1 := GetCache()
	c2 := GetCache()
	if c1 != c2 {
		t.Error("GetCache 应返回同一全局实例")
	}
	if c1 == nil {
		t.Error("GetCache 不应返回 nil")
	}
}
