package fate

import (
	"strings"
	"sync"
	"sync/atomic"
)

// ——— 门禁指标闭环（docs/33 门禁层 M4） ———
//
// 指标分两类：
//   - 本地可算：评估次数、命中评估数、命中率、命中分布（直接由评估计数与
//     命中审计埋点派生）；
//   - 依赖复盘闭环：误报率、抽样漏报率、复审时长——其数据源在 M6（gate_review
//     状态机与人审回流）才具备，这里通过 GateMetricProvider 预留注入点，
//     未注册时以 0 暴露，保证端点字段稳定、可先行接入监控。

// GateMetrics 门禁指标快照（JSON 可序列化，供 /metrics 端点暴露）
type GateMetrics struct {
	Evaluations       int64            `json:"gate_evaluations_total"`     // 门禁评估次数（分母）
	HitEvaluations    int64            `json:"gate_hit_evaluations_total"` // 至少一次命中的评估次数（分子）
	HitRate           float64          `json:"gate_hit_rate"`              // 命中率 = HitEvaluations / Evaluations
	Hits              int64            `json:"gate_hits_total"`            // 命中事件总数
	HitsByLayer       map[string]int64 `json:"gate_hits_by_layer"`         // 分层命中分布
	HitsByLevel       map[string]int64 `json:"gate_hits_by_level"`         // 等级分布 warn / fail
	HitsByCategory    map[string]int64 `json:"gate_hits_by_category"`      // 类别分布
	FalsePositiveRate float64          `json:"gate_false_positive_rate"`   // 误报率（数据源 M6）
	SampleMissRate    float64          `json:"gate_sample_miss_rate"`      // 抽样漏报率（数据源 M6）
	ReviewLatencyMs   float64          `json:"gate_review_latency_ms"`     // 平均复审时长（毫秒，数据源 M6）
}

// GateMetricProvider 复审闭环指标的取值来源；由 M6 的复审/回流模块实现并注册
type GateMetricProvider interface {
	FalsePositiveRate() float64
	SampleMissRate() float64
	ReviewLatencyMs() float64
}

var (
	gateEvaluations    atomic.Int64
	gateHitEvaluations atomic.Int64
	gateMetricMu       sync.Mutex
	gateMetricProvider GateMetricProvider
)

// RecordGateEvaluation 记录一次门禁评估；hit 表示本次评估是否命中（存在非通过项）
func RecordGateEvaluation(hit bool) {
	gateEvaluations.Add(1)
	if hit {
		gateHitEvaluations.Add(1)
	}
}

// SetGateMetricProvider 注册复审闭环指标来源（传 nil 取消）
func SetGateMetricProvider(p GateMetricProvider) {
	gateMetricMu.Lock()
	gateMetricProvider = p
	gateMetricMu.Unlock()
}

// GateMetricsSnapshot 汇总当前门禁指标
func GateMetricsSnapshot() GateMetrics {
	evals := gateEvaluations.Load()
	hitEvals := gateHitEvaluations.Load()

	m := GateMetrics{
		Evaluations:    evals,
		HitEvaluations: hitEvals,
		Hits:           int64(GateAuditTotal()),
		HitsByLayer:    map[string]int64{},
		HitsByLevel:    map[string]int64{},
		HitsByCategory: map[string]int64{},
	}
	if evals > 0 {
		m.HitRate = float64(hitEvals) / float64(evals)
	}
	for k, v := range GateAuditCounts() {
		switch {
		case strings.HasPrefix(k, "layer:"):
			m.HitsByLayer[strings.TrimPrefix(k, "layer:")] = int64(v)
		case strings.HasPrefix(k, "level:"):
			m.HitsByLevel[strings.TrimPrefix(k, "level:")] = int64(v)
		case strings.HasPrefix(k, "category:"):
			m.HitsByCategory[strings.TrimPrefix(k, "category:")] = int64(v)
		}
	}

	gateMetricMu.Lock()
	p := gateMetricProvider
	gateMetricMu.Unlock()
	if p != nil {
		m.FalsePositiveRate = p.FalsePositiveRate()
		m.SampleMissRate = p.SampleMissRate()
		m.ReviewLatencyMs = p.ReviewLatencyMs()
	}
	return m
}

// ResetGateMetrics 清零评估计数并注销指标来源（供测试与工具重置）
func ResetGateMetrics() {
	gateEvaluations.Store(0)
	gateHitEvaluations.Store(0)
	gateMetricMu.Lock()
	gateMetricProvider = nil
	gateMetricMu.Unlock()
}
