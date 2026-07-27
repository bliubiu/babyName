package sqlite

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"name/internal/infrastructure/logger"
)

// 经典数据导入 - 将 JSON 文件中的经典数据写入 SQLite
//
// 统一的三表设计：
//   classics_books      - 书籍级别元数据（如《诗经》《论语》《唐诗三百首》）
//   classics_sections   - 篇章级别（如"国风·周南"、"学而篇"）
//   classics_paragraphs - 段落/句子级别（具体诗文内容）
//
// 覆盖 23 个 JSON 文件（排除 hanzi/word/curated_names/standard_chars/zodiac）

// --- 解析用中间结构 ---

// arrayWork 数组型 JSON 中的单条作品
type arrayWork struct {
	Title      string   `json:"title"`
	Author     string   `json:"author"`
	Dynasty    string   `json:"dynasty"`
	Type       string   `json:"type"`
	Chapter    string   `json:"chapter"`
	Section   string   `json:"section"`
	Content    []string `json:"content"`
	Paragraphs []string `json:"paragraphs"`
	Source     string   `json:"source"`
	Tags       []string `json:"tags"`
	Book       string   `json:"book"`

	// 注意：content 兼容 string 和 []string 两种格式，见 UnmarshalJSON
}

// UnmarshalJSON 兼容 content 可能是 string 或 []string 的情况
func (w *arrayWork) UnmarshalJSON(data []byte) error {
	// 用 map 辅助判断 content 类型
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// 基础字段
	if v, ok := raw["title"].(string); ok {
		w.Title = v
	}
	if v, ok := raw["author"].(string); ok {
		w.Author = v
	}
	if v, ok := raw["dynasty"].(string); ok {
		w.Dynasty = v
	}
	if v, ok := raw["type"].(string); ok {
		w.Type = v
	}
	if v, ok := raw["chapter"].(string); ok {
		w.Chapter = v
	}
	if v, ok := raw["section"].(string); ok {
		w.Section = v
	}
	if v, ok := raw["source"].(string); ok {
		w.Source = v
	}
	if v, ok := raw["book"].(string); ok {
		w.Book = v
	}

	// tags
	if tags, ok := raw["tags"].([]interface{}); ok {
		for _, t := range tags {
			if s, ok := t.(string); ok {
				w.Tags = append(w.Tags, s)
			}
		}
	}

	// content: 可能是 []string 或 string
	switch c := raw["content"].(type) {
	case []interface{}:
		for _, v := range c {
			if s, ok := v.(string); ok {
				w.Content = append(w.Content, s)
			}
		}
	case string:
		w.Content = []string{c}
	}

	// paragraphs: []interface{}
	if p, ok := raw["paragraphs"].([]interface{}); ok {
		for _, v := range p {
			if s, ok := v.(string); ok {
				w.Paragraphs = append(w.Paragraphs, s)
			}
		}
	}

	return nil
}

// bookObject 对象型 JSON（有 content 字段，值为章节数组）
type bookObject struct {
	Title    string        `json:"title"`
	Author   string        `json:"author"`
	Dynasty  string        `json:"dynasty"`
	Book     string        `json:"book"`
	Tags     []string      `json:"tags"`
	Abstract string        `json:"abstract"`
	Content  []bookChapter `json:"content"`
}

type bookChapter struct {
	Title      string   `json:"title"`
	Chapter   string   `json:"chapter"`
	Author    string   `json:"author"`
	Source    string   `json:"source"`
	Paragraphs []string `json:"paragraphs"`
}

// baijiaxing 百家姓特殊结构
type baijiaxing struct {
	Title   string       `json:"title"`
	Author  string       `json:"author"`
	Dynasty string       `json:"dynasty"`
	Book    string       `json:"book"`
	Tags    []string     `json:"tags"`
	Origin  []surnameOrigin `json:"origin"`
}

type surnameOrigin struct {
	Surname string `json:"surname"`
	Place   string `json:"place"`
}

// qianziwen 千字文特殊结构（有 spells 和 content）
type qianziwen struct {
	Title   string        `json:"title"`
	Author  string        `json:"author"`
	Dynasty string        `json:"dynasty"`
	Book    string        `json:"book"`
	Tags    []string      `json:"tags"`
	Spells  []string      `json:"spells"`
	Content []bookChapter `json:"content"`
}

