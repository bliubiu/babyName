package classics

import "sync"

// poetryCharIndex 字符 → 诗词条目快速索引
// 构建自 PoetrySources 硬编码数据，避免每次搜索都遍历全集
var (
	poetryIdx     map[string][]PoetryChar // char → []PoetryChar
	poetryIdxMu   sync.RWMutex
	poetryIdxOnce sync.Once
)

// buildPoetryIndex 构建字符→诗词条目的索引
// 线程安全，只构建一次
// 覆盖全部 P0/P2/P3 来源：硬编码库 + 16 个经典文本全文提取
func buildPoetryIndex() {
	poetryIdxMu.Lock()
	defer poetryIdxMu.Unlock()

	if poetryIdx != nil {
		return
	}

	poetryIdx = make(map[string][]PoetryChar)

	// 索引硬编码的古文名句库
	for _, source := range PoetrySources {
		for _, pc := range source.Chars {
			key := pc.Char
			poetryIdx[key] = append(poetryIdx[key], pc)
		}
	}

	// 索引所有经典文本提取数据，按来源分组
	indexExtracted(copyExtracted(&ShijingExtracted))
	indexExtracted(copyExtracted(&ChuciExtracted))
	indexExtracted(copyExtracted(&GuwenGuanzhiExtracted))

	// shici.json 可能尚未加载完毕，惰性等待
	ensureShiCiLoaded()
	indexExtracted(copyExtracted(&ShiCiExtracted))

	// 元曲 / 乐府 / 辞赋（E2：补齐起名字集，使出典加分类查询能命中）
	indexExtracted(copyExtracted(&YuanQuExtracted))
	indexExtracted(copyExtracted(&YueFuExtracted))
	indexExtracted(copyExtracted(&CiFuExtracted))

	// P2/P3 经典
	indexExtracted(copyExtracted(&LunyuExtracted))
	indexExtracted(copyExtracted(&MengziExtracted))
	indexExtracted(copyExtracted(&DaxueExtracted))
	indexExtracted(copyExtracted(&ZhongyongExtracted))
	indexExtracted(copyExtracted(&SanzijingExtracted))
	indexExtracted(copyExtracted(&QianziwenExtracted))
	indexExtracted(copyExtracted(&DiziguiExtracted))
	indexExtracted(copyExtracted(&YouxueqionglinExtracted))
	indexExtracted(copyExtracted(&ZengguangxianwenExtracted))
	indexExtracted(copyExtracted(&ShenglvqimengExtracted))
	indexExtracted(copyExtracted(&ZhuzijiaxunExtracted))
	indexExtracted(copyExtracted(&QianjiashiExtracted))
	indexExtracted(copyExtracted(&WenzimengqiuExtracted))
	indexExtracted(copyExtracted(&BaijiaxingExtracted))
}

// indexExtracted 将提取结果集追加到全局索引
func indexExtracted(extracted []PoetryChar) {
	for _, pc := range extracted {
		key := pc.Char
		poetryIdx[key] = append(poetryIdx[key], pc)
	}
}

// BuildPoetryIndex 主动构建诗词索引（docs/46 P2 性能）。
//
// 必须在文化物料装载完成后调用，把首个 /generate 请求的索引构建尖峰前移到启动期，
// 避免首请求延迟突增。重复调用安全：内部 sync.Once 保证只构建一次；若此前已通过
// 查询路径惰性触发过，本调用为无操作。
//
// ★ 时序约束：必须在 loadCulturalData（data.Init / InitWithDB）之后调用，否则
// 各 Extracted 字集尚未装载，索引会以空语料固化（poetryIdxOnce 一次性，无法补救）。
func BuildPoetryIndex() {
	poetryIdxOnce.Do(buildPoetryIndex)
}

