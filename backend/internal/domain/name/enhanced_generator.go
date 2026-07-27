package name

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"unicode"

	"name/internal/domain/classics"
	"name/internal/domain/hanzi"
	"name/internal/domain/yijing"
	"name/internal/infrastructure/logger"
	"go.uber.org/zap"
	"golang.org/x/text/unicode/norm"
)

// NameAnalysis 名字分析结构
type NameAnalysis struct {
	// 基础信息
	Surname        string   `json:"surname"`
	GivenName      string   `json:"given_name"`
	FullName       string   `json:"full_name"`
	Pinyin         string   `json:"pinyin"`
	Strokes        int      `json:"strokes"`
	Gender         string   `json:"gender"`
	
	// 五行分析
	Wuxing         string   `json:"wuxing"`
	WuxingAnalysis string   `json:"wuxing_analysis"`
	WuxingScore    float64  `json:"wuxing_score"`
	
	// 八字分析
	BaziScore      int      `json:"bazi_score"`
	BaziAnalysis   string   `json:"bazi_analysis"`
	Xiyongshen     []string `json:"xiyongshen"`
	
	// 音韵分析
	YinyunScore    float64  `json:"yinyun_score"`
	YinyunAnalysis string   `json:"yinyun_analysis"`
	
	// 字义分析
	MeaningScore   float64  `json:"meaning_score"`
	MeaningDetail  string   `json:"meaning_detail"`
	
	// 易经分析
	Hexagram       string   `json:"hexagram"`
	HexagramMeaning string  `json:"hexagram_meaning"`
	
	// 三才分析（天地人：天道阴阳、地道刚柔、人道仁义）
		Sancai         string   `json:"sancai"`
		SancaiScore    float64  `json:"sancai_score"`
		SancaiAnalysis string   `json:"sancai_analysis"`
	
	// 生肖分析
	ZodiacMatch    string   `json:"zodiac_match"`
	ZodiacScore    float64  `json:"zodiac_score"`
	
	// 纳音分析
	NayinScore     float64  `json:"nayin_score"`
	NayinAnalysis  string   `json:"nayin_analysis"`
	
	// 候选名库加分
	NameDBScore    float64  `json:"name_db_score"`
	NameDBSource   string   `json:"name_db_source"`
	
	// 诗词典故
	PoetrySource   string   `json:"poetry_source"`
	PoetryChapter  string   `json:"poetry_chapter"`
	PoetrySentence string   `json:"poetry_sentence"`
	
	// 综合评分
	TotalScore     float64  `json:"total_score"`
	Recommendations []string `json:"recommendations"`
}

// EnhancedNameGenerator 增强版名字生成器
type EnhancedNameGenerator struct {
	*NameGenerator
	analysisCache  map[string]*NameAnalysis
	cacheMu        sync.RWMutex
	yinyunAnalyzer *YinyunAnalyzer
	nameDB         *NameDB
	nameFilter     *NameFilter
	initErr        error // 候选名库初始化错误（可通过 InitError() 获取）

	// 预计算的性别候选字池（避免每次请求扫描全部 HanziData）
	prebuiltMaleChars   []string
	prebuiltFemaleChars []string
	charPoolOnce        sync.Once

	// 诗词句子索引（按首次访问惰性加载）
	poetrySentences         []poetrySentenceEntry
	poetryCharIndex         map[string][]poetrySentenceEntry // 字符→包含该字符的句子列表，用于快速共现检索
	poetryCharSentenceSet   map[string]map[string]bool       // 字符→包含该字符的句子集合（O(1) 交集判断，避免每对新建 map）
	poetrySentenceInfo      map[string]poetrySentenceEntry   // 句子→元数据（共现命中后 O(1) 取出处）
	poetrySentencesOnce     sync.Once
	poetryZodiacOnce        sync.Once                        // 生肖宜用/忌用字集惰性初始化
	zodiacGoodCharSets      map[string]map[string]bool       // 生肖→宜用字集
	zodiacTabooCharSets     map[string]map[string]bool       // 生肖→忌用字集
}

// poetrySentenceEntry 诗词句子条目（用于双字共现检索）
type poetrySentenceEntry struct {
	Source   string
	Chapter  string
	Sentence string
}

// analysisCacheMaxSize 分析缓存最大条目数，超过时清空（防止内存无限增长）
const analysisCacheMaxSize = 2000

// NewEnhancedNameGenerator 创建增强版名字生成器
func NewEnhancedNameGenerator(dataDir string, persisters ...CuratedPersister) *EnhancedNameGenerator {
	var persister CuratedPersister
	if len(persisters) > 0 {
		persister = persisters[0]
	}

	nameDB, err := NewNameDB(dataDir, WithCuratedPersister(persister))
	if err != nil {
		logger.Warn("候选名库加载失败", zap.String("msg", "生成功能将降级，仅使用内置字库"), zap.Error(err))
	}

	eng := &EnhancedNameGenerator{
		NameGenerator: NewNameGenerator(),
		analysisCache:  make(map[string]*NameAnalysis),
		yinyunAnalyzer: NewYinyunAnalyzer(),
		nameDB:         nameDB,
		nameFilter:     NewNameFilter(),
		initErr:        err,
	}
	// 启动时预计算性别候选字池
	eng.initCharPool()
	return eng
}

// InitError 返回构造期间发生的错误（如候选名库加载失败）。
// 调用方应检查此方法以感知降级状态；为 nil 表示初始化正常。
func (eng *EnhancedNameGenerator) InitError() error {
	return eng.initErr
}

// GetNameDB 获取候选名库管理器（只读，用于共享给 API 层）。
// 注意：若构造时候选名库加载失败（见 InitError），可能返回 nil，调用方需做空判断。
func (eng *EnhancedNameGenerator) GetNameDB() *NameDB {
	return eng.nameDB
}

// initCharPool 预计算性别候选字池，避免每次请求扫描全部 HanziData
func (eng *EnhancedNameGenerator) initCharPool() {
	eng.charPoolOnce.Do(func() {
		var malePrelim, femalePrelim []string
		for c, h := range hanzi.HanziData {
			// 常用等级过滤：UsageLevel>=3 且 CurationLevel>=2
			if h.UsageLevel < 3 || h.CurationLevel < 2 {
				continue
			}
			// 生僻字/消极字过滤
			if h.IsRare || h.IsNegative || h.NamePenalty > 20 {
				continue
			}
			// 笔画数合理范围
			if h.Strokes < 3 || h.Strokes > 25 {
				continue
			}
			if h.Gender != "female" {
				malePrelim = append(malePrelim, c)
			}
			if h.Gender != "male" {
				femalePrelim = append(femalePrelim, c)
			}
		}
		eng.prebuiltMaleChars = malePrelim
		eng.prebuiltFemaleChars = femalePrelim
	})
}

// GenerateNamesWithAnalysis 生成带详细分析的名字
func (eng *EnhancedNameGenerator) GenerateNamesWithAnalysis(opts GenerateOptions) ([]*NameAnalysis, error) {
	// 生成基础名字列表
	names := eng.GenerateNamesWithGeneration(
		opts.Surname, opts.Generation, opts.Gender, opts.Xiyongshen, opts.Count, opts.NameLength,
		opts.ExcludeRare, opts.WuxingMatch, opts.SourceClassic, opts.MinStrokes, opts.MaxStrokes,
		opts.IncludePoetry, opts.IncludeClassic, opts.MeaningKeywords, opts.PinyinInitial,
		opts.GenerationPosition, opts.NameType,
	)
	
	if len(names) == 0 {
		return nil, fmt.Errorf("未生成符合条件的名字")
	}
	
	// 并行分析每个名字
	var wg sync.WaitGroup
	analysisList := make([]*NameAnalysis, len(names))
	var mu sync.Mutex
	
	for i, name := range names {
		wg.Add(1)
		go func(index int, n Name) {
			defer wg.Done()
			
			analysis := eng.AnalyzeName(n, opts.Xiyongshen, opts.Zodiac, opts.Nayin)
			
			// 应用硬过滤：不满足硬过滤的名字直接跳过
			if eng.nameFilter != nil {
				filterResult := eng.nameFilter.Filter(n)
				if !filterResult.Valid {
					analysis = nil // 标记为无效，后续跳过
				} else if len(filterResult.Issues) > 0 {
					// 软检查问题加入推荐语
					analysis.Recommendations = append(analysis.Recommendations, filterResult.Issues...)
				}
			}
			
			mu.Lock()
			analysisList[index] = analysis
			mu.Unlock()
		}(i, name)
	}
	
	wg.Wait()
	
	// 过滤掉被硬过滤标记为无效的名字
	filteredList := make([]*NameAnalysis, 0, len(analysisList))
	for _, a := range analysisList {
		if a != nil {
			filteredList = append(filteredList, a)
		}
	}
	
	// 按综合评分排序
	eng.sortByScore(filteredList)
	
	return filteredList, nil
}

