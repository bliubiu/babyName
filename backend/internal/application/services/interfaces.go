package services

import (
	"context"

	"name/internal/domain/bazi"
	"name/internal/domain/name"
	"name/internal/domain/yijing"
	"name/internal/domain/zodiac"
	"name/internal/infrastructure/database"
)

// --- 服务接口定义 ---
// 所有服务方法首参为 ctx context.Context，支持取消信号传播与超时控制

// NameServiceInterface 名字服务接口
type NameServiceInterface interface {
	Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error)
	GenerateWithAnalysis(ctx context.Context, req *GenerateRequest) (*GenerateWithAnalysisResponse, error)
	GetByID(ctx context.Context, id int64) (*name.Name, error)
}

// BaziServiceInterface 八字服务接口
type BaziServiceInterface interface {
	Analyze(ctx context.Context, year, month, day, hour, minute int) (*bazi.BaziAnalysis, error)
}

// YijingServiceInterface 易经服务接口
type YijingServiceInterface interface {
	GetHexagram(ctx context.Context, id int) (*yijing.Hexagram, error)
	GetAllHexagrams(ctx context.Context) ([]yijing.Hexagram, error)
}

// ZodiacServiceInterface 生肖服务接口
type ZodiacServiceInterface interface {
	GetZodiac(ctx context.Context, animal string) (*zodiac.Zodiac, error)
	GetAllZodiacs(ctx context.Context) ([]zodiac.Zodiac, error)
}

// HistoryServiceInterface 历史记录服务接口
type HistoryServiceInterface interface {
	SaveHistory(ctx context.Context, record *HistoryRecord) (string, error)
	BatchSaveHistory(ctx context.Context, records []*HistoryRecord) ([]string, error)
	GetHistory(ctx context.Context) ([]*HistoryRecord, error)
	GetHistoryPage(ctx context.Context, page, limit int) ([]*HistoryRecord, int, error)
	DeleteHistory(ctx context.Context, id string) error
}

// FavoriteServiceInterface 收藏服务接口
type FavoriteServiceInterface interface {
	GetFavorites(ctx context.Context) ([]*FavoriteRecord, error)
	SaveFavorite(ctx context.Context, record *FavoriteRecord) (string, error)
	BatchSaveFavorite(ctx context.Context, records []*FavoriteRecord) ([]string, error)
	DeleteFavorite(ctx context.Context, id string) error
	BatchDeleteFavorite(ctx context.Context, ids []string) error
	CheckFavorite(ctx context.Context, surname, givenName string) bool
}

// ReportServiceInterface 报告服务接口
type ReportServiceInterface interface {
	GeneratePDF(ctx context.Context, data interface{}) ([]byte, error)
	GenerateHTML(ctx context.Context, data interface{}) (string, error)
}

// FeedbackServiceInterface 反馈服务接口
type FeedbackServiceInterface interface {
	SaveNameFeedback(ctx context.Context, req *NameFeedbackRequest) (int64, error)
	SaveNameRequest(ctx context.Context, req *NameRequestRecord) (int64, error)
	GetAlgorithmPerformance(ctx context.Context) ([]*database.AlgorithmPerformance, error)
}
