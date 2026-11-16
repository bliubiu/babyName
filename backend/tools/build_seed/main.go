// build_seed —— 生成单文件种子库 data/namer.db（ADR-010「A + 单库」）
//
// 交付形态决策：只产出一个二进制 namer.db，不产 materials.sql，也不维护
// 「种子库 + 用户库」双文件。测试、CI 门禁、桌面运行期、NSIS 安装包共用
// 这一份库；JSON/CSV 只在本工具的构建期作为输入被读取一次。
//
// 用法（在 backend/ 目录下执行）：
//
//	go run ./tools/build_seed                         # 从 ../data 生成 ../data/namer.db
//	go run ./tools/build_seed --verify-only           # 只校验现有种子库，不重新生成
//	go run ./tools/build_seed --data <源> --out <库>  # 自定义路径
//
// 生成流程：
//  1. 清理目标库及 -wal/-shm 残留，保证是干净的一次性构建；
//  2. sqlite.NewStore + Import 全量导入 12 类物料（与 materials import 同实现）；
//  3. 关闭双连接后，单连接执行 journal_mode=DELETE + VACUUM，收敛成单文件；
//     WAL 头的库在只读打开时可能要求 -wal/-shm，分发态必须消掉；
//  4. 只读自检：integrity_check、必需表非空、用户表为空、输出 sha256。
//
// 修改任何 data/*.json 后必须重跑本工具并重新提交 data/namer.db，
// 否则运行期读到的仍是旧物料（ADR-008 不回落 JSON 的必然结果）。
package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
	"name/internal/infrastructure/database/sqlite"
)

// requiredTables 种子库必须非空的表（与 schema.Definitions 的物料集对应）
var requiredTables = []string{
	"hanzi_data", "kangxi_strokes", "name_frequency", "name_bigram",
	"naming_gate", "forbidden_combos", "allowlist", "compliance_redline",
	"yijing_hexagrams", "zodiac", "classics_extracts", "classics_poem_entries",
	"curated_names",
}

// userTables 允许存在但必须为空的用户数据表（种子库不得夹带本机数据）
var userTables = []string{"favorites", "history", "gate_review", "name_feedback", "shares"}

type report struct {
	OK           bool              `json:"ok"`
	Mode         string            `json:"mode"`
	DBPath       string            `json:"dbPath"`
	Bytes        int64             `json:"bytes"`
	SHA256       string            `json:"sha256"`
	JournalMode  string            `json:"journalMode"`
	Integrity    string            `json:"integrity"`
	Rows         map[string]int    `json:"rows"`
	UserRows     map[string]int    `json:"userRows"`
	Materials    int               `json:"materials,omitempty"`
	ImportFailed int               `json:"importFailed,omitempty"`
	Elapsed      string            `json:"elapsed"`
	Error        string            `json:"error,omitempty"`
}

func main() {
	dataDir := flag.String("data", "../data", "JSON/CSV 源物料目录")
	out := flag.String("out", "../data/namer.db", "输出种子库路径")
	verifyOnly := flag.Bool("verify-only", false, "只校验现有种子库，不重新生成")
	flag.Parse()

	start := time.Now()
	var r report
	if *verifyOnly {
		r = verify(*out)
		r.Mode = "verify-only"
	} else {
		r = build(*dataDir, *out)
		r.Mode = "build"
	}
	r.Elapsed = time.Since(start).Round(time.Millisecond).String()

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(r); err != nil {
		fmt.Fprintf(os.Stderr, "输出报告失败: %v\n", err)
		os.Exit(1)
	}
	if !r.OK {
		os.Exit(1)
	}
}

