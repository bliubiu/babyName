package yijing

import (
	"name/internal/domain/hanzi"
)

// PlumBlossomResult 梅花易数起卦结果
type PlumBlossomResult struct {
	OriginalHexagram  *Hexagram `json:"original_hexagram"`  // 本卦
	ChangedHexagram   *Hexagram `json:"changed_hexagram"`   // 变卦
	InterHexagram     *Hexagram `json:"inter_hexagram"`     // 互卦
	UpperTrigramID    int       `json:"upper_trigram_id"`   // 上卦ID (1-8)
	LowerTrigramID    int       `json:"lower_trigram_id"`   // 下卦ID (1-8)
	UpperTrigramName  string    `json:"upper_trigram_name"` // 上卦名
	LowerTrigramName  string    `json:"lower_trigram_name"` // 下卦名
	MovingYao         int       `json:"moving_yao"`         // 动爻位置 (1-6)
	OriginalYaoLines  [6]int    `json:"original_yao_lines"` // 本卦六爻 (7少阳/8少阴)
	ChangedYaoLines   [6]int    `json:"changed_yao_lines"`  // 变卦六爻
	InterUpperTrigram int       `json:"inter_upper"`        // 互卦上卦ID
	InterLowerTrigram int       `json:"inter_lower"`        // 互卦下卦ID
	Interpretation    string    `json:"interpretation"`     // 综合解读
}

// trigramYaoMaps 八卦对应的三爻值（自下而上，7=少阳/阳爻，8=少阴/阴爻）。
// 先天八卦口诀：乾三连、兑上缺、离中虚、震仰盂、巽下断、坎中满、艮覆碗、坤六断。
// 编码须与 hexagram.go 的 trigramFromYaoLines 一致（阳=1 阴=0，初爻为最高位，
// 8-二进制值=先天卦序）。此前兑/巽、震/艮爻画两两颠倒，导致六爻展示错误、
// 互卦/变卦 ID 与 hexagram.go 正确编码冲突（docs/42 P0-1）。
var trigramYaoMaps = map[int][3]int{
	1: {7, 7, 7}, // 乾☰ 三连 阳阳阳
	2: {7, 7, 8}, // 兑☱ 上缺 阳阳阴
	3: {7, 8, 7}, // 离☲ 中虚 阳阴阳
	4: {7, 8, 8}, // 震☳ 仰盂 阳阴阴
	5: {8, 7, 7}, // 巽☴ 下断 阴阳阳
	6: {8, 7, 8}, // 坎☵ 中满 阴阳阴
	7: {8, 8, 7}, // 艮☶ 覆碗 阴阴阳
	8: {8, 8, 8}, // 坤☷ 六断 阴阴阴
}

// yaoToTrigram 三爻转八卦ID
// 输入 [3]int 型三爻（自下而上，7=少阳，8=少阴），返回八卦ID (1-8)
var yaoToTrigram = map[[3]int]int{
	{7, 7, 7}: 1, // 乾
	{7, 7, 8}: 2, // 兑
	{7, 8, 7}: 3, // 离
	{7, 8, 8}: 4, // 震
	{8, 7, 7}: 5, // 巽
	{8, 7, 8}: 6, // 坎
	{8, 8, 7}: 7, // 艮
	{8, 8, 8}: 8, // 坤
}

// StrokesToTrigram 笔画数转八卦ID (1-8)
// 规则: n%8, 余0→8(坤), 余1→1(乾), 余2→2(兑), ...余7→7(艮)
func StrokesToTrigram(strokes int) int {
	r := strokes % 8
	if r == 0 {
		return 8
	}
	return r
}

// GenerateYaoLines 生成六爻线，从初爻(下)到上爻(上)
func GenerateYaoLines(upperTrig, lowerTrig int) [6]int {
	var lines [6]int
	lowerYao, ok1 := trigramYaoMaps[lowerTrig]
	upperYao, ok2 := trigramYaoMaps[upperTrig]
	if !ok1 || !ok2 {
		return lines
	}
	// 下卦三爻（初、二、三爻）
	lines[0] = lowerYao[0]
	lines[1] = lowerYao[1]
	lines[2] = lowerYao[2]
	// 上卦三爻（四、五、上爻）
	lines[3] = upperYao[0]
	lines[4] = upperYao[1]
	lines[5] = upperYao[2]
	return lines
}

// linesToTrigram 将3个爻还原为八卦ID
func linesToTrigram(yao0, yao1, yao2 int) int {
	key := [3]int{yao0, yao1, yao2}
	if id, ok := yaoToTrigram[key]; ok {
		return id
	}
	return 0
}

