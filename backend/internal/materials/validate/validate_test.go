package validate

import (
	"strings"
	"testing"

	"name/internal/materials/schema"
)

// 本文件守护 docs/31 ADR-004：装载期强校验。
// 每个校验函数覆盖「合法零错误」与「每类违规至少一例」，
// 错误归因四元组（物料/字段/值/规则）必须完整可读。

const testMaterial = "namer"

func TestWord_合法字零错误(t *testing.T) {
	w := schema.Word{
		Char: "永", Strokes: 5, Wuxing: "水", Gender: schema.Male,
		Level: 1, Tone: 3, NameFreqTier: 4,
		PositiveScore: 100, PhoneticScore: 0, ModernScore: 100, ClassicalScore: 100,
		NamePenalty: 0,
	}
	if errs := Word(testMaterial, w); errs.Len() != 0 {
		t.Fatalf("合法字应零错误，实得 %d 条: %v", errs.Len(), errs)
	}
}

func TestWord_空五行合法(t *testing.T) {
	// 空五行 = 「不臆造」，是合法值（schema.ValidWuxing 契约）
	w := schema.Word{Char: "永", Strokes: 5, Gender: schema.Neutral}
	if errs := Word(testMaterial, w); errs.Len() != 0 {
		t.Fatalf("空五行应合法，实得: %v", errs)
	}
}

func TestWord_各类违规可归因(t *testing.T) {
	w := schema.Word{
		Char:    "永水", // 非单字
		Strokes: 0, Wuxing: "气", Gender: "unknown",
		Level: 4, Tone: 5, NameFreqTier: 6,
		PositiveScore: 101, PhoneticScore: -1, ModernScore: 200, ClassicalScore: 101,
		NamePenalty: 101,
	}
	errs := Word(testMaterial, w)
	if errs.Len() != 12 {
		t.Fatalf("期望 12 条错误（12 个违规字段），实得 %d 条: %v", errs.Len(), errs)
	}

	// 每条错误都必须携带完整归因
	for _, e := range errs {
		if e.Material != testMaterial {
			t.Errorf("错误未记录物料名: %+v", e)
		}
		if e.Field == "" || e.Rule == "" {
			t.Errorf("错误缺少字段或规则: %+v", e)
		}
	}
	// 抽查关键规则的文案（防止规则被静默改写）
	rules := make(map[string]string, len(errs))
	for _, e := range errs {
		rules[e.Field] = e.Rule
	}
	for field, want := range map[string]string{
		"char":          "必须是单个汉字",
		"strokes":       "简体笔画必须 ≥1",
		"wuxing":        "五行必须是 金木水火土 之一或空",
		"gender":        "性别必须是 male/female/neutral",
		"level":         "字表等级必须在 0~3",
		"tone":          "声调必须在 0~4",
		"nameFreqTier":  "人名频率档必须在 0~5",
		"positiveScore": "评分必须在 0~100",
	} {
		if rules[field] != want {
			t.Errorf("字段 %s 的规则期望 %q，实得 %q", field, want, rules[field])
		}
	}
}

func TestWord_五评分共用同一规则(t *testing.T) {
	// 五个评分字段越界时都必须命中「评分必须在 0~100」
	w := schema.Word{
		Char: "永", Strokes: 1, Gender: schema.Neutral,
		PositiveScore: 101, PhoneticScore: 101, ModernScore: 101, ClassicalScore: 101, NamePenalty: 101,
	}
	errs := Word(testMaterial, w)
	if errs.Len() != 5 {
		t.Fatalf("期望 5 条评分错误，实得 %d 条", errs.Len())
	}
	for _, e := range errs {
		if e.Rule != "评分必须在 0~100" {
			t.Errorf("字段 %s 规则异常: %q", e.Field, e.Rule)
		}
	}
}

func TestWords_主键重复检测(t *testing.T) {
	ws := []schema.Word{
		{Char: "永", Strokes: 1, Gender: schema.Neutral},
		{Char: "永", Strokes: 2, Gender: schema.Neutral},
	}
	errs := Words(testMaterial, ws, 100)
	if errs.Len() != 1 {
		t.Fatalf("期望 1 条主键重复错误，实得 %d 条: %v", errs.Len(), errs)
	}
	if errs[0].Rule != "主键重复" {
		t.Errorf("规则期望「主键重复」，实得 %q", errs[0].Rule)
	}

	// 重复分支内的 maxErr 提前返回（覆盖 Words 的第二处截断点）
	early := Words(testMaterial, ws, 1)
	if early.Len() != 1 {
		t.Fatalf("maxErr=1 应截断为 1 条，实得 %d 条", early.Len())
	}
}