func build(dataDir, out string) report {
	absOut, err := filepath.Abs(out)
	if err != nil {
		return report{DBPath: out, Error: "解析输出路径失败: " + err.Error()}
	}
	if info, err := os.Stat(dataDir); err != nil || !info.IsDir() {
		return report{DBPath: absOut, Error: "源物料目录不存在: " + dataDir}
	}

	// 1. 清理旧产物（仅本工具自己生成的库文件，不触碰任何源文件）
	for _, p := range []string{absOut, absOut + "-wal", absOut + "-shm"} {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return report{DBPath: absOut, Error: "清理旧种子库失败: " + err.Error()}
		}
	}
	if err := os.MkdirAll(filepath.Dir(absOut), 0o755); err != nil {
		return report{DBPath: absOut, Error: "创建输出目录失败: " + err.Error()}
	}

	// 2. 全量导入（与 cmd/materials import 同一实现）
	store, err := sqlite.NewStore(absOut, dataDir)
	if err != nil {
		return report{DBPath: absOut, Error: "打开 SQLite 失败: " + err.Error()}
	}
	res, err := store.Import(sqlite.ImportOptions{DataDir: dataDir})
	if err != nil {
		_ = store.Close()
		return report{DBPath: absOut, Error: "物料导入失败: " + err.Error()}
	}
	if err := store.Close(); err != nil {
		return report{DBPath: absOut, Error: "关闭导入连接失败: " + err.Error()}
	}
	if res.Failed > 0 {
		return report{DBPath: absOut, Materials: len(res.Results), ImportFailed: res.Failed,
			Error: fmt.Sprintf("有 %d 项物料导入失败", res.Failed)}
	}

	// 3. 单文件封存：消除 WAL 头 + VACUUM 压实（实测 24.1MB → 22.1MB）
	db, err := sql.Open("sqlite", absOut)
	if err != nil {
		return report{DBPath: absOut, Error: "打开封存连接失败: " + err.Error()}
	}
	if _, err := db.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		_ = db.Close()
		return report{DBPath: absOut, Error: "WAL checkpoint 失败: " + err.Error()}
	}
	if _, err := db.Exec("PRAGMA journal_mode=DELETE"); err != nil {
		_ = db.Close()
		return report{DBPath: absOut, Error: "切换 journal_mode 失败: " + err.Error()}
	}
	if _, err := db.Exec("VACUUM"); err != nil {
		_ = db.Close()
		return report{DBPath: absOut, Error: "VACUUM 失败: " + err.Error()}
	}
	if err := db.Close(); err != nil {
		return report{DBPath: absOut, Error: "关闭封存连接失败: " + err.Error()}
	}
	// 清理可能残留的 sidecar，交付物必须是单文件
	for _, p := range []string{absOut + "-wal", absOut + "-shm"} {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return report{DBPath: absOut, Error: "清理 WAL 残留失败: " + err.Error()}
		}
	}

	r := verify(absOut)
	r.Materials = len(res.Results)
	r.ImportFailed = res.Failed
	return r
}

func verify(out string) report {
	absOut, err := filepath.Abs(out)
	if err != nil {
		return report{DBPath: out, Error: "解析库路径失败: " + err.Error()}
	}
	r := report{DBPath: absOut, Rows: map[string]int{}, UserRows: map[string]int{}}

	info, err := os.Stat(absOut)
	if err != nil || info.IsDir() || info.Size() == 0 {
		r.Error = "种子库不存在或为空: " + absOut
		return r
	}
	r.Bytes = info.Size()

	// 交付必须单文件：有 -wal/-shm 说明封存不完整，CI 应红
	for _, suffix := range []string{"-wal", "-shm"} {
		if _, err := os.Stat(absOut + suffix); err == nil {
			r.Error = "种子库存在 WAL 残留文件 " + suffix + "，请重跑 build_seed"
			return r
		}
	}

	db, err := sql.Open("sqlite", absOut+"?mode=ro")
	if err != nil {
		r.Error = "只读打开种子库失败: " + err.Error()
		return r
	}
	defer func() { _ = db.Close() }()

	if err := db.QueryRow("PRAGMA integrity_check").Scan(&r.Integrity); err != nil {
		r.Error = "integrity_check 失败: " + err.Error()
		return r
	}
	if r.Integrity != "ok" {
		r.Error = "integrity_check = " + r.Integrity
		return r
	}
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&r.JournalMode); err != nil {
		r.Error = "读取 journal_mode 失败: " + err.Error()
		return r
	}
	if r.JournalMode != "delete" {
		r.Error = "journal_mode = " + r.JournalMode + "，分发态必须为 delete"
		return r
	}

	for _, table := range requiredTables {
		var n int
		if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil {
			r.Error = "查询 " + table + " 失败: " + err.Error()
			return r
		}
		r.Rows[table] = n
		if n == 0 {
			r.Error = "必需物料表为空: " + table
			return r
		}
	}
	for _, table := range userTables {
		var exists int
		if err := db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&exists); err != nil {
			r.Error = "查询表清单失败: " + err.Error()
			return r
		}
		if exists == 0 {
			continue
		}
		var n int
		if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil {
			r.Error = "查询 " + table + " 失败: " + err.Error()
			return r
		}
		r.UserRows[table] = n
		if n > 0 {
			r.Error = "种子库夹带了用户数据: " + table
			return r
		}
	}

	f, err := os.Open(absOut)
	if err != nil {
		r.Error = "读取库文件失败: " + err.Error()
		return r
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		r.Error = "计算 sha256 失败: " + err.Error()
		return r
	}
	r.SHA256 = hex.EncodeToString(h.Sum(nil))
	r.OK = true
	return r
}
