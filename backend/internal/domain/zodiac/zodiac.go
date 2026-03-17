package zodiac

import "fmt"

// Zodiac 生肖结构
type Zodiac struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Wuxing       string   `json:"wuxing"`
	Compatible   []string `json:"compatible"`
	Conflicting  []string `json:"conflicting"`
	AvoidChars   string   `json:"avoid_chars"`
	LuckyNumbers []int    `json:"lucky_number"`
	LuckyColor   string   `json:"lucky_color"`
	LuckyDir     string   `json:"lucky_direction"`
	GoodPianpang string  `json:"good_pianpang"`
	BadPianpang  string  `json:"bad_pianpang"`
}

// ZodiacList 12生肖数据
var ZodiacList = []Zodiac{
	{
		ID:           1,
		Name:         "鼠",
		Wuxing:       "水",
		Compatible:   []string{"牛", "龙", "猴"},
		Conflicting:  []string{"马", "兔", "羊"},
		AvoidChars:   "午、马、兔、羊",
		LuckyNumbers: []int{2, 3},
		LuckyColor:   "蓝色、金色、绿色",
		LuckyDir:     "东南、东北",
		GoodPianpang: "米、豆、鱼、虫",
		BadPianpang:  "刀、刂、火、灬",
	},
	{
		ID:           2,
		Name:         "牛",
		Wuxing:       "土",
		Compatible:   []string{"鼠", "蛇", "鸡"},
		Conflicting:  []string{"羊", "马", "狗"},
		AvoidChars:   "未、午、戌",
		LuckyNumbers: []int{1, 9},
		LuckyColor:   "黄色、咖啡色",
		LuckyDir:     "东北、东南",
		GoodPianpang: "艹、宀、田、氵",
		BadPianpang:  "石、刀、刂、勹",
	},
	{
		ID:           3,
		Name:         "虎",
		Wuxing:       "木",
		Compatible:   []string{"马", "狗", "兔"},
		Conflicting:  []string{"蛇", "猴"},
		AvoidChars:   "巳、申",
		LuckyNumbers: []int{1, 3, 4},
		LuckyColor:   "蓝色、灰色、橙色",
		LuckyDir:     "西北、东南",
		GoodPianpang: "山、王、氵、月",
		BadPianpang:  "小、忄、刂、艹",
	},
	{
		ID:           4,
		Name:         "兔",
		Wuxing:       "木",
		Compatible:   []string{"羊", "猪", "狗"},
		Conflicting:  []string{"鸡", "龙", "鼠"},
		AvoidChars:   "酉、辰、子",
		LuckyNumbers: []int{3, 4, 9},
		LuckyColor:   "红色、粉色、紫色、金色",
		LuckyDir:     "东南、西北",
		GoodPianpang: "艹、禾、木、宀",
		BadPianpang:  "石、刀、刂、血",
	},
	{
		ID:           5,
		Name:         "龙",
		Wuxing:       "土",
		Compatible:   []string{"鼠", "猴", "鸡"},
		Conflicting:  []string{"狗", "兔"},
		AvoidChars:   "戌、卯",
		LuckyNumbers: []int{1, 6, 7},
		LuckyColor:   "金色、银色、白色",
		LuckyDir:     "西北、西南",
		GoodPianpang: "氵、雨、龙、目",
		BadPianpang:  "土、艹、田、虫",
	},
	{
		ID:           6,
		Name:         "蛇",
		Wuxing:       "火",
		Compatible:   []string{"牛", "鸡"},
		Conflicting:  []string{"虎", "猴", "猪"},
		AvoidChars:   "寅、申、亥",
		LuckyNumbers: []int{2, 8, 9},
		LuckyColor:   "黑色、红色、黄色",
		LuckyDir:     "东南、西南",
		GoodPianpang: "虫、艹、木、辶",
		BadPianpang:  "石、土、刀、刂",
	},
	{
		ID:           7,
		Name:         "马",
		Wuxing:       "火",
		Compatible:   []string{"虎", "羊", "狗"},
		Conflicting:  []string{"鼠", "牛", "兔", "羊"},
		AvoidChars:   "子、丑、卯",
		LuckyNumbers: []int{2, 3, 7},
		LuckyColor:   "黄色、绿色",
		LuckyDir:     "东南、西北",
		GoodPianpang: "艹、马、氵、车",
		BadPianpang:  "石、山、田、虫",
	},
	{
		ID:           8,
		Name:         "羊",
		Wuxing:       "土",
		Compatible:   []string{"兔", "马", "猪"},
		Conflicting:  []string{"牛", "狗", "鼠"},
		AvoidChars:   "丑、子、酉",
		LuckyNumbers: []int{3, 9},
		LuckyColor:   "绿色、红色、紫色",
		LuckyDir:     "西南、东南",
		GoodPianpang: "艹、米、氵、玉",
		BadPianpang:  "石、刀、刂、血",
	},
	{
		ID:           9,
		Name:         "猴",
		Wuxing:       "金",
		Compatible:   []string{"龙", "鼠", "蛇"},
		Conflicting:  []string{"虎", "猪"},
		AvoidChars:   "寅、亥",
		LuckyNumbers: []int{1, 7, 8},
		LuckyColor:   "白色、蓝色、金色",
		LuckyDir:     "西北、东北",
		GoodPianpang: "木、亻、大、王",
		BadPianpang:  "火、灬、日、忄",
	},
	{
		ID:           10,
		Name:         "鸡",
		Wuxing:       "金",
		Compatible:   []string{"牛", "龙", "蛇"},
		Conflicting:  []string{"兔", "狗", "鼠"},
		AvoidChars:   "卯、戌、子",
		LuckyNumbers: []int{5, 7, 8},
		LuckyColor:   "金色、白色、棕色",
		LuckyDir:     "西南、东北",
		GoodPianpang: "米、虫、禾、宀",
		BadPianpang:  "石、刀、刂、艹",
	},
	{
		ID:           11,
		Name:         "狗",
		Wuxing:       "土",
		Compatible:   []string{"虎", "兔", "马"},
		Conflicting:  []string{"龙", "鸡", "羊", "牛"},
		AvoidChars:   "辰、酉、丑、未",
		LuckyNumbers: []int{3, 4, 9},
		LuckyColor:   "红色、绿色、紫色",
		LuckyDir:     "西北、西南",
		GoodPianpang: "艹、禾、米、氵",
		BadPianpang:  "石、刀、刂、田",
	},
	{
		ID:           12,
		Name:         "猪",
		Wuxing:       "水",
		Compatible:   []string{"羊", "兔", "虎"},
		Conflicting:  []string{"蛇", "猴", "猪"},
		AvoidChars:   "巳、申、亥",
		LuckyNumbers: []int{2, 5, 8},
		LuckyColor:   "黄色、灰色、棕色、金色",
		LuckyDir:     "东南、东北",
		GoodPianpang: "氵、艹、米、豆",
		BadPianpang:  "刀、刂、火、灬",
	},
}

