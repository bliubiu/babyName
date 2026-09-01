package fate

import (
	"fmt"
	"math"
	"strings"

	"name/internal/domain/classics"
	"name/internal/domain/zodiac"
)

// Rater 名字评分接口
//
// 借鉴 fate-main 的五维评分体系，每个 Rater 负责一个维度的评分:
//   - WuxingRater:   五行八字匹配度（权重 30%，命理层核心）
//   - WenHuaRater:   文化印象 — 常用度/字义丰富度（权重 20%，字象层）
//   - YinYunRater:   音韵和谐度（权重 20%，字象层）
//   - ShengXiaoRater:生肖匹配度（权重 15%，命理层）
//   - SancaiRater:   天地人三才搭配（权重 15%，新增维度）
//
// 注：原 WuGeRater（熊崎五格数理）已移除，遵循 AGENTS.md 禁用熊崎五格约束。
type Rater interface {
	// Rate 给名字候选评分
	Rate(candidate *NameCandidate, fateData *FateData) NameRating
	// Name 评分维度名称
	Name() string
	// Weight 该维度在总分中的权重（所有 Rater 权重之和应为 1.0）
	Weight() float64
}

// NameRating 单维度评分结果
type NameRating struct {
	Score  float64 `json:"score"`  // 该维度得分（0-100）
	Detail string  `json:"detail"` // 文字解释
}

// RateName 使用一组 Rater 计算名字的综合评分
//
// 方案B+（组合质量封顶）：对于「未获策展认可」的双名组合，
// 在四个机械维度（文化印象/音韵/生肖/三才）封顶 75 分，五行八字维度
// 不封顶（命理层核心，衡量补缺而非质量）。
//
// 背景：全量枚举生成的双名候选池高达 130 万，存在大量「祝董/至律/遂姊/弟昆/
// 至令/团炸/号觅」这类机械凑合的组合——它们的构成字全是一二级规范字、
// Meaning 非空、音韵巧合好听、五行与喜用神匹配、生肖三才恰好高分，四维全满分
// 从而总分飙到 89+ 霸榜，压过「军武/盛明」等真正优质的名字。
//
// 豁免设计：策展白名单（IsCuratedName）是「组合有文化含量」的唯一硬证据——
// 人类认可过的搭配才豁免封顶。诗词出典（HasPoetry/PoetryFrom）在此**不作为豁免
// 条件**：双名场景下它们是单字级证据（engine 中 poetryFound = 甲字||乙字任一单字
// 出现在诗词中），常用字几乎必有单字出典，若据此豁免会架空封顶，让「团炸/号觅」
// 这类单字平凡但组合荒谬的名字继续霸榜。真出典组合（如「疏影」）靠策展白名单
// 或 GetBigramScore 共现分加分，无需此豁免。
//
// 效果：荒谬组合四维封顶 75 后总分上限 = 五行85×0.30 + 75×0.70 = 78，
// 必然低于策展好名；而优质非策展名凭五行匹配度（不封顶）拉开差距。
func RateName(candidate *NameCandidate, fateData *FateData, raters []Rater) NameScore {
	items := make(map[string]float64, len(raters))
	var total float64

	for _, r := range raters {
		rating := r.Rate(candidate, fateData)
		items[r.Name()] = rating.Score
		total += rating.Score * r.Weight()
	}

	// 方案B+：非策展双名（单名无策展概念，不封顶）
	//
	// 豁免条件扩展：除策展好名白名单外，「含精选好字」的组合
	// （IsCuratedN && PositiveScoreN>=85，与 WenHuaRater +8 加分同条件）同样豁免。
	// 否则 WenHuaRater 给精选好字的文化加分会被本封顶截回 75，策展加分机制被架空——
	// 实测喜用神收窄到单五行后白名单组合全部出局，「已谦」（谦=策展∩ps87）与荒谬
	// 组合同聚封顶基准分 69.3（=五行56×0.30+75×0.70），Top50 平均分断言失败。
	premiumChar := (candidate.IsCurated1 && candidate.PositiveScore1 >= 90) ||
		(candidate.IsCurated2 && candidate.PositiveScore2 >= 90)
	if candidate.Char2 != "" && candidate.Char1 != "" &&
		!IsCuratedName(candidate.Char1, candidate.Char2) && !premiumChar {
		capped := []string{"文化印象", "音韵", "生肖", "三才"}
		for _, dim := range capped {
			if items[dim] > 75 {
				total -= (items[dim] - 75) * raterWeight(raters, dim)
				items[dim] = 75
			}
		}
	}

	// 限制总分在 0-100 范围内
	total = clampScore(total)

	// 四舍五入保留一位小数
	total = math.Round(total*10) / 10

	return NameScore{
		Total: total,
		Grade: scoreToGrade(total),
		Items: items,
	}
}

// raterWeight 根据维度名查权重（配合封顶重算总分）
func raterWeight(raters []Rater, dim string) float64 {
	for _, r := range raters {
		if r.Name() == dim {
			return r.Weight()
		}
	}
	return 0
}

// scoreToGrade 将分数转换为等级
func scoreToGrade(score float64) string {
	switch {
	case score >= 90:
		return "上上"
	case score >= 80:
		return "上吉"
	case score >= 70:
		return "中吉"
	case score >= 60:
		return "中平"
	case score >= 50:
		return "中下"
	default:
		return "下下"
	}
}

// --- 内置 Rater 实现 ---

// WuxingRater 五行八字匹配度评分
// 考察名字五行与八字喜用神的匹配程度
type WuxingRater struct {
	weight float64
}

func NewWuxingRater() *WuxingRater {
	return &WuxingRater{weight: 0.25}
}

func NewWuxingRaterWithWeight(w float64) *WuxingRater {
	return &WuxingRater{weight: w}
}

func (r *WuxingRater) Name() string    { return "五行八字" }
func (r *WuxingRater) Weight() float64 { return r.weight }

