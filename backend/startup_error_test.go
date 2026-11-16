package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestWriteStartupErrorLog 启动错误必须落盘，GUI 子系统无控制台时仍可排查
func TestWriteStartupErrorLog(t *testing.T) {
	dir := t.TempDir()
	if err := writeStartupErrorLog(dir, "应用启动失败：未找到传统文化数据目录"); err != nil {
		t.Fatalf("写入启动错误日志失败: %v", err)
	}

	path := filepath.Join(dir, "startup-error.log")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取启动错误日志失败: %v", err)
	}
	if !strings.Contains(string(b), "应用启动失败") {
		t.Fatalf("启动错误日志不含错误前缀: %q", string(b))
	}
	if !strings.Contains(string(b), "未找到传统文化数据目录") {
		t.Fatalf("启动错误日志不含具体原因: %q", string(b))
	}
}

// TestWriteStartupErrorLog_追加不覆盖 连续多次启动失败要保留上一次诊断
func TestWriteStartupErrorLog_追加不覆盖(t *testing.T) {
	dir := t.TempDir()
	if err := writeStartupErrorLog(dir, "第一次失败 E-1001"); err != nil {
		t.Fatalf("首次写入失败: %v", err)
	}
	if err := writeStartupErrorLog(dir, "第二次失败 E-1002"); err != nil {
		t.Fatalf("二次写入失败: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "startup-error.log"))
	if err != nil {
		t.Fatalf("读取失败: %v", err)
	}
	if !strings.Contains(string(b), "第一次失败 E-1001") || !strings.Contains(string(b), "第二次失败 E-1002") {
		t.Fatalf("日志应追加保留两次记录，实际: %q", string(b))
	}
}

// TestClassifyStartupFailure_四类错误代码 分类必须稳定，错误代码用于用户反馈
func TestClassifyStartupFailure_四类错误代码(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code string
		kind startupFailureKind
	}{
		{
			name: "数据目录缺失",
			err:  errors.New("未找到传统文化数据目录，请通过 DATA_DIR 环境变量或 Options.DataDir 指定"),
			code: "E-1001", kind: failureDataDirMissing,
		},
		{
			name: "物料播种失败",
			err:  errors.New("应用装配失败: 物料播种失败: 语料文件损坏"),
			code: "E-1002", kind: failureDataInit,
		},
		{
			name: "数据库不可写",
			err:  errors.New("应用装配失败: unable to open database file"),
			code: "E-1002", kind: failureDataInit,
		},
		{
			name: "WebView2 运行时缺失",
			err:  errors.New("WebView2 runtime not found"),
			code: "E-1003", kind: failureWindowRuntime,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := classifyStartupFailure(tc.err, nil)
			if f.code != tc.code || f.kind != tc.kind {
				t.Fatalf("code=%s kind=%v，期望 %s / %v", f.code, f.kind, tc.code, tc.kind)
			}
			if f.detail != tc.err.Error() {
				t.Fatalf("诊断必须保留原始错误: %q", f.detail)
			}
			if f.heading == "" || len(f.reasons) == 0 || len(f.actions) < 2 {
				t.Fatalf("友好说明三要素不完整: %+v", f)
			}
		})
	}
}

// TestClassifyStartupFailure_Panic 未捕获 panic 必须归为 E-1999 并带调用栈
func TestClassifyStartupFailure_Panic(t *testing.T) {
	f := classifyStartupFailure(nil, "runtime error: invalid memory address")
	if f.code != "E-1999" || f.kind != failureUnexpected {
		t.Fatalf("panic 应归为 E-1999 / failureUnexpected，实际 %s / %v", f.code, f.kind)
	}
	if !strings.Contains(f.detail, "invalid memory address") {
		t.Fatalf("诊断应含 panic 值: %q", f.detail)
	}
	if !strings.Contains(f.stack, "goroutine") {
		t.Fatalf("诊断应含调用栈: %q", f.stack[:min(len(f.stack), 120)])
	}
}

// TestFormatFriendlyMessage_面向用户 弹框不暴露内部术语，且给出可操作信息
func TestFormatFriendlyMessage_面向用户(t *testing.T) {
	f := classifyStartupFailure(
		errors.New("未找到传统文化数据目录，请通过 DATA_DIR 环境变量或 Options.DataDir 指定"), nil)
	msg := formatFriendlyMessage(f)

	for _, internal := range []string{"DATA_DIR", "Options.DataDir", "runtime.Caller", "fmt.Errorf", "bootstrap"} {
		if strings.Contains(msg, internal) {
			t.Fatalf("友好文案不应暴露内部术语 %q: %s", internal, msg)
		}
	}
	for _, want := range []string{"可能原因", "建议按顺序尝试", "E-1001", "startup-error.log", "data 文件夹", "-data", "Ctrl+C"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("友好文案缺少 %q: %s", want, msg)
		}
	}
	if !strings.Contains(msg, "\r\n") {
		t.Fatalf("Windows MessageBoxW 需要 CRLF 换行")
	}
}

// TestFormatDiagnostic_含排查上下文 日志保留技术原因与运行环境，panic 附栈
func TestFormatDiagnostic_含排查上下文(t *testing.T) {
	f := classifyStartupFailure(errors.New("应用装配失败: 物料播种失败"), nil)
	diag := formatDiagnostic(f)
	for _, want := range []string{"错误代码:", "技术原因: 应用装配失败", "程序路径:", "工作目录:", "运行环境:", "诊断结束"} {
		if !strings.Contains(diag, want) {
			t.Fatalf("诊断日志缺少 %q: %s", want, diag)
		}
	}
	if strings.Contains(diag, "【可能原因】") {
		t.Fatalf("诊断日志不应混入弹框排版，避免冗余: %s", diag)
	}

	pf := classifyStartupFailure(nil, "boom")
	if !strings.Contains(formatDiagnostic(pf), "panic 调用栈") {
		t.Fatalf("panic 诊断应含调用栈段落")
	}
}

// TestContainsAny_大小写不敏感 第三方错误文本大小写不定，分类不能漏判
func TestContainsAny_大小写不敏感(t *testing.T) {
	if !containsAny("UNABLE TO OPEN DATABASE FILE", "database") {
		t.Fatal("应大小写不敏感命中 database")
	}
	if containsAny("一切正常", "database", "sqlite") {
		t.Fatal("无关文本不应命中")
	}
}
