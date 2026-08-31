// build_frequency 从 Chinese-Names-Corpus 生成人名频率数据。
//
// 输入：backend/data/raw/corpus/Chinese_Names_Corpus_Gender_120W.txt
// 输出（运行时数据，供 LoadFrequencyDB 加载）：
//   - backend/data/name_frequency.json   — 单字人名频率
//   - backend/data/name_bigram_frequency.json — 双字组合频率
//
// 用法：go run ./cmd/build_frequency
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

// ────────────────────────── 数据结构 ──────────────────────────

// CharFrequency 单字在人名中的出现频率
type CharFrequency struct {
	Char  string `json:"char"`  // 汉字
	Count int    `json:"count"` // 在人名中出现次数
	Rank  int    `json:"rank"`  // 频率排名（1=最高频）
	Tier  int    `json:"tier"`  // 频率档位 1-5（5=最高频）
}

// BigramFrequency 双字组合在人名中的出现频率
type BigramFrequency struct {
	Char1 string `json:"char1"` // 第一个字
	Char2 string `json:"char2"` // 第二个字
	Count int    `json:"count"` // 组合出现次数
	Rank  int    `json:"rank"`  // 频率排名
	Tier  int    `json:"tier"`  // 频率档位 1-5
}

// FrequencyMeta 元信息，便于溯源
type FrequencyMeta struct {
	Source      string `json:"source"`       // 数据来源
	TotalNames  int    `json:"total_names"`  // 有效人名总数
	TotalChars  int    `json:"total_chars"`  // 统计的汉字总数
	UniqueChars int    `json:"unique_chars"` // 不重复汉字数
	BuiltAt     string `json:"built_at"`     // 生成时间
}

// CharFrequencyOutput 单字频率输出文件
type CharFrequencyOutput struct {
	Meta      FrequencyMeta     `json:"meta"`
	Frequency []*CharFrequency `json:"frequency"` // 按频率降序
}

// BigramFrequencyOutput 双字频率输出文件
type BigramFrequencyOutput struct {
	Meta      FrequencyMeta        `json:"meta"`
	Frequency []*BigramFrequency `json:"frequency"` // 按频率降序
}

// ────────────────────────── 姓氏集合 ──────────────────────────

// surnameSet 百家姓前100大常见姓氏（用于识别姓名边界）
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

// 复姓列表
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

// ────────────────────────── 主流程 ──────────────────────────

func main() {
	// 定位项目根目录（从 cmd/build_frequency 往上三级）
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("获取工作目录失败: %v", err)
	}
	// 向上查找包含 backend/data/corpus 的根目录
	projectRoot := findProjectRoot(wd)

	corpusPath := filepath.Join(projectRoot, "backend", "data", "raw", "corpus", "Chinese_Names_Corpus_Gender_120W.txt")
	outDir := filepath.Join(projectRoot, "backend", "data")

	// 1. 读取语料
	names, totalNames := readCorpus(corpusPath)
	fmt.Printf("语料读取完毕: %d 条有效人名\n", totalNames)

	// 2. 统计单字频率
	charFreq := countCharFrequency(names)
	fmt.Printf("单字统计完毕: %d 个不重复汉字\n", len(charFreq))

	// 3. 统计双字组合频率
	bigramFreq := countBigramFrequency(names)
	fmt.Printf("双字组合统计完毕: %d 个不重复组合\n", len(bigramFreq))

	// 4. 排名与分档
	assignCharTiers(charFreq)
	assignBigramTiers(bigramFreq)

	// 5. 计算元信息
	totalChars := 0
	for _, f := range charFreq {
		totalChars += f.Count
	}

	meta := FrequencyMeta{
		Source:      "Chinese-Names-Corpus (wainshine), 120万人名语料",
		TotalNames:  totalNames,
		TotalChars:  totalChars,
		UniqueChars: len(charFreq),
		BuiltAt:     "2026-08-26",
	}

	// 6. 输出单字频率 JSON
	charOutput := CharFrequencyOutput{
		Meta:      meta,
		Frequency: charFreq,
	}
	charJSON, _ := json.MarshalIndent(charOutput, "", "  ")
	charOutPath := filepath.Join(outDir, "name_frequency.json")
	if err := os.WriteFile(charOutPath, charJSON, 0644); err != nil {
		log.Fatalf("写入单字频率文件失败: %v", err)
	}
	fmt.Printf("单字频率已写入: %s (%d 条)\n", charOutPath, len(charFreq))

	// 7. 输出双字频率 JSON
	bigramMeta := meta
	bigramMeta.UniqueChars = len(bigramFreq)
	bigramOutput := BigramFrequencyOutput{
		Meta:      bigramMeta,
		Frequency: bigramFreq,
	}
	bigramJSON, _ := json.MarshalIndent(bigramOutput, "", "  ")
	bigramOutPath := filepath.Join(outDir, "name_bigram_frequency.json")
	if err := os.WriteFile(bigramOutPath, bigramJSON, 0644); err != nil {
		log.Fatalf("写入双字频率文件失败: %v", err)
	}
	fmt.Printf("双字频率已写入: %s (%d 条)\n", bigramOutPath, len(bigramFreq))

	// 8. 打印统计摘要
	printSummary(charFreq, bigramFreq)
}

