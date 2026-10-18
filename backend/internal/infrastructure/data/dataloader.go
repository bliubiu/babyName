// Package data 统一数据加载
//
// 统筹管理所有物料的加载流程，提供统一的启动加载、热更新和缓存访问。
//
// 物料来源优先级（docs/31 ADR-008）：
//
//	① SQLite 物料库（由 `materials import` 显式导入，归一化与校验已在导入期完成）；
//	② JSON/CSV 源文件（**仅**离线工具与测试走此路径）。
//
// 运行期（有库装载器时）统一以 SQLite 为唯一真源：解析 JSON、字段归一化都发生在
// 导入期，运行期只把库内行读成 typed 结构，既不重复归一化，也不重复校验。
// 必需物料在库中缺失即 fail-fast 并提示执行 `materials import`，不再静默回落
// JSON —— 回落会让「物料修订未生效」变成静默故障。
package data

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"name/internal/domain/classics"
	"name/internal/domain/fate"
	"name/internal/domain/hanzi"
	"name/internal/domain/yijing"
	"name/internal/domain/zodiac"
	"name/internal/infrastructure/logger"
	"name/internal/materials/schema"
)

// Source 物料来源标记
type Source string

const (
	// SourceJSON 从 JSON/CSV 源文件装载（离线工具/测试路径）
	SourceJSON Source = "json"
	// SourceSQLite 从 SQLite 物料库装载（运行期默认）
	SourceSQLite Source = "sqlite"
)

// SourcePolicy 运行时物料来源策略
type SourcePolicy struct {
	// Source 首选来源
	Source Source
	// AllowFallback 首选来源缺料时是否回落 JSON
	//
	// 运行期默认关闭：库缺料即 fail-fast（RequireAll=true 生效），
	// 使「库未导入 / 物料修订未重灌」在启动期暴露，而非以旧 JSON 静默兜底。
	AllowFallback bool
	// RequireAll 必需物料（schema.Definition.Required）缺料即报错，不回落
	RequireAll bool
}

var (
	policyMu sync.RWMutex
	// policy 默认「SQLite 唯一真源 + 必需物料缺料即失败」
	policy = SourcePolicy{Source: SourceSQLite, AllowFallback: false, RequireAll: true}
)

// SetSourcePolicy 设置运行时物料来源策略
func SetSourcePolicy(p SourcePolicy) {
	policyMu.Lock()
	defer policyMu.Unlock()
	policy = p
}

// GetSourcePolicy 读取当前物料来源策略
func GetSourcePolicy() SourcePolicy {
	policyMu.RLock()
	defer policyMu.RUnlock()
	return policy
}

// loadedElsewhere 不参与统一装载报告的物料：它们由各自领域消费方从库装载，
// fail-fast 由消费方自证（见 sqlite.loadedElsewhereMaterials）。
var loadedElsewhere = map[string]bool{
	schema.CuratedMaterial:  true,
	schema.ClassicsMaterial: true,
}

// Init 统一装载全部物料（从 JSON/CSV 源文件）
//
// 等价于 InitWithDB(dataDir, nil)，用于离线工具与测试等无库场景。
func Init(dataDir string) error {
	return initFromJSON(dataDir)
}

// InitWithDB 统一装载全部物料，优先从 SQLite 物料库读取
//
// dbLoader 非 nil 且策略首选 SQLite 时走严格库路径：必需物料齐备才启动，
// 否则返回错误引导执行 `materials import`；不再逐物料回落 JSON。
// dbLoader 为 nil（或策略显式指定 JSON）时走纯 JSON 路径。
func InitWithDB(dataDir string, dbLoader DBLoader) error {
	p := GetSourcePolicy()
	if dbLoader == nil || p.Source != SourceSQLite {
		return initFromJSON(dataDir)
	}

	loadedFromDB := make(map[string]bool)
	for _, name := range dbLoader.LoadMaterialsFromDB() {
		loadedFromDB[name] = true
	}

	// 必需物料齐备性校验：缺料在启动期暴露，而不是以旧 JSON 静默兜底
	if err := verifyRequiredMaterials(loadedFromDB); err != nil {
		return err
	}

	// 经典语料：由消费方自装载（不入统一报告），库缺即降级告警而非中断
	if cl, ok := dbLoader.(ClassicsLoader); ok {
		if err := cl.LoadClassicsFromDB(); err != nil {
			if p.RequireAll {
				return err
			}
			logger.Warn("经典语料从库装载失败，诗词取名能力降级", logger.ErrField(err))
		}
	} else {
		logger.Warn("存储实现未提供经典语料装载能力，诗词取名能力不可用")
	}

	logger.Info("物料装载完成",
		logger.String("来源", string(p.Source)),
		logger.String("库内物料", strings.Join(sortedKeys(loadedFromDB), ",")))
	return nil
}

