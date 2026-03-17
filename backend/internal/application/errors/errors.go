package errors

import "fmt"

type Code int

const (
	ErrCodeSuccess          Code = 200
	ErrCodeBadRequest       Code = 400
	ErrCodeUnauthorized     Code = 401
	ErrCodeForbidden        Code = 403
	ErrCodeNotFound         Code = 404
	ErrCodeValidationFailed Code = 422
	ErrCodeInternalError    Code = 500
	ErrCodeServiceUnavailable Code = 503
)

type AppError struct {
	Code    Code   `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[Code: %d] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[Code: %d] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func NewError(code Code, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

func NewErrorWithErr(code Code, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

func BadRequest(message string) *AppError {
	return NewError(ErrCodeBadRequest, message)
}

func NotFound(resource string) *AppError {
	return NewError(ErrCodeNotFound, resource+"不存在")
}

func InternalError(message string) *AppError {
	return NewError(ErrCodeInternalError, message)
}

func ValidationFailed(message string) *AppError {
	return NewError(ErrCodeValidationFailed, message)
}

var (
	ErrInvalidDate    = BadRequest("日期参数不合法")
	ErrInvalidTime    = BadRequest("时间参数不合法")
	ErrInvalidSurname = BadRequest("姓氏参数不合法")
	ErrInvalidGender = BadRequest("性别参数不合法")
	ErrNameNotFound   = NotFound("名字")
	ErrHistoryNotFound = NotFound("历史记录")
)
