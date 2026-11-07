package fate

import (
	"context"
	"math"
	"strings"
	"sync"
	"sync/atomic"

	"name/internal/domain/yue"
)

// singleNameCandidate 组装单名的 NameCandidate
//
// 字段与历史上两处内联构造逐一对齐（单名不注入 Meaning1/Radical/GenderHint/
// bigramCache，以免无意改变 WenHuaRater 的「字义明确 +4」等分支结果）。
// 抽成函数是为了让「枚举期评分」与「入榜后回算明细」走同一条构造路径。
func singleNameCandidate(a charInfo, surnamePinyin, surnameJyutping string) *NameCandidate {
	return &NameCandidate{
		Char1:          a.ch.Char,
		Pinyin1:        a.pinyin,
		WuXing1:        a.ch.WuXing,
		Stroke1:        a.stroke,
		KangxiStroke1:  kangxiStrokeOf(a.ch),
		HasPoetry:      a.poetryFound,
		PoetryFrom:     a.poetryDesc,
		IsRegular:      a.ch.IsRegular,
		CommonLevel1:   a.ch.CommonLevel,
		NameFreqTier1:  a.ch.NameFreqTier,
		NamePenalty1:   a.ch.NamePenalty,
		IsCurated1:     a.ch.IsCurated,
		PositiveScore1: a.ch.PositiveScore,
		SurnamePinyin:  surnamePinyin,
		// 粤拼（首次读音，未收录为空串）：供粤语音律分与粤语谐音风险分使用，并透出展示。
		Jyutping1:       yue.PrimaryJyutping(a.ch.Char),
		SurnameJyutping: surnameJyutping,
		// 释义画像：单名无名2，仅注入字1画像（NoveltyRater 单名分支不比较重叠）
		MeaningProfile1: a.meaningProfile,
	}
}

// doubleNameCandidate 组装双名的 NameCandidate
//
// poetryFound/poetryDesc 由调用方按「甲字||乙字任一有出典」预先合并后传入
// （与引擎内层循环的既有语义一致）。
func doubleNameCandidate(
	a, b charInfo, poetryFound bool, poetryDesc, surnamePinyin, surnameJyutping string, bigramCache *SessionBigramCache,
) *NameCandidate {
	return &NameCandidate{
		Char1:   a.ch.Char,
		Char2:   b.ch.Char,
		Pinyin1: a.pinyin,
		Pinyin2: b.pinyin,
		WuXing1: a.ch.WuXing,
		WuXing2: b.ch.WuXing,
		Stroke1: a.stroke,
		Stroke2: b.stroke,
		// 康熙笔画（未收录降级科学笔画）：供卦象评分器按统一口径起卦（docs/42 P3-12）
		KangxiStroke1: kangxiStrokeOf(a.ch),
		KangxiStroke2: kangxiStrokeOf(b.ch),
		Meaning1:      a.ch.Meaning,
		Meaning2:      b.ch.Meaning,
		Radical1:      a.ch.Radical,
		Radical2:      b.ch.Radical,
		HasPoetry:     poetryFound,
		PoetryFrom:    poetryDesc,
		IsRegular:     a.ch.IsRegular && b.ch.IsRegular,
		CommonLevel1:  a.ch.CommonLevel,
		CommonLevel2:  b.ch.CommonLevel,
		NameFreqTier1: a.ch.NameFreqTier,
		NameFreqTier2: b.ch.NameFreqTier,
		GenderHint:    bestGenderHint(a.ch.GenderHint, b.ch.GenderHint),
		NamePenalty1:  a.ch.NamePenalty,
		NamePenalty2:  b.ch.NamePenalty,
		// 策展覆盖表标记：人工精选起名好字，供 WenHuaRater 文化加分（破荒谬字同分）
		IsCurated1: a.ch.IsCurated,
		IsCurated2: b.ch.IsCurated,
		// 寓意评分：与 IsCurated 结合将策展加分收窄为「策展 ∩ positiveScore>=90」精选好字
		PositiveScore1: a.ch.PositiveScore,
		PositiveScore2: b.ch.PositiveScore,
		// 姓氏拼音取自 input，用于音韵评分器检测跨字谐音
		SurnamePinyin: surnamePinyin,
		// 粤拼（首次读音，未收录为空串）：供粤语音律分与粤语谐音风险分使用，并透出展示。
		Jyutping1:       yue.PrimaryJyutping(a.ch.Char),
		Jyutping2:       yue.PrimaryJyutping(b.ch.Char),
		SurnameJyutping: surnameJyutping,
		// 释义画像：按候选字预计算，消除 N² 次释义字符串扫描
		MeaningProfile1: a.meaningProfile,
		MeaningProfile2: b.meaningProfile,
		// bigramCache per-session 缓存（避免 WenHuaRater/BigramRater
		// 在 N² 笛卡尔积中重复 50 万次 GetBigramScore RLock）
		bigramCache: bigramCache,
	}
}

