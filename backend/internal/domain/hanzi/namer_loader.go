package hanzi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// NamerChar 对应 namer.json 中每一条汉字数据
type NamerChar struct {
	Char    string `json:"char"`
	Pinyin  string `json:"pinyin"`
	Strokes int    `json:"strokes"`
	// KangxiStrokes 康熙字典笔画（来自 kangxi-strokecount.csv，独立于 namer.json 加载），
	// 不持久到 namer.json，运行时由 LoadKangxiStrokesFromCSV 注入。
	KangxiStrokes int    `json:"kangxiStrokes,omitempty"`
	Radical       string `json:"radical"`
	Meaning       string `json:"meaning,omitempty"`
	Wuxing        string `json:"wuxing"`
	Level         int    `json:"level"` // 1=一级常用字, 2=二级通用字, 3=三级专用字

	Gender           string   `json:"gender,omitempty"`
	NamingCategories []string `json:"namingCategories,omitempty"`

	// P0 — 安全过滤
	IsNegative bool `json:"isNegative,omitempty"` // 是否消极含义
	IsRare     bool `json:"isRare,omitempty"`     // 是否生僻字

	// P1 — 评分 / 音韵
	NamePenalty int `json:"namePenalty,omitempty"` // 起名扣分（0-100）
	Tone        int `json:"tone,omitempty"`        // 声调（1-4）

	// P2 — 筛选增强
	StyleTags     []string `json:"styleTags,omitempty"`     // 风格标签
	UsageLevel    int      `json:"usageLevel,omitempty"`    // 常用等级（1-5）
	PositiveScore int      `json:"positiveScore,omitempty"` // 寓意评分（0-100）

	// P3 — 人名频率（来自 Chinese-Names-Corpus 语料统计）
	NameFreqTier int `json:"nameFreqTier,omitempty"` // 人名频率档位（1-5，5=最高频，0=未收录）
}

// StandardCharGroup 标准起名用字（按偏旁分组）
// 与老版 standard_chars.json 的数据结构一致，现已合并进
// namer.json 的顶层 charGroups 字段，作为统一单文件真源的组成部分。
type StandardCharGroup struct {
	Radical string   `json:"radical"`
	Name    string   `json:"name"`
	Meaning string   `json:"meaning"`
	Chars   []string `json:"chars"`
}

// NamerData 顶层容器，对应 namer.json 根结构
type NamerData struct {
	Version     string              `json:"version"`
	Generated   string              `json:"generated"`
	Description string              `json:"description"`
	TotalChars  int                 `json:"totalChars"`
	Chars       []NamerChar         `json:"chars"`
	CharGroups  []StandardCharGroup `json:"charGroups,omitempty"` // 精选起名用字按偏旁分组
}

var (
	// NamerCharMap 汉字 → namer 数据的快速查找表
	NamerCharMap map[string]NamerChar

	// NamerGroups 精选起名用字的分偏旁分组（来自 namer.json 顶层 charGroups）
	NamerGroups []StandardCharGroup

	namerMu sync.RWMutex
)

