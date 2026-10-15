package sqlite

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"name/internal/domain/hanzi"
	"name/internal/infrastructure/logger"
)

// 本文件实现 docs/31 ADR-013：康熙笔画落 `kangxi_strokes` 表，消除
// 「运行时只认 SQLite、不读 JSON/CSV」下的数据空洞。
//
// 背景：ADR-008 要求运行时零 JSON/零 CSV，但 `kangxi-strokecount.csv` 从不落库
// 且无对应表 —— 康熙笔画（河图数理、易经卦象口径）在运行时无来源。
// 本阶段提供「落表 + DB 载入 + 缺失降级」三条路径；
// 默认运行时仍走 CSV（保持行为不变），DB 载入路径在 ADR-008 阶段切换为默认。

// seedKangxiStrokes 建表并从 CSV 导入康熙笔画
//
// CSV 缺失时记 warn 并返回（与现有降级语义一致，不阻断启动）
func (s *Store) seedKangxiStrokes(dataDir string) {
	if _, err := s.writeDB.Exec(`CREATE TABLE IF NOT EXISTS kangxi_strokes (
		char    TEXT PRIMARY KEY,
		strokes INTEGER NOT NULL DEFAULT 0
	)`); err != nil {
		logger.Error("sqlite: 康熙笔画表创建失败", logger.ErrField(err))
		return
	}

	path := resolveKangxiCSV(dataDir)
	if path == "" {
		logger.Warn("sqlite: 康熙笔画 CSV 不存在，跳过入库（运行时按简体笔画兜底）")
		return
	}

	m, err := hanzi.ParseKangxiCSVFile(path)
	if err != nil {
		logger.Error("sqlite: 康熙笔画 CSV 解析失败，跳过入库", logger.ErrField(err))
		return
	}

	tx, err := s.writeDB.Begin()
	if err != nil {
		logger.Error("sqlite: 康熙笔画入库事务开启失败", logger.ErrField(err))
		return
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec("DELETE FROM kangxi_strokes"); err != nil {
		logger.Error("sqlite: 康熙笔画表清空失败", logger.ErrField(err))
		return
	}

	stmt, err := tx.Prepare("INSERT INTO kangxi_strokes(char, strokes) VALUES(?,?)")
	if err != nil {
		logger.Error("sqlite: 康熙笔画插入语句准备失败", logger.ErrField(err))
		return
	}
	defer stmt.Close()

	written := 0
	for char, strokes := range m {
		if _, err := stmt.Exec(char, strokes); err != nil {
			logger.Warn("sqlite: 单个康熙笔画写入失败已跳过", logger.String("字", char), logger.ErrField(err))
			continue
		}
		written++
	}
	if err := tx.Commit(); err != nil {
		logger.Error("sqlite: 康熙笔画入库提交失败", logger.ErrField(err))
		return
	}
	logger.Info("sqlite: 康熙笔画已入库", logger.Int("条数", written))
}

// LoadKangxiStrokesFromDB 从 kangxi_strokes 表载入康熙笔画到内存
//
// 表不存在（旧库，ADR-013 新增）时降级：置空集合并标记已加载，
// 由调用方按「未收录」回落简体笔画，不阻断启动（docs/31 §8.5 例外条款）
func (s *Store) LoadKangxiStrokesFromDB() error {
	rows, err := s.db.Query("SELECT char, strokes FROM kangxi_strokes")
	if err != nil {
		if isMissingTableErr(err) {
			logger.Warn("sqlite: 康熙笔画表不存在（库版本较旧），按简体笔画兜底")
			hanzi.SetKangxiStrokes(map[string]int{})
			return nil
		}
		return fmt.Errorf("查询康熙笔画表失败: %w", err)
	}
	defer func() { _ = rows.Close() }()

	m := make(map[string]int, 8192)
	for rows.Next() {
		var char string
		var strokes int
		if err := rows.Scan(&char, &strokes); err != nil {
			logger.Error("sqlite: 康熙笔画行扫描失败", logger.ErrField(err))
			continue
		}
		if char == "" || strokes <= 0 {
			continue
		}
		m[char] = strokes
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("遍历康熙笔画表失败: %w", err)
	}

	hanzi.SetKangxiStrokes(m)
	logger.Info("sqlite: 康熙笔画已从库载入", logger.Int("条数", len(m)))
	return nil
}

// resolveKangxiCSV 在 raw 目录与 dataDir 根中查找康熙笔画 CSV
func resolveKangxiCSV(dataDir string) string {
	candidates := []string{
		filepath.Join(dataDir, "raw", "kangxi-strokecount.csv"),
		filepath.Join(dataDir, "kangxi-strokecount.csv"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// isMissingTableErr 判断错误是否为「表不存在」
//
// 不引��具体驱动错误类型，避免为一句判断耦合驱动实现；
// SQLite 的缺表错误消息稳定包含 "no such table"
func isMissingTableErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), "no such table")
}
