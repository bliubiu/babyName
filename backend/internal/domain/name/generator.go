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

// NameAnalysis 名字详细分析结果（GenerateWithAnalysis 响应结构）
type NameAnalysis struct {
	// 基础信息
	Surname        string `json:"surname"`
	GivenName      string `json:"given_name"`
	FullName       string `json:"full_name"`
	Pinyin         string `json:"pinyin"`
	Strokes        int    `json:"strokes"`
	Gender         string `json:"gender"`

	// 五行分析
	Wuxing         string  `json:"wuxing"`
	WuxingAnalysis string  `json:"wuxing_analysis"`
	WuxingScore    float64 `json:"wuxing_score"`

	// 八字分析
	BaziScore      int      `json:"bazi_score"`
	BaziAnalysis   string   `json:"bazi_analysis"`
	Xiyongshen     []string `json:"xiyongshen"`

	// 音韵分析
	YinyunScore    float64 `json:"yinyun_score"`
	YinyunAnalysis string  `json:"yinyun_analysis"`

	// 字义分析
	MeaningScore   float64 `json:"meaning_score"`
	MeaningDetail  string  `json:"meaning_detail"`

	// 易经分析
	Hexagram        string `json:"hexagram"`
	HexagramMeaning string `json:"hexagram_meaning"`

	// 三才分析（天地人：天道阴阳、地道刚柔、人道仁义）
	Sancai         string  `json:"sancai"`
	SancaiScore    float64 `json:"sancai_score"`
	SancaiAnalysis string  `json:"sancai_analysis"`

	// 生肖分析
	ZodiacMatch string  `json:"zodiac_match"`
	ZodiacScore float64 `json:"zodiac_score"`

	// 新颖度评分
	NoveltyScore float64 `json:"novelty_score"`

	// 诗词共现评分
	BigramScore float64 `json:"bigram_score"`

	// 人名频率评分（来自 Chinese-Names-Corpus 语料统计）
	FrequencyScore float64 `json:"frequency_score"`

	// 纳音分析
	NayinScore    float64 `json:"nayin_score"`
	NayinAnalysis string  `json:"nayin_analysis"`

	// 候选名库加分
	NameDBScore  float64 `json:"name_db_score"`
	NameDBSource string  `json:"name_db_source"`

	// 诗词典故
	PoetrySource   string `json:"poetry_source"`
	PoetryChapter  string `json:"poetry_chapter"`
	PoetrySentence string `json:"poetry_sentence"`

	// 综合评分
	TotalScore      float64  `json:"total_score"`
	Recommendations []string `json:"recommendations"`
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
