package name

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"name/internal/domain/hanzi"
)

// ============================================================
// 名字过滤器 — 安全策略全面落地
// ============================================================

// NameFilterResult 过滤结果
type NameFilterResult struct {
	Valid  bool
	Issues []string
}

// badCrossSyllable 跨音节不良连读拼音词条
type badCrossSyllable struct {
	pattern  string // 去声调拼音（小写），如 "wangba"
	severity string // "hard" 硬过滤 / "soft" 软提醒
	desc     string // 不良说明
}

// NameFilter 名字过滤器
type NameFilter struct {
	badHomophones        []string            // 不良谐音直配
	badCrossSyllablePinyins []badCrossSyllable // 跨音节不良连读拼音
	eraChars             []string            // 时代特征过强字
	historicalNames      []string            // 历史名人全名
	meaningNegative      []string            // 含义消极字
	structureCategory    map[string]string   // 字形结构分类
	forbiddenEntities    map[string]bool     // 禁止的国家机关单位名称
}

// NewNameFilter 创建名字过滤器
func NewNameFilter() *NameFilter {
	return &NameFilter{
		badHomophones:           initBadHomophones(),
		badCrossSyllablePinyins: initBadCrossSyllablePinyins(),
		eraChars:                initEraChars(),
		historicalNames:         initHistoricalNames(),
		meaningNegative:         initMeaningNegative(),
		structureCategory:       initStructureCategory(),
		forbiddenEntities:       initForbiddenEntities(),
	}
}

// ============================================================
// 规则数据初始化
// ============================================================

// initBadHomophones 初始化 300+ 不良谐音词库
// 按谐音危险级别分类
func initBadHomophones() []string {
	return []string{
		// ===== 一、直接不良字（写入名即不合宜）=====
		"死", "亡", "尸", "坟", "墓", "葬", "棺", "殓", "殇",
		"病", "疾", "疫", "癌", "疯", "瘫", "痴", "呆", "傻",
		"穷", "苦", "贱", "卑", "奴", "隶", "丐",
		"难", "灾", "祸", "患", "厄", "劫", "煞",
		"哀", "悲", "愁", "苦", "痛", "恨", "怨", "怒",
		"丑", "恶", "陋", "怪", "妖", "魔", "鬼", "怪",
		"凶", "险", "危", "毒", "狠", "狡", "猾",
		"败", "溃", "破", "裂", "碎", "毁", "灭",
		"脏", "污", "秽", "浊", "腐", "朽",
		"罚", "刑", "狱", "囚", "犯", "罪", "孽",
		"辱", "耻", "羞", "愧", "惭",
		"骗", "诈", "伪", "假", "奸",
		"吵", "闹", "乱", "扰", "嚣",

		// ===== 二、谐音不雅（组合词）=====
		"史珍香", "史翔", "杜子腾", "杜琦燕", "王霸", "范统",
		"朱一", "朱逸群", "毕云涛", "沈敬兵", "胡一统",
		"刘产", "杨伟", "秦寿", "吴德", "贾效",

		// ===== 三、可能引起不良联想字=====
		"尿", "屎", "屁", "痰", "吐", "呕",
		"臭", "腥", "臊", "膻",
		"妓", "娼", "淫", "秽",
		"赌", "饮", "酗",
	}
}

// initBadCrossSyllablePinyins 初始化跨音节不良连读拼音词库
// 检测姓氏拼音+名字拼音连读时产生的不雅谐音（如"shi zhen xiang"连读类似"屎真香"）
func initBadCrossSyllablePinyins() []badCrossSyllable {
	return []badCrossSyllable{
		// ===== 一级：严重不良 — 一票否决 =====
		{pattern: "wangba",    severity: "hard", desc: "连读类似'王八'，极不雅"},
		{pattern: "shabi",     severity: "hard", desc: "连读类似'傻逼'，粗俗"},
		{pattern: "caonima",   severity: "hard", desc: "连读类似脏话，粗俗"},
		{pattern: "tamade",    severity: "hard", desc: "连读类似'他妈的'，粗俗"},
		{pattern: "shenjingbing", severity: "hard", desc: "连读类似'神经病'"},

		// ===== 二级：不良谐音 — 一票否决 =====
		{pattern: "yangwei",   severity: "hard", desc: "连读谐音'阳痿'"},
		{pattern: "liuchan",   severity: "hard", desc: "连读谐音'流产'"},

		// ===== 三级：提醒级 — 软提醒 =====
		{pattern: "duziteng",  severity: "soft", desc: "连读可能谐音'肚子疼'"},
		{pattern: "duqiyan",   severity: "soft", desc: "连读可能谐音'肚脐眼'"},
	}
}

