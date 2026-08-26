// namer-cli 命令行起名工具的参数模块
//
// 职责：
//   1. 从 TOML 配置文件加载起名参数（字段与 Web API JSON 一致，snake_case）
//   2. 应用命令行显式参数覆盖（优先级：命令行 > TOML > 默认值）
//   3. 复用 Web 层 validator 进行参数校验
//   4. 转换为服务层 GenerateRequest，确保 CLI 与 Web UI 走完全相同的领域链路
package main

import (
	"fmt"
	"strings"

	"name/internal/application/services"
	"name/internal/application/validator"

	"github.com/spf13/viper"
)

// Params CLI 起名参数
// mapstructure 标签用于 viper 解析 TOML；key 与 Web API JSON 字段保持一致，
// 方便对照 API 文档与前端请求体进行测试验证。
type Params struct {
	// 核心输入（与 Web UI 表单一致）
	Surname            string   `mapstructure:"surname"`             // 姓氏
	Generation         string   `mapstructure:"generation"`          // 字辈
	GenerationPosition string   `mapstructure:"generation_position"` // 字辈位置：middle（名中）/ tail（名尾）
	Gender             string   `mapstructure:"gender"`              // 性别：male / female
	BirthYear          int      `mapstructure:"birth_year"`          // 出生年
	BirthMonth         int      `mapstructure:"birth_month"`         // 出生月
	BirthDay           int      `mapstructure:"birth_day"`           // 出生日
	BirthHour          int      `mapstructure:"birth_hour"`          // 出生时（24小时制）
	BirthMinute        int      `mapstructure:"birth_minute"`        // 出生分
	BirthLocation      string   `mapstructure:"birth_location"`      // 出生地
	BirthType          string   `mapstructure:"birth_type"`          // 出生类型

	// 筛选条件（可选）
	NameType        string   `mapstructure:"name_type"`         // 名字类型：single / double
	Preferences     []string `mapstructure:"preferences"`       // 偏好风格
	NameLength      int      `mapstructure:"name_length"`       // 名字长度（不含姓）：1 单字 / 2 双字
	ExcludeRare     bool     `mapstructure:"exclude_rare"`      // 排除生僻字
	WuxingMatch     []string `mapstructure:"wuxing_match"`      // 五行偏好（如 ["水","木"]）
	SourceClassic   string   `mapstructure:"source_classic"`    // 典籍来源（shijing/chuci 等）
	MinStrokes      int      `mapstructure:"min_strokes"`       // 最小笔画
	MaxStrokes      int      `mapstructure:"max_strokes"`       // 最大笔画
	IncludePoetry   bool     `mapstructure:"include_poetry"`    // 包含诗词字库
	IncludeClassic  bool     `mapstructure:"include_classic"`   // 包含经典字库
	MeaningKeywords []string `mapstructure:"meaning_keywords"`  // 寓意关键词
	PinyinInitial   string   `mapstructure:"pinyin_initial"`    // 拼音首字母

	// 避讳长辈：父系/母系直系长辈姓名，生成时排除同形字与同音字
	AvoidElderNames []string `mapstructure:"avoid_elder_names"`

	// CLI 输出控制（不进入服务层请求）
	Count  int    `mapstructure:"count"`  // 展示名字数量上限
	Format string `mapstructure:"format"` // 输出格式：text / json
}

// LoadTOML 从 TOML 文件加载参数
func LoadTOML(path string) (*Params, error) {
	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var p Params
	if err := v.Unmarshal(&p); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}
	return &p, nil
}

