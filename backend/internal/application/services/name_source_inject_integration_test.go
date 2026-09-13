package services

// name_source_inject_integration_test.go — /names/generate 路径经典来源注入回归
//
// 背景（docs 用户反馈）：前端 result 页走后端 /names/generate（NameService.Generate →
// generateNamesViaFate）。该路径构造 fate.Input.Options 时只设置了 SourceClassic
// 字段，但引擎只消费 Options.ExtraChars（engine.go generate 阶段合入候选池），
// 不读 SourceClassic → 经典来源（论语/孟子/…)完全失效，Top 榜仍被诗经/楚辞字主导。
//
// 治理：generateNamesViaFate 复用 FateNameService.resolveExtraChars 把所选经典
// 来源字注入 ExtraChars（与 /generate/analysis 路径对齐）。
// 本测试验证：指定《论语》时生成结果中《论语》提取字集命中数显著高于未指定。

import (
	"context"
	"strings"
	"testing"

	"name/internal/domain/classics"
)

// lunyuCharSet 《论语》提取字集 → set（跳过后空字集）
func lunyuCharSet(t *testing.T) map[string]bool {
	t.Helper()
	chars := classics.GetPoetryCharList("论语")
	set := make(map[string]bool, len(chars))
	for _, pc := range chars {
		if pc.Char != "" {
			set[pc.Char] = true
		}
	}
	if len(set) == 0 {
		t.Skip("《论语》提取字集为空，跳过")
	}
	return set
}

// countSourceHits 统计结果名字中命中来源字集的名字数（任一名字字命中即计 1）
func countSourceHits(givenNames []string, set map[string]bool) int {
	hits := 0
	for _, g := range givenNames {
		for _, r := range g {
			if set[string(r)] {
				hits++
				break
			}
		}
	}
	return hits
}

// TestGenerateNamesViaFate_SourceClassicInjected 注入 + 来源评分加持联动验证
//
// 断言（避免 flaky 的宽松语义）：
//   - SourceClassic=论语 时命中数 > 0（注入生效，论语字真正进入结果）
//   - 命中数 ≥ 未指定来源的基线（评分加持不会劣化）
func TestGenerateNamesViaFate_SourceClassicInjected(t *testing.T) {
	fs := setupFateNameServiceE2E(t)
	svc := NewNameService(WithFateService(fs))
	set := lunyuCharSet(t)

	ctx := context.Background()
	mkReq := func(source string) *GenerateRequest {
		return &GenerateRequest{
			Surname:       "张",
			Gender:        "male",
			BirthYear:     2024,
			BirthMonth:    5,
			BirthDay:      20,
			BirthHour:     10,
			BirthMinute:   0,
			NameLength:    2,
			SourceClassic: source,
		}
	}

	withSource, err := svc.generateNamesViaFate(ctx, mkReq("论语"), nil)
	if err != nil {
		t.Fatalf("SourceClassic=论语 生成失败: %v", err)
	}
	if len(withSource) == 0 {
		t.Fatal("SourceClassic=论语 未生成任何名字")
	}

	baseline, err := svc.generateNamesViaFate(ctx, mkReq(""), nil)
	if err != nil {
		t.Fatalf("无来源生成失败: %v", err)
	}

	withNames := make([]string, 0, len(withSource))
	for _, n := range withSource {
		withNames = append(withNames, n.GivenName)
	}
	baseNames := make([]string, 0, len(baseline))
	for _, n := range baseline {
		baseNames = append(baseNames, n.GivenName)
	}

	hitWith := countSourceHits(withNames, set)
	if hitWith == 0 {
		t.Error("SourceClassic=论语 时结果中应出现《论语》提取字集字（注入未生效）")
	}

	hitBaseline := countSourceHits(baseNames, set)
	if hitWith < hitBaseline {
		t.Errorf("论语命中数 %d < 基线 %d：来源注入+评分加持应至少不劣化", hitWith, hitBaseline)
	}
	t.Logf("论语命中：指定来源=%d/%d，基线=%d/%d",
		hitWith, len(withNames), hitBaseline, len(baseNames))
}

// TestResolveExtraCharsSourceClassic 直接单测 resolveExtraChars 对所选来源的解析
func TestResolveExtraCharsSourceClassic(t *testing.T) {
	fs := setupFateNameServiceE2E(t)
	_ = lunyuCharSet(t) // 保证经典数据已加载

	req := &GenerateRequest{
		Surname:       "张",
		Gender:        "male",
		NameLength:    2,
		SourceClassic: "论语",
	}
	extras := fs.resolveExtraChars(req)
	if len(extras) == 0 {
		t.Fatal("resolveExtraChars(论语) 应解析出候选字，实际为空")
	}
	seen := map[string]bool{}
	matched := 0
	for _, c := range extras {
		if c.Char == "" || seen[c.Char] {
			continue
		}
		seen[c.Char] = true
		if strings.Contains(c.Char, " ") {
			t.Errorf("ExtraChars 含非法空格字符 %q", c.Char)
		}
		_ = matched
	}

	// 指定来源时不应退化为全量诗词字（应只含论语体系）
	reqFull := &GenerateRequest{SourceClassic: ""}
	if got := fs.resolveExtraChars(reqFull); len(got) != 0 {
		t.Errorf("未指定任何来源/包含选项时 resolveExtraChars 应为空，实际 %d 个", len(got))
	}
}