// stripPinyinToneNumber 去掉拼音音节末尾的声调数字（如 shi1 → shi）
func stripPinyinToneNumber(p string) string {
	return strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return -1
		}
		return r
	}, p)
}

// initEraChars 时代特征过强字
func initEraChars() []string {
	return []string{
		// 文革时期
		"红", "卫", "兵", "斗", "批", "改",
		"革", "命", "武", "造", "反",
		// 大跃进时代
		"超", "英", "赶", "美", "钢", "铁",
		"跃", "进", "胜", "天",
		// 过于陈旧
		"婢", "妾", "妃", "嫔", "奴",
	}
}

// initHistoricalNames 避免与历史知名人物同名的完整名单
func initHistoricalNames() []string {
	return []string{
		// 帝王
		"李世民", "赵匡胤", "朱元璋", "嬴政", "刘彻", "杨坚",
		"李隆基", "康熙", "雍正", "乾隆", "赵祯",

		// 历史名人
		"孔子", "孟子", "老子", "庄子", "屈原", "诸葛亮",
		"关羽", "张飞", "赵云", "曹操", "孙权", "刘备",
		"李白", "杜甫", "白居易", "苏轼", "岳飞", "文天祥",
		"林则徐", "孙中山", "鲁迅",
	}
}

// initForbiddenEntities 初始化禁止的国家地区机关单位名称
// 严禁生成此类名称作为人名（AGENTS.md 安全策略）
func initForbiddenEntities() map[string]bool {
	return map[string]bool{
		// 国家名称
		"中国": true, "美国": true, "日本": true, "韩国": true,
		"朝鲜": true, "英国": true, "法国": true, "德国": true,
		"俄国": true, "印度": true, "巴西": true, "加拿大": true,
		"澳洲": true, "泰国": true, "越南": true, "缅甸": true,
		"菲律宾": true, "马来西亚": true, "新加坡": true,

		// 国家机关
		"国务院": true, "全国人大": true, "中央军委": true,
		"最高法院": true, "最高检察院": true, "政治局": true,
		"中纪委": true, "党中央": true,

		// 地区机关（2-3字简称）
		"省政府": true, "市政府": true, "县政府": true,
		"区政府": true, "镇政府": true, "乡政府": true,

		// 军事/警务
		"军委": true, "国防部": true, "公安部": true,
		"安全部": true, "武警": true, "解放军": true,
	}
}

// initMeaningNegative 含义消极字
func initMeaningNegative() []string {
	return []string{
		"哀", "愁", "悲", "痛", "恨", "怨", "怒", "恼",
		"惨", "伤", "凄", "凉", "寒", "冷", "冰",
		"孤", "独", "寡", "寂", "寞", "空", "虚",
		"暗", "黑", "昏", "阴", "幽", "冥",
		"累", "倦", "疲", "乏", "困",
		"悔", "憾", "疚", "愧",
		"骗", "诈", "伪", "佞", "谄",
		"蛮", "横", "暴", "虐", "戾",
	}
}

