package fate

import "strings"

// BadHomophoneInfo 不吉谐音信息
type BadHomophoneInfo struct {
	// Pinyin 拼音（无声调）
	Pinyin string
	// Word 对应的不吉汉字
	Word string
	// Description 说明
	Description string
}

// BadHomophones 不吉谐音列表
//
// 命中规则（严格）：
//  1. pinyin 与本字拼音匹配（精确无声调比对）
//  2. 本字 == Word（同名）时不扣分——拼音相同但本字不同时不视为不吉谐音
//
// 举例：拼音 si 的"死"命中；拼音 si 的"思/丝/斯/四/寺"不命中。
// 之前版本（仅按拼音匹配）会把所有 si 拼音的好字（思/丝/斯）误扣"含不吉谐音:死"。
//
// 按拼音首字母分组，检测名字拼音时逐音节匹配
// 扩展覆盖：常见贬义字、疾病、死亡、灾难、欺诈、偷盗等
var BadHomophones = []BadHomophoneInfo{
	// A
	{Pinyin: "ai", Word: "哀", Description: "哀伤悲痛"},
	{Pinyin: "an", Word: "暗", Description: "黑暗无光"},
	// B
	{Pinyin: "bai", Word: "败", Description: "败落"},
	{Pinyin: "ban", Word: "办", Description: "办事（不顺）"},
	{Pinyin: "bao", Word: "暴", Description: "暴躁"},
	{Pinyin: "bei", Word: "背", Description: "背运"},
	{Pinyin: "bi", Word: "毙", Description: "枪毙"},
	{Pinyin: "bie", Word: "瘪", Description: "瘪三"},
	{Pinyin: "bing", Word: "病", Description: "疾病"},
	{Pinyin: "bo", Word: "薄", Description: "薄命"},
	{Pinyin: "bu", Word: "不", Description: "否定"},
	// C
	{Pinyin: "can", Word: "残", Description: "残废"},
	{Pinyin: "cao", Word: "操", Description: "粗俗不雅"},
	{Pinyin: "ce", Word: "厕", Description: "厕所"},
	{Pinyin: "cen", Word: "涔", Description: "涔汗（病态）"},
	{Pinyin: "cha", Word: "差", Description: "差错"},
	{Pinyin: "chan", Word: "忏", Description: "忏悔（消极）"},
	{Pinyin: "chen", Word: "沉", Description: "沉沦"},
	{Pinyin: "chi", Word: "耻", Description: "耻辱"},
	{Pinyin: "chou", Word: "丑", Description: "丑陋"},
	{Pinyin: "chu", Word: "处", Description: "处罚"},
	{Pinyin: "chuan", Word: "串", Description: "串联（贬）"},
	{Pinyin: "ci", Word: "刺", Description: "刺伤"},
	{Pinyin: "cu", Word: "粗", Description: "粗俗"},
	{Pinyin: "cuo", Word: "挫", Description: "挫折"},
	// D
	{Pinyin: "da", Word: "打", Description: "打击"},
	{Pinyin: "dai", Word: "呆", Description: "呆滞"},
	{Pinyin: "dan", Word: "蛋", Description: "混蛋"},
	{Pinyin: "dao", Word: "倒", Description: "倒台"},
	{Pinyin: "de", Word: "的", Description: "轻浮"},
	{Pinyin: "di", Word: "敌", Description: "敌人"},
	{Pinyin: "die", Word: "跌", Description: "跌倒"},
	{Pinyin: "ding", Word: "盯", Description: "盯梢"},
	{Pinyin: "dong", Word: "冻", Description: "冻伤"},
	{Pinyin: "dou", Word: "斗", Description: "斗殴"},
	{Pinyin: "du", Word: "毒", Description: "毒害"},
	{Pinyin: "duo", Word: "堕", Description: "堕落"},
	// E
	{Pinyin: "e", Word: "恶", Description: "凶恶"},
	{Pinyin: "en", Word: "摁", Description: "摁住"},
	// F
	{Pinyin: "fa", Word: "罚", Description: "处罚"},
	{Pinyin: "fan", Word: "犯", Description: "犯法"},
	{Pinyin: "fei", Word: "废", Description: "废弃"},
	{Pinyin: "fen", Word: "愤", Description: "愤怒"},
	{Pinyin: "feng", Word: "疯", Description: "疯狂"},
	{Pinyin: "fu", Word: "腐", Description: "腐败"},
	{Pinyin: "fu", Word: "妇", Description: "妇女（贬义）"},
	// G
	{Pinyin: "ga", Word: "尬", Description: "尴尬"},
	{Pinyin: "gan", Word: "干", Description: "干涸"},
	{Pinyin: "gao", Word: "搞", Description: "搞砸"},
	{Pinyin: "ge", Word: "割", Description: "割裂"},
	{Pinyin: "gu", Word: "孤", Description: "孤独"},
	{Pinyin: "gua", Word: "寡", Description: "孤寡"},
	{Pinyin: "guai", Word: "怪", Description: "怪癖"},
	{Pinyin: "guan", Word: "管", Description: "管制"},
	{Pinyin: "gui", Word: "鬼", Description: "鬼怪"},
	{Pinyin: "gun", Word: "滚", Description: "滚蛋"},
	// H
	{Pinyin: "ha", Word: "蛤", Description: "蛤蟆"},
	{Pinyin: "hai", Word: "害", Description: "伤害"},
	{Pinyin: "han", Word: "憾", Description: "遗憾"},
	{Pinyin: "hao", Word: "耗", Description: "消耗"},
	{Pinyin: "he", Word: "吓", Description: "恐吓"},
	{Pinyin: "hen", Word: "恨", Description: "仇恨"},
	{Pinyin: "hou", Word: "猴", Description: "猴急"},
	{Pinyin: "hu", Word: "虎", Description: "虎头蛇尾"},
	{Pinyin: "hua", Word: "滑", Description: "滑头"},
	{Pinyin: "huang", Word: "慌", Description: "慌张"},
	{Pinyin: "hui", Word: "毁", Description: "毁坏"},
	{Pinyin: "hun", Word: "昏", Description: "昏暗"},
	{Pinyin: "huo", Word: "祸", Description: "灾祸"},
	// J
	{Pinyin: "ji", Word: "疾", Description: "疾病"},
	{Pinyin: "jia", Word: "假", Description: "虚假"},
	{Pinyin: "jian", Word: "贱", Description: "低贱"},
	{Pinyin: "jiao", Word: "焦", Description: "焦躁"},
	{Pinyin: "jie", Word: "劫", Description: "劫难"},
	{Pinyin: "jin", Word: "禁", Description: "禁止"},
	{Pinyin: "jiu", Word: "救", Description: "救命"},
	{Pinyin: "ju", Word: "拒", Description: "拒绝"},
	{Pinyin: "jue", Word: "绝", Description: "绝路"},
	// K
	{Pinyin: "ka", Word: "卡", Description: "卡住"},
	{Pinyin: "kan", Word: "砍", Description: "砍伤"},
	{Pinyin: "kang", Word: "扛", Description: "扛不住"},
	{Pinyin: "ke", Word: "咳", Description: "咳嗽"},
	{Pinyin: "ken", Word: "啃", Description: "啃老"},
	{Pinyin: "kong", Word: "空", Description: "空虚"},
	{Pinyin: "kou", Word: "扣", Description: "扣押"},
	{Pinyin: "ku", Word: "哭", Description: "哭泣"},
	{Pinyin: "kua", Word: "垮", Description: "垮台"},
	{Pinyin: "kui", Word: "亏", Description: "亏损"},
	// L
	{Pinyin: "la", Word: "辣", Description: "泼辣"},
	{Pinyin: "lai", Word: "赖", Description: "无赖"},
	{Pinyin: "lan", Word: "烂", Description: "腐烂"},
	{Pinyin: "lao", Word: "牢", Description: "牢狱"},
	{Pinyin: "le", Word: "了", Description: "了断"},
	{Pinyin: "lei", Word: "累", Description: "劳累"},
	{Pinyin: "leng", Word: "冷", Description: "冷酷"},
	{Pinyin: "li", Word: "离", Description: "分离"},
	{Pinyin: "lian", Word: "怜", Description: "可怜"},
	{Pinyin: "lie", Word: "烈", Description: "惨烈"},
	{Pinyin: "lin", Word: "吝", Description: "吝啬"},
	{Pinyin: "liu", Word: "溜", Description: "溜走"},
	{Pinyin: "long", Word: "笼", Description: "笼中"},
	{Pinyin: "lu", Word: "碌", Description: "碌碌无为"},
	{Pinyin: "luan", Word: "乱", Description: "混乱"},
	// M
	{Pinyin: "ma", Word: "麻", Description: "麻烦"},
	{Pinyin: "mai", Word: "埋", Description: "埋没"},
	{Pinyin: "man", Word: "瞒", Description: "隐瞒"},
	{Pinyin: "mang", Word: "盲", Description: "盲目"},
	{Pinyin: "mao", Word: "毛", Description: "毛躁"},
	{Pinyin: "mei", Word: "霉", Description: "倒霉"},
	{Pinyin: "men", Word: "闷", Description: "闷闷不乐"},
	{Pinyin: "mi", Word: "迷", Description: "迷惘"},
	{Pinyin: "mian", Word: "免", Description: "免除"},
	{Pinyin: "miao", Word: "渺", Description: "渺小"},
	{Pinyin: "mie", Word: "灭", Description: "毁灭"},
	{Pinyin: "mo", Word: "没", Description: "没落"},
	{Pinyin: "mou", Word: "某", Description: "某人"},
	// N
	{Pinyin: "na", Word: "拿", Description: "拿捏"},
	{Pinyin: "nai", Word: "奈", Description: "无奈"},
	{Pinyin: "nan", Word: "难", Description: "困难"},
	{Pinyin: "nao", Word: "恼", Description: "恼怒"},
	{Pinyin: "nei", Word: "馁", Description: "气馁"},
	{Pinyin: "neng", Word: "能", Description: "能耐"},
	{Pinyin: "ni", Word: "逆", Description: "逆境"},
	{Pinyin: "nian", Word: "撵", Description: "撵走"},
	{Pinyin: "niao", Word: "尿", Description: "粗俗不雅"},
	{Pinyin: "nie", Word: "孽", Description: "孽缘"},
	{Pinyin: "ning", Word: "狞", Description: "狰狞"},
	{Pinyin: "nong", Word: "弄", Description: "弄巧成拙"},
	{Pinyin: "nu", Word: "奴", Description: "奴役"},
	{Pinyin: "nue", Word: "虐", Description: "虐待"},
	{Pinyin: "nuo", Word: "懦", Description: "懦弱"},
	// O
	{Pinyin: "ou", Word: "呕", Description: "呕吐"},
	// P
	{Pinyin: "pa", Word: "怕", Description: "害怕"},
	{Pinyin: "pai", Word: "排", Description: "排斥"},
	{Pinyin: "pan", Word: "叛", Description: "叛徒"},
	{Pinyin: "pao", Word: "抛", Description: "抛弃"},
	{Pinyin: "pei", Word: "赔", Description: "赔钱"},
	{Pinyin: "pen", Word: "喷", Description: "喷人"},
	{Pinyin: "peng", Word: "碰", Description: "碰壁"},
	{Pinyin: "pi", Word: "屁", Description: "粗俗不雅"},
	{Pinyin: "pian", Word: "骗", Description: "欺骗"},
	{Pinyin: "piao", Word: "漂", Description: "漂泊"},
	{Pinyin: "pie", Word: "撇", Description: "撇开"},
	{Pinyin: "pin", Word: "拼", Description: "拼命"},
	{Pinyin: "po", Word: "破", Description: "破败"},
	{Pinyin: "pu", Word: "扑", Description: "扑空"},
	// Q
	{Pinyin: "qi", Word: "弃", Description: "抛弃"},
	{Pinyin: "qian", Word: "欠", Description: "欠债"},
	{Pinyin: "qiang", Word: "枪", Description: "枪伤"},
	{Pinyin: "qiao", Word: "撬", Description: "撬开"},
	{Pinyin: "qie", Word: "切", Description: "切割"},
	{Pinyin: "qin", Word: "擒", Description: "擒拿"},
	{Pinyin: "qiu", Word: "囚", Description: "囚禁"},
	{Pinyin: "qu", Word: "屈", Description: "委屈"},
	{Pinyin: "quan", Word: "劝", Description: "劝退"},
	// R
	{Pinyin: "rang", Word: "让", Description: "让步"},
	{Pinyin: "rao", Word: "绕", Description: "绕弯"},
	{Pinyin: "re", Word: "热", Description: "热血"},
	{Pinyin: "ren", Word: "认", Description: "认命"},
	{Pinyin: "rong", Word: "融", Description: "融解"},
	{Pinyin: "rou", Word: "揉", Description: "揉捏"},
	{Pinyin: "ru", Word: "辱", Description: "侮辱"},
	{Pinyin: "ruan", Word: "软", Description: "软弱"},
	{Pinyin: "rui", Word: "锐", Description: "锐减"},
	// S
	{Pinyin: "sa", Word: "撒", Description: "撒谎"},
	{Pinyin: "sai", Word: "塞", Description: "堵塞"},
	{Pinyin: "san", Word: "散", Description: "散失"},
	{Pinyin: "sang", Word: "丧", Description: "丧事"},
	{Pinyin: "sao", Word: "扫", Description: "扫兴"},
	{Pinyin: "se", Word: "涩", Description: "苦涩"},
	{Pinyin: "sha", Word: "杀", Description: "杀生"},
	{Pinyin: "shai", Word: "晒", Description: "晒干"},
	{Pinyin: "shan", Word: "删", Description: "删除"},
	{Pinyin: "shang", Word: "伤", Description: "伤害"},
	{Pinyin: "shao", Word: "烧", Description: "烧毁"},
	{Pinyin: "she", Word: "折", Description: "折本"},
	{Pinyin: "shen", Word: "甚", Description: "甚嚣"},
	{Pinyin: "sheng", Word: "剩", Description: "剩余"},
	{Pinyin: "shi", Word: "屎", Description: "污秽"},
	{Pinyin: "shou", Word: "瘦", Description: "瘦弱"},
	{Pinyin: "shu", Word: "输", Description: "输赢（输）"},
	{Pinyin: "shuai", Word: "衰", Description: "衰败"},
	{Pinyin: "shuan", Word: "栓", Description: "栓塞"},
	{Pinyin: "shui", Word: "睡", Description: "睡觉"},
	{Pinyin: "shun", Word: "顺", Description: "顺从"},
	{Pinyin: "shuo", Word: "说", Description: "说破"},
	{Pinyin: "si", Word: "死", Description: "死亡"},
	{Pinyin: "song", Word: "送", Description: "送命"},
	{Pinyin: "sou", Word: "搜", Description: "搜查"},
	{Pinyin: "su", Word: "素", Description: "素淡"},
	{Pinyin: "suan", Word: "算", Description: "算计"},
	{Pinyin: "sui", Word: "碎", Description: "破碎"},
	{Pinyin: "sun", Word: "损", Description: "损害"},
	{Pinyin: "suo", Word: "缩", Description: "萎缩"},
	// T
	{Pinyin: "ta", Word: "踏", Description: "踏空"},
	{Pinyin: "tai", Word: "太", Description: "太过"},
	{Pinyin: "tan", Word: "贪", Description: "贪婪"},
	{Pinyin: "tang", Word: "烫", Description: "烫伤"},
	{Pinyin: "tao", Word: "逃", Description: "逃跑"},
	{Pinyin: "te", Word: "特", Description: "特别"},
	{Pinyin: "teng", Word: "疼", Description: "疼痛"},
	{Pinyin: "ti", Word: "剔", Description: "剔除"},
	{Pinyin: "tian", Word: "添", Description: "添堵"},
	{Pinyin: "tiao", Word: "挑", Description: "挑剔"},
	{Pinyin: "tie", Word: "贴", Description: "贴补"},
	{Pinyin: "ting", Word: "停", Description: "停业"},
	{Pinyin: "tong", Word: "痛", Description: "痛苦"},
	{Pinyin: "tou", Word: "偷", Description: "偷盗"},
	{Pinyin: "tu", Word: "秃", Description: "秃废"},
	{Pinyin: "tui", Word: "退", Description: "退步"},
	{Pinyin: "tun", Word: "吞", Description: "吞没"},
	// W
	{Pinyin: "wa", Word: "挖", Description: "挖苦"},
	{Pinyin: "wai", Word: "歪", Description: "歪曲"},
	{Pinyin: "wan", Word: "玩", Description: "玩物丧志"},
	{Pinyin: "wang", Word: "亡", Description: "死亡"},
	{Pinyin: "wei", Word: "危", Description: "危险"},
	{Pinyin: "wen", Word: "紊", Description: "紊乱"},
	{Pinyin: "wo", Word: "卧", Description: "卧病"},
	{Pinyin: "wu", Word: "污", Description: "污秽"},
	// X
	{Pinyin: "xi", Word: "夕", Description: "夕阳"},
	{Pinyin: "xia", Word: "瞎", Description: "瞎眼"},
	{Pinyin: "xian", Word: "险", Description: "危险"},
	{Pinyin: "xiang", Word: "向", Description: "向隅"},
	{Pinyin: "xiao", Word: "消", Description: "消退"},
	{Pinyin: "xie", Word: "谢", Description: "凋谢"},
	{Pinyin: "xin", Word: "辛", Description: "辛苦"},
	{Pinyin: "xing", Word: "刑", Description: "刑罚"},
	{Pinyin: "xiong", Word: "凶", Description: "凶险"},
	{Pinyin: "xiu", Word: "朽", Description: "朽木"},
	{Pinyin: "xu", Word: "虚", Description: "虚伪"},
	{Pinyin: "xuan", Word: "眩", Description: "眩晕"},
	{Pinyin: "xue", Word: "血", Description: "血腥"},
	{Pinyin: "xun", Word: "训", Description: "训斥"},
	// Y
	{Pinyin: "ya", Word: "哑", Description: "哑巴"},
	{Pinyin: "yan", Word: "厌", Description: "厌恶"},
	{Pinyin: "yao", Word: "夭", Description: "夭折"},
	{Pinyin: "ye", Word: "业", Description: "业障"},
	{Pinyin: "yi", Word: "疫", Description: "瘟疫"},
	{Pinyin: "yin", Word: "阴", Description: "阴暗"},
	{Pinyin: "ying", Word: "硬", Description: "强硬"},
	{Pinyin: "yong", Word: "庸", Description: "平庸"},
	{Pinyin: "you", Word: "忧", Description: "忧愁"},
	{Pinyin: "yu", Word: "愚", Description: "愚蠢"},
	{Pinyin: "yuan", Word: "冤", Description: "冤屈"},
	{Pinyin: "yue", Word: "约", Description: "约束"},
	{Pinyin: "yun", Word: "晕", Description: "晕眩"},
	// Z
	{Pinyin: "za", Word: "砸", Description: "砸碎"},
	{Pinyin: "zai", Word: "灾", Description: "灾难"},
	{Pinyin: "zan", Word: "攒", Description: "攒眉"},
	{Pinyin: "zang", Word: "脏", Description: "肮脏"},
	{Pinyin: "zao", Word: "糟", Description: "糟糕"},
	{Pinyin: "ze", Word: "责", Description: "责难"},
	{Pinyin: "zei", Word: "贼", Description: "盗贼"},
	{Pinyin: "zha", Word: "炸", Description: "炸弹"},
	{Pinyin: "zhai", Word: "债", Description: "负债"},
	{Pinyin: "zhan", Word: "占", Description: "占据"},
	{Pinyin: "zhao", Word: "罩", Description: "笼罩"},
	{Pinyin: "zhe", Word: "折", Description: "折磨"},
	{Pinyin: "zhen", Word: "针", Description: "针刺"},
	{Pinyin: "zheng", Word: "挣", Description: "挣扎"},
	{Pinyin: "zhi", Word: "止", Description: "停止"},
	{Pinyin: "zhong", Word: "重", Description: "重压"},
	{Pinyin: "zhou", Word: "皱", Description: "皱眉"},
	{Pinyin: "zhu", Word: "逐", Description: "驱逐"},
	{Pinyin: "zhua", Word: "抓", Description: "抓伤"},
	{Pinyin: "zhuan", Word: "转", Description: "转衰"},
	{Pinyin: "zhuo", Word: "拙", Description: "笨拙"},
	{Pinyin: "zi", Word: "资", Description: "资本"},
	{Pinyin: "zong", Word: "宗", Description: "宗派"},
	{Pinyin: "zu", Word: "阻", Description: "阻碍"},
	{Pinyin: "zui", Word: "罪", Description: "罪过"},
	{Pinyin: "zuo", Word: "作", Description: "作孽"},
}

