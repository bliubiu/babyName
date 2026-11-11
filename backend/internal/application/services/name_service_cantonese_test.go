package services

// name_service_cantonese_test.go — /names/generate 路径「粤语评分开关」回归
//
// 背景（docs/43 P1-2）：GenerateRequest 已暴露 enable_cantonese 字段，
// 但 NameService.generateNamesViaFate 装配 fate.GenerateOptions 时漏传
// EnableCantonese，导致生成链路粤语开关恒不生效；convertFateToNameNames
// 也漏映射 Jyutping / CantoneseScore / CantoneseScoreDetail。
// 测名路径（fate_name_service.go:144 与 :221-227）是同口径参考实现，
// TestFateNameService_E2E_Cantonese 是其既有守护；本文件为生成路径补齐等价守护。
//
// 本文件断言：
//   - 开启：/names/generate 每个名字带 CantoneseScore（>0）、cantonese_score_detail、jyutping
//   - 关闭：CantoneseScore 为 0、明细为空；jyutping 仍始终透传（与测名路径同口径）

import (
	"context"
	"testing"
)

// TestGenerate_CantoneseSwitch 真实链路：开启开关后粤语分必须生效
func TestGenerate_CantoneseSwitch(t *testing.T) {
	fs := setupFateNameServiceE2E(t)
	svc := NewNameService(WithFateService(fs))

	base := GenerateRequest{
		Surname:    "张",
		Gender:     "male",
		BirthYear:  2024,
		BirthMonth: 5,
		BirthDay:   20,
		BirthHour:  10,
		NameLength: 2,
	}

	// ——— 开启粤语评分 ———
	on := base
	on.EnableCantonese = true
	namesOn, _, _, err := svc.generateNamesViaFate(context.Background(), &on, nil, nil)
	if err != nil {
		t.Fatalf("生成失败（粤语开）: %v", err)
	}
	if len(namesOn) == 0 {
		t.Fatal("未生成任何名字")
	}
	for _, n := range namesOn {
		if n.Jyutping == "" {
			t.Errorf("%s 粤语开启但 Jyutping 为空（粤拼标注未透传）", n.FullName)
		}
		if n.CantoneseScore == 0 {
			t.Errorf("%s 粤语开启但 CantoneseScore 为 0（开关未装配进引擎）", n.FullName)
		}
		if len(n.CantoneseScoreDetail) == 0 {
			t.Errorf("%s 粤语开启但 cantonese_score_detail 为空（映射缺失）", n.FullName)
		}
	}

	// ——— 关闭粤语评分（对照：零影响，粤拼仍透传）———
	namesOff, _, _, err := svc.generateNamesViaFate(context.Background(), &base, nil, nil)
	if err != nil {
		t.Fatalf("生成失败（粤语关）: %v", err)
	}
	if len(namesOff) == 0 {
		t.Fatal("未生成任何名字")
	}
	for _, n := range namesOff {
		if n.CantoneseScore != 0 {
			t.Errorf("%s 粤语关闭但 CantoneseScore = %v", n.FullName, n.CantoneseScore)
		}
		if len(n.CantoneseScoreDetail) != 0 {
			t.Errorf("%s 粤语关闭但 cantonese_score_detail 非空", n.FullName)
		}
		if n.Jyutping == "" {
			t.Errorf("%s 粤语关闭但 Jyutping 也为空（粤拼标注应始终透传）", n.FullName)
		}
	}
}
