package middleware

// sanitize.go — 访问日志脱敏原语（docs/43 P1-10）
//
// 访问日志/Recovery 日志此前原样记录完整 RawQuery、c.Errors 与 panic 值：
// 查询串可能携带敏感参数值，错误/panic 串可能超长（日志爆炸）。
// 本文件提供统一的脱敏与截断原语，日志字段一律经此处理。

import (
	"net/url"
	"strings"
)

// sensitiveQueryKeys 需要打码值的查询参数键（小写比较）
var sensitiveQueryKeys = map[string]bool{
	"password": true,
	"passwd":   true,
	"token":    true,
	"secret":   true,
	"key":      true,
	"phone":    true,
	"mobile":   true,
	"idcard":   true,
}

const (
	// maxLogFieldRunes 日志字段最大 rune 数（查询串/错误串统一上限）
	maxLogFieldRunes = 256
	// truncateMark 截断标记（额外占用，不计入 maxLogFieldRunes 语义）
	truncateMark = "…(截断)"
)

// sanitizeQuery 脱敏查询串：敏感键值打码为 ***，整体按 rune 截断。
// 畸形查询串不做解析，仅截断（不得因日志路径引入 panic）。
func sanitizeQuery(raw string) string {
	if raw == "" {
		return ""
	}
	if vals, err := url.ParseQuery(raw); err == nil {
		for k := range vals {
			if sensitiveQueryKeys[strings.ToLower(k)] {
				for i := range vals[k] {
					vals[k][i] = "***"
				}
			}
		}
		return truncateString(vals.Encode(), maxLogFieldRunes)
	}
	return truncateString(raw, maxLogFieldRunes)
}

// truncateString 按 rune 截断到 n 个字符（不断 UTF-8），超长附截断标记。
func truncateString(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	if n <= 0 {
		return ""
	}
	return string(runes[:n]) + truncateMark
}
