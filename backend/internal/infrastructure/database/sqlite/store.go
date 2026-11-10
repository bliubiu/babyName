package sqlite

import (
	"database/sql"
	"fmt"
	"sync"

	_ "modernc.org/sqlite"
	"name/internal/domain/yijing"
	"name/internal/domain/zodiac"
	"name/internal/infrastructure/database"
	"name/internal/infrastructure/logger"
)

type Store struct {
	db        *sql.DB // 读连接池（WAL 模式下可并发读）
	writeDB   *sql.DB // 写连接（单连接，串行写）
	Hexagrams []yijing.Hexagram
	Zodiacs   []zodiac.Zodiac

	// hanziSnap / hanziSnapReady：hanzi_data 全表只读快照（docs/46 P2 性能项）。
	// 静态字库启动期灌入后不再变化，惰性构建一次，消除「全量候选池」查询
	// 每请求全表扫描 + 逐行 JSON 解析 naming_categories。
	// 快照只读共享：下游 databaseHanziToCharacter 仅读取字段、适配器按值拷贝，
	// 不存在请求期改写，故返回缓存指针安全。
	hanziSnapMu    sync.RWMutex
	hanziSnap      []*database.Hanzi
	hanziSnapReady bool
}

var _ database.Store = (*Store)(nil)

func NewStore(dbPath string, dataDir string) (*Store, error) {
	// 写连接：单连接，负责所有写操作
	// busy_timeout=5000：modernc.org/sqlite 默认 0——WAL checkpoint、
	// 外部工具（备份/导入）持锁瞬间，读写路径会直接抛 "database is locked"
	// 而不是等待；给 5 秒重试窗口（docs/28 W5 审计结论）。
	// foreign_keys(ON)：启用 name_feedback.request_id / classics_sections.book_id
	// 声明的外键约束（SQLite 默认关闭，声明形同虚设，docs/43 P1-5）；
	// 连接串参数在每连接建立时执行，不依赖调用顺序。读连接 mode=ro 无写路径，无需启用。
	writeDB, err := sql.Open("sqlite", dbPath+"?_txlock=immediate&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)")
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
		logger.Warn("sqlite: WAL 模式可能未生效", logger.String("journal_mode", journalMode))
	}

	// 读连接池：WAL 模式下读不阻塞写，可并发
	// busy_timeout 同写连接：读路径与 checkpoint 撞锁时等待而非报错
	readDB, err = sql.Open("sqlite", dbPath+"?mode=ro&_pragma=busy_timeout(5000)")
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

	// 会话隔离列属于「存量库也要补齐」的演进项，必须在建表之后单独迁移；
	// 索引依赖新列，不能提前写进 createTables 的建表清单。
	if err := migrateSessionColumns(writeDB); err != nil {
		return nil, err
	}

	// 物料表结构随库创建而就绪（表空；数据由 `materials import` 填充）
	//
	// 与 createTables 分开：物料表是「导入器的产物」，集中在 store_materials_schema.go 定义，
	// 避免建表语句散落在两个文件里。
	if err := createMaterialTables(writeDB); err != nil {
		return nil, err
	}

	// 库结构版本登记与时间口径迁移（docs/43 P2-11）：v1 把运营表存量
	// 本地时间归一 UTC，与写入端（time.Now().UTC()）统一口径。
	if err := ensureSchemaVersion(writeDB); err != nil {
		return nil, fmt.Errorf("schema 版本迁移失败: %w", err)
	}

	store := &Store{
		db:        readDB,
		writeDB:   writeDB,
		Hexagrams: yijing.GetAllHexagrams(),
		Zodiacs:   zodiac.GetAllZodiacs(),
	}

	// 经典语料数据不再在启动时 seed：classics 已接入统一导入器
	// （materials import），运行期从 classics_* / classics_extracts 表装载
	// （docs/31 ADR-008）。精选名同理，见下方注释。

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
			session_id TEXT NOT NULL DEFAULT '',
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
			session_id TEXT NOT NULL DEFAULT '',
			surname TEXT NOT NULL,
			given_name TEXT NOT NULL,
			pinyin TEXT NOT NULL,
			gender TEXT NOT NULL CHECK (gender IN ('male', 'female', 'neutral') OR gender = ''),
			score REAL NOT NULL,
			source TEXT,
			notes TEXT,
			created_at TIMESTAMP NOT NULL
		)`,
		// 收藏/历史列表按「会话内 created_at DESC」分页查询（GetFavoritesPage
		// 等），复合索引 (session_id, created_at DESC) 由 migrateSessionColumns
		// 统一建立；docs/43 P2-11 删除了与此模式不匹配的空转单列索引。
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

		// 经典数据表（诗词、易经、蒙学等）与 hanzi_data / curated_names 一样
		// 是物料表，建表语句统一收敛在 store_materials_schema.go
		//（materialTableDDL），本处不再重复定义（docs/34 双口径）。

		// 注意：hanzi_data 与 curated_names 是物料表，建表语句统一收敛在
		// store_materials_schema.go（materialTableDDL），本处不再重复定义，
		// 避免同一张表两处 DDL 因默认值/列集不同而漂移（docs/34 双口径）。

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

	return nil
}

// migrateSessionColumns 给已存在的 history/favorites 表补 session_id 列与索引。
//
// 存量记录的 session_id 默认为空串，**不会被任何会话查到**：这些记录产生于
// 会话隔离之前，无法确定归属，宁可让它们从访客视图消失，也不能继续全网可见。
// 索引在建列之后创建，避免旧库建表语句引用尚不存在的列而启动失败。
func migrateSessionColumns(db *sql.DB) error {
	for _, table := range []string{"history", "favorites"} {
		cols, err := tableColumns(db, table)
		if err != nil {
			return err
		}
		if cols["session_id"] {
			continue
		}
		if _, err := db.Exec("ALTER TABLE " + table + " ADD COLUMN session_id TEXT NOT NULL DEFAULT ''"); err != nil {
			return fmt.Errorf("给 %s 补 session_id 列失败: %w", table, err)
		}
	}

	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_history_session_created ON history(session_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_favorites_session_created ON favorites(session_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_favorites_session_name ON favorites(session_id, surname, given_name)`,
	}
	for _, q := range indexes {
		if _, err := db.Exec(q); err != nil {
			return fmt.Errorf("创建会话隔离索引失败: %w", err)
		}
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
