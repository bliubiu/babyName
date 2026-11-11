package fate

import (
	"encoding/binary"
	"hash/fnv"
	"sync"

	"name/internal/domain/classics"
)

// BigramResult GetBigramScore 返回结构（用于 per-session / 全局缓存）
//
// 缓存键 = (char1, char2)，缓存值 = 完整返回三元组，
// 同一请求内同 (char1, char2) 只查一次 bigramIdx（O(1)）。
type BigramResult struct {
	Score      int
	SourceDesc string
	Found      bool
}

// SessionBigramCache per-session 二字共现评分缓存
//
// 数据竞争：sessionImpl 持有，generate() 阶段多 worker 并发读
// WenHuaRater/BigramRater 评分。
//
// 实现要点：
//   - 放在 fate 包（而非 classics 包）以避免循环依赖（fate 包已 import classics）。
//   - RLock 优先读缓存命中；Lock 在未命中时写入缓存（写不频繁）。
//   - GetBigramScore 全局函数仍保留作为底层，供缓存未命中时调用。
//   - 读穿策略：先查全局共享缓存（跨请求），未命中再查本 session 局部缓存，
//     再未命中才调底层并回填两层。全局缓存使 16 并发请求复用同一批常用字组合
//     （如「张」+常用字）的已算评分，避免每请求重复计算成千上万次
//     （docs/46 P2 并发：GetBigramScore 是双名 N² 笛卡尔积 hot path）。
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
// 同一 session 内 (a, b) 实际去重的独立对约 12.5 万次。
func (c *SessionBigramCache) GetOrCompute(char1, char2 string) BigramResult {
	if char1 == "" || char2 == "" || char1 == char2 {
		return BigramResult{}
	}
	a, b := char1, char2
	if a > b {
		a, b = b, a
	}
	key := BigramKey{A: a, B: b}

	// 1) 全局共享缓存（跨请求，纯函数结果，先到先得）
	if v, ok := globalBigramCache.get(key); ok {
		return v
	}
	// 2) 本 session 局部缓存
	c.mu.RLock()
	if v, ok := c.mem[key]; ok {
		c.mu.RUnlock()
		globalBigramCache.set(key, v) // 回填全局，惠及后续请求
		return v
	}
	c.mu.RUnlock()

	// 3) 未命中：调底层（仍走全局 RLock 但只查一次）
	score, desc, found := classics.GetBigramScore(char1, char2)
	result := BigramResult{Score: score, SourceDesc: desc, Found: found}

	// 写全局 + 局部
	globalBigramCache.set(key, result)
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

// Size 返回当前 session 局部缓存条目数（诊断/测试用）
func (c *SessionBigramCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.mem)
}

// --- 全局共享二字共现缓存（跨请求） ---
//
// GetBigramScore(char1, char2) 仅依赖静态典籍共现索引，与 fateData / 请求无关，
// 是纯函数，可安全跨请求共享。双名 N² 笛卡尔积中同一批高频字组合（如姓氏+常用字）
// 在并发请求间大量重复，全局缓存可把这部分计算从「每请求 O(N²)」降为「每组合一次」。
//
// 分片：64 个分片、每片独立 RWMutex，避免高并发下单一全局锁成为瓶颈；
// 每片软上限 4096（总容量约 26 万），超出时整片重建（分片化使抖动局部化、
// 不影响其它分片命中率）。bigram 键空间受候选池规模约束，26 万足以覆盖常态热点。

const bigramShardCount = 64

var globalBigramCache = newGlobalBigramCache(bigramShardCount, 1<<12)

type bigramShard struct {
	mu  sync.RWMutex
	m   map[BigramKey]BigramResult
	cap int
}

type globalBigramCacheT struct {
	shards []*bigramShard
}

func newGlobalBigramCache(n, perCap int) *globalBigramCacheT {
	g := &globalBigramCacheT{shards: make([]*bigramShard, n)}
	for i := range g.shards {
		g.shards[i] = &bigramShard{m: make(map[BigramKey]BigramResult, perCap), cap: perCap}
	}
	return g
}

// shardIndex 由键派生分片下标（FNV-1a，分片数为 2 的幂）
func (g *globalBigramCacheT) shardIndex(key BigramKey) uint32 {
	h := fnv.New32a()
	var buf [8]byte
	binary.LittleEndian.PutUint32(buf[:4], uint32(len(key.A)))
	h.Write(buf[:4])
	h.Write([]byte(key.A))
	binary.LittleEndian.PutUint32(buf[:4], uint32(len(key.B)))
	h.Write(buf[:4])
	h.Write([]byte(key.B))
	return h.Sum32() & uint32(len(g.shards)-1)
}

func (g *globalBigramCacheT) get(key BigramKey) (BigramResult, bool) {
	shard := g.shards[g.shardIndex(key)]
	shard.mu.RLock()
	v, ok := shard.m[key]
	shard.mu.RUnlock()
	return v, ok
}

func (g *globalBigramCacheT) set(key BigramKey, v BigramResult) {
	shard := g.shards[g.shardIndex(key)]
	shard.mu.Lock()
	if len(shard.m) >= shard.cap {
		// 软上限：整片重建，简单且抖动局部化
		shard.m = make(map[BigramKey]BigramResult, shard.cap)
	}
	shard.m[key] = v
	shard.mu.Unlock()
}

// globalBigramCacheSize 返回全局缓存条目总数（诊断用）
func globalBigramCacheSize() int {
	n := 0
	for _, s := range globalBigramCache.shards {
		s.mu.RLock()
		n += len(s.m)
		s.mu.RUnlock()
	}
	return n
}
