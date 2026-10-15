package sqlite

import (
	"database/sql"
	"reflect"
	"strings"
	"testing"

	"name/internal/domain/hanzi"
)

// 本文件是 docs/31 §11.3 的「schema 漂移守护测试」本体：
// 断言 Hanzi 结构体字段 ↔ hanzi_data 表列 ↔ 插入语句占位符 三者严格一致。
//
// 任一环节漏改（加了字段忘了写库、写了库忘了建列、建了列忘了加进 INSERT），
// 本测试即失败——这正是修订前「8 个空壳列」的成因。

// hanziDataTableColumns 读出 hanzi_data 的实际列名（PRAGMA table_info）
func hanziDataTableColumns(t *testing.T, s *Store) []string {
	t.Helper()
	rows, err := s.db.Query("PRAGMA table_info(hanzi_data)")
	if err != nil {
		t.Fatalf("读取 hanzi_data 表结构失败: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var cols []string
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
			t.Fatalf("解析 hanzi_data 表结构失败: %v", err)
		}
		cols = append(cols, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("遍历 hanzi_data 表结构失败: %v", err)
	}
	return cols
}

// insertColumns 从 hanziDataInsertSQL 中解析出列名与占位符数量
func insertColumns(t *testing.T) ([]string, int) {
	t.Helper()
	open := strings.Index(hanziDataInsertSQL, "(")
	close := strings.Index(hanziDataInsertSQL, ")")
	if open < 0 || close < open {
		t.Fatal("无法解析 hanziDataInsertSQL 的列清单")
	}
	cols := make([]string, 0)
	for _, c := range strings.Split(hanziDataInsertSQL[open+1:close], ",") {
		cols = append(cols, strings.TrimSpace(c))
	}
	placeholders := strings.Count(hanziDataInsertSQL, "?")
	return cols, placeholders
}

// TestSchema漂移_插入列必须是表列的子集 防止写了不存在的列（运行时才报错）
func TestSchema漂移_插入列必须是表列的子集(t *testing.T) {
	s := newTestStore(t)
	tableCols := hanziDataTableColumns(t, s)
	insertCols, _ := insertColumns(t)

	tableSet := make(map[string]bool, len(tableCols))
	for _, c := range tableCols {
		tableSet[c] = true
	}
	for _, c := range insertCols {
		if !tableSet[c] {
			t.Errorf("插入语句中的列 %q 在 hanzi_data 表中不存在", c)
		}
	}
}

// TestSchema漂移_插入列与表列必须完全一致 漏列同样是漂移（值永远进不了库）
func TestSchema漂移_插入列与表列必须完全一致(t *testing.T) {
	s := newTestStore(t)
	tableCols := hanziDataTableColumns(t, s)
	insertCols, _ := insertColumns(t)

	// updated_at 由数据库自己维护，不参与插入
	filtered := make([]string, 0, len(tableCols))
	for _, c := range tableCols {
		if c != "updated_at" {
			filtered = append(filtered, c)
		}
	}

	if !reflect.DeepEqual(filtered, insertCols) {
		t.Errorf("hanzi_data 表列与插入列不一致：\n表列=%v\n插入=%v", filtered, insertCols)
	}
}

// TestSchema漂移_占位符数量与列数一致 数量不符会在运行时才炸
func TestSchema漂移_占位符数量与列数一致(t *testing.T) {
	cols, placeholders := insertColumns(t)
	if len(cols) != placeholders {
		t.Errorf("插入语句列数 %d 与占位符数量 %d 不一致", len(cols), placeholders)
	}
}

// TestSchema漂移_行值数量与列数一致 防止改列忘改 hanziDataValues
func TestSchema漂移_行值数量与列数一致(t *testing.T) {
	cols, _ := insertColumns(t)
	values := hanziDataValues("字", hanzi.Hanzi{})
	if len(values) != len(cols) {
		t.Errorf("hanziDataValues 返回 %d 个值，插入语句为 %d 列", len(values), len(cols))
	}
}

// TestSchema漂移_补列清单无重复且均已在表中
func TestSchema漂移_补列清单无重复且均已在表中(t *testing.T) {
	s := newTestStore(t)
	tableCols := hanziDataTableColumns(t, s)
	tableSet := make(map[string]bool, len(tableCols))
	for _, c := range tableCols {
		tableSet[c] = true
	}

	seen := make(map[string]bool, len(hanziDataColumnMigrations))
	for _, m := range hanziDataColumnMigrations {
		if seen[m.column] {
			t.Errorf("补列清单中 %q 重复登记", m.column)
		}
		seen[m.column] = true
		if !tableSet[m.column] {
			t.Errorf("补列清单登记的 %q 未出现在 hanzi_data 表中", m.column)
		}
		if m.definition == "" {
			t.Errorf("补列清单中 %q 缺少列定义", m.column)
		}
	}
}

// TestSchema漂移_旧库补列可重复执行 用一个缺少新列的旧库结构验证迁移幂等
func TestSchema漂移_旧库补列可重复执行(t *testing.T) {
	s := newTestStore(t)

	// 模拟旧库：先删依赖 level 的索引，再删列
	if _, err := s.db.Exec("DROP INDEX IF EXISTS idx_hanzi_data_level"); err != nil {
		t.Fatalf("删除索引失败: %v", err)
	}
	for _, col := range []string{"kangxi_strokes", "level", "name_freq_tier"} {
		if _, err := s.db.Exec("ALTER TABLE hanzi_data DROP COLUMN " + col); err != nil {
			t.Fatalf("构造旧库失败（删除列 %s）: %v", col, err)
		}
	}

	// 重复执行迁移，列应被补齐且不报错
	for i := 0; i < 2; i++ {
		if err := migrateHanziDataColumns(s.writeDB); err != nil {
			t.Fatalf("第 %d 次补列迁移失败: %v", i+1, err)
		}
	}

	for _, col := range []string{"kangxi_strokes", "level", "name_freq_tier"} {
		found := false
		for _, c := range hanziDataTableColumns(t, s) {
			if c == col {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("补列后仍缺少列 %q", col)
		}
	}
}
