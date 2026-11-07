package fate

import (
	"math"
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
// 「含精选好字」的豁免另有前置条件：两个字都必须具备命名依据
// （hasNamingEvidence：寓意评分 / 策展分类 / 真实人名语料），详见 nonCuratedCapApplies。
//
// 效果：荒谬组合四维封顶 75 后总分上限 = 五行85×0.30 + 75×0.70 = 78，
// 必然低于策展好名；而优质非策展名凭五行匹配度（不封顶）拉开差距。
func RateName(candidate *NameCandidate, fateData *FateData, raters []Rater) NameScore {
	// 本函数始终返回完整的 Items/Details：如候选来自穷举热路径（skipDetail 为真），
	// 这里临时关闭该开关，让各 Rater 正常构造依据文案（调用后原样还原）。
	prevSkip := candidate.skipDetail
	candidate.skipDetail = false
	defer func() { candidate.skipDetail = prevSkip }()

	items := make(map[string]float64, len(raters))
	details := make(map[string]string, len(raters))
	var total float64

	for _, r := range raters {
		rating := r.Rate(candidate, fateData)
		items[r.Name()] = rating.Score
		details[r.Name()] = rating.Detail
		total += rating.Score * r.Weight()
	}

	// 方案B+：非策展双名（单名无策展概念，不封顶）
	//
	// 豁免条件扩展：除策展好名白名单外，「含精选好字」的组合
	// （IsCuratedN && PositiveScoreN>=90，与 WenHuaRater +8 加分同条件）同样豁免。
	// 否则 WenHuaRater 给精选好字的文化加分会被本封顶截回 75，策展加分机制被架空——
	// 实测喜用神收窄到单五行后白名单组合全部出局，「已谦」（谦=策展∩ps87）与荒谬
	// 组合同聚封顶基准分 69.3（=五行56×0.30+75×0.70），Top50 平均分断言失败。
	if nonCuratedCapApplies(candidate) {
		for k := range cappedDimNames {
			dim := cappedDimNames[k]
			if items[dim] > 75 {
				total -= (items[dim] - 75) * weightOf(raters, dim)
				items[dim] = 75
			}
		}
	}

	// 限制总分在 0-100 范围内
	total = clampScore(total)

	// 四舍五入保留一位小数
	total = math.Round(total*10) / 10

	return NameScore{
		Total:   total,
		Grade:   scoreToGrade(total),
		Items:   items,
		Details: details,
	}
}

// cappedDimNames 非策展双名封顶的四个维度名（与 WenHua/YinYun/ShengXiao/Sancai 对应）
var cappedDimNames = [4]string{"文化印象", "音韵", "生肖", "三才"}

// nonCuratedCapApplies 判定「非策展双名四维封顶」是否适用
//
// 单名无策展概念（Char2 为空）不封顶；策展白名单配对豁免封顶。
//
// 此外「精选好字」触发豁免时，还要求**两个字都具备命名依据**（见 hasNamingEvidence）。
// 这是一处必要的收紧：原实现只要有一个字是精选好字就整组豁免，而 positiveScore
// 只覆盖少量汉字、未覆盖的字（含大量荒谬字）在评分里得不到任何负反馈、与优质字
// 完全同分，于是「精选好字 + 任意字」直接屠榜——实测双名 Top10 有 10/10 是该形态
// （张沚明/张鲛慧/张唣明/张慧僰/张恃泽/张蚂泽/张浩荥/张噬鹏/张蚂宏/张蚂清：
// 好字全是 明/慧/泽/浩/鹏/宏/清，伙伴字全是无信号字）。
//
// 收紧后：两个有命名依据的字搭配精选好字仍可豁免，保留推荐多样性；
// 「好字 + 无依据字」一律回落封顶上限，无法再靠单字拉动整组。
//
// 抽成独立函数是为了让 RateName（全量明细）与 RateNameScore（仅总分）共用同一判据，
// 保证两条路径算出的总分完全一致。
func nonCuratedCapApplies(candidate *NameCandidate) bool {
	if candidate.Char1 == "" || candidate.Char2 == "" {
		return false
	}
	if IsCuratedName(candidate.Char1, candidate.Char2) {
		return false
	}
	premium := (candidate.IsCurated1 && candidate.PositiveScore1 >= 90) ||
		(candidate.IsCurated2 && candidate.PositiveScore2 >= 90)
	if !premium {
		return true
	}
	return !hasNamingEvidence(candidate.IsCurated1, candidate.PositiveScore1, candidate.NameFreqTier1) ||
		!hasNamingEvidence(candidate.IsCurated2, candidate.PositiveScore2, candidate.NameFreqTier2)
}

// hasNamingEvidence 判定单字是否具备「命名依据」
//
// 三个来源任一即可：
//   - 人工寓意评分 positiveScore > 0
//   - 策展起名分类字（IsCurated）
//   - 出现在 95.7 万条真实人名语料中（NameFreqTier > 0）
//
// 注意这是「有没有依据」而非「好不好」——它不参与打分，只用于判断一个字是否有
// 资格参与「豁免封顶」的组队，避免评分体系的信号盲区（positiveScore 覆盖不足）
// 让无信号字白嫖精选好字的豁免。
func hasNamingEvidence(isCurated bool, positiveScore, nameFreqTier int) bool {
	return isCurated || positiveScore > 0 || nameFreqTier > 0
}

// weightOf 按维度名取权重（仅在封顶分支调用，至多 4 次；避免为此构造 map）
func weightOf(raters []Rater, dim string) float64 {
	for _, r := range raters {
		if r.Name() == dim {
			return r.Weight()
		}
	}
	return 0
}

// RateNameScore 只计算综合总分与等级，不构造任何维度明细与依据文案。
//
// 专供双名 N² 枚举热路径：这一阶段每个组合都要评分，但只有约千分之一的
// 组合最终会进入推荐榜并需要展示明细（Items/Details）。相比 RateName 省去：
//   - 两个 map 的分配与逐维度写入
//   - 各 Rater 的 fmt.Sprintf / strings.Join 文案构造（实测约占 22%）及其 GC 压力
//
// 总分与 RateName 完全一致（共用同一封顶判据与取整逻辑），
// 由 TestRateNameScoreMatchesRateName 断言保证。
// 调用方应在条目真正入榜后再用 RateName 回算明细。
func RateNameScore(candidate *NameCandidate, fateData *FateData, raters []Rater) float64 {
	prevSkip := candidate.skipDetail
	candidate.skipDetail = true
	defer func() { candidate.skipDetail = prevSkip }()

	var total float64
	var capScore, capWeight [len(cappedDimNames)]float64

	for _, r := range raters {
		name := r.Name()
		w := r.Weight()
		rating := r.Rate(candidate, fateData)
		total += rating.Score * w
		for k := range cappedDimNames {
			if name == cappedDimNames[k] {
				capScore[k] = rating.Score
				capWeight[k] = w
				break
			}
		}
	}

	if nonCuratedCapApplies(candidate) {
		for k := range cappedDimNames {
			if capScore[k] > 75 {
				total -= (capScore[k] - 75) * capWeight[k]
			}
		}
	}

	total = clampScore(total)
	return math.Round(total*10) / 10
}

// ——— 粤语评分聚合（独立于普通话九维体系）———
//
// 粤语分由 YueYinYunRater（粤语音律）+ YueHomophoneRater（粤语谐音）两项聚合成，
// 经 RateCantonese / RateCantoneseScore 计算。两项权重在 DefaultCantoneseRaters
// 内自行归一（和为 1.0），与普通话九维的权重体系互不干扰。
//
// 设计要点：
//   - 粤语维度不进入 cappedDimNames 封顶集合——封顶是为了压制「非策展双名」的机械
//     凑合组合，而粤语分是平行展示维度，参与封顶会扭曲其独立含义。
//   - RateCantonese 与 RateCantoneseScore 共用同一聚合逻辑，由 TestRateCantoneseScoreMatches
//     钉住一致性（与普通话 RateName/RateNameScore 的契约同构）。

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
