package fate

// CharacterFilterType 字符筛选类型（位标志）
type CharacterFilterType uint

const (
	CharacterFilterTypeDefault     CharacterFilterType = 0
	CharacterFilterTypeChs         CharacterFilterType = 1 << 0 // 简体字
	CharacterFilterTypeCht         CharacterFilterType = 1 << 1 // 繁体字
	CharacterFilterTypeKangxi      CharacterFilterType = 1 << 2 // 康熙字典字
	CharacterFilterTypeNameScience CharacterFilterType = 1 << 3 // 姓名学用字
)

// HasType 检查是否包含指定类型
func (c CharacterFilterType) HasType(t CharacterFilterType) bool {
	if c == CharacterFilterTypeDefault {
		return t == CharacterFilterTypeNameScience
	}
	return (c & t) != 0
}

// StrokeMode 笔画计算模式
type StrokeMode int

const (
	StrokeModeScience    StrokeMode = iota // 科学笔画（默认）
	StrokeModeSimplified                    // 简体笔画
	StrokeModeTraditional                   // 繁体笔画
	StrokeModeKangxi                        // 康熙笔画
)

// Filter 名字候选过滤接口
//
// 借鉴 fate-main 的 Filter 设计，提供可组合的过滤链:
// - 字符级过滤: CheckCharacter
// - 笔画组合级过滤: CheckStrokePair
// - 查询级过滤: QueryFilter（下推到数据库）
// - 严格度: 支持动态降级（strict → moderate → relaxed）
type Filter interface {
	// FilterType 返回字符筛选类型
	FilterType() CharacterFilterType

	// Strictness 返回当前严格度
	Strictness() string

	// SetStrictness 动态设置严格度（用于无结果时自动降级）
	SetStrictness(strictness string)

	// CheckCharacter 检查单字是否通过字符级过滤
	// 包括: 生僻字/笔画范围/五行偏好/部首偏好/拼音回避等
	CheckCharacter(c *Character) bool

	// CheckStrokePair 检查笔画组合是否通过
	// 预留接口用于未来扩展笔画组合检查（如河图数理搭配）
	CheckStrokePair(firstStroke, secondStroke int) bool

	// QueryFilter 返回数据库查询级别的过滤函数
	// 在查询时应用，减少不必要的数据加载
	QueryFilter(query CharacterQuery) CharacterQuery

	// GetCharacterStroke 获取字符的笔画数（按当前 StrokeMode）
	GetCharacterStroke(c *Character) int

	// GetDisplayName 获取显示用字符（简体/繁体转换）
	GetDisplayName(c *Character) string

	// PreferredWuXing 返回显式指定的偏好五行（未指定时返回空）
	// 供引擎判断是否已由用户显式指定五行：未指定时引擎可按喜用神五行收窄候选池
	PreferredWuXing() []string

	// Degrade 降级严格度一个级别：strict → moderate → relaxed → ""（终点）
	// 用于当无匹配结果时自动放宽过滤条件
	Degrade() string

	// NextStrictness 返回下一个严格度级别（不修改当前值）
	NextStrictness() string
}

// FilterOption Filter 构造选项 — Builder 模式
//
// 链式调用示例:
//
//	fo := NewFilterOption().
//	    WithMinStroke(5).
//	    WithMaxStroke(18).
//	    WithAvoidChars("病", "死").
//	    WithStrictness("strict").
//	    WithPreferredWuXing("金", "水")
type FilterOption struct {
	CharacterFilter     bool
	CharacterFilterType CharacterFilterType
	StrokeMode          StrokeMode

	// 笔画范围
	MinStroke int
	MaxStroke int

	// 字符筛选
	RegularFilter  bool // 仅常用字
	CommonFilter   bool // 按常用等级
	MinCommonLevel int
	MaxCommonLevel int

	// 数理过滤（大衍数理、三才五行，遵循传统易学，非熊崎五格）
	DaYanFilter  bool // 大衍数吉凶
	WuXingFilter bool // 三才五行吉凶
	SexFilter    bool // 性别匹配

	// 五行偏好
	PreferredWuXing []string
	AvoidWuXing     []string

	// 字符过滤
	AvoidCharacters   []string
	RequireCharacters []string
	PreferredRadicals []string
	AvoidRadicals     []string
	AvoidPinyin       []string

	// 分类/性别/诗词
	PreferredCategories []string // 精选起名分类
	ExpandCategories    bool     // 是否根据兼容性矩阵自动扩展分类
	GenderFilter        string
	PoetryMode          int // 0-不限, 1-优先, 2-仅诗词

	// 人名频率过滤（来自 Chinese-Names-Corpus 语料统计）
	MinFrequencyTier int // 最小频率档位（1-5，0=不限）
	MaxFrequencyTier int // 最大频率档位（1-5，0=不限）

	// 严格度
	FilterStrictness string // strict / moderate / relaxed

	// 喜用神算法
	XiYongMethod string // balance / geju
}

