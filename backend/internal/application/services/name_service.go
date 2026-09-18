package services

import (
	"context"
	"fmt"
	"strings"
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
	baziAnalyzer   bazi.BaziAnalyzer
	hexagramFinder yijing.HexagramFinder
	ziweiAnalyzer  ziwei.ZiweiAnalyzer
	zodiacFinder   zodiac.ZodiacFinder
	fateService    *FateNameService // fate 路径委托服务（可选）
	cache          cache.Cache
	nameDB         *name.NameDB // 候选名库管理器（API 层共享自学习精选名数据）
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

// WithNameDB 设置候选名库管理器（用于 API 层共享自学习精选名数据）
func WithNameDB(db *name.NameDB) NameServiceOption {
	return func(s *NameService) { s.nameDB = db }
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
	return s.nameDB
}

// GenerateRequest 生成名字请求
type GenerateRequest struct {
	Surname            string `json:"surname" binding:"required"`
	Generation         string `json:"generation"`
	GenerationPosition string `json:"generation_position"`
	Gender             string `json:"gender" binding:"required"`
	BirthYear          int    `json:"birth_year" binding:"required"`
	BirthMonth         int    `json:"birth_month" binding:"required"`
	BirthDay           int    `json:"birth_day" binding:"required"`
	BirthHour          int    `json:"birth_hour" binding:"required"`
	BirthMinute        int    `json:"birth_minute"`
	BirthLocation      string `json:"birth_location"`
	// BirthLongitude 显式出生地经度（度，东经为正）；>0 时优先于地点查表
	BirthLongitude float64  `json:"birth_longitude"`
	BirthType      string   `json:"birth_type"`
	NameType       string   `json:"name_type"`
	Preferences    []string `json:"preferences"`
	NameLength     int      `json:"name_length"`
	ExcludeRare    bool     `json:"exclude_rare"`
	WuxingMatch    []string `json:"wuxing_match"`
	SourceClassic  string   `json:"source_classic"`
	// 新增筛选条件
	MinStrokes      int      `json:"min_strokes"`
	MaxStrokes      int      `json:"max_strokes"`
	IncludePoetry   bool     `json:"include_poetry"`
	IncludeClassic  bool     `json:"include_classic"`
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
	Bazi          bazi.BaziAnalysis     `json:"bazi"`
	Nayin         string                `json:"nayin"`
	Zodiac        string                `json:"zodiac"`
	Hexagram      *yijing.Hexagram      `json:"hexagram"`
	HexagramMatch *yijing.HexagramMatch `json:"hexagram_match,omitempty"`
	Ziwei         *ziwei.ZiweiAnalysis  `json:"ziwei,omitempty"`
	Names         []name.Name           `json:"names"`
}

// GenerateWithAnalysisResponse 带详细分析的名字生成响应
type GenerateWithAnalysisResponse struct {
	Bazi        bazi.BaziAnalysis    `json:"bazi"`
	Nayin       string               `json:"nayin"`
	Zodiac      string               `json:"zodiac"`
	Hexagram    *yijing.Hexagram     `json:"hexagram"`
	Ziwei       *ziwei.ZiweiAnalysis `json:"ziwei,omitempty"`
	Names       []*name.NameAnalysis `json:"names"`
	Suggestions []string             `json:"suggestions"`
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
	if s.fateService == nil {
		return nil, errors.NewError(errors.ErrCodeInternalError, "名字生成引擎未初始化")
	}
	return s.fateService.GenerateWithAnalysis(ctx, req)
}

// performBaziAnalysis 执行八字分析（自动应用真太阳时校正）
// 返回分析结果和耗时；分析失败时返回空 BaziAnalysis 并附带 error，
// 调用者应记录日志后继续（graceful degradation），不阻断生成流程。
func (s *NameService) performBaziAnalysis(req *GenerateRequest) (*bazi.BaziAnalysis, time.Duration, error) {
	baziStart := time.Now()

	y, mo, d, h, mi := req.BirthYear, req.BirthMonth, req.BirthDay, req.BirthHour, max(0, req.BirthMinute)
	// 真太阳时：按出生地经度校正；未收录地点则使用钟表时间
	y, mo, d, h, mi, solarInfo := bazi.ApplyTrueSolar(y, mo, d, h, mi, req.BirthLocation, req.BirthLongitude)
	if solarInfo != nil && solarInfo.Enabled {
		logger.Info("真太阳时校正",
			zap.String("original", solarInfo.Original),
			zap.String("corrected", solarInfo.Corrected),
			zap.Float64("longitude", solarInfo.Longitude),
			zap.Bool("cross_shichen", solarInfo.CrossShichen),
		)
	}

	baziAnalysis, err := s.baziAnalyzer.Analyze(y, mo, d, h, mi)
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

	var names []name.Name
	var err error

	// fate 引擎（唯一生成引擎；FateNameService 未配置时无法生成）
	// 新引擎具备更完善的 Rater 评分链、避讳长辈、负面反馈等能力
	if s.fateService == nil {
		err = errors.NewError(errors.ErrCodeInternalError, "名字生成引擎未初始化")
	} else {
		names, err = s.generateNamesViaFate(ctx, req, baziAnalysis)
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

	// 经典来源/诗词字注入候选池（与 /generate/analysis 路径对齐）：
	// 引擎只消费 Options.ExtraChars 而不读 SourceClassic，旧路径此前漏传导致
	// 前端选择《论语》等经典来源完全失效。复用 FateNameService.resolveExtraChars。
	var extraChars []*fate.Character
	if s.fateService != nil {
		extraChars = s.fateService.resolveExtraChars(req)
	}

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
			ExtraChars:      extraChars,
		},
		AvoidElderNames: req.AvoidElderNames,
	}

	if err := session.Start(ctx, input); err != nil {
		return nil, fmt.Errorf("会话启动失败: %w", err)
	}
	if err := session.Wait(); err != nil {
		return nil, fmt.Errorf("名字生成失败: %w", err)
	}
	// 同 FateNameService：ctx 到期/取消后引擎返回的是被截断的榜单，
	// 必须转成错误交给 handler 映射为 503，而不是当作正常结果返回。
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("名字生成被中止: %w", err)
	}

	output := session.Result()
	if output == nil || len(output.TopNames) == 0 {
		return nil, errors.NewError(errors.ErrCodeBadRequest, "无法生成符合条件的名字，请调整筛选条件")
	}

	return convertFateToNameNames(output.TopNames, req.Gender), nil
}

