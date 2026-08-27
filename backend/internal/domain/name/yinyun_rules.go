package name

import (
	"strings"
	"unicode/utf8"
)

// ============================================================
// 声母分类 — 用于叠双声检测
// ============================================================

// ShengmuCategory 声母按发音部位分类
var ShengmuCategory = map[string]string{
	"b": "唇音", "p": "唇音", "m": "唇音", "f": "唇音",
	"d": "舌尖中音", "t": "舌尖中音", "n": "舌尖中音", "l": "舌尖中音",
	"g": "舌根音", "k": "舌根音", "h": "舌根音",
	"j": "舌面音", "q": "舌面音", "x": "舌面音",
	"zh": "舌尖后音", "ch": "舌尖后音", "sh": "舌尖后音", "r": "舌尖后音",
	"z": "舌尖前音", "c": "舌尖前音", "s": "舌尖前音",
	"y": "零声母", "w": "零声母",
}

// ============================================================
// 韵母分类 — 用于叠韵检测
// ============================================================

// YunmuCategory 韵母按类别分组（开口呼/齐齿呼/合口呼/撮口呼）
var YunmuCategory = map[string]string{
	"a": "开口呼", "o": "开口呼", "e": "开口呼", "ê": "开口呼",
	"ai": "开口呼", "ei": "开口呼", "ao": "开口呼", "ou": "开口呼",
	"an": "开口呼", "en": "开口呼", "ang": "开口呼", "eng": "开口呼",
	"er": "开口呼",
	"i": "齐齿呼", "ia": "齐齿呼", "ie": "齐齿呼", "iao": "齐齿呼",
	"iou": "齐齿呼", "ian": "齐齿呼", "in": "齐齿呼", "iang": "齐齿呼",
	"ing": "齐齿呼",
	"u": "合口呼", "ua": "合口呼", "uo": "合口呼", "uai": "合口呼",
	"uei": "合口呼", "uan": "合口呼", "uen": "合口呼", "uang": "合口呼",
	"ueng": "合口呼", "ong": "合口呼",
	"ü": "撮口呼", "üe": "撮口呼", "üan": "撮口呼", "ün": "撮口呼",
	"iong": "撮口呼",
}

// ============================================================
// 平仄评分 — 三字名 64 种组合评分表
// ============================================================

// PingZeLevel 平仄评级
type PingZeLevel int

const (
	PingZeExcellent PingZeLevel = 5 // 极佳
	PingZeGood      PingZeLevel = 4 // 良好
	PingZeFair      PingZeLevel = 3 // 中等
	PingZePoor      PingZeLevel = 2 // 较差
	PingZeBad       PingZeLevel = 1 // 差
)

// 平仄标记
const (
	Ping = "平" // 一声、二声
	Ze   = "仄" // 三声、四声
)

// pingzeScoreMap 三字名平仄组合评分（8种组合 × 双字名4种 = 12种）
// 评分原则：平仄交替最佳，连续同调较差
var pingzeScoreMap = map[string]PingZeLevel{
	// 三字名
	"平仄平": PingZeExcellent,
	"仄平仄": PingZeExcellent,
	"平仄仄": PingZeGood,
	"仄平平": PingZeGood,
	"平平仄": PingZeGood,
	"仄仄平": PingZeGood,
	"平平平": PingZePoor,
	"仄仄仄": PingZePoor,
	// 双字名
	"平仄": PingZeExcellent,
	"仄平": PingZeExcellent,
	"平平": PingZeFair,
	"仄仄": PingZeFair,
}

// ============================================================
// YinyunAnalyzer 音韵分析器
// ============================================================

// YinyunAnalyzer 音韵分析器
type YinyunAnalyzer struct{}

// NewYinyunAnalyzer 创建音韵分析器
func NewYinyunAnalyzer() *YinyunAnalyzer {
	return &YinyunAnalyzer{}
}

// extractShengmu 提取拼音的声母
func (ya *YinyunAnalyzer) extractShengmu(pinyin string) string {
	pinyin = strings.TrimSpace(pinyin)
	if pinyin == "" {
		return ""
	}

	// 检测双字母声母
	if strings.HasPrefix(pinyin, "zh") ||
		strings.HasPrefix(pinyin, "ch") ||
		strings.HasPrefix(pinyin, "sh") {
		return pinyin[:2]
	}

	// 单字母声母
	first := string([]rune(pinyin)[0])
	if _, ok := ShengmuCategory[first]; ok {
		return first
	}

	// 零声母
	return ""
}

