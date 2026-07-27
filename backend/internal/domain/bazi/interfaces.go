package bazi

// BaziAnalyzer 八字分析接口
// 提供生辰八字分析能力，由应用层适配器实现
type BaziAnalyzer interface {
	Analyze(year, month, day, hour, minute int) (*BaziAnalysis, error)
}
