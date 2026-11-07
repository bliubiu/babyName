package yijing

import (
	"strings"
	"testing"
)

// ============================================================================
// 体用断测试（docs/42 P1-③）
//
// 标准梅花体用：动爻所在卦为「用卦」，静卦为「体卦」。
// 体用生克映射：用生体/体用比和 → 吉；体克用/体生用 → 中；用克体 → 凶。
// 卦气旺衰：体卦五行 × 出生季节（旺相休囚死，docs/42 P1「卦气旺衰替代纯打分」）。
// ============================================================================

// TestAnalyzeTiYong_体用判定 locks「动者为用」：
// 动爻 1-3 → 下卦为用、上卦为体；动爻 4-6 → 上卦为用、下卦为体。
func TestAnalyzeTiYong_体用判定(t *testing.T) {
	cases := []struct {
		name        string
		upper       int
		lower       int
		movingYao   int
		wantTiGua   string
		wantYongGua string
	}{
		{"初爻动下卦为用", 4, 3, 1, "震", "离"}, // 丰卦体例：姓4名3初爻动
		{"三爻动下卦为用", 1, 2, 3, "乾", "兑"},
		{"四爻动上卦为用", 8, 1, 4, "乾", "坤"},
		{"上爻动上卦为用", 6, 8, 6, "坤", "坎"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := AnalyzeTiYong(c.upper, c.lower, c.movingYao, "", nil)
			if got == nil {
				t.Fatal("AnalyzeTiYong 不应返回 nil")
			}
			if got.TiGua != c.wantTiGua || got.YongGua != c.wantYongGua {
				t.Errorf("体用判定错误: 体=%s 用=%s，期望 体=%s 用=%s",
					got.TiGua, got.YongGua, c.wantTiGua, c.wantYongGua)
			}
		})
	}
}

// TestAnalyzeTiYong_动爻越界返回nil 动爻不在 1-6 时无法定用卦
func TestAnalyzeTiYong_动爻越界返回nil(t *testing.T) {
	for _, yao := range []int{0, 7, -1} {
		if got := AnalyzeTiYong(1, 8, yao, "", nil); got != nil {
			t.Errorf("动爻 %d 应返回 nil，实得 %+v", yao, got)
		}
	}
}

// TestAnalyzeTiYong_体用生克映射 locks 五行生克 → 吉凶三档
func TestAnalyzeTiYong_体用生克映射(t *testing.T) {
	cases := []struct {
		name         string
		upper        int
		lower        int
		movingYao    int
		wantVerdict  string
		wantRelation string
	}{
		{"用生体（坤土生乾金）应吉", 8, 1, 5, "吉", "用生体"},
		{"体用比和（乾金同气）应吉", 1, 1, 5, "吉", "比和"},
		{"体克用（坤土克坎水）应中", 6, 8, 5, "中", "体克用"},
		{"体生用（坤土生乾金）应中", 1, 8, 5, "中", "体生用"},
		{"用克体（兑金克震木）应凶", 2, 4, 5, "凶", "用克体"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := AnalyzeTiYong(c.upper, c.lower, c.movingYao, "", nil)
			if got == nil {
				t.Fatal("AnalyzeTiYong 不应返回 nil")
			}
			if got.Verdict != c.wantVerdict {
				t.Errorf("verdict=%q，期望 %q", got.Verdict, c.wantVerdict)
			}
			if got.Relation != c.wantRelation {
				t.Errorf("relation=%q，期望 %q", got.Relation, c.wantRelation)
			}
		})
	}
}

