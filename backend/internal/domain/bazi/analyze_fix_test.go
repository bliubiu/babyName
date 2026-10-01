package bazi

import "testing"

// 本文件固化 docs/29 B1/B2 的修复回归：
// 五行相生表「土生金」与格局判定的季节→五行映射。

// TestCalculateShengKeScore_TuShengJin 相生表锚点：土生金（原表误作土→火）
func TestCalculateShengKeScore_TuShengJin(t *testing.T) {
	// 喜用神=土：土生金，加分项应计入金的计数（旧实现算到火上）
	w := &WuxingResult{Jin: 0, Mu: 0, Shui: 0, Huo: 10, Tu: 0}
	if got := calculateShengKeScore([]string{"土"}, w); got != 0 {
		t.Fatalf("喜用神=土且金计数为 0：得分应为 0，got %d（旧实现会算到火=10）", got)
	}
	w2 := &WuxingResult{Jin: 7, Mu: 0, Shui: 0, Huo: 10, Tu: 0}
	if got := calculateShengKeScore([]string{"土"}, w2); got != 7 {
		t.Fatalf("喜用神=土且金计数 7：得分应为 7，got %d", got)
	}
	// 对照：喜用神=火 → 生土，应计土的计数
	w3 := &WuxingResult{Jin: 0, Mu: 0, Shui: 0, Huo: 0, Tu: 9}
	if got := calculateShengKeScore([]string{"火"}, w3); got != 9 {
		t.Fatalf("喜用神=火且土计数 9：得分应为 9，got %d", got)
	}
}

// TestCalculateBaziPattern_SeasonMapping 格局分支不再死代码：
// 日主五行与季节当令五行比较（春木夏火秋金冬水）
func TestCalculateBaziPattern_SeasonMapping(t *testing.T) {
	cases := []struct {
		strength  string
		rishou    string
		season    string
		want      string
	}{
		{"身旺", "金", "秋", "正格-印比相生格"}, // 秋金得令
		{"身旺", "金", "春", "正格-财官相生格"}, // 春金不得令
		{"身旺", "木", "春", "正格-印比相生格"}, // 春木得令
		{"身旺", "火", "夏", "正格-印比相生格"}, // 夏火得令
		{"身旺", "水", "冬", "正格-印比相生格"}, // 冬水得令
		{"身弱", "木", "春", "从弱格-从印格"},
		{"身弱", "木", "秋", "从弱格-从财格"},
	}
	for _, c := range cases {
		if got := calculateBaziPattern(c.strength, c.rishou, c.season); got != c.want {
			t.Errorf("日主%s×%s日主×%s：got %q，want %q", c.strength, c.rishou, c.season, got, c.want)
		}
	}
}
