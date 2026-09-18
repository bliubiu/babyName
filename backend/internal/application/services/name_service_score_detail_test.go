package services

// name_service_score_detail_test.go — /names/generate 路径「评分依据」透传回归
//
// 背景（docs/24 P2-5）：前端 NameCard / NameDetail 早就写好了
// 「优先使用 score_detail（含依据文字），回退到旧字段」的分支，但
// NameService.Generate → convertFateToNameNames 只把 fate 引擎的
// Score.Items 映射成各分数字段，**丢弃了 Score.Details 依据文字**，
// 且三才维度因 name.Name 当时无对应字段而整项漏映射。
// 结果是该分支永不执行、FullReport 只能用本地文案自造命理分析。
//
// 治理：映射收敛到 applyFateScoreDetail，分数 / 依据文字 / score_detail 三件
// 事一次做完，并与 /generate/analysis 共用 buildScoreDetail 的维度顺序。
//
// 本文件断言：
//   - 八个维度（含三才）的分数都落到 name.Name 对应字段
//   - 依据文字落到既有文字字段，且缺失时不冲掉调用方已填的文案
//   - score_detail 的维度顺序稳定、条目数与命中维度一致
//   - 真实链路（fate 引擎 → convertFateToNameNames）确实产出 score_detail

import (
	"context"
	"testing"

	"name/internal/domain/fate"
	"name/internal/domain/name"
)

// scoreDetailOrder 与 DefaultRaters 权重降序一致的维度顺序（回归基准）
var scoreDetailOrder = []string{
	"五行八字", "文化印象", "音韵", "新颖度", "生肖", "共现", "三才", "人名频率",
}

// fullFateScore 构造八维度齐备的引擎评分（分数 + 依据文字）
func fullFateScore() fate.NameScore {
	return fate.NameScore{
		Total: 91.5,
		Grade: "上上",
		Items: map[string]float64{
			"五行八字": 95, "文化印象": 88, "音韵": 90, "新颖度": 76,
			"生肖": 84, "共现": 70, "三才": 92, "人名频率": 66,
		},
		Details: map[string]string{
			"五行八字": "名字五行属水，与喜用神相合",
			"文化印象": "「清」字取意清澈明净",
			"音韵":   "声调平仄交替，读来朗朗上口",
			"三才":   "天格木、人格水、地格金，三才相生",
		},
	}
}

// TestApplyFateScoreDetail_MapsAllDimensions 八维度分数与依据文字全部落到 name.Name
//
// 其中「三才 → SancaiScore」是本次修复的漏映射点，必须单独钉住。
func TestApplyFateScoreDetail_MapsAllDimensions(t *testing.T) {
	n := name.Name{}
	applyFateScoreDetail(&n, fullFateScore())

	// ① 分数
	scoreCases := []struct {
		dim  string
		got  float64
		want float64
	}{
		{"五行八字", n.WuxingScore, 95},
		{"文化印象", n.MeaningScore, 88},
		{"音韵", n.YinyunScore, 90},
		{"新颖度", n.NoveltyScore, 76},
		{"生肖", n.ZodiacScore, 84},
		{"共现", n.BigramScore, 70},
		{"三才（此前漏映射）", n.SancaiScore, 92},
		{"人名频率", n.FrequencyScore, 66},
	}
	for _, c := range scoreCases {
		if c.got != c.want {
			t.Errorf("%s 分 = %v，期望 %v", c.dim, c.got, c.want)
		}
	}

	// ② 依据文字 → 既有文字字段
	textCases := []struct {
		field string
		got   string
		want  string
	}{
		{"WuxingAnalysis", n.WuxingAnalysis, "名字五行属水，与喜用神相合"},
		{"Yinyun", n.Yinyun, "声调平仄交替，读来朗朗上口"},
		{"MeaningDetail", n.MeaningDetail, "「清」字取意清澈明净"},
		{"SancaiAnalysis", n.SancaiAnalysis, "天格木、人格水、地格金，三才相生"},
	}
	for _, c := range textCases {
		if c.got != c.want {
			t.Errorf("%s = %q，期望 %q", c.field, c.got, c.want)
		}
	}

	// ③ score_detail：条目数 = 命中维度数，顺序与权重降序一致
	if len(n.ScoreDetail) != len(scoreDetailOrder) {
		t.Fatalf("score_detail 条目数 = %d，期望 %d", len(n.ScoreDetail), len(scoreDetailOrder))
	}
	for i, want := range scoreDetailOrder {
		got := n.ScoreDetail[i]
		if got.Name != want {
			t.Errorf("score_detail[%d].Name = %q，期望 %q（顺序漂移）", i, got.Name, want)
		}
		if got.Score != fullFateScore().Items[want] {
			t.Errorf("score_detail[%d] %s 分数 = %v，期望 %v",
				i, want, got.Score, fullFateScore().Items[want])
		}
		// 有依据文字的维度必须带上，没有的保持空串（前端按空串隐藏）
		if wantDetail := fullFateScore().Details[want]; got.Detail != wantDetail {
			t.Errorf("score_detail[%d] %s 依据文字 = %q，期望 %q",
				i, want, got.Detail, wantDetail)
		}
	}
}

