package name

import (
	"testing"
	"unicode/utf8"
)

// TestScoreNameUnified 验证统一评分体系的基本行为
func TestScoreNameUnified(t *testing.T) {
	eng := NewEnhancedNameGenerator("../../data")
	if eng == nil {
		t.Fatal("NewEnhancedNameGenerator failed")
	}

	// 测试单字名评分
	name := &Name{
		Surname:   "王",
		GivenName: "伟",
		Pinyin:    "wei",
		Wuxing:    "土",
		Strokes:   6,
		Gender:    "male",
	}

	xiyongshen := []string{"土", "金"}
	eng.ScoreNameUnified(name, xiyongshen, "龙")

	// 验证各维度评分已填充
	if name.TotalScore <= 0 {
		t.Errorf("TotalScore should be > 0, got %f", name.TotalScore)
	}
	if name.WuxingScore <= 0 {
		t.Errorf("WuxingScore should be > 0, got %f", name.WuxingScore)
	}
	if name.YinyunScore <= 0 {
		t.Errorf("YinyunScore should be > 0, got %f", name.YinyunScore)
	}
	if name.MeaningScore <= 0 {
		t.Errorf("MeaningScore should be > 0, got %f", name.MeaningScore)
	}
	if name.SancaiScore <= 0 {
		t.Errorf("SancaiScore should be > 0, got %f", name.SancaiScore)
	}
	if name.ZodiacScore <= 0 {
		t.Errorf("ZodiacScore should be > 0, got %f", name.ZodiacScore)
	}

	// 验证总分在合理范围
	if name.TotalScore > 100 || name.TotalScore < 0 {
		t.Errorf("TotalScore out of range [0,100]: %f", name.TotalScore)
	}

	t.Logf("王伟 评分结果：Total=%.1f, Wuxing=%.1f, Yinyun=%.1f, Meaning=%.1f, Sancai=%.1f, Zodiac=%.1f, BaZi=%d",
		name.TotalScore, name.WuxingScore, name.YinyunScore, name.MeaningScore, name.SancaiScore, name.ZodiacScore, name.BaZiScore)
}

// TestScoreNameUnified_NoXiyongshen 无喜用神时评分行为
func TestScoreNameUnified_NoXiyongshen(t *testing.T) {
	eng := NewEnhancedNameGenerator("../../data")

	name := &Name{
		Surname:   "张",
		GivenName: "明",
		Pinyin:    "ming",
		Wuxing:    "火",
		Strokes:   8,
		Gender:    "male",
	}

	eng.ScoreNameUnified(name, nil, "")

	if name.TotalScore <= 0 {
		t.Errorf("TotalScore should still be > 0 without xiyongshen, got %f", name.TotalScore)
	}
	if name.TotalScore > 100 {
		t.Errorf("TotalScore should be <= 100, got %f", name.TotalScore)
	}

	t.Logf("无喜用神评分：Total=%.1f, Wuxing=%.1f, Yinyun=%.1f, Sancai=%.1f",
		name.TotalScore, name.WuxingScore, name.YinyunScore, name.SancaiScore)
}

// TestEvalPair 验证双字组合评估
func TestEvalPair(t *testing.T) {
	eng := NewEnhancedNameGenerator("../../data")

	opts := GenerateOptions{
		Surname:    "李",
		Gender:     "male",
		Xiyongshen: []string{"木", "火"},
		Zodiac:     "龙",
	}

	result := eng.evalPair("明", "杰", opts)

	// 验证基本信息
	if result.chars != "明杰" {
		t.Errorf("chars expected '明杰', got '%s'", result.chars)
	}
	if result.pinyin1 == "" || result.pinyin2 == "" {
		t.Errorf("pinyin should not be empty")
	}

	// 验证各维度评分
	if result.wuxingScore <= 0 {
		t.Errorf("wuxingScore should be > 0, got %f", result.wuxingScore)
	}
	if result.yinyunScore <= 0 {
		t.Errorf("yinyunScore should be > 0, got %f", result.yinyunScore)
	}
	if result.meaningScore <= 0 {
		t.Errorf("meaningScore should be > 0, got %f", result.meaningScore)
	}
	if result.sancaiScore <= 0 {
		t.Errorf("sancaiScore should be > 0, got %f", result.sancaiScore)
	}
	if result.totalScore <= 0 {
		t.Errorf("totalScore should be > 0, got %f", result.totalScore)
	}

	t.Logf("明杰 组合评估：Total=%.1f, Wuxing=%.1f, Yinyun=%.1f, Meaning=%.1f, Sancai=%.1f, Zodiac=%.1f",
		result.totalScore, result.wuxingScore, result.yinyunScore, result.meaningScore, result.sancaiScore, result.zodiacScore)
}

