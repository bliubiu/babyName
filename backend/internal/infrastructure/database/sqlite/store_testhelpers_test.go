package sqlite

import (
	"path/filepath"
	"testing"
)

// 本文件保存 sqlite 包测试共用的 Store 构造助手。
//
// 原先这些助手与 hanzi_data 的 legacy 投影路径（seedHanziData /
// ReloadHanziData / hanziDataValues）同住 store_hanzi_seed_test.go。
// 该投影路径已按 docs/34 阶段 A1 整体移除（hanzi_data 唯一写入方为
// 物料导入路径 importNamer），其专属守护测试随之删除，
// 仅保留被多个测试文件复用的 newTestStore。

// newTestStore 建一个临时库 Store（不触发真实物料装载）
func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s, err := NewStore(filepath.Join(dir, "test.db"), dir)
	if err != nil {
		t.Fatalf("创建测试 Store 失败: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}
