package services

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/bliubiu/babyName/internal/application/errors"
	"github.com/bliubiu/babyName/internal/domain/bazi"
	"github.com/bliubiu/babyName/internal/domain/name"
	"github.com/bliubiu/babyName/internal/domain/namestat"
	"github.com/bliubiu/babyName/internal/domain/yijing"
	"github.com/bliubiu/babyName/internal/domain/zodiac"
	"github.com/bliubiu/babyName/internal/domain/ziwei"
	"github.com/bliubiu/babyName/internal/infrastructure/cache"
	"github.com/bliubiu/babyName/internal/infrastructure/database"
	"github.com/bliubiu/babyName/internal/infrastructure/logger"
	"go.uber.org/zap"
)

// NameService 名字服务
type NameService struct {
	store database.Store
}

// NewNameService 创建名字服务
func NewNameService(store database.Store) *NameService {
	return &NameService{store: store}
}

// GenerateRequest 生成名字请求
type GenerateRequest struct {
	Surname        string   `json:"surname" binding:"required"`
	Generation     string   `json:"generation"`
	GenerationPosition string `json:"generation_position"`
	Gender         string   `json:"gender" binding:"required"`
	BirthYear      int      `json:"birth_year" binding:"required"`
	BirthMonth     int      `json:"birth_month" binding:"required"`
	BirthDay       int      `json:"birth_day" binding:"required"`
	BirthHour      int      `json:"birth_hour" binding:"required"`
	BirthMinute    int      `json:"birth_minute"`
	BirthLocation  string   `json:"birth_location"`
	BirthType      string   `json:"birth_type"`
	NameType       string   `json:"name_type"`
	Preferences    []string `json:"preferences"`
	NameLength     int      `json:"name_length"`
	ExcludeRare    bool     `json:"exclude_rare"`
	WuxingMatch    []string `json:"wuxing_match"`
	SourceClassic  string   `json:"source_classic"`
	// 新增筛选条件
	MinStrokes     int      `json:"min_strokes"`
	MaxStrokes     int      `json:"max_strokes"`
	IncludePoetry  bool     `json:"include_poetry"`
	IncludeClassic bool     `json:"include_classic"`
	MeaningKeywords []string `json:"meaning_keywords"`
	PinyinInitial   string   `json:"pinyin_initial"`
}

// GenerateResponse 生成名字响应
type GenerateResponse struct {
	Bazi           bazi.BaziAnalysis       `json:"bazi"`
	Nayin          string                  `json:"nayin"`
	Zodiac         string                  `json:"zodiac"`
	Hexagram       *yijing.Hexagram        `json:"hexagram"`
	HexagramMatch  *yijing.HexagramMatch   `json:"hexagram_match,omitempty"`
	Ziwei          *ziwei.ZiweiAnalysis    `json:"ziwei,omitempty"`
	Names          []name.Name             `json:"names"`
}

// GenerateWithAnalysisResponse 带详细分析的名字生成响应
type GenerateWithAnalysisResponse struct {
	Bazi           bazi.BaziAnalysis       `json:"bazi"`
	Nayin          string                  `json:"nayin"`
	Zodiac         string                  `json:"zodiac"`
	Hexagram       *yijing.Hexagram        `json:"hexagram"`
	Ziwei          *ziwei.ZiweiAnalysis    `json:"ziwei,omitempty"`
	Names          []*name.NameAnalysis    `json:"names"`
	Suggestions    []string                `json:"suggestions"`
}

// Generate 生成名字
func (s *NameService) Generate(req *GenerateRequest) (*GenerateResponse, error) {
	start := time.Now()
	
	// 1. 生成缓存键
	cacheKey := cache.GenerateCacheKey("name:generate", 
		req.Surname, req.Generation, req.Gender, 
		req.BirthYear, req.BirthMonth, req.BirthDay, req.BirthHour, req.BirthMinute, 
		req.BirthLocation, req.NameLength, req.ExcludeRare, req.WuxingMatch, req.SourceClassic)
	
	// 2. 检查缓存
	if cached, found := cache.GetCache().Get(cacheKey); found {
		if cache.GetCache().IsNull(cached) {
			return nil, errors.NewError(errors.ErrCodeBadRequest, "无法生成符合条件的名字，请调整筛选条件")
		}
		if response, ok := cached.(*GenerateResponse); ok {
			logger.Info("Name generation loaded from cache",
				zap.String("surname", req.Surname),
				zap.String("gender", req.Gender),
			)
			return response, nil
		}
	}
	
	// 3. 八字分析
	baziAnalysis, baziDuration := s.performBaziAnalysis(req)
	
	// 4. 生成名字
	names, generateDuration, err := s.generateNames(req, baziAnalysis)
	if err != nil {
		return nil, err
	}
	
	// 5. 计算易经卦象和紫微斗数（并行计算）
	hexagram, hexagramMatch, ziweiAnalysis, parallelDuration := s.calculateHexagramAndZiweiParallel(names, baziAnalysis, req)
	
	// 6. 构建响应
	response := s.buildResponse(baziAnalysis, names, hexagram, hexagramMatch, ziweiAnalysis)
	
	// 7. 记录日志并缓存
	s.logAndCache(start, cacheKey, response, req, baziDuration, generateDuration, parallelDuration)
	
	return response, nil
}

