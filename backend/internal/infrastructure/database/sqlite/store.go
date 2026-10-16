package sqlite

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
	"name/internal/domain/hanzi"
	"name/internal/domain/name"
	"name/internal/domain/yijing"
	"name/internal/domain/zodiac"
	"name/internal/infrastructure/database"
	"name/internal/infrastructure/logger"
)

// metaKeyHanziChecksum data_meta 中记录 hanzi_data 物料指纹的 key（ADR-014）
//
// data_meta 是 key/value/updated_at 结构（无独立 checksum 列），
// 故按约定 key 存放每物料的版本与指纹
const metaKeyHanziChecksum = "materials.hanzi.checksum"

type Store struct {
	db        *sql.DB // 读连接池（WAL 模式下可并发读）
	writeDB   *sql.DB // 写连接（单连接，串行写）
	Hexagrams []yijing.Hexagram
	Zodiacs   []zodiac.Zodiac
}

var _ database.Store = (*Store)(nil)

func NewStore(dbPath string, dataDir string) (*Store, error) {
	// 写连接：单连接，负责所有写操作
	writeDB, err := sql.Open("sqlite", dbPath+"?_txlock=immediate")
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite write connection: %w", err)
	}
	// 任一后续步骤失败都必须关闭写连接（docs/29 P3）：否则进程内残留
	// 持有 namer.db 句柄的连接，Windows 上锁住库文件，且上层回退内存
	// 存储后服务照常运行，泄漏不可见。
	var readDB *sql.DB
	success := false
	defer func() {
		if !success {
			_ = writeDB.Close()
			if readDB != nil {
				_ = readDB.Close()
			}
		}
	}()
	writeDB.SetMaxOpenConns(1) // 写必须串行
	writeDB.SetMaxIdleConns(1)
	writeDB.SetConnMaxLifetime(0)

	if err := writeDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping sqlite write connection: %w", err)
	}

	// 启用 WAL 模式
	if _, err := writeDB.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}
	// 设置 WAL checkpoint 策略
	if _, err := writeDB.Exec("PRAGMA wal_autocheckpoint=1000"); err != nil {
		return nil, fmt.Errorf("failed to set wal_autocheckpoint: %w", err)
	}
	// 验证 WAL 模式是否生效
	var journalMode string
	if err := writeDB.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil || journalMode != "wal" {
		logger.Warn("sqlite: WAL mode may not be enabled", logger.String("journal_mode", journalMode))
	}

	// 读连接池：WAL 模式下读不阻塞写，可并发
	readDB, err = sql.Open("sqlite", dbPath+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite read connection: %w", err)
	}
	readDB.SetMaxOpenConns(16) // 允许多个并发读（WAL 模式下读不阻塞写，提高吞吐）
	readDB.SetMaxIdleConns(8)
	readDB.SetConnMaxLifetime(0)

	if err := readDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping sqlite read connection: %w", err)
	}
	// ★ 不在只读连接上执行 PRAGMA journal_mode=WAL（docs/28 W7）。
	//
	// 原审查记录称该语句「必然失败，每次启动打一条噪音 Warn」。
	// 实测（modernc.org/sqlite，探针见下方说明）**两个判断都不成立**，真实情况更严重：
	//
	//   - 语句**不会失败**：`Exec` 返回 nil，无任何 Warn 输出；
	//   - 语句**真的会写文件**：以 `mode=ro` 打开、库原处 delete 模式时执行
	//     `PRAGMA journal_mode=WAL`，SQLite 文件头偏移 18/19 字节由 (1,1) 变为 (2,2)
	//     ——即 legacy → WAL，**文件被实际改写**。实测数据：
	//         原始 文件头[18]=1 [19]=1  →  事后 文件头[18]=2 [19]=2
	//         新只读连接读到 journal_mode = "wal"
	//
	// 也就是说：问题不是「打噪音日志」，而是「一个显式声明只读的连接在悄悄写库」。
	// 这条语句在本函数里恰好无害（上面写连接已把库设为 WAL，此处退化为 no-op），
	// 但它是一个**依赖调用顺序才安全的陷阱**：任何「先建只读连接、后建写连接」
	// 的重构、或对非 WAL 库（如外部只读副本）的误用，都会变成真实的数据写入。
	//
	// 因此删除该语句，改为**只读校验**：读回 journal_mode 确认与写连接一致。
	// 若不一致说明两个连接（意外地）指向了不同文件，属真正需要告警的情况。
	var readJournalMode string
	if err := readDB.QueryRow("PRAGMA journal_mode").Scan(&readJournalMode); err != nil {
		logger.Warn("sqlite: 读取只读连接的 journal_mode 失败", logger.ErrField(err))
	} else if readJournalMode != "wal" {
		logger.Warn("sqlite: 只读连接的 journal_mode 与预期不符（读写连接可能指向了不同文件）",
			logger.String("journal_mode", readJournalMode))
	}

	if err := createTables(writeDB); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	// 物料表结构随库创建而就绪（表空；数据由 `materials import` 填充）
	//
	// 与 createTables 分开：物料表是「导入器的产物」，集中在 store_materials_schema.go 定义，
	// 避免建表语句散落在两个文件里。
	if err := createMaterialTables(writeDB); err != nil {
		return nil, err
	}

	store := &Store{
		db:        readDB,
		writeDB:   writeDB,
		Hexagrams: yijing.GetAllHexagrams(),
		Zodiacs:   zodiac.GetAllZodiacs(),
	}

	// 导入经典文化数据到 SQLite（诗词、易经、蒙学等）
	if err := store.seedClassicsData(dataDir); err != nil {
		logger.Error("sqlite: seed classics data failed", logger.ErrField(err))
	}

	// 导入精选名库种子数据（仅在表为空时）
	store.seedCuratedNames(dataDir)

	// 注意：字库与康熙笔画**不再**在此处 seed。
	//
	// 数据方向已由「内存 → 库」反转为「库 → 内存」（docs/31 ADR-008）：
	// 物料由 `materials import` 显式写入，运行时装载时只从库读出 typed 结构。
	// 保留旧的 seed 会造成「启动即用内存覆盖库」的回路，使物料修订被静默回滚。

	success = true
	return store, nil
}

