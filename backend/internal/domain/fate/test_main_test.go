package fate_test

import (
	"fmt"
	"os"
	"runtime"
	"testing"

	seedkit "name/internal/testutil/seed"
)

// TestMain 从种子库装配全部物料（ADR-010 单库路线，CI 零 JSON）
//
// 必须用外部测试包 package fate_test：data.InitWithDB 会导入 fate 本身，
// 若 TestMain 放在 package fate 里会形成 import cycle。包内测试文件继续
// 使用轻量的 internal/testutil，不受影响。
//
// 改造前本包只装三份 JSON：classics / naming_quality / forbidden_combos，
// 与运行期 SQLite 路径不是同一套数据语义。现在统一走 data.InitWithDB，
// 门禁字表、禁忌组合、白名单、合规红线、经典语料一次性灌入。
func TestMain(m *testing.M) {
	_, file, _, _ := runtime.Caller(0)
	cleanup, err := seedkit.Init(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "单库物料装配失败，fate 包测试无法运行: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()
	os.Exit(m.Run())
}
