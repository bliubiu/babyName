// Package testutil 提供测试共用工具
package testutil

import (
	"testing"

	"name/internal/paths"
)

// DataDir 基于调用方源文件位置向上查找物料数据目录
//
// 规则由 paths.FindDataDir 统一实现（生产入口 cmd/server 复用同一套），此处
// 仅做测试侧包装：以 namer.json 为标记逐级向上探测，兼容 `backend/data` 与
// 仓库根 `data/` 两种布局及后续任何一次目录重组。
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
	return paths.FindDataDir(callerFile)
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
