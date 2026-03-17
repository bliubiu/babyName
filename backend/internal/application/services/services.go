package services

import (
	"encoding/json"

	"github.com/namemaster/backend/internal/domain/bazi"
	"github.com/namemaster/backend/internal/domain/name"
	"github.com/namemaster/backend/internal/domain/namestat"
	"github.com/namemaster/backend/internal/domain/yijing"
	"github.com/namemaster/backend/internal/domain/zodiac"
	"github.com/namemaster/backend/internal/infrastructure/database/memory"
)

// NameService 名字服务
type NameService struct {
	store *memory.Store
}

// NewNameService 创建名字服务
func NewNameService(store *memory.Store) *NameService {
	return &NameService{store: store}
}

// GenerateRequest 生成名字请求
type GenerateRequest struct {
	Surname        string   `json:"surname" binding:"required"`
	Generation     string   `json:"generation"`
	Gender         string   `json:"gender" binding:"required"`
	BirthYear      int      `json:"birth_year" binding:"required"`
	BirthMonth     int      `json:"birth_month" binding:"required"`
	BirthDay       int      `json:"birth_day" binding:"required"`
	BirthHour      int      `json:"birth_hour" binding:"required"`
	BirthMinute    int      `json:"birth_minute"`
	BirthLocation  string   `json:"birth_location"`
	NameLength     int      `json:"name_length"`
	ExcludeRare    bool     `json:"exclude_rare"`
	WuxingMatch    []string `json:"wuxing_match"`
	SourceClassic  string   `json:"source_classic"`
}

// GenerateResponse 生成名字响应
type GenerateResponse struct {
	Bazi           bazi.BaziAnalysis       `json:"bazi"`
	Nayin          string                  `json:"nayin"`
	Zodiac         string                  `json:"zodiac"`
	Hexagram       *yijing.Hexagram        `json:"hexagram"`
	HexagramMatch *yijing.HexagramMatch    `json:"hexagram_match,omitempty"`
	Names          []name.Name             `json:"names"`
}

// Generate 生成名字
func (s *NameService) Generate(req *GenerateRequest) (*GenerateResponse, error) {
	baziAnalysis := bazi.AnalyzeBazi(
		req.BirthYear,
		req.BirthMonth,
		req.BirthDay,
		req.BirthHour,
		req.BirthMinute,
	)

	nayin := baziAnalysis.Nayin
	zodiacName := bazi.GetZodiac(req.BirthYear)

	generator := name.NewNameGenerator()
	names := generator.GenerateNamesWithGeneration(req.Surname, req.Generation, req.Gender, baziAnalysis.Xiyongshen, 50, req.ExcludeRare, req.WuxingMatch, req.SourceClassic)

	totalStrokes := 0
	for _, n := range names {
		totalStrokes += n.Strokes
	}
	avgStrokes := 10
	if len(names) > 0 {
		avgStrokes = totalStrokes / len(names)
	}

	hexagram := yijing.GetHexagramByStrokes(avgStrokes)

	hexagramMatch := yijing.MatchHexagramWithXiyongshen(hexagram, baziAnalysis.Xiyongshen)

	return &GenerateResponse{
		Bazi:           *baziAnalysis,
		Nayin:          nayin,
		Zodiac:         zodiacName,
		Hexagram:       hexagram,
		HexagramMatch: hexagramMatch,
		Names:          names,
	}, nil
}

// GetByID 根据ID获取名字
func (s *NameService) GetByID(id int64) (*name.Name, error) {
	return nil, nil
}

// BaziService 八字服务
type BaziService struct {
	store *memory.Store
}

// NewBaziService 创建八字服务
func NewBaziService(store *memory.Store) *BaziService {
	return &BaziService{store: store}
}

// Analyze 分析八字
func (s *BaziService) Analyze(year, month, day, hour, minute int) (*bazi.BaziAnalysis, error) {
	return bazi.AnalyzeBazi(year, month, day, hour, minute), nil
}

// YijingService 易经服务
type YijingService struct {
	store *memory.Store
}

// NewYijingService 创建易经服务
func NewYijingService(store *memory.Store) *YijingService {
	return &YijingService{store: store}
}

// GetHexagram 获取卦象
func (s *YijingService) GetHexagram(id int) (*yijing.Hexagram, error) {
	return s.store.GetHexagramByNumber(id), nil
}

