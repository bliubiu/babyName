package sqlite

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"

	"name/internal/domain/classics"
	"name/internal/domain/hanzi"
	"name/internal/infrastructure/logger"
	"name/internal/materials/schema"
)

// 本文件实现 classics 物料的导入（docs/31 ADR-008）。
//
// 与其余物料的两点差异：
//
//  1. 源是 dataDir 下的多个 JSON，而非 schema.Definition 登记的单个 Source。
//     因此版本指纹改用 classicsDataVersion（覆盖全部经典文件的大小）。
//  2. 除 classics_books/sections/paragraphs 三表外，还物化两张运行期消费表：
//     classics_extracts（经典字集）与 classics_poem_entries（诗词索引）。
//     提取逻辑不在此处重写，而是直接调用 classics 域的 LoadFromJSON +
//     ExportExtractSets/ExportPoemEntries —— 与旧运行期 JSON 路径**同一份实现**，
//     物化只是搬运结果，杜绝两套提取规则漂移。

// importClassics 导入经典语料（原三表 + 两张物化表）
func (s *Store) importClassics(def schema.Definition, _ string, res ImportResult, opt ImportOptions) ImportResult {
	dataDir := opt.DataDir
	res.Version = classicsDataVersion(dataDir)

	if opt.DryRun {
		res.Rows = 0
		res.Reason = "试运行：仅统计经典文件，不写库"
		return res
	}

	// 提取字集需用字库补全拼音/释义/五行/性别（extractFromEntry 查 hanzi.GetHanzi）。
	// 导入进程是干净的 CLI 进程，HanziData 必然为空，必须先装字库，
	// 否则物化出的 classics_extracts 四列全空 —— 这是最容易被漏掉的隐性依赖。
	if len(hanzi.HanziData) == 0 {
		if err := hanzi.LoadNamerFromJSON(dataDir); err != nil {
			res.Err = fmt.Sprintf("装载字库以补全经典字集属性失败: %v", err)
			return res
		}
	}

	if err := classics.LoadFromJSON(dataDir); err != nil {
		res.Err = fmt.Sprintf("解析经典语料失败: %v", err)
		return res
	}
	extracts := classics.ExportExtractSets()
	poems := classics.ExportPoemEntries()

	out := s.runImport(def.Name, def.Tables, &res, opt, func(tx *sql.Tx) (int, error) {
		total := 0
		var skipped []string

		// 1. 三表：逐文件导入（复用既有解析策略）
		//
		// 单个文件解析失败**不阻断**整个物料：经典语料是 20+ 个异构 JSON，
		// 其中个别文件存在非标准结构（如 qianjiashi.json 的 paragraphs 元素
		// 为 {subchapter, paragraphs} 嵌套对象）。历史 seedClassicsData 亦为
		// 「记日志后跳过」，此处保持同一语义，但把跳过清单写进 res.Reason，
		// 使其**可见**而非静默。
		for _, fileName := range classicsSourceFiles(dataDir) {
			filePath := filepath.Join(dataDir, fileName)
			rows, err := importClassicsFile(tx, filePath, fileName, classicsFileCategory[fileName])
			if err != nil {
				logger.Warn("经典文件解析失败，已跳过该文件",
					logger.String("file", fileName), logger.ErrField(err))
				skipped = append(skipped, fileName)
				continue
			}
			total += rows
		}
		if len(skipped) > 0 {
			res.Reason = "以下文件结构无法识别已跳过: " + strings.Join(skipped, ",")
			logger.Warn("经典语料存在无法解析的文件", logger.String("文件", strings.Join(skipped, ",")))
		}

		// 2. 字集物化
		estmt, err := tx.Prepare(`INSERT INTO classics_extracts
			(material, seq, char, pinyin, meaning, work, chapter, sentence, wuxing, gender)
			VALUES (?,?,?,?,?,?,?,?,?,?)`)
		if err != nil {
			return 0, err
		}
		defer func() { _ = estmt.Close() }()

		for _, set := range extracts {
			for i, pc := range set.Chars {
				if _, err := estmt.Exec(set.Material, i, pc.Char, pc.Pinyin, pc.Meaning,
					pc.Work, pc.Chapter, pc.Sentence, pc.Wuxing, pc.Gender); err != nil {
					return 0, fmt.Errorf("写入经典字集 %s/%s 失败: %w", set.Material, pc.Char, err)
				}
				total++
			}
		}

		// 3. 诗词索引物化
		pstmt, err := tx.Prepare(`INSERT INTO classics_poem_entries
			(source, title, author, dynasty, type, category, book, tags, content)
			VALUES (?,?,?,?,?,?,?,?,?)`)
		if err != nil {
			return 0, err
		}
		defer func() { _ = pstmt.Close() }()

		// tags/content 均按原顺序落库：normalize.Tags 会重排切片，
		// 而标签顺序（作品名/篇/章）与诗句顺序都是语义的一部分，
		// 排序会让「库路径 == JSON 路径」这条等价性不成立。
		for _, e := range poems {
			if _, err := pstmt.Exec(e.Source, e.Title, e.Author, e.Dynasty, e.Type,
				e.Category, e.Book, mustJSON(e.Tags), mustJSON(e.Content)); err != nil {
				return 0, fmt.Errorf("写入诗词条目《%s》失败: %w", e.Title, err)
			}
			total++
		}

		return total, nil
	})
	if out.Err != "" {
		return *out
	}

	// 4. AUTOINCREMENT 序列归零：保证「同输入多次导入结果等价」（ADR-011）。
	//    sqlite_sequence 不随 DELETE 重置，不归零则二次导入的自增 id 递增，
	//    使幂等比对与「重建即等价」的约定失效。
	for _, t := range []string{"classics_books", "classics_sections", "classics_paragraphs", "classics_poem_entries"} {
		if _, err := s.writeDB.Exec("DELETE FROM sqlite_sequence WHERE name = ?", t); err != nil {
			logger.Warn("sqlite: 重置自增序列失败", logger.String("表", t), logger.ErrField(err))
			break
		}
	}

	return *out
}

