package services

// name_service_risks_test.go — 生成结果页风险体检接线守护（docs/43 P2-4）
//
// 背景：分级风险体检（AssessNameRisks）此前只有测名链路调用，
// 生成结果页既不执行也不展示——同一个名字在「结果页」与「测名页」
// 得到的安全结论不对称。接线后生成响应的每个名字携带 risks，
// 与测名页同一真源（同 provider、同判据）。

import (
	"context"
	"testing"

	"name/internal/domain/fate"
)

// TestGenerate_EmitsRisks 生成响应每个名字的风险体检与直调对拍一致
func TestGenerate_EmitsRisks(t *testing.T) {
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
	names, _, _, err := svc.generateNamesViaFate(context.Background(), req, nil, nil)
	if err != nil {
		t.Fatalf("生成失败: %v", err)
	}
	if len(names) == 0 {
		t.Fatal("未生成任何名字")
	}

	provider := NewHanziDataProvider()
	for _, n := range names {
		// 与测名链路同源对拍：同 provider、同判据
		want := fate.AssessNameRisks(provider, n.Surname, n.GivenName)
		if len(n.Risks) != len(want) {
			t.Errorf("%s risks 条数 = %d，直调 = %d（体检未接线或口径漂移）",
				n.FullName, len(n.Risks), len(want))
			continue
		}
		for i := range want {
			if n.Risks[i].Category != want[i].Category ||
				n.Risks[i].Level != want[i].Level ||
				n.Risks[i].Detail != want[i].Detail {
				t.Errorf("%s risks[%d] = %+v，直调 = %+v", n.FullName, i, n.Risks[i], want[i])
			}
		}
	}
}
