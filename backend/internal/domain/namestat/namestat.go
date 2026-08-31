package namestat

import (
	"sort"

	"name/internal/domain/hanzi"
)

// NameStat 名字统计信息（基于真实人名频率语料库 Chinese-Names-Corpus）
type NameStat struct {
	Name  string  `json:"name"`
	Count int     `json:"count"`
	Rate  float64 `json:"rate"`
	Rank  int     `json:"rank"`
}

// lookup 在频率库中查找名字的出现次数与类别内排名
// 单字名查单字频率表，双字名查双字组合频率表；未收录返回 found=false（不伪造数据）
func lookup(name string) (count, rank int, found bool) {
	if f := hanzi.GetCharFrequency(name); f != nil {
		return f.Count, f.Rank, true
	}
	runes := []rune(name)
	if len(runes) == 2 {
		if f := hanzi.GetBigramFrequency(string(runes[0]), string(runes[1])); f != nil {
			return f.Count, f.Rank, true
		}
	}
	return 0, 0, false
}

// rateOf 计算名字在语料库中的占比（百分比），语料未加载时返回 0
func rateOf(count int) float64 {
	total := hanzi.GetTotalNames()
	if total <= 0 {
		return 0
	}
	return float64(count) / float64(total) * 100
}

// GetNameCount 获取名字在语料库中的出现次数；未收录返回 0
func GetNameCount(name string) int {
	count, _, found := lookup(name)
	if !found {
		return 0
	}
	return count
}

// GetNameStats 获取名字统计信息
func GetNameStats(name string) *NameStat {
	count, rank, found := lookup(name)
	if !found {
		return &NameStat{Name: name}
	}
	return &NameStat{
		Name:  name,
		Count: count,
		Rate:  rateOf(count),
		Rank:  rank,
	}
}

// GetTopNames 获取热门名字排行（合并单字名与双字名，按出现次数降序）
func GetTopNames(limit int) []NameStat {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	merged := make([]NameStat, 0, len(hanzi.CharRanking())+len(hanzi.BigramRanking()))
	for _, f := range hanzi.CharRanking() {
		merged = append(merged, NameStat{Name: f.Char, Count: f.Count, Rate: rateOf(f.Count)})
	}
	for _, f := range hanzi.BigramRanking() {
		merged = append(merged, NameStat{Name: f.Char1 + f.Char2, Count: f.Count, Rate: rateOf(f.Count)})
	}

	sort.Slice(merged, func(i, j int) bool { return merged[i].Count > merged[j].Count })
	if len(merged) > limit {
		merged = merged[:limit]
	}
	// 按合并后的实际位置重新编号
	for i := range merged {
		merged[i].Rank = i + 1
	}
	return merged
}