// TestApplyFateScoreDetail_KeepsExistingTextWhenDetailMissing
// 引擎未给某维度依据文字时，不得用空串冲掉调用方已填好的文案
func TestApplyFateScoreDetail_KeepsExistingTextWhenDetailMissing(t *testing.T) {
	n := name.Name{WuxingAnalysis: "已有五行文案", Yinyun: "已有音韵文案"}
	score := fate.NameScore{
		Items:   map[string]float64{"五行八字": 80, "音韵": 70},
		Details: map[string]string{"五行八字": "", "音韵": "  "},
	}
	applyFateScoreDetail(&n, score)

	if n.WuxingAnalysis != "已有五行文案" {
		t.Errorf("空依据文字覆盖了既有文案: %q", n.WuxingAnalysis)
	}
	if n.Yinyun != "已有音韵文案" {
		t.Errorf("空白依据文字覆盖了既有文案: %q", n.Yinyun)
	}
	// 分数仍应正常落位
	if n.WuxingScore != 80 || n.YinyunScore != 70 {
		t.Errorf("分数映射异常: wuxing=%v yinyun=%v", n.WuxingScore, n.YinyunScore)
	}
}

// TestApplyFateScoreDetail_EmptyAndUnknownDimensions
// 空评分、未知维度不得污染结构，也不得 panic
func TestApplyFateScoreDetail_EmptyAndUnknownDimensions(t *testing.T) {
	// nil 接收者
	applyFateScoreDetail(nil, fullFateScore())

	// 空评分
	empty := name.Name{}
	applyFateScoreDetail(&empty, fate.NameScore{})
	if empty.ScoreDetail != nil {
		t.Errorf("空评分应产出 nil score_detail，实际 %v", empty.ScoreDetail)
	}

	// 未知维度：不映射字段，但 score_detail 仍按白名单顺序过滤（不出现未知项）
	n := name.Name{}
	applyFateScoreDetail(&n, fate.NameScore{
		Items:   map[string]float64{"未知维度": 99, "音韵": 60},
		Details: map[string]string{"未知维度": "不该出现"},
	})
	if len(n.ScoreDetail) != 1 || n.ScoreDetail[0].Name != "音韵" {
		t.Errorf("未知维度应被过滤，实际 score_detail = %+v", n.ScoreDetail)
	}
	if n.YinyunScore != 60 {
		t.Errorf("音韵分 = %v，期望 60", n.YinyunScore)
	}
}

// TestBuildScoreDetail_SkipsMissingDimensions 缺失维度跳过、顺序保持权重降序
func TestBuildScoreDetail_SkipsMissingDimensions(t *testing.T) {
	got := buildScoreDetail(
		map[string]float64{"三才": 92, "五行八字": 95, "共现": 70},
		map[string]string{"五行八字": "依据A"},
	)

	wantOrder := []string{"五行八字", "共现", "三才"}
	if len(got) != len(wantOrder) {
		t.Fatalf("条目数 = %d，期望 %d", len(got), len(wantOrder))
	}
	for i, want := range wantOrder {
		if got[i].Name != want {
			t.Errorf("顺序漂移: [%d] = %q，期望 %q", i, got[i].Name, want)
		}
	}
	if got[0].Detail != "依据A" {
		t.Errorf("依据文字未透传: %q", got[0].Detail)
	}
	if got[1].Detail != "" {
		t.Errorf("无依据文字时应为空串，实际 %q", got[1].Detail)
	}
	// 空输入返回 nil，前端可据 null 走回退分支
	if out := buildScoreDetail(nil, nil); out != nil {
		t.Errorf("空输入应返回 nil，实际 %v", out)
	}
}

// TestGenerate_EmitsScoreDetail 真实链路回归：/names/generate 必须带 score_detail
//
// 这是 P2-5 的端到端断言——修复前该字段恒为空（omitempty ⇒ 响应里根本没有），
// 前端「优先 score_detail」的分支永远走不到。
func TestGenerate_EmitsScoreDetail(t *testing.T) {
	fs := setupFateNameServiceE2E(t)
	svc := NewNameService(WithFateService(fs))

	req := &GenerateRequest{
		Surname:    "张",
		Gender:     "male",
		BirthYear:  2024,
		BirthMonth: 5,
		BirthDay:   20,
		BirthHour:  10,
		NameLength: 2,
	}

	names, err := svc.generateNamesViaFate(context.Background(), req, nil)
	if err != nil {
		t.Fatalf("生成失败: %v", err)
	}
	if len(names) == 0 {
		t.Fatal("未生成任何名字")
	}

	withDetail := 0
	for _, n := range names {
		if len(n.ScoreDetail) == 0 {
			t.Errorf("%s 缺少 score_detail", n.FullName)
			continue
		}
		withDetail++

		// 维度名必须落在白名单内（防止未知维度泄漏到前端）
		allowed := make(map[string]bool, len(scoreDetailOrder))
		for _, d := range scoreDetailOrder {
			allowed[d] = true
		}
		hasText := false
		for _, item := range n.ScoreDetail {
			if !allowed[item.Name] {
				t.Errorf("%s score_detail 出现未知维度 %q", n.FullName, item.Name)
			}
			if item.Detail != "" {
				hasText = true
			}
		}
		// 「为什么好」是本次修复的目标：至少一个维度要带依据文字
		if !hasText {
			t.Errorf("%s score_detail 全部无依据文字，用户仍看不出「为什么好」", n.FullName)
		}
		// 三才分与 score_detail 中的三才条目应一致
		for _, item := range n.ScoreDetail {
			if item.Name == "三才" && item.Score != n.SancaiScore {
				t.Errorf("%s 三才分不一致: score=%v sancai_score=%v",
					n.FullName, item.Score, n.SancaiScore)
			}
		}
	}

	if withDetail != len(names) {
		t.Errorf("带 score_detail 的名字数 = %d/%d，期望全部", withDetail, len(names))
	}
}
