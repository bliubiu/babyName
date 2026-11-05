package fate

// kinds.go — 门禁白名单/红线条目类型枚举（docs/33 门禁层 M1/M2）
//
// 原声明在 materials/schema（docs/43 P1-9 下沉）：这两组 kind 是 fate
// 领域自身的门禁概念，schema 仅承载 JSON 装载，不应反向定义领域枚举。

// 白名单条目类型：allowlist 物料的 kind 取值
const (
	// AllowKindChar 单字级：豁免 naming_gate 等字级黑名单
	AllowKindChar = "char"
	// AllowKindCombo 组合级：豁免 forbidden_combos / 历史人物组合
	AllowKindCombo = "combo"
	// AllowKindBlessing 祈福组合：豁免软性消极惩罚并参与评分加分
	AllowKindBlessing = "blessing"
)

// 合规红线条目类型：compliance_redline 物料的 kind 取值
const (
	// RedlineKindStateOrg 国家/国家机关/地区机关/军事警务等机关单位
	RedlineKindStateOrg = "state_org"
	// RedlineKindPlace 争议地名（行政区划/敏感地名，可交叉行政区划库）
	RedlineKindPlace = "place"
	// RedlineKindPerson 负面历史人物（同名不当联想）
	RedlineKindPerson = "person"
)
