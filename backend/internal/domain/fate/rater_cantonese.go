package fate

import (
	"math"
)

// CantoneseBlendWeight 粤语分并入综合分的固定权重（开关开启时）。
//
// 取值 0.10：轻量并入，让粤语风险/音律对排序有可见但不喧宾夺主的影响；
// 关闭开关时综合分完全由普通话八维决定（零影响）。
const CantoneseBlendWeight = 0.10

// DefaultCantoneseRaters 返回默认的粤语评分器列表（权重内部归一，和为 1.0）。
func DefaultCantoneseRaters() []Rater {
	return []Rater{
		NewYueYinYunRaterWithWeight(0.6),    // 60% 粤语音律
		NewYueHomophoneRaterWithWeight(0.4), // 40% 粤语谐音
	}
}

// RateCantonese 计算名字的粤语分（完整明细版）。
//
// 与 RateName 同构，但输入的 raters 应为 DefaultCantoneseRaters() 返回的粤语评分器。
// 不做非策展双名封顶（粤语维度不参与 cappedDimNames）。
func RateCantonese(candidate *NameCandidate, fateData *FateData, raters []Rater) NameScore {
	prevSkip := candidate.skipDetail
	candidate.skipDetail = false
	defer func() { candidate.skipDetail = prevSkip }()

	items := make(map[string]float64, len(raters))
	details := make(map[string]string, len(raters))
	var total float64
	for _, r := range raters {
		rating := r.Rate(candidate, fateData)
		items[r.Name()] = rating.Score
		details[r.Name()] = rating.Detail
		total += rating.Score * r.Weight()
	}
	total = clampScore(total)
	total = math.Round(total*10) / 10
	return NameScore{
		Total:   total,
		Grade:   scoreToGrade(total),
		Items:   items,
		Details: details,
	}
}

// RateCantoneseScore 只计算粤语总分，不构造维度明细与文案（枚举热路径专用）。
//
// 与 RateNameScore 同构：返回分值与 RateCantonese 完全一致。
func RateCantoneseScore(candidate *NameCandidate, fateData *FateData, raters []Rater) float64 {
	prevSkip := candidate.skipDetail
	candidate.skipDetail = true
	defer func() { candidate.skipDetail = prevSkip }()

	var total float64
	for _, r := range raters {
		total += r.Rate(candidate, fateData).Score * r.Weight()
	}
	total = clampScore(total)
	return math.Round(total*10) / 10
}
