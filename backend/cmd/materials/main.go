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
//	namer materials status  --db ./namer.db                        # 查看物料版本与行数
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"name/internal/infrastructure/database/sqlite"
	"name/internal/materials/schema"
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
	case "status":
		err = runStatus(args)
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
  materials status  [选项]     列出各物料的版本、指纹与行数

import 选项:
  --data    string   物料源目录（默认 ./data）
  --db      string   SQLite 库路径（默认 ./namer.db）
  --only    string   只导入指定物料，逗号分隔（默认全部）
  --lenient          校验问题仅记录不阻断（数据清洗期取证用）
  --dry-run          只校验与统计，不写库

status 选项:
  --db      string   SQLite 库路径（默认 ./namer.db）

说明:
  · 写入语义为「单事务 DELETE + INSERT 全量重建」，同输入多次导入结果等价（ADR-011）
  · 指纹未变的物料会短路跳过，避免数十 MB 物料的无谓重灌（ADR-014）
  · 导入完成后运行时统一从 SQLite 读取，物料变更需重新 import 才生效
  · 物料清单: ` + strings.Join(schema.Names(), ", "))
}

func runImport(args []string) error {
	fs := flag.NewFlagSet("import", flag.ExitOnError)
	dataDir := fs.String("data", "./data", "物料源目录")
	dbPath := fs.String("db", "./namer.db", "SQLite 库路径")
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

	store, err := openStore(*dbPath, *dataDir, !*dryRun)
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
	fmt.Printf("%-16s %-34s %8s  %s\n", "物料", "版本", "行数", "结果")
	fmt.Println(strings.Repeat("-", 92))
	for _, r := range report.Results {
		result := "已导入"
		switch {
		case r.Err != "":
			result = "失败：" + r.Err
		case r.Skipped:
			result = "跳过：" + r.Reason
		}
		fmt.Printf("%-16s %-34s %8d  %s\n", r.Material, truncate(r.Version, 34), r.Rows, result)
	}
	fmt.Println(strings.Repeat("-", 92))
	fmt.Printf("合计: 写入 %d 项 / 跳过 %d 项 / 失败 %d 项，耗时 %s\n",
		report.Written, report.Skipped, report.Failed, elapsed.Round(time.Millisecond))
}

func runStatus(args []string) error {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	dbPath := fs.String("db", "./namer.db", "SQLite 库路径")
	if err := fs.Parse(args); err != nil {
		return err
	}

	store, err := openStore(*dbPath, "", false)
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
