package sqlite

import (
	"database/sql"
	"fmt"
)

// 本文件定义物料的规范化表结构（docs/31 ADR-008 / ADR-010）。
//
// 设计要点：
//   - 每种物料一张主表，主键与业务键对齐（char / name / key）；
//   - 所有跨物料可查询的字段都建成独立列，不再依赖 Go 端回查内存二次过滤；
//   - 建表语句幂等（IF NOT EXISTS），缺列由迁移补齐，两条路径都要能跑。

// materialTableDDL 物料表的建表语句（顺序即导入顺序无关，仅为可读性）
var materialTableDDL = []string{
	// 康熙笔画：数理与卦象的笔画口径（AGENTS.md 要求「数理用康熙」）
	`CREATE TABLE IF NOT EXISTS kangxi_strokes (
		char    TEXT PRIMARY KEY,
		strokes INTEGER NOT NULL DEFAULT 0
	)`,

	// 通用规范汉字表：运行时权威字库
	`CREATE TABLE IF NOT EXISTS hanzi_data (
		char       TEXT PRIMARY KEY,
		pinyin     TEXT NOT NULL DEFAULT '',
		strokes    INTEGER NOT NULL DEFAULT 0,
		kangxi_strokes INTEGER NOT NULL DEFAULT 0,
		radical    TEXT NOT NULL DEFAULT '',
		meaning    TEXT NOT NULL DEFAULT '',
		wuxing     TEXT NOT NULL DEFAULT '',
		gender     TEXT NOT NULL DEFAULT '',
		level      INTEGER NOT NULL DEFAULT 0,
		tone       INTEGER NOT NULL DEFAULT 0,
		name_freq_tier INTEGER NOT NULL DEFAULT 0,
		gender_tags     TEXT NOT NULL DEFAULT '[]',
		style_tags      TEXT NOT NULL DEFAULT '[]',
		usage_level     INTEGER NOT NULL DEFAULT 0,
		positive_score  INTEGER NOT NULL DEFAULT 0,
		phonetic_score  INTEGER NOT NULL DEFAULT 0,
		modern_score    INTEGER NOT NULL DEFAULT 0,
		classical_score INTEGER NOT NULL DEFAULT 0,
		is_polyphonic   INTEGER NOT NULL DEFAULT 0,
		is_rare         INTEGER NOT NULL DEFAULT 0,
		is_negative     INTEGER NOT NULL DEFAULT 0,
		pair_blacklist  TEXT NOT NULL DEFAULT '[]',
		curation_level  INTEGER NOT NULL DEFAULT 0,
		name_penalty    INTEGER NOT NULL DEFAULT 0,
		naming_categories TEXT NOT NULL DEFAULT '[]',
		updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE INDEX IF NOT EXISTS idx_hanzi_data_wuxing ON hanzi_data(wuxing)`,
	`CREATE INDEX IF NOT EXISTS idx_hanzi_data_strokes ON hanzi_data(strokes)`,
	`CREATE INDEX IF NOT EXISTS idx_hanzi_data_pinyin ON hanzi_data(pinyin)`,
	`CREATE INDEX IF NOT EXISTS idx_hanzi_data_level ON hanzi_data(level)`,
	`CREATE INDEX IF NOT EXISTS idx_hanzi_data_gender ON hanzi_data(gender)`,

	// 偏旁分组（实测 29 组小表）
	`CREATE TABLE IF NOT EXISTS char_groups (
		radical TEXT PRIMARY KEY,
		name    TEXT NOT NULL DEFAULT '',
		meaning TEXT NOT NULL DEFAULT '',
		chars   TEXT NOT NULL DEFAULT '[]'
	)`,

	// 词典释义点查表：替代运行时 26MB 的 word.json 急载
	`CREATE TABLE IF NOT EXISTS words (
		char        TEXT PRIMARY KEY,
		explanation TEXT NOT NULL DEFAULT ''
	)`,

	// 单字人名频率
	`CREATE TABLE IF NOT EXISTS name_frequency (
		char  TEXT PRIMARY KEY,
		count INTEGER NOT NULL DEFAULT 0,
		rank  INTEGER NOT NULL DEFAULT 0,
		tier  INTEGER NOT NULL DEFAULT 0
	)`,

	// 双字人名频率
	`CREATE TABLE IF NOT EXISTS name_bigram (
		key   TEXT PRIMARY KEY,
		char1 TEXT NOT NULL DEFAULT '',
		char2 TEXT NOT NULL DEFAULT '',
		count INTEGER NOT NULL DEFAULT 0,
		rank  INTEGER NOT NULL DEFAULT 0,
		tier  INTEGER NOT NULL DEFAULT 0
	)`,

	// 命名质量门禁字表（候选池阶段硬剔除）
	`CREATE TABLE IF NOT EXISTS naming_gate (
		char TEXT PRIMARY KEY
	)`,

	// 禁忌双字组合
	`CREATE TABLE IF NOT EXISTS forbidden_combos (
		combo TEXT PRIMARY KEY
	)`,

	// 易经六十四卦
	`CREATE TABLE IF NOT EXISTS yijing_hexagrams (
		id             INTEGER PRIMARY KEY,
		name           TEXT NOT NULL DEFAULT '',
		number         INTEGER NOT NULL DEFAULT 0,
		symbol         TEXT NOT NULL DEFAULT '',
		upper_trigram  INTEGER NOT NULL DEFAULT 0,
		lower_trigram  INTEGER NOT NULL DEFAULT 0,
		gua_ci         TEXT NOT NULL DEFAULT '',
		xiang_ci       TEXT NOT NULL DEFAULT '',
		yao_ci         TEXT NOT NULL DEFAULT '[]',
		interpretation TEXT NOT NULL DEFAULT ''
	)`,

	// 十二生肖
	`CREATE TABLE IF NOT EXISTS zodiac (
		id             INTEGER PRIMARY KEY,
		name           TEXT NOT NULL DEFAULT '',
		wuxing         TEXT NOT NULL DEFAULT '',
		compatible     TEXT NOT NULL DEFAULT '[]',
		conflicting    TEXT NOT NULL DEFAULT '[]',
		avoid_chars    TEXT NOT NULL DEFAULT '',
		lucky_numbers  TEXT NOT NULL DEFAULT '[]'
	)`,

	// 物料版本与指纹
	`CREATE TABLE IF NOT EXISTS materials_state (
		material   TEXT PRIMARY KEY,
		version    TEXT NOT NULL DEFAULT '',
		checksum   TEXT NOT NULL DEFAULT '',
		row_count  INTEGER NOT NULL DEFAULT 0,
		imported_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`,
}

// materialTables 全部物料表名（守护测试与文档用）
var materialTables = []string{
	"kangxi_strokes", "hanzi_data", "char_groups", "words",
	"name_frequency", "name_bigram", "naming_gate", "forbidden_combos",
	"yijing_hexagrams", "zodiac", "materials_state",
}

// createMaterialTables 建立全部物料表（幂等）
func createMaterialTables(db *sql.DB) error {
	for _, ddl := range materialTableDDL {
		if _, err := db.Exec(ddl); err != nil {
			return fmt.Errorf("创建物料表失败: %w", err)
		}
	}
	return nil
}

// MaterialTables 返回全部物料表名
func MaterialTables() []string {
	out := make([]string, len(materialTables))
	copy(out, materialTables)
	return out
}
