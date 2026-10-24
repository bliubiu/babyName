package namestat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"testing"

	"name/internal/domain/hanzi"
	"name/internal/testutil"
)

// TestMain 加载人名频率数据库（data/name_frequency.json 等）
func TestMain(m *testing.M) {
	_, file, _, _ := runtime.Caller(0)
	dataDir := testutil.DataDir(file)
	if dataDir == "" {
		fmt.Fprintln(os.Stderr, "未找到物料数据目录，namestat 包测试无法运行")
		os.Exit(1)
	}
	if err := hanzi.LoadFrequencyDB(dataDir); err != nil {
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func TestGetNameCount_SingleChar(t *testing.T) {
	// 单字名：查单字频率表
	if got := GetNameCount("梓"); got <= 0 {
		t.Fatalf("GetNameCount(梓) = %d，期望 > 0（真实频率数据应含高频字）", got)
	}
}

func TestGetNameCount_DoubleChar(t *testing.T) {
	// 双字名：查双字组合频率表
	if got := GetNameCount("梓涵"); got <= 0 {
		t.Fatalf("GetNameCount(梓涵) = %d，期望 > 0", got)
	}
}

func TestGetNameCount_Unknown(t *testing.T) {
	// 未收录的名字返回 0（而非伪造一个数字）
	if got := GetNameCount("魑魅"); got != 0 {
		t.Fatalf("GetNameCount(魑魅) = %d，期望 0（生僻组合不应有伪造计数）", got)
	}
}

func TestGetNameStats(t *testing.T) {
	s := GetNameStats("梓涵")
	if s == nil {
		t.Fatal("GetNameStats 返回 nil")
	}
	if s.Count <= 0 {
		t.Fatalf("Count = %d，期望 > 0", s.Count)
	}
	if s.Rate <= 0 {
		t.Fatalf("Rate = %v，期望 > 0（基于真实母体）", s.Rate)
	}
	if s.Rank <= 0 {
		t.Fatalf("Rank = %d，期望 > 0", s.Rank)
	}
	// 省份维度已按决策移除：JSON 序列化不应含 province 键
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("JSON 序列化失败: %v", err)
	}
	if bytes.Contains(data, []byte("province")) {
		t.Fatalf("JSON 不应包含 province 键，got %s", string(data))
	}
}

func TestGetNameStats_Unknown(t *testing.T) {
	s := GetNameStats("魑魅")
	if s == nil {
		t.Fatal("GetNameStats 返回 nil")
	}
	if s.Count != 0 || s.Rate != 0 || s.Rank != 0 {
		t.Fatalf("未知名字应返回全零统计，got %+v", s)
	}
}

func TestGetTopNames(t *testing.T) {
	list := GetTopNames(10, "all")
	if len(list) != 10 {
		t.Fatalf("GetTopNames(10) 返回 %d 条，期望 10", len(list))
	}
	// 严格降序
	for i := 1; i < len(list); i++ {
		if list[i].Count > list[i-1].Count {
			t.Fatalf("排行非降序：%+v > %+v", list[i], list[i-1])
		}
	}
	for _, s := range list {
		if s.Name == "" || s.Count <= 0 {
			t.Fatalf("排行条目非法：%+v", s)
		}
	}
}

// TestGetTopNames_Kind过滤 榜单种类过滤：单字名/双字名分列或合并，
// 解决双字组合长尾（多数组合仅出现 1-2 次）导致榜单被单字名霸占、双字名不可见。
func TestGetTopNames_Kind过滤(t *testing.T) {
	isSingle := func(s NameStat) bool { return len([]rune(s.Name)) == 1 }

	chars := GetTopNames(10, "char")
	if len(chars) != 10 {
		t.Fatalf("GetTopNames(10, char) 返回 %d 条，期望 10", len(chars))
	}
	for _, s := range chars {
		if !isSingle(s) {
			t.Fatalf("char 过滤混入双字名：%+v", s)
		}
	}

	bigrams := GetTopNames(10, "bigram")
	if len(bigrams) != 10 {
		t.Fatalf("GetTopNames(10, bigram) 返回 %d 条，期望 10", len(bigrams))
	}
	for _, s := range bigrams {
		if isSingle(s) {
			t.Fatalf("bigram 过滤混入单字名：%+v", s)
		}
	}

	all := GetTopNames(10, "all")
	empty := GetTopNames(10, "")
	if len(all) != 10 || len(empty) != 10 {
		t.Fatalf("all/默认过滤应等价于合并榜单，实得 %d / %d 条", len(all), len(empty))
	}
	for i := range all {
		if all[i].Name != empty[i].Name {
			t.Fatalf("all 与空 kind 排序不一致：第 %d 项 %q ≠ %q", i, all[i].Name, empty[i].Name)
		}
	}

	// 语料总量（JSON 路径带 meta）恢复后占比才有意义
	if GetTopNames(1, "char")[0].Rate <= 0 {
		t.Fatalf("榜单条目占比应 > 0（语料总量已装载）")
	}
}
