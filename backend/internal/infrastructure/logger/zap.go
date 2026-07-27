package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	Log     *zap.Logger
	Sugar  *zap.SugaredLogger
)

type Config struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	OutputPath string `mapstructure:"output_path"`
}

func Init(cfg *Config) error {
	if cfg == nil {
		cfg = &Config{
			Level:      "debug",
			Format:     "console",
			OutputPath: "logs/app.log",
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

	var encoder zapcore.Encoder
	if cfg.Format == "json" {
		encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		// 如果输出到文件，不使用颜色代码
		if cfg.OutputPath != "" {
			encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
		}
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	var writeSyncer zapcore.WriteSyncer
	if cfg.OutputPath != "" {
		// 提取日志文件所在目录
		dir := filepath.Dir(cfg.OutputPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		// 使用 lumberjack 实现日志轮转
		lj := &lumberjack.Logger{
			Filename:   cfg.OutputPath,
			MaxSize:    100, // MB，单文件最大大小
			MaxBackups: 32,  // 保留旧文件最大个数（与 MaxAge 一致，满足 AGENTS.md 保留 32 天要求）
			MaxAge:     32,  // 保留旧文件最大天数
			Compress:   true, // 压缩旧文件
		}
		writeSyncer = zapcore.AddSync(lj)
	} else {
		writeSyncer = zapcore.AddSync(os.Stdout)
	}

	core := zapcore.NewCore(
		encoder,
		writeSyncer,
		level,
	)

	Log = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	Sugar = Log.Sugar()

	return nil
}

func InitProduction() error {
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

	// 生产环境同时输出到文件和标准输出
	lj := &lumberjack.Logger{
		Filename:   "logs/app.log",
		MaxSize:    100,
		MaxBackups: 32,
		MaxAge:     32,
		Compress:   true,
	}
	_ = os.MkdirAll("logs", 0755)

	writeSyncer := zapcore.NewMultiWriteSyncer(
		zapcore.AddSync(lj),
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
		Log.Sync()
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