// NewFilterOption 创建默认 FilterOption
func NewFilterOption() FilterOption {
	return FilterOption{
		CharacterFilter:     true,
		CharacterFilterType: CharacterFilterTypeDefault,
		StrokeMode:          StrokeModeScience,
		RegularFilter:       true,
		FilterStrictness:    "moderate",
	}
}

// WithMinStroke 设置最小笔画数
func (fo FilterOption) WithMinStroke(n int) FilterOption {
	fo.MinStroke = n
	return fo
}

// WithMaxStroke 设置最大笔画数
func (fo FilterOption) WithMaxStroke(n int) FilterOption {
	fo.MaxStroke = n
	return fo
}

// WithAvoidChars 设置排除字符列表
func (fo FilterOption) WithAvoidChars(chars ...string) FilterOption {
	fo.AvoidCharacters = append(fo.AvoidCharacters, chars...)
	return fo
}

// WithRequireChars 设置必含字符列表
func (fo FilterOption) WithRequireChars(chars ...string) FilterOption {
	fo.RequireCharacters = append(fo.RequireCharacters, chars...)
	return fo
}

// WithPreferredWuXing 设置偏好五行
func (fo FilterOption) WithPreferredWuXing(wx ...string) FilterOption {
	fo.PreferredWuXing = append(fo.PreferredWuXing, wx...)
	return fo
}

// WithAvoidWuXing 设置回避五行
func (fo FilterOption) WithAvoidWuXing(wx ...string) FilterOption {
	fo.AvoidWuXing = append(fo.AvoidWuXing, wx...)
	return fo
}

// WithPreferredRadicals 设置偏好部首
func (fo FilterOption) WithPreferredRadicals(radicals ...string) FilterOption {
	fo.PreferredRadicals = append(fo.PreferredRadicals, radicals...)
	return fo
}

// WithAvoidRadicals 设置回避部首
func (fo FilterOption) WithAvoidRadicals(radicals ...string) FilterOption {
	fo.AvoidRadicals = append(fo.AvoidRadicals, radicals...)
	return fo
}

// WithAvoidPinyin 设置回避拼音
func (fo FilterOption) WithAvoidPinyin(pinyin ...string) FilterOption {
	fo.AvoidPinyin = append(fo.AvoidPinyin, pinyin...)
	return fo
}

// WithStrictness 设置严格度
func (fo FilterOption) WithStrictness(s string) FilterOption {
	fo.FilterStrictness = s
	return fo
}

// WithGenderFilter 设置性别筛选
func (fo FilterOption) WithGenderFilter(gender string) FilterOption {
	fo.GenderFilter = gender
	return fo
}

// WithPreferredCategories 设置精选起名分类偏好
// 指定后引擎只从该分类的起名用字中生成名字
func (fo FilterOption) WithPreferredCategories(categories ...string) FilterOption {
	fo.PreferredCategories = append(fo.PreferredCategories, categories...)
	return fo
}

// WithPoetryMode 设置诗词模式
func (fo FilterOption) WithPoetryMode(mode int) FilterOption {
	fo.PoetryMode = mode
	return fo
}