// initStructureCategory 字形结构分类
func initStructureCategory() map[string]string {
	return map[string]string{
		// 左右结构
		"江": "左右", "河": "左右", "海": "左右", "汉": "左右", "池": "左右",
		"林": "左右", "松": "左右", "柏": "左右", "柳": "左右", "树": "左右",
		"明": "左右", "晴": "左右", "晔": "左右", "晖": "左右",
		"伟": "左右", "俊": "左右", "信": "左右", "修": "左右", "佳": "左右",
		"清": "左右", "浩": "左右", "涵": "左右", "淑": "左右",
		"琳": "左右", "琪": "左右", "瑶": "左右", "瑾": "左右",
		"婷": "左右", "婉": "左右", "媛": "左右",
		"弘": "左右", "强": "左右", "驰": "左右",
		"锦": "左右", "铭": "左右", "钧": "左右", "钰": "左右",
		"曦": "左右", "曜": "左右",
		"瀚": "左右", "澈": "左右", "澄": "左右", "潇": "左右",
		"儒": "左右", "谦": "左右",
		"妍": "左右", "姝": "左右", "妤": "左右", "娟": "左右",
		"娅": "左右", "婵": "左右", "娇": "左右", "娜": "左右",
		"娴": "左右", "靖": "左右", "颜": "左右", "韶": "左右",
		"欣": "左右", "欢": "左右", "歌": "左右", "羽": "左右",
		"翔": "左右", "翱": "左右", "辉": "左右", "耀": "左右",
		"炜": "左右", "恒": "左右", "悦": "左右", "悟": "左右",
		"惟": "左右", "怀": "左右", "恢": "左右", "恪": "左右",
		"恬": "左右", "惜": "左右", "慎": "左右", "愉": "左右",
		"博": "左右", "协": "左右", "衡": "左右", "祯": "左右",
		"禄": "左右", "禧": "左右", "祺": "左右", "禅": "左右",
		"礼": "左右", "祖": "左右", "初": "左右", "被": "左右",
		"禛": "左右", "程": "左右", "稷": "左右",
		"绮": "左右", "维": "左右", "纲": "左右", "纯": "左右",
		"绸": "左右", "绿": "左右",
		"绚": "左右", "练": "左右", "经": "左右",
		"洁": "左右", "润": "左右", "泽": "左右", "波": "左右",
		"洋": "左右", "洪": "左右", "洲": "左右", "浪": "左右",
		"涛": "左右", "源": "左右", "溪": "左右", "流": "左右",
		"泓": "左右", "泌": "左右", "泳": "左右", "治": "左右",
		"注": "左右", "沼": "左右", "沾": "左右", "沿": "左右",
		"况": "左右", "沛": "左右", "沣": "左右", "沅": "左右",
		"沐": "左右", "沥": "左右", "沦": "左右",
		"沧": "左右", "沨": "左右", "汶": "左右", "渱": "左右",

		// 上下结构
		"李": "上下", "花": "上下", "芳": "上下", "芝": "上下", "芸": "上下",
		"芬": "上下", "英": "上下", "若": "上下", "茂": "上下", "荣": "上下",
		"荷": "上下", "莉": "上下", "萌": "上下",
		"思": "上下", "志": "上下", "忠": "上下", "念": "上下",
		"恩": "上下", "慈": "上下", "慧": "上下", "惠": "上下",
		"勇": "上下", "男": "上下",
		"昊": "上下", "昶": "上下", "昱": "上下", "晟": "上下",
		"睿": "上下",
		"雪": "上下", "雷": "上下", "霞": "上下", "霜": "上下",
		"露": "上下", "灵": "上下", "霖": "上下",
		"筠": "上下", "笙": "上下", "箫": "上下",
		"燕": "上下", "然": "上下", "照": "上下",
		"紫": "上下", "悠": "上下", "望": "上下",
		"萱": "上下", "苇": "上下",
		"茗": "上下", "茜": "上下", "芙": "上下", "蓉": "上下",
		"芷": "上下", "蘅": "上下", "薇": "上下", "蕙": "上下",
		"蓓": "上下", "蕾": "上下", "蕊": "上下", "芯": "上下",
		"苹": "上下", "萍": "上下", "蔼": "上下", "蔚": "上下",
		"荻": "上下", "莆": "上下", "菡": "上下", "菁": "上下",
		"慕": "上下", "安": "上下", "宁": "上下", "定": "上下",
		"宜": "上下", "宪": "上下", "客": "上下", "实": "上下",
		"宠": "上下", "宝": "上下", "宗": "上下", "官": "上下",
		"宙": "上下", "宛": "上下", "岚": "上下", "岩": "上下",
		"岳": "上下", "岸": "上下", "峄": "上下", "峥": "上下",
		"嵘": "上下", "崇": "上下", "巍": "上下",
		"岁": "上下", "昆": "上下", "帛": "上下",
		"帝": "上下", "希": "上下", "卓": "上下", "卒": "上下",
		"卉": "上下", "华": "上下", "垂": "上下", "奇": "上下",
		"奔": "上下", "奋": "上下", "契": "上下", "奕": "上下",

		// 全包围
		"国": "全包围", "园": "全包围", "图": "全包围", "囿": "全包围",
		"圆": "全包围", "团": "全包围", "回": "全包围",

		// 半包围
		"道": "半包围", "远": "半包围", "达": "半包围", "运": "半包围",
		"通": "半包围", "逸": "半包围", "迪": "半包围",
		"同": "半包围", "周": "半包围",
		"建": "半包围", "廷": "半包围", "延": "半包围",
		"庭": "半包围", "康": "半包围",
		"超": "半包围", "越": "半包围",
		"展": "半包围", "居": "半包围",
		"起": "半包围", "趁": "半包围", "赴": "半包围",
		"旭": "半包围", "旯": "半包围", "旮": "半包围",
		"风": "半包围", "凤": "半包围", "飞": "半包围",
		"气": "半包围", "氛": "半包围",
		"兆": "半包围",
		"历": "半包围", "压": "半包围", "厚": "半包围", "原": "半包围",
		"厦": "半包围", "厨": "半包围", "厩": "半包围",
		"司": "半包围", "习": "半包围", "民": "半包围",
		"左": "半包围", "右": "半包围", "有": "半包围", "布": "半包围",
		"在": "半包围", "存": "半包围", "灰": "半包围",
		"尤": "半包围", "龙": "半包围",
		"式": "半包围", "武": "半包围", "代": "半包围",
		"岛": "半包围", "幽": "半包围", "画": "半包围",
		"函": "半包围", "凶": "半包围", "出": "半包围",

		// 独体字
		"一": "独体", "二": "独体", "三": "独体", "十": "独体",
		"人": "独体", "大": "独体", "天": "独体", "夫": "独体",
		"木": "独体", "本": "独体", "未": "独体", "末": "独体",
		"水": "独体", "火": "独体", "土": "独体", "金": "独体",
		"日": "独体", "月": "独体", "山": "独体", "石": "独体",
		"田": "独体", "目": "独体", "心": "独体",
		"文": "独体", "方": "独体", "王": "独体", "玉": "独体",
		"中": "独体", "正": "独体", "平": "独体",
		"之": "独体", "也": "独体", "子": "独体",
		"书": "独体", "为": "独体", "生": "独体",
		"白": "独体", "可": "独体", "立": "独体",
		"上": "独体", "下": "独体", "不": "独体", "与": "独体",
		"世": "独体", "东": "独体", "丝": "独体", "两": "独体",
		"丰": "独体", "临": "独体", "久": "独体", "义": "独体",
		"乐": "独体", "九": "独体", "干": "独体", "于": "独体",
		"亏": "独体", "五": "独体", "井": "独体",
		"交": "独体", "亦": "独体", "产": "独体",
		"亩": "独体", "亲": "独体",
		"今": "独体", "以": "独体", "令": "独体",
		"仲": "独体", "仰": "独体", "伊": "独体",
		"会": "独体", "传": "独体", "伦": "独体",
		"伯": "独体", "估": "独体", "伴": "独体",
		"伸": "独体", "似": "独体", "但": "独体",
		"位": "独体", "低": "独体", "何": "独体",
		"佛": "独体", "作": "独体", "你": "独体",
		"佣": "独体", "佩": "独体", "使": "独体",
		"来": "独体", "供": "独体", "依": "独体",
		"侠": "独体", "侧": "独体", "侨": "独体",
		"侯": "独体", "侵": "独体", "便": "独体",
		"促": "独体", "俗": "独体", "保": "独体",
		"俯": "独体", "俱": "独体", "倍": "独体",
		"倏": "独体", "倒": "独体", "候": "独体",
		"倚": "独体", "倦": "独体", "倩": "独体",
		"值": "独体", "倾": "独体", "假": "独体",
		"偏": "独体", "做": "独体", "停": "独体",
		"健": "独体", "偶": "独体", "偷": "独体",
		"偿": "独体", "傅": "独体", "傍": "独体",
		"杰": "独体", "备": "独体", "复": "独体", "太": "独体",
		"失": "独体", "头": "独体", "夷": "独体", "夸": "独体",
		"夹": "独体", "夺": "独体", "买": "独体",
		"寿": "独体", "奉": "独体", "奏": "独体",
		"奖": "独体", "套": "独体", "奢": "独体", "奠": "独体",
		"奥": "独体", "女": "独体", "好": "独体",
		"如": "独体", "妇": "独体", "她": "独体",
		"妈": "独体", "姑": "独体", "姐": "独体",
		"妹": "独体", "妻": "独体", "始": "独体",
		"委": "独体", "威": "独体", "娃": "独体", "娄": "独体",
		"娘": "独体", "娱": "独体", "婆": "独体",
		"孔": "独体", "字": "独体",
		"孝": "独体", "孟": "独体", "季": "独体", "孤": "独体",
		"学": "独体", "孙": "独体", "孜": "独体",
		"寸": "独体", "小": "独体", "少": "独体",
		"尔": "独体", "尖": "独体", "尚": "独体",
		"尧": "独体",
		"尺": "独体", "尽": "独体", "尾": "独体", "局": "独体",
		"层": "独体", "屈": "独体", "届": "独体", "屋": "独体",
		"属": "独体", "屯": "独体",
		"岂": "独体", "川": "独体", "州": "独体", "工": "独体",
		"巧": "独体", "巩": "独体", "巫": "独体", "差": "独体",
		"己": "独体", "已": "独体", "巳": "独体", "巴": "独体",
		"巷": "独体", "巾": "独体", "市": "独体",
		"帅": "独体", "师": "独体",
		"席": "独体", "常": "独体", "幕": "独体",
		"年": "独体", "并": "独体", "幸": "独体", "幼": "独体",
		"广": "独体", "庄": "独体", "庆": "独体",
		"序": "独体", "店": "独体", "府": "独体",
		"底": "独体", "度": "独体", "座": "独体",
		"庸": "独体", "廉": "独体",
		"弓": "独体", "引": "独体",
		"张": "独体", "弥": "独体", "弯": "独体", "弱": "独体",
		"弹": "独体",
		"门": "独体", "闪": "独体", "闭": "独体",
		"问": "独体", "闯": "独体", "闲": "独体", "间": "独体",
		"闵": "独体", "闻": "独体", "阁": "独体", "阅": "独体",
		"队": "独体", "阳": "独体", "阴": "独体",
		"阵": "独体", "阶": "独体", "阻": "独体",
		"阿": "独体", "附": "独体", "际": "独体",
		"陆": "独体", "陇": "独体", "限": "独体", "陕": "独体",
		"云": "独体", "电": "独体",
		"雾": "独体", "青": "独体", "静": "独体", "非": "独体",
		"面": "独体", "革": "独体", "韦": "独体",
		"音": "独体", "页": "独体", "顺": "独体",
		"项": "独体", "须": "独体", "预": "独体",
		"顿": "独体", "领": "独体",
		"食": "独体", "首": "独体", "香": "独体", "马": "独体",
		"骨": "独体", "高": "独体", "鬼": "独体",
		"鱼": "独体", "鸟": "独体", "鹿": "独体",
		"麦": "独体", "麻": "独体", "黄": "独体",
		"黑": "独体", "黍": "独体", "鼓": "独体",
		"鼠": "独体", "鼻": "独体", "齐": "独体",
		"齿": "独体", "龟": "独体",

		// 品字形
		"晶": "品字", "鑫": "品字", "森": "品字", "众": "品字",
		"品": "品字",
	}
}

