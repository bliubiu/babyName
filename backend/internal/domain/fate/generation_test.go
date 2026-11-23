package fate

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestNormalizeGenerationPosition(t *testing.T) {
	cases := map[string]string{
		"tail":   "tail",
		"end":    "tail",
		"last":   "tail",
		" TAIL ": "tail",
		"middle": "middle",
		"":       "middle",
		"foo":    "middle",
	}
	for in, want := range cases {
		if got := NormalizeGenerationPosition(in); got != want {
			t.Errorf("NormalizeGenerationPosition(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestGenerateWithGenerationPosition 回归：字辈位置必须真正影响名字结构。
// middle → 字辈为名的第一个字；tail/end → 字辈为名的最后一个字。
func TestGenerateWithGenerationPosition(t *testing.T) {
	cases := []struct {
		position string
		tail     bool
	}{
		{"middle", false},
		{"tail", true},
		{"end", true}, // 前端历史枚举，须兼容
	}
	for _, c := range cases {
		t.Run(c.position, func(t *testing.T) {
			engine := NewEngine(newCuratedPoolProvider(0), &stubAnalyzer{}, DefaultRaters())
			session := engine.NewSession()
			input := &Input{
				Surname:            "王",
				Gender:             GenderMale,
				Born:               time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
				Generation:         "泽",
				GenerationPosition: c.position,
				Options:            GenerateOptions{NameLength: 2, Count: 20},
			}
			if err := session.Start(context.Background(), input); err != nil {
				t.Fatalf("会话启动失败: %v", err)
			}
			if err := session.Wait(); err != nil {
				t.Fatalf("生成失败: %v", err)
			}
			output := session.Result()
			if output == nil || len(output.TopNames) == 0 {
				t.Fatal("未生成任何候选")
			}
			for _, nr := range output.TopNames {
				if c.tail && !strings.HasSuffix(nr.GivenName, "泽") {
					t.Errorf("position=%s 名字 %q 末字不是字辈「泽」", c.position, nr.GivenName)
				}
				if !c.tail && !strings.HasPrefix(nr.GivenName, "泽") {
					t.Errorf("position=%s 名字 %q 首字不是字辈「泽」", c.position, nr.GivenName)
				}
			}
		})
	}
}
