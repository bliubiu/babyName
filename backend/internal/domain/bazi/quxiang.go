package bazi

// QuxiangLayer 取象层级标识
type QuxiangLayer string

const (
	LayerWuxing  QuxiangLayer = "wuxing"  // 第1层：五行取象
	LayerTenGod  QuxiangLayer = "ten_god" // 第2层：十神取象
	LayerShensha QuxiangLayer = "shensha" // 第3层：神煞取象
	LayerPillar  QuxiangLayer = "pillar"  // 第4层：柱位取象
)

// LayerImage 单层取象结果
type LayerImage struct {
	Layer   QuxiangLayer `json:"layer"`   // 层级标识
	Title   string       `json:"title"`   // 层级标题
	Content string       `json:"content"` // 取象描述
	Summary string       `json:"summary"` // 一句话总结
}

// QuxiangResult 四层取象合成结果
type QuxiangResult struct {
	WuxingImage  *LayerImage `json:"wuxing_image"`  // 第1层：五行取象
	TenGodImage  *LayerImage `json:"ten_god_image"` // 第2层：十神取象
	ShenshaImage *LayerImage `json:"shensha_image"` // 第3层：神煞取象
	PillarImage  *LayerImage `json:"pillar_image"`  // 第4层：柱位取象
	FullProfile  string      `json:"full_profile"`  // 综合命理描述
}

// GenerateQuxiang 生成四层取象
// 整合五行、十神、神煞、柱位四层取象为一套完整的命理描述
func GenerateQuxiang(analysis *BaziAnalysis) *QuxiangResult {
	if analysis == nil {
		return nil
	}

	result := &QuxiangResult{}

	// 第1层：五行取象
	result.WuxingImage = generateWuxingImage(analysis)

	// 第2层：十神取象（需要十神计算结果，暂留空由外部填入）
	// 调用处可通过 result.TenGodImage = BuildTenGodImage(...) 补充

	// 第3层：神煞取象（需要神煞计算结果，暂留空由外部填入）

	// 第4层：柱位取象（需要柱位分析结果，暂留空由外部填入）

	// 合成综合描述
	result.FullProfile = composeFullProfile(result)

	return result
}

// SetTenGodImage 设置十神取象层（由外部调用补充）
func (q *QuxiangResult) SetTenGodImage(img *LayerImage) {
	q.TenGodImage = img
	q.FullProfile = composeFullProfile(q)
}

// SetShenshaImage 设置神煞取象层
func (q *QuxiangResult) SetShenshaImage(img *LayerImage) {
	q.ShenshaImage = img
	q.FullProfile = composeFullProfile(q)
}

// SetPillarImage 设置柱位取象层
func (q *QuxiangResult) SetPillarImage(img *LayerImage) {
	q.PillarImage = img
	q.FullProfile = composeFullProfile(q)
}

// --- 第1层：五行取象 ---

// wuxingNaturalImage 五行自然意象映射
var wuxingNaturalImage = map[string]string{
	"木": "春日林木，生机勃发，有仁德之心，怀生长之力",
	"火": "夏日骄阳，光芒四射，有礼貌之仪，怀热烈之情",
	"土": "厚土载物，沉稳包容，有诚信之德，怀承载之量",
	"金": "秋金肃杀，刚毅果决，有义勇之气，怀变革之力",
	"水": "江河奔涌，智慧绵长，有智谋之才，怀柔韧之性",
}

// wuxingCharacterImage 五行性格意象
var wuxingCharacterImage = map[string]string{
	"木": "性格温和，有恻隐之心，喜条达恶抑郁",
	"火": "性格热情，知礼守节，喜光明恶阴暗",
	"土": "性格稳重，诚信踏实，喜安定恶动荡",
	"金": "性格刚强，讲义气，喜整洁恶杂乱",
	"水": "性格聪慧，善变通，喜自由恶约束",
}

