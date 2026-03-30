package name

import (
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/bliubiu/babyName/internal/domain/hanzi"
	"github.com/bliubiu/babyName/internal/domain/yijing"
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
	
	// 三才五格分析
	Sancai         string   `json:"sancai"`
	SancaiScore    float64  `json:"sancai_score"`
	SancaiAnalysis string   `json:"sancai_analysis"`
	
	// 生肖分析
	ZodiacMatch    string   `json:"zodiac_match"`
	ZodiacScore    float64  `json:"zodiac_score"`
	
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
	analysisCache map[string]*NameAnalysis
	cacheMu       sync.RWMutex
}

// NewEnhancedNameGenerator 创建增强版名字生成器
func NewEnhancedNameGenerator() *EnhancedNameGenerator {
	return &EnhancedNameGenerator{
		NameGenerator: NewNameGenerator(),
		analysisCache: make(map[string]*NameAnalysis),
	}
}

// GenerateNamesWithAnalysis 生成带详细分析的名字
func (eng *EnhancedNameGenerator) GenerateNamesWithAnalysis(
	surname string,
	generation string,
	gender string,
	xiyongshen []string,
	count int,
	nameLength int,
	excludeRare bool,
	wuxingMatch []string,
	sourceClassic string,
	minStrokes int,
	maxStrokes int,
	includePoetry bool,
	includeClassic bool,
	meaningKeywords []string,
	pinyinInitial string,
	generationPosition string,
	nameType string,
	zodiac string,
) ([]*NameAnalysis, error) {
	// 生成基础名字列表
	names := eng.GenerateNamesWithGeneration(
		surname, generation, gender, xiyongshen, count, nameLength,
		excludeRare, wuxingMatch, sourceClassic, minStrokes, maxStrokes,
		includePoetry, includeClassic, meaningKeywords, pinyinInitial,
		generationPosition, nameType,
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
			
			analysis := eng.AnalyzeName(n, xiyongshen, zodiac)
			
			mu.Lock()
			analysisList[index] = analysis
			mu.Unlock()
		}(i, name)
	}
	
	wg.Wait()
	
	// 按综合评分排序
	eng.sortByScore(analysisList)
	
	return analysisList, nil
}

// AnalyzeName 详细分析单个名字
func (eng *EnhancedNameGenerator) AnalyzeName(name Name, xiyongshen []string, zodiac string) *NameAnalysis {
	// 检查缓存
	cacheKey := fmt.Sprintf("%s%s", name.Surname, name.GivenName)
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
	
	// 6. 三才五格分析
	analysis.Sancai, analysis.SancaiScore, analysis.SancaiAnalysis = eng.analyzeSancai(name)
	
	// 7. 生肖分析
	analysis.ZodiacMatch, analysis.ZodiacScore = eng.analyzeZodiac(name, zodiac)
	
	// 8. 诗词典故
	analysis.PoetrySource = name.PoetrySource
	analysis.PoetryChapter = name.PoetryChapter
	analysis.PoetrySentence = name.PoetrySentence
	
	// 9. 计算综合评分
	analysis.TotalScore = eng.calculateTotalScore(analysis)
	
	// 10. 生成推荐语
	analysis.Recommendations = eng.generateRecommendations(analysis)
	
	// 缓存结果
	eng.cacheMu.Lock()
	eng.analysisCache[cacheKey] = analysis
	eng.cacheMu.Unlock()
	
	return analysis
}

// analyzeWuxing 分析五行
func (eng *EnhancedNameGenerator) analyzeWuxing(wuxing string, xiyongshen []string) string {
	if wuxing == "未知" || len(xiyongshen) == 0 {
		return "五行属性明确，有助于平衡命理"
	}
	
	wuxingList := strings.Split(wuxing, "、")
	matched := 0
	for _, w := range wuxingList {
		for _, xy := range xiyongshen {
			if strings.Contains(w, xy) {
				matched++
				break
			}
		}
	}
	
	if matched == len(wuxingList) {
		return fmt.Sprintf("名字五行属%s，与八字喜用神%s完美匹配，大吉", wuxing, strings.Join(xiyongshen, "、"))
	} else if matched > 0 {
		return fmt.Sprintf("名字五行属%s，与八字喜用神%s部分匹配，吉", wuxing, strings.Join(xiyongshen, "、"))
	}
	
	return fmt.Sprintf("名字五行属%s，建议结合八字喜用神%s综合考虑", wuxing, strings.Join(xiyongshen, "、"))
}

