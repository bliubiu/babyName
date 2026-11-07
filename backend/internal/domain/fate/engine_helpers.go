package fate

import (
	"context"
	"strings"
)

// - 范围：父系+母系直系长辈（建议往上两代）
func (s *sessionImpl) buildElderAvoidance(elderNames []string) (map[string]bool, map[string]bool) {
	elderCharSet := make(map[string]bool)
	elderPinyinSet := make(map[string]bool)

	for _, name := range elderNames {
		for _, r := range name {
			ch := string(r)
			// 同形字直接加入排除集
			elderCharSet[ch] = true
			// 查询该字的拼音，加入同音排除集
			if info, err := s.engine.provider.GetCharacter(ch); err == nil && info != nil {
				for _, p := range info.Pinyin {
					elderPinyinSet[p] = true
				}
			}
		}
	}
	return elderCharSet, elderPinyinSet
}

// cancelled 检查 context 是否已取消
func cancelled(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

// maxPerCharMeaningRunes 单字释义保留的最大 rune 数
// （hanzi.json 部分条目含《说文》类数百字长释义，需截断防撑爆响应与前端展示）
const maxPerCharMeaningRunes = 60

// combineCharMeanings 组合候选字释义为名字寓意描述。
// 单名仅取首字释义；双名以「；」连接两字释义；
// 每字释义按 rune 截断至 maxPerCharMeaningRunes 并补省略号。
func combineCharMeanings(m1, m2 string) string {
	return CombineMeanings(m1, m2, maxPerCharMeaningRunes)
}

// CombineMeanings 按指定上限截断并连接两字释义，是释义拼接的**唯一实现**。
//
// 释义拼接原为两份独立拷贝（领域层 combineCharMeanings 与应用层
// combineTwoMeanings，后者因前者未导出而复制），两者常量都硬编码为 60。
// docs/28 W8 指出：这种重复实现没有任何机制阻止两侧漂移，一旦有人只改一边，
// 同一个名字在「结果页」与「测名页」会显示不同寓意——正是本项目反复强调的
// 「双口径」缺陷。现收敛到本函数，应用层直接调用。
func CombineMeanings(m1, m2 string, maxRunes int) string {
	trunc := func(m string) string {
		m = strings.TrimSpace(m)
		if m == "" {
			return ""
		}
		r := []rune(m)
		if len(r) > maxRunes {
			return string(r[:maxRunes]) + "…"
		}
		return m
	}
	c1 := trunc(m1)
	if m2 == "" {
		return c1
	}
	c2 := trunc(m2)
	switch {
	case c1 == "":
		return c2
	case c2 == "":
		return c1
	default:
		return c1 + "；" + c2
	}
}

// firstPinyin 获取拼音列表中的第一个拼音（去掉声调数字）
func firstPinyin(pinyins []string) string {
	if len(pinyins) == 0 {
		return ""
	}
	return pinyins[0]
}

// firstPinyinForSurname 获取姓氏的拼音
func firstPinyinForSurname(surname string, provider CharacterProvider) string {
	if surname == "" {
		return ""
	}
	char, err := provider.GetCharacter(string([]rune(surname)[0]))
	if err != nil || char == nil {
		return ""
	}
	if len(char.Pinyin) == 0 {
		return ""
	}
	return char.Pinyin[0]
}

// narrowPoolByPreference 按用户显式偏好收窄候选池（docs/29 A9）。
//
// 输入：
//   - chars：已完成全部安全/质量过滤的候选池
//   - keywords：用户在「个性补充」输入的寓意关键词
//   - required：用户在「按偏旁选字」点选的用字
//
// 偏好集 = 命中任一关键词的字 ∪ 点选的字。命中判据（三者任一即可，命中面
// 尽量贴近用户直觉）：
//  1. 字本身等于关键词（如用户直接填了「森」）
//  2. 字义 Meaning 含关键词（如「苍」→ 苍茫、苍劲）
//  3. 起名分类 NamingCategory 含关键词（如「品德」「山水」）
//
// 返回值语义：
//   - 非 nil：收窄后的偏好集（按原池顺序保留，字符顺序稳定 → 结果可复现）
//   - nil  ：偏好集为空，调用方应放弃收窄、保持原候选池
//
// 为什么用收窄而不是加分：引擎对双名做的是全量 N×N 枚举，结果表按分数取
// TopN；调整枚举顺序只能省算力，改不了输出。加一个打分维度又会破坏
// RateNameScore 的「权重和为 1.0」前提，把偏好变成隐性加分项。故收窄是
// 唯一既能让偏好真正出现在结果里、又不污染分数语义的做法。
func narrowPoolByPreference(chars []*Character, keywords, required []string) []*Character {
	// 关键词归一：去空白、跳过空串，避免 "a, b" 切分后残留的空关键词
	// 把候选池误收窄成空集。
	terms := make([]string, 0, len(keywords))
	for _, kw := range keywords {
		if kw = strings.TrimSpace(kw); kw != "" {
			terms = append(terms, kw)
		}
	}

	requiredSet := make(map[string]bool, len(required))
	for _, ch := range required {
		if ch = strings.TrimSpace(ch); ch != "" {
			requiredSet[ch] = true
		}
	}

	if len(terms) == 0 && len(requiredSet) == 0 {
		return nil
	}

	narrowed := make([]*Character, 0, len(chars))
	for _, c := range chars {
		if requiredSet[c.Char] || matchesMeaningTerms(c, terms) {
			narrowed = append(narrowed, c)
		}
	}
	if len(narrowed) == 0 {
		// 一个都没命中：不能把候选池清空（否则必然零结果），
		// 交回 nil 让调用方保持原池。
		return nil
	}
	return narrowed
}

// matchesMeaningTerms 判断候选字是否命中任一关键词
func matchesMeaningTerms(c *Character, terms []string) bool {
	for _, term := range terms {
		if c.Char == term || strings.Contains(c.Meaning, term) {
			return true
		}
		for _, cat := range c.NamingCategory {
			if strings.Contains(cat, term) {
				return true
			}
		}
	}
	return false
}

// dedupCharsByName 按汉字去重候选池，保留首次出现。
//
// 保证 (i, j) 枚举出来的组合在「名字」层面唯一，使 worker 局部表可以
// 安全地不做 seen 去重（见 generateDoubleName 中的 NewExcellentTableUnique）。
func dedupCharsByName(chars []*Character) []*Character {
	if len(chars) < 2 {
		return chars
	}
	seen := make(map[string]struct{}, len(chars))
	out := make([]*Character, 0, len(chars))
	for _, c := range chars {
		if _, dup := seen[c.Char]; dup {
			continue
		}
		seen[c.Char] = struct{}{}
		out = append(out, c)
	}
	return out
}

// bestGenderHint 取两个字的性别暗示的"更明确的"那个
func bestGenderHint(a, b string) string {
	if a == b {
		return a
	}
	if a == "neutral" {
		return b
	}
	if b == "neutral" {
		return a
	}
	// 都有具体性别时取第一个
	if a != "" {
		return a
	}
	return b
}

// charPotentialScore 计算候选字的潜力分，用于预排序
// 高潜力字排在前面，使笛卡尔积循环中高分组合优先进入 ExcellentTable
func charPotentialScore(wuxing string, isCurated bool, positiveScore int, poetryFound bool, xiSet map[string]bool) int {
	score := 0
	// 五行匹配：喜用神 +12，生助喜用神 +8
	if xiSet[wuxing] {
		score += 12
	} else {
		for xi := range xiSet {
			if isWuXingSheng(wuxing, xi) {
				score += 8
				break
			}
		}
	}
	// 策展好字 +8
	if isCurated && positiveScore >= 90 {
		score += 8
	}
	// 寓意评分（0-9分，归一化）
	if positiveScore > 0 {
		score += int(positiveScore) / 10
	}
	// 诗词出典 +5
	if poetryFound {
		score += 5
	}
	return score
}

// hasPairBlacklist 检查字符的策展搭配黑名单中是否包含指定字
// 命中表示策展判定该字与此字组合不宜（如 瑾-艳/妍-艳/嫣-艳）
func hasPairBlacklist(ch *Character, other string) bool {
	if ch == nil || len(ch.PairBlacklist) == 0 {
		return false
	}
	for _, bl := range ch.PairBlacklist {
		if bl == other {
			return true
		}
	}
	return false
}

// ensureWuxingDiversity 五行多样性保底策略
//
// 从候选池中选取 topCount 个条目，确保每种五行组合（如金金、火土、金火等）
// 至少有1个代表。选取规则：
// 1. 按分数从高到低遍历候选池
// 2. 对于未出现过的五行组合，直接入选
// 3. 对于已出现过的五行组合，仅在有空位时入选
// 4. 所有空位填满后停止
//
// 这确保了 TopN 中既有高分候选（金金组合），也有不同五行组合的候选（火土/火金等），
// 使 WuxingRater 的间接生助梯度真正发挥作用。
func ensureWuxingDiversity(entries []ExcellentEntry, topCount int) []ExcellentEntry {
	if len(entries) <= topCount {
		return entries
	}

	result := make([]ExcellentEntry, 0, topCount)
	seenWuxing := make(map[string]bool) // 已出现的五行组合
	// 用于第二轮填充时快速判重，避免 O(N²) 嵌套循环
	seenInResult := make(map[string]bool, topCount)

	// 第一轮：每种五行组合取1个代表
	for _, e := range entries {
		if len(result) >= topCount {
			break
		}
		wuxing := e.WuXing1 + e.WuXing2
		if !seenWuxing[wuxing] {
			seenWuxing[wuxing] = true
			result = append(result, e)
			seenInResult[e.Char1+e.Char2] = true
		}
	}

	// 第二轮：未达 topCount 时，按分数填充剩余空位
	for _, e := range entries {
		if len(result) >= topCount {
			break
		}
		key := e.Char1 + e.Char2
		if seenInResult[key] {
			continue
		}
		seenInResult[key] = true
		result = append(result, e)
	}

	return result
}
