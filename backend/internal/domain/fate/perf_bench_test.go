package fate

import (
	"strings"
	"testing"
	"time"
)

// perf_bench_test.go 穷举生成热路径的性能基准
//
// 目的：量化「每评估一个候选组合」的成本，用于判断 8105 字全量枚举
// （池 5000 字时约 2500 万组合）是否可在合理时间内完成。
//
// 运行：
//
//	go test ./internal/domain/fate/ -run '^$' -bench 'BenchmarkCombination|BenchmarkSemantic|BenchmarkComboGuards' -benchtime 20000x

// realisticMeanings 模拟 hanzi.json 中的真实释义长度分布：
// 短释义约 10 字，长释义（《说文》类）可达 100 字以上。
var realisticMeanings = []string{
	"清澈明亮，指水清而透明，引申为纯净、明白事理",
	"美好、善良，与恶相对，亦指技艺高超、擅长某事，如善书者",
	"《说文解字》云：水。出陇西柏道，东至武都沮县入汉。从水，羕声。引申为水波荡漾、绵延不绝之意，又通「漾」，见于《诗经》与楚辞注疏中，古人多用以形容水势盛大、浩渺无边的自然景象",
	"才可以、能力，指人的才干与本领，如贤能、能干，亦指有才能的人",
	"光明、照耀，从日京声，引申为明白、清楚、开朗，常用于形容心胸豁达、前程明亮之人",
	"指玉石的光彩与纹理，引申为珍贵、美好，古人以玉比德，故多用于形容品行高洁、温润如玉的君子",
	"高远、辽阔，从页冥声，指天空深远无边际，亦指人的志向远大、心胸开阔",
}

func benchCandidate() *NameCandidate {
	m1 := realisticMeanings[2] // 长释义，最坏情况
	m2 := realisticMeanings[4]
	return &NameCandidate{
		Char1:          "澄",
		Char2:          "明",
		Pinyin1:        "cheng2",
		Pinyin2:        "ming2",
		WuXing1:        "水",
		WuXing2:        "火",
		Stroke1:        15,
		Stroke2:        8,
		Meaning1:       m1,
		Meaning2:       m2,
		Radical1:       "氵",
		Radical2:       "日",
		IsRegular:      true,
		CommonLevel1:   1,
		CommonLevel2:   1,
		SurnamePinyin:  "zhang1",
		HasPoetry:      true,
		PoetryFrom:     "《楚辞》",
		PositiveScore1: 90,
		PositiveScore2: 90,
		IsCurated1:     true,
		IsCurated2:     true,
		// 引擎实际路径：按候选字预计算并注入释义画像
		MeaningProfile1: meaningProfileOf(m1),
		MeaningProfile2: meaningProfileOf(m2),
	}
}

func benchFateData() *FateData {
	return &FateData{
		WuXingXiji: WuXingXiji{
			XiYongShen:  []string{"水", "金"},
			RiZhu:       "甲",
			RiZhuWuXing: "木",
			QiangRuo:    "身弱",
		},
		BaziInfo: BaziInfo{Zodiac: "龙"},
	}
}

var benchSinkF float64
var benchSinkB bool
var benchSinkI int

// BenchmarkSemanticOverlap 字义重叠计算（NoveltyRater 内调用，最坏情况长释义）
func BenchmarkSemanticOverlap(b *testing.B) {
	a, c := realisticMeanings[2], realisticMeanings[4]
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchSinkI = semanticOverlap(a, c)
	}
}

// BenchmarkComboGuards 组合级预检链（早停之前的所有判断）
// 对应 generateDoubleName 内层循环中 RateName 之前的那一串 if。
func BenchmarkComboGuards(b *testing.B) {
	c1, c2 := "澄", "明"
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if IsBadCombo(c1, c2) {
			continue
		}
		if IsHistoricalFigureCombo(c1, c2) {
			continue
		}
		if IsNonNamingChar(c1) || IsNonNamingChar(c2) {
			continue
		}
		if IsForbiddenEntity(c1 + c2) {
			continue
		}
		benchSinkB = false
	}
}

// BenchmarkCharPotentialScore 潜力分计算（早停判断中每个组合调用两次）
func BenchmarkCharPotentialScore(b *testing.B) {
	xi := map[string]bool{"水": true, "金": true}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchSinkI = charPotentialScore("水", true, 90, true, xi)
	}
}

// BenchmarkRateName 单个候选的完整八维评分（穷举中每组合调用一次）
func BenchmarkRateName(b *testing.B) {
	raters := DefaultRaters()
	fd := benchFateData()
	cand := benchCandidate()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ns := RateName(cand, fd, raters)
		benchSinkF = ns.Total
	}
}

// BenchmarkRateNameFresh 同上，但每次用新 candidate 指针
// （贴近真实：engine 每组合都 new 一个 NameCandidate）
func BenchmarkRateNameFresh(b *testing.B) {
	raters := DefaultRaters()
	fd := benchFateData()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cand := benchCandidate()
		ns := RateName(cand, fd, raters)
		benchSinkF = ns.Total
	}
}

// BenchmarkRateNameNoProfile 未注入释义画像的兜底路径
// （测试/诊断工具构造 NameCandidate 时的情形，走按释义字符串取缓存）
func BenchmarkRateNameNoProfile(b *testing.B) {
	raters := DefaultRaters()
	fd := benchFateData()
	cand := benchCandidate()
	cand.MeaningProfile1, cand.MeaningProfile2 = nil, nil
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ns := RateName(cand, fd, raters)
		benchSinkF = ns.Total
	}
}

// BenchmarkRateNameScore 热路径：只算总分（评分与解释分离）
func BenchmarkRateNameScore(b *testing.B) {
	raters := DefaultRaters()
	fd := benchFateData()
	cand := benchCandidate()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchSinkF = RateNameScore(cand, fd, raters)
	}
}

// BenchmarkCombinationFast 热路径完整成本：预检链 + 只算总分 + 入无去重局部表
func BenchmarkCombinationFast(b *testing.B) {
	raters := DefaultRaters()
	fd := benchFateData()
	table := NewExcellentTableUnique(100)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cand := benchCandidate()
		total := RateNameScore(cand, fd, raters)
		table.TryPush(ExcellentEntry{
			Char1: cand.Char1, Char2: cand.Char2,
			Score: total, Grade: scoreToGrade(total),
		})
	}
}

// BenchmarkCombinationFull 一个组合的完整成本（预检链 + 评分 + 入表）
func BenchmarkCombinationFull(b *testing.B) {
	raters := DefaultRaters()
	fd := benchFateData()
	table := NewExcellentTableWithCap(100)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cand := benchCandidate()
		ns := RateName(cand, fd, raters)
		table.TryPush(ExcellentEntry{
			Char1: cand.Char1, Char2: cand.Char2,
			Score: ns.Total, Grade: ns.Grade,
			Items: ns.Items, Details: ns.Details,
		})
	}
}

// BenchmarkMeaningsLength 打印释义长度，确认最坏情况规模
func TestMeaningsLength(t *testing.T) {
	for i, m := range realisticMeanings {
		t.Logf("meaning[%d] runes=%d", i, len([]rune(m)))
	}
	t.Logf("strings.Repeat 校验: %d", len(strings.Repeat("x", 1)))
	_ = time.Now
}
