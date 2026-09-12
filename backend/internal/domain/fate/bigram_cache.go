package fate

import (
	"sync"

	"name/internal/domain/classics"
)

// BigramResult GetBigramScore 返回结构（用于 per-session 缓存）
//
// 缓存键 = (char1, char2)，缓存值 = 完整返回三元组，
// 同一 session 内同 (char1, char2) 只查一次 bigramIdx（O(1)）。
// 在 N×N 双名笛卡尔积中节省 50 万次 RLock + map 查询。
type BigramResult struct {
	Score      int
	SourceDesc string
	Found      bool
}

// SessionBigramCache per-session 二字共现评分缓存
//
// 数据竞争：sessionImpl 持有，generate() 阶段多 worker 并发读
// WenHuaRater/BigramRater 评分。
// 缓存仅在 generate() 阶段写入，session 结束后随 sessionImpl 释放。
//
// 实现要点：
//   - 放在 fate 包（而非 classics 包）以避免循环依赖（fate 包已 import classics）。
//   - RLock 优先读缓存命中；Lock 在未命中时写入缓存（写不频繁）。
//   - GetBigramScore 全局函数仍保留作为底层，供缓存未命中时调用。
type SessionBigramCache struct {
	mu  sync.RWMutex
	mem map[BigramKey]BigramResult
}

// BigramKey (char1, char2) 排序后的查询键
type BigramKey struct {
	A string // 排序后的较小字
	B string // 排序后的较大字
}

// newSessionBigramCache 创建 session 级缓存
func newSessionBigramCache() *SessionBigramCache {
	return &SessionBigramCache{mem: make(map[BigramKey]BigramResult, 4096)}
}

// GetOrCompute 优先读缓存，未命中时调底层 GetBigramScore 并写入缓存
//
// 这是 hot path：500×500 = 25 万次调用（双名笛卡尔积），
// 同一 session 内 (a, b) 实际去重的独立对约 12.5 万次，
// 缓存命中率预期 ~50%（双名 i≠j，N²/(N² - N) ≈ 1）。
func (c *SessionBigramCache) GetOrCompute(char1, char2 string) BigramResult {
	if char1 == "" || char2 == "" || char1 == char2 {
		return BigramResult{}
	}
	a, b := char1, char2
	if a > b {
		a, b = b, a
	}
	key := BigramKey{A: a, B: b}

	// 读缓存（RLock 几乎无竞争）
	c.mu.RLock()
	if v, ok := c.mem[key]; ok {
		c.mu.RUnlock()
		return v
	}
	c.mu.RUnlock()

	// 未命中：调底层（仍走全局 RLock 但只查一次）
	score, desc, found := classics.GetBigramScore(char1, char2)
	result := BigramResult{Score: score, SourceDesc: desc, Found: found}

	// 写缓存
	c.mu.Lock()
	c.mem[key] = result
	c.mu.Unlock()
	return result
}

// Clear 清空缓存（保留容量），session 复用时调用
func (c *SessionBigramCache) Clear() {
	c.mu.Lock()
	for k := range c.mem {
		delete(c.mem, k)
	}
	c.mu.Unlock()
}

// Size 返回当前缓存条目数（诊断/测试用）
func (c *SessionBigramCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.mem)
}
