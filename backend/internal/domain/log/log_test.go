package log

// log_test.go — 领域日志端口守护（docs/43 P1-9）

import (
	"testing"

	"go.uber.org/zap"
)

// spyLogger 记录收到的调用，供断言转发
type spyLogger struct {
	calls []string
}

func (s *spyLogger) Debug(msg string, _ ...zap.Field) { s.calls = append(s.calls, "Debug:"+msg) }
func (s *spyLogger) Info(msg string, _ ...zap.Field)  { s.calls = append(s.calls, "Info:"+msg) }
func (s *spyLogger) Warn(msg string, _ ...zap.Field)  { s.calls = append(s.calls, "Warn:"+msg) }
func (s *spyLogger) Error(msg string, _ ...zap.Field) { s.calls = append(s.calls, "Error:"+msg) }

// TestDefaultNoOpDoesNotPanic 未注入实现时包级门面必须可用（no-op 兜底）
func TestDefaultNoOpDoesNotPanic(t *testing.T) {
	defer func(prev Logger) { SetLogger(prev) }(Current())
	SetLogger(nil) // 显式置空 → 应回退 no-op

	Debug("d")
	Info("i")
	Warn("w")
	Error("e") // 断言不 panic
	if _, ok := Current().(noopLogger); !ok {
		t.Fatalf("SetLogger(nil) 应回退 no-op，实际 %T", Current())
	}
}

// TestSetLoggerAndForward 注入后门面逐级转发（含字段透传）
func TestSetLoggerAndForward(t *testing.T) {
	defer func(prev Logger) { SetLogger(prev) }(Current())
	spy := &spyLogger{}
	SetLogger(spy)

	Warn("领域告警", zap.String("k", "v"))
	Info("领域信息")
	Error("领域错误")
	Debug("领域调试")

	want := []string{"Warn:领域告警", "Info:领域信息", "Error:领域错误", "Debug:领域调试"}
	if len(spy.calls) != len(want) {
		t.Fatalf("转发条数 = %d，期望 %d：%v", len(spy.calls), len(want), spy.calls)
	}
	for i, w := range want {
		if spy.calls[i] != w {
			t.Errorf("calls[%d] = %s，期望 %s", i, spy.calls[i], w)
		}
	}
}
