package fate

import "strings"

// BadHomophoneInfo 不吉谐音信息
type BadHomophoneInfo struct {
	Pinyin      string // 拼音（无声调）
	Word        string // 对应的不吉汉字
	Description string // 说明
}

// BadHomophones 不吉谐音列表
// 按拼音首字母分组，检测名字拼音时逐音节匹配
var BadHomophones = []BadHomophoneInfo{
	{Pinyin: "bai", Word: "败", Description: "败落"},
	{Pinyin: "bing", Word: "病", Description: "疾病"},
	{Pinyin: "can", Word: "残", Description: "残废"},
	{Pinyin: "chan", Word: "忏", Description: "忏悔（消极）"},
	{Pinyin: "chou", Word: "丑", Description: "丑陋"},
	{Pinyin: "cu", Word: "粗", Description: "粗俗"},
	{Pinyin: "dao", Word: "倒", Description: "倒台"},
	{Pinyin: "du", Word: "毒", Description: "毒害"},
	{Pinyin: "e", Word: "恶", Description: "凶恶"},
	{Pinyin: "fan", Word: "犯", Description: "犯法"},
	{Pinyin: "fu", Word: "腐", Description: "腐败"},
	{Pinyin: "gua", Word: "寡", Description: "孤寡"},
	{Pinyin: "guai", Word: "怪", Description: "怪癖"},
	{Pinyin: "gui", Word: "鬼", Description: "鬼怪"},
	{Pinyin: "hai", Word: "害", Description: "伤害"},
	{Pinyin: "han", Word: "憾", Description: "遗憾"},
	{Pinyin: "hen", Word: "恨", Description: "仇恨"},
	{Pinyin: "hun", Word: "昏", Description: "昏暗"},
	{Pinyin: "huo", Word: "祸", Description: "灾祸"},
	{Pinyin: "jian", Word: "贱", Description: "低贱"},
	{Pinyin: "ku", Word: "哭", Description: "哭泣"},
	{Pinyin: "kui", Word: "亏", Description: "亏损"},
	{Pinyin: "lan", Word: "烂", Description: "腐烂"},
	{Pinyin: "li", Word: "离", Description: "分离"},
	{Pinyin: "luan", Word: "乱", Description: "混乱"},
	{Pinyin: "mai", Word: "埋", Description: "埋没"},
	{Pinyin: "mie", Word: "灭", Description: "毁灭"},
	{Pinyin: "mo", Word: "没", Description: "没落"},
	{Pinyin: "nan", Word: "难", Description: "困难"},
	{Pinyin: "nu", Word: "奴", Description: "奴役"},
	{Pinyin: "pei", Word: "赔", Description: "赔钱"},
	{Pinyin: "pi", Word: "屁", Description: "粗俗不雅"},
	{Pinyin: "po", Word: "破", Description: "破败"},
	{Pinyin: "sang", Word: "丧", Description: "丧事"},
	{Pinyin: "sha", Word: "杀", Description: "杀生"},
	{Pinyin: "shi", Word: "屎", Description: "污秽"},
	{Pinyin: "shu", Word: "输", Description: "输赢（输）"},
	{Pinyin: "shuai", Word: "衰", Description: "衰败"},
	{Pinyin: "si", Word: "死", Description: "死亡"},
	{Pinyin: "tong", Word: "痛", Description: "痛苦"},
	{Pinyin: "tou", Word: "偷", Description: "偷盗"},
	{Pinyin: "tu", Word: "秃", Description: "秃废"},
	{Pinyin: "wang", Word: "亡", Description: "死亡"},
	{Pinyin: "wei", Word: "危", Description: "危险"},
	{Pinyin: "wu", Word: "污", Description: "污秽"},
	{Pinyin: "xian", Word: "险", Description: "危险"},
	{Pinyin: "xiong", Word: "凶", Description: "凶险"},
	{Pinyin: "ya", Word: "哑", Description: "哑巴"},
	{Pinyin: "zei", Word: "贼", Description: "盗贼"},
	{Pinyin: "zang", Word: "脏", Description: "肮脏"},
	{Pinyin: "zhai", Word: "债", Description: "负债"},
}

// BadPinyinCombo 不良拼音组合
// 名字全拼连读时可能产生的不良词汇
type BadPinyinCombo struct {
	Combo       []string // 拼音组合（连续音节）
	Description string   // 说明
}

