package name

import (
	"math/rand"
	"time"

	"github.com/namemaster/backend/internal/domain/bazi"
	"github.com/namemaster/backend/internal/domain/classics"
	"github.com/namemaster/backend/internal/domain/hanzi"
)

// Name 名字结构
type Name struct {
	ID          int64   `json:"id"`
	Surname     string  `json:"surname"`
	Generation  string  `json:"generation"`
	GivenName   string  `json:"given_name"`
	FullName    string  `json:"full_name"`
	Pinyin      string  `json:"pinyin"`
	Meaning     string  `json:"meaning"`
	Wuxing      string  `json:"wuxing"`
	Nayin       string  `json:"nayin"`
	Strokes     int     `json:"strokes"`
	Gender      string  `json:"gender"`
	Score       float64 `json:"score"`
	BaZiScore   int     `json:"bazi_score"`
	Huangli     string  `json:"huangli"`
	Xiang       string  `json:"xiang"`
	PoetrySource string `json:"poetry_source,omitempty"`
	PoetryChapter string `json:"poetry_chapter,omitempty"`
	PoetrySentence string `json:"poetry_sentence,omitempty"`
}

// NameGenerator 名字生成器
type NameGenerator struct {
	Characters []NameChar
	rand      *rand.Rand
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

// HanziMeaningMap 汉字寓意表
var HanziMeaningMap = map[string]string{
	"伟": "伟大、卓越",
	"强": "强大、强壮",
	"磊": "光明磊落",
	"军": "军人、军队",
	"杰": "杰出、优秀",
	"涛": "波涛、浪涛",
	"明": "明亮、聪明",
	"超": "超越、杰出",
	"勇": "勇敢、勇猛",
	"鹏": "大鹏鸟、志向远大",
	"华": "华丽、才华",
	"刚": "刚强、坚硬",
	"平": "平安、公平",
	"辉": "光辉、辉煌",
	"波": "波浪、起伏",
	"峰": "山峰、顶点",
	"飞": "飞翔、自由",
	"龙": "龙、尊贵",
	"浩": "浩大、广阔",
	"宇": "宇宙、气度",
	"晨": "早晨、希望",
	"逸": "安逸、超逸",
	"睿": "睿智、聪明",
	"哲": "智慧、哲理",
	"渊": "深渊、学识",
	"博": "博学、广博",
	"昊": "天空、广阔",
	"然": "自然、如此",
	"轩": "高远、气派",
	"俊": "俊秀、英俊",
	"豪": "豪爽、豪迈",
	"毅": "坚毅、刚毅",
	"文": "文化、文明",
	"武": "武力、勇武",
	"祥": "吉祥、祥瑞",
	"瑞": "吉祥、美好",
	"凯": "胜利、凯旋",
	"成": "成功、成就",
	"盛": "兴盛、旺盛",
	"雄": "雄伟、英雄",
	"鑫": "财富、兴盛",
	"霖": "甘霖、恩泽",
	"思": "思考、思念",
	"远": "远方、长远",
	"航": "航行、飞翔",
	"洋": "海洋、广阔",
	"庆": "庆祝、喜庆",
	"芳": "芳香、美好",
	"兰": "兰花、高雅",
	"梅": "梅花坚强",
	"琴": "琴瑟、和谐",
	"丽": "美丽、漂亮",
	"秀": "秀丽、优秀",
	"英": "英雄、花蕾",
	"敏": "敏捷、聪明",
	"静": "宁静、安静",
	"燕": "燕子、自由",
	"霞": "彩霞、美丽",
	"红": "红色、热",
	"玉": "玉石、美好",
	"珍": "珍贵、宝贵",
	"娟": "娟秀、美好",
	"莉": "茉莉、美好",
	"萍": "浮萍、漂流",
	"颖": "颖悟、聪明",
	"娜": "婀娜、美丽",
	"欣": "欣喜、愉快",
	"怡": "愉快、和悦",
	"菲": "芳菲、花草",
	"雅": "高雅、文雅",
	"芸": "芸香、勤劳",
	"萱": "萱草、忘忧",
	"雨": "雨水、甘露",
	"诗": "诗意、才华",
	"梦": "梦想、希望",
	"瑶": "美玉、珍贵",
	"琳": "美玉、美好",
	"珂": "美玉、珍贵",
	"雯": "云彩、美丽",
	"倩": "倩影、美好",
	"雪": "雪花、纯洁",
	"婷": "婷婷、优美",
	"霜": "霜雪、坚强",
	"翠": "翠绿、美好",
	"碧": "碧绿、美丽",
	"银": "白银、珍贵",
	"金": "黄金、珍贵",
	"珠": "珍珠、珍贵",
}

// HanziWuxingMap 汉字五行表
var HanziWuxingMap = map[string]string{
	"伟": "土", "强": "木", "磊": "土", "军": "木",
	"杰": "木", "涛": "水", "明": "火", "超": "金",
	"勇": "土", "鹏": "水", "华": "水", "刚": "金",
	"平": "水", "辉": "水", "波": "水", "峰": "土",
	"飞": "水", "龙": "火", "浩": "水", "宇": "土",
	"晨": "火", "逸": "土", "睿": "金", "哲": "火",
	"渊": "水", "博": "火", "昊": "火", "然": "火",
	"轩": "土", "俊": "火", "豪": "水", "毅": "土",
	"文": "水", "武": "水", "祥": "金", "瑞": "金",
	"凯": "木", "成": "金", "盛": "火", "雄": "水",
	"鑫": "金", "霖": "水", "思": "金", "远": "土",
	"航": "水", "洋": "水", "庆": "木",
	"芳": "木", "兰": "木", "梅": "木", "琴": "木",
	"丽": "火", "秀": "金", "英": "木", "敏": "水",
	"静": "金", "燕": "土", "霞": "水", "红": "水",
	"玉": "木", "珍": "火", "娟": "木", "莉": "木",
	"萍": "木", "颖": "木", "娜": "火", "欣": "木",
	"怡": "土", "菲": "木", "雅": "木", "芸": "木",
	"萱": "木", "雨": "水", "诗": "金", "梦": "木",
	"瑶": "火", "琳": "木", "珂": "木", "雯": "水",
	"倩": "金", "雪": "水", "婷": "火", "霜": "水",
	"翠": "金", "碧": "水", "银": "金", "金": "金",
	"珠": "火",
}

// HanziStrokesMap 汉字笔画表
var HanziStrokesMap = map[string]int{
	"伟": 7, "强": 12, "磊": 15, "军": 6,
	"杰": 8, "涛": 10, "明": 8, "超": 12,
	"勇": 9, "鹏": 13, "华": 10, "刚": 6,
	"平": 5, "辉": 12, "波": 8, "峰": 10,
	"飞": 9, "龙": 16, "浩": 10, "宇": 6,
	"晨": 11, "逸": 12, "睿": 16, "哲": 10,
	"渊": 11, "博": 12, "昊": 8, "然": 12,
	"轩": 7, "俊": 9, "豪": 14, "毅": 15,
	"文": 4, "武": 8, "祥": 10, "瑞": 13,
	"凯": 8, "成": 6, "盛": 11, "雄": 12,
	"鑫": 24, "霖": 16, "思": 9, "远": 12,
	"航": 10, "洋": 9, "庆": 6,
	"芳": 7, "兰": 5, "梅": 11, "琴": 12,
	"丽": 7, "秀": 7, "英": 8, "敏": 11,
	"静": 14, "燕": 16, "霞": 17, "红": 6,
	"玉": 5, "珍": 9, "娟": 10, "莉": 10,
	"萍": 11, "颖": 13, "娜": 9, "欣": 8,
	"怡": 8, "菲": 11, "雅": 12, "芸": 7,
	"萱": 12, "雨": 8, "诗": 8, "梦": 11,
	"瑶": 13, "琳": 12, "珂": 9, "雯": 12,
	"倩": 10, "雪": 11, "婷": 12, "霜": 17,
	"翠": 14, "碧": 14, "银": 14, "金": 8,
	"珠": 10,
}

// NewNameGenerator 创建名字生成器
func NewNameGenerator() *NameGenerator {
	return &NameGenerator{
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// GetMaleChars 获取男性用字列表
func GetMaleChars() []string {
	var chars []string
	for _, h := range hanzi.HanziData {
		if h.Gender == "male" || h.Gender == "通用" {
			chars = append(chars, h.Char)
		}
	}
	if len(chars) == 0 {
		return CommonMaleNames
	}
	return chars
}

// GetFemaleChars 获取女性用字列表
func GetFemaleChars() []string {
	var chars []string
	for _, h := range hanzi.HanziData {
		if h.Gender == "female" || h.Gender == "通用" {
			chars = append(chars, h.Char)
		}
	}
	if len(chars) == 0 {
		return CommonFemaleNames
	}
	return chars
}

// GetCharInfo 获取汉字信息
func GetCharInfo(char string) (pinyin string, meaning string, wuxing string, strokes int) {
	if h, ok := hanzi.HanziData[char]; ok {
		return h.Pinyin, h.Meaning, h.Wuxing, h.Strokes
	}
	return GetPinyin(char), "美好寓意", "未知", 10
}

// GetClassicNames 获取诗经楚辞名字
func GetClassicNames(gender string) []string {
	var chars []string
	for _, n := range classics.ShijingNames {
		if gender == "male" && (n.Gender == "male" || n.Gender == "通用") {
			chars = append(chars, n.Char)
		} else if gender == "female" && (n.Gender == "female" || n.Gender == "通用") {
			chars = append(chars, n.Char)
		} else if n.Gender == "通用" {
			chars = append(chars, n.Char)
		}
	}
	for _, n := range classics.ChuciNames {
		if gender == "male" && (n.Gender == "male" || n.Gender == "通用") {
			chars = append(chars, n.Char)
		} else if gender == "female" && (n.Gender == "female" || n.Gender == "通用") {
			chars = append(chars, n.Char)
		} else if n.Gender == "通用" {
			chars = append(chars, n.Char)
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
		if gender == "male" && (n.Gender == "male" || n.Gender == "通用") {
			chars = append(chars, n.Char)
		} else if gender == "female" && (n.Gender == "female" || n.Gender == "通用") {
			chars = append(chars, n.Char)
		} else if n.Gender == "通用" {
			chars = append(chars, n.Char)
		}
	}

	if len(chars) == 0 {
		chars = GetClassicNames(gender)
	}

	return chars
}

// GetPoetryCharInfo 获取诗词字的详细信息
func GetPoetryCharInfo(char string) (pinyin, meaning, source, chapter, sentence, wuxing string) {
	for _, ps := range classics.PoetrySources {
		for _, n := range ps.Chars {
			if n.Char == char {
				return n.Pinyin, n.Meaning, n.Work, n.Chapter, n.Sentence, n.Wuxing
			}
		}
	}

	for _, n := range classics.ShijingNames {
		if n.Char == char {
			return n.Pinyin, n.Meaning, n.Source, n.Chapter, "", n.Wuxing
		}
	}
	for _, n := range classics.ChuciNames {
		if n.Char == char {
			return n.Pinyin, n.Meaning, n.Source, n.Chapter, "", n.Wuxing
		}
	}
	return GetPinyin(char), "美好寓意", "诗词", "", "", "未知"
}

// GeneratePoetryNames 生成诗词名字
func (ng *NameGenerator) GeneratePoetryNames(surname string, gender string, xiyongshen []string, count int, source string) []Name {
	var names []Name
	charList := GetPoetryNames(gender, source)

	seen := make(map[string]bool)
	var uniqueChars []string
	for _, c := range charList {
		if !seen[c] {
			seen[c] = true
			uniqueChars = append(uniqueChars, c)
		}
	}

	shuffled := make([]string, len(uniqueChars))
	copy(shuffled, uniqueChars)
	ng.rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	for i := 0; i < len(shuffled) && len(names) < count; i++ {
		char := shuffled[i]
		pinyin, meaning, sourceName, chapter, sentence, wuxing := GetPoetryCharInfo(char)

		strokes := 10
		if h, ok := hanzi.HanziData[char]; ok {
			strokes = h.Strokes
		}

		score := ng.calculateScore(wuxing, xiyongshen)

		meaningDetail := meaning
		if chapter != "" {
			meaningDetail += "（出自" + sourceName + "《" + chapter + "》"
			if sentence != "" {
				meaningDetail += "：" + sentence
			}
			meaningDetail += "）"
		} else if sourceName != "" {
			meaningDetail += "（出自" + sourceName + "）"
		}

		names = append(names, Name{
			ID:        int64(i + 1),
			Surname:   surname,
			GivenName: char,
			Pinyin:    pinyin,
			Meaning:   meaningDetail,
			Wuxing:    wuxing,
			Strokes:   strokes,
			Gender:    gender,
			Score:     score,
		})
	}

	return names
}

// GetClassicNameInfo 获取诗经楚辞字的信息
func GetClassicNameInfo(char string) (pinyin, meaning, source, wuxing string) {
	for _, n := range classics.ShijingNames {
		if n.Char == char {
			return n.Pinyin, n.Meaning, n.Source, n.Wuxing
		}
	}
	for _, n := range classics.ChuciNames {
		if n.Char == char {
			return n.Pinyin, n.Meaning, n.Source, n.Wuxing
		}
	}
	return GetPinyin(char), "美好寓意", "诗经楚辞", "未知"
}

// GenerateClassicNames 生成诗经楚辞名字
func (ng *NameGenerator) GenerateClassicNames(surname string, gender string, xiyongshen []string, count int) []Name {
	var names []Name
	charList := GetClassicNames(gender)

	shuffled := make([]string, len(charList))
	copy(shuffled, charList)
	ng.rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	for i := 0; i < len(shuffled) && len(names) < count; i++ {
		char := shuffled[i]
		pinyin, meaning, source, wuxing := GetClassicNameInfo(char)

		strokes := 10
		if h, ok := hanzi.HanziData[char]; ok {
			strokes = h.Strokes
		}

		score := ng.calculateScore(wuxing, xiyongshen)

		names = append(names, Name{
			ID:        int64(i + 1),
			Surname:   surname,
			GivenName: char,
			Pinyin:    pinyin,
			Meaning:   meaning + "（出自" + source + "）",
			Wuxing:    wuxing,
			Strokes:   strokes,
			Gender:    gender,
			Score:     score,
		})
	}

	return names
}

// GenerateGuwenNames 生成古文名句名字
func (ng *NameGenerator) GenerateGuwenNames(surname string, gender string, xiyongshen []string, count int) []Name {
	var names []Name
	charList := GetGuwenMingjuChars(gender)

	seen := make(map[string]bool)
	var uniqueChars []string
	for _, c := range charList {
		if !seen[c] {
			seen[c] = true
			uniqueChars = append(uniqueChars, c)
		}
	}

	shuffled := make([]string, len(uniqueChars))
	copy(shuffled, uniqueChars)
	ng.rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	for i := 0; i < len(shuffled) && len(names) < count; i++ {
		char := shuffled[i]
		pinyin, meaning, source, sentence, wuxing := GetGuwenMingjuCharInfo(char)

		strokes := 10
		if h, ok := hanzi.HanziData[char]; ok {
			strokes = h.Strokes
		}

		score := ng.calculateScore(wuxing, xiyongshen)

		meaningDetail := meaning
		if source != "" && source != "古文" {
			meaningDetail += "（出自" + source
			if sentence != "" {
				meaningDetail += "：" + sentence
			}
			meaningDetail += "）"
		}

		names = append(names, Name{
			ID:        int64(i + 1),
			Surname:   surname,
			GivenName: char,
			Pinyin:    pinyin,
			Meaning:   meaningDetail,
			Wuxing:    wuxing,
			Strokes:   strokes,
			Gender:    gender,
			Score:     score,
		})
	}

	return names
}

// GenerateNames 生成名字列表
func (ng *NameGenerator) GenerateNames(surname string, gender string, xiyongshen []string, count int) []Name {
	return ng.generateNamesWithOptions(surname, gender, xiyongshen, count, false, nil, "")
}

func (ng *NameGenerator) GenerateNamesWithOptions(surname string, gender string, xiyongshen []string, count int, excludeRare bool, wuxingMatch []string, sourceClassic string) []Name {
	return ng.generateNamesWithOptions(surname, gender, xiyongshen, count, excludeRare, wuxingMatch, sourceClassic)
}

func (ng *NameGenerator) GenerateNamesWithGeneration(surname, generation, gender string, xiyongshen []string, count int, excludeRare bool, wuxingMatch []string, sourceClassic string) []Name {
	names := ng.generateNamesWithOptions(surname, gender, xiyongshen, count, excludeRare, wuxingMatch, sourceClassic)

	if generation != "" {
		for i := range names {
			names[i].Generation = generation
			names[i].FullName = surname + generation + names[i].GivenName
		}
	} else {
		for i := range names {
			names[i].FullName = surname + names[i].GivenName
		}
	}

	return names
}

func (ng *NameGenerator) generateNamesWithOptions(surname string, gender string, xiyongshen []string, count int, excludeRare bool, wuxingMatch []string, sourceClassic string) []Name {
	var names []Name
	var charList []string

	if sourceClassic == "shijing" || sourceClassic == "chuci" || sourceClassic == "poetry" || sourceClassic == "诗经" || sourceClassic == "楚辞" || sourceClassic == "唐诗" || sourceClassic == "古文名句" {
		charList = GetPoetryNames(gender, sourceClassic)
	} else if gender == "male" {
		charList = CommonMaleNames
	} else {
		charList = CommonFemaleNames
	}

	filteredChars := []string{}
	for _, char := range charList {
		if excludeRare && !hanzi.IsCommonChar(char) {
			continue
		}

		if len(wuxingMatch) > 0 {
			wuxing := HanziWuxingMap[char]
			if wuxing == "" {
				if h, ok := hanzi.HanziData[char]; ok {
					wuxing = h.Wuxing
				}
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

		filteredChars = append(filteredChars, char)
	}

	if len(filteredChars) == 0 {
		filteredChars = charList
	}

	shuffled := make([]string, len(filteredChars))
	copy(shuffled, filteredChars)
	ng.rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	for i := 0; i < len(shuffled) && len(names) < count; i++ {
		char := shuffled[i]
		wuxing := HanziWuxingMap[char]
		if wuxing == "" {
			if h, ok := hanzi.HanziData[char]; ok {
				wuxing = h.Wuxing
			}
		}
		if wuxing == "" {
			wuxing = "未知"
		}

		meaning := HanziMeaningMap[char]
		if meaning == "" {
			meaning = "美好寓意"
		}

		strokes := HanziStrokesMap[char]
		if strokes == 0 {
			if h, ok := hanzi.HanziData[char]; ok {
				strokes = h.Strokes
			}
		}
		if strokes == 0 {
			strokes = 10
		}

		score := ng.calculateScore(wuxing, xiyongshen)

		var poetrySrc, poetryChapter, poetrySentence string

		if sourceClassic == "shijing" || sourceClassic == "chuci" || sourceClassic == "poetry" || sourceClassic == "诗经" || sourceClassic == "楚辞" || sourceClassic == "唐诗" {
			if _, m, src, chapter, sentence, _ := GetPoetryCharInfo(char); m != "" {
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

		names = append(names, Name{
			ID:        int64(i + 1),
			Surname:   surname,
			GivenName: char,
			Pinyin:    GetPinyin(char),
			Meaning:   meaning,
			Wuxing:    wuxing,
			Strokes:   strokes,
			Gender:    gender,
			Score:     score,
			PoetrySource: poetrySrc,
			PoetryChapter: poetryChapter,
			PoetrySentence: poetrySentence,
		})
	}

	return names
}

// calculateScore 计算名字评分
func (ng *NameGenerator) calculateScore(wuxing string, xiyongshen []string) float64 {
	baseScore := 70.0

	for _, xy := range xiyongshen {
		if wuxing == xy {
			return 95.0 + ng.rand.Float64()*5
		}
	}

	return baseScore + ng.rand.Float64()*15
}

// CalculateBaZiScore 计算八字评分
func (ng *NameGenerator) CalculateBaZiScore(nameWuxing, dayWuxing string, xiyongshen []string) int {
	score := 50

	if nameWuxing == dayWuxing {
		score += 20
	}

	for _, xy := range xiyongshen {
		if nameWuxing == xy {
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

	if rel, ok := wuxingRelations[dayWuxing]; ok {
		if bonus, ok := rel[nameWuxing]; ok {
			score += bonus
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

// GetPinyin 获取拼音（简化版）
func GetPinyin(char string) string {
	pinyinMap := map[string]string{
		"伟": "wei", "强": "qiang", "磊": "lei", "军": "jun",
		"杰": "jie", "涛": "tao", "明": "ming", "超": "chao",
		"勇": "yong", "鹏": "peng", "华": "hua", "刚": "gang",
		"平": "ping", "辉": "hui", "波": "bo", "峰": "feng",
		"飞": "fei", "龙": "long", "浩": "hao", "宇": "yu",
		"晨": "chen", "逸": "yi", "睿": "rui", "哲": "zhe",
		"yuan": "yuan", "博": "bo", "昊": "hao", "然": "ran",
		"轩": "xuan", "俊": "jun", "豪": "hao", "毅": "yi",
		"文": "wen", "武": "wu", "祥": "xiang", "瑞": "rui",
		"凯": "kai", "成": "cheng", "盛": "sheng", "雄": "xiong",
		"鑫": "xin", "霖": "lin", "思": "si", "远": "yuan",
		"航": "hang", "洋": "yang", "庆": "qing",
		"芳": "fang", "兰": "lan", "梅": "mei", "琴": "qin",
		"丽": "li", "xiu": "xiu", "英": "ying", "敏": "min",
		"静": "jing", "燕": "yan", "xia": "xia", "红": "hong",
		"玉": "yu", "珍": "zhen", "juan": "juan", "莉": "li",
		"萍": "ping", "ying": "ying", "娜": "na", "xin": "xin",
		"yi": "yi", "fei": "fei", "ya": "ya", "yun": "yun",
		"xuan": "xuan", "雨": "yu", "诗": "shi", "梦": "meng",
		"yao": "yao", "琳": "lin", "ke": "ke", "wen": "wen",
		"qian": "qian", "xue": "xue", "ting": "ting", "shuang": "shuang",
		"cui": "cui", "bi": "bi", "银": "yin", "金": "jin",
		"珠": "zhu",
	}

	if p, ok := pinyinMap[char]; ok {
		return p
	}
	return "?"
}

// NameScore 名字评分结果
type NameScore struct {
	Name            string  `json:"name"`
	Score           float64 `json:"score"`
	BaziScore       float64 `json:"bazi_score"`
	NayinScore      float64 `json:"nayin_score"`
	ZodiacScore     float64 `json:"zodiac_score"`
	YijingScore     float64 `json:"yijing_score"`
	MeaningScore    float64 `json:"meaning_score"`
}

// CalculateNameScore 计算名字综合评分
func CalculateNameScore(name string, baziWuxing []string, nayin string, zodiac string, hexagramScore float64) *NameScore {
	score := &NameScore{
		Name:            name,
		Score:           70,
		BaziScore:       20,
		NayinScore:      15,
		ZodiacScore:     15,
		YijingScore:     20,
		MeaningScore:    10,
	}

	return score
}
