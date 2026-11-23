package services

import (
	"testing"

	"name/internal/domain/bazi"
	"name/internal/domain/name"
)

// docs/42 P3-11 回归：每个候选名都必须有独立梅花起卦与体用断卦，
// 而非仅 names[0]。卦象不计入评分（P3-12 口径），只作展示。

func TestCalculateHexagramAndZiweiParallel_PerNameDetail(t *testing.T) {
	s := &NameService{hexagramFinder: &HexagramAdapter{}, ziweiAnalyzer: &ZiweiAdapter{}}
	// 姓「王」康熙 4 画：名笔画 3 → 丰（4,3），名笔画 4 → 震（4,4）
	names := []name.Name{
		{GivenName: "丰儿", Strokes: 7},
		{GivenName: "震儿", Strokes: 8},
	}
	req := &GenerateRequest{Surname: "王", BirthYear: 2000, BirthMonth: 8, BirthDay: 16, BirthHour: 10, Gender: "female"}
	baziAnalysis := &bazi.BaziAnalysis{Season: "秋", Xiyongshen: []string{"金"}}

	_, _, topDetail, ziweiChart, _ := s.calculateHexagramAndZiweiParallel(names, baziAnalysis, req)

	if ziweiChart == nil {
		t.Fatal("紫微排盘不应为空")
	}
	if topDetail == nil || topDetail.OriginalHexagram == nil {
		t.Fatal("顶层 HexagramDetail 不应为空")
	}
	for i, want := range []string{"丰", "震"} {
		n := &names[i]
		if n.HexagramDetail == nil || n.HexagramDetail.OriginalHexagram == nil {
			t.Fatalf("候选名 %d 缺少独立卦象详情", i)
		}
		if got := n.HexagramDetail.OriginalHexagram.Name; got != want {
			t.Errorf("候选名 %d 应得本卦 %s（姓4 名%d），实得 %s", i, want, n.Strokes-4, got)
		}
		if n.TiYong == nil {
			t.Errorf("候选名 %d 缺少体用断卦", i)
		}
	}
	if names[0].HexagramDetail.OriginalHexagram.Number != topDetail.OriginalHexagram.Number {
		t.Errorf("names[0] 独立卦与顶层卦不一致：%d vs %d",
			names[0].HexagramDetail.OriginalHexagram.Number, topDetail.OriginalHexagram.Number)
	}
	// 两名笔画不同 → 卦不同（确非同一卦复制给所有人）
	if names[0].HexagramDetail.OriginalHexagram.Number == names[1].HexagramDetail.OriginalHexagram.Number {
		t.Error("不同笔画的候选名不应得到同一卦")
	}
}

func TestFateNameService_PerNameHexagram(t *testing.T) {
	s := &FateNameService{hexagramFinder: &HexagramAdapter{}}
	names := []*name.NameAnalysis{
		{GivenName: "丰儿", Strokes: 7},
		{GivenName: "震儿", Strokes: 8},
	}
	req := &GenerateRequest{Surname: "王"}
	s.attachFatePerNameHexagrams(names, req, []string{"金"})

	for i, want := range []string{"丰", "震"} {
		n := names[i]
		if n.HexagramDetail == nil || n.HexagramDetail.OriginalHexagram == nil {
			t.Fatalf("fate 路径候选名 %d 缺少独立卦象详情", i)
		}
		if got := n.HexagramDetail.OriginalHexagram.Name; got != want {
			t.Errorf("fate 路径候选名 %d 应得本卦 %s，实得 %s", i, want, got)
		}
		if n.TiYong == nil {
			t.Errorf("fate 路径候选名 %d 缺少体用断卦", i)
		}
	}
}

func TestPerNameHexagram_NilFinder(t *testing.T) {
	d, ti := perNameHexagram(nil, 4, 7, "秋", nil)
	if d != nil || ti != nil {
		t.Error("finder 为 nil 时应返回空")
	}
}

// TestZiweiUsesTrueSolarTime 紫微排盘必须与八字同口径（真太阳时），
// 否则同一出生信息会得到「八字按校正时辰、紫微按钟表时辰」两套结论。
func TestZiweiUsesTrueSolarTime(t *testing.T) {
	s := &NameService{hexagramFinder: &HexagramAdapter{}, ziweiAnalyzer: &ZiweiAdapter{}}
	// 乌鲁木齐（东经 87.6°）00:30 出生，真太阳时约前一日 22:20
	req := &GenerateRequest{
		Surname: "王", BirthYear: 2000, BirthMonth: 8, BirthDay: 16,
		BirthHour: 0, BirthMinute: 30, BirthLongitude: 87.6, Gender: "female",
	}
	names := []name.Name{{GivenName: "丰儿", Strokes: 7}}
	baziAnalysis := &bazi.BaziAnalysis{Season: "秋"}
	_, _, _, chart, _ := s.calculateHexagramAndZiweiParallel(names, baziAnalysis, req)
	if chart == nil {
		t.Fatal("紫微排盘不应为空")
	}
	if chart.Day != 15 || chart.Hour != 22 {
		t.Errorf("紫微应使用真太阳时(8/15 22 时)，实得 %d/%d %d 时",
			chart.Month, chart.Day, chart.Hour)
	}
}

// TestFateZiweiUsesTrueSolarTime FateService 的紫微口径同样须校正。
func TestFateZiweiUsesTrueSolarTime(t *testing.T) {
	s := &FateNameService{ziweiAnalyzer: &ZiweiAdapter{}}
	req := &GenerateRequest{
		BirthYear: 2000, BirthMonth: 8, BirthDay: 16,
		BirthHour: 0, BirthMinute: 30, BirthLongitude: 87.6, Gender: "female",
	}
	chart := s.ziweiForRequest(req)
	if chart == nil {
		t.Fatal("紫微排盘不应为空")
	}
	if chart.Day != 15 || chart.Hour != 22 {
		t.Errorf("紫微应使用真太阳时(8/15 22 时)，实得 %d/%d %d 时",
			chart.Month, chart.Day, chart.Hour)
	}
}
