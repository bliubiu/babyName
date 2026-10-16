package sqlite

import (
	"testing"
	"time"

	"name/internal/domain/fate"
)

// ——— 门禁复审队列（docs/33 门禁层 M6）——
// 校验 pending 候选登记（幂等/去重）、人审状态机、统计指标与收藏回流。

// addPending 登记一条待复审候选并返回其 ID
func addPending(t *testing.T, s *Store, name, layer, category, level, hitValue string) string {
	t.Helper()
	items, err := s.ListGateReviews(fate.ReviewPending, 0)
	if err != nil {
		t.Fatalf("列举待复审失败: %v", err)
	}
	before := len(items)
	inserted, err := s.AddGateReview(fate.ReviewItem{
		Name: name, Layer: layer, Category: category, Level: level, HitValue: hitValue,
	})
	if err != nil {
		t.Fatalf("登记复审候选失败: %v", err)
	}
	if !inserted {
		t.Fatalf("首次登记应返回 inserted=true（name=%s hit=%s）", name, hitValue)
	}
	items, err = s.ListGateReviews(fate.ReviewPending, 0)
	if err != nil {
		t.Fatalf("列举待复审失败: %v", err)
	}
	if len(items) != before+1 {
		t.Fatalf("待复审条目数应 +1，实得 %d", len(items))
	}
	return items[len(items)-1].ID
}

func TestGateReview_Add幂等与默认值(t *testing.T) {
	s := newTestStore(t)

	inserted, err := s.AddGateReview(fate.ReviewItem{
		Name: "张沉默", Layer: "L3", Category: "谐音连读", Level: "fail", HitValue: "沉默",
	})
	if err != nil {
		t.Fatalf("登记失败: %v", err)
	}
	if !inserted {
		t.Fatal("首次登记应 inserted=true")
	}
	// 同键二次登记：去重，不覆盖
	inserted, err = s.AddGateReview(fate.ReviewItem{
		Name: "张沉默", Layer: "L3", Category: "谐音连读", Level: "fail", HitValue: "沉默",
		Status: fate.ReviewConfirmed,
	})
	if err != nil {
		t.Fatalf("二次登记失败: %v", err)
	}
	if inserted {
		t.Fatal("同键重复登记应 inserted=false")
	}

	items, err := s.ListGateReviews("", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("去重后应仅 1 条，实得 %d", len(items))
	}
	it := items[0]
	if it.ID == "" {
		t.Error("应自动生成 ID")
	}
	if it.CreatedAt.IsZero() {
		t.Error("应自动填充创建时间")
	}
	if it.Status != fate.ReviewPending {
		t.Errorf("默认状态应为 pending，实得 %s", it.Status)
	}
	if !it.ReviewedAt.IsZero() {
		t.Error("未复审时 reviewed_at 应为零值")
	}
}

func TestGateReview_Add非法状态报错(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.AddGateReview(fate.ReviewItem{Name: "李四", Status: "bogus"}); err == nil {
		t.Fatal("非法状态应报错")
	}
}