// performBaziAnalysis 执行八字分析
func (s *NameService) performBaziAnalysis(req *GenerateRequest) (*bazi.BaziAnalysis, time.Duration) {
	baziStart := time.Now()
	baziAnalysis, err := bazi.AnalyzeBazi(
		req.BirthYear,
		req.BirthMonth,
		req.BirthDay,
		req.BirthHour,
		req.BirthMinute,
	)
	if err != nil {
		logger.Error(err)
		baziAnalysis = &bazi.BaziAnalysis{}
	}
	baziDuration := time.Since(baziStart)
	
	logger.Info("Bazi analysis completed",
		zap.String("surname", req.Surname),
		zap.Duration("duration", baziDuration),
	)
	
	return baziAnalysis, baziDuration
}

// generateNames 生成名字列表
func (s *NameService) generateNames(req *GenerateRequest, baziAnalysis *bazi.BaziAnalysis) ([]name.Name, time.Duration, error) {
	generateStart := time.Now()
	
	nameLength := req.NameLength
	if nameLength <= 0 {
		nameLength = 2
	} else if nameLength > 4 {
		nameLength = 4
	}
	
	logger.Info("Generate: calling GenerateNamesWithGeneration",
		zap.String("surname", req.Surname),
		zap.String("gender", req.Gender),
		zap.String("generation", req.Generation),
		zap.String("generation_position", req.GenerationPosition),
		zap.String("name_type", req.NameType),
		zap.Int("name_length", nameLength),
	)
	
	generator := name.NewNameGenerator()
	names := generator.GenerateNamesWithGeneration(
		req.Surname, req.Generation, req.Gender, baziAnalysis.Xiyongshen, 50, nameLength, 
		req.ExcludeRare, req.WuxingMatch, req.SourceClassic, 
		req.MinStrokes, req.MaxStrokes, req.IncludePoetry, req.IncludeClassic, 
		req.MeaningKeywords, req.PinyinInitial, req.GenerationPosition, req.NameType,
	)
	
	generateDuration := time.Since(generateStart)
	
	if len(names) == 0 {
		logger.Warn("Generate: no names generated",
			zap.String("surname", req.Surname),
			zap.String("gender", req.Gender),
		)
		return nil, generateDuration, errors.NewError(errors.ErrCodeBadRequest, "无法生成符合条件的名字，请调整筛选条件")
	}
	
	logger.Info("Generate: names generated successfully",
		zap.String("surname", req.Surname),
		zap.Int("count", len(names)),
		zap.Duration("duration", generateDuration),
	)
	
	return names, generateDuration, nil
}

// calculateHexagramAndZiweiParallel 并行计算易经卦象和紫微斗数
func (s *NameService) calculateHexagramAndZiweiParallel(names []name.Name, baziAnalysis *bazi.BaziAnalysis, req *GenerateRequest) (*yijing.Hexagram, *yijing.HexagramMatch, *ziwei.ZiweiAnalysis, time.Duration) {
	start := time.Now()

	totalStrokes := 0
	for _, n := range names {
		totalStrokes += n.Strokes
	}
	avgStrokes := 10
	if len(names) > 0 {
		avgStrokes = totalStrokes / len(names)
	}

	var hexagram *yijing.Hexagram
	var hexagramMatch *yijing.HexagramMatch
	var ziweiAnalysis *ziwei.ZiweiAnalysis

	var wg sync.WaitGroup
	var hexagramMutex sync.Mutex
	var ziweiMutex sync.Mutex

	wg.Add(2)

	go func() {
		defer wg.Done()
		h := yijing.GetHexagramByStrokes(avgStrokes)
		m := yijing.MatchHexagramWithXiyongshen(h, baziAnalysis.Xiyongshen)

		hexagramMutex.Lock()
		hexagram = h
		hexagramMatch = m
		hexagramMutex.Unlock()
	}()

	go func() {
		defer wg.Done()
		z := ziwei.AnalyzeZiwei(req.BirthYear, req.BirthMonth, req.BirthDay, req.BirthHour)

		ziweiMutex.Lock()
		ziweiAnalysis = z
		ziweiMutex.Unlock()
	}()

	wg.Wait()

	duration := time.Since(start)

	logger.Info("Parallel calculation completed",
		zap.String("hexagram", hexagram.Name),
		zap.String("ziwei_ming_gong", ziweiAnalysis.Gongwei),
		zap.Duration("duration", duration),
	)

	return hexagram, hexagramMatch, ziweiAnalysis, duration
}

