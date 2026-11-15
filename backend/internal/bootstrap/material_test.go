package bootstrap

import (
	"errors"
	"path/filepath"
	"testing"

	"name/internal/infrastructure/database/sqlite"
	"name/internal/testutil"
)

// fakeMaterialStore 物料只读 + 导入的替身（模拟 *sqlite.Store 的物料面）
type fakeMaterialStore struct {
	states    []sqlite.MaterialState
	statesErr error
	report    *sqlite.ImportReport
	importErr error

	gotOptions *sqlite.ImportOptions
}

func (f *fakeMaterialStore) MaterialStates() ([]sqlite.MaterialState, error) {
	return f.states, f.statesErr
}

func (f *fakeMaterialStore) Import(opt sqlite.ImportOptions) (*sqlite.ImportReport, error) {
	f.gotOptions = &opt
	return f.report, f.importErr
}

// --- MaterialStates 适配 ---

func TestMaterialStates_转换为桌面端结构(t *testing.T) {
	fake := &fakeMaterialStore{states: []sqlite.MaterialState{
		{Material: "hanzi", Version: "v1", Checksum: "abc", RowCount: 8105, ImportedAt: "2026-11-13 10:00:00"},
	}}
	mgr := newMaterialManager(fake, fake, "D:/data")

	got, err := mgr.MaterialStates()
	if err != nil {
		t.Fatalf("MaterialStates() err = %v, want nil", err)
	}
	if len(got) != 1 || got[0].Material != "hanzi" || got[0].RowCount != 8105 || got[0].ImportedAt != "2026-11-13 10:00:00" {
		t.Fatalf("MaterialStates() = %+v, 字段映射不符", got)
	}
}

func TestMaterialStates_透传错误(t *testing.T) {
	fake := &fakeMaterialStore{statesErr: errors.New("查询物料状态失败")}
	mgr := newMaterialManager(fake, fake, "D:/data")

	if _, err := mgr.MaterialStates(); err == nil || err.Error() != "查询物料状态失败" {
		t.Fatalf("MaterialStates() err = %v, want 透传", err)
	}
}

// --- Reimport 适配 ---

func TestReimport_传递物料目录并转换报告(t *testing.T) {
	fake := &fakeMaterialStore{report: &sqlite.ImportReport{
		Results: []sqlite.ImportResult{
			{Material: "hanzi", Version: "v1", Rows: 8105},
			{Material: "chuci", Err: "校验失败"},
		},
		Written: 1,
		Failed:  1,
		Elapsed: "1.5s",
	}}
	mgr := newMaterialManager(fake, fake, "D:/data")

	got, err := mgr.Reimport()
	if err != nil {
		t.Fatalf("Reimport() err = %v, want nil", err)
	}
	if fake.gotOptions == nil || fake.gotOptions.DataDir != "D:/data" {
		t.Fatalf("Reimport() 未按装配目录导入: %+v", fake.gotOptions)
	}
	if got.Written != 1 || got.Failed != 1 || got.Elapsed != "1.5s" || len(got.Items) != 2 {
		t.Fatalf("Reimport() 报告映射不符: %+v", got)
	}
	if got.Items[0].Material != "hanzi" || got.Items[1].Err != "校验失败" {
		t.Fatalf("Reimport() 条目映射不符: %+v", got.Items)
	}
}

func TestReimport_透传错误(t *testing.T) {
	fake := &fakeMaterialStore{importErr: errors.New("物料导入失败: 脏数据")}
	mgr := newMaterialManager(fake, fake, "D:/data")

	if _, err := mgr.Reimport(); err == nil || err.Error() != "物料导入失败: 脏数据" {
		t.Fatalf("Reimport() err = %v, want 透传", err)
	}
}

// --- 装配接线（经 Run 全链路：存储类型 → 物料管理器 → 绑定行为） ---

func TestRun_内存模式物料管理降级(t *testing.T) {
	dataDir := testutil.MustDataDir(t, callerFile)

	res, err := Run(Options{DataDir: dataDir})
	if err != nil {
		t.Fatalf("内存模式装配失败: %v", err)
	}
	defer func() { _ = res.Close() }()

	// 内存存储无物料库：状态返回空列表而非错误
	states, err := res.App.GetMaterialStates()
	if err != nil {
		t.Fatalf("内存模式物料状态应返回空列表而非错误: %v", err)
	}
	if len(states) != 0 {
		t.Fatalf("内存模式物料状态应为空，实际: %d 条", len(states))
	}
	// 重导必须给出明确提示，而非静默空转
	_, err = res.App.ImportMaterials()
	if err == nil || err.Error() != "当前未使用 SQLite 存储，物料管理不可用" {
		t.Fatalf("内存模式重导应提示不可用，实际: %v", err)
	}
}

func TestRun_SQLite模式物料状态可查(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	fixture := writeMinimalMaterials(t)

	res, err := Run(Options{DBPath: dbPath, DataDir: fixture, SeedFromJSON: true})
	if err != nil {
		t.Fatalf("SQLite 装配失败: %v", err)
	}
	defer func() { _ = res.Close() }()

	states, err := res.App.GetMaterialStates()
	if err != nil {
		t.Fatalf("物料状态查询失败: %v", err)
	}
	if len(states) == 0 {
		t.Fatal("播种后物料状态应为非空")
	}
	for _, s := range states {
		if s.Material == "" || s.RowCount <= 0 {
			t.Fatalf("物料状态字段异常: %+v", s)
		}
	}
}
