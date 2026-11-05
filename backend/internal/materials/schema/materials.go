package schema

import "sort"

// 物料清单：名称 → 定义（docs/31 §八 L1 物料分类）
//
// Name 同时用作 materials_state 表的 material 列值（导入指纹登记）
const (
	// NamerMaterial 通用规范汉字表（namer.json 8105 字 + 偏旁分组）
	NamerMaterial = "namer"
	// KangxiMaterial 康熙笔画（raw/kangxi-strokecount.csv）
	KangxiMaterial = "kangxi"
	// FrequencyMaterial 单字人名频率（name_frequency.json）
	FrequencyMaterial = "name_frequency"
	// BigramMaterial 双字人名频率（name_bigram_frequency.json）
	BigramMaterial = "name_bigram"
	// GateMaterial 命名质量门禁字表（naming_quality.json）
	GateMaterial = "naming_gate"
	// ForbiddenMaterial 禁忌双字组合（forbidden_combos.json）
	ForbiddenMaterial = "forbidden_combos"
	// AllowlistMaterial 显式白名单（allowlist.json）：豁免黑名单（白名单优先）+ 祈福组合
	AllowlistMaterial = "allowlist"
	// ComplianceRedlineMaterial L5 合规红线（compliance_redline.json）：机关单位/争议地名/负面历史人物
	ComplianceRedlineMaterial = "compliance_redline"
	// YijingMaterial 易经六十四卦（yijing.json）
	YijingMaterial = "yijing"
	// ZodiacMaterial 十二生肖（zodiac.json）
	ZodiacMaterial = "zodiac"
	// CuratedMaterial 精选策展名（curated_names.json）
	CuratedMaterial = "curated_names"
	// ClassicsMaterial 古典文学语料（诗经/楚辞/诗词/蒙学，20+ 文件）
	ClassicsMaterial = "classics"
)

// Definition 物料定义：源文件、依赖与写入目标表
type Definition struct {
	// Name 物料名（materials_state 表的 material 列）
	Name string
	// Source 相对 dataDir 的主源文件路径
	//
	// 单文件物料以它为准；多文件物料（当前仅 classics）仍保留主文件作为
	// 哨兵/报告锚点，实际清单以 Files 为准。
	Source string
	// Files 多文件物料的源文件清单（相对 dataDir）
	//
	// 空表示单文件物料（以 Source 为准）。非空时该清单是部署核对与导入的
	// 唯一权威：`materials manifest` 由此输出，classics 导入由此派生文件名。
	// 与 Source 同时存在时，Source 必须也出现在 Files 中（由守护测试保证）。
	Files []string
	// DependsOn 依赖的物料名，导入顺序据此拓扑排列
	DependsOn []string
	// Tables 该物料写入的表（文档与守护测试用）
	//
	// 约定：首项为主表（必然非空），其余为辅表（允许合理为空，
	// 如 namer 的 char_groups 在源无分组时为空）。materialUnchanged
	// 的短路判定只校验主表（docs/34 阶段 B3）。
	Tables []string
	// Required 源文件缺失时是否报错（true=缺物料即失败，false=允许缺失跳过）
	Required bool
}

