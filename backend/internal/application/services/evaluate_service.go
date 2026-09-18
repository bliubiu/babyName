package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"name/internal/application/errors"
	"name/internal/domain/bazi"
	"name/internal/domain/fate"
	"name/internal/domain/name"
	"name/internal/infrastructure/logger"

	"go.uber.org/zap"
)

// ——— 测名（给定姓名 → 完整评分报告 + 风险体检） ———
//
// 对应 docs/27 §4.1 A1+A2：用户真实动线是「先攒名字再筛」，测名是全竞品
// 标配而本项目此前完全缺失的入口。
//
// 关键设计约束（docs/24 P2-6 的教训）：测名分数必须与生成路径**同一条装配链**——
// NameCandidate 由 fate.RateGivenName 按引擎枚举口径组装，FateData 由引擎同款
// BaziAnalyzerAdapter 产出，Raters 用同一份 fate.DefaultRaters()。
// 这样「结果页的名字」与「测名页的名字」分数天然一致，不会出现同名不同分。

// EvaluateRequest 测名请求（生辰字段与 GenerateRequest 同名同义）
type EvaluateRequest struct {
	Surname    string `json:"surname" binding:"required"`
	GivenName  string `json:"given_name" binding:"required"`
	Gender     string `json:"gender" binding:"required"`
	BirthYear  int    `json:"birth_year" binding:"required"`
	BirthMonth int    `json:"birth_month" binding:"required"`
	BirthDay   int    `json:"birth_day" binding:"required"`
	BirthHour  int    `json:"birth_hour" binding:"required"`
	BirthMinute int   `json:"birth_minute"`
	// BirthLocation 出生地（用于真太阳时校正，按地点经度查表）
	BirthLocation string `json:"birth_location"`
	// BirthLongitude 显式经度（度，东经为正）；>0 时优先于地点查表
	BirthLongitude float64 `json:"birth_longitude"`
}

// EvaluateResponse 测名响应
type EvaluateResponse struct {
	FullName string `json:"full_name"`
	// Name 与 /generate 的 names[] 元素**同构**（含 score_detail / 出典 /
	// 各维度分数与依据文字），前端 NameCard/NameDetail 组件可直接复用。
	Name name.Name `json:"name"`
	// Bazi 与 /generate 响应同源（经典八字分析器）
	Bazi bazi.BaziAnalysis `json:"bazi"`
	// Zodiac 生肖（fate 排盘口径）
	Zodiac string `json:"zodiac"`
	// Risks 风险体检清单（谐音/生僻字/多音字/负面联想/门禁字/撞名热度/户籍友好度）
	Risks []fate.RiskItem `json:"risks"`
	// RiskLevel 汇总风险等级（pass / warn / fail）
	RiskLevel string `json:"risk_level"`
}

