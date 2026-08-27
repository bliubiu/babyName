package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"name/internal/application/errors"
	"name/internal/domain/bazi"
	"name/internal/domain/fate"
	"name/internal/domain/name"
	"name/internal/domain/yijing"
	"name/internal/domain/ziwei"
	"name/internal/domain/zodiac"
	"name/internal/infrastructure/cache"
	"name/internal/infrastructure/logger"

	"go.uber.org/zap"
)

// NameService 名字服务
type NameService struct {
	baziAnalyzer     bazi.BaziAnalyzer
	hexagramFinder   yijing.HexagramFinder
	ziweiAnalyzer    ziwei.ZiweiAnalyzer
	enhancedAnalyzer name.EnhancedNameAnalyzer
	zodiacFinder     zodiac.ZodiacFinder
	fateService      *FateNameService // fate 路径委托服务（可选）
	cache            cache.Cache
}

// NameServiceOption 名字服务选项
type NameServiceOption func(*NameService)

// WithBaziAnalyzer 设置八字分析器
func WithBaziAnalyzer(a bazi.BaziAnalyzer) NameServiceOption {
	return func(s *NameService) { s.baziAnalyzer = a }
}

// WithHexagramFinder 设置卦象查找器
func WithHexagramFinder(f yijing.HexagramFinder) NameServiceOption {
	return func(s *NameService) { s.hexagramFinder = f }
}

// WithZiweiAnalyzer 设置紫微斗数分析器
func WithZiweiAnalyzer(a ziwei.ZiweiAnalyzer) NameServiceOption {
	return func(s *NameService) { s.ziweiAnalyzer = a }
}

// WithEnhancedAnalyzer 设置增强分析器
func WithEnhancedAnalyzer(a name.EnhancedNameAnalyzer) NameServiceOption {
	return func(s *NameService) { s.enhancedAnalyzer = a }
}

// WithFateService 设置 fate 名字服务（可选）
// 设置后 GenerateWithAnalysis 将优先委托给 FateNameService
func WithFateService(fs *FateNameService) NameServiceOption {
	return func(s *NameService) { s.fateService = fs }
}

// WithZodiacFinder 设置生肖查找器
func WithZodiacFinder(f zodiac.ZodiacFinder) NameServiceOption {
	return func(s *NameService) { s.zodiacFinder = f }
}

// WithCache 设置缓存
func WithCache(c cache.Cache) NameServiceOption {
	return func(s *NameService) { s.cache = c }
}

