package fate

import (
	"fmt"
	"os"
	"runtime"
	"testing"

	"name/internal/domain/classics"
	"name/internal/testutil"
)

// TestMain 在包测试前加载诗词经典数据
// WenHuaRater 的单名语义共现分支（checkSingleNameBigram）依赖 classics 包，
// 该包通过 ensureShiCiLoaded 惰性等待 shici.json 异步加载完成，
// 独立运行测试时必须先 LoadFromJSON 触发加载，否则会永久阻塞超时。
// 数据目录按 namer.json 向上探测（testutil.DataDir），不依赖固定目录层级。
func TestMain(m *testing.M) {
	_, file, _, _ := runtime.Caller(0)
	dataDir := testutil.DataDir(file)
	if dataDir == "" {
		fmt.Fprintln(os.Stderr, "未找到物料数据目录，fate 包测试无法运行")
		os.Exit(1)
	}
	if err := classics.LoadFromJSON(dataDir); err != nil {
		// 非关键：部分测试不依赖诗词数据
	}
	// 命名质量门禁字表已外置为 data/naming_quality.json，
	// 候选池硬剔除与多数评分测试依赖该表，加载失败视为致命。
	if err := LoadNamingQualityFromJSON(dataDir); err != nil {
		fmt.Fprintf(os.Stderr, "加载命名质量门禁字表失败: %v\n", err)
		os.Exit(1)
	}
	// 禁忌双字组合表（962 条历史清洗成果，data/forbidden_combos.json）
	// IsBadCombo 测试依赖该表；缺失时 LoadForbiddenCombosFromJSON 仅警告不阻断，
	// 但相关测试会基于 ForbiddenComboCount()==0 自动跳过。
	if err := LoadForbiddenCombosFromJSON(dataDir); err != nil {
		fmt.Fprintf(os.Stderr, "加载禁忌双字组合表失败: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}