// AnalyzeName 详细分析单个名字
func (eng *EnhancedNameGenerator) AnalyzeName(name Name, xiyongshen []string, zodiac string, nayin string) *NameAnalysis {
	// 检查缓存（key 包含喜用神/生肖/纳音，避免不同八字分析同一名字返回错误缓存）
	cacheKey := fmt.Sprintf("%s%s|%s|%s|%s", name.Surname, name.GivenName, strings.Join(xiyongshen, ","), zodiac, nayin)
	eng.cacheMu.RLock()
	if cached, ok := eng.analysisCache[cacheKey]; ok {
		eng.cacheMu.RUnlock()
		return cached
	}
	eng.cacheMu.RUnlock()
	
	analysis := &NameAnalysis{
		Surname:   name.Surname,
		GivenName: name.GivenName,
		FullName:  name.Surname + name.GivenName,
		Pinyin:    name.Pinyin,
		Strokes:   name.Strokes,
		Gender:    name.Gender,
		Wuxing:    name.Wuxing,
		Xiyongshen: xiyongshen,
	}
	
	// 1. 五行分析
	analysis.WuxingAnalysis = eng.analyzeWuxing(name.Wuxing, xiyongshen)
	analysis.WuxingScore = eng.calculateWuxingScore(name.Wuxing, xiyongshen)
	
	// 2. 八字分析
	analysis.BaziScore = name.BaZiScore
	analysis.BaziAnalysis = name.BaziScoreDetail
	
	// 3. 音韵分析
	analysis.YinyunScore, analysis.YinyunAnalysis = eng.analyzeYinyun(name)
	
	// 4. 字义分析
	analysis.MeaningScore, analysis.MeaningDetail = eng.analyzeMeaning(name)
	
	// 5. 易经分析
	hexagram := yijing.GetHexagramByStrokes(name.Strokes)
	analysis.Hexagram = hexagram.Name
	analysis.HexagramMeaning = hexagram.Interpretation
	
	// 6. 天地人三才分析
	analysis.Sancai, analysis.SancaiScore, analysis.SancaiAnalysis = eng.analyzeSancai(name)

	// 7. 生肖分析
	analysis.ZodiacMatch, analysis.ZodiacScore = eng.analyzeZodiac(name, zodiac)
	
	// 8. 纳音分析
	analysis.NayinScore, analysis.NayinAnalysis = eng.calculateNayinScore(nayin, name.Wuxing)
	
	// 9. 候选名库评分
	if eng.nameDB != nil {
		analysis.NameDBScore, analysis.NameDBSource = eng.scoreFromNameDB(name.FullName)
	} else {
		analysis.NameDBScore = 0
		analysis.NameDBSource = ""
	}
	
	// 9. 诗词典故
	analysis.PoetrySource = name.PoetrySource
	analysis.PoetryChapter = name.PoetryChapter
	analysis.PoetrySentence = name.PoetrySentence
	
	// 9. 计算综合评分
	analysis.TotalScore = eng.calculateTotalScore(analysis)
	
	// 10. 生成推荐语
	analysis.Recommendations = eng.generateRecommendations(analysis)
	
	// 缓存结果（限制大小，超过阈值时删除旧条目，防止内存无限增长）
	eng.cacheMu.Lock()
	if len(eng.analysisCache) >= analysisCacheMaxSize {
		// 删除 20% 最旧条目，避免全清空导致缓存雪崩
		toDelete := analysisCacheMaxSize / 5
		for k := range eng.analysisCache {
			delete(eng.analysisCache, k)
			toDelete--
			if toDelete <= 0 {
				break
			}
		}
	}
	eng.analysisCache[cacheKey] = analysis
	eng.cacheMu.Unlock()

	return analysis
}

// 五行生克关系表
var wuxingSheng = map[string]string{
	"金": "水",
	"水": "木",
	"木": "火",
	"火": "土",
	"土": "金",
}

var wuxingKe = map[string]string{
	"金": "木",
	"木": "土",
	"土": "水",
	"水": "火",
	"火": "金",
}

// wuxingMatchWeight 计算单个五行与喜用神的生克制化匹配分数
func wuxingMatchWeight(w, xy string) float64 {
	if w == xy {
		return 100.0 // 直接匹配：喜用神直接补益
	}
	if wuxingSheng[w] == xy {
		return 75.0 // 相生：名字五行生喜用神（如金生水），间接补益
	}
	if wuxingSheng[xy] == w {
		return 50.0 // 被生：喜用神生名字五行（如木生火），虽不直接但得生扶
	}
	if wuxingKe[xy] == w {
		return 40.0 // 被克：喜用神克名字五行（如木克土），喜用神可约束
	}
	if wuxingKe[w] == xy {
		return 25.0 // 相克：名字五行克喜用神（如土克水），消耗喜用神力量
	}
	return 50.0 // 无关
}

// analyzeWuxing 分析五行（含生克制化描述）
func (eng *EnhancedNameGenerator) analyzeWuxing(wuxing string, xiyongshen []string) string {
	if wuxing == "未知" || len(xiyongshen) == 0 {
		return "五行属性明确，有助于平衡命理"
	}
	
	wuxingList := strings.Split(wuxing, "、")
	var details []string
	for _, w := range wuxingList {
		for _, xy := range xiyongshen {
			weight := wuxingMatchWeight(w, xy)
			switch weight {
			case 100:
				details = append(details, fmt.Sprintf("'%s'与喜用神'%s'相同，直接补益", w, xy))
			case 75:
				details = append(details, fmt.Sprintf("'%s'生'%s'，间接补益喜用神", w, xy))
			case 50:
				details = append(details, fmt.Sprintf("'%s'受'%s'所生，得生扶", w, xy))
			case 40:
				details = append(details, fmt.Sprintf("'%s'受'%s'所克，喜用神可约束", w, xy))
			case 25:
				details = append(details, fmt.Sprintf("'%s'克'%s'，消耗喜用神", w, xy))
			}
		}
	}
	
	return fmt.Sprintf("名字五行属%s；%s", wuxing, strings.Join(details, "；"))
}

// calculateWuxingScore 计算五行评分（升级版：含生克制化关系 + 名字内部五行关系）
// 单字名直接返回匹配分；双字名引入缩放与关系调节，避免所有名字都100分
func (eng *EnhancedNameGenerator) calculateWuxingScore(wuxing string, xiyongshen []string) float64 {
	if wuxing == "未知" || len(xiyongshen) == 0 {
		return 70.0
	}

	wuxingList := strings.Split(wuxing, "、")

	// 1. 喜用神匹配度评分
	totalWeight := 0.0
	for _, w := range wuxingList {
		best := 0.0
		for _, xy := range xiyongshen {
			weight := wuxingMatchWeight(w, xy)
			if weight > best {
				best = weight
			}
		}
		// 如果没有任何关系匹配（best==0），设为中性50分
		if best == 0 {
			best = 50.0
		}
		totalWeight += best
	}

	// 单字名：直接返回平均匹配分（保持高分语义），但应用最低30分约束
	if len(wuxingList) == 1 {
		if totalWeight < 30 {
			return 30
		}
		return totalWeight
	}

	// 2. 双字名：匹配分缩放到 0-80，留出关系调节空间
	matchScore := (totalWeight / float64(len(wuxingList))) * 0.80

	// 3. 名字内部五行关系调节分（-10 ~ +15）
	// 双字名两字五行间的关系影响整体协调性
	relationBonus := 0.0
	if len(wuxingList) == 2 {
		w1, w2 := wuxingList[0], wuxingList[1]
		if w1 == w2 {
			// 两字同五行：过于偏颇，减分
			relationBonus = -5.0
		} else if wuxingSheng[w1] == w2 || wuxingSheng[w2] == w1 {
			// 相生：五行流畅，加分
			relationBonus = 15.0
		} else if wuxingKe[w1] == w2 || wuxingKe[w2] == w1 {
			// 相克：不协调，减分
			relationBonus = -10.0
		} else {
			// 不同五行但无生克：中性偏正
			relationBonus = 5.0
		}
	}

	score := matchScore + relationBonus

	// 评分范围约束
	if score < 30 {
		score = 30
	}
	if score > 100 {
		score = 100
	}

	return score
}

// analyzeYinyun 分析音韵（使用 YinyunAnalyzer 高级音韵引擎）
func (eng *EnhancedNameGenerator) analyzeYinyun(name Name) (float64, string) {
	if eng.yinyunAnalyzer == nil {
		return 70.0, "音韵引擎未初始化"
	}
	return eng.yinyunAnalyzer.ScoreYinyun(name.Pinyin)
}

// analyzeMeaning 分析字义
func (eng *EnhancedNameGenerator) analyzeMeaning(name Name) (float64, string) {
	score := 80.0
	analysis := "字义分析："
	
	// 分析寓意深度
	if name.PoetrySource != "" {
		score += 10
		analysis += fmt.Sprintf("出自%s，文化底蕴深厚；", name.PoetrySource)
	}
	
	// 分析寓意积极性
	positiveWords := []string{"美", "好", "善", "德", "智", "勇", "文", "武", "福", "寿", "康", "宁"}
	meaning := name.Meaning
	positiveCount := 0
	for _, word := range positiveWords {
		if strings.Contains(meaning, word) {
			positiveCount++
		}
	}
	
	if positiveCount >= 2 {
		score += 5
		analysis += "寓意积极向上，充满正能量"
	} else if positiveCount >= 1 {
		analysis += "寓意美好，吉祥如意"
	} else {
		analysis += "寓意平和，稳重大方"
	}
	
	if score > 100 {
		score = 100
	}
	
	return score, analysis
}

