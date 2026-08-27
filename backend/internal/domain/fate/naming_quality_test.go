package fate

import (
	"math"
	"strings"
	"testing"

	"name/internal/domain/classics"
)

// TestIsNonNamingChar 名字用字质量门禁判断
//
// 门禁字（false→true 需被剔除）：虚词/排行字/口语物名/数字量词/叹词，
// 任何语境下都无命名价值的字。
//
// 非门禁字（需保留）：高频好名字及其变体，严禁误伤。
func TestIsNonNamingChar(t *testing.T) {
	// 应判定为「不适合入名」的门禁字
	gate := []string{
		// 文言虚词/助词
		"之", "其", "于", "以", "而", "则", "且", "乃", "焉", "哉", "兮", "乎",
		"也", "矣", "所", "为", "与", "及", "或", "此", "彼", "尔", "兹", "既",
		// 叹词/语气词
		"啊", "呀", "哦", "嗯", "唉", "咦", "哟", "呜",
		// 排行字
		"伯", "仲", "叔", "季",
		// 口语/物名/无实义字（verify_fate 复测抓到的高分荒谬字）
		"伙", "腕", "膨", "咐", "沽", "混", "冒", "濒", "懈", "迄", "熔", "绰",
		"秧", "瓢", "粱", "罕", "纽", "壬", "沦", "沾", "港",
		// 动词/物名/无实义字（第二轮 verify_fate 复测扩充抓取）
		"带", "陵", "撰", "停", "毛", "逢", "冯", "比", "溉", "俯", "戒", "必",
		"乳", "吞", "咽", "吐", "咬", "啃", "嚼", "吹", "撒", "拉", "拽", "拖",
		"拧", "掐", "扭", "扯", "撕", "剁", "砍", "劈", "砸", "摔", "踩",
		"踢", "踹", "揍", "抠", "脖", "膀", "肢", "窍", "肤",
		// 数字/量词（无命名价值）
		"二", "四", "五", "六", "七", "八", "十", "半", "个", "只", "块", "片",
		// 否定虚字
		"勿", "毋", "弗", "否",
		// 双名全量枚举第三轮抓取（无命名价值字）
		"腰", "侍", "免", "梨", "摘", "老", "忘", "莫", "熏", "被",
		"漱", "便", "沉", "回",
	}
	for _, ch := range gate {
		if !IsNonNamingChar(ch) {
			t.Errorf("IsNonNamingChar(%q) = false, 期望 true（应被门禁剔除）", ch)
		}
	}

	// 必须保留的好名字用字（若被误伤则是灾难）
	allow := []string{
		// 高频好名用字（含虚词/数字属性但已有稳定积极寓意）
		"若", "如", "然", "斯", "唯", "惟", "一", "三", "九", "百", "千", "万",
		// 常规好名字用字
		"华", "毅", "辉", "涛", "轩", "书", "宇", "瑞", "明", "文", "云", "和",
		"子", "睿", "涵", "博", "卓", "雅", "琪", "琳", "雨", "雪", "梅", "兰",
		// 双名全量枚举第三轮抓取中的好字（有典籍典故/策展认可，严禁误伤）
		"都", "路", "年", "微", "白", "江", "北", "河", "凡", "墨",
		"游", "鹤", "媚", "冉", "复",
		// 边界/空值
		"", "a", "1",
	}
	for _, ch := range allow {
		if IsNonNamingChar(ch) {
			t.Errorf("IsNonNamingChar(%q) = true, 期望 false（好名字被误伤）", ch)
		}
	}
}

// TestIsNonNamingCharEmpty 空串安全
func TestIsNonNamingCharEmpty(t *testing.T) {
	if IsNonNamingChar("") {
		t.Error("IsNonNamingChar(\"\") = true, 期望 false")
	}
}

