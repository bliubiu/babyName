package bazi

// HiddenStem 藏干条目
type HiddenStem struct {
	Stem   string
	Weight float64
}

// DizhiHiddenStems 地支藏干及权重（本气+中气+余气，合计约 1.0）
var DizhiHiddenStems = map[string][]HiddenStem{
	"子": {{Stem: "癸", Weight: 1.0}},
	"丑": {{Stem: "己", Weight: 0.6}, {Stem: "癸", Weight: 0.2}, {Stem: "辛", Weight: 0.2}},
	"寅": {{Stem: "甲", Weight: 0.6}, {Stem: "丙", Weight: 0.3}, {Stem: "戊", Weight: 0.1}},
	"卯": {{Stem: "乙", Weight: 1.0}},
	"辰": {{Stem: "戊", Weight: 0.6}, {Stem: "乙", Weight: 0.2}, {Stem: "癸", Weight: 0.2}},
	"巳": {{Stem: "丙", Weight: 0.6}, {Stem: "庚", Weight: 0.3}, {Stem: "戊", Weight: 0.1}},
	"午": {{Stem: "丁", Weight: 0.7}, {Stem: "己", Weight: 0.3}},
	"未": {{Stem: "己", Weight: 0.6}, {Stem: "丁", Weight: 0.2}, {Stem: "乙", Weight: 0.2}},
	"申": {{Stem: "庚", Weight: 0.6}, {Stem: "壬", Weight: 0.3}, {Stem: "戊", Weight: 0.1}},
	"酉": {{Stem: "辛", Weight: 1.0}},
	"戌": {{Stem: "戊", Weight: 0.6}, {Stem: "辛", Weight: 0.2}, {Stem: "丁", Weight: 0.2}},
	"亥": {{Stem: "壬", Weight: 0.7}, {Stem: "甲", Weight: 0.3}},
}

// WeightedWuxing 加权五行力量（浮点）
type WeightedWuxing struct {
	Jin  float64 `json:"jin"`
	Mu   float64 `json:"mu"`
	Shui float64 `json:"shui"`
	Huo  float64 `json:"huo"`
	Tu   float64 `json:"tu"`
}

const (
	// MonthPillarWeight 月令权重：月支对全局旺衰影响最大
	MonthPillarWeight = 1.5
	// StemWeight 天干权重
	StemWeight = 1.0
)

// CalculateWeightedWuxing 按藏干加权统计四柱五行力量
// 天干计 1.0；地支按藏干权重；整柱月令再乘 1.5
func CalculateWeightedWuxing(b *Bazi) *WeightedWuxing {
	w := &WeightedWuxing{}
	if b == nil {
		return w
	}

	addPillar := func(ganzhi string, pillarMul float64) {
		runes := []rune(ganzhi)
		if len(runes) < 2 {
			return
		}
		gan := string(runes[0])
		zhi := string(runes[1])

		if wx, ok := WuxingMap[gan]; ok {
			w.add(wx, StemWeight*pillarMul)
		}
		if stems, ok := DizhiHiddenStems[zhi]; ok {
			for _, hs := range stems {
				if wx, ok := WuxingMap[hs.Stem]; ok {
					w.add(wx, hs.Weight*pillarMul)
				}
			}
		} else if wx, ok := DizhiWuxingMap[zhi]; ok {
			w.add(wx, 1.0*pillarMul)
		}
	}

	addPillar(b.YearGanzhi, 1.0)
	addPillar(b.MonthGanzhi, MonthPillarWeight)
	addPillar(b.DayGanzhi, 1.0)
	addPillar(b.HourGanzhi, 1.0)
	return w
}

func (w *WeightedWuxing) add(elem string, v float64) {
	switch elem {
	case "金":
		w.Jin += v
	case "木":
		w.Mu += v
	case "水":
		w.Shui += v
	case "火":
		w.Huo += v
	case "土":
		w.Tu += v
	}
}

// Get 按五行名取分
func (w *WeightedWuxing) Get(elem string) float64 {
	switch elem {
	case "金":
		return w.Jin
	case "木":
		return w.Mu
	case "水":
		return w.Shui
	case "火":
		return w.Huo
	case "土":
		return w.Tu
	}
	return 0
}

// Total 总力量
func (w *WeightedWuxing) Total() float64 {
	return w.Jin + w.Mu + w.Shui + w.Huo + w.Tu
}

// ToCounts 四舍五入为整数计数（兼容旧 WuxingResult 展示）
func (w *WeightedWuxing) ToCounts() *WuxingResult {
	return &WuxingResult{
		Jin:  roundHalf(w.Jin),
		Mu:   roundHalf(w.Mu),
		Shui: roundHalf(w.Shui),
		Huo:  roundHalf(w.Huo),
		Tu:   roundHalf(w.Tu),
	}
}

