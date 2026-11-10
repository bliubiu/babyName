package sqlite

import (
	"testing"
	"time"

	"name/internal/infrastructure/database"
)

// TestGetHistoryPage_ClampsNegativeLimit 回归（docs/46 P2）：SQLite 下 LIMIT=-1
// 等价于「不限制」，此前会拉取全表；钳制后应回落为单页上限 100。
func TestGetHistoryPage_ClampsNegativeLimit(t *testing.T) {
	s := newTestStore(t)
	const n = 150
	for i := 0; i < n; i++ {
		if id := s.SaveHistory(&database.HistoryRecord{
			SessionID: sessionA, Surname: "王", Gender: "male", CreatedAt: time.Now(),
		}); id == "" {
			t.Fatalf("保存第 %d 条历史失败", i)
		}
	}

	recs, total, err := s.GetHistoryPage(sessionA, 1, -1)
	if err != nil {
		t.Fatalf("GetHistoryPage 失败: %v", err)
	}
	if total != n {
		t.Fatalf("total 应为 %d，实得 %d", n, total)
	}
	if len(recs) != 100 {
		t.Fatalf("limit=-1 应被钳制为 100，实得 %d 条", len(recs))
	}
}

// TestGetHistoryPage_ClampsOverflowLimit 回归：limit 超过单页上限 1000 时回落为 100。
func TestGetHistoryPage_ClampsOverflowLimit(t *testing.T) {
	s := newTestStore(t)
	for i := 0; i < 150; i++ {
		s.SaveHistory(&database.HistoryRecord{
			SessionID: sessionA, Surname: "王", Gender: "male", CreatedAt: time.Now(),
		})
	}
	recs, _, err := s.GetHistoryPage(sessionA, 1, 5000)
	if err != nil {
		t.Fatalf("GetHistoryPage 失败: %v", err)
	}
	if len(recs) != 100 {
		t.Fatalf("limit=5000 应被钳制为 100，实得 %d 条", len(recs))
	}
}

// TestGetHistoryPage_ClampsNegativePage 回归：page<1 回落为 1。
func TestGetHistoryPage_ClampsNegativePage(t *testing.T) {
	s := newTestStore(t)
	for i := 0; i < 10; i++ {
		s.SaveHistory(&database.HistoryRecord{
			SessionID: sessionA, Surname: "王", Gender: "male", CreatedAt: time.Now(),
		})
	}
	recs, _, err := s.GetHistoryPage(sessionA, -5, 20)
	if err != nil {
		t.Fatalf("GetHistoryPage 失败: %v", err)
	}
	if len(recs) != 10 {
		t.Fatalf("page=-5 应钳制为 page=1 并返回全部 10 条，实得 %d 条", len(recs))
	}
}

// TestGetFavoritesPage_ClampsNegativeLimit 回归：收藏分页同样钳制 limit=-1。
func TestGetFavoritesPage_ClampsNegativeLimit(t *testing.T) {
	s := newTestStore(t)
	for i := 0; i < 150; i++ {
		if _, err := s.SaveFavorite(&database.FavoriteRecord{
			SessionID: sessionA, Surname: "王", GivenName: "小明", Gender: "male", Score: 90,
		}); err != nil {
			t.Fatalf("保存第 %d 条收藏失败: %v", i, err)
		}
	}
	recs, total, err := s.GetFavoritesPage(sessionA, 1, -1)
	if err != nil {
		t.Fatalf("GetFavoritesPage 失败: %v", err)
	}
	if total != 150 {
		t.Fatalf("total 应为 150，实得 %d", total)
	}
	if len(recs) != 100 {
		t.Fatalf("limit=-1 应被钳制为 100，实得 %d 条", len(recs))
	}
}