// TestCuratedCharPool 统计型门禁：策展字符池判断
//
// 注入已知策展好名后，池内字（浩/然/文/子）返回 true；
// 荒谬字（姐/赌/货/了）从未在策展库出现，返回 false。
func TestCuratedCharPool(t *testing.T) {
	// 注入小型策展集（模拟 curated_names.json 的精简样本）
	base := []string{"子轩", "浩然", "文远", "白汝"}
	SetCuratedNames(base)
	defer SetCuratedNames(nil)

	inPool := []string{"子", "轩", "浩", "然", "文", "远", "白", "汝"}
	for _, ch := range inPool {
		if !IsCuratedChar(ch) {
			t.Errorf("IsCuratedChar(%q) = false, 期望 true（应为策展字符池内）", ch)
		}
	}

	outPool := []string{"姐", "赌", "货", "了", "隶", "蛤", "溃", ""}
	for _, ch := range outPool {
		if IsCuratedChar(ch) {
			t.Errorf("IsCuratedChar(%q) = true, 期望 false（荒谬字不应在策展池）", ch)
		}
	}
}

// TestSetCuratedNamesEmpty 空集合/空字符串安全
func TestSetCuratedNamesEmpty(t *testing.T) {
	SetCuratedNames([]string{"", "   ", "子"})
	defer SetCuratedNames(nil)

	if IsCuratedChar("") {
		t.Error("IsCuratedChar(\"\") = true, 期望 false")
	}
	if IsCuratedName("", "子") {
		t.Error("IsCuratedName(\"\", \"子\") = true, 期望 false")
	}
}

// TestHistoricalFigureComboExpansion 历史人物字/号专名（新增先秦条目）
func TestHistoricalFigureComboExpansion(t *testing.T) {
	combos := []string{"子文", "子方", "子夏", "子游", "子贡", "子路", "子思", "子产"}
	for _, c := range combos {
		runes := []rune(c)
		if len(runes) != 2 {
			// 单字不适用组合检测，跳过
			continue
		}
		c1, c2 := string(runes[0]), string(runes[1])
		if !IsHistoricalFigureCombo(c1, c2) {
			t.Errorf("IsHistoricalFigureCombo(%q/%q) = false, 期望 true（%s 为历史人物字）", c1, c2, c)
		}
	}

	// 反序也应命中（子文/文子均剔除）
	if !IsHistoricalFigureCombo("文", "子") {
		t.Error("IsHistoricalFigureCombo(文/子) = false, 期望 true（反序检测）")
	}
}

// newCleanCandidate 构造一个不触发诗词/共现等额外加减分的干净候选
// （单名，空字义；HasPoetry 置 true 跳过 checkSemanticPoetry 诗词分支，
//  聚焦统计型门禁段的语义/生僻字惩罚逻辑；单名共现分支
//  checkSingleNameBigram 仍会执行，其 classics 数据由 TestMain 加载）。
func newCleanCandidate(char string, lvl int) *NameCandidate {
	return &NameCandidate{
		Char1:        char,
		Char2:        "",
		Meaning1:     "",
		Meaning2:     "",
		IsRegular:    false,
		HasPoetry:    true, // 跳过 checkSemanticPoetry 数据依赖
		CommonLevel1: lvl,
		CommonLevel2: 0,
	}
}

// TestWenHuaRaterRareCharPenalty 生僻/表外字降权
//
// 基准已从策展库切换为《通用规范汉字表》：level>=3（三级生僻字）或 level==0
// （表外补充字）应触发 -8 分/字降权，符合「首选一级、二级字表，谨慎使用三级字表」。
func TestWenHuaRaterRareCharPenalty(t *testing.T) {
	rater := &WenHuaRater{}
	// 表外补充字（level==0）
	rare := rater.Rate(newCleanCandidate("龘", 0), nil)
	if !strings.Contains(rare.Detail, "-8分") {
		t.Errorf("表外字(level=0)应触发 -8 分降权，详情=%q", rare.Detail)
	}
	// 三级生僻字（level==3）
	level3 := rater.Rate(newCleanCandidate("龘", 3), nil)
	if !strings.Contains(level3.Detail, "-8分") {
		t.Errorf("三级生僻字(level=3)应触发 -8 分降权，详情=%q", level3.Detail)
	}
	// 一级常用字（level==1）不受降权
	common := rater.Rate(newCleanCandidate("龘", 1), nil)
	if strings.Contains(common.Detail, "-8分") {
		t.Errorf("一级常用字(level=1)不应触发 -8 分降权，详情=%q", common.Detail)
	}
}