func (r *WuxingRater) Rate(candidate *NameCandidate, fateData *FateData) NameRating {
	if fateData == nil {
		return NameRating{Score: 80, Detail: "五行信息良好"}
	}

	// 使用 XiYongShen 列表（而非单值 Xi/Ji）评估五行匹配
	// 修复：BalanceXiYongJi 在补最缺时可能覆盖 Yong 但未更新 Ji，
	// 导致 Ji 与 XiYongShen 冲突（如 Ji="土" 但 XiYongShen=["土","金"]）。
	// 使用 XiYongShen 列表使评估标准与引擎收窄池一致。
	xiYongShen := fateData.WuXingXiji.XiYongShen
	jiWuxing := fateData.WuXingXiji.Ji

	// 防御性 fallback：当 XiYongShen 为空时，用单值 Xi 构造列表
	if len(xiYongShen) == 0 && fateData.WuXingXiji.Xi != "" {
		xiYongShen = []string{fateData.WuXingXiji.Xi}
	}

	// 构建喜用神集合（用于间接生助判断）
	xiSet := make(map[string]bool, len(xiYongShen))
	for _, wx := range xiYongShen {
		xiSet[wx] = true
	}

	score := 50.0
	var details []string
	matchCount := 0

	chars := []struct {
		char   string
		wuxing string
	}{{candidate.Char1, candidate.WuXing1}, {candidate.Char2, candidate.WuXing2}}

	for _, c := range chars {
		if c.char == "" || c.wuxing == "" {
			continue
		}
		switch {
		case xiSet[c.wuxing]:
			score += 12
			matchCount++
			details = append(details, fmt.Sprintf("「%s」五行属%s，为喜用神", c.char, c.wuxing))
		case c.wuxing == jiWuxing:
			score -= 10
			details = append(details, fmt.Sprintf("「%s」五行属%s，为忌神", c.char, c.wuxing))
		default:
			// 间接生助梯度：生任一喜用神 > 被任一喜用神所生（泄气）> 中性
			generatesXi := false
			for _, xi := range xiYongShen {
				if isWuXingSheng(c.wuxing, xi) {
					generatesXi = true
					break
				}
			}
			if generatesXi {
				score += 10
				details = append(details, fmt.Sprintf("「%s」五行属%s，生助喜用神", c.char, c.wuxing))
			} else {
				drainedByXi := false
				for _, xi := range xiYongShen {
					if isWuXingSheng(xi, c.wuxing) {
						drainedByXi = true
						break
					}
				}
				if drainedByXi {
					score += 3
					details = append(details, fmt.Sprintf("「%s」五行属%s，喜用神所生（泄气）", c.char, c.wuxing))
				} else {
					score += 3
					details = append(details, fmt.Sprintf("「%s」五行属%s，中性", c.char, c.wuxing))
				}
			}
		}
	}

	// 两字均匹配喜用神额外加分
	if matchCount == 2 {
		score += 5
		details = append(details, "两字皆匹配喜用神，补益力强")
	}

	// 两字五行相生加分
	if candidate.WuXing1 != "" && candidate.WuXing2 != "" {
		if isWuXingSheng(candidate.WuXing1, candidate.WuXing2) ||
			isWuXingSheng(candidate.WuXing2, candidate.WuXing1) {
			score += 10
			details = append(details, "两字五行相生，搭配协调")
		}
		if isWuXingKe(candidate.WuXing1, candidate.WuXing2) ||
			isWuXingKe(candidate.WuXing2, candidate.WuXing1) {
			score -= 8
			details = append(details, "两字五行相克，需注意")
		}
	}

	score = clampScore(score)
	detail := "五行搭配均衡，喜用神匹配良好"
	if len(details) > 0 {
		detail = strings.Join(details, "；")
	}
	return NameRating{Score: score, Detail: detail}
}

// DefaultRaters 返回默认的七维评分器列表
//
// 移除了 WuGeRater（熊崎五格数理），新增 SancaiRater（天地人三才）+ NoveltyRater（新颖度）
// + FrequencyRater（人名频率），权重重新分配：
//   - WuxingRater     24%（命理层核心）
//   - WenHuaRater     14%（字象层）
//   - YinYunRater     14%（字象层）
//   - ShengXiaoRater  11%（命理层）
//   - SancaiRater      8%（三才）
//   - NoveltyRater    14%（新颖度，区分度核心）
//   - BigramRater      9%（诗词二字共现频率）
//   - FrequencyRater   6%（人名频率，真实语料统计）
func DefaultRaters() []Rater {
	return []Rater{
		NewWuxingRaterWithWeight(0.24),    // 24% 命理层
		NewWenHuaRaterWithWeight(0.14),    // 14% 字象层
		NewYinYunRaterWithWeight(0.14),    // 14% 字象层
		NewShengXiaoRaterWithWeight(0.11), // 11% 命理层
		NewSancaiRaterWithWeight(0.08),    //  8% 三才
		NewNoveltyRaterWithWeight(0.14),   // 14% 新颖度
		NewBigramRaterWithWeight(0.09),    //  9% 共现分
		NewFrequencyRaterWithWeight(0.06), //  6% 人名频率
	}
}

// WenHuaRater 文化印象评分
// 考察常用度、字义丰富度、笔画匀称度
type WenHuaRater struct {
	weight float64
}

func NewWenHuaRater() *WenHuaRater {
	return &WenHuaRater{weight: 0.20}
}

func NewWenHuaRaterWithWeight(w float64) *WenHuaRater {
	return &WenHuaRater{weight: w}
}

func (r *WenHuaRater) Name() string    { return "文化印象" }
func (r *WenHuaRater) Weight() float64 { return r.weight }

