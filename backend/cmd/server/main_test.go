package main

// main_test.go — 启动期目录解析（docs/29 B12）与物料缺失 fail-fast（docs/31 ADR-008）
//
// B12 的补正点：docker-entrypoint.sh 按 DATA_DIR 播种字库，而进程原先固定读
// `cwd/data`。两者不一致时，字库播到 A 目录、进程读 B 目录 —— 服务正常启动、
// healthz 通过、生成接口全部返回空结果，且没有任何错误日志指向真因。
//
// ADR-008 的补正点：物料库缺料时原先只 Warn 后继续启动，字库为空同样导致
// healthz 通过而全部起名请求 400。现改为拒绝启动并打印导入指引。

import (
	"path/filepath"
	"strings"
	"testing"

	"name/internal/infrastructure/data"
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

// emptyDBLoader 模拟一张尚未执行 materials import 的空物料库
type emptyDBLoader struct{}

func (emptyDBLoader) LoadMaterialsFromDB() []string { return nil }

// TestMaterialsImportHint 校验物料导入指引命令的构造
//
// 指引直接印在启动失败的 stderr 上，参数拼错等于把用户引向另一条死路。
// 用 POSIX 风格路径（容器内真实形态）构造期望值，避免 Windows 反斜杠被 %q
// 转义后把平台差异误报成缺陷——与上方 B12 测试同理。
func TestMaterialsImportHint(t *testing.T) {
	const (
		dataDir = "/app/data"
		dbPath  = "namer.db"
	)

	got := materialsImportHint(dataDir, dbPath)
	want := `go run ./cmd/materials import --data "/app/data" --db "namer.db"`
	if got != want {
		t.Errorf("materialsImportHint() = %q，期望 %q", got, want)
	}
}

// TestMaterialsImportHint_Windows路径不转义 回归守护
//
// 指引是 shell 命令而非 Go 字面量：若误用 %q，Windows 路径 D:\a\b 会被写成
// "D:\\a\\b"，用户照抄到 cmd/PowerShell 里会指向错误目录。
func TestMaterialsImportHint_Windows路径不转义(t *testing.T) {
	const (
		dataDir = `D:\19-Training\data`
		dbPath  = `C:\db\namer.db`
	)

	got := materialsImportHint(dataDir, dbPath)
	want := `go run ./cmd/materials import --data "D:\19-Training\data" --db "C:\db\namer.db"`
	if got != want {
		t.Errorf("materialsImportHint() = %q，期望 %q（Windows 路径不应被转义）", got, want)
	}
	if strings.Contains(got, `\\`) {
		t.Errorf("指引中出现转义反斜杠，复制到 shell 会失效：%s", got)
	}
}

// TestLoadCulturalDataFromDB_必需物料缺失时返回错误并附带导入指引
//
// 空物料库必须报错，且错误信息同时含「materials import」提示与具体恢复命令，
// 否则调用方无法据此 fail-fast，也无法把用户导向正确的补救动作。
// 假实现不触碰文件系统，故 dataDir 用容器内形态的路径即可。
func TestLoadCulturalDataFromDB_必需物料缺失时返回错误并附带导入指引(t *testing.T) {
	const (
		dataDir = "/app/data"
		dbPath  = "namer.db"
	)

	err := loadCulturalDataFromDB(dataDir, dbPath, emptyDBLoader{})
	if err == nil {
		t.Fatal("空物料库下 loadCulturalDataFromDB 应返回错误，实际为 nil（服务会带着空字库启动）")
	}
	for _, want := range []string{"materials import", dataDir, dbPath} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("错误信息缺少 %q，实际为：%v", want, err)
		}
	}
}

// TestLoadCulturalDataFromDB_错误保留原始原因 校验包装不吞掉底层错误
//
// 调用方与运维要能从错误里读到「缺哪几项物料」，不能只剩一句笼统的启动失败。
func TestLoadCulturalDataFromDB_错误保留原始原因(t *testing.T) {
	err := loadCulturalDataFromDB(t.TempDir(), "namer.db", emptyDBLoader{})
	if err == nil {
		t.Fatal("空物料库下应返回错误，实际为 nil")
	}
	if !strings.Contains(err.Error(), "缺少必需物料") {
		t.Errorf("错误信息未保留底层缺料原因，实际为：%v", err)
	}
}

// TestLoadCulturalData_NonDBLoaderStore 回落纯 JSON 路径时不应附带导入指引
//
// 无库场景（离线工具/测试）修复动作是补数据文件而非导入，给出 import 命令是误导。
func TestLoadCulturalData_NonDBLoaderStore(t *testing.T) {
	// nil store 不实现 data.DBLoader，走 data.Init 的纯 JSON 路径
	err := loadCulturalData(t.TempDir(), "namer.db", nil)
	if err == nil {
		t.Fatal("空数据目录下纯 JSON 路径应返回错误，实际为 nil")
	}
	if strings.Contains(err.Error(), "materials import") {
		t.Errorf("纯 JSON 路径不应附带 materials import 指引，实际为：%v", err)
	}
}

// 编译期断言：空物料库实现了 data.DBLoader
var _ data.DBLoader = emptyDBLoader{}
