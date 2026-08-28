package hanzi

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var RareChars = map[string]bool{
	"龘": true, "靐": true, "齉": true, "齾": true, "龗": true,
	"鱻": true, "麤": true, "灪": true, "馫": true, "譶": true,
	"灨": true, "灩": true, "麷": true, "驫": true, "羴": true,
	// 扩展超生僻字（《通用规范汉字表》表外字）
	"龖": true, "龠": true, "爨": true, "灥": true, "厵": true,
	"𪚥": true, "𠀾": true, "𡿻": true, "𠄌": true, "𠂉": true,
	"𠃋": true, "𠃌": true, "𠆢": true, "𠂊": true, "𠂤": true,
	"㐬": true, "㐭": true, "㐱": true, "㑳": true, "㒭": true,
	"㓦": true, "㕭": true, "㖈": true, "㗂": true, "㘄": true,
}

func IsCommonChar(char string) bool {
	if _, ok := RareChars[char]; ok {
		return false
	}
	if data, ok := HanziData[char]; ok {
		return data.Strokes <= 25
	}
	return true
}

type Hanzi struct {
	Char     string
	Pinyin   string
	Strokes  int
	Radical  string
	Meaning  string
	Wuxing   string
	Gender   string

	// 以下字段来自 chars.json 精选起名用字库
	Tone           int      // 声调
	GenderTags     []string // 性别标签（更细粒度，如 ["male","female"]）
	StyleTags      []string // 风格标签（如 "古典"、"现代"、"幽深"）
	UsageLevel     int      // 常用等级（1-5）
	PositiveScore  int      // 寓意评分（0-100）
	PhoneticScore  int      // 音韵评分（0-100）
	ModernScore    int      // 现代感评分（0-100）
	ClassicalScore int      // 古典感评分（0-100）
	IsPolyphonic   bool     // 是否多音字
	IsRare         bool     // 是否生僻字
	IsNegative     bool     // 是否消极含义
	PairBlacklist  []string // 搭配黑名单
	CurationLevel  int      // 精选等级（1-5）
	NamePenalty    int      // 起名扣分

	// P3 — 人名频率（来自 Chinese-Names-Corpus 语料统计）
	NameFreqTier int // 人名频率档位（1-5，5=最高频，0=未收录）

	// NamingCategories 起名分类标签列表（0~N 个分类）
	// 由 classifier 自动标注 + 策展覆盖表修正
	NamingCategories []string
}

var HanziData = map[string]Hanzi{}

// ForEachHanzi 遍历 HanziData 中的每个汉字（只读、线程安全）
func ForEachHanzi(fn func(Hanzi)) {
	mu.RLock()
	defer mu.RUnlock()
	for _, h := range HanziData {
		fn(h)
	}
}

// --- 运行时 word.json 释义加载 ---

var (
	wordMeanings map[string]string
	wordMu       sync.Mutex
	wordLoaded   bool
)

// wordEntry word.json 条目结构
type wordEntry struct {
	Word        string `json:"word"`
	Explanation string `json:"explanation"`
}

// LoadWordData 从 data/word.json 加载词语释义，用于非精选字的含义回退
func LoadWordData(dir string) error {
	wordMu.Lock()
	defer wordMu.Unlock()

	if wordLoaded {
		return nil
	}

	path := filepath.Join(dir, "word.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取 word.json 失败: %w", err)
	}

	var entries []wordEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("解析 word.json 失败: %w", err)
	}

	wordMeanings = make(map[string]string, len(entries))
	for _, e := range entries {
		if e.Word == "" {
			continue
		}
		char := string([]rune(e.Word)[0])
		if _, exists := wordMeanings[char]; !exists && e.Explanation != "" {
			wordMeanings[char] = e.Explanation
		}
	}

	wordLoaded = true
	return nil
}

// ApplyWuxingOverrides 为 HanziData 中所有汉字标注五行
//
// 优先级：
//   1. CharacterWuxingOverride（字义法覆盖表）— 无条件应用
//   2. 已有五行值 — 保留（人工校正值）
//   3. 部首映射表计算 — 仅对空值兜底
//
// 必须在 HanziData 填充完毕（JSON 加载后）调用
func ApplyWuxingOverrides() {
	for char, data := range HanziData {
		// 1. 覆盖表优先（字义法）
		if wx, ok := CharacterWuxingOverride[char]; ok {
			if wx != data.Wuxing {
				data.Wuxing = wx
				HanziData[char] = data
			}
			continue
		}
		// 2. 已有五行 → 保留（已人工校正）
		if data.Wuxing != "" {
			continue
		}
		// 3. 部首映射兜底
		if wx := GetWuxingByRadical(data.Radical); wx != "" {
			data.Wuxing = wx
			HanziData[char] = data
		}
	}
}

// WordMeaning 返回汉字的词语级释义（来自 word.json），空字符串表示无数据
func WordMeaning(char string) string {
	wordMu.Lock()
	defer wordMu.Unlock()

	if wordMeanings == nil {
		return ""
	}
	return wordMeanings[char]
}
