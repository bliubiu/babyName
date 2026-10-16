package sqlite

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"name/internal/domain/fate"
	"name/internal/infrastructure/logger"
)

// ——— 门禁复审队列（docs/33 门禁层 M6） ———
//
// gate_review 承载 bad-case 回流闭环的人审状态机：命中/收藏产生 pending 候选，
// 人审落定为 confirmed / false_positive / border。表结构在 store.go 定义。

// gateReviewTimeLayout 复审时间的文本存储布局（与 CURRENT_TIMESTAMP 同口径）
const gateReviewTimeLayout = "2006-01-02 15:04:05"

// AddGateReview 幂等登记一条待复审候选（按 名字+层级+类别+命中值 去重）。
//
// 返回 true 表示新插入；false 表示已存在（不覆盖既有结论，保护已落定条目）。
func (s *Store) AddGateReview(item fate.ReviewItem) (bool, error) {
	if item.Status == "" {
		item.Status = fate.ReviewPending
	}
	if !fate.ValidReviewStatus(item.Status) {
		return false, fmt.Errorf("非法复审状态: %s", item.Status)
	}
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now()
	}

	res, err := s.writeDB.Exec(
		`INSERT OR IGNORE INTO gate_review
		 (id, name, layer, category, level, hit_value, status, note, created_at)
		 VALUES (?,?,?,?,?,?,?,?,?)`,
		item.ID, item.Name, item.Layer, item.Category, item.Level,
		item.HitValue, string(item.Status), item.Note, item.CreatedAt.Format(gateReviewTimeLayout),
	)
	if err != nil {
		return false, fmt.Errorf("写入复审候选失败: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("读取写入结果失败: %w", err)
	}
	return n > 0, nil
}

// ListGateReviews 按状态查询复审条目（status 为空表示全部），按创建时间升序。
// limit > 0 时限制条数。
func (s *Store) ListGateReviews(status fate.ReviewStatus, limit int) ([]fate.ReviewItem, error) {
	q := `SELECT id, name, layer, category, level, hit_value, status, note, created_at, reviewed_at
	      FROM gate_review`
	args := make([]any, 0, 2)
	if status != "" {
		q += " WHERE status = ?"
		args = append(args, string(status))
	}
	q += " ORDER BY created_at ASC"
	if limit > 0 {
		q += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("查询复审候选失败: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var items []fate.ReviewItem
	for rows.Next() {
		var (
			it         fate.ReviewItem
			statusText string
			createdAt  string
			reviewedAt sql.NullString
		)
		if err := rows.Scan(&it.ID, &it.Name, &it.Layer, &it.Category, &it.Level,
			&it.HitValue, &statusText, &it.Note, &createdAt, &reviewedAt); err != nil {
			return nil, fmt.Errorf("扫描复审候选失败: %w", err)
		}
		it.Status = fate.ReviewStatus(statusText)
		it.CreatedAt = parseGateReviewTime(createdAt)
		if reviewedAt.Valid {
			it.ReviewedAt = parseGateReviewTime(reviewedAt.String)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历复审候选失败: %w", err)
	}
	return items, nil
}

// DecideGateReview 落定一条复审结论：仅待复审（pending）条目可被决策一次。
func (s *Store) DecideGateReview(id string, status fate.ReviewStatus, note string) error {
	if !fate.ValidReviewStatus(status) || status == fate.ReviewPending {
		return fmt.Errorf("非法复审结论: %s", status)
	}
	res, err := s.writeDB.Exec(
		`UPDATE gate_review SET status = ?, note = ?, reviewed_at = ?
		 WHERE id = ? AND status = ?`,
		string(status), note, time.Now().Format(gateReviewTimeLayout),
		id, string(fate.ReviewPending),
	)
	if err != nil {
		return fmt.Errorf("更新复审结论失败: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("读取更新结果失败: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("复审条目不存在或已落定: %s", id)
	}
	return nil
}

// GateReviewStats 统计各状态条目数
func (s *Store) GateReviewStats() (fate.ReviewMetrics, error) {
	rows, err := s.db.Query(`SELECT status, COUNT(*) FROM gate_review GROUP BY status`)
	if err != nil {
		return fate.ReviewMetrics{}, fmt.Errorf("统计复审状态失败: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var m fate.ReviewMetrics
	for rows.Next() {
		var (
			statusText string
			n          int
		)
		if err := rows.Scan(&statusText, &n); err != nil {
			return fate.ReviewMetrics{}, fmt.Errorf("扫描复审统计失败: %w", err)
		}
		switch fate.ReviewStatus(statusText) {
		case fate.ReviewPending:
			m.Pending += n
		case fate.ReviewConfirmed:
			m.Confirmed += n
		case fate.ReviewFalsePositive:
			m.FalsePositive += n
		case fate.ReviewBorder:
			m.Border += n
		}
	}
	if err := rows.Err(); err != nil {
		return fate.ReviewMetrics{}, fmt.Errorf("遍历复审统计失败: %w", err)
	}
	return m, nil
}

// GateReviewLatencyMs 平均复审时长（毫秒）
func (s *Store) GateReviewLatencyMs() float64 {
	items, err := s.ListGateReviews("", 0)
	if err != nil {
		return 0
	}
	return fate.ReviewLatencyMs(items)
}

// RecordFavoriteCandidate 收藏正样本回流：把被收藏的名字登记为待复审候选
// （docs/33 §九「收藏是隐式正反馈」）。失败仅告警，不影响收藏主流程。
func (s *Store) RecordFavoriteCandidate(surname, givenName string) {
	full := surname + givenName
	if full == "" {
		return
	}
	if _, err := s.AddGateReview(fate.ReviewItem{
		Name:     full,
		Layer:    "favorite",
		Category: "收藏正样本",
		Level:    "info",
		HitValue: full,
		Status:   fate.ReviewPending,
		Note:     "用户收藏触发",
	}); err != nil {
		logger.Warn("门禁复审：登记收藏候选失败", logger.ErrField(err))
	}
}

// ——— fate.GateMetricProvider 实现（供 M4 指标闭环注入真实数据源） ———

var _ fate.GateMetricProvider = (*Store)(nil)

// FalsePositiveRate 误报率（查库失败降级为 0）
func (s *Store) FalsePositiveRate() float64 {
	m, err := s.GateReviewStats()
	if err != nil {
		return 0
	}
	return m.FalsePositiveRate()
}

// SampleMissRate 抽样漏报率（查库失败降级为 0）
func (s *Store) SampleMissRate() float64 {
	m, err := s.GateReviewStats()
	if err != nil {
		return 0
	}
	return m.SampleMissRate()
}

// ReviewLatencyMs 平均复审时长（毫秒）
func (s *Store) ReviewLatencyMs() float64 {
	return s.GateReviewLatencyMs()
}

// parseGateReviewTime 解析复审时间文本，兼容空格分隔与 RFC3339 两种布局
func parseGateReviewTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	if t, err := time.Parse(gateReviewTimeLayout, s); err == nil {
		return t
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	return time.Time{}
}
