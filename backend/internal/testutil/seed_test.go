package testutil_test

import (
	"runtime"
	"testing"

	"name/internal/domain/fate"
	"name/internal/domain/hanzi"
	"name/internal/infrastructure/database/sqlite"
	seedkit "name/internal/testutil/seed"
)

// TestInitFromSeed 单库装配回归：只读种子库必须能灌满全部内存索引。
//
// 这是「CI 零 JSON」的地基：任何一项为空都意味着门禁在拿空数据跑出无意义
// 的绿，必须在装配期暴露而不是等 golden 快照静默漂移。
func TestInitFromSeed(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	path := seedkit.Path(file)
	if path == "" {
		t.Fatal("未找到 data/namer.db 种子库（请先执行 go run ./tools/build_seed）")
	}
	t.Logf("种子库: %s", path)

	cleanup, err := seedkit.Init(file)
	if err != nil {
		t.Fatalf("单库装配失败: %v", err)
	}
	defer cleanup()

	if n := len(hanzi.HanziData); n < 8000 {
		t.Errorf("字库未装载: hanzi_data 内存索引 %d 行，期望 ≥8000", n)
	}
	if n := hanzi.KangxiStrokesCount(); n == 0 {
		t.Error("康熙笔画未装载")
	}
	if n := fate.ForbiddenComboCount(); n == 0 {
		t.Error("禁忌双字组合未装载")
	}
	if n := fate.NonNamingCharCount(); n == 0 {
		t.Error("命名质量门禁字表未装载")
	}
}

// TestNewMaterialLoader_只读不落盘 只读装载器不得改写被 git 跟踪的种子库
func TestNewMaterialLoader_只读不落盘(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	path := seedkit.Path(file)
	if path == "" {
		t.Fatal("未找到 data/namer.db 种子库")
	}
	loader, err := sqlite.NewMaterialLoader(path)
	if err != nil {
		t.Fatalf("打开只读种子库失败: %v", err)
	}
	defer func() { _ = loader.Close() }()
	if got := loader.LoadMaterialsFromDB(); len(got) == 0 {
		t.Error("只读装载器未装载任何物料")
	}
}
