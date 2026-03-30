package validator

import (
	"fmt"
	"strings"
	"time"
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{Field: field, Message: message}
}

func ValidateDate(year, month, day int) error {
	if year < 1900 || year > 2100 {
		return NewValidationError("birth_year", "年份必须在1900-2100之间")
	}
	if month < 1 || month > 12 {
		return NewValidationError("birth_month", "月份必须在1-12之间")
	}
	if day < 1 || day > 31 {
		return NewValidationError("birth_day", "日期必须在1-31之间")
	}

	dateStr := fmt.Sprintf("%04d-%02d-%02d", year, month, day)
	_, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return NewValidationError("birth_date", "日期不合法")
	}

	return nil
}

func ValidateTime(hour, minute int) error {
	if hour < 0 || hour > 23 {
		return NewValidationError("birth_hour", "小时必须在0-23之间")
	}
	if minute < 0 || minute > 59 {
		return NewValidationError("birth_minute", "分钟必须在0-59之间")
	}
	return nil
}

func ValidateSurname(surname string) error {
	if surname == "" {
		return NewValidationError("surname", "姓氏不能为空")
	}
	// 去除首尾空格
	surname = strings.TrimSpace(surname)
	if len(surname) > 4 {
		return NewValidationError("surname", "姓氏长度不能超过4个字符")
	}
	for _, r := range surname {
		if r < '\u4e00' || r > '\u9fa5' {
			return NewValidationError("surname", "姓氏只能包含中文字符")
		}
	}
	return nil
}

func ValidateGender(gender string) error {
	if gender != "male" && gender != "female" {
		return NewValidationError("gender", "性别必须是male或female")
	}
	return nil
}