// TestWenHuaRaterGateCharPenalty 语义门禁字重罚
//
// 门禁字（虚词/排行字/口语物名/动词等任何语境下都无命名价值）应触发 -12 分/字重罚，
// 使其无法仅靠常用字/字义/诗词加分进入推荐榜。
func TestWenHuaRaterGateCharPenalty(t *testing.T) {
	rater := &WenHuaRater{}
	// 覆盖三类门禁字：排行字（仲）、物名/不吉利（带/陵）、动词（撰/停）
	for _, ch := range []string{"仲", "带", "陵", "撰", "停"} {
		got := rater.Rate(newCleanCandidate(ch, 1), nil)
		if !strings.Contains(got.Detail, "-12分") {
			t.Errorf("门禁字 %q 应触发 -12 分重罚，详情=%q", ch, got.Detail)
		}
	}
}

// TestWenHuaRaterCleanCharNoPenalty 优质字不受统计门禁惩罚
//
// 常用一级/二级好字（浩/然/文/子/翰/墨/澄/渊）既非门禁字也非生僻字，不应被惩罚。
func TestWenHuaRaterCleanCharNoPenalty(t *testing.T) {
	rater := &WenHuaRater{}
	for _, ch := range []string{"浩", "然", "文", "子", "翰", "墨", "澄", "渊"} {
		got := rater.Rate(newCleanCandidate(ch, 1), nil)
		if strings.Contains(got.Detail, "-12分") || strings.Contains(got.Detail, "-8分") {
			t.Errorf("优质字 %q 不应触发统计门禁惩罚，详情=%q", ch, got.Detail)
		}
	}
}

// TestHardNegativeCharBlood 体液生理字硬禁用
//
// verify_fate 双名 Top5 混入「血凡」（血=血液，与鲜血/流血强联想），
// 应走 hardNegativeChars 硬禁用层（候选池阶段剔除），而非软惩罚。
func TestHardNegativeCharBlood(t *testing.T) {
	if !IsHardNegativeChar("血") {
		t.Error("IsHardNegativeChar(血) = false, 期望 true（生理秽物字应硬禁用）")
	}
	// 同类生理字应保持硬禁用
	for _, ch := range []string{"骨", "骸", "尸", "棺"} {
		if !IsHardNegativeChar(ch) {
			t.Errorf("IsHardNegativeChar(%q) = false, 期望 true", ch)
		}
	}
	// 对照组：优质字绝不硬禁用
	for _, ch := range []string{"浩", "然", "文", "子", "白", "江", "北"} {
		if IsHardNegativeChar(ch) {
			t.Errorf("IsHardNegativeChar(%q) = true, 期望 false（优质字被误伤）", ch)
		}
	}
}

// TestForbiddenComboGeoName 地名组合禁忌
//
// verify_fate 双名 Top5 混入「白河/江北」（陕西白河县/重庆江北区等存世地名）。
// 白/江/北/河 单字均为优质常用字不入门禁，但两两组合撞地理专名，应组合级剔除。
func TestForbiddenComboGeoName(t *testing.T) {
	if !IsBadCombo("白", "河") {
		t.Error("IsBadCombo(白, 河) = false, 期望 true（白河为存世地名）")
	}
	if !IsBadCombo("江", "北") {
		t.Error("IsBadCombo(江, 北) = false, 期望 true（江北为存世地名）")
	}
	// 反序也应命中（白河/河白、江北/北江均剔除）
	if !IsBadCombo("河", "白") {
		t.Error("IsBadCombo(河, 白) = false, 期望 true（反序检测）")
	}
	// 对照组：优质好名组合绝不被误伤
	for _, c := range [][2]string{{"浩", "然"}, {"文", "子"}, {"白", "墨"}, {"江", "月"}} {
		if IsBadCombo(c[0], c[1]) {
			t.Errorf("IsBadCombo(%q, %q) = true, 期望 false（优质组合被误伤）", c[0], c[1])
		}
	}
}

