package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestWriteStartupErrorLog 启动错误必须落盘，GUI 子系统无控制台时仍可排查
func TestWriteStartupErrorLog(t *testing.T) {
	dir := t.TempDir()
	if err := writeStartupErrorLog(dir, "应用启动失败：未找到传统文化数据目录"); err != nil {
		t.Fatalf("写入启动错误日志失败: %v", err)
	}

	path := filepath.Join(dir, "startup-error.log")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取启动错误日志失败: %v", err)
	}
	if !strings.Contains(string(b), "应用启动失败") {
		t.Fatalf("启动错误日志不含错误前缀: %q", string(b))
	}
	if !strings.Contains(string(b), "未找到传统文化数据目录") {
		t.Fatalf("启动错误日志不含具体原因: %q", string(b))
	}
}