func TestGateReview_List过滤与限数(t *testing.T) {
	s := newTestStore(t)
	idA := addPending(t, s, "张一", "L1", "门禁字", "fail", "一")
	addPending(t, s, "张二", "L1", "门禁字", "fail", "二")
	if err := s.DecideGateReview(idA, fate.ReviewConfirmed, "确认"); err != nil {
		t.Fatal(err)
	}

	pendings, err := s.ListGateReviews(fate.ReviewPending, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(pendings) != 1 || pendings[0].Name != "张二" {
		t.Fatalf("pending 过滤异常: %+v", pendings)
	}
	confirms, err := s.ListGateReviews(fate.ReviewConfirmed, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(confirms) != 1 || confirms[0].ID != idA {
		t.Fatalf("confirmed 过滤异常: %+v", confirms)
	}

	limited, err := s.ListGateReviews("", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(limited) != 1 {
		t.Fatalf("limit=1 应返回 1 条，实得 %d", len(limited))
	}
}

func TestGateReview_Decide状态机(t *testing.T) {
	s := newTestStore(t)
	id := addPending(t, s, "王涵", "favorite", "收藏正样本", "info", "王涵")

	if err := s.DecideGateReview(id, fate.ReviewPending, ""); err == nil {
		t.Fatal("落定为 pending 应报错")
	}
	if err := s.DecideGateReview(id, "bogus", ""); err == nil {
		t.Fatal("非法结论应报错")
	}
	if err := s.DecideGateReview("no-such-id", fate.ReviewBorder, ""); err == nil {
		t.Fatal("不存在的条目应报错")
	}

	if err := s.DecideGateReview(id, fate.ReviewFalsePositive, "误报"); err != nil {
		t.Fatalf("落定失败: %v", err)
	}
	// 已落定不可再次决策
	if err := s.DecideGateReview(id, fate.ReviewConfirmed, "改口"); err == nil {
		t.Fatal("已落定条目不应可重复决策")
	}

	items, err := s.ListGateReviews(fate.ReviewFalsePositive, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Note != "误报" || items[0].ReviewedAt.IsZero() {
		t.Fatalf("落定结果异常: %+v", items)
	}
}

func TestGateReview_统计与复审时长(t *testing.T) {
	s := newTestStore(t)

	// 两条已落定（created 2s 前，便于计算时长）
	base := time.Now().Add(-2 * time.Second)
	for _, target := range []fate.ReviewStatus{fate.ReviewConfirmed, fate.ReviewFalsePositive} {
		inserted, err := s.AddGateReview(fate.ReviewItem{
			Name: "李" + string(target), Layer: "L1", Category: "门禁字", HitValue: string(target),
			CreatedAt: base,
		})
		if err != nil || !inserted {
			t.Fatalf("登记失败: %v", err)
		}
	}
	items, err := s.ListGateReviews(fate.ReviewPending, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range items {
		if err := s.DecideGateReview(it.ID, fate.ReviewStatus(it.HitValue), ""); err != nil {
			t.Fatalf("落定失败: %v", err)
		}
	}
	// 一条仍待复审
	addPending(t, s, "张三", "L1", "门禁字", "fail", "三")

	m, err := s.GateReviewStats()
	if err != nil {
		t.Fatal(err)
	}
	if m.Pending != 1 || m.Confirmed != 1 || m.FalsePositive != 1 || m.Border != 0 {
		t.Fatalf("统计异常: %+v", m)
	}

	lat := s.GateReviewLatencyMs()
	if lat < 1000 {
		t.Fatalf("平均复审时长应 ≥1000ms，实得 %v", lat)
	}
}

func TestGateReview_收藏回流候选(t *testing.T) {
	s := newTestStore(t)

	s.RecordFavoriteCandidate("张", "珀熙")
	s.RecordFavoriteCandidate("张", "珀熙") // 幂等
	s.RecordFavoriteCandidate("", "")    // 空名忽略

	items, err := s.ListGateReviews("", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("收藏候选应去重为 1 条，实得 %d", len(items))
	}
	it := items[0]
	if it.Name != "张珀熙" || it.Layer != "favorite" || it.Category != "收藏正样本" {
		t.Fatalf("收藏候选字段异常: %+v", it)
	}
	if it.Status != fate.ReviewPending {
		t.Errorf("收藏候选应为 pending，实得 %s", it.Status)
	}
}

func TestGateReview_MetricProvider(t *testing.T) {
	s := newTestStore(t)

	// 空库：三项指标均为 0
	if s.FalsePositiveRate() != 0 || s.SampleMissRate() != 0 || s.ReviewLatencyMs() != 0 {
		t.Fatal("空库指标应为 0")
	}

	// 落定 2 confirmed + 1 false_positive + 1 border
	targets := []fate.ReviewStatus{
		fate.ReviewConfirmed, fate.ReviewConfirmed,
		fate.ReviewFalsePositive, fate.ReviewBorder,
	}
	for i, st := range targets {
		inserted, err := s.AddGateReview(fate.ReviewItem{
			Name: "候选人", Layer: "L1", Category: "门禁字",
			HitValue: string(rune('a' + i)), CreatedAt: time.Now(),
		})
		if err != nil || !inserted {
			t.Fatalf("登记失败: %v", err)
		}
		_ = st
	}
	items, err := s.ListGateReviews(fate.ReviewPending, 0)
	if err != nil {
		t.Fatal(err)
	}
	for i, it := range items {
		if err := s.DecideGateReview(it.ID, targets[i], ""); err != nil {
			t.Fatalf("落定失败: %v", err)
		}
	}

	if got := s.FalsePositiveRate(); got != 0.25 {
		t.Errorf("误报率期望 0.25，实得 %v", got)
	}
	if got := s.SampleMissRate(); got != 0.25 {
		t.Errorf("漏报率代理期望 0.25，实得 %v", got)
	}
}

func TestParseGateReviewTime(t *testing.T) {
	if !parseGateReviewTime("").IsZero() {
		t.Error("空串应返回零值")
	}
	if got := parseGateReviewTime("2026-10-16 12:00:00"); got.IsZero() || got.Year() != 2026 {
		t.Errorf("应解析空格布局: %v", got)
	}
	if got := parseGateReviewTime("2026-10-16T12:00:00Z"); got.IsZero() {
		t.Errorf("应解析 RFC3339: %v", got)
	}
	if !parseGateReviewTime("not-a-time").IsZero() {
		t.Error("非法时间应返回零值")
	}
}
