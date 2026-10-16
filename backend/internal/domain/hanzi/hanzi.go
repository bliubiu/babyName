package hanzi

// HanziData 相关全局状态与访问函数见 loader.go / namer_loader.go。

// RareChars 超生僻字集合（《通用规范汉字表》表外字优先规避）
var RareChars = map[string]bool{
	"龘": true, "靐": true, "齉": true, "齾": true, "龗": true,
	"鱻": true, "麤": true, "灪": true, "馫": true, "譶": true,
	"灨": true, "灩": true, "麷": true, "驫": true, "羴": true,
	// 扩展超生僻字（《通用规范汉字表》表外字）
	"龖": true, "龠": true, "爨": true, "灥": true, "厵": true,
	"𪚥": true, "𠀾": true, "𡿻": true, "𠄌": true, "𠂉": true,
	"𠃋": true, "𠃌": true, "𠆢": true, "𠂊": true, "𠂤": true,
	"㐬": true, "㐭": true, "㐱": true, "㑳": true, "㒭": true,
	"㓦": true, "㕭": true, "㖈": true, "㗂": true, "㘄": true,
}

func IsCommonChar(char string) bool {
	if _, ok := RareChars[char]; ok {
		return false
	}
	if data, ok := GetHanzi(char); ok {
		return data.Strokes <= 25
	}
	return true
}

type Hanzi struct {
	Char    string
	Pinyin  string
	Strokes int
	// KangxiStrokes 康熙字典笔画（来自 data/raw/kangxi-strokecount.csv），
	// 河图数理与易经卦象解读按传统易学使用康熙笔画。
	// 0 表示未收录（兜底用 Strokes 简体笔画）。
	KangxiStrokes int
	Radical       string
	Meaning       string
	Wuxing        string
	Gender        string

	// 以下字段来自 chars.json 精选起名用字库
	Tone           int      // 声调
	GenderTags     []string // 性别标签（更细粒度，如 ["male","female"]）
	StyleTags      []string // 风格标签（如 "古典"、"现代"、"幽深"）
	UsageLevel     int      // 常用等级（1-5）
	PositiveScore  int      // 寓意评分（0-100）
	PhoneticScore  int      // 音韵评分（0-100）
	ModernScore    int      // 现代感评分（0-100）
	ClassicalScore int      // 古典感评分（0-100）
	IsPolyphonic   bool     // 是否多音字
	IsRare         bool     // 是否生僻字
	IsNegative     bool     // 是否消极含义
	PairBlacklist  []string // 搭配黑名单
	CurationLevel  int      // 精选等级（1-5）
	NamePenalty    int      // 起名扣分

	// P3 — 人名频率（来自 Chinese-Names-Corpus 语料统计）
	NameFreqTier int // 人名频率档位（1-5，5=最高频，0=未收录）

	// Level 《通用规范汉字表》等级（1=一级常用字，2=二级通用字，3=三级专用字，0=表外）
	//
	// 此前该字段只存在于 NamerChar，导致 SQLite 侧无法按等级过滤（docs/31 ADR-012）
	Level int

	// NamingCategories 起名分类标签列表（0~N 个分类）
	// 由 classifier 自动标注 + 策展覆盖表修正
	NamingCategories []string
}

var HanziData = map[string]Hanzi{}

// ReplaceHanziData 以一份完整字库整体替换 HanziData（SQLite 物料装载路径，ADR-008）
//
// 整体替换而非逐键合并：物料修订与删字都必须生效，逐键合并会留下已删除的残留字。
// 读侧看不到中间态 —— 替换在写锁内一次完成。
func ReplaceHanziData(data map[string]Hanzi) {
	mu.Lock()
	defer mu.Unlock()
	if data == nil {
		data = map[string]Hanzi{}
	}
	HanziData = data
}

// HanziCount 返回字库条数（诊断/测试用）
func HanziCount() int {
	mu.RLock()
	defer mu.RUnlock()
	return len(HanziData)
}

// ForEachHanzi 遍历 HanziData 中的每个汉字（只读、线程安全）
func ForEachHanzi(fn func(Hanzi)) {
	mu.RLock()
	defer mu.RUnlock()
	for _, h := range HanziData {
		fn(h)
	}
}

// ResolveWuxing 按统一优先级解析单字的五行（字形法/字义法修正规则）
//
// 优先级：
//  1. CharacterWuxingOverride（字义法覆盖表）— 无条件应用
//  2. 已有五行值 — 保留（人工校正值）
//  3. 部首映射表计算 — 仅对空值兜底
//
// 抽为无状态函数的原因：物料导入（sqlite.Store.Import）必须与内存装载路径
// 使用**同一套**五行解析规则。原先规则只内嵌在 ApplyWuxingOverrides 里作用于
// 内存 HanziData，导致 hanzi_data.wuxing 落了 namer.json 原值（实测 14 字为空），
// SQL 路径与内存路径五行判定不一致（filter_integration_test 守护）。
//
// 本函数不读写全局状态，可安全并发调用。
func ResolveWuxing(char, current, radical string) string {
	// 1. 覆盖表优先（字义法）
	if wx, ok := CharacterWuxingOverride[char]; ok {
		return wx
	}
	// 2. 已有五行 → 保留（已人工校正）
	if current != "" {
		return current
	}
	// 3. 部首映射兜底（字形法）
	return GetWuxingByRadical(radical)
}

// ApplyWuxingOverrides 为 HanziData 中所有汉字标注五行
//
// 优先级见 ResolveWuxing（单一真源，本函数仅为批量应用）。
//
// 必须在 HanziData 填充完毕（JSON 加载后）调用
//
// 锁约定（docs/29 B9）：本函数会写 HanziData，必须持有 mu 写锁。
// 调用方在填充循环的 mu.Unlock() 之后调用，不得在持锁状态下调用，
// 否则 RWMutex 不可重入会自死锁。
func ApplyWuxingOverrides() {
	mu.Lock()
	defer mu.Unlock()

	for char, data := range HanziData {
		wx := ResolveWuxing(char, data.Wuxing, data.Radical)
		if wx != data.Wuxing {
			data.Wuxing = wx
			HanziData[char] = data
		}
	}
}
