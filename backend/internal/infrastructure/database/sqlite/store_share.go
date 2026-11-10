package sqlite

import (
	"database/sql"
	"fmt"
	"time"

	"name/internal/infrastructure/database"
)

// SaveShare 写入分享快照。id 由应用层生成（crypto/rand），此处只做持久化。
// ON CONFLICT DO NOTHING：随机 id 碰撞时不覆盖既有内容，避免分享链接被改写。
func (s *Store) SaveShare(record *database.ShareRecord) error {
	if record == nil || record.ID == "" || record.Payload == "" {
		return fmt.Errorf("分享快照不完整")
	}
	createdAt := record.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	// 写必须走 writeDB：读连接以 mode=ro 打开，用 s.db.Exec 会直接报只读错误。
	if _, err := s.writeDB.Exec(
		`INSERT INTO shares (id, payload, created_at) VALUES (?, ?, ?)
		 ON CONFLICT(id) DO NOTHING`,
		record.ID, record.Payload, createdAt,
	); err != nil {
		return fmt.Errorf("保存分享快照失败: %w", err)
	}
	return nil
}

// GetShare 按 id 读取分享快照；不存在返回 (nil, nil)，由应用层转 404。
func (s *Store) GetShare(id string) (*database.ShareRecord, error) {
	var record database.ShareRecord
	err := s.db.QueryRow(
		`SELECT id, payload, created_at FROM shares WHERE id = ?`, id,
	).Scan(&record.ID, &record.Payload, &record.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("读取分享快照失败: %w", err)
	}
	return &record, nil
}
