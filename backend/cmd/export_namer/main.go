// export_namer 生成 namer.json — 基于《通用规范汉字表》8105字的统一起名用字数据
//
// 数据来源：
//   - gsc_pinyin.csv （汉字、拼音、部首、笔画、五行）
//   - hanzi.json    （释义补充）
//   - word.json     （释义二次补充）
//
// 输出：
//   - namer.json    （结构化汉字数据，包含五行自动校正、起名分类标注）
//
// 用法：
//   go run cmd/export_namer/main.go -data ../../data -out ../../data/namer.json
package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"name/internal/domain/hanzi"
)

// NamerChar 统一汉字数据条目
type NamerChar struct {
	Char             string   `json:"char"`
	Pinyin           string   `json:"pinyin"`
	Strokes          int      `json:"strokes"`
	Radical          string   `json:"radical"`
	Meaning          string   `json:"meaning,omitempty"`
	Wuxing           string   `json:"wuxing"`
	Level            int      `json:"level"`            // 1=一级常用字, 2=二级通用字, 3=三级专用字
	Gender           string   `json:"gender,omitempty"` // male/female/neutral
	NamingCategories []string `json:"namingCategories,omitempty"`

	// P0 — 安全过滤
	IsNegative bool `json:"isNegative,omitempty"` // 是否消极含义
	IsRare     bool `json:"isRare,omitempty"`     // 是否生僻字

	// P1 — 评分 / 音韵
	NamePenalty int `json:"namePenalty,omitempty"` // 起名扣分（0-100）
	Tone        int `json:"tone,omitempty"`        // 声调（1-4）

	// P2 — 筛选增强
	StyleTags     []string `json:"styleTags,omitempty"`     // 风格标签
	UsageLevel    int      `json:"usageLevel,omitempty"`    // 常用等级（1-5）
	PositiveScore int      `json:"positiveScore,omitempty"` // 寓意评分（0-100）

	// 自动标记
	IsPolyphonic bool `json:"isPolyphonic,omitempty"` // 是否多音字（从 CSV 多拼音自动判定）
}

// NamerData 顶层容器
type NamerData struct {
	Version     string      `json:"version"`
	Generated   string      `json:"generated"`
	Description string      `json:"description"`
	TotalChars  int         `json:"totalChars"`
	Chars       []NamerChar `json:"chars"`
	CharGroups  []standardCharGroup `json:"charGroups,omitempty"` // 精选起名用字分偏旁分组
}

// standardCharGroup 对应 standard_chars.json 的分组结构
type standardCharGroup struct {
	Radical string   `json:"radical"`
	Name    string   `json:"name"`
	Meaning string   `json:"meaning"`
	Chars   []string `json:"chars"`
}

// hanziJSON 对应 hanzi.json 中每行结构
type hanziJSON struct {
	Char    string `json:"char"`
	Pinyin  string `json:"pinyin"`
	Strokes int    `json:"strokes"`
	Radical string `json:"radical"`
	Meaning string `json:"meaning"`
	Wuxing  string `json:"wuxing"`
	Gender  string `json:"gender"`

	// P0 — 安全过滤
	IsNegative bool `json:"isNegative"`
	IsRare     bool `json:"isRare"`

	// P1 — 评分 / 音韵
	NamePenalty int `json:"namePenalty"`
	Tone        int `json:"tone"`

	// P2 — 筛选增强
	StyleTags     []string `json:"styleTags,omitempty"`
	UsageLevel    int      `json:"usageLevel"`
	PositiveScore int      `json:"positiveScore"`
}

// wordJSON 对应 word.json 中每行结构（在线汉语字典）
type wordJSON struct {
	Word        string `json:"word"`
	Explanation string `json:"explanation"`
	Pinyin      string `json:"pinyin"`
	Radicals    string `json:"radicals"`
}