func TestWords_错误上限截断(t *testing.T) {
	ws := []schema.Word{
		{Char: "甲", Strokes: 0}, {Char: "乙", Strokes: 0}, {Char: "丙", Strokes: 0},
	}
	errs := Words(testMaterial, ws, 2)
	if errs.Len() != 2 {
		t.Fatalf("maxErr=2 应截断为 2 条，实得 %d 条", errs.Len())
	}
}

func TestWords_输出顺序稳定(t *testing.T) {
	ws := []schema.Word{
		{Char: "甲", Strokes: 0}, {Char: "甲", Strokes: 1},
		{Char: "乙", Strokes: 0},
	}
	first := Words(testMaterial, ws, 100).Error()
	for i := 0; i < 20; i++ {
		if got := Words(testMaterial, ws, 100).Error(); got != first {
			t.Fatalf("错误输出顺序不稳定：\n首次=%s\n本次=%s", first, got)
		}
	}
}

func TestCharGroup(t *testing.T) {
	known := map[string]bool{"永": true, "水": true}

	// 合法分组零错误
	ok := schema.CharGroup{Radical: "水", Chars: []string{"永", "水"}}
	if errs := CharGroup(testMaterial, ok, known); errs.Len() != 0 {
		t.Fatalf("合法分组应零错误，实得: %v", errs)
	}

	// known 为 nil 时跳过引用完整性检查（调用方未提供字表）
	noKnown := schema.CharGroup{Radical: "水", Chars: []string{"龘"}}
	if errs := CharGroup(testMaterial, noKnown, nil); errs.Len() != 0 {
		t.Fatalf("known=nil 不应报引用错误，实得: %v", errs)
	}

	// 部首为空 + 分组为空 + 非单字 + 字不在字表
	bad := schema.CharGroup{Radical: "", Chars: nil}
	errs := CharGroup(testMaterial, bad, known)
	if errs.Len() != 2 {
		t.Fatalf("期望 2 条错误（部首空 + 分组空），实得 %d 条: %v", errs.Len(), errs)
	}
	unknownChar := schema.CharGroup{Radical: "水", Chars: []string{"永", "龘", "水木"}}
	errs = CharGroup(testMaterial, unknownChar, known)
	if errs.Len() != 2 {
		t.Fatalf("期望 2 条错误（龘不在表 + 水木非单字），实得 %d 条: %v", errs.Len(), errs)
	}
	for _, e := range errs {
		if e.Field != "chars" {
			t.Errorf("错误字段期望 chars，实得 %q", e.Field)
		}
	}
}

func TestMeaning(t *testing.T) {
	ok := schema.Meaning{Char: "永", Explanation: "水流长"}
	if errs := Meaning(testMaterial, ok); errs.Len() != 0 {
		t.Fatalf("合法释义应零错误，实得: %v", errs)
	}
	// 纯空白释义等同为空
	blank := schema.Meaning{Char: "永", Explanation: "   "}
	if errs := Meaning(testMaterial, blank); errs.Len() != 1 {
		t.Fatalf("空白释义应报 1 条错误，实得 %d 条", errs.Len())
	}
	multi := schema.Meaning{Char: "永水", Explanation: "x"}
	if errs := Meaning(testMaterial, multi); errs.Len() != 1 {
		t.Fatalf("非单字应报 1 条错误，实得 %d 条", errs.Len())
	}
}

func TestCharFreq(t *testing.T) {
	ok := schema.CharFreq{Char: "永", Count: 0, Tier: 5}
	if errs := CharFreq(testMaterial, ok); errs.Len() != 0 {
		t.Fatalf("count=0 合法（零频次仍是数据），实得: %v", errs)
	}
	bad := schema.CharFreq{Char: "永水", Count: -1, Tier: 6}
	if errs := CharFreq(testMaterial, bad); errs.Len() != 3 {
		t.Fatalf("期望 3 条错误，实得 %d 条: %v", errs.Len(), errs)
	}
}

