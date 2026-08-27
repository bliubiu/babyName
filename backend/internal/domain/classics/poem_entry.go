package classics

// PoemEntry 完整诗词条目，存储完整元数据
// 用于从名字反查诗词出处、按朝代/作者/分类筛选
type PoemEntry struct {
	// 基本信息
	Title   string // 诗词标题
	Author  string // 作者
	Dynasty string // 朝代（唐/宋/元/先秦等）
	Type    string // 类型（诗/词/曲/赋等）
	Category string // 分类（山水/田园/思乡/送别等）

	// 内容
	Content  []string // 全诗内容（按句分割）
	FullText string   // 完整文本（便于全文搜索）

	// 出处信息
	Source string // 来源（诗经/楚辞/唐诗三百首等）
	Book   string // 书籍/典籍

	// 标签
	Tags []string // 标签（用于分类筛选）

	// 用于快速查找的字符集
	Chars map[string]bool // 本诗包含的所有汉字
}

// PoemIndex 诗词索引，支持多维度查询
type PoemIndex struct {
	// 按字符索引（用于从名字中的字查找诗词）
	byChar map[string][]*PoemEntry

	// 按朝代索引
	byDynasty map[string][]*PoemEntry

	// 按作者索引
	byAuthor map[string][]*PoemEntry

	// 按分类索引
	byCategory map[string][]*PoemEntry

	// 按来源索引（诗经/楚辞/唐诗等）
	bySource map[string][]*PoemEntry

	// 全量数据
	entries []*PoemEntry
}

// NewPoemIndex 创建新的诗词索引
func NewPoemIndex() *PoemIndex {
	return &PoemIndex{
		byChar:     make(map[string][]*PoemEntry),
		byDynasty:  make(map[string][]*PoemEntry),
		byAuthor:   make(map[string][]*PoemEntry),
		byCategory: make(map[string][]*PoemEntry),
		bySource:   make(map[string][]*PoemEntry),
	}
}

// AddEntry 添加诗词条目到索引
func (idx *PoemIndex) AddEntry(entry *PoemEntry) {
	idx.entries = append(idx.entries, entry)

	// 按字符索引
	for char := range entry.Chars {
		idx.byChar[char] = append(idx.byChar[char], entry)
	}

	// 按朝代索引
	if entry.Dynasty != "" {
		idx.byDynasty[entry.Dynasty] = append(idx.byDynasty[entry.Dynasty], entry)
	}

	// 按作者索引
	if entry.Author != "" {
		idx.byAuthor[entry.Author] = append(idx.byAuthor[entry.Author], entry)
	}

	// 按分类索引
	if entry.Category != "" {
		idx.byCategory[entry.Category] = append(idx.byCategory[entry.Category], entry)
	}

	// 按来源索引
	if entry.Source != "" {
		idx.bySource[entry.Source] = append(idx.bySource[entry.Source], entry)
	}
}

// FindByChar 根据字符查找诗词
func (idx *PoemIndex) FindByChar(char string) []*PoemEntry {
	return idx.byChar[char]
}

// FindByDynasty 根据朝代查找诗词
func (idx *PoemIndex) FindByDynasty(dynasty string) []*PoemEntry {
	return idx.byDynasty[dynasty]
}

// FindByAuthor 根据作者查找诗词
func (idx *PoemIndex) FindByAuthor(author string) []*PoemEntry {
	return idx.byAuthor[author]
}

// FindByCategory 根据分类查找诗词
func (idx *PoemIndex) FindByCategory(category string) []*PoemEntry {
	return idx.byCategory[category]
}

// FindBySource 根据来源查找诗词
func (idx *PoemIndex) FindBySource(source string) []*PoemEntry {
	return idx.bySource[source]
}

// FindByChars 根据多个字符查找诗词（取交集）
func (idx *PoemIndex) FindByChars(chars ...string) []*PoemEntry {
	if len(chars) == 0 {
		return nil
	}

	// 从第一个字符的诗词开始
	result := idx.byChar[chars[0]]

	// 与后续字符的诗词取交集
	for _, char := range chars[1:] {
		other := idx.byChar[char]
		result = intersect(result, other)
	}

	return result
}

// FindByCharsUnion 根据多个字符查找诗词（取并集）
func (idx *PoemIndex) FindByCharsUnion(chars ...string) []*PoemEntry {
	seen := make(map[*PoemEntry]bool)
	var result []*PoemEntry

	for _, char := range chars {
		for _, entry := range idx.byChar[char] {
			if !seen[entry] {
				seen[entry] = true
				result = append(result, entry)
			}
		}
	}

	return result
}

// SearchByContent 全文搜索诗词内容
func (idx *PoemIndex) SearchByContent(query string) []*PoemEntry {
	var result []*PoemEntry
	for _, entry := range idx.entries {
		if containsSubstring(entry.FullText, query) {
			result = append(result, entry)
		}
	}
	return result
}

// GetEntries 获取全部诗词条目（只读）
func (idx *PoemIndex) GetEntries() []*PoemEntry {
	return idx.entries
}

// GetStats 获取索引统计信息
func (idx *PoemIndex) GetStats() map[string]int {
	return map[string]int{
		"total_entries": len(idx.entries),
		"total_chars":   len(idx.byChar),
		"dynasties":     len(idx.byDynasty),
		"authors":       len(idx.byAuthor),
		"categories":    len(idx.byCategory),
		"sources":       len(idx.bySource),
	}
}

// intersect 取两个切片的交集
func intersect(a, b []*PoemEntry) []*PoemEntry {
	set := make(map[*PoemEntry]bool)
	for _, entry := range a {
		set[entry] = true
	}

	var result []*PoemEntry
	for _, entry := range b {
		if set[entry] {
			result = append(result, entry)
		}
	}
	return result
}

// containsSubstring 检查字符串是否包含子串（简单实现）
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > 0 && len(substr) > 0 && containsRunes([]rune(s), []rune(substr))))
}

// containsRunes 检查 runes 中是否包含 target
func containsRunes(runes, target []rune) bool {
	if len(target) == 0 {
		return true
	}
	if len(runes) < len(target) {
		return false
	}

	for i := 0; i <= len(runes)-len(target); i++ {
		match := true
		for j := 0; j < len(target); j++ {
			if runes[i+j] != target[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
