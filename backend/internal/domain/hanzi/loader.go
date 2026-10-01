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

	// 先在局部 map 里构建完整数据，再一次性并入全局（docs/29 B9）
	//
	// 原来在整个构建循环期间持有 mu 写锁，且循环内调用 ApplyWuxingOverrides
	// —— 后者会自行加锁，RWMutex 不可重入，加锁改造后这里必然自死锁。
	// 改为「无锁构建 → 短临界区合并 → 锁外做五行覆盖」：
	//   - 读侧看到的永远是完整的一代数据，不会看到写了一半的中间态；
	//   - 任何一次失败（读文件/反序列化/构建）都发生在改动全局之前，
	//     不再需要回滚 —— 原实现的回滚也不完整（只补回旧键，
	//     不删除本轮新增的键），失败后会残留半截数据。
	newData := make(map[string]Hanzi, len(jsonList))
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

		newData[jh.Char] = Hanzi{
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

	mu.Lock()
	for k, v := range newData {
		HanziData[k] = v
	}
	mu.Unlock()

	// 五行覆盖表和部首兜底（必须在 HanziData 填充完毕后运行，内部自行加锁）
	ApplyWuxingOverrides()

	return nil
}

// ReloadFromJSON 重新加载 JSON 数据（热更新）
//
// 旧实现先快照 HanziData、失败后再回滚，但回滚只补回旧键、不删除本轮
// 新增的键，失败后会残留半截数据；且快照与回滚全程无锁，与读侧并发即
// fatal error: concurrent map read and map write（docs/29 B9）。
// 现依赖 LoadFromJSON 的「构建成功才合并」保证原子性：所有可能失败的步骤
// 都在改动全局之前完成，无需回滚。
func ReloadFromJSON(dataDir string) error {
	return LoadFromJSON(dataDir)
}

var mu sync.RWMutex

// GetHanzi 线程安全获取汉字数据
func GetHanzi(char string) (Hanzi, bool) {
	mu.RLock()
	defer mu.RUnlock()
	h, ok := HanziData[char]
	return h, ok
}
