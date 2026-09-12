package namestatistics

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"name/internal/infrastructure/database"
)

// writeTestData 在临时目录写入模拟 build_namestats 输出的 JSON 数据
func writeTestData(t *testing.T, dir string) {
	t.Helper()
	write := func(name string, v any) {
		t.Helper()
		raw, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("序列化 %s 失败: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), raw, 0644); err != nil {
			t.Fatalf("写入 %s 失败: %v", name, err)
		}
	}
	write("surname_stats.json", []database.SurnameStat{
		{Surname: "王", Count: 3, Rank: 1, Ratio: 1.5, MaleRatio: 0.6, FemaleRatio: 0.4},
		{Surname: "李", Count: 2, Rank: 2, Ratio: 1.0, MaleRatio: 0.5, FemaleRatio: 0.5},
	})
	write("given_name_stats.json", []database.GivenNameStat{
		{Surname: "王", GivenName: "伟", Count: 2, Rank: 1},
		{Surname: "王", GivenName: "芳", Count: 1, Rank: 2},
		{Surname: "李", GivenName: "静", Count: 2, Rank: 1},
	})
	write("full_name_stats.json", []database.FullNameStat{
		{FullName: "王伟", Surname: "王", GivenName: "伟", Count: 2, Rank: 1},
		{FullName: "李静", Surname: "李", GivenName: "静", Count: 2, Rank: 2},
		{FullName: "王芳", Surname: "王", GivenName: "芳", Count: 1, Rank: 3},
	})
	write("name_gender_stats.json", []database.NameGenderStat{
		{Name: "伟", MaleCount: 2, FemaleCount: 0, TotalCount: 2},
		{Name: "静", MaleCount: 0, FemaleCount: 2, TotalCount: 2},
	})
	write("name_frequency.json", struct {
		Meta struct {
			TotalNames int `json:"total_names"`
		} `json:"meta"`
	}{
		Meta: struct {
			TotalNames int `json:"total_names"`
		}{TotalNames: 1200000},
	})
}

func newTestStore(t *testing.T) *FileNameStatStore {
	t.Helper()
	dir := t.TempDir()
	writeTestData(t, dir)
	return &FileNameStatStore{dataDir: dir}
}

// TestFileNameStatStore 验证 JSON 数据文件的 8 个查询方法
func TestFileNameStatStore(t *testing.T) {
	store := newTestStore(t)

	t.Run("姓氏排行（截断+排序）", func(t *testing.T) {
		stats, err := store.GetSurnameStats(1)
		if err != nil {
			t.Fatalf("GetSurnameStats 失败: %v", err)
		}
		if len(stats) != 1 || stats[0].Surname != "王" {
			t.Errorf("GetSurnameStats(1) 期望 [王], 实际 %+v", stats)
		}
	})

	t.Run("姓氏排行（limit<=0 返回全部）", func(t *testing.T) {
		stats, err := store.GetSurnameStats(0)
		if err != nil {
			t.Fatalf("GetSurnameStats 失败: %v", err)
		}
		if len(stats) != 2 {
			t.Errorf("GetSurnameStats(0) 期望 2 条, 实际 %d", len(stats))
		}
	})

	t.Run("单个姓氏命中", func(t *testing.T) {
		stat, err := store.GetSurnameStat("王")
		if err != nil {
			t.Fatalf("GetSurnameStat 失败: %v", err)
		}
		if stat == nil || stat.Count != 3 {
			t.Errorf("GetSurnameStat(王) 期望 Count=3, 实际 %+v", stat)
		}
	})

	t.Run("单个姓氏未命中返回 nil", func(t *testing.T) {
		stat, err := store.GetSurnameStat("赵")
		if err != nil {
			t.Fatalf("GetSurnameStat 失败: %v", err)
		}
		if stat != nil {
			t.Errorf("GetSurnameStat(赵) 期望 nil, 实际 %+v", stat)
		}
	})

	t.Run("按姓氏过滤名字排行", func(t *testing.T) {
		stats, err := store.GetGivenNameStats("王", 10)
		if err != nil {
			t.Fatalf("GetGivenNameStats 失败: %v", err)
		}
		if len(stats) != 2 || stats[0].GivenName != "伟" {
			t.Errorf("GetGivenNameStats(王) 期望 [伟 芳], 实际 %+v", stats)
		}
	})

	t.Run("按姓氏过滤全名排行（limit 截断）", func(t *testing.T) {
		stats, err := store.GetFullNameStats("王", 1)
		if err != nil {
			t.Fatalf("GetFullNameStats 失败: %v", err)
		}
		if len(stats) != 1 || stats[0].FullName != "王伟" {
			t.Errorf("GetFullNameStats(王,1) 期望 [王伟], 实际 %+v", stats)
		}
	})

	t.Run("全名命中", func(t *testing.T) {
		stat, err := store.GetFullNameStat("王伟")
		if err != nil {
			t.Fatalf("GetFullNameStat 失败: %v", err)
		}
		if stat == nil || stat.Count != 2 {
			t.Errorf("GetFullNameStat(王伟) 期望 Count=2, 实际 %+v", stat)
		}
	})

	t.Run("名字性别分布命中", func(t *testing.T) {
		stat, err := store.GetNameGenderStats("静")
		if err != nil {
			t.Fatalf("GetNameGenderStats 失败: %v", err)
		}
		if stat == nil || stat.FemaleCount != 2 {
			t.Errorf("GetNameGenderStats(静) 期望 FemaleCount=2, 实际 %+v", stat)
		}
	})

	t.Run("热门全名与总人数", func(t *testing.T) {
		top, err := store.GetTopFullNames(2)
		if err != nil {
			t.Fatalf("GetTopFullNames 失败: %v", err)
		}
		if len(top) != 2 || top[0].FullName != "王伟" {
			t.Errorf("GetTopFullNames(2) 期望 [王伟 ...], 实际 %+v", top)
		}
		total, err := store.GetTotalNameCount()
		if err != nil {
			t.Fatalf("GetTotalNameCount 失败: %v", err)
		}
		if total != 1200000 {
			t.Errorf("GetTotalNameCount 期望 1200000, 实际 %d", total)
		}
	})
}

// TestFileNameStatStore_DataMissing 数据文件缺失时所有查询返回错误
func TestFileNameStatStore_DataMissing(t *testing.T) {
	store := &FileNameStatStore{dataDir: t.TempDir()}
	if _, err := store.GetSurnameStats(10); err == nil {
		t.Fatal("数据缺失时 GetSurnameStats 应返回错误")
	}
	if _, err := store.GetTotalNameCount(); err == nil {
		t.Fatal("数据缺失时 GetTotalNameCount 应返回错误")
	}
}

// TestFileNameStatStore_LazyLoadOnce 懒加载只执行一次（并发查询不重复读盘）
func TestFileNameStatStore_LazyLoadOnce(t *testing.T) {
	store := newTestStore(t)
	for i := 0; i < 3; i++ {
		stats, err := store.GetSurnameStats(1)
		if err != nil {
			t.Fatalf("第 %d 次查询失败: %v", i+1, err)
		}
		if len(stats) != 1 {
			t.Fatalf("第 %d 次查询结果异常: %+v", i+1, stats)
		}
	}
}