// TestWenHuaRaterCuratedCooccurrence 共现分白名单（方案B）
//
// verify_fate 全量枚举的双名候选（逐求/值接/梁董/伏清/法皇）构成字均为一二级规范字
// 且不在门禁表，仅凭 CommonLevel+门禁 条件拦截不了——它们靠典籍同句共现拿满分。
// 方案B：满分共现加成仅授予「策展白名单」认可的搭配（IsCuratedName），
// 其余典籍共现组合一律 ×0.4 降权。
func TestWenHuaRaterCuratedCooccurrence(t *testing.T) {
	// 候选须为双名且两字均为一二级规范字
	cand := &NameCandidate{
		Char1:        "逐",
		Char2:        "求",
		Meaning1:     "",
		Meaning2:     "",
		IsRegular:    true,
		HasPoetry:    true,
		CommonLevel1: 1,
		CommonLevel2: 1,
	}
	// 须确认「逐求」确有典籍共现分（否则共现分支根本不执行，测试无意义）
	if _, _, found := classics.GetBigramScore("逐", "求"); !found {
		t.Skip("数据集未收录「逐求」共现，跳过共现分白名单测试")
	}

	rater := &WenHuaRater{}

	// 1. 未获策展认可 → 共现分彻底归零（方案B+，原方案B为×0.4降权）
	SetCuratedNames(nil)
	ungated := rater.Rate(cand, nil)
	if !strings.Contains(ungated.Detail, "共现分归零") {
		t.Errorf("非策展组合共现分应归零，详情=%q", ungated.Detail)
	}

	// 2. 注入策展白名单（含「逐求」）→ 满分共现加成，无降权字样
	SetCuratedNames([]string{"逐求"})
	defer SetCuratedNames(nil)
	gated := rater.Rate(cand, nil)
	if strings.Contains(gated.Detail, "降权") {
		t.Errorf("策展认可组合不应降权，详情=%q", gated.Detail)
	}
	if !strings.Contains(gated.Detail, "共现于") {
		t.Errorf("策展认可组合应获得满分共现加成，详情=%q", gated.Detail)
	}
}

// TestRateNameNonCuratedCap 方案B+：非策展双名组合四维封顶
//
// verify_fate 全量枚举双名候选池高达 130 万，荒谬组合（祝董/至律/遂姊/弟昆/至令）
// 的构成字全是一二级规范字、Meaning 非空、音韵生肖三才恰好满分，四维齐满分总分
// 89+ 霸榜。方案B+ 在 RateName 聚合层对「非策展且无诗词出典」的双名组合
// 将文化印象/音韵/生肖/三才四维封顶 75（五行八字命理维度不封顶），
// 荒谬组合总分上限 = 85×0.30 + 75×0.70 = 78，必然低于策展/出典好名。
func TestRateNameNonCuratedCap(t *testing.T) {
	SetCuratedNames(nil)
	defer SetCuratedNames(nil)
	raters := DefaultRaters()

	// 构造四维天然满分的非策展双名组合（无诗词出典）
	// 音韵：声调/声母/韵母均不同 → 89+；三才：笔画一奇一偶+差3+五行相生+部首不同 → ~95+；
	// 生肖：两字与生肖五行相生 → 94+；文化：常用+字义+笔画 → 70+。
	// 五行：刻意给出高分 85（命理维度不封顶，用于验证总分上限 = 85×0.3+75×0.7 = 78）。
	cand := &NameCandidate{
		Char1:        "祝",
		Char2:        "董",
		Meaning1:     "祝颂",
		Meaning2:     "董事",
		WuXing1:      "火",
		WuXing2:      "火",
		Pinyin1:      "zhu4",
		Pinyin2:      "dong3",
		Stroke1:      9,
		Stroke2:      12,
		Radical1:     "礻",
		Radical2:     "艹",
		IsRegular:    true,
		CommonLevel1: 1,
		CommonLevel2: 1,
		SurnamePinyin: "wang2",
	}
	sx := &FateData{
		WuXingXiji: WuXingXiji{Xi: "火", Ji: "水"}, // 喜用神火：两字（祝/董）皆火 → 高分
		BaziInfo:   BaziInfo{Zodiac: "马"},         // 马=火，与两字火相生
	}

	sc := RateName(cand, sx, raters)

	// 1. 文化/音韵/生肖/三才四维必须全部 ≤75
	for _, dim := range []string{"文化印象", "音韵", "生肖", "三才"} {
		if sc.Items[dim] > 75 {
			t.Errorf("非策展组合维度 %q = %.1f，期望 ≤75（方案B+封顶）", dim, sc.Items[dim])
		}
	}

	// 2. 五行维度不封顶（命理核心），两字皆火为喜用神 → 高分（≥75）
	if sc.Items["五行八字"] < 75 {
		t.Errorf("五行八字维度 = %.1f，方案B+不应封顶命理维度", sc.Items["五行八字"])
	}

	// 3. 总分上限：五行79×0.30 + 四维75×0.70 = 76.2，必须 <80
	if sc.Total >= 80 {
		t.Errorf("非策展荒谬组合总分 = %.1f，期望 <80（方案B+封顶后上限78）", sc.Total)
	}
}

