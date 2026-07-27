package services

import (
	"context"
	"time"

	"name/internal/domain/yijing"
	"name/internal/infrastructure/cache"
	"name/internal/infrastructure/database"
	"name/internal/infrastructure/logger"

	"go.uber.org/zap"
)

// YijingService 易经服务
type YijingService struct {
	store database.YijingStore
	cache cache.Cache
}

// NewYijingService 创建易经服务
func NewYijingService(store database.YijingStore, cacheInst cache.Cache) *YijingService {
	return &YijingService{store: store, cache: cacheInst}
}

// GetHexagram 获取卦象
func (s *YijingService) GetHexagram(ctx context.Context, id int) (*yijing.Hexagram, error) {
	// 生成缓存键
	cacheKey := cache.GenerateCacheKey("hexagram", id)

	// 检查缓存（MemoryCache 存储）
	if hexagram, found := cache.GetObject[yijing.Hexagram](s.cache, cacheKey); found {
		logger.Info("Hexagram loaded from cache", zap.Int("id", id))
		return hexagram, nil
	}

	// 从存储中获取
	hexagram := s.store.GetHexagramByNumber(id)

	// 缓存结果，有效期7天
	if hexagram != nil {
		s.cache.Set(cacheKey, hexagram, 7*24*time.Hour)
	}

	return hexagram, nil
}

// GetAllHexagrams 获取所有卦象
func (s *YijingService) GetAllHexagrams(ctx context.Context) ([]yijing.Hexagram, error) {
	// 生成缓存键
	cacheKey := "hexagrams:all"

	// 检查缓存
	if hexagrams, found := cache.GetObject[[]yijing.Hexagram](s.cache, cacheKey); found {
		logger.Info("All hexagrams loaded from cache")
		return *hexagrams, nil
	}

	// 从存储中获取
	hexagrams := s.store.GetHexagrams()

	// 缓存结果，有效期7天
	s.cache.Set(cacheKey, hexagrams, 7*24*time.Hour)

	return hexagrams, nil
}
