// build_namestats 从 Chinese-Names-Corpus 生成完整姓名统计数据。
//
// 输入：backend/data/raw/corpus/Chinese_Names_Corpus_Gender_120W.txt
// 输出：
//   - backend/data/surname_stats.json           — 姓氏统计（含性别比例）
//   - backend/data/given_name_stats.json        — 按姓氏分组的名字统计
//   - backend/data/full_name_stats.json         — 全名统计（含性别分布）
//   - backend/data/name_gender_stats.json       — 名字性别统计
//   - backend/data/name_frequency.json          — 单字频率（复用）
//   - backend/data/name_bigram_frequency.json   — 双字组合频率（复用）
//
// 用法：go run ./cmd/build_namestats
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

// ────────────────────────── 数据结构 ──────────────────────────

type SurnameStat struct {
	Surname     string  `json:"surname"`
	Count       int     `json:"count"`
	Ratio       float64 `json:"ratio"`
	Rank        int     `json:"rank"`
	MaleRatio   float64 `json:"male_ratio"`
	FemaleRatio float64 `json:"female_ratio"`
}

type GivenNameStat struct {
	Surname      string `json:"surname"`
	GivenName    string `json:"given_name"`
	Count        int    `json:"count"`
	Rank         int    `json:"rank"`
	MaleCount    int    `json:"male_count"`
	FemaleCount  int    `json:"female_count"`
	UnknownCount int    `json:"unknown_count"`
}

type FullNameStat struct {
	FullName     string `json:"full_name"`
	Surname      string `json:"surname"`
	GivenName    string `json:"given_name"`
	Count        int    `json:"count"`
	MaleCount    int    `json:"male_count"`
	FemaleCount  int    `json:"female_count"`
	UnknownCount int    `json:"unknown_count"`
	Rank         int    `json:"rank"`
}

type NameGenderStat struct {
	Name         string  `json:"name"`
	MaleCount    int     `json:"male_count"`
	FemaleCount  int     `json:"female_count"`
	UnknownCount int     `json:"unknown_count"`
	TotalCount   int     `json:"total_count"`
	MaleRatio    float64 `json:"male_ratio"`
	FemaleRatio  float64 `json:"female_ratio"`
}

type CharFrequency struct {
	Char  string `json:"char"`
	Count int    `json:"count"`
	Rank  int    `json:"rank"`
	Tier  int    `json:"tier"`
}

type BigramFrequency struct {
	Char1 string `json:"char1"`
	Char2 string `json:"char2"`
	Count int    `json:"count"`
	Rank  int    `json:"rank"`
	Tier  int    `json:"tier"`
}

type FrequencyMeta struct {
	Source      string `json:"source"`
	TotalNames  int    `json:"total_names"`
	TotalChars  int    `json:"total_chars"`
	UniqueChars int    `json:"unique_chars"`
	BuiltAt     string `json:"built_at"`
}

type CharFrequencyOutput struct {
	Meta      FrequencyMeta    `json:"meta"`
	Frequency []*CharFrequency `json:"frequency"`
}

type BigramFrequencyOutput struct {
	Meta      FrequencyMeta      `json:"meta"`
	Frequency []*BigramFrequency `json:"frequency"`
}

// ────────────────────────── 姓氏集合 ──────────────────────────

