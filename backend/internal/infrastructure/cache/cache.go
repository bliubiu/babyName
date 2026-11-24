package cache

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// NullMarker 空值标记，用于防止缓存穿透
type NullMarker struct{}

// GetTyped 类型安全的获取辅助函数
func GetTyped[T any](c Cache, key string) (T, bool) {
	var zero T
	val, ok := c.Get(key)
	if !ok {
		return zero, false
	}
	if typed, ok := val.(T); ok {
		return typed, true
	}
	return zero, false
}

// Cache 缓存接口
type Cache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, expiration time.Duration)
	Delete(key string)
	Clear()
	IsNull(value interface{}) bool
	SetNull(key string, expiration time.Duration)
}

// cacheItem 内存缓存项
type cacheItem struct {
	value      interface{}
	expiration time.Time
	seq        uint64 // 写入序号：Windows 下 time.Now 精度约 1ms，同一毫秒内连续写入会并列，必须用序号定序
}

const defaultMaxEntries = 64

// MemoryCache 内存缓存实现
type MemoryCache struct {
	data       map[string]cacheItem
	mu         sync.RWMutex
	stopCh     chan struct{}
	maxEntries int    // 0=不限；默认 64，按写入时间淘汰最旧项，防止生成结果无界增长
	nextSeq    uint64 // 单调递增的写入序号，保证淘汰顺序确定
}

// NewMemoryCache 创建内存缓存，启动过期清理协程
func NewMemoryCache() *MemoryCache {
	return NewMemoryCacheWithLimit(defaultMaxEntries)
}

// NewMemoryCacheWithLimit 创建带容量上限的内存缓存；maxEntries<=0 表示不限制。
func NewMemoryCacheWithLimit(maxEntries int) *MemoryCache {
	c := &MemoryCache{
		data:       make(map[string]cacheItem),
		stopCh:     make(chan struct{}),
		maxEntries: maxEntries,
	}
	go c.cleanupLoop()
	return c
}

// cleanupLoop 定期清理过期缓存
func (c *MemoryCache) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.mu.Lock()
			now := time.Now()
			for k, v := range c.data {
				if !v.expiration.IsZero() && now.After(v.expiration) {
					delete(c.data, k)
				}
			}
			c.mu.Unlock()
		case <-c.stopCh:
			return
		}
	}
}

// Close 停止清理协程
func (c *MemoryCache) Close() {
	close(c.stopCh)
}

var (
	globalCache Cache
	cacheMu     sync.RWMutex
)

// Init 初始化全局内存缓存（移除 sync.Once，允许重新初始化）
// 项目数据持久化策略为 SQLite + 内存缓存，不再使用 Redis。
func Init() {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	// 关闭旧的内存缓存
	if mc, ok := globalCache.(*MemoryCache); ok {
		mc.Close()
	}
	globalCache = NewMemoryCache()
}

// GetCache 获取缓存实例
func GetCache() Cache {
	cacheMu.RLock()
	if globalCache != nil {
		c := globalCache
		cacheMu.RUnlock()
		return c
	}
	cacheMu.RUnlock()

	// 未初始化时加写锁初始化
	cacheMu.Lock()
	defer cacheMu.Unlock()
	// double-check，避免并发场景下重复初始化
	if globalCache == nil {
		globalCache = NewMemoryCache()
	}
	return globalCache
}

// --- MemoryCache 实现 ---

func (c *MemoryCache) IsNull(value interface{}) bool {
	_, ok := value.(NullMarker)
	return ok
}

func (c *MemoryCache) SetNull(key string, expiration time.Duration) {
	c.Set(key, NullMarker{}, expiration)
}

func (c *MemoryCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, found := c.data[key]
	if !found {
		return nil, false
	}

	if !item.expiration.IsZero() && time.Now().After(item.expiration) {
		return nil, false
	}

	return item.value, true
}

func (c *MemoryCache) Set(key string, value interface{}, expiration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var exp time.Time
	if expiration > 0 {
		exp = time.Now().Add(expiration)
	}

	// 容量保护：先惰性清理过期项，仍超限时淘汰写入时间最早的项。
	// 生成结果缓存持有完整候选表，若不设上限，不同生辰/选项的请求会在
	// TTL 窗口内无界累积（桌面端可涨到数百 MB）。
	if c.maxEntries > 0 {
		if _, exists := c.data[key]; !exists && len(c.data) >= c.maxEntries {
			now := time.Now()
			for k, v := range c.data {
				if !v.expiration.IsZero() && now.After(v.expiration) {
					delete(c.data, k)
				}
			}
		}
		for c.maxEntries > 0 && len(c.data) >= c.maxEntries {
			if _, exists := c.data[key]; exists {
				break
			}
			oldestKey := ""
			var oldestSeq uint64
			for k, v := range c.data {
				if oldestKey == "" || v.seq < oldestSeq {
					oldestKey, oldestSeq = k, v.seq
				}
			}
			if oldestKey == "" {
				break
			}
			delete(c.data, oldestKey)
		}
	}

	c.nextSeq++
	c.data[key] = cacheItem{value: value, expiration: exp, seq: c.nextSeq}
}

func (c *MemoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
}

func (c *MemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = make(map[string]cacheItem)
}

// --- 工具函数 ---

// GenerateCacheKey 生成缓存键
func GenerateCacheKey(prefix string, data ...interface{}) string {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Sprintf("%s:%v", prefix, data)
	}
	hash := md5.Sum(jsonData)
	return fmt.Sprintf("%s:%x", prefix, hash)
}

// AddJitter 添加随机抖动时间，防止缓存雪崩
func AddJitter(baseTime time.Duration, jitterPercent float64) time.Duration {
	if jitterPercent <= 0 || jitterPercent >= 1 {
		jitterPercent = 0.2
	}
	jitter := float64(baseTime) * jitterPercent * (rand.Float64()*2 - 1)
	return baseTime + time.Duration(jitter)
}

// SetWithJitter 设置缓存并添加随机抖动时间
func (c *MemoryCache) SetWithJitter(key string, value interface{}, baseTime time.Duration, jitterPercent float64) {
	c.Set(key, value, AddJitter(baseTime, jitterPercent))
}

// SetNullWithJitter 设置空值标记并添加随机抖动时间
func (c *MemoryCache) SetNullWithJitter(key string, baseTime time.Duration, jitterPercent float64) {
	c.SetNull(key, AddJitter(baseTime, jitterPercent))
}

// GetObject 从缓存中获取对象（类型断言）
func GetObject[T any](c Cache, key string) (*T, bool) {
	val, ok := c.Get(key)
	if !ok {
		return nil, false
	}
	if obj, ok := val.(*T); ok {
		return obj, true
	}
	return nil, false
}