// analyzeSancai 分析天地人三才
// 基于《易经·说卦传》："立天之道曰阴与阳，立地之道曰柔与刚，立人之道曰仁与义。兼三才而两之，故《易》六画而成卦。"
// 天道(阴与阳)：声调平仄交替、数理奇偶平衡 → 气的平衡
// 地道(柔与刚)：笔画搭配协调、字形结构多样 → 形的承载
// 人道(仁与义)：字义仁德内涵、诗词文化深度 → 德的修养
func (eng *EnhancedNameGenerator) analyzeSancai(name Name) (string, float64, string) {
	// 天地人三维独立评分
	tiandaoScore := eng.scoreTiandao(name)   // 天道：阴与阳
	didaoScore := eng.scoreDidao(name)       // 地道：柔与刚
	rendaoScore := eng.scoreRendao(name)     // 人道：仁与义

	// 兼三才而两之（《说卦传》）：天30% + 地30% + 人40%
	totalScore := tiandaoScore*0.30 + didaoScore*0.30 + rendaoScore*0.40
	totalScore = math.Round(totalScore*10) / 10

	analysis := fmt.Sprintf("天地人三才：天道(阴阳)%d分，地道(刚柔)%d分，人道(仁义)%d分。",
		int(tiandaoScore), int(didaoScore), int(rendaoScore))

	switch {
	case totalScore >= 85:
		analysis += " 三才兼备，天地人和，大吉之配。"
	case totalScore >= 70:
		analysis += " 三才协调，中庸和谐，平稳顺遂。"
	default:
		analysis += " 三才略有不足，建议从喜用神或诗词意境斟酌调整。"
	}

	return "天地人", totalScore, analysis
}

// scoreTiandao 天道评分：阴与阳
// 评估名字的"气"是否协调——平仄交替为阴中有阳、笔画奇偶平衡为阳中有阴
func (eng *EnhancedNameGenerator) scoreTiandao(name Name) float64 {
	score := 70.0

	// 1. 声调阴阳：平仄交替（阳平+阴平 vs 上声+去声）
	if eng.yinyunAnalyzer != nil && name.Pinyin != "" {
		pinyinParts := strings.Fields(name.Pinyin)
		if len(pinyinParts) > 1 {
			var tones []int
			for _, p := range pinyinParts {
				t := eng.yinyunAnalyzer.extractTone(p)
				if t > 0 {
					tones = append(tones, t)
				}
			}
			// 检查是否平仄交替（平声1-2，仄声3-4）
			hasPing := false
			hasZe := false
			for _, t := range tones {
				if t == 1 || t == 2 {
					hasPing = true
				} else {
					hasZe = true
				}
			}
			if hasPing && hasZe {
				score += 10 // 平仄兼备，阴阳调和
			} else if hasPing {
				score += 3 // 全平，略偏阳
			} else {
				score += 3 // 全仄，略偏阴
			}
		} else if len(pinyinParts) == 1 {
			score += 5 // 单字名，阴阳中和
		}

		// 1b. 姓氏→名字首字声调过渡评分（额外加权）
		if len(pinyinParts) >= 2 && name.Surname != "" && name.GivenName != "" {
			surnameTone := eng.yinyunAnalyzer.extractTone(pinyinParts[0])
			givenTone := eng.yinyunAnalyzer.extractTone(pinyinParts[1])
			if surnameTone > 0 && givenTone > 0 {
				surnamePing := surnameTone == 1 || surnameTone == 2
				givenPing := givenTone == 1 || givenTone == 2
				if surnamePing != givenPing {
					// 姓氏与名字首字平仄不同（如仄→平或平→仄），抑扬顿挫感强
					score += 8
				} else if surnameTone == givenTone {
					// 同声调，略显单调
					score -= 3
				}
				// 其他情况（平仄同但声调不同）不加不扣
			}
		}
	}

	// 2. 数理阴阳：名字总笔画奇偶
	if name.Strokes > 0 {
		if name.Strokes%2 == 0 {
			score += 5 // 偶数笔画，阴中有阳
		} else {
			score += 5 // 奇数笔画，阳中有阴
		}
	}

	// 3. 总笔画适中（不过偏颇）
	if name.Strokes >= 8 && name.Strokes <= 24 {
		score += 5
	}

	if score > 100 {
		score = 100
	}
	if score < 30 {
		score = 30
	}
	return score
}

// scoreDidao 地道评分：柔与刚
// 评估名字的"形"是否协调——结构搭配为刚柔相济、笔画疏密为形之匀称
func (eng *EnhancedNameGenerator) scoreDidao(name Name) float64 {
	score := 70.0

	chars := []rune(name.GivenName)
	if len(chars) >= 2 {
		char1 := string(chars[0])
		char2 := string(chars[1])

		// 1. 两字笔画差距评价刚柔搭配（使用缓存方法减少 map 查找）
		_, _, _, s1 := eng.GetCharInfo(char1)
		_, _, _, s2 := eng.GetCharInfo(char2)
		diff := s1 - s2
		if diff < 0 {
			diff = -diff
		}
		if diff <= 3 {
			score += 8 // 笔画相近，刚柔相济
		} else if diff <= 6 {
			score += 4 // 略有主次，主刚副柔
		}

		// 2. 偏旁部首多样性（使用缓存方法减少 map 查找）
		r1, _ := eng.GetCharExtra(char1)
		r2, _ := eng.GetCharExtra(char2)
		if r1 != "" && r2 != "" && r1 != r2 {
			score += 5 // 不同偏旁，结构多样
		}
	}

	// 3. 字形结构多样性评分（视觉匀称）
	if eng.nameFilter != nil && len(eng.nameFilter.structureCategory) > 0 {
		allChars := name.Surname + name.GivenName
		structures := make([]string, 0, len(allChars))
		for _, r := range allChars {
			char := string(r)
			if s, ok := eng.nameFilter.structureCategory[char]; ok {
				structures = append(structures, s)
			}
		}
		if len(structures) >= 2 {
			// 3a. 统计不同结构数量
			uniqueStructs := make(map[string]bool)
			for _, s := range structures {
				uniqueStructs[s] = true
			}
			switch len(uniqueStructs) {
			case 1:
				score -= 5 // 全同结构，视觉单调
			case 2:
				score += 3 // 两种结构交错，略有变化
			default:
				score += 8 // 三种及以上结构，丰富多样
			}

			// 3b. 检查是否连续相同结构
			for i := 0; i < len(structures)-1; i++ {
				if structures[i] == structures[i+1] {
					score -= 2 // 相邻字同结构，略显重复
					break
				}
			}
		}
	}

	// 4. 总笔画适中（地道有形不可过繁）
	if name.Strokes >= 6 && name.Strokes <= 25 {
		score += 5
	}

	if score > 100 {
		score = 100
	}
	if score < 30 {
		score = 30
	}
	return score
}

// scoreRendao 人道评分：仁与义
// 评估名字的"德"是否深厚——字义仁德为仁义之基、诗词出处为文化之根
func (eng *EnhancedNameGenerator) scoreRendao(name Name) float64 {
	score := 70.0

	// 1. 仁德字检测（人道本于仁义）
	renYiChars := []string{"仁", "义", "德", "善", "智", "信", "礼",
		"和", "正", "清", "贤", "良", "忠", "惠", "慈", "恭", "让",
		"敬", "孝", "悌", "恕"}
	for _, c := range name.GivenName {
		for _, rc := range renYiChars {
			if string(c) == rc {
				score += 10
				break
			}
		}
	}

	// 2. 诗词文化底蕴（人文传承）
	if name.PoetrySource != "" {
		score += 10
	}

	if name.PoetrySentence != "" {
		score += 5 // 有具体诗句更佳
	}

	// 3. 寓意积极正向
	positiveChars := []string{"美", "好", "善", "德", "智", "勇",
		"雅", "文", "明", "光", "辉", "华", "荣", "昌", "盛",
		"瑞", "祥", "福", "康", "宁", "安", "乐", "诗", "书"}
	for _, c := range name.GivenName {
		for _, pc := range positiveChars {
			if string(c) == pc {
				score += 5
				break
			}
		}
	}

	// 4. 含义非消极（已有NameFilter过滤，此处不做扣分）

	if score > 100 {
		score = 100
	}
	if score < 30 {
		score = 30
	}
	return score
}