// TestGenerateUnified_Simple 验证统一生成入口的基本行为
func TestGenerateUnified_Simple(t *testing.T) {
	eng := NewEnhancedNameGenerator("../../data")

	opts := GenerateOptions{
		Surname:    "王",
		Gender:     "male",
		Xiyongshen: []string{"金", "水"},
		Count:      10,
		NameLength: 2,
		ExcludeRare: true,
	}

	names, err := eng.GenerateUnified(opts)
	if err != nil {
		t.Fatalf("GenerateUnified failed: %v", err)
	}

	if len(names) == 0 {
		t.Fatal("GenerateUnified returned no names")
	}

	// 验证结果被正确排序（高分在前）
	for i := 1; i < len(names); i++ {
		if names[i-1].TotalScore < names[i].TotalScore {
			t.Errorf("names should be sorted by TotalScore descending: [%d]%.1f < [%d]%.1f",
				i-1, names[i-1].TotalScore, i, names[i].TotalScore)
		}
	}

	// 验证所有名字都填充了评分
	for _, n := range names {
		if n.TotalScore <= 0 {
			t.Errorf("name %s has invalid TotalScore: %f", n.FullName, n.TotalScore)
		}
		if n.WuxingScore <= 0 {
			t.Errorf("name %s has invalid WuxingScore: %f", n.FullName, n.WuxingScore)
		}
		if n.YinyunScore <= 0 {
			t.Errorf("name %s has invalid YinyunScore: %f", n.FullName, n.YinyunScore)
		}
		if len(n.FullName) == 0 {
			t.Errorf("name has empty FullName")
		}
	}

	t.Logf("生成 %d 个名字，最高分=%.1f, 最低分=%.1f",
		len(names), names[0].TotalScore, names[len(names)-1].TotalScore)

	// 打印前5个名字供人工审查
	for i, n := range names {
		if i >= 5 {
			break
		}
		t.Logf("  %d. %s  Total=%.1f Wuxing=%.1f YinYun=%.1f Meaning=%.1f Sancai=%.1f",
			i+1, n.FullName, n.TotalScore, n.WuxingScore, n.YinyunScore, n.MeaningScore, n.SancaiScore)
	}
}

// TestGenerateUnified_DoubleChar 验证双字名组合评估
func TestGenerateUnified_DoubleChar(t *testing.T) {
	eng := NewEnhancedNameGenerator("../../data")

	opts := GenerateOptions{
		Surname:    "张",
		Gender:     "male",
		Xiyongshen: []string{"木", "火"},
		Count:      15,
		NameLength: 2,
		ExcludeRare: true,
		Zodiac:     "龙",
	}

	names, err := eng.GenerateUnified(opts)
	if err != nil {
		t.Fatalf("GenerateUnified failed: %v", err)
	}

	if len(names) == 0 {
		t.Fatal("GenerateUnified returned no names")
	}

	// 验证全部是双字名（注意中文字符占用多字节，需用 RuneCountInString）
	for _, n := range names {
		if utf8.RuneCountInString(n.GivenName) != 2 {
			t.Errorf("expected 2-char given name, got '%s' (rune_count=%d)", n.FullName, utf8.RuneCountInString(n.GivenName))
		}
	}

	t.Logf("双字名生成：共 %d 个", len(names))
	for i, n := range names {
		if i >= 8 {
			break
		}
		t.Logf("  %d. %s  总分=%.1f  五行=%.1f  音韵=%.1f  字义=%.1f  三才=%.1f  生肖=%.1f",
			i+1, n.FullName, n.TotalScore, n.WuxingScore, n.YinyunScore,
			n.MeaningScore, n.SancaiScore, n.ZodiacScore)
	}
}

