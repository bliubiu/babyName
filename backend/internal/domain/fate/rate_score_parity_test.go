package fate

import (
	"testing"
)

// rate_score_parity_test.go — 评分与解释分离的一致性护栏
//
// RateNameScore（穷举热路径，只算总分）与 RateName（全量明细）必须给出
// 完全相同的 Total，否则「枚举期排名」与「入榜后回算明细」会不一致，
// 表现为榜单分数与明细对不上。
//
// 本文件穷举一批覆盖各分支的候选，逐一断言二者相等。

// parityCandidates 覆盖各评分分支的候选集合：
//   - 双名 / 单名
//   - 策展 ∩ 精选好字（豁免封顶） / 非策展（触发四维封顶）
//   - HasPoetry 真 / 假（后者走 checkSemanticPoetry 语义分支）
//   - 同音、叠韵、同声母组、三连同调（音韵各分支）
//   - 无命名价值字 / 生僻字 / 软消极字
func parityCandidates() []*NameCandidate {
	type spec struct {
		c1, c2                 string
		p1, p2                 string
		wx1, wx2               string
		st1, st2               int
		cur1, cur2             bool
		ps1, ps2               int
		lvl1, lvl2             int
		pen1, pen2             int
		poetry                 bool
		rad1, rad2, genderHint string
	}
	specs := []spec{
		// 策展 ∩ 精选好字：豁免封顶
		{"泽", "明", "ze2", "ming2", "水", "火", 17, 8, true, true, 92, 90, 1, 1, 0, 0, true, "氵", "日", "male"},
		// 非策展：触发四维封顶
		{"拤", "蚂", "qia2", "ma3", "金", "水", 9, 9, false, false, 0, 0, 2, 2, 0, 0, true, "扌", "虫", "neutral"},
		// 无门禁/生僻/软消极字惩罚组合
		{"慧", "僰", "hui4", "bo2", "水", "水", 15, 14, false, false, 70, 60, 1, 0, 13, 0, false, "心", "人", "female"},
		{"恃", "病", "shi4", "bing4", "金", "水", 10, 10, false, false, 0, 0, 1, 1, 0, 0, false, "忄", "疒", "neutral"},
		// 音韵各分支：同音 / 同声母组 / 叠韵 / 三连同调
		{"思", "斯", "si1", "si1", "金", "金", 9, 12, true, true, 90, 88, 1, 1, 0, 0, true, "心", "斤", "neutral"},
		{"北", "波", "bei3", "bo1", "水", "水", 5, 8, false, false, 0, 0, 1, 1, 0, 0, false, "匕", "氵", "neutral"},
		{"清", "青", "qing1", "qing1", "水", "金", 12, 8, false, false, 0, 0, 1, 1, 0, 0, true, "氵", "青", "neutral"},
		// 无拼音（走 p1=="" 早返回分支）
		{"", "", "", "", "", "", 0, 0, false, false, 0, 0, 0, 0, 0, 0, false, "", "", ""},
		// 单名（Char2 为空，不封顶）
		{"涵", "", "han2", "", "水", "", 12, 0, true, false, 91, 0, 1, 0, 0, 0, true, "氵", "", "neutral"},
		{"渊", "", "yuan1", "", "水", "", 12, 0, false, false, 0, 0, 1, 0, 0, 0, false, "氵", "", "neutral"},
		{"毅", "", "yi4", "", "金", "", 15, 0, true, false, 90, 0, 1, 0, 0, 0, false, "殳", "", "male"},
	}

	out := make([]*NameCandidate, 0, len(specs)*len(specs))
	for i := range specs {
		s := specs[i]
		out = append(out, &NameCandidate{
			Char1: s.c1, Char2: s.c2,
			Pinyin1: s.p1, Pinyin2: s.p2,
			WuXing1: s.wx1, WuXing2: s.wx2,
			Stroke1: s.st1, Stroke2: s.st2,
			Meaning1: "清澈明亮，指水清而透明", Meaning2: "光明、照耀，引申为明白",
			Radical1: s.rad1, Radical2: s.rad2,
			HasPoetry: s.poetry, PoetryFrom: "《诗经》",
			IsRegular: true, CommonLevel1: s.lvl1, CommonLevel2: s.lvl2,
			GenderHint:      s.genderHint,
			NamePenalty1:    s.pen1,
			NamePenalty2:    s.pen2,
			IsCurated1:      s.cur1,
			IsCurated2:      s.cur2,
			PositiveScore1:  s.ps1,
			PositiveScore2:  s.ps2,
			SurnamePinyin:   "zhang1",
			MeaningProfile1: meaningProfileOf("清澈明亮，指水清而透明"),
			MeaningProfile2: meaningProfileOf("光明、照耀，引申为明白"),
		})
	}
	return out
}

// TestRateNameScoreMatchesRateName 热路径总分与全量明细总分必须一致
func TestRateNameScoreMatchesRateName(t *testing.T) {
	raters := DefaultRaters()
	fd := benchFateData()

	cands := parityCandidates()
	checked := 0
	for _, c := range cands {
		if c.Char1 == "" {
			continue
		}
		full := RateName(c, fd, raters)
		fast := RateNameScore(c, fd, raters)
		if full.Total != fast {
			t.Errorf("候选 %s%s：RateName.Total=%.1f 与 RateNameScore=%.1f 不一致",
				c.Char1, c.Char2, full.Total, fast)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("未构造出任何候选，测试无效")
	}
	t.Logf("已校验 %d 个候选，RateNameScore 与 RateName 总分一致", checked)
}

// TestRateNameScoreSkipsDetail 热路径不产生任何依据文案与明细，且不污染候选开关
func TestRateNameScoreSkipsDetail(t *testing.T) {
	raters := DefaultRaters()
	fd := benchFateData()

	for _, c := range parityCandidates() {
		if c.Char1 == "" {
			continue
		}
		fast := RateNameScore(c, fd, raters)
		_ = fast
		if c.skipDetail {
			t.Fatalf("候选 %s%s：RateNameScore 返回后 skipDetail 未还原", c.Char1, c.Char2)
		}
		// 随后用 RateName 应能拿到完整明细（开关必须已还原）
		full := RateName(c, fd, raters)
		if len(full.Details) == 0 {
			t.Errorf("候选 %s%s：RateNameScore 之后 RateName 未产出明细", c.Char1, c.Char2)
		}
		for dim, d := range full.Details {
			if d == "" {
				t.Errorf("候选 %s%s 维度 %s 的依据文案为空", c.Char1, c.Char2, dim)
			}
		}
	}
}

// TestRateNameHonoursSkipDetailFlag 明确 skipDetail 时 RateName 也应临时关闭它
//
// RateName 的契约是「始终返回完整明细」，这样入榜回算（走 RateName）
// 不会被枚举期遗留的开关影响。
func TestRateNameHonoursSkipDetailFlag(t *testing.T) {
	raters := DefaultRaters()
	fd := benchFateData()
	c := benchCandidate()
	c.skipDetail = true

	ns := RateName(c, fd, raters)
	if len(ns.Details) == 0 {
		t.Error("RateName 应临时关闭 skipDetail 并产出完整明细")
	}
	if !c.skipDetail {
		t.Error("RateName 返回后应还原候选原有的 skipDetail 取值")
	}
}
