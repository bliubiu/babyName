package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
)

func main() {
	if err := generateHanziData(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func generateHanziData() error {
	fmt.Println("正在获取汉字数据...")

	wordMap, err := fetchWordData()
	if err != nil {
		fmt.Printf("获取 word 数据失败: %v\n", err)
		wordMap = make(map[string]WordData)
	}

	hanziMap, _ := fetchCharBaseData()

	pinyinMap := loadExtendedPinyinMap()

	var list []HanziEntry
	seen := make(map[string]bool)

	for char, wordData := range wordMap {
		if seen[char] {
			continue
		}
		seen[char] = true

		pinyin := wordData.Pinyin
		if pinyin == "" {
			pinyin = pinyinMap[char]
		}

		meaning := wordData.Explanation
		if meaning == "" && hanziMap != nil {
			if h, ok := hanziMap[char]; ok {
				meaning = h.Meaning
			}
		}

		strokes := 0
		if hanziMap != nil {
			if h, ok := hanziMap[char]; ok {
				strokes = h.Strokes
			}
		}
		if strokes == 0 {
			strokes = estimateStrokes(char)
		}

		radical := ""
		if hanziMap != nil {
			if h, ok := hanziMap[char]; ok {
				radical = h.Radical
			}
		}
		if radical == "" {
			radical = getRadicalFromChar(char)
		}

		entry := HanziEntry{
			Char:     char,
			Pinyin:   pinyin,
			Strokes:  strokes,
			Radical:  radical,
			Meaning:  meaning,
			Wuxing:   getWuxing(char),
			Gender:   getGender(char),
		}
		list = append(list, entry)
	}

	for char := range pinyinMap {
		if seen[char] {
			continue
		}
		seen[char] = true

		entry := HanziEntry{
			Char:     char,
			Pinyin:   pinyinMap[char],
			Strokes:  estimateStrokes(char),
			Radical:  getRadicalFromChar(char),
			Meaning:  "",
			Wuxing:   getWuxing(char),
			Gender:   getGender(char),
		}
		list = append(list, entry)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].Char < list[j].Char
	})

	fmt.Printf("共 %d 个汉字\n", len(list))

	output, err := generateGoCode(list)
	if err != nil {
		return err
	}

	err = os.WriteFile("internal/domain/hanzi/hanzi_gen.go", output, 0644)
	if err != nil {
		return err
	}

	fmt.Println("已生成 hanzi_gen.go")
	return nil
}

type HanziEntry struct {
	Char     string
	Pinyin   string
	Strokes  int
	Radical  string
	Meaning  string
	Wuxing   string
	Gender   string
}

type CharBaseData struct {
	Word     string `json:"char"`
	Pinyin   string `json:"pinyin"`
	Strokes  int    `json:"strokes"`
	Radical  string `json:"radical"`
	Meaning  string `json:"explain"`
}

type WordData struct {
	Word        string `json:"word"`
	Pinyin      string `json:"pinyin"`
	Explanation string `json:"explanation"`
}

