package services

import (
	"context"
	"time"

	"name/internal/domain/bazi"
	"name/internal/infrastructure/cache"
	"name/internal/infrastructure/logger"

	"go.uber.org/zap"
)

// BaziService 八字服务
type BaziService struct {
	cache cache.Cache
}

// NewBaziService 创建八字服务
func NewBaziService(cacheInst cache.Cache) *BaziService {
	return &BaziService{cache: cacheInst}
}

// Analyze 分析八字
func (s *BaziService) Analyze(ctx context.Context, year, month, day, hour, minute int) (*bazi.BaziAnalysis, error) {
	// 生成缓存键
	cacheKey := cache.GenerateCacheKey("bazi", year, month, day, hour, minute)

	// 检查缓存（MemoryCache 存储）
	if analysis, found := cache.GetObject[bazi.BaziAnalysis](s.cache, cacheKey); found {
		logger.Info("Bazi analysis loaded from cache",
			zap.Int("year", year),
			zap.Int("month", month),
			zap.Int("day", day),
			zap.Int("hour", hour),
			zap.Int("minute", minute),
		)
		return analysis, nil
	}

	// 执行分析
	analysis, err := bazi.AnalyzeBazi(year, month, day, hour, minute)
	if err != nil {
		logger.Error("八字分析失败", zap.Error(err))
		// 失败时不缓存，避免空对象被缓存 24 小时；同时返回 error 给调用方
		return nil, err
	}

	// 缓存结果，有效期24小时
	s.cache.Set(cacheKey, analysis, 24*time.Hour)

	return analysis, nil
}