// TestGenerateUnified_SingleChar 验证单字名生成
func TestGenerateUnified_SingleChar(t *testing.T) {
	eng := NewEnhancedNameGenerator("../../data")

	opts := GenerateOptions{
		Surname:    "李",
		Gender:     "female",
		Xiyongshen: []string{"水", "木"},
		Count:      10,
		NameLength: 1,
		ExcludeRare: true,
	}

	names, err := eng.GenerateUnified(opts)
	if err != nil {
		t.Fatalf("GenerateUnified failed: %v", err)
	}

	if len(names) == 0 {
		t.Fatal("GenerateUnified returned no single-char names")
	}

	// 验证评分
	for _, n := range names {
		if n.TotalScore < 65 {
			t.Errorf("single-char name %s score %.1f below 65 threshold", n.FullName, n.TotalScore)
		}
		if utf8.RuneCountInString(n.GivenName) != 1 {
			t.Errorf("expected 1-char given name, got '%s' (rune_count=%d)", n.GivenName, utf8.RuneCountInString(n.GivenName))
		}
	}

	t.Logf("单字名生成：共 %d 个", len(names))
	for i, n := range names {
		if i >= 5 {
			break
		}
		t.Logf("  %d. %s  总分=%.1f  五行=%.1f  音韵=%.1f  字义=%.1f",
			i+1, n.FullName, n.TotalScore, n.WuxingScore, n.YinyunScore, n.MeaningScore)
	}
}

// TestGenerateUnified_Filtered 验证过滤条件的组合评估
func TestGenerateUnified_Filtered(t *testing.T) {
	eng := NewEnhancedNameGenerator("../../data")

	// 指定笔画范围和拼音首字母
	opts := GenerateOptions{
		Surname:       "王",
		Gender:        "male",
		Xiyongshen:    []string{"土"},
		Count:         10,
		NameLength:    2,
		ExcludeRare:   true,
		MinStrokes:    5,
		MaxStrokes:    15,
		PinyinInitial: "y",
	}

	names, err := eng.GenerateUnified(opts)
	if err != nil {
		t.Fatalf("GenerateUnified with filter failed: %v", err)
	}

	if len(names) > 0 {
		t.Logf("过滤后生成：%d 个名字", len(names))
		for i, n := range names {
			if i >= 5 {
				break
			}
			t.Logf("  %d. %s  总分=%.1f  笔画=%d",
				i+1, n.FullName, n.TotalScore, n.Strokes)
		}
	} else {
		t.Log("过滤条件苛刻，未生成名字（此情况合理）")
	}
}

// TestSearchPairInPoetry 验证双字共现诗词检索
func TestSearchPairInPoetry(t *testing.T) {
	eng := NewEnhancedNameGenerator("../../data")
	if eng == nil {
		t.Fatal("NewEnhancedNameGenerator failed")
	}

	tests := []struct {
		name   string
		char1  string
		char2  string
		found  bool // 是否预期能找到
	}{
		{
			name:  "常见共现组合",
			char1: "风",
			char2: "云",
			found: true, // "风云" 在楚辞中共现
		},
		{
			name:  "罕见组合-可能不共现",
			char1: "铄",
			char2: "鑫",
			found: false,
		},
		{
			name:  "空字符",
			char1: "",
			char2: "龙",
			found: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found, source, chapter, sentence := eng.searchPairInPoetry(tt.char1, tt.char2)
			if tt.found && !found {
				t.Errorf("expected pair '%s%s' found in poetry, but not found", tt.char1, tt.char2)
			}
			if !tt.found && found {
				// 不加Error，只是记录
				t.Logf("unexpected finding: '%s%s' in %s %s: %s", tt.char1, tt.char2, source, chapter, sentence)
			}
			if found {
				t.Logf("找到共现：%s%s → %s《%s》: %s", tt.char1, tt.char2, source, chapter, sentence)
			}
		})
	}
}

