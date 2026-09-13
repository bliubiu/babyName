package fate

// engine_source_rater_test.go — session 级「经典来源偏好」Rater 装配单测
//
// 背景：engine.raters 在构造期固定（一次性 DefaultRaters），而经典来源（SourceClassic）
// 是 per-request 参数。session.Start 阶段按 input.Options.SourceClassic 把
// 文化印象（WenHuaRater）替换为带来源字集的版本，实现 per-request 来源评分加持。

import (
	"context"
	"testing"
	"time"
)

// TestSessionSourceClassicRaterSwap 验证：
//   - 指定 SourceClassic（词典/拼音均可）时，session.raters 中的文化印象 Rater
//     被替换为带来源字集的 WenHuaRater
//   - 未指定时保持默认（sourceSet 为空，无来源加分）
func TestSessionSourceClassicRaterSwap(t *testing.T) {
	engine := newPinyinTestEngine()

	run := func(source string) *sessionImpl {
		session := engine.NewSession()
		if err := session.Start(context.Background(), &Input{
			Surname: "李",
			Gender:  GenderMale,
			Born:    time.Date(2023, 8, 20, 10, 0, 0, 0, time.UTC),
			Options: GenerateOptions{NameLength: 1, Count: 10, SourceClassic: source},
		}); err != nil {
			t.Fatalf("会话启动失败: %v", err)
		}
		impl := session.(*sessionImpl)
		impl.cancel() // 只需验证 Start 后的 raters 装配，立即取消生成
		<-impl.done
		return impl
	}

	// 场景A：中文来源名 → 文化印象 Rater 带论语字集
	withLunyu := run("论语")
	wrLunyu := findWenHuaRater(t, withLunyu.raters)
	if len(wrLunyu.sourceSet) == 0 {
		t.Error("SourceClassic=论语 时文化印象 Rater 应带来源字集（sourceSet 非空）")
	}

	// 场景B：拼音来源别名 → 同样生效（sourceAlias 归一化）
	withAlias := run("lunyu")
	wrAlias := findWenHuaRater(t, withAlias.raters)
	if len(wrAlias.sourceSet) == 0 {
		t.Error("SourceClassic=lunyu（拼音别名）时应同样构建来源字集")
	}

	// 场景C：未指定 → 保持默认无来源加分
	without := run("")
	wrPlain := findWenHuaRater(t, without.raters)
	if len(wrPlain.sourceSet) != 0 {
		t.Errorf("未指定 SourceClassic 时不应构建来源字集，实际 size=%d", len(wrPlain.sourceSet))
	}
}

// findWenHuaRater 从 rater 列表中定位文化印象 Rater（不存在则 t.Fatal）
func findWenHuaRater(t *testing.T, raters []Rater) *WenHuaRater {
	t.Helper()
	for _, r := range raters {
		if wr, ok := r.(*WenHuaRater); ok {
			return wr
		}
	}
	t.Fatal("raters 中不存在 WenHuaRater（文化印象维度）")
	return nil
}