func (r *WenHuaRater) Rate(candidate *NameCandidate, fateData *FateData) NameRating {
	score := 60.0
	var details []string

	// 常用字加分
	if candidate.IsRegular {
		score += 5
	}
	// 字义明确加分
	if candidate.Meaning1 != "" {
		score += 4
	}
	if candidate.Meaning2 != "" {
		score += 4
	}
	// 笔画匀称加分
	if candidate.Stroke1 > 0 && candidate.Stroke2 > 0 {
		diff := absInt(candidate.Stroke1 - candidate.Stroke2)
		if diff <= 5 {
			score += 2
			details = append(details, "笔画搭配匀称")
		}
	}

	// ——— 策展字加分（人工精选起名好字，破单名同分僵局的治本方案） ———
	// 策展覆盖表（17 个起名分类，约 446 字，characterCategoryList 人工精选）是
	// 荒谬字与好字的强区分信号：verify_fate 复测验证好字（毅/辉/涛/英/琳/雅/梅/强/凯）
	// 8/12 在表内，荒谬字（贪/疟/骂/吠/靶/振/凑/递/备/宦/够/播/蚊/浅/眉/驳/圃/沈/亩/泥/沟）
	// 21/22 不在表内——仅"清"在，而清本身是优质起名字。
	// 背景：单名 Top5 曾因荒谬字与好字五维同分而霸榜，排序由候选池枚举顺序决定，
	// 门禁黑名单打了 11 轮荒谬字换面孔打不完。给策展字 +8 分/字的文化加分可打破同分：
	// 荒谬字不在策展表不得分，优质字得分反超进入 Top5。
	// 注意：不可用 NamingCategory 非空替代（自动分类的五行兜底会让荒谬字"贪"水→"清新水韵"
	// 也获分类），必须精确使用人工策展覆盖表（IsCurated1/IsCurated2）。
	// 二轮收紧：策展表是「分类字表」而非「精选好字表」，平庸字（软/际/映/耿/宝/念/畅/章/好/典）
	// 也在表内拿到 +8，导致 Top5 被平庸字霸榜。结合 PositiveScore>=85（namer.json 寓意评分，
	// 优质字 87-91 有值、平庸/荒谬字为空）把加分收窄为「策展 ∩ positiveScore>=85」精选好字专属。
	curatedCount := 0
	curatedChars := make([]string, 0, 2)
	if candidate.Char1 != "" && candidate.IsCurated1 && candidate.PositiveScore1 >= 90 {
		curatedCount++
		curatedChars = append(curatedChars, candidate.Char1)
	}
	if candidate.Char2 != "" && candidate.IsCurated2 && candidate.PositiveScore2 >= 90 {
		curatedCount++
		curatedChars = append(curatedChars, candidate.Char2)
	}
	if curatedCount > 0 {
		score += float64(curatedCount) * 8
		details = append(details, fmt.Sprintf("「%s」为策展起名好字（+%d分）",
			strings.Join(curatedChars, "、"), curatedCount*8))
	}

	// ——— 诗词出处加分（含近义语义扩展） ———

	// 精确匹配（由 name generator 预计算）
	if candidate.HasPoetry {
		score += 8
		detail := "出自诗词典故"
		if candidate.PoetryFrom != "" {
			detail = "出自" + candidate.PoetryFrom
		}
		details = append(details, detail)
	}

	// 语义扩展匹配（精确匹配未覆盖时检查近义字关联）
	if !candidate.HasPoetry {
		if candidate.Char1 != "" {
			if desc := checkSemanticPoetry(candidate.Char1); desc != "" {
				score += 5
				details = append(details, desc)
			}
		}
		if candidate.Char2 != "" {
			if desc := checkSemanticPoetry(candidate.Char2); desc != "" {
				score += 5
				details = append(details, desc)
			}
		}
	}

	// 二字共现加分（诗经楚辞等经典中的同句搭配）
	// 基准来源：策展白名单 + 《通用规范汉字表》等级 + 语义门禁（多条件联合）：
	//   1. 组合命中策展好名白名单（IsCuratedName）且两字均为一二级规范字且均非门禁字
	//      → 满分共现加成：只有人类策展认可的搭配才配得上满分典籍共现；
	//   2. 其余情况（未获策展认可 / 含门禁字 / 含三级表外生僻字）→ 共现分彻底归零（×0），
	//      避免「逐求/值接/梁董/伏清/法皇/呼母/祝董/至律」等随机组合仅因典籍同句共现而虚高——
	//      它们构成字全是一二级规范字，仅靠 CommonLevel+门禁 条件无法拦截。
	// 说明：策展库在此仅作「共现白名单」参考（哪些搭配被人类认可过），
	// 不作评分基准加分；未获策展认可不等于劣质，只是不能白拿满分共现加成。
	// 方案B+：非策展组合的共现加成从 ×0.4 降权进一步收紧为 ×0（彻底归零），
	// 配合 RateName 聚合层的「非策展双名四维封顶 75」共同压低荒谬组合总分。
	if candidate.Char1 != "" && candidate.Char2 != "" && candidate.Char1 != candidate.Char2 {
		if bgScore, bgSrc, found := classics.GetBigramScore(candidate.Char1, candidate.Char2); found {
			lvlOK := candidate.CommonLevel1 >= 1 && candidate.CommonLevel1 <= 2 &&
				candidate.CommonLevel2 >= 1 && candidate.CommonLevel2 <= 2
			if IsCuratedName(candidate.Char1, candidate.Char2) &&
				!IsNonNamingChar(candidate.Char1) && !IsNonNamingChar(candidate.Char2) && lvlOK {
				score += float64(bgScore)
				details = append(details, fmt.Sprintf("「%s%s」共现于%s（+%d分）",
					candidate.Char1, candidate.Char2, bgSrc, bgScore))
			} else {
				// 未获策展认可 / 含门禁字 / 含生僻字：共现分彻底归零（方案B+）
				details = append(details, fmt.Sprintf("「%s%s」共现于%s，未获策展认可，共现分归零",
					candidate.Char1, candidate.Char2, bgSrc))
			}
		}
	}

	// 策展好名加成已移除：策展库（curated_names.json）不可作为质量基准
	// （其本身是历史人物名采集，混入「仲尼/若兮/七政/与砺」等 962 条垃圾，见探查结论）。
	// 名字搭配质量由「通用规范汉字表一二级 + 语义门禁」独立判定。

	// 单名语义共现：Char1 的近义字与 Char1 在诗词中的搭配
	if candidate.Char2 == "" && candidate.Char1 != "" {
		if desc := checkSingleNameBigram(candidate.Char1); desc != "" {
			score += 3
			details = append(details, desc)
		}
	}

	// ——— 统计型门禁：语义荒谬字惩罚 + 生僻字降权 ———
	// 基准来源已从策展库（curated_names.json）切换到《通用规范汉字表》+ 语义过滤：
	//   1. 语义门禁字（IsNonNamingChar：带/陵/撰/停/毛/逢/冯/比/溉/俯/戒/必 等动词物名，
	//      及虚词/排行/口语/数字类）任何语境下都无命名价值 → 文化印象维度重罚 -12 分/字，
	//      使其无法仅靠常用字/字义/诗词加分进入推荐榜。
	//   2. 生僻字（CommonLevel>=3 即《通用规范汉字表》三级字，或 level==0 表外补充字）
	//      → 降权 -8 分/字，符合「首选一级、二级字表，谨慎使用三级字表」原则。
	// 常用一级/二级好字（浩/然/文/子/白/汝/翰/墨/澄/渊）不受惩罚。
	penalizedChars := make([]string, 0, 2)
	rareChars := make([]string, 0, 2)
	both := []struct {
		char string
		lvl  int
	}{
		{candidate.Char1, candidate.CommonLevel1},
		{candidate.Char2, candidate.CommonLevel2},
	}
	for _, ch := range both {
		if ch.char == "" {
			continue
		}
		if IsNonNamingChar(ch.char) {
			penalizedChars = append(penalizedChars, ch.char)
		} else if ch.lvl >= 3 || ch.lvl == 0 {
			// level==0 为表外补充字（旧版精选字库），level>=3 为三级生僻字
			rareChars = append(rareChars, ch.char)
		}
	}
	if len(penalizedChars) > 0 {
		score -= float64(len(penalizedChars)) * 12
		details = append(details, fmt.Sprintf("含无命名价值字「%s」（-12分/字）",
			strings.Join(penalizedChars, "、")))
	}
	if len(rareChars) > 0 {
		score -= float64(len(rareChars)) * 8
		details = append(details, fmt.Sprintf("含生僻/表外字「%s」（-8分/字）",
			strings.Join(rareChars, "、")))
	}

	// ——— 负面语义惩罚（分层：软惩罚 + 祈福豁免） ———
	// 软惩罚字（病/疾/哀/愁等）保留入池资格但不该得高分；
	// 命中祈福豁免组合（去病/弃疾）时视为积极祈愿，不罚反奖。
	softCount := 0
	negChars := make([]string, 0, 2)
	if candidate.Char1 != "" && IsSoftNegativeChar(candidate.Char1) {
		softCount++
		negChars = append(negChars, candidate.Char1)
	}
	if candidate.Char2 != "" && IsSoftNegativeChar(candidate.Char2) {
		softCount++
		negChars = append(negChars, candidate.Char2)
	}
	if softCount > 0 {
		if candidate.Char1 != "" && candidate.Char2 != "" && IsBlessingCombo(candidate.Char1, candidate.Char2) {
			score += 6
			details = append(details, fmt.Sprintf("「%s%s」以病祈福，寓意祛病安康（+6分）",
				candidate.Char1, candidate.Char2))
		} else {
			score -= float64(softCount) * 8
			details = append(details, fmt.Sprintf("含消极字「%s」，寓意需斟酌（-8分/字）",
				strings.Join(negChars, "、")))
		}
	}

	// ——— 数据层策展扣分（NamePenalty） ———
	// hanzi.json 对部分字人工标注了起名扣分（如 响15/荒12/晦12/虚12/昧12/冥10 等），
	// 表示这些字作为名字有明显减分项，评分阶段按比例扣减，使「至律/值接」类
	// 仅靠典籍共现推高的组合无法进入推荐榜。
	if p := candidate.NamePenalty1 + candidate.NamePenalty2; p > 0 {
		deduct := float64(p) * 0.6
		score -= deduct
		details = append(details, fmt.Sprintf("策展标注起名扣分（-%.1f分）", deduct))
	}

	score = clampScore(score)
	detail := "常用字搭配，字义明确，文化内涵良好"
	if len(details) > 0 {
		detail = strings.Join(details, "；")
	}
	return NameRating{Score: score, Detail: detail}
}

