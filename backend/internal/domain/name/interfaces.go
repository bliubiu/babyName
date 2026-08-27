package name

// EnhancedNameAnalyzer 增强名字分析接口
// 提供带多维评分的名字生成和分析能力，由应用层适配器实现
type EnhancedNameAnalyzer interface {
	GenerateWithAnalysis(opts GenerateOptions) ([]*NameAnalysis, error)
	// GenerateUnified 使用统一评分体系生成名字（含7维评分+双字组合评估）
	GenerateUnified(opts GenerateOptions) ([]Name, error)
}
