package fate

// engine_preference_test.go — 用户显式偏好生效（docs/29 A9）
//
// A9 的原始病灶：GenerateOptions.MeaningKeywords 一路从 HTTP 层传到 engine，
// 但引擎内没有任何读取点 —— 用户输入关键词与完全不输入的结果一模一样。
// 前端更把「按偏旁选字」点选的字拼成 "包含字：木木" 混进 keywords 文本发送，
// 结构化通道 RequiredChars 从未被消费，而 UI 明确承诺「优先从此来源选字」。
//
// 修法：把偏好落到候选池上（收窄而非打分，因为 RateNameScore 权重和恒为 1.0，
// 加维度会把偏好变成隐性加分并破坏分数语义）。
//
// 本文件固化契约：
//  1. narrowPoolByPreference 的收窄判据（字本身/字义/起名分类）与空集回退；
//  2. 端到端：关键词与 RequiredChars 必须真实改变 TopNames 的用字；
//  3. 不传偏好时行为与修复前完全一致（不回归）。

import (
	"context"
	"strings"
	"testing"
	"time"
)

// 偏好测试用候选池：刻意混入多个分类/字义维度，便于验证三类命中判据
func preferenceTestChars() []*Character {
	return []*Character{
		{Char: "泽", Pinyin: []string{"zé"}, WuXing: "水", SimplifiedStroke: 8, IsRegular: true, IsNameable: true, CommonLevel: 1, Meaning: "水积聚的地方，恩泽"},
		{Char: "宇", Pinyin: []string{"yǔ"}, WuXing: "土", SimplifiedStroke: 6, IsRegular: true, IsNameable: true, CommonLevel: 1, Meaning: "屋檐，气宇轩昂"},
		{Char: "轩", Pinyin: []string{"xuān"}, WuXing: "土", SimplifiedStroke: 7, IsRegular: true, IsNameable: true, CommonLevel: 1, Meaning: "高大；气度不凡"},
		{Char: "明", Pinyin: []string{"míng"}, WuXing: "火", SimplifiedStroke: 8, IsRegular: true, IsNameable: true, CommonLevel: 1, Meaning: "明亮，光明"},
		// 靠「起名分类」命中，而非字义
		{Char: "川", Pinyin: []string{"chuān"}, WuXing: "水", SimplifiedStroke: 3, IsRegular: true, IsNameable: true, CommonLevel: 1, Meaning: "河流", NamingCategory: []string{"山水"}},
		{Char: "岳", Pinyin: []string{"yuè"}, WuXing: "土", SimplifiedStroke: 8, IsRegular: true, IsNameable: true, CommonLevel: 1, Meaning: "高大的山", NamingCategory: []string{"山水"}},
		// 偏好集之外的干扰字
		{Char: "翰", Pinyin: []string{"hàn"}, WuXing: "水", SimplifiedStroke: 16, IsRegular: true, IsNameable: true, CommonLevel: 1, Meaning: "文辞，书写"},
		{Char: "墨", Pinyin: []string{"mò"}, WuXing: "水", SimplifiedStroke: 15, IsRegular: true, IsNameable: true, CommonLevel: 1, Meaning: "书写用的黑色颜料"},
	}
}

func newPreferenceTestEngine() Fate {
	provider := &stubProvider{chars: preferenceTestChars()}
	return NewEngine(provider, &stubAnalyzer{}, DefaultRaters())
}

func charSetOf(chars []*Character) map[string]bool {
	set := make(map[string]bool, len(chars))
	for _, c := range chars {
		set[c.Char] = true
	}
	return set
}

// --- narrowPoolByPreference 单元契约 ---

