// Package schema 定义物料的规范基座（Canonical*）与受控词表
//
// 定位（docs/31 ADR-009）：离线 JSON、中间格式、SQLite 表三者的唯一规范来源。
// 运行时 typed 结构与 SQLite 行都从 Canonical* 派生，杜绝三处各写一份而漂移。
package schema

import (
	"sort"
	"strings"
)

// Gender 规范性别枚举
type Gender string

const (
	// Male 男宝适用
	Male Gender = "male"
	// Female 女宝适用
	Female Gender = "female"
	// Neutral 中性（不限性别）
	//
	// 覆盖实测 8105 字中 6266 字无性别标注的情况（docs/31 ADR-014）：
	// 空值若不归一，性别筛选会把这些字整批误剔除
	Neutral Gender = "neutral"
)

// 性别枚举受控词表
var genderSet = map[Gender]bool{Male: true, Female: true, Neutral: true}

// ValidGender 判断是否为规范性别值
func ValidGender(g Gender) bool { return genderSet[g] }

// FiveElements 五行受控词表
var wuxingSet = map[string]bool{"金": true, "木": true, "水": true, "火": true, "土": true}

// ValidWuxing 判断是否为规范五行（空串合法，表示「不臆造」）
func ValidWuxing(w string) bool { return w == "" || wuxingSet[w] }

// Word 规范汉字基座：由 namer.json 逐字记录 + 跨物料合并派生
type Word struct {
	Char          string   `json:"char"`
	Pinyin        string   `json:"pinyin"`
	Strokes       int      `json:"strokes"`
	KangxiStrokes int      `json:"kangxiStrokes"`
	Radical       string   `json:"radical"`
	Meaning       string   `json:"meaning"`
	Wuxing        string   `json:"wuxing"`
	Level         int      `json:"level"`
	Gender        Gender   `json:"gender"`
	Tone          int      `json:"tone"`
	NameFreqTier  int      `json:"nameFreqTier"`
	UsageLevel    int      `json:"usageLevel"`
	PositiveScore int      `json:"positiveScore"`
	PhoneticScore int      `json:"phoneticScore"`
	ModernScore   int      `json:"modernScore"`
	ClassicalSc   int      `json:"classicalScore"`
	CurationLevel int      `json:"curationLevel"`
	NamePenalty   int      `json:"namePenalty"`
	IsPolyphonic  bool     `json:"isPolyphonic"`
	IsRare        bool     `json:"isRare"`
	IsNegative    bool     `json:"isNegative"`
	StyleTags     []string `json:"styleTags"`
	GenderTags    []string `json:"genderTags"`
	PairBlacklist []string `json:"pairBlacklist"`
	NamingCats    []string `json:"namingCategories"`
}

// CharGroup 偏旁分组（namer.json 顶层 charGroups，实测 29 组）
type CharGroup struct {
	Radical string   `json:"radical"`
	Name    string   `json:"name"`
	Meaning string   `json:"meaning"`
	Chars   []string `json:"chars"`
}

// Meaning 词典释义条目（words 表的行模型）
type Meaning struct {
	Char        string `json:"char"`
	Explanation string `json:"explanation"`
}

// CharFreq 单字人名频率
type CharFreq struct {
	Char  string `json:"char"`
	Count int    `json:"count"`
	Rank  int    `json:"rank"`
	Tier  int    `json:"tier"`
}

// BigramFreq 双字人名频率
type BigramFreq struct {
	Char1 string `json:"char1"`
	Char2 string `json:"char2"`
	Count int    `json:"count"`
	Rank  int    `json:"rank"`
	Tier  int    `json:"tier"`
}

// Key 返回双字频率的主键（按 ADR-009 §9.8 校验规则，key 必须等于 Char1+Char2）
func (b BigramFreq) Key() string { return b.Char1 + b.Char2 }

// KangxiStroke 康熙笔画条目
type KangxiStroke struct {
	Char    string `json:"char"`
	Strokes int    `json:"strokes"`
}

// Hexagram 易经卦条目
type Hexagram struct {
	ID             int      `json:"id"`
	Name           string   `json:"name"`
	Number         int      `json:"number"`
	Symbol         string   `json:"symbol"`
	UpperTrigram   int      `json:"upperTrigram"`
	LowerTrigram   int      `json:"lowerTrigram"`
	GuaCi          string   `json:"guaCi"`
	XiangCi        string   `json:"xiangCi"`
	YaoCi          []string `json:"yaoCi"`
	Interpretation string   `json:"interpretation"`
}

// Zodiac 生肖条目
type Zodiac struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Wuxing       string   `json:"wuxing"`
	Compatible   []string `json:"compatible"`
	Conflicting  []string `json:"conflicting"`
	AvoidChars   string   `json:"avoidChars"`
	LuckyNumbers []int    `json:"luckyNumbers"`
}

// NormalizeTags 去空、去重并排序：让同内容的标签序列序列化稳定
func NormalizeTags(in []string) []string {
	if len(in) == 0 {
		return []string{}
	}
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
