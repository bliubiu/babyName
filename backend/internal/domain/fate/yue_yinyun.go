package fate

import (
	"name/internal/domain/yue"
)

// YueYinYunRater 粤语音律评分
//
// 是普通话 YinYunRater 在粤语音系（粤拼 Jyutping）上的平行实现：同样考察
// 「声调流畅度（平仄交替）/ 声母冲突 / 韵母冲突 / 入声收束」，但算子全部来自
// yue 包（粤拼 6 调 + 入声 p/t/k 塞尾 + 粤语声母分组）。
//
// 该评分器独立于普通话八维体系，经 RateCantonese 聚合成「粤语分」单独展示，
// 并在开启粤语评分开关时以固定权重轻量并入综合分（影响排序，但不喧宾夺主）。
//
// 字段来源：NameCandidate 的 Jyutping1/Jyutping2/SurnameJyutping 由引擎在候选
// 组装阶段通过 yue.PrimaryJyutping 预填（首次读音）。未收录字为空串，本评分器降级处理。
//
// 粤语平仄口径：沿用 yue.ToneClass —— 平={1,2}、仄={3,4,5,6}，与普通话
// 「1/2=平、3/4=仄」概念对齐，因此可直接复用 evaluateTonePattern（其 ping 函数
// 仅认 1/2 为平，5/6 自然归入仄），无需为粤语另写一套旋律评分。
type YueYinYunRater struct {
	weight float64
}

func NewYueYinYunRater() *YueYinYunRater {
	return &YueYinYunRater{weight: 0.6}
}

func NewYueYinYunRaterWithWeight(w float64) *YueYinYunRater {
	return &YueYinYunRater{weight: w}
}

func (r *YueYinYunRater) Name() string    { return "粤语音律" }
func (r *YueYinYunRater) Weight() float64 { return r.weight }

func (r *YueYinYunRater) Rate(candidate *NameCandidate, fateData *FateData) NameRating {
	score := 80.0
	ds := newDetailSink(candidate)

	jp1 := candidate.Jyutping1
	jp2 := candidate.Jyutping2
	sjp := candidate.SurnameJyutping

	// 粤拼数据缺失（三字段全空）→ 降级为中性分，不加分也不减分
	if jp1 == "" && jp2 == "" && sjp == "" {
		return NameRating{Score: 75, Detail: "粤拼数据暂缺，音律维度以中性分计"}
	}

	// 单名（jp2 为空）：借姓氏粤拼 + 单字做声调/声母/韵母搭配
	if jp2 == "" {
		if sjp != "" {
			// 声调对比（单字 vs 姓氏）
			tone1 := yue.JyutpingTone(jp1)
			toneS := yue.JyutpingTone(sjp)
			if tone1 != 0 && toneS != 0 {
				if tone1 != toneS {
					score += 6
					ds.add("单字与姓氏声调不同，抑扬顿挫")
				} else {
					score -= 4
					ds.add("单字与姓氏声调相同")
				}
			}
			// 声母对比（按粤语发音部位分组）
			if v, ok, desc := yueShengMuSimilarity(yue.JyutpingInitial(jp1), yue.JyutpingInitial(sjp), !ds.skip); ok {
				score += v
				ds.add(desc)
			}
			// 韵母对比（先去声调）
			if v, ok, desc := yueYunMuSimilarity(jp1, sjp, !ds.skip); ok {
				score += v
				ds.add(desc)
			}
		}
		// 入声收束：单字名以入声（p/t/k 塞尾）收束，短促有力
		if yue.IsEnteringTone(jp1) {
			score += 2
			ds.add("入声收束，短促有力")
		}
		score = clampScore(score)
		detail := ds.String("粤语音律流畅，声调搭配和谐")
		return NameRating{Score: score, Detail: detail}
	}

	// 双名：声调旋律（姓+名1+名2，无姓氏时退化为名1+名2）
	tone1 := yue.JyutpingTone(jp1)
	tone2 := yue.JyutpingTone(jp2)
	toneS := yue.JyutpingTone(sjp)
	if tone1 != 0 && tone2 != 0 {
		if toneS != 0 {
			patternScore, patternDesc := evaluateTonePattern(toneS, tone1, tone2, !ds.skip)
			score += patternScore
			ds.add(patternDesc)
		} else if tone1 != tone2 {
			score += 8
			ds.add("两字声调不同，平仄交替")
		} else {
			score -= 5
			ds.add("两字声调相同")
		}
	}

	// 声母冲突（名1 vs 名2，按粤语发音部位分组梯度评分）
	if v, ok, desc := yueShengMuSimilarity(yue.JyutpingInitial(jp1), yue.JyutpingInitial(jp2), !ds.skip); ok {
		score += v
		ds.add(desc)
	}

	// 韵母冲突（名1 vs 名2，去声调后比较）
	if v, ok, desc := yueYunMuSimilarity(jp1, jp2, !ds.skip); ok {
		score += v
		ds.add(desc)
	}

	// 姓氏与名1 的声母/韵母对比（增强开头清晰度）
	if sjp != "" {
		if v, ok, desc := yueShengMuSimilarity(yue.JyutpingInitial(sjp), yue.JyutpingInitial(jp1), !ds.skip); ok {
			// 同组仅在双名内部已判，开头处同组仅轻微提示，不重复重罚
			if v < 0 {
				score += v * 0.5
			}
			if !ds.skip {
				ds.add(desc)
			}
		}
	}

	// 入声收束：双名以入声收束，短促有力
	if yue.IsEnteringTone(jp2) {
		score += 2
		ds.add("入声收束，短促有力")
	}

	score = clampScore(score)
	detail := ds.String("粤语音律流畅，声调搭配和谐")
	return NameRating{Score: score, Detail: detail}
}