// ============================================================
// 单项过滤方法
// ============================================================

// 1. 生僻字检查（集成 IsRareChar + hanzi.IsCommonChar）
func (nf *NameFilter) checkRareChars(fullName string) []string {
	var issues []string
	for _, r := range fullName {
		char := string(r)
		if IsRareChar(char) {
			issues = append(issues, fmt.Sprintf("'%s'为扩展区生僻字，可能影响日常使用", char))
		} else if !hanzi.IsCommonChar(char) {
			issues = append(issues, fmt.Sprintf("'%s'为生僻字，笔画过多或非常用字", char))
		}
	}
	return issues
}

// 2. 谐音检查（直接匹配 + 拼音模糊匹配）
func (nf *NameFilter) checkHomophone(name Name) []string {
	var issues []string

	// 2a. 直接字符匹配
	fullName := name.FullName
	for _, bad := range nf.badHomophones {
		if strings.Contains(fullName, bad) {
			// 按字符数判断短谐音词（len 按字节对中文恒 >2，须用 RuneCount）
			if utf8.RuneCountInString(bad) <= 2 && utf8.RuneCountInString(fullName) > 2 {
				// 单字或双字的谐音词在名字中单独出现需要更谨慎判断
				issues = append(issues, fmt.Sprintf("名字中包含不宜用字'%s'", bad))
			} else {
				issues = append(issues, fmt.Sprintf("'%s'谐音不雅或含有不良字", bad))
			}
			return issues // 发现一个就返回
		}
	}

	// 2b. 拼音级谐音检查（三字及以上全名拼音）
	pinyinParts := strings.Fields(strings.ToLower(name.Pinyin))
	if len(pinyinParts) < 2 {
		return issues
	}

	// 常见不良拼音组合
	badPinyinCombos := []struct {
		pattern  string
		suggestion string
	}{
		{"si", "可能与'死'谐音"},
		{"wang", "可能与'亡'谐音"},
	}

	fullPinyin := strings.Join(pinyinParts, "")
	for _, combo := range badPinyinCombos {
		if strings.Contains(fullPinyin, combo.pattern) {
			issues = append(issues, fmt.Sprintf("拼音'%s'%s", combo.pattern, combo.suggestion))
		}
	}

	return issues
}