// buildResponse 构建响应对象
func (s *NameService) buildResponse(
	baziAnalysis *bazi.BaziAnalysis,
	names []name.Name,
	hexagram *yijing.Hexagram,
	hexagramMatch *yijing.HexagramMatch,
	ziweiAnalysis *ziwei.ZiweiAnalysis,
) *GenerateResponse {
	nayin := baziAnalysis.Nayin
	// 从八字分析结果中提取年份干支，转换为年份数字
	yearStr := baziAnalysis.Bazi.Year // 例如 "癸卯"
	zodiacName := getZodiacFromYearGanZhi(yearStr)
	
	return &GenerateResponse{
		Bazi:          *baziAnalysis,
		Nayin:         nayin,
		Zodiac:        zodiacName,
		Hexagram:      hexagram,
		HexagramMatch: hexagramMatch,
		Ziwei:         ziweiAnalysis,
		Names:         names,
	}
}

// getZodiacFromYearGanZhi 从年份干支获取生肖
func getZodiacFromYearGanZhi(yearGanZhi string) string {
	// 简化处理：根据地支判断生肖
	// 子鼠、丑牛、寅虎、卯兔、辰龙、巳蛇、午马、未羊、申猴、酉鸡、戌狗、亥猪
	zodiacMap := map[string]string{
		"子": "鼠", "丑": "牛", "寅": "虎", "卯": "兔",
		"辰": "龙", "巳": "蛇", "午": "马", "未": "羊",
		"申": "猴", "酉": "鸡", "戌": "狗", "亥": "猪",
	}
	
	if len(yearGanZhi) >= 2 {
		diZhi := yearGanZhi[1:2] // 取地支
		if zodiac, ok := zodiacMap[diZhi]; ok {
			return zodiac
		}
	}
	return "未知"
}

// logAndCache 记录日志并缓存结果
func (s *NameService) logAndCache(
	start time.Time,
	cacheKey string,
	response *GenerateResponse,
	req *GenerateRequest,
	baziDuration, generateDuration, parallelDuration time.Duration,
) {
	totalDuration := time.Since(start)
	
	logger.Info("Generate: completed",
		zap.String("surname", req.Surname),
		zap.String("gender", req.Gender),
		zap.Duration("total", totalDuration),
		zap.Duration("bazi", baziDuration),
		zap.Duration("generate", generateDuration),
		zap.Duration("parallel", parallelDuration),
		zap.Int("name_count", len(response.Names)),
	)
	
	cache.GetCache().Set(cacheKey, response, 1*time.Hour)
	logger.Info("Name generation cached",
		zap.String("surname", req.Surname),
		zap.String("gender", req.Gender),
		zap.Duration("cache_ttl", 1*time.Hour),
	)
}

// GetByID 根据ID获取名字
func (s *NameService) GetByID(id int64) (*name.Name, error) {
	return nil, nil
}

