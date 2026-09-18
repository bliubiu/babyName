package fate

import (
	"strings"
	"testing"
)

// riskRiskStubProvider 风险体检测试桩（按需构造）
func riskStubProvider(chars ...*Character) *stubProvider {
	return &stubProvider{chars: chars}
}

// TestAssessNameRisks_Level0CharOutOfTable 表外字 → fail
func TestAssessNameRisks_Level0CharOutOfTable(t *testing.T) {
	p := riskStubProvider(&Character{Char: "㵘", Pinyin: []string{"màn"}, CommonLevel: 0})
	items := AssessNameRisks(p, "王", "㵘")
	found := false
	for _, it := range items {
		if it.Category == "生僻字" && it.Level == RiskFail {
			found = true
		}
	}
	if !found {
		t.Fatalf("表外字应产生 fail 级生僻字风险项，items=%v", items)
	}
	if OverallRiskLevel(items) != RiskFail {
		t.Fatalf("含表外字时汇总等级应为 fail")
	}
}

// TestAssessNameRisks_Level3CharWarn 三级字表 → warn（谨慎而非禁止）
func TestAssessNameRisks_Level3CharWarn(t *testing.T) {
	p := riskStubProvider(&Character{Char: "翀", Pinyin: []string{"chōng"}, CommonLevel: 3})
	items := AssessNameRisks(p, "王", "翀")
	found := false
	for _, it := range items {
		if it.Category == "生僻字" && it.Level == RiskWarn {
			found = true
		}
	}
	if !found {
		t.Fatalf("三级字应产生 warn 级生僻字风险项，items=%v", items)
	}
	if OverallRiskLevel(items) == RiskFail {
		t.Fatalf("仅三级字不应判为 fail")
	}
}

// TestAssessNameRisks_MultiPronunciation 多音字 → warn
func TestAssessNameRisks_MultiPronunciation(t *testing.T) {
	p := riskStubProvider(&Character{Char: "行", Pinyin: []string{"xíng", "háng"}, CommonLevel: 1})
	items := AssessNameRisks(p, "王", "行")
	found := false
	for _, it := range items {
		if it.Category == "多音字" && it.Level == RiskWarn {
			found = true
		}
	}
	if !found {
		t.Fatalf("多音字应产生 warn 级风险项，items=%v", items)
	}
}

// TestAssessNameRisks_HardNegative 硬禁用字 → fail
func TestAssessNameRisks_HardNegative(t *testing.T) {
	p := riskStubProvider(&Character{Char: "棺", Pinyin: []string{"guān"}, CommonLevel: 2})
	items := AssessNameRisks(p, "王", "棺")
	found := false
	for _, it := range items {
		if it.Category == "负面联想" && it.Level == RiskFail {
			found = true
		}
	}
	if !found {
		t.Fatalf("硬禁用字应产生 fail 级负面联想风险项，items=%v", items)
	}
}

// TestAssessNameRisks_SoftNegativeBlessingCombo 软惩罚字：普通组合 warn，祈福组合豁免
func TestAssessNameRisks_SoftNegativeBlessingCombo(t *testing.T) {
	p := riskStubProvider(
		&Character{Char: "去", Pinyin: []string{"qù"}, CommonLevel: 1},
		&Character{Char: "病", Pinyin: []string{"bìng"}, CommonLevel: 1},
	)
	// 去病：祈福组合（以病祈福），不应报负面
	items := AssessNameRisks(p, "霍", "去病")
	for _, it := range items {
		if it.Category == "负面联想" {
			t.Fatalf("祈福组合「去病」不应报负面联想，items=%v", items)
		}
	}

	// 单个「病」：应报 warn
	items2 := AssessNameRisks(p, "王", "病")
	found := false
	for _, it := range items2 {
		if it.Category == "负面联想" && it.Level == RiskWarn {
			found = true
		}
	}
	if !found {
		t.Fatalf("软惩罚字单用应产生 warn 级负面联想，items=%v", items2)
	}
}

// TestAssessNameRisks_SensitiveChar 敏感字（大名需大命）→ warn
func TestAssessNameRisks_SensitiveChar(t *testing.T) {
	p := riskStubProvider(&Character{Char: "龙", Pinyin: []string{"lóng"}, CommonLevel: 1})
	items := AssessNameRisks(p, "王", "龙")
	found := false
	for _, it := range items {
		if it.Category == "敏感词" && it.Level == RiskWarn {
			found = true
		}
	}
	if !found {
		t.Fatalf("敏感字应产生 warn 级风险项，items=%v", items)
	}
}

