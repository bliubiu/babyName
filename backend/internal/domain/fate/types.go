// Package fate 起名引擎核心接口与类型定义
//
// 借鉴 fate-main (github.com/babyname/fate) 的接口抽象设计，提供:
// - Fate/Session 生命周期管理
// - Filter 可组合过滤链
// - Rater 可扩展评分器
// - ExcellentTable 流式 Top-N 数据结构
package fate

import (
	"time"
)

// SessionState 命名会话状态
type SessionState int32

const (
	SessionStatePending    SessionState = iota // 等待开始
	SessionStateGenerating                      // 生成中
	SessionStateFinish                          // 生成完成
	SessionStateFailed                          // 生成失败
	SessionStateCanceled                        // 已取消
)

// Gender 性别
type Gender string

const (
	GenderMale   Gender = "male"
	GenderFemale Gender = "female"
)

// Input 命名会话输入参数
type Input struct {
	Surname    string         `json:"surname"`     // 姓氏
	Generation string         `json:"generation"`  // 字辈（可选）
	Gender     Gender         `json:"gender"`      // 性别
	Born       time.Time      `json:"born"`        // 出生时间
	Options    GenerateOptions `json:"options"`     // 生成选项

	// 避讳长辈姓名列表（父系/母系直系长辈，建议往上两代）
	// 引擎将排除这些姓名中的同形字与同音字
	AvoidElderNames []string `json:"avoid_elder_names,omitempty"`
}

// GenerateOptions 名字生成选项
type GenerateOptions struct {
	NameLength      int           // 名字字数（1-4）
	Count           int           // 生成数量
	ExcludeRare     bool          // 排除生僻字
	SourceClassic   string        // 诗词来源（shijing/chuci/poetry 等）
	IncludePoetry   bool          // 包含诗词用字
	IncludeClassic  bool          // 包含经典用字
	MeaningKeywords []string      // 寓意关键词
	PinyinInitial   string        // 拼音首字母
	ExtraChars      []*Character  // 外部注入的额外候选字（诗词/经典字库来源）
}

// NameCandidate 名字候选（不含姓氏，仅名字部分）
type NameCandidate struct {
	Char1    string // 第一个字
	Char2    string // 第二个字（单名时为空）
	Pinyin1  string
	Pinyin2  string
	WuXing1  string // 五行
	WuXing2  string
	Stroke1  int // 笔画数
	Stroke2  int
	Meaning1 string // 字义
	Meaning2 string
	Radical1 string // 部首
	Radical2 string

	HasPoetry   bool   // 是否有诗词出处
	PoetryFrom  string // 诗词来源
	IsRegular   bool   // 是否为常用字

	// IsCurated1/IsCurated2 逐字是否在人工策展起名分类覆盖表中（由 hanzi 层经 Adapter 透传）
	// 策展覆盖表（约 446 字）是荒谬字与好字的强区分信号：好字 8/12 在表内，
	// 荒谬字（贪/疟/骂/吠/靶等）21/22 不在表内。WenHuaRater 据此给策展字文化加分，
	// 打破单名 Top5 荒谬字与好字五维同分的僵局（门禁黑名单打地鼠打不完的治本方案）。
	IsCurated1 bool
	IsCurated2 bool

	// PositiveScore1/PositiveScore2 逐字寓意评分（namer.json positiveScore，0-100）
	// 策展表是「分类字表」而非「精选好字表」，平庸字（软/际/映/耿等）也在表内拿到 +8 文化加分，
	// 导致单名 Top5 被平庸字霸榜。结合 positiveScore>=85 可把策展加分收窄为「精选好字」专属。
	PositiveScore1 int
	PositiveScore2 int

	// CommonLevel1/CommonLevel2 逐字常用等级（《通用规范汉字表》level，1=一级/2=二级/3=三级，0=表外补充字）
	// 由 engine.go 从 Character.CommonLevel 透传，供评分器判断生僻/表外字
	CommonLevel1 int
	CommonLevel2 int
	GenderHint   string // 性别暗示

	// 数据层策展标注（来自 hanzi.json，由 Adapter 映射）
	NamePenalty1 int // 字1 起名扣分（0=无）
	NamePenalty2 int // 字2 起名扣分（0=无）

	SurnamePinyin string // 姓氏拼音，用于音韵评分器检测跨字谐音
}

