package hanzi_test

import (
	"fmt"
	"os"
	"runtime"
	"testing"

	seedkit "name/internal/testutil/seed"
)

// TestMain 从种子库装配全部物料（ADR-010 单库路线，CI 零 JSON）
//
// 外部测试包 package hanzi_test 避免 data → hanzi → testutil 的 import cycle。
// 原先只装 kangxi CSV + namer.json；现在经 data.InitWithDB 从 namer.db
// 读 typed 行，康熙笔画走 kangxi_strokes 表（ADR-013）。
func TestMain(m *testing.M) {
	_, file, _, _ := runtime.Caller(0)
	cleanup, err := seedkit.Init(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "单库物料装配失败，hanzi 包测试无法运行: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()
	os.Exit(m.Run())
}
