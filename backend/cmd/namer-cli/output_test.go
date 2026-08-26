package main

import (
	"encoding/json"
	"strings"
	"testing"

	"name/internal/application/services"
	"name/internal/domain/bazi"
	"name/internal/domain/name"
)

// buildTestResponse 构造用于输出测试的生成结果
func buildTestResponse() *services.GenerateWithAnalysisResponse {
	return &services.GenerateWithAnalysisResponse{
		Bazi: bazi.BaziAnalysis{
			Bazi:       bazi.Bazi{Year: "癸卯", Month: "乙丑", Day: "戊子", Hour: "戊午"},
			Xiyongshen: []string{"水", "金"},
			Nayin:      "海中金",
		},
		Nayin:  "海中金",
		Zodiac: "兔",
		Names: []*name.NameAnalysis{
			{
				Surname: "王", GivenName: "泽宇", FullName: "王泽宇",
				Pinyin: "zé yǔ", Strokes: 16, Wuxing: "水、土",
				TotalScore: 92.5, WuxingScore: 95,
			},
			{
				Surname: "王", GivenName: "浩然", FullName: "王浩然",
				Pinyin: "hào rán", Strokes: 20, Wuxing: "水、金",
				TotalScore: 88.0, WuxingScore: 90,
			},
			{
				Surname: "王", GivenName: "子墨", FullName: "王子墨",
				Pinyin: "zǐ mò", Strokes: 14, Wuxing: "水、土",
				TotalScore: 85.5, WuxingScore: 82,
			},
		},
	}
}

// TestRenderJSON 验证 JSON 输出为合法且包含核心字段的 JSON
func TestRenderJSON(t *testing.T) {
	resp := buildTestResponse()

	out, err := RenderJSON(resp)
	if err != nil {
		t.Fatalf("RenderJSON 返回错误: %v", err)
	}

	// 必须是合法 JSON
	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("输出不是合法 JSON: %v\n内容: %s", err, out)
	}

	// 核心字段必须存在
	for _, key := range []string{"bazi", "nayin", "zodiac", "names"} {
		if _, ok := decoded[key]; !ok {
			t.Errorf("JSON 输出缺少字段 %q", key)
		}
	}

	// 名字全名必须出现
	if !strings.Contains(out, "王泽宇") || !strings.Contains(out, "full_name") {
		t.Error("JSON 输出应包含 full_name 与名字内容")
	}
}

// TestRenderTextContainsKeyInfo 验证文本输出包含八字与名字关键信息
func TestRenderTextContainsKeyInfo(t *testing.T) {
	resp := buildTestResponse()

	out := RenderText(resp, 20)

	for _, want := range []string{"王泽宇", "王浩然", "王子墨", "癸卯", "水", "金", "兔"} {
		if !strings.Contains(out, want) {
			t.Errorf("文本输出应包含「%s」\n实际输出:\n%s", want, out)
		}
	}
}

// TestRenderTextCountLimit 验证展示数量截断
func TestRenderTextCountLimit(t *testing.T) {
	resp := buildTestResponse()

	out := RenderText(resp, 2)

	if !strings.Contains(out, "王泽宇") || !strings.Contains(out, "王浩然") {
		t.Errorf("count=2 应展示前两名\n实际输出:\n%s", out)
	}
	if strings.Contains(out, "王子墨") {
		t.Errorf("count=2 不应展示第三名\n实际输出:\n%s", out)
	}
}

// TestRenderTextEmptyPinyin 验证拼音为空时不输出空括号
func TestRenderTextEmptyPinyin(t *testing.T) {
	resp := buildTestResponse()
	resp.Names[0].Pinyin = ""

	out := RenderText(resp, 20)

	if strings.Contains(out, "（）") {
		t.Errorf("拼音为空时不应输出空括号\n实际输出:\n%s", out)
	}
	if !strings.Contains(out, "王泽宇") {
		t.Errorf("名字仍应正常展示\n实际输出:\n%s", out)
	}
}

// TestRenderTextEmptyNames 验证空结果的友好提示
func TestRenderTextEmptyNames(t *testing.T) {
	resp := buildTestResponse()
	resp.Names = nil

	out := RenderText(resp, 20)

	if !strings.Contains(out, "未生成") {
		t.Errorf("空结果应有「未生成」提示\n实际输出:\n%s", out)
	}
}