// Evaluate 测名：对给定姓名出完整评分报告与风险清单
func (s *NameService) Evaluate(ctx context.Context, req *EvaluateRequest) (*EvaluateResponse, error) {
	if s.fateService == nil {
		return nil, errors.NewError(errors.ErrCodeInternalError, "名字生成引擎未初始化")
	}

	given := strings.TrimSpace(req.GivenName)
	runes := []rune(given)
	if len(runes) == 0 || len(runes) > 2 {
		return nil, errors.NewError(errors.ErrCodeBadRequest, "名字需为 1-2 个汉字")
	}

	// 1. 响应侧 Bazi：经典分析器（真太阳时校正），与 /generate 响应同源
	greq := &GenerateRequest{
		Surname:        req.Surname,
		Gender:         req.Gender,
		BirthYear:      req.BirthYear,
		BirthMonth:     req.BirthMonth,
		BirthDay:       req.BirthDay,
		BirthHour:      req.BirthHour,
		BirthMinute:    req.BirthMinute,
		BirthLocation:  req.BirthLocation,
		BirthLongitude: req.BirthLongitude,
	}
	baziAnalysis, _, baziErr := s.performBaziAnalysis(greq)
	if baziErr != nil {
		logger.Warn("Evaluate: 八字分析失败，按空分析继续",
			zap.String("surname", req.Surname),
			zap.Error(baziErr))
	}

	// 2. 引擎侧 FateData：与生成路径同款适配器 + 真太阳时校正后的出生时刻
	sy, sm, sd, sh, smin := req.BirthYear, req.BirthMonth, req.BirthDay, req.BirthHour, max(0, req.BirthMinute)
	sy, sm, sd, sh, smin, _ = bazi.ApplyTrueSolar(sy, sm, sd, sh, smin, req.BirthLocation, req.BirthLongitude)
	born := time.Date(sy, time.Month(sm), sd, sh, smin, 0, 0, time.UTC)
	fateData, err := NewBaziAnalyzerAdapter().Analyze(born, fate.Gender(req.Gender))
	if err != nil {
		return nil, fmt.Errorf("八字排盘失败: %w", err)
	}

	// 3. 候选池同款过滤器（笔画口径与引擎枚举一致；收窄/频率条件不影响单名装配）
	filter := fate.NewFilterOption().
		WithStrictness("moderate").
		WithGenderFilter(req.Gender).
		Build()

	provider := &HanziDataProvider{}
	cand, score, err := fate.RateGivenName(provider, filter, req.Surname, given, fateData, fate.DefaultRaters())
	if err != nil {
		return nil, errors.NewError(errors.ErrCodeBadRequest, err.Error())
	}

	// 4. 组装 name.Name（与 convertFateToNameNames 同构的转换，供前端直接复用组件）
	n := assembleEvaluatedName(provider, req.Surname, given, req.Gender, cand, score)

	// 5. 风险体检
	risks := fate.AssessNameRisks(provider, req.Surname, given)

	zodiacName := fateData.BaziInfo.Zodiac
	if s.zodiacFinder != nil {
		zodiacName = s.zodiacFinder.FindByYear(req.BirthYear)
	}

	logger.Info("Evaluate: completed",
		zap.String("full_name", n.FullName),
		zap.Float64("total", score.Total),
		zap.Int("risk_items", len(risks)),
	)

	return &EvaluateResponse{
		FullName:  n.FullName,
		Name:      n,
		Bazi:      *baziAnalysis,
		Zodiac:    zodiacName,
		Risks:     risks,
		RiskLevel: string(fate.OverallRiskLevel(risks)),
	}, nil
}

// assembleEvaluatedName 把 RateGivenName 的结果组装为 name.Name
// （与 /generate 的 names[] 元素同构），供测名与探索模式共用。
func assembleEvaluatedName(provider *HanziDataProvider, surname, given, gender string, cand *fate.NameCandidate, score fate.NameScore) name.Name {
	n := name.Name{
		ID:         1,
		Surname:    surname,
		GivenName:  given,
		FullName:   surname + given,
		Pinyin:     strings.TrimSpace(cand.Pinyin1 + " " + cand.Pinyin2),
		Gender:     gender,
		Meaning:    combineTwoMeanings(cand.Meaning1, cand.Meaning2),
		Wuxing:     cand.WuXing1 + cand.WuXing2,
		TotalScore: score.Total,
	}
	if strokes, err := fate.GivenNameStrokes(provider, surname, given); err == nil {
		n.Strokes = strokes
	}
	// 诗词出处：先预填引擎原句，再走结构化回填（与生成路径完全一致的语义）
	if cand.PoetryFrom != "" {
		n.PoetrySource = cand.PoetryFrom
		if n.Meaning == "" {
			n.Meaning = "出自" + cand.PoetryFrom
		}
	}
	enrichPoetryForName(&n, given)
	// 八维分数 + 依据文字 + score_detail，与 /generate、/generate/analysis 同源映射
	applyFateScoreDetail(&n, score)
	return n
}

// combineTwoMeanings 组合两字释义为名字寓意（与引擎 combineCharMeanings 同语义：
// 每字按 rune 截断至 60 字，双名以「；」连接）。独立实现因引擎侧函数未导出。
func combineTwoMeanings(m1, m2 string) string {
	trunc := func(m string) string {
		m = strings.TrimSpace(m)
		if m == "" {
			return ""
		}
		r := []rune(m)
		if len(r) > 60 {
			return string(r[:60]) + "…"
		}
		return m
	}
	c1 := trunc(m1)
	if m2 == "" {
		return c1
	}
	c2 := trunc(m2)
	switch {
	case c1 == "":
		return c2
	case c2 == "":
		return c1
	default:
		return c1 + "；" + c2
	}
}
