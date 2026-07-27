package fate

import (
	"fmt"
	"math"
	"strings"

	"name/internal/domain/classics"
)

// Rater 名字评分接口
//
// 借鉴 fate-main 的五维评分体系，每个 Rater 负责一个维度的评分:
//   - WuxingRater:   五行八字匹配度（权重 30%，命理层核心）
//   - WenHuaRater:   文化印象 — 常用度/字义丰富度（权重 20%，字象层）
//   - YinYunRater:   音韵和谐度（权重 20%，字象层）
//   - ShengXiaoRater:生肖匹配度（权重 15%，命理层）
//   - SancaiRater:   天地人三才搭配（权重 15%，新增维度）
//
// 注：原 WuGeRater（熊崎五格数理）已移除，遵循 AGENTS.md 禁用熊崎五格约束。
type Rater interface {
	// Rate 给名字候选评分
	Rate(candidate *NameCandidate, fateData *FateData) NameRating
	// Name 评分维度名称
	Name() string
	// Weight 该维度在总分中的权重（所有 Rater 权重之和应为 1.0）
	Weight() float64
}

// NameRating 单维度评分结果
type NameRating struct {
	Score  float64 `json:"score"`  // 该维度得分（0-100）
	Detail string  `json:"detail"` // 文字解释
}

// RateName 使用一组 Rater 计算名字的综合评分
func RateName(candidate *NameCandidate, fateData *FateData, raters []Rater) NameScore {
	items := make(map[string]float64, len(raters))
	var total float64

	for _, r := range raters {
		rating := r.Rate(candidate, fateData)
		items[r.Name()] = rating.Score
		total += rating.Score * r.Weight()
	}

	// 限制总分在 0-100 范围内
	total = clampScore(total)
	
	// 四舍五入保留一位小数
	total = math.Round(total*10) / 10

	return NameScore{
		Total: total,
		Grade: scoreToGrade(total),
		Items: items,
	}
}

// scoreToGrade 将分数转换为等级
func scoreToGrade(score float64) string {
	switch {
	case score >= 90:
		return "上上"
	case score >= 80:
		return "上吉"
	case score >= 70:
		return "中吉"
	case score >= 60:
		return "中平"
	case score >= 50:
		return "中下"
	default:
		return "下下"
	}
}

// --- 内置 Rater 实现 ---

// WuxingRater 五行八字匹配度评分
// 考察名字五行与八字喜用神的匹配程度
type WuxingRater struct {
	weight float64
}

func NewWuxingRater() *WuxingRater {
	return &WuxingRater{weight: 0.25}
}

func NewWuxingRaterWithWeight(w float64) *WuxingRater {
	return &WuxingRater{weight: w}
}

func (r *WuxingRater) Name() string   { return "五行八字" }
func (r *WuxingRater) Weight() float64 { return r.weight }

func (r *WuxingRater) Rate(candidate *NameCandidate, fateData *FateData) NameRating {
	if fateData == nil {
		return NameRating{Score: 80, Detail: "五行信息良好"}
	}

	xiWuxing := fateData.WuXingXiji.Xi
	jiWuxing := fateData.WuXingXiji.Ji

	score := 50.0
	var details []string
	matchCount := 0

	chars := []struct {
		char  string
		wuxing string
	}{{candidate.Char1, candidate.WuXing1}, {candidate.Char2, candidate.WuXing2}}

	for _, c := range chars {
		if c.char == "" || c.wuxing == "" {
			continue
		}
		switch {
		case c.wuxing == xiWuxing:
			score += 15
			matchCount++
			details = append(details, fmt.Sprintf("「%s」五行属%s，为喜用神", c.char, c.wuxing))
		case c.wuxing == jiWuxing:
			score -= 10
			details = append(details, fmt.Sprintf("「%s」五行属%s，为忌神", c.char, c.wuxing))
		default:
			score += 3
			details = append(details, fmt.Sprintf("「%s」五行属%s，中性", c.char, c.wuxing))
		}
	}

	// 两字均匹配喜用神额外加分
	if matchCount == 2 {
		score += 5
		details = append(details, "两字皆匹配喜用神，补益力强")
	}

	// 两字五行相生加分
	if candidate.WuXing1 != "" && candidate.WuXing2 != "" {
		if isWuXingSheng(candidate.WuXing1, candidate.WuXing2) ||
			isWuXingSheng(candidate.WuXing2, candidate.WuXing1) {
			score += 10
			details = append(details, "两字五行相生，搭配协调")
		}
		if isWuXingKe(candidate.WuXing1, candidate.WuXing2) ||
			isWuXingKe(candidate.WuXing2, candidate.WuXing1) {
			score -= 8
			details = append(details, "两字五行相克，需注意")
		}
	}

	score = clampScore(score)
	detail := "五行信息良好"
	if len(details) > 0 {
		detail = strings.Join(details, "；")
	}
	return NameRating{Score: score, Detail: detail}
}