// ApplyOverrides 应用命令行显式参数覆盖
//
// 仅当 explicit 中记录了对应参数名（即用户确实在命令行传入了该参数）时，
// 才用 overrides 中的值覆盖当前参数，保证「命令行 > TOML > 默认值」优先级。
func (p *Params) ApplyOverrides(overrides *Params, explicit map[string]bool) {
	if overrides == nil {
		return
	}
	set := func(name string, apply func()) {
		if explicit[name] {
			apply()
		}
	}

	set("surname", func() { p.Surname = overrides.Surname })
	set("generation", func() { p.Generation = overrides.Generation })
	set("generation_position", func() { p.GenerationPosition = overrides.GenerationPosition })
	set("gender", func() { p.Gender = overrides.Gender })
	set("birth_year", func() { p.BirthYear = overrides.BirthYear })
	set("birth_month", func() { p.BirthMonth = overrides.BirthMonth })
	set("birth_day", func() { p.BirthDay = overrides.BirthDay })
	set("birth_hour", func() { p.BirthHour = overrides.BirthHour })
	set("birth_minute", func() { p.BirthMinute = overrides.BirthMinute })
	set("birth_location", func() { p.BirthLocation = overrides.BirthLocation })
	set("birth_type", func() { p.BirthType = overrides.BirthType })
	set("name_type", func() { p.NameType = overrides.NameType })
	set("preferences", func() { p.Preferences = overrides.Preferences })
	set("name_length", func() { p.NameLength = overrides.NameLength })
	set("exclude_rare", func() { p.ExcludeRare = overrides.ExcludeRare })
	set("wuxing_match", func() { p.WuxingMatch = overrides.WuxingMatch })
	set("source_classic", func() { p.SourceClassic = overrides.SourceClassic })
	set("min_strokes", func() { p.MinStrokes = overrides.MinStrokes })
	set("max_strokes", func() { p.MaxStrokes = overrides.MaxStrokes })
	set("include_poetry", func() { p.IncludePoetry = overrides.IncludePoetry })
	set("include_classic", func() { p.IncludeClassic = overrides.IncludeClassic })
	set("meaning_keywords", func() { p.MeaningKeywords = overrides.MeaningKeywords })
	set("pinyin_initial", func() { p.PinyinInitial = overrides.PinyinInitial })
	set("avoid_elder_names", func() { p.AvoidElderNames = overrides.AvoidElderNames })
	set("count", func() { p.Count = overrides.Count })
	set("format", func() { p.Format = overrides.Format })
}

// SetDefaults 填充默认值（不覆盖已有配置）
func (p *Params) SetDefaults() {
	if p.NameLength == 0 {
		p.NameLength = 2 // 默认双字名
	}
	if p.Format == "" {
		p.Format = "text"
	}
	if p.Count == 0 {
		p.Count = 20
	}
}

// Validate 校验参数合法性（复用 Web 层 validator 规则，保证口径一致）
func (p *Params) Validate() error {
	if err := validator.ValidateSurname(p.Surname); err != nil {
		return fmt.Errorf("姓氏校验失败: %w", err)
	}
	if err := validator.ValidateGender(p.Gender); err != nil {
		return fmt.Errorf("性别校验失败: %w", err)
	}
	if err := validator.ValidateDate(p.BirthYear, p.BirthMonth, p.BirthDay); err != nil {
		return fmt.Errorf("出生日期校验失败: %w", err)
	}
	if err := validator.ValidateTime(p.BirthHour, p.BirthMinute); err != nil {
		return fmt.Errorf("出生时间校验失败: %w", err)
	}
	if p.Format != "" && p.Format != "text" && p.Format != "json" {
		return fmt.Errorf("format 只支持 text 或 json，实际: %s", p.Format)
	}
	return nil
}

// ToRequest 转换为服务层生成请求（CLI 与 Web UI 共用同一领域入口）
func (p *Params) ToRequest() *services.GenerateRequest {
	return &services.GenerateRequest{
		Surname:         p.Surname,
		Generation:      p.Generation,
		GenerationPosition: p.GenerationPosition,
		Gender:          p.Gender,
		BirthYear:       p.BirthYear,
		BirthMonth:      p.BirthMonth,
		BirthDay:        p.BirthDay,
		BirthHour:       p.BirthHour,
		BirthMinute:     p.BirthMinute,
		BirthLocation:   p.BirthLocation,
		BirthType:       p.BirthType,
		NameType:        p.NameType,
		Preferences:     p.Preferences,
		NameLength:      p.NameLength,
		ExcludeRare:     p.ExcludeRare,
		WuxingMatch:     p.WuxingMatch,
		SourceClassic:   p.SourceClassic,
		MinStrokes:      p.MinStrokes,
		MaxStrokes:      p.MaxStrokes,
		IncludePoetry:   p.IncludePoetry,
		IncludeClassic:  p.IncludeClassic,
		MeaningKeywords: p.MeaningKeywords,
		PinyinInitial:   p.PinyinInitial,
		AvoidElderNames: p.AvoidElderNames,
	}
}

// ParseCSVList 解析逗号分隔的命令行列表参数（去空白、剔除空项）
func ParseCSVList(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if v := strings.TrimSpace(part); v != "" {
			result = append(result, v)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
