package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"name/internal/domain/hanzi"
	"name/internal/domain/yijing"
	"name/internal/domain/zodiac"
	"name/internal/infrastructure/database"
	"name/internal/infrastructure/logger"
	_ "modernc.org/sqlite"
)

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
	readDB, err := sql.Open("sqlite", dbPath+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite read connection: %w", err)
	}
	readDB.SetMaxOpenConns(16) // 允许多个并发读（WAL 模式下读不阻塞写，提高吞吐）
	readDB.SetMaxIdleConns(8)
	readDB.SetConnMaxLifetime(0)

	if err := readDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping sqlite read connection: %w", err)
	}
	// 读连接也设置 WAL 模式（确保使用同一 WAL 文件）
	if _, err := readDB.Exec("PRAGMA journal_mode=WAL"); err != nil {
		logger.Warn("sqlite: failed to set WAL mode for read connection", logger.ErrField(err))
	}

	if err := createTables(writeDB); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
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

	// 导入汉字字库缓存（仅在表为空时）
	store.seedHanziData()

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
		`CREATE TABLE IF NOT EXISTS favorites (
			id TEXT PRIMARY KEY,
			surname TEXT NOT NULL,
			given_name TEXT NOT NULL,
			pinyin TEXT NOT NULL,
			gender TEXT NOT NULL,
			score INTEGER NOT NULL,
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
			radical    TEXT NOT NULL DEFAULT '',
			meaning    TEXT NOT NULL DEFAULT '',
			wuxing     TEXT NOT NULL DEFAULT '',
			gender     TEXT NOT NULL DEFAULT '',
			tone       INTEGER NOT NULL DEFAULT 0,
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
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}
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
	defer tx.Rollback()

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

func (s *Store) SaveFavorite(record *database.FavoriteRecord) string {
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
	}
	return record.ID
}

func (s *Store) BatchSaveFavorite(records []*database.FavoriteRecord) []string {
	ids := make([]string, len(records))
	tx, err := s.writeDB.Begin()
	if err != nil {
		logger.Error("sqlite: BatchSaveFavorite begin tx failed", logger.ErrField(err))
		return ids
	}

	stmt, err := tx.Prepare(
		`INSERT INTO favorites (id, surname, given_name, pinyin, gender, score, source, notes, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET surname=excluded.surname, given_name=excluded.given_name`,
	)
	if err != nil {
		logger.Error("sqlite: BatchSaveFavorite prepare failed", logger.ErrField(err))
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
		_, err := stmt.Exec(record.ID, record.Surname, record.GivenName, record.Pinyin,
			record.Gender, record.Score, record.Source, record.Notes, record.CreatedAt)
		if err != nil {
			logger.Error("sqlite: BatchSaveFavorite exec failed", logger.ErrField(err))
			failedIndexes = append(failedIndexes, i)
			continue
		}
		ids[i] = record.ID
	}
	_ = stmt.Close()

	if err := tx.Commit(); err != nil {
		logger.Error("sqlite: BatchSaveFavorite commit failed", logger.ErrField(err))
		return ids
	}

	// 事务提交成功后，对失败记录通过单独连接重试（避免单连接死锁）
	for _, i := range failedIndexes {
		ids[i] = s.SaveFavorite(records[i])
	}
	return ids
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
	defer tx.Rollback()

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
	if h, ok := hanzi.HanziData[char]; ok {
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
	return result, nil
}

// --- CuratedStore ---

func (s *Store) seedCuratedNames(dataDir string) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM curated_names").Scan(&count)
	if err != nil || count > 0 {
		return
	}

	path := filepath.Join(dataDir, "curated_names.json")
	data, err := os.ReadFile(path)
	if err != nil {
		logger.Warn("sqlite: seed curated_names.json not found, skipping", logger.ErrField(err))
		return
	}

	var entries []struct {
		Name        string  `json:"name"`
		Pinyin      string  `json:"pinyin"`
		Gender      string  `json:"gender"`
		YinyunScore float64 `json:"yinyun_score"`
	}
	if err := json.Unmarshal(data, &entries); err != nil {
		logger.Warn("sqlite: failed to parse curated_names.json", logger.ErrField(err))
		return
	}

	tx, err := s.writeDB.Begin()
	if err != nil {
		logger.Warn("sqlite: seed curated names tx begin failed", logger.ErrField(err))
		return
	}
	defer tx.Rollback()

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

// seedHanziData 将 hanzi.HanziData 导入到 hanzi_data 表
// 仅在表为空时导入，避免覆盖用户数据
func (s *Store) seedHanziData() {
	var count int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM hanzi_data").Scan(&count); err != nil || count > 0 {
		return
	}

	tx, err := s.writeDB.Begin()
	if err != nil {
		logger.Error("sqlite: seedHanziData begin tx failed", logger.ErrField(err))
		return
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO hanzi_data(
		char, pinyin, strokes, radical, meaning, wuxing, gender,
		tone, gender_tags, style_tags, usage_level,
		positive_score, phonetic_score, modern_score, classical_score,
		is_polyphonic, is_rare, is_negative,
		pair_blacklist, curation_level, name_penalty, naming_categories
	) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		logger.Error("sqlite: seedHanziData prepare failed", logger.ErrField(err))
		return
	}
	defer stmt.Close()

	imported := 0
	for char, h := range hanzi.HanziData {
		genderTags := mustMarshalJSON(h.GenderTags)
		styleTags := mustMarshalJSON(h.StyleTags)
		pairBlacklist := mustMarshalJSON(h.PairBlacklist)
		namingCats := mustMarshalJSON(h.NamingCategories)

		polyphonic := 0
		if h.IsPolyphonic {
			polyphonic = 1
		}
		rare := 0
		if h.IsRare {
			rare = 1
		}
		negative := 0
		if h.IsNegative {
			negative = 1
		}

		if _, err := stmt.Exec(
			char, h.Pinyin, h.Strokes, h.Radical, h.Meaning, h.Wuxing, h.Gender,
			h.Tone, genderTags, styleTags, h.UsageLevel,
			h.PositiveScore, h.PhoneticScore, h.ModernScore, h.ClassicalScore,
			polyphonic, rare, negative,
			pairBlacklist, h.CurationLevel, h.NamePenalty, namingCats,
		); err != nil {
			continue
		}
		imported++
	}
	if err := tx.Commit(); err != nil {
		logger.Error("sqlite: seedHanziData commit failed", logger.ErrField(err))
		return
	}
	logger.Info("sqlite: hanzi_data seeded", logger.Int("count", imported))
}

// ReloadHanziData 热重载汉字字库数据
// 从 hanzi.HanziData 重新同步到 SQLite，用于 JSON 文件热更新后调用
func (s *Store) ReloadHanziData() {
	tx, err := s.writeDB.Begin()
	if err != nil {
		logger.Error("sqlite: ReloadHanziData begin tx failed", logger.ErrField(err))
		return
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM hanzi_data"); err != nil {
		logger.Error("sqlite: ReloadHanziData delete failed", logger.ErrField(err))
		return
	}

	stmt, err := tx.Prepare(`INSERT INTO hanzi_data(
		char, pinyin, strokes, radical, meaning, wuxing, gender,
		tone, gender_tags, style_tags, usage_level,
		positive_score, phonetic_score, modern_score, classical_score,
		is_polyphonic, is_rare, is_negative,
		pair_blacklist, curation_level, name_penalty, naming_categories
	) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		logger.Error("sqlite: ReloadHanziData prepare failed", logger.ErrField(err))
		return
	}
	defer stmt.Close()

	reloaded := 0
	for char, h := range hanzi.HanziData {
		genderTags := mustMarshalJSON(h.GenderTags)
		styleTags := mustMarshalJSON(h.StyleTags)
		pairBlacklist := mustMarshalJSON(h.PairBlacklist)
		namingCats := mustMarshalJSON(h.NamingCategories)

		polyphonic := 0
		if h.IsPolyphonic {
			polyphonic = 1
		}
		rare := 0
		if h.IsRare {
			rare = 1
		}
		negative := 0
		if h.IsNegative {
			negative = 1
		}

		if _, err := stmt.Exec(
			char, h.Pinyin, h.Strokes, h.Radical, h.Meaning, h.Wuxing, h.Gender,
			h.Tone, genderTags, styleTags, h.UsageLevel,
			h.PositiveScore, h.PhoneticScore, h.ModernScore, h.ClassicalScore,
			polyphonic, rare, negative,
			pairBlacklist, h.CurationLevel, h.NamePenalty, namingCats,
		); err != nil {
			continue
		}
		reloaded++
	}
	if err := tx.Commit(); err != nil {
		logger.Error("sqlite: ReloadHanziData commit failed", logger.ErrField(err))
		return
	}
	logger.Info("sqlite: hanzi_data reloaded", logger.Int("count", reloaded))
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