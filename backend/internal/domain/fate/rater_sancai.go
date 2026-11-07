package fate

import (
	"strings"

	"name/internal/domain/zodiac"
)

// SancaiRater 天地人三才评分
//
// 基于《易经·说卦传》三才理论：
//   - 天道曰阴与阳 → 笔画奇偶搭配（奇为阳，偶为阴）
//   - 地道曰柔与刚 → 五行生克关系（相生为顺）
//   - 人道曰仁与义 → 部首字形互补（不同部首代表多元文化内涵）
type SancaiRater struct {
	weight float64
}

func NewSancaiRater() *SancaiRater {
	return &SancaiRater{weight: 0.10}
}

func NewSancaiRaterWithWeight(w float64) *SancaiRater {
	return &SancaiRater{weight: w}
}

func (r *SancaiRater) Name() string    { return "三才" }
func (r *SancaiRater) Weight() float64 { return r.weight }

func (r *SancaiRater) Rate(candidate *NameCandidate, fateData *FateData) NameRating {
	score := 70.0
	ds := newDetailSink(candidate)

	// ——— 天道：阴阳（笔画奇偶搭配）———
	if candidate.Stroke1 > 0 && candidate.Stroke2 > 0 {
		yinYang1 := candidate.Stroke1 % 2 // 1=阳,0=阴
		yinYang2 := candidate.Stroke2 % 2
		if yinYang1 != yinYang2 {
			score += 10
			ds.add("天道阴阳调和：两字笔画一奇一偶")
		} else {
			score += 3
			ds.add("天道阴阳相谐：两字笔画同奇/同偶")
		}

		// 笔画匀称度
		diff := absInt(candidate.Stroke1 - candidate.Stroke2)
		if diff <= 5 {
			score += 5
			ds.add("笔画搭配匀称，刚柔相济")
		} else if diff <= 10 {
			score += 2
			ds.add("笔画差异适中")
		} else {
			score -= 3
			ds.add("笔画差异偏大")
		}
	}

	// ——— 地道：刚柔（五行生克关系）———
	if candidate.WuXing1 != "" && candidate.WuXing2 != "" {
		if isWuXingSheng(candidate.WuXing1, candidate.WuXing2) ||
			isWuXingSheng(candidate.WuXing2, candidate.WuXing1) {
			score += 10
			ds.add("地道刚柔相济：两字五行相生")
		} else if candidate.WuXing1 == candidate.WuXing2 {
			score += 5
			ds.add("地道同气连枝：两字五行相同")
		} else if isWuXingKe(candidate.WuXing1, candidate.WuXing2) ||
			isWuXingKe(candidate.WuXing2, candidate.WuXing1) {
			score -= 5
			ds.add("地道五行相克，需注意调和")
		} else {
			score += 3
			ds.add("地道五行平和")
		}
	}

	// ——— 人道：仁义（部首字形互补）———
	if candidate.Radical1 != "" && candidate.Radical2 != "" {
		if candidate.Radical1 != candidate.Radical2 {
			score += 5
			ds.add("人道多元互补：两字部首不同")
		} else {
			score += 2
			ds.add("人道一脉相承：两字部首相同")
		}
	}

	// 双字齐全加分（单名降级）——三才以双名为佳
	if candidate.Char2 != "" {
		score += 3
		ds.add("双名齐备，天地人三才俱全")

		// 总笔画区间评分（姓+名1+名2 总笔画）
		// 姓氏笔画通常 3-20，名字两字合计 8-30 为宜
		// 总笔画过少（≤10）显得单薄，过多（≥45）书写负担重
		if candidate.Stroke1 > 0 && candidate.Stroke2 > 0 {
			totalStroke := candidate.Stroke1 + candidate.Stroke2
			switch {
			case totalStroke <= 8:
				score -= 3
				ds.addf("名之总笔画%d画偏少，略显单薄", totalStroke)
			case totalStroke >= 10 && totalStroke <= 24:
				score += 4
				ds.addf("名之总笔画%d画，匀称得体", totalStroke)
			case totalStroke >= 25 && totalStroke <= 30:
				score += 1
				ds.addf("名之总笔画%d画，稍繁但可接受", totalStroke)
			case totalStroke >= 31:
				score -= 3
				ds.addf("名之总笔画%d画偏多，书写负担重", totalStroke)
			}
		}
	}

	// 单名（无第二字）笔画适中度检查
	// 背景：单名时 Stroke2==0，三才维度的阴阳/匀称/五行生克/部首互补全部跳过，
	// 历史上单名三才恒定 70 无区分度（与音韵恒定 80 并列构成单名全维同分的半壁）。
	// 此处以单字笔画适中度补充区分度：过简（≤4 画）或过繁（≥23 画）都不宜入名，
	// 与 AGENTS.md「避免笔画过于复杂名字」约束一致。
	if candidate.Char2 == "" && candidate.Stroke1 > 0 {
		switch {
		case candidate.Stroke1 <= 4:
			score -= 3
			ds.add("单字笔画过简，略显单薄")
		case candidate.Stroke1 >= 23:
			score -= 5
			ds.add("单字笔画过繁，书写负担重")
		case candidate.Stroke1 >= 6 && candidate.Stroke1 <= 18:
			score += 6
			ds.add("单字笔画适中，书写匀称")
		}
	}

	score = clampScore(score)
	detail := ds.String("天地人三才搭配均衡，阴阳刚柔相济")
	return NameRating{Score: score, Detail: detail}
}

