// Package testutil 提供测试共用工具
package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// DataDir 基于调用方源文件位置向上查找物料数据目录
//
// 以 namer.json 作为「此处即数据目录」的标记逐级向上探测，因此不依赖固定的
// 目录层级数：`backend/data` 与仓库根 `data/` 两种布局、以及后续任何一次目录
// 重组都能自动适配。
//
// 此前各测试包各自硬算 `../../../data`，目录一移动即全线静默失败（字库加载
// 失败在多数测试里只告警不阻断，表现为「数据为空」而非「测试失败」），
// 排查成本极高。
//
// callerFile 取自调用方的 runtime.Caller：
//
//	_, file, _, _ := runtime.Caller(0)
//	dataDir := testutil.DataDir(file)
//
// 找不到时返回空串，由调用方决定跳过还是失败。
func DataDir(callerFile string) string {
	dir := filepath.Dir(callerFile)
	for i := 0; i < 12; i++ {
		// 情形一：当前目录本身就是数据目录
		if _, err := os.Stat(filepath.Join(dir, "namer.json")); err == nil {
			return dir
		}
		// 情形二：当前目录下的 data/ 子目录是数据目录（仓库根布局）
		candidate := filepath.Join(dir, "data")
		if _, err := os.Stat(filepath.Join(candidate, "namer.json")); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// MustDataDir 同 DataDir，但找不到时以 t.Fatal 终止用例
//
// 用于数据是用例前提的场景：此时「找不到数据」是环境缺陷，应立即失败，
// 而不是带着空字库跑出一片无意义的绿。
func MustDataDir(t testing.TB, callerFile string) string {
	t.Helper()
	dir := DataDir(callerFile)
	if dir == "" {
		t.Fatalf("未找到物料数据目录（从 %s 向上查找 namer.json 失败）", callerFile)
	}
	return dir
}
