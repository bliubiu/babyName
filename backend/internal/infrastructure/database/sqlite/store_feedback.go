package sqlite

import (
	_ "modernc.org/sqlite"
	"name/internal/infrastructure/database"
	"name/internal/infrastructure/logger"
)

func (s *Store) SaveNameRequest(record *database.NameRequest) int64 {
	result, err := s.writeDB.Exec(
		`INSERT INTO name_requests (surname, gender, birth_year, birth_month, birth_day, birth_hour, preferences, name_type, request_hash, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.Surname, record.Gender, record.BirthYear, record.BirthMonth, record.BirthDay, record.BirthHour,
		record.Preferences, record.NameType, record.RequestHash, record.CreatedAt)
	if err != nil {
		logger.Error("sqlite: 保存名字请求失败", logger.ErrField(err))
		return 0
	}
	id, _ := result.LastInsertId()
	return id
}

func (s *Store) GetNameRequestByHash(hash string) *database.NameRequest {
	var r database.NameRequest
	err := s.db.QueryRow(
		`SELECT id, surname, gender, birth_year, birth_month, birth_day, birth_hour, preferences, name_type, request_hash, created_at
		 FROM name_requests WHERE request_hash = ?`, hash).
		Scan(&r.ID, &r.Surname, &r.Gender, &r.BirthYear, &r.BirthMonth, &r.BirthDay, &r.BirthHour,
			&r.Preferences, &r.NameType, &r.RequestHash, &r.CreatedAt)
	if err != nil {
		return nil
	}
	return &r
}

func (s *Store) SaveNameFeedback(record *database.NameFeedback) int64 {
	result, err := s.writeDB.Exec(
		`INSERT INTO name_feedback (request_id, full_name, given_name, is_liked, is_selected, user_rating, feedback_text, algorithm_score, wuxing_match, device_info, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.RequestID, record.FullName, record.GivenName, record.IsLiked, record.IsSelected,
		record.UserRating, record.FeedbackText, record.AlgorithmScore, record.WuxingMatch, record.DeviceInfo, record.CreatedAt)
	if err != nil {
		logger.Error("sqlite: 保存名字反馈失败", logger.ErrField(err))
		return 0
	}
	id, _ := result.LastInsertId()
	return id
}

func (s *Store) GetNameFeedbackByRequestID(requestID int64) []*database.NameFeedback {
	rows, err := s.db.Query(
		`SELECT id, request_id, full_name, given_name, is_liked, is_selected, user_rating, feedback_text, algorithm_score, wuxing_match, device_info, created_at
		 FROM name_feedback WHERE request_id = ?`, requestID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var result []*database.NameFeedback
	for rows.Next() {
		var f database.NameFeedback
		if err := rows.Scan(&f.ID, &f.RequestID, &f.FullName, &f.GivenName, &f.IsLiked, &f.IsSelected,
			&f.UserRating, &f.FeedbackText, &f.AlgorithmScore, &f.WuxingMatch, &f.DeviceInfo, &f.CreatedAt); err != nil {
			continue
		}
		result = append(result, &f)
	}
	if err := rows.Err(); err != nil {
		// 迭代中途的 IO/解码错误此前被当成「正常取完」，静默截断数据（docs/29 P3）
		logger.Error("sqlite: 遍历结果集中途失败", logger.ErrField(err))
	}

	return result
}

func (s *Store) GetAlgorithmPerformance() ([]*database.AlgorithmPerformance, error) {
	rows, err := s.db.Query(
		`SELECT DATE(nr.created_at) as date,
		        COUNT(*) as total_requests,
		        COALESCE(AVG(nf.algorithm_score), 0) as avg_score,
		        COALESCE(SUM(CASE WHEN nf.is_liked THEN 1 ELSE 0 END) * 1.0 / NULLIF(COUNT(nf.id), 0), 0) as like_rate,
		        COALESCE(SUM(CASE WHEN nf.is_selected THEN 1 ELSE 0 END) * 1.0 / NULLIF(COUNT(nf.id), 0), 0) as select_rate
		 FROM name_requests nr
		 LEFT JOIN name_feedback nf ON nf.request_id = nr.id
		 GROUP BY DATE(nr.created_at)
		 ORDER BY date DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*database.AlgorithmPerformance
	for rows.Next() {
		var p database.AlgorithmPerformance
		if err := rows.Scan(&p.Date, &p.TotalRequests, &p.AvgScore, &p.LikeRate, &p.SelectRate); err != nil {
			continue
		}
		result = append(result, &p)
	}
	if err := rows.Err(); err != nil {
		// 迭代中途的 IO/解码错误此前被当成「正常取完」，静默截断数据（docs/29 P3）
		logger.Error("sqlite: 遍历结果集中途失败", logger.ErrField(err))
	}

	return result, nil
}

// --- CuratedStore ---
