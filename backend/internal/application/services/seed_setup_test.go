package services

// services 包端到端测试的单库装配（ADR-010「A + 单库」）
//
// 两个 e2e 文件原本各自 data.Init(dataDir) 读 JSON，且从 curated_names.json
// 注入策展名。改为共用一份只读种子库：物料走 data.InitWithDB（与生产同路径），
// 策展名从 curated_names 表读。这样 CI 不需要任何 JSON 源，golden 快照也不
// 再随入库物料子集漂移。

import (
	"fmt"
	"sync"
	"testing"

	"name/internal/infrastructure/data"
	"name/internal/infrastructure/database/sqlite"
	seedkit "name/internal/testutil/seed"
)

var (
	e2eSeedOnce   sync.Once
	e2eSeedLoader *sqlite.MaterialLoader
	e2eSeedErr    error
)

// e2eSeedDB 装配一次只读种子库，后续用例复用同一份内存索引
func e2eSeedDB(t *testing.T, callerFile string) *sqlite.MaterialLoader {
	t.Helper()
	e2eSeedOnce.Do(func() {
		dbPath := seedkit.Path(callerFile)
		if dbPath == "" {
			e2eSeedErr = fmt.Errorf("未找到种子库 data/namer.db（从 %s 向上探测失败）", callerFile)
			return
		}
		loader, err := sqlite.NewMaterialLoader(dbPath)
		if err != nil {
			e2eSeedErr = err
			return
		}
		if err := data.InitWithDB("", loader); err != nil {
			_ = loader.Close()
			e2eSeedErr = err
			return
		}
		e2eSeedLoader = loader
	})
	if e2eSeedErr != nil {
		t.Fatalf("单库物料装配失败: %v", e2eSeedErr)
	}
	return e2eSeedLoader
}

// e2eCuratedNames 从种子库的 curated_names 表读策展名（替代 curated_names.json）
func e2eCuratedNames(t *testing.T, loader *sqlite.MaterialLoader) []string {
	t.Helper()
	entries, err := loader.LoadAllCuratedNames()
	if err != nil {
		t.Fatalf("从种子库读策展名失败: %v", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.Name != "" {
			names = append(names, e.Name)
		}
	}
	return names
}
