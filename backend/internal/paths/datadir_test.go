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

// TestFindDataDirFromExe_exe同级data含标记 打包布局：exe 同级 data/ 内有 namer.json 即命中
func TestFindDataDirFromExe_exe同级data含标记(t *testing.T) {
	root := t.TempDir()
	writeMarker(t, filepath.Join(root, "data"))

	got := FindDataDirFromExe(filepath.Join(root, "namer-desktop.exe"))
	if want := filepath.Join(root, "data"); got != want {
		t.Fatalf("FindDataDirFromExe = %q, 期望 %q", got, want)
	}
}

// TestFindDataDirFromExe_无data目录 exe 同级无 data/，返回空串交由上层回落
func TestFindDataDirFromExe_无data目录(t *testing.T) {
	root := t.TempDir()

	if got := FindDataDirFromExe(filepath.Join(root, "namer-desktop.exe")); got != "" {
		t.Fatalf("FindDataDirFromExe = %q, 期望空串", got)
	}
}

// TestFindDataDirFromExe_无标记不误判 data/ 存在但缺 namer.json 标记，不视为物料目录
func TestFindDataDirFromExe_无标记不误判(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "data"), 0o755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}

	if got := FindDataDirFromExe(filepath.Join(root, "namer-desktop.exe")); got != "" {
		t.Fatalf("FindDataDirFromExe = %q, 期望空串（data/ 下无 namer.json）", got)
	}
}

// TestFindDataDirFromExe_空exe路径 空串输入直接返回空串，不 panic
func TestFindDataDirFromExe_空exe路径(t *testing.T) {
	if got := FindDataDirFromExe(""); got != "" {
		t.Fatalf("FindDataDirFromExe(\"\") = %q, 期望空串", got)
	}
}

// --- 打包态上溯探测（wails3 build -trimpath 后源码路径探测失效的修复）---

// TestFindDataDirFromExeWithParents_上级data命中 exe 在 backend/bin，仓库根 data/ 应被回溯命中
func TestFindDataDirFromExeWithParents_上级data命中(t *testing.T) {
	root := t.TempDir()
	writeMarker(t, filepath.Join(root, "data"))

	exe := filepath.Join(root, "backend", "bin", "namer-desktop.exe")
	if got := FindDataDirFromExeWithParents(exe); got != filepath.Join(root, "data") {
		t.Fatalf("FindDataDirFromExeWithParents = %q, 期望 %q", got, filepath.Join(root, "data"))
	}
}

// TestFindDataDirFromExeWithParents_同级优先 安装版自带 data/ 时优先同级，不上溯
func TestFindDataDirFromExeWithParents_同级优先(t *testing.T) {
	root := t.TempDir()
	writeMarker(t, filepath.Join(root, "data"))
	bin := filepath.Join(root, "backend", "bin")
	writeMarker(t, filepath.Join(bin, "data"))

	exe := filepath.Join(bin, "namer-desktop.exe")
	if got, want := FindDataDirFromExeWithParents(exe), filepath.Join(bin, "data"); got != want {
		t.Fatalf("应优先命中 exe 同级 data，实际 %q，期望 %q", got, want)
	}
}

// TestFindDataDirFromExeWithParents_当前目录即数据目录 exe 直接放在 data/ 内
func TestFindDataDirFromExeWithParents_当前目录即数据目录(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	writeMarker(t, dataDir)

	got := FindDataDirFromExeWithParents(filepath.Join(dataDir, "namer-desktop.exe"))
	if got != dataDir {
		t.Fatalf("FindDataDirFromExeWithParents = %q, 期望 %q", got, dataDir)
	}
}

// TestFindDataDirFromExeWithParents_空路径 空串输入直接返回空串，不 panic
func TestFindDataDirFromExeWithParents_空路径(t *testing.T) {
	if got := FindDataDirFromExeWithParents(""); got != "" {
		t.Fatalf("FindDataDirFromExeWithParents(\"\") = %q, 期望空串", got)
	}
}
