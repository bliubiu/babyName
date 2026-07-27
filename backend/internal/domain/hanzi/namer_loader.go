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
	Radical string `json:"radical"`
	Meaning string `json:"meaning,omitempty"`
	Wuxing  string `json:"wuxing"`
	Level   int    `json:"level"` // 1=一级常用字, 2=二级通用字, 3=三级专用字

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
}

// NamerData 顶层容器，对应 namer.json 根结构
type NamerData struct {
	Version     string     `json:"version"`
	Generated   string     `json:"generated"`
	Description string     `json:"description"`
	TotalChars  int        `json:"totalChars"`
	Chars       []NamerChar `json:"chars"`
}

var (
	// NamerCharMap 汉字 → namer 数据的快速查找表
	NamerCharMap map[string]NamerChar

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

			Tone:    nc.Tone,
			StyleTags:   nc.StyleTags,
			UsageLevel:  nc.UsageLevel,
			PositiveScore: nc.PositiveScore,
			IsRare:       nc.IsRare,
			IsNegative:   nc.IsNegative,
			NamePenalty:  nc.NamePenalty,

			// 来自 namer 的起名分类
			NamingCategories: nc.NamingCategories,
		}
	}
	mu.Unlock()

	// 五行覆盖表和部首兜底（必须在 HanziData 填充完毕后运行）
	ApplyWuxingOverrides()

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

// GetNamerChar 线程安全获取 namer 字符数据
func GetNamerChar(char string) (NamerChar, bool) {
	namerMu.RLock()
	defer namerMu.RUnlock()
	nc, ok := NamerCharMap[char]
	return nc, ok
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

// GetNamerStats 返回 namer 数据统计
func GetNamerStats() (total int, l1, l2, l3 int) {
	namerMu.RLock()
	defer namerMu.RUnlock()
	total = len(NamerCharMap)
	for _, nc := range NamerCharMap {
		switch nc.Level {
		case 1:
			l1++
		case 2:
			l2++
		case 3:
			l3++
		}
	}
	return
}
