package fate

import ()

// WuxingRater 五行八字匹配度评分
// 考察名字五行与八字喜用神的匹配程度
type WuxingRater struct {
	weight float64
}

func NewWuxingRater() *WuxingRater {
	return &WuxingRater{weight: 0.25}
}

func NewWuxingRaterWithWeight(w float64) *WuxingRater {
	return &WuxingRater{weight: w}
}

func (r *WuxingRater) Name() string    { return "五行八字" }
func (r *WuxingRater) Weight() float64 { return r.weight }

func (r *WuxingRater) Rate(candidate *NameCandidate, fateData *FateData) NameRating {
	if fateData == nil {
		return NameRating{Score: 80, Detail: "五行信息良好"}
	}

	// 使用 XiYongShen 列表（而非单值 Xi/Ji）评估五行匹配
	// 修复：BalanceXiYongJi 在补最缺时可能覆盖 Yong 但未更新 Ji，
	// 导致 Ji 与 XiYongShen 冲突（如 Ji="土" 但 XiYongShen=["土","金"]）。
	// 使用 XiYongShen 列表使评估标准与引擎收窄池一致。
	xiYongShen := fateData.WuXingXiji.XiYongShen
	jiWuxing := fateData.WuXingXiji.Ji

	// 防御性 fallback：当 XiYongShen 为空时，用单值 Xi 构造列表
	if len(xiYongShen) == 0 && fateData.WuXingXiji.Xi != "" {
		xiYongShen = []string{fateData.WuXingXiji.Xi}
	}

	// 构建喜用神集合（用于间接生助判断）
	xiSet := make(map[string]bool, len(xiYongShen))
	for _, wx := range xiYongShen {
		xiSet[wx] = true
	}

	score := 50.0
	ds := newDetailSink(candidate)
	matchCount := 0

	chars := []struct {
		char   string
		wuxing string
	}{{candidate.Char1, candidate.WuXing1}, {candidate.Char2, candidate.WuXing2}}

	for _, c := range chars {
		if c.char == "" || c.wuxing == "" {
			continue
		}
		switch {
		case xiSet[c.wuxing]:
			score += 12
			matchCount++
			ds.addf("「%s」五行属%s，为喜用神", c.char, c.wuxing)
		case c.wuxing == jiWuxing:
			score -= 10
			ds.addf("「%s」五行属%s，为忌神", c.char, c.wuxing)
		default:
			// 间接生助梯度：生任一喜用神 > 被任一喜用神所生（泄气）> 中性
			generatesXi := false
			for _, xi := range xiYongShen {
				if isWuXingSheng(c.wuxing, xi) {
					generatesXi = true
					break
				}
			}
			if generatesXi {
				score += 10
				ds.addf("「%s」五行属%s，生助喜用神", c.char, c.wuxing)
			} else {
				drainedByXi := false
				for _, xi := range xiYongShen {
					if isWuXingSheng(xi, c.wuxing) {
						drainedByXi = true
						break
					}
				}
				if drainedByXi {
					score += 3
					ds.addf("「%s」五行属%s，喜用神所生（泄气）", c.char, c.wuxing)
				} else {
					score += 3
					ds.addf("「%s」五行属%s，中性", c.char, c.wuxing)
				}
			}
		}
	}

	// 两字均匹配喜用神额外加分
	if matchCount == 2 {
		score += 5
		ds.add("两字皆匹配喜用神，补益力强")
	}

	// 两字五行相生加分
	if candidate.WuXing1 != "" && candidate.WuXing2 != "" {
		if isWuXingSheng(candidate.WuXing1, candidate.WuXing2) ||
			isWuXingSheng(candidate.WuXing2, candidate.WuXing1) {
			score += 10
			ds.add("两字五行相生，搭配协调")
		}
		if isWuXingKe(candidate.WuXing1, candidate.WuXing2) ||
			isWuXingKe(candidate.WuXing2, candidate.WuXing1) {
			score -= 8
			ds.add("两字五行相克，需注意")
		}
	}

	score = clampScore(score)
	detail := ds.String("五行搭配均衡，喜用神匹配良好")
	return NameRating{Score: score, Detail: detail}
}

// DefaultRaters 返回默认的九维评分器列表
//
// 移除了 WuGeRater（熊崎五格数理），新增 SancaiRater（天地人三才）+ NoveltyRater（新颖度）
// + FrequencyRater（人名频率），权重重新分配：
//   - WuxingRater     21%（命理层核心）
//   - WenHuaRater     13%（字象层）
//   - YinYunRater     13%（字象层）
//   - ShengXiaoRater  10%（命理层）
//   - SancaiRater      7%（三才）
//   - NoveltyRater    13%（新颖度，区分度核心）
//   - BigramRater      8%（诗词二字共现频率）
//   - FrequencyRater   5%（人名频率，真实语料统计）
//   - GuaXiangRater   10%（卦象体用，docs/42 P3-12 新增）
func DefaultRaters() []Rater {
	return []Rater{
		NewWuxingRaterWithWeight(0.21),    // 21% 命理层
		NewWenHuaRaterWithWeight(0.13),    // 13% 字象层
		NewYinYunRaterWithWeight(0.13),    // 13% 字象层
		NewShengXiaoRaterWithWeight(0.10), // 10% 命理层
		NewSancaiRaterWithWeight(0.07),    //  7% 三才
		NewNoveltyRaterWithWeight(0.13),   // 13% 新颖度
		NewBigramRaterWithWeight(0.08),    //  8% 共现分
		NewFrequencyRaterWithWeight(0.05), //  5% 人名频率
		NewGuaXiangRaterWithWeight(0.10),  // 10% 卦象体用（docs/42 P3-12）
	}
}