// checkSemanticPoetry 检查字符是否有近义关联的诗词出处
// 返回描述文字（空字符串表示无匹配）
func checkSemanticPoetry(char string) string {
	_, synChar, entries := classics.FindPoetryByCharSemantic(char)
	if synChar != "" && len(entries) > 0 {
		entry := entries[0]
		src := entry.Sentence
		if src == "" {
			src = entry.Work + "·" + entry.Chapter
		}
		return fmt.Sprintf("「%s」近义于「%s」，关联「%s」", char, synChar, src)
	}
	return ""
}

// checkSingleNameBigram 检查单名是否通过近义字获得诗词共现加分
func checkSingleNameBigram(char string) string {
	if synChar, coChar, src, found := classics.GetSemanticBigramMatch(char); found {
		return fmt.Sprintf("「%s」借「%s·%s」共现于%s（+3分）", char, synChar, coChar, src)
	}
	return ""
}

// YinYunRater 音韵评分
// 考察声调变化、声母韵母搭配
type YinYunRater struct {
	weight float64
}

func NewYinYunRater() *YinYunRater {
	return &YinYunRater{weight: 0.20}
}

func NewYinYunRaterWithWeight(w float64) *YinYunRater {
	return &YinYunRater{weight: w}
}

func (r *YinYunRater) Name() string    { return "音韵" }
func (r *YinYunRater) Weight() float64 { return r.weight }