var surnameSet = map[string]bool{
	"王": true, "李": true, "张": true, "刘": true, "陈": true,
	"杨": true, "黄": true, "赵": true, "吴": true, "周": true,
	"徐": true, "孙": true, "马": true, "胡": true, "朱": true,
	"郭": true, "何": true, "林": true, "罗": true, "高": true,
	"梁": true, "郑": true, "谢": true, "宋": true, "唐": true,
	"许": true, "邓": true, "冯": true, "韩": true, "曹": true,
	"彭": true, "曾": true, "肖": true, "田": true, "董": true,
	"潘": true, "袁": true, "蔡": true, "蒋": true, "余": true,
	"于": true, "叶": true, "程": true, "魏": true, "苏": true,
	"吕": true, "丁": true, "任": true, "沈": true, "姚": true,
	"卢": true, "傅": true, "钟": true, "姜": true, "崔": true,
	"谭": true, "廖": true, "汪": true, "范": true, "金": true,
	"方": true, "石": true, "夏": true, "熊": true, "陆": true,
	"孔": true, "白": true, "毛": true, "侯": true, "秦": true,
	"顾": true, "孟": true, "薛": true, "尹": true, "江": true,
	"汤": true, "龙": true, "黎": true, "易": true, "常": true,
	"武": true, "乔": true, "贺": true, "赖": true, "龚": true,
	"文": true, "段": true, "史": true, "邹": true, "钱": true,
	"严": true, "邱": true, "温": true, "莫": true, "颜": true,
	"万": true, "康": true, "安": true, "雷": true, "倪": true,
	"樊": true, "华": true, "萧": true, "杜": true, "戴": true,
	"洪": true, "纪": true, "贾": true, "章": true, "邢": true,
	"伍": true, "屈": true, "阮": true, "蓝": true, "闵": true,
	"季": true, "甘": true, "包": true, "关": true, "苗": true,
	"柳": true,
}

var compoundSurnames = map[string]bool{
	"欧阳": true, "司马": true, "上官": true, "诸葛": true,
	"令狐": true, "皇甫": true, "司徒": true, "端木": true,
	"慕容": true, "东方": true, "独孤": true, "南宫": true,
	"长孙": true, "宇文": true, "公孙": true, "百里": true,
	"呼延": true, "东郭": true, "南门": true, "羊舌": true,
	"微生": true, "公户": true, "公玉": true, "公仪": true,
	"梁丘": true, "公仲": true, "公上": true, "公门": true,
	"公山": true, "公坚": true, "左丘": true, "公伯": true,
	"西门": true, "公祖": true, "第五": true, "公乘": true,
	"贯丘": true, "公皙": true, "南荣": true, "东里": true,
	"东宫": true, "仲孙": true, "申屠": true, "夏侯": true,
}

// ────────────────────────── 内部统计结构 ──────────────────────────

type nameRecord struct {
	Surname   string
	GivenName string
	FullName  string
	Gender    string // 男/女/未知
}

type surnameCounter struct {
	Total   int
	Male    int
	Female  int
	Unknown int
}

type givenNameCounter struct {
	Total   int
	Male    int
	Female  int
	Unknown int
}

type fullNameCounter struct {
	Total   int
	Male    int
	Female  int
	Unknown int
}

// ────────────────────────── 主流程 ──────────────────────────

func main() {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("获取工作目录失败: %v", err)
	}
	projectRoot := findProjectRoot(wd)

	corpusPath := filepath.Join(projectRoot, "backend", "data", "raw", "corpus", "Chinese_Names_Corpus_Gender_120W.txt")
	outDir := filepath.Join(projectRoot, "backend", "data")

	fmt.Println("开始读取语料...")
	records, totalNames := readCorpus(corpusPath)
	fmt.Printf("语料读取完毕: %d 条有效人名\n", totalNames)

	fmt.Println("统计姓氏...")
	surnameStats := countSurnames(records)
	fmt.Printf("姓氏统计完毕: %d 个姓氏\n", len(surnameStats))

	fmt.Println("统计名字部分（按姓氏分组）...")
	givenNameStats := countGivenNames(records)
	fmt.Printf("名字部分统计完毕: %d 条\n", len(givenNameStats))

	fmt.Println("统计全名...")
	fullNameStats := countFullNames(records)
	fmt.Printf("全名统计完毕: %d 条\n", len(fullNameStats))

	fmt.Println("统计名字性别分布...")
	nameGenderStats := countNameGenders(records)
	fmt.Printf("名字性别统计完毕: %d 条\n", len(nameGenderStats))

	fmt.Println("统计单字频率...")
	charFreq := countCharFrequency(records)
	fmt.Printf("单字统计完毕: %d 个不重复汉字\n", len(charFreq))

	fmt.Println("统计双字组合频率...")
	bigramFreq := countBigramFrequency(records)
	fmt.Printf("双字组合统计完毕: %d 个不重复组合\n", len(bigramFreq))

	// 分配排名与分档
	assignSurnameRanks(surnameStats, totalNames)
	assignGivenNameRanks(givenNameStats)
	assignFullNameRanks(fullNameStats, totalNames)
	assignCharTiers(charFreq)
	assignBigramTiers(bigramFreq)

	meta := FrequencyMeta{
		Source:      "Chinese-Names-Corpus (wainshine), 120万人名语料",
		TotalNames:  totalNames,
		TotalChars:  sumCharCounts(charFreq),
		UniqueChars: len(charFreq),
		BuiltAt:     "2026-08-26",
	}

	// 输出姓氏统计
	writeSurnameStats(outDir, surnameStats)
	// 输出名字部分统计
	writeGivenNameStats(outDir, givenNameStats)
	// 输出全名统计
	writeFullNameStats(outDir, fullNameStats)
	// 输出名字性别统计
	writeNameGenderStats(outDir, nameGenderStats)
	// 输出单字频率
	writeCharFrequency(outDir, charFreq, meta)
	// 输出双字频率
	writeBigramFrequency(outDir, bigramFreq, meta)

	// 打印摘要
	printSummary(surnameStats, givenNameStats, fullNameStats, nameGenderStats, charFreq, bigramFreq)
}

