package hanzi

import (
	"os"
	"testing"
)

// TestMain 在包测试前加载汉字数据
func TestMain(m *testing.M) {
	// 加载 namer.json（含五行、起名分类等）
	dataDir := "../../../data/"
	// 康熙笔画必须在 namer 之前加载（namer_loader 末尾会合并康熙笔画）
	if err := LoadKangxiStrokesFromCSV(dataDir); err != nil {
		// 非关键：部分测试可能不依赖 kangxi 数据
	}
	if err := LoadNamerFromJSON(dataDir); err != nil {
		// 非关键：部分测试可能不依赖 namer 数据
	}

	os.Exit(m.Run())
}
