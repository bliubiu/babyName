package fate

import (
	"context"
	"math"
	"testing"
	"time"
)

// yueFateData 构造固定八字数据（供粤语评分器 Rate 的 fateData 形参使用）
func yueFateData(t *testing.T) *FateData {
	t.Helper()
	fd, err := (&stubAnalyzer{}).Analyze(time.Date(2023, 8, 20, 10, 0, 0, 0, time.UTC), GenderMale)
	if err != nil {
		t.Fatalf("八字分析失败: %v", err)
	}
	return fd
}

// TestDefaultCantoneseRaters 粤语评分器权重内部归一（和为 1.0），两个维度齐全
func TestDefaultCantoneseRaters(t *testing.T) {
	raters := DefaultCantoneseRaters()
	if len(raters) != 2 {
		t.Fatalf("粤语评分器应有 2 项, got %d", len(raters))
	}
	var sum float64
	names := map[string]bool{}
	for _, r := range raters {
		sum += r.Weight()
		names[r.Name()] = true
	}
	if math.Abs(sum-1.0) > 1e-9 {
		t.Errorf("粤语评分器权重和应为 1.0, got %v", sum)
	}
	if !names["粤语音律"] || !names["粤语谐音"] {
		t.Errorf("粤语维度缺失: %v", names)
	}
}

// TestYueYinYunRater_MissingData 粤拼缺失 → 中性分 75
func TestYueYinYunRater_MissingData(t *testing.T) {
	r := NewYueYinYunRater()
	cand := &NameCandidate{Char1: "某"} // 无粤拼字段
	got := r.Rate(cand, yueFateData(t))
	if got.Score != 75 {
		t.Errorf("粤拼缺失应为中性分 75, got %v", got.Score)
	}
}

// TestYueYinYunRater_Range 有粤拼时分数落在 [0,100] 且带依据文案
func TestYueYinYunRater_Range(t *testing.T) {
	r := NewYueYinYunRater()
	cand := &NameCandidate{
		Char1: "泽", Char2: "宇",
		Jyutping1: "zaak6", Jyutping2: "jyu5", SurnameJyutping: "lei5",
	}
	got := r.Rate(cand, yueFateData(t))
	if got.Score < 0 || got.Score > 100 {
		t.Fatalf("粤语音律分越界: %v", got.Score)
	}
	if got.Detail == "" {
		t.Error("粤语音律依据文案不应为空")
	}
}

// TestYueShengMuSimilarity 粤语声母分组梯度（同组 -5 / 相邻 +2 / 远隔 +5 / 空不判）
func TestYueShengMuSimilarity(t *testing.T) {
	if v, ok, _ := yueShengMuSimilarity("b", "b", false); !ok || v != -5 {
		t.Errorf("同组声母应 -5, got v=%v ok=%v", v, ok)
	}
	if v, ok, _ := yueShengMuSimilarity("b", "f", false); !ok || v != 2 {
		t.Errorf("相邻组声母应 +2, got v=%v ok=%v", v, ok)
	}
	if v, ok, _ := yueShengMuSimilarity("b", "j", false); !ok || v != 5 {
		t.Errorf("远隔组声母应 +5, got v=%v ok=%v", v, ok)
	}
	if _, ok, _ := yueShengMuSimilarity("", "n", false); ok {
		t.Error("空声母不应参与比较")
	}
}

// TestYueYunMuSimilarity 粤语韵母同异（异 +4 / 同 -3）
func TestYueYunMuSimilarity(t *testing.T) {
	if v, ok, _ := yueYunMuSimilarity("nei5", "jyu5", false); !ok || v != 4 {
		t.Errorf("异韵母应 +4, got v=%v ok=%v", v, ok)
	}
	if v, ok, _ := yueYunMuSimilarity("nei5", "dei6", false); !ok || v != -3 {
		t.Errorf("同韵母(叠韵)应 -3, got v=%v ok=%v", v, ok)
	}
}

// TestYueHomophoneRater_ZhuPenalized 朱→猪 命中，粤语谐音分被扣减
func TestYueHomophoneRater_ZhuPenalized(t *testing.T) {
	r := NewYueHomophoneRater()
	cand := &NameCandidate{Char1: "朱", Jyutping1: "zyu1", SurnameJyutping: "lei5"}
	got := r.Rate(cand, yueFateData(t))
	if got.Score >= 100 {
		t.Errorf("朱应被扣减, got %v", got.Score)
	}
	if got.Detail == "" || got.Detail == "粤语读音无不良谐音" {
		t.Errorf("朱应给出不吉谐音说明, got %q", got.Detail)
	}
}

// TestYueHomophoneRater_ZhuBenign 珠(zyu1) 与朱同音但为好字，不应被扣减（回归防线）
func TestYueHomophoneRater_ZhuBenign(t *testing.T) {
	r := NewYueHomophoneRater()
	cand := &NameCandidate{Char1: "珠", Jyutping1: "zyu1", SurnameJyutping: "lei5"}
	got := r.Rate(cand, yueFateData(t))
	if got.Score != 100 {
		t.Errorf("珠不应被扣减, got %v (detail=%q)", got.Score, got.Detail)
	}
}

// TestYueHomophoneRater_MissingData 粤拼缺失 → 中性分 75
func TestYueHomophoneRater_MissingData(t *testing.T) {
	r := NewYueHomophoneRater()
	got := r.Rate(&NameCandidate{Char1: "某"}, yueFateData(t))
	if got.Score != 75 {
		t.Errorf("粤拼缺失应为中性分 75, got %v", got.Score)
	}
}

