package services

import (
	"context"
	"testing"

	"name/internal/domain/bazi"
)

// evaluate_hexagram_wiring_test.go — docs/42 P2-10 收尾回归：
// 测名路径卦象接线 + 生成路径四象限解读（hexagram_interpretation）接线。

// TestBuildResponse_CarriesHexagramInterp 生成响应必须携带 64 卦差异化解读
func TestBuildResponse_CarriesHexagramInterp(t *testing.T) {
	detail := (&HexagramAdapter{}).FindMeihuaDetail(4, 3)
	s := &NameService{}
	resp := s.buildResponse(&bazi.BaziAnalysis{Xiyongshen: []string{"木"}},
		nil, detail.OriginalHexagram, detail, nil, nil, "鼠")
	if resp.HexagramInterp == nil {
		t.Fatal("GenerateResponse.HexagramInterp 未接线")
	}
	// 丰卦差异化文案锚点（interpret.go 表）
	if resp.HexagramInterp.Career == "" || resp.HexagramInterp.Career != "丰大如日中天，宜明断守成防盛极" {
		t.Errorf("丰卦 career 四象限文案不符：%q", resp.HexagramInterp.Career)
	}
	// 喜用神命中（丰=震木/离火，喜用神木）→ Overall 附相合断语
	if resp.HexagramInterp.Overall == detail.OriginalHexagram.Interpretation {
		t.Error("喜用神命中时 Overall 应附「与命格相合」断语")
	}
}

// TestEvaluate_E2E_HexagramWiring 测名响应卦象接线：
// hexagram / hexagram_interpretation / tiyong / name.hexagram_detail 全链路
func TestEvaluate_E2E_HexagramWiring(t *testing.T) {
	svc := setupNameServiceE2E(t)
	resp, err := svc.Evaluate(context.Background(), evalE2ERequest("王", "丰儿"))
	if err != nil {
		t.Fatalf("Evaluate 失败: %v", err)
	}
	if resp.Hexagram == nil {
		t.Fatal("EvaluateResponse.Hexagram 未接线")
	}
	if resp.HexagramInterp == nil || resp.HexagramInterp.Career == "" {
		t.Fatal("EvaluateResponse.HexagramInterp 未接线或四象限为空")
	}
	if resp.TiYong == nil {
		t.Fatal("EvaluateResponse.TiYong 未接线")
	}
	if resp.Name.HexagramDetail == nil || resp.Name.HexagramDetail.OriginalHexagram == nil {
		t.Fatal("Name.HexagramDetail 未接线（前端 NameDetail 嵌入渲染依赖它）")
	}
	// 顶层经文卦与详情本卦同源（同一梅花口径）
	if resp.Hexagram.Number != resp.Name.HexagramDetail.OriginalHexagram.Number {
		t.Errorf("经文卦 #%d 与详情本卦 #%d 不一致",
			resp.Hexagram.Number, resp.Name.HexagramDetail.OriginalHexagram.Number)
	}
	// 四象限文案应来自差异化表而非旧通用模板
	if resp.HexagramInterp.Career == "事业平稳发展，需把握时机" {
		t.Error("四象限应为差异化文案，仍命中旧通用模板")
	}
}
