package services

import (
	"context"

	"name/internal/domain/zodiac"
	"name/internal/infrastructure/database"
)

// ZodiacService 生肖服务
type ZodiacService struct {
	store database.ZodiacStore
}

// NewZodiacService 创建生肖服务
func NewZodiacService(store database.ZodiacStore) *ZodiacService {
	return &ZodiacService{store: store}
}

// GetZodiac 获取生肖
func (s *ZodiacService) GetZodiac(ctx context.Context, animal string) (*zodiac.Zodiac, error) {
	return s.store.GetZodiacByName(animal), nil
}

// GetAllZodiacs 获取所有生肖
func (s *ZodiacService) GetAllZodiacs(ctx context.Context) ([]zodiac.Zodiac, error) {
	return s.store.GetZodiacs(), nil
}
