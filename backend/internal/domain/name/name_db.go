package name

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"

	"name/internal/domain/hanzi"
	"name/internal/infrastructure/logger"
	"go.uber.org/zap"
)

// ============================================================
// 候选名库数据结构
// ============================================================

// CuratedName 精选候选名
type CuratedName struct {
	Name        string   `json:"name"`
	Pinyin      string   `json:"pinyin"`
	Gender      string   `json:"gender"`
	Source      string   `json:"source"`
	Meaning     string   `json:"meaning"`
	Wuxing      string   `json:"wuxing"`
	YinyunScore float64  `json:"yinyun_score"`
	Styles      []string `json:"styles,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// CuratedPersister 精选名持久化接口（由基础设施层实现，避免 domain 依赖 infrastructure）
type CuratedPersister interface {
	SaveCuratedName(name, pinyin, gender string, score float64, source string) error
	LoadAllCuratedNames() ([]CuratedNameEntry, error)
	IsCurated(name string) (bool, error)
}

// CuratedNameEntry 精简版本，仅包含持久化所需字段
type CuratedNameEntry struct {
	Name   string  `json:"name"`
	Pinyin string  `json:"pinyin,omitempty"`
	Gender string  `json:"gender,omitempty"`
	Score  float64 `json:"score"`
	Source string  `json:"source,omitempty"`
}

// NameDB 候选名库管理器
type NameDB struct {
	mu           sync.RWMutex
	dataDir      string
	curatedNames []CuratedName
	shiyunNames  []CuratedName  // 诗云等开源项目精选名字库
	charGroups   []hanzi.StandardCharGroup
	// 持久化
	persister CuratedPersister
	// 缓存
	charIndex   map[string]string   // char → radical
	nameIndex   map[string]bool      // 去重索引
}

// NameDBOption NameDB 构造选项
type NameDBOption func(*NameDB)

// WithCuratedPersister 设置精选名持久化接口
func WithCuratedPersister(p CuratedPersister) NameDBOption {
	return func(db *NameDB) {
		db.persister = p
	}
}

// ShiYunSource 诗云数据源标识常量
const ShiYunSource = "诗云精选"

// NewNameDB 创建候选名库管理器
func NewNameDB(dataDir string, opts ...NameDBOption) (*NameDB, error) {
	db := &NameDB{
		dataDir:    dataDir,
		charIndex:  make(map[string]string),
		nameIndex:  make(map[string]bool),
	}
	for _, opt := range opts {
		opt(db)
	}
	if err := db.Load(); err != nil {
		return db, err
	}
	return db, nil
}

// Load 加载所有候选名库数据
func (db *NameDB) Load() error {
	// 加载精选候选名库（JSON 种子）
	if err := db.loadCuratedNames(); err != nil {
		logger.Warn("加载精选候选名库失败", zap.Error(err))
	}

	// 从持久化存储加载用户自学习精选名
	if err := db.loadCuratedFromPersister(); err != nil {
		logger.Warn("加载自学习精选名失败", zap.Error(err))
	}

	// 加载诗云等开源项目精选名字库
	if err := db.loadShiYunNames(); err != nil {
		logger.Warn("加载诗云精选名字库失败", zap.Error(err))
	}

	// 加载标准起名用字
	if err := db.loadStandardChars(); err != nil {
		logger.Warn("加载标准起名用字失败", zap.Error(err))
	}

	return nil
}

// Reload 热更新所有数据
func (db *NameDB) Reload() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// 清空索引
	db.charIndex = make(map[string]string)
	db.nameIndex = make(map[string]bool)

	return db.Load()
}

// loadCuratedNames 从 JSON 加载精选候选名库
func (db *NameDB) loadCuratedNames() error {
	path := filepath.Join(db.dataDir, "curated_names.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var names []CuratedName
	if err := json.Unmarshal(data, &names); err != nil {
		return err
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	db.curatedNames = names
	for _, n := range names {
		db.nameIndex[n.Name] = true
	}

	logger.Info("已加载精选候选名库", zap.Int("count", len(names)))
	return nil
}

// loadCuratedFromPersister 从持久化存储加载自学习精选名并合并到内存
func (db *NameDB) loadCuratedFromPersister() error {
	if db.persister == nil {
		return nil
	}
	entries, err := db.persister.LoadAllCuratedNames()
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return nil
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	added := 0
	for _, e := range entries {
		if !db.nameIndex[e.Name] {
			db.curatedNames = append(db.curatedNames, CuratedName{
				Name:   e.Name,
				Pinyin: e.Pinyin,
				Gender: e.Gender,
				Source: e.Source,
			})
			db.nameIndex[e.Name] = true
			added++
		}
	}
	if added > 0 {
		logger.Info("已合并自学习精选名", zap.Int("added", added))
	}
	return nil
}

// AddCuratedName 添加一个名字到精选库（内存 + 持久化）
func (db *NameDB) AddCuratedName(name, pinyin, gender string, score float64, source string) error {
	db.mu.Lock()
	// 已存在则跳过
	if db.nameIndex[name] {
		db.mu.Unlock()
		return nil
	}
	db.curatedNames = append(db.curatedNames, CuratedName{
		Name:   name,
		Pinyin: pinyin,
		Gender: gender,
		Source: source,
	})
	db.nameIndex[name] = true
	db.mu.Unlock()

	// 异步持久化（不阻塞调用方）
	if db.persister != nil {
		go func() {
			if err := db.persister.SaveCuratedName(name, pinyin, gender, score, source); err != nil {
				logger.Warn("持久化精选名失败", zap.String("name", name), zap.Error(err))
			}
		}()
	}
	return nil
}

// loadShiYunNames 从 data/ 目录加载精选名字库JSON
// JSON 格式为扁平字符串数组，每个元素是一个双字名字
// 使用文件名前缀白名单（shiyun_*.json），避免误解析结构化JSON文件
func (db *NameDB) loadShiYunNames() error {
	entries, err := os.ReadDir(db.dataDir)
	if err != nil {
		return fmt.Errorf("读取数据目录失败: %w", err)
	}

	var allNames []CuratedName
	totalImported := 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "shiyun_") || !strings.HasSuffix(name, ".json") {
			continue
		}
		path := filepath.Join(db.dataDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			logger.Warn("跳过诗云文件（读取失败）", zap.String("file", name), zap.Error(err))
			continue
		}

		var nameList []string
		if err := json.Unmarshal(data, &nameList); err != nil {
			logger.Warn("跳过诗云文件（JSON解析失败）", zap.String("file", name), zap.Error(err))
			continue
		}

		// 源文件名作为来源标记（去除.json后缀）
		sourceName := strings.TrimSuffix(name, ".json")

		for _, nameStr := range nameList {
			nameStr = strings.TrimSpace(nameStr)
			if nameStr == "" || utf8.RuneCountInString(nameStr) < 2 {
				continue
			}
			allNames = append(allNames, CuratedName{
				Name:   nameStr,
				Source: ShiYunSource,
				Tags:   []string{sourceName},
			})
			totalImported++
		}
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	db.shiyunNames = allNames
	for _, n := range allNames {
		db.nameIndex[n.Name] = true
	}

	if totalImported > 0 {
		logger.Info("已加载诗云精选名字库", zap.Int("files", len(allNames)), zap.Int("total_before_dedup", totalImported))
	}
	return nil
}

// loadStandardChars 加载标准起名用字（按偏旁分组）
//
// 分组数据已并入 namer.json 顶层 charGroups（单一文件真源），由 hanzi 包
// 在 LoadNamerFromJSON 时解析并暴露。此处直接从内存取用，不再读盘
// standard_chars.json；若为空则从 HanziData 按偏旁聚合兜底。
func (db *NameDB) loadStandardChars() error {
	groups := hanzi.GetNamerGroups()
	if len(groups) == 0 {
		// 兜底：由 HanziData（namer.json）按偏旁聚合
		byRadical := make(map[string][]string)
		hanzi.ForEachHanzi(func(h hanzi.Hanzi) {
			if h.Radical != "" {
				byRadical[h.Radical] = append(byRadical[h.Radical], h.Char)
			}
		})
		for radical, chars := range byRadical {
			groups = append(groups, hanzi.StandardCharGroup{
				Radical: radical,
				Chars:   chars,
			})
		}
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	db.charGroups = groups
	for _, g := range groups {
		for _, ch := range g.Chars {
			db.charIndex[ch] = g.Radical
		}
	}

	logger.Info("已加载标准起名用字", zap.Int("radical_groups", len(groups)), zap.Int("chars", len(db.charIndex)))
	return nil
}

// ============================================================
// 查询方法
// ============================================================

// GetCuratedNames 获取精选候选名库（可按性别过滤）
func (db *NameDB) GetCuratedNames(gender string) []CuratedName {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if gender == "" || gender == "通用" {
		result := make([]CuratedName, len(db.curatedNames))
		copy(result, db.curatedNames)
		return result
	}

	var result []CuratedName
	for _, n := range db.curatedNames {
		if n.Gender == gender || n.Gender == "通用" {
			result = append(result, n)
		}
	}
	return result
}

// GetCharRadical 获取汉字所属偏旁
func (db *NameDB) GetCharRadical(char string) string {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.charIndex[char]
}

// GetCharGroups 获取所有偏旁分组
func (db *NameDB) GetCharGroups() []hanzi.StandardCharGroup {
	db.mu.RLock()
	defer db.mu.RUnlock()

	result := make([]hanzi.StandardCharGroup, len(db.charGroups))
	copy(result, db.charGroups)
	return result
}

// GetGroupChars 获取指定偏旁的汉字列表
func (db *NameDB) GetGroupChars(radical string) []string {
	db.mu.RLock()
	defer db.mu.RUnlock()

	for _, g := range db.charGroups {
		if g.Radical == radical {
			result := make([]string, len(g.Chars))
			copy(result, g.Chars)
			return result
		}
	}
	return nil
}

// IsCuratedName 检查是否为精选候选名库中的名字
func (db *NameDB) IsCuratedName(name string) bool {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.nameIndex[name]
}

// SearchCuratedNames 按关键词搜索候选名
func (db *NameDB) SearchCuratedNames(keyword string) []CuratedName {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var result []CuratedName
	keyword = strings.ToLower(keyword)
	for _, n := range db.curatedNames {
		if strings.Contains(strings.ToLower(n.Name), keyword) ||
			strings.Contains(strings.ToLower(n.Source), keyword) ||
			strings.Contains(strings.ToLower(n.Meaning), keyword) {
			result = append(result, n)
		}
	}
	return result
}

// GetShiyunNames 获取诗云精选名字列表
func (db *NameDB) GetShiyunNames() []CuratedName {
	db.mu.RLock()
	defer db.mu.RUnlock()

	result := make([]CuratedName, len(db.shiyunNames))
	copy(result, db.shiyunNames)
	return result
}

// IsShiyunName 检查名字是否来自诗云精选库
func (db *NameDB) IsShiyunName(name string) bool {
	db.mu.RLock()
	defer db.mu.RUnlock()
	// nameIndex 包含所有 curated + shiyun 名字
	return db.nameIndex[name]
}

// GetTopCuratedNames 获取评分最高的候选名
func (db *NameDB) GetTopCuratedNames(gender string, limit int) []CuratedName {
	names := db.GetCuratedNames(gender)

	sort.Slice(names, func(i, j int) bool {
		return names[i].YinyunScore > names[j].YinyunScore
	})

	if limit > 0 && limit < len(names) {
		return names[:limit]
	}
	return names
}

// GetStyles 获取所有可用的风格标签
func (db *NameDB) GetStyles() []string {
	db.mu.RLock()
	defer db.mu.RUnlock()

	styleSet := make(map[string]bool)
	for _, n := range db.curatedNames {
		for _, s := range n.Styles {
			styleSet[s] = true
		}
	}

	var styles []string
	for s := range styleSet {
		styles = append(styles, s)
	}
	sort.Strings(styles)
	return styles
}

// GetNamesByStyle 按风格筛选候选名
func (db *NameDB) GetNamesByStyle(style string) []CuratedName {
	db.mu.RLock()
	defer db.mu.RUnlock()

	var result []CuratedName
	for _, n := range db.curatedNames {
		for _, s := range n.Styles {
			if s == style {
				result = append(result, n)
				break
			}
		}
	}
	return result
}