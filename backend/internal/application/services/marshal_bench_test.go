package services

import (
	"encoding/json"
	"testing"

	"name/internal/domain/bazi"
	"name/internal/domain/name"
	"name/internal/domain/yijing"
)

// marshal_bench_test.go 量化 /generate 响应序列化开销（常驻回归基准）。
//
// 动机（2026.10.25 性能深挖）：/generate 的 HTTP 响应与 saveHistory 的
// 落库 blob 是**同一份 GenerateResponse 的两次 json.Marshal**（gin 渲染
// 一次、saveHistory 再一次），e2e 实测双名基线响应 293.9KB。本基准量化
// 序列化成本，判断其是否构成并发尾延迟的显著贡献项。
//
// 首次实测（构造响应 293.9KB / 50 名，与 e2e 对齐）：
//
//	单次 Marshal  0.64 ms/op（148KB 分配，1 alloc）
//	每请求两次    1.22 ms/op（HTTP 响应 + 落库 blob）
//
// 结论：仅占串行请求 0.846s 的 0.14%，非瓶颈；但「marshal 一次、两处复用」
// 是零风险顺手优化（省 0.6ms/请求）。
//
// 运行（在 backend 目录）：
//
//	go test ./internal/application/services/ -run '^$' \
//	  -bench 'BenchmarkMarshalGenerateResponse' -benchmem -benchtime 2000x

// benchNames 构造 50 个字段规模贴近真实响应的 name.Name
// （e2e 实测：50 名 ≈ 293.9KB，即单名 ~5.9KB JSON）
func benchNames() []name.Name {
	names := make([]name.Name, 50)
	meaning := "指玉石的光彩与纹理，引申为珍贵、美好，古人以玉比德，故多用于形容品行高洁、温润如玉的君子"
	poem := "瞻彼淇奥，绿竹猗猗。有匪君子，如切如磋，如琢如磨。瑟兮僩兮，赫兮咺兮。有匪君子，终不可谖兮。"
	detail := "字义兼水之润下与金之坚刚，与喜用神相生；二字音律平仄相间，韵腹开阔响亮"
	for i := range names {
		names[i] = name.Name{
			ID:             int64(i + 1),
			Surname:        "张",
			GivenName:      "珀泽",
			FullName:       "张珀泽",
			Pinyin:         "po ze",
			Meaning:        meaning,
			Wuxing:         "金水",
			Nayin:          "天河水",
			Strokes:        32,
			Gender:         "male",
			BaZiScore:      88,
			Huangli:        "宜：祭祀、祈福、出行",
			Xiang:          "温润如玉，君子之德",
			PoetrySource:   "诗经",
			PoetryChapter:  "卫风·淇奥",
			PoetrySentence: "有匪君子，如金如锡，如圭如璧",
			PoetryAuthor:   "佚名",
			PoetryDynasty:  "先秦",
			PoetryFullText: poem,
			WuxingAnalysis: detail,
			MeaningDetail:  detail,
			BaziScoreDetail: detail,
			Yinyun:         detail,
			SancaiAnalysis: detail,
			Hexagram:       "泽天夬",
			HexagramMeaning: "决断而和悦，以刚决柔",
			TotalScore:     80.6,
			WuxingScore:    88,
			YinyunScore:    82,
			MeaningScore:   90,
			SancaiScore:    85,
			ZodiacScore:    78,
			NayinScore:     80,
			NoveltyScore:   75,
			BigramScore:    86,
			FrequencyScore: 82,
			ScoreDetail: []name.ScoreDetailItem{
				{Name: "五行八字", Score: 88, Detail: detail},
				{Name: "音韵", Score: 82, Detail: detail},
				{Name: "文化印象", Score: 90, Detail: detail},
				{Name: "三才", Score: 85, Detail: detail},
				{Name: "生肖", Score: 78, Detail: detail},
				{Name: "新颖度", Score: 75, Detail: detail},
				{Name: "共现", Score: 86, Detail: detail},
				{Name: "人名频率", Score: 82, Detail: detail},
			},
			Reasons: []string{"五行相生", "音韵和谐", "寓意典雅"},
			PoetryMatchedChars: []string{"泽"},
		}
	}
	return names
}

func benchResponse() *GenerateResponse {
	return &GenerateResponse{
		Bazi: bazi.BaziAnalysis{
			Xiyongshen:  []string{"木", "水", "金"},
			Rishou:      "甲",
			RishouWuxing: "木",
			Nayin:       "天河水",
			Bazi:        bazi.Bazi{Year: "甲辰", Month: "己巳", Day: "癸亥", Hour: "丁巳"},
			Wuxing:      bazi.WuxingResult{Jin: 2, Mu: 1, Shui: 2, Huo: 1, Tu: 2},
		},
		Nayin:         "天河水",
		Zodiac:        "龙",
		Hexagram:      &yijing.Hexagram{Number: 43, Name: "泽天夬"},
		Names:         benchNames(),
		GenerationID:  "bench-gen-id-0001",
	}
}

// BenchmarkMarshalGenerateResponse 单次序列化（gin HTTP 渲染 或 saveHistory 落库，二选一）
func BenchmarkMarshalGenerateResponse(b *testing.B) {
	resp := benchResponse()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, err := json.Marshal(resp)
		if err != nil {
			b.Fatal(err)
		}
		benchMarshalSink = data
	}
}

// BenchmarkMarshalGenerateResponse_x2 每请求实际发生的两次序列化
// （HTTP 响应一次 + saveHistory 落库一次）
func BenchmarkMarshalGenerateResponse_x2(b *testing.B) {
	resp := benchResponse()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		body, err := json.Marshal(resp)
		if err != nil {
			b.Fatal(err)
		}
		blob, err := json.Marshal(resp)
		if err != nil {
			b.Fatal(err)
		}
		benchMarshalSink = body
		benchMarshalSink2 = blob
	}
}

var benchMarshalSink, benchMarshalSink2 []byte

// TestBenchResponseSize 打印构造响应的实际序列化大小，对齐 e2e 293.9KB
func TestBenchResponseSize(t *testing.T) {
	data, err := json.Marshal(benchResponse())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("构造响应序列化大小: %d 字节（e2e 双名基线实测 293.9KB）", len(data))
}
