package namestat_test

import (
	"fmt"
	"os"
	"runtime"
	"testing"

	seedkit "name/internal/testutil/seed"
)

// TestMain 从种子库装载人名频率（ADR-010 单库路线，CI 零 JSON）
//
// 原先读 data/name_frequency.json；现在经 data.InitWithDB 从 name_frequency /
// name_bigram 表灌入，与运行期同路径。外部测试包避免 import cycle。
func TestMain(m *testing.M) {
	_, file, _, _ := runtime.Caller(0)
	cleanup, err := seedkit.Init(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "单库物料装配失败，namestat 包测试无法运行: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()
	os.Exit(m.Run())
}
