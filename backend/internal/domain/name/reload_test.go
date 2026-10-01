package name

// reload_test.go — NameDB 热更新（docs/29 B11）
//
// B11 的原始病灶：Reload() 先 db.mu.Lock() 再调 Load()，而 Load() 的四个
// 子加载器各自 db.mu.Lock()。sync.RWMutex 不可重入，第二次 Lock 永久阻塞 ——
// 热更新接口表现为「一直转圈直到超时」，没有任何 panic 和日志。
//
// 用例必须带超时守护：死锁时 go test 会整体挂住、连测试二进制都退不出来，
// 那比失败更难定位。故用 goroutine + select 把它变成确定性的 FAIL。

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// reloadWithTimeout 在 timeout 内执行 fn；超时即判定为死锁并让测试失败。
func reloadWithTimeout(t *testing.T, timeout time.Duration, fn func() error) error {
	t.Helper()
	done := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- nil
				t.Errorf("热更新过程中 panic：%v", r)
			}
		}()
		done <- fn()
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		t.Fatalf("热更新在 %v 内未返回：db.mu 疑似被重入加锁（死锁）。"+
			"检查 Reload 是否在持锁状态下调用了 Load（RWMutex 不可重入）", timeout)
		return nil
	}
}

func newReloadTestDB(t *testing.T) *NameDB {
	t.Helper()
	tmpDir := t.TempDir()
	// 写入最小可解析的精选名库，保证 Load 的 JSON 分支真实执行
	content := `[{"name":"测试名","pinyin":"ce shi ming","gender":"男","score":90,"source":"test"}]`
	if err := os.WriteFile(filepath.Join(tmpDir, "curated_names.json"), []byte(content), 0644); err != nil {
		t.Fatalf("写入 curated_names.json 失败：%v", err)
	}
	db, err := NewNameDB(tmpDir, WithCuratedPersister(nil))
	if err != nil {
		t.Fatalf("NewNameDB 失败：%v", err)
	}
	return db
}

// TestNameDB_Reload_DoesNotDeadlock 热更新不得因重入加锁而死锁
func TestNameDB_Reload_DoesNotDeadlock(t *testing.T) {
	db := newReloadTestDB(t)

	// 预热：确认初始加载成功，避免把「本来就空」误判为 Reload 无效
	if got := len(db.GetCuratedNames("男")); got == 0 {
		t.Fatal("初始加载后应存在精选名，测试前提不成立")
	}

	if err := reloadWithTimeout(t, 10*time.Second, db.Reload); err != nil {
		t.Fatalf("Reload 返回错误：%v", err)
	}

	// 重载后数据应被完整重建，而不是被清空后未回填
	if got := len(db.GetCuratedNames("男")); got == 0 {
		t.Error("Reload 后精选名丢失：索引被清空但未重建")
	}
	if !db.IsCuratedName("测试名") {
		t.Error("Reload 后 nameIndex 未重建，IsCuratedName 查不到已加载的名字")
	}
}

// TestNameDB_Reload_ConcurrentWithReaders 热更新与并发读不得崩溃
//
// 修复后 Reload 的锁窗口只有「清空索引」一瞬，读侧 RLock 与之互斥，
// 不应出现 fatal error: concurrent map read and map write。
func TestNameDB_Reload_ConcurrentWithReaders(t *testing.T) {
	db := newReloadTestDB(t)

	done := make(chan struct{})
	readerDone := make(chan struct{})

	// 读侧
	go func() {
		defer close(readerDone)
		for {
			select {
			case <-done:
				return
			default:
			}
			for i := 0; i < 50; i++ {
				_ = db.GetCuratedNames("男")
				_ = db.IsCuratedName("测试名")
				_ = db.GetCharGroups()
				_ = db.GetTopCuratedNames("男", 10)
				_ = db.SearchCuratedNames("测试")
			}
		}
	}()

	// 写侧：连续多次热更新
	for i := 0; i < 3; i++ {
		if err := reloadWithTimeout(t, 10*time.Second, db.Reload); err != nil {
			close(done)
			<-readerDone
			t.Fatalf("第 %d 次 Reload 返回错误：%v", i+1, err)
		}
	}

	close(done)
	select {
	case <-readerDone:
	case <-time.After(10 * time.Second):
		t.Fatal("读侧 goroutine 未退出")
	}
}