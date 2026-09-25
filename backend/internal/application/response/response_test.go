package response

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestGetHTTPStatus_NeverDowngradesToSuccess 盯住 docs/28 W5 的回归点。
//
// 历史缺陷：getHTTPStatus 只列举 400/401/403/404/422/500/503，其余一律返回 200。
// 客户端普遍以 `res.ok`（即 status < 400）判断成败，因此任何未列举的错误码
// 都会被静默吞掉——用户看到「没报错但也没数据」的页面，前端也不会重试。
//
// 实测（修复前）：409 / 429 / 999 / 0 全部返回 HTTP 200。
//
// 本用例的核心断言：**任何非 2xx/3xx 的输入都不得映射为成功码**。
func TestGetHTTPStatus_NeverDowngradesToSuccess(t *testing.T) {
	cases := []struct {
		name string
		code int
		want int
	}{
		// 原白名单：必须保持原样（不能因为改造而改变既有契约）
		{"400 参数错误", 400, http.StatusBadRequest},
		{"401 未授权", 401, http.StatusUnauthorized},
		{"403 禁止访问", 403, http.StatusForbidden},
		{"404 不存在", 404, http.StatusNotFound},
		{"422 验证失败", 422, http.StatusUnprocessableEntity},
		{"500 内部错误", 500, http.StatusInternalServerError},
		{"503 服务不可用", 503, http.StatusServiceUnavailable},

		// ★ 回归点：未列举但语义合法的错误码，必须透传而非降级为 200
		{"409 冲突", 409, http.StatusConflict},
		{"429 限流", 429, http.StatusTooManyRequests},
		{"451 法律原因不可用", 451, http.StatusUnavailableForLegalReasons},
		{"599 区间上界", 599, 599},

		// ★ 回归点：越界/无意义的编码错误，必须归为 5xx（可被监控告警捕获），
		// 绝不降级 200。归 500 而非 400：写错状态码是服务端缺陷，不是客户端问题。
		{"0 零值", 0, http.StatusInternalServerError},
		{"负数", -1, http.StatusInternalServerError},
		{"201 成功码误用", 201, http.StatusInternalServerError},
		{"301 重定向误用", 301, http.StatusInternalServerError},
		{"999 越界", 999, http.StatusInternalServerError},
		{"399 边界", 399, http.StatusInternalServerError},
		{"600 越界", 600, http.StatusInternalServerError},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := getHTTPStatus(tc.code); got != tc.want {
				t.Errorf("getHTTPStatus(%d) = %d，期望 %d", tc.code, got, tc.want)
			}
		})
	}
}

// TestErrorJSON_UnlistedCodeIsNotSuccess 端到端断言：非法/未列举错误码经
// ErrorJSON 输出后，HTTP 状态必须是失败态，且响应体仍保留原始业务码便于排查。
func TestErrorJSON_UnlistedCodeIsNotSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	for _, code := range []int{409, 429, 999, 0, -1, 201} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		ErrorJSON(c, code, "probe")

		if w.Code < 400 || w.Code > 599 {
			t.Errorf("business code=%d 映射为 HTTP %d，落入了成功/重定向区间"+
				"——客户端 res.ok 为真会吞掉该错误", code, w.Code)
		}
		// 响应体里的业务码必须原样保留，不能因为 HTTP 状态兜底而被改写，
		// 否则排障时无法区分「服务端写错码」与「真的 500」。
		if body := w.Body.String(); !containsCode(body, code) {
			t.Errorf("business code=%d 的响应体未保留原始业务码: %s", code, body)
		}
	}
}

// containsCode 粗略检查响应体中出现 `"code":<n>`（容忍空格）。
func containsCode(body string, code int) bool {
	target := []byte(`"code":`)
	b := []byte(body)
	for i := 0; i+len(target) <= len(b); i++ {
		if string(b[i:i+len(target)]) != string(target) {
			continue
		}
		j := i + len(target)
		for j < len(b) && (b[j] == ' ') {
			j++
		}
		// 解析出数字
		neg := false
		if j < len(b) && b[j] == '-' {
			neg = true
			j++
		}
		n, started := 0, false
		for j < len(b) && b[j] >= '0' && b[j] <= '9' {
			n = n*10 + int(b[j]-'0')
			j++
			started = true
		}
		if !started {
			continue
		}
		if neg {
			n = -n
		}
		if n == code {
			return true
		}
	}
	return false
}

// TestSuccessJSON_Unaffected 确认改造未波及成功路径。
func TestSuccessJSON_Unaffected(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	SuccessJSON(c, gin.H{"k": "v"})

	if w.Code != http.StatusOK {
		t.Errorf("SuccessJSON 的 HTTP 状态 = %d，期望 200", w.Code)
	}
	if !containsCode(w.Body.String(), 200) {
		t.Errorf("SuccessJSON 响应体缺少 code=200: %s", w.Body.String())
	}
}