// BadPinyinCombo 不良拼音组合
// 名字全拼连读时可能产生的不良词汇
type BadPinyinCombo struct {
	Combo       []string // 拼音组合（连续音节）
	Description string   // 说明
}

// BadPinyinCombos 检查拼音连读产生的不良词汇
// 扩展覆盖：不雅词汇、疾病、灾难、偷盗、欺诈、死亡等
var BadPinyinCombos = []BadPinyinCombo{
	// 不雅称谓
	{Combo: []string{"wang", "ba"}, Description: "王八（不雅称谓）"},
	{Combo: []string{"wang", "dan"}, Description: "王蛋（不雅称谓）"},
	{Combo: []string{"sha", "bi"}, Description: "不雅词汇"},
	{Combo: []string{"er", "bi"}, Description: "不雅词汇"},
	{Combo: []string{"dou", "bi"}, Description: "不雅词汇"},
	{Combo: []string{"ca", "dan"}, Description: "扯蛋（不雅）"},
	{Combo: []string{"ji", "ba"}, Description: "不雅词汇"},
	{Combo: []string{"gan", "ni"}, Description: "不雅词汇"},
	{Combo: []string{"wo", "cao"}, Description: "不雅词汇"},
	{Combo: []string{"wo", "ri"}, Description: "不雅词汇"},
	// 傻/蠢
	{Combo: []string{"sha", "gua"}, Description: "傻瓜"},
	{Combo: []string{"bai", "chi"}, Description: "白痴"},
	{Combo: []string{"ben", "dan"}, Description: "笨蛋"},
	{Combo: []string{"sha", "dai"}, Description: "傻呆"},
	{Combo: []string{"chun", "dan"}, Description: "蠢蛋"},
	// 疾病/健康
	{Combo: []string{"shen", "jing", "bing"}, Description: "神经病"},
	{Combo: []string{"shen", "bing"}, Description: "神经病（缩写）"},
	{Combo: []string{"fei", "jie"}, Description: "肺结核"},
	{Combo: []string{"gan", "yan"}, Description: "肝炎"},
	{Combo: []string{"gan", "ran"}, Description: "感染"},
	{Combo: []string{"gan", "mao"}, Description: "感冒"},
	{Combo: []string{"fa", "shao"}, Description: "发烧"},
	{Combo: []string{"ke", "sou"}, Description: "咳嗽"},
	{Combo: []string{"pi", "lao"}, Description: "疲劳"},
	{Combo: []string{"zhi", "zhuang"}, Description: "智障"},
	{Combo: []string{"dai", "can"}, Description: "呆残"},
	// 灾难/不幸
	{Combo: []string{"si", "wang"}, Description: "死亡"},
	{Combo: []string{"bei", "ju"}, Description: "悲剧"},
	{Combo: []string{"can", "fei"}, Description: "残废"},
	{Combo: []string{"can", "ren"}, Description: "残忍"},
	{Combo: []string{"shi", "gu"}, Description: "事故"},
	{Combo: []string{"si", "le"}, Description: "死了"},
	{Combo: []string{"wan", "le"}, Description: "完了（完蛋）"},
	{Combo: []string{"tao", "wang"}, Description: "逃亡"},
	{Combo: []string{"wu", "wang"}, Description: "无望"},
	{Combo: []string{"jue", "wang"}, Description: "绝望"},
	{Combo: []string{"ku", "nan"}, Description: "苦难"},
	{Combo: []string{"zai", "nan"}, Description: "灾难"},
	{Combo: []string{"nuo", "ruo"}, Description: "懦弱"},
	{Combo: []string{"bei", "shang"}, Description: "悲伤"},
	// 犯罪/欺诈
	{Combo: []string{"liu", "mang"}, Description: "流氓"},
	{Combo: []string{"hei", "bang"}, Description: "黑帮"},
	{Combo: []string{"du", "pin"}, Description: "毒品"},
	{Combo: []string{"zou", "si"}, Description: "走私"},
	{Combo: []string{"tou", "dao"}, Description: "偷盗"},
	{Combo: []string{"qiang", "jie"}, Description: "抢劫"},
	{Combo: []string{"zha", "pian"}, Description: "诈骗"},
	{Combo: []string{"huo", "hai"}, Description: "祸害"},
	{Combo: []string{"dao", "qie"}, Description: "盗窃"},
	{Combo: []string{"fan", "zui"}, Description: "犯罪"},
	{Combo: []string{"zuo", "an"}, Description: "作案"},
	{Combo: []string{"guo", "shi"}, Description: "过失"},
	{Combo: []string{"wu", "zui"}, Description: "无罪"},
	// 死亡相关
	{Combo: []string{"shi", "wang"}, Description: "死亡"},
	{Combo: []string{"wan", "si"}, Description: "玩死"},
	// 性相关
	{Combo: []string{"se", "qing"}, Description: "色情"},
	{Combo: []string{"ji", "qing"}, Description: "激情（贬义）"},
	// 鬼怪
	{Combo: []string{"di", "yu"}, Description: "地狱"},
	{Combo: []string{"e", "gui"}, Description: "饿鬼"},
	{Combo: []string{"yao", "guai"}, Description: "妖怪"},
	{Combo: []string{"li", "gui"}, Description: "厉鬼"},
	{Combo: []string{"mo", "gui"}, Description: "魔鬼"},
	// 消极情绪
	{Combo: []string{"tong", "ku"}, Description: "痛苦"},
	{Combo: []string{"ju", "jue"}, Description: "拒绝"},
	{Combo: []string{"fa", "nu"}, Description: "发怒"},
	{Combo: []string{"ba", "dao"}, Description: "霸道"},
	{Combo: []string{"qiang", "ying"}, Description: "强硬"},
	{Combo: []string{"leng", "mo"}, Description: "冷漠"},
	{Combo: []string{"cu", "bao"}, Description: "粗暴"},
	// 其他消极
	{Combo: []string{"shi", "bai"}, Description: "失败"},
	{Combo: []string{"can", "ku"}, Description: "残酷"},
	{Combo: []string{"can", "hen"}, Description: "惨恨"},
	{Combo: []string{"shang", "hai"}, Description: "伤害"},
	{Combo: []string{"wu", "ru"}, Description: "侮辱"},
	{Combo: []string{"qin", "lue"}, Description: "侵略"},
	{Combo: []string{"qin", "fan"}, Description: "侵犯"},
	{Combo: []string{"wu", "xian"}, Description: "诬陷"},
	{Combo: []string{"chao", "xiao"}, Description: "嘲笑"},
	{Combo: []string{"qi", "fu"}, Description: "欺负"},
	{Combo: []string{"da", "ji"}, Description: "打击"},
	{Combo: []string{"di", "ren"}, Description: "敌人"},
	{Combo: []string{"dou", "zheng"}, Description: "斗争"},
	{Combo: []string{"zheng", "zha"}, Description: "挣扎"},
	{Combo: []string{"fu", "bai"}, Description: "腐败"},
	{Combo: []string{"tan", "wu"}, Description: "贪污"},
	{Combo: []string{"fei", "hua"}, Description: "废话"},
	{Combo: []string{"dan", "xiao"}, Description: "胆小"},
	{Combo: []string{"wu", "neng"}, Description: "无能"},
	{Combo: []string{"yong", "yuan"}, Description: "永远（谐音：永远走了）"},
	{Combo: []string{"ming", "ming"}, Description: "冥冥（冥界）"},
	{Combo: []string{"gu", "du"}, Description: "孤独"},
	{Combo: []string{"ji", "mo"}, Description: "寂寞"},
}

