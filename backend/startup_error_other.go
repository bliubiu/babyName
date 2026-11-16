//go:build !windows

package main

// 非 Windows 平台（CI 的 linux server build）保留 stderr 输出即可，不弹框。
// 友好话术与诊断文本仍由 startup_failure.go 生成并经 fatalStartup 落盘，
// 保证跨平台行为一致、CI 可直接验证文案。
func showStartupError(string) {}
