package main

// 启动失败的「友好提示」层。
//
// 背景（docs/48 §9.7/§9.8）：wails3 build 使用 -ldflags "-H windowsgui"，
// GUI 子系统没有控制台，任何早期失败或 panic 都表现为「双击后什么都没发生」。
// 仅把 err.Error() 弹出来对用户并不友好：里面混着 DATA_DIR、Options.DataDir
// 这类内部术语，且没有告诉用户能做什么。这里把失败分成四类，分别给出
// 面向最终用户的说明与可操作步骤；原始技术细节只写进 startup-error.log。
//
// 本文件不依赖任何平台 API，纯函数便于单测；弹框由
// startup_error_windows.go / startup_error_other.go 按平台提供。

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"time"
)

// startupFailureKind 启动失败类别
type startupFailureKind int

const (
	failureDataDirMissing startupFailureKind = iota + 1 // E-1001 运行所需数据目录找不到
	failureDataInit                                    // E-1002 物料/数据库初始化失败
	failureWindowRuntime                               // E-1003 窗口/运行时启动失败
	failureUnexpected                                  // E-1999 未捕获的 panic
)

// startupFailure 一次启动失败的结构化描述。
// detail 只进诊断日志，不进弹框主体；stack 仅在 panic 时非空。
type startupFailure struct {
	kind     startupFailureKind
	code     string
	heading  string
	reasons  []string
	actions  []string
	detail   string
	stack    string
	occurred time.Time
}

func (k startupFailureKind) label() string {
	switch k {
	case failureDataDirMissing:
		return "数据目录缺失"
	case failureDataInit:
		return "数据初始化失败"
	case failureWindowRuntime:
		return "窗口/运行时启动失败"
	case failureUnexpected:
		return "未预期异常"
	default:
		return "未知"
	}
}

// classifyStartupFailure 把「返回的错误」或「未捕获的 panic」归类为
// 面向用户的失败描述。panicValue 非 nil 时优先按未预期异常处理。
//
// 分类依据是错误文本的稳定子串（bootstrap 层的错误均为 fmt.Errorf 包装），
// 不引入新的错误类型，避免改动既有装配契约。
func classifyStartupFailure(err error, panicValue any) startupFailure {
	detail := "(无)"
	if err != nil {
		detail = err.Error()
	}
	f := startupFailure{detail: detail, occurred: time.Now()}

	switch {
	case panicValue != nil:
		f.kind = failureUnexpected
		f.code = "E-1999"
		f.heading = "程序遇到未预期的异常，已安全停止。"
		f.detail = fmt.Sprintf("panic: %v", panicValue)
		f.stack = string(debug.Stack())
		f.reasons = []string{
			"运行环境与程序预期不一致（系统组件、权限或文件损坏）",
		}
		f.actions = []string{
			"重新启动程序；若重复出现，重启电脑后再试",
			"使用安装程序覆盖安装到默认目录",
			"请把错误代码与 startup-error.log 一并提供给维护者（日志内含调用栈）",
		}

	case containsAny(detail, "未找到传统文化数据目录", "DATA_DIR", "data directory", "namer.json"):
		f.kind = failureDataDirMissing
		f.code = "E-1001"
		f.heading = "程序没能找到运行所需的「传统文化数据」，无法继续启动。"
		f.reasons = []string{
			"只复制了 namer-desktop.exe，没有一并复制同目录的 data 文件夹",
			"安装包不完整，或 data 文件夹被移动、改名、删除",
			"程序被放在无法读取的位置（如受保护的系统目录）",
		}
		f.actions = []string{
			"使用安装程序重新安装（推荐，会自动附带数据）",
			"把 data 文件夹放回与 namer-desktop.exe 相同的目录，或它的上一级目录",
			"若数据在别处，用命令行启动并指定：namer-desktop.exe -data \"数据文件夹完整路径\"",
			"重新安装后仍失败，请把 startup-error.log 提供给维护者",
		}

	case containsAny(detail,
		"装配失败", "物料播种失败", "物料导入失败", "创建数据库目录失败",
		"查询字库行数失败", "数据库", "sqlite", "database", "disk", "磁盘"):
		f.kind = failureDataInit
		f.code = "E-1002"
		f.heading = "程序在初始化起名语料或数据库时出错，已停止启动。"
		f.reasons = []string{
			"data 文件夹内缺少或损坏了语料文件",
			"程序所在目录没有写入权限，无法创建或更新 namer.db",
			"namer.db 损坏，或磁盘剩余空间不足",
		}
		f.actions = []string{
			"确认程序目录可写：不要放在受保护的系统目录，必要时以管理员身份运行",
			"检查磁盘剩余空间后重新启动",
			"若反复失败，备份后删除 data 文件夹内的 namer.db 让程序重建，或用安装程序覆盖安装",
			"把 startup-error.log 提供给维护者",
		}

	default:
		f.kind = failureWindowRuntime
		f.code = "E-1003"
		f.heading = "程序未能打开主窗口，启动已停止。"
		f.reasons = []string{
			"系统缺少 Microsoft Edge WebView2 运行时，或版本过旧",
			"显卡驱动或系统图形组件异常",
			"安全软件阻止了窗口创建",
		}
		f.actions = []string{
			"安装或更新 Microsoft Edge WebView2 运行时后重启程序",
			"重启电脑后再试，或暂时关闭拦截类安全软件",
			"把 startup-error.log 提供给维护者",
		}
	}
	return f
}

