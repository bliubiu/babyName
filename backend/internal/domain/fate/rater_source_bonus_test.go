package fate

// rater_source_bonus_test.go — WenHuaRater「经典来源偏好加分」单测
//
// 背景（docs 用户反馈）：前端 /names/generate 选择经典来源（如《论语》）时，
// 经典来源字只在 /generate/analysis 路径被 resolveExtraChars 注入候选池，
// 而 /names/generate 路径 Options.ExtraChars 为空 → 古典来源形同虚设；
// 即便注入，评分层也无「来源偏好」概念，Top 榜仍被诗经/楚辞字靠出典加分霸榜。
//
// 治理：
//  1. name_service.go 旧路径补 ExtraChars 注入（见 name_source_inject_integration_test.go）
//  2. WenHuaRater 新增「经典来源偏好加分」：命中用户所选经典字集每字 +5 分
//     （detail「来自【论语】」），引导结果偏向所选来源。

import (
	"strings"
	"testing"

	"name/internal/domain/classics"
)

// TestWenHuaRaterSourceBonus 验证来源偏好加分的构建与评分行为
//
//   - NewWenHuaRaterWithSource 从 classics.GetPoetryCharList(source) 构建来源字集
//   - 命中来源字集的字文化分更高，details 输出「来自【来源】」
//   - 未命中来源字集的字不加分
func TestWenHuaRaterSourceBonus(t *testing.T) {
	sourceChars := classics.GetPoetryCharList("论语")
	if len(sourceChars) == 0 {
		t.Fatal("《论语》提取字集为空，测试前提不成立")
	}

	// 命中样本：论语字集中的第一个字
	var hitChar string
	set := make(map[string]bool, len(sourceChars))
	for _, pc := range sourceChars {
		set[pc.Char] = true
		if pc.Char != "" && hitChar == "" {
			hitChar = pc.Char
		}
	}
	if hitChar == "" {
		t.Fatal("《论语》提取字集中无可用汉字")
	}

	// 对照样本：候选列表里前两个不在论语字集的字（须选两个，避免对照字本身命中来源集）
	nonSource := []string{"芊", "淼", "赫", "虹", "璐", "邈", "翀", "玦", "玵", "祎"}
	var missChar, altChar string
	for _, c := range nonSource {
		if set[c] {
			continue
		}
		if missChar == "" {
			missChar = c
			continue
		}
		if c != missChar {
			altChar = c
			break
		}
	}
	if missChar == "" || altChar == "" {
		t.Skip("找不到两个非论语对照字，随机数据无法构造对照，跳过")
	}

	rater := NewWenHuaRaterWithSource(0.14, "论语")
	if len(rater.sourceSet) == 0 {
		t.Fatal("sourceSet 未从《论语》提取字集构建")
	}
	if !rater.sourceSet[hitChar] {
		t.Errorf("来源字集应包含 %q（取自论语提取字集）", hitChar)
	}
	if rater.sourceSet[missChar] || rater.sourceSet[altChar] {
		t.Errorf("对照字 %q/%q 不应在论语字集中", missChar, altChar)
	}

	// 候选字段对齐 TestWenHuaRaterCuratedBonus 的最小构造（HasPoetry=true 跳过
	// checkSemanticPoetry 近义分支，Char2 非空跳过单名共现分支，评分不受语义数据干扰）
	mkCand := func(a, b string) *NameCandidate {
		return &NameCandidate{
			Char1:        a,
			Char2:        b,
			Meaning1:     "",
			Meaning2:     "",
			IsRegular:    true,
			HasPoetry:    true, // 跳过 checkSemanticPoetry 数据依赖
			CommonLevel1: 1,
			CommonLevel2: 1,
		}
	}

	// 命中样本：Char1=论语字（+5），Char2=对照字（不加）
	hit := rater.Rate(mkCand(hitChar, missChar), nil)
	if !strings.Contains(hit.Detail, "来自【论语】") {
		t.Errorf("来源字 %q 应获得来源偏好加分，详情=%q", hitChar, hit.Detail)
	}

	// 对照样本：两字均非论语字，不加分
	miss := rater.Rate(mkCand(missChar, altChar), nil)
	if strings.Contains(miss.Detail, "来自【论语】") {
		t.Errorf("非来源字 %q/%q 不应获得来源偏好加分，详情=%q", missChar, altChar, miss.Detail)
	}

	if hit.Score <= miss.Score {
		t.Errorf("来源字 %q 文化分 = %.1f，应显著高于非来源字 %q 的 %.1f（+5/字）",
			hitChar, hit.Score, missChar, miss.Score)
	}
}

// TestWenHuaRaterSourceBonusDoubleHit 双字均命中来源字集时按字累计加分
func TestWenHuaRaterSourceBonusDoubleHit(t *testing.T) {
	sourceChars := classics.GetPoetryCharList("论语")
	set := make(map[string]bool, len(sourceChars))
	chars := make([]string, 0, len(sourceChars))
	for _, pc := range sourceChars {
		set[pc.Char] = true
		chars = append(chars, pc.Char)
	}
	var a, b string
	for _, c := range chars {
		if c == "" {
			continue
		}
		if a == "" {
			a = c
			continue
		}
		if c != a {
			b = c
			break
		}
	}
	if a == "" || b == "" {
		t.Skip("论语字集不足两个不同字，跳过")
	}

	rater := NewWenHuaRaterWithSource(0.14, "论语")

	single := rater.Rate(&NameCandidate{
		Char1: a, Char2: b, HasPoetry: true, CommonLevel1: 1, CommonLevel2: 1,
	}, nil)
	// 对照：仅一字命中（另一字为非论语字）
	var miss string
	for _, c := range []string{"芊", "淼", "赫", "虹", "璐", "邈", "翀", "玦", "玵", "祎"} {
		if set[c] {
			continue
		}
		miss = c
		break
	}
	if miss == "" {
		t.Skip("找不到非论语对照字，跳过")
	}
	mixed := rater.Rate(&NameCandidate{
		Char1: a, Char2: miss, HasPoetry: true, CommonLevel1: 1, CommonLevel2: 1,
	}, nil)

	if single.Score < mixed.Score+4 {
		t.Errorf("双来源字得分 = %.1f，应比单来源字 %.1f 高约 +5（每字累计）", single.Score, mixed.Score)
	}
}
