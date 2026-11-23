package cache

import (
	"testing"
	"time"
)

// TestMemoryCacheCapacity 容量上限回归：生成结果包含完整候选表，
// 必须按写入时间淘汰最旧项，避免 TTL 窗口内无界增长。
func TestMemoryCacheCapacity(t *testing.T) {
	c := NewMemoryCacheWithLimit(2)
	t.Cleanup(c.Close)

	c.Set("a", 1, time.Minute)
	c.Set("b", 2, time.Minute)
	c.Set("c", 3, time.Minute)

	if _, ok := c.Get("a"); ok {
		t.Fatal("容量 2 时最旧的 a 应被淘汰")
	}
	for _, k := range []string{"b", "c"} {
		if _, ok := c.Get(k); !ok {
			t.Errorf("最新写入的 %s 不应被淘汰", k)
		}
	}

	// 覆盖已有 key 不应触发淘汰
	c.Set("b", 20, time.Minute)
	if _, ok := c.Get("c"); !ok {
		t.Fatal("覆盖已有 key 不应淘汰其它项")
	}
}

func TestMemoryCacheUnlimited(t *testing.T) {
	c := NewMemoryCacheWithLimit(0)
	t.Cleanup(c.Close)
	for i := 0; i < 100; i++ {
		c.Set(string(rune('a'+i%26))+string(rune('0'+i/26)), i, time.Minute)
	}
	// 仅验证不 panic 且能写入；不限制容量时全部保留
	c.Set("last", "ok", time.Minute)
	if _, ok := c.Get("last"); !ok {
		t.Fatal("不限制容量时写入应命中")
	}
}
