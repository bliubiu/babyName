package zodiac

// ZodiacFinder 生肖查找接口
// 提供根据出生年份查找生肖的能力，由应用层适配器实现
type ZodiacFinder interface {
	FindByYear(year int) string
}
