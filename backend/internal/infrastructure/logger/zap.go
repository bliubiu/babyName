package logger

import (
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	Log   *zap.Logger
	Sugar *zap.SugaredLogger
)

type Config struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	// OutputPath 日志**目录**（语义变更，docs/36 日志路径锚定）：
	// 绝对路径原样使用，相对路径相对 cwd；空值固定「项目根/logs/」。
	// 文件名固定 namer-YYYYMMDD.log，按天轮转，由 dailySink 统一管理。
	OutputPath string `mapstructure:"output_path"`
}

func Init(cfg *Config) error {
	if cfg == nil {
		cfg = &Config{
			Level:  "debug",
			Format: "console",
		}
	}

	level := zapcore.InfoLevel
	if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
		level = zapcore.InfoLevel
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05"),
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 解析日志目录：空值（配置缺省）固定「项目根/logs/」（AGENTS.md 日志约定，
	// docs/36 日志路径锚定）；绝对路径原样使用；相对路径相对 cwd。
	dir := ResolveLogDir(cfg.OutputPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("创建日志目录 %s 失败: %w", dir, err)
	}

	var encoder zapcore.Encoder
	if cfg.Format == "json" {
		encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		// 终端与文件共用同一编码器：文件不支持 ANSI 颜色，
		// 故统一关闭颜色（终端失去彩色，换取文件里干净的可 grep 文本）
		encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// 按天轮转落盘：文件名 namer-YYYYMMDD.log 由 dailySink 维护，
	// 跨天自动切换 + 保留 32 天清理（超过该职责的收入不再交给调用方）。
	sink := newDailySink(dir, time.Now)

	writeSyncer := zapcore.NewMultiWriteSyncer(
		zapcore.AddSync(sink),
		zapcore.AddSync(os.Stdout),
	)

	core := zapcore.NewCore(
		encoder,
		writeSyncer,
		level,
	)

	Log = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	Sugar = Log.Sugar()

	return nil
}

// InitProduction 生产级初始化：JSON 格式 + 同时落盘与标准输出
//
// dir 为日志目录（语义同 Config.OutputPath）；空值 → DefaultLogDir。
// 全部测试环境调用方应传入临时目录，避免测试产物写入真实 logs/。
func InitProduction(dir string) error {
	cfg := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02T15:04:05.000Z07:00"),
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 生产环境同时输出到固定日志目录（按天轮转）和标准输出
	if err := os.MkdirAll(ResolveLogDir(dir), 0o755); err != nil {
		return err
	}
	sink := newDailySink(ResolveLogDir(dir), time.Now)

	writeSyncer := zapcore.NewMultiWriteSyncer(
		zapcore.AddSync(sink),
		zapcore.AddSync(os.Stdout),
	)

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(cfg),
		writeSyncer,
		zapcore.InfoLevel,
	)

	Log = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	Sugar = Log.Sugar()

	return nil
}

func Debug(msg string, fields ...zap.Field) {
	if Log == nil {
		return
	}
	Log.Debug(msg, fields...)
}

func Info(msg string, fields ...zap.Field) {
	if Log == nil {
		return
	}
	Log.Info(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	if Log == nil {
		return
	}
	Log.Warn(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	if Log != nil {
		Log.Fatal(msg, fields...)
	}
	os.Exit(1)
}

// Deprecated: 使用 Error(msg, ...) 替代 LogError 需要先重命名 Error(err) Field 辅助函数
func LogError(msg string, fields ...zap.Field) {
	if Log == nil {
		return
	}
	Log.Error(msg, fields...)
}

// Error 记录错误日志
func Error(msg string, fields ...zap.Field) {
	if Log == nil {
		return
	}
	Log.Error(msg, fields...)
}

func Debugf(template string, args ...interface{}) {
	if Log == nil {
		return
	}
	Sugar.Debugf(template, args...)
}

func Infof(template string, args ...interface{}) {
	if Log == nil {
		return
	}
	Sugar.Infof(template, args...)
}

func Warnf(template string, args ...interface{}) {
	if Log == nil {
		return
	}
	Sugar.Warnf(template, args...)
}

func Errorf(template string, args ...interface{}) {
	if Log == nil {
		return
	}
	Sugar.Errorf(template, args...)
}

func Fatalf(template string, args ...interface{}) {
	if Log != nil {
		Sugar.Fatalf(template, args...)
	}
	os.Exit(1)
}

func With(fields ...zap.Field) *zap.Logger {
	return Log.With(fields...)
}

func Sync() {
	if Log != nil {
		_ = Log.Sync()
	}
}

type Field = zap.Field

func String(key string, value string) Field {
	return zap.String(key, value)
}

func Int(key string, value int) Field {
	return zap.Int(key, value)
}

func Int64(key string, value int64) Field {
	return zap.Int64(key, value)
}

func Duration(key string, value time.Duration) Field {
	return zap.Duration(key, value)
}

func Bool(key string, value bool) Field {
	return zap.Bool(key, value)
}

func Any(key string, value interface{}) Field {
	return zap.Any(key, value)
}

func ErrField(err error) Field {
	return zap.Error(err)
}

func Stringer(key string, value fmt.Stringer) Field {
	return zap.Stringer(key, value)
}
