package log

// Package log — 领域层日志端口（docs/43 P1-9）
//
// 领域层此前直接 import infrastructure/logger，属 DIP 反向（领域层依赖
// 基础设施实现）。本包定义领域层日志端口与包级门面：
//
//   - 领域包只 import 本包，调用形态与原 logger.Warn(...) 完全一致；
//   - 基础设施（infrastructure/logger）以适配器实现 Logger 并在组合根
//     注入（log.SetLogger）；未注入时使用 no-op 默认实现，领域测试与
//     纯函数调用零配置可跑；
//   - 字段类型沿用 zap.Field（第三方库类型，非基础设施包依赖）。

import "go.uber.org/zap"

// Logger 领域层日志端口（由基础设施适配实现）
type Logger interface {
	Debug(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
}

// global 当前生效实现；未注入前为 no-op，保证领域包零配置可运行。
var global Logger = noopLogger{}

// SetLogger 在组合根注入实现（须在服务启动早期调用一次）。
func SetLogger(l Logger) {
	if l == nil {
		l = noopLogger{}
	}
	global = l
}

// Current 返回当前实现（供诊断/测试断言注入是否生效）。
func Current() Logger { return global }

// Debug 记录调试日志（领域包统一入口）。
func Debug(msg string, fields ...zap.Field) { global.Debug(msg, fields...) }

// Info 记录信息日志（领域包统一入口）。
func Info(msg string, fields ...zap.Field) { global.Info(msg, fields...) }

// Warn 记录告警日志（领域包统一入口）。
func Warn(msg string, fields ...zap.Field) { global.Warn(msg, fields...) }

// Error 记录错误日志（领域包统一入口）。
func Error(msg string, fields ...zap.Field) { global.Error(msg, fields...) }

// noopLogger 默认实现：丢弃全部日志（未注入时的安全兜底）。
type noopLogger struct{}

func (noopLogger) Debug(string, ...zap.Field) {}
func (noopLogger) Info(string, ...zap.Field)  {}
func (noopLogger) Warn(string, ...zap.Field)  {}
func (noopLogger) Error(string, ...zap.Field) {}
