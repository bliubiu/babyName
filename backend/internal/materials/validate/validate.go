// Package validate 装载期强校验：脏数据即拒，错误可归因到「物料/字段/值/规则」
//
// 原则（docs/31 ADR-004）：校验在导入期一次完成并落库，
// 运行时装载不再做同一批检查，从而把数据质量前移且可确定性回归。
package validate

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"name/internal/materials/schema"
)

// Error 单条校验错误，携带完整归因信息
type Error struct {
	Material string `json:"material"`
	Field    string `json:"field"`
	Value    string `json:"value"`
	Rule     string `json:"rule"`
}

func (e Error) Error() string {
	return fmt.Sprintf("物料[%s] 字段[%s] 值[%q] 违反规则[%s]", e.Material, e.Field, e.Value, e.Rule)
}

// Errors 校验错误集合
type Errors []Error

func (e Errors) Error() string {
	parts := make([]string, 0, len(e))
	for _, x := range e {
		parts = append(parts, x.Error())
	}
	return strings.Join(parts, "; ")
}

// Len 返回错误条数
func (e Errors) Len() int { return len(e) }

// RuleStat 单条 (字段, 规则) 的聚合计数
type RuleStat struct {
	Field  string `json:"field"`
	Rule   string `json:"rule"`
	Count  int    `json:"count"`
	Sample string `json:"sample"` // 该规则下首个触发值（按 sorted 序）
}

// Summarize 按 (字段, 规则) 聚合计数并附前 sampleN 条明细样例
//
// 大批量脏数据（整列字段非法）时逐条拼接的错误串不可读：聚合让
// 「哪类问题有多少、典型值长什么样」一目了然（docs/34 阶段 C2）。
// 顺序经 sorted() 稳定，输出可 diff、可回归比对。
func (e Errors) Summarize(sampleN int) string {
	if len(e) == 0 {
		return ""
	}
	sorted := e.sorted()

	type ruleKey struct{ field, rule string }
	counts := make(map[ruleKey]int, len(sorted))
	samples := make(map[ruleKey]string, len(sorted))
	order := make([]ruleKey, 0, len(sorted))
	for _, x := range sorted {
		k := ruleKey{x.Field, x.Rule}
		if _, ok := counts[k]; !ok {
			order = append(order, k)
			samples[k] = x.Value
		}
		counts[k]++
	}

	parts := make([]string, 0, len(order)+2)
	parts = append(parts, fmt.Sprintf("共 %d 条校验问题（按规则聚合）", len(sorted)))
	for _, k := range order {
		parts = append(parts, fmt.Sprintf("字段[%s] 规则[%s] × %d（样例 %q）",
			k.field, k.rule, counts[k], samples[k]))
	}
	if n := sampleN; n > 0 {
		if n > len(sorted) {
			n = len(sorted)
		}
		detail := make([]string, 0, n)
		for _, x := range sorted[:n] {
			detail = append(detail, x.Error())
		}
		parts = append(parts, "明细前 "+strconv.Itoa(n)+" 条: "+strings.Join(detail, "; "))
	}
	return strings.Join(parts, "；")
}

// sorted 保证错误输出顺序稳定，便于 diff 与回归比对
func (e Errors) sorted() Errors {
	out := make(Errors, len(e))
	copy(out, e)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Field != out[j].Field {
			return out[i].Field < out[j].Field
		}
		return out[i].Value < out[j].Value
	})
	return out
}

// isSingleRune 判断是否为单个汉字
func isSingleRune(s string) bool {
	return len([]rune(s)) == 1
}