// FindPoetryByChar 查找某字的诗词出处
// 返回该字在所有诗词来源中的条目列表
func FindPoetryByChar(char string) []PoetryChar {
	poetryIdxOnce.Do(buildPoetryIndex)

	poetryIdxMu.RLock()
	defer poetryIdxMu.RUnlock()

	if entries, ok := poetryIdx[char]; ok {
		result := make([]PoetryChar, len(entries))
		copy(result, entries)
		return result
	}
	return nil
}

// FindPoetryByCharSemantic 查找某字及其近义字的诗词出处
// 先精确匹配，无结果时扩展查询近义字
// 返回 (精确匹配结果, 近义扩展匹配的字符, 近义扩展匹配的条目)
func FindPoetryByCharSemantic(char string) (exact []PoetryChar, semanticMatchedChar string, semantic []PoetryChar) {
	exact = FindPoetryByChar(char)
	if len(exact) > 0 {
		return exact, "", nil
	}

	// 精确匹配不到，扩展近义字
	for _, syn := range FindSemanticChars(char) {
		if syn == char {
			continue
		}
		entries := FindPoetryByChar(syn)
		if len(entries) > 0 {
			return nil, syn, entries
		}
	}
	return nil, "", nil
}

// GetSemanticBigramMatch 通过近义字查找单名的诗词共现信息
// 例如名字含"欣"，近义字"喜"在诗句中与"乐"共现，则返回匹配信息
// 返回: (近义字, 搭配字, 出处描述, 是否找到)
func GetSemanticBigramMatch(char string) (string, string, string, bool) {
	_, synChar, entries := FindPoetryByCharSemantic(char)
	if synChar == "" || len(entries) == 0 {
		return "", "", "", false
	}
	entry := entries[0]
	if entry.Sentence == "" {
		return "", "", "", false
	}
	// 从诗句中提取搭配字
	coChars := extractUniqueChars(entry.Sentence)
	for _, c := range coChars {
		if c == synChar || isStopWord(c) {
			continue
		}
		if _, src, found := GetBigramScore(synChar, c); found {
			return synChar, c, src, true
		}
	}
	return "", "", "", false
}

// FindPoetryByChars 查找一组字的诗词出处
// 返回每个字对应的第一条匹配（NameCandidate 需要用）
// 优先返回完整的诗句而非仅单字匹配
//
// 返回:
//   - found: 是否至少有一个字找到诗词出处
//   - sourceDesc: 诗句描述，格式 "《诗经·关雎》：关关雎鸠，在河之洲"
//   - matchedChars: 匹配到的字符列表
func FindPoetryByChars(chars ...string) (found bool, sourceDesc string, matchedChars []string) {
	var (
		bestSentence string
		bestSource   string
		matchCount   int
	)

	for _, ch := range chars {
		if ch == "" {
			continue
		}
		entries := FindPoetryByChar(ch)
		if len(entries) == 0 {
			continue
		}

		matchedChars = append(matchedChars, ch)
		matchCount++

		// 取第一条匹配的条目
		entry := entries[0]
		if bestSource == "" {
			bestSource = entry.Work
		}
		if bestSentence == "" {
			if entry.Sentence != "" {
				bestSentence = entry.Sentence
			} else {
				bestSentence = entry.Work + "·" + entry.Chapter
			}
		}
	}

	if matchCount == 0 {
		return false, "", nil
	}

	// 构建来源描述
	if bestSentence != "" {
		sourceDesc = "「" + bestSentence + "」"
	} else {
		sourceDesc = bestSource
	}

	return true, sourceDesc, matchedChars
}

// HasPoetryChar 检查某个字是否有诗词出处
func HasPoetryChar(char string) bool {
	entries := FindPoetryByChar(char)
	return len(entries) > 0
}

// CountPoetryEntry 统计诗词库中收录的总条目数
func CountPoetryEntry() int {
	poetryIdxOnce.Do(buildPoetryIndex)

	poetryIdxMu.RLock()
	defer poetryIdxMu.RUnlock()

	total := 0
	for _, entries := range poetryIdx {
		total += len(entries)
	}
	return total
}
