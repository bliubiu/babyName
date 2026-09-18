package fate

import (
	"fmt"
	"strings"

	"name/internal/domain/classics"
)

// ——— 测名（给定姓名评分） ———
//
// 背景（docs/27 §4.1 A1）：fate.RateName 虽然导出，但 NameCandidate 的 20+ 字段
// （拼音/五行/笔画/寓意分/常用等级/频率档位/策展标记/释义画像/姓氏拼音）此前只在
// 引擎内部枚举候选时组装，没有任何导出入口。若测名路径自行拼装一份口径不一致的
// 候选，会造成「结果页与测名页同名字不同分」的新双口径问题。
//
// 本文件提供与引擎枚举阶段**同一条装配链**的导出函数：
//   - buildGivenCharInfo：按字还原 charInfo（与 engine.generate 的预计算一致）
//   - doubleNameCandidate / singleNameCandidate：复用引擎既有组装函数
//   - RateGivenName：任意外部姓名 → (NameCandidate, NameScore)
//
// 这样「结果页的名字」与「测名页的名字」共用同一份候选字段与同一组 Rater，
// 分数天然一致（由 evaluate 同构测试钉住）。

// buildGivenCharInfo 按引擎口径还原单个汉字的候选字预计算属性。
// 与 engine.generate 中 infos 的构造逐字段一致：
// 笔画走 filter.GetCharacterStroke（StrokeMode 决定口径）、拼音取第一个、
// 诗词出典走 classics.FindPoetryByChars、释义画像走 meaningProfileOf。
func buildGivenCharInfo(provider CharacterProvider, filter Filter, ch string) (charInfo, error) {
	c, err := provider.GetCharacter(ch)
	if err != nil {
		return charInfo{}, fmt.Errorf("查询汉字「%s」失败: %w", ch, err)
	}
	if c == nil {
		return charInfo{}, fmt.Errorf("汉字「%s」未被字库收录", ch)
	}
	found, desc, _ := classics.FindPoetryByChars(ch)
	return charInfo{
		ch:             c,
		stroke:         filter.GetCharacterStroke(c),
		pinyin:         firstPinyin(c.Pinyin),
		poetryFound:    found,
		poetryDesc:     desc,
		meaningProfile: meaningProfileOf(c.Meaning),
	}, nil
}

// RateGivenName 对「姓氏 + 名字」评分，供测名（/names/evaluate）使用。
//
// 装配口径与生成路径完全一致：
//   - 单/双名候选分别由 singleNameCandidate / doubleNameCandidate 组装
//     （bigramCache 不注入，走 classics.GetBigramScore 慢路径，结果一致）；
//   - 评分用调用方传入的 raters（与引擎装配相同的 fate.DefaultRaters()）
//     与 FateData（与引擎相同的 BaziAnalyzerAdapter 产出）。
//
// surname 仅用于姓氏拼音（音韵谐音检测）与总笔画，不参与评分维度本身。
// 名字支持 1-2 个汉字；未收录汉字返回错误（由调用方转成风险提示或 400）。
func RateGivenName(
	provider CharacterProvider,
	filter Filter,
	surname, given string,
	fateData *FateData,
	raters []Rater,
) (*NameCandidate, NameScore, error) {
	given = strings.TrimSpace(given)
	runes := []rune(given)
	if len(runes) == 0 {
		return nil, NameScore{}, fmt.Errorf("名字为空")
	}
	if len(runes) > 2 {
		return nil, NameScore{}, fmt.Errorf("暂只支持单字名或双字名（收到 %d 个字）", len(runes))
	}

	infos := make([]charInfo, 0, len(runes))
	for _, r := range runes {
		info, err := buildGivenCharInfo(provider, filter, string(r))
		if err != nil {
			return nil, NameScore{}, err
		}
		infos = append(infos, info)
	}

	surnamePinyin := firstPinyinForSurname(surname, provider)

	var cand *NameCandidate
	if len(infos) == 2 {
		a, b := infos[0], infos[1]
		// 与引擎「甲字||乙字任一有出典」的合并语义一致
		poetryFound := a.poetryFound || b.poetryFound
		poetryDesc := ""
		if a.poetryFound {
			poetryDesc = a.poetryDesc
		} else if b.poetryFound {
			poetryDesc = b.poetryDesc
		}
		cand = doubleNameCandidate(a, b, poetryFound, poetryDesc, surnamePinyin, nil)
	} else {
		cand = singleNameCandidate(infos[0], surnamePinyin)
	}

	score := RateName(cand, fateData, raters)
	return cand, score, nil
}

// GivenNameStrokes 计算全名总笔画（姓氏 + 名字，康熙口径）。
// 与生成路径 NameResult.Strokes 的口径一致：姓氏走 GetSurnameStrokes
// （康熙笔画），名字逐字优先 KangxiStroke、未收录时降级 ScienceStroke。
func GivenNameStrokes(provider CharacterProvider, surname, given string) (int, error) {
	l1, l2, err := provider.GetSurnameStrokes(surname)
	if err != nil {
		return 0, fmt.Errorf("获取姓氏笔画失败: %w", err)
	}
	total := 0
	if l1 > 0 {
		total += l1
	}
	if l2 > 0 {
		total += l2
	}
	for _, r := range []rune(strings.TrimSpace(given)) {
		c, err := provider.GetCharacter(string(r))
		if err != nil || c == nil {
			continue
		}
		stroke := c.KangxiStroke
		if stroke == 0 {
			stroke = c.ScienceStroke
		}
		total += stroke
	}
	return total, nil
}
