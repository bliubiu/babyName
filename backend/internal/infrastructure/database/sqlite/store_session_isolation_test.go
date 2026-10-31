package sqlite

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"name/internal/infrastructure/database"
)

const (
	sessionA = "0123456789abcdef0123456789abcdef"
	sessionB = "fedcba9876543210fedcba9876543210"
)

// TestHistorySessionIsolation 历史记录只能被所属会话读写、删除。
func TestHistorySessionIsolation(t *testing.T) {
	s := newTestStore(t)

	id := s.SaveHistory(&database.HistoryRecord{
		SessionID: sessionA,
		Surname:   "王",
		Gender:    "male",
		CreatedAt: time.Now(),
	})
	if id == "" {
		t.Fatal("会话 A 保存历史失败")
	}

	if got := s.GetHistory(sessionA); len(got) != 1 || got[0].ID != id {
		t.Fatalf("会话 A 应看到自己的历史，实得 %+v", got)
	}
	if got := s.GetHistory(sessionB); len(got) != 0 {
		t.Fatalf("会话 B 不应看到会话 A 的历史，实得 %d 条", len(got))
	}
	if _, total, err := s.GetHistoryPage(sessionB, 1, 20); err != nil || total != 0 {
		t.Fatalf("会话 B 分页总数应为 0，实得 total=%d err=%v", total, err)
	}

	if err := s.DeleteHistory(sessionB, id); err != nil {
		t.Fatalf("跨会话删除不应报错: %v", err)
	}
	if got := s.GetHistory(sessionA); len(got) != 1 {
		t.Fatalf("跨会话删除后会话 A 的记录应仍在，实得 %d 条", len(got))
	}

	if err := s.DeleteHistory(sessionA, id); err != nil {
		t.Fatalf("同会话删除失败: %v", err)
	}
	if got := s.GetHistory(sessionA); len(got) != 0 {
		t.Fatalf("同会话删除后应为空，实得 %d 条", len(got))
	}
}

// TestHistoryCrossSessionOverwriteRejected 用他人记录 ID 写入必须被拒绝且不篡改原数据。
func TestHistoryCrossSessionOverwriteRejected(t *testing.T) {
	s := newTestStore(t)

	id := s.SaveHistory(&database.HistoryRecord{
		SessionID: sessionA,
		Surname:   "王",
		CreatedAt: time.Now(),
	})
	if id == "" {
		t.Fatal("会话 A 保存历史失败")
	}

	if got := s.SaveHistory(&database.HistoryRecord{
		ID:        id,
		SessionID: sessionB,
		Surname:   "李",
		CreatedAt: time.Now(),
	}); got != "" {
		t.Fatalf("跨会话覆盖应返回空 ID，实得 %q", got)
	}

	recs := s.GetHistory(sessionA)
	if len(recs) != 1 || recs[0].Surname != "王" {
		t.Fatalf("会话 A 的历史被篡改: %+v", recs)
	}
}

// TestFavoriteSessionIsolation 不同会话可以收藏同名名字，且互相不可见。
func TestFavoriteSessionIsolation(t *testing.T) {
	s := newTestStore(t)

	idA, err := s.SaveFavorite(&database.FavoriteRecord{
		SessionID: sessionA, Surname: "王", GivenName: "涵", Score: 90,
	})
	if err != nil {
		t.Fatalf("会话 A 保存收藏失败: %v", err)
	}
	idB, err := s.SaveFavorite(&database.FavoriteRecord{
		SessionID: sessionB, Surname: "王", GivenName: "涵", Score: 85,
	})
	if err != nil {
		t.Fatalf("会话 B 应能收藏同名名字: %v", err)
	}
	if idA == idB {
		t.Fatal("不同会话的同名收藏不应共享同一条记录")
	}

	if got := s.GetFavorites(sessionA); len(got) != 1 || got[0].ID != idA {
		t.Fatalf("会话 A 应只看到自己的收藏，实得 %+v", got)
	}
	if got := s.GetFavorites(sessionB); len(got) != 1 || got[0].ID != idB {
		t.Fatalf("会话 B 应只看到自己的收藏，实得 %+v", got)
	}
	if got := s.GetFavoriteByName(sessionA, "王", "涵"); got == nil || got.ID != idA {
		t.Fatalf("会话 A 按名查询应命中自己的记录，实得 %+v", got)
	}

	if err := s.DeleteFavorite(sessionB, idA); err != nil {
		t.Fatalf("跨会话删除不应报错: %v", err)
	}
	if got := s.GetFavorites(sessionA); len(got) != 1 {
		t.Fatalf("跨会话删除后会话 A 的收藏应仍在，实得 %d 条", len(got))
	}
}

// TestFavoriteCrossSessionOverwriteRejected 收藏同样禁止跨会话按 ID 覆盖。
func TestFavoriteCrossSessionOverwriteRejected(t *testing.T) {
	s := newTestStore(t)

	id, err := s.SaveFavorite(&database.FavoriteRecord{
		SessionID: sessionA, Surname: "王", GivenName: "涵", Score: 90,
	})
	if err != nil {
		t.Fatalf("会话 A 保存收藏失败: %v", err)
	}

	if _, err := s.SaveFavorite(&database.FavoriteRecord{
		ID: id, SessionID: sessionB, Surname: "李", GivenName: "明", Score: 60,
	}); err == nil {
		t.Fatal("跨会话覆盖收藏应返回错误")
	}

	if got := s.GetFavoriteByName(sessionA, "王", "涵"); got == nil {
		t.Fatal("会话 A 的收藏被覆盖或删除")
	}
}

// TestMigrateSessionColumnsLegacyRowsHidden 存量库补齐 session_id 后，
// 无归属（session_id=”)的旧记录对任何会话都不可见。
func TestMigrateSessionColumnsLegacyRowsHidden(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "legacy.db")

	legacy, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("打开旧库失败: %v", err)
	}
	if _, err := legacy.Exec(`
		CREATE TABLE history (
			id TEXT PRIMARY KEY, surname TEXT, gender TEXT, birth_date TEXT,
			birth_time TEXT, birth_location TEXT, results TEXT, created_at DATETIME
		);
		CREATE TABLE favorites (
			id TEXT PRIMARY KEY, surname TEXT, given_name TEXT, pinyin TEXT,
			gender TEXT, score REAL, source TEXT, notes TEXT, created_at DATETIME
		);
		INSERT INTO history (id, surname, created_at) VALUES ('legacy-h', '王', CURRENT_TIMESTAMP);
		INSERT INTO favorites (id, surname, given_name, created_at) VALUES ('legacy-f', '王', '涵', CURRENT_TIMESTAMP);
	`); err != nil {
		_ = legacy.Close()
		t.Fatalf("构造旧库失败: %v", err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatalf("关闭旧库失败: %v", err)
	}

	s, err := NewStore(path, dir)
	if err != nil {
		t.Fatalf("迁移后打开旧库失败: %v", err)
	}
	defer func() { _ = s.Close() }()

	if got := s.GetHistory(sessionA); len(got) != 0 {
		t.Fatalf("无归属历史不应可见，实得 %d 条", len(got))
	}
	if got := s.GetFavorites(sessionA); len(got) != 0 {
		t.Fatalf("无归属收藏不应可见，实得 %d 条", len(got))
	}

	var count int
	if err := s.writeDB.QueryRow(`SELECT COUNT(*) FROM history WHERE id = 'legacy-h'`).Scan(&count); err != nil {
		t.Fatalf("查询旧记录失败: %v", err)
	}
	if count != 1 {
		t.Fatalf("迁移不应删除旧记录，实得 %d 条", count)
	}
}
