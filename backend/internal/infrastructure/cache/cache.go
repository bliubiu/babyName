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
}

// MemoryCache 内存缓存实现
type MemoryCache struct {
	data   map[string]cacheItem
	mu     sync.RWMutex
	stopCh chan struct{}
}

// NewMemoryCache 创建内存缓存，启动过期清理协程
func NewMemoryCache() *MemoryCache {
	c := &MemoryCache{
		data:   make(map[string]cacheItem),
		stopCh: make(chan struct{}),
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

	c.data[key] = cacheItem{value: value, expiration: exp}
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
