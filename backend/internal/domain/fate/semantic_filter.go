package fate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// semantic_filter.go 负面语义过滤（分层策略）
//
// 解决的问题（verify_fate 验证暴露）：
//   - 单名候选混入「愤/病/涕/霉/鄙/馒/唬/鞭/吮/屑」等字，评分器只加分不惩罚
//     负面含义，导致荒谬名字高分（如「李病」73.7、「王父母」90.4）。
//   - 双名组合「父母」（诗经《蓼莪》共现分加持）、「蜂蜜」等被当作好名字。
//
// 设计原则 —— 负面/消极判断必须分层，不能一刀切：
//   - 传统取名有"以病祈福"的先例：霍去病、辛弃疾（弃疾）都以病/疾入名，
//     寄托"驱除疾病"的祝愿。因此「病」「疾」等字不能硬性禁用。
//   - 生僻/消极字的判断是复杂语义问题，宁缺毋滥，仅对明确负面语义分层处理。
//
// 三层策略（对应 AGENTS.md 安全策略）：
//   1. 硬禁用层（hardNegativeChars）：秽物/尸棺/淫猥/盗匪/暴虐/刑具/贬义类，
//      任何语境都不适合作为名字 → 候选池阶段直接剔除。
//   2. 软惩罚层（softNegativeChars）：病痛/消极情绪类字，
//      不硬禁用（保留"去病/弃疾"类祈福组合），但在评分阶段重罚，
//      使荒谬结果（如「李病」）无法进入推荐榜。
//   3. 祈福豁免层（blessingCombos）：否定/祈愿动词+消极字的组合
//      （去病/弃疾/除疾/愈病），识别为积极祈愿 → 评分不罚反奖。
//   4. 组合禁忌层（forbiddenCombos）：「父母」「蜂蜜」等亲属称谓/物名组合，
//      双名候选命中即剔除（正反序均检测）。

// hardNegativeChars 硬禁用字表（候选池阶段剔除）
//
// 收录原则：任何语境下都无正面命名意义、必然引起不良联想的字。
var hardNegativeChars = map[string]bool{
	// 秽物
	"屎": true, "尿": true, "屁": true, "粪": true, "溺": true,
	// 死亡丧葬
	"尸": true, "棺": true, "墓": true, "坟": true, "葬": true,
	"殓": true, "殇": true, "骸": true, "骨": true,
	// 淫猥
	"淫": true, "奸": true, "娼": true, "妓": true, "嫖": true,
	"妾": true, "狎": true,
	// 盗匪
	"盗": true, "贼": true, "匪": true, "寇": true, "劫": true, "绑": true,
	// 暴虐刑罚
	"暴": true, "虐": true, "戾": true, "凶": true, "屠": true, "戮": true,
	"刑": true, "囚": true, "狱": true, "铐": true, "鞭": true,
	// 腐败霉烂
	"霉": true, "秽": true, "污": true, "浊": true, "臭": true, "腐": true,
	"朽": true, "烂": true, "蚀": true, "蛀": true, "馊": true, "晦": true,
	// 体液生理（同骨/骸，与鲜血/流血强联想，任何语境不宜入名）
	"血": true,
	// 贬义/不雅/低劣
	"鄙": true, "陋": true, "愚": true, "蠢": true, "笨": true, "呆": true,
	"傻": true, "痴": true, "癫": true, "蛮": true, "横": true, "诈": true,
	"伪": true, "佞": true, "谄": true, "骗": true, "贱": true, "奴": true,
	"丐": true, "乞": true, "叛": true, "唬": true, "愤": true, "屑": true,
	"馒": true, "吮": true, "辱": true, "耻": true, "卑": true, "瘸": true,
}

// softNegativeChars 软惩罚字表（评分扣分，不硬禁用）
//
// 收录原则：本身含义消极/病痛/悲苦，但有传统祈福用法或语义上有反转可能
// （去病/弃疾），因此保留入池资格，仅在评分阶段重罚。
var softNegativeChars = map[string]bool{
	// 病痛
	"病": true, "疾": true, "痛": true, "疡": true, "恙": true,
	// 消极情绪
	"哀": true, "愁": true, "悲": true, "苦": true, "恨": true, "怨": true,
	"怒": true, "恼": true, "悔": true, "憾": true, "疚": true, "愧": true,
	"忧": true, "伤": true, "惨": true, "凄": true, "泣": true, "涕": true,
	"哭": true, "郁": true, "闷": true,
}

// blessingCombos 祈福豁免组合表
//
// 否定/祈愿动词 + 消极字 → 传统"以病祈福"的积极寓意（去病/弃疾）。
// 命中时评分不扣分反而加分，尊重传统取名智慧。
var blessingCombos = map[string]bool{
	"去病": true, "弃疾": true, "除疾": true, "愈病": true, "去疾": true,
	"除病": true, "愈疾": true, "安恙": true, "康宁": true,
}

