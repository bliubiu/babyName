package name

import (
	"name/internal/domain/classics"
	"name/internal/domain/hanzi"
)

// Name 名字结构
type Name struct {
	ID              int64     `json:"id"`
	Surname         string    `json:"surname"`
	Generation      string    `json:"generation"`
	GivenName       string    `json:"given_name"`
	FullName        string    `json:"full_name"`
	Pinyin          string    `json:"pinyin"`
	Meaning         string    `json:"meaning"`
	Wuxing          string    `json:"wuxing"`
	Nayin           string    `json:"nayin"`
	Strokes         int       `json:"strokes"`
	Gender          string    `json:"gender"`
	BaZiScore       int       `json:"bazi_score"`
	Huangli         string    `json:"huangli"`
	Xiang           string    `json:"xiang"`
	PoetrySource    string    `json:"poetry_source"`
	PoetryChapter   string    `json:"poetry_chapter"`
	PoetrySentence  string    `json:"poetry_sentence"`
	WuxingAnalysis  string    `json:"wuxing_analysis"`
	BaziScoreDetail string    `json:"bazi_score_detail"`
	Yinyun          string    `json:"yinyun"`
	Reasons         []string  `json:"reasons"`

	// 统一评分体系（多维度综合评分）
	TotalScore    float64 `json:"total_score"`    // 综合评分（0-100）
	WuxingScore   float64 `json:"wuxing_score"`   // 五行评分
	YinyunScore   float64 `json:"yinyun_score"`   // 音韵评分
	MeaningScore  float64 `json:"meaning_score"`  // 字义评分
	SancaiScore   float64 `json:"sancai_score"`   // 天地人三才评分
	ZodiacScore   float64 `json:"zodiac_score"`   // 生肖评分
	NayinScore    float64 `json:"nayin_score"`    // 纳音评分
	NoveltyScore  float64 `json:"novelty_score"`  // 新颖度评分
	BigramScore   float64 `json:"bigram_score"`   // 诗词共现评分
	FrequencyScore float64 `json:"frequency_score"` // 人名频率评分（来自 Chinese-Names-Corpus 语料统计）
	SancaiAnalysis string  `json:"sancai_analysis"` // 三才分析详情
	Hexagram       string  `json:"hexagram"`          // 卦象名称
	HexagramMeaning string `json:"hexagram_meaning"` // 卦象解读
}

// 缓存最大条目数（3500 常用字 + 1605 三级字 ≈ 5105，留余量至 10000）
const maxCacheEntries = 10000

// NameChar 汉字字符
type NameChar struct {
	Char      string
	Pinyin    string
	Meaning   string
	Wuxing    string
	Strokes   int
	Gender    string
}

// GenerateOptions 名字生成选项
type GenerateOptions struct {
	Surname            string
	Generation         string
	Gender             string
	Xiyongshen         []string
	Count              int
	NameLength         int
	ExcludeRare        bool
	WuxingMatch        []string
	SourceClassic      string
	MinStrokes         int
	MaxStrokes         int
	IncludePoetry      bool
	IncludeClassic     bool
	MeaningKeywords    []string
	PinyinInitial      string
	GenerationPosition string
	NameType           string
	Zodiac             string // 生肖，用于增强分析
	Nayin              string // 纳音，用于纳音五行评分

	// 避讳长辈姓名列表（父系/母系直系长辈，建议往上两代）
	// 生成名字时排除同形字与同音字
	AvoidElderNames []string

	// 大名需大命：日主强弱（"身旺"/"身中"/"身弱"/"身衰"）
	// 普通格局排除敏感字，极旺格局（身旺）方可使用
	DayMasterStrength string
}

// CommonMaleNames 常用男性名字
var CommonMaleNames = []string{
	"伟", "强", "磊", "军", "杰", "涛", "明", "超", "勇", "鹏",
	"华", "刚", "平", "辉", "波", "峰", "飞", "龙", "浩", "宇",
	"晨", "逸", "睿", "哲", "渊", "博", "昊", "然", "轩", "俊",
	"豪", "毅", "文", "武", "祥", "瑞", "凯", "成", "盛", "雄",
	"鑫", "霖", "宇", "辰", "哲", "思", "远", "航", "洋", "庆",
}