// stripTone 去除拼音声调并归一化，得到纯拼音（"wánɡ" → "wang"、"wang2" → "wang"）
//
// 适配生产数据两种格式：
//   - Unicode 声调符号：wánɡ → wang（顺带归一化 U+0261 ɡ → g，hanzi.json 中
//     「王」拼音即为此特殊字形，不归一化则谐音词表匹配失败）
//   - 数字后缀：wang2 → wang
func stripTone(pinyin string) string {
	if pinyin == "" {
		return ""
	}
	// 快路径：纯小写 ASCII 字母 + 可选结尾数字声调。
	// 生产数据（hanzi.json/namer.json）绝大多数拼音形如 "zhang" / "ming2"，
	// 走快路径可零分配返回；否则回落下面的 Builder 慢路径处理声调符号。
	if stripped, ok := stripToneASCII(pinyin); ok {
		return stripped
	}
	runes := []rune(pinyin)
	// 1. 数字后缀（wang2 → wang）
	last := runes[len(runes)-1]
	if last >= '1' && last <= '4' {
		runes = runes[:len(runes)-1]
	}
	// 2. Unicode 声调符号 → 基本元音；U+0261(ɡ) → g；ǖ/ǘ/ǚ/ǜ → v（与词表纯拼音一致）
	var b strings.Builder
	for _, r := range runes {
		switch r {
		case 'ā', 'á', 'ǎ', 'à':
			b.WriteRune('a')
		case 'ē', 'é', 'ě', 'è':
			b.WriteRune('e')
		case 'ī', 'í', 'ǐ', 'ì':
			b.WriteRune('i')
		case 'ō', 'ó', 'ǒ', 'ò':
			b.WriteRune('o')
		case 'ū', 'ú', 'ǔ', 'ù':
			b.WriteRune('u')
		case 'ǖ', 'ǘ', 'ǚ', 'ǜ':
			b.WriteRune('v')
		case 'ń', 'ň', 'ǹ':
			b.WriteRune('n')
		case '\u0261': // U+0261 Latin small letter script g（hanzi.json 特殊字形）
			b.WriteRune('g')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// stripToneASCII 纯 ASCII 拼音的零分配去声调。
//
// 仅当整串为小写字母（允许末尾一位 1-4 数字声调）时返回 ok=true，
// 此时结果是原串的子串（不分配内存）；含声调符号 / U+0261 / 其他字符时
// 返回 ok=false，由 stripTone 的慢路径处理。
func stripToneASCII(pinyin string) (string, bool) {
	n := len(pinyin)
	if n == 0 {
		return "", false
	}
	for i := 0; i < n; i++ {
		c := pinyin[i]
		if c >= 'a' && c <= 'z' {
			continue
		}
		// 仅允许末尾一位数字声调
		if c >= '1' && c <= '4' && i == n-1 {
			return pinyin[:n-1], true
		}
		return "", false
	}
	return pinyin, true
}

// CheckBadHomophone 检测单字是否属于不吉谐音词
//
// 自 P1 修复后改为精确匹配：仅当 selfChar == BadHomophones.Word 时才命中。
// 之前按拼音匹配会误杀所有同音好字（思 si→死、秀 xiu→朽 等）。
//
// selfChar 为本字（必传）；空字符串视为未传，按拼音匹配（旧行为，向后兼容）。
func CheckBadHomophone(pinyin string, selfChar ...string) (bool, string) {
	clean := stripTone(pinyin)
	if clean == "" {
		return false, ""
	}
	// 预编译索引：O(1) 命中同拼音条目（原实现每次线性扫描约 300 条词表）。
	// 下标切片保持词表原顺序，因此「返回哪一条」与线性扫描完全一致。
	idxs := homophoneByPinyin[clean]
	if len(idxs) == 0 {
		return false, ""
	}
	strict := len(selfChar) > 0 && selfChar[0] != ""
	for _, idx := range idxs {
		h := BadHomophones[idx]
		if !strict {
			// 兼容模式（selfChar 未传或为空）：仅按拼音匹配（旧行为）
			return true, "谐音「" + h.Word + "」" + h.Description
		}
		// 严格模式：仅当本字 == 谐音词时命中；否则继续比对同拼音的其他词条
		// （如 fu 同时有「腐」「妇」，仅任一匹配才命中，避免误杀同音好字）
		for _, c := range selfChar {
			if c == h.Word {
				return true, "谐音「" + h.Word + "」" + h.Description
			}
		}
	}
	return false, ""
}

// CheckAllBadHomophones 检测名字全部拼音是否有不吉谐音
// 返回 (是否命中, 描述列表)
//
// 拼音与本字按顺序一一对应（如 (sp=pinyinSurname, sp_char=姓, p1=char1拼音, p1_char=char1)）。
// 本字豁免：若拼音命中的不吉谐音词与本字相同（如"思"=="死"），跳过。
func CheckAllBadHomophones(pinyinsChars ...string) (bool, []string) {
	var details []string
	// 偶数参数：拼音 + 本字交替
	for i := 0; i < len(pinyinsChars); i += 2 {
		var pinyin string
		var chars []string
		if i+1 < len(pinyinsChars) {
			pinyin = pinyinsChars[i]
			chars = []string{pinyinsChars[i+1]}
		} else {
			pinyin = pinyinsChars[i]
		}
		if hit, desc := CheckBadHomophone(pinyin, chars...); hit {
			details = append(details, desc)
		}
	}
	return len(details) > 0, details
}

// CheckBadPinyinCombo 检测名字拼音连读是否产生不良词汇
// surname 姓氏拼音，givenNames 名字拼音列表
//
// 优化说明：原实现对 BadPinyinCombos（约 110 条）每次调用各做一次
// strings.Join 重新拼接，再各做一次 strings.Contains —— 单次调用产生
// 110 次字符串分配；在双名 N² 枚举中实测占单次评分 RateName 的 33%。
// 现改为「预编译索引 + 滑窗查表」：拼接串在 init 阶段构建一次，
// 运行时按词长在拼接文本上滑窗做 map 查表，无分配且与线性扫描语义等价。
func CheckBadPinyinCombo(surnamePinyin string, givenPinyins ...string) (bool, string) {
	// 构建完整拼音序列（姓氏 + 名字各字）
	var sb strings.Builder
	sb.Grow(len(surnamePinyin) + 8*len(givenPinyins))
	if surnamePinyin != "" {
		sb.WriteString(stripTone(surnamePinyin))
	}
	for _, p := range givenPinyins {
		sb.WriteString(stripTone(p))
	}
	joined := sb.String()
	if joined == "" {
		return false, ""
	}

	// 滑窗匹配：取词表中「最靠前」的命中项，与原线性扫描的返回顺序一致
	best := -1
	for i := 0; i < len(joined); i++ {
		for _, l := range pinyinComboLens {
			end := i + l
			if end > len(joined) {
				continue
			}
			if idx, ok := pinyinComboIndex[joined[i:end]]; ok && (best < 0 || idx < best) {
				best = idx
			}
		}
	}
	if best < 0 {
		return false, ""
	}
	comboStr := strings.Join(BadPinyinCombos[best].Combo, "")
	return true, "拼音连读「" + comboStr + "」" + BadPinyinCombos[best].Description
}

// ——— 热路径预编译索引 ———
//
// CheckBadPinyinCombo / CheckBadHomophone 在双名候选生成（N² 笛卡尔积）中
// 每个组合各调用一次，是评分链路最大的热点。两个词表都是包级只读常量，
// 因此在 init 阶段构建一次索引，把每次调用从 O(词表规模) 降到 O(文本长度)。
//
// 若未来需要热更新词表，必须调用 rebuildHomophoneIndex 同步重建索引，
// 否则索引与词表不一致。
var (
	homophoneByPinyin map[string][]int // 拼音 → BadHomophones 下标（保持词表顺序）
	pinyinComboIndex  map[string]int   // 拼接串 → BadPinyinCombos 下标（保留首次出现）
	pinyinComboLens   []int            // 拼接串长度去重集合（滑窗用）
)

func init() {
	rebuildHomophoneIndex()
}

// rebuildHomophoneIndex 依据当前词表重建索引（词表热更新后必须调用）
func rebuildHomophoneIndex() {
	homophoneByPinyin = make(map[string][]int, len(BadHomophones))
	for i := range BadHomophones {
		p := BadHomophones[i].Pinyin
		homophoneByPinyin[p] = append(homophoneByPinyin[p], i)
	}

	pinyinComboIndex = make(map[string]int, len(BadPinyinCombos))
	lenSet := make(map[int]bool)
	for i := range BadPinyinCombos {
		s := strings.Join(BadPinyinCombos[i].Combo, "")
		if _, dup := pinyinComboIndex[s]; !dup {
			// 保留首次出现：线性扫描遇到重复拼接串时命中的也是第一条
			pinyinComboIndex[s] = i
		}
		if !lenSet[len(s)] {
			lenSet[len(s)] = true
			pinyinComboLens = append(pinyinComboLens, len(s))
		}
	}
}