// 需要导入的 JSON 文件列表及其分类
var classicsFileCategory = map[string]string{
	"shijing.json":           "诗经",
	"shici.json":             "诗词",
	"chuci.json":             "楚辞",
	"yuanqu.json":            "元曲",
	"yuefu.json":             "乐府",
	"cifu.json":              "辞赋",
	"yijing.json":            "易经",
	"lunyu.json":             "四书",
	"mengzi.json":            "四书",
	"daxue.json":             "四书",
	"zhongyong.json":         "四书",
	"sanzijing-new.json":     "蒙学",
	"sanzijing-traditional.json": "蒙学",
	"baijiaxing.json":        "蒙学",
	"qianziwen.json":         "蒙学",
	"qianjiashi.json":        "蒙学",
	"dizigui.json":           "蒙学",
	"shenglvqimeng.json":     "蒙学",
	"youxueqionglin.json":    "蒙学",
	"zengguangxianwen.json":  "蒙学",
	"guwenguanzhi.json":      "蒙学",
	"zhuzijiaxun.json":       "蒙学",
	"wenzimengqiu.json":      "蒙学",
}

// 排除的文件
var excludedClassicsFiles = map[string]bool{
	"hanzi.json":          true,
	"word.json":           true,
	"curated_names.json":  true,
	"standard_chars.json": true,
	"zodiac.json":         true,
}

// seedClassicsData 将 JSON 文件中的经典数据导入 SQLite
func (s *Store) seedClassicsData(dataDir string) error {
	// 检查是否已有数据
	var count int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM classics_books").Scan(&count); err != nil {
		return fmt.Errorf("检查经典数据失败: %w", err)
	}
	if count > 0 {
		logger.Info("经典数据已存在，跳过导入")
		return nil
	}

	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return fmt.Errorf("读取数据目录失败: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		if excludedClassicsFiles[entry.Name()] {
			continue
		}
		category, ok := classicsFileCategory[entry.Name()]
		if !ok {
			logger.Warn("未分类的 JSON 文件，跳过导入", logger.String("file", entry.Name()))
			continue
		}

		filePath := filepath.Join(dataDir, entry.Name())
		if err := s.importClassicsFile(filePath, entry.Name(), category); err != nil {
			logger.Error("导入经典文件失败",
				logger.String("file", entry.Name()),
				logger.ErrField(err))
			continue
		}
		logger.Info("经典数据导入成功", logger.String("file", entry.Name()))
	}

	return nil
}

// importClassicsFile 导入单个 JSON 文件
func (s *Store) importClassicsFile(filePath, fileName, category string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}

	// 根据文件名选择解析策略
	switch fileName {
	case "baijiaxing.json":
		return s.importBaijiaxing(data, fileName, category)
	case "qianziwen.json":
		return s.importQianziwen(data, fileName, category)
	default:
		// 通用策略：先尝试解析为数组，再尝试解析为对象
		return s.importGeneric(data, fileName, category)
	}
}

// --- 通用导入（覆盖大部分 JSON 文件） ---

func (s *Store) importGeneric(data []byte, fileName, category string) error {
	// 策略1：尝试解析为数组
	var works []arrayWork
	if err := json.Unmarshal(data, &works); err == nil && len(works) > 0 {
		return s.importArrayWorks(works, fileName, category)
	}

	// 策略2：尝试解析为对象（含 content 章节数组）
	var book bookObject
	if err := json.Unmarshal(data, &book); err == nil && len(book.Content) > 0 {
		return s.importBookObject(&book, fileName, category)
	}

	return fmt.Errorf("无法识别的 JSON 结构")
}

