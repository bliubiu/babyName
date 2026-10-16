package fate

import (
	"testing"
)

type stubGateMetricProvider struct{ fp, miss, latency float64 }

func (s stubGateMetricProvider) FalsePositiveRate() float64 { return s.fp }
func (s stubGateMetricProvider) SampleMissRate() float64    { return s.miss }
func (s stubGateMetricProvider) ReviewLatencyMs() float64   { return s.latency }

// resetGateMetricsForTest 保证指标/审计全局态测试间隔离
func resetGateMetricsForTest(t *testing.T) {
	t.Helper()
	ResetGateAudit()
	ResetGateMetrics()
	SetGateAuditThrottle(0)
	t.Cleanup(func() {
		ResetGateAudit()
		ResetGateMetrics()
		SetGateAuditThrottle(defaultGateAuditThrottle)
	})
}

func TestRecordGateEvaluationHitRate(t *testing.T) {
	resetGateMetricsForTest(t)

	RecordGateEvaluation(false)
	RecordGateEvaluation(true)
	RecordGateEvaluation(true)
	RecordGateEvaluation(false)

	m := GateMetricsSnapshot()
	if m.Evaluations != 4 || m.HitEvaluations != 2 {
		t.Fatalf("评估计数异常: %+v", m)
	}
	if m.HitRate != 0.5 {
		t.Fatalf("命中率 = %v, 期望 0.5", m.HitRate)
	}
}

func TestGateMetricsSnapshotNoEvaluation(t *testing.T) {
	resetGateMetricsForTest(t)

	m := GateMetricsSnapshot()
	if m.Evaluations != 0 || m.HitEvaluations != 0 || m.HitRate != 0 {
		t.Fatalf("无评估时命中率应为 0: %+v", m)
	}
	if len(m.HitsByLayer) != 0 || len(m.HitsByLevel) != 0 || len(m.HitsByCategory) != 0 {
		t.Fatalf("无命中时分布应为空: %+v", m)
	}
}

func TestGateMetricsSnapshotDistribution(t *testing.T) {
	resetGateMetricsForTest(t)

	RecordGateHit(GateHit{Layer: "L3", Category: "谐音连读", Level: string(RiskFail), HitValue: "沉默"})
	RecordGateHit(GateHit{Layer: "L1", Category: "门禁字", Level: string(RiskFail), HitValue: "一"})
	RecordGateHit(GateHit{Layer: "L2", Category: "负面联想", Level: string(RiskWarn), HitValue: "病"})

	m := GateMetricsSnapshot()
	if m.Hits != 3 {
		t.Fatalf("命中事件总数 = %d, 期望 3", m.Hits)
	}
	if m.HitsByLayer["L3"] != 1 || m.HitsByLayer["L1"] != 1 || m.HitsByLayer["L2"] != 1 {
		t.Fatalf("分层分布异常: %v", m.HitsByLayer)
	}
	if m.HitsByLevel[string(RiskFail)] != 2 || m.HitsByLevel[string(RiskWarn)] != 1 {
		t.Fatalf("等级分布异常: %v", m.HitsByLevel)
	}
	if m.HitsByCategory["谐音连读"] != 1 || m.HitsByCategory["门禁字"] != 1 {
		t.Fatalf("类别分布异常: %v", m.HitsByCategory)
	}
}

func TestGateMetricsProvider(t *testing.T) {
	resetGateMetricsForTest(t)

	m := GateMetricsSnapshot()
	if m.FalsePositiveRate != 0 || m.SampleMissRate != 0 || m.ReviewLatencyMs != 0 {
		t.Fatalf("未注册来源时应为零: %+v", m)
	}

	SetGateMetricProvider(stubGateMetricProvider{fp: 0.25, miss: 0.1, latency: 12.5})
	m = GateMetricsSnapshot()
	if m.FalsePositiveRate != 0.25 || m.SampleMissRate != 0.1 || m.ReviewLatencyMs != 12.5 {
		t.Fatalf("来源值未透传: %+v", m)
	}

	SetGateMetricProvider(nil)
	m = GateMetricsSnapshot()
	if m.FalsePositiveRate != 0 || m.SampleMissRate != 0 || m.ReviewLatencyMs != 0 {
		t.Fatalf("注销来源后应回零: %+v", m)
	}
}

func TestResetGateMetrics(t *testing.T) {
	resetGateMetricsForTest(t)

	RecordGateEvaluation(true)
	SetGateMetricProvider(stubGateMetricProvider{fp: 0.5})
	ResetGateMetrics()
	m := GateMetricsSnapshot()
	if m.Evaluations != 0 || m.HitEvaluations != 0 || m.FalsePositiveRate != 0 {
		t.Fatalf("重置后应清零: %+v", m)
	}
}

// TestRecordAssessHitsEvaluation 直接覆盖 recordAssessHits 的命中/未命中两分支
func TestRecordAssessHitsEvaluation(t *testing.T) {
	resetGateMetricsForTest(t)

	// 全 pass：评估 +1，命中评估 +0
	recordAssessHits("王", "铭", []RiskItem{
		{Category: "撞名热度", Level: RiskPass, Detail: "重名率低"},
		{Category: "户籍友好度", Level: RiskPass, Detail: "录入无障碍"},
	})
	// 含 warn：评估 +1，命中评估 +1，命中事件 +1
	recordAssessHits("王", "翀", []RiskItem{
		{Category: "生僻字", Level: RiskWarn, Detail: "三级字"},
	})

	m := GateMetricsSnapshot()
	if m.Evaluations != 2 || m.HitEvaluations != 1 {
		t.Fatalf("评估计数异常: %+v", m)
	}
	if m.HitRate != 0.5 {
		t.Fatalf("命中率 = %v, 期望 0.5", m.HitRate)
	}
	if m.Hits != 1 {
		t.Fatalf("命中事件总数 = %d, 期望 1（pass 不计入）", m.Hits)
	}
}

// TestAssessNameRisksRecordsGateEvaluation 验证真实体检路径会写入评估计数
func TestAssessNameRisksRecordsGateEvaluation(t *testing.T) {
	resetGateMetricsForTest(t)

	p := riskStubProvider(&Character{Char: "翀", Pinyin: []string{"chōng"}, CommonLevel: 3})
	_ = AssessNameRisks(p, "王", "翀")

	m := GateMetricsSnapshot()
	if m.Evaluations != 1 {
		t.Fatalf("体检应记录一次评估, 实际 %d", m.Evaluations)
	}
	if m.HitEvaluations != 1 || m.HitRate != 1 {
		t.Fatalf("含三级字体检应计为命中, 实际 %+v", m)
	}
}
