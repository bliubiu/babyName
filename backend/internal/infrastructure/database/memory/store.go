package memory

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/namemaster/backend/internal/domain/yijing"
	"github.com/namemaster/backend/internal/domain/zodiac"
)

type Store struct {
	mu              sync.RWMutex
	History         map[string]*HistoryRecord
	Favorites       map[string]*FavoriteRecord
	Hexagrams       []yijing.Hexagram
	Zodiacs         []zodiac.Zodiac
	currentHistoryID int64
}

type HistoryRecord struct {
	ID             string    `json:"id"`
	Surname        string    `json:"surname"`
	Gender         string    `json:"gender"`
	BirthDate      string    `json:"birth_date"`
	BirthTime      string    `json:"birth_time"`
	BirthLocation  string    `json:"birth_location"`
	Results        string    `json:"results"`
	CreatedAt      time.Time `json:"created_at"`
}

type FavoriteRecord struct {
	ID        string    `json:"id"`
	Surname   string    `json:"surname"`
	GivenName string    `json:"given_name"`
	Pinyin   string    `json:"pinyin"`
	Gender   string    `json:"gender"`
	Score    int       `json:"score"`
	Source   string    `json:"source"`
	Notes    string    `json:"notes"`
	CreatedAt time.Time `json:"created_at"`
}

func NewStore() *Store {
	store := &Store{
		History:   make(map[string]*HistoryRecord),
		Favorites: make(map[string]*FavoriteRecord),
		Hexagrams: yijing.GetAllHexagrams(),
		Zodiacs:   zodiac.GetAllZodiacs(),
	}
	return store
}

func (s *Store) SaveHistory(record *HistoryRecord) string {
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

func (s *Store) GetHistory() []*HistoryRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records := make([]*HistoryRecord, 0, len(s.History))
	for _, record := range s.History {
		records = append(records, record)
	}

	return records
}

func (s *Store) GetHistoryByID(id string) *HistoryRecord {
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

func (s *Store) SaveFavorite(record *FavoriteRecord) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if record.ID == "" {
		record.ID = uuid.New().String()
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now()
	}

	s.Favorites[record.ID] = record
	return record.ID
}

func (s *Store) GetFavorites() []*FavoriteRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records := make([]*FavoriteRecord, 0, len(s.Favorites))
	for _, record := range s.Favorites {
		records = append(records, record)
	}

	return records
}

func (s *Store) DeleteFavorite(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.Favorites[id]; !ok {
		return nil
	}

	delete(s.Favorites, id)
	return nil
}

func (s *Store) GetFavoriteByName(surname, givenName string) *FavoriteRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, record := range s.Favorites {
		if record.Surname == surname && record.GivenName == givenName {
			return record
		}
	}

	return nil
}

func (s *Store) GetHexagramByNumber(id int) *yijing.Hexagram {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := range s.Hexagrams {
		if s.Hexagrams[i].Number == id {
			return &s.Hexagrams[i]
		}
	}
	return nil
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

	for i := range s.Zodiacs {
		if s.Zodiacs[i].Name == name {
			return &s.Zodiacs[i]
		}
	}
	return nil
}

func (s *Store) GetZodiacs() []zodiac.Zodiac {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]zodiac.Zodiac, len(s.Zodiacs))
	copy(result, s.Zodiacs)
	return result
}
