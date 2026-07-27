package services

import (
	"context"

	"name/internal/domain/name"
	"name/internal/infrastructure/database"
)

// FavoriteService 收藏服务
type FavoriteService struct {
	store        database.FavoriteStore
	curatedStore database.CuratedStore // 持久化层，用于自学习
	nameDB       *name.NameDB          // 内存索引，用于实时生效
}

// NewFavoriteService 创建收藏服务
func NewFavoriteService(store database.FavoriteStore) *FavoriteService {
	cs, _ := store.(database.CuratedStore)
	return &FavoriteService{store: store, curatedStore: cs}
}

// SetNameDB 设置 NameDB（自学习实时生效需要）
func (s *FavoriteService) SetNameDB(db *name.NameDB) {
	s.nameDB = db
}

// FavoriteRecord 收藏记录
type FavoriteRecord struct {
	ID        string `json:"id"`
	Surname   string `json:"surname"`
	GivenName string `json:"given_name"`
	Pinyin   string `json:"pinyin"`
	Gender   string `json:"gender"`
	Score    int    `json:"score"`
	Source   string `json:"source"`
	Notes    string `json:"notes"`
}

// GetFavorites 获取收藏列表
func (s *FavoriteService) GetFavorites(ctx context.Context) ([]*FavoriteRecord, error) {
	records := s.store.GetFavorites()
	result := make([]*FavoriteRecord, len(records))
	for i, r := range records {
		result[i] = &FavoriteRecord{
			ID:        r.ID,
			Surname:   r.Surname,
			GivenName: r.GivenName,
			Pinyin:   r.Pinyin,
			Gender:   r.Gender,
			Score:    r.Score,
			Source:   r.Source,
			Notes:    r.Notes,
		}
	}
	return result, nil
}

// SaveFavorite 保存收藏（评分≥80 自动加入精选库）
func (s *FavoriteService) SaveFavorite(ctx context.Context, record *FavoriteRecord) (string, error) {
	existing := s.store.GetFavoriteByName(record.Surname, record.GivenName)
	if existing != nil {
		return existing.ID, nil
	}
	id := s.store.SaveFavorite(&database.FavoriteRecord{
		ID:        record.ID,
		Surname:   record.Surname,
		GivenName: record.GivenName,
		Pinyin:   record.Pinyin,
		Gender:   record.Gender,
		Score:    record.Score,
		Source:   record.Source,
		Notes:    record.Notes,
	})

	// 自学习：评分≥80 自动加入精选库（实时更新内存 + 持久化）
	if record.Score >= 80 {
		fullName := record.Surname + record.GivenName
		if s.nameDB != nil {
			// NameDB.AddCuratedName 同时更新内存索引和持久化
			_ = s.nameDB.AddCuratedName(fullName, record.Pinyin, record.Gender, float64(record.Score), "user_favorite")
		} else if s.curatedStore != nil {
			// 无 NameDB 时直接持久化
			_ = s.curatedStore.SaveCuratedName(fullName, record.Pinyin, record.Gender, float64(record.Score), "user_favorite")
		}
	}

	return id, nil
}

// BatchSaveFavorite 批量保存收藏
func (s *FavoriteService) BatchSaveFavorite(ctx context.Context, records []*FavoriteRecord) ([]string, error) {
	storeRecords := make([]*database.FavoriteRecord, len(records))
	for i, record := range records {
		// 检查是否已存在
		existing := s.store.GetFavoriteByName(record.Surname, record.GivenName)
		if existing != nil {
			// 使用现有ID
			record.ID = existing.ID
		}
		storeRecords[i] = &database.FavoriteRecord{
			ID:        record.ID,
			Surname:   record.Surname,
			GivenName: record.GivenName,
			Pinyin:   record.Pinyin,
			Gender:   record.Gender,
			Score:    record.Score,
			Source:   record.Source,
			Notes:    record.Notes,
		}
	}
	return s.store.BatchSaveFavorite(storeRecords), nil
}

// BatchDeleteFavorite 批量删除收藏
func (s *FavoriteService) BatchDeleteFavorite(ctx context.Context, ids []string) error {
	return s.store.BatchDeleteFavorite(ids)
}

// DeleteFavorite 删除收藏
func (s *FavoriteService) DeleteFavorite(ctx context.Context, id string) error {
	return s.store.DeleteFavorite(id)
}

// CheckFavorite 检查名字是否已收藏
func (s *FavoriteService) CheckFavorite(ctx context.Context, surname, givenName string) bool {
	return s.store.GetFavoriteByName(surname, givenName) != nil
}
