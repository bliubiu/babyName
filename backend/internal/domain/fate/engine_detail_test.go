package fate

import (
	"context"
	"testing"
	"time"
)

// TestGenerateKeepsScoreDetails 确定性打分数据链路：engine 输出结果必须透传每维评分依据
//
// RateName 已在域层保留 Details，但引擎经 ExcellentTable 流式组装 ExcellentEntry 时
// 若只拷贝 Items 而丢弃 Details，输出 NameResult.Score.Details 仍为空。
// 本测试验证从 provider → session → NameResult 的端到端链路不丢失依据文字。
func TestGenerateKeepsScoreDetails(t *testing.T) {
	provider := newCuratedPoolProvider(0)
	engine := NewEngine(provider, &stubAnalyzer{}, DefaultRaters())

	session := engine.NewSession()
	err := session.Start(context.Background(), &Input{
		Surname: "王",
		Gender:  GenderMale,
		Born:    time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
		Options: GenerateOptions{NameLength: 2, Count: 50},
	})
	if err != nil {
		t.Fatalf("会话启动失败: %v", err)
	}
	if err := session.Wait(); err != nil {
		t.Fatalf("生成失败: %v", err)
	}
	output := session.Result()
	if len(output.TopNames) == 0 {
		t.Fatal("应至少生成一个候选")
	}

	// 每个推荐名都必须携带各维度评分依据文字
	for _, nr := range output.TopNames[:10] {
		if len(nr.Score.Details) == 0 {
			t.Errorf("名字「%s」缺少评分依据文字（Details 在引擎透传中被丢弃）", nr.FullName)
			continue
		}
		for dim, detail := range nr.Score.Details {
			if detail == "" {
				t.Errorf("名字「%s」维度 %q 依据文字为空", nr.FullName, dim)
			}
		}
	}
}