// importArrayWorks 处理数组型 JSON（shijing, shici, chuci, yuanqu, yuefu, cifu）
func (s *Store) importArrayWorks(works []arrayWork, fileName, category string) error {
	tx, err := s.writeDB.Begin()
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback()

	// 确定书籍元数据（从第一条推断）
	title := category
	author := ""
	dynasty := ""
	if len(works) > 0 {
		title = category + "合集"
		if works[0].Book != "" {
			title = works[0].Book
		}
		if works[0].Dynasty != "" {
			dynasty = works[0].Dynasty
		}
	}

	// 写入 book
	bookResult, err := tx.Exec(
		`INSERT INTO classics_books (file_name, category, title, author, dynasty)
		 VALUES (?, ?, ?, ?, ?)`,
		fileName, category, title, author, dynasty,
	)
	if err != nil {
		return fmt.Errorf("插入书籍记录失败: %w", err)
	}
	bookID, _ := bookResult.LastInsertId()

	// 准备 section 和 paragraph 插入语句
	secStmt, err := tx.Prepare(
		`INSERT INTO classics_sections (book_id, title, chapter, section, author, source, sort_order)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return fmt.Errorf("预编译 section 插入失败: %w", err)
	}
	defer secStmt.Close()

	paraStmt, err := tx.Prepare(
		`INSERT INTO classics_paragraphs (section_id, content, paragraph_index, char_count)
		 VALUES (?, ?, ?, ?)`,
	)
	if err != nil {
		return fmt.Errorf("预编译 paragraph 插入失败: %w", err)
	}
	defer paraStmt.Close()

	for i, work := range works {
		// 写入 section（每个 array 元素是一条作品/篇章）
		secResult, err := secStmt.Exec(
			bookID, work.Title, work.Chapter, work.Section, work.Author, work.Source, i,
		)
		if err != nil {
			logger.Warn("插入 section 失败，跳过",
				logger.String("title", work.Title),
				logger.ErrField(err))
			continue
		}
		secID, _ := secResult.LastInsertId()

		// 获取段落内容（兼容 content 或 paragraphs 字段）
		lines := work.Content
		if len(lines) == 0 {
			lines = work.Paragraphs
		}
		// 如果 content 是单段长文本（辞赋/乐府），按句号分句
		if len(lines) == 1 && len([]rune(lines[0])) > 100 {
			lines = splitSentences(lines[0])
		}

		for j, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if _, err := paraStmt.Exec(secID, line, j, len([]rune(line))); err != nil {
				logger.Warn("插入 paragraph 失败",
					logger.String("title", work.Title),
					logger.Int("index", j),
					logger.ErrField(err))
				continue
			}
		}
	}

	return tx.Commit()
}

// importBookObject 处理对象型 JSON（lunyu, mengzi, sanzijing, 等）
func (s *Store) importBookObject(book *bookObject, fileName, category string) error {
	tx, err := s.writeDB.Begin()
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback()

	// 写入 book
	bookResult, err := tx.Exec(
		`INSERT INTO classics_books (file_name, category, title, author, dynasty, book, abstract, tags)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		fileName, category, book.Title, book.Author, book.Dynasty,
		book.Book, book.Abstract, strings.Join(book.Tags, ","),
	)
	if err != nil {
		return fmt.Errorf("插入书籍记录失败: %w", err)
	}
	bookID, _ := bookResult.LastInsertId()

	secStmt, err := tx.Prepare(
		`INSERT INTO classics_sections (book_id, title, chapter, author, source, sort_order)
		 VALUES (?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return fmt.Errorf("预编译 section 插入失败: %w", err)
	}
	defer secStmt.Close()

	paraStmt, err := tx.Prepare(
		`INSERT INTO classics_paragraphs (section_id, content, paragraph_index, char_count)
		 VALUES (?, ?, ?, ?)`,
	)
	if err != nil {
		return fmt.Errorf("预编译 paragraph 插入失败: %w", err)
	}
	defer paraStmt.Close()

	for i, ch := range book.Content {
		// 确定章节标题
		secTitle := ch.Chapter
		if secTitle == "" {
			secTitle = ch.Title
		}

		secResult, err := secStmt.Exec(
			bookID, secTitle, ch.Chapter, ch.Author, ch.Source, i,
		)
		if err != nil {
			logger.Warn("插入 section 失败，跳过",
				logger.String("chapter", ch.Chapter),
				logger.ErrField(err))
			continue
		}
		secID, _ := secResult.LastInsertId()

		for j, line := range ch.Paragraphs {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if _, err := paraStmt.Exec(secID, line, j, len([]rune(line))); err != nil {
				logger.Warn("插入 paragraph 失败",
					logger.String("chapter", ch.Chapter),
					logger.Int("index", j),
					logger.ErrField(err))
				continue
			}
		}
	}

	return tx.Commit()
}

// --- 特殊结构导入 ---

// importBaijiaxing 处理百家姓（origin: [{surname, place}]）
func (s *Store) importBaijiaxing(data []byte, fileName, category string) error {
	var book baijiaxing
	if err := json.Unmarshal(data, &book); err != nil {
		return fmt.Errorf("解析百家姓数据失败: %w", err)
	}

	tx, err := s.writeDB.Begin()
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback()

	bookResult, err := tx.Exec(
		`INSERT INTO classics_books (file_name, category, title, author, dynasty, book, tags)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		fileName, category, book.Title, book.Author, book.Dynasty, book.Book,
		strings.Join(book.Tags, ","),
	)
	if err != nil {
		return fmt.Errorf("插入书籍记录失败: %w", err)
	}
	bookID, _ := bookResult.LastInsertId()

	secStmt, err := tx.Prepare(
		`INSERT INTO classics_sections (book_id, title, sort_order)
		 VALUES (?, ?, ?)`,
	)
	if err != nil {
		return fmt.Errorf("预编译 section 插入失败: %w", err)
	}
	defer secStmt.Close()

	paraStmt, err := tx.Prepare(
		`INSERT INTO classics_paragraphs (section_id, content, paragraph_index, char_count)
		 VALUES (?, ?, ?, ?)`,
	)
	if err != nil {
		return fmt.Errorf("预编译 paragraph 插入失败: %w", err)
	}
	defer paraStmt.Close()

	// 按姓氏分节（每 20 个姓氏一节）
	batchSize := 20
	for i := 0; i < len(book.Origin); i += batchSize {
		end := i + batchSize
		if end > len(book.Origin) {
			end = len(book.Origin)
		}
		batch := book.Origin[i:end]

		secResult, err := secStmt.Exec(bookID, fmt.Sprintf("姓氏-%d", i/batchSize+1), i/batchSize)
		if err != nil {
			continue
		}
		secID, _ := secResult.LastInsertId()

		for j, origin := range batch {
			line := fmt.Sprintf("%s（%s）", origin.Surname, origin.Place)
			if _, err := paraStmt.Exec(secID, line, j, len([]rune(line))); err != nil {
				continue
			}
		}
	}

	return tx.Commit()
}