// ────────────────────────── 语料读取 ──────────────────────────

// readCorpus 读取语料文件，返回 (名字列表, 有效总数)。
// 每行格式：姓名,性别  （如 "张伟,男"）
// 跳过：空行、表头、非中文名字、"阿"前缀昵称
func readCorpus(path string) ([]string, int) {
	f, err := os.Open(path)
	if err != nil {
		log.Fatalf("打开语料文件失败: %v", err)
	}
	defer f.Close()

	var names []string
	scanner := bufio.NewScanner(f)
	lineNo := 0

	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())

		// 跳过前3行表头
		if lineNo <= 3 {
			continue
		}
		// 跳过空行
		if line == "" {
			continue
		}

		// 解析 CSV：姓名,性别
		parts := strings.SplitN(line, ",", 2)
		if len(parts) < 1 {
			continue
		}
		name := strings.TrimSpace(parts[0])

		// 过滤：必须是2-4个中文字符
		if !isValidChineseName(name) {
			continue
		}

		names = append(names, name)
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("读取语料失败: %v", err)
	}

	return names, len(names)
}

// isValidChineseName 校验是否为有效中文人名
func isValidChineseName(name string) bool {
	runes := []rune(name)
	// 有效长度：2-4个字
	if len(runes) < 2 || len(runes) > 4 {
		return false
	}
	// 全部必须是中文字符
	for _, r := range runes {
		if !unicode.Is(unicode.Han, r) {
			return false
		}
	}
	// 排除"阿X"昵称格式（阿+单字，非正式名）
	if len(runes) == 2 && runes[0] == '阿' {
		return false
	}
	return true
}

// ────────────────────────── 提取名字部分 ──────────────────────────

// extractGivenName 从全名中提取名字部分（去掉姓氏）。
// 规则：
//   - 复姓（2字）→ 后面的字为名字
//   - 单姓（1字）→ 后面的字为名字
//   - 无法识别姓氏 → 返回空（跳过该名字）
func extractGivenName(fullName string) string {
	runes := []rune(fullName)

	// 复姓检测
	if len(runes) >= 3 {
		prefix2 := string(runes[:2])
		if compoundSurnames[prefix2] {
			return string(runes[2:])
		}
	}

	// 单姓检测
	if len(runes) >= 2 {
		surname1 := string(runes[0])
		if surnameSet[surname1] {
			return string(runes[1:])
		}
	}

	// 无法识别姓氏 → 返回空
	return ""
}

// ────────────────────────── 频率统计 ──────────────────────────

