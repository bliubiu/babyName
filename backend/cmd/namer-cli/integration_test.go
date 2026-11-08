package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"name/internal/testutil"
)

// 本文件将此前的人工冒烟验证固化为自动化端到端回归：
// 编译 namer-cli 真实二进制并以 json 格式执行一次完整生成，
// 断言输出为合法 JSON 且每个候选名的拼音已回填。

// cliJSONResponse RenderJSON 的顶层结构（与 API data 字段一致）
type cliJSONResponse struct {
	Bazi  json.RawMessage `json:"bazi"`
	Names []struct {
		FullName string `json:"full_name"`
		Pinyin   string `json:"pinyin"`
	} `json:"names"`
}

// locateBackendRoot 基于本文件位置定位 backend 根目录
func locateBackendRoot(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..")
}

// TestCLIBinaryJSONOutputPinyinFilled 编译真实二进制执行生成并断言拼音回填
func TestCLIBinaryJSONOutputPinyinFilled(t *testing.T) {
	if testing.Short() {
		t.Skip("短模式跳过二进制集成测试")
	}

	backendRoot := locateBackendRoot(t)

	// 1. 编译当前包到临时可执行文件（测试默认 CWD 即本包目录）
	exeName := "namer-cli-test.exe"
	exePath := filepath.Join(t.TempDir(), exeName)
	build := exec.Command("go", "build", "-o", exePath, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("编译 CLI 失败: %v\n%s", err, out)
	}
	defer func() { _ = os.Remove(exePath) }()

	// 2. 执行一次完整生成（JSON 输出）
	//
	// 显式传 --data 而非依赖 CLI 默认的 ./data：默认值按 CWD 解析，
	// 而数据目录已不在 backend/ 下（曾随目录重组迁移），靠 CWD 命中属脆弱耦合。
	_, thisFile, _, _ := runtime.Caller(0)
	dataDir := testutil.MustDataDir(t, thisFile)
	runArgs := []string{
		"-surname", "王", "-gender", "male",
		"-birth_year", "2024", "-birth_month", "1", "-birth_day", "15", "-birth_hour", "12",
		"-format", "json", "-count", "5",
		"-data", dataDir,
	}
	cmd := exec.Command(exePath, runArgs...)
	cmd.Dir = backendRoot // 数据目录默认 ./data，必须在 backend 下运行
	var stdout, stderr = &cliOutputBuffer{}, &cliOutputBuffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	done := make(chan error, 1)
	if err := cmd.Start(); err != nil {
		t.Fatalf("启动 CLI 失败: %v", err)
	}
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("CLI 执行失败: %v\nstderr: %s", err, stderr.String())
		}
	case <-time.After(3 * time.Minute):
		_ = cmd.Process.Kill()
		t.Fatal("CLI 执行超时（3 分钟）")
	}

	// 3. 解析 JSON 输出并断言拼音回填
	var resp cliJSONResponse
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		t.Fatalf("输出不是合法 JSON: %v\n前 500 字节: %.500s", err, stdout.String())
	}
	if len(resp.Bazi) == 0 {
		t.Error("JSON 输出应包含八字信息 bazi")
	}
	if len(resp.Names) == 0 {
		t.Fatal("应至少生成一个候选名")
	}
	for i, n := range resp.Names {
		if n.FullName == "" {
			t.Errorf("候选 #%d 缺少 full_name", i)
		}
		if n.Pinyin == "" {
			t.Errorf("候选 %q 的拼音为空，应自 hanzi 字库回填", n.FullName)
		}
	}
}

// TestCLIBinaryConfigFileDoesNotPolluteJSONStream 使用 -config + -format json 时，
// 「已加载配置文件」提示必须走 stderr，stdout 只能是纯 JSON（可被 jq 等管道消费）。
func TestCLIBinaryConfigFileDoesNotPolluteJSONStream(t *testing.T) {
	if testing.Short() {
		t.Skip("短模式跳过二进制集成测试")
	}

	backendRoot := locateBackendRoot(t)

	exePath := filepath.Join(t.TempDir(), "namer-cli-test-config.exe")
	build := exec.Command("go", "build", "-o", exePath, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("编译 CLI 失败: %v\n%s", err, out)
	}
	defer func() { _ = os.Remove(exePath) }()

	// example.toml 与测试同目录（cmd/namer-cli/）
	_, thisFile, _, _ := runtime.Caller(0)
	configPath := filepath.Join(filepath.Dir(thisFile), "example.toml")
	cmd := exec.Command(exePath, "-config", configPath, "-format", "json", "-count", "3")
	cmd.Dir = backendRoot
	var stdout, stderr = &cliOutputBuffer{}, &cliOutputBuffer{}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("CLI 执行失败: %v\nstderr: %s", err, stderr.String())
	}

	// stdout 必须是纯 JSON：提示混入会破坏管道消费（曾实测复现）
	var resp cliJSONResponse
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		t.Fatalf("-config + -format json 时 stdout 不是纯 JSON（提示信息污染输出流）: %v\n前 200 字节: %.200s", err, stdout.String())
	}
	if len(resp.Names) == 0 {
		t.Fatal("应至少生成一个候选名")
	}
	// 提示信息应改走 stderr
	if !strings.Contains(stderr.String(), "已加载配置文件") {
		t.Errorf("stderr 应包含配置加载提示，实际: %q", stderr.String())
	}
}

// cliOutputBuffer 线程安全的输出缓冲
type cliOutputBuffer struct {
	data []byte
}

func (b *cliOutputBuffer) Write(p []byte) (int, error) {
	b.data = append(b.data, p...)
	return len(p), nil
}

func (b *cliOutputBuffer) Bytes() []byte { return b.data }

func (b *cliOutputBuffer) String() string { return string(b.data) }
