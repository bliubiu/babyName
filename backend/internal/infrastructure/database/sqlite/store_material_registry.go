package sqlite

import "name/internal/materials/schema"

// 本文件是「物料导入适配器」的唯一注册点（docs/31 §11.9 第 3 项 P0 抽象）。
//
// 背景：importOne 原先用 switch 硬编码「物料名 → importXxx 方法」，新增物料
// 要同时改 switch、schema.Definitions、迁移 DDL 三处，漏改一处即静默跳过。
// 这里把「物料定义」与「导入实现」收敛到一张注册表：init 时按
// schema.Definitions 逐一登记，任何未登记的物料直接 panic（编程错误，必须
// 在编译/启动期暴露），已接入统一导入器的物料与走既有路径的 legacy 物料
// 显式区分。
//
// 未接入统一导入器的物料（classics/curated_names）登记为 legacyMarker，
// 保持既有「跳过并给出原因」语义，但不视为错误。

// Material 单个物料的导入能力抽象
//
// Definition 返回物料定义，供注册表与 schema.Definitions 对齐校验；
// Import 执行「源文件 → 校验 → 写库」，复用 runImport 事务骨架。
type Material interface {
	Definition() schema.Definition
	Import(s *Store, src string, res ImportResult, opt ImportOptions) ImportResult
}

// materialFunc 把与 importXxx 同签名的函数适配为 Material
//
// 用方法表达式（(*Store).importKangxi）登记，避免再写一层结构体样板，
// 也让「注册表条目 ↔ importXxx 实现」在编译期强绑定。
type materialFunc struct {
	name string
	fn   func(s *Store, def schema.Definition, src string, res ImportResult, opt ImportOptions) ImportResult
}

// Definition 按注册名从 schema.Registry 取物料定义
func (m materialFunc) Definition() schema.Definition { return schema.Registry[m.name] }

// Import 注入物料定义后转调底层 importXxx 方法
func (m materialFunc) Import(s *Store, src string, res ImportResult, opt ImportOptions) ImportResult {
	return m.fn(s, m.Definition(), src, res, opt)
}

// legacyImportReason legacy 物料跳过时的统一原因文案
const legacyImportReason = "该物料走既有导入路径（classics/curated），本阶段未接入统一导入器"

// legacyMarker 标记「尚未接入统一导入器」的物料
//
// 这类物料由 store_classics.go 等既有路径导入，统一导入器遇到时跳过并给出
// 原因，不视为错误。
type legacyMarker struct {
	name   string
	reason string
}

// Definition 返回物料定义
func (m legacyMarker) Definition() schema.Definition { return schema.Registry[m.name] }

// Import 直接跳过并给出原因
func (m legacyMarker) Import(_ *Store, _ string, res ImportResult, _ ImportOptions) ImportResult {
	res.Skipped = true
	res.Reason = m.reason
	return res
}

// materialRegistry 物料导入适配器注册表，键为物料名
var materialRegistry = map[string]Material{}

func init() {
	// 已接入统一导入器的物料 → 对应 importXxx 方法表达式
	impls := map[string]func(*Store, schema.Definition, string, ImportResult, ImportOptions) ImportResult{
		schema.KangxiMaterial:            (*Store).importKangxi,
		schema.NamerMaterial:             (*Store).importNamer,
		schema.WordMaterial:              (*Store).importWords,
		schema.FrequencyMaterial:         (*Store).importCharFreq,
		schema.BigramMaterial:            (*Store).importBigram,
		schema.GateMaterial:              (*Store).importGate,
		schema.ForbiddenMaterial:         (*Store).importForbidden,
		schema.AllowlistMaterial:         (*Store).importAllowlist,
		schema.ComplianceRedlineMaterial: (*Store).importComplianceRedline,
		schema.YijingMaterial:            (*Store).importYijing,
		schema.ZodiacMaterial:            (*Store).importZodiac,
	}
	// 走既有路径、暂不接入统一导入器的物料
	legacy := map[string]bool{
		schema.ClassicsMaterial: true,
		schema.CuratedMaterial:  true,
	}

	for _, def := range schema.Definitions {
		switch {
		case legacy[def.Name]:
			materialRegistry[def.Name] = legacyMarker{name: def.Name, reason: legacyImportReason}
		default:
			fn, ok := impls[def.Name]
			if !ok {
				panic("物料 " + def.Name + " 既未登记导入实现，也未标记为 legacy：请补齐 store_material_registry.go")
			}
			materialRegistry[def.Name] = materialFunc{name: def.Name, fn: fn}
		}
	}
}
