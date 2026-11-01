package services

import (
	"testing"

	"name/internal/domain/bazi"
	"name/internal/domain/name"
	"name/internal/domain/yijing"
)

// docs/42 P0-4 回归：梅花完整起卦结果（本/变/互/动爻）必须接线到 API 响应。
// 此前 HexagramAdapter 只暴露单卦查询，前端永远拿不到变卦与互卦。

func TestHexagramAdapter_FindMeihuaDetail(t *testing.T) {
	var finder yijing.HexagramFinder = &HexagramAdapter{}
	detail := finder.FindMeihuaDetail(4, 3)
	if detail == nil || detail.OriginalHexagram == nil || detail.OriginalHexagram.Name != "丰" {
		t.Fatalf("FindMeihuaDetail(4,3) 应得本卦丰，got %+v", detail)
	}
	if detail.ChangedHexagram == nil || detail.InterHexagram == nil {
		t.Fatal("变卦/互卦不应为空")
	}
}

func TestBuildResponse_CarriesHexagramDetail(t *testing.T) {
	detail := (&HexagramAdapter{}).FindMeihuaDetail(4, 3)
	s := &NameService{}
	resp := s.buildResponse(&bazi.BaziAnalysis{}, nil, detail.OriginalHexagram, detail, nil, nil, "鼠")
	if resp.HexagramDetail != detail {
		t.Fatal("GenerateResponse.HexagramDetail 未接线")
	}
}

// TestBuildResponse_CarriesTiYong 体用断卦结论接线（docs/42 P1-③）：
// GenerateResponse.TiYong 必须携带结构化体用分析；旧字段 HexagramMatch
// 由体用结论兼容映射（Score 吉85/中65/凶40 三档）。
func TestBuildResponse_CarriesTiYong(t *testing.T) {
	detail := (&HexagramAdapter{}).FindMeihuaDetail(8, 1)
	// 上坤土、下乾金、五爻动（上卦动）：用=坤土、体=乾金，土生金 → 用生体吉；
	// 体乾金逢秋旺，金为喜用神
	tiYong := yijing.AnalyzeTiYong(8, 1, 5, "秋", []string{"金"})
	s := &NameService{}
	resp := s.buildResponse(&bazi.BaziAnalysis{}, nil, detail.OriginalHexagram, detail, tiYong, nil, "鼠")
	if resp.TiYong != tiYong {
		t.Fatal("GenerateResponse.TiYong 未接线")
	}
	if resp.HexagramMatch == nil {
		t.Fatal("HexagramMatch 兼容映射不应为空（前端仍读 hexagram_match）")
	}
	if resp.HexagramMatch.Score != 85 {
		t.Errorf("用生体（坤土生乾金）应映射吉档 85 分，实得 %d", resp.HexagramMatch.Score)
	}
	if resp.HexagramMatch.Reason != tiYong.Text {
		t.Errorf("Reason 应为文字化解读，实得 %q", resp.HexagramMatch.Reason)
	}
	if resp.HexagramMatch.WuxingCompat != tiYong.WangShuai {
		t.Errorf("WuxingCompat 应为卦气旺衰 %q，实得 %q", tiYong.WangShuai, resp.HexagramMatch.WuxingCompat)
	}
}

// TestHexagramMatchFromTiYong 体用吉凶 → 三档分数映射锁
func TestHexagramMatchFromTiYong(t *testing.T) {
	if hexagramMatchFromTiYong(nil) != nil {
		t.Error("nil 体用结论应映射 nil")
	}
	cases := []struct{ verdict string; wantScore int }{
		{"吉", 85}, {"中", 65}, {"凶", 40},
	}
	for _, c := range cases {
		ti := &yijing.TiYongAnalysis{Verdict: c.verdict, Text: "测试解读", WangShuai: "旺"}
		m := hexagramMatchFromTiYong(ti)
		if m == nil || m.Score != c.wantScore {
			t.Errorf("verdict=%s 应映射 %d 分，实得 %+v", c.verdict, c.wantScore, m)
		}
	}
}

func TestFateHexagramForNames_ReturnsDetail(t *testing.T) {
	s := &FateNameService{hexagramFinder: &HexagramAdapter{}}
	names := []*name.NameAnalysis{{Strokes: 7}}
	req := &GenerateRequest{Surname: "王"}
	hex, detail := s.hexagramForNames(names, req)
	if hex == nil || detail == nil {
		t.Fatalf("hexagramForNames 应同时返回单卦与详情，got %v / %v", hex, detail)
	}
	if detail.OriginalHexagram == nil || hex.Name != detail.OriginalHexagram.Name {
		t.Fatalf("单卦与详情本卦不一致：%s vs %+v", hex.Name, detail.OriginalHexagram)
	}
}

// TestMeihuaStrokesForNames 姓名卦笔画口径锁（docs/42 P0-1）：
// 康熙姓笔画优先 / 查库失败 rune 兜底 / 名笔画 ≤0 兜底 10。
// 两个服务（name_service 与 fate_name_service）共用此口径，不得再各自实现。
func TestMeihuaStrokesForNames(t *testing.T) {
	s, g := meihuaStrokesForNames("王", 7)
	if s != 4 {
		t.Errorf("「王」姓氏笔画 = %d, want 4（康熙）", s)
	}
	if g != 3 {
		t.Errorf("名笔画 = %d, want 3（全名7-姓4）", g)
	}
	if _, g2 := meihuaStrokesForNames("王", 2); g2 != 10 {
		t.Errorf("名笔画≤0 兜底 = %d, want 10", g2)
	}
	if s3, _ := meihuaStrokesForNames("", 0); s3 != 0 {
		t.Errorf("空姓兜底 = %d, want 0", s3)
	}
}