// Definitions 全部物料的集中注册表
//
// 新增物料只需在此登记一行，导入器与校验器会自动纳入（docs/31 ADR-001）
var Definitions = []Definition{
	{
		Name: KangxiMaterial, Source: "raw/kangxi-strokecount.csv",
		DependsOn: []string{}, Tables: []string{"kangxi_strokes"}, Required: true,
	},
	{
		Name: NamerMaterial, Source: "namer.json",
		DependsOn: []string{KangxiMaterial}, Tables: []string{"hanzi_data", "char_groups"}, Required: true,
	},
	{
		Name: FrequencyMaterial, Source: "name_frequency.json",
		DependsOn: []string{NamerMaterial}, Tables: []string{"name_frequency"}, Required: false,
	},
	{
		Name: BigramMaterial, Source: "name_bigram_frequency.json",
		DependsOn: []string{NamerMaterial}, Tables: []string{"name_bigram"}, Required: false,
	},
	{
		Name: GateMaterial, Source: "naming_quality.json",
		DependsOn: []string{NamerMaterial}, Tables: []string{"naming_gate"}, Required: true,
	},
	{
		Name: ForbiddenMaterial, Source: "forbidden_combos.json",
		DependsOn: []string{NamerMaterial}, Tables: []string{"forbidden_combos"}, Required: true,
	},
	{
		Name: AllowlistMaterial, Source: "allowlist.json",
		DependsOn: []string{}, Tables: []string{"allowlist"}, Required: false,
	},
	{
		Name: ComplianceRedlineMaterial, Source: "compliance_redline.json",
		DependsOn: []string{}, Tables: []string{"compliance_redline"}, Required: false,
	},
	{
		Name: YijingMaterial, Source: "yijing.json",
		DependsOn: []string{}, Tables: []string{"yijing_hexagrams"}, Required: true,
	},
	{
		Name: ZodiacMaterial, Source: "zodiac.json",
		DependsOn: []string{}, Tables: []string{"zodiac"}, Required: true,
	},
	{
		Name: CuratedMaterial, Source: "curated_names.json",
		DependsOn: []string{NamerMaterial}, Tables: []string{"curated_names"}, Required: false,
	},
	{
		Name: ClassicsMaterial, Source: "shijing.json", Files: ClassicsFiles,
		DependsOn: []string{NamerMaterial},
		// 表序即导入事务的清表序（runImport 按 Tables 逐表 DELETE）：
		// 子表 classics_sections 经外键引用 classics_books（docs/43 P1-5 启用
		// foreign_keys），必须先删子表再删父表，否则重灌时 FK 违规。
		Tables: []string{
			"classics_sections", "classics_books",
			"classics_extracts", "classics_poem_entries",
		},
		Required: false,
	},
}

// Registry 按名索引的物料定义
var Registry = func() map[string]Definition {
	m := make(map[string]Definition, len(Definitions))
	for _, d := range Definitions {
		m[d.Name] = d
	}
	return m
}()

// Names 返回全部物料名（字典序，保证输出稳定）
func Names() []string {
	names := make([]string, 0, len(Definitions))
	for _, d := range Definitions {
		names = append(names, d.Name)
	}
	sort.Strings(names)
	return names
}

// TopoSort 按 DependsOn 做拓扑排序，返回导入顺序
//
// 有环时报错（防止物料互相依赖导致导入顺序不确定）
func TopoSort(defs []Definition) ([]string, error) {
	byName := make(map[string]Definition, len(defs))
	for _, d := range defs {
		byName[d.Name] = d
	}

	const (
		visiting = 1
		done     = 2
	)
	state := make(map[string]int, len(defs))
	var order []string

	var visit func(name string, path []string) error
	visit = func(name string, path []string) error {
		switch state[name] {
		case done:
			return nil
		case visiting:
			return &CycleError{Cycle: append(append([]string{}, path...), name)}
		}
		d, ok := byName[name]
		if !ok {
			return &UnknownMaterialError{Name: name, Referrer: pathOrEmpty(path)}
		}
		state[name] = visiting
		for _, dep := range d.DependsOn {
			if err := visit(dep, append(path, name)); err != nil {
				return err
			}
		}
		state[name] = done
		order = append(order, name)
		return nil
	}

	for _, d := range defs {
		if err := visit(d.Name, nil); err != nil {
			return nil, err
		}
	}
	return order, nil
}

func pathOrEmpty(p []string) string {
	if len(p) == 0 {
		return ""
	}
	return p[len(p)-1]
}

// CycleError 物料依赖成环
type CycleError struct{ Cycle []string }

func (e *CycleError) Error() string {
	return "物料依赖成环: " + joinDash(e.Cycle)
}

// UnknownMaterialError 依赖了未注册的物料
type UnknownMaterialError struct {
	Name     string
	Referrer string
}

func (e *UnknownMaterialError) Error() string {
	if e.Referrer != "" {
		return "物料 " + e.Referrer + " 依赖了未注册的物料 " + e.Name
	}
	return "物料 " + e.Name + " 未注册"
}

func joinDash(in []string) string {
	out := ""
	for i, s := range in {
		if i > 0 {
			out += " → "
		}
		out += s
	}
	return out
}
