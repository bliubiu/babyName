package sqlite

import (
	"database/sql"
	"fmt"
)

// 本文件实现「运行时统一只读 SQLite 数据源」（docs/31 ADR-008）所需的读取路径。
//
// 定位：物料在导入期已归一化并落库，运行时只做「从库读出 typed 结构」，
// 不再解析 JSON/CSV，也不再做字段归一化。

// MaterialState 物料的库内状态
type MaterialState struct {
	Material   string `json:"material"`
	Version    string `json:"version"`
	Checksum   string `json:"checksum"`
	RowCount   int    `json:"rowCount"`
	ImportedAt string `json:"importedAt"`
}

// MaterialStates 列出全部物料的版本与行数（按物料名排序）
func (s *Store) MaterialStates() ([]MaterialState, error) {
	rows, err := s.db.Query(`SELECT material, version, checksum, row_count,
		COALESCE(CAST(imported_at AS TEXT), '') FROM materials_state ORDER BY material`)
	if err != nil {
		if isMissingTableErr(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询物料状态失败: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []MaterialState
	for rows.Next() {
		var st MaterialState
		if err := rows.Scan(&st.Material, &st.Version, &st.Checksum, &st.RowCount, &st.ImportedAt); err != nil {
			return nil, fmt.Errorf("扫描物料状态失败: %w", err)
		}
		out = append(out, st)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历物料状态失败: %w", err)
	}
	return out, nil
}

// storedMaterialVersion 读出库内记录的物料版本（未记录返回空串）
func (s *Store) storedMaterialVersion(material string) string {
	var v string
	if err := s.db.QueryRow("SELECT version FROM materials_state WHERE material = ?", material).Scan(&v); err != nil {
		return ""
	}
	return v
}

// MaterialRowCount 读取单个物料的落库行数
func (s *Store) MaterialRowCount(material string) (int, error) {
	tables := materialTablesOf(material)
	if len(tables) == 0 {
		return 0, fmt.Errorf("未登记的物料: %s", material)
	}
	total := 0
	for _, t := range tables {
		var n int
		if err := s.db.QueryRow("SELECT COUNT(*) FROM " + t).Scan(&n); err != nil {
			if isMissingTableErr(err) {
				return 0, nil
			}
			return 0, fmt.Errorf("统计 %s 行数失败: %w", t, err)
		}
		total += n
	}
	return total, nil
}

// HasMaterial 判断某物料是否已导入（按 materials_state 记录 + 表非空双重确认）
func (s *Store) HasMaterial(material string) bool {
	var n int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM materials_state WHERE material = ?", material).Scan(&n); err != nil || n == 0 {
		return false
	}
	count, err := s.MaterialRowCount(material)
	return err == nil && count > 0
}

// queryStrings 读取单列字符串结果集
func (s *Store) queryStrings(query string, args ...any) ([]string, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			continue
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// execOn 对可空错误做静默处理的执行辅助（导入路径统一使用）
func execOn(db *sql.DB, stmt string, args ...any) error {
	_, err := db.Exec(stmt, args...)
	return err
}