// forbiddenCombos 双字组合禁忌表
//
// 命中规则：组合（c1+c2）或其反序（c2+c1）命中即剔除。
// 覆盖类别：
//   - 亲属称谓组合：「父母」「爷奶」「爹娘」等（直接语义即为称谓，不适合做名字）
//   - 生活物品/食物词：「蜂蜜」「蜜蜂」等（名字不能是普通物名）
//   - 常见动词/口语组合：作为名字无意义甚至怪异
var forbiddenCombos = map[string]bool{
	// 亲属称谓
	"父母": true, "爸妈": true, "爹娘": true, "爷奶": true,
	"姥姥": true, "姥爷": true, "外公": true, "外婆": true,
	"兄弟": true, "姐妹": true, "哥哥": true, "姐姐": true,
	"弟弟": true, "妹妹": true, "叔叔": true, "阿姨": true,
	"伯伯": true, "姑姑": true, "舅舅": true, "婶婶": true,
	"爷爷": true, "奶奶": true, "儿子": true, "女儿": true,
	"孙子": true, "孙女": true, "爹妈": true,
	// 生活物品/食物词
	"蜂蜜": true, "蜜蜂": true, "面包": true, "馒头": true, "米饭": true,
	"面条": true, "豆腐": true, "鸡蛋": true, "牛奶": true, "苹果": true,
	"香蕉": true, "葡萄": true, "西瓜": true, "土豆": true, "白菜": true,
	"萝卜": true, "豆芽": true, "酱油": true,
	// 常见动词/动宾组合（作为名字无意义）
	"睡觉": true, "吃饭": true, "喝水": true, "跑步": true, "走路": true,
	"看书": true, "写字": true, "唱歌": true, "跳舞": true,
	// 地名组合（verify_fate 第三轮抓取：双名 Top5 混入「白河/江北」等地理名词）
	// 白/江/北/河 单字均为优质常用字（白露/江月/北辰/河川），绝不入门禁字表；
	// 但两两组合为存世地名（陕西白河县/重庆江北区），作为名字撞地理专名，组合级剔除。
	"白河": true, "江北": true,
}

// IsHardNegativeChar 判断单字是否为硬禁用字（候选池阶段剔除）
func IsHardNegativeChar(char string) bool {
	return hardNegativeChars[char]
}

// IsSoftNegativeChar 判断单字是否为软惩罚字（评分阶段扣分）
func IsSoftNegativeChar(char string) bool {
	return softNegativeChars[char]
}

// IsBlessingCombo 判断两字组合是否为祈福豁免组合（去病/弃疾等）
func IsBlessingCombo(c1, c2 string) bool {
	if blessingCombos[c1+c2] {
		return true
	}
	return blessingCombos[c2+c1]
}

// IsBadCombo 判断两个字组合是否为禁忌组合（正反序均检测）
//
// 优先级：动态加载的清洗组合（forbiddenCombosExtra）→ 硬编码常量（forbiddenCombos）
// 动态集为空时仅查硬编码（向后兼容，避免未加载数据时所有组合都通过）
// 用于双名候选生成时剔除「父母」「蜂蜜」等荒谬组合
func IsBadCombo(c1, c2 string) bool {
	if c1 == "" || c2 == "" {
		return false
	}
	forward := c1 + c2
	reverse := c2 + c1
	// 1. 优先查动态加载的 962 条清洗组合（data/forbidden_combos.json）
		forbiddenCombosExtraMu.RLock()
		if _, hit := forbiddenCombosExtra[forward]; hit {
			forbiddenCombosExtraMu.RUnlock()
			return true
		}
		if _, hit := forbiddenCombosExtra[reverse]; hit {
			forbiddenCombosExtraMu.RUnlock()
			return true
		}
		forbiddenCombosExtraMu.RUnlock()
	// 2. 未命中再查硬编码常量（向后兼容 + 兜底）
	if forbiddenCombos[forward] {
		return true
	}
	return forbiddenCombos[reverse]
}

// forbiddenCombosExtra 动态加载的禁忌组合（来自 data/forbidden_combos.json）
// 启动时由 LoadForbiddenCombosFromJSON 注入；为 nil/空时仅使用硬编码 forbiddenCombos
var (
	forbiddenCombosExtraMu sync.RWMutex
	forbiddenCombosExtra   = map[string]bool{}
)

// LoadForbiddenCombosFromJSON 从 data/forbidden_combos.json 加载禁忌组合
//
// 数据文件缺失时降级为空集合（仅警告、不阻断启动），避免缺失字表导致
// 整个起名服务无法启动；文件存在但解析失败时仍返回错误以暴露数据损坏。
//
// 加载后会覆盖既有集合（幂等，可重复调用以热更新）。
func LoadForbiddenCombosFromJSON(dataDir string) error {
	path := filepath.Join(dataDir, "forbidden_combos.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			forbiddenCombosExtraMu.Lock()
			forbiddenCombosExtra = make(map[string]bool)
			forbiddenCombosExtraMu.Unlock()
			fmt.Fprintf(os.Stderr, "警告: 禁忌组合表 %s 不存在，已降级为硬编码常量集合（不加载动态数据）\n", path)
			return nil
		}
		return fmt.Errorf("读取禁忌组合表失败: %w", err)
	}
	var combos []string
	if err := json.Unmarshal(raw, &combos); err != nil {
		return fmt.Errorf("解析禁忌组合表失败: %w", err)
	}

	forbiddenCombosExtraMu.Lock()
	defer forbiddenCombosExtraMu.Unlock()
	forbiddenCombosExtra = make(map[string]bool, len(combos))
	for _, c := range combos {
		if c == "" {
			continue
		}
		// 兼容正反序：双向都加（与 IsBadCombo 一致）
		forbiddenCombosExtra[c] = true
	}
	return nil
}

// ForbiddenComboCount 返回动态禁忌组合数量（诊断/测试用）
func ForbiddenComboCount() int {
	forbiddenCombosExtraMu.RLock()
	defer forbiddenCombosExtraMu.RUnlock()
	return len(forbiddenCombosExtra)
}