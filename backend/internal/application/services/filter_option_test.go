package services

import "testing"

// TestBaseFilterOption_喜用神不硬收窄 回归：只有用户显式 WuxingMatch 才进
// PreferredWuXing（硬过滤）；喜用神交给引擎按「喜用神+生助」收窄，
// 否则生助五行、典籍加字、用户点选字会被服务层提前剔除。
func TestBaseFilterOption_喜用神不硬收窄(t *testing.T) {
	fo := baseFilterOption(&GenerateRequest{Surname: "李", Gender: "male"})
	if got := fo.Build().PreferredWuXing(); len(got) != 0 {
		t.Fatalf("未指定 WuxingMatch 时不应硬收窄五行，got %v", got)
	}

	req := &GenerateRequest{WuxingMatch: []string{"木", "水"}, MinFrequencyTier: 2}
	fo = baseFilterOption(req)
	f := fo.Build()
	if got := f.PreferredWuXing(); len(got) != 2 || got[0] != "木" || got[1] != "水" {
		t.Fatalf("显式 WuxingMatch 应原样进入硬过滤，got %v", got)
	}
	if fo.MinFrequencyTier != 2 {
		t.Fatalf("频率档位未传递，got %d", fo.MinFrequencyTier)
	}
}