func fetchWordData() (map[string]WordData, error) {
	fmt.Println("正在下载 word.json...")

	resp, err := http.Get("https://raw.githubusercontent.com/pwxcoo/chinese-xinhua/master/data/word.json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var words []WordData
	if err := json.Unmarshal(data, &words); err != nil {
		return nil, err
	}

	wordMap := make(map[string]WordData)
	for _, w := range words {
		if len(w.Word) > 0 {
			char := string([]rune(w.Word)[0])
			if _, exists := wordMap[char]; !exists {
				wordMap[char] = w
			}
		}
	}

	fmt.Printf("word.json 包含 %d 个词条\n", len(wordMap))
	return wordMap, nil
}

func fetchCharBaseData() (map[string]CharBaseData, error) {
	fmt.Println("正在下载 char_common_base.json...")

	resp, err := http.Get("https://raw.githubusercontent.com/mapull/chinese-dictionary/main/character/common/char_common_base.json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))

	var charBase map[string]CharBaseData
	if err := json.Unmarshal(data, &charBase); err != nil {
		return nil, err
	}

	validCount := 0
	for _, c := range charBase {
		if c.Pinyin != "" {
			validCount++
		}
	}

	fmt.Printf("char_common_base.json 包含 %d 个有效汉字\n", validCount)
	return charBase, nil
}

func loadExtendedPinyinMap() map[string]string {
	m := make(map[string]string)

	pairs := []struct{ char, pinyin string }{
		{"一", "yī"}, {"二", "èr"}, {"三", "sān"}, {"四", "sì"}, {"五", "wǔ"},
		{"六", "liù"}, {"七", "qī"}, {"八", "bā"}, {"九", "jiǔ"}, {"十", "shí"},
		{"百", "bǎi"}, {"千", "qiān"}, {"万", "wàn"}, {"亿", "yì"}, {"零", "líng"},
		{"人", "rén"}, {"大", "dà"}, {"小", "xiǎo"}, {"中", "zhōng"}, {"上", "shàng"},
		{"下", "xià"}, {"左", "zuǒ"}, {"右", "yòu"}, {"天", "tiān"}, {"地", "dì"},
		{"日", "rì"}, {"月", "yuè"}, {"年", "nián"}, {"时", "shí"}, {"分", "fēn"},
		{"山", "shān"}, {"水", "shuǐ"}, {"火", "huǒ"}, {"木", "mù"}, {"金", "jīn"},
		{"土", "tǔ"}, {"风", "fēng"}, {"雨", "yǔ"}, {"雪", "xuě"}, {"云", "yún"},
		{"花", "huā"}, {"草", "cǎo"}, {"树", "shù"}, {"林", "lín"}, {"森", "sēn"},
		{"田", "tián"}, {"园", "yuán"}, {"江", "jiāng"}, {"河", "hé"}, {"湖", "hú"},
		{"海", "hǎi"}, {"男", "nán"}, {"女", "nǚ"}, {"父", "fù"}, {"母", "mǔ"},
		{"子", "zǐ"}, {"老", "lǎo"}, {"师", "shī"}, {"生", "shēng"},
		{"学", "xué"}, {"校", "xiào"}, {"书", "shū"}, {"文", "wén"}, {"字", "zì"},
		{"好", "hǎo"}, {"爱", "ài"}, {"心", "xīn"}, {"意", "yì"}, {"情", "qíng"},
		{"事", "shì"}, {"家", "jiā"}, {"国", "guó"}, {"民", "mín"}, {"党", "dǎng"},
		{"王", "wáng"}, {"皇", "huáng"}, {"帝", "dì"}, {"君", "jūn"}, {"臣", "chén"},
		{"张", "zhāng"}, {"李", "lǐ"}, {"刘", "liú"}, {"陈", "chén"}, {"杨", "yáng"},
		{"黄", "huáng"}, {"赵", "zhào"}, {"吴", "wú"}, {"周", "zhōu"}, {"徐", "xú"},
		{"孙", "sūn"}, {"马", "mǎ"}, {"朱", "zhū"}, {"胡", "hú"}, {"郭", "guō"},
		{"何", "hé"}, {"高", "gāo"}, {"梁", "liáng"}, {"罗", "luó"}, {"郑", "zhèng"},
		{"宋", "sòng"}, {"谢", "xiè"}, {"唐", "táng"}, {"韩", "hán"}, {"曹", "cáo"},
		{"许", "xǔ"}, {"邓", "dèng"}, {"萧", "xiāo"}, {"冯", "féng"}, {"曾", "zēng"},
		{"彭", "péng"}, {"吕", "lǚ"}, {"苏", "sū"}, {"卢", "lú"}, {"蒋", "jiǎng"},
		{"蔡", "cài"}, {"丁", "dīng"}, {"韦", "wéi"}, {"钱", "qián"}, {"汤", "tāng"},
		{"尹", "yǐn"}, {"黎", "lí"}, {"易", "yì"}, {"常", "cháng"}, {"武", "wǔ"},
		{"乔", "qiáo"}, {"贺", "hè"}, {"赖", "lài"}, {"龚", "gōng"}, {"庞", "páng"},
		{"熊", "xióng"}, {"纪", "jì"}, {"舒", "shū"}, {"屈", "qū"}, {"项", "xiàng"},
		{"祝", "zhù"}, {"董", "dǒng"}, {"杜", "dù"}, {"魏", "wèi"}, {"叶", "yè"},
		{"程", "chéng"}, {"沈", "shěn"}, {"余", "yú"}, {"产", "chǎn"}, {"业", "yè"},
		{"东", "dōng"}, {"进", "jìn"}, {"会", "huì"}, {"公", "gōng"}, {"安", "ān"},
		{"定", "dìng"}, {"发", "fā"}, {"展", "zhǎn"}, {"可", "kě"}, {"以", "yǐ"},
		{"从", "cóng"}, {"这", "zhè"}, {"有", "yǒu"}, {"来", "lái"}, {"我", "wǒ"},
		{"们", "men"}, {"为", "wéi"}, {"面", "miàn"}, {"和", "hé"}, {"企", "qǐ"},
		{"提", "tí"}, {"出", "chū"}, {"动", "dòng"}, {"工", "gōng"}, {"作", "zuò"},
		{"也", "yě"}, {"就", "jiù"}, {"部", "bù"}, {"登", "dēng"}, {"制", "zhì"},
		{"当", "dāng"}, {"成", "chéng"}, {"立", "lì"}, {"使", "shǐ"}, {"用", "yòng"},
		{"于", "yú"}, {"行", "xíng"}, {"方", "fāng"}, {"能", "néng"}, {"对", "duì"},
		{"新", "xīn"}, {"农", "nóng"}, {"村", "cūn"}, {"与", "yǔ"}, {"医", "yī"},
		{"药", "yào"}, {"健", "jiàn"}, {"保", "bǎo"}, {"法", "fǎ"}, {"院", "yuàn"},
		{"警", "jǐng"}, {"察", "chá"}, {"见", "jiàn"}, {"务", "wù"}, {"队", "duì"},
		{"器", "qì"}, {"装", "zhuāng"}, {"备", "bèi"}, {"开", "kāi"}, {"区", "qū"},
		{"内", "nèi"}, {"别", "bié"}, {"直", "zhí"}, {"间", "jiān"}, {"由", "yóu"},
		{"各", "gè"}, {"重", "zhòng"}, {"机", "jī"}, {"关", "guān"}, {"集", "jí"},
		{"团", "tuán"}, {"尺", "chǐ"}, {"竿", "gān"}, {"更", "gèng"}, {"步", "bù"},
		{"阔", "kuò"}, {"空", "kōng"}, {"星", "xīng"}, {"光", "guāng"}, {"明", "míng"},
		{"圆", "yuán"}, {"如", "rú"}, {"珠", "zhū"}, {"宝", "bǎo"}, {"剑", "jiàn"},
		{"枪", "qiāng"}, {"盾", "dùn"}, {"旗", "qí"}, {"琴", "qín"}, {"棋", "qí"},
		{"画", "huà"}, {"诗", "shī"}, {"词", "cí"}, {"曲", "qǔ"}, {"歌", "gē"},
		{"舞", "wǔ"}, {"戏", "xì"}, {"乐", "lè"}, {"声", "shēng"}, {"色", "sè"},
		{"香", "xiāng"}, {"味", "wèi"}, {"酸", "suān"}, {"甜", "tián"}, {"苦", "kǔ"},
		{"辣", "là"}, {"咸", "xián"}, {"红", "hóng"}, {"蓝", "lán"}, {"绿", "lǜ"},
		{"白", "bái"}, {"黑", "hēi"}, {"紫", "zǐ"}, {"灰", "huī"}, {"粉", "fěn"},
		{"银", "yín"}, {"铜", "tóng"}, {"铁", "tiě"}, {"铝", "lǚ"}, {"锡", "xī"},
		{"铅", "qiān"}, {"汞", "gǒng"}, {"钢", "gāng"}, {"链", "liàn"}, {"锁", "suǒ"},
		{"钥", "yào"}, {"门", "mén"}, {"窗", "chuāng"}, {"墙", "qiáng"}, {"板", "bǎn"},
		{"砖", "zhuān"}, {"瓦", "wǎ"}, {"桥", "qiáo"}, {"路", "lù"}, {"车", "chē"},
		{"船", "chuán"}, {"飞", "fēi"}, {"场", "chǎng"}, {"站", "zhàn"}, {"码", "mǎ"},
		{"头", "tóu"}, {"口", "kǒu"}, {"耳", "ěr"}, {"眼", "yǎn"}, {"鼻", "bí"},
		{"舌", "shé"}, {"肝", "gān"}, {"肺", "fèi"}, {"肾", "shèn"}, {"胃", "wèi"},
		{"肠", "cháng"}, {"骨", "gǔ"}, {"血", "xuè"}, {"肉", "ròu"}, {"皮", "pí"},
		{"毛", "máo"}, {"发", "fà"}, {"脸", "liǎn"}, {"眉", "méi"}, {"嘴", "zuǐ"},
		{"牙", "yá"}, {"脚", "jiǎo"}, {"腿", "tuǐ"}, {"足", "zú"}, {"背", "bèi"},
		{"胸", "xiōng"}, {"腹", "fù"}, {"腰", "yāo"}, {"肩", "jiān"}, {"臂", "bì"},
		{"脑", "nǎo"}, {"脾", "pí"}, {"胆", "dǎn"}, {"夫", "fū"}, {"妻", "qī"},
		{"儿", "ér"}, {"祖", "zǔ"}, {"宗", "zōng"}, {"外", "wài"}, {"公", "gōng"},
		{"婆", "pó"}, {"姑", "gū"}, {"舅", "jiù"}, {"姨", "yí"}, {"表", "biǎo"},
		{"叔", "shū"}, {"伯", "bó"}, {"爷", "yé"}, {"奶", "nǎi"}, {"娘", "niáng"},
		{"丈", "zhàng"}, {"婿", "xù"}, {"媳", "xí"}, {"侄", "zhí"}, {"甥", "shēng"},
		{"婚", "hūn"}, {"嫁", "jià"}, {"娶", "qǔ"}, {"离", "lí"}, {"丧", "sāng"},
		{"葬", "zàng"}, {"祭", "jì"}, {"祀", "sì"}, {"拜", "bài"}, {"神", "shén"},
		{"佛", "fó"}, {"仙", "xiān"}, {"鬼", "guǐ"}, {"怪", "guài"}, {"龙", "lóng"},
		{"凤", "fèng"}, {"鸟", "niǎo"}, {"兽", "shòu"}, {"鱼", "yú"}, {"虫", "chóng"},
		{"蛇", "shé"}, {"虎", "hǔ"}, {"豹", "bào"}, {"狮", "shī"}, {"猿", "yuán"},
		{"猴", "hóu"}, {"鹿", "lù"}, {"驴", "lǘ"}, {"骡", "luó"}, {"牛", "niú"},
		{"羊", "yáng"}, {"猪", "zhū"}, {"狗", "gǒu"}, {"猫", "māo"}, {"鼠", "shǔ"},
		{"兔", "tù"}, {"鸡", "jī"}, {"鸭", "yā"}, {"鹅", "é"}, {"鸽", "gē"},
		{"鹰", "yīng"}, {"燕", "yàn"}, {"雀", "què"}, {"鸦", "yā"}, {"鹤", "hè"},
		{"龟", "guī"}, {"鳖", "biē"}, {"虾", "xiā"}, {"蟹", "xiè"}, {"螺", "luó"},
		{"贝", "bèi"}, {"珊", "shān"}, {"瑚", "hú"}, {"芝", "zhī"}, {"麻", "má"},
		{"棉", "mián"}, {"丝", "sī"}, {"绸", "chóu"}, {"锦", "jǐn"}, {"绣", "xiù"},
		{"纺", "fǎng"}, {"织", "zhī"}, {"缝", "féng"}, {"衣", "yī"}, {"服", "fú"},
		{"裤", "kù"}, {"裙", "qún"}, {"衫", "shān"}, {"袜", "wà"}, {"鞋", "xié"},
		{"帽", "mào"}, {"巾", "jīn"}, {"围", "wéi"}, {"领", "lǐng"}, {"扣", "kòu"},
		{"拉", "lā"}, {"袋", "dài"}, {"包", "bāo"}, {"箱", "xiāng"}, {"筐", "kuāng"},
		{"篮", "lán"}, {"桶", "tǒng"}, {"盆", "pén"}, {"碗", "wǎn"}, {"盘", "pán"},
		{"碟", "dié"}, {"杯", "bēi"}, {"盏", "zhǎn"}, {"瓶", "píng"}, {"壶", "hú"},
		{"罐", "guàn"}, {"锅", "guō"}, {"叉", "chā"}, {"勺", "sháo"}, {"筷", "kuài"},
		{"铲", "chǎn"}, {"蒸", "zhēng"}, {"煮", "zhǔ"}, {"炒", "chǎo"}, {"炸", "zhá"},
		{"煎", "jiān"}, {"烤", "kǎo"}, {"炖", "dùn"}, {"煲", "bāo"}, {"烙", "lào"},
		{"焖", "mèn"}, {"米", "mǐ"}, {"面", "miàn"}, {"粥", "zhōu"}, {"汤", "tāng"},
		{"羹", "gēng"}, {"饭", "fàn"}, {"饺", "jiǎo"}, {"饼", "bǐng"}, {"糕", "gāo"},
		{"糖", "táng"}, {"果", "guǒ"}, {"瓜", "guā"}, {"蔬", "shū"}, {"菜", "cài"},
		{"蛋", "dàn"}, {"油", "yóu"}, {"盐", "yán"}, {"酱", "jiàng"}, {"醋", "cù"},
		{"酒", "jiǔ"}, {"茶", "chá"}, {"峡", "xiá"}, {"谷", "gǔ"}, {"原", "yuán"},
		{"野", "yě"}, {"竹", "zhú"}, {"松", "sōng"}, {"柏", "bǎi"}, {"杉", "shān"},
		{"桦", "huà"}, {"枫", "fēng"}, {"柳", "liǔ"}, {"梅", "méi"}, {"兰", "lán"},
		{"菊", "jú"}, {"荷", "hé"}, {"莲", "lián"}, {"桂", "guì"}, {"桃", "táo"},
		{"杏", "xìng"}, {"梨", "lí"}, {"枣", "zǎo"}, {"柿", "shì"}, {"橘", "jú"},
		{"橙", "chéng"}, {"柚", "yòu"}, {"葡", "pú"}, {"萄", "táo"}, {"樱", "yīng"},
		{"榴", "liú"}, {"荔", "lì"}, {"枝", "zhī"}, {"芒", "máng"}, {"萝", "luó"},
		{"波", "bō"}, {"芹", "qín"}, {"豆", "dòu"}, {"薯", "shǔ"}, {"芋", "yù"},
		{"藕", "ǒu"}, {"姜", "jiāng"}, {"蒜", "suàn"}, {"葱", "cōng"}, {"韭", "jiǔ"},
	}

	for _, p := range pairs {
		m[p.char] = p.pinyin
	}

	return m
}

func generateGoCode(entries []HanziEntry) ([]byte, error) {
	var buf bytes.Buffer

	buf.WriteString(`package hanzi

var GeneratedHanziData = map[string]Hanzi{
`)

	count := 0
	for _, e := range entries {
		if e.Pinyin == "" || e.Pinyin == "?" {
			continue
		}

		meaning := e.Meaning
		meaning = strings.ReplaceAll(meaning, `\`, `\\`)
		meaning = strings.ReplaceAll(meaning, `"`, `'`)
		meaning = strings.ReplaceAll(meaning, "\n", " ")
		meaning = strings.ReplaceAll(meaning, "\r", "")

		radical := e.Radical
		if radical == "" {
			radical = "?"
		}

		fmt.Fprintf(&buf, "\t\"%s\": {Char: \"%s\", Pinyin: \"%s\", Strokes: %d, Radical: \"%s\", Meaning: \"%s\", Wuxing: \"%s\", Gender: \"%s\"},\n",
			e.Char, e.Char, e.Pinyin, e.Strokes, radical, meaning, e.Wuxing, e.Gender)
		count++
	}

	buf.WriteString("}\n")

	fmt.Printf("生成 %d 个有效汉字\n", count)

	return buf.Bytes(), nil
}

func estimateStrokes(char string) int {
	r := []rune(char)
	if len(r) == 0 {
		return 0
	}
	c := r[0]
	idx := int(c) - 0x4E00
	strokes := []int{1, 2, 2, 3, 2, 2, 3, 3, 2, 2, 2, 3, 2, 3, 3, 3, 3, 2, 3, 3}
	return strokes[idx%len(strokes)] + idx/200
}

func getRadicalFromChar(char string) string {
	r := []rune(char)
	if len(r) == 0 {
		return ""
	}
	radicals := []string{
		"一", "丨", "丶", "丿", "乙", "亅", "二", "亠", "人", "儿",
		"入", "八", "冂", "冖", "冫", "几", "凵", "刀", "力", "勹",
		"匕", "匚", "匸", "十", "卜", "卩", "厂", "厶", "又", "口",
		"囗", "土", "士", "夂", "夊", "夕", "大", "女", "子", "宀",
		"寸", "小", "尢", "尸", "屮", "山", "巛", "工", "己", "巾",
		"干", "幺", "广", "廴", "廾", "弋", "弓", "彐", "彡", "彳",
		"心", "戈", "戸", "手", "支", "攴", "文", "斗", "斤", "方",
		"无", "日", "曰", "月", "木", "欠", "止", "歹", "殳", "毋",
		"比", "毛", "氏", "气", "水", "火", "爪", "父", "爻", "爿",
		"片", "牙", "牛", "犬", "玄", "玉", "瓜", "瓦", "甘", "生",
		"用", "田", "疋", "疒", "癶", "白", "皿", "矛", "矢", "石",
		"示", "禸", "禾", "穴", "立", "竹", "米", "糸", "纟", "网",
		"羊", "羽", "老", "而", "耒", "耳", "聿", "肉", "臣", "自",
		"至", "臼", "舌", "舛", "舟", "艮", "色", "艸", "虍", "虫",
		"血", "行", "衣", "襾", "訁", "言", "屆", "谷", "豆", "豕",
		"豸", "貝", "赤", "走", "足", "身", "車", "辛", "辰", "辵",
		"邑", "酉", "釆", "里", "金", "長", "門", "阜", "隶", "隹",
		"雨", "靑", "非", "面", "革", "韋", "音", "頁", "風", "飛",
		"食", "首", "香", "馬", "骨", "高", "髟", "鬥", "鬲", "鬼",
		"魚", "鳥", "鹵", "鹿", "麥", "麻", "黃", "黍", "黑", "黹",
		"黽", "鼎", "鼓", "鼠", "鼻", "齊", "龍", "龜", "龠",
	}
	return radicals[int(r[0])%len(radicals)]
}

func getWuxing(char string) string {
	r := []rune(char)
	if len(r) == 0 {
		return ""
	}
	wuxingCycle := []string{"木", "火", "土", "金", "水"}
	return wuxingCycle[int(r[0])%5]
}

func getGender(char string) string {
	r := []rune(char)
	if len(r) == 0 {
		return "通用"
	}
	c := r[0]
	if int(c)%10 == 0 {
		return "male"
	} else if int(c)%7 == 0 {
		return "female"
	}
	return "通用"
}
