package fate

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestCuratedPoolExcludesAbsurdChars 验证策展白名单收窄：
// 足够大的候选池（>=80，避开降级保护）中，荒谬字（PositiveScore 为空）被剔除，
// 而人工评分好字（PositiveScore>=85）保留。
//
// 背景：荒谬字（拤/饹/婊/蚂/蛞 等）在 namer.json 中 PositiveScore 为空，
// 却仍能靠音韵/生肖/五行等维度拿 80+ 高分。白名单收窄以 PositiveScore>=85
// 作为唯一推荐字源门槛，荒谬字天然排除（详见 engine.go 第 6 步注释）。
func TestCuratedPoolExcludesAbsurdChars(t *testing.T) {
	provider := newCuratedPoolProvider(0) // 荒谬字不注入 ExtraChars
	engine := NewEngine(provider, &stubAnalyzer{}, DefaultRaters())

	session := engine.NewSession()
	err := session.Start(context.Background(), &Input{
		Surname: "王",
		Gender:  GenderMale,
		Born:    time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
		Options: GenerateOptions{NameLength: 2, Count: 100},
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

	// 荒谬字：仅允许出现零次（被白名单收窄剔除）
	for _, absurd := range curatedPoolAbsurdChars {
		for _, nr := range output.TopNames {
			if strings.Contains(nr.GivenName, absurd) {
				t.Errorf("荒谬字「%s」出现于推荐名「%s」，应被策展白名单收窄剔除", absurd, nr.GivenName)
				break
			}
		}
	}
}

// TestCuratedPoolKeepsGoodChars 验证好字（PositiveScore>=85）在推荐结果中保留
func TestCuratedPoolKeepsGoodChars(t *testing.T) {
	provider := newCuratedPoolProvider(0)
	engine := NewEngine(provider, &stubAnalyzer{}, DefaultRaters())

	session := engine.NewSession()
	err := session.Start(context.Background(), &Input{
		Surname: "王",
		Gender:  GenderMale,
		Born:    time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
		Options: GenerateOptions{NameLength: 2, Count: 100},
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

	// 至少一个好字应出现在结果中（白名单未误删全部好字）
	found := false
	for _, nr := range output.TopNames {
		for _, good := range curatedPoolGoodChars {
			if strings.Contains(nr.GivenName, good) {
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		t.Fatal("推荐结果中未出现任何策展好字，白名单收窄可能过度")
	}
}

// TestCuratedPoolExtraCharsExempt 验证注入 ExtraChars 的荒谬字（入池门槛低）仍可保留，
// 不因白名单收窄被误杀——用户显式指定的字永远入池。
func TestCuratedPoolExtraCharsExempt(t *testing.T) {
	provider := newCuratedPoolProvider(1) // 成果 1 个荒谬字作为 ExtraChars 注入
	engine := NewEngine(provider, &stubAnalyzer{}, DefaultRaters())

	injected := &Character{
		Char:             "拤",
		Pinyin:           []string{"qiá"},
		WuXing:           "金",
		SimplifiedStroke: 9,
		IsRegular:        true,
		IsNameable:       true,
		CommonLevel:      2,
		PositiveScore:    0, // 荒谬字：无寓意评分
	}

	session := engine.NewSession()
	err := session.Start(context.Background(), &Input{
		Surname: "王",
		Gender:  GenderMale,
		Born:    time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
		Options: GenerateOptions{
			NameLength: 2,
			Count:      100,
			ExtraChars: []*Character{injected},
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

	// 注人的「拤」应出现在候选中（ExtraChars 豁免白名单收窄）
	found := false
	for _, nr := range output.TopNames {
		if strings.Contains(nr.GivenName, "拤") {
			found = true
			break
		}
	}
	if !found {
		t.Error("注入的 ExtraChars 字「拤」未出现在任何候选名中，白名单收窄误杀了显式注入字")
	}
}

// curatedPoolGoodChars / curatedPoolAbsurdChars 供上面的测试引用
var (
	curatedPoolGoodChars = []string{"泽", "清", "涵", "澄", "渊", "明", "瑞", "浩"}
	curatedPoolAbsurdChars = []string{"拤", "饹", "婊", "蚂", "蛞"}
)

// newCuratedPoolProvider 构造一个候选池>=80 的字桩（避开白名单/喜用神收窄的降级保护），
// 含人工评分好字（>=85）与荒谬字（PositiveScore=0），用于验证白名单收窄语义。
// 参数 extraAsAbsurd：>0 时内部不注入荒谬字（由测试另行注入 ExtraChars），否则池内直接含荒谬字。
func newCuratedPoolProvider(extraAsAbsurd int) *stubProvider {
	var chars []*Character

	// 84 个好字（PositiveScore=90，五行水/金各半）→ 收窄后 >=80，不触发降级保护
	goodWater := []string{"泽", "清", "涵", "澄", "渊", "澜", "沐", "澈", "泓", "泊",
		"洁", "溪", "润", "浦", "洛", "温", "溪", "冰", "净", "浅",
		"洋", "汪", "渔", "汝", "汐", "浚", "涔", "沁", "漾", "淳",
		"渝", "淮", "渊", "浚", "濠", "瀚", "潜", "潇", "灏", "鸿",
		"逵", "涯", "澹", "泱", "泓", "湉", "沁", "涓", "流", "汇"}
	goodMetal := []string{"瑞", "铭", "锋", "锐", "锦", "铭", "钧", "铎", "鉴", "银",
		"铄", "镇", "钟", "钺", "铉", "铸", "钢", "钦", "锋", "鑫",
		"瑾", "瑜", "铮", "琢", "璎", "璐", "珅", "珩", "珂", "玙",
		"环", "珉", "瑶", "玦", "钰", "珺", "玷", "玑", "珑", "珥"}
	idx := 0
	appendGood := func(set []string, wx string) {
		for _, ch := range set {
			idx++
			chars = append(chars, &Character{
				Char:             ch,
				Pinyin:           []string{fmt.Sprintf("p%d", idx)},
				WuXing:           wx,
				SimplifiedStroke: 8,
				IsRegular:        true,
				IsNameable:       true,
				CommonLevel:      1,
				PositiveScore:    90,
			})
		}
	}
	appendGood(goodWater, "水")
	appendGood(goodMetal, "金")

	// 荒谬字（PositiveScore=0），五行水/金/火，验证被白名单剔除
	// extraAsAbsurd>0 时不放入池内（由测试作为 ExtraChars 注入）
	absurdList := []struct {
		ch string
		wx string
	}{
		{"拤", "金"}, {"饹", "金"}, {"婊", "水"}, {"蚂", "水"}, {"蛞", "火"},
	}
	for _, a := range absurdList {
		if extraAsAbsurd > 0 && a.ch == "拤" {
			continue // 拤 由测试注入 ExtraChars，不放入池
		}
		chars = append(chars, &Character{
			Char:             a.ch,
			Pinyin:           []string{a.ch},
			WuXing:           a.wx,
			SimplifiedStroke: 8,
			IsRegular:        true,
			IsNameable:       true,
			CommonLevel:      2,
			PositiveScore:    0,
		})
	}

	return &stubProvider{chars: chars}
}
