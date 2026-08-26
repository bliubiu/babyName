package fate

import (
	"context"
	"strings"
	"testing"
	"time"
)

// --- 内存版测试桩：避免依赖 services 层适配器（fate 包不可反向 import services） ---

// stubProvider 内存汉字数据源
type stubProvider struct {
	chars []*Character
}

func (s *stubProvider) GetCharacter(char string) (*Character, error) {
	for _, c := range s.chars {
		if c.Char == char {
			return c, nil
		}
	}
	return nil, nil
}

func (s *stubProvider) FindCharacters(query CharacterQuery) ([]*Character, error) {
	return s.chars, nil
}

func (s *stubProvider) GetSurnameStrokes(surname string) (int, int, error) {
	return len([]rune(surname)), 0, nil
}

func (s *stubProvider) CountCharacters(query CharacterQuery) (int, error) {
	return len(s.chars), nil
}

// stubAnalyzer 固定八字分析结果
type stubAnalyzer struct{}

func (a *stubAnalyzer) Analyze(born time.Time, gender Gender) (*FateData, error) {
	return &FateData{
		BaziInfo: BaziInfo{
			FourPillars: [4]string{"癸卯", "乙丑", "戊子", "戊午"},
			Zodiac:      "兔",
		},
		WuXingXiji: WuXingXiji{
			XiYongShen: []string{"水"},
			RiZhu:      "戊",
		},
	}, nil
}

// newPinyinTestEngine 构造带 4 个带声调拼音候选字的测试引擎
func newPinyinTestEngine() Fate {
	provider := &stubProvider{chars: []*Character{
		{Char: "泽", Pinyin: []string{"zé"}, WuXing: "水", SimplifiedStroke: 8, IsRegular: true, IsNameable: true, CommonLevel: 1},
		{Char: "宇", Pinyin: []string{"yǔ"}, WuXing: "土", SimplifiedStroke: 6, IsRegular: true, IsNameable: true, CommonLevel: 1},
		{Char: "明", Pinyin: []string{"míng"}, WuXing: "火", SimplifiedStroke: 8, IsRegular: true, IsNameable: true, CommonLevel: 1},
		{Char: "轩", Pinyin: []string{"xuān"}, WuXing: "土", SimplifiedStroke: 7, IsRegular: true, IsNameable: true, CommonLevel: 1},
	}}
	return NewEngine(provider, &stubAnalyzer{}, DefaultRaters())
}