// formatFriendlyMessage 生成弹框文案：只讲「发生了什么 / 可能原因 / 我能做什么」，
// 不暴露内部环境变量名、Go 类型或 Options.DataDir 之类的实现细节。
// Windows MessageBoxW 需要 CRLF 换行。
func formatFriendlyMessage(f startupFailure) string {
	var b strings.Builder
	b.WriteString(f.heading)
	b.WriteString("\r\n\r\n【可能原因】\r\n")
	for i, r := range f.reasons {
		fmt.Fprintf(&b, "  %d. %s\r\n", i+1, r)
	}
	b.WriteString("\r\n【建议按顺序尝试】\r\n")
	for i, a := range f.actions {
		fmt.Fprintf(&b, "  %d. %s\r\n", i+1, a)
	}
	fmt.Fprintf(&b, "\r\n错误代码：%s\r\n", f.code)
	b.WriteString("程序不会删除或覆盖您已有的起名记录与收藏。\r\n")
	b.WriteString("如需反馈，请在本窗口按 Ctrl+C 复制内容，并把与本程序同目录的 startup-error.log 一并提供。\r\n")
	return b.String()
}

// formatDiagnostic 生成落盘日志：保留原始技术错误、运行路径与环境，
// panic 时附带调用栈。这部分不弹给用户，供排查与反馈使用。
func formatDiagnostic(f startupFailure) string {
	var b strings.Builder
	fmt.Fprintf(&b, "==== 启动失败诊断 %s ====\n", f.occurred.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&b, "错误代码: %s\n", f.code)
	fmt.Fprintf(&b, "类别: %s\n", f.kind.label())
	fmt.Fprintf(&b, "用户提示: %s\n", f.heading)
	fmt.Fprintf(&b, "技术原因: %s\n", f.detail)
	if exe, e := os.Executable(); e == nil {
		fmt.Fprintf(&b, "程序路径: %s\n", exe)
	}
	if wd, e := os.Getwd(); e == nil {
		fmt.Fprintf(&b, "工作目录: %s\n", wd)
	}
	fmt.Fprintf(&b, "运行环境: %s/%s Go %s\n", runtime.GOOS, runtime.GOARCH, runtime.Version())
	if f.stack != "" {
		fmt.Fprintf(&b, "--- panic 调用栈 ---\n%s\n", f.stack)
	}
	b.WriteString("==== 诊断结束 ====\n")
	return b.String()
}

// containsAny 大小写不敏感地判断 s 是否命中任一关键词。
func containsAny(s string, needles ...string) bool {
	lower := strings.ToLower(s)
	for _, n := range needles {
		if strings.Contains(lower, strings.ToLower(n)) {
			return true
		}
	}
	return false
}