// analyzeZodiac 分析生肖
func (eng *EnhancedNameGenerator) analyzeZodiac(name Name, zodiac string) (string, float64) {
	if zodiac == "" {
		return "生肖匹配良好", 80.0
	}

	eng.loadZodiacCharSets()
	goodSet := eng.zodiacGoodCharSets[zodiac]
	tabooSet := eng.zodiacTabooCharSets[zodiac]

	var details []string
	score := 70.0

	// 1. 生肖宜用字加分（O(1) set 查询，替代 O(n²) 线性扫描）
	matched := 0
	for _, r := range name.GivenName {
		if goodSet[string(r)] {
			matched++
		}
	}
	if matched > 0 {
		score += float64(matched) * 15.0
		details = append(details, fmt.Sprintf("宜用字匹配%d个", matched))
	}

	// 2. 生肖忌用字扣分（基于六冲六害关系，O(1) set 查询）
	tabooMatched := 0
	for _, r := range name.GivenName {
		if tabooSet[string(r)] {
			tabooMatched++
		}
	}
	if tabooMatched > 0 {
		penalty := float64(tabooMatched) * 20.0
		score -= penalty
		details = append(details, fmt.Sprintf("含%d个生肖忌用字，扣%.0f分", tabooMatched, penalty))
	}

	// 限制评分范围
	if score > 100 {
		score = 100
	}
	if score < 30 {
		score = 30
	}

	// 构造分析文本
	var analysis string
	switch {
	case matched > 0 && tabooMatched == 0:
		analysis = fmt.Sprintf("与%s生肖相合，%s", zodiac, strings.Join(details, "；"))
	case matched > 0 && tabooMatched > 0:
		analysis = fmt.Sprintf("与%s生肖：%s", zodiac, strings.Join(details, "；"))
	case tabooMatched > 0:
		analysis = fmt.Sprintf("与%s生肖：%s，建议避免用含六冲六害偏旁的字", zodiac, strings.Join(details, "；"))
	default:
		analysis = fmt.Sprintf("与%s生肖无冲突", zodiac)
	}

	return analysis, score
}

// getZodiacGoodChars 获取生肖宜用字
func (eng *EnhancedNameGenerator) getZodiacGoodChars(zodiac string) []string {
	// 简化版生肖宜用字
	switch zodiac {
	case "鼠":
		return []string{"米", "豆", "禾", "口", "宀", "王"}
	case "牛":
		return []string{"田", "禾", "车", "马", "艹"}
	case "虎":
		return []string{"山", "木", "王", "君", "令"}
	case "兔":
		return []string{"月", "艹", "禾", "口", "宀"}
	case "龙":
		return []string{"日", "月", "星", "云", "王", "大"}
	case "蛇":
		return []string{"艹", "木", "田", "口", "宀"}
	case "马":
		return []string{"艹", "禾", "木", "口", "宀"}
	case "羊":
		return []string{"艹", "禾", "木", "口", "宀"}
	case "猴":
		return []string{"木", "禾", "口", "宀", "王"}
	case "鸡":
		return []string{"米", "豆", "禾", "口", "宀"}
	case "狗":
		return []string{"亻", "人", "宀", "艹", "禾"}
	case "猪":
		return []string{"豆", "米", "艹", "口", "宀"}
	default:
		return []string{}
	}
}

// getZodiacTabooChars 获取生肖忌用字
// 基于生肖六冲、六害关系确定忌用字（偏旁/字根）
func (eng *EnhancedNameGenerator) getZodiacTabooChars(zodiac string) []string {
	switch zodiac {
	case "鼠":
		// 鼠马相冲(午马)、鼠羊相害(未羊)
		return []string{"马", "午", "骏", "腾", "骐", "骥", "驰", "骄", "验",
			"羊", "未", "美", "祥", "群", "善", "义", "羡"}
	case "牛":
		// 牛羊相冲(未羊)、牛马相害(午马)
		return []string{"羊", "未", "美", "祥", "群", "善", "羡", "义",
			"马", "午", "骏", "腾", "骐", "骥", "驰"}
	case "虎":
		// 虎猴相冲(申猴)、虎蛇相害(巳蛇)
		return []string{"猴", "申", "猿", "猢",
			"蛇", "巳", "虫", "虹", "蝶", "蝉", "融"}
	case "兔":
		// 兔鸡相冲(酉鸡)、兔龙相害(辰龙)
		return []string{"鸡", "酉", "凤", "鹏", "鸿", "鹤", "鹅", "鹦",
			"龙", "辰", "珑", "胧", "宠"}
	case "龙":
		// 龙狗相冲(戌狗)、龙兔相害(卯兔)、辰辰自刑
		return []string{"狗", "戌", "犬", "狼", "狈", "猛", "狮",
			"兔", "卯", "逸", "勉",
			"龙", "辰", "珑", "胧", "宠"}
	case "蛇":
		// 蛇猪相冲(亥猪)、蛇虎相害(寅虎)
		return []string{"猪", "亥", "豕", "象", "豪",
			"虎", "寅", "山", "岚", "峥", "嵘", "岭"}
	case "马":
		// 马鼠相冲(子鼠)、马牛相害(丑牛)
		return []string{"鼠", "子", "仔", "孙", "孔", "孺",
			"牛", "丑", "牛", "牲", "牢"}
	case "羊":
		// 牛羊相冲(丑牛)、羊鼠相害(子鼠)
		return []string{"牛", "丑", "牛", "牲", "牢",
			"鼠", "子", "仔", "孙", "孔", "孺"}
	case "猴":
		// 猴虎相冲(寅虎)、猴猪相害(亥猪)
		return []string{"虎", "寅", "山", "岚", "峥", "嵘", "岭",
			"猪", "亥", "豕", "象", "豪"}
	case "鸡":
		// 鸡兔相冲(卯兔)、鸡狗相害(戌狗)、酉酉自刑
		return []string{"兔", "卯", "逸", "勉",
			"狗", "戌", "犬", "狼", "狈", "猛", "狮",
			"鸡", "酉", "凤", "鹏", "鸿", "鹤", "鹅"}
	case "狗":
		// 狗龙相冲(辰龙)、狗鸡相害(酉鸡)
		return []string{"龙", "辰", "珑", "胧", "宠",
			"鸡", "酉", "凤", "鹏", "鸿", "鹤", "鹅"}
	case "猪":
		// 猪蛇相冲(巳蛇)、猪猴相害(申猴)
		return []string{"蛇", "巳", "虫", "虹", "蝶", "蝉", "融",
			"猴", "申", "猿", "猢"}
	default:
		return []string{}
	}
}

// loadZodiacCharSets 惰性初始化生肖宜用/忌用字集合（将 []string 转为 map[string]bool，O(n)→O(1)）
func (eng *EnhancedNameGenerator) loadZodiacCharSets() {
	eng.poetryZodiacOnce.Do(func() {
		eng.zodiacGoodCharSets = make(map[string]map[string]bool, 12)
		eng.zodiacTabooCharSets = make(map[string]map[string]bool, 12)
		zodiacs := []string{"鼠", "牛", "虎", "兔", "龙", "蛇", "马", "羊", "猴", "鸡", "狗", "猪"}
		for _, z := range zodiacs {
			goodSet := make(map[string]bool)
			for _, c := range eng.getZodiacGoodChars(z) {
				goodSet[c] = true
			}
			eng.zodiacGoodCharSets[z] = goodSet

			tabooSet := make(map[string]bool)
			for _, c := range eng.getZodiacTabooChars(z) {
				tabooSet[c] = true
			}
			eng.zodiacTabooCharSets[z] = tabooSet
		}
	})
}

// zodiacCharInSet 检查字符是否在生肖字集中（O(1) 查找，替代线性扫描）
func (eng *EnhancedNameGenerator) zodiacCharInSet(char, zodiac string, isGood bool) bool {
	eng.loadZodiacCharSets()
	if isGood {
		return eng.zodiacGoodCharSets[zodiac][char]
	}
	return eng.zodiacTabooCharSets[zodiac][char]
}

// calculateTotalScore 计算综合评分（使用 ScoringWeights 统一权重配置）
func (eng *EnhancedNameGenerator) calculateTotalScore(analysis *NameAnalysis) float64 {
	totalScore := analysis.WuxingScore*ScoringWeights["wuxing"] +
		float64(analysis.BaziScore)*ScoringWeights["bazi"] +
		analysis.YinyunScore*ScoringWeights["yinyun"] +
		analysis.MeaningScore*ScoringWeights["meaning"] +
		analysis.SancaiScore*ScoringWeights["sancai"] +
		analysis.ZodiacScore*ScoringWeights["zodiac"] +
		analysis.NayinScore*ScoringWeights["nayin"] +
		analysis.NameDBScore*ScoringWeights["namedb"]

	// 四舍五入到一位小数
	totalScore = math.Round(totalScore*10) / 10

	return totalScore
}

// scoreFromNameDB 从候选名库获取评分
func (eng *EnhancedNameGenerator) scoreFromNameDB(fullName string) (float64, string) {
	if eng.nameDB == nil {
		return 0, ""
	}
	
	if eng.nameDB.IsCuratedName(fullName) {
		return 95.0, "出自精选候选名库"
	}
	
	// 检查名字中的每个字是否在标准用字库中
	chars := []rune(fullName)
	matchedChars := 0
	var radicals []string
	
	for _, ch := range chars {
		charStr := string(ch)
		radical := eng.nameDB.GetCharRadical(charStr)
		if radical != "" {
			matchedChars++
			radicals = append(radicals, radical)
		}
	}
	
	if matchedChars == len(chars) {
		radicalStr := strings.Join(radicals, "、")
		return 85.0, fmt.Sprintf("各字均来源于标准起名用字库（偏旁：%s）", radicalStr)
	} else if matchedChars > 0 {
		return 75.0, "部分字来源于标准起名用字库"
	}
	
	return 70.0, "通用名字"
}