// Word 校验一个规范汉字
func Word(material string, w schema.Word) Errors {
	var errs Errors
	add := func(field, value, rule string) {
		errs = append(errs, Error{Material: material, Field: field, Value: value, Rule: rule})
	}

	if !isSingleRune(w.Char) {
		add("char", w.Char, "必须是单个汉字")
	}
	if w.Strokes < 1 {
		add("strokes", fmt.Sprint(w.Strokes), "简体笔画必须 ≥1")
	}
	if !schema.ValidWuxing(w.Wuxing) {
		add("wuxing", w.Wuxing, "五行必须是 金木水火土 之一或空")
	}
	if !schema.ValidGender(w.Gender) {
		add("gender", string(w.Gender), "性别必须是 male/female/neutral")
	}
	if w.Level < 0 || w.Level > 3 {
		add("level", fmt.Sprint(w.Level), "字表等级必须在 0~3")
	}
	if w.Tone < 0 || w.Tone > 4 {
		add("tone", fmt.Sprint(w.Tone), "声调必须在 0~4")
	}
	if w.NameFreqTier < 0 || w.NameFreqTier > 5 {
		add("nameFreqTier", fmt.Sprint(w.NameFreqTier), "人名频率档必须在 0~5")
	}
	for field, v := range map[string]int{
		"positiveScore": w.PositiveScore, "phoneticScore": w.PhoneticScore,
		"modernScore": w.ModernScore, "classicalScore": w.ClassicalScore,
		"namePenalty": w.NamePenalty,
	} {
		if v < 0 || v > 100 {
			add(field, fmt.Sprint(v), "评分必须在 0~100")
		}
	}
	return errs
}

// Words 校验一批汉字，返回错误上限 maxErr 条后的错误
func Words(material string, ws []schema.Word, maxErr int) Errors {
	var errs Errors
	seen := make(map[string]bool, len(ws))
	for _, w := range ws {
		if seen[w.Char] {
			errs = append(errs, Error{Material: material, Field: "char", Value: w.Char, Rule: "主键重复"})
			if len(errs) >= maxErr {
				return errs.sorted()
			}
			continue
		}
		seen[w.Char] = true
		errs = append(errs, Word(material, w)...)
		if len(errs) >= maxErr {
			return errs.sorted()
		}
	}
	return errs.sorted()
}

// CharGroup 校验偏旁分组：部首必填、分组内汉字须已入库
func CharGroup(material string, g schema.CharGroup, known map[string]bool) Errors {
	var errs Errors
	if g.Radical == "" {
		errs = append(errs, Error{Material: material, Field: "radical", Value: g.Radical, Rule: "部首不可为空"})
	}
	if len(g.Chars) == 0 {
		errs = append(errs, Error{Material: material, Field: "chars", Value: g.Radical, Rule: "分组内不可为空"})
	}
	for _, c := range g.Chars {
		if !isSingleRune(c) {
			errs = append(errs, Error{Material: material, Field: "chars", Value: c, Rule: "必须是单个汉字"})
			continue
		}
		if known != nil && !known[c] {
			errs = append(errs, Error{Material: material, Field: "chars", Value: c, Rule: "字不在字表中"})
		}
	}
	return errs
}

// Meaning 校验词典释义条目
func Meaning(material string, m schema.Meaning) Errors {
	var errs Errors
	if !isSingleRune(m.Char) {
		errs = append(errs, Error{Material: material, Field: "char", Value: m.Char, Rule: "必须是单个汉字"})
	}
	if strings.TrimSpace(m.Explanation) == "" {
		errs = append(errs, Error{Material: material, Field: "explanation", Value: m.Char, Rule: "释义不可为空"})
	}
	return errs
}

// CharFreq 校验单字频率
func CharFreq(material string, c schema.CharFreq) Errors {
	var errs Errors
	if !isSingleRune(c.Char) {
		errs = append(errs, Error{Material: material, Field: "char", Value: c.Char, Rule: "必须是单个汉字"})
	}
	if c.Count < 0 {
		errs = append(errs, Error{Material: material, Field: "count", Value: fmt.Sprint(c.Count), Rule: "频次不可为负"})
	}
	if c.Tier < 0 || c.Tier > 5 {
		errs = append(errs, Error{Material: material, Field: "tier", Value: fmt.Sprint(c.Tier), Rule: "档位必须在 0~5"})
	}
	return errs
}