func main() {
	dataDir := flag.String("data", "../../data", "数据目录（含 hanzi.json, gsc_pinyin.csv）")
	outFile := flag.String("out", "../../data/namer.json", "输出文件路径")
	flag.Parse()

	absData, _ := filepath.Abs(*dataDir)
	fmt.Printf("📂 数据目录: %s\n", absData)

	// 生成原料位于 data/raw/ 子目录（与运行时 namer.json 隔离）
	rawDir := filepath.Join(absData, "raw")

	// 1. 加载 gsc_pinyin.csv（8105 字主数据源）
	csvPath := filepath.Join(rawDir, "gsc_pinyin.csv")
	records, err := loadCSV(csvPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ CSV 加载失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ CSV 加载: %d 条记录\n", len(records))

	// 2. 加载 hanzi.json（释义补充）
	hzPath := filepath.Join(rawDir, "hanzi.json")
	hzMap := loadHanziMap(hzPath)
	fmt.Printf("✅ hanzi.json 加载: %d 字（命中 %d/8105）\n", len(hzMap), countOverlap(records, hzMap))

	// 3. 加载 word.json（在线汉语字典，真实释义来源）
	//    word.json 同时被运行时 LoadWordData 使用，故保留在运行时 data/ 根目录。
	wdPath := filepath.Join(absData, "word.json")
	wdMap := loadWordMap(wdPath)
	fmt.Printf("✅ word.json 加载: %d 字（命中 %d/8105）\n", len(wdMap), countOverlapStr(records, wdMap))

	// 4. 构建统一数据
	chars := buildChars(records, hzMap, wdMap)

	// 5. 加载 standard_chars.json（精选起名用字分偏旁分组），并作为
	//    namer.json 顶层 charGroups 合并写入，实现单一文件真源。
	scPath := filepath.Join(rawDir, "standard_chars.json")
	charGroups := loadCharGroups(scPath)
	fmt.Printf("✅ standard_chars.json 加载: %d 个偏旁分组\n", len(charGroups))

	// 6. 输出 namer.json
	data := NamerData{
		Version:     "1.0",
		Generated:   "2026-07-22",
		Description: "起名统一汉字数据 - 《通用规范汉字表》8105字 + 五行校正 + 起名分类标注 + 精选偏旁分组",
		TotalChars:  len(chars),
		Chars:       chars,
		CharGroups:  charGroups,
	}

	// 紧凑输出（用于生产加载）
	out, err := json.Marshal(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ JSON 序列化失败: %v\n", err)
		os.Exit(1)
	}

	absOut, _ := filepath.Abs(*outFile)
	if err := os.WriteFile(absOut, out, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "❌ 写入失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ namer.json 已生成: %s（%d 字, %.1f KB）\n", absOut, len(chars), float64(len(out))/1024)

	// 5. 统计
	printStats(chars)
}

// loadCSV 加载 gsc_pinyin.csv
func loadCSV(path string) ([][]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("CSV 数据不足")
	}
	return records[1:], nil // 跳过标题行
}

// loadCharGroups 加载 standard_chars.json（精选起名用字分组）
func loadCharGroups(path string) []standardCharGroup {
	f, err := os.Open(path)
	if err != nil {
		fmt.Printf("⚠️  standard_chars.json 加载失败: %v（分组将为空）\n", err)
		return nil
	}
	defer f.Close()

	var groups []standardCharGroup
	if err := json.NewDecoder(f).Decode(&groups); err != nil {
		fmt.Printf("⚠️  standard_chars.json 解析失败: %v（分组将为空）\n", err)
		return nil
	}
	return groups
}

// loadHanziMap 加载 hanzi.json 为 char→entry 的 map
func loadHanziMap(path string) map[string]hanziJSON {
	f, err := os.Open(path)
	if err != nil {
		fmt.Printf("⚠️  hanzi.json 加载失败: %v（释义将为空）\n", err)
		return nil
	}
	defer f.Close()

	var list []hanziJSON
	if err := json.NewDecoder(f).Decode(&list); err != nil {
		fmt.Printf("⚠️  hanzi.json 解析失败: %v（释义将为空）\n", err)
		return nil
	}

	m := make(map[string]hanziJSON, len(list))
	for _, h := range list {
		m[h.Char] = h
	}
	return m
}