func createTables(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS history (
			id TEXT PRIMARY KEY,
			surname TEXT NOT NULL,
			gender TEXT NOT NULL,
			birth_date TEXT NOT NULL,
			birth_time TEXT NOT NULL,
			birth_location TEXT,
			results TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL
		)`,
		// score 用 REAL：综合评分是 0-100 的浮点（如 92.7），早前声明为 INTEGER
		// 时前端原样提交的浮点会被 Go 的 int 字段拒绝（400）。
		// 存量库仍是 INTEGER 亲和，SQLite 对无法无损转整数的值本就按 REAL 存储，
		// 因此无需迁移即可读写小数。
		`CREATE TABLE IF NOT EXISTS favorites (
			id TEXT PRIMARY KEY,
			surname TEXT NOT NULL,
			given_name TEXT NOT NULL,
			pinyin TEXT NOT NULL,
			gender TEXT NOT NULL,
			score REAL NOT NULL,
			source TEXT,
			notes TEXT,
			created_at TIMESTAMP NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_favorites_name ON favorites(surname, given_name)`,
		`CREATE INDEX IF NOT EXISTS idx_history_created_at ON history(created_at)`,
		`CREATE TABLE IF NOT EXISTS name_requests (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			surname TEXT NOT NULL,
			gender TEXT NOT NULL,
			birth_year INTEGER,
			birth_month INTEGER,
			birth_day INTEGER,
			birth_hour INTEGER,
			preferences TEXT,
			name_type TEXT,
			request_hash TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_name_requests_hash ON name_requests(request_hash)`,
		`CREATE TABLE IF NOT EXISTS name_feedback (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			request_id INTEGER NOT NULL,
			full_name TEXT NOT NULL,
			given_name TEXT NOT NULL,
			is_liked BOOLEAN NOT NULL,
			is_selected BOOLEAN NOT NULL,
			user_rating INTEGER,
			feedback_text TEXT,
			algorithm_score REAL NOT NULL,
			wuxing_match TEXT,
			device_info TEXT,
			created_at TIMESTAMP NOT NULL,
			FOREIGN KEY (request_id) REFERENCES name_requests(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_name_feedback_request ON name_feedback(request_id)`,

		// 经典数据版本元信息（JSON 热更新对拍）
		`CREATE TABLE IF NOT EXISTS data_meta (
			key        TEXT PRIMARY KEY,
			value      TEXT NOT NULL,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// 经典数据表（诗词、易经、蒙学等）
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
		`CREATE TABLE IF NOT EXISTS classics_paragraphs (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			section_id      INTEGER NOT NULL REFERENCES classics_sections(id),
			content         TEXT NOT NULL,
			paragraph_index INTEGER DEFAULT 0,
			char_count      INTEGER DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_classics_paragraphs_section ON classics_paragraphs(section_id)`,
		`CREATE TABLE IF NOT EXISTS curated_names (
			name       TEXT PRIMARY KEY,
			pinyin     TEXT NOT NULL DEFAULT '',
			gender     TEXT NOT NULL DEFAULT '',
			score      REAL NOT NULL DEFAULT 0,
			source     TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		// 汉字字库缓存表（从 hanzi.HanziData 导入，支持热重载）
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
			usage_level     INTEGER NOT NULL DEFAULT 3,
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

		// 门禁复审队列（docs/33 门禁层 M6）：bad-case 回流闭环的人审状态机。
		// status ∈ pending（待复审）/ confirmed（确认违规）/ false_positive（误报）/
		// border（边界）；key 为 名字+层级+类别+命中值，保证候选幂等去重。
		`CREATE TABLE IF NOT EXISTS gate_review (
			id          TEXT PRIMARY KEY,
			name        TEXT NOT NULL DEFAULT '',
			layer       TEXT NOT NULL DEFAULT '',
			category    TEXT NOT NULL DEFAULT '',
			level       TEXT NOT NULL DEFAULT '',
			hit_value   TEXT NOT NULL DEFAULT '',
			status      TEXT NOT NULL DEFAULT 'pending',
			note        TEXT NOT NULL DEFAULT '',
			created_at  TIMESTAMP NOT NULL,
			reviewed_at TIMESTAMP
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_gate_review_key
			ON gate_review(name, layer, category, hit_value)`,
		`CREATE INDEX IF NOT EXISTS idx_gate_review_status ON gate_review(status)`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}
	}

	// 存量库补列：CREATE TABLE IF NOT EXISTS 不会给已存在的表加列（docs/31 ADR-012）
	if err := migrateHanziDataColumns(db); err != nil {
		return err
	}
	return nil
}

// hanziDataColumnMigrations hanzi_data 的增量补列清单：列名 → 完整列定义
//
// 新增列必须同时登记到本表，否则旧库升级后读写不一致（schema 漂移的护栏）
var hanziDataColumnMigrations = []struct{ column, definition string }{
	{"kangxi_strokes", "INTEGER NOT NULL DEFAULT 0"},
	{"level", "INTEGER NOT NULL DEFAULT 0"},
	{"name_freq_tier", "INTEGER NOT NULL DEFAULT 0"},
}

// migrateHanziDataColumns 给已存在的 hanzi_data 表补齐缺失的新列
//
// 先读 PRAGMA table_info 判断列是否已存在再 ALTER，
// 避免「重复加列」直接报错导致启动失败
func migrateHanziDataColumns(db *sql.DB) error {
	rows, err := db.Query("PRAGMA table_info(hanzi_data)")
	if err != nil {
		return fmt.Errorf("读取 hanzi_data 表结构失败: %w", err)
	}
	existing := make(map[string]bool)
	for rows.Next() {
		var (
			cid       int
			name      string
			ctype     sql.NullString
			notNull   sql.NullInt64
			dfltValue sql.NullString
			pk        sql.NullInt64
		)
		if err := rows.Scan(&cid, &name, &ctype, &notNull, &dfltValue, &pk); err != nil {
			_ = rows.Close()
			return fmt.Errorf("解析 hanzi_data 表结构失败: %w", err)
		}
		existing[name] = true
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return fmt.Errorf("遍历 hanzi_data 表结构失败: %w", err)
	}
	_ = rows.Close()

	for _, m := range hanziDataColumnMigrations {
		if existing[m.column] {
			continue
		}
		stmt := fmt.Sprintf("ALTER TABLE hanzi_data ADD COLUMN %s %s", m.column, m.definition)
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("hanzi_data 补列 %s 失败: %w", m.column, err)
		}
		logger.Info("sqlite: hanzi_data 已补列", logger.String("列", m.column))
	}
	return nil
}

func (s *Store) Close() error {
	err1 := s.writeDB.Close()
	err2 := s.db.Close()
	if err1 != nil {
		return err1
	}
	return err2
}

// --- HistoryStore ---

func (s *Store) SaveHistory(record *database.HistoryRecord) string {
	if record.ID == "" {
		record.ID = uuid.New().String()
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now()
	}

	_, err := s.writeDB.Exec(
		`INSERT INTO history (id, surname, gender, birth_date, birth_time, birth_location, results, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET surname=excluded.surname, gender=excluded.gender,
		   birth_date=excluded.birth_date, birth_time=excluded.birth_time,
		   birth_location=excluded.birth_location, results=excluded.results, created_at=excluded.created_at`,
		record.ID, record.Surname, record.Gender, record.BirthDate,
		record.BirthTime, record.BirthLocation, record.Results, record.CreatedAt,
	)
	if err != nil {
		logger.Error("sqlite: SaveHistory failed", logger.ErrField(err))
		// 失败时返回空字符串，让上层感知错误
		return ""
	}
	return record.ID
}

func (s *Store) BatchSaveHistory(records []*database.HistoryRecord) []string {
	ids := make([]string, len(records))
	tx, err := s.writeDB.Begin()
	if err != nil {
		logger.Error("sqlite: BatchSaveHistory begin tx failed", logger.ErrField(err))
		return ids
	}

	stmt, err := tx.Prepare(
		`INSERT INTO history (id, surname, gender, birth_date, birth_time, birth_location, results, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET surname=excluded.surname, gender=excluded.gender,
		   birth_date=excluded.birth_date, birth_time=excluded.birth_time,
		   birth_location=excluded.birth_location, results=excluded.results, created_at=excluded.created_at`,
	)
	if err != nil {
		logger.Error("sqlite: BatchSaveHistory prepare failed", logger.ErrField(err))
		_ = tx.Rollback()
		return ids
	}

	var failedIndexes []int
	for i, record := range records {
		if record.ID == "" {
			record.ID = uuid.New().String()
		}
		if record.CreatedAt.IsZero() {
			record.CreatedAt = time.Now()
		}
		_, err := stmt.Exec(record.ID, record.Surname, record.Gender, record.BirthDate,
			record.BirthTime, record.BirthLocation, record.Results, record.CreatedAt)
		if err != nil {
			logger.Error("sqlite: BatchSaveHistory exec failed", logger.ErrField(err))
			failedIndexes = append(failedIndexes, i)
			continue
		}
		ids[i] = record.ID
	}
	_ = stmt.Close()

	if err := tx.Commit(); err != nil {
		logger.Error("sqlite: BatchSaveHistory commit failed", logger.ErrField(err))
		return ids
	}

	// 事务提交成功后，对失败记录通过单独连接重试（避免单连接死锁）
	for _, i := range failedIndexes {
		ids[i] = s.SaveHistory(records[i])
	}
	return ids
}

func (s *Store) GetHistory() []*database.HistoryRecord {
	// 默认限制返回100条记录，防止内存溢出
	return s.GetHistoryPageWithLimit(1, 100)
}

// GetHistoryPageWithLimit 获取分页历史记录（内部方法）
func (s *Store) GetHistoryPageWithLimit(page, limit int) []*database.HistoryRecord {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * limit
	rows, err := s.db.Query(`SELECT id, surname, gender, birth_date, birth_time, birth_location, results, created_at FROM history ORDER BY created_at DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		logger.Error("sqlite: GetHistoryPageWithLimit failed", logger.ErrField(err))
		return nil
	}
	defer rows.Close()

	var records []*database.HistoryRecord
	for rows.Next() {
		var r database.HistoryRecord
		if err := rows.Scan(&r.ID, &r.Surname, &r.Gender, &r.BirthDate, &r.BirthTime, &r.BirthLocation, &r.Results, &r.CreatedAt); err != nil {
			logger.Error("sqlite: GetHistoryPageWithLimit scan failed", logger.ErrField(err))
			continue
		}
		records = append(records, &r)
	}
	if err := rows.Err(); err != nil {
		// 迭代中途的 IO/解码错误此前被当成「正常取完」，静默截断数据（docs/29 P3）
		logger.Error("sqlite: rows iteration failed mid-scan", logger.ErrField(err))
	}

	return records
}

func (s *Store) GetHistoryPage(page, limit int) ([]*database.HistoryRecord, int, error) {
	var total int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM history`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count history: %w", err)
	}

	offset := (page - 1) * limit
	rows, err := s.db.Query(`SELECT id, surname, gender, birth_date, birth_time, birth_location, results, created_at FROM history ORDER BY created_at DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query history page: %w", err)
	}
	defer rows.Close()

	var records []*database.HistoryRecord
	for rows.Next() {
		var r database.HistoryRecord
		if err := rows.Scan(&r.ID, &r.Surname, &r.Gender, &r.BirthDate, &r.BirthTime, &r.BirthLocation, &r.Results, &r.CreatedAt); err != nil {
			logger.Error("sqlite: GetHistoryPage scan failed", logger.ErrField(err))
			continue
		}
		records = append(records, &r)
	}
	if err := rows.Err(); err != nil {
		// 迭代中途的 IO/解码错误此前被当成「正常取完」，静默截断数据（docs/29 P3）
		logger.Error("sqlite: rows iteration failed mid-scan", logger.ErrField(err))
	}

	return records, total, nil
}

func (s *Store) GetHistoryByID(id string) *database.HistoryRecord {
	var r database.HistoryRecord
	err := s.db.QueryRow(`SELECT id, surname, gender, birth_date, birth_time, birth_location, results, created_at FROM history WHERE id = ?`, id).
		Scan(&r.ID, &r.Surname, &r.Gender, &r.BirthDate, &r.BirthTime, &r.BirthLocation, &r.Results, &r.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		logger.Error("sqlite: GetHistoryByID failed", logger.ErrField(err))
		return nil
	}
	return &r
}

func (s *Store) DeleteHistory(id string) error {
	_, err := s.writeDB.Exec(`DELETE FROM history WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete history: %w", err)
	}
	return nil
}

func (s *Store) BatchDeleteHistory(ids []string) error {
	tx, err := s.writeDB.Begin()
	if err != nil {
		return fmt.Errorf("batch delete history begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(`DELETE FROM history WHERE id = ?`)
	if err != nil {
		return fmt.Errorf("batch delete history prepare: %w", err)
	}
	defer stmt.Close()

	for _, id := range ids {
		if _, err := stmt.Exec(id); err != nil {
			return fmt.Errorf("batch delete history exec: %w", err)
		}
	}
	return tx.Commit()
}

// --- FavoriteStore ---

func (s *Store) SaveFavorite(record *database.FavoriteRecord) (string, error) {
	if record.ID == "" {
		record.ID = uuid.New().String()
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now()
	}

	_, err := s.writeDB.Exec(
		`INSERT INTO favorites (id, surname, given_name, pinyin, gender, score, source, notes, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET surname=excluded.surname, given_name=excluded.given_name,
		   pinyin=excluded.pinyin, gender=excluded.gender, score=excluded.score,
		   source=excluded.source, notes=excluded.notes, created_at=excluded.created_at`,
		record.ID, record.Surname, record.GivenName, record.Pinyin, record.Gender,
		record.Score, record.Source, record.Notes, record.CreatedAt,
	)
	if err != nil {
		logger.Error("sqlite: SaveFavorite failed", logger.ErrField(err))
		// 写入失败必须把错误上抛（docs/29 A6）：此前吞错仍返回 ID，
		// 服务层据此返回成功，收藏静默丢失且自学习照常入精选库
		return "", err
	}
	return record.ID, nil
}

func (s *Store) BatchSaveFavorite(records []*database.FavoriteRecord) ([]string, error) {
	ids := make([]string, len(records))
	tx, err := s.writeDB.Begin()
	if err != nil {
		logger.Error("sqlite: BatchSaveFavorite begin tx failed", logger.ErrField(err))
		return ids, err
	}

	stmt, err := tx.Prepare(
		`INSERT INTO favorites (id, surname, given_name, pinyin, gender, score, source, notes, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET surname=excluded.surname, given_name=excluded.given_name`,
	)
	if err != nil {
		logger.Error("sqlite: BatchSaveFavorite prepare failed", logger.ErrField(err))
		_ = tx.Rollback()
		return ids, err
	}

	var firstErr error
	var failedIndexes []int
	for i, record := range records {
		if record.ID == "" {
			record.ID = uuid.New().String()
		}
		if record.CreatedAt.IsZero() {
			record.CreatedAt = time.Now()
		}
		_, err := stmt.Exec(record.ID, record.Surname, record.GivenName, record.Pinyin,
			record.Gender, record.Score, record.Source, record.Notes, record.CreatedAt)
		if err != nil {
			logger.Error("sqlite: BatchSaveFavorite exec failed", logger.ErrField(err))
			if firstErr == nil {
				firstErr = err
			}
			failedIndexes = append(failedIndexes, i)
			continue
		}
		ids[i] = record.ID
	}
	_ = stmt.Close()

	if err := tx.Commit(); err != nil {
		logger.Error("sqlite: BatchSaveFavorite commit failed", logger.ErrField(err))
		return ids, err
	}

	// 事务提交成功后，对失败记录通过单独连接重试（避免单连接死锁）
	for _, i := range failedIndexes {
		id, err := s.SaveFavorite(records[i])
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		ids[i] = id
	}
	return ids, firstErr
}

func (s *Store) GetFavorites() []*database.FavoriteRecord {
	rows, err := s.db.Query(`SELECT id, surname, given_name, pinyin, gender, score, source, notes, created_at FROM favorites ORDER BY created_at DESC`)
	if err != nil {
		logger.Error("sqlite: GetFavorites failed", logger.ErrField(err))
		return nil
	}
	defer rows.Close()

	var records []*database.FavoriteRecord
	for rows.Next() {
		var r database.FavoriteRecord
		if err := rows.Scan(&r.ID, &r.Surname, &r.GivenName, &r.Pinyin, &r.Gender, &r.Score, &r.Source, &r.Notes, &r.CreatedAt); err != nil {
			logger.Error("sqlite: GetFavorites scan failed", logger.ErrField(err))
			continue
		}
		records = append(records, &r)
	}
	if err := rows.Err(); err != nil {
		// 迭代中途的 IO/解码错误此前被当成「正常取完」，静默截断数据（docs/29 P3）
		logger.Error("sqlite: rows iteration failed mid-scan", logger.ErrField(err))
	}

	return records
}

func (s *Store) GetFavoritesPage(page, limit int) ([]*database.FavoriteRecord, int, error) {
	var total int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM favorites`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count favorites: %w", err)
	}

	offset := (page - 1) * limit
	rows, err := s.db.Query(`SELECT id, surname, given_name, pinyin, gender, score, source, notes, created_at FROM favorites ORDER BY created_at DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query favorites page: %w", err)
	}
	defer rows.Close()

	var records []*database.FavoriteRecord
	for rows.Next() {
		var r database.FavoriteRecord
		if err := rows.Scan(&r.ID, &r.Surname, &r.GivenName, &r.Pinyin, &r.Gender, &r.Score, &r.Source, &r.Notes, &r.CreatedAt); err != nil {
			logger.Error("sqlite: GetFavoritesPage scan failed", logger.ErrField(err))
			continue
		}
		records = append(records, &r)
	}
	if err := rows.Err(); err != nil {
		// 迭代中途的 IO/解码错误此前被当成「正常取完」，静默截断数据（docs/29 P3）
		logger.Error("sqlite: rows iteration failed mid-scan", logger.ErrField(err))
	}

	return records, total, nil
}

func (s *Store) DeleteFavorite(id string) error {
	_, err := s.writeDB.Exec(`DELETE FROM favorites WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete favorite: %w", err)
	}
	return nil
}

func (s *Store) BatchDeleteFavorite(ids []string) error {
	tx, err := s.writeDB.Begin()
	if err != nil {
		return fmt.Errorf("batch delete favorites begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(`DELETE FROM favorites WHERE id = ?`)
	if err != nil {
		return fmt.Errorf("batch delete favorites prepare: %w", err)
	}
	defer stmt.Close()

	for _, id := range ids {
		if _, err := stmt.Exec(id); err != nil {
			return fmt.Errorf("batch delete favorites exec: %w", err)
		}
	}
	return tx.Commit()
}

func (s *Store) GetFavoriteByName(surname, givenName string) *database.FavoriteRecord {
	var r database.FavoriteRecord
	err := s.db.QueryRow(`SELECT id, surname, given_name, pinyin, gender, score, source, notes, created_at FROM favorites WHERE surname = ? AND given_name = ?`, surname, givenName).
		Scan(&r.ID, &r.Surname, &r.GivenName, &r.Pinyin, &r.Gender, &r.Score, &r.Source, &r.Notes, &r.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		logger.Error("sqlite: GetFavoriteByName failed", logger.ErrField(err))
		return nil
	}
	return &r
}

// --- YijingStore ---

func (s *Store) GetHexagramByNumber(id int) *yijing.Hexagram {
	for i := range s.Hexagrams {
		if s.Hexagrams[i].Number == id {
			// 返回拷贝，避免外部修改污染内部数据 / 并发读写
			h := s.Hexagrams[i]
			return &h
		}
	}
	return nil
}

func (s *Store) GetHexagrams() []yijing.Hexagram {
	result := make([]yijing.Hexagram, len(s.Hexagrams))
	copy(result, s.Hexagrams)
	return result
}

// --- ZodiacStore ---

func (s *Store) GetZodiacByName(name string) *zodiac.Zodiac {
	for i := range s.Zodiacs {
		if s.Zodiacs[i].Name == name {
			// 返回拷贝，避免外部修改污染内部数据 / 并发读写
			z := s.Zodiacs[i]
			return &z
		}
	}
	return nil
}

func (s *Store) GetZodiacs() []zodiac.Zodiac {
	result := make([]zodiac.Zodiac, len(s.Zodiacs))
	copy(result, s.Zodiacs)
	return result
}

// --- HanziStore ---

func (s *Store) GetHanziByChar(char string) *database.Hanzi {
	if h, ok := hanzi.GetHanzi(char); ok {
		return &database.Hanzi{
			Char:    h.Char,
			Pinyin:  h.Pinyin,
			Wuxing:  h.Wuxing,
			Strokes: h.Strokes,
		}
	}
	return nil
}

func (s *Store) GetHanziByWuxing(wuxing string) []*database.Hanzi {
	var result []*database.Hanzi
	for _, h := range hanzi.HanziData {
		if h.Wuxing == wuxing {
			result = append(result, &database.Hanzi{
				Char:    h.Char,
				Pinyin:  h.Pinyin,
				Wuxing:  h.Wuxing,
				Strokes: h.Strokes,
			})
		}
	}
	return result
}

func (s *Store) GetHanziByStrokes(min, max int) []*database.Hanzi {
	var result []*database.Hanzi
	for _, h := range hanzi.HanziData {
		if h.Strokes >= min && h.Strokes <= max {
			result = append(result, &database.Hanzi{
				Char:    h.Char,
				Pinyin:  h.Pinyin,
				Wuxing:  h.Wuxing,
				Strokes: h.Strokes,
			})
		}
	}
	return result
}

func (s *Store) SearchHanzi(keyword string, limit int) []*database.Hanzi {
	if limit <= 0 {
		limit = 50
	}
	var result []*database.Hanzi
	for _, h := range hanzi.HanziData {
		if strings.Contains(h.Char, keyword) || strings.Contains(strings.ToLower(h.Pinyin), strings.ToLower(keyword)) {
			result = append(result, &database.Hanzi{
				Char:    h.Char,
				Pinyin:  h.Pinyin,
				Wuxing:  h.Wuxing,
				Strokes: h.Strokes,
			})
			if len(result) >= limit {
				break
			}
		}
	}
	return result
}

// --- FeedbackStore ---

func (s *Store) SaveNameRequest(record *database.NameRequest) int64 {
	result, err := s.writeDB.Exec(
		`INSERT INTO name_requests (surname, gender, birth_year, birth_month, birth_day, birth_hour, preferences, name_type, request_hash, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.Surname, record.Gender, record.BirthYear, record.BirthMonth, record.BirthDay, record.BirthHour,
		record.Preferences, record.NameType, record.RequestHash, record.CreatedAt)
	if err != nil {
		logger.Error("SaveNameRequest failed", logger.ErrField(err))
		return 0
	}
	id, _ := result.LastInsertId()
	return id
}

func (s *Store) GetNameRequestByHash(hash string) *database.NameRequest {
	var r database.NameRequest
	err := s.db.QueryRow(
		`SELECT id, surname, gender, birth_year, birth_month, birth_day, birth_hour, preferences, name_type, request_hash, created_at
		 FROM name_requests WHERE request_hash = ?`, hash).
		Scan(&r.ID, &r.Surname, &r.Gender, &r.BirthYear, &r.BirthMonth, &r.BirthDay, &r.BirthHour,
			&r.Preferences, &r.NameType, &r.RequestHash, &r.CreatedAt)
	if err != nil {
		return nil
	}
	return &r
}

func (s *Store) SaveNameFeedback(record *database.NameFeedback) int64 {
	result, err := s.writeDB.Exec(
		`INSERT INTO name_feedback (request_id, full_name, given_name, is_liked, is_selected, user_rating, feedback_text, algorithm_score, wuxing_match, device_info, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.RequestID, record.FullName, record.GivenName, record.IsLiked, record.IsSelected,
		record.UserRating, record.FeedbackText, record.AlgorithmScore, record.WuxingMatch, record.DeviceInfo, record.CreatedAt)
	if err != nil {
		logger.Error("SaveNameFeedback failed", logger.ErrField(err))
		return 0
	}
	id, _ := result.LastInsertId()
	return id
}

func (s *Store) GetNameFeedbackByRequestID(requestID int64) []*database.NameFeedback {
	rows, err := s.db.Query(
		`SELECT id, request_id, full_name, given_name, is_liked, is_selected, user_rating, feedback_text, algorithm_score, wuxing_match, device_info, created_at
		 FROM name_feedback WHERE request_id = ?`, requestID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var result []*database.NameFeedback
	for rows.Next() {
		var f database.NameFeedback
		if err := rows.Scan(&f.ID, &f.RequestID, &f.FullName, &f.GivenName, &f.IsLiked, &f.IsSelected,
			&f.UserRating, &f.FeedbackText, &f.AlgorithmScore, &f.WuxingMatch, &f.DeviceInfo, &f.CreatedAt); err != nil {
			continue
		}
		result = append(result, &f)
	}
	if err := rows.Err(); err != nil {
		// 迭代中途的 IO/解码错误此前被当成「正常取完」，静默截断数据（docs/29 P3）
		logger.Error("sqlite: rows iteration failed mid-scan", logger.ErrField(err))
	}

	return result
}

func (s *Store) GetAlgorithmPerformance() ([]*database.AlgorithmPerformance, error) {
	rows, err := s.db.Query(
		`SELECT DATE(nr.created_at) as date,
		        COUNT(*) as total_requests,
		        COALESCE(AVG(nf.algorithm_score), 0) as avg_score,
		        COALESCE(SUM(CASE WHEN nf.is_liked THEN 1 ELSE 0 END) * 1.0 / NULLIF(COUNT(nf.id), 0), 0) as like_rate,
		        COALESCE(SUM(CASE WHEN nf.is_selected THEN 1 ELSE 0 END) * 1.0 / NULLIF(COUNT(nf.id), 0), 0) as select_rate
		 FROM name_requests nr
		 LEFT JOIN name_feedback nf ON nf.request_id = nr.id
		 GROUP BY DATE(nr.created_at)
		 ORDER BY date DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*database.AlgorithmPerformance
	for rows.Next() {
		var p database.AlgorithmPerformance
		if err := rows.Scan(&p.Date, &p.TotalRequests, &p.AvgScore, &p.LikeRate, &p.SelectRate); err != nil {
			continue
		}
		result = append(result, &p)
	}
	if err := rows.Err(); err != nil {
		// 迭代中途的 IO/解码错误此前被当成「正常取完」，静默截断数据（docs/29 P3）
		logger.Error("sqlite: rows iteration failed mid-scan", logger.ErrField(err))
	}

	return result, nil
}

// --- CuratedStore ---

func (s *Store) seedCuratedNames(dataDir string) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM curated_names").Scan(&count)
	if err != nil || count > 0 {
		return
	}

	// 读取统一收敛到 name.LoadCuratedNamesData（P2-12），避免四处各自解析漂移
	entries, err := name.LoadCuratedNamesData(dataDir)
	if err != nil {
		logger.Warn("sqlite: seed curated_names.json 读取失败，跳过", logger.ErrField(err))
		return
	}

	tx, err := s.writeDB.Begin()
	if err != nil {
		logger.Warn("sqlite: seed curated names tx begin failed", logger.ErrField(err))
		return
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare("INSERT OR IGNORE INTO curated_names(name,pinyin,gender,score,source) VALUES(?,?,?,?,?)")
	if err != nil {
		logger.Warn("sqlite: seed curated names prepare failed", logger.ErrField(err))
		return
	}
	defer stmt.Close()

	seeded := 0
	for _, e := range entries {
		name := strings.TrimSpace(e.Name)
		if name == "" {
			continue
		}
		if _, err := stmt.Exec(name, e.Pinyin, e.Gender, e.YinyunScore, "seed"); err != nil {
			continue
		}
		seeded++
	}
	if err := tx.Commit(); err != nil {
		logger.Warn("sqlite: seed curated names commit failed", logger.ErrField(err))
		return
	}
	logger.Info("sqlite: seeded curated names", logger.Int("count", seeded))
}

func (s *Store) SaveCuratedName(name, pinyin, gender string, score float64, source string) error {
	_, err := s.writeDB.Exec(
		"INSERT OR REPLACE INTO curated_names(name,pinyin,gender,score,source,created_at) VALUES(?,?,?,?,?,CURRENT_TIMESTAMP)",
		name, pinyin, gender, score, source,
	)
	return err
}

func (s *Store) LoadAllCuratedNames() ([]database.CuratedNameEntry, error) {
	rows, err := s.db.Query("SELECT name,pinyin,gender,score,source FROM curated_names ORDER BY score DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []database.CuratedNameEntry
	for rows.Next() {
		var e database.CuratedNameEntry
		if err := rows.Scan(&e.Name, &e.Pinyin, &e.Gender, &e.Score, &e.Source); err != nil {
			continue
		}
		result = append(result, e)
	}
	if err := rows.Err(); err != nil {
		// 迭代中途的 IO/解码错误此前被当成「正常取完」，静默截断数据（docs/29 P3）
		logger.Error("sqlite: rows iteration failed mid-scan", logger.ErrField(err))
	}

	return result, nil
}

func (s *Store) IsCurated(name string) (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM curated_names WHERE name=?", name).Scan(&count)
	return count > 0, err
}

func (s *Store) DeleteCuratedName(name string) error {
	_, err := s.writeDB.Exec("DELETE FROM curated_names WHERE name=?", name)
	return err
}

// --- Hanzi 字库缓存 ---

// maxHanziRowFailures 汉字字库重建时允许跳过的最大失败行数
//
// 少量脏行跳过并记录，物料整体损坏（失败率超阈值）则回滚报错（docs/31 ADR-004）
const maxHanziRowFailures = 8

// hanziDataInsertSQL hanzi_data 的插入语句，seed 与 reload 共用唯一真源
//
// 写入列顺序与 hanziDataValues() 的返回值顺序严格对应，改动必须同步
// （hanzi_data 表列、hanziDataColumnMigrations、hanziDataValues 三处须一致）
const hanziDataInsertSQL = `INSERT INTO hanzi_data(
	char, pinyin, strokes, kangxi_strokes, radical, meaning, wuxing, gender,
	level, tone, name_freq_tier, gender_tags, style_tags, usage_level,
	positive_score, phonetic_score, modern_score, classical_score,
	is_polyphonic, is_rare, is_negative,
	pair_blacklist, curation_level, name_penalty, naming_categories
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`

// hanziDataValues 把 hanzi.Hanzi 投影为 hanzi_data 的行值
//
// 列为 seed/reload 共用的唯一映射真源（docs/31 ADR-012），避免两处各写一遍而漂移
func hanziDataValues(char string, h hanzi.Hanzi) []any {
	return []any{
		char, h.Pinyin, h.Strokes, h.KangxiStrokes, h.Radical, h.Meaning, h.Wuxing, h.Gender,
		h.Level, h.Tone, h.NameFreqTier,
		mustMarshalJSON(h.GenderTags), mustMarshalJSON(h.StyleTags), h.UsageLevel,
		h.PositiveScore, h.PhoneticScore, h.ModernScore, h.ClassicalScore,
		boolToInt(h.IsPolyphonic), boolToInt(h.IsRare), boolToInt(h.IsNegative),
		mustMarshalJSON(h.PairBlacklist), h.CurationLevel, h.NamePenalty,
		mustMarshalJSON(h.NamingCategories),
	}
}

// boolToInt 把布尔转成 SQLite 存储用的 0/1
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// HanziRowCount 返回 hanzi_data 行数
//
// 供 services 层 SQLHanziFilter.HasData 甄别「库无数据（未导入物料，
// 应降级 Go 端过滤）」与「有数据但查询无匹配（如实返回空）」——
// 空库回落场景下两者在 SQL 查询返回值上完全一致，是 docs/29 B8
// 修复（失败与无匹配可区分）之后新暴露的孪生陷阱。
func (s *Store) HanziRowCount() (int, error) {
	var n int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM hanzi_data").Scan(&n); err != nil {
		return 0, fmt.Errorf("查询 hanzi_data 行数失败: %w", err)
	}
	return n, nil
}

// seedHanziData 将 hanzi.HanziData 投影到 hanzi_data 表
//
// 语义（docs/31 ADR-011）：同输入多次执行结果等价，且物料的字段修订与删字都能传导到 SQLite。
// 实现为「单事务 DELETE + INSERT 全量重建」，不再使用 INSERT OR IGNORE ——
// 冲突即跳过会让字段修订与删字静默失效，SQL 与内存双读由此长期不一致。
//
// 同时带 checksum 门控（ADR-014）：物料指纹未变且表非空时整段跳过，
// 避免每次启动都对 8105 字做无谓的全量重建。
func (s *Store) seedHanziData() {
	if s.hanziUnchanged() {
		logger.Info("sqlite: 汉字物料未变更，跳过 hanzi_data 重建",
			logger.String("指纹", s.storedHanziChecksum()))
		return
	}
	s.rebuildHanziData("seedHanziData")
}

// storedHanziChecksum 读出库内记录的汉字物料指纹（未记录返回空串）
func (s *Store) storedHanziChecksum() string {
	var v string
	err := s.db.QueryRow("SELECT value FROM data_meta WHERE key = ?", metaKeyHanziChecksum).Scan(&v)
	if err != nil {
		return ""
	}
	return v
}

// currentHanziChecksum 计算当前内存字库的指纹
//
// 指纹取自「字集 + 关键字段」的稳定序列化：内容不变则指纹不变，
// 任一字被增删改都会导致指纹变化，从而触发重建（ADR-014 的门控依据）。
func currentHanziChecksum() string {
	chars := make([]string, 0, len(hanzi.HanziData))
	for c := range hanzi.HanziData {
		chars = append(chars, c)
	}
	sort.Strings(chars)

	h := sha256.New()
	for _, c := range chars {
		hv := hanzi.HanziData[c]
		fmt.Fprintf(h, "%s|%s|%d|%d|%s|%s|%s|%s|%d|%d|%d|%t|%t|%t|%d|%s\n",
			c, hv.Pinyin, hv.Strokes, hv.KangxiStrokes, hv.Radical, hv.Meaning,
			hv.Wuxing, hv.Gender, hv.Level, hv.Tone, hv.NameFreqTier,
			hv.IsPolyphonic, hv.IsRare, hv.IsNegative, hv.PositiveScore,
			strings.Join(hv.NamingCategories, ","))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// hanziUnchanged 判断是否可跳过重建：指纹一致且表非空
func (s *Store) hanziUnchanged() bool {
	stored := s.storedHanziChecksum()
	if stored == "" {
		return false
	}
	var n int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM hanzi_data").Scan(&n); err != nil || n == 0 {
		return false
	}
	return stored == currentHanziChecksum()
}

// ReloadHanziData 热重载汉字字库数据
// 从 hanzi.HanziData 重新同步到 SQLite，用于 JSON 文件热更新后调用
func (s *Store) ReloadHanziData() {
	s.rebuildHanziData("ReloadHanziData")
}

// rebuildHanziData 在单个事务内重建 hanzi_data 表
//
// 事务保证中途失败时表内容回滚到原状，不会留下半截数据；
// HanziData 为空时拒绝重建，避免数据尚未装载就把既有表清空
func (s *Store) rebuildHanziData(op string) {
	if len(hanzi.HanziData) == 0 {
		logger.Warn("sqlite: 汉字数据为空，跳过 hanzi_data 重建以免清空既有数据",
			logger.String("操作", op))
		return
	}

	tx, err := s.writeDB.Begin()
	if err != nil {
		logger.Error("sqlite: 汉字字库重建事务开启失败", logger.String("操作", op), logger.ErrField(err))
		return
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec("DELETE FROM hanzi_data"); err != nil {
		logger.Error("sqlite: 汉字字库清空失败", logger.String("操作", op), logger.ErrField(err))
		return
	}

	stmt, err := tx.Prepare(hanziDataInsertSQL)
	if err != nil {
		logger.Error("sqlite: 汉字字库插入语句准备失败", logger.String("操作", op), logger.ErrField(err))
		return
	}
	defer stmt.Close()

	written, failed := 0, 0
	for char, h := range hanzi.HanziData {
		if _, err := stmt.Exec(hanziDataValues(char, h)...); err != nil {
			// 个别脏数据不拖垮整表，但失败率超阈值即视为物料损坏，整体回滚报错
			failed++
			if failed > maxHanziRowFailures {
				logger.Error("sqlite: 汉字字库写入失败率超阈值，已回滚",
					logger.String("操作", op), logger.Int("失败数", failed),
					logger.Int("阈值", maxHanziRowFailures), logger.String("字", char),
					logger.ErrField(err))
				return
			}
			logger.Warn("sqlite: 单个字写入失败已跳过", logger.String("操作", op),
				logger.String("字", char), logger.ErrField(err))
			continue
		}
		written++
	}

	if err := tx.Commit(); err != nil {
		logger.Error("sqlite: 汉字字库重建提交失败", logger.String("操作", op), logger.ErrField(err))
		return
	}

	// 记录物料指纹，供下次启动的门控比对（ADR-014）
	if failed == 0 {
		if err := s.saveMeta(metaKeyHanziChecksum, currentHanziChecksum()); err != nil {
			logger.Warn("sqlite: 汉字物料指纹写入失败，下次启动将重建", logger.ErrField(err))
		}
	}

	logger.Info("sqlite: 汉字字库已重建", logger.String("操作", op),
		logger.Int("条数", written), logger.Int("跳过", failed))
}

// QueryHanziByWuxing 从 SQLite 按五行查询汉字
func (s *Store) QueryHanziByWuxing(wuxing string) ([]database.Hanzi, error) {
	rows, err := s.db.Query(
		"SELECT char, pinyin, wuxing, strokes FROM hanzi_data WHERE wuxing = ? ORDER BY strokes",
		wuxing,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []database.Hanzi
	for rows.Next() {
		var h database.Hanzi
		if err := rows.Scan(&h.Char, &h.Pinyin, &h.Wuxing, &h.Strokes); err != nil {
			continue
		}
		result = append(result, h)
	}
	if err := rows.Err(); err != nil {
		// 迭代中途的 IO/解码错误此前被当成「正常取完」，静默截断数据（docs/29 P3）
		logger.Error("sqlite: rows iteration failed mid-scan", logger.ErrField(err))
	}

	return result, nil
}

// QueryHanziByStrokes 从 SQLite 按笔画范围查询汉字
func (s *Store) QueryHanziByStrokes(min, max int) ([]database.Hanzi, error) {
	rows, err := s.db.Query(
		"SELECT char, pinyin, wuxing, strokes FROM hanzi_data WHERE strokes >= ? AND strokes <= ? ORDER BY strokes",
		min, max,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []database.Hanzi
	for rows.Next() {
		var h database.Hanzi
		if err := rows.Scan(&h.Char, &h.Pinyin, &h.Wuxing, &h.Strokes); err != nil {
			continue
		}
		result = append(result, h)
	}
	if err := rows.Err(); err != nil {
		// 迭代中途的 IO/解码错误此前被当成「正常取完」，静默截断数据（docs/29 P3）
		logger.Error("sqlite: rows iteration failed mid-scan", logger.ErrField(err))
	}

	return result, nil
}

// mustMarshalJSON 将任意值 JSON 序列化，失败则返回 "[]"
func mustMarshalJSON(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(data)
}

// SearchHanziByFilter 按基础过滤参数查询 hanzi_data（services.SQLiteCharStore 接口实现）
//
// services 包不能 import sqlite 包（分层约束），故用此最简基础参数签名
// 让 *Store 同时实现 services.SQLiteCharStore 接口。组装时通过
// services.NewSQLiteHanziFilter(store) 桥接：
//
//	if s, ok := store.(services.SQLiteCharStore); ok {
//	    provider.SetSQLFilter(services.NewSQLiteHanziFilter(s))
//	}
//
// 实现内联 SQL 查询（避免依赖 HanziFilter 类型）：动态构建 WHERE 子句，
// 命中 idx_hanzi_data_wuxing / idx_hanzi_data_strokes 索引。
//
// P2（ADR-003「SQL 与内存结果等价」）：本方法原先只返回 6 列
// （char/pinyin/wuxing/strokes/usage_level/positive_score）。services 层拿不到
// gender / naming_categories / radical / level / kangxi_strokes / meaning / is_rare，
// 只能在拿到结果后逐字 `hanzi.GetHanzi` 回查内存做二次过滤 —— SQL 沦为「粗筛」，
// 且 SQL 路径与内存路径的过滤结果无法保证一致（docs/31 §11.9 第 2 项）。
// 现已将全部过滤与构造所需列下沉到 hanzi_data 并一并返回，services 层不再回查。
func (s *Store) SearchHanziByFilter(
	wuxing string, minStrokes, maxStrokes int,
	hasPositive, isRegular bool, chars []string, limit int,
) ([]*database.Hanzi, error) {
	var conditions []string
	var args []any

	if wuxing != "" {
		conditions = append(conditions, "wuxing = ?")
		args = append(args, wuxing)
	}
	if minStrokes > 0 {
		conditions = append(conditions, "strokes >= ?")
		args = append(args, minStrokes)
	}
	if maxStrokes > 0 {
		conditions = append(conditions, "strokes <= ?")
		args = append(args, maxStrokes)
	}
	if hasPositive {
		conditions = append(conditions, "positive_score > 0")
	}
	if isRegular {
		conditions = append(conditions, "usage_level IN (1, 2)")
	}
	if len(chars) > 0 {
		placeholders := make([]string, len(chars))
		for i, c := range chars {
			placeholders[i] = "?"
			args = append(args, c)
		}
		conditions = append(conditions, "char IN ("+strings.Join(placeholders, ",")+")")
	}

	if limit <= 0 {
		limit = 10000
	}

	// P2：返回构造 Character 与过滤所需的全部列，避免调用方回查内存补字段。
	// 列顺序与下方 Scan 顺序严格一一对应，改任一侧都必须同步。
	const selectColumns = `char, pinyin, wuxing, strokes, usage_level, positive_score,
		COALESCE(level, 0), COALESCE(kangxi_strokes, 0), COALESCE(gender, ''),
		COALESCE(radical, ''), COALESCE(naming_categories, '[]'), COALESCE(meaning, ''),
		COALESCE(is_rare, 0), COALESCE(is_negative, 0), COALESCE(name_penalty, 0),
		COALESCE(name_freq_tier, 0)`

	query := "SELECT " + selectColumns + " FROM hanzi_data"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY char LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		// 必须上抛而不是返回 nil：nil 会被上层判成「无匹配」，
		// 使注释承诺的「SQL 失败降级 Go 端全表过滤」路径永不可达（docs/29 B8）
		logger.Error("SearchHanziByFilter 查询失败", logger.ErrField(err))
		return nil, fmt.Errorf("SearchHanziByFilter 查询失败: %w", err)
	}
	defer rows.Close()

	var result []*database.Hanzi
	for rows.Next() {
		var h database.Hanzi
		var namingCategories string
		var isRare, isNegative int
		if err := rows.Scan(&h.Char, &h.Pinyin, &h.Wuxing, &h.Strokes,
			&h.UsageLevel, &h.PositiveScore,
			&h.Level, &h.KangxiStrokes, &h.Gender, &h.Radical,
			&namingCategories, &h.Meaning, &isRare,
			&isNegative, &h.NamePenalty, &h.NameFreqTier); err != nil {
			logger.Error("SearchHanziByFilter scan 失败", logger.ErrField(err))
			continue
		}
		h.IsRare = isRare != 0
		h.IsNegative = isNegative != 0
		if namingCategories != "" && namingCategories != "[]" {
			// naming_categories 落库形态为 JSON 数组文本；解析失败按「无分类」处理，
			// 不丢弃整行 —— 否则单条脏数据会连带丢掉该字的全部候选属性。
			var categories []string
			if err := json.Unmarshal([]byte(namingCategories), &categories); err != nil {
				logger.Error("SearchHanziByFilter naming_categories 解析失败",
					logger.ErrField(err), logger.String("char", h.Char))
			} else {
				h.NamingCategories = categories
			}
		}
		result = append(result, &h)
	}
	if err := rows.Err(); err != nil {
		// 迭代中途的 IO/解码错误此前被当成「正常取完」，静默截断数据（docs/29 P3）
		logger.Error("sqlite: rows iteration failed mid-scan", logger.ErrField(err))
	}

	// rows.Err 覆盖迭代中途的 IO/解码错误：不查就等于把截断结果当正常取完
	if err := rows.Err(); err != nil {
		logger.Error("SearchHanziByFilter 迭代失败", logger.ErrField(err))
		return nil, fmt.Errorf("SearchHanziByFilter 迭代失败: %w", err)
	}
	return result, nil
}

// --- NameStatStore 占位实现 ---
//
// 背景：数据库.Store 接口组合了 NameStatStore（8 方法）。姓名统计数据
// （姓氏/名/全名排行、姓名性别统计）由 data 目录 JSON 文件承载
// （surname_stats.json 等，见 cmd/build_namestats），SQLite 无对应表。
// 占位方法返回明确的"不支持"错误（而非静默空数据），让 /namestats/* 接口
// 在接线真正实现前诚实报错，便于调用方降级。
var errNameStatsUnavailable = errors.New("姓名统计数据未接入 SQLite 持久化，当前数据源为 data 目录 JSON 文件")

func (s *Store) GetSurnameStats(limit int) ([]database.SurnameStat, error) {
	return nil, errNameStatsUnavailable
}
func (s *Store) GetSurnameStat(surname string) (*database.SurnameStat, error) {
	return nil, errNameStatsUnavailable
}
func (s *Store) GetGivenNameStats(surname string, limit int) ([]database.GivenNameStat, error) {
	return nil, errNameStatsUnavailable
}
func (s *Store) GetFullNameStats(surname string, limit int) ([]database.FullNameStat, error) {
	return nil, errNameStatsUnavailable
}
func (s *Store) GetFullNameStat(fullName string) (*database.FullNameStat, error) {
	return nil, errNameStatsUnavailable
}
func (s *Store) GetNameGenderStats(name string) (*database.NameGenderStat, error) {
	return nil, errNameStatsUnavailable
}
func (s *Store) GetTopFullNames(limit int) ([]database.FullNameStat, error) {
	return nil, errNameStatsUnavailable
}
func (s *Store) GetTotalNameCount() (int, error) {
	return 0, errNameStatsUnavailable
}