// calculateWuxingScore 计算五行评分
func (eng *EnhancedNameGenerator) calculateWuxingScore(wuxing string, xiyongshen []string) float64 {
	if wuxing == "未知" || len(xiyongshen) == 0 {
		return 70.0
	}
	
	wuxingList := strings.Split(wuxing, "、")
	matched := 0
	for _, w := range wuxingList {
		for _, xy := range xiyongshen {
			if strings.Contains(w, xy) {
				matched++
				break
			}
		}
	}
	
	score := float64(matched) / float64(len(wuxingList)) * 100.0
	if score < 50 {
		score = 50 + score/2
	}
	
	return score
}

// analyzeYinyun 分析音韵
func (eng *EnhancedNameGenerator) analyzeYinyun(name Name) (float64, string) {
	pinyinParts := strings.Split(name.Pinyin, " ")
	if len(pinyinParts) == 0 {
		return 70.0, "音韵和谐，朗朗上口"
	}
	
	// 分析声调组合
	score := 85.0
	analysis := "音韵分析："
	
	// 检查平仄搭配
	pingze := eng.getPingze(pinyinParts)
	if len(pingze) >= 2 {
		if pingze[0] != pingze[1] {
			score += 5
			analysis += "平仄相间，抑扬顿挫；"
		} else {
			analysis += "声调和谐，流畅自然；"
		}
	}
	
	// 检查韵母搭配
	if len(pinyinParts) >= 2 {
		rhyme1 := eng.getRhyme(pinyinParts[0])
		rhyme2 := eng.getRhyme(pinyinParts[1])
		if rhyme1 != rhyme2 {
			score += 5
			analysis += "韵母变化丰富，避免重复"
		} else {
			analysis += "韵母相近，朗朗上口"
		}
	}
	
	if score > 100 {
		score = 100
	}
	
	return score, analysis
}

// getPingze 获取平仄
func (eng *EnhancedNameGenerator) getPingze(pinyinParts []string) []string {
	// 简化处理，实际应该根据声调判断
	result := []string{}
	for _, p := range pinyinParts {
		if len(p) > 0 {
			// 根据韵母判断平仄（简化版）
			if strings.Contains("aoeiuü", string(p[len(p)-1])) {
				result = append(result, "平")
			} else {
				result = append(result, "仄")
			}
		}
	}
	return result
}