func roundHalf(v float64) int {
	if v < 0 {
		return 0
	}
	return int(v + 0.5)
}

// calculateDayMasterStrengthWeighted 按加权力量判日主强弱
// 含月令同类加成：日主同五行或生日主的印在月支则略偏旺
func calculateDayMasterStrengthWeighted(w *WeightedWuxing, rishouWuxing, monthBranch string) string {
	total := w.Total()
	if total <= 0 {
		return "偏弱"
	}

	dayMasterValue := w.Get(rishouWuxing)

	// 印绶（生我）也计入“得地/得生”
	shengWo := shengWoOf(rishouWuxing)
	dayMasterValue += w.Get(shengWo) * 0.6

	// 月支本气若与日主同五行，额外加权
	if monthBranch != "" {
		if stems, ok := DizhiHiddenStems[monthBranch]; ok && len(stems) > 0 {
			if wx, ok := WuxingMap[stems[0].Stem]; ok && wx == rishouWuxing {
				dayMasterValue += 0.8
			}
		}
	}

	ratio := dayMasterValue / total * 100
	switch {
	case ratio >= 40:
		return "身旺"
	case ratio >= 28:
		return "身中"
	case ratio >= 18:
		return "身弱"
	default:
		return "身衰"
	}
}

func shengWoOf(elem string) string {
	// 谁生我：遍历 WuxingShengkeMap，v[0] 为所生
	for k, v := range WuxingShengkeMap {
		if len(v) > 0 && v[0] == elem {
			return k
		}
	}
	return ""
}

func keWoOf(elem string) string {
	for k, v := range WuxingShengkeMap {
		if len(v) > 1 && v[1] == elem {
			return k
		}
	}
	return ""
}

// calculateXiyongshenWeighted 加权喜用神：
// 身旺→克泄耗；身弱→生扶；并优先补最弱五行
func calculateXiyongshenWeighted(w *WeightedWuxing, rishouWuxing, dayMasterStrength string, tiaohou []string) []string {
	if w.Total() <= 0 {
		return []string{rishouWuxing}
	}

	result := make([]string, 0, 3)
	switch dayMasterStrength {
	case "身旺":
		if ke := keWoOf(rishouWuxing); ke != "" {
			result = append(result, ke)
		}
		if list, ok := WuxingShengkeMap[rishouWuxing]; ok && len(list) >= 2 {
			result = append(result, list[0], list[1])
		}
	case "身弱", "身衰":
		result = append(result, rishouWuxing)
		if sheng := shengWoOf(rishouWuxing); sheng != "" {
			result = append(result, sheng)
		}
		// 弱则优先补最缺且非忌
		if weakest := weakestWuxing(w); weakest != "" && weakest != rishouWuxing {
			result = appendUnique(result, weakest)
		}
	default: // 身中
		result = append(result, rishouWuxing)
		if weakest := weakestWuxing(w); weakest != "" {
			result = appendUnique(result, weakest)
		}
	}

	// 调候（冬火夏水等）作为补充喜用，不覆盖主用神
	for _, t := range tiaohou {
		if len(result) < 3 {
			result = appendUnique(result, t)
		}
	}
	return result
}

func weakestWuxing(w *WeightedWuxing) string {
	pairs := []struct {
		name string
		v    float64
	}{
		{"金", w.Jin}, {"木", w.Mu}, {"水", w.Shui}, {"火", w.Huo}, {"土", w.Tu},
	}
	min := -1.0
	name := ""
	for _, p := range pairs {
		if min < 0 || p.v < min {
			min = p.v
			name = p.name
		}
	}
	return name
}

func appendUnique(list []string, v string) []string {
	if v == "" {
		return list
	}
	for _, x := range list {
		if x == v {
			return list
		}
	}
	return append(list, v)
}

// calculateYiyongshenFromWeighted 忌神：取与日主相克且力量偏旺的五行
func calculateYiyongshenFromWeighted(w *WeightedWuxing, rishouWuxing string) []string {
	if w.Total() <= 0 {
		return nil
	}
	ke := keWoOf(rishouWuxing)
	if ke != "" && w.Get(ke) >= w.Total()*0.25 {
		return []string{ke}
	}
	weakest := weakestWuxing(w)
	if weakest != "" {
		if list, ok := WuxingShengkeMap[weakest]; ok && len(list) >= 2 {
			return []string{list[1]}
		}
	}
	return nil
}