// CommonFemaleNames 常用女性名字
var CommonFemaleNames = []string{
	"芳", "兰", "梅", "琴", "丽", "秀", "英", "敏", "静", "丽",
	"燕", "霞", "红", "玉", "珍", "娟", "莉", "萍", "颖", "娜",
	"欣", "怡", "菲", "雅", "芸", "萱", "雨", "诗", "梦", "瑶",
	"琳", "珂", "静", "雅", "雯", "倩", "雪", "梅", "兰", "竹",
	"婷", "霜", "璐", "瑶", "翠", "碧", "银", "金", "玉", "珠",
}

// GetMaleNames 获取男性用字表
func (eng *EnhancedNameGenerator) GetMaleNames() []string {
	eng.charMutex.RLock()
	defer eng.charMutex.RUnlock()
	if len(eng.MaleNames) > 0 {
		return eng.MaleNames
	}
	return CommonMaleNames
}

// GetFemaleNames 获取女性用字表
func (eng *EnhancedNameGenerator) GetFemaleNames() []string {
	eng.charMutex.RLock()
	defer eng.charMutex.RUnlock()
	if len(eng.FemaleNames) > 0 {
		return eng.FemaleNames
	}
	return CommonFemaleNames
}

// GetCharInfo 获取汉字信息
func (eng *EnhancedNameGenerator) GetCharInfo(char string) (pinyin string, meaning string, wuxing string, strokes int) {
	// 先用读锁检查缓存
	eng.charMutex.RLock()
	if info, ok := eng.charInfoCache[char]; ok {
		eng.charMutex.RUnlock()
		return info.pinyin, info.meaning, info.wuxing, info.strokes
	}
	eng.charMutex.RUnlock()

	if h, ok := hanzi.HanziData[char]; ok {
		pinyin = h.Pinyin
		meaning = h.Meaning
		wuxing = h.Wuxing
		strokes = h.Strokes
	} else {
		pinyin = GetPinyin(char)
		meaning = hanzi.WordMeaning(char)
		if meaning == "" {
			meaning = "美好寓意"
		}
		wuxing = "未知"
		strokes = 10
	}

	// 加写锁缓存结果（double-check 避免重复写）
	eng.charMutex.Lock()
	if _, ok := eng.charInfoCache[char]; !ok {
		evictIfNeeded(eng.charInfoCache, maxCacheEntries)
		eng.charInfoCache[char] = struct {
			pinyin  string
			meaning string
			wuxing  string
			strokes int
		}{pinyin, meaning, wuxing, strokes}
	}
	eng.charMutex.Unlock()

	return pinyin, meaning, wuxing, strokes
}

// GetCharExtra 获取汉字的部首和字形结构信息（带缓存）
func (eng *EnhancedNameGenerator) GetCharExtra(char string) (radical string, structure string) {
	// 先用读锁检查缓存
	eng.charMutex.RLock()
	if info, ok := eng.charExtraCache[char]; ok {
		eng.charMutex.RUnlock()
		return info.radical, info.structure
	}
	eng.charMutex.RUnlock()

	if h, ok := hanzi.HanziData[char]; ok {
		radical = h.Radical
	}

	// 加写锁缓存结果（double-check 避免重复写）
	eng.charMutex.Lock()
	if _, ok := eng.charExtraCache[char]; !ok {
		evictIfNeeded(eng.charExtraCache, maxCacheEntries)
		eng.charExtraCache[char] = struct {
			radical   string
			structure string
		}{radical, structure}
	}
	eng.charMutex.Unlock()

	return radical, structure
}

