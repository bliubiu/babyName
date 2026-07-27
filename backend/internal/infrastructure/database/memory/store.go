package memory

import (
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"name/internal/domain/hanzi"
	"name/internal/domain/yijing"
	"name/internal/domain/zodiac"
	"name/internal/infrastructure/database"
)

type Store struct {
	mu              sync.RWMutex
	History         map[string]*database.HistoryRecord
	Favorites       map[string]*database.FavoriteRecord
	FavoritesByName map[string]*database.FavoriteRecord // 以"surname:givenName"为键的索引
	Curated         map[string]*database.CuratedNameEntry
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
		Curated:          make(map[string]*database.CuratedNameEntry),
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

func (s *Store) GetHistoryPage(page, limit int) ([]*database.HistoryRecord, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := len(s.History)
	records := make([]*database.HistoryRecord, 0, total)
	for _, record := range s.History {
		records = append(records, record)
	}

	// 按时间倒序排列（最新在前）
	sort.Slice(records, func(i, j int) bool {
		return records[i].CreatedAt.After(records[j].CreatedAt)
	})

	start, end := paginate(len(records), page, limit)
	return records[start:end], total, nil
}

func (s *Store) GetHistory() []*database.HistoryRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records := make([]*database.HistoryRecord, 0, len(s.History))
	for _, record := range s.History {
		records = append(records, record)
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].CreatedAt.After(records[j].CreatedAt)
	})
	return records
}

func (s *Store) GetHistoryByID(id string) *database.HistoryRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	r := s.History[id]
	// 返回拷贝，避免外部修改污染内部数据
	if r == nil {
		return nil
	}
	copyR := *r
	return &copyR
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

func (s *Store) GetFavoritesPage(page, limit int) ([]*database.FavoriteRecord, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total := len(s.Favorites)
	records := make([]*database.FavoriteRecord, 0, total)
	for _, record := range s.Favorites {
		records = append(records, record)
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].CreatedAt.After(records[j].CreatedAt)
	})

	start, end := paginate(len(records), page, limit)
	return records[start:end], total, nil
}

func (s *Store) GetFavorites() []*database.FavoriteRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records := make([]*database.FavoriteRecord, 0, len(s.Favorites))
	for _, record := range s.Favorites {
		records = append(records, record)
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].CreatedAt.After(records[j].CreatedAt)
	})
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
	r := s.FavoritesByName[key]
	// 返回拷贝，避免外部修改污染内部数据
	if r == nil {
		return nil
	}
	copyR := *r
	return &copyR
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

	h := s.HexagramsByNumber[id]
	// 返回拷贝，避免外部修改污染内部数据
	if h == nil {
		return nil
	}
	copyH := *h
	return &copyH
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

	z := s.ZodiacsByName[name]
	// 返回拷贝，避免外部修改污染内部数据
	if z == nil {
		return nil
	}
	copyZ := *z
	return &copyZ
}

func (s *Store) GetZodiacs() []zodiac.Zodiac {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]zodiac.Zodiac, len(s.Zodiacs))
	copy(result, s.Zodiacs)
	return result
}

// --- HanziStore (memory mode 使用内存汉字库) ---

func (s *Store) GetHanziByChar(char string) *database.Hanzi {
	if h, ok := hanzi.HanziData[char]; ok {
		return &database.Hanzi{
			Char:    h.Char,
			Pinyin:  h.Pinyin,
			Wuxing:  h.Wuxing,
			Strokes: h.Strokes,
		}
	}
	return nil
}

func (s *Store) GetHanziByWuxing(wuxing string) []*database.Hanzi {
	var result []*database.Hanzi
	for _, h := range hanzi.HanziData {
		if h.Wuxing == wuxing {
			result = append(result, &database.Hanzi{
				Char:    h.Char,
				Pinyin:  h.Pinyin,
				Wuxing:  h.Wuxing,
				Strokes: h.Strokes,
			})
		}
	}
	return result
}

func (s *Store) GetHanziByStrokes(min, max int) []*database.Hanzi {
	var result []*database.Hanzi
	for _, h := range hanzi.HanziData {
		if h.Strokes >= min && h.Strokes <= max {
			result = append(result, &database.Hanzi{
				Char:    h.Char,
				Pinyin:  h.Pinyin,
				Wuxing:  h.Wuxing,
				Strokes: h.Strokes,
			})
		}
	}
	return result
}

func (s *Store) SearchHanzi(keyword string, limit int) []*database.Hanzi {
	if limit <= 0 {
		limit = 50
	}
	var result []*database.Hanzi
	for _, h := range hanzi.HanziData {
		if strings.Contains(h.Char, keyword) || strings.Contains(strings.ToLower(h.Pinyin), strings.ToLower(keyword)) {
			result = append(result, &database.Hanzi{
				Char:    h.Char,
				Pinyin:  h.Pinyin,
				Wuxing:  h.Wuxing,
				Strokes: h.Strokes,
			})
			if len(result) >= limit {
				break
			}
		}
	}
	return result
}

// --- FeedbackStore stubs (memory mode 不支持持久化反馈) ---

func (s *Store) SaveNameRequest(record *database.NameRequest) int64 {
	return 0
}

func (s *Store) GetNameRequestByHash(hash string) *database.NameRequest {
	return nil
}

func (s *Store) SaveNameFeedback(record *database.NameFeedback) int64 {
	return 0
}

func (s *Store) GetNameFeedbackByRequestID(requestID int64) []*database.NameFeedback {
	return nil
}

func (s *Store) GetAlgorithmPerformance() ([]*database.AlgorithmPerformance, error) {
	return nil, nil
}

// --- CuratedStore ---

func (s *Store) SaveCuratedName(name, pinyin, gender string, score float64, source string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Curated[name] = &database.CuratedNameEntry{
		Name:   name,
		Pinyin: pinyin,
		Gender: gender,
		Score:  score,
		Source: source,
	}
	return nil
}

func (s *Store) LoadAllCuratedNames() ([]database.CuratedNameEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]database.CuratedNameEntry, 0, len(s.Curated))
	for _, e := range s.Curated {
		result = append(result, *e)
	}
	return result, nil
}

func (s *Store) IsCurated(name string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.Curated[name]
	return ok, nil
}

func (s *Store) DeleteCuratedName(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Curated, name)
	return nil
}

// paginate 计算分页起止索引
func paginate(total, page, limit int) (start, end int) {
	if total == 0 {
		return 0, 0
	}
	start = (page - 1) * limit
	if start < 0 {
		start = 0
	}
	if start >= total {
		return total, total
	}
	end = start + limit
	if end > total {
		end = total
	}
	return
}
