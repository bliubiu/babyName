package sqlite

import (
	_ "modernc.org/sqlite"
	"name/internal/infrastructure/database"
	"name/internal/infrastructure/logger"
)

// SaveCuratedName 写入精选名（用户自学习）
//
// 用 UPSERT 而非 INSERT OR REPLACE：同名冲突时只更新用户可写字段，
// 保留库里已有的策展释义/五行/风格/标签（种子行富字段不被清空）。
func (s *Store) SaveCuratedName(name, pinyin, gender string, score float64, source string) error {
	_, err := s.writeDB.Exec(
		`INSERT INTO curated_names(name,pinyin,gender,score,source,created_at)
		 VALUES(?,?,?,?,?,CURRENT_TIMESTAMP)
		 ON CONFLICT(name) DO UPDATE SET pinyin=excluded.pinyin, gender=excluded.gender,
		   score=excluded.score, source=excluded.source, created_at=CURRENT_TIMESTAMP`,
		name, pinyin, gender, score, source,
	)
	return err
}

func (s *Store) LoadAllCuratedNames() ([]database.CuratedNameEntry, error) {
	rows, err := s.db.Query("SELECT name,pinyin,gender,score,source,meaning,wuxing,styles,tags FROM curated_names ORDER BY score DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []database.CuratedNameEntry
	for rows.Next() {
		var (
			e            database.CuratedNameEntry
			styles, tags string
		)
		if err := rows.Scan(&e.Name, &e.Pinyin, &e.Gender, &e.Score, &e.Source,
			&e.Meaning, &e.Wuxing, &styles, &tags); err != nil {
			continue
		}
		e.Styles = unmarshalTags(styles)
		e.Tags = unmarshalTags(tags)
		result = append(result, e)
	}
	if err := rows.Err(); err != nil {
		// 迭代中途的 IO/解码错误此前被当成「正常取完」，静默截断数据（docs/29 P3）
		logger.Error("sqlite: 遍历结果集中途失败", logger.ErrField(err))
	}

	return result, nil
}

func (s *Store) IsCurated(name string) (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM curated_names WHERE name=?", name).Scan(&count)
	return count > 0, err
}

func (s *Store) DeleteCuratedName(name string) error {
	_, err := s.writeDB.Exec("DELETE FROM curated_names WHERE name=?", name)
	return err
}

// --- Hanzi 字库缓存 ---
