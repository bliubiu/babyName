package fate

// BalanceXiYongJi 平衡用神法 - 根据日主强弱取喜用忌仇
// 日主强：克我者（官杀）为用神，我生者（食伤）为喜神，同我者（比劫）为忌神
// 日主弱：生我者（印）为用神，同我者（比劫）为喜神，克我者（官杀）为忌神
func BalanceXiYongJi(baziInfo BaziInfoForGeJu) XiYongJiChou {
	riZhuGan := string([]rune(baziInfo.SiZhu[2])[:1])
	riZhuWuxing := wuxingOfTianGan(riZhuGan)

	// 计算五行力量
	strength := calculateWuXingFen(baziInfo)
	total := 0.0
	for _, v := range strength {
		total += v
	}

	// 判断日主强弱
	tonglei := tongleiWuxing(riZhuWuxing)
	myScore := 0.0
	for wx, score := range strength {
		if tonglei[wx] {
			myScore += score
		}
	}

	qiangRuo := "弱"
	if myScore > total/2 {
		qiangRuo = "强"
	}

	var xi, yong, ji, chou string
	if qiangRuo == "强" {
		// 日主强：克泄为用
		yong = findKeWoWuxing(riZhuWuxing)   // 官杀
		xi = findWoShengWuxing(riZhuWuxing)  // 食伤
		ji = riZhuWuxing                       // 比劫
		chou = findShengWoWuxing(riZhuWuxing)  // 印
	} else {
		// 日主弱：生扶为用
		yong = riZhuWuxing                       // 比劫
		xi = findShengWoWuxing(riZhuWuxing)  // 印
		ji = findKeWoWuxing(riZhuWuxing)     // 官杀
		chou = findWoKeWuxing(riZhuWuxing)   // 财
	}

	// 补最缺五行
	weakest := findWeakestWuxing(strength)
	if weakest != "" && qiangRuo == "弱" && weakest != riZhuWuxing {
		yong = weakest
	}

	return XiYongJiChou{
		Xi:   xi,
		Yong: yong,
		Ji:   ji,
		Chou: chou,
	}
}

// calculateWuXingFen 计算四柱五行得分
func calculateWuXingFen(baziInfo BaziInfoForGeJu) map[string]float64 {
	wuxingFen := map[string]float64{
		"木": 0, "火": 0, "土": 0, "金": 0, "水": 0,
	}

	for _, gz := range baziInfo.SiZhu {
		runes := []rune(gz)
		if len(runes) < 2 {
			continue
		}
		gan := string(runes[:1])
		zhi := string(runes[1:])

		// 天干五行得分
		ganWx := wuxingOfTianGan(gan)
		wuxingFen[ganWx] += 1.0

		// 地支藏干得分
		if entries, ok := diZhiHiddenStemsWeighted[zhi]; ok {
			for _, entry := range entries {
				wx := wuxingOfTianGan(entry.Stem)
				wuxingFen[wx] += entry.Weight
			}
		}
	}

	return wuxingFen
}

// findWeakestWuxing 找出五行中最弱的
func findWeakestWuxing(wuxingFen map[string]float64) string {
	min := -1.0
	result := ""
	for _, wx := range []string{"木", "火", "土", "金", "水"} {
		score := wuxingFen[wx]
		if min < 0 || score < min {
			min = score
			result = wx
		}
	}
	return result
}
