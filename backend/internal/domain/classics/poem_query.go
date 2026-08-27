package classics

import "sort"

// NamePoetryResult 名字的诗词出处查询结果
type NamePoetryResult struct {
	// 名字信息
	Name string // 完整名字
	Chars []string // 名字中的每个字

	// 匹配的诗词
	Matches []PoetryMatch // 匹配结果列表

	// 统计信息
	TotalMatches int // 总匹配数
	BestMatch    *PoetryMatch // 最佳匹配（优先级最高的）
}

// PoetryMatch 单个诗词匹配
type PoetryMatch struct {
	// 诗词信息
	Poem *PoemEntry // 原始诗词条目

	// 匹配详情
	MatchedChars []string // 匹配到的字符
	MatchScore   int      // 匹配分数（越高越好）
	MatchType    string   // 匹配类型（exact/semantic/context）

	// 出处描述
	SourceDesc string // 格式化的出处描述
	Quote      string // 引用的诗句
}

// QueryNamePoetry 从名字反查诗词出处
// 返回该名字在所有诗词来源中的完整出处信息
func QueryNamePoetry(name string) *NamePoetryResult {
	idx := GetGlobalPoemIndex()
	if idx == nil {
		return nil
	}

	// 提取名字中的每个字
	chars := extractNameChars(name)
	if len(chars) == 0 {
		return nil
	}

	// 查找包含这些字的诗词
	poems := idx.FindByCharsUnion(chars...)

	// 构建匹配结果
	matches := buildMatches(name, chars, poems)

	// 按分数排序
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

// QueryNamePoetryByDynasty 按朝代筛选名字的诗词出处
func QueryNamePoetryByDynasty(name, dynasty string) *NamePoetryResult {
	idx := GetGlobalPoemIndex()
	if idx == nil {
		return nil
	}

	chars := extractNameChars(name)
	if len(chars) == 0 {
		return nil
	}

	// 先按朝代筛选
	poemsByDynasty := idx.FindByDynasty(dynasty)

	// 再从中筛选包含名字中字的诗词
	var filtered []*PoemEntry
	for _, poem := range poemsByDynasty {
		if containsAnyChar(poem, chars) {
			filtered = append(filtered, poem)
		}
	}

	matches := buildMatches(name, chars, filtered)
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

// QueryNamePoetryByAuthor 按作者筛选名字的诗词出处
func QueryNamePoetryByAuthor(name, author string) *NamePoetryResult {
	idx := GetGlobalPoemIndex()
	if idx == nil {
		return nil
	}

	chars := extractNameChars(name)
	if len(chars) == 0 {
		return nil
	}

	// 先按作者筛选
	poemsByAuthor := idx.FindByAuthor(author)

	// 再从中筛选包含名字中字的诗词
	var filtered []*PoemEntry
	for _, poem := range poemsByAuthor {
		if containsAnyChar(poem, chars) {
			filtered = append(filtered, poem)
		}
	}

	matches := buildMatches(name, chars, filtered)
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

// GetPoetrySourceDesc 获取格式化的诗词出处描述
// 返回格式："《诗经·关雎》：关关雎鸠，在河之洲"
func GetPoetrySourceDesc(name string) string {
	result := QueryNamePoetry(name)
	if result == nil || result.BestMatch == nil {
		return ""
	}

	match := result.BestMatch
	poem := match.Poem

	// 构建出处描述
	desc := "《" + poem.Title + "》"
	if poem.Author != "" && poem.Author != "佚名" {
		desc += " " + poem.Author
	}
	if poem.Dynasty != "" {
		desc += "（" + poem.Dynasty + "）"
	}
	desc += "：" + match.Quote

	return desc
}

// 辅助函数

// extractNameChars 从名字中提取每个字
func extractNameChars(name string) []string {
	var chars []string
	for _, r := range name {
		char := string(r)
		if isCJKChar(r) && !isStopWord(char) {
			chars = append(chars, char)
		}
	}
	return chars
}

// buildMatches 构建匹配结果
func buildMatches(name string, chars []string, poems []*PoemEntry) []PoetryMatch {
	var matches []PoetryMatch

	for _, poem := range poems {
		match := buildSingleMatch(name, chars, poem)
		if match != nil {
			matches = append(matches, *match)
		}
	}

	return matches
}

// buildSingleMatch 构建单个匹配
func buildSingleMatch(name string, chars []string, poem *PoemEntry) *PoetryMatch {
	// 找到名字中在诗词里出现的字
	var matchedChars []string
	for _, char := range chars {
		if poem.Chars[char] {
			matchedChars = append(matchedChars, char)
		}
	}

	if len(matchedChars) == 0 {
		return nil
	}

	// 计算匹配分数
	score := calculateMatchScore(matchedChars, poem)

	// 找到包含这些字的诗句
	quote := findQuoteWithChars(poem.Content, matchedChars)

	// 构建出处描述
	sourceDesc := buildSourceDesc(poem, quote)

	return &PoetryMatch{
		Poem:         poem,
		MatchedChars: matchedChars,
		MatchScore:   score,
		MatchType:    determineMatchType(matchedChars, poem),
		SourceDesc:   sourceDesc,
		Quote:        quote,
	}
}

// calculateMatchScore 计算匹配分数
func calculateMatchScore(matchedChars []string, poem *PoemEntry) int {
	score := 0

	// 1. 匹配字数加分（每个字10分）
	score += len(matchedChars) * 10

	// 2. 如果名字中的字在诗题中出现，额外加分
	for _, char := range matchedChars {
		if containsChar(poem.Title, char) {
			score += 20
		}
	}

	// 3. 如果名字中的字在同一句诗中，额外加分
	if len(matchedChars) > 1 {
		quote := findQuoteWithChars(poem.Content, matchedChars)
		if containsAllChars(quote, matchedChars) {
			score += 30
		}
	}

	// 4. 经典来源加分（诗经/楚辞优先）
	switch poem.Source {
	case "诗经":
		score += 15
	case "楚辞":
		score += 12
	case "唐诗", "宋词":
		score += 10
	}

	return score
}

// findQuoteWithChars 找到包含指定字符的诗句
func findQuoteWithChars(content []string, chars []string) string {
	for _, line := range content {
		if containsAllChars(line, chars) {
			return line
		}
	}

	// 如果没有单句包含所有字，返回包含第一个字的最短句子
	if len(chars) > 0 {
		for _, line := range content {
			if containsChar(line, chars[0]) {
				return line
			}
		}
	}

	return ""
}

// buildSourceDesc 构建出处描述
func buildSourceDesc(poem *PoemEntry, quote string) string {
	desc := "《" + poem.Title + "》"
	if poem.Author != "" && poem.Author != "佚名" {
		desc += " " + poem.Author
	}
	if poem.Dynasty != "" {
		desc += "（" + poem.Dynasty + "）"
	}
	if quote != "" {
		desc += "：" + quote
	}
	return desc
}

// determineMatchType 判断匹配类型
func determineMatchType(matchedChars []string, poem *PoemEntry) string {
	if len(matchedChars) == 0 {
		return "none"
	}

	// 检查是否完全匹配（名字中的所有字都在同一句诗中）
	quote := findQuoteWithChars(poem.Content, matchedChars)
	if containsAllChars(quote, matchedChars) {
		return "exact"
	}

	// 检查是否有语义关联
	for _, char := range matchedChars {
		if len(FindSemanticChars(char)) > 0 {
			return "semantic"
		}
	}

	return "context"
}

// sortMatches 按分数排序
func sortMatches(matches []PoetryMatch) {
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].MatchScore > matches[j].MatchScore
	})
}

// containsAnyChar 检查诗词是否包含任意指定字符
func containsAnyChar(poem *PoemEntry, chars []string) bool {
	for _, char := range chars {
		if poem.Chars[char] {
			return true
		}
	}
	return false
}

// containsChar 检查字符串是否包含指定字符
func containsChar(s string, char string) bool {
	for _, r := range s {
		if string(r) == char {
			return true
		}
	}
	return false
}

// containsAllChars 检查字符串是否包含所有指定字符
func containsAllChars(s string, chars []string) bool {
	for _, char := range chars {
		if !containsChar(s, char) {
			return false
		}
	}
	return true
}
