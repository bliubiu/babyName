package fate

import (
	"fmt"
	"strings"

	"name/internal/domain/classics"
)

// WenHuaRater 文化印象评分
// 考察常用度、字义丰富度、笔画匀称度
type WenHuaRater struct {
	weight float64

	// 经典来源偏好加分
	// sourceName  用户选择的经典来源名（论语/诗经…，用于 details 展示）
	// sourceSet   所选来源的提取字集（nil 表示未指定来源，不做来源加分）
	// 由 session.Start 按 per-request 的 Options.SourceClassic 装配（见 engine.go）。
	sourceName string
	sourceSet  map[string]bool
}

func NewWenHuaRater() *WenHuaRater {
	return &WenHuaRater{weight: 0.20}
}

func NewWenHuaRaterWithWeight(w float64) *WenHuaRater {
	return &WenHuaRater{weight: w}
}

// NewWenHuaRaterWithSource 创建带经典来源偏好的文化印象评分器。
// source 支持中文（论语/诗经…）与拼音（lunyu/shijing…）别名（classics.sourceAlias 归一化）。
// Rate 时命中所选来源提取字集的汉字每字 +5 分，引导结果偏向所选经典。
func NewWenHuaRaterWithSource(w float64, source string) *WenHuaRater {
	r := &WenHuaRater{weight: w, sourceName: source}
	chars := classics.GetPoetryCharList(source)
	if len(chars) > 0 {
		r.sourceSet = make(map[string]bool, len(chars))
		for _, pc := range chars {
			if pc.Char != "" {
				r.sourceSet[pc.Char] = true
			}
		}
	}
	return r
}

func (r *WenHuaRater) Name() string    { return "文化印象" }
func (r *WenHuaRater) Weight() float64 { return r.weight }

