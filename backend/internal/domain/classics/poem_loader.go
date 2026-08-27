package classics

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode"

	"name/internal/infrastructure/logger"
	"go.uber.org/zap"
)

// GlobalPoemIndex 全局诗词索引，启动时构建
var (
	GlobalPoemIndex *PoemIndex
	poemIndexMu     sync.RWMutex
)

// GetGlobalPoemIndex 获取全局诗词索引（线程安全）
func GetGlobalPoemIndex() *PoemIndex {
	poemIndexMu.RLock()
	defer poemIndexMu.RUnlock()
	return GlobalPoemIndex
}

// BuildPoemIndexFromJSON 从所有现有JSON文件构建诗词索引
func BuildPoemIndexFromJSON(dataDir string) error {
	idx := NewPoemIndex()

	// 加载诗经
	if err := loadShijingToIndex(idx, dataDir); err != nil {
		logger.Warn("加载诗经到索引失败", zap.Error(err))
	}

	// 加载楚辞
	if err := loadChuciToIndex(idx, dataDir); err != nil {
		logger.Warn("加载楚辞到索引失败", zap.Error(err))
	}

	// 加载唐诗宋词
	if err := loadShiciToIndex(idx, dataDir); err != nil {
		logger.Warn("加载唐诗宋词到索引失败", zap.Error(err))
	}

	// 加载元曲
	if err := loadYuanquToIndex(idx, dataDir); err != nil {
		logger.Warn("加载元曲到索引失败", zap.Error(err))
	}

	// 设置全局索引
	poemIndexMu.Lock()
	GlobalPoemIndex = idx
	poemIndexMu.Unlock()

	stats := idx.GetStats()
	logger.Info("诗词索引构建完成",
		zap.Int("总条目", stats["total_entries"]),
		zap.Int("涵盖汉字", stats["total_chars"]),
		zap.Int("朝代数", stats["dynasties"]),
		zap.Int("作者数", stats["authors"]),
		zap.Int("分类数", stats["categories"]),
		zap.Int("来源数", stats["sources"]),
	)

	return nil
}

// loadShijingToIndex 加载诗经到索引
func loadShijingToIndex(idx *PoemIndex, dataDir string) error {
	path := filepath.Join(dataDir, "shijing.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var entries []ShiJingEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return err
	}

	for _, entry := range entries {
		pe := &PoemEntry{
			Title:    entry.Title,
			Author:   "佚名",
			Dynasty:  "先秦",
			Type:     "诗",
			Category: categorizeShijing(entry.Title),
			Content:  entry.Content,
			FullText: strings.Join(entry.Content, ""),
			Source:   "诗经",
			Book:     entry.Chapter + "·" + entry.Section,
			Tags:     []string{"诗经", entry.Chapter, entry.Section},
			Chars:    extractCharsFromContent(entry.Content),
		}
		idx.AddEntry(pe)
	}

	logger.Info("诗经已加载到索引", zap.Int("条目数", len(entries)))
	return nil
}

// loadChuciToIndex 加载楚辞到索引
func loadChuciToIndex(idx *PoemIndex, dataDir string) error {
	path := filepath.Join(dataDir, "chuci.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var entries []ChuCiEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return err
	}

	for _, entry := range entries {
		author := entry.Author
		if author == "" {
			author = "屈原"
		}
		pe := &PoemEntry{
			Title:    entry.Title,
			Author:   author,
			Dynasty:  "先秦",
			Type:     "辞",
			Category: "抒情",
			Content:  entry.Content,
			FullText: strings.Join(entry.Content, ""),
			Source:   "楚辞",
			Book:     entry.Section,
			Tags:     []string{"楚辞", entry.Section},
			Chars:    extractCharsFromContent(entry.Content),
		}
		idx.AddEntry(pe)
	}

	logger.Info("楚辞已加载到索引", zap.Int("条目数", len(entries)))
	return nil
}

// loadShiciToIndex 加载唐诗宋词到索引
func loadShiciToIndex(idx *PoemIndex, dataDir string) error {
	path := filepath.Join(dataDir, "shici.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var entries []ShiCiEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return err
	}

	for _, entry := range entries {
		// 确定分类
		category := entry.Type
		if category == "" {
			category = categorizeByTitle(entry.Title)
		}

		// 确定标签
		tags := entry.Tags
		if len(tags) == 0 {
			tags = []string{entry.Dynasty, entry.Type}
		}

		pe := &PoemEntry{
			Title:    entry.Title,
			Author:   entry.Author,
			Dynasty:  entry.Dynasty,
			Type:     entry.Type,
			Category: category,
			Content:  entry.Paragraphs,
			FullText: strings.Join(entry.Paragraphs, ""),
			Source:   entry.Source,
			Book:     entry.Source,
			Tags:     tags,
			Chars:    extractCharsFromContent(entry.Paragraphs),
		}
		idx.AddEntry(pe)
	}

	logger.Info("唐诗宋词已加载到索引", zap.Int("条目数", len(entries)))
	return nil
}

// loadYuanquToIndex 加载元曲到索引
func loadYuanquToIndex(idx *PoemIndex, dataDir string) error {
	path := filepath.Join(dataDir, "yuanqu.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var raw struct {
		元曲一 struct {
			关汉卿 []struct {
				Title      string   `json:"title"`
				Author     string   `json:"author"`
				Paragraphs []string `json:"paragraphs"`
			} `json:"关汉卿"`
		} `json:"元曲一"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	for _, entry := range raw.元曲一.关汉卿 {
		pe := &PoemEntry{
			Title:    entry.Title,
			Author:   entry.Author,
			Dynasty:  "元",
			Type:     "曲",
			Category: "杂剧",
			Content:  entry.Paragraphs,
			FullText: strings.Join(entry.Paragraphs, ""),
			Source:   "元曲",
			Book:     "元曲选",
			Tags:     []string{"元曲", "关汉卿"},
			Chars:    extractCharsFromContent(entry.Paragraphs),
		}
		idx.AddEntry(pe)
	}

	logger.Info("元曲已加载到索引", zap.Int("条目数", len(raw.元曲一.关汉卿)))
	return nil
}

// extractCharsFromContent 从内容中提取所有汉字（去重）
func extractCharsFromContent(content []string) map[string]bool {
	chars := make(map[string]bool)
	for _, line := range content {
		for _, r := range line {
			if unicode.Is(unicode.Han, r) {
				chars[string(r)] = true
			}
		}
	}
	return chars
}

// categorizeShijing 根据诗经标题推断分类
func categorizeShijing(title string) string {
	// 简单分类逻辑，可根据需要扩展
	if strings.Contains(title, "风") {
		return "民歌"
	}
	if strings.Contains(title, "雅") {
		return "宫廷"
	}
	if strings.Contains(title, "颂") {
		return "祭祀"
	}
	return "诗歌"
}

// categorizeByTitle 根据标题推断分类
func categorizeByTitle(title string) string {
	// 简单分类逻辑
	keywords := map[string]string{
		"登": "山水",
		"望": "山水",
		"春": "写景",
		"秋": "写景",
		"送": "送别",
		"别": "送别",
		"思": "思乡",
		"忆": "思乡",
		"闺": "闺怨",
		"从军": "边塞",
		"出塞": "边塞",
	}

	for keyword, category := range keywords {
		if strings.Contains(title, keyword) {
			return category
		}
	}
	return "其他"
}
