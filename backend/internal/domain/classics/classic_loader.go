package classics

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"name/internal/domain/hanzi"
	"name/internal/infrastructure/logger"
	"go.uber.org/zap"
)

// ============================================================
// 原始 JSON 结构体——各经典文件对应的 Go 类型
// ============================================================

// ShiJingEntry 诗经 JSON 条目
type ShiJingEntry struct {
	Title   string   `json:"title"`
	Chapter string   `json:"chapter"`
	Section string   `json:"section"`
	Content []string `json:"content"`
}

// ChuCiEntry 楚辞 JSON 条目
type ChuCiEntry struct {
	Title   string   `json:"title"`
	Section string   `json:"section"`
	Author  string   `json:"author"`
	Content []string `json:"content"`
}

// guWenRaw 古文观止 JSON 顶层结构
type guWenRaw struct {
	Content []guWenChapter `json:"content"`
}

type guWenChapter struct {
	Chapter    string   `json:"chapter"`
	Source     string   `json:"source"`
	Author     string   `json:"author"`
	Paragraphs []string `json:"paragraphs"`
}

// GuWenEntry 古文观止适配器（实现 poemEntry）
type GuWenEntry struct {
	Title    string
	Content  []string
	Source   string
}

func (e GuWenEntry) GetTitle() string     { return e.Title }
func (e GuWenEntry) GetContent() []string  { return e.Content }
func (e GuWenEntry) GetSource() string     { return e.Source }

// ShiCiEntry 诗词 JSON 条目（shici.json 格式）
type ShiCiEntry struct {
	Title      string   `json:"title"`
	Author     string   `json:"author"`
	Dynasty    string   `json:"dynasty"`
	Type       string   `json:"type"`
	Paragraphs []string `json:"paragraphs"`
	Source     string   `json:"source"`
	Tags       []string `json:"tags"`
}

func (e ShiCiEntry) GetTitle() string     { return e.Title }
func (e ShiCiEntry) GetContent() []string  { return e.Paragraphs }
func (e ShiCiEntry) GetSource() string {
	if e.Source != "" {
		return e.Source
	}
	return "唐诗宋词"
}

// ClassicSection 通用篇章适配器
// 覆盖 format: { content: [{ chapter: "...", paragraphs: ["..."] }] }
// 适用于：lunyu, mengzi, daxue, zhongyong, sanzijing, qianziwen,
// dizigui, youxueqionglin, zengguangxianwen, shenglvqimeng 等
type ClassicSection struct {
	Title    string
	Content  []string
	Source   string
}

func (s ClassicSection) GetTitle() string     { return s.Title }
func (s ClassicSection) GetContent() []string  { return s.Content }
func (s ClassicSection) GetSource() string     { return s.Source }

// ============================================================
// poemEntry 通用诗歌条目接口
// ============================================================

type poemEntry interface {
	GetTitle() string
	GetContent() []string
	GetSource() string
}

// ShiJingEntry 实现 poemEntry
func (e ShiJingEntry) GetTitle() string     { return e.Title }
func (e ShiJingEntry) GetContent() []string { return e.Content }
func (e ShiJingEntry) GetSource() string    { return "诗经" }

// ChuCiEntry 实现 poemEntry
func (e ChuCiEntry) GetTitle() string   { return e.Title }
func (e ChuCiEntry) GetContent() []string { return e.Content }
func (e ChuCiEntry) GetSource() string    { return "楚辞" }

// shiciReady 通道，shici.json 加载完成后关闭
var shiciReady = make(chan struct{})

// ensureShiCiLoaded 确保唐诗宋词数据已加载完成
// 适用于异步加载场景：shici.json（6.7MB）后台加载，业务入口处同步等待
func ensureShiCiLoaded() {
	<-shiciReady
}

// loadShiCiAsync 在后台协程中加载 shici.json，不阻塞启动
func loadShiCiAsync(dataDir string) {
	go func() {
		if err := loadShiCiFromJSON(dataDir); err != nil {
			logger.Error("唐诗宋词后台加载失败", zap.Error(err))
		}
		close(shiciReady)
	}()

	logger.Info("唐诗宋词数据已提交后台加载（6.7MB），不阻塞启动")
}



// ============================================================
// 加载函数——每个经典文件一个
// ============================================================

// loadShiJingFromJSON 从 shijing.json 提取起名用字
func loadShiJingFromJSON(dataDir string) error {
	path := filepath.Join(dataDir, "shijing.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取 shijing.json 失败: %w", err)
	}

	var entries []ShiJingEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("解析 shijing.json 失败: %w", err)
	}

	extracted := extractFromEntries(toPoemEntries(entries))
	setExtracted(&ShijingExtracted, extracted)
	logger.Info("已从诗经提取起名用字", zap.Int("poems", len(entries)), zap.Int("chars", len(extracted)))
	return nil
}

