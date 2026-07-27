package services

import (
	"context"
	"testing"

	"name/internal/infrastructure/database"
)

// mockFavoriteStore 同时实现 FavoriteStore 和 CuratedStore 接口的内存 mock
type mockFavoriteStore struct {
	favorites     []*database.FavoriteRecord
	curatedNames  []database.CuratedNameEntry
	curatedCalls  int  // 记录 SaveCuratedName 调用次数
	lastCuratedScore float64 // 记录最后一次精选名评分
}

func (m *mockFavoriteStore) SaveFavorite(record *database.FavoriteRecord) string {
	record.ID = "fav-" + record.Surname + record.GivenName
	m.favorites = append(m.favorites, record)
	return record.ID
}

func (m *mockFavoriteStore) BatchSaveFavorite(records []*database.FavoriteRecord) []string {
	ids := make([]string, len(records))
	for i, r := range records {
		ids[i] = m.SaveFavorite(r)
	}
	return ids
}

func (m *mockFavoriteStore) GetFavorites() []*database.FavoriteRecord {
	return m.favorites
}

func (m *mockFavoriteStore) GetFavoritesPage(page, limit int) ([]*database.FavoriteRecord, int, error) {
	return m.favorites, len(m.favorites), nil
}

func (m *mockFavoriteStore) DeleteFavorite(id string) error {
	for i, f := range m.favorites {
		if f.ID == id {
			m.favorites = append(m.favorites[:i], m.favorites[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockFavoriteStore) BatchDeleteFavorite(ids []string) error {
	for _, id := range ids {
		m.DeleteFavorite(id)
	}
	return nil
}

func (m *mockFavoriteStore) GetFavoriteByName(surname, givenName string) *database.FavoriteRecord {
	for _, f := range m.favorites {
		if f.Surname == surname && f.GivenName == givenName {
			return f
		}
	}
	return nil
}

// --- CuratedStore 实现 ---

func (m *mockFavoriteStore) SaveCuratedName(name, pinyin, gender string, score float64, source string) error {
	m.curatedCalls++
	m.lastCuratedScore = score
	m.curatedNames = append(m.curatedNames, database.CuratedNameEntry{
		Name: name, Pinyin: pinyin, Gender: gender, Score: score, Source: source,
	})
	return nil
}

func (m *mockFavoriteStore) LoadAllCuratedNames() ([]database.CuratedNameEntry, error) {
	return m.curatedNames, nil
}

func (m *mockFavoriteStore) IsCurated(name string) (bool, error) {
	for _, c := range m.curatedNames {
		if c.Name == name {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockFavoriteStore) DeleteCuratedName(name string) error {
	for i, c := range m.curatedNames {
		if c.Name == name {
			m.curatedNames = append(m.curatedNames[:i], m.curatedNames[i+1:]...)
			return nil
		}
	}
	return nil
}

// TestSaveFavorite_LowScore 评分<80 不触发自学习精选库
func TestSaveFavorite_LowScore(t *testing.T) {
	store := &mockFavoriteStore{}
	svc := NewFavoriteService(store)

	id, err := svc.SaveFavorite(context.Background(), &FavoriteRecord{
		Surname:   "张",
		GivenName: "三",
		Pinyin:    "zhang san",
		Gender:    "male",
		Score:     70,
	})
	if err != nil {
		t.Fatalf("SaveFavorite 返回错误: %v", err)
	}
	if id == "" {
		t.Error("SaveFavorite 返回空 ID")
	}
	if store.curatedCalls != 0 {
		t.Errorf("评分70不应触发精选库保存，但调用了 %d 次", store.curatedCalls)
	}
}

// TestSaveFavorite_HighScore 评分≥80 触发自学习精选库
func TestSaveFavorite_HighScore(t *testing.T) {
	store := &mockFavoriteStore{}
	svc := NewFavoriteService(store)

	id, err := svc.SaveFavorite(context.Background(), &FavoriteRecord{
		Surname:   "李",
		GivenName: "明轩",
		Pinyin:    "li mingxuan",
		Gender:    "male",
		Score:     85,
	})
	if err != nil {
		t.Fatalf("SaveFavorite 返回错误: %v", err)
	}
	if id == "" {
		t.Error("SaveFavorite 返回空 ID")
	}
	if store.curatedCalls != 1 {
		t.Errorf("评分85应触发1次精选库保存，实际 %d 次", store.curatedCalls)
	}
	if store.lastCuratedScore != 85 {
		t.Errorf("精选名评分应为 85，实际 %v", store.lastCuratedScore)
	}
}

// TestSaveFavorite_Existing 不重复保存已存在的收藏
func TestSaveFavorite_Existing(t *testing.T) {
	store := &mockFavoriteStore{}
	// 预置一条已存在的收藏
	store.favorites = append(store.favorites, &database.FavoriteRecord{
		ID:        "existing-id",
		Surname:   "王",
		GivenName: "涵",
	})
	svc := NewFavoriteService(store)

	id, err := svc.SaveFavorite(context.Background(), &FavoriteRecord{
		Surname:   "王",
		GivenName: "涵",
		Score:     90,
	})
	if err != nil {
		t.Fatalf("SaveFavorite 返回错误: %v", err)
	}
	if id != "existing-id" {
		t.Errorf("已存在收藏应返回原 ID \"existing-id\"，实际 %q", id)
	}
	if len(store.favorites) != 1 {
		t.Errorf("不应重复保存，收藏数应为 1，实际 %d", len(store.favorites))
	}
}

// TestCheckFavorite 检查名字是否已收藏
func TestCheckFavorite(t *testing.T) {
	store := &mockFavoriteStore{}
	store.favorites = append(store.favorites, &database.FavoriteRecord{
		Surname:   "赵",
		GivenName: "宇",
	})
	svc := NewFavoriteService(store)

	if !svc.CheckFavorite(context.Background(), "赵", "宇") {
		t.Error("CheckFavorite(\"赵\",\"宇\") = false, 期望 true（已收藏）")
	}
	if svc.CheckFavorite(context.Background(), "赵", "三") {
		t.Error("CheckFavorite(\"赵\",\"三\") = true, 期望 false（未收藏）")
	}
}

// TestDeleteFavorite 删除收藏
func TestDeleteFavorite(t *testing.T) {
	store := &mockFavoriteStore{}
	store.favorites = append(store.favorites, &database.FavoriteRecord{
		ID:        "del-1",
		Surname:   "钱",
		GivenName: "睿",
	})
	svc := NewFavoriteService(store)

	if err := svc.DeleteFavorite(context.Background(), "del-1"); err != nil {
		t.Fatalf("DeleteFavorite 返回错误: %v", err)
	}
	if len(store.favorites) != 0 {
		t.Errorf("删除后收藏数应为 0，实际 %d", len(store.favorites))
	}
}
