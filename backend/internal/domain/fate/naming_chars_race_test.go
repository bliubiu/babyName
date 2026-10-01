package fate

import (
	"fmt"
	"sync"
	"testing"
)

// 本文件固化 docs/29 A1 的修复回归：命名索引的懒初始化与增量写入
// 必须并发安全。旧实现用无同步原语的布尔值模拟 sync.Once，
// 并发首调会双双构建并写同一 map，触发不可恢复的 fatal。

// TestNamingIndexConcurrentFirstAccess 并发首次访问：多 goroutine 同时
// 首调读路径与写路径，不得触发 concurrent map writes fatal，且计数自洽
func TestNamingIndexConcurrentFirstAccess(t *testing.T) {
	// 重置包级状态以模拟「进程启动后第一次访问」（测试独占包内状态）
	namingOnce = sync.Once{}
	namingIdx = nil

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			switch i % 4 {
			case 0:
				_ = IsNamingChar("明")
			case 1:
				_ = GetNamingCharsByCategory("品德")
			case 2:
				_, _ = CountNamingChars()
			case 3:
				AddNamingChar(fmt.Sprintf("测%d", i), []string{"品德"}, "neutral")
			}
		}(i)
	}
	wg.Wait()

	total, _ := CountNamingChars()
	if total == 0 {
		t.Fatal("并发初始化后索引为空")
	}
}
