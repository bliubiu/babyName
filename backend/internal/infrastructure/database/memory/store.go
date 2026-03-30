package memory

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/bliubiu/babyName/internal/domain/yijing"
	"github.com/bliubiu/babyName/internal/domain/zodiac"
	"github.com/bliubiu/babyName/internal/infrastructure/database"
)

type Store struct {
	mu              sync.RWMutex
	History         map[string]*database.HistoryRecord
	Favorites       map[string]*database.FavoriteRecord
	FavoritesByName map[string]*database.FavoriteRecord // 以"surname:givenName"为键的索引
	Hexagrams       []yijing.Hexagram
	HexagramsByNumber map[int]*yijing.Hexagram // 以编号为键的索引
	Zodiacs         []zodiac.Zodiac
	ZodiacsByName   map[string]*zodiac.Zodiac // 以名称为键的索引
	currentHistoryID int64
}

// 确保Store实现了database.Store接口
var _ database.Store = (*Store)(nil)

func NewStore() *Store {
	hexagrams := yijing.GetAllHexagrams()
	zodiacs := zodiac.GetAllZodiacs()
	
	// 构建索引
	hexagramsByNumber := make(map[int]*yijing.Hexagram)
	for i := range hexagrams {
		hexagramsByNumber[hexagrams[i].Number] = &hexagrams[i]
	}
	
	zodiacsByName := make(map[string]*zodiac.Zodiac)
	for i := range zodiacs {
		zodiacsByName[zodiacs[i].Name] = &zodiacs[i]
	}
	
	store := &Store{
		History:          make(map[string]*database.HistoryRecord),
		Favorites:        make(map[string]*database.FavoriteRecord),
		FavoritesByName:  make(map[string]*database.FavoriteRecord),
		Hexagrams:        hexagrams,
		HexagramsByNumber: hexagramsByNumber,
		Zodiacs:          zodiacs,
		ZodiacsByName:    zodiacsByName,
	}
	return store
}

func (s *Store) SaveHistory(record *database.HistoryRecord) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if record.ID == "" {
		record.ID = uuid.New().String()
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now()
	}

	s.History[record.ID] = record
	return record.ID
}

func (s *Store) GetHistory() []*database.HistoryRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records := make([]*database.HistoryRecord, 0, len(s.History))
	for _, record := range s.History {
		records = append(records, record)
	}

	return records
}

func (s *Store) GetHistoryByID(id string) *database.HistoryRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.History[id]
}

func (s *Store) DeleteHistory(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.History[id]; !ok {
		return nil
	}

	delete(s.History, id)
	return nil
}

func (s *Store) BatchSaveHistory(records []*database.HistoryRecord) []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	ids := make([]string, len(records))
	for i, record := range records {
		if record.ID == "" {
			record.ID = uuid.New().String()
		}
		if record.CreatedAt.IsZero() {
			record.CreatedAt = time.Now()
		}

		s.History[record.ID] = record
		ids[i] = record.ID
	}

	return ids
}

func (s *Store) BatchDeleteHistory(ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, id := range ids {
		delete(s.History, id)
	}

	return nil
}

func (s *Store) SaveFavorite(record *database.FavoriteRecord) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if record.ID == "" {
		record.ID = uuid.New().String()
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now()
	}

	s.Favorites[record.ID] = record
	// 更新按名称索引
	key := record.Surname + ":" + record.GivenName
	s.FavoritesByName[key] = record
	return record.ID
}

func (s *Store) GetFavorites() []*database.FavoriteRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records := make([]*database.FavoriteRecord, 0, len(s.Favorites))
	for _, record := range s.Favorites {
		records = append(records, record)
	}

	return records
}

func (s *Store) DeleteFavorite(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.Favorites[id]
	if !ok {
		return nil
	}

	// 从名称索引中删除
	key := record.Surname + ":" + record.GivenName
	delete(s.FavoritesByName, key)

	// 从主映射中删除
	delete(s.Favorites, id)
	return nil
}

func (s *Store) GetFavoriteByName(surname, givenName string) *database.FavoriteRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := surname + ":" + givenName
	return s.FavoritesByName[key]
}

func (s *Store) BatchSaveFavorite(records []*database.FavoriteRecord) []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	ids := make([]string, len(records))
	for i, record := range records {
		if record.ID == "" {
			record.ID = uuid.New().String()
		}
		if record.CreatedAt.IsZero() {
			record.CreatedAt = time.Now()
		}

		s.Favorites[record.ID] = record
		// 更新按名称索引
		key := record.Surname + ":" + record.GivenName
		s.FavoritesByName[key] = record
		ids[i] = record.ID
	}

	return ids
}

func (s *Store) BatchDeleteFavorite(ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, id := range ids {
		if record, ok := s.Favorites[id]; ok {
			// 从名称索引中删除
			key := record.Surname + ":" + record.GivenName
			delete(s.FavoritesByName, key)
			// 从主映射中删除
			delete(s.Favorites, id)
		}
	}

	return nil
}

func (s *Store) GetHexagramByNumber(id int) *yijing.Hexagram {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.HexagramsByNumber[id]
}

func (s *Store) GetHexagrams() []yijing.Hexagram {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]yijing.Hexagram, len(s.Hexagrams))
	copy(result, s.Hexagrams)
	return result
}

func (s *Store) GetZodiacByName(name string) *zodiac.Zodiac {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.ZodiacsByName[name]
}

func (s *Store) GetZodiacs() []zodiac.Zodiac {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]zodiac.Zodiac, len(s.Zodiacs))
	copy(result, s.Zodiacs)
	return result
}
