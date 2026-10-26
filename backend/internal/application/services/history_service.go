package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	apperrors "name/internal/application/errors"
	"name/internal/infrastructure/database"
)

// HistoryService 历史记录服务
type HistoryService struct {
	store database.HistoryStore
}

// NewHistoryService 创建历史记录服务
func NewHistoryService(store database.HistoryStore) *HistoryService {
	return &HistoryService{store: store}
}

// HistoryRecord 历史记录
//
// SessionID 是记录归属的访客会话标识，不进入 JSON 响应（避免泄露 bearer）。
type HistoryRecord struct {
	ID            string          `json:"id"`
	SessionID     string          `json:"-"`
	Surname       string          `json:"surname"`
	Gender        string          `json:"gender"`
	BirthDate     string          `json:"birth_date"`
	BirthTime     string          `json:"birth_time"`
	BirthLocation string          `json:"birth_location"`
	Results       json.RawMessage `json:"results"`
	CreatedAt     time.Time       `json:"created_at"`
}

// SaveHistory 保存历史记录
func (s *HistoryService) SaveHistory(ctx context.Context, record *HistoryRecord) (string, error) {
	sid, err := NormalizeSessionID(record.SessionID)
	if err != nil {
		return "", err
	}
	storeRecord := &database.HistoryRecord{
		ID:            record.ID,
		SessionID:     sid,
		Surname:       record.Surname,
		Gender:        record.Gender,
		BirthDate:     record.BirthDate,
		BirthTime:     record.BirthTime,
		BirthLocation: record.BirthLocation,
		Results:       string(record.Results),
	}
	id := s.store.SaveHistory(storeRecord)
	if id == "" {
		return "", fmt.Errorf("保存历史记录失败")
	}
	return id, nil
}

// BatchSaveHistory 批量保存历史记录
func (s *HistoryService) BatchSaveHistory(ctx context.Context, records []*HistoryRecord) ([]string, error) {
	storeRecords := make([]*database.HistoryRecord, len(records))
	for i, record := range records {
		sid, err := NormalizeSessionID(record.SessionID)
		if err != nil {
			return nil, err
		}
		storeRecords[i] = &database.HistoryRecord{
			ID:            record.ID,
			SessionID:     sid,
			Surname:       record.Surname,
			Gender:        record.Gender,
			BirthDate:     record.BirthDate,
			BirthTime:     record.BirthTime,
			BirthLocation: record.BirthLocation,
			Results:       string(record.Results),
		}
	}
	return s.store.BatchSaveHistory(storeRecords), nil
}

// toHistoryRecord 存储层记录 → 应用层 DTO（GetHistory/GetHistoryPage/
// GetHistoryRecord 三处共用，避免映射口径分叉）。
func toHistoryRecord(r *database.HistoryRecord) *HistoryRecord {
	return &HistoryRecord{
		ID:            r.ID,
		Surname:       r.Surname,
		Gender:        r.Gender,
		BirthDate:     r.BirthDate,
		BirthTime:     r.BirthTime,
		BirthLocation: r.BirthLocation,
		Results:       json.RawMessage(r.Results),
		CreatedAt:     r.CreatedAt,
	}
}

// GetHistory 获取当前会话的历史记录
func (s *HistoryService) GetHistory(ctx context.Context, sessionID string) ([]*HistoryRecord, error) {
	sid, err := NormalizeSessionID(sessionID)
	if err != nil {
		return nil, err
	}
	records := s.store.GetHistory(sid)
	result := make([]*HistoryRecord, len(records))
	for i, r := range records {
		result[i] = toHistoryRecord(r)
	}
	return result, nil
}

// GetHistoryPage 获取当前会话的分页历史记录
func (s *HistoryService) GetHistoryPage(ctx context.Context, sessionID string, page, limit int) ([]*HistoryRecord, int, error) {
	sid, err := NormalizeSessionID(sessionID)
	if err != nil {
		return nil, 0, err
	}
	records, total, err := s.store.GetHistoryPage(sid, page, limit)
	if err != nil {
		return nil, 0, err
	}
	result := make([]*HistoryRecord, len(records))
	for i, r := range records {
		result[i] = toHistoryRecord(r)
	}
	return result, total, nil
}

// GetHistoryRecord 按 ID 获取当前会话的单条历史记录（SSR「历史回看」用）。
// 不属于当前会话的记录与不存在同样返回 ErrCodeNotFound，避免泄露存在性。
func (s *HistoryService) GetHistoryRecord(ctx context.Context, sessionID, id string) (*HistoryRecord, error) {
	sid, err := NormalizeSessionID(sessionID)
	if err != nil {
		return nil, err
	}
	r := s.store.GetHistoryByID(sid, id)
	if r == nil {
		return nil, apperrors.NewError(apperrors.ErrCodeNotFound, "历史记录不存在或已删除")
	}
	return toHistoryRecord(r), nil
}

// DeleteHistory 删除当前会话的历史记录
func (s *HistoryService) DeleteHistory(ctx context.Context, sessionID, id string) error {
	sid, err := NormalizeSessionID(sessionID)
	if err != nil {
		return err
	}
	return s.store.DeleteHistory(sid, id)
}
