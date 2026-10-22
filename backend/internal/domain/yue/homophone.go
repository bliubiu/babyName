package yue

import (
	"strings"
)

// ——— 粤语谐音风险词库 ———
//
// 设计要点（与 fate/homophone.go 的「本字豁免」互补）：
//   - YueBadHomophones   通用不吉字，带本字豁免——仅当名字用字本身等于该字才命中，
//                        避免「珠 zyu1（珍珠）/ 思 si1」等同音好字被「豬/屎」误伤。
//   - YueHomophonePairs 明确的「好字 → 不雅谐音」配对，直接点名，无本字豁免。
//                        用来精确标记柯(o1)→屙、朱(zyu1)→豬 这类字义好但粤语谐音不雅的用字，
//                        同时绝不波及珠(zyu1) 等同音好字（通用表已用本字豁免挡住，这里只列明确配对）。
//   - YueBadCombos      粤语连读不良词（含姓氏拼音），滑窗查表，沿用 Mandarin 的 O(文本长度) 机制。
//   - 懒音近音歧义       对给定字的粤拼生成「懒音变体」（n↔l、∅↔ng、n↔ng、t↔k），
//                        任一变体命中配对的不雅字粤拼即记为 warn（近音歧义），如 吳 ng4 懒读 m4 似「唔」。

// YueBadHomophone 粤语通用不吉字（按粤拼无声调匹配，带本字豁免）。
type YueBadHomophone struct {
	Pinyin      string // 粤拼（无声调，如 sei2→sei）
	Word        string // 对应的不吉汉字（本字）
	Description string // 说明
}

// YueBadHomophones 粤语通用不吉字表（初版种子，可按需扩充）。
//
// 命中规则：仅当 名字用字本身 == Word 且粤拼（无声调）匹配时命中（本字豁免），
// 因此不会误杀同音好字（珠/思/河…）。若名字真的用了这些字，则记为不吉谐音。
var YueBadHomophones = []YueBadHomophone{
	{Pinyin: "si2", Word: "屎", Description: "排泄物（污秽）"},
	{Pinyin: "cat6", Word: "柒", Description: "粗口（不雅）"},
	{Pinyin: "gau1", Word: "鳩", Description: "粗口（贬义）"},
	{Pinyin: "diu2", Word: "屌", Description: "粗口（极不雅）"},
	{Pinyin: "hat6dat6", Word: "核突", Description: "令人反感、丑陋恶心"},
	{Pinyin: "beng6", Word: "病", Description: "疾病"},
	{Pinyin: "maang4", Word: "盲", Description: "失明"},
	{Pinyin: "be1", Word: "跛", Description: "跛脚"},
	{Pinyin: "aa2", Word: "啞", Description: "失声"},
	{Pinyin: "mung2", Word: "懵", Description: "糊涂迟钝"},
	{Pinyin: "ngong6", Word: "戇", Description: "愚笨"},
	{Pinyin: "seoi1", Word: "衰", Description: "衰败、倒霉"},
	{Pinyin: "sei2", Word: "死", Description: "死亡"},
	{Pinyin: "fu2", Word: "苦", Description: "痛苦"},
	{Pinyin: "kung4", Word: "窮", Description: "贫穷"},
	{Pinyin: "fai3", Word: "廢", Description: "废弃无用"},
	{Pinyin: "ban6", Word: "笨", Description: "愚笨"},
	{Pinyin: "cau2", Word: "醜", Description: "丑陋"},
	{Pinyin: "cau3", Word: "臭", Description: "恶臭"},
	{Pinyin: "cin5", Word: "賤", Description: "低贱"},
	{Pinyin: "jyu2", Word: "瘀", Description: "淤青、丢脸"},
	{Pinyin: "haat1", Word: "乞", Description: "乞讨（卑贱）"},
	{Pinyin: "gwai2", Word: "鬼", Description: "鬼怪（不吉）"},
	{Pinyin: "haau4", Word: "姣", Description: "轻佻风骚"},
	{Pinyin: "suk1gwat1", Word: "縮骨", Description: "吝啬刻薄"},
	{Pinyin: "ci1", Word: "痴", Description: "痴傻"},
}

