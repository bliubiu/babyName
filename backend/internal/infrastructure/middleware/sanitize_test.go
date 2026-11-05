package middleware

// sanitize_test.go — 访问日志脱敏回归（docs/43 P1-10）
//
// 访问日志此前原样记录完整 RawQuery（可能携带敏感参数值）与无界的
// errors/panic 字符串。本文件守护：
//   - 敏感键（password/token/secret/key/phone/mobile/idcard）的值打码
//   - 查询串与错误串超长截断（防日志爆炸/注入）
//   - 畸形查询串不 panic，仅截断

import (
	"strings"
	"testing"
)

// TestSanitizeQuery_MasksSensitiveKeys 敏感键值必须打码（大小写不敏感）
func TestSanitizeQuery_MasksSensitiveKeys(t *testing.T) {
	cases := []struct{ name, in, wantSub, forbiddenSub string }{
		{"密码", "surname=张&password=abc123", "password=%2A%2A%2A", "abc123"},
		{"令牌", "token=eyJhbGciOi", "token=%2A%2A%2A", "eyJhbGciOi"},
		{"大写键", "Password=x&b=1", "Password=%2A%2A%2A", ""},
		{"手机号", "phone=13800138000&b=2", "phone=%2A%2A%2A", "13800138000"},
	}
	for _, c := range cases {
		got := sanitizeQuery(c.in)
		if !strings.Contains(got, c.wantSub) {
			t.Errorf("%s: sanitizeQuery(%q) = %q，应包含打码后的 %s", c.name, c.in, got, c.wantSub)
		}
		if c.forbiddenSub != "" && strings.Contains(got, c.forbiddenSub) {
			t.Errorf("%s: sanitizeQuery(%q) = %q，泄露了敏感值", c.name, c.in, got)
		}
	}
}

// TestSanitizeQuery_KeepsNormalParams 普通参数原样保留
func TestSanitizeQuery_KeepsNormalParams(t *testing.T) {
	got := sanitizeQuery("surname=%E5%BC%A0&gender=male")
	if !strings.Contains(got, "gender=male") {
		t.Errorf("普通参数丢失: %q", got)
	}
}

// TestSanitizeQuery_TruncatesLong 超长查询串截断（按 rune，不断 UTF-8）
func TestSanitizeQuery_TruncatesLong(t *testing.T) {
	long := "k=" + strings.Repeat("长", 500)
	got := sanitizeQuery(long)
	if got == "" {
		t.Fatal("不应为空")
	}
	if len([]rune(got)) > maxLogFieldRunes+10 { // 截断标记留了余量
		t.Errorf("截断后仍过长: %d rune", len([]rune(got)))
	}
}

// TestSanitizeQuery_MalformedDoesNotPanic 畸形查询串只截断不 panic
func TestSanitizeQuery_MalformedDoesNotPanic(t *testing.T) {
	for _, in := range []string{"%", "&&&", "k=%zz", "\x00\x01"} {
		got := sanitizeQuery(in)
		if len([]rune(got)) > maxLogFieldRunes+10 {
			t.Errorf("畸形串 %q 未截断: %q", in, got)
		}
	}
}

// TestSanitizeQuery_Empty 空串原样返回
func TestSanitizeQuery_Empty(t *testing.T) {
	if got := sanitizeQuery(""); got != "" {
		t.Errorf("空串应返回空，实际 %q", got)
	}
}

// TestTruncateString 截断 helper：限长、含截断标记
func TestTruncateString(t *testing.T) {
	if got := truncateString("abcdef", 3); got == "abcdef" || got == "" {
		t.Errorf("超长未生效: %q", got)
	}
	if got := truncateString("abc", 10); got != "abc" {
		t.Errorf("未超长不应截断: %q", got)
	}
}
