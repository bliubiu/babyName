package hanzi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// ────────────────────────── 数据结构 ──────────────────────────

// CharFrequency 单字在人名中的出现频率
type CharFrequency struct {
	Char  string `json:"char"`
	Count int    `json:"count"`
	Rank  int    `json:"rank"`
	Tier  int    `json:"tier"` // 频率档位 1-5（5=最高频）
}

// BigramFrequency 双字组合在人名中的出现频率
type BigramFrequency struct {
	Char1 string `json:"char1"`
	Char2 string `json:"char2"`
	Count int    `json:"count"`
	Rank  int    `json:"rank"`
	Tier  int    `json:"tier"`
}

// frequencyMeta JSON 文件中的元信息
type frequencyMeta struct {
	Source      string `json:"source"`
	TotalNames  int    `json:"total_names"`
	TotalChars  int    `json:"total_chars"`
	UniqueChars int    `json:"unique_chars"`
	BuiltAt     string `json:"built_at"`
}

// charFrequencyOutput 单字频率 JSON 顶层结构
type charFrequencyOutput struct {
	Meta      frequencyMeta    `json:"meta"`
	Frequency []*CharFrequency `json:"frequency"`
}

// bigramFrequencyOutput 双字频率 JSON 顶层结构
type bigramFrequencyOutput struct {
	Meta      frequencyMeta       `json:"meta"`
	Frequency []*BigramFrequency `json:"frequency"`
}

// ────────────────────────── 频率数据库 ──────────────────────────

// FrequencyDB 人名频率数据库
//
// 数据来源：Chinese-Names-Corpus（wainshine），120万中文人名语料。
// 用于两个场景：
//   1. 评分维度：FrequencyRater 基于单字频率 + 双字组合频率打分
//   2. 候选池排序：高频字优先进入组合，提高生成效率
type FrequencyDB struct {
	charMap    map[string]*CharFrequency   // 单字 → 频率
	bigramMap  map[string]*BigramFrequency  // "AB" → 频率
	charRank   []*CharFrequency            // 按频率降序（用于 TopN 查询）
	bigramRank []*BigramFrequency          // 按频率降序
	totalNames int                         // 语料库总人数（meta.total_names）
	loaded     bool
}

var (
	freqDB     FrequencyDB
	freqDBOnce sync.Once
	freqDBMu   sync.RWMutex
)

// LoadFrequencyDB 从 data 目录加载人名频率数据
//
// 加载两个文件：
//   - name_frequency.json：单字人名频率
//   - name_bigram_frequency.json：双字组合频率
//
// 只加载 Tier ≥ 2 的条目（占总数据量的 ~50%，过滤掉极低频噪声）
func LoadFrequencyDB(dataDir string) error {
	freqDBMu.Lock()
	defer freqDBMu.Unlock()

	if freqDB.loaded {
		return nil
	}

	// 加载单字频率
	charPath := filepath.Join(dataDir, "name_frequency.json")
	charData, err := os.ReadFile(charPath)
	if err != nil {
		return err // 文件不存在时返回错误，调用方可决定是否忽略
	}

	var charOutput charFrequencyOutput
	if err := json.Unmarshal(charData, &charOutput); err != nil {
		return err
	}

	charMap := make(map[string]*CharFrequency, len(charOutput.Frequency))
	var charRank []*CharFrequency
	for _, f := range charOutput.Frequency {
		if f.Tier < 2 {
			continue // 跳过 Tier 1 极低频字
		}
		cp := *f
		charMap[f.Char] = &cp
		charRank = append(charRank, &cp)
	}

	// 加载双字频率（仅 Tier ≥ 2，减半文件大小）
	bigramPath := filepath.Join(dataDir, "name_bigram_frequency.json")
	bigramData, err := os.ReadFile(bigramPath)
	if err != nil {
		return err
	}

	var bigramOutput bigramFrequencyOutput
	if err := json.Unmarshal(bigramData, &bigramOutput); err != nil {
		return err
	}

	bigramMap := make(map[string]*BigramFrequency, len(bigramOutput.Frequency)/2)
	var bigramRank []*BigramFrequency
	for _, f := range bigramOutput.Frequency {
		if f.Tier < 2 {
			continue
		}
		cp := *f
		key := f.Char1 + f.Char2
		bigramMap[key] = &cp
		bigramRank = append(bigramRank, &cp)
	}

	freqDB = FrequencyDB{
		charMap:    charMap,
		bigramMap:  bigramMap,
		charRank:   charRank,
		bigramRank: bigramRank,
		totalNames: charOutput.Meta.TotalNames,
		loaded:     true,
	}

	return nil
}

// GetCharFrequency 获取单字人名频率，未收录时返回 nil
func GetCharFrequency(char string) *CharFrequency {
	freqDBMu.RLock()
	defer freqDBMu.RUnlock()
	return freqDB.charMap[char]
}

// GetBigramFrequency 获取双字组合频率，未收录时返回 nil
func GetBigramFrequency(char1, char2 string) *BigramFrequency {
	key := char1 + char2
	freqDBMu.RLock()
	defer freqDBMu.RUnlock()
	return freqDB.bigramMap[key]
}

// GetCharTier 获取字的人名频率档位（1-5），未收录时返回 0
func GetCharTier(char string) int {
	freqDBMu.RLock()
	defer freqDBMu.RUnlock()
	if f, ok := freqDB.charMap[char]; ok {
		return f.Tier
	}
	return 0
}

// GetBigramTier 获取双字组合频率档位（1-5），未收录时返回 0
func GetBigramTier(char1, char2 string) int {
	key := char1 + char2
	freqDBMu.RLock()
	defer freqDBMu.RUnlock()
	if f, ok := freqDB.bigramMap[key]; ok {
		return f.Tier
	}
	return 0
}

// IsHighFrequency 判断是否为人名高频字（Tier ≥ threshold）
func IsHighFrequency(char string, threshold int) bool {
	return GetCharTier(char) >= threshold
}

// FrequencyDBLoaded 频率数据库是否已加载
func FrequencyDBLoaded() bool {
	freqDBMu.RLock()
	defer freqDBMu.RUnlock()
	return freqDB.loaded
}

// FrequencyDBStats 返回频率数据库统计信息
func FrequencyDBStats() (charCount, bigramCount int) {
	freqDBMu.RLock()
	defer freqDBMu.RUnlock()
	return len(freqDB.charMap), len(freqDB.bigramMap)
}

// GetTotalNames 返回频率语料库总人数（meta.total_names），未加载时返回 0
func GetTotalNames() int {
	freqDBMu.RLock()
	defer freqDBMu.RUnlock()
	return freqDB.totalNames
}

// GetTopChars 返回频率最高的 N 个字（Tier ≥ minTier）
func GetTopChars(n int, minTier int) []*CharFrequency {
	freqDBMu.RLock()
	defer freqDBMu.RUnlock()

	var result []*CharFrequency
	for _, f := range freqDB.charRank {
		if f.Tier < minTier {
			break
		}
		result = append(result, f)
		if len(result) >= n {
			break
		}
	}
	return result
}

// CharRanking 返回按频率降序的单字排行切片（只读，调用方不得修改元素）
func CharRanking() []*CharFrequency {
	freqDBMu.RLock()
	defer freqDBMu.RUnlock()
	return freqDB.charRank
}

// BigramRanking 返回按频率降序的双字组合排行切片（只读，调用方不得修改元素）
func BigramRanking() []*BigramFrequency {
	freqDBMu.RLock()
	defer freqDBMu.RUnlock()
	return freqDB.bigramRank
}