// TestQuickCheckPair 验证双字组合轻量预剪枝
func TestQuickCheckPair(t *testing.T) {
	eng := NewEnhancedNameGenerator("../../data")
	if eng == nil {
		t.Fatal("NewEnhancedNameGenerator failed")
	}

	tests := []struct {
		name   string
		char1  string
		char2  string
		opts   GenerateOptions
		expect bool // 预期quickCheckPair返回true还是false
	}{
		{
			name:  "正常组合-无喜用神过滤",
			char1: "风",
			char2: "鸣",
			opts: GenerateOptions{
				Xiyongshen: []string{},
			},
			expect: true, // 风(fēng,4画)≠鸣(míng,8画)，均≤40画，和12≤60 → true
		},
		{
			name:  "同音字",
			char1: "明",
			char2: "鸣",
			opts: GenerateOptions{
				Xiyongshen: []string{},
			},
			expect: false, // 明(mínɡ)和鸣(míng)去除声调后都是"ming" → 同音跳过
		},
		{
			name:  "笔画过多（备选字不存在于精选库时使用默认笔画）",
			char1: "鑫",
			char2: "山",
			opts: GenerateOptions{
				Xiyongshen: []string{},
			},
			expect: true, // 鑫未在精选字库中(24画)，回退默认10画 → 10+3≤60 通过
		},
		{
			name:  "同音（同字）",
			char1: "山",
			char2: "山",
			opts: GenerateOptions{
				Xiyongshen: []string{},
			},
			expect: false, // 同字同拼音 → 跳过
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := eng.quickCheckPair(tt.char1, tt.char2, tt.opts)
			if got != tt.expect {
				t.Errorf("quickCheckPair(%q, %q) = %v, want %v", tt.char1, tt.char2, got, tt.expect)
			}
		})
	}
}

// TestTiandiRenSancai 验证天地人三才评分引擎
// 天道评分取决于声调平仄和笔画奇偶
// 地道评分取决于两字笔画差距和偏旁多样性
// 人道评分取决于仁德字、诗词出处和积极正向字
// TestZodiacTaboo 验证生肖忌用字扣分逻辑
func TestZodiacTaboo(t *testing.T) {
	eng := NewEnhancedNameGenerator("../../data")

	tests := []struct {
		name       string
		givenName  string
		zodiac     string
		wantMinScore float64
		wantMaxScore float64
		desc       string
	}{
		// 鼠与马相冲 → 含马部字应扣分
		{givenName: "天骏", zodiac: "鼠", wantMinScore: 0, wantMaxScore: 79, desc: "鼠+骏(马部)应扣分"},
		// 兔与鸡相冲 → 含鸡部字应扣分
		{givenName: "天凤", zodiac: "兔", wantMinScore: 0, wantMaxScore: 79, desc: "兔+凤(鸟部)应扣分"},
		// 无冲突生肖
		{givenName: "天", zodiac: "鼠", wantMinScore: 70, wantMaxScore: 100, desc: "鼠+天无冲突"},
		{givenName: "天", zodiac: "兔", wantMinScore: 70, wantMaxScore: 100, desc: "兔+天无冲突"},
		// 空生肖 → 默认80分
		{givenName: "骏", zodiac: "", wantMinScore: 80, wantMaxScore: 80, desc: "空生肖应返回默认分"},
	}

	for _, tc := range tests {
		n := Name{
			Surname:   "王",
			GivenName: tc.givenName,
			Pinyin:    "wang",
			Wuxing:    "土",
			Strokes:   6,
		}
		_, score := eng.analyzeZodiac(n, tc.zodiac)
		if score < tc.wantMinScore || score > tc.wantMaxScore {
			t.Errorf("analyzeZodiac(%q, zodiac=%q) = %.1f, want [%.0f, %.0f] - %s",
				tc.givenName, tc.zodiac, score, tc.wantMinScore, tc.wantMaxScore, tc.desc)
		}
	}
}

// TestZodiacTabooChars_Data 验证各生肖忌用字数据不为空
func TestZodiacTabooChars_Data(t *testing.T) {
	eng := NewEnhancedNameGenerator("../../data")

	zodiacs := []string{"鼠", "牛", "虎", "兔", "龙", "蛇", "马", "羊", "猴", "鸡", "狗", "猪"}
	for _, z := range zodiacs {
		taboo := eng.getZodiacTabooChars(z)
		if len(taboo) == 0 {
			t.Errorf("getZodiacTabooChars(%q) returned empty list", z)
		}
	}
}

