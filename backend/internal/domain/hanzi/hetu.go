package hanzi

// HetuWuxing 河图数理五行
// 以笔画数的个位数映射五行，源自《河图洛书》：
//
//	1/6=水（天一生水，地六成之）
//	2/7=火（地二生火，天七成之）
//	3/8=木（天三生木，地八成之）
//	4/9=金（地四生金，天九成之）
//	5/0=土（天五生土，地十成之）
func HetuWuxing(strokes int) string {
	tail := strokes % 10
	switch tail {
	case 1, 6:
		return "水"
	case 2, 7:
		return "火"
	case 3, 8:
		return "木"
	case 4, 9:
		return "金"
	case 5, 0:
		return "土"
	default:
		return ""
	}
}

// HetuWuxingOfChar 获取单个汉字的河图数理五行
func HetuWuxingOfChar(char string) string {
	h, ok := HanziData[char]
	if !ok {
		return ""
	}
	return HetuWuxing(h.Strokes)
}

// HetuWuxingOfName 获取姓名的河图数理五行
// surname=姓, givenNames=名（可1或2字）
// 返回每个字的河图五行
func HetuWuxingOfName(surname string, givenNames ...string) []string {
	result := make([]string, 0, 1+len(givenNames))
	if wx := HetuWuxingOfChar(surname); wx != "" {
		result = append(result, wx)
	}
	for _, g := range givenNames {
		if wx := HetuWuxingOfChar(g); wx != "" {
			result = append(result, wx)
		}
	}
	return result
}

// HetuTotalWuxing 计算姓名总笔画的河图五行
func HetuTotalWuxing(surname string, givenNames ...string) string {
	total := 0
	if h, ok := HanziData[surname]; ok {
		total += h.Strokes
	}
	for _, g := range givenNames {
		if h, ok := HanziData[g]; ok {
			total += h.Strokes
		}
	}
	return HetuWuxing(total)
}