// fillEntryDetails 为进入推荐榜的条目回算各维度得分与依据文案
//
// 评分与解释分离：枚举阶段只算总分（RateNameScore），Items/Details 留空；
// 本函数按条目携带的候选字下标（idx1/idx2）从 infos 还原 NameCandidate，
// 再调用完整 RateName 回算一次明细，填回条目供服务层透传。
//
// 代价可忽略：条目数 ≤ topCount*10（约 500~1000），相对 N² 枚举量级约千分之一。
// 回算结果的总分与枚举期一致（两条路径共用封顶判据与取整逻辑，
// 由 TestRateNameScoreMatchesRateName 断言保证）。
// blendScore 计算候选名的最终排序得分，并透出粤语子分。
//
// 开启粤语评分时，把粤语分（RateCantoneseScore）按固定权重 CantoneseBlendWeight
// 并入主分，使粤语音律/谐音对排序有轻量但可见的影响；关闭时原样返回主分（零影响）。
// 返回的 canto 为粤语分（0-100），供 ExcellentEntry.CantoneseTotal 透传展示。
func (s *sessionImpl) blendScore(cand *NameCandidate, fateData *FateData) (finalScore float64, canto float64) {
	mainTotal := RateNameScore(cand, fateData, s.raters)
	if !s.enableCantonese || len(s.cantoneseRaters) == 0 {
		return mainTotal, 0
	}
	canto = RateCantoneseScore(cand, fateData, s.cantoneseRaters)
	blended := mainTotal*(1-CantoneseBlendWeight) + canto*CantoneseBlendWeight
	return math.Round(blended*10) / 10, canto
}

func (s *sessionImpl) fillEntryDetails(
	entries []ExcellentEntry, infos []charInfo, fateData *FateData, surnamePinyin string,
) {
	for k := range entries {
		e := &entries[k]
		if e.idx1 < 0 || e.idx1 >= len(infos) {
			continue
		}
		a := infos[e.idx1]
		var cand *NameCandidate
		if e.idx2 >= 0 && e.idx2 < len(infos) {
			b := infos[e.idx2]
			poetryFound := a.poetryFound || b.poetryFound
			poetryDesc := ""
			if a.poetryFound {
				poetryDesc = a.poetryDesc
			} else if b.poetryFound {
				poetryDesc = b.poetryDesc
			}
			cand = doubleNameCandidate(a, b, poetryFound, poetryDesc, surnamePinyin, s.surnameJyutping, s.bigramCache)
		} else {
			cand = singleNameCandidate(a, surnamePinyin, s.surnameJyutping)
		}
		ns := RateName(cand, fateData, s.raters)
		e.Items = ns.Items
		e.Details = ns.Details
		// 粤拼标注：始终回填（供前端展示，与粤语评分开关无关）
		e.Jyutping1 = cand.Jyutping1
		e.Jyutping2 = cand.Jyutping2
		e.SurnameJyutping = cand.SurnameJyutping
		// 粤语分明细：仅在开启粤语评分时回算（粤语音律 + 粤语谐音两项）。
		// 总分在枚举期已按 CantoneseBlendWeight 并入 e.Score，此处补充分项文案。
		if s.enableCantonese && len(s.cantoneseRaters) > 0 {
			cs := RateCantonese(cand, fateData, s.cantoneseRaters)
			e.CantoneseTotal = cs.Total
			e.CantoneseItems = cs.Items
			e.CantoneseDetails = cs.Details
		}
	}
}