// TestZodiacGoodChars_Data 验证各生肖宜用字数据不为空
func TestZodiacGoodChars_Data(t *testing.T) {
	eng := NewEnhancedNameGenerator("../../data")

	zodiacs := []string{"鼠", "牛", "虎", "兔", "龙", "蛇", "马", "羊", "猴", "鸡", "狗", "猪"}
	for _, z := range zodiacs {
		good := eng.getZodiacGoodChars(z)
		if len(good) == 0 {
			t.Errorf("getZodiacGoodChars(%q) returned empty list", z)
		}
	}
}

func TestTiandiRenSancai(t *testing.T) {
	eng := NewEnhancedNameGenerator("../../data")

	tests := []struct {
		name        string
		surname     string
		givenName   string
		pinyin      string
		strokes     int
		wuxing      string
		expectScore float64 // 期望三才评分≥此值（仁德字+诗词会拉开差距）
	}{
		{
			name: "单字名",
			surname: "王", givenName: "伟",
			pinyin: "wei", strokes: 6, wuxing: "土",
			expectScore: 65,
		},
		{
			name: "双字名-高频字",
			surname: "张", givenName: "明杰",
			pinyin: "ming jie", strokes: 16, wuxing: "火、木",
			expectScore: 65,
		},
		{
			name: "含仁德字-仁",
			surname: "李", givenName: "仁杰",
			pinyin: "ren jie", strokes: 14, wuxing: "木、木",
			expectScore: 70,
		},
		{
			name: "含仁德字+诗词出处",
			surname: "王", givenName: "德馨",
			pinyin: "de xin", strokes: 20, wuxing: "火、金",
			expectScore: 70,
		},
		{
			name: "平仄交替-理想音韵",
			surname: "王", givenName: "浩然",
			pinyin: "hao ran", strokes: 16, wuxing: "水、金",
			expectScore: 68,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name := &Name{
				Surname:   tt.surname,
				GivenName: tt.givenName,
				Pinyin:    tt.pinyin,
				Wuxing:    tt.wuxing,
				Strokes:   tt.strokes,
				Gender:    "male",
			}
			eng.ScoreNameUnified(name, []string{"金", "水"}, "龙")

			// 验证三才评分（天地人）
			if name.SancaiScore < tt.expectScore {
				t.Errorf("TiandiRenSancai score too low: got %.1f, expect >= %.1f", name.SancaiScore, tt.expectScore)
			}
			if name.SancaiScore > 100 || name.SancaiScore < 30 {
				t.Errorf("TiandiRenSancai score out of range [30,100]: %.1f", name.SancaiScore)
			}
			if name.SancaiAnalysis == "" {
				t.Error("SancaiAnalysis should not be empty")
			}

			t.Logf("天地人三才: 评分=%.1f 分析=%s",
				name.SancaiScore, name.SancaiAnalysis)
		})
	}
}

// TestWuxingShengKe 验证五行生克制化评分
func TestWuxingShengKe(t *testing.T) {
	eng := NewEnhancedNameGenerator("../../data")
	if eng == nil {
		t.Fatal("NewEnhancedNameGenerator failed")
	}

	tests := []struct {
		wuxing     string
		xiyongshen []string
		wantMin    float64
		wantMax    float64
		desc       string
	}{
		// 直接匹配
		{wuxing: "金", xiyongshen: []string{"金"}, wantMin: 95, wantMax: 100, desc: "金=金直接匹配"},
		{wuxing: "水", xiyongshen: []string{"水"}, wantMin: 95, wantMax: 100, desc: "水=水直接匹配"},
		// 相生 金生水
		{wuxing: "金", xiyongshen: []string{"水"}, wantMin: 70, wantMax: 80, desc: "金生水→75"},
		{wuxing: "木", xiyongshen: []string{"火"}, wantMin: 70, wantMax: 80, desc: "木生火→75"},
		// 被生 水生木
		{wuxing: "木", xiyongshen: []string{"水"}, wantMin: 45, wantMax: 55, desc: "水生木→50"},
		// 相克 土克水 → raw=25, 但floor为30
		{wuxing: "土", xiyongshen: []string{"水"}, wantMin: 30, wantMax: 35, desc: "土克水→扣分至30"},
		// 被克 水克火 → raw=40
		{wuxing: "火", xiyongshen: []string{"水"}, wantMin: 35, wantMax: 45, desc: "水克火→40"},
		// 无喜用神 → 默认70
		{wuxing: "木", xiyongshen: []string{}, wantMin: 70, wantMax: 70, desc: "无喜用神默认70"},
		// 未知五行 → 默认70
		{wuxing: "未知", xiyongshen: []string{"土"}, wantMin: 70, wantMax: 70, desc: "未知五行默认70"},
	}

	for _, tc := range tests {
		score := eng.calculateWuxingScore(tc.wuxing, tc.xiyongshen)
		if score < tc.wantMin || score > tc.wantMax {
			t.Errorf("calculateWuxingScore(%q, %v) = %.1f, want [%.0f, %.0f] - %s",
				tc.wuxing, tc.xiyongshen, score, tc.wantMin, tc.wantMax, tc.desc)
		}
	}
}

