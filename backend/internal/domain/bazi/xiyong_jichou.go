package bazi

// xiyong_jichou.go — 忌神/仇神推导（docs/43 P1-3 阶段1）
//
// 背景：weighted 算法（calculateXiyongshenWeighted）产出喜用神列表，
// 但无忌神/仇神概念；此前服务层装配 FateData.WuXingXiji 时由
// fate.BalanceXiYongJi（另一套强弱口径）补齐 Ji/Chou，造成同一响应内
// 「喜用神用 B 侧、强弱栏与忌仇用 A 侧」的口径混搭，可能自相矛盾。
//
// 收敛后：喜用神与强弱均以本包 weighted 算法为唯一权威（命理层一次性
// 决策，字象层纯消费），忌/仇按同一强弱结论由传统扶抑法推导：
//   - 身旺：忌比劫（再帮身则亢）、仇印（生比劫助忌）
//   - 身弱/身衰：忌官杀（克我者）、仇财（耗身且生官）
//   - 身中：中性，无忌仇

// DeriveJiChou 按日主强弱推导忌神与仇神（扶抑法）。
// dayMasterStrength 取值与 calculateDayMasterStrengthWeighted 一致：
// 「身旺」「身弱」「身衰」「身中」；其余值按身中处理。
func DeriveJiChou(rishouWuxing, dayMasterStrength string) (ji, chou string) {
	switch dayMasterStrength {
	case "身旺":
		// 忌比劫：身旺再见同类则过亢；仇印：印生比劫，助忌
		return rishouWuxing, shengWoOf(rishouWuxing)
	case "身弱", "身衰":
		// 忌官杀：克我者本已伤弱之身；仇财：财耗弱身又生官杀
		return keWoOf(rishouWuxing), woKeOf(rishouWuxing)
	default:
		return "", ""
	}
}

// woKeOf 我克者（财才）：WuxingShengkeMap[x][1] 为 x 所克
func woKeOf(elem string) string {
	if v, ok := WuxingShengkeMap[elem]; ok && len(v) > 1 {
		return v[1]
	}
	return ""
}
