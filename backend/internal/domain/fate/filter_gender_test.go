package fate

import "testing"

// TestWithGenderFilter_性别枚举归一化 回归：SSR 表单的 "男"/"女" 必须归一为
// canonical male/female——历史缺陷是原样落地后与 GenderHint（male/female/
// neutral）永不相等，性别筛选静默失效，候选池退化为只剩中性字。
func TestWithGenderFilter_性别枚举归一化(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"男", "male"}, {"male", "male"}, {"M", "male"},
		{"女", "female"}, {"female", "female"}, {"F", "female"},
		{"", ""},       // 空串=不过滤，保持现有语义
		{"bad", "bad"}, // 未知输入原样返回，由下游校验兜底
	}
	for _, tt := range tests {
		fo := NewFilterOption().WithGenderFilter(tt.in)
		if fo.GenderFilter != tt.want {
			t.Errorf("WithGenderFilter(%q).GenderFilter = %q，期望 %q", tt.in, fo.GenderFilter, tt.want)
		}
	}
}