// TestRateNameCuratedExempt 方案B+：策展双名组合豁免四维封顶
//
// 策展白名单（IsCuratedName）与诗词出典是「组合有文化含量」的两个硬证据。
// 命中策展的组合即使四维天然高分也不封顶，保证优质名维持高分。
func TestRateNameCuratedExempt(t *testing.T) {
	SetCuratedNames([]string{"祝董"})
	defer SetCuratedNames(nil)
	raters := DefaultRaters()

	cand := &NameCandidate{
		Char1:        "祝",
		Char2:        "董",
		Meaning1:     "祝颂",
		Meaning2:     "董事",
		WuXing1:      "火",
		WuXing2:      "火",
		Pinyin1:      "zhu4",
		Pinyin2:      "dong3",
		Stroke1:      9,
		Stroke2:      12,
		Radical1:     "礻",
		Radical2:     "艹",
		IsRegular:    true,
		CommonLevel1: 1,
		CommonLevel2: 1,
		SurnamePinyin: "wang2",
	}
	sx := &FateData{
		WuXingXiji: WuXingXiji{Xi: "火", Ji: "水"}, // 喜用神火：两字皆火 → 五行高分
		BaziInfo:   BaziInfo{Zodiac: "马"},         // 马=火，与两字火相生
	}

	sc := RateName(cand, sx, raters)

	// 策展组合豁免封顶：音韵/三才/生肖应保持天然高分（>75）
	for _, dim := range []string{"音韵", "三才", "生肖"} {
		if sc.Items[dim] <= 75 {
			t.Errorf("策展组合维度 %q = %.1f，期望豁免封顶保持 >75", dim, sc.Items[dim])
		}
	}

	// 总分不应被封顶压制到 70 以下（权重调整后预期降低，含6%人名频率维度）
	if sc.Total < 74 {
		t.Errorf("策展组合总分 = %.1f，期望豁免封顶 ≥74", sc.Total)
	}
}

// TestRateNamePoetryNotExempt 方案B+：单字诗词出典不豁免双名组合封顶
//
// 双名场景下 HasPoetry/PoetryFrom 是单字级证据（engine 中 poetryFound =
// 两字任一单字出现在诗词中即 true），常用字几乎必有单字出典——「团炸/号觅」
// 这类组合荒谬但单字平凡的名字会借它规避封顶。因此诗词出典不作为豁免条件：
// 非策展双名即使有单字出典，四维仍须封顶。
func TestRateNamePoetryNotExempt(t *testing.T) {
	SetCuratedNames(nil)
	defer SetCuratedNames(nil)
	raters := DefaultRaters()

	cand := &NameCandidate{
		Char1:        "浩",
		Char2:        "然",
		Meaning1:     "浩大",
		Meaning2:     "自然",
		WuXing1:      "水",
		WuXing2:      "火",
		Pinyin1:      "hao4",
		Pinyin2:      "ran2",
		Stroke1:      10,
		Stroke2:      12,
		Radical1:     "氵",
		Radical2:     "灬",
		IsRegular:    true,
		CommonLevel1: 1,
		CommonLevel2: 1,
		PoetryFrom:   "《孟子·公孙丑》",
		HasPoetry:    true,
		SurnamePinyin: "lai2",
	}
	sx := &FateData{
		WuXingXiji: WuXingXiji{Xi: "木", Ji: "土"}, // 浩(水)生木，为喜用神 → 五行高分
		BaziInfo:   BaziInfo{Zodiac: "虎"},
	}

	sc := RateName(cand, sx, raters)

	// 音韵天然高分（声调4/2不同、声母h/r不同、韵母ao/an不同 → ~97）
	// 非策展 + 单字出典 ≠ 组合有文化含量 → 仍须封顶到 75
	if sc.Items["音韵"] > 75 {
		t.Errorf("非策展组合（即便有单字出典）音韵 = %.1f，期望 ≤75（方案B+封顶）", sc.Items["音韵"])
	}
}

