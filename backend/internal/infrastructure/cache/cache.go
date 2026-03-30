package cache

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"context"
)

// NullMarker 空值标记，用于防止缓存穿透
type NullMarker struct{}

// Cache 缓存接口
type Cache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, expiration time.Duration)
	Delete(key string)
	Clear()
	IsNull(value interface{}) bool
	SetNull(key string, expiration time.Duration)
}

// MemoryCache 内存缓存实现
type MemoryCache struct {
	data map[string]cacheItem
	mu   sync.RWMutex // 添加读写锁，防止并发问题
}

type cacheItem struct {
	value      interface{}
	expiration time.Time
}

// RedisCache Redis分布式缓存实现
type RedisCache struct {
	client *redis.Client
	ctx    context.Context
}

// Config 缓存配置
type Config struct {
	Type     string // "memory" 或 "redis"
	RedisURL string // Redis连接URL
}

var (
	memoryCache *MemoryCache
	redisCache  *RedisCache
	once        sync.Once
	cacheConfig Config
)

// Init 初始化缓存
func Init(config Config) {
	cacheConfig = config
}

// GetCache 获取缓存实例
func GetCache() Cache {
	once.Do(func() {
		if cacheConfig.Type == "redis" && cacheConfig.RedisURL != "" {
			opt, err := redis.ParseURL(cacheConfig.RedisURL)
			if err != nil {
				memoryCache = &MemoryCache{
					data: make(map[string]cacheItem),
				}
				return
			}

			client := redis.NewClient(opt)
			ctx := context.Background()

			_, err = client.Ping(ctx).Result()
			if err != nil {
				memoryCache = &MemoryCache{
					data: make(map[string]cacheItem),
				}
				return
			}

			redisCache = &RedisCache{
				client: client,
				ctx:    ctx,
			}
		} else {
			memoryCache = &MemoryCache{
				data: make(map[string]cacheItem),
			}
		}
	})

	if redisCache != nil {
		return redisCache
	}
	return memoryCache
}

// IsNull 检查是否为空值标记
func (c *MemoryCache) IsNull(value interface{}) bool {
	_, ok := value.(NullMarker)
	return ok
}

// SetNull 设置空值标记（防止缓存穿透）
func (c *MemoryCache) SetNull(key string, expiration time.Duration) {
	c.Set(key, NullMarker{}, expiration)
}

// Get 获取缓存
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

// Set 设置缓存
func (c *MemoryCache) Set(key string, value interface{}, expiration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var exp time.Time
	if expiration > 0 {
		exp = time.Now().Add(expiration)
	}

	c.data[key] = cacheItem{
		value:      value,
		expiration: exp,
	}
}

// Delete 删除缓存
func (c *MemoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
}

// Clear 清空缓存
func (c *MemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = make(map[string]cacheItem)
}

// IsNull 检查是否为空值标记
func (c *RedisCache) IsNull(value interface{}) bool {
	_, ok := value.(NullMarker)
	return ok
}

// SetNull 设置空值标记（防止缓存穿透）
func (c *RedisCache) SetNull(key string, expiration time.Duration) {
	c.Set(key, NullMarker{}, expiration)
}

// Get 获取缓存（Redis实现）
func (c *RedisCache) Get(key string) (interface{}, bool) {
	val, err := c.client.Get(c.ctx, key).Result()
	if err != nil {
		return nil, false
	}

	var result interface{}
	err = json.Unmarshal([]byte(val), &result)
	if err != nil {
		return nil, false
	}

	return result, true
}

// Set 设置缓存（Redis实现）
func (c *RedisCache) Set(key string, value interface{}, expiration time.Duration) {
	data, err := json.Marshal(value)
	if err != nil {
		return
	}

	c.client.Set(c.ctx, key, data, expiration)
}

// Delete 删除缓存（Redis实现）
func (c *RedisCache) Delete(key string) {
	c.client.Del(c.ctx, key)
}

// Clear 清空缓存（Redis实现）
func (c *RedisCache) Clear() {
	c.client.FlushAll(c.ctx)
}

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
// baseTime: 基础过期时间
// jitterPercent: 抖动百分比（0-1），例如 0.2 表示 ±20%
func AddJitter(baseTime time.Duration, jitterPercent float64) time.Duration {
	if jitterPercent <= 0 || jitterPercent >= 1 {
		jitterPercent = 0.2 // 默认 20% 抖动
	}
	
	jitter := float64(baseTime) * jitterPercent * (rand.Float64()*2 - 1)
	return baseTime + time.Duration(jitter)
}

// SetWithJitter 设置缓存并添加随机抖动时间
func (c *MemoryCache) SetWithJitter(key string, value interface{}, baseTime time.Duration, jitterPercent float64) {
	expiration := AddJitter(baseTime, jitterPercent)
	c.Set(key, value, expiration)
}

// SetNullWithJitter 设置空值标记并添加随机抖动时间（防止缓存穿透）
func (c *MemoryCache) SetNullWithJitter(key string, baseTime time.Duration, jitterPercent float64) {
	expiration := AddJitter(baseTime, jitterPercent)
	c.SetNull(key, expiration)
}