// YueHomophonePair 明确的「好字 → 不雅谐音」配对（无本字豁免，直接点名）。
//
// Char 为名字用字（好字），Word 为其粤语不雅谐音字。Jyutping 为好字粤拼（按无声调匹配），
// BadJyutping 为不雅字粤拼（用于近音/懒音判定）。Severity=fail 表示同音强警示，
// =warn 表示近音/懒音歧义（弱警示）。
type YueHomophonePair struct {
	Char        string // 名字用字（好字）
	Word        string // 不雅谐音汉字
	Jyutping    string // 好字粤拼（无声调比对）
	BadJyutping string // 不雅字粤拼（无声调比对，近音/懒音判定）
	Severity    string // "fail" | "warn"
	Description string // 说明
}

var YueHomophonePairs = []YueHomophonePair{
	{Char: "柯", Word: "屙", Jyutping: "o1", BadJyutping: "o1", Severity: "fail", Description: "粤语「柯」与「屙」同音，粗俗不雅"},
	{Char: "朱", Word: "猪", Jyutping: "zyu1", BadJyutping: "zyu1", Severity: "fail", Description: "粤语「朱」与「猪」同音，易生不雅联想"},
	{Char: "苟", Word: "狗", Jyutping: "gau2", BadJyutping: "gau2", Severity: "fail", Description: "粤语「苟」与「狗」同音"},
	{Char: "史", Word: "屎", Jyutping: "si2", BadJyutping: "si2", Severity: "fail", Description: "粤语「史」与「屎」同音"},
	{Char: "赖", Word: "癞", Jyutping: "laai6", BadJyutping: "laai6", Severity: "fail", Description: "粤语「赖」与「癞」同音，癞痢恶疾"},
	{Char: "吴", Word: "唔", Jyutping: "ng4", BadJyutping: "m4", Severity: "warn", Description: "粤语「吴」懒读近「唔」（否定），近音歧义"},
	{Char: "陈", Word: "尘", Jyutping: "can4", BadJyutping: "can4", Severity: "warn", Description: "粤语「陈」与「尘」同音，尘土意象"},
	{Char: "黎", Word: "篱", Jyutping: "lai4", BadJyutping: "lai4", Severity: "warn", Description: "粤语「黎」与「篱」同音，篱笆意象偏土"},
}

// YueBadCombo 粤语连读不良词（多音节，无声调粤拼序列）。
type YueBadCombo struct {
	Combo       []string // 粤拼音节序列（无声调）
	Description string
}

// YueBadCombos 粤语连读不良词（含姓氏拼音），初版种子。
var YueBadCombos = []YueBadCombo{
	{Combo: []string{"ng4", "dak1"}, Description: "吳得→唔得（不行）"},
	{Combo: []string{"zyu1", "gu2", "lik6"}, Description: "朱古力→豬骨力（巧克力谐音，戏谑不雅）"},
	{Combo: []string{"si2", "taai3"}, Description: "史太→屎太（不雅）"},
	{Combo: []string{"o1", "naam4"}, Description: "柯南→屙南（不雅联想）"},
	{Combo: []string{"ng4", "nang4"}, Description: "吳能→唔能（不能）"},
	{Combo: []string{"wong4", "baan2"}, Description: "王板→黄板（戏谑）"},
	{Combo: []string{"zyu1", "baan2"}, Description: "朱板→豬板（不雅）"},
}

// YueHomophoneHit 单条粤语谐音命中结果。
type YueHomophoneHit struct {
	Severity    string // "fail" | "warn"
	Description string // 中文说明
}

// ——— 热路径预编译索引（与 fate/homophone.go 同构）———

var (
	yueHomophoneByPinyin map[string][]int // 无声调粤拼 → YueBadHomophones 下标
	yueComboIndex        map[string]int   // 连读拼接串 → YueBadCombos 下标
	yueComboLens         []int            // 拼接串长度去重集合（滑窗用）
	yuePairBadPinyin     map[string][]int // 配对不雅字无声调粤拼 → YueHomophonePairs 下标（懒音近音判定用）
)

func init() {
	rebuildYueHomophoneIndex()
}

