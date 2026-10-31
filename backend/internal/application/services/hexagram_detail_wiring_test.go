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
	resp := s.buildResponse(&bazi.BaziAnalysis{}, nil, detail.OriginalHexagram, nil, detail, nil, "鼠")
	if resp.HexagramDetail != detail {
		t.Fatal("GenerateResponse.HexagramDetail 未接线")
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
