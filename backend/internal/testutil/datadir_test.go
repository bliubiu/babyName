package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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

// TestDataDir_当前目录即数据目录 情形一：callerFile 所在目录直接含 namer.json
func TestDataDir_当前目录即数据目录(t *testing.T) {
	dir := t.TempDir()
	writeMarker(t, dir)

	got := DataDir(filepath.Join(dir, "some_test.go"))
	if got != dir {
		t.Fatalf("DataDir = %q, 期望 %q", got, dir)
	}
}

// TestDataDir_命中data子目录 情形二：仓库根布局，namer.json 位于 data/ 子目录
func TestDataDir_命中data子目录(t *testing.T) {
	root := t.TempDir()
	writeMarker(t, filepath.Join(root, "data"))

	got := DataDir(filepath.Join(root, "backend", "internal", "domain", "hanzi", "x_test.go"))
	if want := filepath.Join(root, "data"); got != want {
		t.Fatalf("DataDir = %q, 期望 %q", got, want)
	}
}

// TestDataDir_向上多级探测 不依赖固定层级，中间目录无标记也能逐级上溯
func TestDataDir_向上多级探测(t *testing.T) {
	root := t.TempDir()
	writeMarker(t, root)

	deep := filepath.Join(root, "a", "b", "c", "d", "e")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}
	got := DataDir(filepath.Join(deep, "deep_test.go"))
	if got != root {
		t.Fatalf("DataDir = %q, 期望 %q", got, root)
	}
}

// TestDataDir_优先就近原则 存在多级数据目录时返回最近的一级
func TestDataDir_优先就近原则(t *testing.T) {
	root := t.TempDir()
	writeMarker(t, root)
	near := filepath.Join(root, "backend", "data")
	writeMarker(t, near)

	got := DataDir(filepath.Join(near, "pkg", "x_test.go"))
	if got != near {
		t.Fatalf("DataDir = %q, 期望就近的 %q", got, near)
	}
}

// TestDataDir_未找到返回空串 找不到 namer.json 时返回空串而非错误路径
func TestDataDir_未找到返回空串(t *testing.T) {
	// 临时目录向上直至文件系统根均无 namer.json，可稳定覆盖未命中分支
	dir := t.TempDir()
	if got := DataDir(filepath.Join(dir, "x_test.go")); got != "" {
		t.Fatalf("DataDir = %q, 期望空串", got)
	}
}

// TestDataDir_无标记的data子目录不误判 data/ 存在但其中没有 namer.json，不应视为数据目录
func TestDataDir_无标记的data子目录不误判(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "data"), 0o755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}
	if got := DataDir(filepath.Join(root, "data", "sub", "x_test.go")); got != "" {
		t.Fatalf("DataDir = %q, 期望空串（data/ 下无 namer.json）", got)
	}
}

// TestMustDataDir_找到时返回路径 成功路径与 DataDir 结果一致
func TestMustDataDir_找到时返回路径(t *testing.T) {
	root := t.TempDir()
	writeMarker(t, root)

	got := MustDataDir(t, filepath.Join(root, "x_test.go"))
	if got != root {
		t.Fatalf("MustDataDir = %q, 期望 %q", got, root)
	}
}

// TestMustDataDir_找不到时用例失败 子进程验证 Fatal 路径：中文提示 + 非零退出码
func TestMustDataDir_找不到时用例失败(t *testing.T) {
	const childEnv = "TESTUTIL_MUSTDATADIR_FATAL_CHILD"
	if os.Getenv(childEnv) == "1" {
		// 子进程：用真实 t 触发 MustDataDir 的 Fatalf
		MustDataDir(t, filepath.Join(t.TempDir(), "x_test.go"))
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestMustDataDir_找不到时用例失败", "-test.v")
	cmd.Env = append(os.Environ(), childEnv+"=1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("子进程应以非零退出码结束，实际成功；输出:\n%s", out)
	}
	text := string(out)
	if !strings.Contains(text, "未找到物料数据目录") {
		t.Fatalf("子进程输出缺少中文提示，实际:\n%s", text)
	}
}

// TestMustDataDir_真实仓库布局可用 集成：从本文件向上探测能命中仓库根 data/
func TestMustDataDir_真实仓库布局可用(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	dir := MustDataDir(t, file)
	if _, err := os.Stat(filepath.Join(dir, "namer.json")); err != nil {
		t.Fatalf("MustDataDir 返回的 %q 下无 namer.json: %v", dir, err)
	}
}
