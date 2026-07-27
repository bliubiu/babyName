package fate

import "testing"

// TestIsSensitiveChar 敏感字集合判断（AGENTS.md 规则2 "大名需大命"）
func TestIsSensitiveChar(t *testing.T) {
	sensitive := []string{"龙", "凤", "乾", "坤", "圣", "贤", "天", "帝", "皇", "神", "仙", "君"}
	for _, ch := range sensitive {
		if !IsSensitiveChar(ch) {
			t.Errorf("IsSensitiveChar(%q) = false, 期望 true（敏感字集合）", ch)
		}
	}

	normal := []string{"明", "轩", "宇", "涵", "睿", "", "a", "1"}
	for _, ch := range normal {
		if IsSensitiveChar(ch) {
			t.Errorf("IsSensitiveChar(%q) = true, 期望 false（非敏感字）", ch)
		}
	}
}

// TestIsForbiddenEntity 禁止国家机关单位名称判断（AGENTS.md 安全策略）
func TestIsForbiddenEntity(t *testing.T) {
	// 全名匹配
	forbidden := []string{"中国", "美国", "国务院", "全国人大", "军委", "国防部"}
	for _, name := range forbidden {
		if !IsForbiddenEntity(name) {
			t.Errorf("IsForbiddenEntity(%q) = false, 期望 true（禁止实体）", name)
		}
	}

	// 名字部分匹配（去掉姓氏后的部分）
	if !IsForbiddenEntity("李国务院") {
		t.Error("IsForbiddenEntity(\"李国务院\") = false, 期望 true（包含禁止实体）")
	}

	// 正常名字
	normal := []string{"张三", "李明", "王轩宇", "赵涵睿"}
	for _, name := range normal {
		if IsForbiddenEntity(name) {
			t.Errorf("IsForbiddenEntity(%q) = true, 期望 false（正常名字）", name)
		}
	}
}

// TestIsExtremeStrongPattern 极旺格局判断（专旺格/从强格）
// 极旺格局可承载敏感字，普通格局需移除（"谦受益"原则）
func TestIsExtremeStrongPattern(t *testing.T) {
	// nil 安全检查
	if isExtremeStrongPattern(nil) {
		t.Error("isExtremeStrongPattern(nil) = true, 期望 false")
	}

	// 普通格局（非身旺）→ false
	normalFate := &FateData{
		WuXingXiji: WuXingXiji{
			QiangRuo:    "身弱",
			RiZhuWuXing: "木",
		},
	}
	if isExtremeStrongPattern(normalFate) {
		t.Error("身弱格局应返回 false")
	}

	// 身旺但日主五行空 → false
	emptyWuxing := &FateData{
		WuXingXiji: WuXingXiji{
			QiangRuo:    "身旺",
			RiZhuWuXing: "",
		},
	}
	if isExtremeStrongPattern(emptyWuxing) {
		t.Error("日主五行为空应返回 false")
	}

	// 身旺 + 日主木 + 同类占比 >= 3/4 → true
	// 木的同类：木（比劫）、水（印，水生木）
	strongFate := &FateData{
		WuXingXiji: WuXingXiji{
			QiangRuo:    "身旺",
			RiZhuWuXing: "木",
		},
		BaziInfo: BaziInfo{
			WuXing: [4]string{"木", "水", "木", "火"},
		},
	}
	if !isExtremeStrongPattern(strongFate) {
		t.Error("身旺+日主木+四柱同类3个应返回 true（极旺格局）")
	}

	// 身旺 + 日主木 + 同类占比 < 3/4 → false
	weakStrongFate := &FateData{
		WuXingXiji: WuXingXiji{
			QiangRuo:    "身旺",
			RiZhuWuXing: "木",
		},
		BaziInfo: BaziInfo{
			WuXing: [4]string{"木", "火", "土", "金"},
		},
	}
	if isExtremeStrongPattern(weakStrongFate) {
		t.Error("身旺+日主木+四柱同类1个应返回 false（非极旺）")
	}

	// 身旺 + 日主火 + 同类正好3个 → true
	// 火的同类：火（比劫）、木（印，木生火）
	fireStrong := &FateData{
		WuXingXiji: WuXingXiji{
			QiangRuo:    "身旺",
			RiZhuWuXing: "火",
		},
		BaziInfo: BaziInfo{
			WuXing: [4]string{"火", "木", "火", "土"},
		},
	}
	if !isExtremeStrongPattern(fireStrong) {
		t.Error("身旺+日主火+四柱同类3个应返回 true")
	}
}
