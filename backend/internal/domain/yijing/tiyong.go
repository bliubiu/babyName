package yijing

import (
	"fmt"
	"strings"
)

// ============================================================================
// 梅花易数体用断（docs/42 P1-③：体用框架 + 卦气旺衰替代纯打分）
//
// 传统梅花体用（《梅花易数》体用总诀）：起卦后以动爻所在之卦为「用卦」，
// 静而不动之卦为「体卦」。体为主、为己，用为事、为应。
//   - 用生体：大吉（受外力滋养）
//   - 体用比和：吉（同气相求）
//   - 体克用：小吉（我能制事，但需费力）→ 三档映射为「中」
//   - 体生用：泄气（耗损于外）→ 三档映射为「中」
//   - 用克体：凶（受事所制）
//
// 卦气旺衰：以体卦五行对照出生季节（月令）定旺相休囚死——
// 旺=当令、相=令所生、休=生令者、囚=克令者、死=令所克。
// 体卦得令（旺相）能任克泄，失令（休囚死）喜生扶，为文字化建议的依据。
//
// 与 docs/42 计划第 6 条「名卦为体、姓卦为用」的偏差说明：
// 计划稿按生成来源定体用，与《梅花易数》「动者为用」的传统口径不符
// （AGENTS.md 要求严格遵循传统易学理论），故本实现以动爻位置定体用；
// 姓起上卦、名起下卦不变，动爻落于何卦即以何卦为用。
// ============================================================================

// TiYongAnalysis 姓名卦体用断结果（文字化呈现，不打分）
type TiYongAnalysis struct {
	TiGua      string `json:"ti_gua"`      // 体卦名（静卦）
	YongGua    string `json:"yong_gua"`    // 用卦名（动爻所在卦）
	TiWuxing   string `json:"ti_wuxing"`   // 体卦五行
	YongWuxing string `json:"yong_wuxing"` // 用卦五行
	Relation   string `json:"relation"`    // 用生体/比和/体克用/体生用/用克体
	Verdict    string `json:"verdict"`     // 吉/中/凶
	WangShuai  string `json:"wang_shuai"`  // 体卦旺衰：旺/相/休/囚/死（无季节时空）
	XiyongNote string `json:"xiyong_note"` // 体卦与八字喜用神关系说明（无输入时空）
	Text       string `json:"text"`        // 文字化解读
}

// seasonWangShuaiTable 卦气旺衰表：季节 → 五行旺衰
//
// 标准口径：旺=当令、相=令所生、休=生令者、囚=克令者、死=令所克。
// 土寄四维（辰戌丑未月当令）随季节归并为春夏秋冬四档，从简处理。
var seasonWangShuaiTable = map[string]map[string]string{
	"春": {"木": "旺", "火": "相", "水": "休", "金": "囚", "土": "死"},
	"夏": {"火": "旺", "土": "相", "木": "休", "水": "囚", "金": "死"},
	"秋": {"金": "旺", "水": "相", "土": "休", "火": "囚", "木": "死"},
	"冬": {"水": "旺", "木": "相", "金": "休", "土": "囚", "火": "死"},
}

// wangShuaiAdvice 体卦旺衰的文字化建议
var wangShuaiAdvice = map[string]string{
	"旺": "体卦得令而旺，能任财官，主名主精力充沛、根基稳固",
	"相": "体卦次旺，得时得势，主名主成长势头良好",
	"休": "体卦气机平缓，宜静养蓄势，中平之象",
	"囚": "体卦受令气所困，宜有生扶之字相济，防力不从心",
	"死": "体卦失令气衰，亟需卦外生扶（名中补其五行），否则根基偏弱",
}