func countOverlap(records [][]string, hzMap map[string]hanziJSON) int {
	return countOverlapGeneric(records, len(hzMap), func(ch string) bool {
		_, ok := hzMap[ch]
		return ok
	})
}

func countOverlapStr(records [][]string, m map[string]string) int {
	return countOverlapGeneric(records, len(m), func(ch string) bool {
		_, ok := m[ch]
		return ok
	})
}

func countOverlapGeneric(records [][]string, _ int, exists func(string) bool) int {
	n := 0
	for _, r := range records {
		if len(r) < 2 {
			continue
		}
		if exists(r[1]) {
			n++
		}
	}
	return n
}

// loadWordMap 加载 word.json（在线汉语字典）为 char→explanation 映射
func loadWordMap(path string) map[string]string {
	f, err := os.Open(path)
	if err != nil {
		fmt.Printf("⚠️  word.json 加载失败: %v（将不使用）\n", err)
		return nil
	}
	defer f.Close()

	var list []wordJSON
	if err := json.NewDecoder(f).Decode(&list); err != nil {
		fmt.Printf("⚠️  word.json 解析失败: %v（将不使用）\n", err)
		return nil
	}

	m := make(map[string]string, len(list))
	for _, w := range list {
		exp := strings.TrimSpace(w.Explanation)
		if exp == "" || w.Word == "" {
			continue
		}
		// 提取简洁释义：取第一个自然段或前 200 字
		exp = extractConciseMeaning(exp)
		if len(exp) <= 20 {
			continue
		}
		// 以首字为 key（word.json 中多字词取首字）
		firstChar := string([]rune(w.Word)[0])
		if _, exists := m[firstChar]; !exists {
			m[firstChar] = exp
		}
	}
	return m
}

// extractConciseMeaning 从 word.json 的详细释义中提取简洁可读的释义
// 策略：取第一句核心定义，限制在 300 字以内
func extractConciseMeaning(exp string) string {
	// 移除头部空格/换行
	exp = strings.TrimSpace(exp)

	// 标记词列表（词典格式标记，非释义本身），优先跳过取之后的实际内容
	skipMarkers := []string{"同本义", "又如", "常用"}
	// 先检查是否以标记词开头（或内容极短），取标记词之后的实际释义
	markerPos := len(exp)
	for _, marker := range skipMarkers {
		idx := strings.Index(exp, marker)
		if idx >= 0 && idx < markerPos {
			markerPos = idx
		}
	}
	// 如果标记词出现在开头 20 字以内，说明前半段只是标题，取标记词之后的内容
	if markerPos < 20 {
		after := exp[markerPos:]
		// 跳过标记词本身和后续空白
		for _, marker := range skipMarkers {
			after = strings.TrimPrefix(after, marker)
		}
		after = strings.TrimSpace(after)
		if len([]rune(after)) > 10 {
			exp = after
		}
	}

	// 限制总长度
	runes := []rune(exp)
	if len(runes) > 300 {
		runes = runes[:300]
	}

	result := strings.TrimSpace(string(runes))
	// 移除连续空白
	result = strings.Join(strings.Fields(result), " ")
	return result
}

