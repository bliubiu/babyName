package ziwei

// ZiweiAnalyzer 紫微斗数分析接口
// 提供紫微斗数排盘分析能力，由应用层适配器实现
// gender 用于大限顺逆（阳男阴女顺行，阴男阳女逆行），取值 "男"/"male" 或 "女"/"female"
//
// 返回完整 ZiweiChart（12 宫落星 + 14 主星 + 辅星 + 杂曜 + 四化 + 大限 + 亮度 + 分析文本），
// 不再截断为摘要。前端据此渲染完整盘面。
type ZiweiAnalyzer interface {
	Analyze(year, month, day, hour int, gender string) *ZiweiChart
}