// CalcChangedHexagram 计算变卦
func CalcChangedHexagram(upperTrig, lowerTrig int, movingYao int) (int, int) {
	lines := GenerateYaoLines(upperTrig, lowerTrig)
	if movingYao < 1 || movingYao > 6 {
		return upperTrig, lowerTrig
	}
	// 翻转动爻（7<->8）
	if lines[movingYao-1] == 7 {
		lines[movingYao-1] = 8
	} else {
		lines[movingYao-1] = 7
	}
	newLower := linesToTrigram(lines[0], lines[1], lines[2])
	newUpper := linesToTrigram(lines[3], lines[4], lines[5])
	if newLower == 0 {
		newLower = lowerTrig
	}
	if newUpper == 0 {
		newUpper = upperTrig
	}
	return newUpper, newLower
}

// CalcInterHexagram 计算互卦（本卦2-4爻为下卦，3-5爻为上卦）
func CalcInterHexagram(upperTrig, lowerTrig int) (int, int) {
	lines := GenerateYaoLines(upperTrig, lowerTrig)
	interLower := linesToTrigram(lines[1], lines[2], lines[3])
	interUpper := linesToTrigram(lines[2], lines[3], lines[4])
	if interLower == 0 {
		interLower = lowerTrig
	}
	if interUpper == 0 {
		interUpper = upperTrig
	}
	return interUpper, interLower
}

// GetMeihuaDetailByStrokes 梅花易数姓名起卦（笔画版，口径唯一）：
// 上卦 = 姓笔画 %8（余 0 取 8），下卦 = 名总笔画 %8（余 0 取 8），
// 动爻 = (姓 + 名) %6（余 0 取 6）。返回本卦、变卦、互卦与综合解读。
// GetHexagramByMeihuaName（单卦）与 GetHexagramByPlumBlossom（字符版）均委托此函数，
// 消除历史双口径漂移（docs/42 P0-1/P0-5）。
func GetMeihuaDetailByStrokes(surnameStrokes, givenStrokes int) *PlumBlossomResult {
	if surnameStrokes <= 0 || givenStrokes <= 0 {
		return nil
	}

	upperTrigram := TrigramFromStrokes(surnameStrokes)
	lowerTrigram := TrigramFromStrokes(givenStrokes)

	total := surnameStrokes + givenStrokes
	movingYao := total % 6
	if movingYao == 0 {
		movingYao = 6
	}

	// 本卦：按先天上下卦查表（上下卦组合码 ≠ 通行本卦序，不能互查）
	originalHex := FindHexagramByTrigrams(upperTrigram, lowerTrigram)

	// 变卦
	changedUpper, changedLower := CalcChangedHexagram(upperTrigram, lowerTrigram, movingYao)
	changedHex := FindHexagramByTrigrams(changedUpper, changedLower)

	// 互卦
	interUpper, interLower := CalcInterHexagram(upperTrigram, lowerTrigram)
	interHex := FindHexagramByTrigrams(interUpper, interLower)

	// 六爻
	originalLines := GenerateYaoLines(upperTrigram, lowerTrigram)
	changedLines := GenerateYaoLines(changedUpper, changedLower)

	// 综合解读
	interpretation := GenerateHexagramInterpretation(
		originalHex, changedHex, interHex,
		upperTrigram, lowerTrigram,
		movingYao, originalLines,
	)

	return &PlumBlossomResult{
		OriginalHexagram:  originalHex,
		ChangedHexagram:   changedHex,
		InterHexagram:     interHex,
		UpperTrigramID:    upperTrigram,
		LowerTrigramID:    lowerTrigram,
		UpperTrigramName:  TrigramNameMap[upperTrigram],
		LowerTrigramName:  TrigramNameMap[lowerTrigram],
		MovingYao:         movingYao,
		OriginalYaoLines:  originalLines,
		ChangedYaoLines:   changedLines,
		InterUpperTrigram: interUpper,
		InterLowerTrigram: interLower,
		Interpretation:    interpretation,
	}
}

// GetHexagramByPlumBlossom 梅花易数姓名起卦（字符版）：姓氏与名字符逐一取康熙笔画，
// 下卦口径统一为「名总笔画 %8」（旧实现只取名首字，非标准梅花姓名法，docs/42 P0-5）。
func GetHexagramByPlumBlossom(surname, givenName1, givenName2 string) *PlumBlossomResult {
	getStrokes := func(char string) int {
		if char == "" {
			return 0
		}
		if h, ok := hanzi.GetHanzi(char); ok {
			return h.Strokes
		}
		return 0
	}

	surnameStrokes := getStrokes(surname)
	given1Strokes := getStrokes(givenName1)
	given2Strokes := getStrokes(givenName2)

	if surnameStrokes == 0 || given1Strokes == 0 {
		return nil
	}

	return GetMeihuaDetailByStrokes(surnameStrokes, given1Strokes+given2Strokes)
}