// convertFateToNameNames 将 fate 引擎的 NameResult 转换为旧 name.Name 结构
// 保留 fate 的多维度评分（Items 映射到各分数字段），以便 buildResponse 和前端能正常渲染
// gender 为请求方性别（fate.NameResult 无该字段），并同步做诗词出处结构化回填
// （enrichPoetryForName，与 GenerateWithAnalysis 路径对齐）。
func convertFateToNameNames(results []fate.NameResult, gender string) []name.Name {
	names := make([]name.Name, len(results))
	for i, nr := range results {
		n := name.Name{
			ID:         int64(i + 1),
			Surname:    nr.Surname,
			GivenName:  nr.GivenName,
			FullName:   nr.FullName,
			Pinyin:     nr.Pinyin,
			Gender:     gender,
			Meaning:    nr.Meaning,
			Wuxing:     nr.WuXing,
			Strokes:    nr.Strokes,
			TotalScore: nr.Score.Total,
		}
		// 诗词出处映射（临时预填引擎原句，enrichPoetryForName 命中索引时覆盖为典籍名）
		if nr.PoetryFrom != "" {
			n.PoetrySource = nr.PoetryFrom
			if n.Meaning == "" {
				n.Meaning = "出自" + nr.PoetryFrom
			}
		}
		// 出典完整结构化回填（作品·篇目·原句·作者·朝代·全诗），与 analysis 路径一致
		enrichPoetryForName(&n, nr.GivenName)
		// 多维度评分映射：Items 分数 + Details 依据文字 + score_detail 明细，
		// 统一由 applyFateScoreDetail 处理，与 /generate/analysis 路径同源。
		// （此前 /generate 只映射了分数、丢弃了依据文字，导致前端
		//   NameCard/NameDetail 的「优先 score_detail」分支永不执行，
		//   FullReport 只能本地自造命理文案 —— docs/24 P2-5。）
		applyFateScoreDetail(&n, nr.Score)
		n.Reasons = nr.Reasons
		names[i] = n
	}
	return names
}

