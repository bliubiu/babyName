package fate

import (
	"strings"
)

// YinYunRater 音韵评分
// 考察声调变化、声母韵母搭配
type YinYunRater struct {
	weight float64
}

func NewYinYunRater() *YinYunRater {
	return &YinYunRater{weight: 0.20}
}

func NewYinYunRaterWithWeight(w float64) *YinYunRater {
	return &YinYunRater{weight: w}
}

func (r *YinYunRater) Name() string    { return "音韵" }
func (r *YinYunRater) Weight() float64 { return r.weight }

func (r *YinYunRater) Rate(candidate *NameCandidate, fateData *FateData) NameRating {
	score := 80.0
	ds := newDetailSink(candidate)

	p1 := candidate.Pinyin1
	p2 := candidate.Pinyin2
	sp := candidate.SurnamePinyin

	// 单名（p2 为空）：借姓氏拼音 + 单字做声调/声母/韵母搭配与谐音检测，
	// 恢复音韵维度区分度。历史缺陷：直接返回恒定 80 使单名 Top5 全部同分，
	// 荒谬字与好字并列，排序由候选池枚举顺序决定 → 门禁字永远打不完。
	if p1 == "" {
		return NameRating{Score: score, Detail: "声调起伏有致，韵母搭配和谐"}
	}

	if p2 == "" {
		if sp != "" {
			// 声调对比（单字 vs 姓氏）
			tone1 := getToneFromPinyin(p1)
			toneS := getToneFromPinyin(sp)
			if tone1 != 0 && toneS != 0 {
				if tone1 != toneS {
					score += 6
					ds.add("单字与姓氏声调不同，抑扬顿挫")
				} else {
					score -= 4
					ds.add("单字与姓氏声调相同")
				}
			}
			// 声母对比
			sm1 := getShengMu(p1)
			smS := getShengMu(sp)
			if sm1 != "" && smS != "" {
				if sm1 != smS {
					score += 4
					ds.add("单字与姓氏声母不同，发音清晰")
				} else {
					score -= 3
					ds.add("单字与姓氏声母相同")
				}
			}
			// 韵母对比（先归一化：生产数据为带声调符号拼音，如 ánɡ/ǎnɡ 实际都是 ang）
			ym1 := stripTone(getYunMu(p1))
			ymS := stripTone(getYunMu(sp))
			if ym1 != "" && ymS != "" {
				if ym1 != ymS {
					score += 3
					ds.add("单字与姓氏韵母不同，朗朗上口")
				} else {
					score -= 3
					ds.add("单字与姓氏韵母相同")
				}
			}
			// 单字不吉谐音检测（本字豁免：传入本字避免同音好字被误判；
			// 姓氏不参与逐字匹配——见 CheckBadPinyinCombo 的连读检测）
			if hit, descs := CheckAllBadHomophones(p1, candidate.Char1); hit {
				score -= float64(len(descs)) * 10
				if !ds.skip {
					ds.add("含不吉谐音: " + strings.Join(descs, "、"))
				}
			}
		}
		score = clampScore(score)
		detail := ds.String("声调起伏有致，韵母搭配和谐")
		return NameRating{Score: score, Detail: detail}
	}

	// 声调检查
	tone1 := getToneFromPinyin(p1)
	tone2 := getToneFromPinyin(p2)
	toneS := getToneFromPinyin(sp)

	if tone1 != 0 && tone2 != 0 {
		// 三字声调旋律评分（姓+名1+名2）
		// 平仄规律：1/2=平，3/4=仄；好名字讲究平仄交替
		if toneS != 0 {
			patternScore, patternDesc := evaluateTonePattern(toneS, tone1, tone2, !ds.skip)
			score += patternScore
			ds.add(patternDesc)
		} else {
			// 无姓氏拼音时退化为两字比较
			if tone1 != tone2 {
				score += 8
				ds.add("两字声调不同，抑扬顿挫")
			} else {
				score -= 5
				ds.add("两字声调相同")
			}
		}
	}

	// 声母检查（按发音部位分组梯度评分）
	sm1 := getShengMu(p1)
	sm2 := getShengMu(p2)
	if sm1 != "" && sm2 != "" {
		if simScore, ok, simDesc := shengMuSimilarity(sm1, sm2, !ds.skip); ok {
			score += simScore
			ds.add(simDesc)
		}
	}

	// 韵母检查（先归一化声调符号，ánɡ/ǎnɡ 均为 ang，避免同韵误判）
	ym1 := stripTone(getYunMu(p1))
	ym2 := stripTone(getYunMu(p2))
	if ym1 != ym2 && ym1 != "" && ym2 != "" {
		score += 4
		ds.add("韵母不同，朗朗上口")
	} else if ym1 == ym2 && ym1 != "" {
		score -= 3
	}

	// ——— 谐音检测（包含姓氏拼音，确保检测跨字谐音如"杜子腾"→肚子疼） ———

	// 1. 逐字检测不吉谐音（本字豁免：把每字的拼音与本字一一对应传入，
	//    避免"思"拼音 si 与"死"谐音混淆等常见好字被误判。
	//    姓氏不参与逐字匹配：按拼音匹配负面词会产生「赵→罩」「王→亡」类误报，
	//    姓氏相关的糟糕联想（如"杜子腾"→肚子疼）由下方连读检测 CheckBadPinyinCombo 负责）
	pinyinCharPairs := []string{p1, candidate.Char1}
	if p2 != "" {
		pinyinCharPairs = append(pinyinCharPairs, p2, candidate.Char2)
	}
	if hit, descs := CheckAllBadHomophones(pinyinCharPairs...); hit {
		penalty := float64(len(descs)) * 10
		score -= penalty
		if !ds.skip {
			ds.add("含不吉谐音: " + strings.Join(descs, "、"))
		}
	}

	// 2. 拼音连读不良组合检测（包含姓氏拼音）
	surnamePy := candidate.SurnamePinyin
	if hit, comboDesc := CheckBadPinyinCombo(surnamePy, p1, p2); hit {
		score -= 15
		ds.add(comboDesc)
	}

	score = clampScore(score)
	detail := ds.String("声调起伏有致，韵母搭配和谐")
	return NameRating{Score: score, Detail: detail}
}

