package sqlite

import (
	"database/sql"
	"fmt"
	"strings"

	"name/internal/infrastructure/logger"
)

// 本文件定义物料的规范化表结构（docs/31 ADR-008 / ADR-010）。
//
// 设计要点：
//   - 每种物料一张主表，主键与业务键对齐（char / name / key）；
//   - 所有跨物料可查询的字段都建成独立列，不再依赖 Go 端回查内存二次过滤；
//   - 建表语句幂等（IF NOT EXISTS），缺列由迁移补齐，两条路径都要能跑。

// materialTableDDL 物料表的建表语句（顺序即导入顺序无关，仅为可读性）
var materialTableDDL = []string{
	// 康熙笔画：数理与卦象的笔画口径（AGENTS.md 要求「数理用康熙」）
	`CREATE TABLE IF NOT EXISTS kangxi_strokes (
		char    TEXT PRIMARY KEY,
		strokes INTEGER NOT NULL DEFAULT 0
	)`,

	// 通用规范汉字表：运行时权威字库
	`CREATE TABLE IF NOT EXISTS hanzi_data (
		char       TEXT PRIMARY KEY,
		pinyin     TEXT NOT NULL DEFAULT '',
		strokes    INTEGER NOT NULL DEFAULT 0,
		kangxi_strokes INTEGER NOT NULL DEFAULT 0,
		radical    TEXT NOT NULL DEFAULT '',
		meaning    TEXT NOT NULL DEFAULT '',
		wuxing     TEXT NOT NULL DEFAULT '',
		gender     TEXT NOT NULL DEFAULT '',
		level      INTEGER NOT NULL DEFAULT 0,
		tone       INTEGER NOT NULL DEFAULT 0,
		name_freq_tier INTEGER NOT NULL DEFAULT 0,
		gender_tags     TEXT NOT NULL DEFAULT '[]',
		style_tags      TEXT NOT NULL DEFAULT '[]',
		usage_level     INTEGER NOT NULL DEFAULT 0,
		positive_score  INTEGER NOT NULL DEFAULT 0,
		phonetic_score  INTEGER NOT NULL DEFAULT 0,
		modern_score    INTEGER NOT NULL DEFAULT 0,
		classical_score INTEGER NOT NULL DEFAULT 0,
		is_polyphonic   INTEGER NOT NULL DEFAULT 0,
		is_rare         INTEGER NOT NULL DEFAULT 0,
		is_negative     INTEGER NOT NULL DEFAULT 0,
		pair_blacklist  TEXT NOT NULL DEFAULT '[]',
		curation_level  INTEGER NOT NULL DEFAULT 0,
		name_penalty    INTEGER NOT NULL DEFAULT 0,
		naming_categories TEXT NOT NULL DEFAULT '[]',
		updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE INDEX IF NOT EXISTS idx_hanzi_data_wuxing ON hanzi_data(wuxing)`,
	`CREATE INDEX IF NOT EXISTS idx_hanzi_data_strokes ON hanzi_data(strokes)`,
	`CREATE INDEX IF NOT EXISTS idx_hanzi_data_pinyin ON hanzi_data(pinyin)`,
	`CREATE INDEX IF NOT EXISTS idx_hanzi_data_level ON hanzi_data(level)`,
	`CREATE INDEX IF NOT EXISTS idx_hanzi_data_gender ON hanzi_data(gender)`,

	// 偏旁分组（实测 29 组小表）
	`CREATE TABLE IF NOT EXISTS char_groups (
		radical TEXT PRIMARY KEY,
		name    TEXT NOT NULL DEFAULT '',
		meaning TEXT NOT NULL DEFAULT '',
		chars   TEXT NOT NULL DEFAULT '[]'
	)`,

	// 单字人名频率
	`CREATE TABLE IF NOT EXISTS name_frequency (
		char  TEXT PRIMARY KEY,
		count INTEGER NOT NULL DEFAULT 0,
		rank  INTEGER NOT NULL DEFAULT 0,
		tier  INTEGER NOT NULL DEFAULT 0
	)`,

	// 双字人名频率
	`CREATE TABLE IF NOT EXISTS name_bigram (
		key   TEXT PRIMARY KEY,
		char1 TEXT NOT NULL DEFAULT '',
		char2 TEXT NOT NULL DEFAULT '',
		count INTEGER NOT NULL DEFAULT 0,
		rank  INTEGER NOT NULL DEFAULT 0,
		tier  INTEGER NOT NULL DEFAULT 0
	)`,

	// 命名质量门禁字表（候选池阶段硬剔除）
	`CREATE TABLE IF NOT EXISTS naming_gate (
		char TEXT PRIMARY KEY
	)`,

	// 禁忌双字组合
	`CREATE TABLE IF NOT EXISTS forbidden_combos (
		combo TEXT PRIMARY KEY
	)`,

	// 显式白名单：豁免黑名单（白名单优先）+ 祈福组合（docs/33 门禁层 M1）
	// kind ∈ char（单字级豁免）/ combo（组合级豁免）/ blessing（祈福组合）
	`CREATE TABLE IF NOT EXISTS allowlist (
		entry TEXT PRIMARY KEY,
		kind  TEXT NOT NULL DEFAULT 'combo'
	)`,

	// L5 合规红线：机关单位/争议地名/负面历史人物（docs/33 门禁层 M2）
	// kind ∈ state_org（机关单位）/ place（争议地名）/ person（负面历史人物）
	`CREATE TABLE IF NOT EXISTS compliance_redline (
		entry TEXT PRIMARY KEY,
		kind  TEXT NOT NULL DEFAULT 'state_org'
	)`,

	// 易经六十四卦
	`CREATE TABLE IF NOT EXISTS yijing_hexagrams (
		id             INTEGER PRIMARY KEY,
		name           TEXT NOT NULL DEFAULT '',
		number         INTEGER NOT NULL DEFAULT 0,
		symbol         TEXT NOT NULL DEFAULT '',
		upper_trigram  INTEGER NOT NULL DEFAULT 0,
		lower_trigram  INTEGER NOT NULL DEFAULT 0,
		gua_ci         TEXT NOT NULL DEFAULT '',
		xiang_ci       TEXT NOT NULL DEFAULT '',
		yao_ci         TEXT NOT NULL DEFAULT '[]',
		tuan_ci        TEXT NOT NULL DEFAULT '',
		xiao_xiang     TEXT NOT NULL DEFAULT '[]',
		interpretation TEXT NOT NULL DEFAULT ''
	)`,

	// 十二生肖
	`CREATE TABLE IF NOT EXISTS zodiac (
		id             INTEGER PRIMARY KEY,
		name           TEXT NOT NULL DEFAULT '',
		wuxing         TEXT NOT NULL DEFAULT '',
		compatible     TEXT NOT NULL DEFAULT '[]',
		conflicting    TEXT NOT NULL DEFAULT '[]',
		avoid_chars    TEXT NOT NULL DEFAULT '',
		lucky_numbers  TEXT NOT NULL DEFAULT '[]'
	)`,

	// 经典语料两表（docs/31 ADR-008：经典物料接入统一导入器，
// 建表语句随物料表一并收敛到本文件，不再散落在 createTables）
	//
	//   classics_books      - 书籍级元数据（如《诗经》《论语》）
	//   classics_sections   - 篇章级（国风·周南 / 学而篇）
	//
	// 原先的 classics_paragraphs（段落级全文）已删除（docs/34 阶段 E-B）：
	// 它没有任何运行期读者，却以 114139 行承载了与 classics_poem_entries
	// 重复 42.2% 的诗文文本，实测占库体积 8%。诗文全文以 classics_poem_entries
	// 的 content 为单一真源。
	`CREATE TABLE IF NOT EXISTS classics_books (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		file_name  TEXT NOT NULL,
		category   TEXT NOT NULL,
		title      TEXT NOT NULL,
		author     TEXT,
		dynasty    TEXT,
		book       TEXT,
		abstract   TEXT,
		tags       TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_classics_books_file ON classics_books(file_name)`,
	`CREATE TABLE IF NOT EXISTS classics_sections (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		book_id     INTEGER NOT NULL REFERENCES classics_books(id),
		title       TEXT,
		chapter     TEXT,
		section     TEXT,
		author      TEXT,
		source      TEXT,
		extra       TEXT,
		sort_order  INTEGER DEFAULT 0
	)`,
	`CREATE INDEX IF NOT EXISTS idx_classics_sections_book ON classics_sections(book_id)`,

	// 经典字集物化表：导入期由 extractFromEntries 算出、运行期逐行还原，
	// 使运行期不再需要从 classics_* 三表二次实现提取规则（docs/31 ADR-008）。
	// seq 保留 extractFromEntries 的码点升序（下游候选池注入顺序依赖它）。
	//
	// 只落「诗词侧」属性；pinyin/meaning/wuxing/gender 是 hanzi_data 的列，
	// 不在此反规范化（docs/34 阶段 E-A）：实测 18 组字集把每个汉字平均抄了
	// 3.3 遍，单是 meaning 一列就占全库 11.5%。
	`CREATE TABLE IF NOT EXISTS classics_extracts (
		material TEXT NOT NULL,
		seq      INTEGER NOT NULL DEFAULT 0,
		char     TEXT NOT NULL,
		work     TEXT NOT NULL DEFAULT '',
		chapter  TEXT NOT NULL DEFAULT '',
		sentence TEXT NOT NULL DEFAULT '',
		PRIMARY KEY (material, char)
	)`,

	// 诗词索引物化表：FullText/Chars 均为 content 的纯函数，不落库
	`CREATE TABLE IF NOT EXISTS classics_poem_entries (
		id       INTEGER PRIMARY KEY AUTOINCREMENT,
		source   TEXT NOT NULL DEFAULT '',
		title    TEXT NOT NULL DEFAULT '',
		author   TEXT NOT NULL DEFAULT '',
		dynasty  TEXT NOT NULL DEFAULT '',
		type     TEXT NOT NULL DEFAULT '',
		category TEXT NOT NULL DEFAULT '',
		book     TEXT NOT NULL DEFAULT '',
		tags     TEXT NOT NULL DEFAULT '[]',
		content  TEXT NOT NULL DEFAULT '[]'
	)`,

	// 精选策展名（curated_names.json）：种子行（source=seed）与用户自学习行
	// （source=user_favorite）同表共存。扩列保留策展释义/五行/风格/标签，
	// 使运行时改读库后 SearchCuratedNames / GetStyles 等不退化（docs/31 ADR-008）。
	`CREATE TABLE IF NOT EXISTS curated_names (
		name       TEXT PRIMARY KEY,
		pinyin     TEXT NOT NULL DEFAULT '',
		gender     TEXT NOT NULL DEFAULT '',
		score      REAL NOT NULL DEFAULT 0,
		source     TEXT NOT NULL DEFAULT '',
		meaning    TEXT NOT NULL DEFAULT '',
		wuxing     TEXT NOT NULL DEFAULT '',
		styles     TEXT NOT NULL DEFAULT '[]',
		tags       TEXT NOT NULL DEFAULT '[]',
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,

	// 物料版本与指纹
	`CREATE TABLE IF NOT EXISTS materials_state (
		material   TEXT PRIMARY KEY,
		version    TEXT NOT NULL DEFAULT '',
		checksum   TEXT NOT NULL DEFAULT '',
		row_count  INTEGER NOT NULL DEFAULT 0,
		imported_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,
}

// curatedNamesColumnMigrations curated_names 的增量补列清单：列名 → 完整列定义
//
// 新增列必须同时登记到本表，否则旧库升级后读写不一致（schema 漂移的护栏）。
var curatedNamesColumnMigrations = []struct{ column, definition string }{
	{"meaning", "TEXT NOT NULL DEFAULT ''"},
	{"wuxing", "TEXT NOT NULL DEFAULT ''"},
	{"styles", "TEXT NOT NULL DEFAULT '[]'"},
	{"tags", "TEXT NOT NULL DEFAULT '[]'"},
}

// yijingHexagramsColumnMigrations yijing_hexagrams 的增量补列清单：
// 彖传/小象传自 docs/玄学/易经 语料结构化后加入（docs/42 P1）
var yijingHexagramsColumnMigrations = []struct{ column, definition string }{
	{"tuan_ci", "TEXT NOT NULL DEFAULT ''"},
	{"xiao_xiang", "TEXT NOT NULL DEFAULT '[]'"},
}

// materialTables 全部物料表名（守护测试与文档用）
var materialTables = []string{
	"kangxi_strokes", "hanzi_data", "char_groups",
	"name_frequency", "name_bigram", "naming_gate", "forbidden_combos",
	"allowlist", "compliance_redline", "yijing_hexagrams", "zodiac",
	"curated_names",
	"classics_books", "classics_sections", "classics_extracts", "classics_poem_entries",
	"materials_state",
}

// createMaterialTables 建立全部物料表（幂等）
func createMaterialTables(db *sql.DB) error {
	for _, ddl := range materialTableDDL {
		if _, err := db.Exec(ddl); err != nil {
			return fmt.Errorf("创建物料表失败: %w", err)
		}
	}
	// 存量库补列：CREATE TABLE IF NOT EXISTS 不会给已存在的表加列（docs/31 ADR-012）。
	// 补列随物料表结构就绪而执行，保证「新增列登记在补列清单」这一
	// 单一真源与建表语句在同一处维护。
	if err := migrateHanziDataColumns(db); err != nil {
		return err
	}
	if err := migrateCuratedNamesColumns(db); err != nil {
		return err
	}
	if err := migrateYijingHexagramsColumns(db); err != nil {
		return err
	}
	if err := migrateClassicsSchema(db); err != nil {
		return err
	}
	return nil
}

// migrateYijingHexagrams 给已存在的 yijing_hexagrams 表补齐缺失的新列
// （彖传/小象传，docs/42 P1）。先探测列是否存在再 ALTER，幂等可重入。
func migrateYijingHexagramsColumns(db *sql.DB) error {
	existing, err := tableColumns(db, "yijing_hexagrams")
	if err != nil {
		return err
	}
	for _, m := range yijingHexagramsColumnMigrations {
		if existing[m.column] {
			continue
		}
		stmt := fmt.Sprintf("ALTER TABLE yijing_hexagrams ADD COLUMN %s %s", m.column, m.definition)
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("yijing_hexagrams 补列 %s 失败: %w", m.column, err)
		}
	}
	return nil
}

