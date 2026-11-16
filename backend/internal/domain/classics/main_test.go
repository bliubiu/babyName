package classics_test

import (
	"fmt"
	"os"
	"runtime"
	"testing"

	seedkit "name/internal/testutil/seed"
)

// TestMain 从种子库装配经典语料（ADR-010 单库路线，CI 零 JSON）
//
// 原先 LoadFromJSON 走 shici.json 异步装载；现在经 data.InitWithDB →
// LoadClassicsFromDB 从 classics_extracts / classics_poem_entries 灌入，
// 与生产同路径，也不再存在等待 shiciReady 通道的时序问题。
func TestMain(m *testing.M) {
	_, file, _, _ := runtime.Caller(0)
	cleanup, err := seedkit.Init(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "单库物料装配失败，classics 包测试无法运行: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()
	os.Exit(m.Run())
}
