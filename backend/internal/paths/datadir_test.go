package paths

import (
	"os"
	"path/filepath"
	"testing"
)

// writeMarker 在 dir 下写入 namer.json 标记文件，使其成为合法数据目录
func writeMarker(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "namer.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("写入 namer.json 失败: %v", err)
	}
}

// TestFindDataDir_当前目录即数据目录 情形一：callerFile 所在目录直接含 namer.json
func TestFindDataDir_当前目录即数据目录(t *testing.T) {
	dir := t.TempDir()
	writeMarker(t, dir)

	got := FindDataDir(filepath.Join(dir, "some_test.go"))
	if got != dir {
		t.Fatalf("FindDataDir = %q, 期望 %q", got, dir)
	}
}

// TestFindDataDir_命中data子目录 情形二：仓库根布局，namer.json 位于 data/ 子目录
func TestFindDataDir_命中data子目录(t *testing.T) {
	root := t.TempDir()
	writeMarker(t, filepath.Join(root, "data"))

	got := FindDataDir(filepath.Join(root, "backend", "internal", "domain", "hanzi", "x_test.go"))
	if want := filepath.Join(root, "data"); got != want {
		t.Fatalf("FindDataDir = %q, 期望 %q", got, want)
	}
}

// TestFindDataDir_向上多级探测 不依赖固定层级，中间目录无标记也能逐级上溯
func TestFindDataDir_向上多级探测(t *testing.T) {
	root := t.TempDir()
	writeMarker(t, root)

	deep := filepath.Join(root, "a", "b", "c", "d", "e")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}
	got := FindDataDir(filepath.Join(deep, "deep_test.go"))
	if got != root {
		t.Fatalf("FindDataDir = %q, 期望 %q", got, root)
	}
}

// TestFindDataDir_优先就近原则 存在多级数据目录时返回最近的一级
func TestFindDataDir_优先就近原则(t *testing.T) {
	root := t.TempDir()
	writeMarker(t, root)
	near := filepath.Join(root, "backend", "data")
	writeMarker(t, near)

	got := FindDataDir(filepath.Join(near, "pkg", "x_test.go"))
	if got != near {
		t.Fatalf("FindDataDir = %q, 期望就近的 %q", got, near)
	}
}

// TestFindDataDir_未找到返回空串 找不到 namer.json 时返回空串而非错误路径
func TestFindDataDir_未找到返回空串(t *testing.T) {
	// 临时目录向上直至文件系统根均无 namer.json，可稳定覆盖未命中分支
	dir := t.TempDir()
	if got := FindDataDir(filepath.Join(dir, "x_test.go")); got != "" {
		t.Fatalf("FindDataDir = %q, 期望空串", got)
	}
}

// TestFindDataDir_无标记的data子目录不误判 data/ 存在但其中没有 namer.json，不应视为数据目录
func TestFindDataDir_无标记的data子目录不误判(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "data"), 0o755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}
	if got := FindDataDir(filepath.Join(root, "data", "sub", "x_test.go")); got != "" {
		t.Fatalf("FindDataDir = %q, 期望空串（data/ 下无 namer.json）", got)
	}
}
