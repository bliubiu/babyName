package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code    int         `json:"code"`
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func Success(data interface{}) Response {
	return Response{
		Code:    200,
		Success: true,
		Data:    data,
	}
}

func Error(code int, message string) Response {
	return Response{
		Code:    code,
		Success: false,
		Message: message,
	}
}

func SuccessJSON(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Success(data))
}

func ErrorJSON(c *gin.Context, code int, message string) {
	c.JSON(getHTTPStatus(code), Error(code, message))
}

// getHTTPStatus 把业务错误码映射为 HTTP 状态码。
//
// ★ 历史缺陷（docs/28 W5）：原实现只列举 400/401/403/404/422/500/503 七个码，
// 其余一律 `return 200`。这意味着任何一个不在名单里的错误码——例如 409 冲突、
// 429 限流、或将来新增的业务码——都会被当成 **HTTP 成功**返回。
// 客户端普遍以 `response.ok` / `res.status < 400` 判断成败，于是这类错误会被
// **静默吞掉**：用户看到的是一个「没有报错但也没有数据」的页面，前端也不会重试。
//
// 实测（修复前）：code=409/429/999/0 全部返回 HTTP 200。
//
// 现改为「按错误码区间兜底」：
//   - 合法的 4xx/5xx 直接透传（含原名单，也涵盖 409/429/451 等未列举码）；
//   - 其余任何值（0、负数、1xx/2xx/3xx、>599）都是编码错误，
//     按**服务端内部错误 500** 处理，绝不再降级成 200。
//
// 之所以兜底到 500 而非 400：调用方写错状态码属于服务端缺陷，
// 归为 5xx 才能被监控告警捕获；若归为 4xx 会被误认为是客户端问题而长期潜伏。
func getHTTPStatus(code int) int {
	// 合法范围直接透传：400-599 覆盖全部客户端/服务端错误语义。
	if code >= 400 && code <= 599 {
		return code
	}
	// 显式列举历史白名单中不在 400-599 内的项（当前为空，保留以便将来扩展
	// 需要把某个业务码映射到非 4xx/5xx 时在此处集中声明）。
	// 任何未命中者都视为编码错误 → 500。
	return http.StatusInternalServerError
}