// GenerateWithAnalysis 生成带详细分析的名字
func (s *NameService) GenerateWithAnalysis(req *GenerateRequest) (*GenerateWithAnalysisResponse, error) {
	start := time.Now()
	
	// 生成缓存键
	cacheKey := cache.GenerateCacheKey("name:analysis", 
		req.Surname, req.Generation, req.Gender, 
		req.BirthYear, req.BirthMonth, req.BirthDay, req.BirthHour, req.BirthMinute, 
		req.BirthLocation, req.NameLength, req.ExcludeRare, req.WuxingMatch, req.SourceClassic)
	
	// 检查缓存
	if cached, found := cache.GetCache().Get(cacheKey); found {
		if response, ok := cached.(*GenerateWithAnalysisResponse); ok {
			logger.Info("Name analysis loaded from cache",
				zap.String("surname", req.Surname),
				zap.String("gender", req.Gender),
			)
			return response, nil
		}
	}
	
	// 八字分析
	baziAnalysis, err := bazi.AnalyzeBazi(
		req.BirthYear,
		req.BirthMonth,
		req.BirthDay,
		req.BirthHour,
		req.BirthMinute,
	)
	if err != nil {
		logger.Error(err)
		baziAnalysis = &bazi.BaziAnalysis{}
	}
	
	nayin := baziAnalysis.Nayin
	zodiacName := bazi.GetZodiac(req.BirthYear)
	
	// 使用增强版名字生成器
	generator := name.NewEnhancedNameGenerator()
	
	// 生成带分析的名字
	nameAnalyses, err := generator.GenerateNamesWithAnalysis(
		req.Surname,
		req.Generation,
		req.Gender,
		baziAnalysis.Xiyongshen,
		50, // 生成50个名字
		req.NameLength,
		req.ExcludeRare,
		req.WuxingMatch,
		req.SourceClassic,
		req.MinStrokes,
		req.MaxStrokes,
		req.IncludePoetry,
		req.IncludeClassic,
		req.MeaningKeywords,
		req.PinyinInitial,
		req.GenerationPosition,
		req.NameType,
		zodiacName,
	)
	
	if err != nil {
		return nil, err
	}
	
	// 获取名字建议
	preferences := map[string]interface{}{
		"style":        req.NameType,
		"wuxing":       req.WuxingMatch,
		"min_strokes":  req.MinStrokes,
		"max_strokes":  req.MaxStrokes,
	}
	suggestions, _ := generator.GetNameSuggestions(req.Surname, req.Gender, preferences)
	
	// 计算平均笔画数
	totalStrokes := 0
	for _, na := range nameAnalyses {
		totalStrokes += na.Strokes
	}
	avgStrokes := 10
	if len(nameAnalyses) > 0 {
		avgStrokes = totalStrokes / len(nameAnalyses)
	}
	
	// 易经分析
	hexagram := yijing.GetHexagramByStrokes(avgStrokes)
	
	// 紫微斗数分析
	ziweiAnalysis := ziwei.AnalyzeZiwei(req.BirthYear, req.BirthMonth, req.BirthDay, req.BirthHour)
	
	response := &GenerateWithAnalysisResponse{
		Bazi:        *baziAnalysis,
		Nayin:       nayin,
		Zodiac:      zodiacName,
		Hexagram:    hexagram,
		Ziwei:       ziweiAnalysis,
		Names:       nameAnalyses,
		Suggestions: suggestions,
	}
	
	totalDuration := time.Since(start)
	logger.Info("Generate with analysis completed",
		zap.String("surname", req.Surname),
		zap.String("gender", req.Gender),
		zap.Duration("total", totalDuration),
		zap.Int("name_count", len(nameAnalyses)),
	)
	
	// 缓存结果，有效期1小时
	cache.GetCache().Set(cacheKey, response, 1*time.Hour)
	logger.Info("Name analysis cached",
		zap.String("surname", req.Surname),
		zap.String("gender", req.Gender),
	)
	
	return response, nil
}

// BaziService 八字服务
type BaziService struct {
	store database.Store
}

// NewBaziService 创建八字服务
func NewBaziService(store database.Store) *BaziService {
	return &BaziService{store: store}
}

// Analyze 分析八字
func (s *BaziService) Analyze(year, month, day, hour, minute int) (*bazi.BaziAnalysis, error) {
	// 生成缓存键
	cacheKey := cache.GenerateCacheKey("bazi", year, month, day, hour, minute)
	
	// 检查缓存
	if cached, found := cache.GetCache().Get(cacheKey); found {
		if analysis, ok := cached.(*bazi.BaziAnalysis); ok {
			logger.Info("Bazi analysis loaded from cache",
				zap.Int("year", year),
				zap.Int("month", month),
				zap.Int("day", day),
				zap.Int("hour", hour),
				zap.Int("minute", minute),
			)
			return analysis, nil
		}
	}
	
	// 执行分析
	analysis, err := bazi.AnalyzeBazi(year, month, day, hour, minute)
	if err != nil {
		logger.Error(err)
		analysis = &bazi.BaziAnalysis{}
	}
	
	// 缓存结果，有效期24小时
	cache.GetCache().Set(cacheKey, analysis, 24*time.Hour)
	
	logger.Info("Bazi analysis cached",
		zap.Int("year", year),
		zap.Int("month", month),
		zap.Int("day", day),
		zap.Int("hour", hour),
		zap.Int("minute", minute),
	)
	
	return analysis, nil
}

