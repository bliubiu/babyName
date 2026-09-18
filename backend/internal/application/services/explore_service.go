package services

import (
	"context"

	"name/internal/application/errors"
	"name/internal/domain/fate"
	"name/internal/domain/name"
)

// ——— 探索模式（换一批） ———
//
// 对应 docs/27 §4.2 B2（借鉴 fate v4 的 Explore()）：Top-N 之外随机采样
// 「另一批」候选，解决「看来看去就这几个」。
//
// 实现要点：
//   - 生成完成时已把 Top-N 全部 MarkShown（registerExploreContext），因此
//     ExcellentTable.Explore 只会返回未上过榜的候选——与本次结果**零交集**；
//   - 每个采样条目用与测名同一条装配链（fate.RateGivenName）回算完整八维
//     分数与依据文字，保证「换一批」出来的名字与结果页/测名页同分同口径；
//   - Explore 自带已展示去重与 100 名展示上限，同一会话反复调用不重复。

// ExploreRequest 换一批请求
type ExploreRequest struct {
	// GenerationID 生成响应返回的会话标识
	GenerationID string `json:"generation_id" binding:"required"`
	// Count 换一批数量（默认 10，上限 30）
	Count int `json:"count"`
}

// ExploreResponse 换一批响应
type ExploreResponse struct {
	GenerationID string      `json:"generation_id"`
	Names        []name.Name `json:"names"`
	// Remaining 该会话还能换出的剩余候选数（0 表示已取尽）
	Remaining int `json:"remaining"`
}

// ExploreNames 探索模式：从指定生成会话的候选表中随机采样一批未展示过的名字
func (s *NameService) ExploreNames(ctx context.Context, req *ExploreRequest) (*ExploreResponse, error) {
	if req == nil || req.GenerationID == "" {
		return nil, errors.NewError(errors.ErrCodeBadRequest, "缺少 generation_id")
	}
	count := req.Count
	if count <= 0 {
		count = 10
	}
	if count > 30 {
		count = 30
	}

	s.exploreMu.Lock()
	ec, ok := s.exploreStore[req.GenerationID]
	s.exploreMu.Unlock()
	if !ok {
		return nil, errors.NewError(errors.ErrCodeBadRequest, "生成会话不存在或已过期，请重新生成")
	}

	entries := ec.table.Explore(count, nil)
	if len(entries) == 0 {
		return &ExploreResponse{
			GenerationID: req.GenerationID,
			Names:        []name.Name{},
			Remaining:    0,
		}, nil
	}

	// 回算完整明细：与测名同一条装配链（同 FateData / 同 Raters / 同过滤器口径）
	filter := fate.NewFilterOption().
		WithStrictness("moderate").
		WithGenderFilter(ec.gender).
		Build()
	provider := &HanziDataProvider{}

	names := make([]name.Name, 0, len(entries))
	for _, e := range entries {
		given := e.Char1 + e.Char2
		cand, score, err := fate.RateGivenName(provider, filter, ec.surname, given, ec.fateData, fate.DefaultRaters())
		if err != nil {
			// 个别条目回算失败（如字库异常）不影响整批，跳过即可
			continue
		}
		n := assembleEvaluatedName(provider, ec.surname, given, ec.gender, cand, score)
		names = append(names, n)
	}

	return &ExploreResponse{
		GenerationID: req.GenerationID,
		Names:        names,
		Remaining:    ec.table.Len() - ec.table.ShownCount(),
	}, nil
}