// AnalyzeTiYong 姓名卦体用断
//
// upperTrigram/lowerTrigram 为本卦上下卦 ID（1-8，TrigramWuxingMap 口径），
// movingYao 为动爻位置（1-6，自下而上；越界返回 nil 表示无法定用卦），
// birthSeason 为出生季节（春/夏/秋/冬，取自八字月支；空则跳过旺衰），
// xiyongshen 为八字喜用神五行（空则跳过喜忌对照）。
func AnalyzeTiYong(upperTrigram, lowerTrigram, movingYao int, birthSeason string, xiyongshen []string) *TiYongAnalysis {
	if movingYao < 1 || movingYao > 6 {
		return nil
	}
	if upperTrigram < 1 || upperTrigram > 8 || lowerTrigram < 1 || lowerTrigram > 8 {
		return nil
	}

	// 动者为用：动爻在下三爻（1-3）→ 下卦为用、上卦为体；
	// 在上三爻（4-6）→ 上卦为用、下卦为体
	tiID, yongID := upperTrigram, lowerTrigram
	if movingYao >= 4 {
		tiID, yongID = lowerTrigram, upperTrigram
	}

	tiWuxing := TrigramWuxingMap[tiID]
	yongWuxing := TrigramWuxingMap[yongID]

	relation, verdict := tiYongRelation(tiWuxing, yongWuxing)

	a := &TiYongAnalysis{
		TiGua:      TrigramNameMap[tiID],
		YongGua:    TrigramNameMap[yongID],
		TiWuxing:   tiWuxing,
		YongWuxing: yongWuxing,
		Relation:   relation,
		Verdict:    verdict,
	}

	// 卦气旺衰：体卦五行 × 出生季节
	if table, ok := seasonWangShuaiTable[birthSeason]; ok {
		a.WangShuai = table[tiWuxing]
	}

	// 体卦与八字喜用神对照（单向判断，避免过度设计）
	a.XiyongNote = xiyongNoteFor(tiWuxing, xiyongshen)

	a.Text = buildTiYongText(a, birthSeason)
	return a
}

// tiYongRelation 体用五行生克关系与吉凶判定
func tiYongRelation(tiWuxing, yongWuxing string) (relation, verdict string) {
	switch {
	case tiWuxing == yongWuxing:
		return "比和", "吉"
	case shengTo(yongWuxing) == tiWuxing: // 用生体
		return "用生体", "吉"
	case shengTo(tiWuxing) == yongWuxing: // 体生用（泄气）
		return "体生用", "中"
	case keTo(tiWuxing) == yongWuxing: // 体克用（小吉）
		return "体克用", "中"
	default: // 用克体
		return "用克体", "凶"
	}
}

// shengTo 我所生之五行（木生火…）
func shengTo(wuxing string) string {
	switch wuxing {
	case "木":
		return "火"
	case "火":
		return "土"
	case "土":
		return "金"
	case "金":
		return "水"
	case "水":
		return "木"
	}
	return ""
}

// keTo 我所克之五行（木克土…）
func keTo(wuxing string) string {
	switch wuxing {
	case "木":
		return "土"
	case "土":
		return "水"
	case "水":
		return "火"
	case "火":
		return "金"
	case "金":
		return "木"
	}
	return ""
}

// xiyongNoteFor 体卦五行与喜用神的关系说明；无喜用神输入时返回空
func xiyongNoteFor(tiWuxing string, xiyongshen []string) string {
	if len(xiyongshen) == 0 {
		return ""
	}
	for _, xy := range xiyongshen {
		if xy == tiWuxing {
			return fmt.Sprintf("体卦五行%s为八字喜用神，卦象直接补益命局", tiWuxing)
		}
	}
	// 生扶/泄秀等次级关系：喜用神生体 / 体生喜用神
	for _, xy := range xiyongshen {
		if shengTo(xy) == tiWuxing {
			return fmt.Sprintf("喜用神%s生扶体卦%s，卦气有所依托", xy, tiWuxing)
		}
	}
	for _, xy := range xiyongshen {
		if shengTo(tiWuxing) == xy {
			return fmt.Sprintf("体卦%s生喜用神%s，泄秀于用神，间接调和命局", tiWuxing, xy)
		}
	}
	return fmt.Sprintf("体卦五行%s与喜用神%s无直接生扶关系", tiWuxing, strings.Join(xiyongshen, "、"))
}

// buildTiYongText 组装文字化解读（体用生克 + 卦气旺衰 + 喜用对照）
func buildTiYongText(a *TiYongAnalysis, birthSeason string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "体卦%s（%s），用卦%s（%s），%s，断为「%s」。",
		a.TiGua, a.TiWuxing, a.YongGua, a.YongWuxing, relationPhrase(a.Relation), a.Verdict)

	if advice, ok := wangShuaiAdvice[a.WangShuai]; ok {
		fmt.Fprintf(&b, "%s月%s；", birthSeason, advice)
	}
	if a.XiyongNote != "" {
		b.WriteString(a.XiyongNote)
		b.WriteString("。")
	}
	return b.String()
}

// relationPhrase 生克关系的白话短语
func relationPhrase(relation string) string {
	switch relation {
	case "用生体":
		return "用生体，外力滋养自身"
	case "比和":
		return "体用比和，同气相求"
	case "体克用":
		return "体克用，我能制事但需费力"
	case "体生用":
		return "体生用，泄气耗损于外"
	default:
		return "用克体，受事所制"
	}
}
