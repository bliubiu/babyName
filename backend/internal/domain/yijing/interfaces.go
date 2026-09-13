package yijing

// HexagramFinder 易经卦象查找接口
// 提供卦象查找和喜用神匹配能力，由应用层适配器实现
type HexagramFinder interface {
	FindByStrokes(strokes int) *Hexagram
	// FindByMeihuaName 梅花易数姓名卦：姓笔画→上卦，名笔画→下卦
	FindByMeihuaName(surnameStrokes, givenStrokes int) *Hexagram
	FindByNumber(id int) *Hexagram
	FindAll() []Hexagram
	MatchXiyongshen(hexagram *Hexagram, xiyongshen []string) *HexagramMatch
}