// buildChars 构建 8105 字的完整数据
func buildChars(records [][]string, hzMap map[string]hanziJSON, wdMap map[string]string) []NamerChar {
	chars := make([]NamerChar, 0, len(records))

	for _, r := range records {
		if len(r) < 6 {
			continue
		}

		seqStr := strings.TrimSpace(r[0])
		seq, _ := strconv.Atoi(seqStr)
		ch := strings.TrimSpace(r[1])
		pinyin := strings.TrimSpace(r[2])
		radical := strings.TrimSpace(r[3])
		strokeStr := strings.TrimSpace(r[4])
		strokes, _ := strconv.Atoi(strokeStr)
		csvWuxing := strings.TrimSpace(r[5])

		// 确定等级
		level := 3
		if seq <= 3500 {
			level = 1
		} else if seq <= 6500 {
			level = 2
		}

		// 从 hanzi.json 补充数据
		hz, hasHz := hzMap[ch]

		// 释义：word.json 真实字典释义 > hanzi.json 释义 > 空（由 ClassifyNaming 生成简要描述）
		meaning := ""
		if wdMeaning, ok := wdMap[ch]; ok {
			meaning = wdMeaning
		}
		if meaning == "" && hasHz && hz.Meaning != "" {
			// hanzi.json 中非占位的释义（长度>20）才采用
			if len([]rune(hz.Meaning)) > 20 {
				meaning = hz.Meaning
			}
		}

		// 五行：CSV 优先，其次 hanzi.json，其次部首兜底
		wuxing := csvWuxing
		if wuxing == "" || wuxing == "-" {
			if hasHz && hz.Wuxing != "" {
				wuxing = hz.Wuxing
			}
		}
		if wuxing == "" || wuxing == "-" {
			wuxing = hanzi.GetWuxingByRadical(radical)
		}
		if wuxing == "" || wuxing == "-" {
			wuxing = "土" // 终极兜底：土为中和
		}

		// 性别倾向：优先 hanzi.json
		gender := "neutral"
		if hasHz && hz.Gender != "" && hz.Gender != "通用" {
			gender = hz.Gender
		}

		// 起名分类
		cats := hanzi.ClassifyNaming(ch, radical, meaning, wuxing)

		// 自动检测多音字：拼音包含逗号分隔的多个拼音
		isPoly := strings.Contains(pinyin, ",")

		nc := NamerChar{
			Char:             ch,
			Pinyin:           pinyin,
			Strokes:          strokes,
			Radical:          radical,
			Meaning:          meaning,
			Wuxing:           wuxing,
			Level:            level,
			Gender:           gender,
			NamingCategories: cats,
			IsPolyphonic: isPoly,

			// 从 hanzi.json 传入补充字段（P0-P2）
			IsNegative: hasHz && hz.IsNegative,
			IsRare:     hasHz && hz.IsRare,
			NamePenalty: func() int {
				if hasHz { return hz.NamePenalty }
				return 0
			}(),
			Tone: func() int {
				if hasHz { return hz.Tone }
				return 0
			}(),
			StyleTags: func() []string {
				if hasHz { return hz.StyleTags }
				return nil
			}(),
			UsageLevel: func() int {
				if hasHz { return hz.UsageLevel }
				return 0
			}(),
			PositiveScore: func() int {
				if hasHz { return hz.PositiveScore }
				return 0
			}(),
		}
		chars = append(chars, nc)
	}

	return chars
}

func printStats(chars []NamerChar) {
	t1, t2, t3 := 0, 0, 0
	wxDist := make(map[string]int)
	catCount := 0
	for _, c := range chars {
		switch c.Level {
		case 1:
			t1++
		case 2:
			t2++
		case 3:
			t3++
		}
		wxDist[c.Wuxing]++
		if len(c.NamingCategories) > 0 {
			catCount++
		}
	}
	catDist := make(map[string]int)
	for _, c := range chars {
		for _, cat := range c.NamingCategories {
			catDist[cat]++
		}
	}
	fmt.Printf("📊 统计:\n")
	fmt.Printf("   一级字: %d  二级字: %d  三级字: %d\n", t1, t2, t3)
	// 字义覆盖统计
	withMeaning := 0
	for _, c := range chars {
		if c.Meaning != "" {
			withMeaning++
		}
	}
	fmt.Printf("   有字义: %d/%d (%.1f%%)\n", withMeaning, len(chars), float64(withMeaning)/float64(len(chars))*100)
	fmt.Printf("   有起名分类: %d (%.1f%%)\n", catCount, float64(catCount)/float64(len(chars))*100)
	fmt.Printf("   五行分布: %v\n", wxDist)
	fmt.Printf("   分类分布:\n")
	for _, name := range hanzi.NamingCategoryNames {
		fmt.Printf("     %-10s %4d\n", name, catDist[name])
	}
}