// GetClassicNames 获取诗经楚辞名字（含从原文提取的起名用字）
func GetClassicNames(gender string) []string {
	var chars []string
	// 从 ShijingNames 获取（硬编码条目，当前为空）
	for _, n := range classics.ShijingNames {
		if n.Gender == "通用" || n.Gender == gender {
			chars = append(chars, n.Char)
		}
	}
	// 从 ChuciNames 获取（硬编码条目，当前为空）
	for _, n := range classics.ChuciNames {
		if n.Gender == "通用" || n.Gender == gender {
			chars = append(chars, n.Char)
		}
	}
	// 从诗经原文提取的 PoetryChar 数据
	for _, pc := range classics.ShijingExtracted {
		if pc.Gender == "通用" || pc.Gender == gender {
			chars = append(chars, pc.Char)
		}
	}
	// 从楚辞原文提取的 PoetryChar 数据
	for _, pc := range classics.ChuciExtracted {
		if pc.Gender == "通用" || pc.Gender == gender {
			chars = append(chars, pc.Char)
		}
	}
	return chars
}

// GetPoetryNames 获取诗词名字（支持诗经、楚辞、唐诗、宋词等）
func GetPoetryNames(gender string, source string) []string {
	var chars []string

	poetryChars := classics.GetPoetryCharList(source)

	for _, n := range poetryChars {
		if n.Gender == "通用" || n.Gender == gender {
			chars = append(chars, n.Char)
		}
	}

	if len(chars) == 0 {
		chars = GetClassicNames(gender)
	}

	return chars
}

var poetryCharMap map[string]struct {
	pinyin   string
	meaning  string
	source   string
	chapter  string
	sentence string
	wuxing   string
}

func init() {
	poetryCharMap = make(map[string]struct {
		pinyin   string
		meaning  string
		source   string
		chapter  string
		sentence string
		wuxing   string
	})

	// 加载PoetrySources中的字符
	for _, ps := range classics.PoetrySources {
		for _, n := range ps.Chars {
			poetryCharMap[n.Char] = struct {
				pinyin   string
				meaning  string
				source   string
				chapter  string
				sentence string
				wuxing   string
			}{
				pinyin:   n.Pinyin,
				meaning:  n.Meaning,
				source:   n.Work,
				chapter:  n.Chapter,
				sentence: n.Sentence,
				wuxing:   n.Wuxing,
			}
		}
	}

	// 加载ShijingNames中的字符
	for _, n := range classics.ShijingNames {
		if _, exists := poetryCharMap[n.Char]; !exists {
			poetryCharMap[n.Char] = struct {
				pinyin   string
				meaning  string
				source   string
				chapter  string
				sentence string
				wuxing   string
			}{
				pinyin:   n.Pinyin,
				meaning:  n.Meaning,
				source:   n.Source,
				chapter:  n.Chapter,
				sentence: "",
				wuxing:   n.Wuxing,
			}
		}
	}

	// 加载ChuciNames中的字符
	for _, n := range classics.ChuciNames {
		if _, exists := poetryCharMap[n.Char]; !exists {
			poetryCharMap[n.Char] = struct {
				pinyin   string
				meaning  string
				source   string
				chapter  string
				sentence string
				wuxing   string
			}{
				pinyin:   n.Pinyin,
				meaning:  n.Meaning,
				source:   n.Source,
				chapter:  n.Chapter,
				sentence: "",
				wuxing:   n.Wuxing,
			}
		}
	}
}

