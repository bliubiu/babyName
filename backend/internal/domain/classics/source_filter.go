package classics

import (
	"sort"
	"strings"
)

// sourceFamilyGroups 来源族 → 展示层 PoemEntry.Source 集合的映射
//
// 背景（2026.10.21）：候选池侧与展示层用的是两套来源命名体系：
//
//	候选池侧 GetPoetryCharList("唐诗宋词")  → 7 个子集：唐诗三百首/宋词三百首/
//	                                        元曲/乐府诗集/水墨唐诗/辞赋/唐诗补录
//	展示层  PoemEntry.Source               → 具体的子集名（同上）
//
// 用户在前端选「唐诗宋词」时，若展示层不按来源筛选，出处仍会显示「诗经」，
// 与用户的选择脱节（实测：选唐诗宋词，BestMatch 却是诗经《南有嘉鱼》）。
//
// 语义：来源族是「用户可选的典籍入口」，展示层按族筛选后再排序；
// 若该族内无任何命中，则回退到全库（不因筛选而丢失出处）。
var sourceFamilyGroups = map[string][]string{
	"诗经":   {"诗经"},
	"楚辞":   {"楚辞"},
	"古文观止": {"古文观止"},
	"唐诗宋词": {"唐诗三百首", "宋词三百首", "元曲", "乐府诗集", "水墨唐诗", "辞赋", "唐诗补录"},
	"元曲":   {"元曲"},
	"乐府":   {"乐府诗集"},
	"辞赋":   {"辞赋"},
	"论语":   {"论语"},
	"孟子":   {"孟子"},
	"大学":   {"大学"},
	"中庸":   {"中庸"},
	"三字经":  {"三字经", "三字经-new"},
	"千字文":  {"千字文"},
	"弟子规":  {"弟子规"},
	"幼学琼林": {"幼学琼林"},
	"增广贤文": {"增广贤文"},
	"声律启蒙": {"声律启蒙"},
	"朱子家训": {"朱子家训"},
	"千家诗":  {"千家诗"},
	"文字蒙求": {"文字蒙求"},
	"百家姓":  {"百家姓"},
}

// SourceFamilyMembers 返回某来源族在展示层对应的 PoemEntry.Source 集合。
// 未登记的来源返回 nil（表示不筛选）。
func SourceFamilyMembers(source string) []string {
	if source == "" {
		return nil
	}
	// 别名归一（如 shijing → 诗经）
	if alias, ok := sourceAlias[source]; ok {
		source = alias
	}
	if members, ok := sourceFamilyGroups[source]; ok {
		return members
	}
	// 未登记的族：若展示层存在同名 Source，则按同名精确匹配
	return []string{source}
}

// QueryNamePoetryBySource 按来源族筛选后查询名字的诗词出处。
//
// 与 QueryNamePoetry 的区别：后者在**全库**上匹配并取分数最高的一条，
// 忽略用户选择的来源；本函数先把候选诗篇限定在指定来源族内再排序，
// 使「选了唐诗宋词 → 出处显示唐诗宋词」成立。
//
// 回退策略：若该来源族内无任何命中（名字用字不在该族字集中），
// 回退到全库查询，避免因筛选导致出处整体消失——此时结果来源可能与用户
// 选择不同，但保留了「有出处可展示」的价值。
func QueryNamePoetryBySource(name, source string) *NamePoetryResult {
	members := SourceFamilyMembers(source)
	if len(members) == 0 {
		return QueryNamePoetry(name)
	}

	idx := GetGlobalPoemIndex()
	if idx == nil {
		return nil
	}

	chars := extractNameChars(name)
	if len(chars) == 0 {
		return nil
	}

	// 在指定来源族内取候选诗篇
	var scoped []*PoemEntry
	seen := make(map[*PoemEntry]bool)
	for _, member := range members {
		for _, poem := range idx.FindBySource(member) {
			if !seen[poem] {
				seen[poem] = true
				scoped = append(scoped, poem)
			}
		}
	}
	// 该族内无诗篇（如展示层未收录该来源）→ 回退全库
	if len(scoped) == 0 {
		return QueryNamePoetry(name)
	}

	// 只保留含有名字用字的诗篇
	var candidates []*PoemEntry
	for _, poem := range scoped {
		if containsAnyChar(poem, chars) {
			candidates = append(candidates, poem)
		}
	}
	if len(candidates) == 0 {
		// 该族内无命中 → 回退全库
		return QueryNamePoetry(name)
	}

	matches := buildMatches(name, chars, candidates)
	sortMatches(matches)

	result := &NamePoetryResult{
		Name:         name,
		Chars:        chars,
		Matches:      matches,
		TotalMatches: len(matches),
	}
	if len(matches) > 0 {
		result.BestMatch = &matches[0]
	}
	return result
}

// MatchedSourceMembers 返回查询结果中命中的来源是否属于指定来源族。
// 用于前端/服务层自检：BestMatch.Source 是否在族内。
func MatchedSourceMembers(source, poemSource string) bool {
	for _, member := range SourceFamilyMembers(source) {
		if member == poemSource {
			return true
		}
	}
	return false
}

// normalizeSourceName 归一来源名（去除空白与「《》」包裹），便于比较。
func normalizeSourceName(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "《")
	s = strings.TrimSuffix(s, "》")
	return s
}

// SortedSourceFamilies 返回全部已登记来源族（按名称排序），供巡检/文档使用。
func SortedSourceFamilies() []string {
	families := make([]string, 0, len(sourceFamilyGroups))
	for name := range sourceFamilyGroups {
		families = append(families, name)
	}
	sort.Strings(families)
	return families
}