// verifyRequiredMaterials 校验必需物料是否齐备
//
// 只校验 schema.Definition.Required=true 的物料：可选物料（频率、白名单、
// 合规红线、策展名、经典语料）缺失属可降级场景，由各消费方自行告警。
func verifyRequiredMaterials(loaded map[string]bool) error {
	var missing []string
	for _, def := range schema.Definitions {
		if !def.Required || loadedElsewhere[def.Name] || loaded[def.Name] {
			continue
		}
		missing = append(missing, def.Name)
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("物料库缺少必需物料 %d 项: %s（请执行 materials import 导入）",
		len(missing), strings.Join(missing, ", "))
}

// initFromJSON 从 JSON/CSV 源文件装载全部物料（离线工具与测试路径）
//
// 各步骤的降级口径与运行期保持一致：只有字库/康熙笔画/易经是硬依赖，
// 其余（生肖、门禁、禁忌、白名单、合规红线、频率、语料）缺失仅告警降级。
func initFromJSON(dataDir string) error {
	// 1. 康熙笔画（河图数理与易经卦象的笔画口径）
	//    依赖顺序：必须先于 namer —— namer 装载会把康熙笔画注入字库
	if err := hanzi.LoadKangxiStrokesFromCSV(dataDir); err != nil {
		return fmt.Errorf("加载康熙笔画表失败: %w", err)
	}

	// 2. 《通用规范汉字表》8105 字（运行时唯一汉字数据源）
	//
	// namer.json 含五行校正、起名分类标注与并入的精选偏旁分组（charGroups）。
	// 历史遗留的 hanzi.json / standard_chars.json 已降级为 export_namer 的
	// 离线生成原料，不再于运行时加载。
	if err := hanzi.LoadNamerFromJSON(dataDir); err != nil {
		return fmt.Errorf("加载 namer 数据失败: %w", err)
	}

	// 3. 易经六十四卦
	if err := yijing.LoadFromJSON(dataDir); err != nil {
		// 与语料同理：易经缺失只影响卦象解读，不应拖垮字库
		logger.Warn("加载易经数据失败，卦象解读能力降级", logger.ErrField(err))
	}

	// 4. 古典文学语料（诗经/楚辞/诗词/蒙学）
	//
	// 缺失**不阻断**：语料缺失只影响诗词取名，属于可降级能力；
	// 若在此返回错误，会把前面已装载的字库/易经一并作废 ——
	// 这是「语料缺失拖垮全局」的反模式。
	if err := classics.LoadFromJSON(dataDir); err != nil {
		logger.Warn("加载古典文学语料失败，诗词取名能力降级", logger.ErrField(err))
	}

	// 5. 生肖
	if err := zodiac.LoadFromJSON(dataDir); err != nil {
		logger.Warn("加载生肖数据失败，生肖合名能力降级", logger.ErrField(err))
	}

	// 6. 命名质量门禁字表（候选池阶段硬剔除）
	if err := fate.LoadNamingQualityFromJSON(dataDir); err != nil {
		logger.Warn("加载命名质量门禁字表失败，候选池门禁降级", logger.ErrField(err))
	}

	// 7. 禁忌双字组合（与硬编码常量合并，参与 IsBadCombo 判定）
	if err := fate.LoadForbiddenCombosFromJSON(dataDir); err != nil {
		logger.Warn("加载禁忌双字组合表失败，组合门禁降级", logger.ErrField(err))
	}

	// 7.5 显式白名单（docs/33 门禁层 M1）：豁免黑名单（白名单优先）+ 祈福组合。
	//     可选物料，缺失时降级为内置默认（仅祈福组合），不阻断。
	if err := fate.LoadAllowlistFromJSON(dataDir); err != nil {
		logger.Warn("加载白名单表失败，白名单豁免降级为内置默认", logger.ErrField(err))
	}

	// 7.6 L5 合规红线（docs/33 门禁层 M2）：机关单位/争议地名/负面历史人物。
	//     可选物料，缺失时降级为内置兜底（机关单位），不阻断。
	if err := fate.LoadComplianceRedlineFromJSON(dataDir); err != nil {
		logger.Warn("加载合规红线表失败，合规红线降级为内置兜底", logger.ErrField(err))
	}

	// 8. 人名频率库（Tier≥2）
	if err := hanzi.LoadFrequencyDB(dataDir); err != nil {
		logger.Warn("加载人名频率失败，频率评分将降级", logger.ErrField(err))
	}

	// 9. 精选策展名（收藏自学习数据）由 name 包与 handlers 侧单独加载，
	//    此处不重复装载，保持既有职责边界。

	logger.Info("物料装载完成", logger.String("来源", string(SourceJSON)))
	return nil
}

// sortedKeys 返回 map 的有序键，保证日志输出稳定可比对
func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// DBLoader 抽象 SQLite 物料装载能力，避免 data 包直接依赖具体存储实现
type DBLoader interface {
	// LoadMaterialsFromDB 从库装载全部可用物料，返回已装载的物料名清单
	LoadMaterialsFromDB() []string
}

// ClassicsLoader 经典语料的库装载能力（可选接口）
//
// 经典语料不入统一装载报告（LoadMaterialsFromDB 负责的物料集），由消费方
// 单独自证：库有则从物化表装载，库缺则报错/降级，不与统一报告互相牵连。
type ClassicsLoader interface {
	LoadClassicsFromDB() error
}

// ReloadAll 热更新所有数据
//
// 变更点（docs/31 ADR-006）：不再无条件全量重新 Init，
// 而是按当前来源策略重跑装载 —— 库路径只重建内存索引与库内映射。
func ReloadAll(dataDir string, dbLoader DBLoader) error {
	return InitWithDB(dataDir, dbLoader)
}