// TestNarrowPoolByPreference_ThreeMatchRules 三条命中判据都必须生效
func TestNarrowPoolByPreference_ThreeMatchRules(t *testing.T) {
	chars := preferenceTestChars()

	tests := []struct {
		name     string
		keywords []string
		required []string
		want     []string
	}{
		{
			name:     "字本身等于关键词",
			keywords: []string{"泽"},
			want:     []string{"泽"},
		},
		{
			name:     "字义含关键词",
			keywords: []string{"光明"},
			want:     []string{"明"},
		},
		{
			name:     "起名分类含关键词",
			keywords: []string{"山水"},
			want:     []string{"川", "岳"},
		},
		{
			name:     "点选用字命中",
			required: []string{"翰", "墨"},
			want:     []string{"翰", "墨"},
		},
		{
			name:     "关键词与点选字取并集",
			keywords: []string{"光明"},
			required: []string{"墨"},
			want:     []string{"明", "墨"},
		},
		{
			name:     "多个关键词取并集",
			keywords: []string{"光明", "山水"},
			want:     []string{"明", "川", "岳"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := narrowPoolByPreference(chars, tc.keywords, tc.required)
			gotSet := charSetOf(got)
			if len(gotSet) != len(tc.want) {
				t.Errorf("收窄结果 = %v，期望恰好 %v", gotSet, tc.want)
			}
			for _, w := range tc.want {
				if !gotSet[w] {
					t.Errorf("字 %q 应命中 %v（keywords=%v required=%v）", w, tc.name, tc.keywords, tc.required)
				}
			}
		})
	}
}

// TestNarrowPoolByPreference_EmptyInputsKeepsPool 无偏好输入时不收窄
func TestNarrowPoolByPreference_EmptyInputsKeepsPool(t *testing.T) {
	chars := preferenceTestChars()

	cases := []struct {
		name     string
		keywords []string
		required []string
	}{
		{"全空", nil, nil},
		{"空字符串关键词", []string{"", "   "}, nil},
		{"空字符串点选", nil, []string{"", "  "}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := narrowPoolByPreference(chars, tc.keywords, tc.required); got != nil {
				t.Errorf("无有效偏好时应返回 nil（保持原池），实际收窄为 %v", charSetOf(got))
			}
		})
	}
}

// TestNarrowPoolByPreference_NoHitKeepsPool 关键词全不命中时不得清空候选池
//
// 这是最危险的一条：若返回空集，双名枚举必然零结果，用户看到的是
// 「起不出名字」而非「关键词没匹配上」。
func TestNarrowPoolByPreference_NoHitKeepsPool(t *testing.T) {
	chars := preferenceTestChars()

	got := narrowPoolByPreference(chars, []string{"量子纠缠", "赛博朋克"}, []string{"龘"})
	if got != nil {
		t.Errorf("偏好全部落空时应返回 nil 让调用方保持原池，实际 = %v", charSetOf(got))
	}
}

// TestNarrowPoolByPreference_PreservesOrder 收窄后顺序必须与原池一致
//
// 枚举顺序决定同分候选的先后，进而影响 TopN —— 顺序抖动会让用户
// 反复刷新看到不同结果。
func TestNarrowPoolByPreference_PreservesOrder(t *testing.T) {
	chars := preferenceTestChars()

	got := narrowPoolByPreference(chars, []string{"光明", "山水", "文辞"}, nil)
	var order []string
	for _, c := range got {
		order = append(order, c.Char)
	}

	// 原池顺序：泽 宇 轩 明 川 岳 翰 墨
	want := []string{"明", "川", "岳", "翰"}
	if strings.Join(order, "") != strings.Join(want, "") {
		t.Errorf("收窄顺序 = %v，期望 %v（须与原池相对顺序一致）", order, want)
	}
}

// --- 端到端：偏好必须真实影响输出 ---

func runSessionWithOptions(t *testing.T, engine Fate, opts GenerateOptions) *Output {
	t.Helper()
	session := engine.NewSession()
	err := session.Start(context.Background(), &Input{
		Surname: "李",
		Gender:  GenderMale,
		Born:    time.Date(2023, 8, 20, 10, 0, 0, 0, time.UTC),
		Options: opts,
	})
	if err != nil {
		t.Fatalf("会话启动失败: %v", err)
	}
	if err := session.Wait(); err != nil {
		t.Fatalf("名字生成失败: %v", err)
	}
	output := session.Result()
	if output == nil {
		t.Fatal("生成结果不应为 nil")
	}
	return output
}

// nameChars 收集输出中出现的所有名字用字
func nameChars(output *Output) map[string]bool {
	set := make(map[string]bool)
	for _, nr := range output.TopNames {
		for _, r := range nr.GivenName {
			set[string(r)] = true
		}
	}
	return set
}