// TestScoreDidao_Structure 验证字形结构评分
func TestScoreDidao_Structure(t *testing.T) {
	eng := NewEnhancedNameGenerator("../../data")
	if eng == nil {
		t.Fatal("NewEnhancedNameGenerator failed")
	}

	tests := []struct {
		surname    string
		givenName  string
		pinyin     string
		wantMin    float64
		wantMax    float64
		desc       string
	}{
		// 左右+上下结构 → 多样
		{surname: "王", givenName: "明李", pinyin: "wang ming li",
			wantMin: 70, wantMax: 100, desc: "王(独体)+明(左右)+李(上下)→结构多样"},
		// 单字名
		{surname: "王", givenName: "文", pinyin: "wang wen",
			wantMin: 65, wantMax: 100, desc: "单字名"},
	}

	for _, tc := range tests {
		n := Name{
			Surname:   tc.surname,
			GivenName: tc.givenName,
			Pinyin:    tc.pinyin,
			Strokes:   len([]rune(tc.givenName)) * 6,
		}
		score := eng.scoreDidao(n)
		if score < tc.wantMin || score > tc.wantMax {
			t.Errorf("scoreDidao(%q %q) = %.1f, want [%.0f, %.0f] - %s",
				tc.surname, tc.givenName, score, tc.wantMin, tc.wantMax, tc.desc)
		}
	}
}

// TestExtractNayinWuxing 验证从纳音名称提取五行属性
func TestExtractNayinWuxing(t *testing.T) {
	tests := []struct {
		nayin  string
		want   string
		desc   string
	}{
		{nayin: "海中金", want: "金", desc: "海中金→金"},
		{nayin: "炉中火", want: "火", desc: "炉中火→火"},
		{nayin: "大林木", want: "木", desc: "大林木→木"},
		{nayin: "涧下水", want: "水", desc: "涧下水→水"},
		{nayin: "壁上土", want: "土", desc: "壁上土→土"},
		{nayin: "", want: "", desc: "空字符串→空"},
		{nayin: "剑锋金", want: "金", desc: "剑锋金→金"},
		{nayin: "霹雳火", want: "火", desc: "霹雳火→火"},
		{nayin: "杨柳木", want: "木", desc: "杨柳木→木"},
		{nayin: "长流水", want: "水", desc: "长流水→水"},
		{nayin: "城头土", want: "土", desc: "城头土→土"},
	}

	for _, tc := range tests {
		got := extractNayinWuxing(tc.nayin)
		if got != tc.want {
			t.Errorf("extractNayinWuxing(%q) = %q, want %q - %s", tc.nayin, got, tc.want, tc.desc)
		}
	}
}

