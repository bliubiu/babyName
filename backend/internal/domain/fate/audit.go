package fate

import (
	"regexp"
	"sync"
	"time"
)

// ——— 门禁命中审计埋点（docs/33 门禁层 M3） ———
//
// 目的：把「命中侧」结构化为可观测事件，供运营闭环（命中率/误报分析）与
// tools/gate_audit 消费。设计要点：
//
//   - 脱敏优先：写入前对 名字/命中值/说明 统一做 PII 掩码（身份证/手机号/邮箱）。
//     名字虽是产物、天然无 PII，但埋点必须对「误传用户输入」也安全，故不例外。
//   - 热路径友好：命中可能在双名 N² 评分中被高频触发，采用「全量计数 + 详情节流」
//     —— 计数始终精确累加，详情按 (层级,类别,等级,命中值) 去重且窗口内只留一条。
//   - 分层解耦：domain 只产出事件，落日志/指标由基础设施注册 GateAuditSink 完成，
//     domain 不依赖任何 infrastructure 包。

// GateHit 一次门禁命中事件（结构化、已脱敏）
type GateHit struct {
	Time     time.Time `json:"time"`      // 命中时间
	Source   string    `json:"source"`    // 产生路径：评分链路 / 测名体检
	Layer    string    `json:"layer"`     // 层级：L0~L5
	Category string    `json:"category"`  // 命中类别
	Level    string    `json:"level"`     // 等级：warn / fail
	Name     string    `json:"name"`      // 名字（产物，已脱敏）
	HitValue string    `json:"hit_value"` // 命中值（如谐音词，已脱敏）
	Rule     string    `json:"rule"`      // 可解释说明（已脱敏）
}

// GateAuditSink 审计事件的落地接收器，由基础设施层注册
type GateAuditSink func(GateHit)

const (
	// gateAuditRingCapacity 详情环形缓冲容量（计数不受此限制）
	gateAuditRingCapacity = 256
	// defaultGateAuditThrottle 同一命中键的详情节流窗口
	defaultGateAuditThrottle = time.Second
)

var (
	auditMu       sync.Mutex
	auditSink     GateAuditSink
	auditRing     []GateHit
	auditTotal    int
	auditCounts   = make(map[string]int)
	auditLastSeen = make(map[string]time.Time)
	auditThrottle = defaultGateAuditThrottle
)

// SetGateAuditSink 注册（或传 nil 取消）审计事件接收器
func SetGateAuditSink(sink GateAuditSink) {
	auditMu.Lock()
	auditSink = sink
	auditMu.Unlock()
}

// SetGateAuditThrottle 调整详情节流窗口；窗口 <= 0 表示不节流（逐条记录，供测试）
func SetGateAuditThrottle(window time.Duration) {
	if window < 0 {
		window = 0
	}
	auditMu.Lock()
	auditThrottle = window
	auditMu.Unlock()
}

// RecordGateHit 记录一次门禁命中。
//
// 计数（total 与各维度）始终精确累加；详情受节流约束，窗口内同一命中键只留一条。
// 脱敏在入队前完成；sink 在锁外调用，避免阻塞与死锁。
func RecordGateHit(hit GateHit) {
	hit.Name = maskAuditPII(hit.Name)
	hit.HitValue = maskAuditPII(hit.HitValue)
	hit.Rule = maskAuditPII(hit.Rule)

	auditMu.Lock()
	now := time.Now()
	if hit.Time.IsZero() {
		hit.Time = now
	}
	auditTotal++
	auditCounts["level:"+hit.Level]++
	auditCounts["layer:"+hit.Layer]++
	if hit.Category != "" {
		auditCounts["category:"+hit.Category]++
	}

	detail := true
	if auditThrottle > 0 {
		key := hit.Layer + "|" + hit.Category + "|" + hit.Level + "|" + hit.HitValue
		if last, ok := auditLastSeen[key]; ok && now.Sub(last) < auditThrottle {
			detail = false
		} else {
			auditLastSeen[key] = now
		}
	}
	if detail {
		auditRing = append(auditRing, hit)
		if len(auditRing) > gateAuditRingCapacity {
			auditRing = auditRing[len(auditRing)-gateAuditRingCapacity:]
		}
	}
	sink := auditSink
	auditMu.Unlock()

	if detail && sink != nil {
		sink(hit)
	}
}

// GateAuditTotal 命中事件总数（不受节流影响）
func GateAuditTotal() int {
	auditMu.Lock()
	defer auditMu.Unlock()
	return auditTotal
}

// GateAuditCounts 各维度计数的副本（键形如 level:fail / layer:L3 / category:谐音连读）
func GateAuditCounts() map[string]int {
	auditMu.Lock()
	defer auditMu.Unlock()
	out := make(map[string]int, len(auditCounts))
	for k, v := range auditCounts {
		out[k] = v
	}
	return out
}

// GateAuditRecent 返回最近至多 limit 条命中详情（按时间正序）；limit <= 0 返回全部
func GateAuditRecent(limit int) []GateHit {
	auditMu.Lock()
	defer auditMu.Unlock()
	n := len(auditRing)
	if limit <= 0 || limit > n {
		limit = n
	}
	out := make([]GateHit, limit)
	copy(out, auditRing[n-limit:])
	return out
}

// ResetGateAudit 清空计数与详情（保留 sink 与节流配置），供测试与工具重置
func ResetGateAudit() {
	auditMu.Lock()
	auditRing = nil
	auditTotal = 0
	auditCounts = make(map[string]int)
	auditLastSeen = make(map[string]time.Time)
	auditMu.Unlock()
}

// 脱敏规则：身份证 → 邮箱 → 手机号。
//
// 身份证（18 位）最先，避免被手机号规则吃掉尾部 11 位；邮箱在手机号之前，
// 使「手机号@域名」这类邮箱名整体掩码（否则只剩域名）。
var (
	auditIDCardRe = regexp.MustCompile(`\d{17}[\dXx]`)
	auditEmailRe  = regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`)
	auditPhoneRe  = regexp.MustCompile(`1[3-9]\d{9}`)
)

// maskAuditPII 对文本做 PII 掩码，命中片段整体替换为 ***
func maskAuditPII(s string) string {
	if s == "" {
		return s
	}
	s = auditIDCardRe.ReplaceAllString(s, "***")
	s = auditEmailRe.ReplaceAllString(s, "***")
	s = auditPhoneRe.ReplaceAllString(s, "***")
	return s
}

// riskCategoryLayer 把风险体检类别归入门禁分层（docs/33 §二）
func riskCategoryLayer(category string) string {
	switch category {
	case "格式":
		return "L0"
	case "生僻字", "门禁字", "户籍友好度":
		return "L1"
	case "负面联想", "撞名热度", "敏感词":
		return "L2"
	case "谐音", "多音字", "谐音连读":
		return "L3"
	default:
		return "L4"
	}
}
