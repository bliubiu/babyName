package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"name/internal/infrastructure/logger"
	"name/internal/materials/schema"
)

// 经典语料解析与落库（docs/31 ADR-008：经典物料接入统一导入器）
//
// 两表设计（docs/34 阶段 E-B）：
//
//	classics_books    - 书籍级元数据（如《诗经》《论语》《唐诗三百首》）
//	classics_sections - 篇章级（国风·周南 / 学而篇）
//
// 原三表中的 classics_paragraphs（段落级全文）已删除：它没有任何运行期读者，
// 却以 11 万余行承载了与 classics_poem_entries.content 重复 42.2% 的诗文文本。
// 诗文全文以 classics_poem_entries 为单一真源，本文件只负责书籍/篇章骨架。
//
// 另有 classics_extracts / classics_poem_entries 两张物化表承载运行期真正
// 消费的经典字集与诗词索引，由 importClassics 在同一事务内一并写入，
// 提取逻辑复用 classics 域的 extractFromEntries（见 classics/snapshot.go）。

// --- 解析用中间结构 ---

// arrayWork 数组型 JSON 中的单条作品
type arrayWork struct {
	Title      string   `json:"title"`
	Author     string   `json:"author"`
	Dynasty    string   `json:"dynasty"`
	Type       string   `json:"type"`
	Chapter    string   `json:"chapter"`
	Section    string   `json:"section"`
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
	Chapter    string   `json:"chapter"`
	Author     string   `json:"author"`
	Source     string   `json:"source"`
	Paragraphs []string `json:"paragraphs"`
}

// baijiaxing 百家姓特殊结构
type baijiaxing struct {
	Title   string          `json:"title"`
	Author  string          `json:"author"`
	Dynasty string          `json:"dynasty"`
	Book    string          `json:"book"`
	Tags    []string        `json:"tags"`
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
	"shijing.json":               "诗经",
	"shici.json":                 "诗词",
	"chuci.json":                 "楚辞",
	"yuanqu.json":                "元曲",
	"yuefu.json":                 "乐府",
	"cifu.json":                  "辞赋",
	"yijing.json":                "易经",
	"lunyu.json":                 "四书",
	"mengzi.json":                "四书",
	"daxue.json":                 "四书",
	"zhongyong.json":             "四书",
	"sanzijing-new.json":         "蒙学",
	"sanzijing-traditional.json": "蒙学",
	"baijiaxing.json":            "蒙学",
	"qianziwen.json":             "蒙学",
	"qianjiashi.json":            "蒙学",
	"dizigui.json":               "蒙学",
	"shenglvqimeng.json":         "蒙学",
	"youxueqionglin.json":        "蒙学",
	"zengguangxianwen.json":      "蒙学",
	"guwenguanzhi.json":          "蒙学",
	"zhuzijiaxun.json":           "蒙学",
	"wenzimengqiu.json":          "蒙学",
}