// generateWuxingImage 生成五行取象描述
func generateWuxingImage(analysis *BaziAnalysis) *LayerImage {
	if analysis == nil {
		return nil
	}

	content := "【五行分布】"
	elements := []struct {
		wx    string
		count int
	}{
		{"金", analysis.Wuxing.Jin},
		{"木", analysis.Wuxing.Mu},
		{"水", analysis.Wuxing.Shui},
		{"火", analysis.Wuxing.Huo},
		{"土", analysis.Wuxing.Tu},
	}

	// 按数量从高到低排序
	for i := 0; i < len(elements); i++ {
		for j := i + 1; j < len(elements); j++ {
			if elements[j].count > elements[i].count {
				elements[i], elements[j] = elements[j], elements[i]
			}
		}
	}

	first := true
	for _, e := range elements {
		if !first {
			content += "，"
		}
		content += e.wx + "=" + string(rune('0'+e.count))
		first = false
	}
	content += "。"

	// 日主五行取象
	if img, ok := wuxingNaturalImage[analysis.RishouWuxing]; ok {
		content += "日主" + analysis.Rishou + "属" + analysis.RishouWuxing + "：" + img + "。"
	}
	if img, ok := wuxingCharacterImage[analysis.RishouWuxing]; ok {
		content += img + "。"
	}

	// 身强身弱
	content += "日主" + analysis.DayMasterStrength + "，"
	if len(analysis.Xiyongshen) > 0 {
		content += "喜用神为" + joinStrings(analysis.Xiyongshen, "、") + "。"
	}
	if len(analysis.Yiyongshen) > 0 {
		content += "忌神为" + joinStrings(analysis.Yiyongshen, "、") + "。"
	}

	// 纳音取象
	if analysis.Nayin != "" {
		content += "年命纳音" + analysis.Nayin + "。"
	}

	summary := "日主" + analysis.Rishou + "(" + analysis.RishouWuxing + ")，" + analysis.DayMasterStrength
	if len(analysis.Xiyongshen) > 0 {
		summary += "，喜" + joinStrings(analysis.Xiyongshen, "、")
	}

	return &LayerImage{
		Layer:   LayerWuxing,
		Title:   "五行取象",
		Content: content,
		Summary: summary,
	}
}

// --- 辅助函数 ---

// joinStrings 将字符串切片用分隔符连接
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

// composeFullProfile 合成完整命理描述
func composeFullProfile(result *QuxiangResult) string {
	profile := ""

	if result.WuxingImage != nil {
		profile += "【" + result.WuxingImage.Title + "】\n" + result.WuxingImage.Content + "\n\n"
	}
	if result.TenGodImage != nil {
		profile += "【" + result.TenGodImage.Title + "】\n" + result.TenGodImage.Content + "\n\n"
	}
	if result.ShenshaImage != nil {
		profile += "【" + result.ShenshaImage.Title + "】\n" + result.ShenshaImage.Content + "\n\n"
	}
	if result.PillarImage != nil {
		profile += "【" + result.PillarImage.Title + "】\n" + result.PillarImage.Content + "\n\n"
	}

	return profile
}

// --- 以下为各层取象的构建函数，供外部在获取完整数据后调用 ---

// BuildTenGodImage 构建十神取象描述
// tenGods: 十神计算结果（由 ten_god.go 提供）
func BuildTenGodImage(analysis *BaziAnalysis, tenGods *PillarTenGods) *LayerImage {
	if analysis == nil || tenGods == nil {
		return nil
	}

	content := "【十神分布】"
	content += "年干" + tenGods.YearStem + "为" + tenGods.YearGod + "，"
	content += "月干" + tenGods.MonthStem + "为" + tenGods.MonthGod + "，"
	content += "日干" + tenGods.DayStem + "为" + tenGods.DayGod + "（自身），"
	content += "时干" + tenGods.HourStem + "为" + tenGods.HourGod + "。"

	// 藏干十神
	if len(tenGods.HideHeaven) > 0 {
		content += "地支藏干："
		for i, hh := range tenGods.HideHeaven {
			if i > 0 {
				content += "，"
			}
			content += hh.Stem + "(" + hh.TenGod + "/" + hh.HideType + ")"
		}
		content += "。"
	}

	// 典型十神组合解读
	monthGodMeanings := map[string]string{
		"正官": "月柱正官，为人正直，有管理才能，适合公职或管理岗位",
		"七杀": "月柱七杀，魄力过人，敢作敢当，但需注意控制脾气",
		"正印": "月柱正印，学业有成，多得长辈提携，福泽深厚",
		"偏印": "月柱偏印，思维独特，有特殊才华，适合研究或创作",
		"正财": "月柱正财，财运稳定，善于理财，生活富足",
		"偏财": "月柱偏财，慷慨大方，有投资眼光，常有意外之财",
		"食神": "月柱食神，福气深厚，性格温和，有艺术天赋",
		"伤官": "月柱伤官，聪明伶俐，才华横溢，但需防锋芒过露",
		"比肩": "月柱比肩，朋友众多，社交广泛，但竞争也大",
		"劫财": "月柱劫财，兄弟姊妹缘厚，乐于助人，但需防破财",
	}

	if meaning, ok := monthGodMeanings[tenGods.MonthGod]; ok {
		content += meaning + "。"
	}

	summary := "十神：" + tenGods.YearGod + "/" + tenGods.MonthGod + "/" + tenGods.DayGod + "/" + tenGods.HourGod

	return &LayerImage{
		Layer:   LayerTenGod,
		Title:   "十神取象",
		Content: content,
		Summary: summary,
	}
}