// countCharFrequency 统计名字中每个字的出现频率。
// 注意：我们统计的是**名字部分**（去掉姓氏后）的用字频率，
// 这才是起名时需要参考的"哪些字常被用来取名"。
func countCharFrequency(names []string) []*CharFrequency {
	freqMap := make(map[string]int)

	for _, name := range names {
		given := extractGivenName(name)
		if given == "" {
			continue
		}
		// 统计名字中每个字
		for _, r := range []rune(given) {
			ch := string(r)
			freqMap[ch]++
		}
	}

	// 转为切片并排序
	result := make([]*CharFrequency, 0, len(freqMap))
	for ch, count := range freqMap {
		result = append(result, &CharFrequency{
			Char:  ch,
			Count: count,
		})
	}

	// 按频率降序排列
	sort.Slice(result, func(i, j int) bool {
		return result[i].Count > result[j].Count
	})

	// 分配排名
	for i, f := range result {
		f.Rank = i + 1
	}

	return result
}

// countBigramFrequency 统计名字中相邻两字的组合频率。
// 同样只统计名字部分（去掉姓氏后）。
func countBigramFrequency(names []string) []*BigramFrequency {
	freqMap := make(map[string]int)

	for _, name := range names {
		given := extractGivenName(name)
		runes := []rune(given)
		// 至少2个字才能产生组合
		if len(runes) < 2 {
			continue
		}
		// 滑动窗口统计相邻双字
		for i := 0; i < len(runes)-1; i++ {
			bigram := string(runes[i]) + string(runes[i+1])
			freqMap[bigram]++
		}
	}

	// 转为切片并排序
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

// ────────────────────────── 档位计算 ──────────────────────────

// assignCharTiers 按频率分布分配 Tier 1-5 档位。
//
// 分档策略（基于百分位）：
//   - Tier 5: Top 1%   — 超高频字（如"子""轩""涵"）
//   - Tier 4: Top 5%   — 高频字
//   - Tier 3: Top 20%  — 中频字
//   - Tier 2: Top 50%  — 低频字
//   - Tier 1: 底部 50% — 极低频字
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

// assignBigramTiers 按频率分布分配 Tier 1-5 档位。
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

// ────────────────────────── 工具函数 ──────────────────────────

// findProjectRoot 从给定目录向上查找包含 backend/data/corpus 的项目根目录
func findProjectRoot(startDir string) string {
	dir := startDir
	for {
		corpusDir := filepath.Join(dir, "backend", "data", "raw", "corpus")
		if info, err := os.Stat(corpusDir); err == nil && info.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// 到达根目录，回退到已知位置
			break
		}
		dir = parent
	}
	// 回退：使用当前工作目录
	return startDir
}

// printSummary 打印统计摘要
func printSummary(charFreq []*CharFrequency, bigramFreq []*BigramFrequency) {
	fmt.Println("\n═══════════════════════════════════════════")
	fmt.Println("  人名频率统计摘要")
	fmt.Println("═══════════════════════════════════════════")

	// 单字分布
	tierCount := make(map[int]int)
	for _, f := range charFreq {
		tierCount[f.Tier]++
	}
	fmt.Println("\n【单字频率分布】")
	for tier := 5; tier >= 1; tier-- {
		count := tierCount[tier]
		pct := float64(count) / float64(len(charFreq)) * 100
		bar := strings.Repeat("█", int(math.Min(pct, 50)))
		fmt.Printf("  Tier %d: %5d 字 (%5.1f%%) %s\n", tier, count, pct, bar)
	}

	// Top 20 高频字
	fmt.Println("\n【Top 20 高频名字用字】")
	for i := 0; i < 20 && i < len(charFreq); i++ {
		f := charFreq[i]
		fmt.Printf("  %2d. %s  出现 %6d 次  Tier %d\n", i+1, f.Char, f.Count, f.Tier)
	}

	// 双字组合 Top 20
	fmt.Println("\n【Top 20 高频名字组合】")
	for i := 0; i < 20 && i < len(bigramFreq); i++ {
		f := bigramFreq[i]
		fmt.Printf("  %2d. %s%s  出现 %5d 次  Tier %d\n", i+1, f.Char1, f.Char2, f.Count, f.Tier)
	}

	fmt.Println("\n═══════════════════════════════════════════")

	// 辅助定位工具
	_ = math.MaxInt
	_ = sort.Strings
}
