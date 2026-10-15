package hanzi

import (
	"reflect"
	"testing"
)

// 本文件守护 docs/31 ADR-012：NamerChar → Hanzi 是唯一映射真源，且不丢字段。
//
// 修订前 namer_loader.go 内联构造 Hanzi 字面量，漏掉 Level / NameFreqTier / IsPolyphonic，
// 其中 IsPolyphonic 在 namer.json 里本有 1279 条真值却被结构体静默丢弃。

func TestToHanzi_字段完整映射(t *testing.T) {
	nc := NamerChar{
		Char: "行", Pinyin: "xing2", Strokes: 6, KangxiStrokes: 16,
		Radical: "彳", Meaning: "道路", Wuxing: "水", Level: 1,
		Gender: "male", Tone: 2, StyleTags: []string{"古典"}, UsageLevel: 5,
		PositiveScore: 90, IsRare: true, IsNegative: true, IsPolyphonic: true,
		NamePenalty: 6, NameFreqTier: 3, NamingCategories: []string{"品德"},
	}

	h := ToHanzi(nc)

	checks := []struct {
		name string
		got  any
		want any
	}{
		{"Char", h.Char, "行"}, {"Pinyin", h.Pinyin, "xing2"},
		{"Strokes", h.Strokes, 6}, {"KangxiStrokes", h.KangxiStrokes, 16},
		{"Radical", h.Radical, "彳"}, {"Meaning", h.Meaning, "道路"},
		{"Wuxing", h.Wuxing, "水"}, {"Gender", h.Gender, "male"},
		{"Level", h.Level, 1}, {"Tone", h.Tone, 2},
		{"UsageLevel", h.UsageLevel, 5}, {"PositiveScore", h.PositiveScore, 90},
		{"IsRare", h.IsRare, true}, {"IsNegative", h.IsNegative, true},
		{"IsPolyphonic", h.IsPolyphonic, true}, {"NamePenalty", h.NamePenalty, 6},
		{"NameFreqTier", h.NameFreqTier, 3},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s 期望 %v，实得 %v", c.name, c.want, c.got)
		}
	}

	if !reflect.DeepEqual(h.StyleTags, nc.StyleTags) {
		t.Errorf("StyleTags 期望 %v，实得 %v", nc.StyleTags, h.StyleTags)
	}
	if !reflect.DeepEqual(h.NamingCategories, nc.NamingCategories) {
		t.Errorf("NamingCategories 期望 %v，实得 %v", nc.NamingCategories, h.NamingCategories)
	}
}

// TestToHanzi_性别归一 由 mapper 统一承担，消除各 loader 各自写一份枚举映射
func TestToHanzi_性别归一(t *testing.T) {
	cases := []struct{ in, want string }{
		{"male", "male"}, {"female", "female"}, {"neutral", "neutral"},
	}
	for _, c := range cases {
		if got := ToHanzi(NamerChar{Char: "字", Gender: c.in}).Gender; got != c.want {
			t.Errorf("性别 %q 期望 %q，实得 %q", c.in, c.want, got)
		}
	}
}

// TestToHanzi_无真值字段保持零值 namer.json 不提供这些字段，
// 必须保持零值而非用其他字段伪造（docs/31 §9.9）
func TestToHanzi_无真值字段保持零值(t *testing.T) {
	h := ToHanzi(NamerChar{Char: "字", Pinyin: "zi4", PositiveScore: 88})

	if h.PhoneticScore != 0 {
		t.Errorf("phoneticScore 应为 0（物料无此值），实得 %d", h.PhoneticScore)
	}
	if h.ModernScore != 0 {
		t.Errorf("modernScore 应为 0（物料无此值），实得 %d", h.ModernScore)
	}
	if h.ClassicalScore != 0 {
		t.Errorf("classicalScore 应为 0（物料无此值），实得 %d", h.ClassicalScore)
	}
	if h.CurationLevel != 0 {
		t.Errorf("curationLevel 应为 0（物料无此值），实得 %d", h.CurationLevel)
	}
	if len(h.GenderTags) != 0 {
		t.Errorf("genderTags 应为空（物料无此值），实得 %v", h.GenderTags)
	}
	if len(h.PairBlacklist) != 0 {
		t.Errorf("pairBlacklist 应为空（物料无此值），实得 %v", h.PairBlacklist)
	}
}

func TestToHanziFromJSON_字段完整映射(t *testing.T) {
	jh := JSONHanzi{
		Char: "行", Pinyin: "xing2", Strokes: 6, Radical: "彳", Meaning: "道路",
		Gender: "female", Tone: 2, GenderTags: []string{"male"},
		StyleTags: []string{"古典"}, UsageLevel: 5, PositiveScore: 90,
		PhoneticScore: 80, ModernScore: 70, ClassicalScore: 95,
		IsPolyphonic: true, IsRare: true, IsNegative: true,
		PairBlacklist: []string{"组合甲"}, CurationLevel: 4, NamePenalty: 6,
	}
	h := ToHanziFromJSON(jh, "水", []string{"品德"})

	if h.Wuxing != "水" {
		t.Errorf("wuxing 期望 水，实得 %s", h.Wuxing)
	}
	if h.Gender != "female" {
		t.Errorf("gender 期望 female，实得 %s", h.Gender)
	}
	if h.PhoneticScore != 80 || h.ModernScore != 70 || h.ClassicalScore != 95 {
		t.Errorf("三项评分未透传：%d/%d/%d", h.PhoneticScore, h.ModernScore, h.ClassicalScore)
	}
	if !h.IsPolyphonic || !h.IsRare || !h.IsNegative {
		t.Error("布尔标记未透传")
	}
	if h.CurationLevel != 4 || h.NamePenalty != 6 {
		t.Errorf("策展等级/扣分未透传：%d/%d", h.CurationLevel, h.NamePenalty)
	}
	if !reflect.DeepEqual(h.GenderTags, jh.GenderTags) {
		t.Errorf("genderTags 未透传：%v", h.GenderTags)
	}
	if !reflect.DeepEqual(h.PairBlacklist, jh.PairBlacklist) {
		t.Errorf("pairBlacklist 未透传：%v", h.PairBlacklist)
	}
	if !reflect.DeepEqual(h.NamingCategories, []string{"品德"}) {
		t.Errorf("namingCategories 期望 [品德]，实得 %v", h.NamingCategories)
	}
}

// TestHanzi含Level字段 防止有人为「少投影一个字段」而删掉 Level
func TestHanzi含Level字段(t *testing.T) {
	tp := reflect.TypeOf(Hanzi{})
	if _, ok := tp.FieldByName("Level"); !ok {
		t.Fatal("Hanzi 缺少 Level 字段：SQLite 侧无法按字表等级过滤")
	}
}

// TestNamerChar含IsPolyphonic字段 守护 namer.json 自带的 1279 条多音字真值不被丢弃
func TestNamerChar含IsPolyphonic字段(t *testing.T) {
	tp := reflect.TypeOf(NamerChar{})
	f, ok := tp.FieldByName("IsPolyphonic")
	if !ok {
		t.Fatal("NamerChar 缺少 IsPolyphonic 字段：namer.json 的多音字标记会被静默丢弃")
	}
	if got := f.Tag.Get("json"); got != "isPolyphonic,omitempty" {
		t.Errorf("IsPolyphonic 的 json tag 期望 isPolyphonic,omitempty，实得 %s", got)
	}
}