// BuildShenshaImage 构建神煞取象描述
// shensha: 神煞查询结果（由 shensha.go 提供）
func BuildShenshaImage(shensha *PillarShensha) *LayerImage {
	if shensha == nil {
		return nil
	}

	content := ""

	// 日神煞为重点
	if len(shensha.DayGods) > 0 {
		content += "【日神煞】"
		for i, g := range shensha.DayGods {
			if i > 0 {
				content += "、"
			}
			content += g
		}
		content += "。"
	}

	if len(shensha.MonthGods) > 0 {
		content += "月神煞："
		for i, g := range shensha.MonthGods {
			if i > 0 {
				content += "、"
			}
			content += g
		}
		content += "。"
	}

	// 取象解读
	imageMap := map[string]string{
		"天德":   "天德贵人，逢凶化吉，一生贵人相助",
		"月德":   "月德贵人，福泽深厚，遇事有人帮",
		"天乙贵人": "天乙贵人，危难之时自有贵人相助",
		"文昌":   "文昌星入命，学业有成，才思敏捷",
		"桃花":   "桃花星动，异性缘佳，风流洒脱",
		"驿马":   "驿马星动，奔波劳碌，适合外出发展",
		"华盖":   "华盖星照，有佛道缘分，性格孤傲",
		"孤辰":   "孤辰星入命，内心孤独，喜独处",
		"寡宿":   "寡宿星入命，性格独立，晚年安宁",
		"天喜":   "天喜星照，喜事临门，婚姻美满",
		"劫煞":   "劫煞入命，需防小人，注意意外",
		"咸池":   "咸池星动，情感丰富，桃花旺盛",
	}

	for _, g := range shensha.DayGods {
		if img, ok := imageMap[g]; ok {
			content += img + "。"
		}
	}

	summary := "神煞："
	if len(shensha.DayGods) > 0 {
		summary += shensha.DayGods[0]
	}
	if len(shensha.DayGods) > 1 {
		summary += "/" + shensha.DayGods[1]
	}

	return &LayerImage{
		Layer:   LayerShensha,
		Title:   "神煞取象",
		Content: content,
		Summary: summary,
	}
}

// BuildPillarImage 构建柱位取象描述
// pillars: 四柱分析结果（由 pillar.go 提供）
func BuildPillarImage(pillars *FourPillarAnalysis) *LayerImage {
	if pillars == nil {
		return nil
	}

	content := ""

	// 对四柱逐柱取象
	positions := []*PillarAnalysis{pillars.Year, pillars.Month, pillars.Day, pillars.Hour}
	for _, p := range positions {
		if p == nil {
			continue
		}
		content += "【" + p.Position + "柱】" + p.Ganzhi + "："
		content += "天干" + p.HeavenStem + "(" + p.StemWuxing + ")，"
		content += "地支" + p.EarthBranch + "(" + p.BranchWuxing + ")"
		if p.Nayin != "" {
			content += "，纳音" + p.Nayin
		}
		content += "。"

		// 藏干
		if len(p.HideHeaven) > 0 {
			content += "藏干："
			for i, hh := range p.HideHeaven {
				if i > 0 {
					content += "、"
				}
				content += hh.Stem + "(" + hh.Wuxing + "/" + hh.ZhiType + ")"
			}
			content += "。"
		}

		// 位置取象
		if p.Description != "" {
			content += p.Description + "。"
		}
		if p.Image != "" {
			content += p.Image + "。"
		}
		content += "\n"
	}

	summary := ""
	if pillars.Day != nil {
		summary = "日柱" + pillars.Day.Ganzhi + "，" + pillars.Day.Description
	}

	return &LayerImage{
		Layer:   LayerPillar,
		Title:   "柱位取象",
		Content: content,
		Summary: summary,
	}
}

// GenerateFullProfile 一键生成完整命理描述（整合四层取象）
// 需要在外部先计算好十神、神煞、柱位分析，然后传入
func GenerateFullProfile(
	analysis *BaziAnalysis,
	tenGods *PillarTenGods,
	shensha *PillarShensha,
	pillars *FourPillarAnalysis,
) string {
	q := GenerateQuxiang(analysis)
	if tenGods != nil {
		q.SetTenGodImage(BuildTenGodImage(analysis, tenGods))
	}
	if shensha != nil {
		q.SetShenshaImage(BuildShenshaImage(shensha))
	}
	if pillars != nil {
		q.SetPillarImage(BuildPillarImage(pillars))
	}
	return q.FullProfile
}
