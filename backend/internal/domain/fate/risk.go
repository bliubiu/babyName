package fate

import (
	"fmt"
	"strings"
)

// ——— 风险体检清单 ———
//
// 背景（docs/27 §4.1 A2）：谐音/生僻字/多音字/负面联想/门禁字/撞名热度的判据
// 全部已存在于引擎与数据层（homophone.go / semantic_filter.go / naming_quality.go /
// sensitive_chars.go / Character.NameFreqTier），缺的只是一个「聚合为结构化清单」
// 的出口。本文件把这些判据聚合为逐项 pass/warn/fail 的清单，供测名报告与
// 生成结果的风险面板复用。

// RiskLevel 风险等级
type RiskLevel string

const (
	RiskPass RiskLevel = "pass" // 通过
	RiskWarn RiskLevel = "warn" // 提示（可用但需斟酌）
	RiskFail RiskLevel = "fail" // 不建议
)

// RiskItem 单项风险
type RiskItem struct {
	Category string    `json:"category"` // 类别（谐音/多音字/生僻字/负面联想/门禁字/撞名热度/户籍友好度/敏感词）
	Level    RiskLevel `json:"level"`    // 等级
	Detail   string    `json:"detail"`   // 中文说明（人话，不暴露内部术语）
}

// OverallRiskLevel 汇总风险等级：有 fail → fail；否则有 warn → warn；否则 pass。
func OverallRiskLevel(items []RiskItem) RiskLevel {
	level := RiskPass
	for _, it := range items {
		switch it.Level {
		case RiskFail:
			return RiskFail
		case RiskWarn:
			level = RiskWarn
		}
	}
	return level
}

