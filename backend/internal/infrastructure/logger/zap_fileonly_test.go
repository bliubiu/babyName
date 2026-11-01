package logger

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestInit_FileOnly不写标准输出 CLI/JSON 工具场景：日志只落盘，
// 不得写 stdout 污染结果输出（此前 namer-cli 的 JSON 输出被 INFO 日志行破坏）。
func TestInit_FileOnly不写标准输出(t *testing.T) {
	dir := t.TempDir()

	// 劫持 stdout 捕获写入
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("创建管道失败: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	if err := Init(&Config{Level: "info", Format: "console", OutputPath: dir, FileOnly: true}); err != nil {
		t.Fatalf("Init 失败: %v", err)
	}
	Info("FileOnly测试日志", String("k", "v"))
	Sync()
	Close() // 释放日志文件句柄（Windows 下 TempDir 清理需要）

	_ = w.Close()
	out, _ := io.ReadAll(r)
	if len(bytes.TrimSpace(out)) > 0 {
		t.Errorf("FileOnly 模式不应写 stdout，实得 %q", string(out))
	}

	// 日志文件应有内容
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) == 0 {
		t.Fatalf("日志目录 %s 应有轮转日志文件: %v", dir, err)
	}
	data, err := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	if err != nil {
		t.Fatalf("读取日志文件失败: %v", err)
	}
	if !strings.Contains(string(data), "FileOnly测试日志") {
		t.Errorf("日志文件应包含写入内容，实得 %q", string(data))
	}
}