// BadPinyinCombos 检查拼音连读产生的不良词汇
var BadPinyinCombos = []BadPinyinCombo{
	{Combo: []string{"wang", "ba"}, Description: "王八（不雅称谓）"},
	{Combo: []string{"wang", "dan"}, Description: "王蛋（不雅称谓）"},
	{Combo: []string{"sha", "bi"}, Description: "不雅词汇"},
	{Combo: []string{"er", "bi"}, Description: "不雅词汇"},
	{Combo: []string{"sha", "gua"}, Description: "傻瓜"},
	{Combo: []string{"bai", "chi"}, Description: "白痴"},
	{Combo: []string{"shen", "jing", "bing"}, Description: "神经病"},
	{Combo: []string{"shen", "bing"}, Description: "不雅词汇"},
	{Combo: []string{"dai", "mao"}, Description: "玳帽（消极寓意）"},
	{Combo: []string{"liu", "mang"}, Description: "流氓"},
	{Combo: []string{"dou", "bi"}, Description: "不雅词汇"},
	{Combo: []string{"hei", "bang"}, Description: "黑帮"},
	{Combo: []string{"huang", "se"}, Description: "黄色（情色）"},
	{Combo: []string{"di", "yu"}, Description: "地狱"},
	{Combo: []string{"e", "gui"}, Description: "饿鬼"},
	{Combo: []string{"yao", "guai"}, Description: "妖怪"},
	{Combo: []string{"li", "gui"}, Description: "厉鬼"},
	{Combo: []string{"tao", "wang"}, Description: "逃亡"},
	{Combo: []string{"shang", "hai"}, Description: "上海（地名，用于名字不妥）"},
	{Combo: []string{"si", "wang"}, Description: "死亡"},
	{Combo: []string{"bei", "ju"}, Description: "悲剧"},
	{Combo: []string{"can", "fei"}, Description: "残废"},
	{Combo: []string{"shi", "gu"}, Description: "事故"},
	{Combo: []string{"an", "le", "si"}, Description: "安乐死"},
	{Combo: []string{"shi", "bai"}, Description: "失败"},
	{Combo: []string{"mei", "gui"}, Description: "玫瑰（用于名字花哨）"},
}

// stripTone 去除拼音声调数字后缀（"wang2" → "wang"）
func stripTone(pinyin string) string {
	if pinyin == "" {
		return ""
	}
	runes := []rune(pinyin)
	last := string(runes[len(runes)-1])
	if last >= "1" && last <= "4" {
		return string(runes[:len(runes)-1])
	}
	return pinyin
}

// CheckBadHomophone 检测单个拼音是否有不吉谐音
// 返回 (是否命中, 命中描述)
func CheckBadHomophone(pinyin string) (bool, string) {
	clean := stripTone(pinyin)
	if clean == "" {
		return false, ""
	}
	for _, h := range BadHomophones {
		if clean == h.Pinyin {
			return true, "谐音「" + h.Word + "」" + h.Description
		}
	}
	return false, ""
}

// CheckAllBadHomophones 检测名字全部拼音是否有不吉谐音
// 返回 (是否命中, 描述列表)
func CheckAllBadHomophones(pinyins ...string) (bool, []string) {
	var details []string
	for _, p := range pinyins {
		if hit, desc := CheckBadHomophone(p); hit {
			details = append(details, desc)
		}
	}
	return len(details) > 0, details
}

// CheckBadPinyinCombo 检测名字拼音连读是否产生不良词汇
// surname 姓氏拼音，givenNames 名字拼音列表
func CheckBadPinyinCombo(surnamePinyin string, givenPinyins ...string) (bool, string) {
	// 构建完整拼音序列（姓氏 + 名字各字）
	var allPinyins []string
	if surnamePinyin != "" {
		allPinyins = append(allPinyins, stripTone(surnamePinyin))
	}
	for _, p := range givenPinyins {
		allPinyins = append(allPinyins, stripTone(p))
	}
	if len(allPinyins) == 0 {
		return false, ""
	}

	// 滑动窗口检查所有组合
	joined := strings.Join(allPinyins, "")

	for _, combo := range BadPinyinCombos {
		comboStr := strings.Join(combo.Combo, "")
		if strings.Contains(joined, comboStr) {
			return true, "拼音连读「" + comboStr + "」" + combo.Description
		}
	}

	return false, ""
}
