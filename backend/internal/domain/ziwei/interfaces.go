package ziwei

// ZiweiAnalyzer 紫微斗数分析接口
// 提供紫微斗数排盘分析能力，由应用层适配器实现
// gender 用于大限顺逆（阳男阴女顺行，阴男阳女逆行），取值 "男"/"male" 或 "女"/"female"
type ZiweiAnalyzer interface {
	Analyze(year, month, day, hour int, gender string) *ZiweiAnalysis
}
