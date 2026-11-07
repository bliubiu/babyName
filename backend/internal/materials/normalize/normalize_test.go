package normalize

import (
	"reflect"
	"strings"
	"testing"

	"name/internal/materials/schema"
)

// 本文件守护 docs/31 ADR-009：归一化是有序纯函数步骤，
// 离线导入与在线装载共用同一实现，且必须满足幂等性 norm(norm(x))==norm(x)。

func TestGender_枚举归一(t *testing.T) {
	cases := []struct {
		in   string
		want schema.Gender
	}{
		{"男", schema.Male}, {"male", schema.Male}, {"MALE", schema.Male}, {"m", schema.Male},
		{" Male ", schema.Male},
		{"女", schema.Female}, {"female", schema.Female}, {"FEMALE", schema.Female}, {"f", schema.Female},
		{"中性", schema.Neutral}, {"neutral", schema.Neutral}, {"Neutral", schema.Neutral},
		{"n", schema.Neutral}, {"none", schema.Neutral},
		{"", schema.Neutral}, // 空值归一为中性（ADR-014 行为变更）
		{"未知值", schema.Neutral}, {"male ", schema.Male},
	}
	for _, c := range cases {
		if got := Gender(c.in); got != c.want {
			t.Errorf("Gender(%q) 期望 %q，实得 %q", c.in, c.want, got)
		}
	}
}

func TestLevel_归一(t *testing.T) {
	for _, v := range []int{1, 2, 3} {
		if got := Level(v); got != v {
			t.Errorf("Level(%d) 应原样保留，实得 %d", v, got)
		}
	}
	for _, v := range []int{0, 4, -1, 99} {
		if got := Level(v); got != 0 {
			t.Errorf("Level(%d) 期望归零（表外），实得 %d", v, got)
		}
	}
}

func TestTone_归一(t *testing.T) {
	for v := 0; v <= 4; v++ {
		if got := Tone(v); got != v {
			t.Errorf("Tone(%d) 应原样保留，实得 %d", v, got)
		}
	}
	for _, v := range []int{-1, 5, 100} {
		if got := Tone(v); got != 0 {
			t.Errorf("Tone(%d) 期望归零，实得 %d", v, got)
		}
	}
}

func TestTier_归一(t *testing.T) {
	for v := 1; v <= 5; v++ {
		if got := Tier(v); got != v {
			t.Errorf("Tier(%d) 应原样保留，实得 %d", v, got)
		}
	}
	for _, v := range []int{0, 6, -1} {
		if got := Tier(v); got != 0 {
			t.Errorf("Tier(%d) 期望归零，实得 %d", v, got)
		}
	}
}

func TestScore_截断(t *testing.T) {
	for _, v := range []int{0, 50, 100} {
		if got := Score(v); got != v {
			t.Errorf("Score(%d) 应原样保留，实得 %d", v, got)
		}
	}
	if got := Score(-5); got != 0 {
		t.Errorf("Score(-5) 期望 0，实得 %d", got)
	}
	if got := Score(150); got != 100 {
		t.Errorf("Score(150) 期望截断到 100，实得 %d", got)
	}
}

func TestClamp(t *testing.T) {
	if got := Clamp(5, 0, 10); got != 5 {
		t.Errorf("Clamp(5,0,10) 期望 5，实得 %d", got)
	}
	if got := Clamp(-3, 0, 10); got != 0 {
		t.Errorf("Clamp(-3,0,10) 期望 0，实得 %d", got)
	}
	if got := Clamp(30, 0, 10); got != 10 {
		t.Errorf("Clamp(30,0,10) 期望 10，实得 %d", got)
	}
	// min>max 时返回 min（防御性契约，调用方传错区间时不静默产出区间外值）
	if got := Clamp(5, 8, 2); got != 8 {
		t.Errorf("Clamp(5,8,2) 期望 8，实得 %d", got)
	}
}

