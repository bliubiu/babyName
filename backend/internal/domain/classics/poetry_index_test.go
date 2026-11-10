package classics

import "testing"

// TestBuildPoetryIndex_Idempotent 主动预热不应 panic，且重复调用为无操作（docs/46 P2）。
//
// 首请求延迟尖峰前移到启动期的前提是 BuildPoetryIndex 可被安全重复调用：
// 内部 sync.Once 保证只构建一次，第二次调用直接返回。本用例验证幂等性，
// 不依赖经典语料物料（硬编码 PoetrySources 始终可用）。
func TestBuildPoetryIndex_Idempotent(t *testing.T) {
	BuildPoetryIndex()
	first := CountPoetryEntry()

	BuildPoetryIndex() // 第二次应为无操作
	if got := CountPoetryEntry(); got != first {
		t.Errorf("重复调用 BuildPoetryIndex 后条目数变化：%d -> %d", first, got)
	}
}
