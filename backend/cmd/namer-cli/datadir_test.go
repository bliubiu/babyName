package main

import (
	"os"
	"path/filepath"
	"testing"
)

// writeDataMarker 在 dir 下写入 namer.json 标记文件，使其成为合法数据目录
func writeDataMarker(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "namer.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("写入 namer.json 失败: %v", err)
	}
}

// setCliSourceFile 替换探测起点并返回还原函数
//
// 用于模拟「仅解压可执行文件、无仓库结构」的场景：起点落在临时目录时
// 探测必无果，从而走到 ./data 回落分支。
func setCliSourceFile(file string) func() {
	old := cliSourceFile
	cliSourceFile = func() string { return file }
	return func() { cliSourceFile = old }
}

// TestResolveDataDir 覆盖 --data 显式传入、探测命中、探测无果回落三种分支
func TestResolveDataDir(t *testing.T) {
	// 无仓库结构的探测起点：临时目录及其上层均无 namer.json
	noRepo := t.TempDir()
	// 命中场景：探测起点位于一个含 data/namer.json 的仓库内
	repo := t.TempDir()
	writeDataMarker(t, filepath.Join(repo, "data"))
	repoCaller := filepath.Join(repo, "backend", "cmd", "namer-cli", "x_test.go")

	tests := []struct {
		name    string
		flagVal string
		probe   string
		want    string
	}{
		{
			name:    "--data 显式传入时以其为准",
			flagVal: "/custom/data",
			probe:   noRepo,
			want:    "/custom/data",
		},
		{
			name:    "--data 前后空白应被裁剪",
			flagVal: "  /custom/data  ",
			probe:   noRepo,
			want:    "/custom/data",
		},
		{
			name:    "--data 为纯空白时视为未传入，探测命中仓库根 data/",
			flagVal: "   \t ",
			probe:   repoCaller,
			want:    filepath.Join(repo, "data"),
		},
		{
			name:    "未传入 --data 且探测命中仓库根 data/",
			flagVal: "",
			probe:   repoCaller,
			want:    filepath.Join(repo, "data"),
		},
		{
			name:    "未传入 --data 且探测无果时回落 ./data",
			flagVal: "",
			probe:   noRepo,
			want:    "./data",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			restore := setCliSourceFile(tc.probe)
			defer restore()

			if got := resolveDataDir(tc.flagVal); got != tc.want {
				t.Errorf("resolveDataDir(%q) = %q，期望 %q", tc.flagVal, got, tc.want)
			}
		})
	}
}

// TestResolveDataDir_仓库内探测到仓库根 物料源文件位于仓库根 data/ 而非
// backend/data。从 backend/ 运行且不传 --data 时应命中仓库根 data/。
//
// 本用例刻意不替换 cliSourceFile，以真实仓库布局作为探测起点。
func TestResolveDataDir_仓库内探测到仓库根(t *testing.T) {
	got := resolveDataDir("")
	if _, err := os.Stat(filepath.Join(got, "namer.json")); err != nil {
		t.Fatalf("探测得到 %q，但其下无 namer.json: %v", got, err)
	}
}