// extractYunmu 提取拼音的韵母（去掉声调后的韵腹+韵尾）
func (ya *YinyunAnalyzer) extractYunmu(pinyin string) string {
	pinyin = strings.TrimSpace(pinyin)
	if pinyin == "" {
		return ""
	}

	// 去掉声调数字和声调符号（如 hào → hao）
	pinyin = ya.normalizePinyin(pinyin)

	// 去掉声母
	shengmu := ""
	if strings.HasPrefix(pinyin, "zh") || strings.HasPrefix(pinyin, "ch") || strings.HasPrefix(pinyin, "sh") {
		shengmu = pinyin[:2]
	} else {
		first := string([]rune(pinyin)[0])
		if _, ok := ShengmuCategory[first]; ok {
			shengmu = first
		}
	}

	if shengmu != "" {
		return pinyin[len(shengmu):]
	}
	return pinyin
}

// normalizePinyin 去掉声调数字和声调符号，得到纯拼音
func (ya *YinyunAnalyzer) normalizePinyin(pinyin string) string {
	return strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return -1
		}
		switch r {
		case 'ā', 'á', 'ǎ', 'à':
			return 'a'
		case 'ē', 'é', 'ě', 'è':
			return 'e'
		case 'ī', 'í', 'ǐ', 'ì':
			return 'i'
		case 'ō', 'ó', 'ǒ', 'ò':
			return 'o'
		case 'ū', 'ú', 'ǔ', 'ù':
			return 'u'
		case 'ǖ', 'ǘ', 'ǚ', 'ǜ':
			return 'v'
		case 'ń', 'ň', 'ǹ':
			return 'n'
		}
		return r
	}, pinyin)
}

// extractTone 提取拼音声调（1/2/3/4）
// 支持两种格式：
//   - 数字后缀：wang2 → 2
//   - Unicode 声调符号：wáng → 2
func (ya *YinyunAnalyzer) extractTone(pinyin string) int {
	pinyin = strings.TrimSpace(pinyin)
	if pinyin == "" {
		return 0
	}
	// 1. 数字后缀（wang2 → 2）
	last := pinyin[len(pinyin)-1]
	if last >= '1' && last <= '4' {
		return int(last - '0')
	}

	// 2. Unicode 声调符号（wáng → 2）
	for _, r := range pinyin {
		switch r {
		case 'ā', 'ē', 'ī', 'ō', 'ū', 'ǖ':
			return 1
		case 'á', 'é', 'í', 'ó', 'ú', 'ǘ', 'ń':
			return 2
		case 'ǎ', 'ě', 'ǐ', 'ǒ', 'ǔ', 'ǚ', 'ň':
			return 3
		case 'à', 'è', 'ì', 'ò', 'ù', 'ǜ', 'ǹ':
			return 4
		}
	}

	return 0
}

// getPingZe 获取声调对应的平仄
func (ya *YinyunAnalyzer) getPingZe(tone int) string {
	if tone == 1 || tone == 2 {
		return Ping
	}
	return Ze
}

// ============================================================
// 叠双声 / 叠韵检测
// ============================================================

// HasDieShuangSheng 检测两个拼音是否叠双声（声母属于同一发音部位）
// 例如：zhāng(张)和chén(陈)都是舌尖后音 → 叠双声 → 音感不佳
func (ya *YinyunAnalyzer) HasDieShuangSheng(pinyin1, pinyin2 string) bool {
	sm1 := ya.extractShengmu(pinyin1)
	sm2 := ya.extractShengmu(pinyin2)
	if sm1 == "" || sm2 == "" {
		return false
	}
	cat1, ok1 := ShengmuCategory[sm1]
	cat2, ok2 := ShengmuCategory[sm2]
	if !ok1 || !ok2 {
		return false
	}
	return cat1 == cat2 && sm1 != sm2
}

// HasDieYun 检测两个拼音是否叠韵（韵母属于同一分类且不同）
// 例如：fēng(风)和yáng(扬) → 都是开口呼 → 叠韵 → 音感不佳
func (ya *YinyunAnalyzer) HasDieYun(pinyin1, pinyin2 string) bool {
	ym1 := ya.extractYunmu(pinyin1)
	ym2 := ya.extractYunmu(pinyin2)
	if ym1 == "" || ym2 == "" {
		return false
	}
	cat1, ok1 := YunmuCategory[ym1]
	cat2, ok2 := YunmuCategory[ym2]
	if !ok1 || !ok2 {
		return false
	}
	return cat1 == cat2 && ym1 != ym2
}

// HasCompleteDieShuangSheng 检测是否完全叠双声（声母完全相同）
// 例如：zhāng(张)和zhōu(周) → 声母都是zh → 完全叠双声 → 很差
func (ya *YinyunAnalyzer) HasCompleteDieShuangSheng(pinyin1, pinyin2 string) bool {
	sm1 := ya.extractShengmu(pinyin1)
	sm2 := ya.extractShengmu(pinyin2)
	return sm1 != "" && sm1 == sm2
}

// HasCompleteDieYun 检测是否完全叠韵（韵母完全相同）
// 例如：fāng(方)和yáng(扬) → 韵母都是ang → 完全叠韵 → 较差
func (ya *YinyunAnalyzer) HasCompleteDieYun(pinyin1, pinyin2 string) bool {
	ym1 := ya.extractYunmu(pinyin1)
	ym2 := ya.extractYunmu(pinyin2)
	return ym1 != "" && ym1 == ym2
}

