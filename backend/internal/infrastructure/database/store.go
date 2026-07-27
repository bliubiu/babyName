package database

import (
	"time"

	"name/internal/domain/yijing"
	"name/internal/domain/zodiac"
)

// PageResult 通用分页结果
type PageResult[T any] struct {
	Items      []T `json:"items"`
	Total      int `json:"total"`
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalPages int `json:"total_pages"`
}

// HistoryStore 历史记录存储接口
type HistoryStore interface {
	SaveHistory(record *HistoryRecord) string
	BatchSaveHistory(records []*HistoryRecord) []string
	GetHistory() []*HistoryRecord
	GetHistoryPage(page, limit int) ([]*HistoryRecord, int, error)
	GetHistoryByID(id string) *HistoryRecord
	DeleteHistory(id string) error
	BatchDeleteHistory(ids []string) error
}

// FavoriteStore 收藏存储接口
type FavoriteStore interface {
	SaveFavorite(record *FavoriteRecord) string
	BatchSaveFavorite(records []*FavoriteRecord) []string
	GetFavorites() []*FavoriteRecord
	GetFavoritesPage(page, limit int) ([]*FavoriteRecord, int, error)
	DeleteFavorite(id string) error
	BatchDeleteFavorite(ids []string) error
	GetFavoriteByName(surname, givenName string) *FavoriteRecord
}

// HanziStore 汉字查询接口
type HanziStore interface {
	GetHanziByChar(char string) *Hanzi
	GetHanziByWuxing(wuxing string) []*Hanzi
	GetHanziByStrokes(min, max int) []*Hanzi
	SearchHanzi(keyword string, limit int) []*Hanzi
}

// FeedbackStore 名字反馈存储接口
type FeedbackStore interface {
	SaveNameRequest(record *NameRequest) int64
	GetNameRequestByHash(hash string) *NameRequest
	SaveNameFeedback(record *NameFeedback) int64
	GetNameFeedbackByRequestID(requestID int64) []*NameFeedback
	GetAlgorithmPerformance() ([]*AlgorithmPerformance, error)
}

// YijingStore 易经存储接口
type YijingStore interface {
	GetHexagramByNumber(id int) *yijing.Hexagram
	GetHexagrams() []yijing.Hexagram
}

// ZodiacStore 生肖存储接口
type ZodiacStore interface {
	GetZodiacByName(name string) *zodiac.Zodiac
	GetZodiacs() []zodiac.Zodiac
}

// CuratedStore 精选名存储接口
type CuratedStore interface {
	SaveCuratedName(name string, pinyin string, gender string, score float64, source string) error
	LoadAllCuratedNames() ([]CuratedNameEntry, error)
	IsCurated(name string) (bool, error)
	DeleteCuratedName(name string) error
}

// CuratedNameEntry 精选名条目
type CuratedNameEntry struct {
	Name    string  `json:"name"`
	Pinyin  string  `json:"pinyin,omitempty"`
	Gender  string  `json:"gender,omitempty"`
	Score   float64 `json:"score"`
	Source  string  `json:"source,omitempty"`
}

// Store 定义完整存储接口（向后兼容，组合所有子接口）
type Store interface {
	HistoryStore
	FavoriteStore
	HanziStore
	FeedbackStore
	YijingStore
	ZodiacStore
	CuratedStore
}

// HistoryRecord 历史记录结构
type HistoryRecord struct {
	ID            string    `json:"id"`
	Surname       string    `json:"surname"`
	Gender        string    `json:"gender"`
	BirthDate     string    `json:"birth_date"`
	BirthTime     string    `json:"birth_time"`
	BirthLocation string    `json:"birth_location"`
	Results       string    `json:"results"`
	CreatedAt     time.Time `json:"created_at"`
}

// FavoriteRecord 收藏记录结构
type FavoriteRecord struct {
	ID        string    `json:"id"`
	Surname   string    `json:"surname"`
	GivenName string    `json:"given_name"`
	Pinyin    string    `json:"pinyin"`
	Gender    string    `json:"gender"`
	Score     int       `json:"score"`
	Source    string    `json:"source"`
	Notes     string    `json:"notes"`
	CreatedAt time.Time `json:"created_at"`
}

// Hanzi 汉字数据结构
type Hanzi struct {
	Char    string `json:"char"`
	Pinyin  string `json:"pinyin"`
	Wuxing  string `json:"wuxing"`
	Strokes int    `json:"strokes"`
}

// NameRequest 名字生成请求记录
type NameRequest struct {
	ID          int64     `json:"id"`
	Surname     string    `json:"surname"`
	Gender      string    `json:"gender"`
	BirthYear   int       `json:"birth_year"`
	BirthMonth  int       `json:"birth_month"`
	BirthDay    int       `json:"birth_day"`
	BirthHour   int       `json:"birth_hour"`
	Preferences string    `json:"preferences"`
	NameType    string    `json:"name_type"`
	RequestHash string    `json:"request_hash"`
	CreatedAt   time.Time `json:"created_at"`
}

// NameFeedback 名字反馈记录
type NameFeedback struct {
	ID            int64     `json:"id"`
	RequestID     int64     `json:"request_id"`
	FullName      string    `json:"full_name"`
	GivenName     string    `json:"given_name"`
	IsLiked       bool      `json:"is_liked"`
	IsSelected    bool      `json:"is_selected"`
	UserRating    *int      `json:"user_rating,omitempty"`
	FeedbackText  string    `json:"feedback_text,omitempty"`
	AlgorithmScore float64  `json:"algorithm_score"`
	WuxingMatch   string    `json:"wuxing_match,omitempty"`
	DeviceInfo    string    `json:"device_info,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// AlgorithmPerformance 算法性能统计
type AlgorithmPerformance struct {
	Date          string  `json:"date"`
	TotalRequests int     `json:"total_requests"`
	AvgScore      float64 `json:"avg_score"`
	LikeRate      float64 `json:"like_rate"`
	SelectRate    float64 `json:"select_rate"`
}
