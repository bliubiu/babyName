package fate

import "strings"

// NormalizeGenerationPosition 统一字辈位置枚举。
//
// 引擎内部只认 "tail"（字辈固定最后），其余一律按 "middle"（固定中间）处理；
// 前端历史值 "end"、外部接口可能传入的 "last" 都归一到 "tail"，避免静默失效。
func NormalizeGenerationPosition(position string) string {
	switch strings.ToLower(strings.TrimSpace(position)) {
	case "tail", "end", "last":
		return "tail"
	default:
		return "middle"
	}
}
