package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"name/internal/application/services"
)

// writeTempConfig 写入临时 TOML 配置文件并返回路径
func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("写入临时配置失败: %v", err)
	}
	return path
}

// TestLoadTOML 验证从 TOML 文件加载起名参数
func TestLoadTOML(t *testing.T) {
	path := writeTempConfig(t, `
surname = "王"
gender = "male"
birth_year = 2024
birth_month = 1
birth_day = 15
birth_hour = 12
birth_minute = 30
birth_location = "北京"
generation = "承"
name_length = 1
exclude_rare = true
wuxing_match = ["水", "木"]
avoid_elder_names = ["王建国"]
format = "json"
count = 10
`)

	p, err := LoadTOML(path)
	if err != nil {
		t.Fatalf("LoadTOML 返回错误: %v", err)
	}

	if p.Surname != "王" {
		t.Errorf("Surname = %q, 期望 %q", p.Surname, "王")
	}
	if p.Gender != "male" {
		t.Errorf("Gender = %q, 期望 %q", p.Gender, "male")
	}
	if p.BirthYear != 2024 || p.BirthMonth != 1 || p.BirthDay != 15 {
		t.Errorf("出生日期 = %d-%d-%d, 期望 2024-1-15", p.BirthYear, p.BirthMonth, p.BirthDay)
	}
	if p.BirthHour != 12 || p.BirthMinute != 30 {
		t.Errorf("出生时间 = %d:%d, 期望 12:30", p.BirthHour, p.BirthMinute)
	}
	if p.BirthLocation != "北京" {
		t.Errorf("BirthLocation = %q, 期望 %q", p.BirthLocation, "北京")
	}
	if p.Generation != "承" {
		t.Errorf("Generation = %q, 期望 %q", p.Generation, "承")
	}
	if p.NameLength != 1 {
		t.Errorf("NameLength = %d, 期望 1", p.NameLength)
	}
	if !p.ExcludeRare {
		t.Error("ExcludeRare = false, 期望 true")
	}
	if len(p.WuxingMatch) != 2 || p.WuxingMatch[0] != "水" || p.WuxingMatch[1] != "木" {
		t.Errorf("WuxingMatch = %v, 期望 [水 木]", p.WuxingMatch)
	}
	if len(p.AvoidElderNames) != 1 || p.AvoidElderNames[0] != "王建国" {
		t.Errorf("AvoidElderNames = %v, 期望 [王建国]", p.AvoidElderNames)
	}
	if p.Format != "json" {
		t.Errorf("Format = %q, 期望 %q", p.Format, "json")
	}
	if p.Count != 10 {
		t.Errorf("Count = %d, 期望 10", p.Count)
	}
}

// TestLoadTOMLFileNotFound 验证配置文件不存在时返回明确错误
func TestLoadTOMLFileNotFound(t *testing.T) {
	_, err := LoadTOML(filepath.Join(t.TempDir(), "不存在.toml"))
	if err == nil {
		t.Fatal("配置文件不存在时应返回错误，实际返回 nil")
	}
	if !strings.Contains(err.Error(), "读取配置文件失败") {
		t.Errorf("错误信息应包含「读取配置文件失败」，实际: %v", err)
	}
}

// TestApplyOverridesPriority 验证优先级合并：显式命令行参数 > TOML 配置 > 默认值
func TestApplyOverridesPriority(t *testing.T) {
	base := &Params{
		Surname:    "李",
		Gender:     "female",
		BirthYear:  2023,
		BirthMonth: 8,
		BirthDay:   20,
		BirthHour:  10,
		NameLength: 2,
		WuxingMatch: []string{"水"},
		Format:     "text",
	}

	// 仅 gender 与 min_strokes 为命令行显式传入
	overrides := &Params{
		Gender:     "male",
		MinStrokes: 5,
		Surname:    "张", // 未显式传入，不应生效
	}
	explicit := map[string]bool{"gender": true, "min_strokes": true}

	base.ApplyOverrides(overrides, explicit)

	if base.Gender != "male" {
		t.Errorf("显式命令行参数应覆盖 TOML: Gender = %q, 期望 male", base.Gender)
	}
	if base.MinStrokes != 5 {
		t.Errorf("显式命令行参数应覆盖默认值: MinStrokes = %d, 期望 5", base.MinStrokes)
	}
	if base.Surname != "李" {
		t.Errorf("未显式传入的命令行参数不应覆盖 TOML: Surname = %q, 期望 李", base.Surname)
	}
	if base.WuxingMatch == nil || len(base.WuxingMatch) != 1 || base.WuxingMatch[0] != "水" {
		t.Errorf("未被覆盖的 TOML 切片应保留: WuxingMatch = %v, 期望 [水]", base.WuxingMatch)
	}
}

// TestApplyOverridesExplicitSlice 验证切片参数整体替换而非追加
func TestApplyOverridesExplicitSlice(t *testing.T) {
	base := &Params{WuxingMatch: []string{"水"}}
	overrides := &Params{WuxingMatch: []string{"木", "火"}}

	base.ApplyOverrides(overrides, map[string]bool{"wuxing_match": true})

	if len(base.WuxingMatch) != 2 || base.WuxingMatch[0] != "木" || base.WuxingMatch[1] != "火" {
		t.Errorf("切片应整体替换: WuxingMatch = %v, 期望 [木 火]", base.WuxingMatch)
	}
}