// rebuildYueHomophoneIndex 依据当前词表重建索引（词表热更新后必须调用）。
func rebuildYueHomophoneIndex() {
	yueHomophoneByPinyin = make(map[string][]int, len(YueBadHomophones))
	for i := range YueBadHomophones {
		p := StripJyutpingTone(YueBadHomophones[i].Pinyin)
		yueHomophoneByPinyin[p] = append(yueHomophoneByPinyin[p], i)
	}

	yuePairBadPinyin = make(map[string][]int, len(YueHomophonePairs))
	for i := range YueHomophonePairs {
		p := StripJyutpingTone(YueHomophonePairs[i].BadJyutping)
		yuePairBadPinyin[p] = append(yuePairBadPinyin[p], i)
	}

	yueComboIndex = make(map[string]int, len(YueBadCombos))
	lenSet := make(map[int]bool)
	for i := range YueBadCombos {
		// 键统一为「无声调」拼接串，与 CheckYueBadCombo 运行时 StripJyutpingTone 后的
		// 滑窗文本口径一致（词表保留声调便于人读，索引必须去调，否则永远匹配不上）。
		var sb strings.Builder
		for _, c := range YueBadCombos[i].Combo {
			sb.WriteString(StripJyutpingTone(c))
		}
		s := sb.String()
		if _, dup := yueComboIndex[s]; !dup {
			yueComboIndex[s] = i
		}
		if !lenSet[len(s)] {
			lenSet[len(s)] = true
			yueComboLens = append(yueComboLens, len(s))
		}
	}
}

// CheckNameYueHomophones 检测名字「给定字」的粤语谐音风险（不含姓氏，姓氏交由连读检测）。
//
// 入参按顺序成对：pinyinsChars = [jp1, char1, jp2, char2, ...]。
// 返回命中列表（fail/warn 分严宽两档）。本函数只扫描名字用字，不传姓氏拼音，
// 避免「赵→罩」类姓氏误报（姓氏参与的连读不雅由 CheckYueBadCombo 负责）。
func CheckNameYueHomophones(pinyinsChars ...string) []YueHomophoneHit {
	var hits []YueHomophoneHit
	for i := 0; i < len(pinyinsChars); i += 2 {
		var jp, char string
		if i+1 < len(pinyinsChars) {
			jp, char = pinyinsChars[i], pinyinsChars[i+1]
		} else {
			jp = pinyinsChars[i]
		}
		clean := StripJyutpingTone(jp)
		if clean == "" || char == "" {
			continue
		}
		// 1) 通用不吉字（本字豁免：仅当名字用字本身 == Word 才命中）
		if idxs, ok := yueHomophoneByPinyin[clean]; ok {
			for _, idx := range idxs {
				if YueBadHomophones[idx].Word == char {
					hits = append(hits, YueHomophoneHit{
						Severity:    "fail",
						Description: "粤语谐音「" + char + "」" + YueBadHomophones[idx].Description,
					})
					break
				}
			}
		}
		// 2) 明确的「好字→不雅谐音」配对（直接点名，无本字豁免）
		for _, p := range YueHomophonePairs {
			if p.Char == char {
				severity := p.Severity
				desc := "粤语谐音：" + p.Char + "（" + jp + "）近似「" + p.Word + "」——" + p.Description
				hits = append(hits, YueHomophoneHit{Severity: severity, Description: desc})
				break
			}
		}
		// 3) 懒音近音歧义：本字粤拼的「懒音变体」（排除本字本身）命中某配对的不雅字
		//    粤拼 → warn。排除本字本身是为了贯彻「不误伤同音好字」原则：若本字粤拼
		//    本身即与某不雅字同音（如 珠 zyu1 与 朱→豬 的 BadJyutping zyu1 相同），
		//    该情况只对「明确点名的好字」（第 2 步的 Char）生效，绝不波及珠这类同音好字；
		//    只有本字经懒音规则（n↔l、∅↔ng 等）**变读**后才近似不雅字，才记为 warn。
		for _, v := range lazyVariants(clean) {
			if v == clean {
				continue
			}
			if idxs, ok := yuePairBadPinyin[v]; ok {
				for _, idx := range idxs {
					p := YueHomophonePairs[idx]
					if p.Char == char {
						continue // 已在第 2 步精确命中，避免重复
					}
					hits = append(hits, YueHomophoneHit{
						Severity:    "warn",
						Description: "粤语近音歧义：" + char + "（" + jp + "）懒读近似「" + p.Word + "」——" + p.Description,
					})
					break
				}
			}
		}
	}
	return hits
}

