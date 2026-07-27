package services

import (
	"context"

	"name/internal/infrastructure/database"
)

// FeedbackService 名字反馈服务
type FeedbackService struct {
	feedbackStore database.FeedbackStore
	hanziStore    database.HanziStore
}

// NewFeedbackService 创建反馈服务
func NewFeedbackService(feedbackStore database.FeedbackStore, hanziStore database.HanziStore) *FeedbackService {
	return &FeedbackService{
		feedbackStore: feedbackStore,
		hanziStore:    hanziStore,
	}
}

// NameFeedbackRequest 前端反馈请求
type NameFeedbackRequest struct {
	RequestID      int64   `json:"request_id"`
	FullName       string  `json:"full_name"`
	GivenName      string  `json:"given_name"`
	IsLiked        bool    `json:"is_liked"`
	IsSelected     bool    `json:"is_selected"`
	UserRating     *int    `json:"user_rating,omitempty"`
	FeedbackText   string  `json:"feedback_text,omitempty"`
	AlgorithmScore float64 `json:"algorithm_score"`
	WuxingMatch    string  `json:"wuxing_match,omitempty"`
	DeviceInfo     string  `json:"device_info,omitempty"`
}

// SaveNameFeedback 保存名字反馈
func (s *FeedbackService) SaveNameFeedback(ctx context.Context, req *NameFeedbackRequest) (int64, error) {
	record := &database.NameFeedback{
		RequestID:      req.RequestID,
		FullName:       req.FullName,
		GivenName:      req.GivenName,
		IsLiked:        req.IsLiked,
		IsSelected:     req.IsSelected,
		UserRating:     req.UserRating,
		FeedbackText:   req.FeedbackText,
		AlgorithmScore: req.AlgorithmScore,
		WuxingMatch:    req.WuxingMatch,
		DeviceInfo:     req.DeviceInfo,
	}
	return s.feedbackStore.SaveNameFeedback(record), nil
}

// NameRequestRecord 名字生成请求记录
type NameRequestRecord struct {
	Surname     string `json:"surname"`
	Gender      string `json:"gender"`
	BirthYear   int    `json:"birth_year"`
	BirthMonth  int    `json:"birth_month"`
	BirthDay    int    `json:"birth_day"`
	BirthHour   int    `json:"birth_hour"`
	Preferences string `json:"preferences"`
	NameType    string `json:"name_type"`
	RequestHash string `json:"request_hash"`
}

// SaveNameRequest 保存名字生成请求
func (s *FeedbackService) SaveNameRequest(ctx context.Context, req *NameRequestRecord) (int64, error) {
	record := &database.NameRequest{
		Surname:     req.Surname,
		Gender:      req.Gender,
		BirthYear:   req.BirthYear,
		BirthMonth:  req.BirthMonth,
		BirthDay:    req.BirthDay,
		BirthHour:   req.BirthHour,
		Preferences: req.Preferences,
		NameType:    req.NameType,
		RequestHash: req.RequestHash,
	}
	return s.feedbackStore.SaveNameRequest(record), nil
}

// GetAlgorithmPerformance 获取算法性能统计
func (s *FeedbackService) GetAlgorithmPerformance(ctx context.Context) ([]*database.AlgorithmPerformance, error) {
	return s.feedbackStore.GetAlgorithmPerformance()
}

// GetHanziByChar 按汉字查询
func (s *FeedbackService) GetHanziByChar(ctx context.Context, char string) *database.Hanzi {
	return s.hanziStore.GetHanziByChar(char)
}

// SearchHanzi 搜索汉字
func (s *FeedbackService) SearchHanzi(ctx context.Context, keyword string, limit int) []*database.Hanzi {
	return s.hanziStore.SearchHanzi(keyword, limit)
}