// LoadNamerFromJSON 从 namer.json 加载《通用规范汉字表》8105 字数据
//
// 加载后同时：
//  1. 在 NamerCharMap 中建立 char→NamerChar 映射
//  2. 将 namer 数据合并到 HanziData（HanziData[char] 已存在时不覆盖）
//
// 这样现有所有依赖 HanziData 的代码无需改动即可获得 8105 字覆盖。
func LoadNamerFromJSON(dataDir string) error {
	path := filepath.Join(dataDir, "namer.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var nd NamerData
	if err := json.Unmarshal(data, &nd); err != nil {
		return err
	}

	namerMu.Lock()
	defer namerMu.Unlock()

	m := make(map[string]NamerChar, len(nd.Chars))
	for _, nc := range nd.Chars {
		m[nc.Char] = nc
	}
	NamerCharMap = m

	// 承载精选起名用字分组（来自顶层 charGroups，取代独立 standard_chars.json）
	NamerGroups = nd.CharGroups

	// 同步到 HanziData — 以 namer 数据为准
	mu.Lock()
	for _, nc := range nd.Chars {
		// 全量覆盖：namer.json 是《通用规范汉字表》8105 字的权威数据源
		HanziData[nc.Char] = Hanzi{
			Char:    nc.Char,
			Pinyin:  nc.Pinyin,
			Strokes: nc.Strokes,
			Radical: nc.Radical,
			Meaning: nc.Meaning,
			Wuxing:  nc.Wuxing,
			Gender:  nc.Gender,

			Tone:          nc.Tone,
			StyleTags:     nc.StyleTags,
			UsageLevel:    nc.UsageLevel,
			PositiveScore: nc.PositiveScore,
			IsRare:        nc.IsRare,
			IsNegative:    nc.IsNegative,
			NamePenalty:   nc.NamePenalty,

			// 来自 namer 的起名分类
			NamingCategories: nc.NamingCategories,
		}
	}
	mu.Unlock()

	// 五行覆盖表和部首兜底（必须在 HanziData 填充完毕后运行）
	ApplyWuxingOverrides()

	// 康熙笔画合并：若 kangxi-strokecount.csv 已加载，把康熙笔画注入
	// HanziData[char].KangxiStrokes 与 NamerCharMap[char].KangxiStrokes。
	// 未加载时 KangxiStrokes 保持 0（兜底用简体笔画 Strokes）。
	mergeKangxiStrokes()

	// 加载人名频率数据并合并到 HanziData（仅对 namer.json 8105 字生效）
	if err := LoadFrequencyDB(dataDir); err == nil {
		mu.Lock()
		for char, data := range HanziData {
			if tier := GetCharTier(char); tier > 0 {
				data.NameFreqTier = tier
				HanziData[char] = data
			}
		}
		mu.Unlock()
	}
	// 频率文件不存在时静默跳过（非强制依赖）

	return nil
}

// ReloadNamerFromJSON 热更新 namer 数据
func ReloadNamerFromJSON(dataDir string) error {
	// 备份
	namerMu.RLock()
	oldMap := NamerCharMap
	namerMu.RUnlock()

	if err := LoadNamerFromJSON(dataDir); err != nil {
		namerMu.Lock()
		NamerCharMap = oldMap
		namerMu.Unlock()
		return err
	}
	return nil
}

// GetNamerLevel 获取汉字等级（1/2/3），不含时返回 0
func GetNamerLevel(char string) int {
	namerMu.RLock()
	defer namerMu.RUnlock()
	nc, ok := NamerCharMap[char]
	if !ok {
		return 0
	}
	return nc.Level
}

// GetNamerGroups 获取精选起名用字的分偏旁分组（namer.json 顶层 charGroups）
func GetNamerGroups() []StandardCharGroup {
	namerMu.RLock()
	defer namerMu.RUnlock()
	return NamerGroups
}

// GetCharRadicalFromNamer 从 namer 数据获取汉字的偏旁（取代对 standard_chars.json 的磁盘读取）
func GetCharRadicalFromNamer(char string) string {
	namerMu.RLock()
	defer namerMu.RUnlock()
	nc, ok := NamerCharMap[char]
	if !ok {
		return ""
	}
	return nc.Radical
}

// mergeKangxiStrokes 把康熙笔画注入 HanziData 与 NamerCharMap
//
// 设计原则：
//   - 仅当 kangxi-strokecount.csv 已加载（IsKangxiStrokesLoaded()）且收录该字时注入。
//   - HanziData[char].KangxiStrokes 不为 0 时覆盖现有值（0 为未收录）；
//   - NamerCharMap[char].KangxiStrokes 同样覆盖（不持久到 namer.json）。
//
// 调用方：LoadNamerFromJSON 末尾（在频率数据合并前，确保 NamerCharMap 已建立）。
// **锁约定**：调用方已持有 namerMu 写锁与 mu 写锁，本函数不得再 Lock（否则死锁）。
func mergeKangxiStrokes() {
	if !IsKangxiStrokesLoaded() {
		return
	}

	kangxiMu.RLock()
	count := len(kangxiStrokesMap)
	if count == 0 {
		kangxiMu.RUnlock()
		return
	}
	// 在锁内复制所需数据（避免函数返回后 GetKangxiStrokes 跨锁调用）
	strokesSnapshot := make(map[string]int, count)
	for k, v := range kangxiStrokesMap {
		strokesSnapshot[k] = v
	}
	kangxiMu.RUnlock()

	for char, nc := range NamerCharMap {
		if ks, ok := strokesSnapshot[char]; ok && ks > 0 {
			nc.KangxiStrokes = ks
			NamerCharMap[char] = nc

			if h, ok := HanziData[char]; ok {
				h.KangxiStrokes = ks
				HanziData[char] = h
			}
		}
	}
}
