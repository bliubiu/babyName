//go:build !windows

package main

// 非 Windows 平台（CI 的 linux server build）保留 stderr 输出即可，不弹框。
func showStartupError(string) {}
