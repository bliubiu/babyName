package classics

// PoemNamer 诗词命名器，负责为名字提供诗词出处标注
type PoemNamer struct {
	index *PoemIndex
}

// NewPoemNamer 创建新的诗词命名器
func NewPoemNamer() *PoemNamer {
	return &PoemNamer{
		index: GetGlobalPoemIndex(),
	}
}

// AnnotateName 为名字标注诗词出处
// 返回完整的诗词出处信息，包括诗句、作者、朝代等
func (pn *PoemNamer) AnnotateName(name string) *NamePoetryResult {
	if pn.index == nil {
		return nil
	}
	return QueryNamePoetry(name)
}

// GetSourceDesc 获取名字的诗词出处描述（格式化字符串）
func (pn *PoemNamer) GetSourceDesc(name string) string {
	return GetPoetrySourceDesc(name)
}

// GetSourceDescByDynasty 按朝代筛选获取诗词出处描述
func (pn *PoemNamer) GetSourceDescByDynasty(name, dynasty string) string {
	result := QueryNamePoetryByDynasty(name, dynasty)
	if result == nil || result.BestMatch == nil {
		return ""
	}
	return result.BestMatch.SourceDesc
}

// GetSourceDescByAuthor 按作者筛选获取诗词出处描述
func (pn *PoemNamer) GetSourceDescByAuthor(name, author string) string {
	result := QueryNamePoetryByAuthor(name, author)
	if result == nil || result.BestMatch == nil {
		return ""
	}
	return result.BestMatch.SourceDesc
}

// HasPoetryOrigin 检查名字是否有诗词出处
func (pn *PoemNamer) HasPoetryOrigin(name string) bool {
	result := QueryNamePoetry(name)
	return result != nil && result.TotalMatches > 0
}

// GetPoemCount 获取诗词库中的诗词总数
func (pn *PoemNamer) GetPoemCount() int {
	if pn.index == nil {
		return 0
	}
	return len(pn.index.entries)
}

// GetStats 获取诗词索引统计信息
func (pn *PoemNamer) GetStats() map[string]int {
	if pn.index == nil {
		return nil
	}
	return pn.index.GetStats()
}

// SearchByContent 全文搜索诗词内容
func (pn *PoemNamer) SearchByContent(query string) []*PoemEntry {
	if pn.index == nil {
		return nil
	}
	return pn.index.SearchByContent(query)
}

// FindPoemsByChar 根据字符查找诗词
func (pn *PoemNamer) FindPoemsByChar(char string) []*PoemEntry {
	if pn.index == nil {
		return nil
	}
	return pn.index.FindByChar(char)
}

// FindPoemsBySource 根据来源查找诗词（如"诗经"、"楚辞"）
func (pn *PoemNamer) FindPoemsBySource(source string) []*PoemEntry {
	if pn.index == nil {
		return nil
	}
	return pn.index.FindBySource(source)
}

// globalPoemNamer 全局诗词命名器
var globalPoemNamer *PoemNamer

// GetGlobalPoemNamer 获取全局诗词命名器
func GetGlobalPoemNamer() *PoemNamer {
	if globalPoemNamer == nil {
		globalPoemNamer = NewPoemNamer()
	}
	return globalPoemNamer
}
