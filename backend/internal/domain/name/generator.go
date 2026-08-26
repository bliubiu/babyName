package name

import (
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"

	"name/internal/domain/bazi"
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
	Score           float64   `json:"score"`
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
	TotalScore   float64 `json:"total_score"`   // 综合评分（0-100）
	WuxingScore  float64 `json:"wuxing_score"`  // 五行评分
	YinyunScore  float64 `json:"yinyun_score"`  // 音韵评分
	MeaningScore float64 `json:"meaning_score"` // 字义评分
	SancaiScore  float64 `json:"sancai_score"`  // 天地人三才评分
	ZodiacScore  float64 `json:"zodiac_score"`  // 生肖评分
	NayinScore   float64 `json:"nayin_score"`   // 纳音评分
	NoveltyScore float64 `json:"novelty_score"` // 新颖度评分
	BigramScore  float64 `json:"bigram_score"`  // 诗词共现评分
	SancaiAnalysis string `json:"sancai_analysis"` // 三才分析详情
	Hexagram       string `json:"hexagram"`          // 卦象名称
	HexagramMeaning string `json:"hexagram_meaning"` // 卦象解读
}

// 缓存最大条目数（3500 常用字 + 1605 三级字 ≈ 5105，留余量至 10000）
const maxCacheEntries = 10000

// NameGenerator 名字生成器
type NameGenerator struct {
	Characters []NameChar
	rand       *rand.Rand
	randMutex  sync.Mutex
	charMutex  sync.RWMutex

	// 可配置的常用字表
	MaleNames   []string
	FemaleNames []string

	// 缓存（带最大条目限制，防止内存泄漏）
	charInfoCache map[string]struct {
		pinyin   string
		meaning  string
		wuxing   string
		strokes  int
	}
	charExtraCache map[string]struct {
		radical   string
		structure string
	}
	poetryInfoCache map[string]struct {
		pinyin   string
		meaning  string
		source   string
		chapter  string
		sentence string
		wuxing   string
	}
}

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

// NewNameGenerator 创建名字生成器
func NewNameGenerator() *NameGenerator {
	ng := &NameGenerator{
		rand:      rand.New(rand.NewSource(time.Now().UnixNano())),
		charInfoCache: make(map[string]struct {
			pinyin  string
			meaning string
			wuxing  string
			strokes int
		}),
		charExtraCache: make(map[string]struct {
			radical   string
			structure string
		}),
		poetryInfoCache: make(map[string]struct {
			pinyin   string
			meaning  string
			source   string
			chapter  string
			sentence string
			wuxing   string
		}),
	}

	ng.MaleNames = CommonMaleNames
	ng.FemaleNames = CommonFemaleNames

	return ng
}

// SetMaleNames 设置男性用字表
func (ng *NameGenerator) SetMaleNames(names []string) {
	ng.charMutex.Lock()
	defer ng.charMutex.Unlock()
	ng.MaleNames = names
}

// SetFemaleNames 设置女性用字表
func (ng *NameGenerator) SetFemaleNames(names []string) {
	ng.charMutex.Lock()
	defer ng.charMutex.Unlock()
	ng.FemaleNames = names
}

// GetMaleNames 获取男性用字表
func (ng *NameGenerator) GetMaleNames() []string {
	ng.charMutex.RLock()
	defer ng.charMutex.RUnlock()
	if len(ng.MaleNames) > 0 {
		return ng.MaleNames
	}
	return CommonMaleNames
}

// GetFemaleNames 获取女性用字表
func (ng *NameGenerator) GetFemaleNames() []string {
	ng.charMutex.RLock()
	defer ng.charMutex.RUnlock()
	if len(ng.FemaleNames) > 0 {
		return ng.FemaleNames
	}
	return CommonFemaleNames
}