// TestCalculateNayinScore 验证纳音五行评分
func TestCalculateNayinScore(t *testing.T) {
	eng := NewEnhancedNameGenerator("../../data")
	if eng == nil {
		t.Fatal("NewEnhancedNameGenerator failed")
	}

	tests := []struct {
		nayin      string
		nameWuxing string
		desc       string
	}{
		{nayin: "海中金", nameWuxing: "金", desc: "纳音金+名字金→相同"},
		{nayin: "海中金", nameWuxing: "土", desc: "纳音金+名字土→土生金"},
		{nayin: "炉中火", nameWuxing: "木", desc: "纳音火+名字木→木生火"},
		{nayin: "涧下水", nameWuxing: "火", desc: "纳音水+名字火→水克火"},
		{nayin: "", nameWuxing: "金", desc: "空纳音→默认评分"},
		{nayin: "大林木", nameWuxing: "", desc: "空名字五行→默认评分"},
	}

	for _, tc := range tests {
		score, analysis := eng.calculateNayinScore(tc.nayin, tc.nameWuxing)
		if score < 30 || score > 100 {
			t.Errorf("calculateNayinScore(%q, %q) = %.1f, score out of range [30,100] - %s", tc.nayin, tc.nameWuxing, score, tc.desc)
		}
		if score > 0 && analysis == "" {
			t.Errorf("calculateNayinScore(%q, %q) returned empty analysis - %s", tc.nayin, tc.nameWuxing, tc.desc)
		}
		// 纳音相同时应≥85
		if tc.nayin != "" && tc.nameWuxing != "" && extractNayinWuxing(tc.nayin) == tc.nameWuxing && score < 85 {
			t.Errorf("calculateNayinScore(%q, %q) = %.1f, want >=85 for same wuxing - %s", tc.nayin, tc.nameWuxing, score, tc.desc)
		}
	}
}

// TestScoreTiandao_ToneTransition 验证姓氏→名字声调过渡评分
func TestScoreTiandao_ToneTransition(t *testing.T) {
	eng := NewEnhancedNameGenerator("../../data")
	if eng == nil {
		t.Fatal("NewEnhancedNameGenerator failed")
	}

	// 声调不同（仄+平）：姓氏4声+名字1声 → 平仄不同，过渡加分
	namePingZe := Name{
		Surname:   "赵",
		GivenName: "天华",
		Pinyin:    "zhao4 tian1 hua2",
		Strokes:   15,
	}
	scoreDiff := eng.scoreTiandao(namePingZe)

	// 同声调（平+平）：姓氏2声+名字1声 → 平仄相同（都是平），不同韵母，不加不扣
	nameSame := Name{
		Surname:   "王",
		GivenName: "天明",
		Pinyin:    "wang2 tian1 ming2",
		Strokes:   15,
	}
	scoreSame := eng.scoreTiandao(nameSame)

	// 仄→平应该比平→平得分高（有过渡加分）
	if scoreDiff <= scoreSame {
		t.Errorf("仄→平(赵天华) scoreTiandao=%.1f 应高于 平→平(王天明) scoreTiandao=%.1f", scoreDiff, scoreSame)
	}

	// 同声调测试：姓氏1声+名字1声 → 同声调，应扣分
	nameSameTone := Name{
		Surname:   "江",
		GivenName: "天华",
		Pinyin:    "jiang1 tian1 hua2",
		Strokes:   15,
	}
	scoreSameTone := eng.scoreTiandao(nameSameTone)
	if scoreSameTone > scoreSame {
		t.Errorf("同声调(江天华) scoreTiandao=%.1f 应低于 平同调不同(王天明) scoreTiandao=%.1f", scoreSameTone, scoreSame)
	}

	t.Logf("仄→平(赵天华)=%.1f, 平→平(王天明)=%.1f, 同声调(江天华)=%.1f", scoreDiff, scoreSame, scoreSameTone)
}

// TestNayinScoreInNameAnalysis 验证纳音评分集成到NameAnalysis中
func TestNayinScoreInNameAnalysis(t *testing.T) {
	eng := NewEnhancedNameGenerator("../../data")
	if eng == nil {
		t.Fatal("NewEnhancedNameGenerator failed")
	}

	name := Name{
		Surname:   "王",
		GivenName: "明",
		Pinyin:    "wang ming",
		Wuxing:    "火",
		Strokes:   8,
	}

	analysis := eng.AnalyzeName(name, []string{"木", "火"}, "龙", "炉中火")
	if analysis.NayinScore < 30 || analysis.NayinScore > 100 {
		t.Errorf("NayinScore=%.1f out of range [30,100]", analysis.NayinScore)
	}
	if analysis.NayinAnalysis == "" {
		t.Errorf("NayinAnalysis should not be empty")
	}
	t.Logf("纳音评分: %.1f, 分析: %s", analysis.NayinScore, analysis.NayinAnalysis)
}
