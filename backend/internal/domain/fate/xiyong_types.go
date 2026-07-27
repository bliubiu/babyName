package fate

// XiYongMethod 喜用神算法类型
type XiYongMethod int

const (
	XiYongMethodBalance XiYongMethod = iota // 平衡用神法
	XiYongMethodGeJu                        // 格局用神法
)

// XiYongJiChou 喜用忌仇四神
type XiYongJiChou struct {
	Xi   string `json:"xi"`   // 喜神
	Yong string `json:"yong"` // 用神
	Ji   string `json:"ji"`   // 忌神
	Chou string `json:"chou"` // 仇神
}

// GeJuType 格局类型
type GeJuType int

const (
	GeJuZhengGuan GeJuType = iota // 正官格
	GeJuQiSha                     // 七杀格
	GeJuZhengCai                  // 正财格
	GeJuPianCai                   // 偏财格
	GeJuZhengYin                  // 正印格
	GeJuPianYin                   // 偏印格
	GeJuShiShen                   // 食神格
	GeJuShangGuan                 // 伤官格
	GeJuUnknown                   // 未知格局
)

// GeJuInfo 格局信息
type GeJuInfo struct {
	Type     GeJuType `json:"type"`
	Name     string   `json:"name"`
	YongShen string   `json:"yong_shen"`
	XiShen   string   `json:"xi_shen"`
	JiShen   string   `json:"ji_shen"`
	ChouShen string   `json:"chou_shen"`
	Analysis string   `json:"analysis"`
}

// WuxingStrength 五行力量
type WuxingStrength struct {
	WuxingFen map[string]float64 `json:"wuxing_fen"`
	Total     float64            `json:"total"`
}

// BaziInfoForGeJu 格局计算用八字信息
// 适配自 bazi 包，不依赖外部库
type BaziInfoForGeJu struct {
	SiZhu  [4]string `json:"si_zhu"`  // 四柱（年柱/月柱/日柱/时柱）
	WuXing [4]string `json:"wu_xing"` // 四柱五行
}