// 2c. 跨音节连读不良谐音检查
// 将全名拼音音节去声调后连成一串，检测是否包含已知不良模式
// 例如："shi zhen xiang" → "shizhenxiang" → 可读为"屎真香"
func (nf *NameFilter) checkCrossSyllableBadReading(name Name) []string {
	var issues []string

	pinyinParts := strings.Fields(strings.ToLower(name.Pinyin))
	if len(pinyinParts) < 2 {
		return issues
	}

	// 去声调 + 连接成串
	var sb strings.Builder
	for _, p := range pinyinParts {
		sb.WriteString(stripPinyinToneNumber(p))
	}
	fullPinyin := sb.String()

	if fullPinyin == "" {
		return issues
	}

	for _, bc := range nf.badCrossSyllablePinyins {
		if strings.Contains(fullPinyin, bc.pattern) {
			issues = append(issues, fmt.Sprintf("拼音连读'%s'%s", fullPinyin, bc.desc))
		}
	}

	return issues
}

// 3. 时代特征检查
func (nf *NameFilter) checkEra(name Name) []string {
	var issues []string
	surname := name.Surname
	givenName := name.GivenName

	// 名字（不含姓）中出现时代特征字才警告
	for _, c := range givenName {
		char := string(c)
		for _, era := range nf.eraChars {
			if char == era {
				issues = append(issues, fmt.Sprintf("'%s'具有较强时代特征，请确认是否符合期望", char))
			}
		}
	}
	_ = surname
	return issues
}

