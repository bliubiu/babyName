package services

import "name/internal/domain/fate"

// baseFilterOption 两条生成路径共用的基础过滤选项。
//
// 关键约束：喜用神**不得**在这里硬收窄候选池。喜用神是评分/偏好口径，
// 引擎会按「喜用神 + 生助喜用神」收窄并带 <80 字的降级保护；若在 SQL
// 或 CheckCharacter 层提前硬过滤，生助五行、典籍加字、用户点选字都会被
// 静默剔除（历史缺陷：方案B 分支因此不可达）。
// 只有用户显式指定 WuxingMatch 时才做硬收窄——那是明确的用户要求。
func baseFilterOption(req *GenerateRequest) fate.FilterOption {
	fo := fate.NewFilterOption().
		WithMinStroke(req.MinStrokes).
		WithMaxStroke(req.MaxStrokes).
		WithGenderFilter(req.Gender).
		WithStrictness("moderate")

	if len(req.WuxingMatch) > 0 {
		fo = fo.WithPreferredWuXing(req.WuxingMatch...)
	}

	// 人名频率过滤（来自 Chinese-Names-Corpus 语料统计）
	if req.MinFrequencyTier > 0 || req.MaxFrequencyTier > 0 {
		fo = fo.WithFrequencyTier(req.MinFrequencyTier, req.MaxFrequencyTier)
	}

	return fo
}