// AssessNameRisks 对「姓氏 + 名字」做风险体检。
//
// 判据与生成路径完全同源（同一批门禁函数与同一份字库数据），
// 因此生成结果里的名字做体检，结论与评分器的惩罚逻辑互相印证。
// 未收录汉字不报错，而是作为 fail 项输出（测名场景下比直接 400 更有用）。
func AssessNameRisks(provider CharacterProvider, surname, given string) []RiskItem {
	items := make([]RiskItem, 0, 8)
	given = strings.TrimSpace(given)
	runes := []rune(given)
	if len(runes) == 0 || len(runes) > 2 {
		items = append(items, RiskItem{
			Category: "格式",
			Level:    RiskFail,
			Detail:   "名字需为 1-2 个汉字",
		})
		return items
	}

	// 逐字加载（未收录不中断，降级为 fail 项继续体检其余字）
	chars := make([]*Character, 0, len(runes))
	for _, r := range runes {
		c, err := provider.GetCharacter(string(r))
		if err != nil || c == nil {
			items = append(items, RiskItem{
				Category: "生僻字",
				Level:    RiskFail,
				Detail:   fmt.Sprintf("「%s」未被规范字库收录，户籍登记与证件办理可能受阻", string(r)),
			})
			chars = append(chars, nil)
			continue
		}
		chars = append(chars, c)
	}

	// ——— 字级检查 ———
	pinyinOf := func(c *Character) string {
		if c == nil || len(c.Pinyin) == 0 {
			return ""
		}
		return c.Pinyin[0]
	}
	for _, c := range chars {
		if c == nil {
			continue
		}
		ch := c.Char

		// 生僻字：《通用规范汉字表》三级字 → 谨慎；表外字 → 不建议
		switch {
		case c.CommonLevel == 0:
			items = append(items, RiskItem{
				Category: "生僻字",
				Level:    RiskFail,
				Detail:   fmt.Sprintf("「%s」为字表外字，户籍系统可能无法录入", ch),
			})
		case c.CommonLevel >= 3:
			items = append(items, RiskItem{
				Category: "生僻字",
				Level:    RiskWarn,
				Detail:   fmt.Sprintf("「%s」属三级字表（专用领域用字），录入通常可行但识别率低", ch),
			})
		}

		// 门禁字：虚词/排行字/口语物名/数字量词等，任何语境下都无命名价值
		if IsNonNamingChar(ch) {
			items = append(items, RiskItem{
				Category: "门禁字",
				Level:    RiskFail,
				Detail:   fmt.Sprintf("「%s」为无命名价值字（虚词/口语物名/排行字类）", ch),
			})
		}

		// 负面语义（分层：硬禁用 → fail；软惩罚 → warn，祈福组合豁免）
		isBlessing := len(chars) == 2 && chars[0] != nil && chars[1] != nil &&
			IsBlessingCombo(chars[0].Char, chars[1].Char)
		switch {
		case IsHardNegativeChar(ch) || c.IsNegative:
			items = append(items, RiskItem{
				Category: "负面联想",
				Level:    RiskFail,
				Detail:   fmt.Sprintf("「%s」含负面含义，不宜入名", ch),
			})
		case IsSoftNegativeChar(ch) && !isBlessing:
			items = append(items, RiskItem{
				Category: "负面联想",
				Level:    RiskWarn,
				Detail:   fmt.Sprintf("「%s」字义偏消极，寓意需斟酌", ch),
			})
		}

		// 数据层策展扣分
		if c.NamePenalty > 0 {
			items = append(items, RiskItem{
				Category: "负面联想",
				Level:    RiskWarn,
				Detail:   fmt.Sprintf("「%s」被标注起名扣分 %d 分，用作名字有明显减分项", ch, c.NamePenalty),
			})
		}

		// 敏感字：「大名需大命」（龙/凤/乾/坤/天/帝等），普通格局慎用
		if IsSensitiveChar(ch) {
			items = append(items, RiskItem{
				Category: "敏感词",
				Level:    RiskWarn,
				Detail:   fmt.Sprintf("「%s」气场极强，非极旺格局慎用（谦受益）", ch),
			})
		}

		// 多音字
		if len(c.Pinyin) > 1 {
			items = append(items, RiskItem{
				Category: "多音字",
				Level:    RiskWarn,
				Detail:   fmt.Sprintf("「%s」为多音字（%s），日常叫法可能被读错", ch, strings.Join(c.Pinyin, " / ")),
			})
		}
	}

	// ——— 谐音检查（只针对名字用字；姓氏交由下方连读检测） ———
	// 姓氏是既定事实，不进逐字谐音匹配：按其拼音匹配负面词会产生
	// 「赵→罩」「王→亡」类误报（几乎每个姓都有负面同音字）。
	// 姓氏参与的糟糕联想（如「杜子腾→肚子疼」）由 CheckBadPinyinCombo 连读检测覆盖。
	if surname != "" {
		surnamePinyin := firstPinyinForSurname(surname, provider)
		if surnamePinyin != "" {
			pairs := make([]string, 0, 6)
			for _, c := range chars {
				if c == nil {
					continue
				}
				pairs = append(pairs, pinyinOf(c), c.Char)
			}
			if len(pairs) > 0 {
				if hit, descs := CheckAllBadHomophones(pairs...); hit {
					items = append(items, RiskItem{
						Category: "谐音",
						Level:    RiskFail,
						Detail:   "含不吉谐音：" + strings.Join(descs, "、"),
					})
				}
			}
			givenPinyins := make([]string, 0, len(chars))
			for _, c := range chars {
				if c != nil && pinyinOf(c) != "" {
					givenPinyins = append(givenPinyins, pinyinOf(c))
				}
			}
			if len(givenPinyins) > 0 {
				if hit, desc := CheckBadPinyinCombo(surnamePinyin, givenPinyins...); hit {
					items = append(items, RiskItem{
						Category: "谐音",
						Level:    RiskFail,
						Detail:   desc,
					})
				}
			}
		}
	}

	// ——— 组合级检查（双名） ———
	if len(chars) == 2 && chars[0] != nil && chars[1] != nil {
		c1, c2 := chars[0], chars[1]
		if IsBadCombo(c1.Char, c2.Char) {
			items = append(items, RiskItem{
				Category: "负面联想",
				Level:    RiskFail,
				Detail:   fmt.Sprintf("「%s%s」为日常禁忌组合（亲属称谓/物名/动词类）", c1.Char, c2.Char),
			})
		}
		if IsHistoricalFigureCombo(c1.Char, c2.Char) {
			items = append(items, RiskItem{
				Category: "敏感词",
				Level:    RiskWarn,
				Detail:   fmt.Sprintf("「%s%s」与历史人物名号撞车，易引发联想", c1.Char, c2.Char),
			})
		}
		if hasPairBlacklist(c1, c2.Char) || hasPairBlacklist(c2, c1.Char) {
			items = append(items, RiskItem{
				Category: "负面联想",
				Level:    RiskWarn,
				Detail:   fmt.Sprintf("「%s」「%s」为策展标注的不宜搭配", c1.Char, c2.Char),
			})
		}
	}

	// ——— 撞名热度（人名语料频率档位） ———
	maxTier := 0
	for _, c := range chars {
		if c != nil && c.NameFreqTier > maxTier {
			maxTier = c.NameFreqTier
		}
	}
	switch {
	case maxTier >= 5:
		items = append(items, RiskItem{
			Category: "撞名热度",
			Level:    RiskWarn,
			Detail:   "含高频人名用字，同学重名概率较高",
		})
	case maxTier >= 3:
		items = append(items, RiskItem{
			Category: "撞名热度",
			Level:    RiskPass,
			Detail:   "用字在人名语料中较常见，重名率中等",
		})
	default:
		items = append(items, RiskItem{
			Category: "撞名热度",
			Level:    RiskPass,
			Detail:   "用字在人名语料中较少见，重名率低",
		})
	}

	// ——— 户籍友好度（常用度 + 笔画负担） ———
	allCommon := len(chars) > 0
	for _, c := range chars {
		if c == nil || c.CommonLevel < 1 || c.CommonLevel > 2 {
			allCommon = false
			break
		}
	}
	strokes := 0
	for _, c := range chars {
		if c == nil {
			continue
		}
		s := c.KangxiStroke
		if s == 0 {
			s = c.ScienceStroke
		}
		strokes += s
	}
	switch {
	case allCommon && strokes <= 32:
		items = append(items, RiskItem{
			Category: "户籍友好度",
			Level:    RiskPass,
			Detail:   "全为一二级规范字，录入无障碍",
		})
	case strokes > 32:
		items = append(items, RiskItem{
			Category: "户籍友好度",
			Level:    RiskWarn,
			Detail:   fmt.Sprintf("名字合计 %d 画，书写负担偏重", strokes),
		})
	default:
		items = append(items, RiskItem{
			Category: "户籍友好度",
			Level:    RiskWarn,
			Detail:   "含三级/表外字，录入通常可行但可能被误读误写",
		})
	}

	// ——— 国家机关单位名称（AGENTS.md 安全策略） ———
	fullName := surname + given
	if IsForbiddenEntity(fullName) {
		items = append(items, RiskItem{
			Category: "敏感词",
			Level:    RiskFail,
			Detail:   "与国家机关单位名称相同，禁止使用",
		})
	}

	return items
}