// nayinWuxingMap 纳音名称→五行映射（纳音名称末尾字即为五行属性）
var nayinWuxingMap = map[string]string{
	"金": "金",
	"火": "火",
	"木": "木",
	"水": "水",
	"土": "土",
}

// extractNayinWuxing 从纳音名称提取五行属性（如 "海中金" → "金"）
func extractNayinWuxing(nayin string) string {
	if nayin == "" {
		return ""
	}
	runes := []rune(nayin)
	if len(runes) == 0 {
		return ""
	}
	lastChar := string(runes[len(runes)-1])
	if w, ok := nayinWuxingMap[lastChar]; ok {
		return w
	}
	return ""
}

// calculateNayinScore 计算纳音五行评分
// 名字五行与出生年纳音五行的生克关系
func (eng *EnhancedNameGenerator) calculateNayinScore(nayin string, nameWuxing string) (float64, string) {
	nayinWuxing := extractNayinWuxing(nayin)
	if nayinWuxing == "" || nameWuxing == "" || nameWuxing == "未知" {
		return 70.0, "纳音五行信息不足，默认中等评分"
	}

	wuxingList := strings.Split(nameWuxing, "、")
	if len(wuxingList) == 0 {
		return 70.0, "纳音五行信息不足"
	}

	// 纳音五行与名字每个字的五行评估生克关系，取平均
	totalWeight := 0.0
	for _, w := range wuxingList {
		weight := wuxingMatchWeight(nayinWuxing, w)
		switch weight {
		case 100:
			totalWeight += 90 // 纳音与名字五行相同，吉利
		case 75:
			totalWeight += 80 // 纳音生名字五行，得纳音生扶
		case 50:
			totalWeight += 70 // 名字五行生纳音
		case 40:
			totalWeight += 55 // 名字五行克纳音，克制年命
		case 25:
			totalWeight += 45 // 纳音克名字五行，被年命克制
		default:
			totalWeight += 60
		}
	}

	score := totalWeight / float64(len(wuxingList))
	if score > 100 {
		score = 100
	}
	if score < 30 {
		score = 30
	}

	var analysis string
	switch {
	case score >= 85:
		analysis = fmt.Sprintf("纳音'%s'与名字五行相生相合，大吉", nayin)
	case score >= 70:
		analysis = fmt.Sprintf("纳音'%s'与名字五行协调，平顺", nayin)
	default:
		analysis = fmt.Sprintf("纳音'%s'与名字五行略有克制，建议斟酌", nayin)
	}

	return score, analysis
}

// generateRecommendations 生成推荐语
func (eng *EnhancedNameGenerator) generateRecommendations(analysis *NameAnalysis) []string {
	recommendations := []string{}
	
	if analysis.TotalScore >= 90 {
		recommendations = append(recommendations, "🌟 极佳选择：综合评分优秀，强烈推荐")
	} else if analysis.TotalScore >= 80 {
		recommendations = append(recommendations, "⭐ 优秀选择：综合评分良好，值得考虑")
	} else if analysis.TotalScore >= 70 {
		recommendations = append(recommendations, "👍 不错选择：综合评分中等，可以考虑")
	}
	
	if analysis.WuxingScore >= 90 {
		recommendations = append(recommendations, "五行完美：与八字喜用神高度匹配")
	}
	
	if analysis.YinyunScore >= 90 {
		recommendations = append(recommendations, "音韵优美：平仄搭配和谐，抑扬顿挫")
	} else if analysis.YinyunScore >= 80 {
		recommendations = append(recommendations, "音韵协调：读起来流畅自然")
	}
	
	if analysis.PoetrySource != "" {
		recommendations = append(recommendations, fmt.Sprintf("文化底蕴：出自%s，富有诗意", analysis.PoetrySource))
	}
	
	if analysis.SancaiScore >= 85 {
		recommendations = append(recommendations, "天地人和：天道阴阳、地道刚柔、人道仁义兼备")
	} else if analysis.SancaiScore >= 70 {
		recommendations = append(recommendations, "三才协调：天地人和谐，中庸顺遂")
	}
	
	if analysis.ZodiacScore >= 85 {
		recommendations = append(recommendations, "生肖相合：与宝宝生肖相宜")
	}
	
	if analysis.NameDBScore >= 90 && analysis.NameDBSource != "" {
		recommendations = append(recommendations, fmt.Sprintf("品质保障：%s", analysis.NameDBSource))
	}
	
	return recommendations
}

// sortByScore 按综合评分排序
func (eng *EnhancedNameGenerator) sortByScore(list []*NameAnalysis) {
	sort.Slice(list, func(i, j int) bool {
		return list[i].TotalScore > list[j].TotalScore
	})
}

// GetNameSuggestions 根据条件获取名字建议
func (eng *EnhancedNameGenerator) GetNameSuggestions(
	surname string,
	gender string,
	preferences map[string]interface{},
) ([]string, error) {
	suggestions := []string{}
	
	// 根据偏好生成建议
	if style, ok := preferences["style"].(string); ok {
		switch style {
		case "traditional":
			suggestions = append(suggestions, "建议选择传统经典的名字，如诗经楚辞中的字")
		case "modern":
			suggestions = append(suggestions, "建议选择现代简洁的名字，避免生僻字")
		case "literary":
			suggestions = append(suggestions, "建议选择富有文学气息的名字，如诗词典故")
		case "elegant":
			suggestions = append(suggestions, "建议选择优雅大方的名字，适合女孩")
		case "strong":
			suggestions = append(suggestions, "建议选择阳刚有力的名字，适合男孩")
		}
	}
	
	// 五行建议
	if wuxing, ok := preferences["wuxing"].([]string); ok && len(wuxing) > 0 {
		suggestions = append(suggestions, fmt.Sprintf("建议优先选择五行属%s的字", strings.Join(wuxing, "、")))
	}
	
	// 笔画建议
	if minStrokes, ok := preferences["min_strokes"].(int); ok {
		if maxStrokes, ok := preferences["max_strokes"].(int); ok {
			suggestions = append(suggestions, fmt.Sprintf("建议笔画数在%d-%d之间", minStrokes, maxStrokes))
		}
	}
	
	return suggestions, nil
}

// ValidateName 验证名字是否合适（使用综合过滤器）
func (eng *EnhancedNameGenerator) ValidateName(name Name) (bool, []string) {
	result := eng.nameFilter.Filter(name)
	return result.Valid, result.Issues
}

// hasBadHomophones 检查是否有不良谐音（保留接口兼容）
func (eng *EnhancedNameGenerator) hasBadHomophones(name string) bool {
	return false // 已由 NameFilter 替代
}

// ============================================================
// 统一评分体系 — 对 Name 做7维综合评分
// ============================================================

// ScoreNameUnified 对已生成的 Name 填充所有维度评分
// 这是统一评分体系的核心方法，将评分直接写回 Name 结构体
func (eng *EnhancedNameGenerator) ScoreNameUnified(name *Name, xiyongshen []string, zodiac string) {
	// 1. 五行评分（基于喜用神匹配）
	name.WuxingScore = eng.calculateWuxingScore(name.Wuxing, xiyongshen)

	// 2. 音韵评分（使用 YinyunAnalyzer）
	yinyunScore, yinyunDesc := eng.analyzeYinyun(*name)
	name.YinyunScore = yinyunScore
	if yinyunDesc != "" {
		name.Yinyun = yinyunDesc
	}

	// 3. 字义评分（评估寓意的深度和积极性）
	meaningScore, meaningDesc := eng.analyzeMeaning(*name)
	name.MeaningScore = meaningScore
	if meaningDesc != "" && name.Meaning == "" {
		// 仅在 Name 本身无寓意描述时使用
	}

	// 4. 天地人三才评分（传统易学三才，非熊崎五格数理）
	_, sancaiScore, sancaiAnalysis := eng.analyzeSancai(*name)
	name.SancaiScore = sancaiScore
	name.SancaiAnalysis = sancaiAnalysis

	// 5. 生肖评分
	if zodiac != "" {
		_, name.ZodiacScore = eng.analyzeZodiac(*name, zodiac)
	} else {
		name.ZodiacScore = 80.0
	}

	// 6. 纳音评分
	name.NayinScore, _ = eng.calculateNayinScore(name.Nayin, name.Wuxing)

	// 7. 易经卦象
	hexagram := yijing.GetHexagramByStrokes(name.Strokes)
	name.Hexagram = hexagram.Name
	name.HexagramMeaning = hexagram.Interpretation

	// 7. 计算综合评分
	name.TotalScore = eng.calculateTotalScoreFromName(name)
}

// ScoringWeights 统一多维度评分权重表
// 所有评分入口（calcPairTotalScore / calculateTotalScoreFromName）共用同一份权重配置，
// 避免预筛选排序与最终评分排序不一致的问题。
var ScoringWeights = map[string]float64{
	"wuxing":  0.20,
	"bazi":    0.15,
	"yinyun":  0.20,
	"meaning": 0.15,
	"sancai":  0.15,
	"zodiac":  0.08,
	"nayin":   0.04,
	"namedb":  0.03,
}

