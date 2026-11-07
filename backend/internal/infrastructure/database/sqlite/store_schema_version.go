package sqlite

// store_schema_version.go — 库结构版本与时间口径迁移（docs/43 P2-11）
//
// 实测事实（探针，2026-11-07）：modernc driver 把 time.Time 序列化为
// 带本机时区偏移的 ISO8601 串（"2026-11-06T08:00:00+08:00"），回读时
// 恒转回 Local；SQL datetime() 无法解析带偏移后缀的串（返回 NULL）——
// 故时间口径迁移只能在 Go 侧逐行做，不能写 SQL datetime 表达式。
//
// v1 迁移：运营表（history/favorites）存量 created_at 归一为 UTC；
// 写入端（SaveHistory/SaveFavorite）同步改为 time.Now().UTC()。
// PRAGMA user_version 防重入；判据为存储原串形态（回读恒 Local，
// 不能以 time.Time 的 Location 判断），非 UTC 串（不以 Z 结尾且非
// +00:00 偏移）才归一，天然幂等。

import (
	"database/sql"
	"strings"
	"time"

	"name/internal/infrastructure/logger"
)

// schemaVersionCurrent 当前库结构版本。
// v1：运营表 created_at 归一 UTC（本文件 ensureSchemaVersion）。
const schemaVersionCurrent = 1

// ensureSchemaVersion 读取 user_version 并按需执行迁移，最后登记版本。
// 必须在建表（createTables）之后调用。
func ensureSchemaVersion(db *sql.DB) error {
	var v int
	if err := db.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
		return err
	}
	if v >= schemaVersionCurrent {
		return nil
	}

	for _, table := range []string{"history", "favorites"} {
		n, err := migrateTimesToUTC(db, table)
		if err != nil {
			return err
		}
		if n > 0 {
			logger.Warn("sqlite: 运营表时间口径已归一 UTC（v1 迁移）",
				logger.String("表", table), logger.Int64("行数", n))
		}
	}

	if _, err := db.Exec("PRAGMA user_version = 1"); err != nil {
		return err
	}
	logger.Info("sqlite: 库结构版本已登记", logger.Int("user_version", schemaVersionCurrent))
	return nil
}

// isUTCStamp 判断 driver 写入的时间串是否已是 UTC 口径：
// UTC 的 time.Time 经 driver 序列化以 Z 结尾（或 +00:00 偏移）。
func isUTCStamp(raw string) bool {
	raw = strings.TrimSpace(raw)
	return strings.HasSuffix(raw, "Z") || strings.HasSuffix(raw, "+00:00")
}

// migrateTimesToUTC 把一张运营表的时间列逐行归一为 UTC。
// 读原串判据 → 解析为 time.Time（driver 往返无损，探针实证）→
// .UTC() 写回。单事务包裹，失败整体回滚。
func migrateTimesToUTC(db *sql.DB, table string) (int64, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := tx.Query(`SELECT rowid, created_at AS raw, created_at AS t FROM ` + table + ` WHERE created_at IS NOT NULL`)
	if err != nil {
		return 0, err
	}
	type fixRow struct {
		rowid int64
		t     time.Time
	}
	var toFix []fixRow
	for rows.Next() {
		var (
			rowid int64
			raw   sql.NullString
			t     time.Time
		)
		if err := rows.Scan(&rowid, &raw, &t); err != nil {
			_ = rows.Close()
			return 0, err
		}
		if !raw.Valid || raw.String == "" || isUTCStamp(raw.String) {
			continue
		}
		toFix = append(toFix, fixRow{rowid: rowid, t: t})
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return 0, err
	}
	_ = rows.Close()

	var n int64
	for _, r := range toFix {
		res, err := tx.Exec(`UPDATE `+table+` SET created_at = ? WHERE rowid = ?`, r.t.UTC(), r.rowid)
		if err != nil {
			return 0, err
		}
		if affected, _ := res.RowsAffected(); affected > 0 {
			n++
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return n, nil
}
