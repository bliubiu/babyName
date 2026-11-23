package fate

import (
	"context"
	"testing"
	"time"
)

func TestNormalizePinyinInitial(t *testing.T) {
	cases := map[string]string{
		"z":    "z",
		"Z ":   "z",
		" zh ": "zh",
		"":     "",
		"1z":   "z",
	}
	for in, want := range cases {
		if got := normalizePinyinInitial(in); got != want {
			t.Errorf("normalizePinyinInitial(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFilterByPinyinInitial(t *testing.T) {
	chars := []*Character{
		{Char: "泽", Pinyin: []string{"zé"}},
		{Char: "哲", Pinyin: []string{"zhé"}},
		{Char: "明", Pinyin: []string{"míng"}},
	}
	z := filterByPinyinInitial(chars, "z")
	if len(z) != 2 || z[0].Char != "泽" || z[1].Char != "哲" {
		t.Errorf("initial=z 应命中 泽/哲，got %v", charsOf(z))
	}
	zh := filterByPinyinInitial(chars, "zh")
	if len(zh) != 1 || zh[0].Char != "哲" {
		t.Errorf("initial=zh 应只命中 哲，got %v", charsOf(zh))
	}
}

func charsOf(chars []*Character) []string {
	out := make([]string, len(chars))
	for i, c := range chars {
		out[i] = c.Char
	}
	return out
}

func newOptionTestEngine(chars []*Character) Fate {
	return NewEngine(&stubProvider{chars: chars}, &stubAnalyzer{}, DefaultRaters())
}

func runSingleName(t *testing.T, engine Fate, opts GenerateOptions) []string {
	t.Helper()
	session := engine.NewSession()
	input := &Input{
		Surname: "王",
		Gender:  GenderMale,
		Born:    time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
		Options: opts,
	}
	if err := session.Start(context.Background(), input); err != nil {
		t.Fatalf("会话启动失败: %v", err)
	}
	if err := session.Wait(); err != nil {
		t.Fatalf("生成失败: %v", err)
	}
	output := session.Result()
	if output == nil {
		t.Fatal("输出为空")
	}
	var names []string
	for _, nr := range output.TopNames {
		names = append(names, nr.GivenName)
	}
	return names
}

// TestExcludeRare_Engine 回归：开关必须按 IsRare 精确剔除，
// 不能依赖 IsRegular（一级/二级字也可能是人工标注的生僻字）。
func TestExcludeRare_Engine(t *testing.T) {
	chars := []*Character{
		{Char: "龘", Pinyin: []string{"dá"}, WuXing: "水", SimplifiedStroke: 48, KangxiStroke: 48,
			IsRegular: true, IsNameable: true, IsRare: true, CommonLevel: 1},
		{Char: "清", Pinyin: []string{"qīng"}, WuXing: "水", SimplifiedStroke: 11, KangxiStroke: 11,
			IsRegular: true, IsNameable: true, CommonLevel: 1},
	}
	names := runSingleName(t, newOptionTestEngine(chars), GenerateOptions{NameLength: 1, Count: 5, ExcludeRare: true})
	for _, n := range names {
		if n == "龘" {
			t.Fatal("ExcludeRare=true 仍出现生僻字「龘」")
		}
	}
	if len(names) == 0 {
		t.Fatal("排除后应至少保留「清」")
	}
}

// TestPinyinInitial_Engine 回归：拼音首字母必须真正过滤候选池。
func TestPinyinInitial_Engine(t *testing.T) {
	chars := []*Character{
		{Char: "泽", Pinyin: []string{"zé"}, WuXing: "水", SimplifiedStroke: 8, KangxiStroke: 8,
			IsRegular: true, IsNameable: true, CommonLevel: 1},
		{Char: "哲", Pinyin: []string{"zhé"}, WuXing: "火", SimplifiedStroke: 10, KangxiStroke: 10,
			IsRegular: true, IsNameable: true, CommonLevel: 1},
		{Char: "明", Pinyin: []string{"míng"}, WuXing: "火", SimplifiedStroke: 8, KangxiStroke: 8,
			IsRegular: true, IsNameable: true, CommonLevel: 1},
	}
	names := runSingleName(t, newOptionTestEngine(chars), GenerateOptions{NameLength: 1, Count: 5, PinyinInitial: "zh"})
	if len(names) == 0 {
		t.Fatal("initial=zh 应命中「哲」")
	}
	for _, n := range names {
		if n != "哲" {
			t.Fatalf("initial=zh 只应返回「哲」，got %v", names)
		}
	}
}