// calculateTotalScoreFromName 从 Name 的各维度评分计算加权总分
// 权重配置与 ScoringWeights 保持一致
func (eng *EnhancedNameGenerator) calculateTotalScoreFromName(name *Name) float64 {
	total := name.WuxingScore*ScoringWeights["wuxing"] +
		float64(name.BaZiScore)*ScoringWeights["bazi"] +
		name.YinyunScore*ScoringWeights["yinyun"] +
		name.MeaningScore*ScoringWeights["meaning"] +
		name.SancaiScore*ScoringWeights["sancai"] +
		name.ZodiacScore*ScoringWeights["zodiac"] +
		name.NayinScore*ScoringWeights["nayin"]

	// 候选名库加分
	if eng.nameDB != nil {
		if dbScore, _ := eng.scoreFromNameDB(name.FullName); dbScore > 0 {
			total += dbScore * ScoringWeights["namedb"]
		}
	}

	// 保留 Score 字段向后兼容
	name.Score = math.Round(total)

	total = math.Round(total*10) / 10
	if total > 100 {
		total = 100
	}
	if total < 0 {
		total = 0
	}
	return total
}

// ============================================================
// 双字组合评估 — 评估两个字作为名字组合的综合质量
// ============================================================

// pairEvalResult 双字组合评分结果
type pairEvalResult struct {
	chars        string // char1 + char2
	pinyin1      string
	pinyin2      string
	meaning1     string
	meaning2     string
	wuxing1      string
	wuxing2      string
	strokes1     int
	strokes2     int
	wuxingScore  float64 // 双字合并五行匹配
	yinyunScore  float64 // 两字之间的音韵和谐
	meaningScore float64 // 组合寓意
	pairPhrase   string  // 诗词共现的原句（非空表示有词组语义）
	sancaiScore  float64 // 天地人三才评分
	zodiacScore  float64 // 生肖匹配
	totalScore   float64 // 综合
}

// stripPinyinTone 去除拼音声调符号（"mínɡ" → "ming"），用于同音比较
// 基于 Unicode NFD 分解 + 过滤组合变音标记（Mn 类别），优于手写 switch
func stripPinyinTone(py string) string {
	if py == "" {
		return ""
	}

	// 先替换 ü 及其变体为 v（NFD 分解后会丢失 ü 身份）
	py = strings.NewReplacer(
		"ǖ", "v", "ǘ", "v", "ǚ", "v", "ǜ", "v", "ü", "v",
	).Replace(py)

	// NFD 分解：将带声调字符拆为基础字符 + 组合变音标记
	// 如 "míng" → "mi" + "n" + combining acute (U+0301) + "ɡ"
	nfd := norm.NFD.String(py)

	b := make([]rune, 0, len(nfd))
	for _, r := range nfd {
		// 过滤所有非间距组合标记（Mn），即去除声调符号
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		// ɡ (U+0261) NFD 无法分解，单独处理
		if r == 'ɡ' {
			b = append(b, 'g')
			continue
		}
		b = append(b, r)
	}
	return string(b)
}

// quickCheckPair 轻量预剪枝：快速检查两字组合是否值得进入完整 evalPair 评估
// 返回 false 表示可以直接跳过（明显不合格）
func (eng *EnhancedNameGenerator) quickCheckPair(char1, char2 string, opts GenerateOptions) bool {
	// 1. 笔画快速检查：单字笔画超过 40 或总和超过 60 则跳过（书写过于复杂）
	_, _, _, st1 := eng.GetCharInfo(char1)
	_, _, _, st2 := eng.GetCharInfo(char2)
	if st1 > 40 || st2 > 40 {
		return false
	}
	if st1+st2 > 60 {
		return false
	}
	// 1b. 笔画差检查：两字笔画相差超过 25，字形不协调
	if st1 > 0 && st2 > 0 {
		diff := st1 - st2
		if diff < 0 {
			diff = -diff
		}
		if diff > 25 {
			return false
		}
	}

	// 2. 五行快速检查：需要喜用神匹配时，两字五行与喜用神没有交集则跳过
	if len(opts.Xiyongshen) > 0 {
		_, _, wx1, _ := eng.GetCharInfo(char1)
		_, _, wx2, _ := eng.GetCharInfo(char2)
		allGood := false
		for _, xys := range opts.Xiyongshen {
			if xys == wx1 || xys == wx2 {
				allGood = true
				break
			}
		}
		if !allGood {
			return false
		}
	}

	// 3. 音韵快速检查：避免同音字（连续同音读起来不顺）
	// 先规范化拼音（去除声调符号，统一字母形式），再比较
	py1, _, _, _ := eng.GetCharInfo(char1)
	py2, _, _, _ := eng.GetCharInfo(char2)
	if stripPinyinTone(py1) == stripPinyinTone(py2) {
		return false
	}
	// 3b. 叠双声检查：两字声母相同且韵母读音相近（元音相同）→绕口
	// 取拼音首字母（声母），如果声母相同且去调后拼音前2字符相同则判定为绕口
	if len(py1) > 0 && len(py2) > 0 {
		init1 := string(py1[0])
		init2 := string(py2[0])
		if init1 == init2 {
			// 对大部分汉字，拼音首字母即声母；韵腹取第2字符比较
			// 若声母相同且韵腹（第2字符）也相同 → 高度叠韵，读起来绕口
			vowel1 := ""
			vowel2 := ""
			if len(py1) > 1 {
				vowel1 = string(py1[1])
			}
			if len(py2) > 1 {
				vowel2 = string(py2[1])
			}
			if vowel1 != "" && vowel1 == vowel2 {
				return false
			}
		}
	}

	return true
}

// evalPair 评估两个字作为名字组合的综合得分
func (eng *EnhancedNameGenerator) evalPair(char1, char2 string, opts GenerateOptions) pairEvalResult {
	result := pairEvalResult{
		chars: char1 + char2,
	}

	// 获取两字信息
	result.pinyin1, result.meaning1, result.wuxing1, result.strokes1 = eng.GetCharInfo(char1)
	result.pinyin2, result.meaning2, result.wuxing2, result.strokes2 = eng.GetCharInfo(char2)

	// ----- 1. 五行评分（合并两字五行 vs 喜用神）-----
	combinedWuxing := result.wuxing1 + "、" + result.wuxing2
	if result.wuxing1 == result.wuxing2 {
		combinedWuxing = result.wuxing1
	}
	result.wuxingScore = eng.calculateWuxingScore(combinedWuxing, opts.Xiyongshen)

	// ----- 2. 音韵评分（两字之间的协调性）-----
	pinyinCombo := result.pinyin1 + " " + result.pinyin2
	if eng.yinyunAnalyzer != nil {
		result.yinyunScore, _ = eng.yinyunAnalyzer.ScoreYinyun(pinyinCombo)
	} else {
		result.yinyunScore = 75.0
	}

	// ----- 3. 组合寓意评分-----
	result.meaningScore = eng.evalPairMeaning(char1, char2, result.meaning1, result.meaning2)

	// ----- 3.5 词组语义评分（诗词共现检索）-----
	if found, src, ch, sen := eng.searchPairInPoetry(char1, char2); found {
		result.pairPhrase = sen
		result.meaningScore += 12 // 诗词共现，词组有出处，+12分
		// 有具体章节出处额外加分
		if ch != "" && src != "" {
			result.meaningScore += 3 // 完整溯源 +3 分
		}
		if result.meaningScore > 100 {
			result.meaningScore = 100
		}
	}

	// ----- 4. 天地人三才评分（基于姓氏+两字名）-----
	// 构造临时 Name 以复用现有的 sancai 分析
	tempName := Name{
		Surname:   opts.Surname,
		GivenName: char1 + char2,
		Strokes:   result.strokes1 + result.strokes2,
	}
	_, result.sancaiScore, _ = eng.analyzeSancai(tempName)

	// ----- 5. 生肖评分（检查两字是否都宜用）-----
	if opts.Zodiac != "" {
		_, result.zodiacScore = eng.analyzeZodiac(tempName, opts.Zodiac)
	} else {
		result.zodiacScore = 80.0
	}

	// ----- 6. 综合评分（加权）-----
	baziScore := 50.0
	if len(opts.Xiyongshen) > 0 {
		baziScore = 80.0
	}
	_ = baziScore

	result.totalScore = eng.calcPairTotalScore(result, opts.Xiyongshen)

	return result
}

// evalPairMeaning 评估两字组合的寓意深度
func (eng *EnhancedNameGenerator) evalPairMeaning(char1, char2, meaning1, meaning2 string) float64 {
	score := 75.0

	// 1. 两字都不是默认未知寓意
	if meaning1 != "美好寓意" && meaning1 != "" {
		score += 5
	}
	if meaning2 != "美好寓意" && meaning2 != "" {
		score += 5
	}

	// 2. 两字含义有协同性
	combined := char1 + char2
	// 检查是否有诗词典故关联
	if eng.nameDB != nil {
		if eng.nameDB.IsCuratedName(combined) {
			score += 10 // 在候选名库中，寓意质量有保障
		}
	}

	// 3. 积极正向字检测
	positiveWords := []string{"美", "好", "善", "德", "智", "勇", "仁", "义",
		"福", "寿", "康", "宁", "安", "乐", "瑞", "祥",
		"文", "武", "英", "杰", "俊", "贤", "雅",
		"明", "光", "辉", "华", "荣", "昌", "盛",
		"清", "正", "直", "刚", "毅", "坚", "和", "平",
		"芳", "兰", "芝", "玉", "雪", "月", "风", "云",
		"诗", "书", "礼", "乐", "星", "辰", "海", "山"}

	positiveCount := 0
	for _, w := range positiveWords {
		if w == char1 || w == char2 {
			positiveCount++
		}
	}
	score += float64(positiveCount) * 5
	if positiveCount >= 2 {
		score += 3 // 两字都是积极字，额外加分
	}

	if score > 100 {
		score = 100
	}
	return score
}