// classicExtractsLegacyColumns classics_extracts 已废弃的冗余列
//
// 这四列是 hanzi_data 的反规范化副本（docs/34 阶段 E-A）。它们携带的 meaning
// 还与 hanzi_data 存在口径差异（物化时取自未截断的原始 namer.json，字库表
// 则是 200 rune 截断版），一并消除。
var classicExtractsLegacyColumns = []string{"pinyin", "meaning", "wuxing", "gender"}

// migrateClassicsSchema 经典物料表结构演进：去冗余列 + 清理已废弃表
//
//  1. classics_extracts 若仍带 legacy 列，按「建新表 → 迁数据 → 换名」重建。
//     用重建而非 ALTER DROP COLUMN：后者对被索引/被外键引用的列有限制，
//     跨 SQLite 版本行为也不一致。
//  2. classics_paragraphs 已整体废弃（无运行期读者 + 42.2% 文本重复），
//     直接 DROP。该语句幂等，新库上是无操作。
func migrateClassicsSchema(db *sql.DB) error {
	existing, err := tableColumns(db, "classics_extracts")
	if err != nil {
		return err
	}

	changed := false

	needsRebuild := false
	for _, c := range classicExtractsLegacyColumns {
		if existing[c] {
			needsRebuild = true
			break
		}
	}
	if needsRebuild {
		// 四步重建包单事务（docs/43 P1-6）：SQLite 的 DDL（CREATE/DROP/ALTER）
		// 可回滚，任一步失败即整体回滚——旧表数据完好、无 _new 孤儿表残留。
		// 原实现逐条 Exec 自动提交，中途失败时 CREATE 出的 _new 表已落库，
		// 残留表会阻断下次重建（再次 CREATE 报 already exists）。
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("开启重建事务失败: %w", err)
		}
		for _, q := range []string{
			`CREATE TABLE classics_extracts_new (
				material TEXT NOT NULL,
				seq      INTEGER NOT NULL DEFAULT 0,
				char     TEXT NOT NULL,
				work     TEXT NOT NULL DEFAULT '',
				chapter  TEXT NOT NULL DEFAULT '',
				sentence TEXT NOT NULL DEFAULT '',
				PRIMARY KEY (material, char)
			)`,
			`INSERT OR REPLACE INTO classics_extracts_new (material, seq, char, work, chapter, sentence)
			 SELECT material, seq, char, work, chapter, sentence FROM classics_extracts`,
			`DROP TABLE classics_extracts`,
			`ALTER TABLE classics_extracts_new RENAME TO classics_extracts`,
		} {
			if _, err := tx.Exec(q); err != nil {
				_ = tx.Rollback()
				return fmt.Errorf("重建 classics_extracts 失败（已整体回滚）（%s）: %w", q, err)
			}
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("提交重建事务失败: %w", err)
		}
		changed = true
		logger.Info("sqlite: classics_extracts 已去除冗余列",
			logger.String("移除", strings.Join(classicExtractsLegacyColumns, ",")))
	}

	// 幂等语句；先探测表是否真的存在，避免每次启动都白跑一次 DROP + VACUUM
	if hasTable(db, "classics_paragraphs") {
		if _, err := db.Exec(`DROP TABLE classics_paragraphs`); err != nil {
			return fmt.Errorf("清理已废弃的 classics_paragraphs 失败: %w", err)
		}
		changed = true
		logger.Info("sqlite: 已删除无读取方的 classics_paragraphs")
	}

	if !changed {
		return nil
	}

	// 结构变更后回收空间：DROP TABLE / 重建表只把页还给 freelist，文件不会变小。
	// 实测删除 classics_paragraphs + 压缩 extracts 后空闲页达 56.6%（48MB → 51MB
	// 不降反升）。VACUUM 重建库文件回收；因只在结构真的变化时执行，属一次性成本。
	if _, err := db.Exec(`VACUUM`); err != nil {
		// VACUUM 失败不影响正确性（只是文件没变小），仅告警
		logger.Warn("sqlite: 经典表结构迁移后回收空间失败（库文件未缩小）", logger.ErrField(err))
	}
	return nil
}

