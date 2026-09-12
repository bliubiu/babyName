package namestatistics

import (
	"name/internal/infrastructure/database"
)

// nameStatisticsService 姓名统计服务实现
type nameStatisticsService struct {
	store database.NameStatStore
}

// NewNameStatisticsService 创建姓名统计服务
func NewNameStatisticsService(store database.NameStatStore) NameStatisticsService {
	return &nameStatisticsService{store: store}
}

func (s *nameStatisticsService) GetSurnameStats(limit int) ([]database.SurnameStat, error) {
	return s.store.GetSurnameStats(limit)
}

func (s *nameStatisticsService) GetSurnameStat(surname string) (*database.SurnameStat, error) {
	return s.store.GetSurnameStat(surname)
}

func (s *nameStatisticsService) GetGivenNameStats(surname string, limit int) ([]database.GivenNameStat, error) {
	return s.store.GetGivenNameStats(surname, limit)
}

func (s *nameStatisticsService) GetFullNameStats(surname string, limit int) ([]database.FullNameStat, error) {
	return s.store.GetFullNameStats(surname, limit)
}

func (s *nameStatisticsService) GetFullNameStat(fullName string) (*database.FullNameStat, error) {
	return s.store.GetFullNameStat(fullName)
}

func (s *nameStatisticsService) GetNameGenderStats(name string) (*database.NameGenderStat, error) {
	return s.store.GetNameGenderStats(name)
}

func (s *nameStatisticsService) GetTopFullNames(limit int) ([]database.FullNameStat, error) {
	return s.store.GetTopFullNames(limit)
}

func (s *nameStatisticsService) GetTotalNameCount() (int, error) {
	return s.store.GetTotalNameCount()
}
