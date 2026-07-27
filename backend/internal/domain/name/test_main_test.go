package name

import (
	"os"
	"testing"

	"name/internal/domain/hanzi"
)

func TestMain(m *testing.M) {
	// 加载汉字数据（起名生成器依赖 HanziData）
	dataDir := "../../../data/"
	if err := hanzi.LoadNamerFromJSON(dataDir); err != nil {
		// 非关键
	}

	os.Exit(m.Run())
}
