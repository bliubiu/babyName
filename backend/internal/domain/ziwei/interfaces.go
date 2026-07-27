package ziwei

// ZiweiAnalyzer 紫微斗数分析接口
// 提供紫微斗数排盘分析能力，由应用层适配器实现
type ZiweiAnalyzer interface {
	Analyze(year, month, day, hour int) *ZiweiAnalysis
}