// ShengXiaoRater 生肖匹配评分
// 考察名字五行与生肖五行的生克关系
type ShengXiaoRater struct {
	weight float64
}

func NewShengXiaoRater() *ShengXiaoRater {
	return &ShengXiaoRater{weight: 0.10}
}

func NewShengXiaoRaterWithWeight(w float64) *ShengXiaoRater {
	return &ShengXiaoRater{weight: w}
}

func (r *ShengXiaoRater) Name() string    { return "生肖" }
func (r *ShengXiaoRater) Weight() float64 { return r.weight }

func (r *ShengXiaoRater) Rate(candidate *NameCandidate, fateData *FateData) NameRating {
	if fateData == nil {
		return NameRating{Score: 80, Detail: "生肖与名字五行搭配和谐"}
	}

	score := 80.0
	ds := newDetailSink(candidate)
	zodiac := fateData.BaziInfo.Zodiac
	zodiacWx := getZodiacWuXing(zodiac)

	if zodiacWx == "" {
		return NameRating{Score: score, Detail: "生肖五行信息待完善"}
	}

	ds.addf("生肖%s，五行属%s", zodiac, zodiacWx)

	for _, pair := range []struct {
		char   string
		wuxing string
	}{{candidate.Char1, candidate.WuXing1}, {candidate.Char2, candidate.WuXing2}} {
		if pair.char == "" || pair.wuxing == "" {
			continue
		}
		if isWuXingSheng(zodiacWx, pair.wuxing) || isWuXingSheng(pair.wuxing, zodiacWx) {
			score += 7
			ds.addf("「%s」与生肖五行相生", pair.char)
		}
		if isWuXingKe(zodiacWx, pair.wuxing) || isWuXingKe(pair.wuxing, zodiacWx) {
			score -= 5
			ds.addf("「%s」与生肖五行相克", pair.char)
		}
	}

	score = clampScore(score)
	detail := ds.String("生肖五行搭配合理，无明显冲突")
	return NameRating{Score: score, Detail: detail}
}