// loadChuCiFromJSON 从 chuci.json 提取起名用字
func loadChuCiFromJSON(dataDir string) error {
	path := filepath.Join(dataDir, "chuci.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取 chuci.json 失败: %w", err)
	}

	var entries []ChuCiEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("解析 chuci.json 失败: %w", err)
	}

	extracted := extractFromEntries(toPoemEntries(entries))
	setExtracted(&ChuciExtracted, extracted)
	logger.Info("已从楚辞提取起名用字", zap.Int("poems", len(entries)), zap.Int("chars", len(extracted)))
	return nil
}

// loadGuWenFromJSON 从 guwenguanzhi.json 提取起名用字
func loadGuWenFromJSON(dataDir string) error {
	path := filepath.Join(dataDir, "guwenguanzhi.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取 guwenguanzhi.json 失败: %w", err)
	}

	var raw guWenRaw
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("解析 guwenguanzhi.json 失败: %w", err)
	}

	entries := make([]poemEntry, 0, len(raw.Content))
	for _, ch := range raw.Content {
		source := ch.Source
		if source == "" {
			source = "古文观止"
		}
		entries = append(entries, GuWenEntry{
			Title:   ch.Chapter,
			Content: ch.Paragraphs,
			Source:  source,
		})
	}

	extracted := extractFromEntries(entries)
	setExtracted(&GuwenGuanzhiExtracted, extracted)
	logger.Info("已从古文观止提取起名用字", zap.Int("articles", len(entries)), zap.Int("chars", len(extracted)))
	return nil
}

// loadShiCiFromJSON 从 shici.json 提取起名用字
func loadShiCiFromJSON(dataDir string) error {
	path := filepath.Join(dataDir, "shici.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取 shici.json 失败: %w", err)
	}

	var entries []ShiCiEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("解析 shici.json 失败: %w", err)
	}

	extracted := extractFromEntries(toPoemEntries(entries))
	setExtracted(&ShiCiExtracted, extracted)
	logger.Info("已从唐诗宋词提取起名用字", zap.Int("poems", len(entries)), zap.Int("chars", len(extracted)))
	return nil
}