// GetAllHexagrams 获取所有卦象
func (s *YijingService) GetAllHexagrams() ([]yijing.Hexagram, error) {
	return s.store.GetHexagrams(), nil
}

// ZodiacService 生肖服务
type ZodiacService struct {
	store *memory.Store
}

// NewZodiacService 创建生肖服务
func NewZodiacService(store *memory.Store) *ZodiacService {
	return &ZodiacService{store: store}
}

// GetZodiac 获取生肖
func (s *ZodiacService) GetZodiac(animal string) (*zodiac.Zodiac, error) {
	return s.store.GetZodiacByName(animal), nil
}

// GetAllZodiacs 获取所有生肖
func (s *ZodiacService) GetAllZodiacs() ([]zodiac.Zodiac, error) {
	return s.store.GetZodiacs(), nil
}

// HistoryService 历史记录服务
type HistoryService struct {
	store *memory.Store
}

// NewHistoryService 创建历史记录服务
func NewHistoryService(store *memory.Store) *HistoryService {
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
func (s *HistoryService) SaveHistory(record *HistoryRecord) (string, error) {
	storeRecord := &memory.HistoryRecord{
		ID:            record.ID,
		Surname:       record.Surname,
		Gender:        record.Gender,
		BirthDate:     record.BirthDate,
		BirthTime:     record.BirthTime,
		BirthLocation: record.BirthLocation,
		Results:       string(record.Results),
	}
	return s.store.SaveHistory(storeRecord), nil
}

// GetHistory 获取历史记录
func (s *HistoryService) GetHistory() ([]*HistoryRecord, error) {
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

// DeleteHistory 删除历史记录
func (s *HistoryService) DeleteHistory(id string) error {
	s.store.DeleteHistory(id)
	return nil
}

// FavoriteService 收藏服务
type FavoriteService struct {
	store *memory.Store
}

// NewFavoriteService 创建收藏服务
func NewFavoriteService(store *memory.Store) *FavoriteService {
	return &FavoriteService{store: store}
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
func (s *FavoriteService) GetFavorites() ([]*FavoriteRecord, error) {
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

// SaveFavorite 保存收藏
func (s *FavoriteService) SaveFavorite(record *FavoriteRecord) (string, error) {
	existing := s.store.GetFavoriteByName(record.Surname, record.GivenName)
	if existing != nil {
		return existing.ID, nil
	}
	return s.store.SaveFavorite(&memory.FavoriteRecord{
		ID:        record.ID,
		Surname:   record.Surname,
		GivenName: record.GivenName,
		Pinyin:   record.Pinyin,
		Gender:   record.Gender,
		Score:    record.Score,
		Source:   record.Source,
		Notes:    record.Notes,
	}), nil
}

// DeleteFavorite 删除收藏
func (s *FavoriteService) DeleteFavorite(id string) error {
	return s.store.DeleteFavorite(id)
}

// CheckFavorite 检查名字是否已收藏
func (s *FavoriteService) CheckFavorite(surname, givenName string) bool {
	return s.store.GetFavoriteByName(surname, givenName) != nil
}

// StatService 统计服务
type StatService struct {
	store *memory.Store
}

// NewStatService 创建统计服务
func NewStatService(store *memory.Store) *StatService {
	return &StatService{store: store}
}

// GetNameStats 获取名字统计信息
func (s *StatService) GetNameStats(name string) (*namestat.NameStat, error) {
	return namestat.GetNameStats(name), nil
}

// GetProvinceStats 获取省份名字统计
func (s *StatService) GetProvinceStats(name, province string) (*namestat.NameStat, error) {
	stats := namestat.GetProvinceNameStats(name, province)
	return stats, nil
}

// ReportService 报告服务
type ReportService struct {
	store *memory.Store
}

// NewReportService 创建报告服务
func NewReportService(store *memory.Store) *ReportService {
	return &ReportService{store: store}
}

// GeneratePDF 生成PDF报告
func (s *ReportService) GeneratePDF(data interface{}) ([]byte, error) {
	return []byte("PDF generation not implemented"), nil
}

// GenerateHTML 生成HTML报告
func (s *ReportService) GenerateHTML(data interface{}) (string, error) {
	return "<html><body><h1>Report</h1></body></html>", nil
}
