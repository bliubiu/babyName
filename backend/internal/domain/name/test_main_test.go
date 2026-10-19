package name

import (
	"fmt"
	"os"
	"runtime"
	"testing"

	"name/internal/domain/hanzi"
	"name/internal/testutil"
)

func TestMain(m *testing.M) {
	_, file, _, _ := runtime.Caller(0)
	dataDir := testutil.DataDir(file)
	if dataDir == "" {
		fmt.Fprintln(os.Stderr, "未找到物料数据目录，name 包测试无法运行")
		os.Exit(1)
	}
	// 加载汉字数据（起名生成器依赖 HanziData）
	if err := hanzi.LoadNamerFromJSON(dataDir); err != nil {
		// 非关键
	}

	os.Exit(m.Run())
}
