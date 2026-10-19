package hanzi

import (
	"fmt"
	"os"
	"runtime"
	"testing"

	"name/internal/testutil"
)

// TestMain 在包测试前加载汉字数据
func TestMain(m *testing.M) {
	// 数据目录按 namer.json 向上探测，不依赖固定目录层级（testutil.DataDir）
	_, file, _, _ := runtime.Caller(0)
	dataDir := testutil.DataDir(file)
	if dataDir == "" {
		fmt.Fprintln(os.Stderr, "未找到物料数据目录，hanzi 包测试无法运行")
		os.Exit(1)
	}
	// 加载 namer.json（含五行、起名分类等）
	// 康熙笔画必须在 namer 之前加载（namer_loader 末尾会合并康熙笔画）
	if err := LoadKangxiStrokesFromCSV(dataDir); err != nil {
		// 非关键：部分测试可能不依赖 kangxi 数据
	}
	if err := LoadNamerFromJSON(dataDir); err != nil {
		// 非关键：部分测试可能不依赖 namer 数据
	}

	os.Exit(m.Run())
}
