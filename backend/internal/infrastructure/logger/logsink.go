package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"name/internal/paths"
)

// 本文件实现固定目录 + 按天轮转的日志落盘（AGENTS.md 日志约定）：
//
//   - 目录：固定「项目根/logs/」（go.mod 所在目录，与 cwd 无关——cwd 相对路径
//     曾让同一配置在仓库根与 backend/ 两处各产出一份 namer.log，docs/36）；
//   - 文件名：namer-YYYYMMDD.log，每天一个，写入时按本地日期自动切换；
//   - 保留：32 天，启动与跨天切换时清理过期文件。
//
// 选用自实现 sink 而非 lumberjack：lumberjack 只按「大小」滚动且归档名
// 不含固定日期段（namer-YYYYMMDD.log 要求按天逐字面对应文件名），
// 与 AGENTS.md「按日期轮转、每日一个文件」的约定不匹配。

const (
	// logFilePrefix 文件名前缀（项目名，AGENTS.md：namer-YYYYMMDD.log）
	logFilePrefix = "namer-"
	// logFileExt 文件扩展名
	logFileExt = ".log"
	// logDayLayout 文件名中的日期格式（YYYYMMDD）
	logDayLayout = "20060102"
	// logRetentionDays 日志保留天数（AGENTS.md：保留 32 天日志，自动清理过期）
	logRetentionDays = 32
)

// loggerSourceFile 以本文件自身位置为「项目根」探测锚点
//
// 提取为包级变量是为了可测：与 cmd/server 的 mainSourceFile 同一范式，
// 不需要真的离开仓库即可通过替换变量定位到临时目录。
var loggerSourceFile = func() string {
	_, file, _, _ := runtime.Caller(0)
	return file
}

// DefaultLogDir 固定日志目录：项目根/logs/
//
// 从仓库根或 backend/ 两种 cwd 启动都指向 backend/logs/；探测不到 go.mod
// （容器内 trimpath 构建、或进程运行于任意目录）时回落 cwd/logs/。
func DefaultLogDir() string {
	if base := paths.FindModuleDir(loggerSourceFile()); base != "" {
		return filepath.Join(base, "logs")
	}
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	return filepath.Join(cwd, "logs")
}

// ResolveLogDir 解析日志目录
//
// 优先级：绝对路径原样使用 > 相对路径相对 cwd > 空值固定 DefaultLogDir。
// 相对路径随 cwd 漂移是历史上日志散射成多份的根因，故仅限显式
// 指定（-log mylogs）时保留该语义；缺省配置必须建立「空值 → 固定目录」。
func ResolveLogDir(outputPath string) string {
	d := strings.TrimSpace(outputPath)
	if d == "" {
		return DefaultLogDir()
	}
	if !filepath.IsAbs(d) {
		if cwd, err := os.Getwd(); err == nil {
			d = filepath.Join(cwd, d)
		}
	}
	return d
}

// dailySink 按天轮转的追加写句柄（实现 zapcore.WriteSyncer）
//
// 写入前比对当前本地日期，跨天自动切换到新文件并清理过期归档；
// 跨天切换打开失败时不吞错误（宁可让该行日志写失败，也不能让日志
// 静默流向错误文件）。
type dailySink struct {
	mu  sync.Mutex
	dir string
	// now 当前时间取值函数；注入 fake 以便测试覆盖跨天切换与保留清理
	now func() time.Time
	// day 当前已打开文件的日期段（YYYYMMDD）
	day string
	f   *os.File
}

// newDailySink 创建按天轮转 sink；now 为 nil 时用本地当前时间
func newDailySink(dir string, now func() time.Time) *dailySink {
	if now == nil {
		now = time.Now
	}
	return &dailySink{dir: dir, now: now}
}

// Write 写入一批日志字节；跨天时先切换文件
func (s *dailySink) Write(p []byte) (n int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	day := s.now().Format(logDayLayout)
	if day != s.day {
		if err := s.openDay(day); err != nil {
			return 0, err
		}
	}
	return s.f.Write(p)
}

// openDay 关闭旧日文件、以追加模式打开新日文件，并清理过期归档
func (s *dailySink) openDay(day string) error {
	if s.f != nil {
		_ = s.f.Close()
		s.f = nil
	}
	f, err := os.OpenFile(s.fileName(day), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %w", err)
	}
	s.f = f
	s.day = day
	s.cleanupExpired()
	return nil
}

// fileName 日志文件完整路径：dir/namer-YYYYMMDD.log
func (s *dailySink) fileName(day string) string {
	return filepath.Join(s.dir, logFilePrefix+day+logFileExt)
}

// cleanupExpired 清理超过保留期（32 天）的按天归档文件
//
// 只删符合 namer-YYYYMMDD.log 命名格式且日期早于 today-32 天的文件，
// 同目录的其它文件（如手工保留的调试日志）不碰。
func (s *dailySink) cleanupExpired() {
	today, err := time.ParseInLocation(logDayLayout, s.day, time.Local)
	if err != nil {
		return
	}
	cutoff := today.AddDate(0, 0, -logRetentionDays)

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, logFilePrefix) || !strings.HasSuffix(name, logFileExt) {
			continue
		}
		dayStr := strings.TrimSuffix(strings.TrimPrefix(name, logFilePrefix), logFileExt)
		dayDate, err := time.ParseInLocation(logDayLayout, dayStr, time.Local)
		if err != nil {
			continue // 命名接近但日期段非法的文件不动
		}
		if dayDate.Before(cutoff) {
			_ = os.Remove(filepath.Join(s.dir, name))
		}
	}
}

// Sync 刷新内核缓冲（zap 的 Sync() 顺着 WriteSyncer 链调用到此处）
func (s *dailySink) Sync() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.f != nil {
		return s.f.Sync()
	}
	return nil
}

// Close 关闭当前日志文件
func (s *dailySink) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.f != nil {
		err := s.f.Close()
		s.f = nil
		return err
	}
	return nil
}
