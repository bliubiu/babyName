package fate

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestGenerateRejectsUnsupportedNameLength 3/4 等长度此前被静默当双名处理，
// 必须显式报错，避免响应与请求参数不符。
func TestGenerateRejectsUnsupportedNameLength(t *testing.T) {
	engine := NewEngine(newCuratedPoolProvider(0), &stubAnalyzer{}, DefaultRaters())

	for _, length := range []int{3, 4} {
		session := engine.NewSession()
		err := session.Start(context.Background(), &Input{
			Surname: "王",
			Gender:  GenderMale,
			Born:    time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
			Options: GenerateOptions{NameLength: length, Count: 3},
		})
		if err == nil {
			err = session.Wait()
		}
		if err == nil {
			t.Fatalf("NameLength=%d 应报错，实际成功", length)
		}
		if !strings.Contains(err.Error(), "名字长度只支持") {
			t.Errorf("NameLength=%d 错误信息应说明支持范围，实际: %v", length, err)
		}
	}
}