// calculateHexagramAndZiweiParallel 并行计算易经卦象和紫微斗数
// 姓名卦采用梅花易数：姓笔画→上卦，名笔画→下卦（取首个候选名）
func (s *NameService) calculateHexagramAndZiweiParallel(names []name.Name, baziAnalysis *bazi.BaziAnalysis, req *GenerateRequest) (*yijing.Hexagram, *yijing.HexagramMatch, *ziwei.ZiweiAnalysis, time.Duration) {
	start := time.Now()

	// 姓氏笔画（康熙优先）
	surnameStrokes := 0
	if l1, l2, err := (&HanziDataProvider{}).GetSurnameStrokes(req.Surname); err == nil {
		surnameStrokes = l1 + l2
	}
	if surnameStrokes == 0 {
		surnameStrokes = len([]rune(req.Surname)) // 极端兜底
	}

	// 名笔画：取首个候选的名部分笔画（全名笔画 - 姓笔画）
	givenStrokes := 0
	if len(names) > 0 {
		givenStrokes = names[0].Strokes - surnameStrokes
	}
	if givenStrokes <= 0 {
		givenStrokes = 10
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
		h := s.hexagramFinder.FindByMeihuaName(surnameStrokes, givenStrokes)
		m := s.hexagramFinder.MatchXiyongshen(h, baziAnalysis.Xiyongshen)

		hexagramMutex.Lock()
		hexagram = h
		hexagramMatch = m
		hexagramMutex.Unlock()
	}()

	go func() {
		defer wg.Done()
		z := s.ziweiAnalyzer.Analyze(req.BirthYear, req.BirthMonth, req.BirthDay, req.BirthHour, req.Gender)

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
		zap.Int("surname_strokes", surnameStrokes),
		zap.Int("given_strokes", givenStrokes),
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

// applyFateScoreDetail 把 fate 引擎的 NameScore（Items 分数 + Details 依据文字）
// 平铺到 name.Name 的既有字段，并生成 score_detail 明细。
//
// 为什么要走这一层：`name.Name` 是 /names/generate 的对外结构，字段名是历史命名的
// （wuxing_score / yinyun / meaning_detail …），而引擎侧用中文维度名聚合。
// 两者之间的映射此前散落在两个路径各自的 for 循环里，已出现漏映射（三才分）与
// 漏依据文字（Details 全程只映射分数）两处漂移，故收敛到一处。
func applyFateScoreDetail(n *name.Name, score fate.NameScore) {
	if n == nil {
		return
	}

	// ① 各维度分数 → 对应分数字段
	for k, v := range score.Items {
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
		case "三才":
			n.SancaiScore = v
		}
	}

	// ② 各维度依据文字 → 对应文字字段（前端按既有字段名直接渲染，无需改组件）
	// 空/纯空白视为「引擎未提供」，保留调用方已填的文案（如 buildResponse 阶段的兜底）
	for k, v := range score.Details {
		if strings.TrimSpace(v) == "" {
			continue
		}
		switch k {
		case "五行八字":
			n.WuxingAnalysis = v
		case "音韵":
			n.Yinyun = v
		case "文化印象":
			n.MeaningDetail = v
		case "三才":
			n.SancaiAnalysis = v
		}
	}

	// ③ 评分明细（维度名/分数/依据文字），与 /generate/analysis 同源同序
	n.ScoreDetail = buildScoreDetail(score.Items, score.Details)
}

// buildScoreDetail 按固定维度顺序输出评分明细（供 /generate 与 /generate/analysis 共用）。
// 顺序与 DefaultRaters 权重降序一致，确保推荐名展示稳定；维度内聚为
// 「维度名/分数/依据文字」，前端无需硬编码维度即可渲染评分分解。
func buildScoreDetail(items map[string]float64, details map[string]string) []name.ScoreDetailItem {
	detailOrder := []string{"五行八字", "文化印象", "音韵", "新颖度", "生肖", "共现", "三才", "人名频率"}
	var out []name.ScoreDetailItem
	for _, dim := range detailOrder {
		score, ok := items[dim]
		if !ok {
			continue
		}
		out = append(out, name.ScoreDetailItem{
			Name:   dim,
			Score:  score,
			Detail: details[dim],
		})
	}
	return out
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
