package paths

import (
	"os"
	"path/filepath"
	"testing"
)

// TestFindModuleDir_模块根命中 go.mod 位于 callerFile 所在目录时直接返回该目录
func TestFindModuleDir_模块根命中(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatalf("写入 go.mod 失败: %v", err)
	}

	got := FindModuleDir(filepath.Join(dir, "main.go"))
	if got != dir {
		t.Fatalf("FindModuleDir = %q, 期望 %q", got, dir)
	}
}

// TestFindModuleDir_向上多级探测 模块根在祖级目录时逐级上溯命中
func TestFindModuleDir_向上多级探测(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatalf("写入 go.mod 失败: %v", err)
	}

	deep := filepath.Join(root, "cmd", "server")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}
	got := FindModuleDir(filepath.Join(deep, "main.go"))
	if got != root {
		t.Fatalf("FindModuleDir = %q, 期望 %q", got, root)
	}
}

// TestFindModuleDir_未找到返回空 向上直至文件系统根均无 go.mod 时返回空串
func TestFindModuleDir_未找到返回空(t *testing.T) {
	dir := t.TempDir()
	if got := FindModuleDir(filepath.Join(dir, "x.go")); got != "" {
		t.Fatalf("FindModuleDir = %q, 期望空串", got)
	}
}
