package yijing

import (
	"os"
	"testing"

	"name/internal/domain/hanzi"
)

func TestMain(m *testing.M) {
	// 加载汉字数据（梅花易数依赖 HanziData 中的笔画数）
	dataDir := "../../../data/"
	if err := hanzi.LoadNamerFromJSON(dataDir); err != nil {
		// 非关键：部分测试可能不依赖汉字数据
	}

	os.Exit(m.Run())
}
