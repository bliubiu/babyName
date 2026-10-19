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

// TestDataDir_委托探测逻辑可用 DataDir 包装 paths.FindDataDir，命中场景应透传结果
//
// 探测的各分支（命中当前目录、命中 data/ 子目录、向上多级、未命中）由
// internal/paths 的单测覆盖，此处只验证包装层确实走通委托。
func TestDataDir_委托探测逻辑可用(t *testing.T) {
	root := t.TempDir()
	writeMarker(t, filepath.Join(root, "data"))

	got := DataDir(filepath.Join(root, "backend", "internal", "hanzi", "x_test.go"))
	if want := filepath.Join(root, "data"); got != want {
		t.Fatalf("DataDir = %q, 期望 %q", got, want)
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
