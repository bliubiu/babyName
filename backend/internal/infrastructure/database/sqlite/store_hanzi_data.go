package sqlite

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
	"name/internal/infrastructure/database"
	"name/internal/infrastructure/logger"
)

// hanziDataInsertSQL hanzi_data 的插入语句
//
// 写入列顺序与 wordValues()（store_materials_import.go）的返回值顺序严格对应，
// 改动必须同步（hanzi_data 表列、hanziDataColumnMigrations、wordValues 三处一致）。
//
// hanzi_data 的唯一写入方是物料导入路径 importNamer（docs/34 阶段 A1）：
// 内存态 hanzi.HanziData → SQLite 的 legacy 投影路径（seedHanziData/rebuildHanziData/
// ReloadHanziData）因无生产调用方且字段派生口径不同，已整体移除，避免双口径污染。
const hanziDataInsertSQL = `INSERT INTO hanzi_data(
	char, pinyin, strokes, kangxi_strokes, radical, meaning, wuxing, gender,
	level, tone, name_freq_tier, gender_tags, style_tags, usage_level,
	positive_score, phonetic_score, modern_score, classical_score,
	is_polyphonic, is_rare, is_negative,
	pair_blacklist, curation_level, name_penalty, naming_categories
) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`

// boolToInt 把布尔转成 SQLite 存储用的 0/1
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// HanziRowCount 返回 hanzi_data 行数
//
// 供 services 层 SQLHanziFilter.HasData 甄别「库无数据（未导入物料，
// 应降级 Go 端过滤）」与「有数据但查询无匹配（如实返回空）」——
// 空库回落场景下两者在 SQL 查询返回值上完全一致，是 docs/29 B8
// 修复（失败与无匹配可区分）之后新暴露的孪生陷阱。
func (s *Store) HanziRowCount() (int, error) {
	var n int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM hanzi_data").Scan(&n); err != nil {
		return 0, fmt.Errorf("查询 hanzi_data 行数失败: %w", err)
	}
	return n, nil
}