// calcPairTotalScore 计算双字组合综合评分
// 使用与 calculateTotalScoreFromName 相同的 ScoringWeights 配置，
// 仅使用预筛阶段可得的维度（wuxing/yinyun/meaning/sancai/zodiac），
// 不可得维度（bazi/nayin/namedb）通过 baziBonus 近似补偿。
func (eng *EnhancedNameGenerator) calcPairTotalScore(r pairEvalResult, xiyongshen []string) float64 {
	// 使用统一权重，仅取预筛可用的维度
	available := []string{"wuxing", "yinyun", "meaning", "sancai", "zodiac"}
	total := 0.0
	weightSum := 0.0
	for _, dim := range available {
		switch dim {
		case "wuxing":
			total += r.wuxingScore * ScoringWeights[dim]
		case "yinyun":
			total += r.yinyunScore * ScoringWeights[dim]
		case "meaning":
			total += r.meaningScore * ScoringWeights[dim]
		case "sancai":
			total += r.sancaiScore * ScoringWeights[dim]
		case "zodiac":
			total += r.zodiacScore * ScoringWeights[dim]
		}
		weightSum += ScoringWeights[dim]
	}

	// 八字匹配加分（近似补偿不可得的 bazi 维度）
	if len(xiyongshen) > 0 && r.wuxingScore >= 80 {
		total += 5.0 * ScoringWeights["bazi"] // 归一化到统一权重尺度
		weightSum += ScoringWeights["bazi"]
	}

	// 归一化：使权重和恢复为 1.0，与最终评分尺度一致
	if weightSum > 0 {
		total /= weightSum
	}

	if total > 100 {
		total = 100
	}
	if total < 0 {
		total = 0
	}
	return math.Round(total*10) / 10
}

// ============================================================
// 评分引导生成 — 统一评分 + 双字组合评估
// ============================================================

// GenerateUnified 使用统一评分体系生成名字
// 核心改进：
//   - 双字名：遍历所有字符组合，对每对进行综合评分，按分排序
//   - 评分覆盖：五行、音韵、寓意、天地人三才、生肖、卦象
//   - 低分自动淘汰（<65分）
//   - 候选名库名字优先展示
func (eng *EnhancedNameGenerator) GenerateUnified(opts GenerateOptions) ([]Name, error) {
	if opts.Surname == "" {
		return nil, fmt.Errorf("姓氏不能为空")
	}
	if opts.Count <= 0 {
		opts.Count = 20
	}

	// 1. 构建候选字符池
	candidates := eng.buildCandidates(opts)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("无符合条件的候选字")
	}

	var names []Name
	generated := make(map[string]bool)

	if opts.NameLength == 2 || opts.NameType == "double" {
		// 双字名：组合评估
		names = eng.generateDoubleNames(opts, candidates, generated)
	} else {
		// 单字名：逐个评估
		names = eng.generateSingleNames(opts, candidates, generated)
	}

	if len(names) == 0 {
		return nil, fmt.Errorf("未生成符合条件的名字")
	}

	// 3. 综合排序
	sort.Slice(names, func(i, j int) bool {
		return names[i].TotalScore > names[j].TotalScore
	})

	// 4. 填充FullName和ID
	for i := range names {
		names[i].FullName = opts.Surname + names[i].GivenName
		names[i].ID = int64(i + 1)
	}

	// 5. 限制返回数量
	if len(names) > opts.Count {
		names = names[:opts.Count]
	}

	return names, nil
}

