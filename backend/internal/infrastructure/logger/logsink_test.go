package logger

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeDay 注入 fake 时钟（固定起始日，可经 add 推进）
type fakeDay struct {
	day time.Time
}

func (f *fakeDay) now() time.Time { return f.day }
func (f *fakeDay) add(days int)   { f.day = f.day.AddDate(0, 0, days) }
func (f *fakeDay) format() string { return f.day.Format(logDayLayout) }

// TestDailySink_首写落当日文件 首次写入应创建 namer-YYYYMMDD.log 并写入内容
func TestDailySink_首写落当日文件(t *testing.T) {
	dir := t.TempDir()
	day := &fakeDay{day: time.Date(2026, 10, 21, 10, 0, 0, 0, time.Local)}
	s := newDailySink(dir, day.now)
	t.Cleanup(func() { _ = s.Close() })

	if _, err := s.Write([]byte("hello\n")); err != nil {
		t.Fatalf("Write 失败: %v", err)
	}
	path := s.fileName(day.format())
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读日志文件失败: %v", err)
	}
	if string(data) != "hello\n" {
		t.Fatalf("文件内容 = %q, 期望 %q", string(data), "hello\n")
	}
}

// TestDailySink_跨天自动切换 跨天后写入应落新文件，旧文件内容保留
func TestDailySink_跨天自动切换(t *testing.T) {
	dir := t.TempDir()
	day := &fakeDay{day: time.Date(2026, 10, 21, 23, 59, 58, 0, time.Local)}
	s := newDailySink(dir, day.now)
	t.Cleanup(func() { _ = s.Close() })

	if _, err := s.Write([]byte("day1\n")); err != nil {
		t.Fatalf("Write day1 失败: %v", err)
	}
	day.add(1) // 跨天
	if _, err := s.Write([]byte("day2\n")); err != nil {
		t.Fatalf("Write day2 失败: %v", err)
	}

	d1, err := os.ReadFile(s.fileName("20261021"))
	if err != nil || string(d1) != "day1\n" {
		t.Fatalf("旧日文件内容 = %q, err=%v, 期望 %q", string(d1), err, "day1\n")
	}
	d2, err := os.ReadFile(s.fileName("20261022"))
	if err != nil || string(d2) != "day2\n" {
		t.Fatalf("新日文件内容 = %q, err=%v, 期望 %q", string(d2), err, "day2\n")
	}
}

// TestDailySink_同日追加不截断 同一天多次写应追加而非覆盖
func TestDailySink_同日追加不截断(t *testing.T) {
	dir := t.TempDir()
	day := &fakeDay{day: time.Date(2026, 10, 21, 10, 0, 0, 0, time.Local)}
	s := newDailySink(dir, day.now)
	t.Cleanup(func() { _ = s.Close() })

	if _, err := s.Write([]byte("a\n")); err != nil {
		t.Fatalf("Write a 失败: %v", err)
	}
	if _, err := s.Write([]byte("b\n")); err != nil {
		t.Fatalf("Write b 失败: %v", err)
	}
	data, _ := os.ReadFile(s.fileName("20261021"))
	if string(data) != "a\nb\n" {
		t.Fatalf("内容 = %q, 期望 %q", string(data), "a\nb\n")
	}
}

// TestDailySink_过期归档清理 超过 32 天的按天归档应在跨天切换时被删除
func TestDailySink_过期归档清理(t *testing.T) {
	dir := t.TempDir()
	stale := filepath.Join(dir, "namer-20260101.log")
	fresh := filepath.Join(dir, "namer-20261001.log")
	unrelated := filepath.Join(dir, "keep-me.log")
	for _, p := range []string{stale, fresh, unrelated} {
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatalf("准备归档文件失败: %v", err)
		}
	}

	day := &fakeDay{day: time.Date(2026, 10, 21, 10, 0, 0, 0, time.Local)}
	s := newDailySink(dir, day.now)
	t.Cleanup(func() { _ = s.Close() })

	// 首写触发 openDay → cleanupExpired
	if _, err := s.Write([]byte("today\n")); err != nil {
		t.Fatalf("Write 失败: %v", err)
	}

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("过期归档未清理（2026-10-21 - 32天 = 2026-09-19，20260101 应删除）")
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Fatalf("保留期内归档不应删除: %v", err)
	}
	if _, err := os.Stat(unrelated); err != nil {
		t.Fatalf("非 namer- 命名的文件不应删除: %v", err)
	}
}

// TestDailySink_并发写安全 并发 goroutine 写入不应竞争（-race 下验证）
func TestDailySink_并发写安全(t *testing.T) {
	dir := t.TempDir()
	day := &fakeDay{day: time.Date(2026, 10, 21, 10, 0, 0, 0, time.Local)}
	s := newDailySink(dir, day.now)
	t.Cleanup(func() { _ = s.Close() })

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, _ = s.Write([]byte{byte('a' + i), '\n'})
		}(i)
	}
	wg.Wait()
	// 共 8 行，各自单字节 + 换行
	data, _ := os.ReadFile(s.fileName("20261021"))
	if got := len(strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")); got != 8 {
		t.Fatalf("并发写入行数 = %d, 期望 8（内容=%q）", got, string(data))
	}
}

// TestResolveLogDir_优先级 空值锚定/绝对原样/相对 cwd 三种解析分支
func TestResolveLogDir_优先级(t *testing.T) {
	t.Run("空值固定默认目录", func(t *testing.T) {
		want := DefaultLogDir()
		if got := ResolveLogDir(""); got != want {
			t.Fatalf("ResolveLogDir(\"\") = %q, 期望 %q", got, want)
		}
	})
	t.Run("空格空白同样按空值处理", func(t *testing.T) {
		if got := ResolveLogDir("   "); got != DefaultLogDir() {
			t.Fatalf("ResolveLogDir(\"  \") = %q, 期望默认目录", got)
		}
	})
	t.Run("绝对路径原样", func(t *testing.T) {
		want := filepath.Join(os.TempDir(), "mylogs")
		if got := ResolveLogDir(want); got != want {
			t.Fatalf("ResolveLogDir(%q) = %q", want, got)
		}
	})
	t.Run("相对路径相对cwd", func(t *testing.T) {
		got := ResolveLogDir("mylogs")
		if !filepath.IsAbs(got) {
			t.Fatalf("相对路径应解析为绝对路径, got %q", got)
		}
		if !strings.HasSuffix(got, filepath.Join("mylogs")) {
			t.Fatalf("应保留相对段, got %q", got)
		}
	})
}

// TestDefaultLogDir_锚定 默认目录应位于 go.mod 模块根的 logs/ 子目录
//
// loggerSourceFile 指向仓库内源文件，向上探测必命中 backend/go.mod；
// 校验形态（模块根/logs）而非具体盘符，机器无关。
func TestDefaultLogDir_锚定(t *testing.T) {
	got := DefaultLogDir()
	if filepath.Base(got) != "logs" {
		t.Fatalf("默认目录应名为 logs/, got %q", got)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(got), "go.mod")); err != nil {
		t.Fatalf("默认日志目录应锚定在模块根/logs（父目录含 go.mod）, got %q, err=%v",
			filepath.Dir(got), err)
	}
}
