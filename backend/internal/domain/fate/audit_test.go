package fate

import (
	"strings"
	"testing"
	"time"
)

// restoreAudit 恢复审计全局态，避免测试间串扰
func restoreAudit(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		SetGateAuditSink(nil)
		SetGateAuditThrottle(defaultGateAuditThrottle)
		ResetGateAudit()
	})
}

func TestMaskAuditPII(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"空串", "", ""},
		{"普通名字不变", "张伟", "张伟"},
		{"手机号掩码", "联系 13812345678 咨询", "联系 *** 咨询"},
		{"身份证掩码", "证件 110101199003074567 号", "证件 *** 号"},
		{"身份证X尾号掩码", "证件 11010119900307456X 号", "证件 *** 号"},
		{"邮箱掩码", "邮箱 a.b-c@example.com 收", "邮箱 *** 收"},
		{"多模式同时掩码", "13812345678@qq.com", "***"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := maskAuditPII(c.in); got != c.want {
				t.Fatalf("maskAuditPII(%q) = %q, 期望 %q", c.in, got, c.want)
			}
		})
	}
}

func TestRiskCategoryLayer(t *testing.T) {
	cases := map[string]string{
		"格式":    "L0",
		"生僻字":   "L1",
		"门禁字":   "L1",
		"户籍友好度": "L1",
		"负面联想":  "L2",
		"撞名热度":  "L2",
		"敏感词":   "L2",
		"谐音":    "L3",
		"多音字":   "L3",
		"谐音连读":  "L3",
		"未知类别":  "L4",
	}
	for category, want := range cases {
		if got := riskCategoryLayer(category); got != want {
			t.Fatalf("riskCategoryLayer(%q) = %q, 期望 %q", category, got, want)
		}
	}
}

func TestRecordGateHitCountsAndRecent(t *testing.T) {
	ResetGateAudit()
	SetGateAuditThrottle(0)
	restoreAudit(t)

	RecordGateHit(GateHit{Layer: "L3", Category: "谐音连读", Level: string(RiskFail), Name: "张沉默", HitValue: "沉默", Rule: "不吉谐音"})
	RecordGateHit(GateHit{Layer: "L1", Category: "门禁字", Level: string(RiskFail), Name: "张一", HitValue: "一"})

	if got := GateAuditTotal(); got != 2 {
		t.Fatalf("总数 = %d, 期望 2", got)
	}
	counts := GateAuditCounts()
	if counts["level:fail"] != 2 {
		t.Fatalf("level:fail = %d, 期望 2", counts["level:fail"])
	}
	if counts["layer:L3"] != 1 || counts["layer:L1"] != 1 {
		t.Fatalf("layer 计数异常: %v", counts)
	}
	if counts["category:谐音连读"] != 1 {
		t.Fatalf("category 计数异常: %v", counts)
	}

	recent := GateAuditRecent(10)
	if len(recent) != 2 {
		t.Fatalf("详情条数 = %d, 期望 2", len(recent))
	}
	if recent[0].Name != "张沉默" || recent[1].Name != "张一" {
		t.Fatalf("详情顺序/内容异常: %+v", recent)
	}
	if recent[0].Time.IsZero() {
		t.Fatal("命中时间不应为零值")
	}

	// 返回副本：外部修改不影响内部
	counts["level:fail"] = 99
	if GateAuditCounts()["level:fail"] != 2 {
		t.Fatal("GateAuditCounts 应返回副本")
	}
	recent[0].Name = "篡改"
	if GateAuditRecent(10)[0].Name != "张沉默" {
		t.Fatal("GateAuditRecent 应返回副本")
	}
}

func TestGateAuditRecentLimit(t *testing.T) {
	ResetGateAudit()
	SetGateAuditThrottle(0)
	restoreAudit(t)

	for i := 0; i < 5; i++ {
		RecordGateHit(GateHit{Layer: "L1", Category: "门禁字", Level: string(RiskFail), HitValue: string(rune('a' + i))})
	}
	if got := len(GateAuditRecent(3)); got != 3 {
		t.Fatalf("limit=3 返回 %d 条, 期望 3", got)
	}
	if got := len(GateAuditRecent(0)); got != 5 {
		t.Fatalf("limit=0 返回 %d 条, 期望全部 5", got)
	}
	if got := len(GateAuditRecent(100)); got != 5 {
		t.Fatalf("limit 超界返回 %d 条, 期望 5", got)
	}
}

func TestRecordGateHitThrottle(t *testing.T) {
	ResetGateAudit()
	SetGateAuditThrottle(time.Minute)
	restoreAudit(t)

	hit := GateHit{Layer: "L3", Category: "谐音连读", Level: string(RiskFail), HitValue: "沉默"}
	for i := 0; i < 5; i++ {
		RecordGateHit(hit)
	}
	if got := GateAuditTotal(); got != 5 {
		t.Fatalf("节流不应影响计数: 总数 = %d, 期望 5", got)
	}
	if got := len(GateAuditRecent(10)); got != 1 {
		t.Fatalf("同键窗口内详情应为 1 条, 实际 %d", got)
	}

	// 不同命中键互不节流
	hit.HitValue = "死亡"
	RecordGateHit(hit)
	if got := len(GateAuditRecent(10)); got != 2 {
		t.Fatalf("不同键应各自记录, 实际 %d 条", got)
	}
}

