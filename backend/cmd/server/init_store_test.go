package main

// init_store_test.go — 启动期存储选择（docs/31 ADR-008 的 fail-fast 补强）
//
// 此前 SQLite 打开失败会静默回退内存存储：服务照常监听、healthz/readyz
// 全部通过，但收藏/历史在重启后全部消失，监控完全看不到持久化已失效。
// 现在只允许「显式配置 memory」时使用内存；SQLite 失败与未知驱动一律报错。

import (
	"path/filepath"
	"testing"

	"name/internal/infrastructure/database/memory"
	"name/internal/infrastructure/database/sqlite"
)

func TestInitStore(t *testing.T) {
	t.Run("显式 memory 驱动返回内存存储", func(t *testing.T) {
		store, err := initStore("memory", filepath.Join(t.TempDir(), "unused.db"), t.TempDir())
		if err != nil {
			t.Fatalf("initStore(memory) 返回错误: %v", err)
		}
		if _, ok := store.(*memory.Store); !ok {
			t.Fatalf("initStore(memory) = %T，期望 *memory.Store", store)
		}
	})

	t.Run("未知驱动拒绝启动", func(t *testing.T) {
		if _, err := initStore("postgres", "ignored.db", t.TempDir()); err == nil {
			t.Fatal("initStore(postgres) 应返回错误，但返回 nil")
		}
	})

	t.Run("SQLite 打开失败不得回退内存", func(t *testing.T) {
		// 父目录不存在 → sqlite 无法创建数据库文件；必须返回错误而非内存库。
		dbPath := filepath.Join(t.TempDir(), "no-such-dir", "namer.db")
		store, err := initStore("sqlite", dbPath, t.TempDir())
		if err == nil {
			t.Fatalf("initStore(sqlite) 在无效路径上应返回错误，实际得到 %T", store)
		}
		if store != nil {
			t.Fatalf("失败时不得返回存储实例，实际得到 %T", store)
		}
	})

	t.Run("SQLite 正常路径返回持久化存储", func(t *testing.T) {
		dir := t.TempDir()
		store, err := initStore("sqlite", filepath.Join(dir, "namer.db"), dir)
		if err != nil {
			t.Fatalf("initStore(sqlite) 返回错误: %v", err)
		}
		sqliteStore, ok := store.(*sqlite.Store)
		if !ok {
			t.Fatalf("initStore(sqlite) = %T，期望 *sqlite.Store", store)
		}
		// Windows 下不关闭会锁住临时目录，导致 t.TempDir 清理失败。
		if err := sqliteStore.Close(); err != nil {
			t.Fatalf("关闭 SQLite 失败: %v", err)
		}
	})
}
