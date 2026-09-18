package fate

import (
	"context"
	"strings"
	"testing"
	"time"
)

// rateNameTestProvider 测名测试桩：两个字均可入名
func rateNameTestProvider() *stubProvider {
	return &stubProvider{chars: []*Character{
		{Char: "泽", Pinyin: []string{"zé"}, WuXing: "水", SimplifiedStroke: 8, ScienceStroke: 8, KangxiStroke: 17, IsRegular: true, IsNameable: true, CommonLevel: 1, Meaning: "恩泽、润泽"},
		{Char: "宇", Pinyin: []string{"yǔ"}, WuXing: "土", SimplifiedStroke: 6, ScienceStroke: 6, KangxiStroke: 6, IsRegular: true, IsNameable: true, CommonLevel: 1, Meaning: "屋宇、气宇轩昂"},
	}}
}

// TestRateGivenName_DoubleName 双名评分：分数在 [0,100]，八维齐全，结果确定
func TestRateGivenName_DoubleName(t *testing.T) {
	p := rateNameTestProvider()
	fateData, _ := (&stubAnalyzer{}).Analyze(time.Now(), GenderMale)
	filter := NewFilterOption().WithStrictness("moderate").Build()

	cand, score, err := RateGivenName(p, filter, "李", "泽宇", fateData, DefaultRaters())
	if err != nil {
		t.Fatalf("RateGivenName 失败: %v", err)
	}
	if score.Total < 0 || score.Total > 100 {
		t.Fatalf("总分越界: %v", score.Total)
	}
	if cand.Char1 != "泽" || cand.Char2 != "宇" || cand.Pinyin1 != "zé" || cand.Pinyin2 != "yǔ" ||
		cand.WuXing1 != "水" || cand.WuXing2 != "土" {
		t.Fatalf("候选字段装配错误: %+v", cand)
	}
	if len(score.Items) != len(DefaultRaters()) {
		t.Fatalf("八维评分缺失: got %d dims, want %d", len(score.Items), len(DefaultRaters()))
	}
	// 确定性：同输入两次调用总分一致
	_, score2, _ := RateGivenName(p, filter, "李", "泽宇", fateData, DefaultRaters())
	if score2.Total != score.Total {
		t.Fatalf("同输入评分不一致: %v vs %v", score.Total, score2.Total)
	}
	if len(score.Details) == 0 {
		t.Fatal("依据文字不应为空（RateName 全量明细）")
	}
}

// TestRateGivenName_SingleName 单名评分
func TestRateGivenName_SingleName(t *testing.T) {
	p := rateNameTestProvider()
	fateData, _ := (&stubAnalyzer{}).Analyze(time.Now(), GenderMale)
	filter := NewFilterOption().WithStrictness("moderate").Build()

	cand, score, err := RateGivenName(p, filter, "李", "泽", fateData, DefaultRaters())
	if err != nil {
		t.Fatalf("RateGivenName 失败: %v", err)
	}
	if cand.Char2 != "" {
		t.Fatalf("单名候选不应有第二字: %+v", cand)
	}
	if score.Total < 0 || score.Total > 100 {
		t.Fatalf("总分越界: %v", score.Total)
	}
}

// TestRateGivenName_UncollectedChar 未收录汉字报错
func TestRateGivenName_UncollectedChar(t *testing.T) {
	p := rateNameTestProvider()
	fateData, _ := (&stubAnalyzer{}).Analyze(time.Now(), GenderMale)
	filter := NewFilterOption().WithStrictness("moderate").Build()

	_, _, err := RateGivenName(p, filter, "李", "龘", fateData, DefaultRaters())
	if err == nil || !strings.Contains(err.Error(), "龘") {
		t.Fatalf("未收录字应报错并指出该字，err=%v", err)
	}
}

// TestRateGivenName_TooLong 超过两字报错
func TestRateGivenName_TooLong(t *testing.T) {
	p := rateNameTestProvider()
	fateData, _ := (&stubAnalyzer{}).Analyze(time.Now(), GenderMale)
	filter := NewFilterOption().WithStrictness("moderate").Build()

	_, _, err := RateGivenName(p, filter, "李", "泽宇轩", fateData, DefaultRaters())
	if err == nil {
		t.Fatal("三字名应报错")
	}
}

// TestRateGivenName_MatchesEngine 与引擎枚举同分：同一候选池下，
// 引擎 Top-N 中「泽宇」的枚举期总分应与 RateGivenName 的装配链评分一致
// （防「测名页与结果页同名字不同分」的回归测试）。
func TestRateGivenName_MatchesEngine(t *testing.T) {
	p := rateNameTestProvider()
	engine := NewEngine(p, &stubAnalyzer{}, DefaultRaters())
	session := engine.NewSession()
	if err := session.Start(context.Background(), &Input{
		Surname: "李",
		Gender:  GenderMale,
		Born:    time.Date(2023, 8, 20, 10, 0, 0, 0, time.UTC),
		Options: GenerateOptions{NameLength: 2, Count: 10},
	}); err != nil {
		t.Fatalf("会话启动失败: %v", err)
	}
	if err := session.Wait(); err != nil {
		t.Fatalf("生成失败: %v", err)
	}
	output := session.Result()

	var engineScore *float64
	for _, nr := range output.TopNames {
		if nr.GivenName == "泽宇" {
			s := nr.Score.Total
			engineScore = &s
			break
		}
	}
	if engineScore == nil {
		t.Skip("引擎未枚举出「泽宇」（候选池被过滤），跳过同分断言")
	}

	fateData, _ := (&stubAnalyzer{}).Analyze(time.Now(), GenderMale)
	filter := NewFilterOption().WithStrictness("moderate").Build()
	_, score, err := RateGivenName(p, filter, "李", "泽宇", fateData, DefaultRaters())
	if err != nil {
		t.Fatalf("RateGivenName 失败: %v", err)
	}
	if score.Total != *engineScore {
		t.Fatalf("同名不同分：测名=%v 引擎=%v", score.Total, *engineScore)
	}
}

// TestGivenNameStrokes 总笔画 = 姓氏笔画 + 名字康熙笔画
func TestGivenNameStrokes(t *testing.T) {
	p := rateNameTestProvider()
	total, err := GivenNameStrokes(p, "李", "泽宇")
	if err != nil {
		t.Fatalf("GivenNameStrokes 失败: %v", err)
	}
	// 李 = 7 画（stub GetSurnameStrokes 返回 rune 数…此处实际为 1）
	// stub 实现返回 len(runes)=1，泽宇 = 17+6 = 23 → 合计 24
	if total != 1+17+6 {
		t.Fatalf("总笔画错误: got %d, want %d", total, 1+17+6)
	}
}
