// Package yue 粤语（粤拼 Jyutping）音系数据与音韵解析。
//
// 数据来源（见 data/raw/yue/）：
//   - yyzd 粤语字典（简体/繁体两份《粤语字典_(粤拼).csv》）：汉字 → 粤拼 主源
//   - HKSCS2016.json：香港增补字符集，cantonese 字段补漏（多含生僻字）
//   - wordslist.json：粤语语料单字词，粤拼带声调，作为第三层补充
//
// 生成产物 jyutping.json（简体/繁体双映射）由上述三源合并、去重、按字建立，
// 经 //go:embed 静态嵌入，运行时零文件依赖。
//
// 音系解析聚焦粤拼 6 调 + 入声（p/t/k 塞尾），为粤语音律分（声调流畅度 /
// 声母冲突 / 韵母冲突）与粤语谐音风险分提供基础算子。
package yue

import (
	_ "embed"
	"encoding/json"
	"sort"
	"strings"
)

//go:embed jyutping.json
var jyutpingJSON []byte

// jyutpingData 嵌入的粤拼映射：简体与繁体两个键空间。
type jyutpingData struct {
	Simplified  map[string][]string `json:"simplified"`
	Traditional map[string][]string `json:"traditional"`
}

// data 进程级只读粤拼表（init 阶段加载一次）。
var data jyutpingData

func init() {
	if err := json.Unmarshal(jyutpingJSON, &data); err != nil {
		// 嵌入的静态数据解析失败属于启动期硬错误，直接 panic 以快速暴露问题。
		panic("yue: 解析 jyutping.json 失败: " + err.Error())
	}
	if data.Simplified == nil {
		data.Simplified = map[string][]string{}
	}
	if data.Traditional == nil {
		data.Traditional = map[string][]string{}
	}
}

// JyutpingOf 返回汉字的全部粤拼读音（含多音）。
//
// 优先查简体表，未命中再查繁体表（命名主用简体，但字形相同的繁体读法亦可复用）。
// 未收录返回 nil（调用方应降级处理，不报错）。
func JyutpingOf(char string) []string {
	if v, ok := data.Simplified[char]; ok {
		return v
	}
	return data.Traditional[char]
}

// PrimaryJyutping 返回汉字的第一个粤拼读音（用于评分与展示的确定性取值）。
// 未收录返回空串。
func PrimaryJyutping(char string) string {
	vs := JyutpingOf(char)
	if len(vs) == 0 {
		return ""
	}
	return vs[0]
}

// HasJyutping 汉字是否已有粤拼数据。
func HasJyutping(char string) bool {
	return len(JyutpingOf(char)) > 0
}

// StripJyutpingTone 去除粤拼尾部声调数字（1-6），得到纯音节。
// "nei5" → "nei"、"jyu5" → "jyu"、"aa1" → "aa"。
// 无尾数字（理论上不应出现）原样返回。
func StripJyutpingTone(jp string) string {
	if jp == "" {
		return ""
	}
	runes := []rune(jp)
	last := runes[len(runes)-1]
	if last >= '1' && last <= '6' {
		return string(runes[:len(runes)-1])
	}
	return jp
}

// JyutpingTone 提取粤拼声调（1-6），无法识别返回 0。
func JyutpingTone(jp string) int {
	if jp == "" {
		return 0
	}
	runes := []rune(jp)
	last := runes[len(runes)-1]
	if last >= '1' && last <= '6' {
		return int(last - '0')
	}
	return 0
}

// yueInitials 粤拼声母（按长度降序，便于优先匹配两字母声母 gw/kw/ng）。
var yueInitials = []string{"gw", "kw", "ng", "b", "p", "m", "f", "d", "t", "n", "l", "g", "k", "h", "w", "z", "c", "s", "j"}

// JyutpingInitial 提取粤拼声母；零声母（以元音或 j/y 起头的纯韵腹）返回空串。
// "nei5" → "n"、"gwong2" → "gw"、"aa1" → ""、"jyu5" → "j"、"ng5" → "ng"。
func JyutpingInitial(jp string) string {
	core := StripJyutpingTone(jp)
	if core == "" {
		return ""
	}
	for _, sm := range yueInitials {
		if strings.HasPrefix(core, sm) {
			return sm
		}
	}
	return ""
}

// JyutpingFinal 提取粤拼韵母（去声母、去声调后的剩余部分）。
// "nei5" → "ei"、"gwong2" → "ong"、"aa1" → "aa"、"m4" → "m"、"ng5" → ""。
func JyutpingFinal(jp string) string {
	core := StripJyutpingTone(jp)
	if core == "" {
		return ""
	}
	for _, sm := range yueInitials {
		if strings.HasPrefix(core, sm) {
			return core[len(sm):]
		}
	}
	return core
}

// IsEnteringTone 判定粤拼是否为入声（韵母以塞音尾 p/t/k 收束）。
// 入声音节短促，在名字收束感评估中单独处理。
func IsEnteringTone(jp string) bool {
	f := JyutpingFinal(jp)
	if f == "" {
		return false
	}
	switch f[len(f)-1] {
	case 'p', 't', 'k':
		return true
	}
	return false
}

// ToneClass 把粤拼 6 调归为「平(0)/仄(1)」两类，供声调流畅度评估。
//
// 粤语传统平仄：平声 = 1（高平）、2（中升）；仄声 = 3（中平? 实际归仄更稳妥）、
// 4（低降）、5（低升）、6（阳去/低平）。此处取简化口径：平={1,2}，仄={3,4,5,6}，
// 与普通话「1/2=平、3/4=仄」概念对齐，便于复用 evaluateTonePattern 的平仄交替逻辑。
func ToneClass(tone int) int {
	switch tone {
	case 1, 2:
		return 0 // 平
	case 3, 4, 5, 6:
		return 1 // 仄
	}
	return -1
}

// yueShengMuGroups 粤拼声母按发音部位分组（沿用 fate.shengMuGroups 的梯度设计）。
//
// 同组（口型变化小）→ 听感模糊；相邻组 → 过渡自然；远隔组 → 吐字清晰。
var yueShengMuGroups = map[string]int{
	"b": 1, "p": 1, "m": 1, // 双唇
	"f": 2,                         // 唇齿
	"d": 3, "t": 3, "n": 3, "l": 3, // 齿龈（舌尖中）
	"g": 4, "k": 4, "ng": 4, "h": 4, // 软腭/舌根（含舌根鼻音 ng）
	"gw": 5, "kw": 5, "w": 5, // 唇化软腭 / 半元音
	"z": 6, "c": 6, "s": 6, // 齿龈塞擦/擦（平舌）
	"j": 7, // 硬腭（粤语唯一舌面音）
}

// yueShengMuGroupNames 分组中文名（用于评分文案）。
var yueShengMuGroupNames = map[int]string{
	1: "双唇", 2: "唇齿", 3: "齿龈", 4: "舌根", 5: "唇化/半元音", 6: "平舌", 7: "硬腭",
}

// ShengMuGroup 返回声母的发音部位分组号（未知声母返回 0）。
func ShengMuGroup(sm string) int { return yueShengMuGroups[sm] }

// ShengMuGroupName 返回声母分组的汉字名。
func ShengMuGroupName(group int) string {
	if n, ok := yueShengMuGroupNames[group]; ok {
		return n
	}
	return "未知"
}

// AllChars 返回已收录粤拼的全部汉字（供测试/覆盖率统计）。
func AllChars() []string {
	out := make([]string, 0, len(data.Simplified))
	for c := range data.Simplified {
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}
