package memory

import (
	"fmt"
	"time"

	"name/internal/infrastructure/database"
)

// SaveShare 内存实现：与 SQLite 同语义，已存在的 id 不覆盖。
func (s *Store) SaveShare(record *database.ShareRecord) error {
	if record == nil || record.ID == "" || record.Payload == "" {
		return fmt.Errorf("分享快照不完整")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.Shares[record.ID]; exists {
		return nil
	}
	copied := *record
	if copied.CreatedAt.IsZero() {
		copied.CreatedAt = time.Now().UTC()
	}
	s.Shares[record.ID] = &copied
	return nil
}

// GetShare 不存在返回 (nil, nil)，由应用层转 404。
func (s *Store) GetShare(id string) (*database.ShareRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	record, ok := s.Shares[id]
	if !ok {
		return nil, nil
	}
	copied := *record
	return &copied, nil
}
