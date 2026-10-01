package services

import (
	"context"

	"name/internal/domain/name"
	"name/internal/infrastructure/database"
	"name/internal/infrastructure/logger"

	"go.uber.org/zap"
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
	Pinyin   string  `json:"pinyin"`
	Gender   string  `json:"gender"`
	Score    float64 `json:"score"` // 与 name.Name.TotalScore 同口径（浮点）
	Source   string  `json:"source"`
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
	id, err := s.store.SaveFavorite(&database.FavoriteRecord{
		ID:        record.ID,
		Surname:   record.Surname,
		GivenName: record.GivenName,
		Pinyin:   record.Pinyin,
		Gender:   record.Gender,
		Score:    record.Score,
		Source:   record.Source,
		Notes:    record.Notes,
	})
	if err != nil {
		// 写库失败必须上抛（docs/29 A6）：此前吞错后仍返回 ID + nil error，
		// 用户看到「收藏成功」实际没存上，且自学习照常把名字塞进精选库，
		// 形成「收藏失败但进了精选库」的不一致
		return "", err
	}

	// 自学习：评分≥80 自动加入精选库（实时更新内存 + 持久化）。
	// 仅在收藏确实落库后执行。
	if record.Score >= 80 {
		fullName := record.Surname + record.GivenName
		if s.nameDB != nil {
			// NameDB.AddCuratedName 同时更新内存索引和持久化。
			// 失败不阻断收藏本身（收藏已成功），但必须告警，避免自学习
			// 静默失效：用户重复收藏同分名字也进不了精选库（P2-8）。
			if err := s.nameDB.AddCuratedName(fullName, record.Pinyin, record.Gender, record.Score, "user_favorite"); err != nil {
				logger.Warn("收藏自学习：加入内存精选库失败", zap.String("name", fullName), zap.Error(err))
			}
		} else if s.curatedStore != nil {
			// 无 NameDB 时直接持久化
			if err := s.curatedStore.SaveCuratedName(fullName, record.Pinyin, record.Gender, record.Score, "user_favorite"); err != nil {
				logger.Warn("收藏自学习：持久化精选名失败", zap.String("name", fullName), zap.Error(err))
			}
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
	ids, err := s.store.BatchSaveFavorite(storeRecords)
	if err != nil {
		return ids, err
	}
	return ids, nil
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
