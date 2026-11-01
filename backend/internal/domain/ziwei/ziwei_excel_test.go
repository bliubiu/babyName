package ziwei

import (
	"testing"
)

// ziwei_excel_test.go Excel 派扩展杂曜（天刑/三台/八座/恩光/天贵/天才/天寿/
// 台辅/封诰/天官/天福/天厨）iztro 2.6.3 实排对拍（docs/41 第 7 节遗留项）。
//
// 锚点由临时工程 npm 安装 iztro@2.6.3 后以 astro.bySolar 实排 8 组生辰提取
// （脚本存于临时目录，不随仓库分发）；宫名已从 iztro 命名归一为本项目
// Gongs 口径（iztro「仆役」= 本项目「交友」）。

type excelAnchorCase struct {
	year, month, day, hour int
	gender                 string
	want                   map[string]string // 星 → 宫名
}

var excelAnchorCases = []excelAnchorCase{
	{
		2000, 8, 16, 10, "女", // 庚辰年七月十七 巳时（既有样本盘）
		map[string]string{
			"天刑": "命宫", "三台": "兄弟宫", "八座": "子女宫", "恩光": "交友宫",
			"天贵": "子女宫", "天才": "官禄宫", "天寿": "福德宫", "台辅": "财帛宫",
			"封诰": "官禄宫", "天官": "财帛宫", "天福": "田宅宫", "天厨": "兄弟宫",
		},
	},
	{
		2004, 2, 15, 0, "男", // 闰二月前 子时男
		map[string]string{
			"天刑": "疾厄宫", "三台": "福德宫", "八座": "财帛宫", "恩光": "疾厄宫",
			"天贵": "父母宫", "天才": "财帛宫", "天寿": "财帛宫", "台辅": "官禄宫",
			"封诰": "命宫", "天官": "交友宫", "天福": "疾厄宫", "天厨": "田宅宫",
		},
	},
	{
		2004, 3, 10, 23, "男", // 闰二月下半 + 晚子时（日系偏移按次日计）
		map[string]string{
			"天刑": "疾厄宫", "三台": "夫妻宫", "八座": "夫妻宫", "恩光": "福德宫",
			"天贵": "财帛宫", "天才": "财帛宫", "天寿": "财帛宫", "台辅": "田宅宫",
			"封诰": "兄弟宫", "天官": "官禄宫", "天福": "迁移宫", "天厨": "福德宫",
		},
	},
	{
		1985, 6, 21, 12, "女",
		map[string]string{
			"天刑": "父母宫", "三台": "兄弟宫", "八座": "田宅宫", "恩光": "迁移宫",
			"天贵": "命宫", "天才": "父母宫", "天寿": "父母宫", "台辅": "命宫",
			"封诰": "财帛宫", "天官": "官禄宫", "天福": "财帛宫", "天厨": "迁移宫",
		},
	},
	{
		1990, 11, 5, 3, "男", // 寅时（hour 3-4 → idx2）
		map[string]string{
			"天刑": "子女宫", "三台": "夫妻宫", "八座": "命宫", "恩光": "交友宫",
			"天贵": "田宅宫", "天才": "迁移宫", "天寿": "夫妻宫", "台辅": "命宫",
			"封诰": "财帛宫", "天官": "田宅宫", "天福": "夫妻宫", "天厨": "迁移宫",
		},
	},
	{
		2023, 8, 8, 18, "女", // 酉时
		map[string]string{
			"天刑": "官禄宫", "三台": "财帛宫", "八座": "夫妻宫", "恩光": "兄弟宫",
			"天贵": "兄弟宫", "天才": "田宅宫", "天寿": "子女宫", "台辅": "交友宫",
			"封诰": "父母宫", "天官": "财帛宫", "天福": "疾厄宫", "天厨": "父母宫",
		},
	},
	{
		2012, 5, 20, 22, "男", // 亥时
		map[string]string{
			"天刑": "迁移宫", "三台": "迁移宫", "八座": "财帛宫", "恩光": "子女宫",
			"天贵": "父母宫", "天才": "官禄宫", "天寿": "福德宫", "台辅": "兄弟宫",
			"封诰": "疾厄宫", "天官": "官禄宫", "天福": "命宫", "天厨": "田宅宫",
		},
	},
	{
		1999, 12, 30, 13, "女", // 未时（hour 13-14 → idx7）
		map[string]string{
			"天刑": "福德宫", "三台": "疾厄宫", "八座": "子女宫", "恩光": "疾厄宫",
			"天贵": "田宅宫", "天才": "田宅宫", "天寿": "交友宫", "台辅": "财帛宫",
			"封诰": "官禄宫", "天官": "官禄宫", "天福": "子女宫", "天厨": "田宅宫",
		},
	},
}

// TestExcelStars_Iztro对拍 8 盘 12 星全对拍 + 全盘 24 杂曜唯一落位
func TestExcelStars_Iztro对拍(t *testing.T) {
	for ci, c := range excelAnchorCases {
		chart := CalculateZiweiChart(c.year, c.month, c.day, c.hour, c.gender)
		if chart == nil {
			t.Fatalf("case%d: 排盘失败", ci+1)
		}
		for star, wantGong := range c.want {
			gong := findStarPalace(chart.SuiXing, star)
			if gong == "" {
				t.Errorf("case%d: 星 %s 未落任何宫", ci+1, star)
				continue
			}
			if gong != wantGong {
				t.Errorf("case%d: 星 %s 应落 %s，实落 %s", ci+1, star, wantGong, gong)
			}
		}
		// 24 颗杂曜全部恰好落位一次
		seen := map[string]int{}
		for _, stars := range chart.SuiXing {
			for _, s := range stars {
				seen[s]++
			}
		}
		if len(seen) != len(SuiXingList) {
			t.Errorf("case%d: 落位杂曜 %d 种，期望 %d", ci+1, len(seen), len(SuiXingList))
		}
		for star, n := range seen {
			if n != 1 {
				t.Errorf("case%d: 杂曜 %s 落位 %d 次，期望恰 1 次", ci+1, star, n)
			}
		}
	}
}

// findStarPalace 在杂曜分布中查星所在宫名，未找到返回空串
func findStarPalace(suiXing map[string][]string, star string) string {
	for gong, stars := range suiXing {
		for _, s := range stars {
			if s == star {
				return gong
			}
		}
	}
	return ""
}

// TestSuiXingList_Completeness SuiXingList 与安布清单一致：每一颗都能在
// 任意一盘上落位（防清单与安布函数脱节）
func TestSuiXingList_Completeness(t *testing.T) {
	chart := CalculateZiweiChart(2000, 8, 16, 10, "女")
	for _, star := range SuiXingList {
		if findStarPalace(chart.SuiXing, star) == "" {
			t.Errorf("SuiXingList 中的 %s 未在任何宫落位", star)
		}
	}
}
