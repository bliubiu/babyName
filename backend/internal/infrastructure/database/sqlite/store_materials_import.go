package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"name/internal/domain/fate"
	"name/internal/domain/hanzi"
	"name/internal/domain/name"
	"name/internal/domain/yijing"
	"name/internal/domain/zodiac"
	"name/internal/infrastructure/logger"
	"name/internal/materials/checksum"
	"name/internal/materials/normalize"
	"name/internal/materials/schema"
	"name/internal/materials/validate"
)

// 本文件实现 `materials import`：把 JSON/CSV 源物料归一化、校验后
// 单事务写入 SQLite，并把物料指纹登记到 materials_state（docs/31 ADR-008）。
//
// 写入语义（ADR-011）：一律 DELETE + INSERT 全量重建置于单事务内，
// 保证「同输入多次导入结果等价」且物料修订、删字都能传导。
// 未变更的物料按指纹短路跳过，避免数十 MB 物料的无谓重灌。

// maxImportErrors 单次导入最多收集的校验错误条数（超出即中止并报错）
const maxImportErrors = 20

// ImportOptions 导入参数
type ImportOptions struct {
	// DataDir 物料源目录（含 namer.json / raw/kangxi-strokecount.csv 等）
	DataDir string
	// Materials 只导入指定物料；为空表示全部
	Materials []string
	// Lenient 校验不通过时仅记录不阻断（数据清洗期取证用，ADR-009）
	Lenient bool
	// DryRun 只校验与统计，不写库
	DryRun bool

	// deps 本次导入流程预解析的跨物料依赖（docs/34 阶段 A2）
	//
	// 由 Import() 在分发前统一构建，各 importXxx 只读不写。目的是让
	// importNamer 不再回读只读连接 s.db 取康熙笔画/频率档——「依赖物料已落库」
	// 这个前提在部分导入、增量重灌、跨连接 WAL 可见性下都脆弱。
	// 包外调用方无需设置（nil 时各 importXxx 退化为无依赖合并）。
	deps *importDeps
}

// importDeps 导入流程内注入的已解析跨物料依赖
type importDeps struct {
	// kangxi 康熙笔画表（char → strokes），来自 kangxi-strokecount.csv
	kangxi map[string]int
}

// ImportResult 单次导入的汇总结果
type ImportResult struct {
	Material string `json:"material"`
	Source   string `json:"source"`
	Checksum string `json:"checksum"`
	Version  string `json:"version"`
	Rows     int    `json:"rows"`
	Skipped  bool   `json:"skipped"`
	// Elapsed 本物料导入全耗时（含指纹计算，B6 观测口径，docs/40）
	Elapsed string `json:"elapsed,omitempty"`
	Reason  string `json:"reason,omitempty"`
	Err     string `json:"error,omitempty"`
}

// ImportReport 一次 import 命令的完整报告
type ImportReport struct {
	Results   []ImportResult `json:"results"`
	Written   int            `json:"written"`
	Skipped   int            `json:"skipped"`
	Failed    int            `json:"failed"`
	DryRun    bool           `json:"dryRun"`
	StartedAt string         `json:"startedAt"`
	// PartialCommitted（docs/40 A3）：物料本体事务已全部提交，但随后的
	// 派生维护（syncNameFreqTier / pruneKangxiStrokes）失败 —— 库内主数据
	// 与派生列处于不一致半态。CLI 应提示用 --lenient 重建修复。
	PartialCommitted bool `json:"partialCommitted,omitempty"`
	// Elapsed（docs/40 B6）整次 Import 的服务端耗时：deps 预解析 + 写入 +
	// 派生维护 + 失败上抛全链路均计入；JSON 报告供 CI 消费耗时位。
	Elapsed string `json:"elapsed,omitempty"`
}

// Import 把全部（或指定）物料导入 SQLite
func (s *Store) Import(opt ImportOptions) (*ImportReport, error) {
	defs, err := resolveMaterials(opt.Materials)
	if err != nil {
		return nil, err
	}
	order, err := schema.TopoSort(defs)
	if err != nil {
		return nil, err
	}

	if !opt.DryRun {
		if err := createMaterialTables(s.writeDB); err != nil {
			return nil, err
		}
	}

	// 预解析跨物料依赖：康熙笔画由源文件直接解析后注入，importNamer 不再回读库
	// （docs/34 阶段 A2，见 importDeps 注释）。
	// docs/40 A4：解析失败改为硬失败。此前「留空 map 不阻断」会让 namer 把
	// kangxi_strokes=0 落库并登记指纹，之后无论修复 CSV 还是重灌 kangxi，
	// 指纹短路都会让 namer 的 0 列永久存在 —— 静默降级制造不可自愈的脏数据。
	deps, err := s.resolveImportDeps(order, opt.DataDir)
	if err != nil {
		return nil, err
	}
	opt.deps = deps

	alImportStart := time.Now()
	report := &ImportReport{DryRun: opt.DryRun, StartedAt: alImportStart.Format("2006-01-02 15:04:05")}
	for _, name := range order {
		def := schema.Registry[name]
		r := s.importOne(def, opt)
		report.Results = append(report.Results, r)
		switch {
		case r.Err != "":
			report.Failed++
		case r.Skipped:
			report.Skipped++
		default:
			report.Written++
		}
	}

	// 派生维护：独立事务的收尾步骤（docs/40 A3）
	if !opt.DryRun {
		if err := s.runDerivedMaintenance(); err != nil {
			// 物料本体事务已全部提交，此处失败只影响派生列（name_freq_tier /
			// 孤儿康熙行），不回滚已落库主数据 —— 明确标记半态，交由调用方
			// （CLI/CI）提示修复路径。
			report.PartialCommitted = true
			report.Elapsed = time.Since(alImportStart).String()
			return report, fmt.Errorf("派生维护失败（物料已提交，属部分提交状态，建议 --lenient 重建）: %w", err)
		}
	}
	report.Elapsed = time.Since(alImportStart).String()
	return report, nil
}