// CheckYueBadCombo 检测名字粤语拼音连读是否产生不良词汇（含姓氏拼音）。
//
// surnameJp 姓氏粤拼，givenJps 名字各字粤拼。滑窗匹配 YueBadCombos（预编译索引，O(文本长度)）。
func CheckYueBadCombo(surnameJp string, givenJps ...string) (bool, string) {
	var sb strings.Builder
	if surnameJp != "" {
		sb.WriteString(StripJyutpingTone(surnameJp))
	}
	for _, p := range givenJps {
		sb.WriteString(StripJyutpingTone(p))
	}
	joined := sb.String()
	if joined == "" {
		return false, ""
	}
	best := -1
	for i := 0; i < len(joined); i++ {
		for _, l := range yueComboLens {
			end := i + l
			if end > len(joined) {
				continue
			}
			if idx, ok := yueComboIndex[joined[i:end]]; ok && (best < 0 || idx < best) {
				best = idx
			}
		}
	}
	if best < 0 {
		return false, ""
	}
	comboStr := strings.Join(YueBadCombos[best].Combo, "")
	return true, "粤语连读「" + comboStr + "」" + YueBadCombos[best].Description
}

// lazySubstitutions 懒音/近音替换规则（经典粤语懒音对）。
// 每对的两个元素可互转：n↔l、∅↔ng（零声母与舌根鼻音）、n↔ng（韵尾）、t↔k（塞音韵尾）。
var lazySubstitutions = [][2]string{
	{"n", "l"},
	{"", "ng"}, // 零声母 ↔ ng（如 亞 aa ↔ 牙 nga；吳 ng ↔ 唔 m 经此近似）
	{"n", "ng"},
	{"t", "k"},
}

// lazyVariants 对无声调粤拼生成懒音变体（返回含自身的去重集合）。
//
// 仅做「首字母 n/l 互换」「前缀 ∅/ng 互换」「尾字母 n/ng 互换」「尾字母 t/k 互换」，
// 覆盖最常见的懒音歧义；变体用于与配对不雅字粤拼比对，控制误报范围。
func lazyVariants(jp string) []string {
	if jp == "" {
		return nil
	}
	set := map[string]struct{}{jp: {}}
	add := func(v string) {
		if v != "" && v != jp {
			set[v] = struct{}{}
		}
	}
	// 首字母 n/l 互换
	switch {
	case strings.HasPrefix(jp, "n"):
		add("l" + jp[1:])
	case strings.HasPrefix(jp, "l"):
		add("n" + jp[1:])
	}
	// 前缀 ∅/ng 互换
	if strings.HasPrefix(jp, "ng") {
		add(jp[2:]) // 去掉 ng
	} else if isVowelStart(jp) {
		add("ng" + jp) // 加 ng
	}
	// 尾字母 n/ng 互换
	if strings.HasSuffix(jp, "n") {
		add(jp[:len(jp)-1] + "ng")
	} else if strings.HasSuffix(jp, "ng") {
		add(jp[:len(jp)-2] + "n")
	}
	// 尾字母 t/k 互换
	if strings.HasSuffix(jp, "t") {
		add(jp[:len(jp)-1] + "k")
	} else if strings.HasSuffix(jp, "k") {
		add(jp[:len(jp)-1] + "t")
	}
	out := make([]string, 0, len(set))
	for v := range set {
		out = append(out, v)
	}
	return out
}

// isVowelStart 判断粤拼是否以元音（零声母）起头。
func isVowelStart(jp string) bool {
	if jp == "" {
		return false
	}
	switch jp[0] {
	case 'a', 'e', 'i', 'o', 'u', 'y':
		return true
	}
	return false
}
