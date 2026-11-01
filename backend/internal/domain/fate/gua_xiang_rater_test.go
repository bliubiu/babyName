package fate

import (
	"math"
	"testing"

	"name/internal/domain/yijing"
)

// ============================================================================
// 卦象体用评分测试（docs/42 P3-12）
//
// 口径约束（记忆硬约束）：
//   - 起卦入口唯一 yijing.GetMeihuaDetailByStrokes（姓%8上卦/名总%8下卦/和%6动爻）；
//   - 笔画走康熙口径（与结果页姓名卦一致），候选携带 KangxiStroke1/2，
//     姓氏康熙笔画经 FateData.SurnameStrokes 注入。
// ============================================================================

// TestGuaXiangScoreMapping 体用关系 × 卦气旺衰 → 分值映射（纯函数）
//
// 基准：用生体/比和 90；体克用/体生用 70；用克体 45。
// 旺衰微调：旺/相 +5；休 0；囚/死 -5。
func TestGuaXiangScoreMapping(t *testing.T) {
	cases := []struct {
		relation  string
		wangShuai string
		want      float64
	}{
		{"用生体", "旺", 95},
		{"用生体", "相", 95},
		{"用生体", "休", 90},
		{"用生体", "囚", 85},
		{"用生体", "死", 85},
		{"比和", "旺", 95},
		{"比和", "休", 90},
		{"比和", "", 90}, // 无季节输入：不微调
		{"体克用", "休", 70},
		{"体克用", "死", 65},
		{"体生用", "休", 70},
		{"体生用", "囚", 65},
		{"用克体", "旺", 50},
		{"用克体", "相", 50},
		{"用克体", "休", 45},
		{"用克体", "", 45},
	}
	for _, c := range cases {
		if got := guaXiangScore(c.relation, c.wangShuai); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("guaXiangScore(%q, %q) = %v, want %v", c.relation, c.wangShuai, got, c.want)
		}
	}
}

// TestGuaXiangRater_UnknownRelation 未知关系回落中平分（防御性）
func TestGuaXiangRater_UnknownRelation(t *testing.T) {
	if got := guaXiangScore("", ""); got != 60 {
		t.Errorf("guaXiangScore(\"\") = %v, want 60（中平）", got)
	}
}

// TestGuaXiangRater_Degraded 降级路径：信息不足按中平 60 计，不 panic
func TestGuaXiangRater_Degraded(t *testing.T) {
	r := NewGuaXiangRaterWithWeight(0.10)

	if got := r.Rate(nil, nil); got.Score != 60 {
		t.Errorf("fateData=nil 时 Score = %v, want 60", got.Score)
	}

	// 候选缺少姓氏笔画注入（FateData.SurnameStrokes=0）
	cand := &NameCandidate{Char1: "泽", Stroke1: 8, KangxiStroke1: 8}
	if got := r.Rate(cand, &FateData{}); got.Score != 60 {
		t.Errorf("SurnameStrokes=0 时 Score = %v, want 60", got.Score)
	}

	// 候选缺少康熙笔画
	fd := &FateData{SurnameStrokes: 7, BirthSeason: "夏"}
	if got := r.Rate(&NameCandidate{Char1: "泽", Stroke1: 8}, fd); got.Score != 60 {
		t.Errorf("KangxiStroke 缺失时 Score = %v, want 60", got.Score)
	}
}