// 4. 历史名人冲突检查
// 检测全名匹配（如"李世民"）以及名字部分与历史名人匹配（如名"世民"）
func (nf *NameFilter) checkHistoricalCollision(fullName string) []string {
	var issues []string
	for _, hn := range nf.historicalNames {
		if fullName == hn {
			issues = append(issues, fmt.Sprintf("与历史名人%s同名，建议避免", hn))
			return issues
		}
		// 对2字以上历史名人名，检查名字部分是否冲突
		hnRunes := []rune(hn)
		if len(hnRunes) > 2 {
			// 历史名人去掉姓氏后的名字部分（如"李世民"→"世民"）
			hnGiven := string(hnRunes[1:])
			if len(hnGiven) > 1 && len(hnGiven) <= len([]rune(fullName)) {
				fnRunes := []rune(fullName)
				fnGiven := string(fnRunes[1:])
				if fnGiven == hnGiven {
					issues = append(issues, fmt.Sprintf("名字与历史名人%s的名字'%s'相同，建议避免", hn, hnGiven))
					return issues
				}
			}
		}
	}
	return issues
}

// 4b. 国家机关单位名称禁止检查
// 检测全名或名字部分是否匹配禁止的国家地区机关单位名称
func (nf *NameFilter) checkForbiddenEntity(fullName string) []string {
	var issues []string
	// 全名匹配
	if nf.forbiddenEntities[fullName] {
		issues = append(issues, fmt.Sprintf("名字'%s'为国家机关单位名称，禁止使用", fullName))
		return issues
	}
	// 名字部分匹配（去掉姓氏后的部分）
	fnRunes := []rune(fullName)
	if len(fnRunes) > 1 {
		givenPart := string(fnRunes[1:])
		if nf.forbiddenEntities[givenPart] {
			issues = append(issues, fmt.Sprintf("名字'%s'为国家机关单位名称，禁止使用", givenPart))
			return issues
		}
	}
	// 全名包含禁止实体（如"李国务院"）
	for entity := range nf.forbiddenEntities {
		if strings.Contains(fullName, entity) && fullName != entity {
			issues = append(issues, fmt.Sprintf("名字包含国家机关单位名称'%s'，禁止使用", entity))
			return issues
		}
	}
	return issues
}