// QueryHanziByWuxing 从 SQLite 按五行查询汉字
func (s *Store) QueryHanziByWuxing(wuxing string) ([]database.Hanzi, error) {
	rows, err := s.db.Query(
		"SELECT char, pinyin, wuxing, strokes FROM hanzi_data WHERE wuxing = ? ORDER BY strokes",
		wuxing,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []database.Hanzi
	for rows.Next() {
		var h database.Hanzi
		if err := rows.Scan(&h.Char, &h.Pinyin, &h.Wuxing, &h.Strokes); err != nil {
			continue
		}
		result = append(result, h)
	}
	if err := rows.Err(); err != nil {
		// 迭代中途的 IO/解码错误此前被当成「正常取完」，静默截断数据（docs/29 P3）
		logger.Error("sqlite: 遍历结果集中途失败", logger.ErrField(err))
	}

	return result, nil
}

// QueryHanziByStrokes 从 SQLite 按笔画范围查询汉字
func (s *Store) QueryHanziByStrokes(min, max int) ([]database.Hanzi, error) {
	rows, err := s.db.Query(
		"SELECT char, pinyin, wuxing, strokes FROM hanzi_data WHERE strokes >= ? AND strokes <= ? ORDER BY strokes",
		min, max,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []database.Hanzi
	for rows.Next() {
		var h database.Hanzi
		if err := rows.Scan(&h.Char, &h.Pinyin, &h.Wuxing, &h.Strokes); err != nil {
			continue
		}
		result = append(result, h)
	}
	if err := rows.Err(); err != nil {
		// 迭代中途的 IO/解码错误此前被当成「正常取完」，静默截断数据（docs/29 P3）
		logger.Error("sqlite: 遍历结果集中途失败", logger.ErrField(err))
	}

	return result, nil
}

// SearchHanziByFilter 按基础过滤参数查询 hanzi_data（services.SQLiteCharStore 接口实现）
//
// services 包不能 import sqlite 包（分层约束），故用此最简基础参数签名
// 让 *Store 同时实现 services.SQLiteCharStore 接口。组装时通过
// services.NewSQLiteHanziFilter(store) 桥接：
//
//	if s, ok := store.(services.SQLiteCharStore); ok {
//	    provider.SetSQLFilter(services.NewSQLiteHanziFilter(s))
//	}
//
// 实现内联 SQL 查询（避免依赖 HanziFilter 类型）：动态构建 WHERE 子句，
// 命中 idx_hanzi_data_wuxing / idx_hanzi_data_strokes 索引。
//
// P2（ADR-003「SQL 与内存结果等价」）：本方法原先只返回 6 列
// （char/pinyin/wuxing/strokes/usage_level/positive_score）。services 层拿不到
// gender / naming_categories / radical / level / kangxi_strokes / meaning / is_rare，
// 只能在拿到结果后逐字 `hanzi.GetHanzi` 回查内存做二次过滤 —— SQL 沦为「粗筛」，
// 且 SQL 路径与内存路径的过滤结果无法保证一致（docs/31 §11.9 第 2 项）。
// 现已将全部过滤与构造所需列下沉到 hanzi_data 并一并返回，services 层不再回查。
func (s *Store) SearchHanziByFilter(
	wuxing string, minStrokes, maxStrokes int,
	hasPositive, isRegular bool, chars []string, limit int,
) ([]*database.Hanzi, error) {
	var conditions []string
	var args []any

	if wuxing != "" {
		conditions = append(conditions, "wuxing = ?")
		args = append(args, wuxing)
	}
	if minStrokes > 0 {
		conditions = append(conditions, "strokes >= ?")
		args = append(args, minStrokes)
	}
	if maxStrokes > 0 {
		conditions = append(conditions, "strokes <= ?")
		args = append(args, maxStrokes)
	}
	if hasPositive {
		conditions = append(conditions, "positive_score > 0")
	}
	if isRegular {
		conditions = append(conditions, "usage_level IN (1, 2)")
	}
	if len(chars) > 0 {
		placeholders := make([]string, len(chars))
		for i, c := range chars {
			placeholders[i] = "?"
			args = append(args, c)
		}
		conditions = append(conditions, "char IN ("+strings.Join(placeholders, ",")+")")
	}

	if limit <= 0 {
		limit = 10000
	}

	// P2：返回构造 Character 与过滤所需的全部列，避免调用方回查内存补字段。
	// 列顺序与下方 Scan 顺序严格一一对应，改任一侧都必须同步。
	const selectColumns = `char, pinyin, wuxing, strokes, usage_level, positive_score,
		COALESCE(level, 0), COALESCE(kangxi_strokes, 0), COALESCE(gender, ''),
		COALESCE(radical, ''), COALESCE(naming_categories, '[]'), COALESCE(meaning, ''),
		COALESCE(is_rare, 0), COALESCE(is_negative, 0), COALESCE(name_penalty, 0),
		COALESCE(name_freq_tier, 0)`

	query := "SELECT " + selectColumns + " FROM hanzi_data"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY char LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		// 必须上抛而不是返回 nil：nil 会被上层判成「无匹配」，
		// 使注释承诺的「SQL 失败降级 Go 端全表过滤」路径永不可达（docs/29 B8）
		logger.Error("SearchHanziByFilter 查询失败", logger.ErrField(err))
		return nil, fmt.Errorf("SearchHanziByFilter 查询失败: %w", err)
	}
	defer rows.Close()

	var result []*database.Hanzi
	for rows.Next() {
		var h database.Hanzi
		var namingCategories string
		var isRare, isNegative int
		if err := rows.Scan(&h.Char, &h.Pinyin, &h.Wuxing, &h.Strokes,
			&h.UsageLevel, &h.PositiveScore,
			&h.Level, &h.KangxiStrokes, &h.Gender, &h.Radical,
			&namingCategories, &h.Meaning, &isRare,
			&isNegative, &h.NamePenalty, &h.NameFreqTier); err != nil {
			logger.Error("SearchHanziByFilter scan 失败", logger.ErrField(err))
			continue
		}
		h.IsRare = isRare != 0
		h.IsNegative = isNegative != 0
		if namingCategories != "" && namingCategories != "[]" {
			// naming_categories 落库形态为 JSON 数组文本；解析失败按「无分类」处理，
			// 不丢弃整行 —— 否则单条脏数据会连带丢掉该字的全部候选属性。
			var categories []string
			if err := json.Unmarshal([]byte(namingCategories), &categories); err != nil {
				logger.Error("SearchHanziByFilter naming_categories 解析失败",
					logger.ErrField(err), logger.String("char", h.Char))
			} else {
				h.NamingCategories = categories
			}
		}
		result = append(result, &h)
	}
	if err := rows.Err(); err != nil {
		// 迭代中途的 IO/解码错误此前被当成「正常取完」，静默截断数据（docs/29 P3）
		logger.Error("sqlite: 遍历结果集中途失败", logger.ErrField(err))
	}

	// rows.Err 覆盖迭代中途的 IO/解码错误：不查就等于把截断结果当正常取完
	if err := rows.Err(); err != nil {
		logger.Error("SearchHanziByFilter 迭代失败", logger.ErrField(err))
		return nil, fmt.Errorf("SearchHanziByFilter 迭代失败: %w", err)
	}
	return result, nil
}

// --- NameStatStore 占位实现 ---
//
// 背景：数据库.Store 接口组合了 NameStatStore（8 方法）。姓名统计数据
// （姓氏/名/全名排行、姓名性别统计）由 data 目录 JSON 文件承载
// （surname_stats.json 等，见 cmd/build_namestats），SQLite 无对应表。
// 占位方法返回明确的"不支持"错误（而非静默空数据），让 /namestats/* 接口
// 在接线真正实现前诚实报错，便于调用方降级。
var errNameStatsUnavailable = errors.New("姓名统计数据未接入 SQLite 持久化，当前数据源为 data 目录 JSON 文件")
