package fate

import "strings"

// sensitiveChars 敏感字集合（"大"名需"大"命）
// 这类字格局宏大，普通命格难以承载，易"亢龙有悔"
// 仅专旺格/从强格（日主极旺）方可使用
//
// 依据：AGENTS.md 规则2 "大名需大命"
var sensitiveChars = map[string]bool{
	"龙": true, "凤": true, "乾": true, "坤": true,
	"圣": true, "贤": true, "天": true, "帝": true,
	"皇": true, "神": true, "仙": true, "君": true,
}

// IsSensitiveChar 判断字符是否为敏感字
func IsSensitiveChar(char string) bool {
	return sensitiveChars[char]
}

// isExtremeStrongPattern 检测八字是否为极旺格局（专旺格/从强格）
//
// 判定条件（需同时满足）：
//  1. 日主身旺（QiangRuo == "身旺"）
//  2. 四柱天干中同类五行（比劫+印）占比 >= 3/4
//     - 比劫：与日主同五行
//     - 印：生日主的五行
//
// 极旺格局可承载敏感字，普通格局则需移除（"谦受益"原则）
func isExtremeStrongPattern(fateData *FateData) bool {
	if fateData == nil {
		return false
	}

	// 条件1：日主身旺
	if fateData.WuXingXiji.QiangRuo != "身旺" {
		return false
	}

	riZhuWuxing := fateData.WuXingXiji.RiZhuWuXing
	if riZhuWuxing == "" {
		return false
	}

	// 计算同类五行：比劫（同五行）+ 印（生我者）
	tonglei := tongleiWuxing(riZhuWuxing)
	if tonglei == nil {
		return false
	}

	// 统计四柱天干五行中同类的数量
	tongleiCount := 0
	for _, wx := range fateData.BaziInfo.WuXing {
		if tonglei[wx] {
			tongleiCount++
		}
	}

	// 条件2：同类五行占比 >= 3/4
	return tongleiCount >= 3
}

// forbiddenEntities 禁止的国家地区机关单位名称
// 严禁生成此类名称作为人名（AGENTS.md 安全策略）
var forbiddenEntities = map[string]bool{
	// 国家名称
	"中国": true, "美国": true, "日本": true, "韩国": true,
	"朝鲜": true, "英国": true, "法国": true, "德国": true,
	"俄国": true, "印度": true, "巴西": true, "加拿大": true,
	"澳洲": true, "泰国": true, "越南": true, "缅甸": true,
	"菲律宾": true, "马来西亚": true, "新加坡": true,

	// 国家机关
	"国务院": true, "全国人大": true, "中央军委": true,
	"政治局": true, "中纪委": true, "党中央": true,

	// 地区机关（2-3字简称）
	"省政府": true, "市政府": true, "县政府": true,
	"区政府": true, "镇政府": true, "乡政府": true,

	// 军事/警务
	"军委": true, "国防部": true, "公安部": true,
	"安全部": true, "武警": true, "解放军": true,
}

// IsForbiddenEntity 检查全名或名字部分是否为禁止的国家机关单位名称
// 检查规则：
//  1. 全名匹配（如"中国"）
//  2. 名字部分匹配（去掉姓氏后的部分，如姓"李"名"国务院"→"国务院"）
//  3. 全名包含禁止实体（如"李国务院"）
func IsForbiddenEntity(fullName string) bool {
	if forbiddenEntities[fullName] {
		return true
	}
	runes := []rune(fullName)
	if len(runes) > 1 {
		givenPart := string(runes[1:])
		if forbiddenEntities[givenPart] {
			return true
		}
	}
	for entity := range forbiddenEntities {
		if len([]rune(entity)) > 1 && containsRune(fullName, entity) {
			return true
		}
	}
	return false
}

// containsRune 检查 s 是否包含 substr（按 rune 比较）
func containsRune(s, substr string) bool {
	return strings.Contains(s, substr)
}
