package services

import (
	"context"
	"fmt"
	"time"

	"name/internal/application/errors"
	"name/internal/domain/bazi"
	"name/internal/domain/fate"
	"name/internal/domain/hanzi"
	"name/internal/domain/name"
	"name/internal/infrastructure/logger"

	"go.uber.org/zap"
)

// FateNameService 基于 fate 引擎的名字服务
// 从 NameService 拆分而来，专职处理 fate 路径的带分析生成，
// 避免单个 NameService 承担过多职责（God Object 倾向）。
type FateNameService struct {
	engine fate.Fate
}

// NewFateNameService 创建 fate 名字服务
func NewFateNameService(engine fate.Fate) *FateNameService {
	return &FateNameService{engine: engine}
}

// buildFilterOption 根据请求构建 FilterOption
// 优先使用请求的 WuxingMatch（用户显式指定五行）；
// 否则按喜用神五行收紧候选池（缩小全量枚举规模，提升性能）。
func (s *FateNameService) buildFilterOption(req *GenerateRequest) fate.Filter {
	fo := fate.NewFilterOption().
		WithMinStroke(req.MinStrokes).
		WithMaxStroke(req.MaxStrokes).
		WithGenderFilter(req.Gender).
		WithStrictness("moderate")

	// 用户显式指定五行偏好时，直接以其收窄候选池（避免全量枚举 8105²）
	if len(req.WuxingMatch) > 0 {
		fo = fo.WithPreferredWuXing(req.WuxingMatch...)
	}

	return fo.Build()
}

// GenerateWithAnalysis 使用 fate 引擎生成带详细分析的名字
func (s *FateNameService) GenerateWithAnalysis(ctx context.Context, req *GenerateRequest) (*GenerateWithAnalysisResponse, error) {
	born := time.Date(req.BirthYear, time.Month(req.BirthMonth), req.BirthDay, req.BirthHour, req.BirthMinute, 0, 0, time.UTC)

	session := s.engine.NewSessionWithFilter(
		s.buildFilterOption(req),
	)

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
			ExtraChars:      s.resolveExtraChars(req),
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
	if output == nil {
		return nil, errors.NewError(errors.ErrCodeBadRequest, "无法生成符合条件的名字，请调整筛选条件")
	}

	// 将 fate output 转换为服务层响应
	names := make([]*name.NameAnalysis, 0, len(output.TopNames))
	for _, nr := range output.TopNames {
		na := &name.NameAnalysis{
			Surname:    nr.Surname,
			GivenName:  nr.GivenName,
			FullName:   nr.FullName,
			Pinyin:     nr.Pinyin,
			Strokes:    nr.Strokes,
			Wuxing:     nr.WuXing,
			TotalScore: nr.Score.Total,
		}
		// 映射 fate 各维度评分到 NameAnalysis 字段
		for k, v := range nr.Score.Items {
			switch k {
			case "五行八字":
				na.WuxingScore = v
			case "音韵":
				na.YinyunScore = v
			case "生肖":
				na.ZodiacScore = v
			case "文化印象":
				na.MeaningScore = v
			case "新颖度":
				na.NoveltyScore = v
			case "共现":
				na.BigramScore = v
			}
		}
		// 诗词出处回填（engine 构建 NameResult 时未设 PoetryFrom，需单独传递）
		if nr.PoetryFrom != "" {
			na.PoetrySource = nr.PoetryFrom
		}
		names = append(names, na)
	}

	response := &GenerateWithAnalysisResponse{
		Names:       names,
		Suggestions: generateNameSuggestions(names),
	}

	if output.FateData != nil {
		// 用 fate 的八字数据填充响应
		xiYongShen := output.FateData.WuXingXiji.XiYongShen
		nayin := ""
		if len(output.FateData.BaziInfo.NaYin) > 0 {
			nayin = output.FateData.BaziInfo.NaYin[0]
		}
		baziAnalysis := bazi.BaziAnalysis{
			Xiyongshen:   xiYongShen,
			Rishou:       output.FateData.WuXingXiji.RiZhu,
			RishouWuxing: output.FateData.WuXingXiji.RiZhuWuXing,
			Nayin:        nayin,
		}

		// 填充四柱（年/月/日/时）和五行分布
		fourPillars := output.FateData.BaziInfo.FourPillars
		if len(fourPillars) == 4 {
			baziAnalysis.Bazi = bazi.Bazi{
				Year:        fourPillars[0],
				Month:       fourPillars[1],
				Day:         fourPillars[2],
				Hour:        fourPillars[3],
				YearGanzhi:  fourPillars[0],
				MonthGanzhi: fourPillars[1],
				DayGanzhi:   fourPillars[2],
				HourGanzhi:  fourPillars[3],
			}
			// 统计五行分布：天干+地支各算一个，共 8 个五行
			for _, pillar := range fourPillars {
				runes := []rune(pillar)
				if len(runes) < 2 {
					continue
				}
				tg := string(runes[0])
				dz := string(runes[1])
				if wx, ok := bazi.WuxingMap[tg]; ok {
					incrementWuxing(&baziAnalysis.Wuxing, wx)
				}
				if wx, ok := bazi.DizhiWuxingMap[dz]; ok {
					incrementWuxing(&baziAnalysis.Wuxing, wx)
				}
			}
		}

		response.Bazi = baziAnalysis
		response.Nayin = nayin
		response.Zodiac = output.FateData.BaziInfo.Zodiac
	}

	logger.Info("GenerateWithAnalysis: completed via fate engine",
		zap.String("surname", req.Surname),
		zap.Int("name_count", len(names)),
	)

	return response, nil
}

// incrementWuxing 按五行名称递增 WuxingResult 对应字段
func incrementWuxing(w *bazi.WuxingResult, wuxing string) {
	switch wuxing {
	case "金":
		w.Jin++
	case "木":
		w.Mu++
	case "水":
		w.Shui++
	case "火":
		w.Huo++
	case "土":
		w.Tu++
	}
}

// resolveExtraChars 根据请求的诗词/经典来源解析额外候选字
// 将 SourceClassic / IncludePoetry / IncludeClassic 转为 []*fate.Character，
// 供 engine.generate() 合入主候选池（"加字不缩池"策略）。
func (s *FateNameService) resolveExtraChars(req *GenerateRequest) []*fate.Character {
	if req.SourceClassic == "" && !req.IncludePoetry && !req.IncludeClassic {
		return nil
	}

	// 1. 按来源收集字名列表
	var charNames []string
	if req.SourceClassic != "" {
		charNames = append(charNames, name.GetPoetryNames(req.Gender, req.SourceClassic)...)
	}
	if req.IncludePoetry && req.SourceClassic == "" {
		// IncludePoetry=true 但未指定具体来源时，取全量诗词字
		charNames = append(charNames, name.GetPoetryNames(req.Gender, "poetry")...)
	}
	if req.IncludeClassic {
		charNames = append(charNames, name.GetClassicNames(req.Gender)...)
	}

	if len(charNames) == 0 {
		return nil
	}

	// 2. 去重后转为 fate.Character（查 HanziData 获取完整属性）
	seen := make(map[string]bool, len(charNames))
	result := make([]*fate.Character, 0, len(charNames))
	for _, cn := range charNames {
		if seen[cn] {
			continue
		}
		seen[cn] = true
		h, ok := hanzi.HanziData[cn]
		if !ok {
			continue
		}
		result = append(result, hanziToCharacter(h))
	}
	return result
}