// YijingService 易经服务
type YijingService struct {
	store database.Store
}

// NewYijingService 创建易经服务
func NewYijingService(store database.Store) *YijingService {
	return &YijingService{store: store}
}

// GetHexagram 获取卦象
func (s *YijingService) GetHexagram(id int) (*yijing.Hexagram, error) {
	// 生成缓存键
	cacheKey := cache.GenerateCacheKey("hexagram", id)
	
	// 检查缓存
	if cached, found := cache.GetCache().Get(cacheKey); found {
		if hexagram, ok := cached.(*yijing.Hexagram); ok {
			logger.Info("Hexagram loaded from cache", zap.Int("id", id))
			return hexagram, nil
		}
	}
	
	// 从存储中获取
	hexagram := s.store.GetHexagramByNumber(id)
	
	// 缓存结果，有效期7天
	if hexagram != nil {
		cache.GetCache().Set(cacheKey, hexagram, 7*24*time.Hour)
		logger.Info("Hexagram cached", zap.Int("id", id))
	}
	
	return hexagram, nil
}

// GetAllHexagrams 获取所有卦象
func (s *YijingService) GetAllHexagrams() ([]yijing.Hexagram, error) {
	// 生成缓存键
	cacheKey := "hexagrams:all"
	
	// 检查缓存
	if cached, found := cache.GetCache().Get(cacheKey); found {
		if hexagrams, ok := cached.([]yijing.Hexagram); ok {
			logger.Info("All hexagrams loaded from cache")
			return hexagrams, nil
		}
	}
	
	// 从存储中获取
	hexagrams := s.store.GetHexagrams()
	
	// 缓存结果，有效期7天
	cache.GetCache().Set(cacheKey, hexagrams, 7*24*time.Hour)
	logger.Info("All hexagrams cached")
	
	return hexagrams, nil
}

// ZodiacService 生肖服务
type ZodiacService struct {
	store database.Store
}

// NewZodiacService 创建生肖服务
func NewZodiacService(store database.Store) *ZodiacService {
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
	store database.Store
}