func (r *YinYunRater) Rate(candidate *NameCandidate, fateData *FateData) NameRating {
	score := 80.0
	var details []string

	p1 := candidate.Pinyin1
	p2 := candidate.Pinyin2
	sp := candidate.SurnamePinyin

	// 单名（p2 为空）：借姓氏拼音 + 单字做声调/声母/韵母搭配与谐音检测，
	// 恢复音韵维度区分度。历史缺陷：直接返回恒定 80 使单名 Top5 全部同分，
	// 荒谬字与好字并列，排序由候选池枚举顺序决定 → 门禁字永远打不完。
	if p1 == "" {
		return NameRating{Score: score, Detail: "声调起伏有致，韵母搭配和谐"}
	}

	if p2 == "" {
		if sp != "" {
			// 声调对比（单字 vs 姓氏）
			tone1 := getToneFromPinyin(p1)
			toneS := getToneFromPinyin(sp)
			if tone1 != 0 && toneS != 0 {
				if tone1 != toneS {
					score += 6
					details = append(details, "单字与姓氏声调不同，抑扬顿挫")
				} else {
					score -= 4
					details = append(details, "单字与姓氏声调相同")
				}
			}
			// 声母对比
			sm1 := getShengMu(p1)
			smS := getShengMu(sp)
			if sm1 != "" && smS != "" {
				if sm1 != smS {
					score += 4
					details = append(details, "单字与姓氏声母不同，发音清晰")
				} else {
					score -= 3
					details = append(details, "单字与姓氏声母相同")
				}
			}
			// 韵母对比（先归一化：生产数据为带声调符号拼音，如 ánɡ/ǎnɡ 实际都是 ang）
			ym1 := stripTone(getYunMu(p1))
			ymS := stripTone(getYunMu(sp))
			if ym1 != "" && ymS != "" {
				if ym1 != ymS {
					score += 3
					details = append(details, "单字与姓氏韵母不同，朗朗上口")
				} else {
					score -= 3
					details = append(details, "单字与姓氏韵母相同")
				}
			}
			// 姓氏+单字连读谐音检测（本字豁免：传入本字避免同音好字被误判）
			if hit, descs := CheckAllBadHomophones(sp, "", p1, candidate.Char1); hit {
				score -= float64(len(descs)) * 10
				details = append(details, "含不吉谐音: "+strings.Join(descs, "、"))
			}
		}
		score = clampScore(score)
		detail := "声调起伏有致，韵母搭配和谐"
		if len(details) > 0 {
			detail = strings.Join(details, "；")
		}
		return NameRating{Score: score, Detail: detail}
	}

	// 声调检查
	tone1 := getToneFromPinyin(p1)
	tone2 := getToneFromPinyin(p2)
	toneS := getToneFromPinyin(sp)

	if tone1 != 0 && tone2 != 0 {
		// 三字声调旋律评分（姓+名1+名2）
		// 平仄规律：1/2=平，3/4=仄；好名字讲究平仄交替
		if toneS != 0 {
			patternScore, patternDesc := evaluateTonePattern(toneS, tone1, tone2)
			score += patternScore
			details = append(details, patternDesc)
		} else {
			// 无姓氏拼音时退化为两字比较
			if tone1 != tone2 {
				score += 8
				details = append(details, "两字声调不同，抑扬顿挫")
			} else {
				score -= 5
				details = append(details, "两字声调相同")
			}
		}
	}

	// 声母检查（按发音部位分组梯度评分）
	sm1 := getShengMu(p1)
	sm2 := getShengMu(p2)
	if sm1 != "" && sm2 != "" {
		if simScore, simDesc := shengMuSimilarity(sm1, sm2); simDesc != "" {
			score += simScore
			details = append(details, simDesc)
		}
	}

	// 韵母检查（先归一化声调符号，ánɡ/ǎnɡ 均为 ang，避免同韵误判）
	ym1 := stripTone(getYunMu(p1))
	ym2 := stripTone(getYunMu(p2))
	if ym1 != ym2 && ym1 != "" && ym2 != "" {
		score += 4
		details = append(details, "韵母不同，朗朗上口")
	} else if ym1 == ym2 && ym1 != "" {
		score -= 3
	}

	// ——— 谐音检测（包含姓氏拼音，确保检测跨字谐音如"杜子腾"→肚子疼） ———

	// 1. 逐字检测不吉谐音（本字豁免：把每字的拼音与本字一一对应传入，
	//    避免"思"拼音 si 与"死"谐音混淆等常见好字被误判）
	pinyinCharPairs := []string{p1, candidate.Char1}
	if candidate.SurnamePinyin != "" {
		// 姓氏本字从 input 取不到（未传），用空字符串占位（不会触发本字豁免）
		pinyinCharPairs = append([]string{candidate.SurnamePinyin, ""}, pinyinCharPairs...)
	}
	if p2 != "" {
		pinyinCharPairs = append(pinyinCharPairs, p2, candidate.Char2)
	}
	if hit, descs := CheckAllBadHomophones(pinyinCharPairs...); hit {
		penalty := float64(len(descs)) * 10
		score -= penalty
		details = append(details, "含不吉谐音: "+strings.Join(descs, "、"))
	}

	// 2. 拼音连读不良组合检测（包含姓氏拼音）
	surnamePy := candidate.SurnamePinyin
	if hit, comboDesc := CheckBadPinyinCombo(surnamePy, p1, p2); hit {
		score -= 15
		details = append(details, comboDesc)
	}

	score = clampScore(score)
	detail := "声调起伏有致，韵母搭配和谐"
	if len(details) > 0 {
		detail = strings.Join(details, "；")
	}
	return NameRating{Score: score, Detail: detail}
}

// ShengXiaoRater 生肖匹配评分
// 考察名字五行与生肖五行的生克关系
type ShengXiaoRater struct {
	weight float64
}

func NewShengXiaoRater() *ShengXiaoRater {
	return &ShengXiaoRater{weight: 0.10}
}

func NewShengXiaoRaterWithWeight(w float64) *ShengXiaoRater {
	return &ShengXiaoRater{weight: w}
}

func (r *ShengXiaoRater) Name() string    { return "生肖" }
func (r *ShengXiaoRater) Weight() float64 { return r.weight }

func (r *ShengXiaoRater) Rate(candidate *NameCandidate, fateData *FateData) NameRating {
	if fateData == nil {
		return NameRating{Score: 80, Detail: "生肖与名字五行搭配和谐"}
	}

	score := 80.0
	var details []string
	zodiac := fateData.BaziInfo.Zodiac
	zodiacWx := getZodiacWuXing(zodiac)

	if zodiacWx == "" {
		return NameRating{Score: score, Detail: "生肖五行信息待完善"}
	}

	details = append(details, fmt.Sprintf("生肖%s，五行属%s", zodiac, zodiacWx))

	for _, pair := range []struct {
		char   string
		wuxing string
	}{{candidate.Char1, candidate.WuXing1}, {candidate.Char2, candidate.WuXing2}} {
		if pair.char == "" || pair.wuxing == "" {
			continue
		}
		if isWuXingSheng(zodiacWx, pair.wuxing) || isWuXingSheng(pair.wuxing, zodiacWx) {
			score += 7
			details = append(details, fmt.Sprintf("「%s」与生肖五行相生", pair.char))
		}
		if isWuXingKe(zodiacWx, pair.wuxing) || isWuXingKe(pair.wuxing, zodiacWx) {
			score -= 5
			details = append(details, fmt.Sprintf("「%s」与生肖五行相克", pair.char))
		}
	}

	score = clampScore(score)
	detail := "生肖五行搭配合理，无明显冲突"
	if len(details) > 0 {
		detail = strings.Join(details, "；")
	}
	return NameRating{Score: score, Detail: detail}
}

// SancaiRater 天地人三才评分
//
// 基于《易经·说卦传》三才理论：
//   - 天道曰阴与阳 → 笔画奇偶搭配（奇为阳，偶为阴）
//   - 地道曰柔与刚 → 五行生克关系（相生为顺）
//   - 人道曰仁与义 → 部首字形互补（不同部首代表多元文化内涵）
type SancaiRater struct {
	weight float64
}

func NewSancaiRater() *SancaiRater {
	return &SancaiRater{weight: 0.10}
}