// TestAnalyzeTiYong_卦气旺衰 locks 旺相休囚死表（体卦五行 × 出生季节）。
// 标准口径：旺=当令、相=令所生、休=生令者、囚=克令者、死=令所克。
func TestAnalyzeTiYong_卦气旺衰(t *testing.T) {
	cases := []struct {
		wuxing string // 体卦五行
		season string
		want   string
	}{
		{"木", "春", "旺"}, // 春木当令
		{"火", "春", "相"}, // 木生火
		{"水", "春", "休"}, // 水生木
		{"金", "春", "囚"}, // 金克木
		{"土", "春", "死"}, // 木克土
		{"火", "夏", "旺"},
		{"金", "秋", "旺"},
		{"水", "冬", "旺"},
		{"火", "冬", "死"}, // 水克火
		{"金", "夏", "死"}, // 火克金
	}
	for _, c := range cases {
		// 按五行反查一个卦：木→震(4)、火→离(3)、土→坤(8)、金→乾(1)、水→坎(6)
		trigram := map[string]int{"木": 4, "火": 3, "土": 8, "金": 1, "水": 6}[c.wuxing]
		// 固定动爻 1（下卦为用），体卦=上卦
		got := AnalyzeTiYong(trigram, 8, 1, c.season, nil)
		if got == nil {
			t.Fatalf("五行 %s 季节 %s：不应返回 nil", c.wuxing, c.season)
		}
		if got.TiWuxing != c.wuxing {
			t.Fatalf("体卦五行=%q，期望 %q", got.TiWuxing, c.wuxing)
		}
		if got.WangShuai != c.want {
			t.Errorf("五行 %s 季节 %s：旺衰=%q，期望 %q", c.wuxing, c.season, got.WangShuai, c.want)
		}
	}
}

// TestAnalyzeTiYong_无季节跳过旺衰 season 为空时 WangShuai 为空，不误判
func TestAnalyzeTiYong_无季节跳过旺衰(t *testing.T) {
	got := AnalyzeTiYong(8, 1, 5, "", nil)
	if got == nil {
		t.Fatal("不应返回 nil")
	}
	if got.WangShuai != "" {
		t.Errorf("无季节时旺衰应为空，实得 %q", got.WangShuai)
	}
}

// TestAnalyzeTiYong_喜用神对照 locks 体卦五行与八字喜用神的关系表述
func TestAnalyzeTiYong_喜用神对照(t *testing.T) {
	// 上=乾金 下=坤土 动五爻：用=坤土 体=乾金（用生体，吉）
	got := AnalyzeTiYong(8, 1, 5, "秋", []string{"金"})
	if got == nil {
		t.Fatal("不应返回 nil")
	}
	if !strings.Contains(got.XiyongNote, "喜用神") || !strings.Contains(got.XiyongNote, "金") {
		t.Errorf("体卦为喜用神时说明应点名五行金，实得 %q", got.XiyongNote)
	}

	// 体卦五行与喜用神无直接关联：体=乾金，喜用=木（金克木，无生扶关系描述）
	got2 := AnalyzeTiYong(8, 1, 5, "", []string{"木"})
	if got2 == nil {
		t.Fatal("不应返回 nil")
	}
	if got2.XiyongNote == "" {
		t.Error("喜用神说明不应为空（无关联时也应说明）")
	}

	// 无喜用神输入：跳过对照
	got3 := AnalyzeTiYong(8, 1, 5, "", nil)
	if got3 == nil {
		t.Fatal("不应返回 nil")
	}
	if got3.XiyongNote != "" {
		t.Errorf("无喜用神输入时说明应为空，实得 %q", got3.XiyongNote)
	}
}

// TestAnalyzeTiYong_文字化解读 locks Text 必含体用卦名与吉凶判定（文字化呈现替代打分）
func TestAnalyzeTiYong_文字化解读(t *testing.T) {
	got := AnalyzeTiYong(2, 4, 5, "春", []string{"水"})
	if got == nil {
		t.Fatal("不应返回 nil")
	}
	for _, kw := range []string{"体卦", "用卦", "震", "兑", "凶"} {
		if !strings.Contains(got.Text, kw) {
			t.Errorf("解读应含 %q，实得 %q", kw, got.Text)
		}
	}
}