// classicsSourceFiles 返回待导入的经典文件名（字典序，保证导入结果可复现）
//
// 清单唯一真源为 schema.ClassicsFiles（Definition.Files），本函数只做
// 「清单 → 存在性过滤」：不再反向扫描目录，避免目录里多出的 JSON 被静默
// 多导，也避免清单漏登记导致静默少导。文件缺失时跳过，保持 classics
// Required=false 的可选语义（缺失清单由 materials manifest --check 暴露）。
func classicsSourceFiles(dataDir string) []string {
	def := schema.Registry[schema.ClassicsMaterial]
	out := make([]string, 0, len(def.Files))
	for _, name := range def.Files {
		if _, err := os.Stat(filepath.Join(dataDir, filepath.FromSlash(name))); err != nil {
			continue
		}
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// classicsDataVersion 经典物料的版本指纹：文件名 + 字节大小
//
// 经典物料的「源」是 dataDir 下的多个 JSON（不是 schema.Definition 登记的
// 单个 Source 文件），因此版本指纹不能用通用路径的 res.Checksum，
// 否则任一非主源文件变更都不会触发重灌。
func classicsDataVersion(dataDir string) string {
	files := classicsSourceFiles(dataDir)
	var total int64
	h := new(strings.Builder)
	for _, name := range files {
		info, err := os.Stat(filepath.Join(dataDir, name))
		if err != nil {
			continue
		}
		fmt.Fprintf(h, "%s:%d;", name, info.Size())
		total += info.Size()
	}
	return fmt.Sprintf("n=%d;size=%d;%s", len(files), total, h.String())
}

// importClassicsFile 在给定事务内导入单个经典 JSON 文件的书籍/篇章骨架，返回写入行数
func importClassicsFile(tx *sql.Tx, filePath, fileName, category string) (int, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return 0, fmt.Errorf("读取文件失败: %w", err)
	}

	// 根据文件名选择解析策略
	switch fileName {
	case "baijiaxing.json":
		return importBaijiaxing(tx, data, fileName, category)
	case "qianziwen.json":
		return importQianziwen(tx, data, fileName, category)
	default:
		// 通用策略：先尝试解析为数组，再尝试解析为对象
		return importGeneric(tx, data, fileName, category)
	}
}

// --- 通用导入（覆盖大部分 JSON 文件） ---

func importGeneric(tx *sql.Tx, data []byte, fileName, category string) (int, error) {
	// 策略1：尝试解析为数组
	var works []arrayWork
	if err := json.Unmarshal(data, &works); err == nil && len(works) > 0 {
		return importArrayWorks(tx, works, fileName, category)
	}

	// 策略2：尝试解析为对象（含 content 章节数组）
	var book bookObject
	if err := json.Unmarshal(data, &book); err == nil && len(book.Content) > 0 {
		return importBookObject(tx, &book, fileName, category)
	}

	return 0, fmt.Errorf("无法识别的 JSON 结构")
}

// importArrayWorks 处理数组型 JSON（shijing, shici, chuci, yuanqu, yuefu, cifu）
func importArrayWorks(tx *sql.Tx, works []arrayWork, fileName, category string) (int, error) {
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
		return 0, fmt.Errorf("插入书籍记录失败: %w", err)
	}
	bookID, _ := bookResult.LastInsertId()

	secStmt, err := tx.Prepare(
		`INSERT INTO classics_sections (book_id, title, chapter, section, author, source, sort_order)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return 0, fmt.Errorf("预编译 section 插入失败: %w", err)
	}
	defer func() { _ = secStmt.Close() }()

	rows := 0
	for i, work := range works {
		// 写入 section（每个 array 元素是一条作品/篇章）
		if _, err := secStmt.Exec(
			bookID, work.Title, work.Chapter, work.Section, work.Author, work.Source, i,
		); err != nil {
			logger.Warn("插入 section 失败，跳过",
				logger.String("title", work.Title),
				logger.ErrField(err))
			continue
		}
		rows++
	}

	return rows, nil
}

// importBookObject 处理对象型 JSON（lunyu, mengzi, sanzijing, 等）
func importBookObject(tx *sql.Tx, book *bookObject, fileName, category string) (int, error) {
	// 写入 book
	bookResult, err := tx.Exec(
		`INSERT INTO classics_books (file_name, category, title, author, dynasty, book, abstract, tags)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		fileName, category, book.Title, book.Author, book.Dynasty,
		book.Book, book.Abstract, strings.Join(book.Tags, ","),
	)
	if err != nil {
		return 0, fmt.Errorf("插入书籍记录失败: %w", err)
	}
	bookID, _ := bookResult.LastInsertId()

	secStmt, err := tx.Prepare(
		`INSERT INTO classics_sections (book_id, title, chapter, author, source, sort_order)
		 VALUES (?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return 0, fmt.Errorf("预编译 section 插入失败: %w", err)
	}
	defer func() { _ = secStmt.Close() }()

	rows := 0
	for i, ch := range book.Content {
		// 确定章节标题
		secTitle := ch.Chapter
		if secTitle == "" {
			secTitle = ch.Title
		}
		if _, err := secStmt.Exec(
			bookID, secTitle, ch.Chapter, ch.Author, ch.Source, i,
		); err != nil {
			logger.Warn("插入 section 失败，跳过",
				logger.String("chapter", ch.Chapter),
				logger.ErrField(err))
			continue
		}
		rows++
	}

	return rows, nil
}

// --- 特殊结构导入 ---

// importBaijiaxing 处理百家姓（origin: [{surname, place}]）
//
// 只登记姓氏分节骨架：origin 的「姓（地）」正文原落在已废弃的
// classics_paragraphs，现随该表一并移除（docs/34 阶段 E-B）。
func importBaijiaxing(tx *sql.Tx, data []byte, fileName, category string) (int, error) {
	var book baijiaxing
	if err := json.Unmarshal(data, &book); err != nil {
		return 0, fmt.Errorf("解析百家姓数据失败: %w", err)
	}

	bookResult, err := tx.Exec(
		`INSERT INTO classics_books (file_name, category, title, author, dynasty, book, tags)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		fileName, category, book.Title, book.Author, book.Dynasty, book.Book,
		strings.Join(book.Tags, ","),
	)
	if err != nil {
		return 0, fmt.Errorf("插入书籍记录失败: %w", err)
	}
	bookID, _ := bookResult.LastInsertId()

	secStmt, err := tx.Prepare(
		`INSERT INTO classics_sections (book_id, title, sort_order) VALUES (?, ?, ?)`,
	)
	if err != nil {
		return 0, fmt.Errorf("预编译 section 插入失败: %w", err)
	}
	defer func() { _ = secStmt.Close() }()

	// 按姓氏分节（每 20 个姓氏一节）
	const batchSize = 20
	rows := 0
	for i := 0; i < len(book.Origin); i += batchSize {
		if _, err := secStmt.Exec(bookID, fmt.Sprintf("姓氏-%d", i/batchSize+1), i/batchSize); err != nil {
			continue
		}
		rows++
	}

	return rows, nil
}

