//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

// showStartupError 在 GUI 子系统下弹出错误框。
//
// wails3 build 使用 -ldflags "-H windowsgui"，进程没有控制台，
// fmt.Fprintln(os.Stderr, ...) 用户看不到；用 user32!MessageBoxW 保证
// 目录解析 / 装配等早期失败与未捕获 panic 都可感知。
//
// msg 由 startup_failure.go 的 formatFriendlyMessage 生成：面向最终用户的
// 说明 + 可能原因 + 可操作步骤 + 错误代码，不含 DATA_DIR / Options.DataDir
// 等内部术语；完整技术诊断只写入 startup-error.log。
// 标志：MB_ICONERROR | MB_SETFOREGROUND | MB_TOPMOST（保证弹框可见且置顶）。
func showStartupError(msg string) {
	text, err := syscall.UTF16PtrFromString(msg)
	if err != nil {
		return
	}
	title, err := syscall.UTF16PtrFromString("起名 NameMaster 启动失败")
	if err != nil {
		return
	}
	user32 := syscall.NewLazyDLL("user32.dll")
	messageBox := user32.NewProc("MessageBoxW")
	_, _, _ = messageBox.Call(
		0,
		uintptr(unsafe.Pointer(text)),
		uintptr(unsafe.Pointer(title)),
		0x10|0x10000|0x40000,
	)
}
