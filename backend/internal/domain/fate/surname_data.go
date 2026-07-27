package fate

// SurnameInfo 姓氏属性信息
type SurnameInfo struct {
	Char           string // 姓氏
	KangxiStroke   int    // 康熙笔画
	WuXing         string // 五行属性
	PopulationRank int    // 人口排名（0 表示未知）
}

// IsValid 是否有效
func (s SurnameInfo) IsValid() bool {
	return s.Char != "" && s.KangxiStroke > 0
}

// SurnameData 常见姓氏康熙笔画与五行属性表
//
// 数据来源：康熙字典 + 百家姓人口排名。康熙笔画用于河图数理与易经卦象解读，
// 与简体/繁体笔画可能不同（如"陈"简体7画、康熙16画）。
// 如需补充姓氏，可向该表添加条目。未收录的姓氏将降级使用简体笔画。
var SurnameData = map[string]SurnameInfo{
	// ——— 前 40 大常见姓氏（数据经 ai4naming 项目验证）———
	"王": {Char: "王", KangxiStroke: 4, WuXing: "土", PopulationRank: 1},
	"李": {Char: "李", KangxiStroke: 7, WuXing: "木", PopulationRank: 2},
	"张": {Char: "张", KangxiStroke: 11, WuXing: "火", PopulationRank: 3},
	"刘": {Char: "刘", KangxiStroke: 15, WuXing: "金", PopulationRank: 4},
	"陈": {Char: "陈", KangxiStroke: 16, WuXing: "火", PopulationRank: 5},
	"杨": {Char: "杨", KangxiStroke: 13, WuXing: "木", PopulationRank: 6},
	"黄": {Char: "黄", KangxiStroke: 12, WuXing: "土", PopulationRank: 7},
	"赵": {Char: "赵", KangxiStroke: 14, WuXing: "火", PopulationRank: 8},
	"吴": {Char: "吴", KangxiStroke: 7, WuXing: "木", PopulationRank: 9},
	"周": {Char: "周", KangxiStroke: 8, WuXing: "金", PopulationRank: 10},
	"徐": {Char: "徐", KangxiStroke: 10, WuXing: "金", PopulationRank: 11},
	"孙": {Char: "孙", KangxiStroke: 10, WuXing: "金", PopulationRank: 12},
	"马": {Char: "马", KangxiStroke: 10, WuXing: "水", PopulationRank: 13},
	"胡": {Char: "胡", KangxiStroke: 11, WuXing: "土", PopulationRank: 14},
	"朱": {Char: "朱", KangxiStroke: 6, WuXing: "木", PopulationRank: 15},
	"郭": {Char: "郭", KangxiStroke: 15, WuXing: "木", PopulationRank: 16},
	"何": {Char: "何", KangxiStroke: 7, WuXing: "木", PopulationRank: 17},
	"林": {Char: "林", KangxiStroke: 8, WuXing: "木", PopulationRank: 18},
	"罗": {Char: "罗", KangxiStroke: 20, WuXing: "火", PopulationRank: 19},
	"高": {Char: "高", KangxiStroke: 10, WuXing: "木", PopulationRank: 20},
	"梁": {Char: "梁", KangxiStroke: 11, WuXing: "火", PopulationRank: 21},
	"郑": {Char: "郑", KangxiStroke: 19, WuXing: "火", PopulationRank: 22},
	"谢": {Char: "谢", KangxiStroke: 17, WuXing: "金", PopulationRank: 23},
	"宋": {Char: "宋", KangxiStroke: 7, WuXing: "金", PopulationRank: 24},
	"唐": {Char: "唐", KangxiStroke: 10, WuXing: "火", PopulationRank: 25},
	"许": {Char: "许", KangxiStroke: 11, WuXing: "木", PopulationRank: 26},
	"邓": {Char: "邓", KangxiStroke: 19, WuXing: "火", PopulationRank: 27},
	"冯": {Char: "冯", KangxiStroke: 12, WuXing: "水", PopulationRank: 28},
	"韩": {Char: "韩", KangxiStroke: 17, WuXing: "水", PopulationRank: 29},
	"曹": {Char: "曹", KangxiStroke: 11, WuXing: "金", PopulationRank: 30},
	"彭": {Char: "彭", KangxiStroke: 12, WuXing: "水", PopulationRank: 31},
	"曾": {Char: "曾", KangxiStroke: 12, WuXing: "金", PopulationRank: 32},
	"肖": {Char: "肖", KangxiStroke: 9, WuXing: "木", PopulationRank: 33},
	"田": {Char: "田", KangxiStroke: 5, WuXing: "火", PopulationRank: 34},
	"董": {Char: "董", KangxiStroke: 15, WuXing: "火", PopulationRank: 35},
	"潘": {Char: "潘", KangxiStroke: 16, WuXing: "水", PopulationRank: 36},
	"袁": {Char: "袁", KangxiStroke: 10, WuXing: "土", PopulationRank: 37},
	"蔡": {Char: "蔡", KangxiStroke: 17, WuXing: "木", PopulationRank: 38},
	"蒋": {Char: "蒋", KangxiStroke: 17, WuXing: "木", PopulationRank: 39},
	"余": {Char: "余", KangxiStroke: 7, WuXing: "土", PopulationRank: 40},

	// ——— 41-100 常见姓氏（补充）———
	"于": {Char: "于", KangxiStroke: 3, WuXing: "土"},
	"叶": {Char: "叶", KangxiStroke: 15, WuXing: "土"},
	"程": {Char: "程", KangxiStroke: 12, WuXing: "木"},
	"魏": {Char: "魏", KangxiStroke: 18, WuXing: "木"},
	"苏": {Char: "苏", KangxiStroke: 22, WuXing: "木"},
	"吕": {Char: "吕", KangxiStroke: 7, WuXing: "火"},
	"丁": {Char: "丁", KangxiStroke: 2, WuXing: "火"},
	"任": {Char: "任", KangxiStroke: 6, WuXing: "金"},
	"沈": {Char: "沈", KangxiStroke: 8, WuXing: "水"},
	"姚": {Char: "姚", KangxiStroke: 9, WuXing: "土"},
	"卢": {Char: "卢", KangxiStroke: 16, WuXing: "火"},
	"傅": {Char: "傅", KangxiStroke: 12, WuXing: "水"},
	"钟": {Char: "钟", KangxiStroke: 20, WuXing: "金"},
	"姜": {Char: "姜", KangxiStroke: 9, WuXing: "土"},
	"崔": {Char: "崔", KangxiStroke: 11, WuXing: "木"},
	"谭": {Char: "谭", KangxiStroke: 19, WuXing: "火"},
	"廖": {Char: "廖", KangxiStroke: 14, WuXing: "火"},
	"汪": {Char: "汪", KangxiStroke: 8, WuXing: "水"},
	"范": {Char: "范", KangxiStroke: 11, WuXing: "水"},
	"金": {Char: "金", KangxiStroke: 8, WuXing: "金"},
	"方": {Char: "方", KangxiStroke: 4, WuXing: "水"},
	"石": {Char: "石", KangxiStroke: 5, WuXing: "金"},
	"夏": {Char: "夏", KangxiStroke: 10, WuXing: "火"},
	"熊": {Char: "熊", KangxiStroke: 14, WuXing: "土"},
	"陆": {Char: "陆", KangxiStroke: 16, WuXing: "火"},
	"孔": {Char: "孔", KangxiStroke: 4, WuXing: "木"},
	"白": {Char: "白", KangxiStroke: 5, WuXing: "水"},
	"毛": {Char: "毛", KangxiStroke: 4, WuXing: "水"},
	"侯": {Char: "侯", KangxiStroke: 9, WuXing: "水"},
	"秦": {Char: "秦", KangxiStroke: 10, WuXing: "火"},
	"顾": {Char: "顾", KangxiStroke: 21, WuXing: "木"},
	"孟": {Char: "孟", KangxiStroke: 8, WuXing: "水"},
	"薛": {Char: "薛", KangxiStroke: 19, WuXing: "木"},
	"尹": {Char: "尹", KangxiStroke: 4, WuXing: "土"},
	"江": {Char: "江", KangxiStroke: 7, WuXing: "水"},
	"汤": {Char: "汤", KangxiStroke: 13, WuXing: "水"},
	"龙": {Char: "龙", KangxiStroke: 16, WuXing: "水"},
	"黎": {Char: "黎", KangxiStroke: 15, WuXing: "土"},
	"易": {Char: "易", KangxiStroke: 8, WuXing: "火"},
	"常": {Char: "常", KangxiStroke: 11, WuXing: "金"},
	"武": {Char: "武", KangxiStroke: 8, WuXing: "水"},
	"乔": {Char: "乔", KangxiStroke: 12, WuXing: "木"},
	"贺": {Char: "贺", KangxiStroke: 12, WuXing: "水"},
	"赖": {Char: "赖", KangxiStroke: 17, WuXing: "火"},
	"龚": {Char: "龚", KangxiStroke: 22, WuXing: "木"},
	"文": {Char: "文", KangxiStroke: 4, WuXing: "水"},
	"段": {Char: "段", KangxiStroke: 9, WuXing: "金"},
	"史": {Char: "史", KangxiStroke: 5, WuXing: "金"},
	"邹": {Char: "邹", KangxiStroke: 17, WuXing: "火"},
	"钱": {Char: "钱", KangxiStroke: 16, WuXing: "金"},
	"严": {Char: "严", KangxiStroke: 20, WuXing: "木"},
	"邱": {Char: "邱", KangxiStroke: 12, WuXing: "木"},
	"温": {Char: "温", KangxiStroke: 14, WuXing: "土"},
	"莫": {Char: "莫", KangxiStroke: 13, WuXing: "水"},
	"颜": {Char: "颜", KangxiStroke: 18, WuXing: "木"},
	"万": {Char: "万", KangxiStroke: 15, WuXing: "土"},
	"康": {Char: "康", KangxiStroke: 11, WuXing: "木"},
	"安": {Char: "安", KangxiStroke: 6, WuXing: "土"},
	"雷": {Char: "雷", KangxiStroke: 13, WuXing: "水"},
	"倪": {Char: "倪", KangxiStroke: 10, WuXing: "金"},
	"樊": {Char: "樊", KangxiStroke: 15, WuXing: "木"},
	"华": {Char: "华", KangxiStroke: 14, WuXing: "水"},
	"萧": {Char: "萧", KangxiStroke: 19, WuXing: "木"},
	"杜": {Char: "杜", KangxiStroke: 7, WuXing: "木"},
	"戴": {Char: "戴", KangxiStroke: 18, WuXing: "火"},
	"洪": {Char: "洪", KangxiStroke: 10, WuXing: "水"},
	"纪": {Char: "纪", KangxiStroke: 9, WuXing: "木"},
	"贾": {Char: "贾", KangxiStroke: 13, WuXing: "金"},
	"章": {Char: "章", KangxiStroke: 11, WuXing: "火"},
	"邢": {Char: "邢", KangxiStroke: 11, WuXing: "金"},
	"伍": {Char: "伍", KangxiStroke: 6, WuXing: "木"},
	"屈": {Char: "屈", KangxiStroke: 8, WuXing: "木"},
	"阮": {Char: "阮", KangxiStroke: 12, WuXing: "木"},
	"蓝": {Char: "蓝", KangxiStroke: 20, WuXing: "木"},
	"闵": {Char: "闵", KangxiStroke: 12, WuXing: "木"},
	"季": {Char: "季", KangxiStroke: 8, WuXing: "木"},
	"甘": {Char: "甘", KangxiStroke: 5, WuXing: "木"},
	"包": {Char: "包", KangxiStroke: 5, WuXing: "水"},
	"关": {Char: "关", KangxiStroke: 19, WuXing: "木"},
	"苗": {Char: "苗", KangxiStroke: 11, WuXing: "水"},
	"柳": {Char: "柳", KangxiStroke: 9, WuXing: "木"},
}

// GetSurnameInfo 查询姓氏的康熙笔画与五行信息
// 返回 (SurnameInfo, found)。未收录时返回零值 + false。
func GetSurnameInfo(surname string) (SurnameInfo, bool) {
	info, ok := SurnameData[surname]
	return info, ok
}

// LookupSurnameStrokes 查找姓氏康熙笔画
// 单姓返回 (l1, 0)；复姓依次返回各字笔画
// 未收录的姓氏逐字查，返回 0
func LookupSurnameStrokes(surname string) (int, int) {
	runes := []rune(surname)
	if len(runes) == 0 {
		return 0, 0
	}

	l1 := lookupOneSurname(runes[0])
	if len(runes) >= 2 {
		return l1, lookupOneSurname(runes[1])
	}
	return l1, 0
}

// lookupOneSurname 查单个姓氏字的康熙笔画
func lookupOneSurname(r rune) int {
	info, ok := SurnameData[string(r)]
	if !ok {
		return 0
	}
	return info.KangxiStroke
}
