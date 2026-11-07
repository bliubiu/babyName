package fate

import (
	"sort"
	"sync"
)

func NewNoveltyRater() *NoveltyRater {
	return &NoveltyRater{weight: 0.12}
}

func NewNoveltyRaterWithWeight(w float64) *NoveltyRater {
	return &NoveltyRater{weight: w}
}

func (r *NoveltyRater) Name() string    { return "新颖度" }
func (r *NoveltyRater) Weight() float64 { return r.weight }

func (r *NoveltyRater) Rate(candidate *NameCandidate, fateData *FateData) NameRating {
	score := 75.0
	ds := newDetailSink(candidate)

	// 1. 同义叠加惩罚：两个字的 Meaning 有大量重叠字时扣分
	//    例如 "明"+"亮" 都含"明亮"义 → 语义冗余
	if candidate.Char1 != "" && candidate.Char2 != "" &&
		candidate.Meaning1 != "" && candidate.Meaning2 != "" {
		overlap := meaningOverlap(candidate)
		if overlap >= 3 {
			score -= float64(overlap) * 3
			ds.addf("语义重叠较多（-%d分）", overlap*3)
		} else if overlap >= 2 {
			score -= 2
			ds.add("语义略有重叠（-2分）")
		}
	}

	// 2. 笔画搭配评分
	if candidate.Stroke1 > 0 && candidate.Stroke2 > 0 {
		diff := absInt(candidate.Stroke1 - candidate.Stroke2)
		avg := (candidate.Stroke1 + candidate.Stroke2) / 2
		switch {
		case diff == 0:
			score -= 3
			ds.add("笔画相同，缺少变化（-3分）")
		case diff >= 1 && diff <= 3:
			score += 5
			ds.add("笔画搭配匀称（+5分）")
		case diff >= 4 && diff <= 8:
			score += 3
			ds.add("繁简有致（+3分）")
		default:
			score -= 2
			ds.add("笔画差异过大（-2分）")
		}
		// 两字都很简单或都很复杂
		if avg <= 5 {
			score -= 3
			ds.add("两字笔画均偏少（-3分）")
		} else if avg >= 20 {
			score -= 2
			ds.add("两字笔画均偏多（-2分）")
		}
	}

	// 3. 单名惩罚：单名天然缺少搭配变化
	if candidate.Char2 == "" {
		score -= 5
		ds.add("单名缺少字形搭配变化（-5分）")
	}

	// 4. 部首重复惩罚
	if candidate.Char1 != "" && candidate.Char2 != "" &&
		candidate.Radical1 != "" && candidate.Radical2 != "" {
		if candidate.Radical1 == candidate.Radical2 {
			score -= 4
			ds.addf("部首「%s」重复，缺少变化（-4分）", candidate.Radical1)
		}
	}

	// 5. 同音惩罚（声韵母完全相同）
	if candidate.Char1 != "" && candidate.Char2 != "" &&
		candidate.Pinyin1 != "" && candidate.Pinyin2 != "" {
		p1 := stripTone(candidate.Pinyin1)
		p2 := stripTone(candidate.Pinyin2)
		if p1 == p2 && p1 != "" {
			score -= 5
			ds.addf("两字同音「%s」，缺乏韵律变化（-5分）", p1)
		}
	}

	score = clampScore(score)
	detail := ds.String("字形搭配有变化，语义差异化良好")
	return NameRating{Score: score, Detail: detail}
}

// semanticOverlap 计算两个字义字符串的重叠字数（排除标点/空格）
//
// 语义与历史实现完全一致：统计 a 中「出现在 b 内」的字符个数，
// a 中的重复字符重复计数，a 中的标点/空白不计。
// 实现改为先取释义画像再归并（见 meaningProfile），避免 O(|a|·|b|) 扫描。
func semanticOverlap(a, b string) int {
	if a == "" || b == "" {
		return 0
	}
	return semanticOverlapProfiles(meaningProfileOf(a), meaningProfileOf(b))
}

