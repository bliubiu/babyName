package bazi

import (
	"strings"

	"name/internal/domain/bazi/tyme"
)

// HideHeavenInfo 地支藏干信息
type HideHeavenInfo struct {
	Stem    string `json:"stem"`     // 天干
	Wuxing  string `json:"wuxing"`   // 五行
	ZhiType string `json:"zhi_type"` // 本气/中气/余气
}

// PillarAnalysis 单柱分析
type PillarAnalysis struct {
	Position     string           `json:"position"`      // 年/月/日/时
	Ganzhi       string           `json:"ganzhi"`        // 干支全称
	HeavenStem   string           `json:"heaven_stem"`   // 天干
	EarthBranch  string           `json:"earth_branch"`  // 地支
	StemWuxing   string           `json:"stem_wuxing"`   // 天干五行
	BranchWuxing string           `json:"branch_wuxing"` // 地支五行
	Nayin        string           `json:"nayin"`         // 纳音
	NayinWuxing  string           `json:"nayin_wuxing"`  // 纳音五行
	HideHeaven   []HideHeavenInfo `json:"hide_heaven"`   // 藏干
	Description  string           `json:"description"`   // 位置描述
	Image        string           `json:"image"`         // 取象描述
}

// FourPillarAnalysis 四柱分析结果
type FourPillarAnalysis struct {
	Year  *PillarAnalysis `json:"year"`
	Month *PillarAnalysis `json:"month"`
	Day   *PillarAnalysis `json:"day"`
	Hour  *PillarAnalysis `json:"hour"`
}

// stripWhitespace 去除字符串中的空白
func stripWhitespace(s string) string {
	return strings.Map(func(r rune) rune {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			return -1
		}
		return r
	}, s)
}

// AnalyzePillar 分析单柱
func AnalyzePillar(position, ganzhi string) *PillarAnalysis {
	if ganzhi == "" {
		return nil
	}
	runes := []rune(ganzhi)
	if len(runes) < 2 {
		return nil
	}

	stem := string(runes[0])
	branch := string(runes[1])

	stemWx := WuxingMap[stem]
	branchWx := DizhiWuxingMap[branch]

	var nayin, nayinWx string
	if len([]rune(ganzhi)) >= 2 {
		if n, ok := NayinMap[ganzhi]; ok {
			nayin = n
			if nw, ok := NayinWuxingMap[n]; ok {
				nayinWx = nw
			}
		}
	}

	// 藏干
	var hideHeaven []HideHeavenInfo
	eb, err := tyme.EarthBranch{}.FromName(branch)
	if err == nil {
		hideStems := eb.GetHideHeavenStems()
		for _, hs := range hideStems {
			stemName := hs.GetHeavenStem().GetName()
			stemWx := WuxingMap[stemName]
			zhiType := hs.GetType().GetName()
			hideHeaven = append(hideHeaven, HideHeavenInfo{
				Stem:    stemName,
				Wuxing:  stemWx,
				ZhiType: zhiType,
			})
		}
	}

	description := DescribePillarPosition(position)
	image := PillarImage(position, stem, branch, stemWx, branchWx)

	return &PillarAnalysis{
		Position:     position,
		Ganzhi:       ganzhi,
		HeavenStem:   stem,
		EarthBranch:  branch,
		StemWuxing:   stemWx,
		BranchWuxing: branchWx,
		Nayin:        nayin,
		NayinWuxing:  nayinWx,
		HideHeaven:   hideHeaven,
		Description:  description,
		Image:        image,
	}
}

// AnalyzeFourPillars 分析四柱
func AnalyzeFourPillars(bazi *Bazi) *FourPillarAnalysis {
	if bazi == nil {
		return nil
	}
	return &FourPillarAnalysis{
		Year:  AnalyzePillar("年", bazi.YearGanzhi),
		Month: AnalyzePillar("月", bazi.MonthGanzhi),
		Day:   AnalyzePillar("日", bazi.DayGanzhi),
		Hour:  AnalyzePillar("时", bazi.HourGanzhi),
	}
}

// DescribePillarPosition 位置描述
func DescribePillarPosition(position string) string {
	switch position {
	case "年":
		return "年柱代表祖上、父母、童年和家族根基，反映早年家境与遗传禀赋"
	case "月":
		return "月柱代表父母环境、兄弟手足和青年运势，体现事业根基与社会关系"
	case "日":
		return "日柱代表自身命运、配偶关系和中年运势，是八字核心所在"
	case "时":
		return "时柱代表子女、下属和晚年运势，反映人生归宿与最终成就"
	default:
		return ""
	}
}

// PillarImage 位置取象描述
func PillarImage(position, stem, branch, stemWx, branchWx string) string {
	// 天干自然意象
	stemImage := map[string]string{
		"甲": "参天大树，栋梁之材",
		"乙": "花草藤蔓，柔韧多姿",
		"丙": "太阳之火，温暖照耀",
		"丁": "灯烛之火，夜中光明",
		"戊": "厚土高岗，沉稳厚重",
		"己": "田园沃土，滋养万物",
		"庚": "刀剑之金，刚毅变革",
		"辛": "珠玉之金，精致细腻",
		"壬": "江河之水，浩瀚奔流",
		"癸": "雨露之水，润物无声",
	}

	// 地支意象
	branchImage := map[string]string{
		"子": "子水涵泳，智珠在握",
		"丑": "丑土蓄势，厚积薄发",
		"寅": "寅木勃发，生机盎然",
		"卯": "卯木繁茂，欣欣向荣",
		"辰": "辰土化育，万物生长",
		"巳": "巳火炎上，热情洋溢",
		"午": "午火鼎盛，光芒四射",
		"未": "未土收藏，硕果累累",
		"申": "申金肃杀，刚健有力",
		"酉": "酉金收敛，精致完美",
		"戌": "戌土归藏，蓄势待发",
		"亥": "亥水渊深，智慧莫测",
	}

	parts := []string{}
	if img, ok := stemImage[stem]; ok {
		parts = append(parts, img)
	}
	if img, ok := branchImage[branch]; ok {
		parts = append(parts, img)
	}

	if len(parts) > 0 {
		return strings.Join(parts, "，")
	}
	return ""
}