func NewSancaiRaterWithWeight(w float64) *SancaiRater {
	return &SancaiRater{weight: w}
}

func (r *SancaiRater) Name() string    { return "三才" }
func (r *SancaiRater) Weight() float64 { return r.weight }

func (r *SancaiRater) Rate(candidate *NameCandidate, fateData *FateData) NameRating {
	score := 70.0
	var details []string

	// ——— 天道：阴阳（笔画奇偶搭配）———
	if candidate.Stroke1 > 0 && candidate.Stroke2 > 0 {
		yinYang1 := candidate.Stroke1 % 2 // 1=阳,0=阴
		yinYang2 := candidate.Stroke2 % 2
		if yinYang1 != yinYang2 {
			score += 10
			details = append(details, "天道阴阳调和：两字笔画一奇一偶")
		} else {
			score += 3
			details = append(details, "天道阴阳相谐：两字笔画同奇/同偶")
		}

		// 笔画匀称度
		diff := absInt(candidate.Stroke1 - candidate.Stroke2)
		if diff <= 5 {
			score += 5
			details = append(details, "笔画搭配匀称，刚柔相济")
		} else if diff <= 10 {
			score += 2
			details = append(details, "笔画差异适中")
		} else {
			score -= 3
			details = append(details, "笔画差异偏大")
		}
	}

	// ——— 地道：刚柔（五行生克关系）———
	if candidate.WuXing1 != "" && candidate.WuXing2 != "" {
		if isWuXingSheng(candidate.WuXing1, candidate.WuXing2) ||
			isWuXingSheng(candidate.WuXing2, candidate.WuXing1) {
			score += 10
			details = append(details, "地道刚柔相济：两字五行相生")
		} else if candidate.WuXing1 == candidate.WuXing2 {
			score += 5
			details = append(details, "地道同气连枝：两字五行相同")
		} else if isWuXingKe(candidate.WuXing1, candidate.WuXing2) ||
			isWuXingKe(candidate.WuXing2, candidate.WuXing1) {
			score -= 5
			details = append(details, "地道五行相克，需注意调和")
		} else {
			score += 3
			details = append(details, "地道五行平和")
		}
	}

	// ——— 人道：仁义（部首字形互补）———
	if candidate.Radical1 != "" && candidate.Radical2 != "" {
		if candidate.Radical1 != candidate.Radical2 {
			score += 5
			details = append(details, "人道多元互补：两字部首不同")
		} else {
			score += 2
			details = append(details, "人道一脉相承：两字部首相同")
		}
	}

	// 双字齐全加分（单名降级）——三才以双名为佳
	if candidate.Char2 != "" {
		score += 3
		details = append(details, "双名齐备，天地人三才俱全")

		// 总笔画区间评分（姓+名1+名2 总笔画）
		// 姓氏笔画通常 3-20，名字两字合计 8-30 为宜
		// 总笔画过少（≤10）显得单薄，过多（≥45）书写负担重
		if candidate.Stroke1 > 0 && candidate.Stroke2 > 0 {
			totalStroke := candidate.Stroke1 + candidate.Stroke2
			switch {
			case totalStroke <= 8:
				score -= 3
				details = append(details, fmt.Sprintf("名之总笔画%d画偏少，略显单薄", totalStroke))
			case totalStroke >= 10 && totalStroke <= 24:
				score += 4
				details = append(details, fmt.Sprintf("名之总笔画%d画，匀称得体", totalStroke))
			case totalStroke >= 25 && totalStroke <= 30:
				score += 1
				details = append(details, fmt.Sprintf("名之总笔画%d画，稍繁但可接受", totalStroke))
			case totalStroke >= 31:
				score -= 3
				details = append(details, fmt.Sprintf("名之总笔画%d画偏多，书写负担重", totalStroke))
			}
		}
	}

	// 单名（无第二字）笔画适中度检查
	// 背景：单名时 Stroke2==0，三才维度的阴阳/匀称/五行生克/部首互补全部跳过，
	// 历史上单名三才恒定 70 无区分度（与音韵恒定 80 并列构成单名全维同分的半壁）。
	// 此处以单字笔画适中度补充区分度：过简（≤4 画）或过繁（≥23 画）都不宜入名，
	// 与 AGENTS.md「避免笔画过于复杂名字」约束一致。
	if candidate.Char2 == "" && candidate.Stroke1 > 0 {
		switch {
		case candidate.Stroke1 <= 4:
			score -= 3
			details = append(details, "单字笔画过简，略显单薄")
		case candidate.Stroke1 >= 23:
			score -= 5
			details = append(details, "单字笔画过繁，书写负担重")
		case candidate.Stroke1 >= 6 && candidate.Stroke1 <= 18:
			score += 6
			details = append(details, "单字笔画适中，书写匀称")
		}
	}

	score = clampScore(score)
	detail := "天地人三才搭配均衡，阴阳刚柔相济"
	if len(details) > 0 {
		detail = strings.Join(details, "；")
	}
	return NameRating{Score: score, Detail: detail}
}

// --- 辅助函数 ---

