package yijing

import (
	"name/internal/domain/hanzi"
)

// PlumBlossomResult 梅花易数起卦结果
type PlumBlossomResult struct {
	OriginalHexagram   *Hexagram `json:"original_hexagram"`    // 本卦
	ChangedHexagram    *Hexagram `json:"changed_hexagram"`     // 变卦
	InterHexagram      *Hexagram `json:"inter_hexagram"`       // 互卦
	UpperTrigramID     int       `json:"upper_trigram_id"`     // 上卦ID (1-8)
	LowerTrigramID     int       `json:"lower_trigram_id"`     // 下卦ID (1-8)
	UpperTrigramName   string    `json:"upper_trigram_name"`   // 上卦名
	LowerTrigramName   string    `json:"lower_trigram_name"`   // 下卦名
	MovingYao          int       `json:"moving_yao"`           // 动爻位置 (1-6)
	OriginalYaoLines   [6]int    `json:"original_yao_lines"`   // 本卦六爻 (7少阳/8少阴)
	ChangedYaoLines    [6]int    `json:"changed_yao_lines"`    // 变卦六爻
	InterUpperTrigram  int       `json:"inter_upper"`          // 互卦上卦ID
	InterLowerTrigram  int       `json:"inter_lower"`          // 互卦下卦ID
	Interpretation     string    `json:"interpretation"`       // 综合解读
}

// trigramYaoMaps 八卦对应的三爻值（从下到上，7=少阳，8=少阴）
var trigramYaoMaps = map[int][3]int{
	1: {7, 7, 7}, // 乾☰ 阳阳阳
	2: {8, 7, 7}, // 兑☱ 阴阳阳
	3: {7, 8, 7}, // 离☲ 阳阴阳
	4: {8, 8, 7}, // 震☳ 阴阴阳
	5: {7, 7, 8}, // 巽☴ 阳阳阴
	6: {8, 7, 8}, // 坎☵ 阴阳阴
	7: {7, 8, 8}, // 艮☶ 阳阴阴
	8: {8, 8, 8}, // 坤☷ 阴阴阴
}

// yaoToTrigram 三爻转八卦ID
// 输入 [3]int 型三爻（7=少阳，8=少阴），返回八卦ID (1-8)
var yaoToTrigram = map[[3]int]int{
	{7, 7, 7}: 1, // 乾
	{8, 7, 7}: 2, // 兑
	{7, 8, 7}: 3, // 离
	{8, 8, 7}: 4, // 震
	{7, 7, 8}: 5, // 巽
	{8, 7, 8}: 6, // 坎
	{7, 8, 8}: 7, // 艮
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

// GetHexagramByPlumBlossom 梅花易数姓名起卦
// surname: 姓, givenName1: 名第一字, givenName2: 名第二字（可选，传空字符串）
// 起卦法：上卦=姓笔画%8, 下卦=名第一字笔画%8, 动爻=总笔画%6
func GetHexagramByPlumBlossom(surname, givenName1, givenName2 string) *PlumBlossomResult {
	// 获取笔画数
	getStrokes := func(char string) int {
		if char == "" {
			return 0
		}
		if h, ok := hanzi.HanziData[char]; ok {
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

	// 上卦
	upperTrigram := StrokesToTrigram(surnameStrokes)
	// 下卦
	lowerTrigram := StrokesToTrigram(given1Strokes)

	// 总笔画
	total := surnameStrokes + given1Strokes + given2Strokes

	// 动爻
	movingYao := total % 6
	if movingYao == 0 {
		movingYao = 6
	}

	// 本卦
	hexNumber := (upperTrigram-1)*8 + lowerTrigram
	originalHex := GetHexagramByNumber(hexNumber)

	// 变卦
	changedUpper, changedLower := CalcChangedHexagram(upperTrigram, lowerTrigram, movingYao)
	changedNumber := (changedUpper-1)*8 + changedLower
	changedHex := GetHexagramByNumber(changedNumber)

	// 互卦
	interUpper, interLower := CalcInterHexagram(upperTrigram, lowerTrigram)
	interNumber := (interUpper-1)*8 + interLower
	interHex := GetHexagramByNumber(interNumber)

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
		OriginalHexagram:   originalHex,
		ChangedHexagram:    changedHex,
		InterHexagram:      interHex,
		UpperTrigramID:     upperTrigram,
		LowerTrigramID:     lowerTrigram,
		UpperTrigramName:   TrigramNameMap[upperTrigram],
		LowerTrigramName:   TrigramNameMap[lowerTrigram],
		MovingYao:          movingYao,
		OriginalYaoLines:   originalLines,
		ChangedYaoLines:    changedLines,
		InterUpperTrigram:  interUpper,
		InterLowerTrigram:  interLower,
		Interpretation:     interpretation,
	}
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