// ============================================================
// 平仄评分计算
// ============================================================

// ScorePingZe 评分平仄组合
func (ya *YinyunAnalyzer) ScorePingZe(pinyinParts []string) (score PingZeLevel, pattern string) {
	if len(pinyinParts) == 0 {
		return PingZeFair, ""
	}

	// 提取平仄序列
	var pzSeq []string
	for _, p := range pinyinParts {
		tone := ya.extractTone(p)
		pzSeq = append(pzSeq, ya.getPingZe(tone))
	}
	pattern = strings.Join(pzSeq, "")

	// 查评分表
	if s, ok := pingzeScoreMap[pattern]; ok {
		return s, pattern
	}

	// 默认中等
	return PingZeFair, pattern
}

// ============================================================
// 综合音韵评分（供 enhanced_generator 调用）
// ============================================================

// ScoreYinyun 综合音韵评分（0-100）
// 考虑维度：平仄搭配、叠双声、叠韵、同音检测
func (ya *YinyunAnalyzer) ScoreYinyun(pinyin string) (score float64, analysis string) {
	if pinyin == "" {
		return 70.0, "音韵分析：拼音信息缺失，默认中等评分"
	}

	pinyinParts := strings.Fields(pinyin)
	if len(pinyinParts) == 0 {
		return 70.0, "音韵分析：未获取到拼音分段"
	}

	var details []string
	score = 85.0 // 基础分

	// 1. 平仄评分（权重0.35）
	pzScore, _ := ya.ScorePingZe(pinyinParts)
	switch pzScore {
	case PingZeExcellent:
		details = append(details, "平仄搭配极佳，抑扬顿挫")
		score += 8
	case PingZeGood:
		details = append(details, "平仄搭配良好，朗朗上口")
		score += 5
	case PingZeFair:
		details = append(details, "平仄搭配中等，较为平淡")
	case PingZePoor:
		details = append(details, "平仄搭配较差，略显单调")
		score -= 5
	case PingZeBad:
		details = append(details, "平仄搭配不佳，建议调整")
		score -= 10
	}

	// 2. 叠双声检测（权重0.25）
	hasAlliteration := false
	hasFullAlliteration := false
	if len(pinyinParts) >= 2 {
		for i := 0; i < len(pinyinParts)-1; i++ {
			if ya.HasCompleteDieShuangSheng(pinyinParts[i], pinyinParts[i+1]) {
				hasFullAlliteration = true
			}
			if ya.HasDieShuangSheng(pinyinParts[i], pinyinParts[i+1]) {
				hasAlliteration = true
			}
		}
	}
	if hasFullAlliteration {
		details = append(details, "存在声母完全相同（完全叠双声），建议换字避免绕口")
		score -= 10
	} else if hasAlliteration {
		details = append(details, "存在叠双声（声母发音部位相同），建议留意读音")
		score -= 5
	}

	// 3. 叠韵检测（权重0.25）
	hasAssonance := false
	hasFullAssonance := false
	if len(pinyinParts) >= 2 {
		for i := 0; i < len(pinyinParts)-1; i++ {
			if ya.HasCompleteDieYun(pinyinParts[i], pinyinParts[i+1]) {
				hasFullAssonance = true
			}
			if ya.HasDieYun(pinyinParts[i], pinyinParts[i+1]) {
				hasAssonance = true
			}
		}
	}
	if hasFullAssonance {
		details = append(details, "存在韵母完全相同（完全叠韵），略显单调")
		score -= 8
	} else if hasAssonance {
		details = append(details, "存在叠韵，声韵搭配尚可")
		score -= 3
	}

	// 4. 韵母变化丰富度（权重0.15）
	if len(pinyinParts) >= 2 {
		yunmuSet := make(map[string]bool)
		for _, p := range pinyinParts {
			ym := ya.extractYunmu(p)
			if ym != "" {
				yunmuSet[ym] = true
			}
		}
		if len(yunmuSet) == len(pinyinParts) {
			details = append(details, "韵母变化丰富")
			score += 5
		}
	}

	// 限制评分范围
	if score > 100 {
		score = 100
	}
	if score < 30 {
		score = 30
	}

	analysis = "音韵分析：" + strings.Join(details, "；")
	return score, analysis
}

// IsRareChar 检查是否为生僻字（根据 Unicode 编码范围判断）
func IsRareChar(char string) bool {
	if char == "" {
		return false
	}
	r, _ := utf8.DecodeRuneInString(char)
	// CJK 统一表意文字扩展区（生僻字）
	if r >= 0x3400 && r <= 0x4DBF {
		return true
	}
	if r >= 0x20000 && r <= 0x2A6DF {
		return true
	}
	if r >= 0x2A700 && r <= 0x2B73F {
		return true
	}
	if r >= 0x2B740 && r <= 0x2B81F {
		return true
	}
	return false
}