func TestBigramFreq(t *testing.T) {
	ok := schema.BigramFreq{Char1: "司", Char2: "马", Count: 10, Tier: 5}
	if errs := BigramFreq(testMaterial, ok); errs.Len() != 0 {
		t.Fatalf("合法双字应零错误，实得: %v", errs)
	}
	bad := schema.BigramFreq{Char1: "司马", Char2: "", Count: -1, Tier: 9}
	if errs := BigramFreq(testMaterial, bad); errs.Len() != 4 {
		t.Fatalf("期望 4 条错误，实得 %d 条: %v", errs.Len(), errs)
	}
}

func TestKangxi(t *testing.T) {
	ok := schema.KangxiStroke{Char: "永", Strokes: 5}
	if errs := Kangxi(testMaterial, ok); errs.Len() != 0 {
		t.Fatalf("合法康熙笔画应零错误，实得: %v", errs)
	}
	zero := schema.KangxiStroke{Char: "永", Strokes: 0}
	if errs := Kangxi(testMaterial, zero); errs.Len() != 1 {
		t.Fatalf("strokes=0 应报错，实得 %d 条", errs.Len())
	}
	multi := schema.KangxiStroke{Char: "永水", Strokes: 5}
	if errs := Kangxi(testMaterial, multi); errs.Len() != 1 {
		t.Fatalf("非单字应报错，实得 %d 条", errs.Len())
	}
}

func TestHexagram(t *testing.T) {
	seen := map[int]bool{2: true}
	ok := schema.Hexagram{ID: 1, Name: "乾"}
	if errs := Hexagram(testMaterial, ok, seen); errs.Len() != 0 {
		t.Fatalf("合法卦应零错误，实得: %v", errs)
	}
	dup := schema.Hexagram{ID: 2, Name: "坤"}
	if errs := Hexagram(testMaterial, dup, seen); errs.Len() != 1 {
		t.Fatalf("卦序重复应报 1 条错误，实得 %d 条", errs.Len())
	}
	out := schema.Hexagram{ID: 65, Name: ""}
	if errs := Hexagram(testMaterial, out, nil); errs.Len() != 2 {
		t.Fatalf("越界+空名应报 2 条错误，实得 %d 条", errs.Len())
	}
	// seen 为 nil 时跳过唯一性检查
	if errs := Hexagram(testMaterial, dup, nil); errs.Len() != 0 {
		t.Fatalf("seen=nil 不应报重复，实得: %v", errs)
	}
}

func TestZodiac(t *testing.T) {
	ok := schema.Zodiac{ID: 1, Name: "鼠"}
	if errs := Zodiac(testMaterial, ok); errs.Len() != 0 {
		t.Fatalf("合法生肖应零错误，实得: %v", errs)
	}
	bad := schema.Zodiac{ID: 13, Name: ""}
	if errs := Zodiac(testMaterial, bad); errs.Len() != 2 {
		t.Fatalf("越界+空名应报 2 条错误，实得 %d 条", errs.Len())
	}
}

func TestRune(t *testing.T) {
	if errs := Rune(testMaterial, "chars", "永"); errs.Len() != 0 {
		t.Fatalf("单字应零错误，实得: %v", errs)
	}
	if errs := Rune(testMaterial, "chars", "永水"); errs.Len() != 1 {
		t.Fatalf("非单字应报 1 条错误，实得 %d 条", errs.Len())
	}
	if errs := Rune(testMaterial, "chars", ""); errs.Len() != 1 {
		t.Fatalf("空串应报 1 条错误，实得 %d 条", errs.Len())
	}
}

func TestCombo(t *testing.T) {
	if errs := Combo(testMaterial, "司马"); errs.Len() != 0 {
		t.Fatalf("双字组合应零错误，实得: %v", errs)
	}
	for _, v := range []string{"司", "司马光", ""} {
		if errs := Combo(testMaterial, v); errs.Len() != 1 {
			t.Errorf("组合 %q 应报 1 条错误，实得 %d 条", v, errs.Len())
		}
	}
}

func TestError_文本格式(t *testing.T) {
	e := Error{Material: "namer", Field: "char", Value: "永水", Rule: "必须是单个汉字"}
	want := "物料[namer] 字段[char] 值[\"永水\"] 违反规则[必须是单个汉字]"
	if got := e.Error(); got != want {
		t.Fatalf("错误文本期望 %q，实得 %q", want, got)
	}
}

