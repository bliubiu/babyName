package services

import (
	"context"

	"name/internal/domain/namestat"
)

// StatService 统计服务
type StatService struct{}

// NewStatService 创建统计服务
func NewStatService() *StatService {
	return &StatService{}
}

// GetNameStats 获取名字统计信息
func (s *StatService) GetNameStats(ctx context.Context, name string) (*namestat.NameStat, error) {
	return namestat.GetNameStats(name), nil
}

// GetProvinceStats 获取省份名字统计
func (s *StatService) GetProvinceStats(ctx context.Context, name, province string) (*namestat.NameStat, error) {
	stats := namestat.GetProvinceNameStats(name, province)
	return stats, nil
}
