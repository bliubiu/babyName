package database

import (
	"time"

	"github.com/bliubiu/babyName/internal/domain/yijing"
	"github.com/bliubiu/babyName/internal/domain/zodiac"
)

// Store 定义存储接口
type Store interface {
	// 历史记录相关方法
	SaveHistory(record *HistoryRecord) string
	BatchSaveHistory(records []*HistoryRecord) []string
	GetHistory() []*HistoryRecord
	GetHistoryByID(id string) *HistoryRecord
	DeleteHistory(id string) error
	BatchDeleteHistory(ids []string) error

	// 收藏相关方法
	SaveFavorite(record *FavoriteRecord) string
	BatchSaveFavorite(records []*FavoriteRecord) []string
	GetFavorites() []*FavoriteRecord
	DeleteFavorite(id string) error
	BatchDeleteFavorite(ids []string) error
	GetFavoriteByName(surname, givenName string) *FavoriteRecord

	// 易经和生肖相关方法
	GetHexagramByNumber(id int) *yijing.Hexagram
	GetHexagrams() []yijing.Hexagram
	GetZodiacByName(name string) *zodiac.Zodiac
	GetZodiacs() []zodiac.Zodiac
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