// GetPoetryCharInfo 获取诗词字的详细信息
// 优先使用增强的 PoemIndex（按分数排序），回退到旧索引
func (eng *EnhancedNameGenerator) GetPoetryCharInfo(char string) (pinyin, meaning, source, chapter, sentence, wuxing string) {
	// 先用读锁检查缓存
	eng.charMutex.RLock()
	if info, ok := eng.poetryInfoCache[char]; ok {
		eng.charMutex.RUnlock()
		return info.pinyin, info.meaning, info.source, info.chapter, info.sentence, info.wuxing
	}
	eng.charMutex.RUnlock()

	// 优先尝试增强的 PoemIndex（按朝代/作者/分类筛选，分数排序）
	if poemIdx := classics.GetGlobalPoemIndex(); poemIdx != nil {
		if entries := poemIdx.FindByChar(char); len(entries) > 0 {
			// 从增强索引中查找最优匹配
			bestEntry := findBestPoemEntry(char, entries)
			if bestEntry != nil {
				pinyin = GetPinyin(char)
				meaning = hanzi.HanziData[char].Meaning
				source = bestEntry.Source
				chapter = bestEntry.Title
				if len(bestEntry.Content) > 0 {
					sentence = findSentenceWithChar(bestEntry.Content, char)
				}
				wuxing = hanzi.HanziData[char].Wuxing

				// 加写锁缓存结果
				eng.charMutex.Lock()
				if _, ok := eng.poetryInfoCache[char]; !ok {
					evictIfNeeded(eng.poetryInfoCache, maxCacheEntries)
					eng.poetryInfoCache[char] = struct {
						pinyin   string
						meaning  string
						source   string
						chapter  string
						sentence string
						wuxing   string
					}{pinyin, meaning, source, chapter, sentence, wuxing}
				}
				eng.charMutex.Unlock()
				return
			}
		}
	}

	// 回退到旧索引
	if info, exists := poetryCharMap[char]; exists {
		pinyin = info.pinyin
		meaning = info.meaning
		source = info.source
		chapter = info.chapter
		sentence = info.sentence
		wuxing = info.wuxing
	} else if entries := classics.FindPoetryByChar(char); len(entries) > 0 {
		// 从运行时加载的经典提取数据中查找
		entry := entries[0]
		pinyin = entry.Pinyin
		meaning = entry.Meaning
		source = entry.Work
		chapter = entry.Chapter
		sentence = entry.Sentence
		wuxing = entry.Wuxing
	} else {
		pinyin = GetPinyin(char)
		meaning = "美好寓意"
		source = "诗词"
		chapter = ""
		sentence = ""
		wuxing = "未知"
	}
	
	// 加写锁缓存结果（double-check 避免重复写）
	eng.charMutex.Lock()
	if _, ok := eng.poetryInfoCache[char]; !ok {
		evictIfNeeded(eng.poetryInfoCache, maxCacheEntries)
		eng.poetryInfoCache[char] = struct {
			pinyin   string
			meaning  string
			source   string
			chapter  string
			sentence string
			wuxing   string
		}{pinyin, meaning, source, chapter, sentence, wuxing}
	}
	eng.charMutex.Unlock()
	
	return pinyin, meaning, source, chapter, sentence, wuxing
}

// findBestPoemEntry 从多个诗词条目中选择最佳匹配
// 优先级：诗经 > 楚辞 > 唐诗 > 宋词 > 其他
func findBestPoemEntry(char string, entries []*classics.PoemEntry) *classics.PoemEntry {
	if len(entries) == 0 {
		return nil
	}

	// 按来源优先级排序
	sourcePriority := map[string]int{
		"诗经": 1,
		"楚辞": 2,
		"唐诗": 3,
		"宋词": 4,
		"元曲": 5,
		"古文观止": 6,
	}

	var best *classics.PoemEntry
	bestPriority := 999

	for _, entry := range entries {
		priority, ok := sourcePriority[entry.Source]
		if !ok {
			priority = 10
		}
		if priority < bestPriority {
			bestPriority = priority
			best = entry
		}
	}

	return best
}

// findSentenceWithChar 从诗句列表中找到包含指定字的最短句子
func findSentenceWithChar(content []string, char string) string {
	for _, line := range content {
		if containsCharStr(line, char) {
			return line
		}
	}
	return ""
}

// containsCharStr 检查字符串是否包含指定字符
func containsCharStr(s, char string) bool {
	for _, r := range s {
		if string(r) == char {
			return true
		}
	}
	return false
}