func TestMeaning_截断按Rune(t *testing.T) {
	if got := Meaning("  流水  "); got != "流水" {
		t.Errorf("Meaning 应 trim，实得 %q", got)
	}
	// 恰好等于上限不动
	exact := strings.Repeat("释", meaningMaxRunes)
	if got := Meaning(exact); got != exact {
		t.Errorf("恰好 %d rune 的释义不应被截断", meaningMaxRunes)
	}
	// 超出一个 rune 即截断（按 rune 不按字节，中文三字节一字）
	over := strings.Repeat("释", meaningMaxRunes+1)
	got := Meaning(over)
	if n := len([]rune(got)); n != meaningMaxRunes {
		t.Fatalf("超长释义应截断到 %d rune，实得 %d", meaningMaxRunes, n)
	}
}

func TestStroke_负数归零(t *testing.T) {
	if got := Stroke(5); got != 5 {
		t.Errorf("Stroke(5) 期望 5，实得 %d", got)
	}
	if got := Stroke(-1); got != 0 {
		t.Errorf("Stroke(-1) 期望 0，实得 %d", got)
	}
}

func TestTags_委托受控归一(t *testing.T) {
	got := Tags([]string{"b", "a", "a", ""})
	want := []string{"a", "b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Tags 期望 %v，实得 %v", want, got)
	}
}

func TestWuxing_受控词表(t *testing.T) {
	if got := Wuxing(" 水 "); got != "水" {
		t.Errorf("Wuxing 应 trim 后保留合法值，实得 %q", got)
	}
	if got := Wuxing("气"); got != "" {
		t.Errorf("Wuxing 对非法值应清空（不臆造），实得 %q", got)
	}
	if got := Wuxing(""); got != "" {
		t.Errorf("Wuxing 空值应保持空，实得 %q", got)
	}
}

// validWord 构造一个全字段合法的规范字，作为各用例的基座
func validWord() schema.Word {
	return schema.Word{
		Char: "永", Pinyin: "yong3", Strokes: 5, KangxiStrokes: 5,
		Radical: "水", Meaning: "水流长", Wuxing: "水",
		Level: 1, Gender: schema.Male, Tone: 3, NameFreqTier: 4, UsageLevel: 3,
		PositiveScore: 90, PhoneticScore: 80, ModernScore: 70, ClassicalScore: 95,
		CurationLevel: 2, NamePenalty: 5,
		IsPolyphonic: false, IsRare: false, IsNegative: false,
		StyleTags: []string{"古典"}, GenderTags: []string{"male"},
		PairBlacklist: []string{}, NamingCats: []string{"品德"},
	}
}

