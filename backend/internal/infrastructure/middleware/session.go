package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"name/internal/application/services"
	"name/internal/infrastructure/logger"
)

const (
	// SessionCookieName 访客会话标识 cookie。与 SSR 生成结果会话同名，
	// 但两者用途不同：cookie 值用于收藏/历史归属，进程内 map 只缓存生成结果。
	SessionCookieName = "nm_sid"
	// SessionIDHeader API 客户端显式携带会话标识的请求头。
	SessionIDHeader = "X-Session-ID"

	sessionIDContextKey = "nm_session_id"
	// sessionCookieMaxAge cookie 有效期 30 天：收藏/历史是持久数据，
	// 不能沿用生成结果的 1 小时会话 TTL（否则访客很快看不到自己的数据）。
	sessionCookieMaxAge = 30 * 24 * 60 * 60
)

// EnsureSessionID 解析当前请求的会话标识，缺失或非法时生成新标识并下发 cookie。
//
// 解析顺序：X-Session-ID 头（API 客户端）→ nm_sid cookie（浏览器）→ 新建。
// 非法值一律视为「无会话」并重新生成，绝不回落为空串——空串会命中迁移前的
// 无归属数据，导致隔离失效。
func EnsureSessionID(c *gin.Context) (string, error) {
	if sid := GetSessionID(c); sid != "" {
		return sid, nil
	}

	raw := strings.TrimSpace(c.GetHeader(SessionIDHeader))
	if raw == "" {
		raw, _ = c.Cookie(SessionCookieName)
	}
	if sid, err := services.NormalizeSessionID(raw); err == nil {
		c.Set(sessionIDContextKey, sid)
		return sid, nil
	}

	sid, err := services.NewSessionID()
	if err != nil {
		return "", err
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(SessionCookieName, sid, sessionCookieMaxAge, "/", "", false, true)
	c.Set(sessionIDContextKey, sid)
	return sid, nil
}

// GetSessionID 读取本次请求已解析的会话标识；未挂载 SessionID 中间件且尚未
// 调用 EnsureSessionID 时返回空串。
func GetSessionID(c *gin.Context) string {
	v, ok := c.Get(sessionIDContextKey)
	if !ok {
		return ""
	}
	sid, _ := v.(string)
	return sid
}

// SessionID 会话标识中间件：保证每个请求都带归属标识，供收藏/历史落库与查询。
func SessionID() gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, err := EnsureSessionID(c); err != nil {
			logger.Error("初始化会话标识失败", logger.ErrField(err))
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "会话初始化失败，请稍后重试",
			})
			return
		}
		c.Next()
	}
}
