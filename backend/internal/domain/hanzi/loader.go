package hanzi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// JSONHanzi JSON 格式的汉字数据
type JSONHanzi struct {
	Char    string `json:"char"`
	Pinyin  string `json:"pinyin"`
	Strokes int    `json:"strokes"`
	Radical string `json:"radical"`
	Meaning string `json:"meaning"`
	Wuxing  string `json:"wuxing"`
	Gender  string `json:"gender"`

	// 精选起名用字扩展字段
	Tone           int      `json:"tone,omitempty"`
	GenderTags     []string `json:"genderTags,omitempty"`
	StyleTags      []string `json:"styleTags,omitempty"`
	UsageLevel     int      `json:"usageLevel,omitempty"`
	PositiveScore  int      `json:"positiveScore,omitempty"`
	PhoneticScore  int      `json:"phoneticScore,omitempty"`
	ModernScore    int      `json:"modernScore,omitempty"`
	ClassicalScore int      `json:"classicalScore,omitempty"`
	IsPolyphonic   bool     `json:"isPolyphonic,omitempty"`
	IsRare         bool     `json:"isRare,omitempty"`
	IsNegative     bool     `json:"isNegative,omitempty"`
	PairBlacklist  []string `json:"pairBlacklist,omitempty"`
	CurationLevel  int      `json:"curationLevel,omitempty"`
	NamePenalty    int      `json:"namePenalty,omitempty"`

	// NamingCategories 起名分类标签（如 ["品德","文雅"]）
	// 可来源于 JSON 数据自带的标注，或由分类器自动补充
	NamingCategories []string `json:"namingCategories,omitempty"`
}

// LoadFromJSON 从 JSON 文件加载汉字数据（历史实现，读 data/raw/hanzi.json）
//
// 注意：该函数当前无运行时调用方——运行时统一由 LoadNamerFromJSON 从
// namer.json 加载。hanzi.json 已物理隔离至 data/raw/ 生成原料目录。
// 若需读取，请传入 raw 目录，例如 filepath.Join(dataDir, "raw")。
func LoadFromJSON(dataDir string) error {
	path := filepath.Join(dataDir, "hanzi.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var jsonList []JSONHanzi
	if err := json.Unmarshal(data, &jsonList); err != nil {
		return err
	}

	mu.Lock()
	defer mu.Unlock()

	for _, jh := range jsonList {
		// 五行判定优先级：覆盖表(字义法) > 部首映射(字形法) > JSON原始值(兜底)
		wuxing := jh.Wuxing
		if overrideWx, ok := CharacterWuxingOverride[jh.Char]; ok {
			wuxing = overrideWx
		} else if radWx := GetWuxingByRadical(jh.Radical); radWx != "" {
			wuxing = radWx
		}

		// 起名分类：优先使用 JSON 自带的标注，否则自动分类
		namingCats := jh.NamingCategories
		if len(namingCats) == 0 {
			namingCats = ClassifyNaming(jh.Char, jh.Radical, jh.Meaning, wuxing)
		}

		HanziData[jh.Char] = Hanzi{
			Char:    jh.Char,
			Pinyin:  jh.Pinyin,
			Strokes: jh.Strokes,
			Radical: jh.Radical,
			Meaning: jh.Meaning,
			Wuxing:  wuxing,
			Gender:  jh.Gender,

			Tone:           jh.Tone,
			GenderTags:     jh.GenderTags,
			StyleTags:      jh.StyleTags,
			UsageLevel:     jh.UsageLevel,
			PositiveScore:  jh.PositiveScore,
			PhoneticScore:  jh.PhoneticScore,
			ModernScore:    jh.ModernScore,
			ClassicalScore: jh.ClassicalScore,
			IsPolyphonic:   jh.IsPolyphonic,
			IsRare:         jh.IsRare,
			IsNegative:     jh.IsNegative,
			PairBlacklist:  jh.PairBlacklist,
			CurationLevel:  jh.CurationLevel,
			NamePenalty:    jh.NamePenalty,

			NamingCategories: namingCats,
		}
	}

	// 五行覆盖表和部首兜底（必须在 HanziData 填充完毕后运行）
	ApplyWuxingOverrides()

	return nil
}

// ReloadFromJSON 重新加载 JSON 数据（热更新）
func ReloadFromJSON(dataDir string) error {
	baseData := make(map[string]Hanzi)
	for k, v := range HanziData {
		baseData[k] = v
	}

	if err := LoadFromJSON(dataDir); err != nil {
		for k, v := range baseData {
			HanziData[k] = v
		}
		return err
	}

	return nil
}

var mu sync.RWMutex

// GetHanzi 线程安全获取汉字数据
func GetHanzi(char string) (Hanzi, bool) {
	mu.RLock()
	defer mu.RUnlock()
	h, ok := HanziData[char]
	return h, ok
}