// ────────────────────────── 语料读取 ──────────────────────────

func readCorpus(path string) ([]nameRecord, int) {
	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("打开语料文件失败: %v", err)
	}
	defer f.Close()

	var records []nameRecord
	scanner := bufio.NewScanner(f)
	lineNo := 0

	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())

		if lineNo <= 3 {
			continue
		}
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ",", 2)
		if len(parts) < 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		gender := strings.TrimSpace(parts[1])

		if !isValidChineseName(name) {
			continue
		}

		surname, given := extractSurnameAndGiven(name)
		if surname == "" || given == "" {
			continue
		}

		// 标准化性别
		genderNorm := normalizeGender(gender)

		records = append(records, nameRecord{
			Surname:   surname,
			GivenName: given,
			FullName:  name,
			Gender:    genderNorm,
		})
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("读取语料失败: %v", err)
	}

	return records, len(records)
}

func isValidChineseName(name string) bool {
	runes := []rune(name)
	if len(runes) < 2 || len(runes) > 4 {
		return false
	}
	for _, r := range runes {
		if !unicode.Is(unicode.Han, r) {
			return false
		}
	}
	if len(runes) == 2 && runes[0] == '阿' {
		return false
	}
	return true
}

func normalizeGender(gender string) string {
	switch gender {
	case "男", "M", "male", "1":
		return "男"
	case "女", "F", "female", "2":
		return "女"
	default:
		return "未知"
	}
}

func extractSurnameAndGiven(fullName string) (string, string) {
	runes := []rune(fullName)

	if len(runes) >= 3 {
		prefix2 := string(runes[:2])
		if compoundSurnames[prefix2] {
			return prefix2, string(runes[2:])
		}
	}

	if len(runes) >= 2 {
		surname1 := string(runes[0])
		if surnameSet[surname1] {
			return surname1, string(runes[1:])
		}
	}

	return "", ""
}

// ────────────────────────── 统计函数 ──────────────────────────