// LoadClassicsFromDB 从库装载经典语料（运行期唯一数据源，docs/31 ADR-008）
//
// 覆盖三类运行期状态：18 组经典字集、全局诗词索引。任一为空即视为库未导入，
// 直接返回错误由调用方 fail-fast（不回落 JSON）—— 回落策略见 data 包策略开关。
func (s *Store) LoadClassicsFromDB() error {
	if !s.tableHasRows("classics_extracts") {
		return fmt.Errorf("经典语料库缺失 classics_extracts，请执行 materials import")
	}

	rows, err := s.db.Query(`SELECT material, char, pinyin, meaning, work, chapter,
		sentence, wuxing, gender FROM classics_extracts`)
	if err != nil {
		return fmt.Errorf("查询 classics_extracts 失败: %w", err)
	}
	var extracts []classics.ExtractRow
	for rows.Next() {
		var r classics.ExtractRow
		if err := rows.Scan(&r.Material, &r.Char, &r.Pinyin, &r.Meaning, &r.Work,
			&r.Chapter, &r.Sentence, &r.Wuxing, &r.Gender); err != nil {
			_ = rows.Close()
			return fmt.Errorf("扫描 classics_extracts 行失败: %w", err)
		}
		extracts = append(extracts, r)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return fmt.Errorf("遍历 classics_extracts 失败: %w", err)
	}
	_ = rows.Close()

	if err := classics.LoadExtractSets(extracts); err != nil {
		return err
	}

	poems, err := s.loadPoemEntryRows()
	if err != nil {
		return err
	}
	classics.LoadPoemEntries(poems)

	logger.Info("sqlite: 经典语料已从库载入",
		logger.Int("字集条目数", len(extracts)),
		logger.Int("诗词条目数", len(poems)))
	return nil
}

// loadPoemEntryRows 读出物化的诗词索引条目
func (s *Store) loadPoemEntryRows() ([]classics.PoemEntryRow, error) {
	rows, err := s.db.Query(`SELECT source, title, author, dynasty, type, category, book, tags, content
		FROM classics_poem_entries ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("查询 classics_poem_entries 失败: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []classics.PoemEntryRow
	for rows.Next() {
		var (
			r             classics.PoemEntryRow
			tags, content string
		)
		if err := rows.Scan(&r.Source, &r.Title, &r.Author, &r.Dynasty, &r.Type,
			&r.Category, &r.Book, &tags, &content); err != nil {
			return nil, fmt.Errorf("扫描 classics_poem_entries 行失败: %w", err)
		}
		r.Tags = unmarshalTags(tags)
		r.Content = unmarshalTags(content)
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历 classics_poem_entries 失败: %w", err)
	}
	return out, nil
}