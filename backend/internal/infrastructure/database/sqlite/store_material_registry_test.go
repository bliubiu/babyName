package sqlite

import (
	"testing"

	"name/internal/materials/schema"
)

// store_material_registry_test.go — 物料导入注册表守护（docs/31 §11.9 P0）
//
// importOne 改为按注册表多态分发后，注册表与 schema.Definitions 必须严格
// 一一对应：漏登记会让物料静默无法导入，幽灵登记会掩盖已删除的物料。

// TestMaterialRegistry_与物料定义一一对应 注册表与会话必须是双射
func TestMaterialRegistry_与物料定义一一对应(t *testing.T) {
	defs := make(map[string]bool, len(schema.Definitions))
	for _, d := range schema.Definitions {
		defs[d.Name] = true
		if _, ok := materialRegistry[d.Name]; !ok {
			t.Errorf("物料 %s 未登记导入实现（importOne 将无法导入）", d.Name)
		}
	}
	for name := range materialRegistry {
		if !defs[name] {
			t.Errorf("注册表条目 %s 在 schema.Definitions 中不存在（幽灵登记）", name)
		}
	}
}

// TestMaterialRegistry_Definition与注册名一致 每个条目的定义名必须等于其注册键
func TestMaterialRegistry_Definition与注册名一致(t *testing.T) {
	for name, m := range materialRegistry {
		if got := m.Definition().Name; got != name {
			t.Errorf("注册键 %s 与 Definition().Name %s 不一致", name, got)
		}
	}
}

// TestMaterialRegistry_已接入物料不使用legacy 已接入统一导入器的物料不得登记为 legacy
func TestMaterialRegistry_已接入物料不使用legacy(t *testing.T) {
	integrated := []string{
		schema.KangxiMaterial, schema.NamerMaterial,
		schema.FrequencyMaterial, schema.BigramMaterial, schema.GateMaterial,
		schema.ForbiddenMaterial, schema.AllowlistMaterial,
		schema.ComplianceRedlineMaterial, schema.YijingMaterial, schema.ZodiacMaterial,
		schema.CuratedMaterial,
	}
	for _, name := range integrated {
		m, ok := materialRegistry[name]
		if !ok {
			t.Fatalf("物料 %s 未登记", name)
		}
		if _, isLegacy := m.(legacyMarker); isLegacy {
			t.Errorf("物料 %s 已接入统一导入器，不得登记为 legacy", name)
		}
	}
}

// TestMaterialRegistry_Legacy物料跳过并给出原因 legacy 物料必须跳过且原因非空
//
// curated 已接入统一导入器（importCurated），此处仅剩 classics 待阶段 3 接入。
func TestMaterialRegistry_Legacy物料跳过并给出原因(t *testing.T) {
	for _, name := range []string{schema.ClassicsMaterial} {
		m, ok := materialRegistry[name]
		if !ok {
			t.Fatalf("legacy 物料 %s 未登记", name)
		}
		res := m.Import(nil, "不存在的源", ImportResult{Material: name}, ImportOptions{})
		if !res.Skipped {
			t.Errorf("legacy 物料 %s 应跳过，实得 skipped=%v err=%q", name, res.Skipped, res.Err)
		}
		if res.Reason == "" {
			t.Errorf("legacy 物料 %s 跳过时必须给出原因", name)
		}
	}
}