// BigramFreq 校验双字频率（含主键一致性，docs/31 §9.8 跨物料规则）
func BigramFreq(material string, b schema.BigramFreq) Errors {
	var errs Errors
	if !isSingleRune(b.Char1) {
		errs = append(errs, Error{Material: material, Field: "char1", Value: b.Char1, Rule: "必须是单个汉字"})
	}
	if !isSingleRune(b.Char2) {
		errs = append(errs, Error{Material: material, Field: "char2", Value: b.Char2, Rule: "必须是单个汉字"})
	}
	if b.Count < 0 {
		errs = append(errs, Error{Material: material, Field: "count", Value: fmt.Sprint(b.Count), Rule: "频次不可为负"})
	}
	if b.Tier < 0 || b.Tier > 5 {
		errs = append(errs, Error{Material: material, Field: "tier", Value: fmt.Sprint(b.Tier), Rule: "档位必须在 0~5"})
	}
	return errs
}

// Kangxi 校验康熙笔画
func Kangxi(material string, k schema.KangxiStroke) Errors {
	var errs Errors
	if !isSingleRune(k.Char) {
		errs = append(errs, Error{Material: material, Field: "char", Value: k.Char, Rule: "必须是单个汉字"})
	}
	if k.Strokes < 1 {
		errs = append(errs, Error{Material: material, Field: "strokes", Value: fmt.Sprint(k.Strokes), Rule: "康熙笔画必须 ≥1"})
	}
	return errs
}

// Hexagram 校验易经卦：ID ∈ [1,64] 且唯一（docs/31 §9.8）
func Hexagram(material string, h schema.Hexagram, seen map[int]bool) Errors {
	var errs Errors
	if h.ID < 1 || h.ID > 64 {
		errs = append(errs, Error{Material: material, Field: "id", Value: fmt.Sprint(h.ID), Rule: "卦序必须在 1~64"})
	} else if seen != nil && seen[h.ID] {
		errs = append(errs, Error{Material: material, Field: "id", Value: fmt.Sprint(h.ID), Rule: "卦序重复"})
	}
	if h.Name == "" {
		errs = append(errs, Error{Material: material, Field: "name", Value: h.Name, Rule: "卦名不可为空"})
	}
	return errs
}

// Zodiac 校验生肖：ID ∈ [1,12]
func Zodiac(material string, z schema.Zodiac) Errors {
	var errs Errors
	if z.ID < 1 || z.ID > 12 {
		errs = append(errs, Error{Material: material, Field: "id", Value: fmt.Sprint(z.ID), Rule: "生肖序号必须在 1~12"})
	}
	if z.Name == "" {
		errs = append(errs, Error{Material: material, Field: "name", Value: z.Name, Rule: "生肖名不可为空"})
	}
	return errs
}

// Rune 校验单字集合条目（门禁字表）
func Rune(material, field, value string) Errors {
	if !isSingleRune(value) {
		return Errors{{Material: material, Field: field, Value: value, Rule: "必须是单个汉字"}}
	}
	return nil
}

// Combo 校验双字组合（禁忌表）
func Combo(material, value string) Errors {
	if len([]rune(value)) != 2 {
		return Errors{{Material: material, Field: "combo", Value: value, Rule: "必须是两个汉字组成的组合"}}
	}
	return nil
}

// Entity 校验合规红线条目（机关单位/争议地名/负面历史人物，docs/33 M2）
//
// 红线实体为 2~8 字的名称/专名，且不得含空白字符（避免切词歧义与脏数据）。
func Entity(material, field, value string) Errors {
	var errs Errors
	add := func(rule string) {
		errs = append(errs, Error{Material: material, Field: field, Value: value, Rule: rule})
	}
	n := len([]rune(value))
	if n < 2 || n > 8 {
		add("长度必须为 2~8 个汉字")
	}
	if value != strings.TrimSpace(value) || strings.ContainsAny(value, " \t\r\n") {
		add("不可含空白字符")
	}
	return errs
}