// hasTable 判断表是否存在
func hasTable(db *sql.DB, table string) bool {
	var n int
	return db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&n) == nil && n > 0
}

// tableColumns 读出表的列名集合（表不存在时返回空集合，不报错）
func tableColumns(db *sql.DB, table string) (map[string]bool, error) {
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return nil, fmt.Errorf("读取 %s 表结构失败: %w", table, err)
	}
	defer func() { _ = rows.Close() }()

	out := make(map[string]bool)
	for rows.Next() {
		var (
			cid       int
			col       string
			ctype     sql.NullString
			notNull   sql.NullInt64
			dfltValue sql.NullString
			pk        sql.NullInt64
		)
		if err := rows.Scan(&cid, &col, &ctype, &notNull, &dfltValue, &pk); err != nil {
			return nil, fmt.Errorf("解析 %s 表结构失败: %w", table, err)
		}
		out[col] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历 %s 表结构失败: %w", table, err)
	}
	return out, nil
}

// migrateCuratedNamesColumns 给已存在的 curated_names 表补齐缺失的新列
//
// 先读 PRAGMA table_info 判断列是否已存在再 ALTER，避免重复加列报错。
// 覆盖「早期仅 5 列的库」升级到「扩列保真」后的 schema 一致性。
func migrateCuratedNamesColumns(db *sql.DB) error {
	rows, err := db.Query("PRAGMA table_info(curated_names)")
	if err != nil {
		return fmt.Errorf("读取 curated_names 表结构失败: %w", err)
	}
	existing := make(map[string]bool)
	for rows.Next() {
		var (
			cid       int
			col       string
			ctype     sql.NullString
			notNull   sql.NullInt64
			dfltValue sql.NullString
			pk        sql.NullInt64
		)
		if err := rows.Scan(&cid, &col, &ctype, &notNull, &dfltValue, &pk); err != nil {
			_ = rows.Close()
			return fmt.Errorf("解析 curated_names 表结构失败: %w", err)
		}
		existing[col] = true
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return fmt.Errorf("遍历 curated_names 表结构失败: %w", err)
	}
	_ = rows.Close()

	for _, m := range curatedNamesColumnMigrations {
		if existing[m.column] {
			continue
		}
		stmt := fmt.Sprintf("ALTER TABLE curated_names ADD COLUMN %s %s", m.column, m.definition)
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("curated_names 补列 %s 失败: %w", m.column, err)
		}
	}
	return nil
}

// MaterialTables 返回全部物料表名
func MaterialTables() []string {
	out := make([]string, len(materialTables))
	copy(out, materialTables)
	return out
}
