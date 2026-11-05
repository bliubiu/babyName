package sqlite

// store_materials_schema_rebuild_test.go — classics_extracts 重建原子性回归（docs/43 P1-6）
//
// 背景：legacy 列重建走「CREATE new → INSERT SELECT → DROP old → RENAME」四步，
// 原实现逐条 Exec 无事务：中途失败时已执行的步骤不会回滚——例如第二步失败，
// 第一步的 classics_extracts_new 孤儿表已落库（残留且无人清理），旧表虽在
// 但重建永远无法再次开始（再次 CREATE 报 already exists）。
//
// 修复：四步包进单事务（SQLite DDL 可回滚），失败即整体回滚，无 _new 残留。
//
// 故障注入方式：旧表塞一行 char=NULL（新表 char NOT NULL），INSERT SELECT
// 必在第二步失败——覆盖「第一步已执行」的中途失败场景。

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// buildLegacyExtractsDB 构造一个带 legacy 列 classics_extracts 旧表的库
// （char 列刻意可空，以便注入违反新表 NOT NULL 约束的行）
func buildLegacyExtractsDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "legacy.db"))
	if err != nil {
		t.Fatalf("打开测试库失败: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ddl := `CREATE TABLE classics_extracts (
		material TEXT, seq INTEGER, char TEXT, work TEXT,
		chapter TEXT, sentence TEXT,
		pinyin TEXT, meaning TEXT, wuxing TEXT, gender TEXT)`
	if _, err := db.Exec(ddl); err != nil {
		t.Fatalf("建 legacy 表失败: %v", err)
	}
	rows := []string{
		`INSERT INTO classics_extracts VALUES ('shijing', 1, '关', '关雎', '周南', '关关雎鸠', 'guan', '拟声', '木', 'neutral')`,
		`INSERT INTO classics_extracts VALUES ('shijing', 2, '雎', '关雎', '周南', '在河之洲', 'ju', '鸟名', '水', 'neutral')`,
		`INSERT INTO classics_extracts VALUES ('chuci', 1, '离', '离骚', '', '帝高阳之苗裔', 'li', '离开', '火', 'neutral')`,
		// 注入行：char 为 NULL，迁移到新表（char NOT NULL）时 INSERT SELECT 必失败
		`INSERT INTO classics_extracts (material, seq, char) VALUES ('bad', 9, NULL)`,
	}
	for _, q := range rows {
		if _, err := db.Exec(q); err != nil {
			t.Fatalf("灌入 legacy 数据失败: %v", err)
		}
	}
	return db
}

// TestMigrateClassicsSchema_RebuildIsAtomic 重建中途失败必须整体回滚：
// 旧表数据完好、legacy 列保留、无 classics_extracts_new 孤儿表残留
func TestMigrateClassicsSchema_RebuildIsAtomic(t *testing.T) {
	db := buildLegacyExtractsDB(t)

	err := migrateClassicsSchema(db)
	if err == nil {
		t.Fatal("注入 char=NULL 行后重建应失败（违反新表 NOT NULL），实际成功——注入未生效")
	}

	// ① 无 _new 孤儿表残留
	if hasTable(db, "classics_extracts_new") {
		t.Fatal("重建失败后仍存在 classics_extracts_new 孤儿表——四步未包事务，残留将阻断下次重建")
	}

	// ② 旧表数据完好
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM classics_extracts`).Scan(&n); err != nil {
		t.Fatalf("读取旧表行数失败: %v", err)
	}
	if n != 4 {
		t.Fatalf("旧表行数 = %d，期望 4（重建失败不得丢数据）", n)
	}

	// ③ legacy 列仍在（重建确实未发生，等待下次成功迁移）
	cols, err := tableColumns(db, "classics_extracts")
	if err != nil {
		t.Fatalf("读取列信息失败: %v", err)
	}
	for _, c := range classicExtractsLegacyColumns {
		if !cols[c] {
			t.Errorf("legacy 列 %s 消失——部分重建发生", c)
		}
	}
}
