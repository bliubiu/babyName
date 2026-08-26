package fate

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// --- NameResult.Meaning 回填测试（与 engine_pinyin_test.go 同一套桩结构） ---
//
// 缺陷背景：TopNames 组装处从未赋值 Meaning，HTTP 层 empty_meaning=50/50。
// 本文件固化三个契约：
//  1. 单名释义必须回填自候选字数据
//  2. 双名释义应同时包含两字的释义内容
//  3. 超长字典释义必须有截断上限（hanzi.json 的说文解字段可达数百 rune）

// longTestMeaning 模拟字典中的超长释义（《说文》类条目可达数百字）
var longTestMeaning = strings.Repeat("光明照耀，", 40) // 200 runes

// newMeaningTestEngine 构造带释义的候选字测试引擎
func newMeaningTestEngine() Fate {
	provider := &stubProvider{chars: []*Character{
		{Char: "泽", Pinyin: []string{"zé"}, WuXing: "水", SimplifiedStroke: 8, IsRegular: true, IsNameable: true, CommonLevel: 1, Meaning: "水积聚的地方，恩泽"},
		{Char: "宇", Pinyin: []string{"yǔ"}, WuXing: "土", SimplifiedStroke: 6, IsRegular: true, IsNameable: true, CommonLevel: 1, Meaning: "屋檐，泛指房屋，气宇轩昂"},
		{Char: "明", Pinyin: []string{"míng"}, WuXing: "火", SimplifiedStroke: 8, IsRegular: true, IsNameable: true, CommonLevel: 1, Meaning: longTestMeaning},
		{Char: "轩", Pinyin: []string{"xuān"}, WuXing: "土", SimplifiedStroke: 7, IsRegular: true, IsNameable: true, CommonLevel: 1, Meaning: "高大；气度不凡"},
	}}
	return NewEngine(provider, &stubAnalyzer{}, DefaultRaters())
}

// meaningPrefixes 各候选字释义在组合结果中至少应保留的前缀（长释义取前 50 rune，
// 与实现的每字 60 rune 截断上限保持兼容）
func meaningPrefixes() map[string]string {
	mingPrefix := string([]rune(longTestMeaning)[:50])
	return map[string]string{
		"泽": "水积聚的地方",
		"宇": "屋檐",
		"明": mingPrefix,
		"轩": "气度不凡",
	}
}

// TestSingleNameMeaningFilled 单名结果的释义必须回填自候选字数据
func TestSingleNameMeaningFilled(t *testing.T) {
	output := runSession(t, newMeaningTestEngine(), 1)

	if len(output.TopNames) == 0 {
		t.Fatal("应至少生成一个候选")
	}
	for _, nr := range output.TopNames {
		if strings.TrimSpace(nr.Meaning) == "" {
			t.Errorf("名字 %s 的释义为空，应回填候选字释义", nr.FullName)
		}
	}
}

// TestSingleNameMeaningMatchesChar 单名释义应来自该名用字本身
func TestSingleNameMeaningMatchesChar(t *testing.T) {
	prefixes := meaningPrefixes()
	output := runSession(t, newMeaningTestEngine(), 1)

	for _, nr := range output.TopNames {
		runes := []rune(nr.GivenName)
		if len(runes) != 1 {
			t.Errorf("单名用例出现多字名 %q", nr.GivenName)
			continue
		}
		want, ok := prefixes[string(runes[0])]
		if !ok {
			t.Fatalf("测试数据缺少字 %q 的释义前缀", string(runes[0]))
		}
		if !strings.Contains(nr.Meaning, want) {
			t.Errorf("单名 %s 释义 %q 未包含字 %q 的释义 %q", nr.FullName, nr.Meaning, string(runes[0]), want)
		}
	}
}

// TestDoubleNameMeaningContainsBothChars 双名释义应同时包含两字的释义内容
func TestDoubleNameMeaningContainsBothChars(t *testing.T) {
	prefixes := meaningPrefixes()
	output := runSession(t, newMeaningTestEngine(), 2)

	if len(output.TopNames) == 0 {
		t.Fatal("应至少生成一个候选")
	}
	for _, nr := range output.TopNames {
		runes := []rune(nr.GivenName)
		if len(runes) != 2 {
			continue
		}
		for _, ch := range runes {
			want, ok := prefixes[string(ch)]
			if !ok {
				t.Fatalf("测试数据缺少字 %q 的释义前缀", string(ch))
			}
			if !strings.Contains(nr.Meaning, want) {
				t.Errorf("双名 %s 释义未包含字 %q 的释义 %q（实际释义开头: %q）",
					nr.FullName, string(ch), want, truncateForLog(nr.Meaning))
			}
		}
	}
}

// TestMeaningLengthCapped 释义组合长度应有上限，防止超长字典释义撑爆响应与前端展示
func TestMeaningLengthCapped(t *testing.T) {
	// 契约：每字释义截断至 60 rune（+省略号），双名以「；」连接 → 总长 ≤ 130
	const meaningCap = 130
	output := runSession(t, newMeaningTestEngine(), 2)

	for _, nr := range output.TopNames {
		if n := utf8.RuneCountInString(nr.Meaning); n > meaningCap {
			t.Errorf("双名 %s 释义长度 %d 超过上限 %d（开头: %q）",
				nr.FullName, n, meaningCap, truncateForLog(nr.Meaning))
		}
	}
}

// truncateForLog 日志输出用的安全截断
func truncateForLog(s string) string {
	r := []rune(s)
	if len(r) <= 30 {
		return s
	}
	return string(r[:30]) + "…"
}
