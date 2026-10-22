package fate

import (
	"strings"

	"name/internal/domain/yue"
)

// YueHomophoneRater 粤语谐音风险评分
//
// 是普通话 YinYunRater 内嵌的「逐字不吉谐音 + 连读不良组合」检测，在粤语音系
// （粤拼 Jyutping）上的平行实现。检测范围：
//   - 粤语逐字谐音：yue.CheckNameYueHomophones（含通用不吉字本字豁免 +
//     明确「好字→不雅谐音」配对点名 + 懒音近音歧义 warn）
//   - 粤语连读不良词：yue.CheckYueBadCombo（含姓氏拼音，滑窗查表）
//
// 关于「粤普双音兼容」：普通话的谐音风险已由主体系 YinYunRater 覆盖
// （CheckAllBadHomophones / CheckBadPinyinCombo），若此处再重复检测普通话会
// 造成双计，使粤语子分的展示失真。因此本评分器只计「粤语特有」风险，而
// 普通话风险由主分承担——两者在综合分层面自然「双音兼容」，互不掩盖。
//
// 该评分器独立于普通话八维体系，经 RateCantonese 聚合成「粤语分」单独展示，
// 并在开启粤语评分开关时以固定权重轻量并入综合分。
type YueHomophoneRater struct {
	weight float64
}

func NewYueHomophoneRater() *YueHomophoneRater {
	return &YueHomophoneRater{weight: 0.4}
}

func NewYueHomophoneRaterWithWeight(w float64) *YueHomophoneRater {
	return &YueHomophoneRater{weight: w}
}

func (r *YueHomophoneRater) Name() string    { return "粤语谐音" }
func (r *YueHomophoneRater) Weight() float64 { return r.weight }

func (r *YueHomophoneRater) Rate(candidate *NameCandidate, fateData *FateData) NameRating {
	// 音律维度以「风险扣分」衡量：基础分 100，命中不良谐音按比例扣减。
	score := 100.0
	ds := newDetailSink(candidate)

	jp1 := candidate.Jyutping1
	jp2 := candidate.Jyutping2
	sjp := candidate.SurnameJyutping

	// 粤拼数据缺失（三字段全空）→ 降级为中性分，不计风险也不误报
	if jp1 == "" && jp2 == "" && sjp == "" {
		return NameRating{Score: 75, Detail: "粤拼数据暂缺，谐音维度以中性分计"}
	}

	// 1) 粤语逐字谐音（不含姓氏：姓氏参与的连读不雅交由下方连读检测）
	pairs := make([]string, 0, 4)
	if jp1 != "" {
		pairs = append(pairs, jp1, candidate.Char1)
	}
	if jp2 != "" {
		pairs = append(pairs, jp2, candidate.Char2)
	}
	if len(pairs) > 0 {
		hits := yue.CheckNameYueHomophones(pairs...)
		for _, h := range hits {
			switch h.Severity {
			case "fail":
				score -= 25
				ds.add(h.Description)
			case "warn":
				score -= 8
				ds.add(h.Description)
			}
		}
	}

	// 2) 粤语连读不良组合（含姓氏拼音）
	givenJps := make([]string, 0, 2)
	if jp1 != "" {
		givenJps = append(givenJps, jp1)
	}
	if jp2 != "" {
		givenJps = append(givenJps, jp2)
	}
	if len(givenJps) > 0 {
		if hit, desc := yue.CheckYueBadCombo(sjp, givenJps...); hit {
			score -= 20
			ds.add(desc)
		}
	}

	score = clampScore(score)
	detail := ds.String("粤语读音无不良谐音")
	return NameRating{Score: score, Detail: detail}
}

// CantoneseRiskSummary 粤语谐音风险文字汇总（供风险体检 / 结果展示复用）。
//
// 返回 (是否有 fail 级命中, 是否有 warn 级命中, 描述列表)。空粤拼数据时不报风险。
func CantoneseRiskSummary(candidate *NameCandidate) (hasFail, hasWarn bool, descs []string) {
	jp1 := candidate.Jyutping1
	jp2 := candidate.Jyutping2
	sjp := candidate.SurnameJyutping
	if jp1 == "" && jp2 == "" && sjp == "" {
		return false, false, nil
	}
	pairs := make([]string, 0, 4)
	if jp1 != "" {
		pairs = append(pairs, jp1, candidate.Char1)
	}
	if jp2 != "" {
		pairs = append(pairs, jp2, candidate.Char2)
	}
	if len(pairs) > 0 {
		for _, h := range yue.CheckNameYueHomophones(pairs...) {
			descs = append(descs, h.Description)
			if h.Severity == "fail" {
				hasFail = true
			} else {
				hasWarn = true
			}
		}
	}
	givenJps := make([]string, 0, 2)
	if jp1 != "" {
		givenJps = append(givenJps, jp1)
	}
	if jp2 != "" {
		givenJps = append(givenJps, jp2)
	}
	if len(givenJps) > 0 {
		if hit, desc := yue.CheckYueBadCombo(sjp, givenJps...); hit {
			descs = append(descs, desc)
			// 连读不良词按 fail 级处理（与普通话连读检测一致）
			hasFail = true
		}
	}
	return hasFail, hasWarn, descs
}

// joinCantoneseRisk 把粤语风险描述按「；」连接（空时返回 fallback）。
func joinCantoneseRisk(descs []string, fallback string) string {
	if len(descs) == 0 {
		return fallback
	}
	return strings.Join(descs, "；")
}
