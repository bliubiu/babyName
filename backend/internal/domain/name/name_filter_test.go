package name

import (
	"strings"
	"testing"
)

func TestNewNameFilter(t *testing.T) {
	nf := NewNameFilter()
	if nf == nil {
		t.Fatal("NewNameFilter() returned nil")
	}
}

// --- 硬过滤测试 ---

func TestFilter_Hard_RareChars(t *testing.T) {
	nf := NewNameFilter()

	// 正常名字应通过（用 HanziData 中笔画 <= 25 的字）
	result := nf.Filter(Name{FullName: "王一", GivenName: "一"})
	if !result.Valid {
		t.Errorf("'王一' should be valid, got issues: %v", result.Issues)
	}
}

func TestFilter_Hard_Homophone(t *testing.T) {
	nf := NewNameFilter()

	// 正常名字
	result := nf.Filter(Name{FullName: "王天", GivenName: "天", Pinyin: "wang tian"})
	if !result.Valid {
		t.Errorf("'王天' should be valid, got issues: %v", result.Issues)
	}

	// 包含不良字应淘汰
	badNames := []Name{
		{FullName: "王死", GivenName: "死", Pinyin: "wang si"},
		{FullName: "王亡", GivenName: "亡", Pinyin: "wang wang"},
		{FullName: "王病", GivenName: "病", Pinyin: "wang bing"},
	}
	for _, n := range badNames {
		result := nf.Filter(n)
		if result.Valid {
			t.Errorf("Filter(%q) should be invalid, got issues: %v", n.FullName, result.Issues)
		}
	}
}

func TestFilter_Hard_HistoricalCollision(t *testing.T) {
	nf := NewNameFilter()

	tests := []struct {
		fullName  string
		givenName string
		valid     bool
	}{
		{"李世民", "世民", false},
		{"诸葛亮", "诸葛亮", false},
		{"王天一", "天一", true},
	}

	for _, tc := range tests {
		result := nf.Filter(Name{FullName: tc.fullName, GivenName: tc.givenName})
		if result.Valid != tc.valid {
			t.Errorf("Filter(%q).Valid = %v, want %v; issues: %v", tc.fullName, result.Valid, tc.valid, result.Issues)
		}
	}
}

func TestFilter_Hard_NegativeMeaning(t *testing.T) {
	nf := NewNameFilter()

	tests := []struct {
		fullName  string
		givenName string
		valid     bool
	}{
		{"王哀", "哀", false},
		{"王愁", "愁", false},
		{"王恨", "恨", false},
		{"王天", "天", true},
		{"王大", "大", true},
	}

	for _, tc := range tests {
		result := nf.Filter(Name{FullName: tc.fullName, GivenName: tc.givenName})
		if result.Valid != tc.valid {
			t.Errorf("Filter(%q).Valid = %v, want %v; issues: %v", tc.fullName, result.Valid, tc.valid, result.Issues)
		}
	}
}

func TestFilter_Hard_DuplicateChar(t *testing.T) {
	nf := NewNameFilter()

	tests := []struct {
		fullName  string
		givenName string
		valid     bool
	}{
		{"王天天", "天天", false},
		{"王大大", "大大", false},
		{"王天", "天", true},
		{"王大", "大", true},
	}

	for _, tc := range tests {
		result := nf.Filter(Name{FullName: tc.fullName, GivenName: tc.givenName})
		if result.Valid != tc.valid {
			t.Errorf("Filter(%q).Valid = %v, want %v; issues: %v", tc.fullName, result.Valid, tc.valid, result.Issues)
		}
	}
}

func TestFilter_Hard_BadFullName(t *testing.T) {
	nf := NewNameFilter()

	tests := []struct {
		fullName  string
		givenName string
		valid     bool
	}{
		{"杜子腾", "子腾", false},
		{"史珍香", "珍香", false},
		{"王天一", "天一", true},
	}

	for _, tc := range tests {
		result := nf.Filter(Name{FullName: tc.fullName, GivenName: tc.givenName})
		if result.Valid != tc.valid {
			t.Errorf("Filter(%q).Valid = %v, want %v; issues: %v", tc.fullName, result.Valid, tc.valid, result.Issues)
		}
	}
}

// --- 软检查测试 ---

