// Package gateaudit 把领域层的门禁命中事件落地为结构化日志（docs/33 门禁层 M3）。
//
// 该包是「domain 产出事件 → 基础设施落地」的解耦点：领域层 fate 只负责生成
// 脱敏后的 GateHit，本包负责把它写成可被日志系统收集的结构化条目，后续 M4 的
// 指标派生也会挂在这里。
package gateaudit

import (
	"name/internal/domain/fate"
	"name/internal/infrastructure/logger"
)

// Install 注册门禁命中审计的落地接收器：以结构化字段写入应用日志。
//
// 脱敏已在领域层（fate.RecordGateHit）完成，这里只做落地，不再额外处理。
// 未初始化日志时 logger.Info 是空操作，因此可在任何时机安全调用。
func Install() {
	fate.SetGateAuditSink(func(h fate.GateHit) {
		logger.Info("门禁命中",
			logger.String("来源", h.Source),
			logger.String("层级", h.Layer),
			logger.String("类别", h.Category),
			logger.String("等级", h.Level),
			logger.String("名字", h.Name),
			logger.String("命中值", h.HitValue),
			logger.String("说明", h.Rule),
		)
	})
}
