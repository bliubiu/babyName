package sqlite

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
	"name/internal/infrastructure/database"
	"name/internal/infrastructure/logger"
)

func (s *Store) SaveHistory(record *database.HistoryRecord) string {
	if record.ID == "" {
		record.ID = uuid.New().String()
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now().UTC() // docs/43 P2-11：时间口径统一 UTC（存量由 v1 迁移归一）
	}

	// ON CONFLICT ... WHERE 只允许同一会话更新自己的记录：即使调用方
	// 传入他人记录的 ID，UPDATE 也不会执行（RowsAffected=0），由这里显式报错。
	result, err := s.writeDB.Exec(
		`INSERT INTO history (id, session_id, surname, gender, birth_date, birth_time, birth_location, results, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET surname=excluded.surname, gender=excluded.gender,
		   birth_date=excluded.birth_date, birth_time=excluded.birth_time,
		   birth_location=excluded.birth_location, results=excluded.results, created_at=excluded.created_at
		 WHERE history.session_id = excluded.session_id`,
		record.ID, record.SessionID, record.Surname, record.Gender, record.BirthDate,
		record.BirthTime, record.BirthLocation, record.Results, record.CreatedAt,
	)
	if err != nil {
		logger.Error("sqlite: 保存历史失败", logger.ErrField(err))
		// 失败时返回空字符串，让上层感知错误
		return ""
	}
	if n, err := result.RowsAffected(); err == nil && n == 0 {
		logger.Warn("sqlite: SaveHistory 拒绝跨会话覆盖", logger.String("id", record.ID))
		return ""
	}
	return record.ID
}

func (s *Store) GetHistory(sessionID string) []*database.HistoryRecord {
	// 默认限制返回100条记录，防止内存溢出
	return s.GetHistoryPageWithLimit(sessionID, 1, 100)
}

// GetHistoryPageWithLimit 获取分页历史记录（内部方法）
func (s *Store) GetHistoryPageWithLimit(sessionID string, page, limit int) []*database.HistoryRecord {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * limit
	rows, err := s.db.Query(`SELECT id, surname, gender, birth_date, birth_time, birth_location, results, created_at FROM history WHERE session_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`, sessionID, limit, offset)
	if err != nil {
		logger.Error("sqlite: 分页查询历史失败", logger.ErrField(err))
		return nil
	}
	defer rows.Close()

	var records []*database.HistoryRecord
	for rows.Next() {
		var r database.HistoryRecord
		if err := rows.Scan(&r.ID, &r.Surname, &r.Gender, &r.BirthDate, &r.BirthTime, &r.BirthLocation, &r.Results, &r.CreatedAt); err != nil {
			logger.Error("sqlite: 分页查询历史：扫描失败", logger.ErrField(err))
			continue
		}
		r.SessionID = sessionID
		records = append(records, &r)
	}
	if err := rows.Err(); err != nil {
		// 迭代中途的 IO/解码错误此前被当成「正常取完」，静默截断数据（docs/29 P3）
		logger.Error("sqlite: 遍历结果集中途失败", logger.ErrField(err))
	}

	return records
}

func (s *Store) GetHistoryPage(sessionID string, page, limit int) ([]*database.HistoryRecord, int, error) {
	var total int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM history WHERE session_id = ?`, sessionID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count history: %w", err)
	}

	offset := (page - 1) * limit
	rows, err := s.db.Query(`SELECT id, surname, gender, birth_date, birth_time, birth_location, results, created_at FROM history WHERE session_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`, sessionID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query history page: %w", err)
	}
	defer rows.Close()

	var records []*database.HistoryRecord
	for rows.Next() {
		var r database.HistoryRecord
		if err := rows.Scan(&r.ID, &r.Surname, &r.Gender, &r.BirthDate, &r.BirthTime, &r.BirthLocation, &r.Results, &r.CreatedAt); err != nil {
			logger.Error("sqlite: 查询历史：扫描失败", logger.ErrField(err))
			continue
		}
		r.SessionID = sessionID
		records = append(records, &r)
	}
	if err := rows.Err(); err != nil {
		// 迭代中途的 IO/解码错误此前被当成「正常取完」，静默截断数据（docs/29 P3）
		logger.Error("sqlite: 遍历结果集中途失败", logger.ErrField(err))
	}

	return records, total, nil
}

func (s *Store) GetHistoryByID(sessionID, id string) *database.HistoryRecord {
	var r database.HistoryRecord
	err := s.db.QueryRow(`SELECT id, surname, gender, birth_date, birth_time, birth_location, results, created_at FROM history WHERE id = ? AND session_id = ?`, id, sessionID).
		Scan(&r.ID, &r.Surname, &r.Gender, &r.BirthDate, &r.BirthTime, &r.BirthLocation, &r.Results, &r.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		logger.Error("sqlite: 按 ID 查询历史失败", logger.ErrField(err))
		return nil
	}
	r.SessionID = sessionID
	return &r
}

func (s *Store) DeleteHistory(sessionID, id string) error {
	_, err := s.writeDB.Exec(`DELETE FROM history WHERE id = ? AND session_id = ?`, id, sessionID)
	if err != nil {
		return fmt.Errorf("delete history: %w", err)
	}
	return nil
}

func (s *Store) BatchDeleteHistory(sessionID string, ids []string) error {
	tx, err := s.writeDB.Begin()
	if err != nil {
		return fmt.Errorf("batch delete history begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(`DELETE FROM history WHERE id = ? AND session_id = ?`)
	if err != nil {
		return fmt.Errorf("batch delete history prepare: %w", err)
	}
	defer stmt.Close()

	for _, id := range ids {
		if _, err := stmt.Exec(id, sessionID); err != nil {
			return fmt.Errorf("batch delete history exec: %w", err)
		}
	}
	return tx.Commit()
}

// --- FavoriteStore ---
