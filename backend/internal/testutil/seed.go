package testutil

// 种子库路径探测（ADR-010「A + 单库」）
//
// 本文件只做路径探测，不导入 data / sqlite：领域包的内部测试（package hanzi
// 等）也依赖 testutil，一旦这里导入 data 就形成 import cycle。
// 真正的装载助手在子包 internal/testutil/seed，由 package X_test 外部
// TestMain 调用。

import (
	"os"
	"path/filepath"
	"testing"
)

// SeedDBPath 定位种子库 data/namer.db
//
// 依赖 paths.FindDataDir 的目录探测（namer.json 或 namer.db 标记），因此
// 源码布局、CI 干净检出、安装目录三种形态都能命中。找不到或文件明显不是
// 完整种子库（≤1MB）时返回空串，由调用方决定报错还是跳过。
func SeedDBPath(callerFile string) string {
	dir := DataDir(callerFile)
	if dir == "" {
		return ""
	}
	path := filepath.Join(dir, "namer.db")
	info, err := os.Stat(path)
	if err != nil || info.IsDir() || info.Size() <= 1<<20 {
		return ""
	}
	return path
}

// MustSeedDBPath 同 SeedDBPath，但找不到时以 t.Fatal 终止用例
func MustSeedDBPath(t testing.TB, callerFile string) string {
	t.Helper()
	path := SeedDBPath(callerFile)
	if path == "" {
		t.Fatalf("未找到种子库 data/namer.db（从 %s 向上探测失败，或文件不足 1MB）；请先执行 go run ./tools/build_seed", callerFile)
	}
	return path
}
