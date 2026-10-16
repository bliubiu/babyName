package fate

import (
	"sort"
	"time"
)

// ——— bad-case 回流闭环：复审状态机与指标（docs/33 门禁层 M6） ———
//
// 闭环链路：命中(warn/fail)与收藏正样本 → gate_review(pending) →
// 人审落定(confirmed / false_positive / border) → gate_patch.jsonl →
// `materials import` 版本化合入 forbidden_combos / naming_gate / allowlist。
//
// 本文件只承载纯逻辑（状态机与指标口径），持久化在基础设施层（sqlite）。

// ReviewStatus 复审条目状态（docs/33 §九 状态机）
type ReviewStatus string

const (
	// ReviewPending 待复审：命中侧或收藏侧产生的候选
	ReviewPending ReviewStatus = "pending"
	// ReviewConfirmed 确认违规 → 增补 forbidden_combos / naming_gate
	ReviewConfirmed ReviewStatus = "confirmed"
	// ReviewFalsePositive 误报 → 写入 allowlist（白名单优先于黑名单）
	ReviewFalsePositive ReviewStatus = "false_positive"
	// ReviewBorder 边界 → 调整阈值/权重，不进词表
	ReviewBorder ReviewStatus = "border"
)

// ValidReviewStatus 判断是否为合法状态
func ValidReviewStatus(s ReviewStatus) bool {
	switch s {
	case ReviewPending, ReviewConfirmed, ReviewFalsePositive, ReviewBorder:
		return true
	}
	return false
}

// CanDecide 判断某状态能否被一次人审决策覆盖：仅待复审（pending）可决策
func CanDecide(current ReviewStatus) bool {
	return current == ReviewPending
}

// ReviewItem 复审条目：命中侧候选 + 人审结论
type ReviewItem struct {
	ID         string       `json:"id"`
	Name       string       `json:"name"`
	Layer      string       `json:"layer"`
	Category   string       `json:"category"`
	Level      string       `json:"level"`
	HitValue   string       `json:"hit_value"`
	Status     ReviewStatus `json:"status"`
	Note       string       `json:"note"`
	CreatedAt  time.Time    `json:"created_at"`
	ReviewedAt time.Time    `json:"reviewed_at,omitempty"`
}

// ReviewKey 幂等去重键：名字 + 层级 + 类别 + 命中值
func ReviewKey(name, layer, category, hitValue string) string {
	return name + "\x00" + layer + "\x00" + category + "\x00" + hitValue
}

// ReviewMetrics 复审闭环统计
type ReviewMetrics struct {
	Pending       int `json:"pending"`
	Confirmed     int `json:"confirmed"`
	FalsePositive int `json:"false_positive"`
	Border        int `json:"border"`
}

// Reviewed 已落定（已复审）总数
func (m ReviewMetrics) Reviewed() int {
	return m.Confirmed + m.FalsePositive + m.Border
}

// FalsePositiveRate 误报率 = 误报 / 已落定（无已落定取 0，保证 ≤1）
func (m ReviewMetrics) FalsePositiveRate() float64 {
	if d := m.Reviewed(); d > 0 {
		return float64(m.FalsePositive) / float64(d)
	}
	return 0
}

// SampleMissRate 抽样漏报率代理 = 边界项 / 已落定（边界视为疑似漏报，待调阈值）
func (m ReviewMetrics) SampleMissRate() float64 {
	if d := m.Reviewed(); d > 0 {
		return float64(m.Border) / float64(d)
	}
	return 0
}

// TallyReviews 按状态统计条目数
func TallyReviews(items []ReviewItem) ReviewMetrics {
	var m ReviewMetrics
	for _, it := range items {
		switch it.Status {
		case ReviewPending:
			m.Pending++
		case ReviewConfirmed:
			m.Confirmed++
		case ReviewFalsePositive:
			m.FalsePositive++
		case ReviewBorder:
			m.Border++
		}
	}
	return m
}

// ReviewLatencyMs 平均复审时长（毫秒）：仅统计已复审且 ReviewedAt 晚于 CreatedAt 的条目
func ReviewLatencyMs(items []ReviewItem) float64 {
	var total time.Duration
	var n int
	for _, it := range items {
		if it.ReviewedAt.IsZero() || !it.ReviewedAt.After(it.CreatedAt) {
			continue
		}
		total += it.ReviewedAt.Sub(it.CreatedAt)
		n++
	}
	if n == 0 {
		return 0
	}
	return float64(total.Milliseconds()) / float64(n)
}

// SortReviews 按创建时间升序（稳定），供列表与导出确定性输出
func SortReviews(items []ReviewItem) {
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
}
