package response

import (
	"github.com/gin-gonic/gin"
)

type Response struct {
	Code    int         `json:"code"`
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type PageInfo struct {
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

type PageResponse struct {
	Code    int         `json:"code"`
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Page    *PageInfo  `json:"page,omitempty"`
}

func Success(data interface{}) Response {
	return Response{
		Code:    200,
		Success: true,
		Data:    data,
	}
}

func SuccessWithMessage(data interface{}, message string) Response {
	return Response{
		Code:    200,
		Success: true,
		Message: message,
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

func ErrorWithData(code int, message string, data interface{}) Response {
	return Response{
		Code:    code,
		Success: false,
		Message: message,
		Data:    data,
	}
}

func Page(data interface{}, page, pageSize int, total int64) PageResponse {
	return PageResponse{
		Code:    200,
		Success: true,
		Data:    data,
		Page: &PageInfo{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
	}
}

func SuccessJSON(c *gin.Context, data interface{}) {
	c.JSON(200, Success(data))
}

func SuccessWithMessageJSON(c *gin.Context, data interface{}, message string) {
	c.JSON(200, SuccessWithMessage(data, message))
}

func ErrorJSON(c *gin.Context, code int, message string) {
	c.JSON(getHTTPStatus(code), Error(code, message))
}

func getHTTPStatus(code int) int {
	switch code {
	case 400:
		return 400
	case 401:
		return 401
	case 403:
		return 403
	case 404:
		return 404
	case 422:
		return 422
	case 500:
		return 500
	case 503:
		return 503
	default:
		return 200
	}
}

var (
	OK                  = Error(200, "成功")
	BadRequest          = Error(400, "请求参数错误")
	Unauthorized        = Error(401, "未授权")
	Forbidden          = Error(403, "禁止访问")
	NotFound            = Error(404, "资源不存在")
	ValidationFailed   = Error(422, "数据验证失败")
	InternalServerError = Error(500, "服务器内部错误")
	ServiceUnavailable  = Error(503, "服务不可用")
)