func clampScore(s float64) float64 {
	if s > 100 {
		return 100
	}
	if s < 0 {
		return 0
	}
	return s
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// 五行生克表
var wuXingShengMap = map[string]string{
	"木": "火", "火": "土", "土": "金", "金": "水", "水": "木",
}

var wuXingKeMap = map[string]string{
	"木": "土", "土": "水", "水": "火", "火": "金", "金": "木",
}

func isWuXingSheng(a, b string) bool {
	if v, ok := wuXingShengMap[a]; ok {
		return v == b
	}
	return false
}

func isWuXingKe(a, b string) bool {
	if v, ok := wuXingKeMap[a]; ok {
		return v == b
	}
	return false
}

func getZodiacWuXing(zodiacName string) string {
	z := zodiac.GetZodiacByName(zodiacName)
	if z == nil {
		return ""
	}
	return z.Wuxing
}

// getToneFromPinyin 提取拼音声调（1/2/3/4），返回 0 表示无法识别
//
// 支持两种格式（生产数据 hanzi.json 为 Unicode 声调符号，测试常用数字后缀）：
//   - 数字后缀：wang2 → 2
//   - Unicode 声调符号：wáng → 2、tí → 2、shǎnɡ → 3
//
// 修复背景：旧实现只识别末尾数字，而生产数据全部是 Unicode 声调符号，
// 导致单名/双名音韵的声调维度全部失效（tone 恒为 0，比较被跳过）——
// 这是单名 Top5 长期全同分、荒谬字与好字并列的根因之一。
func getToneFromPinyin(pinyin string) int {
	pinyin = strings.TrimSpace(pinyin)
	if pinyin == "" {
		return 0
	}
	runes := []rune(pinyin)
	// 1. 数字后缀（wang2 → 2）
	last := runes[len(runes)-1]
	if last >= '1' && last <= '4' {
		return int(last - '0')
	}
	// 2. Unicode 声调符号（wáng → 2、tí → 2、shǎnɡ → 3）
	for _, r := range runes {
		switch r {
		case 'ā', 'ē', 'ī', 'ō', 'ū', 'ǖ':
			return 1
		case 'á', 'é', 'í', 'ó', 'ú', 'ǘ', 'ń':
			return 2
		case 'ǎ', 'ě', 'ǐ', 'ǒ', 'ǔ', 'ǚ', 'ň':
			return 3
		case 'à', 'è', 'ì', 'ò', 'ù', 'ǜ', 'ǹ':
			return 4
		}
	}
	return 0
}

var shengmuList = []string{"zh", "ch", "sh", "b", "p", "m", "f", "d", "t", "n", "l", "g", "k", "h", "j", "q", "x", "z", "c", "s", "r", "y", "w"}

func getShengMu(pinyin string) string {
	for _, sm := range shengmuList {
		if len(pinyin) >= len(sm) && pinyin[:len(sm)] == sm {
			return sm
		}
	}
	return ""
}

func getYunMu(pinyin string) string {
	sm := getShengMu(pinyin)
	if sm == "" {
		return pinyin
	}
	return pinyin[len(sm):]
}

// --- NoveltyRater 新颖度评分器 ---

// NoveltyRater 新颖度评分
// 考察名字组合的新颖程度：惩罚同义叠加、笔画失衡、常用套路组合
// 奖励：繁简搭配、语义差异化
type NoveltyRater struct {
	weight float64
}

func NewNoveltyRater() *NoveltyRater {
	return &NoveltyRater{weight: 0.12}
}

func NewNoveltyRaterWithWeight(w float64) *NoveltyRater {
	return &NoveltyRater{weight: w}
}

func (r *NoveltyRater) Name() string    { return "新颖度" }
func (r *NoveltyRater) Weight() float64 { return r.weight }

func (r *NoveltyRater) Rate(candidate *NameCandidate, fateData *FateData) NameRating {
	score := 75.0
	var details []string

	// 1. 同义叠加惩罚：两个字的 Meaning 有大量重叠字时扣分
	//    例如 "明"+"亮" 都含"明亮"义 → 语义冗余
	if candidate.Char1 != "" && candidate.Char2 != "" &&
		candidate.Meaning1 != "" && candidate.Meaning2 != "" {
		overlap := semanticOverlap(candidate.Meaning1, candidate.Meaning2)
		if overlap >= 3 {
			score -= float64(overlap) * 3
			details = append(details, fmt.Sprintf("语义重叠较多（-%d分）", overlap*3))
		} else if overlap >= 2 {
			score -= 2
			details = append(details, "语义略有重叠（-2分）")
		}
	}

	// 2. 笔画搭配评分
	if candidate.Stroke1 > 0 && candidate.Stroke2 > 0 {
		diff := absInt(candidate.Stroke1 - candidate.Stroke2)
		avg := (candidate.Stroke1 + candidate.Stroke2) / 2
		switch {
		case diff == 0:
			score -= 3
			details = append(details, "笔画相同，缺少变化（-3分）")
		case diff >= 1 && diff <= 3:
			score += 5
			details = append(details, "笔画搭配匀称（+5分）")
		case diff >= 4 && diff <= 8:
			score += 3
			details = append(details, "繁简有致（+3分）")
		default:
			score -= 2
			details = append(details, "笔画差异过大（-2分）")
		}
		// 两字都很简单或都很复杂
		if avg <= 5 {
			score -= 3
			details = append(details, "两字笔画均偏少（-3分）")
		} else if avg >= 20 {
			score -= 2
			details = append(details, "两字笔画均偏多（-2分）")
		}
	}

	// 3. 单名惩罚：单名天然缺少搭配变化
	if candidate.Char2 == "" {
		score -= 5
		details = append(details, "单名缺少字形搭配变化（-5分）")
	}

	// 4. 部首重复惩罚
	if candidate.Char1 != "" && candidate.Char2 != "" &&
		candidate.Radical1 != "" && candidate.Radical2 != "" {
		if candidate.Radical1 == candidate.Radical2 {
			score -= 4
			details = append(details, fmt.Sprintf("部首「%s」重复，缺少变化（-4分）", candidate.Radical1))
		}
	}

	// 5. 同音惩罚（声韵母完全相同）
	if candidate.Char1 != "" && candidate.Char2 != "" &&
		candidate.Pinyin1 != "" && candidate.Pinyin2 != "" {
		p1 := stripTone(candidate.Pinyin1)
		p2 := stripTone(candidate.Pinyin2)
		if p1 == p2 && p1 != "" {
			score -= 5
			details = append(details, fmt.Sprintf("两字同音「%s」，缺乏韵律变化（-5分）", p1))
		}
	}

	score = clampScore(score)
	detail := "字形搭配有变化，语义差异化良好"
	if len(details) > 0 {
		detail = strings.Join(details, "；")
	}
	return NameRating{Score: score, Detail: detail}
}

// semanticOverlap 计算两个字义字符串的重叠字数（排除标点/空格）
func semanticOverlap(a, b string) int {
	count := 0
	for _, r := range a {
		if strings.ContainsRune(b, r) && r != '，' && r != '、' && r != '。' &&
			r != ' ' && r != '；' && r != '—' {
			count++
		}
	}
	return count
}

// BigramRater 二字共现评分器
//
// 基于诗词经典字库的二字共现频率：两个字在同一诗句中出现的次数越多，
// 说明古人在用字时倾向于将它们搭配使用，名字组合越典雅自然。
//
// 评分逻辑（来自 classics.GetBigramScore 0-10 归一化）：
//   - freq >= 5 → 10 分（高频共现，如「关关雎鸠」的关+雎）
//   - freq >= 3 → 7 分
//   - freq >= 2 → 4 分
//   - freq == 1 → 2 分
//   - 未命中 → 0 分（机械凑合的组合）
//
// 映射到 0-100 维度分：score × 10 + 40 基础分
// 即 0 分（未命中）→ 40 分，10 分（高频）→ 100 分
// 单名时退化为单字诗词频率（HasPoetry）判断
type BigramRater struct {
	weight float64
}

func NewBigramRater() *BigramRater {
	return &BigramRater{weight: 0.10}
}

func NewBigramRaterWithWeight(w float64) *BigramRater {
	return &BigramRater{weight: w}
}

func (r *BigramRater) Name() string    { return "共现" }
func (r *BigramRater) Weight() float64 { return r.weight }

func (r *BigramRater) Rate(candidate *NameCandidate, fateData *FateData) NameRating {
	baseScore := 40.0
	var details []string

	if candidate.Char1 != "" && candidate.Char2 != "" {
		// 双名：查二字共现评分
		score, sourceDesc, found := classics.GetBigramScore(candidate.Char1, candidate.Char2)
		if found {
			dimension := baseScore + float64(score)*6.0 // 40 + 0~60 = 40~100
			dimension = clampScore(dimension)
			detail := fmt.Sprintf("共现频次评分 %d/10", score)
			if sourceDesc != "" {
				detail += "；" + sourceDesc
			}
			details = append(details, detail)
			return NameRating{Score: dimension, Detail: strings.Join(details, "；")}
		}
		// 未命中共现库，检查单字诗词频率作为降级信号
		if candidate.HasPoetry {
			dimension := baseScore + 10.0 // 50 分：有单字出典
			details = append(details, "单字有诗词出典（无二字共现）")
			return NameRating{Score: dimension, Detail: strings.Join(details, "；")}
		}
	} else if candidate.Char1 != "" {
		// 单名：退化为单字诗词频率
		if candidate.HasPoetry {
			dimension := baseScore + 10.0
			details = append(details, "单字有诗词出典")
			return NameRating{Score: dimension, Detail: strings.Join(details, "；")}
		}
	}

	// 无任何诗词信号
	return NameRating{Score: baseScore, Detail: "该组合暂无诗词典籍中的二字共现记录"}
}

// evaluateTonePattern 三字声调旋律评分（姓+名1+名2）
//
// 平仄分类：1/2 为平，3/4 为仄。
// 传统音韵学认为名字讲究平仄交替，有抑扬顿挫之美：
//   - 平平仄 / 平仄仄 / 仄平平 / 仄仄平 → 有变化，好
//   - 平平平 / 仄仄仄 → 三连同调，单调
//   - 尾字声调决定整体收束感：仄声收束（3/4）有力度，平声收束（1/2）有余韵
func evaluateTonePattern(toneS, tone1, tone2 int) (float64, string) {
	ping := func(t int) bool { return t == 1 || t == 2 } // 平声

	sP, c1P, c2P := ping(toneS), ping(tone1), ping(tone2)
	flat := func(p bool) string {
		if p {
			return "平"
		}
		return "仄"
	}

	pattern := fmt.Sprintf("%s%s%s", flat(sP), flat(c1P), flat(c2P))

	// 统计平/仄个数
	pingCount := 0
	if sP {
		pingCount++
	}
	if c1P {
		pingCount++
	}
	if c2P {
		pingCount++
	}
	zeCount := 3 - pingCount

	switch {
	case pingCount == 3:
		return -8, "三平相连，声调单调"
	case zeCount == 3:
		return -6, "三仄相连，声调沉闷"
	case c1P != c2P:
		// 名字两字平仄交替：最佳组合
		if sP != c1P {
			// 姓与名1也交替：三字全交替，最优
			return 12, fmt.Sprintf("声调旋律%s，三字平仄交替", pattern)
		}
		return 8, fmt.Sprintf("声调旋律%s，名字平仄交替", pattern)
	case c1P == c2P && sP != c1P:
		// 名字同调但与姓不同：姓平名仄或姓仄名平
		return 3, fmt.Sprintf("声调旋律%s，姓与名声调对比", pattern)
	default:
		// 名字同调且姓与名1同调：部分重叠
		return -2, fmt.Sprintf("声调旋律%s，部分声调重叠", pattern)
	}
}

// shengMuGroup 声母按发音部位分组
//
// 同组声母（如 b/p 都是双唇音）连读时口型变化小，听感模糊；
// 不同组声母（如 x/zh）连读时口型变化大，发音清晰。
var shengMuGroups = map[string]int{
	"b": 1, "p": 1, "m": 1, // 双唇音
	"f": 2,                         // 唇齿音
	"d": 3, "t": 3, "n": 3, "l": 3, // 舌尖中音
	"g": 4, "k": 4, "h": 4, // 舌根音
	"j": 5, "q": 5, "x": 5, // 舌面音
	"zh": 6, "ch": 6, "sh": 6, "r": 6, // 翘舌音
	"z": 7, "c": 7, "s": 7, // 平舌音
	"y": 8, "w": 8, // 零声母/半元音
}

// shengMuSimilarity 声母相似度评分
//
// 返回值范围 [-5, 5]：
//   - 同组 → -5（口型变化小，听感模糊）
//   - 相邻组（发音部位差1）→ 2（有变化但过渡自然）
//   - 远隔组（差≥3）→ 5（口型变化大，发音清晰）
func shengMuSimilarity(sm1, sm2 string) (float64, string) {
	g1, ok1 := shengMuGroups[sm1]
	g2, ok2 := shengMuGroups[sm2]
	if !ok1 || !ok2 {
		return 0, ""
	}
	if g1 == g2 {
		return -5, fmt.Sprintf("声母「%s/%s」同属%s音，听感模糊", sm1, sm2, shengMuGroupName(g1))
	}
	diff := g1 - g2
	if diff < 0 {
		diff = -diff
	}
	if diff == 1 {
		return 2, fmt.Sprintf("声母「%s/%s」发音部位相邻，过渡自然", sm1, sm2)
	}
	return 5, fmt.Sprintf("声母「%s/%s」发音部位差异大，吐字清晰", sm1, sm2)
}

// shengMuGroupName 声母分组中文名
func shengMuGroupName(group int) string {
	names := map[int]string{
		1: "双唇", 2: "唇齿", 3: "舌尖中", 4: "舌根",
		5: "舌面", 6: "翘舌", 7: "平舌", 8: "零声母",
	}
	if n, ok := names[group]; ok {
		return n
	}
	return "未知"
}
