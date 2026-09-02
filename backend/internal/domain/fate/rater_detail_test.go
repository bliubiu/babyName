package fate

import (
	"strings"
	"testing"
)

// TestRateNameKeepsDetails 确定性打分：RateName 聚合后必须保留每维评分依据文字
//
// 各 Rater 的 Rate 返回 NameRating{Score, Detail}，Detail 是该维度的打分依据
// （如「喜用神火，两字皆火」）。此前聚合层只写 items[r.Name()] = rating.Score，
// Detail 被丢弃，导致前端只能展示分数条、无法展示「为什么打这个分」。
// 本测试验证 Details 透传为确定性评分 UI 的数据基础。
func TestRateNameKeepsDetails(t *testing.T) {
	SetCuratedNames([]string{"晴朗"})
	defer SetCuratedNames(nil)
	raters := DefaultRaters()

	cand := &NameCandidate{
		Char1:         "晴",
		Char2:         "朗",
		Meaning1:      "晴空",
		Meaning2:      "明朗",
		WuXing1:       "火",
		WuXing2:       "火",
		Pinyin1:       "qing2",
		Pinyin2:       "lang3",
		Stroke1:       12,
		Stroke2:       10,
		Radical1:      "日",
		Radical2:      "月",
		IsRegular:     true,
		CommonLevel1:  1,
		CommonLevel2:  1,
		SurnamePinyin: "wang2",
	}
	sx := &FateData{
		WuXingXiji: WuXingXiji{Xi: "火", Ji: "水"},
		BaziInfo:   BaziInfo{Zodiac: "马"},
	}

	sc := RateName(cand, sx, raters)

	// 每个参与评分的维度都必须有依据文字
	for _, r := range raters {
		dim := r.Name()
		if _, ok := sc.Details[dim]; !ok {
			t.Errorf("维度 %q 缺少评分依据文字（Details 未透传）", dim)
			continue
		}
		if strings.TrimSpace(sc.Details[dim]) == "" {
			t.Errorf("维度 %q 的依据文字为空", dim)
		}
	}

	// 命理维度依据应具体（非占位符）
	if !strings.Contains(sc.Details["五行八字"], "火") && !strings.Contains(sc.Details["五行八字"], "喜") {
		t.Errorf("五行八字依据应包含喜用神说明，实际：%q", sc.Details["五行八字"])
	}
}