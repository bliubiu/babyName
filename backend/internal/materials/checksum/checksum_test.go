package checksum

import (
	"os"
	"path/filepath"
	"testing"
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

// TestBytes_空输入 空内容同样有确定指纹（subscribe 侧 HashBytes(nil) 即此语义）
func TestBytes_空输入(t *testing.T) {
	want := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if got := Bytes(nil); got != want {
		t.Errorf("空输入指纹应为空串 sha256，实得 %s", got)
	}
}
