package yijing_test

import (
	"fmt"
	"os"
	"runtime"
	"testing"

	seedkit "name/internal/testutil/seed"
)

// TestMain 从种子库装配全部物料（ADR-010 单库路线，CI 零 JSON）
//
// 外部测试包 package yijing_test 避免 data → yijing 的 import cycle。
// 原先只装 namer.json 供梅花易数取笔画；现在同时灌入六十四卦数据。
func TestMain(m *testing.M) {
	_, file, _, _ := runtime.Caller(0)
	cleanup, err := seedkit.InitHanziOnly(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "单库物料装配失败，yijing 包测试无法运行: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()
	os.Exit(m.Run())
}
