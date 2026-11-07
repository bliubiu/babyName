package services

import (
	"context"
)

// --- 服务接口定义 ---
// 所有服务方法首参为 ctx context.Context，支持取消信号传播与超时控制

// NameServiceInterface 名字服务接口
type NameServiceInterface interface {
	Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error)
	// Evaluate 测名：给定姓名 → 完整评分报告 + 风险体检
	Evaluate(ctx context.Context, req *EvaluateRequest) (*EvaluateResponse, error)
	// ExploreNames 探索模式：从指定生成会话的候选表中换一批
	ExploreNames(ctx context.Context, req *ExploreRequest) (*ExploreResponse, error)
}

// HistoryServiceInterface 历史记录服务接口
//
// 读/删均需传入当前访客的会话标识；实现层会校验标识格式并按会话过滤。
type HistoryServiceInterface interface {
	SaveHistory(ctx context.Context, record *HistoryRecord) (string, error)
	GetHistory(ctx context.Context, sessionID string) ([]*HistoryRecord, error)
	GetHistoryPage(ctx context.Context, sessionID string, page, limit int) ([]*HistoryRecord, int, error)
	// GetHistoryRecord 按 ID 获取当前会话的单条历史记录（SSR「历史回看」用）
	GetHistoryRecord(ctx context.Context, sessionID, id string) (*HistoryRecord, error)
	DeleteHistory(ctx context.Context, sessionID, id string) error
}

// FavoriteServiceInterface 收藏服务接口
type FavoriteServiceInterface interface {
	GetFavorites(ctx context.Context, sessionID string) ([]*FavoriteRecord, error)
	// GetFavoritesPage 分页获取当前会话收藏，返回当页记录与总数（HTTP 路径走这个）
	GetFavoritesPage(ctx context.Context, sessionID string, page, limit int) ([]*FavoriteRecord, int, error)
	SaveFavorite(ctx context.Context, record *FavoriteRecord) (string, error)
	DeleteFavorite(ctx context.Context, sessionID, id string) error
}

// ReportServiceInterface 报告服务接口
type ReportServiceInterface interface {
	GeneratePDF(ctx context.Context, data interface{}) ([]byte, error)
}
