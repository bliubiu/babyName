package namestatistics

import (
	"name/internal/infrastructure/database"
)

// NameStatisticsService 姓名统计领域服务接口
type NameStatisticsService interface {
	// GetSurnameStats 获取姓氏排行
	GetSurnameStats(limit int) ([]database.SurnameStat, error)

	// GetSurnameStat 获取单个姓氏统计
	GetSurnameStat(surname string) (*database.SurnameStat, error)

	// GetGivenNameStats 获取指定姓氏下的名字排行
	GetGivenNameStats(surname string, limit int) ([]database.GivenNameStat, error)

	// GetFullNameStats 获取指定姓氏下的全名排行
	GetFullNameStats(surname string, limit int) ([]database.FullNameStat, error)

	// GetFullNameStat 获取全名统计
	GetFullNameStat(fullName string) (*database.FullNameStat, error)

	// GetNameGenderStats 获取名字性别分布
	GetNameGenderStats(name string) (*database.NameGenderStat, error)

	// GetTopFullNames 获取热门全名排行
	GetTopFullNames(limit int) ([]database.FullNameStat, error)

	// GetTotalNameCount 获取语料库总人名数
	GetTotalNameCount() (int, error)
}