// yueShengMuSimilarity 粤语声母相似度评分（与普通话 shengMuSimilarity 同梯度）。
//
// 返回 (分值, 是否可用, 描述文字)：
//   - 同组 → -5（口型变化小，听感模糊）
//   - 相邻组（发音部位差1）→ 2（过渡自然）
//   - 远隔组（差≥3）→ 5（吐字清晰）
//
// 空声母（零声母/半元音 j/w/ng 起头）与其它声母不强行比较，返回 ok=false。
func yueShengMuSimilarity(sm1, sm2 string, wantDesc bool) (float64, bool, string) {
	g1 := yue.ShengMuGroup(sm1)
	g2 := yue.ShengMuGroup(sm2)
	if g1 == 0 || g2 == 0 {
		return 0, false, ""
	}
	if g1 == g2 {
		if !wantDesc {
			return -5, true, ""
		}
		return -5, true, "粤语声母「" + sm1 + "/" + sm2 + "」同属" + yue.ShengMuGroupName(g1) + "音，听感模糊"
	}
	diff := g1 - g2
	if diff < 0 {
		diff = -diff
	}
	if diff == 1 {
		if !wantDesc {
			return 2, true, ""
		}
		return 2, true, "粤语声母「" + sm1 + "/" + sm2 + "」发音部位相邻，过渡自然"
	}
	if !wantDesc {
		return 5, true, ""
	}
	return 5, true, "粤语声母「" + sm1 + "/" + sm2 + "」发音部位差异大，吐字清晰"
}

// yueYunMuSimilarity 粤语韵母相似度评分（去声调后比较）。
//
// 不同韵母 → +4（朗朗上口）；相同韵母 → -3（叠韵，听感粘连）。
// 任一为空（零声母整字即韵腹）时回退到整音节比较，避免误判。
func yueYunMuSimilarity(jp1, jp2 string, wantDesc bool) (float64, bool, string) {
	if jp1 == "" || jp2 == "" {
		return 0, false, ""
	}
	f1 := yue.StripJyutpingTone(yue.JyutpingFinal(jp1))
	f2 := yue.StripJyutpingTone(yue.JyutpingFinal(jp2))
	if f1 == "" || f2 == "" {
		// 零声母字：韵母即整音节，直接用整音节比较
		f1 = yue.StripJyutpingTone(jp1)
		f2 = yue.StripJyutpingTone(jp2)
	}
	if f1 == "" || f2 == "" {
		return 0, false, ""
	}
	if f1 != f2 {
		if !wantDesc {
			return 4, true, ""
		}
		return 4, true, "粤语韵母不同，朗朗上口"
	}
	if !wantDesc {
		return -3, true, ""
	}
	return -3, true, "粤语叠韵（韵母相同），听感粘连"
}