// TestAssessNameRisks_NameFreqTier 撞名热度分档
func TestAssessNameRisks_NameFreqTier(t *testing.T) {
	hot := riskStubProvider(&Character{Char: "伟", Pinyin: []string{"wěi"}, CommonLevel: 1, NameFreqTier: 5})
	items := AssessNameRisks(hot, "王", "伟")
	found := false
	for _, it := range items {
		if it.Category == "撞名热度" && it.Level == RiskWarn {
			found = true
		}
	}
	if !found {
		t.Fatalf("频率档位 5 应产生 warn 级撞名热度，items=%v", items)
	}

	rare := riskStubProvider(&Character{Char: "屹", Pinyin: []string{"yì"}, CommonLevel: 1, NameFreqTier: 1})
	items2 := AssessNameRisks(rare, "王", "屹")
	for _, it := range items2 {
		if it.Category == "撞名热度" && it.Level == RiskWarn {
			t.Fatalf("低频字不应报撞名热度 warn，items=%v", items2)
		}
	}
}

// TestAssessNameRisks_UncollectedCharKeepsGoing 未收录字不中断，且产出 fail 项
func TestAssessNameRisks_UncollectedCharKeepsGoing(t *testing.T) {
	p := riskStubProvider(&Character{Char: "伟", Pinyin: []string{"wěi"}, CommonLevel: 1, NameFreqTier: 5})
	items := AssessNameRisks(p, "王", "伟龘")
	failCount := 0
	hasUncollected := false
	for _, it := range items {
		if it.Level == RiskFail {
			failCount++
		}
		if strings.Contains(it.Detail, "龘") {
			hasUncollected = true
		}
	}
	if !hasUncollected {
		t.Fatalf("未收录字「龘」应产出风险项，items=%v", items)
	}
	if failCount == 0 {
		t.Fatalf("含未收录字时汇总应为 fail")
	}
}

// TestAssessNameRisks_BadLength 非法长度
func TestAssessNameRisks_BadLength(t *testing.T) {
	p := riskStubProvider()
	items := AssessNameRisks(p, "王", "王一二三四")
	if len(items) != 1 || items[0].Level != RiskFail {
		t.Fatalf("超长名字应直接产出格式 fail 项，items=%v", items)
	}
	if OverallRiskLevel(items) != RiskFail {
		t.Fatalf("格式非法时汇总等级应为 fail")
	}
}

// TestAssessNameRisks_SurnameNotPunishedForHomophone 姓氏同音负面词不归咎名字
func TestAssessNameRisks_SurnameNotPunishedForHomophone(t *testing.T) {
	p := riskStubProvider(
		&Character{Char: "赵", Pinyin: []string{"zhào"}, CommonLevel: 1},
		&Character{Char: "心", Pinyin: []string{"xīn"}, CommonLevel: 1},
	)
	// 姓氏「赵」拼音 zhào 与负面词「罩」同音，但姓氏是既定事实，不应判名字谐音 fail
	items := AssessNameRisks(p, "赵", "心")
	for _, it := range items {
		if it.Category == "谐音" {
			t.Fatalf("姓氏同音负面词不应归咎名字，items=%v", items)
		}
	}
}

// TestAssessNameRisks_ComboHomophoneStillCatches 姓氏+名字连读的糟糕联想仍拦截
func TestAssessNameRisks_ComboHomophoneStillCatches(t *testing.T) {
	p := riskStubProvider(
		&Character{Char: "王", Pinyin: []string{"wáng"}, CommonLevel: 1},
		&Character{Char: "赵", Pinyin: []string{"zhào"}, CommonLevel: 1},
		&Character{Char: "八", Pinyin: []string{"bā"}, CommonLevel: 1},
	)
	// 「王八」为姓氏+名字连读的不雅称谓，应判 fail（由连读检测负责）
	items := AssessNameRisks(p, "王", "八")
	found := false
	for _, it := range items {
		if it.Category == "谐音" && it.Level == RiskFail && strings.Contains(it.Detail, "王八") {
			found = true
		}
	}
	if !found {
		t.Fatalf("「王八」连读应产生谐音 fail，items=%v", items)
	}

	// 对照：姓「赵」名「八」无不良连读，不应报谐音
	items2 := AssessNameRisks(p, "赵", "八")
	for _, it := range items2 {
		if it.Category == "谐音" {
			t.Fatalf("「赵八」不应报谐音，items=%v", items2)
		}
	}
}

// TestAssessNameRisks_GoodNamePass 正常好字 → 无 fail，汇总 pass/warn
func TestAssessNameRisks_GoodNamePass(t *testing.T) {
	p := riskStubProvider(
		&Character{Char: "浩", Pinyin: []string{"hào"}, CommonLevel: 1, NameFreqTier: 4},
		&Character{Char: "然", Pinyin: []string{"rán"}, CommonLevel: 1, NameFreqTier: 4},
	)
	items := AssessNameRisks(p, "李", "浩然")
	for _, it := range items {
		if it.Level == RiskFail {
			t.Fatalf("正常名字不应有 fail 项，items=%v", items)
		}
	}
}
