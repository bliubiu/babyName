// Package normalize 物料归一化：有序纯函数步骤，把源记录收敛为规范基座
//
// 原则（docs/31 ADR-009）：
//   - 步骤为纯函数 Record → Record，有序、可单测、离线在线同源；
//   - 归一化只在导入期跑一次，落库即规范态，运行时不再 normalize；
//   - 不臆造数据：源没有的值保持零值，绝不用其他字段伪造。
package normalize

import (
	"strings"
	"unicode/utf8"

	"name/internal/materials/schema"
)

// meaningMaxRunes 释义长度上限（超出截断，避免超长文本撑爆表）
const meaningMaxRunes = 200

// Gender 把源性别枚举归一为规范值
//
// 覆盖中文（男/女）与英文混用；空值归一为 neutral（docs/31 ADR-014）：
// 实测 8105 字中 6266 字无性别标注，不归一会让这些字在性别筛选下被整批剔除
func Gender(raw string) schema.Gender {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "男", "male", "m":
		return schema.Male
	case "女", "female", "f":
		return schema.Female
	case "中性", "neutral", "n", "none":
		return schema.Neutral
	case "":
		return schema.Neutral
	default:
		return schema.Neutral
	}
}

// Level 归一字表等级：缺失或非法 → 0（表外）
func Level(v int) int {
	if v >= 1 && v <= 3 {
		return v
	}
	return 0
}

// Tone 归一声调：仅 0-4 合法，越界归零（不臆造）
func Tone(v int) int {
	if v >= 0 && v <= 4 {
		return v
	}
	return 0
}

// Tier 归一档位（1-5），越界归零
func Tier(v int) int {
	if v >= 1 && v <= 5 {
		return v
	}
	return 0
}

// Score 归一评分（0-100），越界截断到边界
func Score(v int) int {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

// Clamp 通用区间截断，min>max 时返回 min
func Clamp(v, min, max int) int {
	if min > max {
		return min
	}
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// Meaning 归一释义：trim + 按 rune 截断
func Meaning(s string) string {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) <= meaningMaxRunes {
		return s
	}
	runes := []rune(s)
	return string(runes[:meaningMaxRunes])
}

// Stroke 归一笔画：负值归零
func Stroke(v int) int {
	if v < 0 {
		return 0
	}
	return v
}

// Tags 归一标签集合：去空去重排序
func Tags(in []string) []string { return schema.NormalizeTags(in) }

// Wuxing 归一五行：非法值清空（不臆造，不落受控词表之外的值）
func Wuxing(v string) string {
	v = strings.TrimSpace(v)
	if !schema.ValidWuxing(v) {
		return ""
	}
	return v
}

// Word 逐字段归一（NamerChar → schema.Word 的规范形态）
func Word(w schema.Word) schema.Word {
	w.Pinyin = strings.TrimSpace(w.Pinyin)
	w.Radical = strings.TrimSpace(w.Radical)
	w.Meaning = Meaning(w.Meaning)

	// 五行只接受受控值，其余清空（不臆造）
	if !schema.ValidWuxing(w.Wuxing) {
		w.Wuxing = ""
	}

	w.Level = Level(w.Level)
	w.Tone = Tone(w.Tone)
	w.NameFreqTier = Tier(w.NameFreqTier)
	w.UsageLevel = Clamp(w.UsageLevel, 0, 5)
	w.CurationLevel = Clamp(w.CurationLevel, 0, 5)
	w.PositiveScore = Score(w.PositiveScore)
	w.PhoneticScore = Score(w.PhoneticScore)
	w.ModernScore = Score(w.ModernScore)
	w.ClassicalScore = Score(w.ClassicalScore)
	w.NamePenalty = Score(w.NamePenalty)
	w.Strokes = Stroke(w.Strokes)
	w.KangxiStrokes = Stroke(w.KangxiStrokes)

	w.Gender = Gender(string(w.Gender))
	w.StyleTags = Tags(w.StyleTags)
	w.GenderTags = Tags(w.GenderTags)
	w.PairBlacklist = Tags(w.PairBlacklist)
	w.NamingCats = Tags(w.NamingCats)
	return w
}

// CharGroup 归一偏旁分组
func CharGroup(g schema.CharGroup) schema.CharGroup {
	g.Radical = strings.TrimSpace(g.Radical)
	g.Name = strings.TrimSpace(g.Name)
	g.Meaning = Meaning(g.Meaning)
	g.Chars = schema.NormalizeTags(g.Chars)
	return g
}

// BigramFreq 归一双字频率并补出主键
func BigramFreq(b schema.BigramFreq) schema.BigramFreq {
	b.Char1 = strings.TrimSpace(b.Char1)
	b.Char2 = strings.TrimSpace(b.Char2)
	b.Count = Stroke(b.Count)
	b.Rank = Stroke(b.Rank)
	b.Tier = Tier(b.Tier)
	return b
}

// CharFreq 归一单字频率
func CharFreq(c schema.CharFreq) schema.CharFreq {
	c.Char = strings.TrimSpace(c.Char)
	c.Count = Stroke(c.Count)
	c.Rank = Stroke(c.Rank)
	c.Tier = Tier(c.Tier)
	return c
}