// NewNameService 创建名字服务
func NewNameService(opts ...NameServiceOption) *NameService {
	s := &NameService{}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// GetNameDB 获取候选名库管理器（用于 API 层共享自学习精选名数据）
func (s *NameService) GetNameDB() *name.NameDB {
	if a, ok := s.enhancedAnalyzer.(*EnhancedNameAnalyzerAdapter); ok {
		return a.GetNameDB()
	}
	return nil
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

	// 避讳长辈：父系/母系直系长辈姓名（建议往上两代）
	// 生成名字时排除同形字与同音字，避免"压运"
	AvoidElderNames []string `json:"avoid_elder_names"`

	// 人名频率过滤（来自 Chinese-Names-Corpus 语料统计）
	MinFrequencyTier int `json:"min_frequency_tier"` // 最小频率档位（1-5，0=不限）
	MaxFrequencyTier int `json:"max_frequency_tier"` // 最大频率档位（1-5，0=不限）
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
func (s *NameService) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	start := time.Now()

		// 1. 八字分析
	baziAnalysis, baziDuration, baziErr := s.performBaziAnalysis(req)
	analysisFailed := baziAnalysis == nil || len(baziAnalysis.Xiyongshen) == 0
	if analysisFailed {
		logger.Warn("Generate: 八字分析结果不完整，跳过喜用神匹配",
			zap.String("surname", req.Surname),
			zap.Error(baziErr),
		)
	}

		// 2. 生成名字
	names, generateDuration, err := s.generateNames(ctx, req, baziAnalysis)
	if err != nil {
		return nil, err
	}

	// 3. 计算易经卦象和紫微斗数（并行计算）
	hexagram, hexagramMatch, ziweiAnalysis, parallelDuration := s.calculateHexagramAndZiweiParallel(names, baziAnalysis, req)

	// 4. 构建响应
	zodiacName := s.zodiacFinder.FindByYear(req.BirthYear)
	response := s.buildResponse(baziAnalysis, names, hexagram, hexagramMatch, ziweiAnalysis, zodiacName)

	// 5. 记录日志
	s.logGeneration(start, response, req, baziDuration, generateDuration, parallelDuration)

	return response, nil
}

// GenerateWithAnalysis 生成带详细分析的名字
func (s *NameService) GenerateWithAnalysis(ctx context.Context, req *GenerateRequest) (*GenerateWithAnalysisResponse, error) {
	// 优先委托给 fate 名字服务
	if s.fateService != nil {
		return s.fateService.GenerateWithAnalysis(ctx, req)
	}

	// 1. 八字分析
	baziAnalysis, _, _ := s.performBaziAnalysis(req) // 分析失败时返回空 BaziAnalysis，后续流程继续

	nayin := baziAnalysis.Nayin
	zodiacName := s.zodiacFinder.FindByYear(req.BirthYear)

	// 2. 使用增强版名字生成器
	nameAnalyses, err := s.enhancedAnalyzer.GenerateWithAnalysis(name.GenerateOptions{
		Surname:            req.Surname,
		Generation:         req.Generation,
		Gender:             req.Gender,
		Xiyongshen:         baziAnalysis.Xiyongshen,
		Count:              50,
		NameLength:         req.NameLength,
		ExcludeRare:        req.ExcludeRare,
		WuxingMatch:        req.WuxingMatch,
		SourceClassic:      req.SourceClassic,
		MinStrokes:         req.MinStrokes,
		MaxStrokes:         req.MaxStrokes,
		IncludePoetry:      req.IncludePoetry,
		IncludeClassic:     req.IncludeClassic,
		MeaningKeywords:    req.MeaningKeywords,
		PinyinInitial:      req.PinyinInitial,
		GenerationPosition: req.GenerationPosition,
		NameType:           req.NameType,
		Zodiac:             zodiacName,
		Nayin:              nayin,
	})

	if err != nil {
		return nil, err
	}

	// 3. 计算平均笔画数
	totalStrokes := 0
	for _, na := range nameAnalyses {
		totalStrokes += na.Strokes
	}
	avgStrokes := 10
	if len(nameAnalyses) > 0 {
		avgStrokes = totalStrokes / len(nameAnalyses)
	}

	// 4. 易经分析
	hexagram := s.hexagramFinder.FindByStrokes(avgStrokes)

	// 5. 紫微斗数分析
	ziweiAnalysis := s.ziweiAnalyzer.Analyze(req.BirthYear, req.BirthMonth, req.BirthDay, req.BirthHour)

	response := &GenerateWithAnalysisResponse{
		Bazi:        *baziAnalysis,
		Nayin:       nayin,
		Zodiac:      zodiacName,
		Hexagram:    hexagram,
		Ziwei:       ziweiAnalysis,
		Names:       nameAnalyses,
		Suggestions: generateNameSuggestions(nameAnalyses),
	}

	logger.Info("Name generation with analysis completed",
		zap.String("surname", req.Surname),
		zap.String("gender", req.Gender),
		zap.Int("name_count", len(nameAnalyses)),
	)

	return response, nil
}

// performBaziAnalysis 执行八字分析
// 返回分析结果和耗时；分析失败时返回空 BaziAnalysis 并附带 error，
// 调用者应记录日志后继续（graceful degradation），不阻断生成流程。
func (s *NameService) performBaziAnalysis(req *GenerateRequest) (*bazi.BaziAnalysis, time.Duration, error) {
	baziStart := time.Now()
	baziAnalysis, err := s.baziAnalyzer.Analyze(
		req.BirthYear,
		req.BirthMonth,
		req.BirthDay,
		req.BirthHour,
		req.BirthMinute,
	)
	baziDuration := time.Since(baziStart)

	if err != nil {
		logger.Error("八字分析失败，使用空分析继续生成名字", zap.Error(err))
		return &bazi.BaziAnalysis{}, baziDuration, err
	}

	logger.Info("八字分析完成",
		zap.String("surname", req.Surname),
		zap.Duration("duration", baziDuration),
	)

	return baziAnalysis, baziDuration, nil
}

// generateNames 生成名字列表
func (s *NameService) generateNames(ctx context.Context, req *GenerateRequest, baziAnalysis *bazi.BaziAnalysis) ([]name.Name, time.Duration, error) {
	generateStart := time.Now()

	nameLength := req.NameLength
	if nameLength <= 0 {
		nameLength = 2
	} else if nameLength > 4 {
		nameLength = 4
	}

	var names []name.Name
	var err error

	// fate 引擎优先（当 FateNameService 配置时）
	// 新引擎具备更完善的 Rater 评分链、避讳长辈、负面反馈等能力
	if s.fateService != nil {
		names, err = s.generateNamesViaFate(ctx, req, baziAnalysis)
	} else {
		// 旧引擎路径（仅当 fate 未配置时，保持向后兼容）
		zodiacName := s.zodiacFinder.FindByYear(req.BirthYear)

		genOpts := name.GenerateOptions{
			Surname:            req.Surname,
			Generation:         req.Generation,
			Gender:             req.Gender,
			Xiyongshen:         baziAnalysis.Xiyongshen,
			Count:              50,
			NameLength:         nameLength,
			ExcludeRare:        req.ExcludeRare,
			WuxingMatch:        req.WuxingMatch,
			SourceClassic:      req.SourceClassic,
			MinStrokes:         req.MinStrokes,
			MaxStrokes:         req.MaxStrokes,
			IncludePoetry:      req.IncludePoetry,
			IncludeClassic:     req.IncludeClassic,
			MeaningKeywords:    req.MeaningKeywords,
			PinyinInitial:      req.PinyinInitial,
			GenerationPosition: req.GenerationPosition,
			NameType:           req.NameType,
			Zodiac:             zodiacName,
			Nayin:              baziAnalysis.Nayin,
			AvoidElderNames:    req.AvoidElderNames,
			DayMasterStrength:  baziAnalysis.DayMasterStrength,
		}

		logger.Info("Generate: generating names via classic engine",
			zap.String("surname", req.Surname),
			zap.String("gender", req.Gender),
			zap.Int("name_length", nameLength),
		)

		if s.enhancedAnalyzer != nil {
			names, err = s.enhancedAnalyzer.GenerateUnified(genOpts)
			if err != nil {
				logger.Warn("GenerateUnified failed",
					zap.Error(err),
				)
			}
		} else {
			err = errors.NewError(errors.ErrCodeInternalError, "未配置名字生成引擎")
		}
	}

	generateDuration := time.Since(generateStart)

	if err != nil {
		logger.Warn("Generate: name generation failed",
			zap.String("surname", req.Surname),
			zap.Error(err),
		)
		return nil, generateDuration, errors.NewError(errors.ErrCodeBadRequest, "无法生成符合条件的名字，请调整筛选条件")
	}

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

// generateNamesViaFate 通过 fate 引擎生成名字
func (s *NameService) generateNamesViaFate(ctx context.Context, req *GenerateRequest, baziAnalysis *bazi.BaziAnalysis) ([]name.Name, error) {
	born := time.Date(req.BirthYear, time.Month(req.BirthMonth), req.BirthDay, req.BirthHour, max(0, req.BirthMinute), 0, 0, time.UTC)

	fo := fate.NewFilterOption().
		WithMinStroke(req.MinStrokes).
		WithMaxStroke(req.MaxStrokes).
		WithGenderFilter(req.Gender).
		WithStrictness("moderate")

	// 用经典喜用神收窄候选池，保证名字五行与 API 响应 Bazi 的喜用神一致
	// （fate 引擎内部 BalanceXiYongJi 与经典 BaziAdapter 算法不同，
	//   若不传入，Top10 可能全是引擎内部喜用神五行，与响应 Bazi 矛盾）
	// 用户显式指定五行偏好时优先用用户的
	if len(req.WuxingMatch) > 0 {
		fo = fo.WithPreferredWuXing(req.WuxingMatch...)
	} else if baziAnalysis != nil && len(baziAnalysis.Xiyongshen) > 0 {
		fo = fo.WithPreferredWuXing(baziAnalysis.Xiyongshen...)
	}

	// 人名频率过滤（来自 Chinese-Names-Corpus 语料统计）
	if req.MinFrequencyTier > 0 || req.MaxFrequencyTier > 0 {
		fo = fo.WithFrequencyTier(req.MinFrequencyTier, req.MaxFrequencyTier)
	}

	session := s.fateService.engine.NewSessionWithFilter(fo.Build())

	input := &fate.Input{
		Surname:    req.Surname,
		Gender:     fate.Gender(req.Gender),
		Born:       born,
		Generation: req.Generation,
		Options: fate.GenerateOptions{
			NameLength:      req.NameLength,
			Count:           50,
			ExcludeRare:     req.ExcludeRare,
			SourceClassic:   req.SourceClassic,
			IncludePoetry:   req.IncludePoetry,
			IncludeClassic:  req.IncludeClassic,
			MeaningKeywords: req.MeaningKeywords,
			PinyinInitial:   req.PinyinInitial,
		},
		AvoidElderNames: req.AvoidElderNames,
	}

	if err := session.Start(ctx, input); err != nil {
		return nil, fmt.Errorf("会话启动失败: %w", err)
	}
	if err := session.Wait(); err != nil {
		return nil, fmt.Errorf("名字生成失败: %w", err)
	}

	output := session.Result()
	if output == nil || len(output.TopNames) == 0 {
		return nil, errors.NewError(errors.ErrCodeBadRequest, "无法生成符合条件的名字，请调整筛选条件")
	}

	return convertFateToNameNames(output.TopNames), nil
}

// convertFateToNameNames 将 fate 引擎的 NameResult 转换为旧 name.Name 结构
// 保留 fate 的多维度评分（Items 映射到各分数字段），以便 buildResponse 和前端能正常渲染
func convertFateToNameNames(results []fate.NameResult) []name.Name {
	names := make([]name.Name, len(results))
	for i, nr := range results {
		n := name.Name{
			ID:         int64(i + 1),
			Surname:    nr.Surname,
			GivenName:  nr.GivenName,
			FullName:   nr.FullName,
			Pinyin:     nr.Pinyin,
			Meaning:    nr.Meaning,
			Wuxing:     nr.WuXing,
			Strokes:    nr.Strokes,
			TotalScore: nr.Score.Total,
		}
		// 诗词出处映射
		if nr.PoetryFrom != "" {
			n.PoetrySource = nr.PoetryFrom
			if n.Meaning == "" {
				n.Meaning = "出自" + nr.PoetryFrom
			}
		}
		// 多维度评分映射（fate Rater 链产出的 Items → name.Name 分数字段）
		for k, v := range nr.Score.Items {
			switch k {
			case "五行八字":
				n.WuxingScore = v
			case "音韵":
				n.YinyunScore = v
			case "文化印象":
				n.MeaningScore = v
			case "生肖":
				n.ZodiacScore = v
			case "新颖度":
				n.NoveltyScore = v
			case "共现":
				n.BigramScore = v
			case "人名频率":
				n.FrequencyScore = v
			}
		}
		n.Reasons = nr.Reasons
		names[i] = n
	}
	return names
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
		h := s.hexagramFinder.FindByStrokes(avgStrokes)
		m := s.hexagramFinder.MatchXiyongshen(h, baziAnalysis.Xiyongshen)

		hexagramMutex.Lock()
		hexagram = h
		hexagramMatch = m
		hexagramMutex.Unlock()
	}()

	go func() {
		defer wg.Done()
		z := s.ziweiAnalyzer.Analyze(req.BirthYear, req.BirthMonth, req.BirthDay, req.BirthHour)

		ziweiMutex.Lock()
		ziweiAnalysis = z
		ziweiMutex.Unlock()
	}()

	wg.Wait()

	duration := time.Since(start)

	hexagramName := ""
	ziweiGongwei := ""
	if hexagram != nil {
		hexagramName = hexagram.Name
	}
	if ziweiAnalysis != nil {
		ziweiGongwei = ziweiAnalysis.Gongwei
	}
	logger.Info("Parallel calculation completed",
		zap.String("hexagram", hexagramName),
		zap.String("ziwei_ming_gong", ziweiGongwei),
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
	zodiacName string,
) *GenerateResponse {
	nayin := baziAnalysis.Nayin

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

// logGeneration 记录生成日志
func (s *NameService) logGeneration(
	start time.Time,
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
}

// generateNameSuggestions 生成起名建议（包级函数，供 NameService 和 FateNameService 共用）
func generateNameSuggestions(analyses []*name.NameAnalysis) []string {
	suggestions := []string{}

	// 根据评分给出建议
	lowScoreCount := 0
	for _, a := range analyses {
		if a.TotalScore < 80 {
			lowScoreCount++
		}
	}
	if lowScoreCount > len(analyses)/2 {
		suggestions = append(suggestions, "建议调整筛选条件以获得更高分的名字")
	}

	if len(suggestions) == 0 {
		suggestions = append(suggestions, "当前名字组合较为理想，可根据个人喜好进行选择")
	}

	return suggestions
}

// GetByID 根据ID获取名字
func (s *NameService) GetByID(ctx context.Context, id int64) (*name.Name, error) {
	return nil, errors.NewError(errors.ErrCodeNotFound, "暂不支持按ID查询名字，请使用生成接口")
}
