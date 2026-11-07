package fate

import (
	"fmt"

	"name/internal/domain/classics"
)

func NewBigramRater() *BigramRater {
	return &BigramRater{weight: 0.10}
}

func NewBigramRaterWithWeight(w float64) *BigramRater {
	return &BigramRater{weight: w}
}

func (r *BigramRater) Name() string    { return "共现" }
func (r *BigramRater) Weight() float64 { return r.weight }

func (r *BigramRater) Rate(candidate *NameCandidate, fateData *FateData) NameRating {
	baseScore := 40.0
	ds := newDetailSink(candidate)

	if candidate.Char1 != "" && candidate.Char2 != "" {
		// 双名：查二字共现评分（bigramCache 注入时走缓存路径）
		var score int
		var sourceDesc string
		var found bool
		if candidate.bigramCache != nil {
			res := candidate.bigramCache.GetOrCompute(candidate.Char1, candidate.Char2)
			score, sourceDesc, found = res.Score, res.SourceDesc, res.Found
		} else {
			score, sourceDesc, found = classics.GetBigramScore(candidate.Char1, candidate.Char2)
		}
		if found {
			dimension := baseScore + float64(score)*6.0 // 40 + 0~60 = 40~100
			dimension = clampScore(dimension)
			// 原文案为「共现频次评分 N/10」与「；<sourceDesc>」两段拼接，
			// 这里拆成两条 sink 记录，最终 join 结果与原来一致。
			ds.addf("共现频次评分 %d/10", score)
			if sourceDesc != "" {
				ds.add(sourceDesc)
			}
			return NameRating{Score: dimension, Detail: ds.String("")}
		}
		// 未命中共现库，检查单字诗词频率作为降级信号
		if candidate.HasPoetry {
			dimension := baseScore + 10.0 // 50 分：有单字出典
			ds.add("单字有诗词出典（无二字共现）")
			return NameRating{Score: dimension, Detail: ds.String("")}
		}
	} else if candidate.Char1 != "" {
		// 单名：退化为单字诗词频率
		if candidate.HasPoetry {
			dimension := baseScore + 10.0
			ds.add("单字有诗词出典")
			return NameRating{Score: dimension, Detail: ds.String("")}
		}
	}

	// 无任何诗词信号
	return NameRating{Score: baseScore, Detail: "该组合暂无诗词典籍中的二字共现记录"}
}

// evaluateTonePattern 三字声调旋律评分（姓+名1+名2）
//
// 平仄分类：1/2 为平，3/4 为仄。
// 传统音韵学认为名字讲究平仄交替，有抑扬顿挫之美：
//   - 平平仄 / 平仄仄 / 仄平平 / 仄仄平 → 有变化，好
//   - 平平平 / 仄仄仄 → 三连同调，单调
//   - 尾字声调决定整体收束感：仄声收束（3/4）有力度，平声收束（1/2）有余韵
//
// wantDesc 为假时不构造描述文字（穷举热路径），返回空串——分值不受影响。
func evaluateTonePattern(toneS, tone1, tone2 int, wantDesc bool) (float64, string) {
	ping := func(t int) bool { return t == 1 || t == 2 } // 平声
	sP, c1P, c2P := ping(toneS), ping(tone1), ping(tone2)

	// desc 惰性构造：pattern 的 Sprintf 也在其中，避免热路径无谓分配
	desc := func(suffix string) string {
		if !wantDesc {
			return ""
		}
		flat := func(p bool) string {
			if p {
				return "平"
			}
			return "仄"
		}
		pattern := flat(sP) + flat(c1P) + flat(c2P)
		return fmt.Sprintf("声调旋律%s，"+suffix, pattern)
	}

	// 统计平/仄个数
	pingCount := 0
	if sP {
		pingCount++
	}
	if c1P {
		pingCount++
	}
	if c2P {
		pingCount++
	}
	zeCount := 3 - pingCount

	switch {
	case pingCount == 3:
		return -8, desc("三平相连，声调单调")
	case zeCount == 3:
		return -6, desc("三仄相连，声调沉闷")
	case c1P != c2P:
		// 名字两字平仄交替：最佳组合
		if sP != c1P {
			// 姓与名1也交替：三字全交替，最优
			return 12, desc("三字平仄交替")
		}
		return 8, desc("名字平仄交替")
	case c1P == c2P && sP != c1P:
		// 名字同调但与姓不同：姓平名仄或姓仄名平
		return 3, desc("姓与名声调对比")
	default:
		// 名字同调且姓与名1同调：部分重叠
		return -2, desc("部分声调重叠")
	}
}

// shengMuGroup 声母按发音部位分组
//
// 同组声母（如 b/p 都是双唇音）连读时口型变化小，听感模糊；
// 不同组声母（如 x/zh）连读时口型变化大，发音清晰。
var shengMuGroups = map[string]int{
	"b": 1, "p": 1, "m": 1, // 双唇音
	"f": 2,                         // 唇齿音
	"d": 3, "t": 3, "n": 3, "l": 3, // 舌尖中音
	"g": 4, "k": 4, "h": 4, // 舌根音
	"j": 5, "q": 5, "x": 5, // 舌面音
	"zh": 6, "ch": 6, "sh": 6, "r": 6, // 翘舌音
	"z": 7, "c": 7, "s": 7, // 平舌音
	"y": 8, "w": 8, // 零声母/半元音
}

// shengMuSimilarity 声母相似度评分
//
// 返回 (分值, 是否可用, 描述文字)：
//   - 分值范围 [-5, 5]：
//     同组 → -5（口型变化小，听感模糊）
//     相邻组（发音部位差1）→ 2（有变化但过渡自然）
//     远隔组（差≥3）→ 5（口型变化大，发音清晰）
//   - 是否可用：任一拼音不在分组表内时为 false，此时分值/描述无意义
//   - wantDesc 为假时不构造描述文字（穷举热路径）
func shengMuSimilarity(sm1, sm2 string, wantDesc bool) (float64, bool, string) {
	g1, ok1 := shengMuGroups[sm1]
	g2, ok2 := shengMuGroups[sm2]
	if !ok1 || !ok2 {
		return 0, false, ""
	}
	if g1 == g2 {
		if !wantDesc {
			return -5, true, ""
		}
		return -5, true, fmt.Sprintf("声母「%s/%s」同属%s音，听感模糊", sm1, sm2, shengMuGroupName(g1))
	}
	diff := g1 - g2
	if diff < 0 {
		diff = -diff
	}
	if diff == 1 {
		if !wantDesc {
			return 2, true, ""
		}
		return 2, true, fmt.Sprintf("声母「%s/%s」发音部位相邻，过渡自然", sm1, sm2)
	}
	if !wantDesc {
		return 5, true, ""
	}
	return 5, true, fmt.Sprintf("声母「%s/%s」发音部位差异大，吐字清晰", sm1, sm2)
}

// shengMuGroupNames 声母分组中文名（包级常量，避免每次调用重建 map）
var shengMuGroupNames = map[int]string{
	1: "双唇", 2: "唇齿", 3: "舌尖中", 4: "舌根",
	5: "舌面", 6: "翘舌", 7: "平舌", 8: "零声母",
}

// shengMuGroupName 声母分组中文名
func shengMuGroupName(group int) string {
	if n, ok := shengMuGroupNames[group]; ok {
		return n
	}
	return "未知"
}
