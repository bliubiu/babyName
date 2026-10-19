// Package paths 提供与工作目录无关的路径解析
//
// 供生产入口（cmd/server）与测试共用同一套「定位物料数据目录」的规则。
// 两者各写一套曾是缺陷源：测试侧硬算 `../../../data`、进程侧固定 `cwd/data`，
// 目录重组时一侧崩溃一侧静默，排查成本极高。
package paths

import (
	"os"
	"path/filepath"
)

// maxDataDirDepth 向上探测的最大层数
//
// 仓库内最深的调用方（如 backend/internal/infrastructure/database/sqlite）
// 距仓库根约 5 层，12 留足余量；同时避免在无 namer.json 的文件系统上
// 一路爬到根目录才发现无果。
const maxDataDirDepth = 12

// FindDataDir 基于调用方源文件位置向上查找物料数据目录
//
// 以 namer.json 作为「此处即数据目录」的标记逐级向上探测，因此不依赖固定的
// 目录层级数：`backend/data` 与仓库根 `data/` 两种布局、以及后续任何一次目录
// 重组都能自动适配。存在多级数据目录时返回最近的一级。
//
// callerFile 通常取自调用方的 runtime.Caller：
//
//	_, file, _, _ := runtime.Caller(0)
//	dir := paths.FindDataDir(file)
//
// 找不到时返回空串，由调用方决定回落、跳过还是失败。
func FindDataDir(callerFile string) string {
	dir := filepath.Dir(callerFile)
	for i := 0; i < maxDataDirDepth; i++ {
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
