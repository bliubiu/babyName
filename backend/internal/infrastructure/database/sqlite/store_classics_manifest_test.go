package sqlite

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"name/internal/materials/schema"
)

// 本文件守护 docs/40 的「入库清单单一真源」：classics 导入的文件集合
// 必须与 schema.ClassicsFiles、classicsFileCategory 三者一致，任何一处
// 单边新增/删除都会被测试拦截。

// TestClassicsManifest_与分类表双向一致 清单 ⇄ 分类表必须互为子集。
func TestClassicsManifest_与分类表双向一致(t *testing.T) {
	def := schema.Registry[schema.ClassicsMaterial]
	if len(def.Files) == 0 {
		t.Fatal("classics 未登记 Files 清单")
	}

	inManifest := make(map[string]bool, len(def.Files))
	for _, f := range def.Files {
		if inManifest[f] {
			t.Errorf("classics 清单存在重复文件 %q", f)
		}
		inManifest[f] = true
		if _, ok := classicsFileCategory[f]; !ok {
			t.Errorf("清单文件 %q 未在 classicsFileCategory 登记分类", f)
		}
	}

	for f := range classicsFileCategory {
		if !inManifest[f] {
			t.Errorf("classicsFileCategory 文件 %q 不在 schema.ClassicsFiles 清单", f)
		}
	}
}

// TestClassicsManifest_排除非经典文件 防止把字库/词库/策展名等误登记为经典。
func TestClassicsManifest_排除非经典文件(t *testing.T) {
	nonClassics := []string{
		"hanzi.json", "word.json", "curated_names.json",
		"standard_chars.json", "zodiac.json",
	}
	for _, bad := range nonClassics {
		for _, f := range schema.ClassicsFiles {
			if f == bad {
				t.Errorf("非经典文件 %q 不应进入 classics 清单", bad)
			}
		}
	}
}

// TestClassicsSourceFiles_按清单且过滤缺失 导入只认清单；缺失文件被过滤，
// 目录里的清单外 JSON 不会被顺带导入。
func TestClassicsSourceFiles_按清单且过滤缺失(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "shijing.json"), []byte("[]"), 0o644); err != nil {
		t.Fatalf("写入测试文件失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "not-in-manifest.json"), []byte("[]"), 0o644); err != nil {
		t.Fatalf("写入清单外文件失败: %v", err)
	}

	got := classicsSourceFiles(dir)
	want := []string{"shijing.json"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("清单内存在的文件期望 %v，实得 %v", want, got)
	}
}
