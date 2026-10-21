package paths

import (
	"os"
	"path/filepath"
)

// maxModuleDirDepth 向上探测 go.mod 的最大层数（与 FindDataDir 同源余量）
const maxModuleDirDepth = 12

// FindModuleDir 基于调用方源文件位置向上查找 Go 模块根（go.mod 所在目录）
//
// 日志/存储等运行时产物需要与 cwd 无关的「项目根」锚点：后端进程可能从仓库根
// 或 backend/ 两种 cwd 启动，相对路径会让同一配置在两处产出两份文件（双库
// 分叉的同根问题，docs/36）。以 go.mod（backend/go.mod）为锚逐级向上探测，
// 本机两种启动目录都会命中 backend/；容器内（/app）可执行文件由模块根构建，
// 同样命中 /app。
//
// callerFile 通常取自调用方的 runtime.Caller：
//
//	_, file, _, _ := runtime.Caller(0)
//	dir := paths.FindModuleDir(file)
//
// 找不到时返回空串，由调用方决定回落（通常回落 cwd）。
func FindModuleDir(callerFile string) string {
	dir := filepath.Dir(callerFile)
	for i := 0; i < maxModuleDirDepth; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
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