// WuXingXiji 五行喜忌
type WuXingXiji struct {
	Xi          string   `json:"xi"`           // 喜神五行
	Ji          string   `json:"ji"`           // 忌神五行
	YongShen    string   `json:"yong_shen"`    // 用神
	Chou        string   `json:"chou"`         // 仇神
	XiYongShen  []string `json:"xi_yong_shen"` // 喜用神列表（排序推荐）
	RiZhu       string   `json:"ri_zhu"`       // 日主
	RiZhuWuXing string   `json:"ri_zhu_wu_xing"`
	QiangRuo    string   `json:"qiang_ruo"` // 强弱
}

// BaziInfo 八字信息
type BaziInfo struct {
	FourPillars [4]string // 四柱（年/月/日/时）
	WuXing      [4]string // 四柱五行
	NaYin       [4]string // 四柱纳音
	Zodiac      string    // 生肖
}

// FateData 命运数据（八字分析结果）
type FateData struct {
	BaziInfo      BaziInfo    `json:"bazi_info"`
	WuXingXiji    WuXingXiji  `json:"wu_xing_xiji"`
	TiaoHouYongShen string   `json:"tiao_hou_yong_shen"` // 调候用神
}

// NameScore 名字评分
type NameScore struct {
	Total float64            `json:"total"` // 总分（0-100）
	Grade string             `json:"grade"` // 等级（上上/上吉/中吉/中平/中下/下下）
	Items map[string]float64 `json:"items"` // 各维度得分明细
}

// NameResult 名字完整分析结果
type NameResult struct {
	Rank       int        `json:"rank"`
	Surname    string     `json:"surname"`
	GivenName  string     `json:"given_name"`
	FullName   string     `json:"full_name"`
	Pinyin     string     `json:"pinyin"`
	Strokes    int        `json:"strokes"`
	WuXing     string     `json:"wu_xing"`
	Score      NameScore  `json:"score"`
	Meaning    string     `json:"meaning,omitempty"`
	PoetryFrom string     `json:"poetry_from,omitempty"`
	Reasons    []string   `json:"reasons,omitempty"`
}

// Character 汉字简略信息（对外接口用，不绑定具体模型）
type Character struct {
	Char              string   `json:"char"`
	Pinyin            []string `json:"pinyin"`
	WuXing            string   `json:"wu_xing"`
	SimplifiedStroke  int      `json:"simplified_stroke"`
	TraditionalStroke int      `json:"traditional_stroke"`
	KangxiStroke      int      `json:"kangxi_stroke"`
	ScienceStroke     int      `json:"science_stroke"`
	Radical           string   `json:"radical"`
	Meaning           string   `json:"meaning"`
	IsRegular         bool     `json:"is_regular"`
	IsNameable        bool     `json:"is_nameable"`
	CommonLevel       int      `json:"common_level"`
	GenderHint        string   `json:"gender_hint"`
	HasPoetry         bool     `json:"has_poetry"`
	NamingCategory    []string `json:"naming_category,omitempty"` // 精选起名分类标签

	// 数据层策展标注（来自 hanzi.json，由 Adapter 映射，供引擎/评分器消费）
	IsNegative    bool     `json:"is_negative,omitempty"`    // 策展判定不宜入名（压迫级）
	NamePenalty   int      `json:"name_penalty,omitempty"`   // 起名扣分（0-20，越高越不宜）
	PairBlacklist []string `json:"pair_blacklist,omitempty"` // 搭配黑名单（与这些字组合亦不宜）

	// IsCurated 是否在人工策展起名分类覆盖表中（由 hanzi.IsCuratedNamingChar 经 Adapter 透传）
	// 策展覆盖表是荒谬字与好字的强区分信号，WenHuaRater 据此给策展字文化加分破同分。
	IsCurated bool `json:"is_curated,omitempty"`

	// PositiveScore 寓意评分（namer.json positiveScore，0-100）
	// 策展表是「分类字表」而非「精选好字表」，平庸字（软/际/映/耿 等）也在表内。
	// 结合 PositiveScore>=85 可将策展文化加分收窄为精选好字专属（优质字 87-91，平庸/荒谬字空）。
	PositiveScore int `json:"positive_score,omitempty"`
}

// CharacterQuery 汉字查询接口（用于 Filter 下推查询条件）
type CharacterQuery interface {
	WhereRegular() CharacterQuery
	WhereNameable() CharacterQuery
	WhereStrokeEQ(stroke int) CharacterQuery
	WhereStrokeGTE(min int) CharacterQuery
	WhereStrokeLTE(max int) CharacterQuery
	WhereWuXingIn(wuxing ...string) CharacterQuery
	WhereWuXingNotIn(wuxing ...string) CharacterQuery
	WhereCharIn(chars ...string) CharacterQuery
	WhereGenderHint(gender string) CharacterQuery
	WhereNamingCategory(category string) CharacterQuery
}
