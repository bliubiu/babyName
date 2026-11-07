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

func (s *Store) SaveFavorite(record *database.FavoriteRecord) (string, error) {
	if record.ID == "" {
		record.ID = uuid.New().String()
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now().UTC() // docs/43 P2-11：时间口径统一 UTC（存量由 v1 迁移归一）
	}

	result, err := s.writeDB.Exec(
		`INSERT INTO favorites (id, session_id, surname, given_name, pinyin, gender, score, source, notes, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET surname=excluded.surname, given_name=excluded.given_name,
		   pinyin=excluded.pinyin, gender=excluded.gender, score=excluded.score,
		   source=excluded.source, notes=excluded.notes, created_at=excluded.created_at
		 WHERE favorites.session_id = excluded.session_id`,
		record.ID, record.SessionID, record.Surname, record.GivenName, record.Pinyin, record.Gender,
		record.Score, record.Source, record.Notes, record.CreatedAt,
	)
	if err != nil {
		logger.Error("sqlite: 保存收藏失败", logger.ErrField(err))
		// 写入失败必须把错误上抛（docs/29 A6）：此前吞错仍返回 ID，
		// 服务层据此返回成功，收藏静默丢失且自学习照常入精选库
		return "", err
	}
	if n, err := result.RowsAffected(); err == nil && n == 0 {
		// 传入的 ID 属于其他会话：拒绝写入，避免跨会话覆盖/劫持收藏。
		logger.Warn("sqlite: SaveFavorite 拒绝跨会话覆盖", logger.String("id", record.ID))
		return "", fmt.Errorf("收藏记录不存在或无权修改")
	}
	return record.ID, nil
}

func (s *Store) GetFavorites(sessionID string) []*database.FavoriteRecord {
	rows, err := s.db.Query(`SELECT id, surname, given_name, pinyin, gender, score, source, notes, created_at FROM favorites WHERE session_id = ? ORDER BY created_at DESC`, sessionID)
	if err != nil {
		logger.Error("sqlite: 查询收藏失败", logger.ErrField(err))
		return nil
	}
	defer rows.Close()

	var records []*database.FavoriteRecord
	for rows.Next() {
		var r database.FavoriteRecord
		if err := rows.Scan(&r.ID, &r.Surname, &r.GivenName, &r.Pinyin, &r.Gender, &r.Score, &r.Source, &r.Notes, &r.CreatedAt); err != nil {
			logger.Error("sqlite: 查询收藏：扫描失败", logger.ErrField(err))
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

func (s *Store) GetFavoritesPage(sessionID string, page, limit int) ([]*database.FavoriteRecord, int, error) {
	var total int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM favorites WHERE session_id = ?`, sessionID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count favorites: %w", err)
	}

	offset := (page - 1) * limit
	rows, err := s.db.Query(`SELECT id, surname, given_name, pinyin, gender, score, source, notes, created_at FROM favorites WHERE session_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`, sessionID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query favorites page: %w", err)
	}
	defer rows.Close()

	var records []*database.FavoriteRecord
	for rows.Next() {
		var r database.FavoriteRecord
		if err := rows.Scan(&r.ID, &r.Surname, &r.GivenName, &r.Pinyin, &r.Gender, &r.Score, &r.Source, &r.Notes, &r.CreatedAt); err != nil {
			logger.Error("sqlite: 分页查询收藏：扫描失败", logger.ErrField(err))
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

func (s *Store) DeleteFavorite(sessionID, id string) error {
	_, err := s.writeDB.Exec(`DELETE FROM favorites WHERE id = ? AND session_id = ?`, id, sessionID)
	if err != nil {
		return fmt.Errorf("delete favorite: %w", err)
	}
	return nil
}

func (s *Store) BatchDeleteFavorite(sessionID string, ids []string) error {
	tx, err := s.writeDB.Begin()
	if err != nil {
		return fmt.Errorf("batch delete favorites begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(`DELETE FROM favorites WHERE id = ? AND session_id = ?`)
	if err != nil {
		return fmt.Errorf("batch delete favorites prepare: %w", err)
	}
	defer stmt.Close()

	for _, id := range ids {
		if _, err := stmt.Exec(id, sessionID); err != nil {
			return fmt.Errorf("batch delete favorites exec: %w", err)
		}
	}
	return tx.Commit()
}

func (s *Store) GetFavoriteByName(sessionID, surname, givenName string) *database.FavoriteRecord {
	var r database.FavoriteRecord
	err := s.db.QueryRow(`SELECT id, surname, given_name, pinyin, gender, score, source, notes, created_at FROM favorites WHERE session_id = ? AND surname = ? AND given_name = ?`, sessionID, surname, givenName).
		Scan(&r.ID, &r.Surname, &r.GivenName, &r.Pinyin, &r.Gender, &r.Score, &r.Source, &r.Notes, &r.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		logger.Error("sqlite: 按名查询收藏失败", logger.ErrField(err))
		return nil
	}
	r.SessionID = sessionID
	return &r
}

// --- YijingStore ---
