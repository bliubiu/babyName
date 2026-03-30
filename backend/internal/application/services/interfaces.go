package services

import (
	"github.com/bliubiu/babyName/internal/domain/bazi"
	"github.com/bliubiu/babyName/internal/domain/name"
	"github.com/bliubiu/babyName/internal/domain/namestat"
	"github.com/bliubiu/babyName/internal/domain/yijing"
	"github.com/bliubiu/babyName/internal/domain/zodiac"
)

// NameServiceInterface 名字服务接口
type NameServiceInterface interface {
	Generate(req *GenerateRequest) (*GenerateResponse, error)
	GetByID(id int64) (*name.Name, error)
}

// BaziServiceInterface 八字服务接口
type BaziServiceInterface interface {
	Analyze(year, month, day, hour, minute int) (*bazi.BaziAnalysis, error)
}

// YijingServiceInterface 易经服务接口
type YijingServiceInterface interface {
	GetHexagram(id int) (*yijing.Hexagram, error)
	GetAllHexagrams() ([]yijing.Hexagram, error)
}

// ZodiacServiceInterface 生肖服务接口
type ZodiacServiceInterface interface {
	GetZodiac(animal string) (*zodiac.Zodiac, error)
	GetAllZodiacs() ([]zodiac.Zodiac, error)
}

// HistoryServiceInterface 历史记录服务接口
type HistoryServiceInterface interface {
	SaveHistory(record *HistoryRecord) (string, error)
	BatchSaveHistory(records []*HistoryRecord) ([]string, error)
	GetHistory() ([]*HistoryRecord, error)
	DeleteHistory(id string) error
}

// FavoriteServiceInterface 收藏服务接口
type FavoriteServiceInterface interface {
	GetFavorites() ([]*FavoriteRecord, error)
	SaveFavorite(record *FavoriteRecord) (string, error)
	BatchSaveFavorite(records []*FavoriteRecord) ([]string, error)
	DeleteFavorite(id string) error
	BatchDeleteFavorite(ids []string) error
	CheckFavorite(surname, givenName string) bool
}

// StatServiceInterface 统计服务接口
type StatServiceInterface interface {
	GetNameStats(name string) (*namestat.NameStat, error)
	GetProvinceStats(name, province string) (*namestat.NameStat, error)
}

// ReportServiceInterface 报告服务接口
type ReportServiceInterface interface {
	GeneratePDF(data interface{}) ([]byte, error)
	GenerateHTML(data interface{}) (string, error)
}
