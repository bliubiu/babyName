package namestat

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"name/internal/domain/hanzi"
)

// packageDir 返回本测试文件所在目录（不依赖运行 CWD）
func packageDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Dir(file)
}

// TestMain 加载人名频率数据库（backend/data/name_frequency.json 等）
func TestMain(m *testing.M) {
	dataDir := filepath.Join(packageDir(), "..", "..", "..", "data")
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
	list := GetTopNames(10)
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