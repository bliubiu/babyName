package fate

import (
	"fmt"
	"strings"

	"name/internal/domain/hanzi"
)

// FrequencyRater 人名频率评分器
//
// 基于 Chinese-Names-Corpus 120万人名语料统计的真实人名使用频率。
// 评分逻辑：
//   - 单字频率分：NameFreqTier 映射到 0-100 分（Tier5=100, Tier1=20）
//   - 双字组合频率分：bigram Tier 映射到 0-100 分
//   - 综合分 = 单字平均 × 0.6 + 组合分 × 0.4
//
// 未收录的字返回默认分 50（中性），不影响整体排序。
//
// 设计目标：
//   - 作为辅助信号，让"人名中确实常用的字"获得轻微加分
//   - 权重控制在 5% 以内，不主导排序（避免推荐"梓轩""梓涵"等爆款）
//   - 高频信号主要用于候选池排序（预筛选），而非最终评分
type FrequencyRater struct {
	weight float64
}

func NewFrequencyRater() *FrequencyRater {
	return &FrequencyRater{weight: 0.05}
}

func NewFrequencyRaterWithWeight(w float64) *FrequencyRater {
	return &FrequencyRater{weight: w}
}

func (r *FrequencyRater) Name() string   { return "人名频率" }
func (r *FrequencyRater) Weight() float64 { return r.weight }

func (r *FrequencyRater) Rate(candidate *NameCandidate, _ *FateData) NameRating {
	// 如果频率数据库未加载，返回默认分
	if !hanzi.FrequencyDBLoaded() {
		return NameRating{Score: 50, Detail: "人名频率数据未加载"}
	}

	var details []string
	var scores []float64

	// 1. 逐字频率评分
	tier1 := candidate.NameFreqTier1
	tier2 := candidate.NameFreqTier2

	charScore1 := tierToScore(tier1)
	charScore2 := tierToScore(tier2)

	// 单名只看一个字
	if candidate.Char2 == "" {
		scores = append(scores, charScore1)
		if tier1 > 0 {
			details = append(details, fmt.Sprintf("「%s」频率Tier%d", candidate.Char1, tier1))
		} else {
			details = append(details, fmt.Sprintf("「%s」未收录", candidate.Char1))
		}
	} else {
		// 双名取两字平均
		avg := (charScore1 + charScore2) / 2
		scores = append(scores, avg)
		if tier1 > 0 && tier2 > 0 {
			details = append(details, fmt.Sprintf("「%s」Tier%d「%s」Tier%d", candidate.Char1, tier1, candidate.Char2, tier2))
		} else if tier1 > 0 {
			details = append(details, fmt.Sprintf("「%s」Tier%d「%s」未收录", candidate.Char1, tier1, candidate.Char2))
		} else if tier2 > 0 {
			details = append(details, fmt.Sprintf("「%s」未收录「%s」Tier%d", candidate.Char1, candidate.Char2, tier2))
		} else {
			details = append(details, fmt.Sprintf("「%s」「%s」均未收录", candidate.Char1, candidate.Char2))
		}
	}

	// 2. 双字组合频率评分（仅双名）
	if candidate.Char2 != "" {
		bigram := hanzi.GetBigramFrequency(candidate.Char1, candidate.Char2)
		if bigram != nil {
			bigramScore := tierToScore(bigram.Tier)
			scores = append(scores, bigramScore)
			details = append(details, fmt.Sprintf("组合「%s%s」Tier%d(%d次)",
				candidate.Char1, candidate.Char2, bigram.Tier, bigram.Count))
		} else {
			// 组合未收录 → 中性 50 分
			scores = append(scores, 50)
		}
	}

	// 3. 计算综合分
	finalScore := averageScores(scores)

	return NameRating{
		Score:  clampScore(finalScore),
		Detail: strings.Join(details, "；"),
	}
}

// tierToScore 将频率档位映射到 0-100 分
//
// Tier 5（Top 1%）→ 100
// Tier 4（Top 5%）→ 80
// Tier 3（Top 20%）→ 60
// Tier 2（Top 50%）→ 40
// Tier 1（底部 50%）→ 20
// 未收录（0）→ 50（中性）
func tierToScore(tier int) float64 {
	switch tier {
	case 5:
		return 100
	case 4:
		return 80
	case 3:
		return 60
	case 2:
		return 40
	case 1:
		return 20
	default:
		return 50 // 未收录 → 中性
	}
}

// averageScores 计算分数切片的加权平均
func averageScores(scores []float64) float64 {
	if len(scores) == 0 {
		return 50
	}
	var sum float64
	for _, s := range scores {
		sum += s
	}
	return sum / float64(len(scores))
}