func countSurnames(records []nameRecord) []*SurnameStat {
	counter := make(map[string]*surnameCounter)

	for _, r := range records {
		c := counter[r.Surname]
		if c == nil {
			c = &surnameCounter{}
			counter[r.Surname] = c
		}
		c.Total++
		switch r.Gender {
		case "男":
			c.Male++
		case "女":
			c.Female++
		default:
			c.Unknown++
		}
	}

	result := make([]*SurnameStat, 0, len(counter))
	for surname, c := range counter {
		maleRatio := 0.0
		femaleRatio := 0.0
		if c.Total > 0 {
			maleRatio = float64(c.Male) / float64(c.Total)
			femaleRatio = float64(c.Female) / float64(c.Total)
		}
		result = append(result, &SurnameStat{
			Surname:     surname,
			Count:       c.Total,
			Ratio:       float64(c.Total), // 临时占位，后续 assignSurnameRanks 会计算实际比例
			MaleRatio:   maleRatio,
			FemaleRatio: femaleRatio,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Count > result[j].Count
	})

	return result
}

func countGivenNames(records []nameRecord) []*GivenNameStat {
	counter := make(map[string]*givenNameCounter) // key: surname|givenName

	for _, r := range records {
		key := r.Surname + "|" + r.GivenName
		c := counter[key]
		if c == nil {
			c = &givenNameCounter{}
			counter[key] = c
		}
		c.Total++
		switch r.Gender {
		case "男":
			c.Male++
		case "女":
			c.Female++
		default:
			c.Unknown++
		}
	}

	result := make([]*GivenNameStat, 0, len(counter))
	for key, c := range counter {
		parts := strings.Split(key, "|")
		surname := parts[0]
		given := parts[1]
		result = append(result, &GivenNameStat{
			Surname:      surname,
			GivenName:    given,
			Count:        c.Total,
			MaleCount:    c.Male,
			FemaleCount:  c.Female,
			UnknownCount: c.Unknown,
		})
	}

	// 按姓氏分组，每组内按频率排序
	sort.Slice(result, func(i, j int) bool {
		if result[i].Surname != result[j].Surname {
			return result[i].Surname < result[j].Surname
		}
		return result[i].Count > result[j].Count
	})

	return result
}

func countFullNames(records []nameRecord) []*FullNameStat {
	counter := make(map[string]*fullNameCounter)

	for _, r := range records {
		c := counter[r.FullName]
		if c == nil {
			c = &fullNameCounter{}
			counter[r.FullName] = c
		}
		c.Total++
		switch r.Gender {
		case "男":
			c.Male++
		case "女":
			c.Female++
		default:
			c.Unknown++
		}
	}

	result := make([]*FullNameStat, 0, len(counter))
	for fullName, c := range counter {
		surname, given := extractSurnameAndGiven(fullName)
		result = append(result, &FullNameStat{
			FullName:     fullName,
			Surname:      surname,
			GivenName:    given,
			Count:        c.Total,
			MaleCount:    c.Male,
			FemaleCount:  c.Female,
			UnknownCount: c.Unknown,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Count > result[j].Count
	})

	return result
}

func countNameGenders(records []nameRecord) []*NameGenderStat {
	// 统计名字部分（去掉姓氏）的性别分布
	counter := make(map[string]*fullNameCounter)

	for _, r := range records {
		c := counter[r.GivenName]
		if c == nil {
			c = &fullNameCounter{}
			counter[r.GivenName] = c
		}
		c.Total++
		switch r.Gender {
		case "男":
			c.Male++
		case "女":
			c.Female++
		default:
			c.Unknown++
		}
	}

	result := make([]*NameGenderStat, 0, len(counter))
	for name, c := range counter {
		maleRatio := 0.0
		femaleRatio := 0.0
		if c.Total > 0 {
			maleRatio = float64(c.Male) / float64(c.Total)
			femaleRatio = float64(c.Female) / float64(c.Total)
		}
		result = append(result, &NameGenderStat{
			Name:         name,
			MaleCount:    c.Male,
			FemaleCount:  c.Female,
			UnknownCount: c.Unknown,
			TotalCount:   c.Total,
			MaleRatio:    maleRatio,
			FemaleRatio:  femaleRatio,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].TotalCount > result[j].TotalCount
	})

	return result
}

func countCharFrequency(records []nameRecord) []*CharFrequency {
	freqMap := make(map[string]int)

	for _, r := range records {
		for _, ch := range []rune(r.GivenName) {
			freqMap[string(ch)]++
		}
	}

	result := make([]*CharFrequency, 0, len(freqMap))
	for ch, count := range freqMap {
		result = append(result, &CharFrequency{Char: ch, Count: count})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Count > result[j].Count
	})

	for i, f := range result {
		f.Rank = i + 1
	}

	return result
}

func countBigramFrequency(records []nameRecord) []*BigramFrequency {
	freqMap := make(map[string]int)

	for _, r := range records {
		runes := []rune(r.GivenName)
		if len(runes) < 2 {
			continue
		}
		for i := 0; i < len(runes)-1; i++ {
			bigram := string(runes[i]) + string(runes[i+1])
			freqMap[bigram]++
		}
	}

	result := make([]*BigramFrequency, 0, len(freqMap))
	for bg, count := range freqMap {
		runes := []rune(bg)
		result = append(result, &BigramFrequency{
			Char1: string(runes[0]),
			Char2: string(runes[1]),
			Count: count,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Count > result[j].Count
	})

	for i, f := range result {
		f.Rank = i + 1
	}

	return result
}

// ────────────────────────── 排名与分档 ──────────────────────────

func assignSurnameRanks(stats []*SurnameStat, totalNames int) {
	for i, s := range stats {
		s.Rank = i + 1
		if totalNames > 0 {
			s.Ratio = float64(s.Count) / float64(totalNames) * 100
		}
	}
}

func assignGivenNameRanks(stats []*GivenNameStat) {
	// 每个姓氏内部重新编号
	lastSurname := ""
	rank := 0
	for _, s := range stats {
		if s.Surname != lastSurname {
			lastSurname = s.Surname
			rank = 1
		} else {
			rank++
		}
		s.Rank = rank
	}
}

func assignFullNameRanks(stats []*FullNameStat, totalNames int) {
	for i, s := range stats {
		s.Rank = i + 1
	}
}

func assignCharTiers(freq []*CharFrequency) {
	if len(freq) == 0 {
		return
	}
	n := len(freq)
	for i, f := range freq {
		pct := float64(i+1) / float64(n) * 100
		switch {
		case pct <= 1.0:
			f.Tier = 5
		case pct <= 5.0:
			f.Tier = 4
		case pct <= 20.0:
			f.Tier = 3
		case pct <= 50.0:
			f.Tier = 2
		default:
			f.Tier = 1
		}
	}
}

func assignBigramTiers(freq []*BigramFrequency) {
	if len(freq) == 0 {
		return
	}
	n := len(freq)
	for i, f := range freq {
		pct := float64(i+1) / float64(n) * 100
		switch {
		case pct <= 1.0:
			f.Tier = 5
		case pct <= 5.0:
			f.Tier = 4
		case pct <= 20.0:
			f.Tier = 3
		case pct <= 50.0:
			f.Tier = 2
		default:
			f.Tier = 1
		}
	}
}

// ────────────────────────── 写入文件 ──────────────────────────

func writeSurnameStats(outDir string, stats []*SurnameStat) {
	data, _ := json.MarshalIndent(stats, "", "  ")
	path := filepath.Join(outDir, "surname_stats.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		log.Fatalf("写入姓氏统计失败: %v", err)
	}
	fmt.Printf("姓氏统计已写入: %s (%d 条)\n", path, len(stats))
}

func writeGivenNameStats(outDir string, stats []*GivenNameStat) {
	// 只保留每个姓氏下 Top 100 的名字，减小文件体积
	filtered := make([]*GivenNameStat, 0)
	surnameCount := make(map[string]int)
	for _, s := range stats {
		if surnameCount[s.Surname] < 100 {
			filtered = append(filtered, s)
			surnameCount[s.Surname]++
		}
	}
	data, _ := json.MarshalIndent(filtered, "", "  ")
	path := filepath.Join(outDir, "given_name_stats.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		log.Fatalf("写入名字部分统计失败: %v", err)
	}
	fmt.Printf("名字部分统计已写入: %s (%d 条, 每姓氏Top100)\n", path, len(filtered))
}

func writeFullNameStats(outDir string, stats []*FullNameStat) {
	// 只保留 Top 50000 全名，减小文件体积
	filtered := stats
	if len(stats) > 50000 {
		filtered = stats[:50000]
	}
	data, _ := json.MarshalIndent(filtered, "", "  ")
	path := filepath.Join(outDir, "full_name_stats.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		log.Fatalf("写入全名统计失败: %v", err)
	}
	fmt.Printf("全名统计已写入: %s (%d 条, Top50000)\n", path, len(filtered))
}

func writeNameGenderStats(outDir string, stats []*NameGenderStat) {
	data, _ := json.MarshalIndent(stats, "", "  ")
	path := filepath.Join(outDir, "name_gender_stats.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		log.Fatalf("写入名字性别统计失败: %v", err)
	}
	fmt.Printf("名字性别统计已写入: %s (%d 条)\n", path, len(stats))
}

func writeCharFrequency(outDir string, freq []*CharFrequency, meta FrequencyMeta) {
	output := CharFrequencyOutput{Meta: meta, Frequency: freq}
	data, _ := json.MarshalIndent(output, "", "  ")
	path := filepath.Join(outDir, "name_frequency.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		log.Fatalf("写入单字频率失败: %v", err)
	}
	fmt.Printf("单字频率已写入: %s (%d 条)\n", path, len(freq))
}

func writeBigramFrequency(outDir string, freq []*BigramFrequency, meta FrequencyMeta) {
	output := BigramFrequencyOutput{Meta: meta, Frequency: freq}
	data, _ := json.MarshalIndent(output, "", "  ")
	path := filepath.Join(outDir, "name_bigram_frequency.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		log.Fatalf("写入双字频率失败: %v", err)
	}
	fmt.Printf("双字频率已写入: %s (%d 条)\n", path, len(freq))
}

func sumCharCounts(freq []*CharFrequency) int {
	sum := 0
	for _, f := range freq {
		sum += f.Count
	}
	return sum
}

func findProjectRoot(startDir string) string {
	dir := startDir
	for {
		corpusDir := filepath.Join(dir, "backend", "data", "raw", "corpus")
		if info, err := os.Stat(corpusDir); err == nil && info.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return startDir
}

func printSummary(surnameStats []*SurnameStat, givenNameStats []*GivenNameStat, fullNameStats []*FullNameStat, nameGenderStats []*NameGenderStat, charFreq []*CharFrequency, bigramFreq []*BigramFrequency) {
	fmt.Println("\n═══════════════════════════════════════════")
	fmt.Println("  姓名统计摘要")
	fmt.Println("═══════════════════════════════════════════")

	fmt.Println("\n【Top 20 姓氏】")
	for i := 0; i < 20 && i < len(surnameStats); i++ {
		s := surnameStats[i]
		fmt.Printf("  %2d. %s  %8d人 (%5.2f%%)  男:%.1f%% 女:%.1f%%\n", i+1, s.Surname, s.Count, s.Ratio, s.MaleRatio*100, s.FemaleRatio*100)
	}

	fmt.Println("\n【Top 20 名字组合（全名）】")
	for i := 0; i < 20 && i < len(fullNameStats); i++ {
		f := fullNameStats[i]
		fmt.Printf("  %2d. %s  %5d人  男:%d 女:%d 未知:%d\n", i+1, f.FullName, f.Count, f.MaleCount, f.FemaleCount, f.UnknownCount)
	}

	fmt.Println("\n【Top 20 名字部分（去姓氏）】")
	for i := 0; i < 20 && i < len(nameGenderStats); i++ {
		g := nameGenderStats[i]
		fmt.Printf("  %2d. %s  %5d人  男:%.1f%% 女:%.1f%%\n", i+1, g.Name, g.TotalCount, g.MaleRatio*100, g.FemaleRatio*100)
	}

	fmt.Println("\n【Top 20 高频名字用字】")
	for i := 0; i < 20 && i < len(charFreq); i++ {
		f := charFreq[i]
		fmt.Printf("  %2d. %s  出现 %6d 次  Tier %d\n", i+1, f.Char, f.Count, f.Tier)
	}

	fmt.Println("\n【Top 20 高频名字组合】")
	for i := 0; i < 20 && i < len(bigramFreq); i++ {
		f := bigramFreq[i]
		fmt.Printf("  %2d. %s%s  出现 %5d 次  Tier %d\n", i+1, f.Char1, f.Char2, f.Count, f.Tier)
	}

	fmt.Println("\n═══════════════════════════════════════════")
}