// 5. 笔画平衡检查
func (nf *NameFilter) checkStrokeBalance(name Name) []string {
	var issues []string

	// 获取各字笔画数
	givenRunes := []rune(name.GivenName)
	if len(givenRunes) < 2 {
		return issues
	}

	// 拆分名字拼音获取各字笔画
	strokes := make([]int, len(givenRunes))

	for i, r := range givenRunes {
		char := string(r)
		if h, ok := hanzi.HanziData[char]; ok {
			strokes[i] = h.Strokes
		}
	}

	if len(strokes) >= 2 {
		maxStrokes := 0
		minStrokes := 100
		for _, s := range strokes {
			if s > maxStrokes {
				maxStrokes = s
			}
			if s < minStrokes {
				minStrokes = s
			}
		}

		if maxStrokes-minStrokes > 15 {
			issues = append(issues, fmt.Sprintf("名字各字笔画差异过大（%d-%d画），建议控制在15画以内", minStrokes, maxStrokes))
		}

		// 笔画过多提醒
		total := 0
		for _, s := range strokes {
			total += s
		}
		if total > 40 {
			issues = append(issues, fmt.Sprintf("名字总笔画数偏多（%d画），可能不便书写", total))
		}
	}

	return issues
}

// 6. 含义消极检查
func (nf *NameFilter) checkNegativeMeaning(name Name) []string {
	var issues []string
	givenName := name.GivenName

	for _, c := range givenName {
		char := string(c)
		for _, neg := range nf.meaningNegative {
			if char == neg {
				issues = append(issues, fmt.Sprintf("'%s'含义偏消极，建议更换", char))
			}
		}
	}
	return issues
}

// 7. 字形结构平衡检查
func (nf *NameFilter) checkStructureBalance(surname, givenName string) []string {
	var issues []string

	// 获取各字结构
	allChars := surname + givenName
	structures := make([]string, 0, len(allChars))
	hasStructure := false

	for _, r := range allChars {
		char := string(r)
		if s, ok := nf.structureCategory[char]; ok {
			structures = append(structures, s)
			hasStructure = true
		} else {
			structures = append(structures, "未知")
		}
	}

	if !hasStructure || len(structures) < 2 {
		return issues
	}

	// 检查是否三连相同结构
	for i := 0; i < len(structures)-2; i++ {
		if structures[i] == structures[i+1] && structures[i+1] == structures[i+2] && structures[i] != "未知" {
			issues = append(issues, fmt.Sprintf("姓氏与名字三字均属'%s'结构，视觉上略显单调", structures[i]))
			return issues
		}
	}

	// 检查是否两连相同结构（名字内部）
	if len(structures) >= 3 {
		for i := 1; i < len(structures)-1; i++ {
			if structures[i] == structures[i+1] && structures[i] != "未知" {
				issues = append(issues, fmt.Sprintf("名字中两字均为'%s'结构", structures[i]))
				break
			}
		}
	}

	return issues
}

// 8. 重复字检查
func (nf *NameFilter) checkDuplicateChar(givenName string) []string {
	charCount := make(map[rune]int)
	for _, r := range givenName {
		charCount[r]++
		if charCount[r] > 1 {
			return []string{fmt.Sprintf("名字中'%c'字重复", r)}
		}
	}
	return nil
}

// 9. 含义积极度检查
// 名字中不能全是含义中性的字，至少有一个有明确美好寓意的字
func (nf *NameFilter) checkPositiveMeaning(name Name) []string {
	positiveChars := []string{
		"美", "好", "善", "德", "智", "勇", "仁", "义", "礼", "信",
		"福", "寿", "康", "宁", "安", "乐", "喜", "瑞", "祥",
		"文", "武", "英", "杰", "俊", "贤", "良", "才", "雅",
		"明", "光", "辉", "耀", "华", "荣", "昌", "盛",
		"清", "洁", "正", "直", "刚", "毅", "坚", "强",
		"和", "平", "顺", "通", "达", "远",
		"芳", "香", "芬", "菲", "兰", "芝", "玉",
	}

	hasPositive := false
	for _, c := range name.GivenName {
		char := string(c)
		for _, p := range positiveChars {
			if char == p {
				hasPositive = true
				break
			}
		}
		if hasPositive {
			break
		}
	}

	if !hasPositive {
		return []string{"名字中所有字均为中性含义，建议至少包含一个有美好寓意的字"}
	}
	return nil
}

// ============================================================
// 综合过滤
// ============================================================