// importQianziwen 处理千字文（有 spells 和 content 双字段）
func importQianziwen(tx *sql.Tx, data []byte, fileName, category string) (int, error) {
	var book qianziwen
	if err := json.Unmarshal(data, &book); err != nil {
		return 0, fmt.Errorf("解析千字文数据失败: %w", err)
	}

	bookResult, err := tx.Exec(
		`INSERT INTO classics_books (file_name, category, title, author, dynasty, book, tags)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		fileName, category, book.Title, book.Author, book.Dynasty, book.Book,
		strings.Join(book.Tags, ","),
	)
	if err != nil {
		return 0, fmt.Errorf("插入书籍记录失败: %w", err)
	}
	bookID, _ := bookResult.LastInsertId()

	secStmt, err := tx.Prepare(
		`INSERT INTO classics_sections (book_id, title, sort_order) VALUES (?, ?, ?)`,
	)
	if err != nil {
		return 0, fmt.Errorf("预编译 section 插入失败: %w", err)
	}
	defer func() { _ = secStmt.Close() }()

	rows := 0

	// Section 1: 拼音
	if _, err := secStmt.Exec(bookID, "千字文·拼音", 0); err == nil {
		rows++
	}

	// Section 2+: content 章节
	for i, ch := range book.Content {
		if _, err := secStmt.Exec(bookID, ch.Chapter, i+1); err != nil {
			continue
		}
		rows++
	}

	return rows, nil
}

// 注：原 splitSentences（把辞赋/乐府的单段长文本按句末标点切分）随
// classics_paragraphs 一并移除 —— 它的唯一调用点就是把切分结果写进段落表。
// 诗文全文现以 classics_poem_entries.content 为单一真源，由 classics 域
// 的 extractCharsFromContent / buildSentenceMap 自行按需切句。