// DefaultRaters 返回默认的五维评分器列表
//
// 移除了 WuGeRater（熊崎五格数理），新增 SancaiRater（天地人三才），权重重新分配：
//   - WuxingRater 30%（命理层核心，原 35%，腾挪 5% 给三才）
//   - WenHuaRater 20%（字象层，原 25%，腾挪 5% 给三才）
//   - YinYunRater 20%（字象层，原 25%，腾挪 5% 给三才）
//   - ShengXiaoRater 15%（命理层，不变）
//   - SancaiRater  15%（新增维度：天地人三才搭配）
func DefaultRaters() []Rater {
	return []Rater{
		NewWuxingRaterWithWeight(0.30),    // 30% 命理层
		NewWenHuaRaterWithWeight(0.20),    // 20% 字象层
		NewYinYunRaterWithWeight(0.20),    // 20% 字象层
		NewShengXiaoRaterWithWeight(0.15), // 15% 命理层
		NewSancaiRaterWithWeight(0.15),    // 15% 新增：天地人三才
	}
}

// WenHuaRater 文化印象评分
// 考察常用度、字义丰富度、笔画匀称度
type WenHuaRater struct {
	weight float64
}

func NewWenHuaRater() *WenHuaRater {
	return &WenHuaRater{weight: 0.20}
}

func NewWenHuaRaterWithWeight(w float64) *WenHuaRater {
	return &WenHuaRater{weight: w}
}

func (r *WenHuaRater) Name() string   { return "文化印象" }
func (r *WenHuaRater) Weight() float64 { return r.weight }

func (r *WenHuaRater) Rate(candidate *NameCandidate, fateData *FateData) NameRating {
	score := 60.0
	var details []string

	// 常用字加分
	if candidate.IsRegular {
		score += 5
	}
	// 字义明确加分
	if candidate.Meaning1 != "" {
		score += 4
	}
	if candidate.Meaning2 != "" {
		score += 4
	}
	// 笔画匀称加分
	if candidate.Stroke1 > 0 && candidate.Stroke2 > 0 {
		diff := absInt(candidate.Stroke1 - candidate.Stroke2)
		if diff <= 5 {
			score += 2
			details = append(details, "笔画搭配匀称")
		}
	}

	// ——— 诗词出处加分（含近义语义扩展） ———

	// 精确匹配（由 name generator 预计算）
	if candidate.HasPoetry {
		score += 8
		detail := "出自诗词典故"
		if candidate.PoetryFrom != "" {
			detail = "出自" + candidate.PoetryFrom
		}
		details = append(details, detail)
	}

	// 语义扩展匹配（精确匹配未覆盖时检查近义字关联）
	if !candidate.HasPoetry {
		if candidate.Char1 != "" {
			if desc := checkSemanticPoetry(candidate.Char1); desc != "" {
				score += 5
				details = append(details, desc)
			}
		}
		if candidate.Char2 != "" {
			if desc := checkSemanticPoetry(candidate.Char2); desc != "" {
				score += 5
				details = append(details, desc)
			}
		}
	}

	// 二字共现加分（诗经楚辞等经典中的同句搭配）
	if candidate.Char1 != "" && candidate.Char2 != "" && candidate.Char1 != candidate.Char2 {
		if bgScore, bgSrc, found := classics.GetBigramScore(candidate.Char1, candidate.Char2); found {
			score += float64(bgScore)
			details = append(details, fmt.Sprintf("「%s%s」共现于%s（+%d分）",
				candidate.Char1, candidate.Char2, bgSrc, bgScore))
		}
	}

	// 单名语义共现：Char1 的近义字与 Char1 在诗词中的搭配
	if candidate.Char2 == "" && candidate.Char1 != "" {
		if desc := checkSingleNameBigram(candidate.Char1); desc != "" {
			score += 3
			details = append(details, desc)
		}
	}

	score = clampScore(score)
	detail := "文化印象良好"
	if len(details) > 0 {
		detail = strings.Join(details, "；")
	}
	return NameRating{Score: score, Detail: detail}
}