// generateSingleName 单名生成：串行迭代候选字
//
// 候选集较小（~1100），串行足够快，无需并发。
// 关键差异（vs 双名）：
//   - 单层循环（无 i×j 笛卡尔积）
//   - IsNonNamingChar 兜底（单名字是名字的全部，门禁字必须剔除）
//   - 无 IsBadCombo / IsHistoricalFigureCombo（单字无组合级过滤）
//   - 无早停（候选集小，全枚举 + RateName 性能可接受）
//
// generateWithGeneration 字辈固定槽位 × 自由枚举（generate 的字辈分支）
//
// 语义（engine.go:700 注释所述分支的完整实现）：
//   - 字辈字固定占一个槽位（GenerationPosition：middle=名的第一个字，默认；
//     tail=名的最后一个字），其余槽位从候选池自由枚举；
//   - handled=false（回退常规枚举）只发生在字辈字不合法时：
//     多字、候选池未命中、门禁字或单笔画过滤拒绝；
//   - 成功时全部候选已写入 table 并 Finalize，调用方不再走
//     generateSingleName / generateDoubleName。
//
// 组合级过滤与评分口径完全复用 generateDoubleName 内循环的一套门禁
// （CheckStrokePair / isComboExcluded / IsBadCombo / IsHistoricalFigureCombo /
// 门禁字 / PairBlacklist），保证字辈路径与常规路径的评分与淘汰口径一致；
// 枚举期只算总分不构造明细文案（与常规路径同一约定）。
//
// 串行执行：固定字 × 候选池 ≤ 8105 组合，单请求规模可控，不再引入分片并发。
func (s *sessionImpl) generateWithGeneration(
	ctx context.Context,
	input *Input,
	infos []charInfo,
	table *ExcellentTable,
	totalCount *atomic.Int64,
	fateData *FateData,
	surnamePinyin string,
) ([]charInfo, bool) {
	genRunes := []rune(strings.TrimSpace(input.Generation))
	if len(genRunes) != 1 {
		return infos, false
	}
	genChar := string(genRunes[0])

	// 字辈字必须命中候选池（才能拿到笔画/拼音/诗典/释义画像），查不到即回退
	genIdx := -1
	for i := range infos {
		if infos[i].ch.Char == genChar {
			genIdx = i
			break
		}
	}
	if genIdx < 0 {
		return infos, false
	}
	genInfo := infos[genIdx]

	// 字辈字本身是门禁字（虚词/排行字/口语物名/数字量词等）无命名价值，
	// 按约定回退常规枚举（常规路径也会把该字从候选池剔除，行为一致）。
	if IsNonNamingChar(genChar) {
		return infos, false
	}

	s.setStage(sessionStageScore, 30)

	nameLen := input.Options.NameLength
	if nameLen <= 0 {
		nameLen = 2
	}

	if nameLen == 1 {
		// 单名 + 字辈：字辈字即是名的固定内容，无自由槽位
		if !s.filter.CheckStrokePair(genInfo.stroke, 0) {
			return infos, false
		}
		cand := singleNameCandidate(genInfo, surnamePinyin, s.surnameJyutping)
		final, canto := s.blendScore(cand, fateData)
		table.TryPush(ExcellentEntry{
			Char1:          genInfo.ch.Char,
			Pinyin1:        genInfo.pinyin,
			Meaning1:       genInfo.ch.Meaning,
			Score:          final,
			Grade:          scoreToGrade(final),
			WuXing1:        genInfo.ch.WuXing,
			Stroke1:        genInfo.stroke,
			KangxiStroke1:  genInfo.ch.KangxiStroke,
			HasPoetry:      genInfo.poetryFound,
			PoetryFrom:     genInfo.poetryDesc,
			NameFreqTier1:  genInfo.ch.NameFreqTier,
			CantoneseTotal: canto,
			idx1:           genIdx,
			idx2:           -1,
		})
		totalCount.Add(1)
		table.Finalize()
		return infos, true
	}

	// 双名：字辈固定占一个槽位，其余槽位从候选池自由枚举
	genFirst := input.GenerationPosition != "tail"
	pool := float64(max(len(infos), 1))
	done := 0
	for j := range infos {
		if cancelled(ctx) {
			break
		}
		// 双名不用重复字：字辈字已固定，另一槽位须为不同字
		if infos[j].ch.Char == genChar {
			continue
		}

		var a, b charInfo
		if genFirst {
			a, b = genInfo, infos[j]
		} else {
			a, b = infos[j], genInfo
		}

		if !s.filter.CheckStrokePair(a.stroke, b.stroke) {
			continue
		}

		// 负面反馈：排除组合（读取不可变快照，无锁）
		if s.isComboExcluded(a.ch.Char, b.ch.Char) {
			continue
		}

		// 负面语义：组合禁忌（亲属称谓/物名/动词等）
		if IsBadCombo(a.ch.Char, b.ch.Char) {
			continue
		}

		// 历史人物字/号专名：剔除与历史人物撞车的组合
		if IsHistoricalFigureCombo(a.ch.Char, b.ch.Char) {
			continue
		}

		// 名字用字质量门禁（纯语义）：任一位置为门禁字即剔除
		if IsNonNamingChar(a.ch.Char) || IsNonNamingChar(b.ch.Char) {
			continue
		}

		// 数据层搭配黑名单：策展标注的 PairBlacklist 双向剔除
		if hasPairBlacklist(a.ch, b.ch.Char) || hasPairBlacklist(b.ch, a.ch.Char) {
			continue
		}

		poetryFound := a.poetryFound || b.poetryFound
		poetryDesc := ""
		if poetryFound {
			if a.poetryFound {
				poetryDesc = a.poetryDesc
			} else {
				poetryDesc = b.poetryDesc
			}
		}

		// 枚举期只算总分：不构造 Items/Details 与各维度解释文案
		cand := doubleNameCandidate(a, b, poetryFound, poetryDesc, surnamePinyin, s.surnameJyutping, s.bigramCache)
		final, canto := s.blendScore(cand, fateData)

		idxFirst, idxSecond := j, genIdx
		if genFirst {
			idxFirst, idxSecond = genIdx, j
		}
		table.TryPush(ExcellentEntry{
			Char1:          a.ch.Char,
			Char2:          b.ch.Char,
			Pinyin1:        a.pinyin,
			Pinyin2:        b.pinyin,
			Meaning1:       a.ch.Meaning,
			Meaning2:       b.ch.Meaning,
			Score:          final,
			Grade:          scoreToGrade(final),
			WuXing1:        a.ch.WuXing,
			WuXing2:        b.ch.WuXing,
			Stroke1:        a.stroke,
			Stroke2:        b.stroke,
			KangxiStroke1:  a.ch.KangxiStroke,
			KangxiStroke2:  b.ch.KangxiStroke,
			HasPoetry:      poetryFound,
			PoetryFrom:     poetryDesc,
			NameFreqTier1:  a.ch.NameFreqTier,
			NameFreqTier2:  b.ch.NameFreqTier,
			CantoneseTotal: canto,
			idx1:           idxFirst,
			idx2:           idxSecond,
		})
		totalCount.Add(1)

		done++
		// 进度：评分阶段 30% 起步推进到 85%（与常规枚举阶段口径一致）
		s.percentX.Store(int64((30 + 55*float64(done)/pool) * 100))
	}
	table.Finalize()
	return infos, true
}

