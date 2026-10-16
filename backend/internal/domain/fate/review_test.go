package fate

import (
	"testing"
	"time"
)

// ——— 复审状态机与指标（docs/33 门禁层 M6）——

func TestValidReviewStatus(t *testing.T) {
	for _, s := range []ReviewStatus{ReviewPending, ReviewConfirmed, ReviewFalsePositive, ReviewBorder} {
		if !ValidReviewStatus(s) {
			t.Errorf("%s 应为合法状态", s)
		}
	}
	if ValidReviewStatus("nope") {
		t.Error("未知状态应非法")
	}
	if ValidReviewStatus("") {
		t.Error("空状态应非法")
	}
}

func TestCanDecide(t *testing.T) {
	if !CanDecide(ReviewPending) {
		t.Error("pending 应可决策")
	}
	for _, s := range []ReviewStatus{ReviewConfirmed, ReviewFalsePositive, ReviewBorder} {
		if CanDecide(s) {
			t.Errorf("%s 不应可再决策", s)
		}
	}
}

func TestReviewKey(t *testing.T) {
	a := ReviewKey("张沉默", "L3", "谐音连读", "沉默")
	b := ReviewKey("张沉默", "L3", "谐音连读", "沉默")
	c := ReviewKey("张沉默", "L3", "谐音连读", "沉没")
	if a != b {
		t.Error("同键应相等")
	}
	if a == c {
		t.Error("命中值不同应产生不同键")
	}
	if ReviewKey("x", "", "", "") == ReviewKey("", "x", "", "") {
		t.Error("分段键不应因字段位置不同而碰撞")
	}
}

func TestTallyReviews(t *testing.T) {
	items := []ReviewItem{
		{Status: ReviewPending},
		{Status: ReviewConfirmed},
		{Status: ReviewConfirmed},
		{Status: ReviewFalsePositive},
		{Status: ReviewBorder},
		{Status: "unknown"}, // 未知状态忽略
	}
	m := TallyReviews(items)
	if m.Pending != 1 || m.Confirmed != 2 || m.FalsePositive != 1 || m.Border != 1 {
		t.Fatalf("统计异常: %+v", m)
	}
	if m.Reviewed() != 4 {
		t.Fatalf("已落定应为 4，实得 %d", m.Reviewed())
	}
}

func TestReviewMetricsRates(t *testing.T) {
	var empty ReviewMetrics
	if empty.FalsePositiveRate() != 0 || empty.SampleMissRate() != 0 {
		t.Error("无已落定时比率应为 0")
	}
	m := ReviewMetrics{Confirmed: 2, FalsePositive: 1, Border: 1}
	if got := m.FalsePositiveRate(); got != 0.25 {
		t.Errorf("误报率期望 0.25，实得 %v", got)
	}
	if got := m.SampleMissRate(); got != 0.25 {
		t.Errorf("漏报率代理期望 0.25，实得 %v", got)
	}
}

func TestReviewLatencyMs(t *testing.T) {
	base := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	items := []ReviewItem{
		{CreatedAt: base, ReviewedAt: base.Add(2 * time.Second)},
		{CreatedAt: base, ReviewedAt: base.Add(4 * time.Second)},
		{CreatedAt: base},                                     // 未复审：忽略
		{CreatedAt: base, ReviewedAt: base},                   // 等于：忽略
		{CreatedAt: base, ReviewedAt: base.Add(-time.Second)}, // 早于：忽略
	}
	if got := ReviewLatencyMs(items); got != 3000 {
		t.Errorf("平均复审时长期望 3000ms，实得 %v", got)
	}
	if got := ReviewLatencyMs(nil); got != 0 {
		t.Errorf("空集合期望 0，实得 %v", got)
	}
}

func TestSortReviews(t *testing.T) {
	base := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	items := []ReviewItem{
		{ID: "c", CreatedAt: base.Add(2 * time.Hour)},
		{ID: "a", CreatedAt: base},
		{ID: "b", CreatedAt: base.Add(time.Hour)},
	}
	SortReviews(items)
	if items[0].ID != "a" || items[1].ID != "b" || items[2].ID != "c" {
		t.Fatalf("排序异常: %+v", items)
	}
}