// TestSetDefaults 验证默认值填充
func TestSetDefaults(t *testing.T) {
	p := &Params{}
	p.SetDefaults()

	if p.NameLength != 2 {
		t.Errorf("NameLength 默认应为 2（双字名）, 实际 %d", p.NameLength)
	}
	if p.Format != "text" {
		t.Errorf("Format 默认应为 text, 实际 %q", p.Format)
	}
	if p.Count != 20 {
		t.Errorf("Count 默认应为 20, 实际 %d", p.Count)
	}
}

// TestSetDefaultsNotOverwrite 验证默认值不会覆盖已有配置
func TestSetDefaultsNotOverwrite(t *testing.T) {
	p := &Params{NameLength: 1, Format: "json", Count: 5}
	p.SetDefaults()

	if p.NameLength != 1 || p.Format != "json" || p.Count != 5 {
		t.Errorf("SetDefaults 不应覆盖已有值: %+v", p)
	}
}

// TestValidate 验证参数校验（复用 Web 层 validator 规则）
func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		params  Params
		wantErr string // 期望错误包含的关键词，空串表示期望通过
	}{
		{
			name: "合法参数",
			params: Params{
				Surname: "王", Gender: "male",
				BirthYear: 2024, BirthMonth: 1, BirthDay: 15,
				BirthHour: 12, BirthMinute: 30,
			},
		},
		{
			name:    "缺少姓氏",
			params:  Params{Gender: "male", BirthYear: 2024, BirthMonth: 1, BirthDay: 15, BirthHour: 12},
			wantErr: "姓氏",
		},
		{
			name:    "非法性别",
			params:  Params{Surname: "王", Gender: "other", BirthYear: 2024, BirthMonth: 1, BirthDay: 15, BirthHour: 12},
			wantErr: "性别",
		},
		{
			name:    "非法年份",
			params:  Params{Surname: "王", Gender: "male", BirthYear: 1800, BirthMonth: 1, BirthDay: 15, BirthHour: 12},
			wantErr: "年份",
		},
		{
			name:    "非法小时",
			params:  Params{Surname: "王", Gender: "male", BirthYear: 2024, BirthMonth: 1, BirthDay: 15, BirthHour: 25},
			wantErr: "小时",
		},
		{
			name:    "非法日期",
			params:  Params{Surname: "王", Gender: "male", BirthYear: 2024, BirthMonth: 2, BirthDay: 31, BirthHour: 12},
			wantErr: "日期不合法",
		},
		{
			name:    "非法输出格式",
			params:  Params{Surname: "王", Gender: "male", BirthYear: 2024, BirthMonth: 1, BirthDay: 15, BirthHour: 12, Format: "xml"},
			wantErr: "format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.params.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("期望校验通过，实际返回错误: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("期望返回包含「%s」的错误，实际为 nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("错误信息应包含「%s」，实际: %v", tt.wantErr, err)
			}
		})
	}
}

// TestParseCSVList 验证逗号分隔的命令行列表参数解析
func TestParseCSVList(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"", nil},
		{"水", []string{"水"}},
		{"水,木", []string{"水", "木"}},
		{" 水 , 木 ", []string{"水", "木"}},
		{"水,,木", []string{"水", "木"}}, // 空项剔除
	}

	for _, tt := range tests {
		got := ParseCSVList(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("ParseCSVList(%q) = %v, 期望 %v", tt.input, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("ParseCSVList(%q) = %v, 期望 %v", tt.input, got, tt.want)
				break
			}
		}
	}
}

// TestToRequest 验证转换为服务层请求结构体（字段对齐 GenerateRequest）
func TestToRequest(t *testing.T) {
	p := &Params{
		Surname:         "王",
		Generation:      "承",
		GenerationPosition: "middle",
		Gender:          "male",
		BirthYear:       2024,
		BirthMonth:      1,
		BirthDay:        15,
		BirthHour:       12,
		BirthMinute:     30,
		BirthLocation:   "北京",
		NameType:        "double",
		Preferences:     []string{"文雅"},
		NameLength:      2,
		ExcludeRare:     true,
		WuxingMatch:     []string{"水"},
		SourceClassic:   "shijing",
		MinStrokes:      3,
		MaxStrokes:      18,
		IncludePoetry:   true,
		IncludeClassic:  true,
		MeaningKeywords: []string{"志"},
		PinyinInitial:   "z",
		AvoidElderNames: []string{"王建国"},
	}

	req := p.ToRequest()
	if req == nil {
		t.Fatal("ToRequest 不应返回 nil")
	}

	want := services.GenerateRequest{
		Surname:         "王",
		Generation:      "承",
		GenerationPosition: "middle",
		Gender:          "male",
		BirthYear:       2024,
		BirthMonth:      1,
		BirthDay:        15,
		BirthHour:       12,
		BirthMinute:     30,
		BirthLocation:   "北京",
		NameType:        "double",
		Preferences:     []string{"文雅"},
		NameLength:      2,
		ExcludeRare:     true,
		WuxingMatch:     []string{"水"},
		SourceClassic:   "shijing",
		MinStrokes:      3,
		MaxStrokes:      18,
		IncludePoetry:   true,
		IncludeClassic:  true,
		MeaningKeywords: []string{"志"},
		PinyinInitial:   "z",
		AvoidElderNames: []string{"王建国"},
	}

	if !reflect.DeepEqual(*req, want) {
		t.Errorf("ToRequest() = %+v\n期望            %+v", *req, want)
	}
}