// GenerateHexagramInterpretation 生成梅花易数综合解读
func GenerateHexagramInterpretation(
	original, changed, inter *Hexagram,
	upperTrig, lowerTrig int,
	movingYao int,
	originalLines [6]int,
) string {
	text := ""

	// 基本信息
	upperName := TrigramNameMap[upperTrig]
	lowerName := TrigramNameMap[lowerTrig]
	text += "上卦" + upperName + TrigramMap[upperTrig] + "，"
	text += "下卦" + lowerName + TrigramMap[lowerTrig] + "。\n"

	// 本卦
	if original != nil {
		text += "【本卦】" + original.Name + original.Symbol + "\n"
		text += original.GuaCi + "\n"
		text += original.Interpretation + "\n"
	}

	// 动爻
	yaoName := ""
	switch movingYao {
	case 1:
		yaoName = "初爻"
	case 2:
		yaoName = "二爻"
	case 3:
		yaoName = "三爻"
	case 4:
		yaoName = "四爻"
	case 5:
		yaoName = "五爻"
	case 6:
		yaoName = "上爻"
	}
	lineType := "少阳（阳）"
	if originalLines[movingYao-1] == 8 {
		lineType = "少阴（阴）"
	}
	text += "【动爻】第" + yaoName + "动，原为" + lineType + "，动则阴阳互变。\n"

	// 变卦
	if changed != nil {
		text += "【变卦】" + changed.Name + changed.Symbol + "\n"
		text += changed.Interpretation + "\n"
	}

	// 互卦
	if inter != nil {
		text += "【互卦】" + inter.Name + inter.Symbol + "\n"
	}

	// 五行分析
	upperWx := TrigramWuxingMap[upperTrig]
	lowerWx := TrigramWuxingMap[lowerTrig]
	text += "【五行】上卦" + upperName + "(" + upperWx + ")，下卦" + lowerName + "(" + lowerWx + ")"
	if upperWx == lowerWx {
		text += "，上下卦五行相同，比和之象"
	} else if shengMap[upperWx] == lowerWx {
		text += "，上卦生下卦，吉"
	} else if shengMap[lowerWx] == upperWx {
		text += "，下卦生上卦，吉"
	} else if keMap[upperWx] == lowerWx {
		text += "，上卦克下卦，凶中藏吉"
	} else if keMap[lowerWx] == upperWx {
		text += "，下克上，需谨慎行事"
	}
	text += "。\n"

	return text
}

// 五行生克辅助映射
var shengMap = map[string]string{
	"木": "火",
	"火": "土",
	"土": "金",
	"金": "水",
	"水": "木",
}

var keMap = map[string]string{
	"木": "土",
	"火": "金",
	"土": "水",
	"金": "木",
	"水": "火",
}

// buildMeihuaDayanInterpretation 梅花体用框架解读（供两数起卦）：
// 动爻所在之卦为「用」，另一卦为「体」；体用五行生克定吉凶方向（文字化呈现，不打分——
// 参考 meihua-yishu 项目的实验结论：纯五行生克算吉凶接近抛硬币，卦象应作「象」来读）。
func buildMeihuaDayanInterpretation(
	original, changed, inter *Hexagram,
	upperTrig, lowerTrig, movingYao int,
	yaoLines []int,
) string {
	if original == nil {
		return ""
	}
	text := "【本卦】" + original.Name + original.Symbol + "\n" +
		original.GuaCi + "\n" + original.Interpretation + "\n"

	// 体用判定：动爻在 1-3 爻则下卦为用，4-6 爻则上卦为用
	tiTrig, yongTrig := lowerTrig, upperTrig
	if movingYao >= 4 {
		tiTrig, yongTrig = upperTrig, lowerTrig
	}
	tiWx, yongWx := TrigramWuxingMap[tiTrig], TrigramWuxingMap[yongTrig]
	posName := map[int]string{1: "初爻", 2: "二爻", 3: "三爻", 4: "四爻", 5: "五爻", 6: "上爻"}
	text += "【体用】动爻在" + posName[movingYao] + "，" + TrigramNameMap[yongTrig] +
		"为用卦、" + TrigramNameMap[tiTrig] + "为体卦。"

	switch {
	case yongWx == tiWx:
		text += "体用比和（" + tiWx + "），诸事平顺。\n"
	case shengMap[yongWx] == tiWx:
		text += "用生体（" + yongWx + "生" + tiWx + "），得外助，吉。\n"
	case keMap[tiWx] == yongWx:
		text += "体克用（" + tiWx + "克" + yongWx + "），事可成。\n"
	case keMap[yongWx] == tiWx:
		text += "用克体（" + yongWx + "克" + tiWx + "），受制于人，宜守不宜进。\n"
	default: // 体生用
		text += "体生用（" + tiWx + "生" + yongWx + "），费力耗神，收获有限。\n"
	}

	if inter != nil {
		text += "【互卦】" + inter.Name + inter.Symbol + "（事情发展过程）\n"
	}
	if changed != nil {
		text += "【变卦】" + changed.Name + changed.Symbol + "（最终趋势）\n" + changed.Interpretation + "\n"
	}
	return text
}