// NewHistoryService 创建历史记录服务
func NewHistoryService(store database.Store) *HistoryService {
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
	storeRecord := &database.HistoryRecord{
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

// BatchSaveHistory 批量保存历史记录
func (s *HistoryService) BatchSaveHistory(records []*HistoryRecord) ([]string, error) {
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
	store database.Store
}

// NewFavoriteService 创建收藏服务
func NewFavoriteService(store database.Store) *FavoriteService {
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
	return s.store.SaveFavorite(&database.FavoriteRecord{
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

// BatchSaveFavorite 批量保存收藏
func (s *FavoriteService) BatchSaveFavorite(records []*FavoriteRecord) ([]string, error) {
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
func (s *FavoriteService) BatchDeleteFavorite(ids []string) error {
	return s.store.BatchDeleteFavorite(ids)
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
	store database.Store
}

// NewStatService 创建统计服务
func NewStatService(store database.Store) *StatService {
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
	store database.Store
}

// NewReportService 创建报告服务
func NewReportService(store database.Store) *ReportService {
	return &ReportService{store: store}
}

// GeneratePDF 生成PDF报告
func (s *ReportService) GeneratePDF(data interface{}) ([]byte, error) {
	// 这里应该实现PDF生成逻辑
	// 由于没有实际的PDF生成库，这里返回一个简单的PDF文件结构
	// 实际项目中应该使用如unidoc/unipdf等库
	pdfContent := `%PDF-1.4
1 0 obj
<< /Type /Catalog
   /Pages 2 0 R
>>
endobj
2 0 obj
<< /Type /Pages
   /Kids [3 0 R]
   /Count 1
>>
endobj
3 0 obj
<< /Type /Page
   /Parent 2 0 R
   /MediaBox [0 0 612 792]
   /Contents 4 0 R
   /Resources << /Font << /F1 5 0 R >> >>
>>
endobj
4 0 obj
<< /Length 72 >>
stream
BT
/F1 24 Tf
100 700 Td
(宝宝起名报告) Tj
ET
BT
/F1 12 Tf
100 650 Td
(这是一个PDF报告示例) Tj
ET
endstream
endobj
5 0 obj
<< /Type /Font
   /Subtype /Type1
   /Name /F1
   /BaseFont /Helvetica
   /Encoding /WinAnsiEncoding
>>
endobj
xref
0 6
0000000000 65535 f 
0000000010 00000 n 
0000000067 00000 n 
0000000118 00000 n 
0000000208 00000 n 
0000000315 00000 n 
trailer
<< /Size 6
   /Root 1 0 R
>>
%%EOF`
	return []byte(pdfContent), nil
}

// GenerateHTML 生成HTML报告
func (s *ReportService) GenerateHTML(data interface{}) (string, error) {
	// 检查数据类型
	if generateData, ok := data.(map[string]interface{}); ok {
		// 构建HTML报告
		htmlContent := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>宝宝起名报告</title>
	<style>
		body {
			font-family: Arial, sans-serif;
			line-height: 1.6;
			margin: 0;
			padding: 20px;
			background-color: #f5f5f5;
		}
		.container {
			max-width: 800px;
			margin: 0 auto;
			background-color: white;
			padding: 40px;
			border-radius: 10px;
			box-shadow: 0 0 10px rgba(0,0,0,0.1);
		}
		h1 {
			color: #333;
			text-align: center;
			margin-bottom: 30px;
		}
		.section {
			margin-bottom: 30px;
		}
		.section h2 {
			color: #555;
			border-bottom: 2px solid #e0e0e0;
			padding-bottom: 10px;
			margin-bottom: 20px;
		}
		.info-item {
			margin-bottom: 10px;
		}
		.info-label {
			font-weight: bold;
			display: inline-block;
			width: 120px;
		}
		.name-list {
			list-style: none;
			padding: 0;
		}
		.name-item {
			padding: 10px;
			border-bottom: 1px solid #f0f0f0;
		}
		.name-item:hover {
			background-color: #f9f9f9;
		}
		.name-score {
			font-weight: bold;
			color: #4CAF50;
		}
	</style>
</head>
<body>
	<div class="container">
		<h1>宝宝起名报告</h1>
		
		<div class="section">
			<h2>基本信息</h2>
			<div class="info-item">
				<span class="info-label">姓氏：</span>
				<span>` + getStringValue(generateData, "surname") + `</span>
			</div>
			<div class="info-item">
				<span class="info-label">性别：</span>
				<span>` + getStringValue(generateData, "gender") + `</span>
			</div>
			<div class="info-item">
				<span class="info-label">出生日期：</span>
				<span>` + getStringValue(generateData, "birth_date") + `</span>
			</div>
			<div class="info-item">
				<span class="info-label">出生时间：</span>
				<span>` + getStringValue(generateData, "birth_time") + `</span>
			</div>
		</div>
		
		<div class="section">
			<h2>八字分析</h2>
			<div class="info-item">
				<span class="info-label">八字：</span>
				<span>` + getStringValue(generateData, "bazi") + `</span>
			</div>
			<div class="info-item">
				<span class="info-label">五行：</span>
				<span>` + getStringValue(generateData, "wuxing") + `</span>
			</div>
			<div class="info-item">
				<span class="info-label">喜用神：</span>
				<span>` + getStringValue(generateData, "xiyongshen") + `</span>
			</div>
		</div>
		
		<div class="section">
			<h2>推荐名字</h2>
			<ul class="name-list">`
			
			// 添加推荐名字列表
		if names, ok := generateData["names"].([]interface{}); ok {
			for _, name := range names {
				if nameMap, ok := name.(map[string]interface{}); ok {
					htmlContent += `
					<li class="name-item">
						<div><strong>` + getStringValue(nameMap, "full_name") + `</strong> <span class="name-score">` + getStringValue(nameMap, "score") + `分</span></div>
						<div>拼音：` + getStringValue(nameMap, "pinyin") + `</div>
						<div>寓意：` + getStringValue(nameMap, "meaning") + `</div>
						<div>五行：` + getStringValue(nameMap, "wuxing") + `</div>
					</li>`
				}
			}
		}
			
			htmlContent += `
			</ul>
		</div>
	</div>
</body>
</html>`
		return htmlContent, nil
	}
	return "<html><body><h1>报告生成失败</h1></body></html>", nil
}

// getStringValue 获取字符串值
func getStringValue(data map[string]interface{}, key string) string {
	if value, ok := data[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
		return fmt.Sprintf("%v", value)
	}
	return ""
}
