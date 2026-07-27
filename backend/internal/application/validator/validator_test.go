package validator

import (
	"strings"
	"testing"
)

// TestValidateDate 覆盖日期校验的边界与合法性
func TestValidateDate(t *testing.T) {
	tests := []struct {
		name               string
		year, month, day   int
		wantErr            bool
		wantField          string
	}{
		{"合法日期", 2000, 6, 15, false, ""},
		{"闰年2月29合法", 2020, 2, 29, false, ""},
		{"平年2月29非法", 2019, 2, 29, true, "birth_date"},
		{"2000闰年2月29合法", 2000, 2, 29, false, ""},
		{"4月31日非法", 2000, 4, 31, true, "birth_date"},
		{"4月30日合法", 2000, 4, 30, false, ""},
		{"年份过小", 1899, 1, 1, true, "birth_year"},
		{"年份下界", 1900, 1, 1, false, ""},
		{"年份上界", 2100, 12, 31, false, ""},
		{"年份过大", 2101, 1, 1, true, "birth_year"},
		{"月份过小", 2000, 0, 1, true, "birth_month"},
		{"月份过大", 2000, 13, 1, true, "birth_month"},
		{"日过小", 2000, 1, 0, true, "birth_day"},
		{"日过大", 2000, 1, 32, true, "birth_day"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDate(tt.year, tt.month, tt.day)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ValidateDate(%d,%d,%d) 期望错误，实际 nil", tt.year, tt.month, tt.day)
				}
				ve, ok := err.(*ValidationError)
				if !ok {
					t.Fatalf("期望 *ValidationError，实际 %T", err)
				}
				if tt.wantField != "" && ve.Field != tt.wantField {
					t.Errorf("Field 期望 %s，实际 %s", tt.wantField, ve.Field)
				}
			} else if err != nil {
				t.Errorf("ValidateDate(%d,%d,%d) 期望 nil，实际 %v", tt.year, tt.month, tt.day, err)
			}
		})
	}
}

// TestValidateTime 覆盖时间校验边界
func TestValidateTime(t *testing.T) {
	tests := []struct {
		name         string
		hour, minute int
		wantErr      bool
		wantField    string
	}{
		{"合法时间", 12, 30, false, ""},
		{"零点", 0, 0, false, ""},
		{"23点59分", 23, 59, false, ""},
		{"小时过小", -1, 0, true, "birth_hour"},
		{"小时过大", 24, 0, true, "birth_hour"},
		{"分钟过小", 12, -1, true, "birth_minute"},
		{"分钟过大", 12, 60, true, "birth_minute"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTime(tt.hour, tt.minute)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ValidateTime(%d,%d) 期望错误，实际 nil", tt.hour, tt.minute)
				}
				ve, ok := err.(*ValidationError)
				if !ok {
					t.Fatalf("期望 *ValidationError，实际 %T", err)
				}
				if tt.wantField != "" && ve.Field != tt.wantField {
					t.Errorf("Field 期望 %s，实际 %s", tt.wantField, ve.Field)
				}
			} else if err != nil {
				t.Errorf("ValidateTime(%d,%d) 期望 nil，实际 %v", tt.hour, tt.minute, err)
			}
		})
	}
}

// TestValidateSurname 覆盖姓氏校验：空、空格、复姓、超长、非中文
func TestValidateSurname(t *testing.T) {
	tests := []struct {
		name      string
		surname   string
		wantErr   bool
		wantField string
	}{
		{"单字姓", "李", false, ""},
		{"复姓欧阳", "欧阳", false, ""},
		{"复姓南宫", "南宫", false, ""},
		{"空字符串", "", true, "surname"},
		{"纯空格", "   ", true, "surname"},
		{"含字母", "A", true, "surname"},
		{"中文夹数字", "李1", true, "surname"},
		{"五字超长", "张王李赵刘", true, "surname"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSurname(tt.surname)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ValidateSurname(%q) 期望错误，实际 nil", tt.surname)
				}
				ve, ok := err.(*ValidationError)
				if !ok {
					t.Fatalf("期望 *ValidationError，实际 %T", err)
				}
				if tt.wantField != "" && ve.Field != tt.wantField {
					t.Errorf("Field 期望 %s，实际 %s", tt.wantField, ve.Field)
				}
			} else if err != nil {
				t.Errorf("ValidateSurname(%q) 期望 nil，实际 %v", tt.surname, err)
			}
		})
	}
}

// TestValidateGivenName 覆盖名字校验：空、空格、多字、超长、非中文
func TestValidateGivenName(t *testing.T) {
	tests := []struct {
		name    string
		given   string
		wantErr bool
	}{
		{"单字名", "天", false},
		{"双字名", "伯通", false},
		{"三字名", "诸葛孔明", false},
		{"空字符串", "", true},
		{"纯空格", "  ", true},
		{"含字母", "Tom", true},
		{"九字超长", "一二三四五六七八九", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGivenName(tt.given)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ValidateGivenName(%q) 期望错误，实际 nil", tt.given)
				}
			} else if err != nil {
				t.Errorf("ValidateGivenName(%q) 期望 nil，实际 %v", tt.given, err)
			}
		})
	}
}

// TestValidateGender 覆盖性别校验
func TestValidateGender(t *testing.T) {
	tests := []struct {
		name    string
		gender  string
		wantErr bool
	}{
		{"男", "male", false},
		{"女", "female", false},
		{"空", "", true},
		{"其他", "unknown", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGender(tt.gender)
			if tt.wantErr && err == nil {
				t.Errorf("ValidateGender(%q) 期望错误，实际 nil", tt.gender)
			} else if !tt.wantErr && err != nil {
				t.Errorf("ValidateGender(%q) 期望 nil，实际 %v", tt.gender, err)
			}
		})
	}
}

// TestValidationError 验证错误结构体格式
func TestValidationError(t *testing.T) {
	ve := NewValidationError("surname", "姓氏不能为空")
	if ve.Field != "surname" {
		t.Errorf("Field 期望 surname，实际 %s", ve.Field)
	}
	if ve.Message != "姓氏不能为空" {
		t.Errorf("Message 期望 姓氏不能为空，实际 %s", ve.Message)
	}
	want := "surname: 姓氏不能为空"
	if got := ve.Error(); got != want {
		t.Errorf("Error() 期望 %q，实际 %q", want, got)
	}
}

// TestValidationErrorEmpty 确保空字段也能正常输出
func TestValidationErrorEmpty(t *testing.T) {
	ve := NewValidationError("", "出错")
	if !strings.Contains(ve.Error(), "出错") {
		t.Errorf("Error() 应包含消息，实际 %q", ve.Error())
	}
}