// runDerivedMaintenance 所有物料事务提交后的派生维护（docs/40 A3）
//
// 独立 BEGIN IMMEDIATE 事务包裹，原子执行：
//  1. name_freq_tier 单一来源归一化（docs/34 阶段 A3）：
//     以 name_frequency 表为唯一真源重算派生列，无论哪一侧物料变更或被
//     指纹短路跳过，派生链都不断。
//  2. kangxi_strokes 裁剪到 hanzi_data 字符集（docs/37 E1）：
//     CSV 含 6.3 万条，仅约 7.8 千属于字库，其余为孤儿，运行期候选池永不
//     命中（LoadKangxiStrokesFromDB 全表入内存，每条查不到）。落库后按字集
//     裁剪既缩小库体积与内存，又不影响消费方（GetKangxiStrokes 对缺失字
//     返回 0，与裁剪前行为一致）。必须放在所有物料写入完成之后——拓扑上
//     kangxi 先于 namer，此处二者均已落库，裁剪安全；hanzi_data 为空时跳过，
//     避免误删全部康熙数据。
//
// 与旧实现的差异（docs/40 A3）：原先两个步骤在无事务保护下裸执行，失败时
// 物料已提交、派生列不一致，调用方无从感知半态。
func (s *Store) runDerivedMaintenance() error {
	tx, err := s.writeDB.Begin()
	if err != nil {
		return fmt.Errorf("开启派生维护事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`UPDATE hanzi_data SET name_freq_tier = COALESCE(
		(SELECT tier FROM name_frequency
		 WHERE name_frequency.char = hanzi_data.char AND tier >= 2), 0)`); err != nil {
		return fmt.Errorf("归一化人名频率档位失败: %w", err)
	}

	// 裁剪前置保护（沿用旧 pruneKangxiStrokes 语义）：hanzi_data 为空时
	// `NOT IN (空集合)` 恒真，会误删全部康熙数据 —— 先计数再裁剪。
	var hanziRows int
	if err := tx.QueryRow("SELECT COUNT(*) FROM hanzi_data").Scan(&hanziRows); err != nil {
		return fmt.Errorf("查询 hanzi_data 行数失败: %w", err)
	}
	if hanziRows > 0 {
		res, err := tx.Exec(`DELETE FROM kangxi_strokes
			WHERE char NOT IN (SELECT char FROM hanzi_data)`)
		if err != nil {
			return fmt.Errorf("裁剪康熙笔画孤儿行失败: %w", err)
		}
		if affected, e := res.RowsAffected(); e == nil && affected > 0 {
			logger.Infof("已裁剪 kangxi_strokes 孤儿行: %d", affected)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("提交派生维护事务失败: %w", err)
	}
	return nil
}

// resolveImportDeps 预解析导入顺序中各物料对外暴露的依赖数据
//
// 只解析「被其它物料合并消费」的依赖（当前仅康熙笔画）。docs/40 A4：
// 解析失败改为硬失败（返回 error 中止整次导入）—— 静默降级会让 importNamer
// 把 kangxi_strokes=0 落库并登记指纹，之后无论修复 CSV 还是重灌 kangxi，
// 指纹短路都会让 namer 的 0 列永久存在，属于不可自愈的脏数据。
//
// 注意：order 中不存在 kangxi 物料时依赖留空（显式排除场景），由 importNamer
// 自行按其 Required 语义处理，不算解析失败。
func (s *Store) resolveImportDeps(order []string, dataDir string) (*importDeps, error) {
	deps := &importDeps{kangxi: map[string]int{}}
	for _, name := range order {
		if name != schema.KangxiMaterial {
			continue
		}
		def, ok := schema.Registry[name]
		if !ok {
			continue
		}
		src := filepath.Join(dataDir, filepath.FromSlash(def.Source))
		m, err := parseKangxiFile(src)
		if err != nil {
			return nil, fmt.Errorf("解析依赖源文件失败（%s）: %w", src, err)
		}
		deps.kangxi = m
	}
	return deps, nil
}

// parseKangxiFile 解析康熙笔画 CSV 为 char → strokes 映射
//
// importKangxi 与 resolveImportDeps 共用同一解析入口，避免两份口径漂移。
func parseKangxiFile(src string) (map[string]int, error) {
	m, err := hanzi.ParseKangxiCSVFile(src)
	if err != nil {
		return nil, err
	}
	out := make(map[string]int, len(m))
	for c, v := range m {
		out[c] = normalize.Stroke(v)
	}
	return out, nil
}

// resolveMaterials 按名称过滤物料定义，并自动补齐依赖闭包
//
// 只导入 namer 时必须同时导入它依赖的 kangxi，否则康熙笔画合并无从进行；
// 依赖链上的物料同样参与拓扑排序，保证顺序正确。
func resolveMaterials(names []string) ([]schema.Definition, error) {
	if len(names) == 0 {
		return schema.Definitions, nil
	}

	picked := make(map[string]bool, len(names))
	unknown := make([]string, 0)
	for _, n := range names {
		if _, ok := schema.Registry[n]; !ok {
			unknown = append(unknown, n)
		}
		picked[n] = true
	}
	if len(unknown) > 0 {
		return nil, fmt.Errorf("未知物料 %s（可用：%s）",
			strings.Join(unknown, ", "), strings.Join(schema.Names(), ", "))
	}

	// 广度优先补齐依赖闭包
	queue := make([]string, 0, len(picked))
	for n := range picked {
		queue = append(queue, n)
	}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		d, ok := schema.Registry[cur]
		if !ok {
			continue
		}
		for _, dep := range d.DependsOn {
			if picked[dep] {
				continue
			}
			picked[dep] = true
			queue = append(queue, dep)
		}
	}

	out := make([]schema.Definition, 0, len(picked))
	for _, d := range schema.Definitions {
		if picked[d.Name] {
			out = append(out, d)
		}
	}
	return out, nil
}

// importOne 导入单个物料
func (s *Store) importOne(def schema.Definition, opt ImportOptions) ImportResult {
	res := ImportResult{Material: def.Name, Source: def.Source}
	startedAt := time.Now()

	// classics 特化目录指纹（docs/40 A1）：
	// classics 的源是 dataDir 下多个 JSON（classicsSourceFiles 扫描，稳定序），
	// 而非 def.Source 登记的单文件。旧实现指纹只取 shijing.json，任一其余
	// 文件变更都不触发重灌 —— 指纹门控对 classics 形同虚设。
	// 此处提前走目录指纹路径：校验全部文件存在性 + 合并 sha256。
	if def.Name == schema.ClassicsMaterial {
		return s.importOneClassics(def, opt, res, startedAt)
	}

	src := filepath.Join(opt.DataDir, filepath.FromSlash(def.Source))
	if _, err := os.Stat(src); err != nil {
		if def.Required {
			res.Err = fmt.Sprintf("必需物料源缺失: %s", src)
		} else {
			res.Skipped = true
			res.Reason = "可选物料源缺失，跳过"
		}
		res.Elapsed = time.Since(startedAt).String()
		return res
	}

	sum, err := checksum.File(src)
	if err != nil {
		res.Err = err.Error()
		return res
	}
	res.Checksum = sum

	// 指纹未变则短路，避免无谓重灌（ADR-014）
	if !opt.DryRun && !opt.Lenient && s.materialUnchanged(def.Name, sum) {
		res.Skipped = true
		res.Reason = "指纹未变更，跳过重灌"
		res.Rows = s.materialRowCount(def.Name)
		res.Version = s.storedMaterialVersion(def.Name)
		res.Elapsed = time.Since(startedAt).String()
		return res
	}

	// 解析 → 归一化 → 校验 → 写入：按注册表多态分发（store_material_registry.go）
	m, ok := materialRegistry[def.Name]
	if !ok {
		res.Err = fmt.Sprintf("物料 %s 未登记导入实现（见 store_material_registry.go）", def.Name)
		return res
	}
	out := m.Import(s, src, res, opt)
	out.Elapsed = time.Since(startedAt).String()
	return out
}

// importOneClassics classics 目录指纹特化路径（docs/40 A1）
//
// 指纹 = classicsSourceFiles 稳定序扫描 → checksum.Files 合并；
// 保留旧门控语义：def.Source 哨兵文件（shijing.json）缺失 → 物料级跳过
// （classics 为 Required=false，可选源缺失不阻断）。哨兵存在后目录扫描
// 才启动；任一目录文件缺失即整体中断。
func (s *Store) importOneClassics(def schema.Definition, opt ImportOptions, res ImportResult, startedAt time.Time) ImportResult {
	sentinel := filepath.Join(opt.DataDir, filepath.FromSlash(def.Source))
	if _, err := os.Stat(sentinel); err != nil {
		res.Skipped = true
		res.Reason = "可选物料源缺失，跳过"
		return res
	}

	files := classicsSourceFiles(opt.DataDir)
	if len(files) == 0 {
		res.Err = fmt.Sprintf("经典语料扫描为空（目录 %s 无登记 JSON 或不可读）", opt.DataDir)
		return res
	}
	for _, f := range files {
		if _, err := os.Stat(filepath.Join(opt.DataDir, f)); err != nil {
			res.Err = fmt.Sprintf("经典语料源文件缺失: %s: %v", f, err)
			return res
		}
	}
	sum, err := checksum.Files(opt.DataDir, files)
	if err != nil {
		res.Err = err.Error()
		return res
	}
	res.Checksum = sum

	if !opt.DryRun && !opt.Lenient && s.materialUnchanged(def.Name, sum) {
		res.Skipped = true
		res.Reason = "指纹未变更，跳过重灌"
		res.Rows = s.materialRowCount(def.Name)
		res.Version = s.storedMaterialVersion(def.Name)
		res.Elapsed = time.Since(startedAt).String()
		return res
	}

	out := s.importClassics(def, def.Source, res, opt)
	out.Elapsed = time.Since(startedAt).String()
	return out
}

// materialUnchanged 判断库内记录的指纹是否与传入一致且主表非空
//
// 只校验主表（Tables[0]，schema.Definition 约定）：辅表允许合理为空
// （如 namer 的 char_groups 源无分组时），此前要求全部表非空会让这类
// 物料每次导入都重灌，指纹门控形同虚设（docs/34 阶段 B3）。
func (s *Store) materialUnchanged(material, sum string) bool {
	var stored string
	err := s.db.QueryRow("SELECT checksum FROM materials_state WHERE material = ?", material).Scan(&stored)
	if err != nil || stored == "" || stored != sum {
		return false
	}
	tables := materialTablesOf(material)
	if len(tables) == 0 {
		return false
	}
	var n int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM " + tables[0]).Scan(&n); err != nil || n == 0 {
		return false
	}
	return true
}

// materialTablesOf 物料对应的表
func materialTablesOf(material string) []string {
	if d, ok := schema.Registry[material]; ok {
		return d.Tables
	}
	return nil
}

// materialRowCount 物料当前落库行数（用于报告）
func (s *Store) materialRowCount(material string) int {
	total := 0
	for _, t := range materialTablesOf(material) {
		var n int
		if err := s.db.QueryRow("SELECT COUNT(*) FROM " + t).Scan(&n); err == nil {
			total += n
		}
	}
	return total
}

// runImport 事务骨架：单事务内清表 → 写入 → 登记状态
func (s *Store) runImport(material string, tables []string, res *ImportResult, opt ImportOptions, write func(tx *sql.Tx) (int, error)) *ImportResult {
	if opt.DryRun {
		res.Rows = 0
		res.Reason = "试运行：仅校验，未写库"
		return res
	}

	tx, err := s.writeDB.Begin()
	if err != nil {
		res.Err = fmt.Sprintf("开启导入事务失败: %v", err)
		return res
	}
	defer func() { _ = tx.Rollback() }()

	for _, t := range tables {
		if _, err := tx.Exec("DELETE FROM " + t); err != nil {
			res.Err = fmt.Sprintf("清空表 %s 失败: %v", t, err)
			return res
		}
	}

	n, err := write(tx)
	if err != nil {
		res.Err = err.Error()
		return res
	}

	if err := s.registerMaterialState(tx, material, res.Version, res.Checksum, n); err != nil {
		res.Err = err.Error()
		return res
	}

	if err := tx.Commit(); err != nil {
		res.Err = fmt.Sprintf("提交导入事务失败: %v", err)
		return res
	}
	res.Rows = n
	logger.Info("sqlite: 物料已导入", logger.String("物料", material), logger.Int("行数", n))
	return res
}

// registerMaterialState 在导入事务内登记物料指纹与行数
//
// 供 runImport 与自带事务的 importXxx（如 importCurated）共用，避免
// materials_state 的 UPSERT 语句两处维护。
func (s *Store) registerMaterialState(tx *sql.Tx, material, version, checksum string, rows int) error {
	if _, err := tx.Exec(`INSERT INTO materials_state(material, version, checksum, row_count)
		VALUES(?,?,?,?)
		ON CONFLICT(material) DO UPDATE SET version = excluded.version,
			checksum = excluded.checksum, row_count = excluded.row_count,
			imported_at = CURRENT_TIMESTAMP`,
		material, version, checksum, rows); err != nil {
		return fmt.Errorf("登记物料状态失败: %v", err)
	}
	return nil
}

// pendingErrors 收集待上报的校验错误（由各 importXxx 设置）
//
// 注意：不得用包级全局传递 —— 并发导入或多物料连续导入时会互相污染。
// 校验结果一律由各 importXxx 直接写入 res.Err 并返回。

// failValidate 记录校验错误；非 lenient 时直接返回错误字符串
//
// 输出走 Errors.Summarize 聚合（docs/34 阶段 C2）：按 (字段, 规则) 计数 +
// 前几条明细，避免大批量脏数据时逐条拼接出不可读的长串。
func failValidate(errs validate.Errors, opt ImportOptions) string {
	if len(errs) == 0 {
		return ""
	}
	if opt.Lenient {
		logger.Warn("sqlite: 校验发现问题（lenient 模式继续导入）", logger.Int("问题数", len(errs)))
		return ""
	}
	return errs.Summarize(3)
}

// mustJSON 序列化标签切片；出错时退化为空数组而非中断导入
func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(b)
}

// --- 各物料导入实现 ---

// importKangxi 导入康熙笔画
func (s *Store) importKangxi(def schema.Definition, src string, res ImportResult, opt ImportOptions) ImportResult {
	m, err := parseKangxiFile(src)
	if err != nil {
		res.Err = err.Error()
		return res
	}
	res.Version = "csv"

	items := make([]schema.KangxiStroke, 0, len(m))
	for c, v := range m {
		items = append(items, schema.KangxiStroke{Char: c, Strokes: v})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Char < items[j].Char })

	var errs validate.Errors
	for _, it := range items {
		errs = append(errs, validate.Kangxi(def.Name, it)...)
		if len(errs) >= maxImportErrors {
			break
		}
	}
	if msg := failValidate(errs, opt); msg != "" {
		res.Err = msg
		return res
	}

	return *s.runImport(def.Name, def.Tables, &res, opt, func(tx *sql.Tx) (int, error) {
		stmt, err := tx.Prepare("INSERT INTO kangxi_strokes(char, strokes) VALUES(?,?)")
		if err != nil {
			return 0, err
		}
		defer func() { _ = stmt.Close() }()
		for _, it := range items {
			if _, err := stmt.Exec(it.Char, it.Strokes); err != nil {
				return 0, fmt.Errorf("写入康熙笔画 %s 失败: %w", it.Char, err)
			}
		}
		return len(items), nil
	})
}

// namerJSON namer.json 的磁盘结构
type namerJSON struct {
	Version    string                    `json:"version"`
	Generated  string                    `json:"generated"`
	TotalChars int                       `json:"totalChars"`
	Chars      []hanzi.NamerChar         `json:"chars"`
	CharGroups []hanzi.StandardCharGroup `json:"charGroups"`
}

// importNamer 导入汉字表与偏旁分组（康熙笔画由导入流程注入，不再回读库）
func (s *Store) importNamer(def schema.Definition, src string, res ImportResult, opt ImportOptions) ImportResult {
	raw, err := os.ReadFile(src)
	if err != nil {
		res.Err = fmt.Sprintf("读取 %s 失败: %v", src, err)
		return res
	}
	var nj namerJSON
	if err := json.Unmarshal(raw, &nj); err != nil {
		res.Err = fmt.Sprintf("解析 %s 失败: %v", src, err)
		return res
	}
	res.Version = nj.Version + "@" + nj.Generated

	// 康熙笔画合并（docs/31 §9.6 跨物料合并优先级：康熙只填该列，不动简体笔画）。
	//
	// 数据由 Import() 在分发前从源文件预解析后注入（docs/34 阶段 A2），不再回读
	// 只读连接 s.db —— 回读隐含「依赖物料已落库 + WAL 跨连接可见」两个前提，
	// 在部分导入与增量重灌下都脆弱。
	kangxi := map[string]int{}
	if opt.deps != nil && len(opt.deps.kangxi) > 0 {
		for c, v := range opt.deps.kangxi {
			kangxi[c] = v
		}
	}

	// name_freq_tier 不在此处合并：该列是 name_frequency 的派生列，由 Import()
	// 末尾的 syncNameFreqTier 以库表为唯一真源统一归一化（docs/34 阶段 A3），
	// 避免「importNamer 回读 + importCharFreq UPDATE」两处维护的口径漂移。

	// 逐字归一：namer.json 实测仅 17 个字段
	//   - 有真值：char/pinyin/strokes/radical/meaning/wuxing/gender/level/tone/
	//     styleTags/usageLevel/positiveScore/isRare/isNegative/isPolyphonic/namePenalty/
	//     namingCategories
	//   - 无真值（保持零值，不得伪造）：phoneticScore/modernScore/classicalScore/
	//     curationLevel/genderTags/pairBlacklist/nameFreqTier（后者由频率物料派生）
	words := make([]schema.Word, 0, len(nj.Chars))
	for _, nc := range nj.Chars {
		// 五行必须走与内存装载路径同一套解析规则（hanzi.ResolveWuxing）：
		// 字义法覆盖表 > 已有值 > 部首字形法兜底。
		// 此前此处直接落 nc.Wuxing 原值，漏掉了覆盖表与部首兜底，
		// 实测有 14 字在 hanzi_data 中为空而内存侧有值，
		// 导致「SQL 与内存结果等价」守护测试失败（docs/31 §11.9 第 2 项）。
		wuxing := hanzi.ResolveWuxing(nc.Char, nc.Wuxing, nc.Radical)

		w := normalize.Word(schema.Word{
			Char: nc.Char, Pinyin: nc.Pinyin, Strokes: nc.Strokes,
			KangxiStrokes: kangxi[nc.Char],
			Radical:       nc.Radical, Meaning: nc.Meaning, Wuxing: wuxing,
			Level: nc.Level, Gender: normalize.Gender(nc.Gender),
			Tone:       nc.Tone,
			UsageLevel: nc.UsageLevel, PositiveScore: nc.PositiveScore,
			NamePenalty:  nc.NamePenalty,
			IsPolyphonic: nc.IsPolyphonic, IsRare: nc.IsRare, IsNegative: nc.IsNegative,
			StyleTags:  nc.StyleTags,
			NamingCats: nc.NamingCategories,
		})
		words = append(words, w)
	}
	sort.Slice(words, func(i, j int) bool { return words[i].Char < words[j].Char })

	if msg := failValidate(validate.Words(def.Name, words, maxImportErrors), opt); msg != "" {
		res.Err = msg
		return res
	}

	// 偏旁分组：分组内汉字必须已入库（引用完整性）
	known := make(map[string]bool, len(words))
	for _, w := range words {
		known[w.Char] = true
	}
	groups := make([]schema.CharGroup, 0, len(nj.CharGroups))
	for _, g := range nj.CharGroups {
		groups = append(groups, normalize.CharGroup(schema.CharGroup{
			Radical: g.Radical, Name: g.Name, Meaning: g.Meaning, Chars: g.Chars,
		}))
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].Radical < groups[j].Radical })

	var gerrs validate.Errors
	for _, g := range groups {
		gerrs = append(gerrs, validate.CharGroup(def.Name, g, known)...)
		if len(gerrs) >= maxImportErrors {
			break
		}
	}
	if msg := failValidate(gerrs, opt); msg != "" {
		res.Err = msg
		return res
	}

	return *s.runImport(def.Name, def.Tables, &res, opt, func(tx *sql.Tx) (int, error) {
		stmt, err := tx.Prepare(hanziDataInsertSQL)
		if err != nil {
			return 0, err
		}
		defer func() { _ = stmt.Close() }()
		n := 0
		for _, w := range words {
			if _, err := stmt.Exec(wordValues(w)...); err != nil {
				return 0, fmt.Errorf("写入汉字 %s 失败: %w", w.Char, err)
			}
			n++
		}

		gstmt, err := tx.Prepare("INSERT INTO char_groups(radical, name, meaning, chars) VALUES(?,?,?,?)")
		if err != nil {
			return 0, err
		}
		defer func() { _ = gstmt.Close() }()
		for _, g := range groups {
			if _, err := gstmt.Exec(g.Radical, g.Name, g.Meaning, mustJSON(g.Chars)); err != nil {
				return 0, fmt.Errorf("写入偏旁分组 %s 失败: %w", g.Radical, err)
			}
			n++
		}
		return n, nil
	})
}

// wordValues 把规范汉字转成 hanzi_data 行值（与运行时投影保持同一语义）
func wordValues(w schema.Word) []any {
	return []any{
		w.Char, w.Pinyin, w.Strokes, w.KangxiStrokes, w.Radical, w.Meaning,
		w.Wuxing, string(w.Gender), w.Level, w.Tone, w.NameFreqTier,
		mustJSON(w.GenderTags), mustJSON(w.StyleTags), w.UsageLevel,
		w.PositiveScore, w.PhoneticScore, w.ModernScore, w.ClassicalScore,
		boolToInt(w.IsPolyphonic), boolToInt(w.IsRare), boolToInt(w.IsNegative),
		mustJSON(w.PairBlacklist), w.CurationLevel, w.NamePenalty,
		mustJSON(w.NamingCats),
	}
}

// freqJSON 人名频率的磁盘结构
type freqJSON struct {
	Meta struct {
		Source      string `json:"source"`
		TotalNames  int    `json:"total_names"`
		TotalChars  int    `json:"total_chars"`
		UniqueChars int    `json:"unique_chars"`
		BuiltAt     string `json:"built_at"`
	} `json:"meta"`
	Frequency []hanzi.CharFrequency `json:"frequency"`
}

// importCharFreq 导入单字人名频率
func (s *Store) importCharFreq(def schema.Definition, src string, res ImportResult, opt ImportOptions) ImportResult {
	raw, err := os.ReadFile(src)
	if err != nil {
		res.Err = fmt.Sprintf("读取 %s 失败: %v", src, err)
		return res
	}
	var fj freqJSON
	if err := json.Unmarshal(raw, &fj); err != nil {
		res.Err = fmt.Sprintf("解析 %s 失败: %v", src, err)
		return res
	}
	res.Version = fj.Meta.BuiltAt

	items := make([]schema.CharFreq, 0, len(fj.Frequency))
	for _, f := range fj.Frequency {
		items = append(items, normalize.CharFreq(schema.CharFreq{
			Char: f.Char, Count: f.Count, Rank: f.Rank, Tier: f.Tier,
		}))
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Char < items[j].Char })

	var errs validate.Errors
	for _, c := range items {
		errs = append(errs, validate.CharFreq(def.Name, c)...)
		if len(errs) >= maxImportErrors {
			break
		}
	}
	if msg := failValidate(errs, opt); msg != "" {
		res.Err = msg
		return res
	}

	return *s.runImport(def.Name, def.Tables, &res, opt, func(tx *sql.Tx) (int, error) {
		stmt, err := tx.Prepare("INSERT INTO name_frequency(char, count, rank, tier) VALUES(?,?,?,?)")
		if err != nil {
			return 0, err
		}
		defer func() { _ = stmt.Close() }()
		for _, c := range items {
			if _, err := stmt.Exec(c.Char, c.Count, c.Rank, c.Tier); err != nil {
				return 0, fmt.Errorf("写入单字频率 %s 失败: %w", c.Char, err)
			}
		}

		// name_freq_tier 回填已收敛到 Import() 末尾的 syncNameFreqTier
		// （docs/34 阶段 A3）：以 name_frequency 表为唯一真源单点归一化，
		// 不再在本函数内 UPDATE hanzi_data，避免两处维护的口径漂移。
		return len(items), nil
	})
}

// bigramJSON 双字频率的磁盘结构
type bigramJSON struct {
	Meta struct {
		Source  string `json:"source"`
		BuiltAt string `json:"built_at"`
	} `json:"meta"`
	Frequency []hanzi.BigramFrequency `json:"frequency"`
}

// importBigram 导入双字人名频率
func (s *Store) importBigram(def schema.Definition, src string, res ImportResult, opt ImportOptions) ImportResult {
	raw, err := os.ReadFile(src)
	if err != nil {
		res.Err = fmt.Sprintf("读取 %s 失败: %v", src, err)
		return res
	}
	var bj bigramJSON
	if err := json.Unmarshal(raw, &bj); err != nil {
		res.Err = fmt.Sprintf("解析 %s 失败: %v", src, err)
		return res
	}
	res.Version = bj.Meta.BuiltAt

	items := make([]schema.BigramFreq, 0, len(bj.Frequency))
	for _, f := range bj.Frequency {
		items = append(items, normalize.BigramFreq(schema.BigramFreq{
			Char1: f.Char1, Char2: f.Char2, Count: f.Count, Rank: f.Rank, Tier: f.Tier,
		}))
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Key() < items[j].Key() })

	var errs validate.Errors
	for _, b := range items {
		errs = append(errs, validate.BigramFreq(def.Name, b)...)
		if len(errs) >= maxImportErrors {
			break
		}
	}
	if msg := failValidate(errs, opt); msg != "" {
		res.Err = msg
		return res
	}

	return *s.runImport(def.Name, def.Tables, &res, opt, func(tx *sql.Tx) (int, error) {
		stmt, err := tx.Prepare("INSERT INTO name_bigram(key, char1, char2, count, rank, tier) VALUES(?,?,?,?,?,?)")
		if err != nil {
			return 0, err
		}
		defer func() { _ = stmt.Close() }()
		for _, b := range items {
			if _, err := stmt.Exec(b.Key(), b.Char1, b.Char2, b.Count, b.Rank, b.Tier); err != nil {
				return 0, fmt.Errorf("写入双字频率 %s 失败: %w", b.Key(), err)
			}
		}
		return len(items), nil
	})
}

// importGate 导入命名质量门禁字表（走字符串数组骨架，见 importStringSliceMaterial）
func (s *Store) importGate(def schema.Definition, src string, res ImportResult, opt ImportOptions) ImportResult {
	return s.importStringSliceMaterial(def, src, res, opt,
		func(n int) string { return fmt.Sprintf("%d-chars", n) },
		func(material, value string) validate.Errors { return validate.Rune(material, "char", value) },
		"INSERT INTO naming_gate(char) VALUES(?)", "门禁字",
	)
}

// importForbidden 导入禁忌双字组合（走字符串数组骨架，见 importStringSliceMaterial）
func (s *Store) importForbidden(def schema.Definition, src string, res ImportResult, opt ImportOptions) ImportResult {
	return s.importStringSliceMaterial(def, src, res, opt,
		func(n int) string { return fmt.Sprintf("%d-combos", n) },
		validate.Combo,
		"INSERT INTO forbidden_combos(combo) VALUES(?)", "禁忌组合",
	)
}

// importAllowlist 导入显式白名单（字 / 组合 / 祈福组合），docs/33 门禁层 M1
//
// 源格式：{"version":"...","chars":[...],"combos":[...],"blessings":[...]}
// 落库：allowlist(entry PK, kind)，kind ∈ char/combo/blessing（白名单优先于黑名单）。
// 读写骨架见 importKindRowsMaterial。
func (s *Store) importAllowlist(def schema.Definition, src string, res ImportResult, opt ImportOptions) ImportResult {
	read := func(src string) (string, []kindRow, validate.Errors, error) {
		raw, err := os.ReadFile(src)
		if err != nil {
			return "", nil, nil, fmt.Errorf("读取 %s 失败: %v", src, err)
		}
		var doc struct {
			Version   string   `json:"version"`
			Chars     []string `json:"chars"`
			Combos    []string `json:"combos"`
			Blessings []string `json:"blessings"`
		}
		if err := json.Unmarshal(raw, &doc); err != nil {
			return "", nil, nil, fmt.Errorf("解析 %s 失败: %v", src, err)
		}
		rows := make([]kindRow, 0, len(doc.Chars)+len(doc.Combos)+len(doc.Blessings))
		var errs validate.Errors
		for _, c := range normalize.Tags(doc.Chars) {
			errs = append(errs, validate.Rune(def.Name, "char", c)...)
			rows = append(rows, kindRow{entry: c, kind: fate.AllowKindChar})
		}
		for _, c := range normalize.Tags(doc.Combos) {
			errs = append(errs, validate.Combo(def.Name, c)...)
			rows = append(rows, kindRow{entry: c, kind: fate.AllowKindCombo})
		}
		for _, c := range normalize.Tags(doc.Blessings) {
			errs = append(errs, validate.Combo(def.Name, c)...)
			rows = append(rows, kindRow{entry: c, kind: fate.AllowKindBlessing})
		}
		return doc.Version, rows, errs, nil
	}
	return s.importKindRowsMaterial(def, src, res, opt, "白名单", read)
}

// importComplianceRedline 导入 L5 合规红线（机关单位/争议地名/负面历史人物），docs/33 门禁层 M2
//
// 源格式：{"version":"...","state_orgs":[...],"places":[...],"persons":[...]}
// 落库：compliance_redline(entry PK, kind)，kind ∈ state_org/place/person。
// 读写骨架见 importKindRowsMaterial。
func (s *Store) importComplianceRedline(def schema.Definition, src string, res ImportResult, opt ImportOptions) ImportResult {
	read := func(src string) (string, []kindRow, validate.Errors, error) {
		raw, err := os.ReadFile(src)
		if err != nil {
			return "", nil, nil, fmt.Errorf("读取 %s 失败: %v", src, err)
		}
		var doc struct {
			Version   string   `json:"version"`
			StateOrgs []string `json:"state_orgs"`
			Places    []string `json:"places"`
			Persons   []string `json:"persons"`
		}
		if err := json.Unmarshal(raw, &doc); err != nil {
			return "", nil, nil, fmt.Errorf("解析 %s 失败: %v", src, err)
		}
		rows := make([]kindRow, 0, len(doc.StateOrgs)+len(doc.Places)+len(doc.Persons))
		var errs validate.Errors
		for _, e := range normalize.Tags(doc.StateOrgs) {
			errs = append(errs, validate.Entity(def.Name, "state_org", e)...)
			rows = append(rows, kindRow{entry: e, kind: fate.RedlineKindStateOrg})
		}
		for _, e := range normalize.Tags(doc.Places) {
			errs = append(errs, validate.Entity(def.Name, "place", e)...)
			rows = append(rows, kindRow{entry: e, kind: fate.RedlineKindPlace})
		}
		for _, e := range normalize.Tags(doc.Persons) {
			errs = append(errs, validate.Entity(def.Name, "person", e)...)
			rows = append(rows, kindRow{entry: e, kind: fate.RedlineKindPerson})
		}
		return doc.Version, rows, errs, nil
	}
	return s.importKindRowsMaterial(def, src, res, opt, "合规红线", read)
}

// importYijing 导入易经六十四卦
func (s *Store) importYijing(def schema.Definition, src string, res ImportResult, opt ImportOptions) ImportResult {
	raw, err := os.ReadFile(src)
	if err != nil {
		res.Err = fmt.Sprintf("读取 %s 失败: %v", src, err)
		return res
	}
	var list []yijing.Hexagram
	if err := json.Unmarshal(raw, &list); err != nil {
		res.Err = fmt.Sprintf("解析 %s 失败: %v", src, err)
		return res
	}
	res.Version = fmt.Sprintf("%d-hexagrams", len(list))

	items := make([]schema.Hexagram, 0, len(list))
	for _, h := range list {
		items = append(items, schema.Hexagram{
			ID: h.ID, Name: h.Name, Number: h.Number, Symbol: h.Symbol,
			UpperTrigram: h.UpperTrigram, LowerTrigram: h.LowerTrigram,
			GuaCi: h.GuaCi, XiangCi: h.XiangCi, YaoCi: normalize.Tags(h.YaoCi),
			TuanCi: h.TuanCi, XiaoXiang: normalize.Tags(h.XiaoXiang),
			Interpretation: h.Interpretation,
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })

	seen := make(map[int]bool, len(items))
	var errs validate.Errors
	for _, h := range items {
		errs = append(errs, validate.Hexagram(def.Name, h, seen)...)
		seen[h.ID] = true
		if len(errs) >= maxImportErrors {
			break
		}
	}
	if msg := failValidate(errs, opt); msg != "" {
		res.Err = msg
		return res
	}

	return *s.runImport(def.Name, def.Tables, &res, opt, func(tx *sql.Tx) (int, error) {
		stmt, err := tx.Prepare(`INSERT INTO yijing_hexagrams(id, name, number, symbol,
			upper_trigram, lower_trigram, gua_ci, xiang_ci, yao_ci, tuan_ci, xiao_xiang, interpretation)
			VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`)
		if err != nil {
			return 0, err
		}
		defer func() { _ = stmt.Close() }()
		for _, h := range items {
			if _, err := stmt.Exec(h.ID, h.Name, h.Number, h.Symbol, h.UpperTrigram,
				h.LowerTrigram, h.GuaCi, h.XiangCi, mustJSON(h.YaoCi), h.TuanCi, mustJSON(h.XiaoXiang),
				h.Interpretation); err != nil {
				return 0, fmt.Errorf("写入卦象 %d 失败: %w", h.ID, err)
			}
		}
		return len(items), nil
	})
}

// importZodiac 导入十二生肖
func (s *Store) importZodiac(def schema.Definition, src string, res ImportResult, opt ImportOptions) ImportResult {
	raw, err := os.ReadFile(src)
	if err != nil {
		res.Err = fmt.Sprintf("读取 %s 失败: %v", src, err)
		return res
	}
	var list []zodiac.Zodiac
	if err := json.Unmarshal(raw, &list); err != nil {
		res.Err = fmt.Sprintf("解析 %s 失败: %v", src, err)
		return res
	}
	res.Version = fmt.Sprintf("%d-zodiacs", len(list))

	items := make([]schema.Zodiac, 0, len(list))
	for _, z := range list {
		items = append(items, schema.Zodiac{
			ID: z.ID, Name: z.Name, Wuxing: normalize.Wuxing(z.Wuxing),
			Compatible: normalize.Tags(z.Compatible), Conflicting: normalize.Tags(z.Conflicting),
			AvoidChars: strings.TrimSpace(z.AvoidChars), LuckyNumbers: z.LuckyNumbers,
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })

	var errs validate.Errors
	for _, z := range items {
		errs = append(errs, validate.Zodiac(def.Name, z)...)
		if len(errs) >= maxImportErrors {
			break
		}
	}
	if msg := failValidate(errs, opt); msg != "" {
		res.Err = msg
		return res
	}

	return *s.runImport(def.Name, def.Tables, &res, opt, func(tx *sql.Tx) (int, error) {
		stmt, err := tx.Prepare(`INSERT INTO zodiac(id, name, wuxing, compatible,
			conflicting, avoid_chars, lucky_numbers) VALUES(?,?,?,?,?,?,?)`)
		if err != nil {
			return 0, err
		}
		defer func() { _ = stmt.Close() }()
		for _, z := range items {
			if _, err := stmt.Exec(z.ID, z.Name, z.Wuxing, mustJSON(z.Compatible),
				mustJSON(z.Conflicting), z.AvoidChars, mustJSON(z.LuckyNumbers)); err != nil {
				return 0, fmt.Errorf("写入生肖 %d 失败: %w", z.ID, err)
			}
		}
		return len(items), nil
	})
}

// userFavoriteSource 用户自学习精选名的 source 标记（与 favorite_service 保持一致）
const userFavoriteSource = "user_favorite"

// importCurated 导入精选策展名（curated_names.json）
//
// 与其它物料不同，curated_names 表同时承载种子行（source=seed）与用户自学习行
// （source=user_favorite，收藏功能写入）。故不能走 runImport 的「整表重建」语义，
// 否则会清掉用户收藏：改为只重建非用户行，用户行原样保留；种子写入用
// INSERT OR IGNORE，与保留的用户行同名时让位用户行、不覆盖其富字段。
func (s *Store) importCurated(def schema.Definition, src string, res ImportResult, opt ImportOptions) ImportResult {
	raw, err := os.ReadFile(src)
	if err != nil {
		res.Err = fmt.Sprintf("读取 %s 失败: %v", src, err)
		return res
	}
	var entries []name.CuratedName
	if err := json.Unmarshal(raw, &entries); err != nil {
		res.Err = fmt.Sprintf("解析 %s 失败: %v", src, err)
		return res
	}

	// 归一化：去空名、同名首见即锁
	seen := make(map[string]bool, len(entries))
	items := make([]name.CuratedName, 0, len(entries))
	for _, e := range entries {
		n := strings.TrimSpace(e.Name)
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		e.Name = n
		items = append(items, e)
	}
	res.Version = fmt.Sprintf("%d-names", len(items))

	if opt.DryRun {
		res.Rows = 0
		res.Reason = "试运行：仅校验，未写库"
		return res
	}

	tx, err := s.writeDB.Begin()
	if err != nil {
		res.Err = fmt.Sprintf("开启导入事务失败: %v", err)
		return res
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`DELETE FROM curated_names WHERE source <> ?`, userFavoriteSource); err != nil {
		res.Err = fmt.Sprintf("清理旧策展名失败: %v", err)
		return res
	}

	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO curated_names
		(name,pinyin,gender,score,source,meaning,wuxing,styles,tags)
		VALUES(?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		res.Err = fmt.Sprintf("预编译写入失败: %v", err)
		return res
	}
	defer func() { _ = stmt.Close() }()

	for _, e := range items {
		if _, err := stmt.Exec(e.Name, e.Pinyin, e.Gender, e.YinyunScore, "seed",
			e.Meaning, e.Wuxing, mustJSON(normalize.Tags(e.Styles)), mustJSON(normalize.Tags(e.Tags))); err != nil {
			res.Err = fmt.Sprintf("写入策展名 %s 失败: %v", e.Name, err)
			return res
		}
	}

	if err := s.registerMaterialState(tx, def.Name, res.Version, res.Checksum, len(items)); err != nil {
		res.Err = err.Error()
		return res
	}

	if err := tx.Commit(); err != nil {
		res.Err = fmt.Sprintf("提交导入事务失败: %v", err)
		return res
	}
	res.Rows = len(items)
	logger.Info("sqlite: 物料已导入", logger.String("物料", def.Name), logger.Int("行数", len(items)))
	return res
}

// readStringSlice 读取一个 JSON 字符串数组文件
func readStringSlice(src string) ([]string, error) {
	raw, err := os.ReadFile(src)
	if err != nil {
		return nil, fmt.Errorf("读取 %s 失败: %w", src, err)
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("解析 %s 失败: %w", src, err)
	}
	return out, nil
}

// --- 通用导入骨架（docs/34 阶段 C3）---
//
// 各 importXxx 曾高度同构（读文件 → 归一 → 校验 → runImport），每类物料
// 手写 30~60 行样板。按「源结构 × 落库形态」归并为两个骨架，物料间只差
// 少量差异参数；新增同类物料只需在注册表登记 + 写一个分发函数。

// importStringSliceMaterial 「JSON 字符串数组 → 单列单表」物料的骨架
// （naming_gate / forbidden_combos）
//
// version 由调用方按条目数生成（各物料版本串格式不同）；
// check 为逐条校验函数（签名兼容 validate.Rune / validate.Combo 的适配）。
func (s *Store) importStringSliceMaterial(
	def schema.Definition, src string, res ImportResult, opt ImportOptions,
	version func(n int) string,
	check func(material, value string) validate.Errors,
	insertSQL, writeLabel string,
) ImportResult {
	values, err := readStringSlice(src)
	if err != nil {
		res.Err = err.Error()
		return res
	}
	items := normalize.Tags(values)
	res.Version = version(len(items))

	var errs validate.Errors
	for _, v := range items {
		errs = append(errs, check(def.Name, v)...)
		if len(errs) >= maxImportErrors {
			break
		}
	}
	if msg := failValidate(errs, opt); msg != "" {
		res.Err = msg
		return res
	}

	return *s.runImport(def.Name, def.Tables, &res, opt, func(tx *sql.Tx) (int, error) {
		stmt, err := tx.Prepare(insertSQL)
		if err != nil {
			return 0, err
		}
		defer func() { _ = stmt.Close() }()
		for _, v := range items {
			if _, err := stmt.Exec(v); err != nil {
				return 0, fmt.Errorf("写入%s %s 失败: %w", writeLabel, v, err)
			}
		}
		return len(items), nil
	})
}

// kindRow 一条 (entry, kind) 落库行（allowlist / compliance_redline 共用形态）
type kindRow struct {
	entry string
	kind  string
}

// importKindRowsMaterial 「多分类 JSON 对象 → (entry, kind) 两列表」物料的骨架
// （allowlist / compliance_redline）
//
// read 由调用方提供：负责读文件、按类别字段归一 + 逐类校验，返回
// 版本串与 (entry, kind) 行集合；骨架统一做 failValidate 与去重落库
// （entry 首见即锁，跨类别重复只保留首次出现的 kind）。
func (s *Store) importKindRowsMaterial(
	def schema.Definition, src string, res ImportResult, opt ImportOptions,
	writeLabel string,
	read func(src string) (version string, rows []kindRow, errs validate.Errors, err error),
) ImportResult {
	version, rows, errs, err := read(src)
	if err != nil {
		res.Err = err.Error()
		return res
	}
	res.Version = version

	if msg := failValidate(errs, opt); msg != "" {
		res.Err = msg
		return res
	}

	// 两列表形态：表名取主表（Definitions 约定 Tables[0]），列固定 (entry, kind)
	insertSQL := "INSERT INTO " + def.Tables[0] + "(entry, kind) VALUES(?,?)"
	return *s.runImport(def.Name, def.Tables, &res, opt, func(tx *sql.Tx) (int, error) {
		stmt, err := tx.Prepare(insertSQL)
		if err != nil {
			return 0, err
		}
		defer func() { _ = stmt.Close() }()
		seen := make(map[string]bool, len(rows))
		n := 0
		for _, r := range rows {
			if r.entry == "" || seen[r.entry] {
				continue
			}
			seen[r.entry] = true
			if _, err := stmt.Exec(r.entry, r.kind); err != nil {
				return 0, fmt.Errorf("写入%s %s 失败: %w", writeLabel, r.entry, err)
			}
			n++
		}
		return n, nil
	})
}
