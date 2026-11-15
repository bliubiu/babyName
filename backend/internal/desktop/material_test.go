package desktop

import (
	"errors"
	"testing"
)

// fakeMaterialManager 物料管理替身
type fakeMaterialManager struct {
	states      []MaterialState
	statesErr   error
	report      *MaterialImportReport
	reimportErr error

	reimportCalled bool
}

func (f *fakeMaterialManager) MaterialStates() ([]MaterialState, error) {
	return f.states, f.statesErr
}

func (f *fakeMaterialManager) Reimport() (*MaterialImportReport, error) {
	f.reimportCalled = true
	return f.report, f.reimportErr
}

// newMaterialTestApp 构造仅挂物料源的 App
func newMaterialTestApp(m MaterialManager) *App {
	return &App{materials: m}
}

// --- GetMaterialStates ---

func TestGetMaterialStates_返回物料状态列表(t *testing.T) {
	want := []MaterialState{
		{Material: "hanzi", Version: "v1", Checksum: "abc", RowCount: 8105, ImportedAt: "2026-11-13 10:00:00"},
		{Material: "yijing", Version: "v2", Checksum: "def", RowCount: 64, ImportedAt: "2026-11-13 10:00:00"},
	}
	app := newMaterialTestApp(&fakeMaterialManager{states: want})

	got, err := app.GetMaterialStates()
	if err != nil {
		t.Fatalf("GetMaterialStates() err = %v, want nil", err)
	}
	if len(got) != 2 || got[0].Material != "hanzi" || got[0].RowCount != 8105 {
		t.Fatalf("GetMaterialStates() = %+v, want %+v", got, want)
	}
}

func TestGetMaterialStates_源错误透传用户文案(t *testing.T) {
	app := newMaterialTestApp(&fakeMaterialManager{statesErr: errors.New("查询物料状态失败: db broken")})

	_, err := app.GetMaterialStates()
	if err == nil {
		t.Fatal("GetMaterialStates() err = nil, want error")
	}
	if err.Error() != "查询物料状态失败: db broken" {
		t.Fatalf("err = %q, want 保留原始文案", err.Error())
	}
}

func TestGetMaterialStates_内存模式返回空列表(t *testing.T) {
	app := newMaterialTestApp(nil)

	got, err := app.GetMaterialStates()
	if err != nil {
		t.Fatalf("GetMaterialStates() err = %v, want nil（内存模式静默空列表）", err)
	}
	if len(got) != 0 {
		t.Fatalf("GetMaterialStates() = %+v, want 空", got)
	}
}

// --- ImportMaterials ---

func TestImportMaterials_委托重导并返回报告(t *testing.T) {
	report := &MaterialImportReport{
		Items:   []MaterialImportItem{{Material: "hanzi", Version: "v1", Rows: 8105}},
		Written: 1,
	}
	fake := &fakeMaterialManager{report: report}
	app := newMaterialTestApp(fake)

	got, err := app.ImportMaterials()
	if err != nil {
		t.Fatalf("ImportMaterials() err = %v, want nil", err)
	}
	if !fake.reimportCalled {
		t.Fatal("ImportMaterials() 未调用物料源 Reimport")
	}
	if got.Written != 1 || len(got.Items) != 1 || got.Items[0].Material != "hanzi" {
		t.Fatalf("ImportMaterials() = %+v, want %+v", got, report)
	}
}

func TestImportMaterials_失败文案可行动(t *testing.T) {
	app := newMaterialTestApp(&fakeMaterialManager{reimportErr: errors.New("物料导入失败: namer.json 脏数据")})

	_, err := app.ImportMaterials()
	if err == nil {
		t.Fatal("ImportMaterials() err = nil, want error")
	}
	if err.Error() != "物料导入失败: namer.json 脏数据" {
		t.Fatalf("err = %q, want 保留原始文案", err.Error())
	}
}

func TestImportMaterials_内存模式报错提示(t *testing.T) {
	app := newMaterialTestApp(nil)

	_, err := app.ImportMaterials()
	if err == nil {
		t.Fatal("ImportMaterials() err = nil, want error")
	}
	if err.Error() != "当前未使用 SQLite 存储，物料管理不可用" {
		t.Fatalf("err = %q, want 内存模式提示", err.Error())
	}
}
