package yijing

// HexagramFinder 易经卦象查找接口
// 提供卦象查找和喜用神匹配能力，由应用层适配器实现
type HexagramFinder interface {
	FindByStrokes(strokes int) *Hexagram
	// FindByMeihuaName 梅花易数姓名卦：姓笔画→上卦，名笔画→下卦
	FindByMeihuaName(surnameStrokes, givenStrokes int) *Hexagram
	// FindMeihuaDetail 梅花易数姓名完整起卦：本卦/变卦/互卦/动爻与六爻（docs/42 P0-4）
	FindMeihuaDetail(surnameStrokes, givenStrokes int) *PlumBlossomResult
	FindByNumber(id int) *Hexagram
	FindAll() []Hexagram
	MatchXiyongshen(hexagram *Hexagram, xiyongshen []string) *HexagramMatch
}
