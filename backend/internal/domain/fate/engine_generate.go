package fate

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync/atomic"

	"name/internal/domain/classics"
)

// generate 核心生成逻辑
func (s *sessionImpl) generate(ctx context.Context, input *Input) (*Output, error) {
	s.setStage(sessionStageBazi, 2)

	// 1. 八字分析
	fateData, err := s.engine.analyzer.Analyze(input.Born, input.Gender)
	if err != nil {
		return nil, fmt.Errorf("八字分析失败: %w", err)
	}
	s.setStage(sessionStageBazi, 8)

	// 2. 获取姓氏笔画信息（用于总笔画数计算，支撑河图数理与易经卦象解读）
	l1, l2, err := s.engine.provider.GetSurnameStrokes(input.Surname)
	if err != nil {
		return nil, fmt.Errorf("获取姓氏笔画失败: %w", err)
	}
	// 姓氏康熙笔画注入 FateData，供卦象评分器按统一口径起卦（docs/42 P3-12）
	fateData.SurnameStrokes = l1 + l2

	// 3. 构建查询条件并加载汉字
	query := NewBasicCharacterQuery()
	query = s.filter.QueryFilter(query).(*BasicCharacterQuery)
	allChars, err := s.engine.provider.FindCharacters(query)
	if err != nil {
		return nil, fmt.Errorf("加载汉字数据失败: %w", err)
	}

	// 4. 合并外部注入的额外候选字（诗词/经典字库），去重
	if len(input.Options.ExtraChars) > 0 {
		existingChars := make(map[string]bool, len(allChars))
		for _, c := range allChars {
			existingChars[c.Char] = true
		}
		for _, c := range input.Options.ExtraChars {
			if !existingChars[c.Char] {
				allChars = append(allChars, c)
			}
		}
	}

	// 5. 对每个字符进行过滤（含负面反馈排除的字符）
	// 排除集由 isCharExcluded / isComboExcluded 查询，两者读取不可变快照，
	// 无锁、无竞争（见 exclusionSnap 字段说明）。

	// 避讳长辈：构建同形字与同音字排除集
	// 规则：同形字=侵佔福分，同音字=气场冲撞（"压运"）
	elderCharSet, elderPinyinSet := s.buildElderAvoidance(input.AvoidElderNames)

	// 大名需大命：检测极旺格局（专旺格/从强格）
	// 普通格局需移除敏感字（龙/凤/乾/坤/圣/贤/天/帝/皇/神/仙/君）
	allowSensitive := isExtremeStrongPattern(fateData)

	// 负面反馈排除通过 s.isCharExcluded / s.isComboExcluded 查询，两者读取
	// 不可变快照（atomic.Pointer），对写方 ExcludeChar/ExcludeCombo/
	// ClearExclusions 无锁可见（修复 docs/28 W4：读路径此前完全不加锁）。
	//
	// 时序语义：快照指针在**每次调用**时加载，因此生成进行中新增的排除项会
	// 立即对「尚未评估的组合」生效（无需等待下一次 generate）。
	validChars := make([]*Character, 0, len(allChars))
	for _, c := range allChars {
		if s.isCharExcluded(c.Char) {
			continue
		}
		// 避讳长辈：排除同形字
		if elderCharSet[c.Char] {
			continue
		}
		// 避讳长辈：排除同音字（任一拼音命中即排除）
		if len(elderPinyinSet) > 0 {
			avoid := false
			for _, p := range c.Pinyin {
				if elderPinyinSet[p] {
					avoid = true
					break
				}
			}
			if avoid {
				continue
			}
		}
		// 大名需大命：普通格局排除敏感字
		if !allowSensitive && IsSensitiveChar(c.Char) {
			continue
		}
		// 负面语义：硬禁用字（秽物/尸棺/淫猥/盗匪/暴虐/贬义等）直接剔除
		// 软惩罚字（病/疾/哀/愁等）保留入池资格，由评分器重罚（支持"去病/弃疾"式祈福）
		if IsHardNegativeChar(c.Char) || c.IsNegative {
			continue
		}
		// 名字用字质量门禁：虚词/排行字/口语物名/数字量词/叹词等
		// 在「组合级」由策展感知门禁处理（IsNonNamingCombo），此处不剔除，
		// 以免误伤策展好名（如「与砺」「若兮」中的「与/兮」）。
		// 单名场景（无策展单名）在单名循环内单独剔除。
		if s.filter.CheckCharacter(c) {
			validChars = append(validChars, c)
		}
	}

	// 6. 全放行《通用规范汉字表》字源
	//
	// 历史：曾用「人工寓意评分>=85」作为唯一推荐字源门槛，把候选池收窄到
	// 200-360 字，导致多次生成名字高度近似。且 namer.json 8105 字中仅 563 字
	// 被人工评分（93% 无评分），门槛信号极度稀缺；再叠加五行收窄后往往只剩
	// 几十字可组合。
	//
	// 现改为「全放行」：filter 各层（性别/笔画/禁用字/负面语义/生僻字/门禁表）
	// 是唯一字源防线，荒谬字由以下多层防线兜底，不再用评分门槛一刀切：
	//   - 字级：IsHardNegativeChar / IsNegative / CheckCharacter 已剔除硬负面与门禁字
	//   - 组合级：策展感知质量门禁（IsNonNamingCombo）、禁忌组合（IsBadCombo）
	//   - 评分级：WenHuaRater 仅对「策展∩评分>=90」加分，PositiveScore 空无文化分；
	//     非策展双名四维封顶 75，荒谬组合天然低分掉榜
	// 入选检查由 engine_curated_pool_test.go 改为在真实数据规模上断言推荐榜无荒谬字。
	//

	// 7. 喜用神五行收窄候选池（性能优化 + 方案B：含生助五行）
	//
	// 旧版（纯性能优化）：仅保留喜用神五行字 → 所有候选五行相同 → WuxingRater 恒分，无区分度。
	// 方案B：收窄池扩展为「喜用神 + 生助喜用神」的五行集合。
	//   例：喜用神=土、金 → 收窄池含土、金（喜用）+ 火（火生土）+ 土（土生金已含）→ 合计3种五行。
	//   这样 WuxingRater 的间接生助梯度（生喜用+8 / 中性+3）才能真正区分候选。
	//
	// 收窄后若候选过少（<80），放弃收窄避免过度限制（降级保护）。
	// 单名例外：候选池仅 ~1100 字，串行枚举足够快，无需性能收窄。
	effectiveWuxing := s.filter.PreferredWuXing()
	if input.Options.NameLength != 1 && len(effectiveWuxing) == 0 && len(fateData.WuXingXiji.XiYongShen) > 0 {
		// 构建收窄集：喜用神 + 每个喜用神的"生我"元素（谁生我）
		// 五行相生：木→火→土→金→水→木（wuXingShengMap[a]=b 表示 a 生 b）
		// "生我"即反查：wuXingShengMap[x]==target → x 生 target
		narrowSet := make(map[string]bool)
		for _, wx := range fateData.WuXingXiji.XiYongShen {
			narrowSet[wx] = true // 直接匹配喜用神
		}
		// 反查"谁生我"：遍历所有五行，找到生喜用神的元素
		for _, wx := range fateData.WuXingXiji.XiYongShen {
			for src, dst := range wuXingShengMap {
				if dst == wx {
					narrowSet[src] = true // src 生 wx（生助喜用神）
				}
			}
		}

		narrowed := make([]*Character, 0, len(validChars))
		for _, c := range validChars {
			if narrowSet[c.WuXing] {
				narrowed = append(narrowed, c)
			}
		}
		// 降级保护：收窄后候选过少时保留全量，避免结果单一或空
		if len(narrowed) >= 80 {
			validChars = narrowed
		}
	}

	// 7.2 用户显式偏好生效：寓意关键词 + 偏旁选字（docs/29 A9）
	//
	// 缺陷背景：MeaningKeywords 此前只在服务层被赋值，engine/generator 无任何
	// 读取点 —— 字段一路传递后即被丢弃，用户输入关键词与完全不输入的结果
	// 一模一样，而 UI 明确承诺「起名将优先从此来源选字」。前端更把偏旁选字
	// 拼成 "包含字：木木" 混进 keywords 文本发送，结构化通道从未被读取。
	//
	// 修法：把用户的显式偏好落到**候选池**上——这既让偏好真正可见（枚举是
	// 全量 N×N、结果表按分数取 TopN，仅调整枚举顺序不足以影响输出），又不
	// 改动评分器权重与分数语义（RateNameScore 的权重和恒为 1.0，不能加维度）。
	//
	// 语义：命中关键词字义/起名分类的字，以及用户点选的字，组成「偏好集」；
	// 偏好集非空时把候选池收窄到偏好集（这才是「优先从此来源选字」）；
	// 偏好集为空（关键词全不命中任何字）时放弃收窄，保持原有行为。
	if narrowed := narrowPoolByPreference(validChars, input.Options.MeaningKeywords, input.Options.RequiredChars); narrowed != nil {
		validChars = narrowed
	}

	// 7.1 候选池按汉字去重
	//
	// 重复汉字会让 (i, j) 组合在「名字」层面重复——同一 Char1+Char2 被枚举多次。
	// 历史上由 ExcellentTable 的 seen map 兜底去重（首次写入获胜）；而枚举期的
	// worker 局部表为省下每次 push 的字符串拼接已改为不去重（NewExcellentTableUnique），
	// 因此必须在池层保证组合语义唯一。保留首次出现，与历史上「seen 首次写入获胜」
	// 的结果完全一致（同字不同下标构造出的候选字段相同、得分相同）。
	//
	// 生产数据中候选池本身按字唯一，此处为外部注入（ExtraChars）与上游数据异常的兜底；
	// 顺带避免对重复字做无谓的重复枚举。
	validChars = dedupCharsByName(validChars)

	// 6. 预计算每个候选字的固定属性（使用包级 charInfo 类型，避免双重循环内重复调用
	// GetCharacterStroke / firstPinyin / 诗词检索）

	infos := make([]charInfo, len(validChars))
	for i, c := range validChars {
		found, desc, _ := classics.FindPoetryByChars(c.Char)
		infos[i] = charInfo{
			ch:             c,
			stroke:         s.filter.GetCharacterStroke(c),
			pinyin:         firstPinyin(c.Pinyin),
			poetryFound:    found,
			poetryDesc:     desc,
			meaningProfile: meaningProfileOf(c.Meaning),
		}
	}

	// 预排序候选池：按「潜力分」降序排列
	// 潜力分 = 五行匹配(0-24) + 策展好字(0-16) + 寓意评分(0-9) + 诗词出典(0-5)
	// 高潜力字排在前面，使笛卡尔积循环中高分组合优先进入 ExcellentTable，
	// 后续低潜力组合在早停检查中被跳过，减少无效评分计算。
	//
	// ★ 同潜力分必须按字典序决胜（不能只用稳定排序）：infos 源自候选池的
	// map 遍历（顺序随机），稳定排序会把这份随机性带进同分段的相对次序，
	// 进而随机化 worker 分片边界与早停阈值 —— 实测同请求两次生成的 Top-N
	// 连分数都不同（77.1 vs 77）。全序后整条枚举链路（分片、早停、合并、
	// 入榜）对候选池遍历顺序完全不敏感。
	xiSet := make(map[string]bool)
	if fateData != nil {
		for _, wx := range fateData.WuXingXiji.XiYongShen {
			xiSet[wx] = true
		}
	}
	sort.Slice(infos, func(i, j int) bool {
		pi := charPotentialScore(infos[i].ch.WuXing, infos[i].ch.IsCurated, infos[i].ch.PositiveScore, infos[i].poetryFound, xiSet)
		pj := charPotentialScore(infos[j].ch.WuXing, infos[j].ch.IsCurated, infos[j].ch.PositiveScore, infos[j].poetryFound, xiSet)
		if pi != pj {
			return pi > pj
		}
		return infos[i].ch.Char < infos[j].ch.Char
	})

	surnamePinyin := firstPinyinForSurname(input.Surname, s.engine.provider)

	// 生成名字候选 */
	nameLen := input.Options.NameLength
	if nameLen <= 0 {
		nameLen = 2
	}
	topCount := input.Options.Count
	if topCount <= 0 {
		topCount = 100
	}

	// 表的容量统一取「本请求真正需要的池大小」而非固定的 10000（docs/28 W6）。
	//
	// 历史缺陷：这里先 `NewExcellentTable()`（容量恒为 10000，堆切片 + 两个 map
	// 一次预分配），紧接着 generateDoubleName 内部又 `*table = NewExcellentTable()`
	// 重造一份，第一份**从未被写入任何条目**就直接变成垃圾。按 ExcellentEntry
	// 约 200 字节估算，每次请求白白分配又丢弃约 2MB。
	//
	// 现在：容量按 topCount*10 计算——这正是下游 poolSize（Top10N，用于五行多样性
	// 保底）所需的规模，既不再超额预分配 10000，也不再需要事后重造。
	// 注意此处用 NewExcellentTableWithCap（容量自适应），它仍需保留去重能力
	// （合并各分片时条目来源不同，必须按 Char1+Char2 去重）。
	poolSize := topCount * 10
	if poolSize < minExcellentTablePoolSize {
		// 小 topCount 场景留一个下限，避免堆频繁替换（每次替换 O(log n)）
		poolSize = minExcellentTablePoolSize
	}
	table := NewExcellentTableWithCap(poolSize)
	// totalCount 统计「完成 RateName 评分」的候选组合数（被过滤/早停跳过的组合不计数），
	// 用于展示生成规模统计，语义为「已评分组合数」而非「总尝试组合数」（B11）。
	var totalCount atomic.Int64

	// 用户显式指定字辈（Generation）时走「固定槽位 × 自由枚举」分支：
	// 字辈字固定占一个槽位，其余槽位仍从候选池自由枚举。
	// 该分支返回 handled=false 表示字辈字不合法（多字/查库失败），回退常规枚举。
	handled := false
	if input.Generation != "" {
		infos, handled = s.generateWithGeneration(ctx, input, infos, table, &totalCount, fateData, surnamePinyin)
	}
	if !handled {
		if nameLen == 1 {
			// 单名：仅迭代第一个字（候选集小，串行足够快）
			s.setStage(sessionStageScore, 30)
			s.generateSingleName(ctx, infos, table, &totalCount, fateData, surnamePinyin)
		} else {
			// 双名：迭代 Char1 × Char2（全量枚举，按外层 Char1 分片并行）
			// 合并结果**就地写入**传入的 table，不再重造（docs/28 W6）。
			s.generateDoubleName(ctx, infos, table, &totalCount, fateData, surnamePinyin, topCount, xiSet)
		}
	}

	// 7. 构建 TopNames 输出（过滤禁止的国家机关单位名称）
	s.setStage(sessionStageRank, 90)
	// 五行多样性保底策略：取更多候选（Top10N），确保每种五行组合至少1个代表，
	// 避免单一五行组合（如金金）垄断排名。未达多样性要求时退化为纯分数排序。
	// 注意：建表容量即 topCount*10，故正常情况下 table.Len() >= poolSize，
	// 只有候选池极小或全部被过滤时才会触发下面这条下调。
	if poolSize > table.Len() {
		poolSize = table.Len()
	}
	topEntries := table.TopN(poolSize)
	if poolSize > topCount {
		topEntries = ensureWuxingDiversity(topEntries, topCount)
	}

	// 6.1 惰性明细回算（评分与解释分离）
	//
	// 枚举阶段只算总分（RateNameScore），不构造各维度的 Items/Details 与
	// 解释文案；此处只对真正进入推荐榜的条目（≤ poolSize 条，约千分之一量级）
	// 用 RateName 回算一次完整明细，填回条目供下游透传。
	// 回算用独立候选（skipDetail=false），不影响枚举期的复用候选。
	s.fillEntryDetails(topEntries, infos, fateData, surnamePinyin)
	s.setStage(sessionStageRank, 97)

	topNames := make([]NameResult, 0, len(topEntries))
	rank := 0
	for _, e := range topEntries {
		givenName := e.Char1 + e.Char2
		fullName := input.Surname + givenName
		// 国家机关单位名称禁止检查（AGENTS.md 安全策略）
		if IsForbiddenEntity(fullName) {
			continue
		}
		rank++
		// 计算总笔画数（姓氏笔画 + 名字笔画）
		totalStrokes := 0
		if l1 > 0 {
			totalStrokes += l1
		}
		if l2 > 0 {
			totalStrokes += l2
		}
		// 笔画口径统一为康熙字典笔画（KangxiStroke1/2），与姓氏走 l1+l2
		// 同一口径（LookupSurnameStrokes 返回康熙笔画）。
		// 修复前 e.Stroke1+Stroke2 走的是 filter.GetCharacterStroke（默认 ScienceStroke），
		// 导致"姓用康熙、名用科学"的总笔画错误（docs/19 报告 B3）。
		// KangxiStroke1/2 == 0 时降级为 Stroke1/2（未收录兜底）。
		nameStroke1 := e.KangxiStroke1
		if nameStroke1 == 0 {
			nameStroke1 = e.Stroke1
		}
		nameStroke2 := e.KangxiStroke2
		if nameStroke2 == 0 {
			nameStroke2 = e.Stroke2
		}
		totalStrokes += nameStroke1 + nameStroke2
		nr := NameResult{
			Rank:      rank,
			Surname:   input.Surname,
			GivenName: givenName,
			FullName:  fullName,
			// 拼音回填：ExcellentEntry 已携带预计算的候选字读音，
			// 双名以空格组合，单名仅首字读音（TrimSpace 清理尾随空格）
			Pinyin: strings.TrimSpace(e.Pinyin1 + " " + e.Pinyin2),
			// 粤拼标注回填：姓 + 名各字粤拼以空格组合（未收录字跳过，TrimSpace 清理）
			Jyutping: strings.TrimSpace(e.SurnameJyutping + " " + e.Jyutping1 + " " + e.Jyutping2),
			// 释义回填：候选字释义组合为名字寓意描述，
			// 单名仅首字释义，双名以「；」连接；超长释义按 rune 截断防撑爆响应
			Meaning: combineCharMeanings(e.Meaning1, e.Meaning2),
			Strokes: totalStrokes,
			WuXing:  e.WuXing1 + e.WuXing2,
			// 诗词出处回填：从 ExcellentEntry 透传
			PoetryFrom: e.PoetryFrom,
			Score: NameScore{
				Total:   e.Score,
				Grade:   e.Grade,
				Items:   e.Items,
				Details: e.Details,
			},
		}
		// 粤语分透出：仅在开启粤语评分且已回算明细时附带（nil 表示未开启，JSON 省略）。
		if s.enableCantonese && e.CantoneseItems != nil {
			nr.CantoneseScore = &NameScore{
				Total:   e.CantoneseTotal,
				Grade:   scoreToGrade(e.CantoneseTotal),
				Items:   e.CantoneseItems,
				Details: e.CantoneseDetails,
			}
		}
		topNames = append(topNames, nr)
	}

	s.setStage(sessionStageRank, 100)

	// ★ 取消/超时必须显式失败（docs/28 W3）。
	//
	// 各枚举循环在 `cancelled(ctx)` 为真时只是 `return`（静默退出该 worker），
	// 主流程照常往下走并组装出 TopNames。结果是：会话被取消，但 session.Wait()
	// 正常返回 nil、State() 显示 SessionStateFinish、Result() 给出一份
	// **被截断的榜单**——上层（服务层 / handler）若不额外检查 ctx.Err()
	// 就会把它当成正常结果回 200，客户端拿到不完整的名单却毫不知情、不会重试。
	//
	// 这里在返回前统一判定：只要 ctx 已结束，就以 ctx 的错误作为本次生成的结果，
	// 让失败**在领域层自身**可见，而不是依赖每个调用方都记得补一次 ctx.Err() 检查。
	// 返回的 error 包装了 ctx.Err()，因此调用方仍可用
	// errors.Is(err, context.Canceled/DeadlineExceeded) 判定并映射为 503。
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("名字生成被中止: %w", err)
	}

	return &Output{
		Input:          input,
		FateData:       fateData,
		TopNames:       topNames,
		ExcellentTable: table,
		TotalCount:     int(totalCount.Load()),
	}, nil
}