// loadClassicFromJSON 加载标准格式的经典文件
// 格式: { content: [{ chapter: "...", paragraphs: ["..."] }] }
// target: 存放提取结果的指针
func loadClassicFromJSON(dataDir, filename, source string, target *[]PoetryChar) error {
	path := filepath.Join(dataDir, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取 %s 失败: %w", filename, err)
	}

	var raw struct {
		Content []struct {
			Chapter    string   `json:"chapter"`
			Paragraphs []string `json:"paragraphs"`
		} `json:"content"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("解析 %s 失败: %w", filename, err)
	}

	entries := make([]poemEntry, 0, len(raw.Content))
	for _, ch := range raw.Content {
		title := ch.Chapter
		if title == "" {
			title = source
		}
		entries = append(entries, ClassicSection{
			Title:   title,
			Content: ch.Paragraphs,
			Source:  source,
		})
	}

	extracted := extractFromEntries(entries)
	setExtracted(target, extracted)

	totalSections := 0
	totalLines := 0
	for _, ch := range raw.Content {
		totalSections++
		totalLines += len(ch.Paragraphs)
	}
	logger.Info("已从经典提取起名用字", zap.String("source", source), zap.Int("sections", totalSections), zap.Int("chars", len(extracted)))
	return nil
}

// ============================================================
// 类型转换辅助
// ============================================================

// toPoemEntries 将实现了 poemEntry 的具体类型切片转为接口切片
func toPoemEntries[T poemEntry](entries []T) []poemEntry {
	result := make([]poemEntry, len(entries))
	for i, e := range entries {
		result[i] = e
	}
	return result
}

// ============================================================
// 核心提取逻辑（通用）
// ============================================================

// extractFromEntries 从任意 poemEntry 切片中提取起名用字
// 去重、过滤停用词、填充属性一步完成
func extractFromEntries(entries []poemEntry) []PoetryChar {
	seen := make(map[string]*PoetryChar)

	for _, entry := range entries {
		extractFromEntry(entry.GetTitle(), entry.GetContent(), entry.GetSource(), seen)
	}

	return mapToSortedSlice(seen)
}

// extractFromEntry 从单个诗歌条目中提取所有汉字，去重并查 HanziData 补充属性
func extractFromEntry(title string, content []string, source string, seen map[string]*PoetryChar) {
	// 拼接所有诗句
	allText := strings.Join(content, "")

	// 提取第一行中包含本字的最短诗句用于 sentence
	sentenceMap := buildSentenceMap(content)

	for _, r := range allText {
		char := string(r)

		// 仅保留汉字（CJK 统一表意文字区间）
		if !isCJKChar(r) {
			continue
		}

		// 跳过停用词
		if isStopWord(char) {
			continue
		}

		// 去重
		if _, exists := seen[char]; exists {
			continue
		}

		// 从 HanziData 查找属性
		pc := PoetryChar{
			Char:    char,
			Work:    source,
			Chapter: title,
		}

		if h, ok := hanzi.HanziData[char]; ok {
			pc.Pinyin = h.Pinyin
			pc.Meaning = h.Meaning
			pc.Wuxing = h.Wuxing
			pc.Gender = h.Gender
		} else {
			// 不在 HanziData 中，设默认值
			pc.Pinyin = ""
			pc.Meaning = ""
			pc.Wuxing = ""
			pc.Gender = "通用"
		}

		// 查找包含本字的最短诗句
		if sentence, ok := findShortestSentence(char, sentenceMap); ok {
			pc.Sentence = sentence
		}

		seen[char] = &pc
	}
}

// buildSentenceMap 构建字符 → 诗句列表的映射，用于反查出处
func buildSentenceMap(content []string) map[string][]string {
	sentenceMap := make(map[string][]string)
	for _, line := range content {
		// 分割完整诗句（句号、问号、感叹号、分号分割）
		segments := splitSentences(line)
		for _, seg := range segments {
			seg = strings.TrimSpace(seg)
			if seg == "" {
				continue
			}
			// 记录本诗句包含的所有字
			charsInLine := extractUniqueChars(seg)
			for _, c := range charsInLine {
				if isStopWord(c) || !isCJKChar([]rune(c)[0]) {
					continue
				}
				sentenceMap[c] = append(sentenceMap[c], seg)
			}
		}
	}
	return sentenceMap
}

// findShortestSentence 找到包含 char 的最短诗句
func findShortestSentence(char string, sentenceMap map[string][]string) (string, bool) {
	sentences, ok := sentenceMap[char]
	if !ok || len(sentences) == 0 {
		return "", false
	}
	shortest := sentences[0]
	for _, s := range sentences[1:] {
		if len(s) < len(shortest) {
			shortest = s
		}
	}
	return shortest, true
}

// splitSentences 按标点分割诗句为独立句子
func splitSentences(text string) []string {
	var result []string
	var buf strings.Builder
	for _, r := range text {
		if r == '。' || r == '？' || r == '！' || r == '；' || r == '，' || r == '、' {
			if buf.Len() > 0 {
				result = append(result, buf.String())
				buf.Reset()
			}
		} else {
			buf.WriteRune(r)
		}
	}
	if buf.Len() > 0 {
		result = append(result, buf.String())
	}
	return result
}

// extractUniqueChars 从文本中提取唯一汉字（去停用词）
func extractUniqueChars(text string) []string {
	seen := make(map[string]bool)
	var chars []string
	for _, r := range text {
		char := string(r)
		if !isCJKChar(r) || isStopWord(char) || seen[char] {
			continue
		}
		seen[char] = true
		chars = append(chars, char)
	}
	return chars
}

// mapToSortedSlice 将 map 转换为排序后的 PoetryChar 切片
func mapToSortedSlice(seen map[string]*PoetryChar) []PoetryChar {
	result := make([]PoetryChar, 0, len(seen))
	for _, pc := range seen {
		result = append(result, *pc)
	}
	return result
}

// ============================================================
// 辅助函数
// ============================================================

// isCJKChar 判断是否为 CJK 统一表意文字（汉字）
func isCJKChar(r rune) bool {
	return unicode.Is(unicode.Han, r)
}

// stopWords 起名用字——过滤无实际含义的古文虚词/停用词
var stopWords = map[string]bool{
	"之": true, "乎": true, "者": true, "也": true, "而": true,
	"其": true, "以": true, "于": true, "兮": true, "哉": true,
	"焉": true, "矣": true, "欤": true, "耶": true, "尔": true,
	"乃": true, "且": true, "所": true, "为": true, "与": true,
	"则": true, "何": true, "如": true, "若": true, "虽": true,
	"然": true, "但": true, "或": true, "既": true, "及": true,
	"不": true, "勿": true, "毋": true, "未": true, "非": true,
	"岂": true, "盍": true, "讵": true, "遽": true,
	"此": true, "彼": true, "夫": true, "惟": true, "唯": true,
	"嗟": true, "噫": true, "吁": true, "嘻": true,
	"猗": true, "已": true, "各": true, "每": true,
	"谁": true, "孰": true, "胡": true, "曷": true, "诸": true,
	"来": true, "爰": true,
	"曰": true, "云": true, "谓": true, "言": true,
}

func isStopWord(char string) bool {
	return stopWords[char]
}