func (s *sessionImpl) generateSingleName(
	ctx context.Context,
	infos []charInfo,
	table *ExcellentTable,
	totalCount *atomic.Int64,
	fateData *FateData,
	surnamePinyin string,
) {
	for i := range infos {
		if cancelled(ctx) {
			break
		}
		// 进度：单名串行，30% 起步按外层推进到 85%
		s.percentX.Store(int64((30 + 55*float64(i)/float64(max(len(infos), 1))) * 100))
		a := infos[i]
		if !s.filter.CheckStrokePair(a.stroke, 0) {
			continue
		}
		// 名字用字质量门禁：单名若为门禁字（虚词/排行字/口语物名等）直接剔除。
		// 策展库无单名，故单名门禁不会误伤策展推荐。
		if IsNonNamingChar(a.ch.Char) {
			continue
		}

		// 枚举期只算总分：不构造 Items/Details 与各维度解释文案
		// （评分与解释分离，入榜后再按 idx1 回算完整明细）。
		cand := singleNameCandidate(a, surnamePinyin, s.surnameJyutping)
		final, canto := s.blendScore(cand, fateData)
		entry := ExcellentEntry{
			Char1:          a.ch.Char,
			Pinyin1:        a.pinyin,
			Meaning1:       a.ch.Meaning,
			Score:          final,
			Grade:          scoreToGrade(final),
			WuXing1:        a.ch.WuXing,
			Stroke1:        a.stroke,
			KangxiStroke1:  a.ch.KangxiStroke,
			HasPoetry:      a.poetryFound,
			PoetryFrom:     a.poetryDesc,
			NameFreqTier1:  a.ch.NameFreqTier,
			CantoneseTotal: canto,
			idx1:           i,
			idx2:           -1,
		}
		table.TryPush(entry)
		totalCount.Add(1)
	}
	table.Finalize()
}