// TestMeaningKeywords_ChangesOutput 寓意关键词必须真实改变结果用字
//
// 这是 A9 的核心契约：修复前该字段被引擎丢弃，输出与不传完全一致。
func TestMeaningKeywords_ChangesOutput(t *testing.T) {
	base := runSessionWithOptions(t, newPreferenceTestEngine(),
		GenerateOptions{NameLength: 1, Count: 10})
	withKw := runSessionWithOptions(t, newPreferenceTestEngine(),
		GenerateOptions{NameLength: 1, Count: 10, MeaningKeywords: []string{"光明"}})

	used := nameChars(withKw)
	if !used["明"] {
		t.Errorf("关键词「光明」应使结果包含「明」，实际用字 = %v", used)
	}

	// 偏好集之外的字不应再出现
	for c := range used {
		if c == "泽" || c == "宇" || c == "轩" || c == "川" || c == "岳" || c == "翰" || c == "墨" {
			t.Errorf("偏好集外的字 %q 仍出现在结果中，用字 = %v", c, used)
		}
	}

	// 无关键词时结果里不应只有「明」——否则本用例无法区分修复前后
	baseUsed := nameChars(base)
	if len(baseUsed) <= 1 {
		t.Fatalf("无关键词基线的用字应多于 1 个，实际 = %v", baseUsed)
	}
}

// TestRequiredChars_ChangesOutput 点选用字必须真实出现在结果中
func TestRequiredChars_ChangesOutput(t *testing.T) {
	output := runSessionWithOptions(t, newPreferenceTestEngine(),
		GenerateOptions{NameLength: 1, Count: 10, RequiredChars: []string{"翰", "墨"}})

	used := nameChars(output)
	for _, want := range []string{"翰", "墨"} {
		if !used[want] {
			t.Errorf("点选用字 %q 应出现在结果中，实际用字 = %v", want, used)
		}
	}
	for _, other := range []string{"泽", "宇", "轩", "明", "川", "岳"} {
		if used[other] {
			t.Errorf("未点选的字 %q 不应出现在结果中，用字 = %v", other, used)
		}
	}
}

// TestRequiredChars_NoMatchKeepsResults 点选字不存在时仍应正常出结果
//
// 用户可能通过「自定义加字」传入字库外的字；此时偏好集为空，
// 必须回退到原候选池，而不是产出零结果。
func TestRequiredChars_NoMatchKeepsResults(t *testing.T) {
	output := runSessionWithOptions(t, newPreferenceTestEngine(),
		GenerateOptions{NameLength: 1, Count: 10, RequiredChars: []string{"龘", "齾"}})

	if len(output.TopNames) == 0 {
		t.Fatal("点选字全不命中时应回退原候选池，不应零结果")
	}
}

// TestNoPreference_NoBehaviorChange 未传偏好时结果与修复前一致（防回归）
func TestNoPreference_NoBehaviorChange(t *testing.T) {
	first := runSessionWithOptions(t, newPreferenceTestEngine(),
		GenerateOptions{NameLength: 1, Count: 10})
	second := runSessionWithOptions(t, newPreferenceTestEngine(),
		GenerateOptions{NameLength: 1, Count: 10})

	if len(first.TopNames) != len(second.TopNames) {
		t.Fatalf("无偏好输入应完全可复现：两次结果数不同 %d vs %d",
			len(first.TopNames), len(second.TopNames))
	}
	for i := range first.TopNames {
		if first.TopNames[i].FullName != second.TopNames[i].FullName {
			t.Errorf("无偏好输入应完全可复现：第 %d 个名字 %q vs %q",
				i, first.TopNames[i].FullName, second.TopNames[i].FullName)
		}
	}
}

// TestPreference_Deterministic 同一偏好重复查询结果稳定
func TestPreference_Deterministic(t *testing.T) {
	opts := GenerateOptions{NameLength: 2, Count: 10, MeaningKeywords: []string{"山水"}}

	first := runSessionWithOptions(t, newPreferenceTestEngine(), opts)
	second := runSessionWithOptions(t, newPreferenceTestEngine(), opts)

	if len(first.TopNames) != len(second.TopNames) {
		t.Fatalf("相同偏好两次结果数不同：%d vs %d", len(first.TopNames), len(second.TopNames))
	}
	for i := range first.TopNames {
		if first.TopNames[i].FullName != second.TopNames[i].FullName {
			t.Errorf("相同偏好两次结果不一致：第 %d 个 %q vs %q",
				i, first.TopNames[i].FullName, second.TopNames[i].FullName)
		}
	}
}