package fate

import "fmt"

// determineGeJu 根据八字信息判断格局
// siZhu: 四柱字符串数组 [年柱, 月柱, 日柱, 时柱]（如 "甲子"）
func determineGeJu(baziInfo BaziInfoForGeJu, strength WuxingStrength) *GeJuInfo {
	siZhu := baziInfo.SiZhu
	// 日柱天干（日主）
	riZhuGan := string([]rune(siZhu[2])[:1])
	// 月支（月柱的地支部分）
	yueZhi := string([]rune(siZhu[1])[1:])

	// 取月支的主气（藏干中权重最高的）
	var mainQi string
	if stems, ok := diZhiHiddenStemsWeighted[yueZhi]; ok && len(stems) > 0 {
		mainQi = stems[0].Stem
	}
	if mainQi == "" {
		return &GeJuInfo{
			Type:     GeJuUnknown,
			Name:     "格局待定",
			YongShen: "需详细排盘确认",
			XiShen:   "需详细排盘确认",
			JiShen:   "需详细排盘确认",
			ChouShen: "需详细排盘确认",
			Analysis: "出生时间信息不足，无法准确判断格局。",
		}
	}

	// 月支主气对日主的关系 = 十神 → 格局
	shiShen := getShiShen(riZhuGan, mainQi)
	geJuType := shiShenToGeJu(shiShen)

	if geJuType == GeJuUnknown {
		// 无法判定格局，按日主强弱平衡处理
		var yongShen, xiShen, jiShen string
		if strength.Total > 50 {
			yongShen = "克泄"
			xiShen = "火土"
			jiShen = "水木"
		} else {
			yongShen = "生扶"
			xiShen = "水木"
			jiShen = "火土"
		}
		return &GeJuInfo{
			Type:     GeJuUnknown,
			Name:     "普通格局",
			YongShen: yongShen,
			XiShen:   xiShen,
			JiShen:   jiShen,
			ChouShen: "需综合判断",
			Analysis: fmt.Sprintf("日主%s，五行总分%.1f，以平衡为原则。", riZhuGan, strength.Total),
		}
	}

	// 判断是否为羊刃
	isYangRen := yueZhi == getYangRenZhi(riZhuGan)

	// 根据格局取喜用神
	xiYongJi := geJuXiYongJi(geJuType, riZhuGan, strength, isYangRen)

	// 生成分析文字
	analysis := generateGeJuAnalysis(geJuType, riZhuGan, xiYongJi, isYangRen)

	return &GeJuInfo{
		Type:     geJuType,
		Name:     geJuTypeName(geJuType),
		YongShen: xiYongJi.Yong,
		XiShen:   xiYongJi.Xi,
		JiShen:   xiYongJi.Ji,
		ChouShen: xiYongJi.Chou,
		Analysis: analysis,
	}
}

// getShiShen 获取天干之间的十神关系
func getShiShen(dayGan, target string) string {
	if m, ok := tianGanShiShenMap[dayGan]; ok {
		if ss, ok := m[target]; ok {
			return ss
		}
	}

	// 若十神表中未直接命中，通过五行关系推导
	targetWx := wuxingOfTianGan(target)
	dayWx := wuxingOfTianGan(dayGan)
	if dayWx == "" || targetWx == "" {
		return ""
	}

	if rel, ok := wuxingRelations[dayWx]; ok {
		if result, ok := rel[targetWx]; ok {
			return result
		}
	}
	return ""
}

// shiShenToGeJu 十神转格局类型
func shiShenToGeJu(shiShen string) GeJuType {
	switch shiShen {
	case "正官":
		return GeJuZhengGuan
	case "七杀", "偏官":
		return GeJuQiSha
	case "正财":
		return GeJuZhengCai
	case "偏财":
		return GeJuPianCai
	case "正印":
		return GeJuZhengYin
	case "偏印":
		return GeJuPianYin
	case "食神":
		return GeJuShiShen
	case "伤官":
		return GeJuShangGuan
	default:
		return GeJuUnknown
	}
}

// getYangRenZhi 获取天干的羊刃地支
func getYangRenZhi(gan string) string {
	m := map[string]string{
		"甲": "卯", "乙": "寅", "丙": "午", "丁": "巳",
		"戊": "午", "己": "巳", "庚": "酉", "辛": "申",
		"壬": "子", "癸": "亥",
	}
	return m[gan]
}

// geJuXiYongJi 根据格局类型取喜用忌仇
func geJuXiYongJi(geJu GeJuType, riZhuGan string, strength WuxingStrength, isYangRen bool) XiYongJiChou {
	switch geJu {
	case GeJuZhengGuan:
		return geJuZhengGuanYong(riZhuGan, strength, isYangRen)
	case GeJuQiSha:
		return geJuQiShaYong(riZhuGan, strength, isYangRen)
	case GeJuZhengCai:
		return geJuZhengCaiYong(riZhuGan, strength, isYangRen)
	case GeJuPianCai:
		return geJuPianCaiYong(riZhuGan, strength, isYangRen)
	case GeJuZhengYin:
		return geJuZhengYinYong(riZhuGan, strength, isYangRen)
	case GeJuPianYin:
		return geJuPianYinYong(riZhuGan, strength, isYangRen)
	case GeJuShiShen:
		return geJuShiShenYong(riZhuGan, strength, isYangRen)
	case GeJuShangGuan:
		return geJuShangGuanYong(riZhuGan, strength, isYangRen)
	default:
		return XiYongJiChou{}
	}
}

