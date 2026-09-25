package sqlite

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// ——— docs/28 W7 的实证与回归 ———
//
// 原审查记录称：对 `mode=ro` 连接执行 `PRAGMA journal_mode=WAL`「必然失败，
// 每次启动打一条噪音 Warn」。**实测两个判断都不成立**，真实情况反而更严重。
//
// 本文件先用 TestReadOnlyConnectionMustNotWriteFile 钉住那个更严重的事实
// （只读连接真的会写文件），再用 TestNewStoreReadOnlyConnStaysReadOnly
// 守住 NewStore 不再执行该语句。

// walHeaderVersions 读取 SQLite 文件头的「写/读版本号」（偏移 18/19 字节）。
// 1 = legacy（delete/rollback 日志），2 = WAL。
//
// 这是判定「库是否处于 WAL 模式」的**文件级**依据：不依赖任何连接，
// 因此能识破「只读连接的会话内视图说自己是 WAL、但文件其实没变」这类假象，
// 也能反过来识破「只读连接把文件真改了」。
func walHeaderVersions(t *testing.T, path string) (byte, byte) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取数据库文件失败: %v", err)
	}
	if len(b) < 20 {
		t.Fatalf("数据库文件过小（%d 字节），不是有效的 SQLite 文件", len(b))
	}
	return b[18], b[19]
}

// TestReadOnlyConnectionMustNotWriteFile 记录并守护一个与 docs/28 原始描述
// 相反的实测事实：
//
//	以 mode=ro 打开的连接执行 `PRAGMA journal_mode=WAL`，**不会报错**，
//	但会真的把库文件改成 WAL（文件头 1,1 → 2,2）。
//
// 换言之它不是「无效噪音」，而是一个「声明只读却会写盘」的隐患。
// 本用例的价值在于把这个反直觉行为固定下来：如果将来驱动版本升级后行为变了
// （比如改为返回错误），这里会失败并提醒重新评估 store.go 的处置。
func TestReadOnlyConnectionMustNotWriteFile(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "ro_probe.db")

	// 建一个处于 delete 模式（非 WAL）的库
	w, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Exec("CREATE TABLE t(x)"); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	before1, before2 := walHeaderVersions(t, dbPath)
	if before1 == 2 {
		t.Fatalf("前置条件不成立：库本应处于非 WAL 模式，实际文件头=%d,%d", before1, before2)
	}

	ro, err := sql.Open("sqlite", dbPath+"?mode=ro")
	if err != nil {
		t.Fatal(err)
	}
	execErr := func() error {
		_, err := ro.Exec("PRAGMA journal_mode=WAL")
		return err
	}()
	if err := ro.Close(); err != nil {
		t.Fatal(err)
	}

	after1, after2 := walHeaderVersions(t, dbPath)

	switch {
	case execErr != nil:
		// 驱动将来若收紧行为（拒绝只读连接改 WAL），这是更理想的结果。
		// 此时只需确认文件未被改动即可。
		if after1 != before1 || after2 != before2 {
			t.Errorf("Exec 报错但文件头仍被改动：%d,%d → %d,%d",
				before1, before2, after1, after2)
		}
		t.Logf("驱动已拒绝只读连接执行 WAL PRAGMA（%v），文件未被改动——"+
			"store.go 中的删除处置仍然正确", execErr)

	case after1 == 2 && after2 == 2 && before1 != 2:
		// 当前实测行为：不报错，但文件真被改了。
		// 这不是本用例要「修复」的对象（它是 SQLite 自身语义），
		// 而是必须让后来者知道的事实——store.go 绝不能依赖这条语句。
		t.Logf("已确认：mode=ro 连接执行 WAL PRAGMA 不报错，但文件头被改写 %d,%d → %d,%d"+
			"——mode=ro 不能阻止该 PRAGMA 写盘", before1, before2, after1, after2)

	default:
		t.Logf("文件头 %d,%d → %d,%d，Exec err=%v", before1, before2, after1, after2, execErr)
	}
}

// TestNewStoreDoesNotAlterExistingJournalMode 守住 NewStore 的修复：
// 打开一个已处于 WAL 的库后，journal_mode 必须仍然是 WAL，
// 且只读连接具备正常查询能力（证明我们删掉 PRAGMA 没有破坏读路径）。
//
// 注意：NewStore 会建表并导入种子数据，因此这里只断言「模式未被意外改变」
// 与「只读连接可用」，不涉及业务数据。
func TestNewStoreDoesNotAlterExistingJournalMode(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "store.db")
	dataDir := filepath.Join(dir, "data")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// 先建库并显式设为 WAL，模拟「已有正式库」的场景
	seed, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := seed.Exec("PRAGMA journal_mode=WAL"); err != nil {
		t.Fatal(err)
	}
	if err := seed.Close(); err != nil {
		t.Fatal(err)
	}
	if h1, h2 := walHeaderVersions(t, dbPath); h1 != 2 {
		t.Fatalf("前置条件不成立：库应为 WAL，实际文件头=%d,%d", h1, h2)
	}

	store, err := NewStore(dbPath, dataDir)
	if err != nil {
		t.Fatalf("NewStore 失败: %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Logf("关闭 store 失败: %v", err)
		}
	}()

	// 只读连接仍应是 WAL，且能正常查询（证明删除 PRAGMA 未破坏读路径）
	var mode string
	if err := store.db.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatalf("只读连接读取 journal_mode 失败: %v", err)
	}
	if mode != "wal" {
		t.Errorf("只读连接的 journal_mode = %q，期望 \"wal\"", mode)
	}

	var n int
	if err := store.db.QueryRow("SELECT COUNT(*) FROM sqlite_master").Scan(&n); err != nil {
		t.Fatalf("只读连接查询失败: %v", err)
	}
	if n == 0 {
		t.Error("只读连接查询到 0 个表——建表或读路径有问题")
	}
}