// TestYinYunRaterSingleNameDiscrimination 单名音韵维度区分度
//
// 背景：verify_fate 单名 Top5 长期全部同分（如 73.7/73.7/73.7），因单名时
// YinYunRater 对 p2 为空直接返回恒定 80，音韵维度完全失去区分度，荒谬字
// （缔/跳/熄/抖）与优质字同分并列，排序由候选池枚举顺序决定 → 门禁字
// 永远打不完。
//
// 治本：单名场景应借姓氏拼音 + 单字做声调/声母/韵母搭配与谐音检测，
// 使音韵维度恢复区分度（好字高于荒谬字）。
func TestYinYunRaterSingleNameDiscrimination(t *testing.T) {
	rater := &YinYunRater{}

	// 单名且无姓氏拼音 → 信息不足，回退到温和基准值（不报错即可）
	noSurname := rater.Rate(&NameCandidate{Char1: "浩", Pinyin1: "hao4"}, nil)
	if noSurname.Score <= 0 || noSurname.Score > 100 {
		t.Errorf("单名无姓氏拼音音韵 = %.1f，期望在 0-100 内", noSurname.Score)
	}

	// 单音韵维度本就不该区分「荒谬字/好字」——两者与姓氏搭配同样顺口是合理的。
	// 治本目标是：单名音韵不再对所有字恒定返回同一分值（原实现固定 80），
	// 而是对「拼音搭配」敏感——不同声调/声母/韵母搭配得不同分。
	// 例：王(wang2)+浩(hao4)：声调2/4异(+6)、声母w/h异(+4)、韵母ang/ao异(+3) → ~93
	//   王(wang2)+华(hua2)：声调2/2同(-4)、声母w/h异(+4)、韵母ang/ua异(+3) → ~83
	good := rater.Rate(&NameCandidate{
		Char1: "浩", Pinyin1: "hao4", SurnamePinyin: "wang2",
	}, nil)
	other := rater.Rate(&NameCandidate{
		Char1: "华", Pinyin1: "hua2", SurnamePinyin: "wang2",
	}, nil)

	// 关键断言：单名音韵对拼音搭配敏感，不同搭配产生实际差异分（>5 分）
	if math.Abs(good.Score-other.Score) < 5.0 {
		t.Errorf("单名音韵对拼音搭配不敏感：浩=%.1f 华=%.1f，仍近似恒定", good.Score, other.Score)
	}
}

// TestGetToneFromPinyinUnicode 声调解析支持 Unicode 声调符号（生产数据格式）
//
// 背景：hanzi.json 拼音为 Unicode 声调符号（wánɡ/tí/jué/nǔ/tān/shǎnɡ），
// 而旧 getToneFromPinyin 只识别末尾数字 '1'-'4' → 声调维度对生产数据全部失效
// （tone 恒 0），单名音韵只剩声母/韵母比较，不同字对同一姓氏恒定同分
// （如 提/诀/努/瘫/晌 对「王」全部 80+4+3=87）→ 单名 Top5 全同分的直接元凶。
func TestGetToneFromPinyinUnicode(t *testing.T) {
	cases := []struct {
		pinyin string
		want   int
	}{
		// Unicode 声调符号（生产数据 hanzi.json 格式）
		{"wánɡ", 2}, {"tí", 2}, {"jué", 2}, {"nǔ", 3}, {"tān", 1}, {"shǎnɡ", 3},
		{"huá", 2}, {"yì", 4}, {"huī", 1}, {"jūn", 1}, {"tāo", 1}, {"hào", 4},
		// 数字后缀（测试/词表格式）仍须兼容
		{"wang2", 2}, {"hao4", 4}, {"hua2", 2}, {"lai2", 2},
		// 空/无声调
		{"", 0}, {"wang", 0}, {},
	}
	for _, c := range cases {
		got := getToneFromPinyin(c.pinyin)
		if got != c.want {
			t.Errorf("getToneFromPinyin(%q) = %d，期望 %d", c.pinyin, got, c.want)
		}
	}
}

