package namestat

type NameStat struct {
	Name       string  `json:"name"`
	Count      int     `json:"count"`
	Rate       float64 `json:"rate"`
	Rank       int     `json:"rank"`
	Province   string  `json:"province,omitempty"`
	YearRange  string  `json:"year_range,omitempty"`
}

var CommonNameStats = map[string]int{
	"伟": 2456000, "芳": 2120000, "娜": 1890000, "秀英": 1650000,
	"敏": 1580000, "静": 1520000, "丽": 2150000, "强": 1420000,
	"军": 1380000, "杰": 1350000, "涛": 1280000, "明": 1250000,
	"超": 1180000, "勇": 1150000, "鹏": 1120000, "华": 1200000,
	"磊": 1050000, "刚": 980000, "平": 950000, "辉": 920000,
	"波": 890000, "峰": 860000, "飞": 830000, "龙": 800000,
	"浩": 780000, "宇": 750000, "晨": 720000, "逸": 690000,
	"睿": 660000, "哲": 630000, "渊": 600000, "博": 570000,
	"昊": 540000, "然": 510000, "轩": 480000, "俊": 450000,
	"豪": 420000, "毅": 390000, "文": 360000, "武": 330000,
	"祥": 300000, "瑞": 280000, "凯": 260000, "成": 240000,
	"盛": 220000, "雄": 200000, "鑫": 190000, "霖": 180000,
	"雨": 2100000, "婷": 1950000, "雪": 1820000, "梅": 1680000,
	"兰": 1550000, "琴": 1420000, "娟": 1380000, "莉": 1320000,
	"萍": 1250000, "颖": 1180000, "怡": 1120000, "菲": 1050000,
	"雅": 980000, "芸": 920000, "萱": 850000, "诗": 780000,
	"梦": 720000, "瑶": 680000, "琳": 640000, "珂": 600000,
	"雯": 560000, "倩": 520000, "翠": 580000, "碧": 440000,
	"银": 400000, "玉": 380000, "珠": 350000, "珍": 320000,
	"红": 3100000, "燕": 2850000, "霞": 2650000, "英": 2400000,
	"桂": 1800000, "香": 1650000, "菊": 1450000,
	"凤": 1050000, "花": 950000, "云": 880000,
	"清": 820000, "青": 750000, "丹": 680000, "虹": 620000,
	"荣": 540000, "芝": 500000,
}

var ProvinceNameStats = map[string]map[string]int{
	"北京": {"伟": 85000, "芳": 72000, "娜": 68000, "静": 65000, "杰": 62000, "涛": 58000},
	"上海": {"伟": 92000, "芳": 78000, "娜": 72000, "静": 68000, "明": 65000, "杰": 62000},
	"广东": {"伟": 125000, "芳": 98000, "娜": 92000, "静": 85000, "勇": 82000, "杰": 78000},
	"浙江": {"伟": 88000, "芳": 72000, "娜": 68000, "静": 62000, "杰": 58000, "磊": 55000},
	"江苏": {"伟": 95000, "芳": 78000, "娜": 72000, "静": 68000, "明": 62000, "杰": 58000},
	"四川": {"伟": 82000, "芳": 68000, "娜": 62000, "静": 58000, "勇": 55000, "杰": 52000},
	"湖北": {"伟": 78000, "芳": 65000, "娜": 58000, "静": 55000, "勇": 52000, "杰": 48000},
	"湖南": {"伟": 82000, "芳": 68000, "娜": 62000, "静": 58000, "强": 55000, "杰": 52000},
	"山东": {"伟": 92000, "芳": 78000, "娜": 72000, "静": 68000, "强": 65000, "杰": 62000},
	"河南": {"伟": 98000, "芳": 82000, "娜": 75000, "静": 72000, "强": 68000, "杰": 65000},
}

func GetNameCount(name string) int {
	count, ok := CommonNameStats[name]
	if ok {
		return count
	}
	return 0
}

func GetNameStats(name string) *NameStat {
	count := GetNameCount(name)
	if count == 0 {
		return &NameStat{
			Name:  name,
			Count: 0,
			Rate:  0.0,
			Rank:  0,
		}
	}

	rank := 1
	for _, c := range CommonNameStats {
		if c > count {
			rank++
		}
	}

	rate := float64(count) / 140000000.0 * 100

	return &NameStat{
		Name:  name,
		Count: count,
		Rate:  rate,
		Rank:  rank,
	}
}

func GetTopNames(limit int) []NameStat {
	stats := make([]NameStat, 0, len(CommonNameStats))
	for name, count := range CommonNameStats {
		rate := float64(count) / 140000000.0 * 100
		stats = append(stats, NameStat{
			Name: name,
			Count: count,
			Rate: rate,
		})
	}

	for i := 0; i < len(stats)-1; i++ {
		for j := i + 1; j < len(stats); j++ {
			if stats[j].Count > stats[i].Count {
				stats[i], stats[j] = stats[j], stats[i]
			}
		}
	}

	if limit > 0 && len(stats) > limit {
		stats = stats[:limit]
	}

	return stats
}

func GetProvinceNameStats(name, province string) *NameStat {
	provinceStats, ok := ProvinceNameStats[province]
	if !ok {
		provinceStats = ProvinceNameStats["北京"]
	}

	count, ok := provinceStats[name]
	if !ok {
		count = GetNameCount(name) / 100
	}

	total := 10000000.0

	return &NameStat{
		Name:     name,
		Count:    count,
		Rate:     float64(count) / total * 100,
		Province: province,
	}
}