func TestFilter_Soft_PositiveMeaning(t *testing.T) {
	nf := NewNameFilter()

	// 不含积极含义字的双字名 - 软提醒不影响 valid
	result := nf.Filter(Name{FullName: "王中中", GivenName: "中中"})
	_ = result
}

func TestFilter_Soft_StructureBalance(t *testing.T) {
	nf := NewNameFilter()

	// 双字名 — 只测试不 panic
	result := nf.Filter(Name{FullName: "王天人", GivenName: "天人", Surname: "王"})
	_ = result
}

func TestFilter_Edge_EmptyGivenName(t *testing.T) {
	nf := NewNameFilter()

	// 防御测试：空名字不能 panic
	result := nf.Filter(Name{FullName: "王", GivenName: ""})
	_ = result
}

func TestFilter_Valid_Names(t *testing.T) {
	nf := NewNameFilter()

	tests := []struct {
		fullName  string
		givenName string
	}{
		{"王天一", "天一"},
		{"李大文", "大文"},
		{"张中人", "中人"},
		{"刘文天", "文天"},
	}

	for _, tc := range tests {
		result := nf.Filter(Name{FullName: tc.fullName, GivenName: tc.givenName})
		if !result.Valid {
			t.Errorf("Filter(%q) should be valid, got issues: %v", tc.fullName, result.Issues)
		}
	}
}

// --- 跨音节连读不良拼音测试 ---

func TestCheckCrossSyllableBadReading_Hard(t *testing.T) {
	nf := NewNameFilter()

	tests := []struct {
		name    string
		pinyin  string
		want    int // 期望发现的 hard 级 issue 数量
	}{
		{name: "王霸-王八", pinyin: "wang ba", want: 1},             // wangba → 王八
		{name: "杨伟-阳痿", pinyin: "yang wei", want: 1},            // yangwei
		{name: "刘产-流产", pinyin: "liu chan", want: 1},            // liuchan
		{name: "王八蛋",    pinyin: "wang ba dan", want: 1},         // wangbadan
		// 正常拼音不应触发
		{name: "正常-天",   pinyin: "wang tian", want: 0},
		{name: "正常-明",   pinyin: "zhang ming", want: 0},
		{name: "正常-文天", pinyin: "liu wen tian", want: 0},
		{name: "单字名",    pinyin: "wang", want: 0},
	}

	for _, tc := range tests {
		// 直接调用检测方法，绕过稀有字过滤
		issues := nf.checkCrossSyllableBadReading(Name{
			FullName:  "王" + tc.name,
			GivenName: tc.name,
			Pinyin:    tc.pinyin,
		})
		hardCount := 0
		for _, issue := range issues {
			for _, bc := range nf.badCrossSyllablePinyins {
				if bc.severity == "hard" && strings.Contains(issue, bc.desc) {
					hardCount++
					break
				}
			}
		}
		if hardCount != tc.want {
			t.Errorf("checkCrossSyllableBadReading(%q) hard issues = %d, want %d; all issues: %v",
				tc.pinyin, hardCount, tc.want, issues)
		}
	}
}

func TestCheckCrossSyllableBadReading_Soft(t *testing.T) {
	nf := NewNameFilter()

	// 软提醒级连读
	issues := nf.checkCrossSyllableBadReading(Name{
		FullName:  "杜子腾",
		GivenName: "子腾",
		Pinyin:    "du zi teng",
	})
	found := false
	for _, issue := range issues {
		if strings.Contains(issue, "肚子疼") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("checkCrossSyllableBadReading('du zi teng') should contain '肚子疼' warning, got: %v", issues)
	}
}

func TestCheckCrossSyllableBadReading_Normal(t *testing.T) {
	nf := NewNameFilter()

	normalPinyins := []string{
		"wang hao ran",
		"li ming yuan",
		"chen si rui",
		"zhao zi xuan",
		"li wen tian",
		"zhang yi ming",
	}

	for _, p := range normalPinyins {
		issues := nf.checkCrossSyllableBadReading(Name{
			FullName:  "王test",
			GivenName: "test",
			Pinyin:    p,
		})
		if len(issues) > 0 {
			t.Errorf("checkCrossSyllableBadReading(%q) should have no issues, got: %v", p, issues)
		}
	}
}