// WithExpandCategories 设置是否根据兼容性矩阵自动扩展分类
// 启用后，当设置 PreferredCategories 时，引擎会自动引入兼容分类的字
func (fo FilterOption) WithExpandCategories(enable bool) FilterOption {
	fo.ExpandCategories = enable
	return fo
}

// WithRegularFilter 设置仅常用字
func (fo FilterOption) WithRegularFilter(enable bool) FilterOption {
	fo.RegularFilter = enable
	return fo
}

// WithDaYanFilter 设置大衍数吉凶过滤
func (fo FilterOption) WithDaYanFilter(enable bool) FilterOption {
	fo.DaYanFilter = enable
	return fo
}

// WithWuXingFilter 设置三才五行过滤
func (fo FilterOption) WithWuXingFilter(enable bool) FilterOption {
	fo.WuXingFilter = enable
	return fo
}

// WithSexFilter 设置性别过滤
func (fo FilterOption) WithSexFilter(enable bool) FilterOption {
	fo.SexFilter = enable
	return fo
}

// WithXiYongMethod 设置喜用神算法
func (fo FilterOption) WithXiYongMethod(method string) FilterOption {
	fo.XiYongMethod = method
	return fo
}

// WithFrequencyTier 设置人名频率档位过滤范围
func (fo FilterOption) WithFrequencyTier(min, max int) FilterOption {
	fo.MinFrequencyTier = min
	fo.MaxFrequencyTier = max
	return fo
}

// WithStrokeMode 设置笔画计算模式
func (fo FilterOption) WithStrokeMode(mode StrokeMode) FilterOption {
	fo.StrokeMode = mode
	return fo
}

// Build 构建 Filter 实例
func (fo FilterOption) Build() Filter {
	return newFilterFromOption(fo)
}

// newFilterFromOption 根据 FilterOption 构建具体的 Filter
func newFilterFromOption(fo FilterOption) Filter {
	f := &filterImpl{
		filterType: fo.CharacterFilterType,
		strokeMode: fo.StrokeMode,
		strictness: fo.FilterStrictness,
		option:     fo,
	}
	f.init()
	return f
}

// filterImpl Filter 接口实现
type filterImpl struct {
	filterType CharacterFilterType
	strokeMode StrokeMode
	strictness string
	option     FilterOption
}

func (f *filterImpl) FilterType() CharacterFilterType {
	return f.filterType
}

func (f *filterImpl) Strictness() string {
	return f.strictness
}

func (f *filterImpl) SetStrictness(s string) {
	f.strictness = s
}

