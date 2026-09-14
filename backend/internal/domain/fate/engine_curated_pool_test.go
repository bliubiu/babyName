package fate

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestCuratedPoolExcludesAbsurdChars 验证全放行后荒谬字仍不进入推荐榜：
// 荒谬字（拤/饹/婊/蚂/蛞）PositiveScore 为空，虽与好字一同入池参与组合，
// 但靠以下防线被压制在榜单之外：
//   - WenHuaRater 仅对「策展∩评分>=90」加分，荒谬字无文化分
//   - 非策展双名四维封顶 75，荒谬组合天然低分
//   - 组合级质量门禁（IsNonNamingCombo / IsBadCombo）
//
// 背景：曾用「人工评分>=85」白名单一刀切充当唯一字源门槛（仅 563 字被评分，
// 候选池被压到 200-360 字导致名字高度近似）。现依产品决策全放行《通用规范
// 汉字表》字源，荒谬字防御下沉到评分与组合门禁（详见 engine.go 第 6 步注释）。
func TestCuratedPoolExcludesAbsurdChars(t *testing.T) {
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

	// 荒谬字：不得进入推荐榜（被评分/门禁防线压制）
	for _, absurd := range curatedPoolAbsurdChars {
		for _, nr := range output.TopNames {
			if strings.Contains(nr.GivenName, absurd) {
				t.Errorf("荒谬字「%s」进入推荐名「%s」，评分/门禁防线未将其压制", absurd, nr.GivenName)
				break
			}
		}
	}
}

// TestCuratedPoolKeepsGoodChars 验证好字（PositiveScore 高分字）在推荐结果中保留
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

	// 至少一个好字应出现在结果中（全放行未稀释好字上榜）
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
		t.Fatal("推荐结果中未出现任何策展好字，全放行后好字未能上榜")
	}
}

// curatedPoolGoodChars / curatedPoolAbsurdChars 供上面的测试引用
var (
	curatedPoolGoodChars   = []string{"泽", "清", "涵", "澄", "渊", "明", "瑞", "浩"}
	curatedPoolAbsurdChars = []string{"拤", "饹", "婊", "蚂", "蛞"}
)

// newCuratedPoolProvider 构造一个候选池>=80 的字桩（避开喜用神收窄的降级保护），
// 含人工评分好字（PositiveScore=90）与荒谬字（PositiveScore=0），
// 用于验证全放行后荒谬字被评分/门禁防线压制在推荐榜外。
// 参数 extraAsAbsurd 兼容历史调用：>0 时「拤」由测试另行注入 ExtraChars（池内不放）。
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