func TestErrors_集合语义(t *testing.T) {
	if (Errors(nil)).Len() != 0 {
		t.Error("nil Errors 的 Len 应为 0")
	}
	errs := Errors{
		{Material: "a", Field: "f1", Value: "v1", Rule: "r1"},
		{Material: "a", Field: "f2", Value: "v2", Rule: "r2"},
	}
	if errs.Len() != 2 {
		t.Fatalf("Len 期望 2，实得 %d", errs.Len())
	}
	msg := errs.Error()
	if !strings.Contains(msg, "字段[f1]") || !strings.Contains(msg, "字段[f2]") {
		t.Errorf("集合消息应包含全部错误：%q", msg)
	}
	if !strings.Contains(msg, "; ") {
		t.Errorf("多条错误应以分号连接：%q", msg)
	}
}

// TestErrors_Summarize聚合 守护 docs/34 阶段 C2：
// 同规则多条错误必须聚合为一条计数，输出稳定可回归。
func TestErrors_Summarize聚合(t *testing.T) {
	if got := (Errors(nil)).Summarize(3); got != "" {
		t.Errorf("空集合 Summarize 应为空串，实得 %q", got)
	}

	errs := Errors{
		{Material: "a", Field: "strokes", Value: "0", Rule: "简体笔画必须 ≥1"},
		{Material: "a", Field: "strokes", Value: "-2", Rule: "简体笔画必须 ≥1"},
		{Material: "a", Field: "char", Value: "永水", Rule: "必须是单个汉字"},
		{Material: "a", Field: "char", Value: "和淼", Rule: "必须是单个汉字"},
		{Material: "a", Field: "char", Value: "空", Rule: "必须是单个汉字"},
	}
	got := errs.Summarize(2)

	// 聚合计数：strokes × 2、char × 3（样例取 sorted 后首个值）
	if !strings.Contains(got, "共 5 条校验问题") {
		t.Errorf("应报告总条数：%q", got)
	}
	if !strings.Contains(got, "字段[strokes] 规则[简体笔画必须 ≥1] × 2（样例 \"-2\"）") {
		t.Errorf("strokes 规则应聚合为 ×2：%q", got)
	}
	if !strings.Contains(got, "字段[char] 规则[必须是单个汉字] × 3（样例 \"和淼\"）") {
		t.Errorf("char 规则应聚合为 ×3：%q", got)
	}
	// 明细条数受 sampleN 限制
	if !strings.Contains(got, "明细前 2 条") {
		t.Errorf("明细应截断到前 2 条：%q", got)
	}

	// 稳定性：同输入两次输出必须一致（sorted 保证）
	if again := errs.Summarize(2); again != got {
		t.Errorf("Summarize 输出不稳定：\n%q\n%q", got, again)
	}

	// sampleN 超过总数时不截断
	if full := errs.Summarize(100); !strings.Contains(full, "明细前 5 条") {
		t.Errorf("sampleN 超过总数时应给全部明细：%q", full)
	}
}

func TestEntity(t *testing.T) {
	// 合法：2~8 个汉字且不含空白
	for _, v := range []string{"北京", "省高院", "国务院", "中共中央"} {
		if errs := Entity(testMaterial, "state_org", v); errs.Len() != 0 {
			t.Errorf("Entity(%q) 应零错误，实得: %v", v, errs)
		}
	}
	// 非法：单字 / 超长 / 含空白
	for _, v := range []string{"京", "二三四五六七八九十", "北 京", " 北京"} {
		if errs := Entity(testMaterial, "state_org", v); errs.Len() == 0 {
			t.Errorf("Entity(%q) 应报错，实得零错误", v)
		}
	}
	// 错误归因到物料/字段
	errs := Entity(testMaterial, "place", "京")
	if errs.Len() != 1 || errs[0].Material != testMaterial || errs[0].Field != "place" {
		t.Errorf("错误归因错误：%+v", errs)
	}
	// 同时违反两条规则（超长 + 含空白）应报 2 条
	if two := Entity(testMaterial, "state_org", "二 三四五六七八九"); two.Len() != 2 {
		t.Errorf("超长且含空白应报 2 条错误，实得 %d：%v", two.Len(), two)
	}
}
