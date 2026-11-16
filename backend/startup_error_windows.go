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
// 目录解析 / 装配等早期失败可感知。标志：MB_ICONERROR | MB_SETFOREGROUND。
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
		0x10|0x10000,
	)
}