// GetCharInfo 获取汉字信息
func (ng *NameGenerator) GetCharInfo(char string) (pinyin string, meaning string, wuxing string, strokes int) {
	// 先用读锁检查缓存
	ng.charMutex.RLock()
	if info, ok := ng.charInfoCache[char]; ok {
		ng.charMutex.RUnlock()
		return info.pinyin, info.meaning, info.wuxing, info.strokes
	}
	ng.charMutex.RUnlock()

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
	ng.charMutex.Lock()
	if _, ok := ng.charInfoCache[char]; !ok {
		evictIfNeeded(ng.charInfoCache, maxCacheEntries)
		ng.charInfoCache[char] = struct {
			pinyin  string
			meaning string
			wuxing  string
			strokes int
		}{pinyin, meaning, wuxing, strokes}
	}
	ng.charMutex.Unlock()

	return pinyin, meaning, wuxing, strokes
}

// GetCharExtra 获取汉字的部首和字形结构信息（带缓存）
func (ng *NameGenerator) GetCharExtra(char string) (radical string, structure string) {
	// 先用读锁检查缓存
	ng.charMutex.RLock()
	if info, ok := ng.charExtraCache[char]; ok {
		ng.charMutex.RUnlock()
		return info.radical, info.structure
	}
	ng.charMutex.RUnlock()

	if h, ok := hanzi.HanziData[char]; ok {
		radical = h.Radical
	}

	// 加写锁缓存结果（double-check 避免重复写）
	ng.charMutex.Lock()
	if _, ok := ng.charExtraCache[char]; !ok {
		evictIfNeeded(ng.charExtraCache, maxCacheEntries)
		ng.charExtraCache[char] = struct {
			radical   string
			structure string
		}{radical, structure}
	}
	ng.charMutex.Unlock()

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

// GetPoetrySourceNames 获取诗词来源列表
func GetPoetrySourceNames() []string {
	return []string{"诗经", "楚辞", "唐诗", "宋词", "乐府", "古文名句"}
}

// GetGuwenMingjuChars 获取古文名句用字列表
func GetGuwenMingjuChars(gender string) []string {
	var chars []string
	guwenChars := classics.GetGuwenMingjuCharList(gender)
	for _, c := range guwenChars {
		chars = append(chars, c.Char)
	}
	return chars
}

// GetGuwenMingjuCharInfo 获取古文名句字的详细信息
func GetGuwenMingjuCharInfo(char string) (pinyin, meaning, source, sentence, wuxing string) {
	guwenChars := classics.GetGuwenMingjuChars()
	for _, c := range guwenChars {
		if c.Char == char {
			return c.Pinyin, c.Meaning, c.Source, c.Sentence, c.Wuxing
		}
	}
	return GetPinyin(char), "美好寓意", "古文", "", "未知"
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
func (ng *NameGenerator) GetPoetryCharInfo(char string) (pinyin, meaning, source, chapter, sentence, wuxing string) {
	// 先用读锁检查缓存
	ng.charMutex.RLock()
	if info, ok := ng.poetryInfoCache[char]; ok {
		ng.charMutex.RUnlock()
		return info.pinyin, info.meaning, info.source, info.chapter, info.sentence, info.wuxing
	}
	ng.charMutex.RUnlock()

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
	ng.charMutex.Lock()
	if _, ok := ng.poetryInfoCache[char]; !ok {
		evictIfNeeded(ng.poetryInfoCache, maxCacheEntries)
		ng.poetryInfoCache[char] = struct {
			pinyin   string
			meaning  string
			source   string
			chapter  string
			sentence string
			wuxing   string
		}{pinyin, meaning, source, chapter, sentence, wuxing}
	}
	ng.charMutex.Unlock()

	return pinyin, meaning, source, chapter, sentence, wuxing
}

func (ng *NameGenerator) GenerateNamesWithGeneration(surname, generation, gender string, xiyongshen []string, count int, nameLength int, excludeRare bool, wuxingMatch []string, sourceClassic string, minStrokes int, maxStrokes int, includePoetry bool, includeClassic bool, meaningKeywords []string, pinyinInitial string, generationPosition string, nameType string) []Name {
	return ng.Generate(GenerateOptions{
		Surname:            surname,
		Generation:         generation,
		Gender:             gender,
		Xiyongshen:         xiyongshen,
		Count:              count,
		NameLength:         nameLength,
		ExcludeRare:        excludeRare,
		WuxingMatch:        wuxingMatch,
		SourceClassic:      sourceClassic,
		MinStrokes:         minStrokes,
		MaxStrokes:         maxStrokes,
		IncludePoetry:      includePoetry,
		IncludeClassic:     includeClassic,
		MeaningKeywords:    meaningKeywords,
		PinyinInitial:      pinyinInitial,
		GenerationPosition: generationPosition,
		NameType:           nameType,
	})
}

// Generate 使用选项生成名字
func (ng *NameGenerator) Generate(opts GenerateOptions) []Name {
	surname := opts.Surname
	generation := opts.Generation
	nameLength := opts.NameLength
	nameType := opts.NameType
	generationPosition := opts.GenerationPosition

	// 处理单字名+字辈的情况：不需生成姓名，直接返回空列表
	if (nameLength == 1 || nameType == "single") && generation != "" {
		return []Name{}
	}

	// 计算实际需要生成的名字字符数
	var actualNameLength int
	if nameLength == 1 {
		actualNameLength = 1
	} else if nameLength == 2 {
		if generation != "" {
			actualNameLength = 1
		} else {
			actualNameLength = 2
		}
	} else {
		actualNameLength = nameLength
	}

	names := ng.generateNamesWithOptions(
		surname, opts.Gender, opts.Xiyongshen, opts.Count, actualNameLength,
		opts.ExcludeRare, opts.WuxingMatch, opts.SourceClassic,
		opts.MinStrokes, opts.MaxStrokes, opts.IncludePoetry, opts.IncludeClassic,
		opts.MeaningKeywords, opts.PinyinInitial,
	)

	// 避讳长辈：后置过滤，移除包含长辈同形字的名字（安全网）
	if len(opts.AvoidElderNames) > 0 {
		elderCharSet, _ := buildElderAvoidanceSets(opts.AvoidElderNames)
		if len(elderCharSet) > 0 {
			filtered := names[:0]
			for _, n := range names {
				avoid := false
				for _, r := range n.GivenName {
					if elderCharSet[string(r)] {
						avoid = true
						break
					}
				}
				if !avoid {
					filtered = append(filtered, n)
				}
			}
			names = filtered
		}
	}

	// 大名需大命：普通格局后置过滤敏感字（安全网）
	if !allowSensitiveChar(opts.DayMasterStrength) {
		filtered := names[:0]
		for _, n := range names {
			avoid := false
			for _, r := range n.GivenName {
				if isSensitiveNameChar(string(r)) {
					avoid = true
					break
				}
			}
			if !avoid {
				filtered = append(filtered, n)
			}
		}
		names = filtered
	}

	if generation != "" {
		for i := range names {
			names[i].Generation = generation
			if generationPosition == "middle" {
				names[i].FullName = surname + generation + names[i].GivenName
			} else {
				names[i].FullName = surname + names[i].GivenName + generation
			}
		}
	} else {
		for i := range names {
			names[i].FullName = surname + names[i].GivenName
		}
	}

	// 按评分排序
	sort.Slice(names, func(i, j int) bool {
		return names[i].Score > names[j].Score
	})

	return names
}

func (ng *NameGenerator) generateNamesWithOptions(surname string, gender string, xiyongshen []string, count int, nameLength int, excludeRare bool, wuxingMatch []string, sourceClassic string, minStrokes int, maxStrokes int, includePoetry bool, includeClassic bool, meaningKeywords []string, pinyinInitial string) []Name {
	var names []Name
	var charList []string
	// 合并多个来源的字符
	allChars := make(map[string]bool)
	
	// 基础字符集
	if sourceClassic == "shijing" || sourceClassic == "chuci" || sourceClassic == "poetry" || sourceClassic == "诗经" || sourceClassic == "楚辞" || sourceClassic == "唐诗" || sourceClassic == "古文名句" || sourceClassic == "古文观止" || sourceClassic == "唐诗宋词" {
		poetryChars := GetPoetryNames(gender, sourceClassic)
		for _, char := range poetryChars {
			allChars[char] = true
		}
	}
	
	// 包含诗词
	if includePoetry {
		poetryChars := GetPoetryNames(gender, "poetry")
		for _, char := range poetryChars {
			allChars[char] = true
		}
	}
	
	// 包含经典
	if includeClassic {
		classicChars := GetClassicNames(gender)
		for _, char := range classicChars {
			allChars[char] = true
		}
	}
	
	// 如果没有指定来源，使用性别默认字符
	if len(allChars) == 0 {
		if gender == "male" {
			for _, char := range CommonMaleNames {
				allChars[char] = true
			}
		} else {
			for _, char := range CommonFemaleNames {
				allChars[char] = true
			}
		}
	}
	
	// 转换为切片
	for char := range allChars {
		charList = append(charList, char)
	}
	filteredChars := []string{}
	for _, char := range charList {
		if excludeRare && !hanzi.IsCommonChar(char) {
			continue
		}

		if len(wuxingMatch) > 0 {
			wuxing := ""
			if h, ok := hanzi.HanziData[char]; ok {
				wuxing = h.Wuxing
			}
			hasMatch := false
			for _, w := range wuxingMatch {
				if wuxing == w {
					hasMatch = true
					break
				}
			}
			if !hasMatch {
				continue
			}
		}
		
		// 笔画数筛选
		strokes := 0
		if h, ok := hanzi.HanziData[char]; ok {
			strokes = h.Strokes
		} else {
			strokes = 10
		}
		if minStrokes > 0 && strokes < minStrokes {
			continue
		}
		if maxStrokes > 0 && strokes > maxStrokes {
			continue
		}
		
		// 拼音首字母筛选
		if pinyinInitial != "" {
			pinyin := GetPinyin(char)
			if len(pinyin) > 0 && pinyin[0:1] != pinyinInitial {
				continue
			}
		}
		
		// 寓意关键词筛选
		if len(meaningKeywords) > 0 {
			meaning := ""
			if h, ok := hanzi.HanziData[char]; ok {
				meaning = h.Meaning
			}
			hasKeyword := false
			for _, keyword := range meaningKeywords {
				if len(meaning) > 0 && (keyword == "" || contains(meaning, keyword)) {
					hasKeyword = true
					break
				}
			}
			if !hasKeyword {
				continue
			}
		}

		filteredChars = append(filteredChars, char)
	}

	if len(filteredChars) == 0 {
		filteredChars = charList
	}

	shuffled := make([]string, len(filteredChars))
	copy(shuffled, filteredChars)
	ng.randMutex.Lock()
	ng.rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	ng.randMutex.Unlock()
	generatedNames := make(map[string]bool)
	var mu sync.Mutex
	var wg sync.WaitGroup
	
	// 并发生成名字
	batchSize := 100 // 每批处理的字符数
	maxWorkers := 4  // 最大并发数
	
	// 计算需要的批次数
	batches := len(shuffled) / batchSize
	if len(shuffled)%batchSize > 0 {
		batches++
	}
	
	// 限制批次数，避免生成过多名字
	if batches > maxWorkers {
		batches = maxWorkers
	}
	
	for batch := 0; batch < batches; batch++ {
		wg.Add(1)
		go func(batchIndex int) {
			defer wg.Done()
			
			startIdx := batchIndex * batchSize
			endIdx := (batchIndex + 1) * batchSize
			if endIdx > len(shuffled) {
				endIdx = len(shuffled)
			}
			
			localNames := []Name{}
			localGeneratedNames := make(map[string]bool)
			
			for i := startIdx; i < endIdx && len(localNames) < count; {
				var givenName string
				var pinyin string
				var meaning string
				var wuxing string
				var strokes int
				var poetrySrc, poetryChapter, poetrySentence string

				if nameLength == 1 {
					// 生成1个字符的名字部分（单字名：姓氏 + 1个名字字符）
					if i >= len(shuffled) {
						i++
						continue
					}
					char := shuffled[i]
					givenName = char
					pinyin, meaning, wuxing, strokes = ng.GetCharInfo(char)

					if sourceClassic == "shijing" || sourceClassic == "chuci" || sourceClassic == "poetry" || sourceClassic == "诗经" || sourceClassic == "楚辞" || sourceClassic == "唐诗" || sourceClassic == "古文观止" || sourceClassic == "唐诗宋词" {
						if _, m, src, chapter, sentence, _ := ng.GetPoetryCharInfo(char); m != "" {
							poetrySrc = src
							poetryChapter = chapter
							poetrySentence = sentence
							if chapter != "" {
								meaning = m + "（出自" + src + "《" + chapter + "》"
								if sentence != "" {
									meaning += "：" + sentence
								}
								meaning += "）"
							} else {
								meaning = m + "（出自" + src + "）"
							}
						}
					}
					i++ // 只使用一个字符
				} else if nameLength == 2 {
					// 生成2个字符的名字部分（双字名：姓氏 + 2个名字字符）
					if i+1 >= len(shuffled) {
						i++
						continue
					}
					char1 := shuffled[i]
					char2 := shuffled[i+1]
					givenName = char1 + char2
					
					pinyin1, meaning1, wuxing1, strokes1 := ng.GetCharInfo(char1)
					pinyin2, meaning2, wuxing2, strokes2 := ng.GetCharInfo(char2)
					
					pinyin = pinyin1 + " " + pinyin2
					meaning = meaning1 + "、" + meaning2
					wuxing = wuxing1 + "、" + wuxing2
					strokes = strokes1 + strokes2
					i += 2 // 跳过两个字符
				} else if nameLength == 3 {
					// 生成3个字符的名字部分
					if i+2 >= len(shuffled) {
						i++
						continue
					}
					char1 := shuffled[i]
					char2 := shuffled[i+1]
					char3 := shuffled[i+2]
					givenName = char1 + char2 + char3
					
					pinyin1, meaning1, wuxing1, strokes1 := ng.GetCharInfo(char1)
					pinyin2, meaning2, wuxing2, strokes2 := ng.GetCharInfo(char2)
					pinyin3, meaning3, wuxing3, strokes3 := ng.GetCharInfo(char3)
					
					pinyin = pinyin1 + " " + pinyin2 + " " + pinyin3
					meaning = meaning1 + "、" + meaning2 + "、" + meaning3
					wuxing = wuxing1 + "、" + wuxing2 + "、" + wuxing3
					strokes = strokes1 + strokes2 + strokes3
					i += 3 // 跳过三个字符
				} else if nameLength == 4 {
					// 生成4个字符的名字部分
					if i+3 >= len(shuffled) {
						i++
						continue
					}
					char1 := shuffled[i]
					char2 := shuffled[i+1]
					char3 := shuffled[i+2]
					char4 := shuffled[i+3]
					givenName = char1 + char2 + char3 + char4
					
					pinyin1, meaning1, wuxing1, strokes1 := ng.GetCharInfo(char1)
					pinyin2, meaning2, wuxing2, strokes2 := ng.GetCharInfo(char2)
					pinyin3, meaning3, wuxing3, strokes3 := ng.GetCharInfo(char3)
					pinyin4, meaning4, wuxing4, strokes4 := ng.GetCharInfo(char4)
					
					pinyin = pinyin1 + " " + pinyin2 + " " + pinyin3 + " " + pinyin4
					meaning = meaning1 + "、" + meaning2 + "、" + meaning3 + "、" + meaning4
					wuxing = wuxing1 + "、" + wuxing2 + "、" + wuxing3 + "、" + wuxing4
					strokes = strokes1 + strokes2 + strokes3 + strokes4
					i += 4 // 跳过四个字符
				} else {
					// 默认生成1个字符的名字部分
					if i >= len(shuffled) {
						i++
						continue
					}
					char := shuffled[i]
					givenName = char
					pinyin, meaning, wuxing, strokes = ng.GetCharInfo(char)

					if sourceClassic == "shijing" || sourceClassic == "chuci" || sourceClassic == "poetry" || sourceClassic == "诗经" || sourceClassic == "楚辞" || sourceClassic == "唐诗" || sourceClassic == "古文观止" || sourceClassic == "唐诗宋词" {
						if _, m, src, chapter, sentence, _ := ng.GetPoetryCharInfo(char); m != "" {
							poetrySrc = src
							poetryChapter = chapter
							poetrySentence = sentence
							if chapter != "" {
								meaning = m + "（出自" + src + "《" + chapter + "》"
								if sentence != "" {
									meaning += "：" + sentence
								}
								meaning += "）"
							} else {
								meaning = m + "（出自" + src + "）"
							}
						}
					}
					i++ // 只使用一个字符
				}

				// 允许单字名（只有姓氏）
				// if givenName == "" {
				// 	continue
				// }

				// 检查本地和全局重复
				if localGeneratedNames[givenName] {
					continue
				}
				
				mu.Lock()
				if generatedNames[givenName] {
					mu.Unlock()
					continue
				}
				// 标记为已生成
				generatedNames[givenName] = true
				mu.Unlock()
				
				// 标记本地已生成
				localGeneratedNames[givenName] = true

				score := ng.calculateScore(wuxing, xiyongshen)

				// 生成推荐理由
				reasons := []string{}
				if score >= 90 {
					reasons = append(reasons, "高分名字")
				}
				if poetrySrc != "" {
					reasons = append(reasons, "诗词典故")
				}
				if wuxing != "未知" {
					reasons = append(reasons, "五行匹配")
				}

				// 生成五行分析
				wuxingAnalysis := ""
				if wuxing != "未知" {
					wuxingAnalysis = "名字五行属性为" + wuxing + "，与八字命理相协调，有助于宝宝的运势发展。"
				}

				// 生成八字评分详情
				baziScoreDetail := ""
				baziScore := ng.CalculateBaZiScore(wuxing, "", xiyongshen)
				if baziScore >= 80 {
					baziScoreDetail = "八字评分较高，名字与宝宝的生辰八字匹配度良好，有助于宝宝的成长和发展。"
				} else if baziScore >= 60 {
					baziScoreDetail = "八字评分适中，名字与宝宝的生辰八字有一定匹配度，适合宝宝使用。"
				} else {
					baziScoreDetail = "八字评分较低，建议考虑其他名字选项。"
				}

				// 生成音韵意境
				yinyun := ""
				if len(pinyin) > 0 {
					yinyun = "名字音韵和谐，读起来朗朗上口，给人留下深刻印象。"
				}

				localNames = append(localNames, Name{
					ID:              int64(len(localNames) + 1),
					Surname:         surname,
					GivenName:       givenName,
					Pinyin:          pinyin,
					Meaning:         meaning,
					Wuxing:          wuxing,
					Strokes:         strokes,
					Gender:          gender,
					Score:           score,
					PoetrySource:    poetrySrc,
					PoetryChapter:   poetryChapter,
					PoetrySentence:  poetrySentence,
					WuxingAnalysis:  wuxingAnalysis,
					BaziScoreDetail: baziScoreDetail,
					Yinyun:          yinyun,
					Reasons:         reasons,
				})
			}
			
			// 将本地生成的名字添加到全局列表
			mu.Lock()
			names = append(names, localNames...)
			mu.Unlock()
		}(batch)
	}
	
	// 等待所有并发任务完成
	wg.Wait()
	
	// 限制返回的名字数量
if len(names) > count {
		names = names[:count]
	}

	return names
}

// containsWuxing 检查名字五行中是否包含指定五行
func containsWuxing(nameWuxing, targetWuxing string) bool {
	if nameWuxing == targetWuxing {
		return true
	}
	// 处理多五行情况，如"土、水"
	for _, w := range []string{"木", "火", "土", "金", "水"} {
		if w == targetWuxing && contains(nameWuxing, w) {
			return true
		}
	}
	return false
}

// calculateScore 计算名字评分（确定性算法，无随机因素）
func (ng *NameGenerator) calculateScore(wuxing string, xiyongshen []string) float64 {
	baseScore := 70.0

	// 检查五行匹配喜用神
	matchCount := 0
	for _, xy := range xiyongshen {
		if containsWuxing(wuxing, xy) {
			matchCount++
		}
	}

	if matchCount > 0 {
		// 每匹配一个喜用神加5分，最高加25分
		bonus := float64(matchCount) * 5.0
		if bonus > 25.0 {
			bonus = 25.0
		}
		return baseScore + 25.0 + bonus
	}

	return baseScore
}

// CalculateBaZiScore 计算八字评分
func (ng *NameGenerator) CalculateBaZiScore(nameWuxing, dayWuxing string, xiyongshen []string) int {
	score := 50

	if containsWuxing(nameWuxing, dayWuxing) {
		score += 20
	}

	for _, xy := range xiyongshen {
		if containsWuxing(nameWuxing, xy) {
			score += 30
			break
		}
	}

	wuxingRelations := map[string]map[string]int{
		"木": {"火": 15, "土": -10},
		"火": {"土": 15, "金": -10},
		"土": {"金": 15, "水": -10},
		"金": {"水": 15, "木": -10},
		"水": {"木": 15, "土": -10},
	}

	// 处理多五行情况，检查名字中是否有与日主五行有生克关系的五行
	if rel, ok := wuxingRelations[dayWuxing]; ok {
		for _, w := range []string{"木", "火", "土", "金", "水"} {
			if containsWuxing(nameWuxing, w) {
				if bonus, ok := rel[w]; ok {
					score += bonus
					break
				}
			}
		}
	}

	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}

	return score
}

// GetNameNayin 获取名字的纳音五行
func GetNameNayin(surname, givenName, dayGan, dayZhi string) string {
	if len(givenName) > 0 {
		if len(dayZhi) > 0 {
			ganzhi := dayGan + dayZhi
			return bazi.GetNayin(ganzhi)
		}
	}
	return ""
}

// CheckHuangliYiji 检查黄历宜忌
func CheckHuangliYiji(yiList, jiList []string) (yi string, ji string) {
	auspiciousMap := map[string]int{
		"嫁娶": 10, "纳采": 8, "订盟": 8, "祭祀": 10,
		"祈福": 10, "求嗣": 10, "开光": 8, "出行": 6,
		"移徙": 6, "入宅": 6, "安床": 5, "动土": -5,
		"破土": -10, "安葬": -10, "开市": 5, "立券": 5,
		"交易": 5,
	}

	inauspiciousMap := map[string]int{
		"开市": -3, "立券": -2, "交易": -2, "动土": -5,
		"破土": -10, "安葬": -10, "伐木": -5, "纳畜": -3,
	}

	yiScore := 0
	for _, y := range yiList {
		yiScore += auspiciousMap[y]
	}

	jiScore := 0
	for _, j := range jiList {
		jiScore += inauspiciousMap[j]
	}

	totalScore := yiScore + jiScore
	if totalScore >= 10 {
		yi = "大吉"
	} else if totalScore >= 5 {
		yi = "吉"
	} else if totalScore >= 0 {
		yi = "平"
	} else if totalScore >= -5 {
		ji = "凶"
	} else {
		ji = "大凶"
	}

	return
}

// EnrichNameWithBaZi 使用八字和黄历数据增强名字
func (ng *NameGenerator) EnrichNameWithBaZi(name *Name, year, month, day int) *Name {
	lc := bazi.GetLunarCalendar(year, month, day, 12, 0)
	dayWuxing := bazi.DizhiWuxingMap[lc.DayZhi]

	ng.CalculateBaZiScore(name.Wuxing, dayWuxing, nil)

	hl := bazi.GetHuangli(year, month, day)
	yi, ji := CheckHuangliYiji(hl.Yi, hl.Ji)

	if yi != "" {
		name.Huangli = yi
	} else if ji != "" {
		name.Huangli = ji
	} else {
		name.Huangli = "平"
	}

	if lc.DayNayin != "" {
		name.Nayin = lc.DayNayin
	}

	return name
}

// EnrichNamesWithBaZi 批量增强名字的八字和黄历信息
func (ng *NameGenerator) EnrichNamesWithBaZi(names []Name, year, month, day int) []Name {
	for i := range names {
		ng.EnrichNameWithBaZi(&names[i], year, month, day)
	}
	return names
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
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