// TestStripToneUnicode 谐音检测拼音归一化支持 Unicode 声调符号与 U+0261 特殊字形
//
// 背景：hanzi.json「王」拼音为 wánɡ，末尾是 U+0261（Latin script g）而非普通 g；
// 旧 stripTone 只去数字后缀，既不规整声调符也无法归一 U+0261 → 谐音词表
// （wang/dao/si...纯拼音）匹配全部失效，不吉谐音检测形同虚设。
func TestStripToneUnicode(t *testing.T) {
	cases := []struct {
		pinyin string
		want   string
	}{
		{"wánɡ", "wang"}, {"tí", "ti"}, {"jué", "jue"}, {"nǔ", "nu"},
		{"tān", "tan"}, {"shǎnɡ", "shang"}, {"huá", "hua"}, {"yì", "yi"},
		{"wang2", "wang"}, {"hao4", "hao"},
		{"", ""},
	}
	for _, c := range cases {
		got := stripTone(c.pinyin)
		if got != c.want {
			t.Errorf("stripTone(%q) = %q，期望 %q", c.pinyin, got, c.want)
		}
	}
}

// TestYinYunRaterUnicodeDiscrimination 生产数据格式（Unicode 声调符号）下单名音韵区分度
//
// 回归场景：verify_fate「男2024龙单名」中 提/诀/努/瘫/晌 对姓氏「王」音韵曾全部恒定 87，
// 修复 getToneFromPinyin（识别 Unicode 声调符号）后应按声调错开。
//
// 注意：不能选 提/努 做对照——提(tí)谐音「亡」(王)、努(nǔ)谐音「奴」，各自命中
// 谐音惩罚后分数被拉平（73/73），恰好证明谐音检测在 Unicode 格式下已生效。
// 此处用不命中单字谐音的字验证声调区分度：
//   - 华 huá / 王 wánɡ：声调 2/2 相同(-4)、声母 h/w 异(+4)、韵母 ua/ang 异(+3) → 83
//   - 浩 hào / 王 wánɡ：声调 4/2 相异(+6)、声母 h/w 异(+4)、韵母 ao/ang 异(+3) → 93
func TestYinYunRaterUnicodeDiscrimination(t *testing.T) {
	rater := &YinYunRater{}
	// 同调（华 huá 与王 wánɡ 均 2 声）→ 声调扣分，应低于异调
	same := rater.Rate(&NameCandidate{Char1: "华", Pinyin1: "huá", SurnamePinyin: "wánɡ"}, nil)
	// 异调（浩 hào 4 声 vs 王 wánɡ 2 声）→ 声调加分，应高于同调
	diff := rater.Rate(&NameCandidate{Char1: "浩", Pinyin1: "hào", SurnamePinyin: "wánɡ"}, nil)
	if same.Score >= diff.Score {
		t.Errorf("Unicode 声调符号下同调华(%.1f)应低于异调浩(%.1f)，声调维度未生效",
			same.Score, diff.Score)
	}
}

// TestSancaiRaterSingleNameDiscrimination 单名三才维度区分度（笔画适中度）
//
// 背景：单名 Stroke2==0 时阴阳/匀称/五行生克/部首互补全部跳过，三才恒定 70，
// 与音韵恒定 80 共同构成「单名全维同分」；修复后以单字笔画适中度补充区分度。
func TestSancaiRaterSingleNameDiscrimination(t *testing.T) {
	rater := &SancaiRater{}

	// 笔画适中（8/12/15 画）→ 加分，从基准 70 提高
	mid := rater.Rate(&NameCandidate{Char1: "浩", Stroke1: 11}, nil)
	if mid.Score <= 70 {
		t.Errorf("单名笔画适中三才 = %.1f，期望 > 70（笔画适中加分）", mid.Score)
	}

	// 笔画过简（≤4 画）→ 减分，低于适中档
	simple := rater.Rate(&NameCandidate{Char1: "一", Stroke1: 1}, nil)
	if simple.Score >= mid.Score {
		t.Errorf("单名笔画过简三才 = %.1f，期望 < 适中档 %.1f", simple.Score, mid.Score)
	}

	// 笔画过繁（≥23 画）→ 减分，与 AGENTS.md「避免笔画过于复杂名字」一致
	heavy := rater.Rate(&NameCandidate{Char1: "龘", Stroke1: 48}, nil)
	if heavy.Score >= mid.Score {
		t.Errorf("单名笔画过繁三才 = %.1f，期望 < 适中档 %.1f", heavy.Score, mid.Score)
	}
}

