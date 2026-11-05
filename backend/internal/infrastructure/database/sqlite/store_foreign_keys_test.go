package sqlite

// store_foreign_keys_test.go — 外键约束启用回归（docs/43 P1-5）
//
// 背景：name_feedback.request_id 与 classics_sections.book_id 声明了
// FOREIGN KEY，但连接串从未启用 PRAGMA foreign_keys（SQLite 默认关闭），
// 约束完全不生效，孤儿行可写入。
//
// 修复：写连接串追加 _pragma=foreign_keys(ON)（modernc.org/sqlite 的
// 连接串参数在每个新连接建立时执行，不依赖调用顺序；读连接为 mode=ro
// 不执行写路径，无需启用）。本文件守护：
//   - 孤儿行（父表无对应 id）插入必须失败
//   - 先父后子的正常写入顺序不受影响（与 importBookObject 实际顺序一致）

import (
	"testing"
)

// TestForeignKeys_OrphanInsertMustFail 孤儿行插入必须被外键约束拒绝
func TestForeignKeys_OrphanInsertMustFail(t *testing.T) {
	s := newTestStore(t)

	if _, err := s.writeDB.Exec(
		`INSERT INTO classics_sections (book_id, title, sort_order) VALUES (999999, '孤儿章节', 0)`); err == nil {
		t.Fatal("book_id=999999（classics_books 无此行）插入成功——外键未启用")
	}

	if _, err := s.writeDB.Exec(
		`INSERT INTO name_feedback (request_id, full_name, given_name, is_liked, is_selected, algorithm_score, created_at)
		 VALUES (999999, '张三丰', '三丰', 1, 0, 90.0, '2026-11-06 00:00:00')`); err == nil {
		t.Fatal("request_id=999999（name_requests 无此行）插入成功——外键未启用")
	}
}

// TestForeignKeys_ParentThenChildSucceeds 先父后子的正常写入不受约束影响
func TestForeignKeys_ParentThenChildSucceeds(t *testing.T) {
	s := newTestStore(t)

	res, err := s.writeDB.Exec(
		`INSERT INTO classics_books (file_name, category, title) VALUES ('fk_test.json', '经部', '外键测试书')`)
	if err != nil {
		t.Fatalf("插入父表 classics_books 失败: %v", err)
	}
	bookID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("读取 LastInsertId 失败: %v", err)
	}

	if _, err := s.writeDB.Exec(
		`INSERT INTO classics_sections (book_id, title, sort_order) VALUES (?, '合法章节', 0)`, bookID); err != nil {
		t.Fatalf("先父后子插入 classics_sections 失败: %v", err)
	}

	reqRes, err := s.writeDB.Exec(
		`INSERT INTO name_requests (surname, gender, request_hash, created_at) VALUES ('张', 'male', 'fk-test-hash', '2026-11-06 00:00:00')`)
	if err != nil {
		t.Fatalf("插入父表 name_requests 失败: %v", err)
	}
	reqID, err := reqRes.LastInsertId()
	if err != nil {
		t.Fatalf("读取 name_requests LastInsertId 失败: %v", err)
	}

	if _, err := s.writeDB.Exec(
		`INSERT INTO name_feedback (request_id, full_name, given_name, is_liked, is_selected, algorithm_score, created_at)
		 VALUES (?, '张三丰', '三丰', 1, 0, 90.0, '2026-11-06 00:00:00')`, reqID); err != nil {
		t.Fatalf("先父后子插入 name_feedback 失败: %v", err)
	}
}
