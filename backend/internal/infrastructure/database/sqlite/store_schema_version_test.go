package sqlite

// store_schema_version_test.go — v1 迁移（时间口径归一 UTC）行为守护（docs/43 P2-11）

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"name/internal/infrastructure/database"
)

// buildLegacyTZDB 手工建库并插入「driver 序列化的本地时间串」存量行
// （模拟 v0 时代 SaveHistory 写入的数据），随后关闭连接。
func buildLegacyTZDB(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "legacy.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("打开库失败: %v", err)
	}
	if err := createTables(db); err != nil {
		t.Fatalf("建表失败: %v", err)
	}
	legacy := time.Date(2026, 11, 6, 8, 0, 0, 0, time.Local)
	if _, err := db.Exec(
		`INSERT INTO history (id, session_id, surname, gender, birth_date, birth_time, results, created_at)
		 VALUES ('h1', 's1', '张', 'male', '2026-05-20', '10:00', '[]', ?)`, legacy); err != nil {
		t.Fatalf("灌入存量行失败: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("关闭失败: %v", err)
	}
	return path
}

// TestSchemaVersion_TimeMigrationV1 v1 迁移：存量本地时间归一 UTC 且时刻不变
func TestSchemaVersion_TimeMigrationV1(t *testing.T) {
	legacy := time.Date(2026, 11, 6, 8, 0, 0, 0, time.Local)
	path := buildLegacyTZDB(t)

	s, err := NewStore(path, t.TempDir())
	if err != nil {
		t.Fatalf("NewStore（触发迁移）失败: %v", err)
	}
	defer func() { _ = s.Close() }()

	// ① 版本登记
	var v int
	if err := s.db.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
		t.Fatalf("读取 user_version 失败: %v", err)
	}
	if v != schemaVersionCurrent {
		t.Fatalf("user_version = %d，期望 %d", v, schemaVersionCurrent)
	}

	// ② 存储串已是 UTC 口径
	var raw string
	if err := s.db.QueryRow(`SELECT created_at FROM history WHERE id='h1'`).Scan(&raw); err != nil {
		t.Fatalf("读取迁移后串失败: %v", err)
	}
	if !isUTCStamp(raw) {
		t.Errorf("迁移后存储串仍非 UTC 口径: %q", raw)
	}

	// ③ 时刻不变（往返无损，仅时区表达变化）
	var got time.Time
	if err := s.db.QueryRow(`SELECT created_at FROM history WHERE id='h1'`).Scan(&got); err != nil {
		t.Fatalf("回读失败: %v", err)
	}
	if !got.Equal(legacy) {
		t.Errorf("迁移改变时刻: %v ≠ %v", got, legacy)
	}

	// ④ 幂等：再次开库不重复平移
	s2, err := NewStore(path, t.TempDir())
	if err != nil {
		t.Fatalf("二次 NewStore 失败: %v", err)
	}
	defer func() { _ = s2.Close() }()
	var raw2 string
	if err := s2.db.QueryRow(`SELECT created_at FROM history WHERE id='h1'`).Scan(&raw2); err != nil {
		t.Fatalf("二次读取失败: %v", err)
	}
	if raw2 != raw {
		t.Errorf("二次开库串变化（迁移非幂等）: %q → %q", raw, raw2)
	}
}

// TestSaveHistory_UsesUTC 写入端统一 UTC（与迁移后存量同口径）
func TestSaveHistory_UsesUTC(t *testing.T) {
	s := newTestStore(t)
	before := time.Now().UTC()
	id := s.SaveHistory(&database.HistoryRecord{
		SessionID: "s-tz",
		Surname:   "张",
		Gender:    "male",
		BirthDate: "2026-05-20",
		Results:   "[]",
	})
	after := time.Now().UTC()
	if id == "" {
		t.Fatal("保存失败")
	}
	var got time.Time
	if err := s.db.QueryRow(`SELECT created_at FROM history WHERE id=?`, id).Scan(&got); err != nil {
		t.Fatalf("回读失败: %v", err)
	}
	if got.Location() != time.UTC {
		t.Errorf("新写入应为 UTC 时区，实际 %v", got.Location())
	}
	if got.Before(before.Add(-time.Second)) || got.After(after.Add(time.Second)) {
		t.Errorf("时间戳不在合理窗口: %v", got)
	}
}