// checkSemanticPoetry 检查字符是否有近义关联的诗词出处
// 返回描述文字（空字符串表示无匹配）
func checkSemanticPoetry(char string) string {
	_, synChar, entries := classics.FindPoetryByCharSemantic(char)
	if synChar != "" && len(entries) > 0 {
		entry := entries[0]
		src := entry.Sentence
		if src == "" {
			src = entry.Work + "·" + entry.Chapter
		}
		return fmt.Sprintf("「%s」近义于「%s」，关联「%s」", char, synChar, src)
	}
	return ""
}

// checkSingleNameBigram 检查单名是否通过近义字获得诗词共现加分
func checkSingleNameBigram(char string) string {
	if synChar, coChar, src, found := classics.GetSemanticBigramMatch(char); found {
		return fmt.Sprintf("「%s」借「%s·%s」共现于%s（+3分）", char, synChar, coChar, src)
	}
	return ""
}

// YinYunRater 音韵评分
// 考察声调变化、声母韵母搭配
type YinYunRater struct {
	weight float64
}

func NewYinYunRater() *YinYunRater {
	return &YinYunRater{weight: 0.20}
}

func NewYinYunRaterWithWeight(w float64) *YinYunRater {
	return &YinYunRater{weight: w}
}

func (r *YinYunRater) Name() string   { return "音韵" }
func (r *YinYunRater) Weight() float64 { return r.weight }

func (r *YinYunRater) Rate(candidate *NameCandidate, fateData *FateData) NameRating {
	score := 80.0
	var details []string

	p1 := candidate.Pinyin1
	p2 := candidate.Pinyin2

	if p1 == "" || p2 == "" {
		return NameRating{Score: score, Detail: "音韵信息良好"}
	}

	// 声调检查
	tone1 := getToneFromPinyin(p1)
	tone2 := getToneFromPinyin(p2)
	if tone1 != tone2 && tone1 != 0 && tone2 != 0 {
		score += 8
		details = append(details, "两字声调不同，抑扬顿挫")
	} else if tone1 == tone2 && tone1 != 0 {
		score -= 5
		details = append(details, "两字声调相同")
	}

	// 声母检查
	sm1 := getShengMu(p1)
	sm2 := getShengMu(p2)
	if sm1 != sm2 && sm1 != "" && sm2 != "" {
		score += 5
		details = append(details, "声母不同，发音清晰")
	} else if sm1 == sm2 && sm1 != "" {
		score -= 3
	}

	// 韵母检查
	ym1 := getYunMu(p1)
	ym2 := getYunMu(p2)
	if ym1 != ym2 && ym1 != "" && ym2 != "" {
		score += 4
		details = append(details, "韵母不同，朗朗上口")
	} else if ym1 == ym2 && ym1 != "" {
		score -= 3
	}

	// ——— 谐音检测（包含姓氏拼音，确保检测跨字谐音如"杜子腾"→肚子疼） ———

	// 1. 逐字检测不吉谐音
	allPinyins := []string{p1}
	if candidate.SurnamePinyin != "" {
		allPinyins = append([]string{candidate.SurnamePinyin}, allPinyins...)
	}
	if p2 != "" {
		allPinyins = append(allPinyins, p2)
	}
	if hit, descs := CheckAllBadHomophones(allPinyins...); hit {
		penalty := float64(len(descs)) * 10
		score -= penalty
		details = append(details, "含不吉谐音: "+strings.Join(descs, "、"))
	}

	// 2. 拼音连读不良组合检测（包含姓氏拼音）
	surnamePy := candidate.SurnamePinyin
	if hit, comboDesc := CheckBadPinyinCombo(surnamePy, p1, p2); hit {
		score -= 15
		details = append(details, comboDesc)
	}

	score = clampScore(score)
	detail := "音韵信息良好"
	if len(details) > 0 {
		detail = strings.Join(details, "；")
	}
	return NameRating{Score: score, Detail: detail}
}

// ShengXiaoRater 生肖匹配评分
// 考察名字五行与生肖五行的生克关系
type ShengXiaoRater struct {
	weight float64
}

func NewShengXiaoRater() *ShengXiaoRater {
	return &ShengXiaoRater{weight: 0.10}
}

func NewShengXiaoRaterWithWeight(w float64) *ShengXiaoRater {
	return &ShengXiaoRater{weight: w}
}

func (r *ShengXiaoRater) Name() string   { return "生肖" }
func (r *ShengXiaoRater) Weight() float64 { return r.weight }