// === 各格局喜用神算法 ===

func geJuZhengGuanYong(gan string, _ WuxingStrength, _ bool) XiYongJiChou {
	riWx := wuxingOfTianGan(gan)
	return XiYongJiChou{
		Yong: riWx,
		Xi:   findShengWoWuxing(riWx),
		Ji:   findKeWoWuxing(riWx),
		Chou: findWoKeWuxing(riWx),
	}
}

func geJuQiShaYong(gan string, _ WuxingStrength, _ bool) XiYongJiChou {
	riWx := wuxingOfTianGan(gan)
	return XiYongJiChou{
		Yong: findKeWoWuxing(riWx),
		Xi:   riWx,
		Ji:   findShengWoWuxing(riWx),
		Chou: findWoShengWuxing(riWx),
	}
}

func geJuZhengCaiYong(gan string, _ WuxingStrength, _ bool) XiYongJiChou {
	riWx := wuxingOfTianGan(gan)
	return XiYongJiChou{
		Yong: findWoShengWuxing(riWx),
		Xi:   riWx,
		Ji:   findKeWoWuxing(riWx),
		Chou: findShengWoWuxing(riWx),
	}
}

func geJuPianCaiYong(gan string, strength WuxingStrength, isYangRen bool) XiYongJiChou {
	return geJuZhengCaiYong(gan, strength, isYangRen)
}

func geJuZhengYinYong(gan string, _ WuxingStrength, _ bool) XiYongJiChou {
	riWx := wuxingOfTianGan(gan)
	return XiYongJiChou{
		Yong: findShengWoWuxing(riWx),
		Xi:   riWx,
		Ji:   findWoKeWuxing(riWx),
		Chou: findWoShengWuxing(riWx),
	}
}

func geJuPianYinYong(gan string, strength WuxingStrength, isYangRen bool) XiYongJiChou {
	return geJuZhengYinYong(gan, strength, isYangRen)
}

func geJuShiShenYong(gan string, _ WuxingStrength, _ bool) XiYongJiChou {
	riWx := wuxingOfTianGan(gan)
	return XiYongJiChou{
		Yong: findWoShengWuxing(riWx),
		Xi:   findShengWoWuxing(riWx),
		Ji:   findKeWoWuxing(riWx),
		Chou: riWx,
	}
}

func geJuShangGuanYong(gan string, _ WuxingStrength, _ bool) XiYongJiChou {
	riWx := wuxingOfTianGan(gan)
	return XiYongJiChou{
		Yong: findWoKeWuxing(riWx),
		Xi:   findShengWoWuxing(riWx),
		Ji:   riWx,
		Chou: findWoShengWuxing(riWx),
	}
}

// === 辅助函数 ===

func geJuTypeName(t GeJuType) string {
	names := map[GeJuType]string{
		GeJuZhengGuan: "正官格",
		GeJuQiSha:     "七杀格",
		GeJuZhengCai:  "正财格",
		GeJuPianCai:   "偏财格",
		GeJuZhengYin:  "正印格",
		GeJuPianYin:   "偏印格",
		GeJuShiShen:   "食神格",
		GeJuShangGuan: "伤官格",
		GeJuUnknown:   "未知格局",
	}
	return names[t]
}

func generateGeJuAnalysis(geJu GeJuType, riZhuGan string, xiYongJi XiYongJiChou, isYangRen bool) string {
	name := geJuTypeName(geJu)
	riWx := wuxingOfTianGan(riZhuGan)

	analysis := fmt.Sprintf("日主%s，%s。", riWx, name)
	analysis += fmt.Sprintf("用神：%s，喜神：%s，忌神：%s，仇神：%s。", xiYongJi.Yong, xiYongJi.Xi, xiYongJi.Ji, xiYongJi.Chou)

	if isYangRen {
		analysis += "羊刃当权，需注意制化。"
	}

	return analysis
}

// === 五行辅助函数 ===

func wuxingOfTianGan(gan string) string {
	if wx, ok := tianGanWuxingMap[gan]; ok {
		return wx
	}
	return ""
}

func tongleiWuxing(wx string) map[string]bool {
	result := make(map[string]bool)
	switch wx {
	case "木":
		result["木"] = true
		result["水"] = true
	case "火":
		result["火"] = true
		result["木"] = true
	case "土":
		result["土"] = true
		result["火"] = true
	case "金":
		result["金"] = true
		result["土"] = true
	case "水":
		result["水"] = true
		result["金"] = true
	}
	return result
}

func findKeWoWuxing(myWuxing string) string {
	keMap := map[string]string{
		"木": "金",
		"火": "水",
		"土": "木",
		"金": "火",
		"水": "土",
	}
	return keMap[myWuxing]
}

func findShengWoWuxing(wx string) string {
	m := map[string]string{"木": "水", "火": "木", "土": "火", "金": "土", "水": "金"}
	return m[wx]
}

func findWoShengWuxing(wx string) string {
	m := map[string]string{"木": "火", "火": "土", "土": "金", "金": "水", "水": "木"}
	return m[wx]
}

func findWoKeWuxing(wx string) string {
	m := map[string]string{"木": "土", "火": "金", "土": "水", "金": "木", "水": "火"}
	return m[wx]
}