// generateDoubleName 双名生成：i×j 笛卡尔积 + worker 并发
//
// 关键差异（vs 单名）：
//   - 嵌套笛卡尔积 N²
//   - 按 CPU 分片并发（外层 i 分片，内层 j 全量——worker 间无重叠）
//   - 组合级过滤（IsBadCombo / IsHistoricalFigureCombo / PairBlacklist 等）
//   - 早停：local.IsFull() 时按 (a, b) 组合潜力分与本地堆最小分比较
//
// localTable 容量自适应 topCount（避免无谓的 10000 容量浪费内存）。
// 末尾合并各分片局部 Top-N 到调用方传入的 table（**就地填充，不再重造**）。
//
// ★ 历史缺陷（docs/28 W6）：本函数的 table 参数曾是 `**ExcellentTable`，
// 末尾执行 `*table = NewExcellentTable()` —— 把 generate() 里刚建好的、容量
// 10000 的表整份丢弃再重造一个。前者从未写入任何条目，纯属垃圾。
// 现改为 `*ExcellentTable`，直接向已有表 TryPush 合并结果。
func (s *sessionImpl) generateDoubleName(
	ctx context.Context,
	infos []charInfo,
	table *ExcellentTable,
	totalCount *atomic.Int64,
	fateData *FateData,
	surnamePinyin string,
	topCount int,
	xiSet map[string]bool,
) {
	workers := workerCountHint(len(infos))

	// localTable 容量自适应 topCount（避免无谓的 10000 容量浪费内存）：
	// 取 topCount*2 留余量避免抖动，溢出时 TryPush 按堆顶最小分替换。
	// topCount 在 NewSessionWithFilter 后由 input.Options.Count 决定（默认 50）。
	localCap := topCount * 2
	if localCap < 100 {
		localCap = 100
	}

	chunkSize := (len(infos) + workers - 1) / workers
	localTables := make([]*ExcellentTable, workers)
	var wg sync.WaitGroup

	// 进度上报：所有 worker 共享一个外层完成计数器（0 → len(infos)），
	// 按推进比例把评分阶段映射到 30%–85%。只写 percentX，不改阶段。
	var outerDone atomic.Int64

	for w := 0; w < workers; w++ {
		start := w * chunkSize
		end := start + chunkSize
		if end > len(infos) {
			end = len(infos)
		}
		if start >= end {
			continue
		}

		// 获取全局 worker 令牌：总量 = CPU 核数，跨请求共享（P2-7）。
		// 高并发下后来的 worker 在此排队，最多等 CPU 核数个在跑，杜绝
		// 「并发请求数 × NumCPU」的超订放大。ctx 取消时直接放弃令牌占用，
		// 避免积累无谓的排队位。
		// 先捕获通道指针再收发：归还必须归还「获取时那一枚」令牌。
		// 若归还时读全局变量，测试替换通道后，旧 worker 会从新通道
		// 抽走令牌，导致新 worker 永远拿不到令牌而死锁。
		limit := candidateWorkerLimit
		select {
		case limit <- struct{}{}:
		case <-ctx.Done():
			return
		}

		wg.Add(1)
		go func(start, end, w int) {
			defer wg.Done()
			defer func() { <-limit }()
			// 并发钳制观测：本 worker 进入枚举前记录活跃计数与峰值，
			// 退出时递减。生产为零开销的原子递增，仅峰值恒定供测试断言。
			cur := candidateWorkerActive.Add(1)
			defer candidateWorkerActive.Add(-1)
			for {
				peak := candidateWorkerPeak.Load()
				if cur <= peak || candidateWorkerPeak.CompareAndSwap(peak, cur) {
					break
				}
			}
			// worker 局部表：本分片的 (i, j) 组合天然唯一，无需 seen 去重
			// （省下每个条目一次「两个汉字拼接成字符串 + map 写入」的分配）。
			local := NewExcellentTableUnique(localCap)

			// 潜力分预计算：内层循环每轮都要用，改为按字预算成数组。
			// 原实现每次组合都重算 a、b 两个字的潜力分，其中 a 的部分与内层
			// 下标 j 无关，属于纯重复计算（N² 次里算了 N³ 量级的一半）。
			potential := make([]int, len(infos))
			for k := range infos {
				potential[k] = charPotentialScore(
					infos[k].ch.WuXing, infos[k].ch.IsCurated,
					infos[k].ch.PositiveScore, infos[k].poetryFound, xiSet)
			}

			for i := start; i < end; i++ {
				if cancelled(ctx) {
					return
				}
				a := infos[i]
				if !s.filter.CheckStrokePair(a.stroke, 0) {
					continue
				}
				pa := potential[i]

				for j := range infos {
					if cancelled(ctx) {
						return
					}
					b := infos[j]

					// 早停前置：表已满时，潜力分明显不足的组合不可能最终入榜，
					// 直接跳过后续「组合级门禁 + 九维评分」，避免为注定淘汰的组合
					// 反复做字符串判重与谐音检索。
					//
					// 语义等价：原顺序为「门禁 → 早停 → 评分」，被早停跳过的组合
					// 本就不会产生任何结果，互换不改变实际被评分的组合集合，
					// 也不改变 totalCount（它只统计完成 RateName 的组合）。
					if cutoff := local.earlyStopCutoff(); cutoff > 0 &&
						float64(pa+potential[j]) < cutoff {
						continue
					}

					if !s.filter.CheckStrokePair(a.stroke, b.stroke) {
						continue
					}

					// 避免重复字（双名一般不用相同字）
					if a.ch.Char == b.ch.Char {
						continue
					}

					// 负面反馈：排除组合（sessionImpl 方法，读取不可变快照，无锁）
					if s.isComboExcluded(a.ch.Char, b.ch.Char) {
						continue
					}

					// 负面语义：组合禁忌（亲属称谓/物名/动词等，「父母」「蜂蜜」类直接剔除）
					if IsBadCombo(a.ch.Char, b.ch.Char) {
						continue
					}

					// 历史人物字/号专名：剔除「仲尼」「尼仲」「孔明」等与历史人物撞车组合
					// （GetBigramScore 会因《论语》"仲尼曰"等典籍共现给高分，需在此拦截）
					if IsHistoricalFigureCombo(a.ch.Char, b.ch.Char) {
						continue
					}

					// 名字用字质量门禁（纯语义）：若任一位置为门禁字
					// （虚词/排行字/口语物名/数字量词等），组合即无命名价值，
					// 直接剔除——不复依赖策展库豁免。
					// 策展库已降级为纯出典参考：其本身是历史人物名采集
					// （混入仲尼/若兮/七政/与砺 等 962 条垃圾），不可作为质量基准。
					if IsNonNamingChar(a.ch.Char) || IsNonNamingChar(b.ch.Char) {
						continue
					}

					// 数据层搭配黑名单：策展标注的 PairBlacklist（如 瑾-艳/妍-艳/嫣-艳）
					if hasPairBlacklist(a.ch, b.ch.Char) || hasPairBlacklist(b.ch, a.ch.Char) {
						continue
					}

					poetryFound := a.poetryFound || b.poetryFound
					poetryDesc := ""
					if a.poetryFound {
						poetryDesc = a.poetryDesc
					} else if b.poetryFound {
						poetryDesc = b.poetryDesc
					}

					// 枚举期只算总分：不构造 Items/Details 与各维度解释文案
					// （评分与解释分离，入榜后再按 idx1/idx2 回算完整明细）。
					cand := doubleNameCandidate(a, b, poetryFound, poetryDesc, surnamePinyin, s.surnameJyutping, s.bigramCache)
					final, canto := s.blendScore(cand, fateData)
					entry := ExcellentEntry{
						Char1:          a.ch.Char,
						Char2:          b.ch.Char,
						Pinyin1:        a.pinyin,
						Pinyin2:        b.pinyin,
						Meaning1:       a.ch.Meaning,
						Meaning2:       b.ch.Meaning,
						Score:          final,
						Grade:          scoreToGrade(final),
						WuXing1:        a.ch.WuXing,
						WuXing2:        b.ch.WuXing,
						Stroke1:        a.stroke,
						Stroke2:        b.stroke,
						KangxiStroke1:  a.ch.KangxiStroke,
						KangxiStroke2:  b.ch.KangxiStroke,
						HasPoetry:      poetryFound,
						PoetryFrom:     poetryDesc,
						NameFreqTier1:  a.ch.NameFreqTier,
						NameFreqTier2:  b.ch.NameFreqTier,
						CantoneseTotal: canto,
						idx1:           i,
						idx2:           j,
					}
					local.TryPush(entry)
					totalCount.Add(1)
				}
				// 外层 i 完成一步：共享计数器推进评分阶段进度（30%–85%）
				done := outerDone.Add(1)
				s.percentX.Store(int64((30 + 55*float64(done)/float64(max(len(infos), 1))) * 100))
			}
			localTables[w] = local
		}(start, end, w)
	}

	wg.Wait()

	// 合并各分片的局部 Top-N 到调用方传入的主表。
	// 全局 Top-k 必落在「某分片的局部 Top-容量」内，因此只需合并各分片保留的条目。
	//
	// ★ 此前此处是 `*table = NewExcellentTable()`（丢弃调用方刚建好的表并重造
	// 一个容量 10000 的新表），即 docs/28 W6 所指的重复预分配。现就地填充。
	for _, lt := range localTables {
		if lt == nil {
			continue
		}
		lt.Finalize()
		for _, e := range lt.entries {
			table.TryPush(e)
		}
	}
	table.Finalize()
}