// GetPinyin 获取拼音（简化版）
// 优先级：1. HanziData（JSON 数据源，覆盖 5000+ 汉字） 2. 硬编码常见字兜底
func GetPinyin(char string) string {
	// 1. 优先从 HanziData 查询（来自 hanzi.json，含完整拼音数据）
	if h, ok := hanzi.HanziData[char]; ok && h.Pinyin != "" {
		return h.Pinyin
	}

	// 2. 硬编码常见字兜底（HanziData 未覆盖的冷门字）
	pinyinMap := map[string]string{
		"伟": "wei", "强": "qiang", "磊": "lei", "军": "jun",
		"杰": "jie", "涛": "tao", "明": "ming", "超": "chao",
		"勇": "yong", "鹏": "peng", "华": "hua", "刚": "gang",
		"平": "ping", "辉": "hui", "波": "bo", "峰": "feng",
		"飞": "fei", "龙": "long", "浩": "hao", "宇": "yu",
		"晨": "chen", "逸": "yi", "睿": "rui", "哲": "zhe",
		"渊": "yuan", "博": "bo", "昊": "hao", "然": "ran",
		"轩": "xuan", "俊": "jun", "豪": "hao", "毅": "yi",
		"文": "wen", "武": "wu", "祥": "xiang", "瑞": "rui",
		"凯": "kai", "成": "cheng", "盛": "sheng", "雄": "xiong",
		"鑫": "xin", "霖": "lin", "思": "si", "远": "yuan",
		"航": "hang", "洋": "yang", "庆": "qing",
		"芳": "fang", "兰": "lan", "梅": "mei", "琴": "qin",
		"丽": "li", "秀": "xiu", "英": "ying", "敏": "min",
		"静": "jing", "燕": "yan", "霞": "xia", "红": "hong",
		"玉": "yu", "珍": "zhen", "娟": "juan", "莉": "li",
		"萍": "ping", "颖": "ying", "娜": "na", "欣": "xin",
		"怡": "yi", "菲": "fei", "雅": "ya", "芸": "yun",
		"萱": "xuan", "雨": "yu", "诗": "shi", "梦": "meng",
		"瑶": "yao", "琳": "lin", "珂": "ke", "雯": "wen",
		"倩": "qian", "雪": "xue", "婷": "ting", "霜": "shuang",
		"翠": "cui", "碧": "bi", "银": "yin", "金": "jin",
		"珠": "zhu", "宾": "bin",
	}

	if p, ok := pinyinMap[char]; ok {
		return p
	}
	return char
}

// sensitiveNameChars 敏感字集合（"大"名需"大"命，AGENTS.md 规则2）
// 普通格局需移除，仅极旺格局（身旺）方可使用
var sensitiveNameChars = map[string]bool{
	"龙": true, "凤": true, "乾": true, "坤": true,
	"圣": true, "贤": true, "天": true, "帝": true,
	"皇": true, "神": true, "仙": true, "君": true,
}

// isSensitiveNameChar 判断字符是否为敏感字
func isSensitiveNameChar(char string) bool {
	return sensitiveNameChars[char]
}

// allowSensitiveChar 根据日主强弱判断是否允许敏感字
// 仅"身旺"（极旺格局）允许使用敏感字
func allowSensitiveChar(dayMasterStrength string) bool {
	return dayMasterStrength == "身旺"
}

// buildElderAvoidanceSets 构建避讳长辈的同形字与同音拼音排除集
// 规则（AGENTS.md 规则1）：同形字=侵佔福分，同音字=气场冲撞（"压运"）
func buildElderAvoidanceSets(elderNames []string) (map[string]bool, map[string]bool) {
	elderCharSet := make(map[string]bool)
	elderPinyinSet := make(map[string]bool)
	for _, name := range elderNames {
		for _, r := range name {
			ch := string(r)
			elderCharSet[ch] = true
			if p := GetPinyin(ch); p != "" && p != ch {
				elderPinyinSet[p] = true
			}
		}
	}
	return elderCharSet, elderPinyinSet
}

// evictIfNeeded 缓存淘汰：当 map 达到上限时批量淘汰 ~25% 条目
// 批量淘汰减少后续淘汰频率（替代逐条随机淘汰）
func evictIfNeeded[V any](m map[string]V, limit int) {
	if len(m) >= limit {
		target := limit / 4 // 淘汰约 25% 条目
		for k := range m {
			delete(m, k)
			target--
			if target <= 0 {
				return
			}
		}
	}
}
