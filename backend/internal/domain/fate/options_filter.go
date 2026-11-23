package fate

import "strings"

// normalizePinyinInitial 归一化拼音首字母输入：小写、去空白、只保留字母。
// 空串或全非字母时返回空，调用方据此跳过筛选（避免误输入把候选池清空）。
func normalizePinyinInitial(initial string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(initial)) {
		if r >= 'a' && r <= 'z' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// filterByPinyinInitial 保留任一读音以 initial 开头的字。
// 读音先经 stripTone 去声调/数字，故 "z" 可命中 z/zh，"zh" 只命中 zh。
func filterByPinyinInitial(chars []*Character, initial string) []*Character {
	if initial == "" {
		return chars
	}
	result := make([]*Character, 0, len(chars))
	for _, c := range chars {
		for _, p := range c.Pinyin {
			if strings.HasPrefix(strings.ToLower(stripTone(p)), initial) {
				result = append(result, c)
				break
			}
		}
	}
	return result
}