func GetZodiacByName(name string) *Zodiac {
	for i := range ZodiacList {
		if ZodiacList[i].Name == name {
			return &ZodiacList[i]
		}
	}
	return nil
}

func GetZodiacByYear(year int) *Zodiac {
	zodiacIndex := (year - 1900) % 12
	if zodiacIndex < 0 {
		zodiacIndex += 12
	}
	return &ZodiacList[zodiacIndex]
}

func GetZodiacByID(id int) *Zodiac {
	for i := range ZodiacList {
		if ZodiacList[i].ID == id {
			return &ZodiacList[i]
		}
	}
	return nil
}

func GetAllZodiacs() []Zodiac {
	return ZodiacList
}

type ZodiacMatch struct {
	Zodiac        *Zodiac
	PianpangScore int    `json:"pianpang_score"`
	PianpangDesc  string `json:"pianpang_desc"`
}

func MatchNameWithZodiac(name string, zodiacName string) *ZodiacMatch {
	z := GetZodiacByName(zodiacName)
	if z == nil {
		return nil
	}

	score := 50
	desc := ""

	goodPianpangs := parsePianpang(z.GoodPianpang)
	badPianpangs := parsePianpang(z.BadPianpang)

	goodCount := 0
	badCount := 0

	for _, char := range name {
		for _, good := range goodPianpangs {
			if containsRune(good, char) {
				goodCount++
				break
			}
		}
		for _, bad := range badPianpangs {
			if containsRune(bad, char) {
				badCount++
				break
			}
		}
	}

	score += goodCount * 15
	score -= badCount * 10

	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}

	if goodCount > 0 && badCount == 0 {
		desc = fmt.Sprintf("名字含%d个生肖喜用偏旁，无忌用偏旁，非常吉利", goodCount)
	} else if goodCount > 0 && badCount > 0 {
		desc = fmt.Sprintf("名字含%d个喜用偏旁，%d个忌用偏旁", goodCount, badCount)
	} else if goodCount == 0 && badCount > 0 {
		desc = fmt.Sprintf("名字含%d个忌用偏旁，不利生肖运势", badCount)
	} else {
		desc = "名字与生肖偏旁无特殊关联"
	}

	return &ZodiacMatch{
		Zodiac:        z,
		PianpangScore: score,
		PianpangDesc:  desc,
	}
}

func parsePianpang(pianpangStr string) []string {
	var result []string
	var current string
	for _, r := range pianpangStr {
		if r == '、' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(r)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func containsRune(pianpang string, char rune) bool {
	for _, r := range pianpang {
		if r == char {
			return true
		}
	}
	return false
}