// buildCandidates 构建候选字符池（应用基础过滤：性别/笔画/生僻字/五行）
func (eng *EnhancedNameGenerator) buildCandidates(opts GenerateOptions) []string {
	charMap := make(map[string]bool)

	// 从性别用字表加载
	var baseChars []string
	if opts.Gender == "male" {
		baseChars = eng.GetMaleNames()
	} else {
		baseChars = eng.GetFemaleNames()
	}
	for _, c := range baseChars {
		charMap[c] = true
	}

	// 如果有诗词来源，加入诗词用字
	if opts.SourceClassic != "" {
		poetryChars := GetPoetryNames(opts.Gender, opts.SourceClassic)
		for _, c := range poetryChars {
			charMap[c] = true
		}
	}

	// 扩展字池：使用预计算的性别候选字池替代全表扫描 HanziData
	// 只有当基础池较小或未指定特定来源时扩展，以保证名字多样性
	if opts.SourceClassic == "" && !opts.IncludePoetry && !opts.IncludeClassic {
		prebuilt := eng.prebuiltMaleChars
		if opts.Gender == "female" {
			prebuilt = eng.prebuiltFemaleChars
		}
		for _, c := range prebuilt {
			if charMap[c] {
				continue // 已在池中
			}
			charMap[c] = true
		}
	}

	// 避讳长辈：构建同形字与同音拼音排除集
	elderCharSet, elderPinyinSet := buildElderAvoidanceSets(opts.AvoidElderNames)

	// 大名需大命：普通格局排除敏感字
	allowSensitive := allowSensitiveChar(opts.DayMasterStrength)

	// 应用过滤
	var result []string
	for c := range charMap {
		// 避讳长辈：排除同形字
		if elderCharSet[c] {
			continue
		}
		// 避讳长辈：排除同音字
		if len(elderPinyinSet) > 0 {
			cp := GetPinyin(c)
			if elderPinyinSet[cp] {
				continue
			}
		}
		// 大名需大命：普通格局排除敏感字
		if !allowSensitive && isSensitiveNameChar(c) {
			continue
		}

		// 生僻字过滤
		if opts.ExcludeRare {
			if IsRareChar(c) {
				continue
			}
			if !hanzi.IsCommonChar(c) {
				continue
			}
		}

		// 笔画过滤
		strokes := 10
		if h, ok := hanzi.HanziData[c]; ok {
			strokes = h.Strokes
		}
		if opts.MinStrokes > 0 && strokes < opts.MinStrokes {
			continue
		}
		if opts.MaxStrokes > 0 && strokes > opts.MaxStrokes {
			continue
		}

		// 拼音首字母过滤
		if opts.PinyinInitial != "" {
			pinyin := GetPinyin(c)
			if len(pinyin) > 0 && string(pinyin[0]) != opts.PinyinInitial {
				continue
			}
		}

		// 五行过滤
		if len(opts.WuxingMatch) > 0 {
			wx := ""
			if h, ok := hanzi.HanziData[c]; ok {
				wx = h.Wuxing
			}
			matched := false
			for _, w := range opts.WuxingMatch {
				if wx == w {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		// 硬过滤（安全策略）
		tmpName := Name{FullName: opts.Surname + c, GivenName: c}
		if eng.nameFilter != nil {
			result := eng.nameFilter.Filter(tmpName)
			if !result.Valid {
				continue
			}
		}

			result = append(result, c)
	}

	// 候选字数量剪枝：当候选字超过 100 时，按精选等级+常用等级排序保留 Top 100
	// 双字名组合复杂度为 O(n²)，100 字 → 10K 组合，22,500→10,000 (-55%)
	// 优先保留高精选等级的字，确保质量不因减少候选数而明显下降
	const maxCandidates = 100
	if len(result) > maxCandidates {
		sort.SliceStable(result, func(i, j int) bool {
			hi, hji := hanzi.HanziData[result[i]], hanzi.HanziData[result[j]]
			// 优先 CurationLevel，次优 UsageLevel
			if hi.CurationLevel != hji.CurationLevel {
				return hi.CurationLevel > hji.CurationLevel
			}
			return hi.UsageLevel > hji.UsageLevel
		})
		result = result[:maxCandidates]
	}

	// 去重
	seen := make(map[string]bool)
	var unique []string
	for _, c := range result {
		if !seen[c] {
			seen[c] = true
			unique = append(unique, c)
		}
	}

	return unique
}

// generateDoubleNames 双字名生成：遍历所有组合，评分排序
func (eng *EnhancedNameGenerator) generateDoubleNames(opts GenerateOptions, candidates []string, generated map[string]bool) []Name {
	type scoredPair struct {
		chars string
		score float64
	}

	// 先进行候选名库优先匹配
	var prebuilt []Name

	// 1. 候选名库优先
	if eng.nameDB != nil {
		curatedList := eng.nameDB.GetCuratedNames(opts.Gender)
		for _, cn := range curatedList {
			if len(cn.Name) < 2 {
				continue
			}
			if generated[cn.Name] {
				continue
			}
			// 检查性别
			if opts.Gender == "male" && cn.Gender == "female" {
				continue
			}
			if opts.Gender == "female" && cn.Gender == "male" {
				continue
			}
			generated[cn.Name] = true
			name := eng.buildNameFromPair(cn.Name, opts)
			if name != nil {
				prebuilt = append(prebuilt, *name)
			}
		}
	}

	// 2. 遍历字符组合，预评分筛选
	var scored []scoredPair
	threshold := 65.0 // 最低及格分

	for _, c1 := range candidates {
		for _, c2 := range candidates {
			// 跳过重复字组合
			if c1 == c2 {
				continue
			}

			chars := c1 + c2
			if generated[chars] {
				continue
			}

			// 硬过滤（安全策略）
			tmpName := Name{
				Surname:   opts.Surname,
				GivenName: chars,
				FullName:  opts.Surname + chars,
			}
			if eng.nameFilter != nil {
				fr := eng.nameFilter.Filter(tmpName)
				if !fr.Valid {
					continue
				}
			}

			// 轻量预剪枝：快速排除明显不合格的组合（避免全量评估开销）
			if !eng.quickCheckPair(c1, c2, opts) {
				continue
			}

			// 双字组合综合评估
			eval := eng.evalPair(c1, c2, opts)

			if eval.totalScore >= threshold {
				scored = append(scored, scoredPair{chars, eval.totalScore})
			}
		}
	}

	// 3. 按评分排序
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// 限制处理量（最多取前200个做完整分析）
	maxProcess := 200
	if len(scored) > maxProcess {
		scored = scored[:maxProcess]
	}

	// 4. 构建 Name 并填充完整评分
	var names []Name
	for _, sp := range scored {
		if generated[sp.chars] {
			continue
		}
		generated[sp.chars] = true
		name := eng.buildNameFromPair(sp.chars, opts)
		if name != nil {
			names = append(names, *name)
		}
	}

	// 5. 候选名库名字放在最前面（确保优先展示）
	allNames := append(prebuilt, names...)
	return allNames
}

// generateSingleNames 单字名生成
func (eng *EnhancedNameGenerator) generateSingleNames(opts GenerateOptions, candidates []string, generated map[string]bool) []Name {
	var names []Name
	for _, c := range candidates {
		if generated[c] {
			continue
		}
		generated[c] = true

		pinyin, meaning, wuxing, strokes := eng.GetCharInfo(c)

		name := Name{
			Surname:   opts.Surname,
			GivenName: c,
			Pinyin:    pinyin,
			Meaning:   meaning,
			Wuxing:    wuxing,
			Strokes:   strokes,
			Gender:    opts.Gender,
		}

		// 诗词溯源
		if opts.SourceClassic != "" || opts.IncludePoetry {
			if _, m, src, chapter, sentence, wx := eng.GetPoetryCharInfo(c); m != "" {
				meaning = m
				name.PoetrySource = src
				name.PoetryChapter = chapter
				name.PoetrySentence = sentence
				if chapter != "" {
					meaning += "（出自" + src + "《" + chapter + "》"
					if sentence != "" {
						meaning += "：" + sentence
					}
					meaning += "）"
				} else if src != "" {
					meaning += "（出自" + src + "）"
				}
				name.Meaning = meaning
				if wx != "未知" {
					name.Wuxing = wx
				}
			}
		}

		// 统一评分
		eng.ScoreNameUnified(&name, opts.Xiyongshen, opts.Zodiac)

		// 低分淘汰
		if name.TotalScore < 65 {
			continue
		}

		names = append(names, name)
	}
	return names
}

// buildNameFromPair 从双字组合构建 Name 对象（含完整评分）
func (eng *EnhancedNameGenerator) buildNameFromPair(chars string, opts GenerateOptions) *Name {
	if len(chars) < 2 {
		return nil
	}

	runes := []rune(chars)
	char1 := string(runes[0])
	char2 := string(runes[1])

	pinyin1, meaning1, wuxing1, strokes1 := eng.GetCharInfo(char1)
	pinyin2, meaning2, wuxing2, strokes2 := eng.GetCharInfo(char2)

	pinyin := pinyin1 + " " + pinyin2
	strokes := strokes1 + strokes2

	// 合并五行
	combinedWuxing := wuxing1 + "、" + wuxing2
	if wuxing1 == wuxing2 {
		combinedWuxing = wuxing1
	}

	// 合并寓意
	meaning := meaning1 + "、" + meaning2

	// 诗词溯源（优先双字共现检索，回退到单字溯源）
	poetrySrc, poetryChapter, poetrySentence := "", "", ""
	if opts.SourceClassic != "" || opts.IncludePoetry {
		// 先尝试双字共现检索（词组语义）
		if found, src, ch, sen := eng.searchPairInPoetry(char1, char2); found {
			poetrySrc = src
			poetryChapter = ch
			poetrySentence = sen
			if ch != "" {
				meaning = meaning1 + "、" + meaning2
				meaning += "（" + src + "《" + ch + "》" + sen + "）"
			} else {
				meaning = meaning1 + "、" + meaning2 + "（出自" + src + "）"
			}
		} else {
			// 回退到单字溯源
			if _, m1, src1, ch1, sen1, _ := eng.GetPoetryCharInfo(char1); m1 != "" {
				poetrySrc = src1
				poetryChapter = ch1
				poetrySentence = sen1
			}
			if _, m2, src2, ch2, sen2, _ := eng.GetPoetryCharInfo(char2); m2 != "" {
				if poetrySrc == "" {
					poetrySrc = src2
					poetryChapter = ch2
					poetrySentence = sen2
				}
			}
			if poetrySrc != "" {
				if poetryChapter != "" {
					meaning = meaning1 + "（出自" + poetrySrc + "《" + poetryChapter + "》）" +
						"、" + meaning2
				} else {
					meaning = meaning1 + "（出自" + poetrySrc + "）" + "、" + meaning2
				}
			}
		}
	}

	name := &Name{
		Surname:        opts.Surname,
		GivenName:      chars,
		Pinyin:         pinyin,
		Meaning:        meaning,
		Wuxing:         combinedWuxing,
		Strokes:        strokes,
		Gender:         opts.Gender,
		PoetrySource:   poetrySrc,
		PoetryChapter:  poetryChapter,
		PoetrySentence: poetrySentence,
	}

	// 统一评分
	eng.ScoreNameUnified(name, opts.Xiyongshen, opts.Zodiac)

	return name
}

// loadPoetrySentences 惰性加载所有诗词句子（去重），同时构建字符串→句子倒排索引
// 以及 charSentenceSet（O(1) 交集判断）和 sentenceInfo（快速获取原句元数据）
func (eng *EnhancedNameGenerator) loadPoetrySentences() {
	eng.poetrySentencesOnce.Do(func() {
		eng.poetryCharIndex = make(map[string][]poetrySentenceEntry)
		eng.poetryCharSentenceSet = make(map[string]map[string]bool)
		eng.poetrySentenceInfo = make(map[string]poetrySentenceEntry)
		seen := make(map[string]bool)
		for _, ps := range classics.PoetrySources {
			for _, pc := range ps.Chars {
				if pc.Sentence == "" || seen[pc.Sentence] {
					continue
				}
				seen[pc.Sentence] = true
				entry := poetrySentenceEntry{
					Source:   pc.Work,
					Chapter:  pc.Chapter,
					Sentence: pc.Sentence,
				}
				eng.poetrySentences = append(eng.poetrySentences, entry)
				eng.poetrySentenceInfo[pc.Sentence] = entry

				// 对句中每个字符建立倒排索引 + 句子集合
				seenChar := make(map[rune]bool)
				for _, r := range pc.Sentence {
					if seenChar[r] {
						continue
					}
					seenChar[r] = true
					char := string(r)
					eng.poetryCharIndex[char] = append(eng.poetryCharIndex[char], entry)
					if eng.poetryCharSentenceSet[char] == nil {
						eng.poetryCharSentenceSet[char] = make(map[string]bool)
					}
					eng.poetryCharSentenceSet[char][pc.Sentence] = true
				}
			}
		}
	})
}

// searchPairInPoetry 搜索两个字是否在诗词句子中共现（使用预构建集合，零分配）
// 例如 "星驰" → "星驰" 同时出现在某诗句中则返回出处
// 返回：(是否共现, 出处, 章节, 完整诗句)
func (eng *EnhancedNameGenerator) searchPairInPoetry(char1, char2 string) (bool, string, string, string) {
	if char1 == "" || char2 == "" {
		return false, "", "", ""
	}

	eng.loadPoetrySentences()

	set1 := eng.poetryCharSentenceSet[char1]
	set2 := eng.poetryCharSentenceSet[char2]
	if len(set1) == 0 || len(set2) == 0 {
		return false, "", "", ""
	}

	// 遍历较小的集合，在较大的集合中做 O(1) 查询
	if len(set1) > len(set2) {
		set1, set2 = set2, set1
		char1, char2 = char2, char1
	}

	for sent := range set1 {
		if set2[sent] {
			// 共现命中，从 sentenceInfo 中 O(1) 取出处元数据
			if info, ok := eng.poetrySentenceInfo[sent]; ok {
				return true, info.Source, info.Chapter, info.Sentence
			}
		}
	}

	return false, "", "", ""
}
