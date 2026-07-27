package services

import (
	"context"
	"encoding/json"
	"fmt"

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
type HistoryRecord struct {
	ID             string          `json:"id"`
	Surname        string          `json:"surname"`
	Gender         string          `json:"gender"`
	BirthDate      string          `json:"birth_date"`
	BirthTime      string          `json:"birth_time"`
	BirthLocation  string          `json:"birth_location"`
	Results        json.RawMessage `json:"results"`
}

// SaveHistory 保存历史记录
func (s *HistoryService) SaveHistory(ctx context.Context, record *HistoryRecord) (string, error) {
	storeRecord := &database.HistoryRecord{
		ID:            record.ID,
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
		storeRecords[i] = &database.HistoryRecord{
			ID:            record.ID,
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

// GetHistory 获取历史记录
func (s *HistoryService) GetHistory(ctx context.Context) ([]*HistoryRecord, error) {
	records := s.store.GetHistory()
	result := make([]*HistoryRecord, len(records))
	for i, r := range records {
		result[i] = &HistoryRecord{
			ID:            r.ID,
			Surname:       r.Surname,
			Gender:        r.Gender,
			BirthDate:     r.BirthDate,
			BirthTime:     r.BirthTime,
			BirthLocation: r.BirthLocation,
			Results:       json.RawMessage(r.Results),
		}
	}
	return result, nil
}

// GetHistoryPage 获取分页历史记录
func (s *HistoryService) GetHistoryPage(ctx context.Context, page, limit int) ([]*HistoryRecord, int, error) {
	records, total, err := s.store.GetHistoryPage(page, limit)
	if err != nil {
		return nil, 0, err
	}
	result := make([]*HistoryRecord, len(records))
	for i, r := range records {
		result[i] = &HistoryRecord{
			ID:            r.ID,
			Surname:       r.Surname,
			Gender:        r.Gender,
			BirthDate:     r.BirthDate,
			BirthTime:     r.BirthTime,
			BirthLocation: r.BirthLocation,
			Results:       json.RawMessage(r.Results),
		}
	}
	return result, total, nil
}

// DeleteHistory 删除历史记录
func (s *HistoryService) DeleteHistory(ctx context.Context, id string) error {
	return s.store.DeleteHistory(id)
}