// overlapExcludedRune 不计入重叠的标点/空白（与历史实现保持一致）
func overlapExcludedRune(r rune) bool {
	switch r {
	case '，', '、', '。', ' ', '；', '—':
		return true
	}
	return false
}

// runeCount 单字符及其在释义中的出现次数
type runeCount struct {
	r rune
	n int32
}

// meaningProfile 释义字符画像：去重后按 rune 升序的「字符 → 出现次数」序列
//
// 目的：把字义重叠从 O(|a|·|b|) 的逐字符包含扫描，降为两个有序切片的归并
// O(|A|+|B|)。释义是《说文》类长文本（实测中位 230 字、最长 290 字），
// 而候选字只有数千个，因此「按释义缓存画像」的摊薄成本几乎为零。
type meaningProfile struct {
	ordered []runeCount
}

// meaningProfileCache 释义字符串 → *meaningProfile（供未注入画像的调用方兜底）
var meaningProfileCache sync.Map

// meaningProfileOf 取得（并缓存）释义画像
func meaningProfileOf(meaning string) *meaningProfile {
	if v, ok := meaningProfileCache.Load(meaning); ok {
		return v.(*meaningProfile)
	}
	p := buildMeaningProfile(meaning)
	actual, _ := meaningProfileCache.LoadOrStore(meaning, p)
	return actual.(*meaningProfile)
}

// buildMeaningProfile 将释义解析为「字符 → 出现次数」的升序画像。
//
// 注意这里保留标点（不预先剔除）：原实现只对 a 的字符做标点过滤，
// b 的字符集是原样参与成员判断的，若在画像阶段剔除标点会改变结果。
func buildMeaningProfile(meaning string) *meaningProfile {
	counts := make(map[rune]int32, 64)
	for _, r := range meaning {
		counts[r]++
	}
	ordered := make([]runeCount, 0, len(counts))
	for r, n := range counts {
		ordered = append(ordered, runeCount{r: r, n: n})
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].r < ordered[j].r })
	return &meaningProfile{ordered: ordered}
}

// semanticOverlapProfiles 计算两段释义画像的重叠字数（升序归并）
func semanticOverlapProfiles(a, b *meaningProfile) int {
	if a == nil || b == nil {
		return 0
	}
	ia, ib, count := 0, 0, 0
	for ia < len(a.ordered) && ib < len(b.ordered) {
		switch {
		case a.ordered[ia].r == b.ordered[ib].r:
			if !overlapExcludedRune(a.ordered[ia].r) {
				count += int(a.ordered[ia].n)
			}
			ia++
			ib++
		case a.ordered[ia].r < b.ordered[ib].r:
			ia++
		default:
			ib++
		}
	}
	return count
}

// meaningOverlap 计算 NameCandidate 两字的字义重叠（优先用注入画像，避免字符串取缓存）
func meaningOverlap(candidate *NameCandidate) int {
	p1, p2 := candidate.MeaningProfile1, candidate.MeaningProfile2
	if p1 == nil {
		p1 = meaningProfileOf(candidate.Meaning1)
	}
	if p2 == nil {
		p2 = meaningProfileOf(candidate.Meaning2)
	}
	return semanticOverlapProfiles(p1, p2)
}

// BigramRater 二字共现评分器
//
// 基于诗词经典字库的二字共现频率：两个字在同一诗句中出现的次数越多，
// 说明古人在用字时倾向于将它们搭配使用，名字组合越典雅自然。
//
// 评分逻辑（来自 classics.GetBigramScore 0-10 归一化）：
//   - freq >= 5 → 10 分（高频共现，如「关关雎鸠」的关+雎）
//   - freq >= 3 → 7 分
//   - freq >= 2 → 4 分
//   - freq == 1 → 2 分
//   - 未命中 → 0 分（机械凑合的组合）
//
// 映射到 0-100 维度分：score × 10 + 40 基础分
// 即 0 分（未命中）→ 40 分，10 分（高频）→ 100 分
// 单名时退化为单字诗词频率（HasPoetry）判断
type BigramRater struct {
	weight float64
}