// --- 辅助函数 ---

func clampScore(s float64) float64 {
	if s > 100 {
		return 100
	}
	if s < 0 {
		return 0
	}
	return s
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// 五行生克表
var wuXingShengMap = map[string]string{
	"木": "火", "火": "土", "土": "金", "金": "水", "水": "木",
}

var wuXingKeMap = map[string]string{
	"木": "土", "土": "水", "水": "火", "火": "金", "金": "木",
}

func isWuXingSheng(a, b string) bool {
	if v, ok := wuXingShengMap[a]; ok {
		return v == b
	}
	return false
}

func isWuXingKe(a, b string) bool {
	if v, ok := wuXingKeMap[a]; ok {
		return v == b
	}
	return false
}

func getZodiacWuXing(zodiacName string) string {
	z := zodiac.GetZodiacByName(zodiacName)
	if z == nil {
		return ""
	}
	return z.Wuxing
}

// getToneFromPinyin 提取拼音声调（1/2/3/4），返回 0 表示无法识别
//
// 支持两种格式（生产数据 hanzi.json 为 Unicode 声调符号，测试常用数字后缀）：
//   - 数字后缀：wang2 → 2
//   - Unicode 声调符号：wáng → 2、tí → 2、shǎnɡ → 3
//
// 修复背景：旧实现只识别末尾数字，而生产数据全部是 Unicode 声调符号，
// 导致单名/双名音韵的声调维度全部失效（tone 恒为 0，比较被跳过）——
// 这是单名 Top5 长期全同分、荒谬字与好字并列的根因之一。
func getToneFromPinyin(pinyin string) int {
	pinyin = strings.TrimSpace(pinyin)
	if pinyin == "" {
		return 0
	}
	runes := []rune(pinyin)
	// 1. 数字后缀（wang2 → 2）
	last := runes[len(runes)-1]
	if last >= '1' && last <= '4' {
		return int(last - '0')
	}
	// 2. Unicode 声调符号（wáng → 2、tí → 2、shǎnɡ → 3）
	for _, r := range runes {
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

var shengmuList = []string{"zh", "ch", "sh", "b", "p", "m", "f", "d", "t", "n", "l", "g", "k", "h", "j", "q", "x", "z", "c", "s", "r", "y", "w"}

func getShengMu(pinyin string) string {
	for _, sm := range shengmuList {
		if len(pinyin) >= len(sm) && pinyin[:len(sm)] == sm {
			return sm
		}
	}
	return ""
}

func getYunMu(pinyin string) string {
	sm := getShengMu(pinyin)
	if sm == "" {
		return pinyin
	}
	return pinyin[len(sm):]
}

// --- NoveltyRater 新颖度评分器 ---

// NoveltyRater 新颖度评分
// 考察名字组合的新颖程度：惩罚同义叠加、笔画失衡、常用套路组合
// 奖励：繁简搭配、语义差异化
type NoveltyRater struct {
	weight float64
}