// runSession 执行一次生成并返回输出
func runSession(t *testing.T, engine Fate, nameLength int) *Output {
	t.Helper()
	session := engine.NewSession()
	err := session.Start(context.Background(), &Input{
		Surname: "李",
		Gender:  GenderMale,
		Born:    time.Date(2023, 8, 20, 10, 0, 0, 0, time.UTC),
		Options: GenerateOptions{NameLength: nameLength, Count: 10},
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

// TestSingleNamePinyinFilled 单名结果的拼音必须回填自候选字数据
func TestSingleNamePinyinFilled(t *testing.T) {
	output := runSession(t, newPinyinTestEngine(), 1)

	if len(output.TopNames) == 0 {
		t.Fatal("应至少生成一个候选")
	}
	for _, nr := range output.TopNames {
		if strings.TrimSpace(nr.Pinyin) == "" {
			t.Errorf("名字 %s 的拼音为空，应回填候选字拼音", nr.FullName)
		}
	}
}

// TestDoubleNamePinyinCombined 双名结果的拼音应为两字拼音以空格组合
func TestDoubleNamePinyinCombined(t *testing.T) {
	output := runSession(t, newPinyinTestEngine(), 2)

	if len(output.TopNames) == 0 {
		t.Fatal("应至少生成一个候选")
	}

	validPinyin := map[string]bool{"zé": true, "yǔ": true, "míng": true, "xuān": true}
	for _, nr := range output.TopNames {
		parts := strings.Fields(nr.Pinyin)
		if len(parts) != 2 {
			t.Errorf("双名 %s 拼音应为两个字读音的组合，实际: %q", nr.FullName, nr.Pinyin)
			continue
		}
		for _, p := range parts {
			if !validPinyin[p] {
				t.Errorf("双名 %s 拼音含非法音节 %q（完整拼音 %q）", nr.FullName, p, nr.Pinyin)
			}
		}
	}
}

// TestSingleNamePinyinNoTrailingSpace 单名拼音不应残留尾随空格
func TestSingleNamePinyinNoTrailingSpace(t *testing.T) {
	output := runSession(t, newPinyinTestEngine(), 1)

	for _, nr := range output.TopNames {
		if nr.Pinyin != strings.TrimSpace(nr.Pinyin) {
			t.Errorf("单名 %s 拼音含首尾空白: %q", nr.FullName, nr.Pinyin)
		}
		if strings.Contains(nr.Pinyin, "  ") {
			t.Errorf("单名 %s 拼音含连续空格: %q", nr.FullName, nr.Pinyin)
		}
	}
}

// --- 诗词字注入测试 ---

// 扩大桩字池至 8 字（五行混合），确保枚举有足够候选且五行收窄触发降级保护（<80）
func newExpandedStubProvider() *stubProvider {
	return &stubProvider{chars: []*Character{
		{Char: "泽", Pinyin: []string{"zé"}, WuXing: "水", SimplifiedStroke: 8, IsRegular: true, IsNameable: true, CommonLevel: 1},
		{Char: "宇", Pinyin: []string{"yǔ"}, WuXing: "土", SimplifiedStroke: 6, IsRegular: true, IsNameable: true, CommonLevel: 1},
		{Char: "明", Pinyin: []string{"míng"}, WuXing: "火", SimplifiedStroke: 8, IsRegular: true, IsNameable: true, CommonLevel: 1},
		{Char: "轩", Pinyin: []string{"xuān"}, WuXing: "土", SimplifiedStroke: 7, IsRegular: true, IsNameable: true, CommonLevel: 1},
		{Char: "涵", Pinyin: []string{"hán"}, WuXing: "水", SimplifiedStroke: 11, IsRegular: true, IsNameable: true, CommonLevel: 1},
		{Char: "瑞", Pinyin: []string{"ruì"}, WuXing: "金", SimplifiedStroke: 13, IsRegular: true, IsNameable: true, CommonLevel: 1},
		{Char: "哲", Pinyin: []string{"zhé"}, WuXing: "火", SimplifiedStroke: 10, IsRegular: true, IsNameable: true, CommonLevel: 1},
		{Char: "辰", Pinyin: []string{"chén"}, WuXing: "土", SimplifiedStroke: 7, IsRegular: true, IsNameable: true, CommonLevel: 1},
	}}
}

// TestExtraCharsInjectIntoPool 通过 GenerateOptions.ExtraChars 注入的诗词字必须出现在最终候选中
func TestExtraCharsInjectIntoPool(t *testing.T) {
	provider := newExpandedStubProvider()
	engine := NewEngine(provider, &stubAnalyzer{}, DefaultRaters())

	injectedChar := &Character{
		Char:             "诗",
		Pinyin:           []string{"shī"},
		WuXing:           "金", // 不在喜用神"水"中，但字池<80不触发收窄
		SimplifiedStroke: 8,
		IsRegular:        true,
		IsNameable:       true,
		CommonLevel:      1,
	}

	session := engine.NewSession()
	err := session.Start(context.Background(), &Input{
		Surname: "王",
		Gender:  GenderMale,
		Born:    time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
		Options: GenerateOptions{
			NameLength: 2,
			Count:      50,
			ExtraChars: []*Character{injectedChar},
		},
	})
	if err != nil {
		t.Fatalf("会话启动失败: %v", err)
	}
	if err := session.Wait(); err != nil {
		t.Fatalf("生成失败: %v", err)
	}

	output := session.Result()
	if len(output.TopNames) == 0 {
		t.Fatal("应至少生成一个候选")
	}

	found := false
	for _, nr := range output.TopNames {
		if strings.Contains(nr.GivenName, "诗") {
			found = true
			break
		}
	}
	if !found {
		t.Error("注入的诗词字「诗」未出现在任何候选名中，ExtraChars 未被消费")
	}
}

// TestExtraCharsWordPinyinIncluded 注入字的拼音必须出现在对应名字的 Pinyin 字段中
func TestExtraCharsWordPinyinIncluded(t *testing.T) {
	provider := newExpandedStubProvider()
	engine := NewEngine(provider, &stubAnalyzer{}, DefaultRaters())

	injectedChar := &Character{
		Char:             "悦",
		Pinyin:           []string{"yuè"},
		WuXing:           "金",
		SimplifiedStroke: 10,
		IsRegular:        true,
		IsNameable:       true,
		CommonLevel:      1,
	}

	session := engine.NewSession()
	err := session.Start(context.Background(), &Input{
		Surname: "李",
		Gender:  GenderFemale,
		Born:    time.Date(2023, 8, 20, 10, 0, 0, 0, time.UTC),
		Options: GenerateOptions{
			NameLength: 2,
			Count:      5,
			ExtraChars: []*Character{injectedChar},
		},
	})
	if err != nil {
		t.Fatalf("会话启动失败: %v", err)
	}
	if err := session.Wait(); err != nil {
		t.Fatalf("生成失败: %v", err)
	}

	output := session.Result()
	if len(output.TopNames) == 0 {
		t.Fatal("应至少生成一个候选")
	}

	for _, nr := range output.TopNames {
		if strings.Contains(nr.GivenName, "悦") && !strings.Contains(nr.Pinyin, "yuè") {
			t.Errorf("含注入字「悦」的名字 %q 拼音缺少 yuè，实际: %q", nr.FullName, nr.Pinyin)
		}
	}
}
