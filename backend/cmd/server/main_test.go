package main

// main_test.go — 启动期目录解析（docs/29 B12）
//
// B12 的补正点：docker-entrypoint.sh 按 DATA_DIR 播种字库，而进程原先固定读
// `cwd/data`。两者不一致时，字库播到 A 目录、进程读 B 目录 —— 服务正常启动、
// healthz 通过、生成接口全部返回空结果，且没有任何错误日志指向真因。

import (
	"path/filepath"
	"testing"
)

func TestCwdDataDir(t *testing.T) {
	const cwd = "/app"

	tests := []struct {
		name     string
		dataDir  string
		want     string
		wantNote string
	}{
		{
			name:    "未设置 DATA_DIR 时回落到工作目录下的 data/",
			dataDir: "",
			want:    filepath.Join(cwd, "data"),
		},
		{
			name:    "设置了 DATA_DIR 时以环境变量为准",
			dataDir: "/app/custom-data",
			want:    "/app/custom-data",
		},
		{
			name:    "DATA_DIR 为纯空白时视为未设置",
			dataDir: "   \t ",
			want:    filepath.Join(cwd, "data"),
		},
		{
			name:    "DATA_DIR 前后空白应被裁剪",
			dataDir: "  /app/data  ",
			want:    "/app/data",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DATA_DIR", tc.dataDir)
			if got := cwdDataDir(cwd); got != tc.want {
				t.Errorf("cwdDataDir(%q) = %q，期望 %q（DATA_DIR=%q）", cwd, got, tc.want, tc.dataDir)
			}
		})
	}
}

// TestCwdDataDir_MatchesEntrypointDefault 与入口脚本的默认值保持一致
//
// 入口脚本：DATA_DIR=${DATA_DIR:-/app/data}
// 本函数：DATA_DIR 为空 → cwd/data（容器内 cwd=/app，即 /app/data）
// 两者必须解析到同一目录，否则播种与读取分叉。
//
// 注意：断言用 filepath.Join 构造期望值而非硬编码 "/" —— 测试在 Windows 上
// 运行时 filepath 会产出反斜杠，硬编码会把平台差异误报成缺陷。
// 容器内为 Linux，filepath.Join("/app","data") 即 /app/data。
func TestCwdDataDir_MatchesEntrypointDefault(t *testing.T) {
	t.Setenv("DATA_DIR", "")

	// 容器内工作目录
	got := cwdDataDir("/app")
	if want := filepath.Join("/app", "data"); got != want {
		t.Errorf("容器内默认数据目录 = %q，期望 %q（须与入口脚本一致）", got, want)
	}

	// 本地开发：cwd 为 backend/，落在 backend/data
	localCwd := filepath.Join("repo", "backend")
	if got, want := cwdDataDir(localCwd), filepath.Join(localCwd, "data"); got != want {
		t.Errorf("本地默认数据目录 = %q，期望 %q", got, want)
	}
}