func (r *ShengXiaoRater) Rate(candidate *NameCandidate, fateData *FateData) NameRating {
	if fateData == nil {
		return NameRating{Score: 80, Detail: "生肖信息良好"}
	}

	score := 80.0
	var details []string
	zodiac := fateData.BaziInfo.Zodiac
	zodiacWx := getZodiacWuXing(zodiac)

	if zodiacWx == "" {
		return NameRating{Score: score, Detail: "生肖信息良好"}
	}

	details = append(details, fmt.Sprintf("生肖%s，五行属%s", zodiac, zodiacWx))

	for _, pair := range []struct {
		char  string
		wuxing string
	}{{candidate.Char1, candidate.WuXing1}, {candidate.Char2, candidate.WuXing2}} {
		if pair.char == "" || pair.wuxing == "" {
			continue
		}
		if isWuXingSheng(zodiacWx, pair.wuxing) || isWuXingSheng(pair.wuxing, zodiacWx) {
			score += 7
			details = append(details, fmt.Sprintf("「%s」与生肖五行相生", pair.char))
		}
		if isWuXingKe(zodiacWx, pair.wuxing) || isWuXingKe(pair.wuxing, zodiacWx) {
			score -= 5
			details = append(details, fmt.Sprintf("「%s」与生肖五行相克", pair.char))
		}
	}

	score = clampScore(score)
	detail := "生肖信息良好"
	if len(details) > 0 {
		detail = strings.Join(details, "；")
	}
	return NameRating{Score: score, Detail: detail}
}

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

func (r *SancaiRater) Name() string   { return "三才" }
func (r *SancaiRater) Weight() float64 { return r.weight }

func (r *SancaiRater) Rate(candidate *NameCandidate, fateData *FateData) NameRating {
	score := 70.0
	var details []string

	// ——— 天道：阴阳（笔画奇偶搭配）———
	if candidate.Stroke1 > 0 && candidate.Stroke2 > 0 {
		yinYang1 := candidate.Stroke1 % 2 // 1=阳,0=阴
		yinYang2 := candidate.Stroke2 % 2
		if yinYang1 != yinYang2 {
			score += 10
			details = append(details, "天道阴阳调和：两字笔画一奇一偶")
		} else {
			score += 3
			details = append(details, "天道阴阳相谐：两字笔画同奇/同偶")
		}

		// 笔画匀称度
		diff := absInt(candidate.Stroke1 - candidate.Stroke2)
		if diff <= 5 {
			score += 5
			details = append(details, "笔画搭配匀称，刚柔相济")
		} else if diff <= 10 {
			score += 2
			details = append(details, "笔画差异适中")
		} else {
			score -= 3
			details = append(details, "笔画差异偏大")
		}
	}

	// ——— 地道：刚柔（五行生克关系）———
	if candidate.WuXing1 != "" && candidate.WuXing2 != "" {
		if isWuXingSheng(candidate.WuXing1, candidate.WuXing2) ||
			isWuXingSheng(candidate.WuXing2, candidate.WuXing1) {
			score += 10
			details = append(details, "地道刚柔相济：两字五行相生")
		} else if candidate.WuXing1 == candidate.WuXing2 {
			score += 5
			details = append(details, "地道同气连枝：两字五行相同")
		} else if isWuXingKe(candidate.WuXing1, candidate.WuXing2) ||
			isWuXingKe(candidate.WuXing2, candidate.WuXing1) {
			score -= 5
			details = append(details, "地道五行相克，需注意调和")
		} else {
			score += 3
			details = append(details, "地道五行平和")
		}
	}

	// ——— 人道：仁义（部首字形互补）———
	if candidate.Radical1 != "" && candidate.Radical2 != "" {
		if candidate.Radical1 != candidate.Radical2 {
			score += 5
			details = append(details, "人道多元互补：两字部首不同")
		} else {
			score += 2
			details = append(details, "人道一脉相承：两字部首相同")
		}
	}

	// 双字齐全加分（单名降级）——三才以双名为佳
	if candidate.Char2 != "" {
		score += 5
		details = append(details, "双名齐备，天地人三才俱全")
	}

	score = clampScore(score)
	detail := "三才信息良好"
	if len(details) > 0 {
		detail = strings.Join(details, "；")
	}
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

func getZodiacWuXing(zodiac string) string {
	m := map[string]string{
		"鼠": "水", "牛": "土", "虎": "木", "兔": "木",
		"龙": "土", "蛇": "火", "马": "火", "羊": "土",
		"猴": "金", "鸡": "金", "狗": "土", "猪": "水",
	}
	return m[zodiac]
}

func getToneFromPinyin(pinyin string) int {
	runes := []rune(pinyin)
	if len(runes) == 0 {
		return 0
	}
	last := runes[len(runes)-1]
	if last >= '1' && last <= '4' {
		return int(last - '0')
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
