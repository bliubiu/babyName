// materials 命令行工具：把 JSON/CSV 源物料导入 SQLite，并查询物料状态
//
// 定位（docs/31 ADR-008）：运行时统一只读 SQLite，物料不再由运行时装载 JSON。
// 因此导入动作显式化为一个 CLI 子命令，导入期完成归一化与校验，一次烘焙进库。
//
// 用法示例：
//
//	namer materials import --data ./data --db ./namer.db          # 导入全部物料
//	namer materials import --data ./data --db ./namer.db --dry-run # 只校验不写库
//	namer materials import --data ./data --db ./namer.db --only namer,kangxi
//	namer materials import --data ./data --db ./namer.db --lenient # 校验问题仅记录
//	namer materials load   --data ./data --db ./namer.db          # 库 → 内存装载自检
//	namer materials status  --db ./namer.db                        # 查看物料版本与行数
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"name/internal/domain/classics"
	"name/internal/domain/fate"
	"name/internal/infrastructure/config"
	"name/internal/infrastructure/data"
	"name/internal/infrastructure/database/sqlite"
	"name/internal/materials/review"
	"name/internal/materials/schema"
	"name/internal/materials/subscribe"
)

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(2)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	var err error
	switch cmd {
	case "import":
		err = runImport(args)
	case "load":
		err = runLoad(args)
	case "status":
		err = runStatus(args)
	case "manifest":
		err = runManifest(args)
	case "lock-init":
		err = runLockInit(args)
	case "lock-verify":
		err = runLockVerify(args)
	case "lock-diff":
		err = runLockDiff(args)
	case "patch":
		err = runPatch(args)
	case "stage":
		err = runStage(args)
	case "observe":
		err = runObserve(args)
	case "promote":
		err = runPromote(args)
	case "abort":
		err = runAbort(args)
	case "review":
		err = runReview(args)
	case "help", "-h", "--help":
		printHelp()
		return
	default:
		printHelp()
		fmt.Fprintf(os.Stderr, "未知子命令: %s\n", cmd)
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "执行失败: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`materials —— 物料导入与状态查询工具

用法:
  materials import [选项]     从 JSON/CSV 源导入物料到 SQLite（归一化+校验+单事务写入）
  materials load   [选项]     从 SQLite 库装载物料到内存索引（运行期装载路径自检）
  materials status  [选项]     列出各物料的版本、指纹与行数
  materials manifest [选项]    打印/校验入库源清单（物料 → 源文件，部署核对用）
  materials lock-init [选项]   生成门禁物料订阅清单 materials.lock（上游源+产物指纹）
  materials lock-verify [选项] 校验 materials.lock 中登记的产物指纹是否一致
  materials lock-diff [选项]   对两版本文本行差分，生成 gate_patch.jsonl 补丁
  materials patch [选项]       把补丁应用到 data 物料 JSON（--rollback 反向应用即回滚）
  materials stage [选项]       登记物料候选版本（灰度暂存，docs/33 M7）
  materials observe [选项]     查看候选相对生效版本的差异预演（不改变状态）
  materials promote [选项]     把候选提升为生效版本（旧版自动存档，可回退）
  materials abort [选项]       丢弃候选，或 --to <指纹> 回退生效版本
  materials review [子命令]    门禁复审队列（docs/33 M6）：list | decide | export

import 选项:
  --data    string   物料源目录（默认 ./data）
  --db      string   SQLite 库路径（默认读配置 database.path 的绝对路径；可显式覆盖）
  --only    string   只导入指定物料，逗号分隔（默认全部）
  --lenient          校验问题仅记录不阻断（数据清洗期取证用）
  --dry-run          只校验与统计，不写库
  --json             以 JSON 输出报告（供 CI 消费）

load 选项:
  --db      string   SQLite 库路径（默认读配置 database.path 的绝对路径；可显式覆盖）
  --data    string   物料源目录（仅用于错误提示；运行期不读 JSON）
  --json             以 JSON 输出装载报告（供 CI 消费）

  必需物料在库中缺失时命令以非 0 退出码结束并提示执行 materials import。

status 选项:
  --db      string   SQLite 库路径（默认读配置 database.path 的绝对路径；可显式覆盖）

manifest 选项:
  --data    string   物料源目录（默认 ./data）
  --check            校验清单内必需源文件是否齐全（缺失即非 0 退出）
  --json             以 JSON 输出清单（供 CI 消费）

订阅管线选项（docs/33 门禁层 M5）:
  lock-init:   --data ./data --out ./data/materials.lock --version 1.0 [--source naming_gate=repo@commit,...]
  lock-verify: --lock ./data/materials.lock --data ./data
  lock-diff:   --old old.txt --new new.txt --material naming_gate [--kind combo] [--out patch.jsonl]
  patch:       --file patch.jsonl --data ./data [--rollback]

灰度发布选项（docs/33 门禁层 M7 / ADR-015，切分粒度=物料整体）:
  stage:    --material allowlist --from ./candidate.json --data ./data [--lock ./data/materials.lock] [--source 说明]
  observe:  --material allowlist --data ./data [--lock ./data/materials.lock] [--limit 20]
  promote:  --material allowlist --data ./data [--lock ./data/materials.lock]
  abort:    --material allowlist --data ./data [--lock ./data/materials.lock] [--to <指纹>]

复审队列选项（docs/33 门禁层 M6）:
  review list:   --db <库路径> [--status pending|confirmed|false_positive|border] [--limit N]
  review decide: --db <库路径> --id <条目ID> --status confirmed|false_positive|border [--note 备注]
  review export: --db <库路径> [--out gate_patch.jsonl]   # 已落定条目 → gate_patch 补丁

说明:
  · 写入语义为「单事务 DELETE + INSERT 全量重建」，同输入多次导入结果等价（ADR-011）
  · 指纹未变的物料会短路跳过，避免数十 MB 物料的无谓重灌（ADR-014）
  · 导入完成后运行时统一从 SQLite 读取，物料变更需重新 import 才生效
  · 补丁为 append-only 的 JSONL 操作行；反向应用（Invert）即得回滚补丁
  · 物料清单: ` + strings.Join(schema.Names(), ", "))
}

func runImport(args []string) error {
	fs := flag.NewFlagSet("import", flag.ExitOnError)
	dataDir := fs.String("data", "./data", "物料源目录")
	dbPath := fs.String("db", "", "SQLite 库路径（默认读配置 database.path 的绝对路径）")
	only := fs.String("only", "", "只导入指定物料（逗号分隔）")
	lenient := fs.Bool("lenient", false, "校验问题仅记录不阻断")
	dryRun := fs.Bool("dry-run", false, "只校验不写库")
	asJSON := fs.Bool("json", false, "以 JSON 输出报告（供 CI 消费）")
	if err := fs.Parse(args); err != nil {
		return err
	}

	opt := sqlite.ImportOptions{
		DataDir:   *dataDir,
		Materials: splitList(*only),
		Lenient:   *lenient,
		DryRun:    *dryRun,
	}

	store, err := openStore(resolveDBPath(*dbPath), *dataDir, !*dryRun)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	start := time.Now()
	report, err := store.Import(opt)
	if err != nil {
		return err
	}
	elapsed := time.Since(start)

	if *asJSON {
		printJSONReport(report)
	} else {
		printImportReport(report, elapsed, *dryRun)
	}

	// 经典语料字集/条目统计（docs/39 巡检增强）：仅写库后报告真实落库结果，
	// dry-run 未落库，反映的会是旧库状态，故跳过以免误导。
	if !*dryRun && !*asJSON {
		printClassicsReport(store)
	}

	if report.Failed > 0 {
		return fmt.Errorf("有 %d 项物料导入失败", report.Failed)
	}
	if !*asJSON {
		fmt.Println("\n提示: 运行时统一从 SQLite 读取物料，修改源文件后需重新执行 import 方可生效。")
	}
	return nil
}

// printImportReport 以表格形式输出导入报告
func printImportReport(report *sqlite.ImportReport, elapsed time.Duration, dryRun bool) {
	if dryRun {
		fmt.Println("=== 试运行（未写库）===")
	} else {
		fmt.Println("=== 导入完成 ===")
	}
	// 耗时列（docs/40 B6）：物料级耗时下钻，此前只有汇总耗时，慢物料无法定位。
	fmt.Printf("%-16s %-34s %8s  %10s  %s\n", "物料", "版本", "行数", "耗时", "结果")
	fmt.Println(strings.Repeat("-", 104))
	for _, r := range report.Results {
		result := "已导入"
		switch {
		case r.Err != "":
			result = "失败：" + r.Err
		case r.Skipped:
			result = "跳过：" + r.Reason
		}
		fmt.Printf("%-16s %-34s %8d  %10s  %s\n",
			r.Material, truncate(r.Version, 34), r.Rows, r.Elapsed, result)
	}
	fmt.Println(strings.Repeat("-", 104))
	if report.PartialCommitted {
		fmt.Println("\n⚠ 部分提交：物料本体已落库，但派生维护失败 —— 建议 --lenient 重建修复派生列")
	}
	fmt.Printf("合计: 写入 %d 项 / 跳过 %d 项 / 失败 %d 项，耗时 %s\n",
		report.Written, report.Skipped, report.Failed, elapsed.Round(time.Millisecond))
}

// printClassicsReport 打印经典语料字集/条目统计，并标记「已登记物料但字集缺失」的静默遗漏。
//
// 这是 docs/39 的巡检增强：此前元曲/乐府/辞赋仅在诗词展示索引（classics_poem_entries）
// 出现，却未进入出典加分字集（classics_extracts），属于「索引齐全但字集缺失」，
// 巡检中无任何信号。现逐 material 核对字集行数，缺失即告警，使此类遗漏可被肉眼发现。
func printClassicsReport(s *sqlite.Store) {
	stats := s.ClassicsStats()
	if len(stats.ExtractByMaterial) == 0 && len(stats.PoemBySource) == 0 {
		fmt.Println("\n（经典语料未导入：classics_extracts / classics_poem_entries 均为空，请先执行 materials import）")
		return
	}

	fmt.Println("\n=== 经典字集提取数 (classics_extracts) ===")
	var missing []string
	for _, m := range classics.RegisteredMaterials() {
		n := stats.ExtractByMaterial[m]
		mark := ""
		if n == 0 {
			mark = "  ⚠ 缺失"
			missing = append(missing, m)
		}
		fmt.Printf("  %-22s %8d%s\n", m, n, mark)
	}
	fmt.Printf("  合计字集条目: %d\n", stats.TotalExtracts)

	fmt.Println("\n=== 经典诗词条目数 (classics_poem_entries) ===")
	for _, src := range sortedStringKeys(stats.PoemBySource) {
		fmt.Printf("  %-12s %8d\n", src, stats.PoemBySource[src])
	}
	fmt.Printf("  合计诗词条目: %d\n", stats.TotalPoems)

	if len(missing) > 0 {
		fmt.Printf("\n⚠ 以下已登记字集物料在库中字集行数为 0（出典加分静默遗漏风险）：%s\n", strings.Join(missing, ", "))
	}
}

// sortedStringKeys 返回 map 的键按字典序排序后的切片（巡检报告稳定输出用）
func sortedStringKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// runLoad 从库装载物料到进程内内存索引（运行期装载路径的独立验证入口）
//
// 与 import 的分工：import 负责「JSON → 库」，load 负责「库 → 内存」。
// 服务启动时两者都会发生（load 走 data.InitWithDB）；本子命令把装载单独
// 暴露出来，用于导入后自检与 CI 校验「库中物料是否足以支撑完整起名」。
//
// --data 仍需给出：经典语料的物化在导入期完成，但本命令用于校验时
// 若库缺经典语料会明确报错，不会回落 JSON。
func runLoad(args []string) error {
	fs := flag.NewFlagSet("load", flag.ExitOnError)
	dbPath := fs.String("db", "", "SQLite 库路径（默认读配置 database.path 的绝对路径）")
	dataDir := fs.String("data", "./data", "物料源目录（仅用于错误提示，不读 JSON）")
	asJSON := fs.Bool("json", false, "以 JSON 输出装载报告（供 CI 消费）")
	if err := fs.Parse(args); err != nil {
		return err
	}

	store, err := openStore(resolveDBPath(*dbPath), *dataDir, false)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	start := time.Now()
	if err := data.InitWithDB(*dataDir, store); err != nil {
		return fmt.Errorf("装载失败: %w", err)
	}
	report := store.LoadMaterialsFromDBWithReport()
	elapsed := time.Since(start)

	if *asJSON {
		return printJSON(map[string]any{"loaded": report.Loaded, "missing": report.Missing})
	}

	fmt.Println("=== 装载完成 ===")
	for _, name := range report.Loaded {
		fmt.Printf("  已装载 %s\n", name)
	}
	if len(report.Missing) > 0 {
		fmt.Println("  缺失物料（可降级）:")
		for _, name := range report.Missing {
			fmt.Printf("    - %s\n", name)
		}
	}
	fmt.Printf("合计: 装载 %d 项 / 缺失 %d 项，耗时 %s\n",
		len(report.Loaded), len(report.Missing), elapsed.Round(time.Millisecond))
	return nil
}

func runStatus(args []string) error {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	dbPath := fs.String("db", "", "SQLite 库路径（默认读配置 database.path 的绝对路径）")
	if err := fs.Parse(args); err != nil {
		return err
	}

	store, err := openStore(resolveDBPath(*dbPath), "", false)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	states, err := store.MaterialStates()
	if err != nil {
		return err
	}
	fmt.Printf("%-16s %-24s %10s  %s\n", "物料", "版本", "行数", "指纹(前16位)")
	fmt.Println(strings.Repeat("-", 76))
	for _, st := range states {
		fmt.Printf("%-16s %-24s %10d  %s\n", st.Material, truncate(st.Version, 24), st.RowCount, truncate(st.Checksum, 16))
	}
	if len(states) == 0 {
		fmt.Println("(库中暂无物料记录，请先执行 materials import)")
	}

	// 经典语料字集/条目统计（docs/39 巡检增强）：逐 material 核对字集是否落库，
	// 标记「已登记物料但字集缺失」的静默遗漏（元曲/乐府/辞赋曾属此类）。
	printClassicsReport(store)
	return nil
}

// manifestFileRow 入库清单中单个源文件的存在性状态（JSON 输出用）
type manifestFileRow struct {
	Material string `json:"material"`
	File     string `json:"file"`
	Required bool   `json:"required"`
	Exists   bool   `json:"exists"`
}

// runManifest 打印/校验入库源清单（docs/40）
//
// 清单来自 schema.Definitions（单文件 Source / 多文件 Files），不做目录扫描，
// 因此目录里多出的 JSON 不会被误纳入；--check 只校验必需源文件是否齐全。
func runManifest(args []string) error {
	fs := flag.NewFlagSet("manifest", flag.ExitOnError)
	dataDir := fs.String("data", "./data", "物料源目录")
	check := fs.Bool("check", false, "校验清单内必需源文件是否齐全（缺失即失败）")
	asJSON := fs.Bool("json", false, "以 JSON 输出（供 CI 消费）")
	if err := fs.Parse(args); err != nil {
		return err
	}

	entries := schema.Manifest()
	rows := make([]manifestFileRow, 0, len(schema.AllSourceFiles()))
	unique := make(map[string]bool)
	missing := make(map[string]bool)
	for _, e := range entries {
		for _, f := range e.Files {
			_, err := os.Stat(filepath.Join(*dataDir, filepath.FromSlash(f)))
			exists := err == nil
			rows = append(rows, manifestFileRow{
				Material: e.Material,
				File:     f,
				Required: e.Required,
				Exists:   exists,
			})
			unique[f] = true
			if e.Required && !exists {
				missing[f] = true
			}
		}
	}

	if *asJSON {
		if err := printJSON(rows); err != nil {
			return err
		}
	} else {
		fmt.Printf("入库源清单：%d 个物料、%d 个唯一文件\n", len(entries), len(unique))
		fmt.Printf("%-20s %-6s %-6s %s\n", "物料", "必需", "状态", "源文件")
		fmt.Println(strings.Repeat("-", 72))
		for _, r := range rows {
			required, status := "-", "✓"
			if r.Required {
				required = "是"
			}
			if !r.Exists {
				status = "✗"
			}
			fmt.Printf("%-20s %-6s %-6s %s\n", r.Material, required, status, r.File)
		}
	}

	if !*check {
		return nil
	}
	if len(missing) > 0 {
		names := make([]string, 0, len(missing))
		for f := range missing {
			names = append(names, f)
		}
		sort.Strings(names)
		return fmt.Errorf("必需源文件缺失 %d 项: %s", len(names), strings.Join(names, ", "))
	}
	if !*asJSON {
		fmt.Printf("\n校验通过：清单内必需源文件齐全（%d 个唯一文件）\n", len(unique))
	}
	return nil
}

// openStore 打开物料库
//
// import 模式需要写权限（写物料与状态）；status 模式以只读打开，
// 避免为了查看状态而意外创建空库
func openStore(dbPath, dataDir string, writable bool) (*sqlite.Store, error) {
	if writable {
		if dir := filepath.Dir(dbPath); dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("创建库目录失败: %w", err)
			}
		}
	}
	return sqlite.NewStore(dbPath, dataDir)
}

// resolveDBPath 把 --db 的空默认值收敛到单一权威库路径（docs/36 双库分叉治理）。
//
// 优先级：显式 --db > 配置文件 database.path（现为绝对路径）> 兜底 backend/namer.db。
// 这样无论从仓库根还是 backend/ 目录运行 materials 子命令，都连同一个库，
// 杜绝 "./namer.db" 相对路径随 cwd 漂移造成双库分叉。配置读取失败或路径非绝对时
// 回落到 backend/namer.db（相对当前 cwd 解析为 <根>/backend/namer.db）。
func resolveDBPath(dbPath string) string {
	if dbPath != "" {
		return dbPath
	}
	if cfg, err := config.Load(""); err == nil && cfg.Database.Path != "" && filepath.IsAbs(cfg.Database.Path) {
		return cfg.Database.Path
	}
	return filepath.Join("backend", "namer.db")
}

func splitList(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 3 {
		return string(r[:n])
	}
	return string(r[:n-3]) + "..."
}

// 输出 JSON 报告（供 CI 消费）
func printJSONReport(report *sqlite.ImportReport) {
	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "报告序列化失败: %v\n", err)
		return
	}
	fmt.Println(string(b))
}

// printJSON 以 JSON 输出任意值（供 CI 消费）
func printJSON(v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("报告序列化失败: %w", err)
	}
	fmt.Println(string(b))
	return nil
}

// —— 订阅管线子命令（docs/33 门禁层 M5）——

// runLockInit 生成门禁物料订阅清单
func runLockInit(args []string) error {
	fs := flag.NewFlagSet("lock-init", flag.ExitOnError)
	dataDir := fs.String("data", "./data", "物料源目录")
	out := fs.String("out", "./data/materials.lock", "lock 输出路径")
	version := fs.String("version", "1.0", "lock 版本号")
	sourceSpec := fs.String("source", "", "上游来源，格式 material=url,material=url")
	if err := fs.Parse(args); err != nil {
		return err
	}

	sources := make(map[string]string)
	for _, kv := range splitList(*sourceSpec) {
		if i := strings.Index(kv, "="); i > 0 {
			sources[kv[:i]] = kv[i+1:]
		}
	}

	lock, err := subscribe.BuildLock(*dataDir, *version, sources)
	if err != nil {
		return err
	}
	if err := subscribe.SaveLock(*out, lock); err != nil {
		return err
	}
	fmt.Printf("已生成 %s：登记 %d 项门禁物料\n", *out, len(lock.Entries))
	for _, e := range lock.Entries {
		fmt.Printf("  %-20s %s  %s\n", e.Material, truncate(e.ArtifactSHA256, 16), e.Generated)
	}
	return nil
}

// runLockVerify 校验 lock 中登记的产物指纹
func runLockVerify(args []string) error {
	fs := flag.NewFlagSet("lock-verify", flag.ExitOnError)
	lockPath := fs.String("lock", "./data/materials.lock", "lock 路径")
	dataDir := fs.String("data", "./data", "物料源目录")
	if err := fs.Parse(args); err != nil {
		return err
	}

	lock, err := subscribe.LoadLock(*lockPath)
	if err != nil {
		return err
	}
	mismatches := lock.Verify(*dataDir)
	if len(mismatches) == 0 {
		fmt.Printf("lock 校验通过：%d 项物料指纹一致\n", len(lock.Entries))
		return nil
	}
	fmt.Printf("lock 校验发现 %d 项不一致：\n", len(mismatches))
	for _, m := range mismatches {
		fmt.Printf("  %-20s %s  期望 %s 实际 %s\n", m.Material, m.Reason, truncate(m.Want, 16), truncate(m.Got, 16))
	}
	return fmt.Errorf("lock 校验未通过")
}

// runLockDiff 对两版本文本做行差分并生成补丁
func runLockDiff(args []string) error {
	fs := flag.NewFlagSet("lock-diff", flag.ExitOnError)
	oldPath := fs.String("old", "", "旧版本文本（缺省视为空）")
	newPath := fs.String("new", "", "新版本文本（每行一个条目）")
	material := fs.String("material", "", "目标物料名")
	kind := fs.String("kind", "", "条目类别（allowlist/compliance_redline 必填）")
	out := fs.String("out", "", "补丁输出路径（缺省打印到标准输出）")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *newPath == "" || *material == "" {
		return fmt.Errorf("必须指定 --new 与 --material")
	}

	oldLines, err := readLinesFile(*oldPath)
	if err != nil {
		return err
	}
	newLines, err := readLinesFile(*newPath)
	if err != nil {
		return err
	}
	d := subscribe.DiffLines(oldLines, newLines)
	entries := subscribe.BuildPatch(*material, *kind, d)
	b, err := subscribe.FormatPatch(entries)
	if err != nil {
		return err
	}
	if *out == "" {
		fmt.Print(string(b))
		return nil
	}
	if err := os.WriteFile(*out, b, 0o644); err != nil {
		return fmt.Errorf("写入补丁失败: %w", err)
	}
	fmt.Printf("已生成补丁 %s：新增 %d / 移除 %d\n", *out, len(d.Added), len(d.Removed))
	return nil
}

// runPatch 把补丁应用到 data 物料 JSON（--rollback 反向应用即回滚）
func runPatch(args []string) error {
	fs := flag.NewFlagSet("patch", flag.ExitOnError)
	file := fs.String("file", "", "补丁文件（JSONL）")
	dataDir := fs.String("data", "./data", "物料源目录")
	rollback := fs.Bool("rollback", false, "反向应用补丁（回滚）")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *file == "" {
		return fmt.Errorf("必须指定 --file")
	}

	b, err := os.ReadFile(*file)
	if err != nil {
		return fmt.Errorf("读取补丁失败: %w", err)
	}
	entries, err := subscribe.ParsePatch(b)
	if err != nil {
		return err
	}
	if *rollback {
		entries = subscribe.Invert(entries)
	}

	byMaterial := make(map[string][]subscribe.PatchEntry)
	var order []string
	for _, e := range entries {
		if _, ok := byMaterial[e.Material]; !ok {
			order = append(order, e.Material)
		}
		byMaterial[e.Material] = append(byMaterial[e.Material], e)
	}
	for _, m := range order {
		if err := subscribe.ApplyToFile(*dataDir, m, byMaterial[m]); err != nil {
			return fmt.Errorf("应用物料 %s 失败: %w", m, err)
		}
		fmt.Printf("已应用物料 %s：%d 条操作\n", m, len(byMaterial[m]))
	}
	fmt.Println("提示: 数据文件已更新，请执行 materials import 使其生效。")
	return nil
}

func readLinesFile(path string) ([]string, error) {
	if path == "" {
		return nil, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取 %s 失败: %w", path, err)
	}
	return subscribe.ParseLines(string(b)), nil
}

// —— 灰度发布子命令（docs/33 门禁层 M7 / ADR-015）——

// runStage 登记物料候选版本（灰度暂存）
func runStage(args []string) error {
	fs := flag.NewFlagSet("stage", flag.ExitOnError)
	dataDir := fs.String("data", "./data", "物料源目录")
	lockPath := fs.String("lock", "./data/materials.lock", "lock 路径")
	material := fs.String("material", "", "目标物料名")
	from := fs.String("from", "", "候选文件路径（JSON）")
	source := fs.String("source", "", "候选来源说明")
	operator := fs.String("operator", "", "操作者标识")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *material == "" || *from == "" {
		return fmt.Errorf("必须指定 --material 与 --from")
	}
	rec, err := subscribe.Stage(*dataDir, *lockPath, *material, *from, *source, *operator)
	if err != nil {
		return err
	}
	fmt.Printf("已登记候选：%s  指纹 %s  （%s）\n", rec.Material, truncate(rec.To, 16), rec.Time)
	fmt.Println("提示: 执行 materials observe 查看差异预演，promote 提升生效，abort 丢弃候选。")
	return nil
}

// runObserve 查看候选相对生效版本的差异预演
func runObserve(args []string) error {
	fs := flag.NewFlagSet("observe", flag.ExitOnError)
	dataDir := fs.String("data", "./data", "物料源目录")
	lockPath := fs.String("lock", "./data/materials.lock", "lock 路径")
	material := fs.String("material", "", "目标物料名")
	limit := fs.Int("limit", 20, "每类差异最多显示条数（0=全部）")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *material == "" {
		return fmt.Errorf("必须指定 --material")
	}
	diff, err := subscribe.Observe(*dataDir, *lockPath, *material)
	if err != nil {
		return err
	}
	fmt.Printf("=== %s 候选预演 ===\n", diff.Material)
	fmt.Printf("生效指纹: %s\n候选指纹: %s\n", truncate(diff.Current, 16), truncate(diff.Candidate, 16))
	fmt.Printf("新增 %d 条 / 移除 %d 条\n", len(diff.Added), len(diff.Removed))
	printEntries("+ 新增", diff.Added, *limit)
	printEntries("- 移除", diff.Removed, *limit)
	return nil
}

// printEntries 打印条目明细（超限时提示省略数量）
func printEntries(title string, items []string, limit int) {
	if len(items) == 0 {
		return
	}
	fmt.Printf("%s:\n", title)
	shown := items
	if limit > 0 && len(items) > limit {
		shown = items[:limit]
	}
	for _, s := range shown {
		fmt.Printf("    %s\n", s)
	}
	if len(shown) < len(items) {
		fmt.Printf("    ...（其余 %d 条省略）\n", len(items)-len(shown))
	}
}

// runPromote 把候选提升为生效版本
func runPromote(args []string) error {
	fs := flag.NewFlagSet("promote", flag.ExitOnError)
	dataDir := fs.String("data", "./data", "物料源目录")
	lockPath := fs.String("lock", "./data/materials.lock", "lock 路径")
	material := fs.String("material", "", "目标物料名")
	operator := fs.String("operator", "", "操作者标识")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *material == "" {
		return fmt.Errorf("必须指定 --material")
	}
	rec, err := subscribe.Promote(*dataDir, *lockPath, *material, *operator)
	if err != nil {
		return err
	}
	fmt.Printf("已提升生效：%s  %s → %s\n", rec.Material, truncate(rec.From, 16), truncate(rec.To, 16))
	fmt.Println("提示: 数据文件已更新，请执行 materials import 使其生效；回退执行 materials abort --to <旧指纹>。")
	return nil
}

// runAbort 丢弃候选，或 --to <指纹> 回退生效版本
func runAbort(args []string) error {
	fs := flag.NewFlagSet("abort", flag.ExitOnError)
	dataDir := fs.String("data", "./data", "物料源目录")
	lockPath := fs.String("lock", "./data/materials.lock", "lock 路径")
	material := fs.String("material", "", "目标物料名")
	to := fs.String("to", "", "回退到该指纹（缺省仅丢弃候选）")
	operator := fs.String("operator", "", "操作者标识")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *material == "" {
		return fmt.Errorf("必须指定 --material")
	}
	rec, err := subscribe.Abort(*dataDir, *lockPath, *material, *to, *operator)
	if err != nil {
		return err
	}
	if rec.To == "" {
		fmt.Printf("已丢弃候选：%s\n", rec.Material)
		return nil
	}
	fmt.Printf("已回退：%s  %s → %s\n", rec.Material, truncate(rec.From, 16), truncate(rec.To, 16))
	fmt.Println("提示: 数据文件已更新，请执行 materials import 使其生效。")
	return nil
}

// —— 复审队列子命令（docs/33 门禁层 M6）——

// runReview 分发 review 子命令：list | decide | export
func runReview(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("review 需要子命令: list | decide | export")
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "list":
		return runReviewList(rest)
	case "decide":
		return runReviewDecide(rest)
	case "export":
		return runReviewExport(rest)
	default:
		return fmt.Errorf("未知 review 子命令: %s（可用 list | decide | export）", sub)
	}
}

// runReviewList 列出复审候选
func runReviewList(args []string) error {
	fs := flag.NewFlagSet("review list", flag.ExitOnError)
	dbPath := fs.String("db", "", "SQLite 库路径（默认读配置 database.path 的绝对路径）")
	status := fs.String("status", "", "按状态过滤：pending|confirmed|false_positive|border（缺省全部）")
	limit := fs.Int("limit", 0, "最多显示条数（0=全部）")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *status != "" && !fate.ValidReviewStatus(fate.ReviewStatus(*status)) {
		return fmt.Errorf("非法状态: %s", *status)
	}

	store, err := openStore(resolveDBPath(*dbPath), "", true)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	items, err := store.ListGateReviews(fate.ReviewStatus(*status), *limit)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		fmt.Println("(无复审条目)")
		return nil
	}
	fmt.Printf("%-38s %-14s %-12s %-8s %-14s %s\n", "ID", "状态", "名字", "层级", "类别", "命中值")
	fmt.Println(strings.Repeat("-", 96))
	for _, it := range items {
		fmt.Printf("%-38s %-14s %-12s %-8s %-14s %s\n",
			it.ID, it.Status, it.Name, it.Layer, truncate(it.Category, 14), it.HitValue)
	}
	return nil
}

// runReviewDecide 落定一条复审结论
func runReviewDecide(args []string) error {
	fs := flag.NewFlagSet("review decide", flag.ExitOnError)
	dbPath := fs.String("db", "", "SQLite 库路径（默认读配置 database.path 的绝对路径）")
	id := fs.String("id", "", "复审条目 ID")
	status := fs.String("status", "", "结论：confirmed|false_positive|border")
	note := fs.String("note", "", "备注")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *id == "" || *status == "" {
		return fmt.Errorf("必须指定 --id 与 --status")
	}
	if !fate.ValidReviewStatus(fate.ReviewStatus(*status)) || *status == string(fate.ReviewPending) {
		return fmt.Errorf("非法结论: %s（可用 confirmed|false_positive|border）", *status)
	}

	store, err := openStore(resolveDBPath(*dbPath), "", true)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	if err := store.DecideGateReview(*id, fate.ReviewStatus(*status), *note); err != nil {
		return err
	}
	fmt.Printf("已落定复审 %s → %s\n", *id, *status)
	return nil
}

// runReviewExport 把已落定条目导出为 gate_patch.jsonl
func runReviewExport(args []string) error {
	fs := flag.NewFlagSet("review export", flag.ExitOnError)
	dbPath := fs.String("db", "", "SQLite 库路径（默认读配置 database.path 的绝对路径）")
	out := fs.String("out", "", "补丁输出路径（缺省打印到标准输出）")
	if err := fs.Parse(args); err != nil {
		return err
	}

	store, err := openStore(resolveDBPath(*dbPath), "", true)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	var items []fate.ReviewItem
	for _, st := range []fate.ReviewStatus{fate.ReviewConfirmed, fate.ReviewFalsePositive} {
		got, err := store.ListGateReviews(st, 0)
		if err != nil {
			return err
		}
		items = append(items, got...)
	}
	fate.SortReviews(items)

	entries, err := review.ToPatch(items)
	if err != nil {
		return err
	}
	b, err := subscribe.FormatPatch(entries)
	if err != nil {
		return err
	}
	if *out == "" {
		fmt.Print(string(b))
		return nil
	}
	if err := os.WriteFile(*out, b, 0o644); err != nil {
		return fmt.Errorf("写入补丁失败: %w", err)
	}
	fmt.Printf("已导出补丁 %s：共 %d 条（已落定 %d 条）\n", *out, len(entries), len(items))
	return nil
}