// Filter 完整过滤流程
func (nf *NameFilter) Filter(name Name) *NameFilterResult {
	result := &NameFilterResult{
		Valid:  true,
		Issues: []string{},
	}

	// 预计算一次谐音与跨音节连读结果，供 hardFilter/softCheck 共用，避免重复解析拼音
	homophoneIssues := nf.checkHomophone(name)
	crossSyllableIssues := nf.checkCrossSyllableBadReading(name)

	// 阶段1：一票否决 — 硬过滤，不满足直接淘汰
	hardIssues := nf.hardFilter(name, homophoneIssues, crossSyllableIssues)
	if len(hardIssues) > 0 {
		result.Valid = false
		result.Issues = hardIssues
		return result
	}

	// 阶段2：软检查 — 不直接淘汰，但记录建议
	result.Issues = nf.softCheck(name, homophoneIssues, crossSyllableIssues)

	if len(result.Issues) > 0 {
		result.Valid = true // 软检查不影响有效性，但给出建议
	}

	return result
}

// hardFilter 硬过滤 — 一票否决
// homophoneIssues/crossSyllableIssues 为 Filter 预计算的结果，避免重复解析拼音
func (nf *NameFilter) hardFilter(name Name, homophoneIssues, crossSyllableIssues []string) []string {
	var issues []string

	// 1. 生僻字 → 硬过滤
	rareIssues := nf.checkRareChars(name.FullName)
	if len(rareIssues) > 0 {
		issues = append(issues, rareIssues...)
		return issues // 立即返回
	}

	// 2. 不良谐音 → 硬过滤（复用预计算结果）
	for _, issue := range homophoneIssues {
		if strings.Contains(issue, "不宜用字") || strings.Contains(issue, "谐音不雅") {
			issues = append(issues, issue)
			return issues // 立即返回
		}
	}

	// 3. 历史名人冲突 → 硬过滤
	historicalIssues := nf.checkHistoricalCollision(name.FullName)
	if len(historicalIssues) > 0 {
		issues = append(issues, historicalIssues...)
		return issues
	}

	// 3b. 国家机关单位名称 → 硬过滤（AGENTS.md 安全策略）
	entityIssues := nf.checkForbiddenEntity(name.FullName)
	if len(entityIssues) > 0 {
		issues = append(issues, entityIssues...)
		return issues
	}

	// 4. 重复字 → 硬过滤
	dupIssues := nf.checkDuplicateChar(name.GivenName)
	if len(dupIssues) > 0 {
		issues = append(issues, dupIssues...)
		return issues
	}

	// 5. 含义消极 → 硬过滤
	negIssues := nf.checkNegativeMeaning(name)
	if len(negIssues) > 0 {
		issues = append(issues, negIssues...)
		return issues
	}

	// 6. 跨音节不良连读拼音 → 硬过滤（复用预计算结果，只处理 severity="hard" 的项）
	for _, ci := range crossSyllableIssues {
		for _, bc := range nf.badCrossSyllablePinyins {
			if bc.severity == "hard" && strings.Contains(ci, bc.desc) {
				issues = append(issues, ci)
				return issues
			}
		}
	}

	return issues
}

// softCheck 软检查 — 仅给出建议
// homophoneIssues/crossSyllableIssues 为 Filter 预计算的结果，避免重复解析拼音
func (nf *NameFilter) softCheck(name Name, homophoneIssues, crossSyllableIssues []string) []string {
	var issues []string

	// 1. 时代特征（软提醒）
	eraIssues := nf.checkEra(name)
	issues = append(issues, eraIssues...)

	// 2. 笔画平衡（软提醒）
	strokeIssues := nf.checkStrokeBalance(name)
	issues = append(issues, strokeIssues...)

	// 3. 字形平衡（软提醒）
	structIssues := nf.checkStructureBalance(name.Surname, name.GivenName)
	issues = append(issues, structIssues...)

	// 4. 含义积极（软提醒）
	positiveIssues := nf.checkPositiveMeaning(name)
	issues = append(issues, positiveIssues...)

	// 5. 谐音软提醒（复用预计算结果）
	for _, issue := range homophoneIssues {
		if !strings.Contains(issue, "不宜用字") && !strings.Contains(issue, "谐音不雅") {
			issues = append(issues, issue)
		}
	}

	// 6. 跨音节连读不良拼音 → 软提醒（复用预计算结果，只处理 severity="soft" 的项）
	for _, ci := range crossSyllableIssues {
		for _, bc := range nf.badCrossSyllablePinyins {
			if bc.severity == "soft" && strings.Contains(ci, bc.desc) {
				issues = append(issues, ci)
				break
			}
		}
	}

	return issues
}