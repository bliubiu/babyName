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

// isDataDir 判定目录是否为物料数据目录
//
// 两类标记：
//   - namer.json：JSON/CSV 源物料目录（离线导入工具、开发期布局）；
//   - namer.db：单库路线的种子库（CI、安装包、零 JSON 运行期）。
//
// 两者并存时语义不变，仍是同一个 data/ 目录；标记扩展到 namer.db 是为了
// 让「只随装一个二进制库」的分发形态不需要再塞一个占位 JSON。
func isDataDir(dir string) bool {
	for _, marker := range [2]string{"namer.json", "namer.db"} {
		info, err := os.Stat(filepath.Join(dir, marker))
		if err != nil || info.IsDir() {
			continue
		}
		// 开发期偶有 0 字节 / 4KB 的空 namer.db 残留在非 data/ 目录（例如
		// backend/namer.db）。若只判「文件存在」，exe 上溯会把这种目录误当
		// 物料目录。种子库实测约 22MB，这里要求 >1MB 才算有效标记。
		if marker == "namer.db" && info.Size() <= 1<<20 {
			continue
		}
		return true
	}
	return false
}

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
	// callerFile 必须是绝对路径。-trimpath 构建产物中 runtime.Caller(0) 返回
	// 模块相对路径，下面的逐级 stat 会以进程 CWD 为锚点解析：CWD 恰在仓库
	// 结构内时会假命中 DevRepo（数据串台 / 播种按相对路径找不到源物料），
	// 且 DevRepo 优先级高于 exe 同级 data/ 探测，打包语义被整体带偏。
	// dev 源码中 runtime.Caller 恒为绝对路径，此校验不影响开发期探测。
	if !filepath.IsAbs(callerFile) {
		return ""
	}
	dir := filepath.Dir(callerFile)
	for i := 0; i < maxDataDirDepth; i++ {
		// 情形一：当前目录本身就是数据目录。
		// 只用 namer.json 判定：仓库根另有开发期运行库 namer.db，若把 namer.db
		// 也算进「当前目录」会提前命中仓库根而非真正的 data/ 物料目录。
		if _, err := os.Stat(filepath.Join(dir, "namer.json")); err == nil {
			return dir
		}
		// 情形二：当前目录下的 data/ 子目录是数据目录（仓库根 / 安装目录布局）。
		// 单库路线（ADR-010）下分发的 data/ 里只有 namer.db 没有 JSON，
		// 故判据扩展到「namer.json 或 namer.db 任一存在」。
		candidate := filepath.Join(dir, "data")
		if isDataDir(candidate) {
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

// FindDataDirFromExe 基于可执行文件位置探测「exe 同级 data/」打包布局
//
// 打包分发后不存在仓库源码结构（FindDataDir 无从上溯），物料随安装器
// 落在 exe 同级 data/ 内。以 namer.json 为标记判定，避免把恰好同名的
// 空目录误当物料目录。exePath 为空或探测不命中返回空串。
func FindDataDirFromExe(exePath string) string {
	if exePath == "" {
		return ""
	}
	candidate := filepath.Join(filepath.Dir(exePath), "data")
	if isDataDir(candidate) {
		return candidate
	}
	return ""
}

// FindDataDirFromExeWithParents 以 exe 实际位置为锚点，逐级向上探测物料目录。
//
// 背景（docs/48 §9）：wails3 build 带 -trimpath，runtime.Caller(0) 返回的
// 编译期源文件绝对路径被抹成模块相对路径，dev 侧基于源码位置的 FindDataDir
// 在打包产物中必然失效；若 exe 同级又恰好没有随装 data/（例如直接运行
// backend/bin/namer-desktop.exe），所有来源落空，应用在日志初始化之前就
// 静默退出。以 exe 真实路径上溯可同时覆盖两种布局：
//   - 安装版：第一级即命中安装目录自带的 data/（优先级最高）
//   - 源码内构建：backend/bin 上溯两级命中仓库根 data/
//
// 与 FindDataDirFromExe 的区别仅在于「是否上溯」；原函数保持只查同级，
// 以维持既有测试契约。命中判据统一为 namer.json 标记，避免误判空目录。
func FindDataDirFromExeWithParents(exePath string) string {
	if exePath == "" {
		return ""
	}
	dir := filepath.Dir(exePath)
	for i := 0; i < maxDataDirDepth; i++ {
		// 情形一：当前目录下的 data/ 是物料目录（仓库根 / 安装目录布局）
		candidate := filepath.Join(dir, "data")
		if isDataDir(candidate) {
			return candidate
		}
		// 情形二：当前目录本身就是物料目录（exe 直接放在 data/ 内）
		if isDataDir(dir) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}
