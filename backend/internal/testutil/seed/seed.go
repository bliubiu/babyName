// Package seed 单库物料装配助手（ADR-010「A + 单库」）
//
// 为何独立成包：`data.InitWithDB` 会导入全部 domain 包；若把本助手放在
// internal/testutil 里，领域包的**内部**测试（package hanzi 等）再导入
// testutil 就形成 import cycle。拆出子包后，领域包改用 package X_test 的
// 外部 TestMain 调用本助手，包内测试仍可用轻量的 testutil.MustDataDir。
//
// 改造后：测试、CI 门禁、桌面运行期、NSIS 安装包共用同一份 data/namer.db。
// JSON/CSV 只作为 tools/build_seed 的构建期输入，不入库、不参与门禁。
package seed

import (
	"fmt"
	"testing"

	"name/internal/domain/hanzi"
	"name/internal/infrastructure/data"
	"name/internal/infrastructure/database/sqlite"
	"name/internal/testutil"
)

// Path 定位种子库 data/namer.db（找不到或明显不完整时返回空串）
func Path(callerFile string) string {
	return testutil.SeedDBPath(callerFile)
}

// MustPath 同 Path，但找不到时以 t.Fatal 终止用例
func MustPath(t testing.TB, callerFile string) string {
	t.Helper()
	path := Path(callerFile)
	if path == "" {
		t.Fatalf("未找到种子库 data/namer.db（从 %s 向上探测失败，或文件不足 1MB）；请先执行 go run ./tools/build_seed", callerFile)
	}
	return path
}

// Init 从种子库装载全部物料到包级内存索引
//
// 返回 cleanup（关闭只读连接）；os.Exit 的 TestMain 里不调用也无妨。
// dataDir 传空串：库路径下 data.InitWithDB 完全不读 JSON。
func Init(callerFile string) (func(), error) {
	dbPath := Path(callerFile)
	if dbPath == "" {
		return nil, fmt.Errorf("未找到种子库 data/namer.db（从 %s 向上探测失败）；请先执行 go run ./tools/build_seed", callerFile)
	}
	loader, err := sqlite.NewMaterialLoader(dbPath)
	if err != nil {
		return nil, err
	}
	if err := data.InitWithDB("", loader); err != nil {
		_ = loader.Close()
		return nil, fmt.Errorf("从种子库装载物料失败: %w", err)
	}
	return func() { _ = loader.Close() }, nil
}

// InitHanziOnly 只从种子库装载字库与康熙笔画，不覆盖其他领域数据
//
// 供 yijing 包金标准对拍使用：该包测试针对代码内置 HexagramList，不能让
// 库内卦象覆盖它。仍然零 JSON，且字库来源与生产一致。
func InitHanziOnly(callerFile string) (func(), error) {
	dbPath := Path(callerFile)
	if dbPath == "" {
		return nil, fmt.Errorf("未找到种子库 data/namer.db（从 %s 向上探测失败）", callerFile)
	}
	loader, err := sqlite.NewMaterialLoader(dbPath)
	if err != nil {
		return nil, err
	}
	loader.LoadCoreCharDataFromDB()
	if len(hanzi.HanziData) == 0 {
		_ = loader.Close()
		return nil, fmt.Errorf("种子库字库为空: %s", dbPath)
	}
	return func() { _ = loader.Close() }, nil
}

// MustInit 同 Init，失败即 t.Fatal
func MustInit(t testing.TB, callerFile string) func() {
	t.Helper()
	cleanup, err := Init(callerFile)
	if err != nil {
		t.Fatalf("单库物料装配失败: %v", err)
	}
	return cleanup
}