// TestRateCantoneseScoreMatchesRateCantonese 粤语总分两条路径一致（与普通话契约同构）
func TestRateCantoneseScoreMatchesRateCantonese(t *testing.T) {
	fateData := yueFateData(t)
	raters := DefaultCantoneseRaters()
	cand := &NameCandidate{
		Char1: "泽", Char2: "宇",
		Pinyin1: "zé", Pinyin2: "yǔ",
		Jyutping1: "zaak6", Jyutping2: "jyu5", SurnameJyutping: "lei5",
	}
	full := RateCantonese(cand, fateData, raters)
	fast := RateCantoneseScore(cand, fateData, raters)
	if full.Total != fast {
		t.Errorf("RateCantonese(%v) != RateCantoneseScore(%v)", full.Total, fast)
	}
	if len(full.Items) != 2 || len(full.Details) != 2 {
		t.Errorf("粤语分明细应含两项维度: items=%v details=%v", full.Items, full.Details)
	}
}

// TestBlendScore 粤语分并入综合分的混合公式（开=加权并入，关=零影响）
func TestBlendScore(t *testing.T) {
	fateData := yueFateData(t)
	cand := &NameCandidate{
		Char1: "泽", Char2: "宇",
		Pinyin1: "zé", Pinyin2: "yǔ",
		WuXing1: "水", WuXing2: "土", Stroke1: 8, Stroke2: 6,
		SurnamePinyin: "li",
		Jyutping1:     "zaak6", Jyutping2: "jyu5", SurnameJyutping: "lei5",
	}
	main := RateNameScore(cand, fateData, DefaultRaters())
	canto := RateCantoneseScore(cand, fateData, DefaultCantoneseRaters())
	wantBlended := math.Round((main*(1-CantoneseBlendWeight)+canto*CantoneseBlendWeight)*10) / 10

	// 开启：混合
	on := &sessionImpl{raters: DefaultRaters(), enableCantonese: true, cantoneseRaters: DefaultCantoneseRaters()}
	final, gotCanto := on.blendScore(cand, fateData)
	if final != wantBlended {
		t.Errorf("开启时混合分=%v, want %v (main=%v canto=%v)", final, wantBlended, main, canto)
	}
	if gotCanto != canto {
		t.Errorf("粤语分透出=%v, want %v", gotCanto, canto)
	}

	// 关闭：零影响，粤语分透出 0
	off := &sessionImpl{raters: DefaultRaters()}
	finalOff, cantoOff := off.blendScore(cand, fateData)
	if finalOff != main {
		t.Errorf("关闭时应等于主分 %v, got %v", main, finalOff)
	}
	if cantoOff != 0 {
		t.Errorf("关闭时粤语分应透出 0, got %v", cantoOff)
	}
}

// TestCantoneseScoreEndToEnd 端到端：开关控制粤语分是否透出到 NameResult
func TestCantoneseScoreEndToEnd(t *testing.T) {
	provider := &stubProvider{chars: []*Character{
		{Char: "泽", Pinyin: []string{"zé"}, WuXing: "水", SimplifiedStroke: 8, IsRegular: true, IsNameable: true, CommonLevel: 1},
		{Char: "宇", Pinyin: []string{"yǔ"}, WuXing: "土", SimplifiedStroke: 6, IsRegular: true, IsNameable: true, CommonLevel: 1},
		{Char: "明", Pinyin: []string{"míng"}, WuXing: "火", SimplifiedStroke: 8, IsRegular: true, IsNameable: true, CommonLevel: 1},
		{Char: "轩", Pinyin: []string{"xuān"}, WuXing: "土", SimplifiedStroke: 7, IsRegular: true, IsNameable: true, CommonLevel: 1},
	}}
	engine := NewEngine(provider, &stubAnalyzer{}, DefaultRaters())

	run := func(enable bool) *Output {
		session := engine.NewSession()
		if err := session.Start(context.Background(), &Input{
			Surname: "李", Gender: GenderMale, Born: time.Date(2023, 8, 20, 10, 0, 0, 0, time.UTC),
			Options: GenerateOptions{NameLength: 2, Count: 5, EnableCantonese: enable},
		}); err != nil {
			t.Fatalf("会话启动失败: %v", err)
		}
		if err := session.Wait(); err != nil {
			t.Fatalf("生成失败: %v", err)
		}
		out := session.Result()
		if out == nil || len(out.TopNames) == 0 {
			t.Fatal("应有候选结果")
		}
		return out
	}

	// 开启：每条结果都带粤语分，含两项维度，总分在 [0,100]
	on := run(true)
	for _, nr := range on.TopNames {
		if nr.CantoneseScore == nil {
			t.Fatalf("开启粤语评分时 %s 应带粤语分", nr.FullName)
		}
		cs := nr.CantoneseScore
		if cs.Total < 0 || cs.Total > 100 {
			t.Errorf("粤语分越界: %v", cs.Total)
		}
		if cs.Items["粤语音律"] == 0 || cs.Items["粤语谐音"] == 0 {
			t.Errorf("粤语分明细维度缺失: %v", cs.Items)
		}
	}

	// 关闭：粤语分为 nil（JSON 省略），综合分不受粤语影响
	off := run(false)
	for _, nr := range off.TopNames {
		if nr.CantoneseScore != nil {
			t.Errorf("关闭粤语评分时 %s 不应带粤语分", nr.FullName)
		}
	}
}
