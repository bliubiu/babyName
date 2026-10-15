package checksum

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFile_内容相同则指纹相同(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.json")
	b := filepath.Join(dir, "b.json")
	if err := os.WriteFile(a, []byte(`{"x":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte(`{"x":1}`), 0o644); err != nil {
		t.Fatal(err)
	}

	sa, err := File(a)
	if err != nil {
		t.Fatalf("计算 a 指纹失败: %v", err)
	}
	sb, err := File(b)
	if err != nil {
		t.Fatalf("计算 b 指纹失败: %v", err)
	}
	if sa != sb {
		t.Errorf("内容相同的文件指纹应一致：%s != %s", sa, sb)
	}
	if len(sa) != 64 {
		t.Errorf("sha256 十六进制长度应为 64，实得 %d", len(sa))
	}
}

func TestFile_改一字节即变更(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.json")
	if err := os.WriteFile(p, []byte(`{"x":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	before, err := File(p)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(p, []byte(`{"x":2}`), 0o644); err != nil {
		t.Fatal(err)
	}
	after, err := File(p)
	if err != nil {
		t.Fatal(err)
	}
	if before == after {
		t.Fatal("内容变更后指纹必须变化，否则门控会漏掉物料修订")
	}
}

func TestFile_文件不存在报错(t *testing.T) {
	if _, err := File(filepath.Join(t.TempDir(), "缺失.json")); err == nil {
		t.Fatal("文件不存在时应返回错误")
	}
}

func TestFile_对目录调用报错(t *testing.T) {
	// os.Open 对目录成功但读取失败，覆盖 io.Copy 的错误分支：
	// 调用方误传目录路径时必须拿到错误而不是空指纹
	if _, err := File(t.TempDir()); err == nil {
		t.Fatal("对目录调用 File 应返回错误")
	}
}

func TestBytes_与File结果一致(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.bin")
	content := []byte("起名物料指纹测试")
	if err := os.WriteFile(p, content, 0o644); err != nil {
		t.Fatal(err)
	}
	sf, err := File(p)
	if err != nil {
		t.Fatal(err)
	}
	if got := Bytes(content); got != sf {
		t.Errorf("Bytes 与 File 结果应一致：%s != %s", got, sf)
	}
}

func TestStatOf_读出大小与修改时间(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.bin")
	if err := os.WriteFile(p, []byte("12345"), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := StatOf(p)
	if err != nil {
		t.Fatalf("读取文件状态失败: %v", err)
	}
	if st.Size != 5 {
		t.Errorf("size 期望 5，实得 %d", st.Size)
	}
	if st.ModTime == 0 {
		t.Error("modTime 不应为 0")
	}

	if _, err := StatOf(filepath.Join(dir, "缺失.bin")); err == nil {
		t.Error("文件不存在时 StatOf 应报错")
	}
}

// TestCache_快速路径与全量一致 未变更时复用缓存，结果必须与全量计算相同
func TestCache_快速路径与全量一致(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.json")
	if err := os.WriteFile(p, []byte(`{"v":1}`), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewCache()
	first, err := c.Of(p)
	if err != nil {
		t.Fatal(err)
	}
	second, err := c.Of(p) // 命中快速路径
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Errorf("快速路径结果应与首次一致：%s != %s", first, second)
	}

	full, err := File(p)
	if err != nil {
		t.Fatal(err)
	}
	if second != full {
		t.Errorf("缓存结果应与全量计算一致：%s != %s", second, full)
	}
}

// TestCache_内容变更后重算 size+mtime 变化时必须重算，不得返回旧指纹
func TestCache_内容变更后重算(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.json")
	if err := os.WriteFile(p, []byte(`{"v":1}`), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewCache()
	before, err := c.Of(p)
	if err != nil {
		t.Fatal(err)
	}

	// 改为不同长度，确保 size 也变化
	if err := os.WriteFile(p, []byte(`{"v":22222}`), 0o644); err != nil {
		t.Fatal(err)
	}
	after, err := c.Of(p)
	if err != nil {
		t.Fatal(err)
	}
	if before == after {
		t.Fatal("内容变更后缓存必须失效重算")
	}
}

// TestCache_内容变更但大小不变仍需重算 防止「等长修改」被快速路径漏掉
func TestCache_内容变更但大小不变仍需重算(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.json")
	if err := os.WriteFile(p, []byte(`{"v":1}`), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewCache()
	before, err := c.Of(p)
	if err != nil {
		t.Fatal(err)
	}

	// 等长修改，并推进 mtime 以确保缓存失效
	if err := os.WriteFile(p, []byte(`{"v":9}`), 0o644); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(p, future, future); err != nil {
		t.Fatal(err)
	}

	after, err := c.Of(p)
	if err != nil {
		t.Fatal(err)
	}
	if before == after {
		t.Fatal("等长内容修改后缓存必须失效（mtime 已推进）")
	}
}

func TestCache_Forget强制重算(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.json")
	if err := os.WriteFile(p, []byte(`{"v":1}`), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewCache()
	first, err := c.Of(p)
	if err != nil {
		t.Fatal(err)
	}
	c.Forget(p)
	second, err := c.Of(p)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Error("内容未变，Forget 前后指纹应一致")
	}

	if err := os.WriteFile(p, []byte(`{"v":2}`), 0o644); err != nil {
		t.Fatal(err)
	}
	third, err := c.Of(p)
	if err != nil {
		t.Fatal(err)
	}
	if third == second {
		t.Error("Forget 后应重新读取文件内容")
	}
}

func TestCache_文件缺失报错(t *testing.T) {
	c := NewCache()
	if _, err := c.Of(filepath.Join(t.TempDir(), "缺失.json")); err == nil {
		t.Fatal("文件不存在时 Cache.Of 应报错")
	}
}

func TestCache_对目录调用报错(t *testing.T) {
	// os.Stat 对目录成功而读取失败：覆盖 Of 中 StatOf 成功但 File 失败的分支
	c := NewCache()
	if _, err := c.Of(t.TempDir()); err == nil {
		t.Fatal("Cache.Of 对目录应报错")
	}
	// 失败的路径不得进缓存（否则下次同参数直接返回空指纹）
	if _, err := c.Of(t.TempDir()); err == nil {
		t.Fatal("不同目录同样应报错")
	}
}

func TestCache_各路径独立缓存(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.json")
	b := filepath.Join(dir, "b.json")
	if err := os.WriteFile(a, []byte(`{"a":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte(`{"b":1}`), 0o644); err != nil {
		t.Fatal(err)
	}

	c := NewCache()
	sa, err := c.Of(a)
	if err != nil {
		t.Fatal(err)
	}
	sb, err := c.Of(b)
	if err != nil {
		t.Fatal(err)
	}
	if sa == sb {
		t.Error("不同文件内容不同，指纹不应相同")
	}
}
