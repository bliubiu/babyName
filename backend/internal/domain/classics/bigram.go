package classics

import (
	"sync"
)

// BigramEntry 二字共现信息（用于 WenHuaRater 加分）
type BigramEntry struct {
	Char1      string // 汉字1（排序后的较小者）
	Char2      string // 汉字2
	Frequency  int    // 共现频次（在多少句诗歌中同时出现）
	SourceDesc string // 出处描述（取频次最高的例句出处）
}

// bigramIndex 二字共现索引
// key: "char1+char2"（排序后拼接），value: BigramEntry
var (
	bigramIdx  map[string]*BigramEntry
	bigramMu   sync.RWMutex
	bigramOnce sync.Once
)

// SessionBigramCache per-session 二字共现评分缓存的全局别名
//
// 实际定义在 fate 包（避免 fate → classics → fate 循环依赖），
// 此处仅 re-export 类型供 classics 包内部使用。
// 调用方应优先使用 fate.SessionBigramCache。

// groupKey 二字共现的分组键（source + sentence）
type groupKey struct {
	source   string // work 名
	sentence string // 诗句
}

// buildBigramIndex 从 PoetrySources + 全量经典提取文本构建二字共现索引
//
// 策略：以诗句句子（Sentence）为单位，提取同句所有字符的两两组合。
// 例如 "采薇采薇，薇亦作止" 中包含的字符有 {采,薇}，
// 则组合 (采,薇) 频次 +1。
//
// 覆盖来源：
//   - PoetrySources（硬编码库：诗经、楚辞等）
//   - P0 经典：诗经原文、楚辞原文、古文观止
//   - P1 诗词：全唐诗宋词（shici.json）
//   - P2 儒家：论语、孟子、大学、中庸
//   - P3 蒙学：三字经、千字文、弟子规、幼学琼林、增广贤文、声律启蒙、朱子家训、千家诗、文字蒙求、百家姓
func buildBigramIndex() {
	bigramMu.Lock()
	defer bigramMu.Unlock()

	if bigramIdx != nil {
		return
	}

	bigramIdx = make(map[string]*BigramEntry)

	// 按 (source + sentence) 分组，找出同句字符
	groupChars := make(map[groupKey][]string)

	// 1. 处理 PoetrySources 硬编码库
	for sourceKey, source := range PoetrySources {
		for _, pc := range source.Chars {
			if pc.Sentence == "" {
				continue
			}
			key := groupKey{source: sourceKey, sentence: pc.Sentence}
			groupChars[key] = append(groupChars[key], pc.Char)
		}
	}

	// 2. 处理全量经典提取文本
	indexExtractedForBigram(groupChars, ShijingExtracted)
	indexExtractedForBigram(groupChars, ChuciExtracted)
	indexExtractedForBigram(groupChars, GuwenGuanzhiExtracted)
	ensureShiCiLoaded()
	indexExtractedForBigram(groupChars, ShiCiExtracted)

	indexExtractedForBigram(groupChars, LunyuExtracted)
	indexExtractedForBigram(groupChars, MengziExtracted)
	indexExtractedForBigram(groupChars, DaxueExtracted)
	indexExtractedForBigram(groupChars, ZhongyongExtracted)
	indexExtractedForBigram(groupChars, SanzijingExtracted)
	indexExtractedForBigram(groupChars, QianziwenExtracted)
	indexExtractedForBigram(groupChars, DiziguiExtracted)
	indexExtractedForBigram(groupChars, YouxueqionglinExtracted)
	indexExtractedForBigram(groupChars, ZengguangxianwenExtracted)
	indexExtractedForBigram(groupChars, ShenglvqimengExtracted)
	indexExtractedForBigram(groupChars, ZhuzijiaxunExtracted)
	indexExtractedForBigram(groupChars, QianjiashiExtracted)
	indexExtractedForBigram(groupChars, WenzimengqiuExtracted)
	indexExtractedForBigram(groupChars, BaijiaxingExtracted)

	// 对每组内部去重并生成所有两两组合
	for gk, chars := range groupChars {
		// 组内字符去重，同时过滤停用字
		unique := removeDupAndFilterStopWords(chars)
		if len(unique) < 2 {
			continue
		}

		// 出处描述
		workName := gk.source
		sampleSentence := gk.sentence

		// 生成所有两两组合
		for i := 0; i < len(unique); i++ {
			for j := i + 1; j < len(unique); j++ {
				a, b := unique[i], unique[j]
				if a == b {
					continue
				}
				// 排序确保 key 唯一
				if a > b {
					a, b = b, a
				}
				key := a + "+" + b

				entry, exists := bigramIdx[key]
				if !exists {
					bigramIdx[key] = &BigramEntry{
						Char1:      a,
						Char2:      b,
						Frequency:  1,
						SourceDesc: workName + "·" + sampleSentence,
					}
				} else {
					entry.Frequency++
				}
			}
		}
	}
}

// indexExtractedForBigram 将提取文本追加到 bigram 的 groupChars 中
func indexExtractedForBigram(groupChars map[groupKey][]string, extracted []PoetryChar) {
	for _, pc := range extracted {
		if pc.Sentence == "" {
			continue
		}
		source := pc.Work
		if source == "" {
			source = pc.Chapter
		}
		key := groupKey{source: source, sentence: pc.Sentence}
		groupChars[key] = append(groupChars[key], pc.Char)
	}
}

// removeDupAndFilterStopWords 去重并过滤停用字
func removeDupAndFilterStopWords(s []string) []string {
	seen := make(map[string]bool, len(s))
	r := make([]string, 0, len(s))
	for _, v := range s {
		if seen[v] {
			continue
		}
		seen[v] = true
		if isStopWord(v) {
			continue
		}
		r = append(r, v)
	}
	return r
}

// GetBigramScore 查询两个字的共现评分
//
// 返回值：
//   - score: 共现评分（0~10），基于频次归一化
//   - sourceDesc: 出处描述（如有）
//   - found: 是否在共现库中
//
// 归一化逻辑：
//   - freq >= 5 → 10 分
//   - freq >= 3 → 7 分
//   - freq >= 2 → 4 分
//   - freq == 1 → 2 分
func GetBigramScore(char1, char2 string) (score int, sourceDesc string, found bool) {
	if char1 == "" || char2 == "" || char1 == char2 {
		return 0, "", false
	}

	bigramOnce.Do(buildBigramIndex)

	// 排序确保 key 匹配
	a, b := char1, char2
	if a > b {
		a, b = b, a
	}

	bigramMu.RLock()
	entry, ok := bigramIdx[a+"+"+b]
	bigramMu.RUnlock()

	if !ok {
		return 0, "", false
	}

	// 归一化评分
	switch {
	case entry.Frequency >= 5:
		score = 10
	case entry.Frequency >= 3:
		score = 7
	case entry.Frequency >= 2:
		score = 4
	default:
		score = 2
	}

	return score, entry.SourceDesc, true
}