func TestWord_全字段归一(t *testing.T) {
	w := validWord()
	// 注入各类越界/脏值
	w.Pinyin = "  yong3 "
	w.Radical = " 水 "
	w.Meaning = "  释义  "
	w.Wuxing = "气"         // 非法五行 → 清空
	w.Level = 9            // → 0
	w.Tone = 9             // → 0
	w.NameFreqTier = 9     // → 0
	w.UsageLevel = 9       // → 5（截断）
	w.CurationLevel = -1   // → 0
	w.PositiveScore = 150  // → 100
	w.PhoneticScore = -1   // → 0
	w.ModernScore = 200    // → 100
	w.ClassicalScore = 101 // → 100
	w.NamePenalty = 101    // → 100
	w.Strokes = -1         // → 0
	w.KangxiStrokes = -1   // → 0
	w.Gender = "男"         // → male
	w.StyleTags = []string{"b", "a", "a"}
	w.PairBlacklist = []string{"x", "x"}
	w.NamingCats = []string{"", " 品德 "}

	got := Word(w)

	if got.Pinyin != "yong3" || got.Radical != "水" || got.Meaning != "释义" {
		t.Errorf("字符串字段未 trim：%q/%q/%q", got.Pinyin, got.Radical, got.Meaning)
	}
	if got.Wuxing != "" {
		t.Errorf("非法五行应清空，实得 %q", got.Wuxing)
	}
	if got.Level != 0 || got.Tone != 0 || got.NameFreqTier != 0 {
		t.Errorf("越界等级/声调/档位应归零：%d/%d/%d", got.Level, got.Tone, got.NameFreqTier)
	}
	if got.UsageLevel != 5 {
		t.Errorf("UsageLevel 应截断到 5，实得 %d", got.UsageLevel)
	}
	if got.CurationLevel != 0 {
		t.Errorf("CurationLevel 负数应归零，实得 %d", got.CurationLevel)
	}
	if got.PositiveScore != 100 || got.PhoneticScore != 0 || got.ModernScore != 100 ||
		got.ClassicalScore != 100 || got.NamePenalty != 100 {
		t.Errorf("评分截断不符：%d/%d/%d/%d/%d", got.PositiveScore, got.PhoneticScore,
			got.ModernScore, got.ClassicalScore, got.NamePenalty)
	}
	if got.Strokes != 0 || got.KangxiStrokes != 0 {
		t.Errorf("笔画负数应归零：%d/%d", got.Strokes, got.KangxiStrokes)
	}
	if got.Gender != schema.Male {
		t.Errorf("中文性别应归一为 male，实得 %q", got.Gender)
	}
	if !reflect.DeepEqual(got.StyleTags, []string{"a", "b"}) {
		t.Errorf("StyleTags 归一异常：%v", got.StyleTags)
	}
	if !reflect.DeepEqual(got.PairBlacklist, []string{"x"}) {
		t.Errorf("PairBlacklist 归一异常：%v", got.PairBlacklist)
	}
	if !reflect.DeepEqual(got.NamingCats, []string{"品德"}) {
		t.Errorf("NamingCats 归一异常：%v", got.NamingCats)
	}
	// 布尔与未涉及字段原样保留（基座三布尔均为 false，归一不得翻转）
	if got.IsPolyphonic || got.IsRare || got.IsNegative {
		t.Error("布尔字段不应被归一改动")
	}
}

func TestWord_幂等(t *testing.T) {
	w := validWord()
	w.Wuxing = "气"
	w.StyleTags = []string{"b", "a", "a"}
	once := Word(w)
	twice := Word(once)
	if !reflect.DeepEqual(once, twice) {
		t.Fatalf("Word 归一不幂等：\n一次=%+v\n二次=%+v", once, twice)
	}
}

func TestWord_合法值原样保留(t *testing.T) {
	w := validWord()
	got := Word(w)
	if !reflect.DeepEqual(got, w) {
		t.Fatalf("已合法的字不应被改动：\n期望=%+v\n实得=%+v", w, got)
	}
}

func TestCharGroup_归一(t *testing.T) {
	g := schema.CharGroup{Radical: " 水 ", Name: " 水部 ", Meaning: "  与水相关  ", Chars: []string{"永", "永", ""}}
	got := CharGroup(g)
	if got.Radical != "水" || got.Name != "水部" || got.Meaning != "与水相关" {
		t.Errorf("CharGroup 字符串未 trim：%+v", got)
	}
	if !reflect.DeepEqual(got.Chars, []string{"永"}) {
		t.Errorf("CharGroup.Chars 归一异常：%v", got.Chars)
	}
}

func TestBigramFreq_归一(t *testing.T) {
	b := schema.BigramFreq{Char1: " 司 ", Char2: " 马 ", Count: -3, Rank: -1, Tier: 9}
	got := BigramFreq(b)
	if got.Char1 != "司" || got.Char2 != "马" {
		t.Errorf("双字未 trim：%q/%q", got.Char1, got.Char2)
	}
	if got.Count != 0 || got.Rank != 0 || got.Tier != 0 {
		t.Errorf("负值/越界未归零：%d/%d/%d", got.Count, got.Rank, got.Tier)
	}
}

func TestCharFreq_归一(t *testing.T) {
	c := schema.CharFreq{Char: " 永 ", Count: -2, Rank: 7, Tier: 3}
	got := CharFreq(c)
	if got.Char != "永" {
		t.Errorf("单字未 trim：%q", got.Char)
	}
	if got.Count != 0 {
		t.Errorf("负频次应归零，实得 %d", got.Count)
	}
	if got.Rank != 7 || got.Tier != 3 {
		t.Errorf("合法值应保留：%d/%d", got.Rank, got.Tier)
	}
}