// importQianziwen 处理千字文（有 spells 和 content 双字段）
func (s *Store) importQianziwen(data []byte, fileName, category string) error {
	var book qianziwen
	if err := json.Unmarshal(data, &book); err != nil {
		return fmt.Errorf("解析千字文数据失败: %w", err)
	}

	tx, err := s.writeDB.Begin()
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer tx.Rollback()

	bookResult, err := tx.Exec(
		`INSERT INTO classics_books (file_name, category, title, author, dynasty, book, tags)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		fileName, category, book.Title, book.Author, book.Dynasty, book.Book,
		strings.Join(book.Tags, ","),
	)
	if err != nil {
		return fmt.Errorf("插入书籍记录失败: %w", err)
	}
	bookID, _ := bookResult.LastInsertId()

	secStmt, err := tx.Prepare(
		`INSERT INTO classics_sections (book_id, title, sort_order)
		 VALUES (?, ?, ?)`,
	)
	if err != nil {
		return fmt.Errorf("预编译 section 插入失败: %w", err)
	}
	defer secStmt.Close()

	paraStmt, err := tx.Prepare(
		`INSERT INTO classics_paragraphs (section_id, content, paragraph_index, char_count)
		 VALUES (?, ?, ?, ?)`,
	)
	if err != nil {
		return fmt.Errorf("预编译 paragraph 插入失败: %w", err)
	}
	defer paraStmt.Close()

	// Section 1: 拼音
	secResult1, err := secStmt.Exec(bookID, "千字文·拼音", 0)
	if err == nil {
		secID1, _ := secResult1.LastInsertId()
		for i, spell := range book.Spells {
			if _, err := paraStmt.Exec(secID1, spell, i, len([]rune(spell))); err != nil {
				continue
			}
		}
	}

	// Section 2+: content 章节
	for i, ch := range book.Content {
		secResult, err := secStmt.Exec(bookID, ch.Chapter, i+1)
		if err != nil {
			continue
		}
		secID, _ := secResult.LastInsertId()

		for j, line := range ch.Paragraphs {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if _, err := paraStmt.Exec(secID, line, j, len([]rune(line))); err != nil {
				continue
			}
		}
	}

	return tx.Commit()
}

// --- 工具函数 ---

// splitSentences 按句末标点分句，用于处理单段长文本（cifu, yuefu）
func splitSentences(text string) []string {
	var result []string
	// 按句号、问号、感叹号、分号、换行分句，保留分隔符
	line := strings.Builder{}
	for _, r := range text {
		line.WriteRune(r)
		if r == '。' || r == '？' || r == '！' || r == '\n' || r == '；' {
			s := strings.TrimSpace(line.String())
			if s != "" {
				result = append(result, s)
			}
			line.Reset()
		}
	}
	// 剩余部分
	if remain := strings.TrimSpace(line.String()); remain != "" {
		result = append(result, remain)
	}
	return result
}