// TestGuaXiangRater_RealHexagram 起卦端到端：注入已知卦表，验证打分与缓存一致性
//
// 用例一：姓 7 画（上卦 7 艮土）+ 单名 3 画（下卦 3 离火），动爻 (7+3)%6=4，
// 动者为用 → 用=艮土、体=离火，体生用（基准 70）；夏月体卦火旺（+5）→ 75。
//
// 用例二：姓 7 画 + 双名 8+4=12 画（下卦 12%8=4 震木），动爻 19%6=1，
// 动者为用 → 用=震木、体=艮土，木克土为用克体（基准 45）；夏月体卦土相（+5）→ 50。
func TestGuaXiangRater_RealHexagram(t *testing.T) {
	yijing.SetHexagrams([]yijing.Hexagram{
		{ID: 1, Name: "山火贲", Number: 22, UpperTrigram: 7, LowerTrigram: 3},
		{ID: 2, Name: "山雷颐", Number: 27, UpperTrigram: 7, LowerTrigram: 4},
	})

	r := NewGuaXiangRaterWithWeight(0.10)

	t.Run("单名_体生用_夏月旺", func(t *testing.T) {
		cand := &NameCandidate{Char1: "灿", Stroke1: 3, KangxiStroke1: 3}
		fd := &FateData{SurnameStrokes: 7, BirthSeason: "夏"}
		got := r.Rate(cand, fd)
		if got.Score != 75 {
			t.Errorf("Score = %v, want 75（体生用70 + 夏月旺5），Detail: %s", got.Score, got.Detail)
		}
		if got.Detail == "" {
			t.Error("Detail 不应为空")
		}

		// 缓存命中：同参复跑分值一致（不打总分漂移）
		if again := r.Rate(cand, fd); again.Score != got.Score {
			t.Errorf("缓存复跑 Score = %v, want %v", again.Score, got.Score)
		}
	})

	t.Run("双名_用克体_夏月相", func(t *testing.T) {
		cand := &NameCandidate{
			Char1: "泽", Stroke1: 8, KangxiStroke1: 8,
			Char2: "雷", Stroke2: 4, KangxiStroke2: 4,
		}
		fd := &FateData{SurnameStrokes: 7, BirthSeason: "夏"}
		if got := r.Rate(cand, fd); got.Score != 50 {
			t.Errorf("Score = %v, want 50（用克体45 + 夏月相5），Detail: %s", got.Score, got.Detail)
		}
	})

	t.Run("无季节_不加旺衰微调", func(t *testing.T) {
		cand := &NameCandidate{Char1: "灿", Stroke1: 3, KangxiStroke1: 3}
		fd := &FateData{SurnameStrokes: 7}
		if got := r.Rate(cand, fd); got.Score != 70 {
			t.Errorf("Score = %v, want 70（体生用基准，无季节不微调）", got.Score)
		}
	})
}

// TestGuaXiangRater_SkipDetail 热路径开关：skipDetail 时仅省文案，分值不变
func TestGuaXiangRater_SkipDetail(t *testing.T) {
	yijing.SetHexagrams([]yijing.Hexagram{
		{ID: 1, Name: "山火贲", Number: 22, UpperTrigram: 7, LowerTrigram: 3},
	})

	r := NewGuaXiangRaterWithWeight(0.10)
	fd := &FateData{SurnameStrokes: 7, BirthSeason: "夏"}

	full := &NameCandidate{Char1: "灿", Stroke1: 3, KangxiStroke1: 3}
	skip := &NameCandidate{Char1: "灿", Stroke1: 3, KangxiStroke1: 3, skipDetail: true}

	a, b := r.Rate(full, fd), r.Rate(skip, fd)
	if a.Score != b.Score {
		t.Errorf("skipDetail 改变了分值: %v vs %v", a.Score, b.Score)
	}
	if b.Detail != "" {
		t.Errorf("skipDetail 时 Detail 应为空, got %q", b.Detail)
	}
}

// TestDefaultRaters_IncludesGuaXiang DefaultRaters 含「卦象」维度（权重 10%）且总和为 1.0
func TestDefaultRaters_IncludesGuaXiang(t *testing.T) {
	raters := DefaultRaters()
	var sum float64
	var guaXiang *Rater
	for i := range raters {
		if raters[i].Name() == "卦象" {
			guaXiang = &raters[i]
		}
		sum += raters[i].Weight()
	}
	if guaXiang == nil {
		t.Fatal("DefaultRaters 缺少「卦象」维度（docs/42 P3-12）")
	}
	if (*guaXiang).Weight() != 0.10 {
		t.Errorf("卦象权重 = %v, want 0.10", (*guaXiang).Weight())
	}
	if math.Abs(sum-1.0) > 1e-9 {
		t.Errorf("DefaultRaters 权重和 = %v, want 1.0", sum)
	}
}