// TestWenHuaRaterCuratedBonus 策展字文化加分（治本方案：破荒谬字同分）
//
// 背景：verify_fate 单名 Top5 荒谬字（贪/疟/骂/吠/靶/振/凑/递/备/宦/够/播/蚊/浅/
// 眉/驳/圃/沈/亩/泥/沟）与好字五维同分霸榜，门禁黑名单（IsNonNamingChar）打了
// 11 轮荒谬字换面孔打不完——荒谬字与好字在字级字段（等级/IsRegular/IsNameable/
// 扣分/消极）上完全不可区分。人工策展覆盖表（17 分类约 446 字，IsCurated1/2）
// 是唯一强区分信号：好字 8/12 在表内，荒谬字 21/22 不在。
// 治本：WenHuaRater 给策展字 +8 分/字 文化加分，荒谬字不得分 → 总分拉开差距，
// 优质字进入 Top5、荒谬字自然跌出。
// 二轮收紧：策展表是「分类字表」而非「精选好字表」，平庸字（软/际/映/耿/宝/念/
// 畅/章/好/典）也在表内拿到 +8 导致 Top5 被平庸字霸榜。策展加分进一步要求
// PositiveScore>=85（namer.json 寓意评分：优质字 87-91 有值、平庸/荒谬字为空），
// 把加分收窄为「策展 ∩ positiveScore>=85」精选好字专属。
func TestWenHuaRaterCuratedBonus(t *testing.T) {
	rater := &WenHuaRater{}

	mkCand := func(isCurated bool, positiveScore int) *NameCandidate {
		return &NameCandidate{
			Char1:        "贪", // 单名（荒谬字，但用于非策展对照/策展对照差异仅来自 IsCurated1）
			Char2:        "",
			Meaning1:     "",
			Meaning2:     "",
			IsRegular:    true,
			HasPoetry:    true, // 跳过 checkSemanticPoetry 数据依赖
			CommonLevel1: 1,
			CommonLevel2: 0,
			IsCurated1:   isCurated,
			PositiveScore1: positiveScore,
		}
	}

	// 非策展字（荒谬字）→ 无策展加分
	plain := rater.Rate(mkCand(false, 0), nil)
	if strings.Contains(plain.Detail, "策展起名好字") {
		t.Errorf("非策展荒谬字不应获得策展加分，详情=%q", plain.Detail)
	}

	// 策展字（positiveScore>=90 精选好字）→ 文化维度 +8 分，且显著高于非策展同字
	boni := rater.Rate(mkCand(true, 91), nil)
	if !strings.Contains(boni.Detail, "策展起名好字") {
		t.Errorf("策展好字应获得策展加分，详情=%q", boni.Detail)
	}
	if boni.Score < plain.Score+7 {
		t.Errorf("策展好字文化分 = %.1f，期望比非策展 %.1f 高约 +8 分", boni.Score, plain.Score)
	}

	// 平庸字：虽在策展表内（IsCurated1=true）但 positiveScore 为空 → 不加分
	// （软/际/映/耿/宝/念/畅/章/好/典 均属此类，是 Top5 平庸字霸榜的根源）
	mediocre := rater.Rate(mkCand(true, 0), nil)
	if strings.Contains(mediocre.Detail, "策展起名好字") {
		t.Errorf("平庸字（在策展表但正分为空）不应获得策展加分，详情=%q", mediocre.Detail)
	}
	if mediocre.Score >= boni.Score {
		t.Errorf("平庸字文化分 = %.1f，期望显著低于策展好字 %.1f", mediocre.Score, boni.Score)
	}

	// 双名：仅一字策展 → +8 分
	mix := &NameCandidate{
		Char1: "毅", Char2: "贪", // 毅=策展好字（在表+正分91），贪=荒谬字（不在表）
		Meaning1: "", Meaning2: "",
		IsRegular:    true,
		HasPoetry:    true,
		CommonLevel1: 1,
		CommonLevel2: 1,
		IsCurated1:   true,
		IsCurated2:   false,
		PositiveScore1: 91,
		PositiveScore2: 0,
	}
	mixRating := rater.Rate(mix, nil)
	if !strings.Contains(mixRating.Detail, "策展起名好字") {
		t.Errorf("双名含策展好字应获得策展加分，详情=%q", mixRating.Detail)
	}
}