func (f *filterImpl) CheckCharacter(c *Character) bool {
	// 回避字符检查
	for _, ac := range f.option.AvoidCharacters {
		if c.Char == ac {
			return false
		}
	}

	// 笔画范围检查
	stroke := f.GetCharacterStroke(c)
	if f.option.MinStroke > 0 && stroke < f.option.MinStroke {
		return false
	}
	if f.option.MaxStroke > 0 && stroke > f.option.MaxStroke {
		return false
	}

	// 常用字检查
	if f.option.RegularFilter && !c.IsRegular {
		return false
	}

	// 常用等级检查
	if f.option.CommonFilter {
		if c.CommonLevel < f.option.MinCommonLevel || c.CommonLevel > f.option.MaxCommonLevel {
			return false
		}
	}

	// 五行偏好检查
	if len(f.option.PreferredWuXing) > 0 {
		matched := false
		for _, wx := range f.option.PreferredWuXing {
			if c.WuXing == wx {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	if len(f.option.AvoidWuXing) > 0 {
		for _, wx := range f.option.AvoidWuXing {
			if c.WuXing == wx {
				return false
			}
		}
	}

	// 部首偏好检查
	if len(f.option.PreferredRadicals) > 0 {
		matched := false
		for _, rad := range f.option.PreferredRadicals {
			if c.Radical == rad {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	if len(f.option.AvoidRadicals) > 0 {
		for _, rad := range f.option.AvoidRadicals {
			if c.Radical == rad {
				return false
			}
		}
	}

	// 拼音回避检查
	if len(f.option.AvoidPinyin) > 0 {
		for _, p := range c.Pinyin {
			for _, ap := range f.option.AvoidPinyin {
				if p == ap {
					return false
				}
			}
		}
	}

	// 性别检查
	if f.option.GenderFilter != "" {
		if c.GenderHint != f.option.GenderFilter && c.GenderHint != "neutral" {
			return false
		}
	}

	// 人名频率档位检查（来自 Chinese-Names-Corpus 语料统计）
	if f.option.MinFrequencyTier > 0 {
		if c.NameFreqTier < f.option.MinFrequencyTier {
			return false
		}
	}
	if f.option.MaxFrequencyTier > 0 {
		if c.NameFreqTier > f.option.MaxFrequencyTier {
			return false
		}
	}

	return true
}

func (f *filterImpl) CheckStrokePair(firstStroke, secondStroke int) bool {
	// 熊崎五格数理过滤已移除（遵循 AGENTS.md 禁用约束）
	// 预留接口用于未来扩展河图数理搭配等传统易学笔画检查
	return true
}

// Degrade 降级严格度：strict → moderate → relaxed → ""（终点）
func (f *filterImpl) Degrade() string {
	f.strictness = f.NextStrictness()
	return f.strictness
}

// NextStrictness 返回下一个严格度级别（不修改当前值）
func (f *filterImpl) NextStrictness() string {
	switch f.strictness {
	case "strict":
		return "moderate"
	case "moderate":
		return "relaxed"
	default:
		return ""
	}
}

func (f *filterImpl) QueryFilter(query CharacterQuery) CharacterQuery {
	q := query

	if f.option.RegularFilter {
		q = q.WhereRegular().WhereNameable()
	}
	if f.option.MinStroke > 0 {
		q = q.WhereStrokeGTE(f.option.MinStroke)
	}
	if f.option.MaxStroke > 0 {
		q = q.WhereStrokeLTE(f.option.MaxStroke)
	}
	if len(f.option.PreferredWuXing) > 0 {
		q = q.WhereWuXingIn(f.option.PreferredWuXing...)
	}
	if len(f.option.AvoidWuXing) > 0 {
		q = q.WhereWuXingNotIn(f.option.AvoidWuXing...)
	}
	if f.option.GenderFilter != "" {
		q = q.WhereGenderHint(f.option.GenderFilter)
	}
	if len(f.option.PreferredCategories) > 0 {
		if f.option.ExpandCategories {
			// 启用风格兼容扩展：收集主分类+兼容分类的所有字，用 WhereCharIn 传入
			var expandedChars []string
			for _, cat := range f.option.PreferredCategories {
				chars := GetCompatibleCharsByCategory(cat, true, 0)
				expandedChars = append(expandedChars, chars...)
			}
			// 去重
			seen := make(map[string]bool)
			var unique []string
			for _, ch := range expandedChars {
				if !seen[ch] {
					seen[ch] = true
					unique = append(unique, ch)
				}
			}
			q = q.WhereCharIn(unique...)
		} else {
			q = q.WhereNamingCategory(f.option.PreferredCategories[0])
		}
	}

	return q
}

func (f *filterImpl) GetCharacterStroke(c *Character) int {
	switch f.strokeMode {
	case StrokeModeSimplified:
		return c.SimplifiedStroke
	case StrokeModeTraditional:
		return c.TraditionalStroke
	case StrokeModeKangxi:
		return c.KangxiStroke
	default:
		if c.ScienceStroke > 0 {
			return c.ScienceStroke
		}
		return c.KangxiStroke
	}
}

func (f *filterImpl) GetDisplayName(c *Character) string {
	return c.Char
}

// PreferredWuXing 返回显式指定的偏好五行（未指定时返回空）
func (f *filterImpl) PreferredWuXing() []string {
	return f.option.PreferredWuXing
}

func (f *filterImpl) init() {
	// 初始化默认值
	if f.strictness == "" {
		f.strictness = "moderate"
	}
}