func TestGateAuditSink(t *testing.T) {
	ResetGateAudit()
	SetGateAuditThrottle(0)
	restoreAudit(t)

	var got []GateHit
	SetGateAuditSink(func(h GateHit) { got = append(got, h) })

	RecordGateHit(GateHit{Layer: "L4", Category: "负面联想", Level: string(RiskWarn), Name: "李四", Rule: "消极"})
	if len(got) != 1 || got[0].Name != "李四" {
		t.Fatalf("sink 未收到事件: %+v", got)
	}

	// 取消 sink 后不再回调
	SetGateAuditSink(nil)
	RecordGateHit(GateHit{Layer: "L4", Category: "负面联想", Level: string(RiskWarn), Name: "王五"})
	if len(got) != 1 {
		t.Fatalf("取消 sink 后不应再回调, 实际 %d 条", len(got))
	}
	if GateAuditTotal() != 2 {
		t.Fatalf("取消 sink 不应影响计数, 总数 = %d", GateAuditTotal())
	}
}

func TestRecordGateHitMasksPII(t *testing.T) {
	ResetGateAudit()
	SetGateAuditThrottle(0)
	restoreAudit(t)

	RecordGateHit(GateHit{
		Layer:    "L3",
		Category: "谐音连读",
		Level:    string(RiskFail),
		Name:     "张13812345678",
		HitValue: "a@b.com",
		Rule:     "手机 13900000000",
	})
	r := GateAuditRecent(1)[0]
	if strings.Contains(r.Name, "13812345678") || strings.Contains(r.HitValue, "@") || strings.Contains(r.Rule, "13900000000") {
		t.Fatalf("埋点未脱敏: %+v", r)
	}
	if r.Name != "张***" || r.HitValue != "***" || r.Rule != "手机 ***" {
		t.Fatalf("脱敏结果异常: %+v", r)
	}
}

func TestResetGateAudit(t *testing.T) {
	ResetGateAudit()
	SetGateAuditThrottle(0)
	restoreAudit(t)

	RecordGateHit(GateHit{Layer: "L1", Category: "门禁字", Level: string(RiskFail)})
	ResetGateAudit()
	if GateAuditTotal() != 0 {
		t.Fatalf("重置后总数应为 0, 实际 %d", GateAuditTotal())
	}
	if len(GateAuditCounts()) != 0 {
		t.Fatalf("重置后计数应为空, 实际 %v", GateAuditCounts())
	}
	if len(GateAuditRecent(5)) != 0 {
		t.Fatal("重置后详情应为空")
	}
}

func TestSetGateAuditThrottleNegative(t *testing.T) {
	ResetGateAudit()
	SetGateAuditThrottle(0)
	restoreAudit(t)

	// 负数窗口按 0（不节流）处理：连续同键也不去重
	SetGateAuditThrottle(-time.Second)
	hit := GateHit{Layer: "L1", Category: "门禁字", Level: string(RiskFail), HitValue: "一"}
	RecordGateHit(hit)
	RecordGateHit(hit)
	if got := len(GateAuditRecent(10)); got != 2 {
		t.Fatalf("负窗口应按不节流处理, 实际 %d 条", got)
	}
}

func TestRecordGateHitPreservesTimeAndRingCap(t *testing.T) {
	ResetGateAudit()
	SetGateAuditThrottle(0)
	restoreAudit(t)

	// 显式时间应被保留（不再覆盖为当前时间）
	fixed := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	RecordGateHit(GateHit{Time: fixed, Layer: "L1", Category: "门禁字", Level: string(RiskFail), HitValue: "时刻"})
	if got := GateAuditRecent(1)[0].Time; !got.Equal(fixed) {
		t.Fatalf("应保留调用方传入的时间, 实际 %v", got)
	}

	// 超过环形容量时只保留最近 gateAuditRingCapacity 条
	ResetGateAudit()
	for i := 0; i < gateAuditRingCapacity+5; i++ {
		RecordGateHit(GateHit{Layer: "L1", Category: "门禁字", Level: string(RiskFail), HitValue: "一"})
	}
	if got := len(GateAuditRecent(0)); got != gateAuditRingCapacity {
		t.Fatalf("环形缓冲应封顶 %d, 实际 %d", gateAuditRingCapacity, got)
	}
	if got := GateAuditTotal(); got != gateAuditRingCapacity+5 {
		t.Fatalf("计数不受环形容量影响, 实际 %d", got)
	}
}