func (r *WenHuaRater) Rate(candidate *NameCandidate, fateData *FateData) NameRating {
	score := 60.0
	ds := newDetailSink(candidate)

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
			ds.add("笔画搭配匀称")
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
	// 也在表内拿到 +8，导致 Top5 被平庸字霸榜。结合 PositiveScore>=90（namer.json 寓意评分，
	// 优质字 87-91 有值、平庸/荒谬字为空）把加分收窄为「策展 ∩ positiveScore>=90」精选好字专属。
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
		ds.addf("「%s」为策展起名好字（+%d分）",
			strings.Join(curatedChars, "、"), curatedCount*8)
	}

	// ——— 诗词出处加分（含近义语义扩展） ———

	// 精确匹配（由 name generator 预计算）
	if candidate.HasPoetry {
		score += 8
		// 文案拼接本身会分配，先判 skip 再构造
		if !ds.skip {
			if candidate.PoetryFrom != "" {
				ds.add("出自" + candidate.PoetryFrom)
			} else {
				ds.add("出自诗词典故")
			}
		}
	}

	// 语义扩展匹配（精确匹配未覆盖时检查近义字关联）
	// wantDesc=!ds.skip：跳过热路径上无谓的文案格式化，仅在需要时构造
	if !candidate.HasPoetry {
		if candidate.Char1 != "" {
			if ok, desc := checkSemanticPoetry(candidate.Char1, !ds.skip); ok {
				score += 5
				ds.add(desc)
			}
		}
		if candidate.Char2 != "" {
			if ok, desc := checkSemanticPoetry(candidate.Char2, !ds.skip); ok {
				score += 5
				ds.add(desc)
			}
		}
	}

	// ——— 经典来源偏好加分 ———
	// 用户显式选择经典来源（Options.SourceClassic：论语/诗经…）时，命中所选来源
	// 提取字集的字每字 +5 分，引导结果偏向所选经典。这是"引导"而非"背书"：
	// 分值低于策展好字（+8）与单字出典（+8），仅凭加分不足以让劣质来源字霸榜。
	// 未指定来源时 sourceSet 为空，循环零开销（DefaultRaters 默认构造不受影响）。
	// 与 resolveExtraChars 的候选池注入联动：注入让论语字"进得去"，加分让它们"排得上"。
	if len(r.sourceSet) > 0 {
		for _, ch := range []string{candidate.Char1, candidate.Char2} {
			if ch != "" && r.sourceSet[ch] {
				score += 5
				ds.addf("「%s」来自【%s】选字（+5分）", ch, r.sourceName)
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
		// bigramCache 注入时优先走 cache，未注入时走慢路径（classics.GetBigramScore）
		var bgScore int
		var bgSrc string
		var found bool
		if candidate.bigramCache != nil {
			res := candidate.bigramCache.GetOrCompute(candidate.Char1, candidate.Char2)
			bgScore, bgSrc, found = res.Score, res.SourceDesc, res.Found
		} else {
			bgScore, bgSrc, found = classics.GetBigramScore(candidate.Char1, candidate.Char2)
		}
		if found {
			lvlOK := candidate.CommonLevel1 >= 1 && candidate.CommonLevel1 <= 2 &&
				candidate.CommonLevel2 >= 1 && candidate.CommonLevel2 <= 2
			if IsCuratedName(candidate.Char1, candidate.Char2) &&
				!IsNonNamingChar(candidate.Char1) && !IsNonNamingChar(candidate.Char2) && lvlOK {
				score += float64(bgScore)
				ds.addf("「%s%s」共现于%s（+%d分）",
					candidate.Char1, candidate.Char2, bgSrc, bgScore)
			} else {
				// 未获策展认可 / 含门禁字 / 含生僻字：共现分彻底归零（方案B+）
				ds.addf("「%s%s」共现于%s，未获策展认可，共现分归零",
					candidate.Char1, candidate.Char2, bgSrc)
			}
		}
	}

	// 策展好名加成已移除：策展库（curated_names.json）不可作为质量基准
	// （其本身是历史人物名采集，混入「仲尼/若兮/七政/与砺」等 962 条垃圾，见探查结论）。
	// 名字搭配质量由「通用规范汉字表一二级 + 语义门禁」独立判定。

	// 单名语义共现：Char1 的近义字与 Char1 在诗词中的搭配
	if candidate.Char2 == "" && candidate.Char1 != "" {
		if ok, desc := checkSingleNameBigram(candidate.Char1, !ds.skip); ok {
			score += 3
			ds.add(desc)
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
		ds.addf("含无命名价值字「%s」（-12分/字）",
			strings.Join(penalizedChars, "、"))
	}
	if len(rareChars) > 0 {
		score -= float64(len(rareChars)) * 8
		ds.addf("含生僻/表外字「%s」（-8分/字）",
			strings.Join(rareChars, "、"))
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
			ds.addf("「%s%s」以病祈福，寓意祛病安康（+6分）",
				candidate.Char1, candidate.Char2)
		} else {
			score -= float64(softCount) * 8
			ds.addf("含消极字「%s」，寓意需斟酌（-8分/字）",
				strings.Join(negChars, "、"))
		}
	}

	// ——— 数据层策展扣分（NamePenalty） ———
	// hanzi.json 对部分字人工标注了起名扣分（如 响15/荒12/晦12/虚12/昧12/冥10 等），
	// 表示这些字作为名字有明显减分项，评分阶段按比例扣减，使「至律/值接」类
	// 仅靠典籍共现推高的组合无法进入推荐榜。
	if p := candidate.NamePenalty1 + candidate.NamePenalty2; p > 0 {
		deduct := float64(p) * 0.6
		score -= deduct
		ds.addf("策展标注起名扣分（-%.1f分）", deduct)
	}

	score = clampScore(score)
	detail := ds.String("常用字搭配，字义明确，文化内涵良好")
	return NameRating{Score: score, Detail: detail}
}

// checkSemanticPoetry 检查字符是否有近义关联的诗词出处
//
// 返回 (是否命中, 描述文字)。wantDesc 为假时不构造描述文字（穷举热路径），
// 只返回是否命中供调用方计分。
func checkSemanticPoetry(char string, wantDesc bool) (bool, string) {
	_, synChar, entries := classics.FindPoetryByCharSemantic(char)
	if synChar == "" || len(entries) == 0 {
		return false, ""
	}
	if !wantDesc {
		return true, ""
	}
	entry := entries[0]
	src := entry.Sentence
	if src == "" {
		src = entry.Work + "·" + entry.Chapter
	}
	return true, fmt.Sprintf("「%s」近义于「%s」，关联「%s」", char, synChar, src)
}

// checkSingleNameBigram 检查单名是否通过近义字获得诗词共现加分
//
// 返回 (是否命中, 描述文字)；wantDesc 为假时不构造描述文字（穷举热路径）。
func checkSingleNameBigram(char string, wantDesc bool) (bool, string) {
	synChar, coChar, src, found := classics.GetSemanticBigramMatch(char)
	if !found {
		return false, ""
	}
	if !wantDesc {
		return true, ""
	}
	return true, fmt.Sprintf("「%s」借「%s·%s」共现于%s（+3分）", char, synChar, coChar, src)
}