// getRhyme 获取韵母
func (eng *EnhancedNameGenerator) getRhyme(pinyin string) string {
	// 简化处理，提取韵母
	vowels := "aoeiuü"
	rhyme := ""
	for i := len(pinyin) - 1; i >= 0; i-- {
		if strings.Contains(vowels, string(pinyin[i])) {
			rhyme = string(pinyin[i]) + rhyme
		} else if rhyme != "" {
			break
		}
	}
	return rhyme
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

// analyzeSancai 分析三才五格
func (eng *EnhancedNameGenerator) analyzeSancai(name Name) (string, float64, string) {
	// 简化版三才五格分析
	
	// 计算天格、人格、地格
	tiang := eng.getTiang(name.Surname)
	reng := eng.getReng(name.Surname, name.GivenName)
	deg := eng.getDeg(name.GivenName)
	
	// 三才配置
	sancai := fmt.Sprintf("%s%s%s", 
		eng.getWuxingByStrokes(tiang),
		eng.getWuxingByStrokes(reng),
		eng.getWuxingByStrokes(deg))
	
	// 评分
	score := 75.0
	analysis := fmt.Sprintf("三才配置：天格%d(%s)，人格%d(%s)，地格%d(%s)。",
		tiang, eng.getWuxingByStrokes(tiang),
		reng, eng.getWuxingByStrokes(reng),
		deg, eng.getWuxingByStrokes(deg))
	
	// 三才吉凶判断
	if strings.Contains(sancai, "木火土") || strings.Contains(sancai, "金水木") {
		score += 15
		analysis += "三才配置大吉，基础稳固，成功运佳"
	} else if strings.Contains(sancai, "火土金") || strings.Contains(sancai, "土金水") {
		score += 10
		analysis += "三才配置吉，境遇安泰，身心健全"
	} else {
		analysis += "三才配置平稳，宜稳健发展"
	}
	
	if score > 100 {
		score = 100
	}
	
	return sancai, score, analysis
}

// getTiang 计算天格
func (eng *EnhancedNameGenerator) getTiang(surname string) int {
	// 简化计算，实际应该根据姓氏笔画
	return len(surname) * 5
}

// getReng 计算人格
func (eng *EnhancedNameGenerator) getReng(surname, givenName string) int {
	// 简化计算
	return len(surname)*5 + len(givenName)*3
}

// getDeg 计算地格
func (eng *EnhancedNameGenerator) getDeg(givenName string) int {
	// 简化计算
	return len(givenName) * 5
}

// getWuxingByStrokes 根据笔画数获取五行
func (eng *EnhancedNameGenerator) getWuxingByStrokes(strokes int) string {
	// 1、2为木，3、4为火，5、6为土，7、8为金，9、0为水
	remainder := strokes % 10
	switch remainder {
	case 1, 2:
		return "木"
	case 3, 4:
		return "火"
	case 5, 6:
		return "土"
	case 7, 8:
		return "金"
	case 9, 0:
		return "水"
	default:
		return "土"
	}
}

// analyzeZodiac 分析生肖
func (eng *EnhancedNameGenerator) analyzeZodiac(name Name, zodiac string) (string, float64) {
	if zodiac == "" {
		return "生肖匹配良好", 80.0
	}
	
	// 生肖宜用字分析
	goodChars := eng.getZodiacGoodChars(zodiac)
	nameChars := strings.Split(name.GivenName, "")
	
	matched := 0
	for _, char := range nameChars {
		for _, goodChar := range goodChars {
			if char == goodChar {
				matched++
				break
			}
		}
	}
	
	score := 70.0 + float64(matched)*15.0
	if score > 100 {
		score = 100
	}
	
	if matched > 0 {
		return fmt.Sprintf("与%s生肖相合，宜用字匹配%d个", zodiac, matched), score
	}
	
	return fmt.Sprintf("与%s生肖无冲突", zodiac), score
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

// calculateTotalScore 计算综合评分
func (eng *EnhancedNameGenerator) calculateTotalScore(analysis *NameAnalysis) float64 {
	// 权重配置
	weights := map[string]float64{
		"wuxing":  0.25,
		"bazi":    0.20,
		"yinyun":  0.15,
		"meaning": 0.15,
		"sancai":  0.15,
		"zodiac":  0.10,
	}
	
	// 计算加权平均分
	totalScore := analysis.WuxingScore*weights["wuxing"] +
		float64(analysis.BaziScore)*weights["bazi"] +
		analysis.YinyunScore*weights["yinyun"] +
		analysis.MeaningScore*weights["meaning"] +
		analysis.SancaiScore*weights["sancai"] +
		analysis.ZodiacScore*weights["zodiac"]
	
	// 四舍五入到一位小数
	totalScore = math.Round(totalScore*10) / 10
	
	return totalScore
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
		recommendations = append(recommendations, "音韵优美：声调搭配和谐，朗朗上口")
	}
	
	if analysis.PoetrySource != "" {
		recommendations = append(recommendations, fmt.Sprintf("文化底蕴：出自%s，富有诗意", analysis.PoetrySource))
	}
	
	if analysis.SancaiScore >= 85 {
		recommendations = append(recommendations, "三才吉配：天格人格地格配置优良")
	}
	
	if analysis.ZodiacScore >= 85 {
		recommendations = append(recommendations, "生肖相合：与宝宝生肖相宜")
	}
	
	return recommendations
}

// sortByScore 按综合评分排序
func (eng *EnhancedNameGenerator) sortByScore(list []*NameAnalysis) {
	// 使用冒泡排序（数据量小，简单即可）
	n := len(list)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if list[j].TotalScore < list[j+1].TotalScore {
				list[j], list[j+1] = list[j+1], list[j]
			}
		}
	}
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

// ValidateName 验证名字是否合适
func (eng *EnhancedNameGenerator) ValidateName(name Name) (bool, []string) {
	issues := []string{}
	
	// 检查生僻字
	for _, char := range name.GivenName {
		if !hanzi.IsCommonChar(string(char)) {
			issues = append(issues, fmt.Sprintf("'%c'为生僻字，可能影响日常使用", char))
		}
	}
	
	// 检查谐音
	if eng.hasBadHomophones(name.FullName) {
		issues = append(issues, "名字可能存在不良谐音")
	}
	
	// 检查重复字
	charCount := make(map[rune]int)
	for _, char := range name.GivenName {
		charCount[char]++
		if charCount[char] > 1 {
			issues = append(issues, "名字中不应有重复字")
			break
		}
	}
	
	return len(issues) == 0, issues
}

// hasBadHomophones 检查是否有不良谐音
func (eng *EnhancedNameGenerator) hasBadHomophones(name string) bool {
	// 简化版谐音检查
	badWords := []string{"死", "病", "穷", "苦", "难", "灾", "祸"}
	for _, word := range badWords {
		if strings.Contains(name, word) {
			return true
		}
	}
	return false
}
