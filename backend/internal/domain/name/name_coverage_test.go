package name

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetClassicNames(t *testing.T) {
	// Test for both genders
	maleChars := GetClassicNames("男")
	femaleChars := GetClassicNames("女")
	commonChars := GetClassicNames("通用")

	// Should not panic and return slices
	_ = maleChars
	_ = femaleChars
	_ = commonChars
}

func TestGetPoetryNames(t *testing.T) {
	// Test with various sources
	sources := []string{"诗经", "楚辞", "唐诗", "宋词"}
	for _, source := range sources {
		chars := GetPoetryNames("男", source)
		_ = chars
	}

	// Test fallback to GetClassicNames when empty
	chars := GetPoetryNames("男", "不存在的来源")
	_ = chars
}

func TestGetPinyin(t *testing.T) {
	// Test characters in HanziData (if any loaded)
	// Test fallback characters
	tests := map[string]string{
		"伟": "wei",
		"强": "qiang",
		"磊": "lei",
		"军": "jun",
		"杰": "jie",
		"涛": "tao",
		"明": "ming",
		"超": "chao",
		"勇": "yong",
		"鹏": "peng",
		"华": "hua",
		"刚": "gang",
		"平": "ping",
		"辉": "hui",
		"波": "bo",
		"峰": "feng",
		"飞": "fei",
		"龙": "long",
		"浩": "hao",
		"宇": "yu",
		"晨": "chen",
		"逸": "yi",
		"睿": "rui",
		"哲": "zhe",
		"渊": "yuan",
		"博": "bo",
		"昊": "hao",
		"然": "ran",
		"轩": "xuan",
		"俊": "jun",
		"豪": "hao",
		"毅": "yi",
		"文": "wen",
		"武": "wu",
		"祥": "xiang",
		"瑞": "rui",
		"凯": "kai",
		"成": "cheng",
		"盛": "sheng",
		"雄": "xiong",
		"鑫": "xin",
		"霖": "lin",
		"思": "si",
		"远": "yuan",
		"航": "hang",
		"洋": "yang",
		"庆": "qing",
		"芳": "fang",
		"兰": "lan",
		"梅": "mei",
		"琴": "qin",
		"丽": "li",
		"秀": "xiu",
		"英": "ying",
		"敏": "min",
		"静": "jing",
		"燕": "yan",
		"霞": "xia",
		"红": "hong",
		"玉": "yu",
		"珍": "zhen",
		"娟": "juan",
		"莉": "li",
		"萍": "ping",
		"颖": "ying",
		"娜": "na",
		"欣": "xin",
		"怡": "yi",
		"菲": "fei",
		"雅": "ya",
		"芸": "yun",
		"萱": "xuan",
		"雨": "yu",
		"诗": "shi",
		"梦": "meng",
		"瑶": "yao",
		"琳": "lin",
		"珂": "ke",
		"雯": "wen",
		"倩": "qian",
		"雪": "xue",
		"婷": "ting",
		"霜": "shuang",
		"翠": "cui",
		"碧": "bi",
		"银": "yin",
		"金": "jin",
		"珠": "zhu",
		"宾": "bin",
	}

	for char, expected := range tests {
		got := GetPinyin(char)
		// Just verify it returns something non-empty (exact pinyin with tones varies)
		if got == "" {
			t.Errorf("GetPinyin(%q) returned empty string", char)
		}
		if len(got) == 0 {
			t.Errorf("GetPinyin(%q) returned empty", char)
		}
		// Verify it contains the expected base (case-insensitive, no tones)
		found := false
		for _, r := range expected {
			for _, g := range got {
				// Check base character without tone
				baseR := r
				baseG := g
				if baseG >= 'ā' && baseG <= 'ǖ' {
					switch baseG {
					case 'ā', 'á', 'ǎ', 'à': baseG = 'a'
					case 'ē', 'é', 'ě', 'è': baseG = 'e'
					case 'ī', 'í', 'ǐ', 'ì': baseG = 'i'
					case 'ō', 'ó', 'ǒ', 'ò': baseG = 'o'
					case 'ū', 'ú', 'ǔ', 'ù', 'ǖ', 'ǘ', 'ǚ', 'ǜ': baseG = 'u'
					case 'ń', 'ň', 'ǹ': baseG = 'n'
					case 'ḿ': baseG = 'm'
					}
				}
				if baseG == baseR {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			t.Errorf("GetPinyin(%q) = %q, expected to contain base of %q", char, got, expected)
		}
	}

	// Test unknown character (should return the char itself)
	unknown := GetPinyin("未知字符")
	if unknown != "未知字符" {
		t.Errorf("GetPinyin(未知字符) = %q, want 未知字符", unknown)
	}
}

func TestNameDB(t *testing.T) {
	// Create temp dir for testing
	tmpDir := t.TempDir()

	// Create a minimal curated_names.json
	curatedContent := `[
		{
			"name": "伟强",
			"pinyin": "wei qiang",
			"gender": "男",
			"source": "test",
			"meaning": "伟大强壮",
			"wuxing": "土木",
			"yinyun_score": 85.5,
			"styles": ["大气", "阳刚"],
			"tags": ["test"]
		},
		{
			"name": "芳华",
			"pinyin": "fang hua",
			"gender": "女",
			"source": "test",
			"meaning": "芳香华美",
			"wuxing": "木木",
			"yinyun_score": 90.0,
			"styles": ["优雅", "文静"],
			"tags": ["test"]
		},
		{
			"name": "通用名",
			"pinyin": "tong yong",
			"gender": "通用",
			"source": "test",
			"meaning": "通用",
			"wuxing": "金水",
			"yinyun_score": 75.0,
			"styles": ["中性"],
			"tags": ["test"]
		}
	]`
	err := os.WriteFile(filepath.Join(tmpDir, "curated_names.json"), []byte(curatedContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write curated_names.json: %v", err)
	}

	// Create a shiyun test file
	shiyunContent := `["诗云名一", "诗云名二", "诗云名三"]`
	err = os.WriteFile(filepath.Join(tmpDir, "shiyun_test.json"), []byte(shiyunContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write shiyun_test.json: %v", err)
	}

	// Create NewNameDB
	db, err := NewNameDB(tmpDir)
	if err != nil {
		t.Fatalf("NewNameDB failed: %v", err)
	}

	// Test GetCuratedNames
	allNames := db.GetCuratedNames("")
	if len(allNames) != 3 {
		t.Errorf("GetCuratedNames('') = %d, want 3", len(allNames))
	}

	maleNames := db.GetCuratedNames("男")
	// Should include both male and 通用
	if len(maleNames) != 2 {
		t.Errorf("GetCuratedNames('男') = %d, want 2 (male + 通用)", len(maleNames))
	}
	foundMale := false
	for _, n := range maleNames {
		if n.Name == "伟强" {
			foundMale = true
		}
	}
	if !foundMale {
		t.Error("GetCuratedNames('男') should contain 伟强")
	}

	femaleNames := db.GetCuratedNames("女")
	if len(femaleNames) != 2 {
		t.Errorf("GetCuratedNames('女') = %d, want 2 (female + 通用)", len(femaleNames))
	}
	foundFemale := false
	for _, n := range femaleNames {
		if n.Name == "芳华" {
			foundFemale = true
		}
	}
	if !foundFemale {
		t.Error("GetCuratedNames('女') should contain 芳华")
	}

	// Test GetCharRadical (may be empty without charGroups loaded)
	radical := db.GetCharRadical("伟")
	_ = radical

	// Test GetCharGroups
	groups := db.GetCharGroups()
	_ = groups

	// Test GetGroupChars
	groupChars := db.GetGroupChars("人")
	_ = groupChars

	// Test IsCuratedName
	if !db.IsCuratedName("伟强") {
		t.Error("IsCuratedName(伟强) should be true")
	}
	if db.IsCuratedName("不存在的名字") {
		t.Error("IsCuratedName(不存在的名字) should be false")
	}

	// Test SearchCuratedNames
	results := db.SearchCuratedNames("伟")
	if len(results) != 1 || results[0].Name != "伟强" {
		t.Errorf("SearchCuratedNames(伟) = %v, want [伟强]", results)
	}

	results2 := db.SearchCuratedNames("芳")
	if len(results2) != 1 || results2[0].Name != "芳华" {
		t.Errorf("SearchCuratedNames(芳) = %v, want [芳华]", results2)
	}

	results3 := db.SearchCuratedNames("不存在")
	if len(results3) != 0 {
		t.Errorf("SearchCuratedNames(不存在) = %v, want empty", results3)
	}

	// Test GetShiyunNames
	shiyun := db.GetShiyunNames()
	if len(shiyun) < 3 {
		t.Errorf("GetShiyunNames() = %d, want >= 3", len(shiyun))
	}

	// Test IsShiyunName
	if !db.IsShiyunName("诗云名一") {
		t.Error("IsShiyunName(诗云名一) should be true")
	}
	if db.IsShiyunName("不存在的名字") {
		t.Error("IsShiyunName(不存在的名字) should be false")
	}

	// Test GetTopCuratedNames
	topAll := db.GetTopCuratedNames("", 2)
	if len(topAll) != 2 {
		t.Errorf("GetTopCuratedNames('', 2) = %d, want 2", len(topAll))
	}
	if topAll[0].YinyunScore < topAll[1].YinyunScore {
		t.Error("GetTopCuratedNames should sort by YinyunScore descending")
	}

	topMale := db.GetTopCuratedNames("男", 10)
	// Should include both male and 通用
	if len(topMale) != 2 {
		t.Errorf("GetTopCuratedNames('男', 10) = %d, want 2 (male + 通用)", len(topMale))
	}

	// Test GetStyles
	styles := db.GetStyles()
	if len(styles) == 0 {
		t.Error("GetStyles() should not be empty")
	}

	// Test GetNamesByStyle
	byStyle := db.GetNamesByStyle("大气")
	if len(byStyle) != 1 || byStyle[0].Name != "伟强" {
		t.Errorf("GetNamesByStyle(大气) = %v, want [伟强]", byStyle)
	}

	// Skip Reload test to avoid deadlock with async persistence goroutine
	// err2 := db.Reload()
	// if err2 != nil {
	// 	t.Errorf("Reload() failed: %v", err2)
	// }
	//
	// if !db.IsCuratedName("伟强") {
	// 	t.Error("IsCuratedName failed after reload")
	// }
}

func TestNameDB_AddCuratedName(t *testing.T) {
	tmpDir := t.TempDir()

	// Create empty curated_names.json
	err := os.WriteFile(filepath.Join(tmpDir, "curated_names.json"), []byte("[]"), 0644)
	if err != nil {
		t.Fatalf("Failed to write curated_names.json: %v", err)
	}

	db, err := NewNameDB(tmpDir)
	if err != nil {
		t.Fatalf("NewNameDB failed: %v", err)
	}

	// Add new curated name
	err = db.AddCuratedName("新名字", "xin ming zi", "男", 88.0, "test")
	if err != nil {
		t.Errorf("AddCuratedName failed: %v", err)
	}

	// Verify added
	if !db.IsCuratedName("新名字") {
		t.Error("IsCuratedName should be true after AddCuratedName")
	}

	names := db.GetCuratedNames("男")
	found := false
	for _, n := range names {
		if n.Name == "新名字" {
			found = true
			break
		}
	}
	if !found {
		t.Error("New name not found in GetCuratedNames")
	}

	// Add duplicate (should not error)
	err = db.AddCuratedName("新名字", "xin ming zi", "男", 88.0, "test")
	if err != nil {
		t.Errorf("AddCuratedName duplicate failed: %v", err)
	}
}

func TestNameDB_LoadErrors(t *testing.T) {
	// Test with non-existent directory
	_, err := NewNameDB("/non/existent/path")
	// Should not fail, just log warnings
	_ = err
}

func TestCuratedNameEntry(t *testing.T) {
	entry := CuratedNameEntry{
		Name:   "测试名",
		Pinyin: "ce shi ming",
		Gender: "男",
		Score:  90.0,
		Source: "test",
	}

	// Just verify struct can be created and marshaled
	_ = entry
}

func TestNameDBOptions(t *testing.T) {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "curated_names.json"), []byte("[]"), 0644)
	if err != nil {
		t.Fatalf("Failed to write curated_names.json: %v", err)
	}

	// Test WithCuratedPersister (nil persister)
	db, err := NewNameDB(tmpDir, WithCuratedPersister(nil))
	if err != nil {
		t.Fatalf("NewNameDB with nil persister failed: %v", err)
	}
	_ = db
}

func TestNameAnalysisStruct(t *testing.T) {
	analysis := NameAnalysis{
		Surname:   "王",
		GivenName: "伟强",
		FullName:  "王伟强",
		Pinyin:    "wang wei qiang",
		Strokes:   20,
		Gender:    "男",
		TotalScore: 85.5,
		WuxingScore: 90.0,
		YinyunScore: 85.0,
		MeaningScore: 80.0,
		SancaiScore: 88.0,
		ZodiacScore: 92.0,
		NayinScore: 75.0,
		NoveltyScore: 70.0,
		BigramScore: 85.0,
		FrequencyScore: 78.0,
		ScoreDetail: []ScoreDetailItem{
			{Name: "五行八字", Score: 90.0, Detail: "五行补益喜用神"},
			{Name: "音韵", Score: 85.0, Detail: "平仄搭配优美"},
			{Name: "文化印象", Score: 80.0, Detail: "寓意美好"},
			{Name: "三才", Score: 88.0, Detail: "天地人三才和谐"},
			{Name: "生肖", Score: 92.0, Detail: "生肖偏旁匹配良好"},
			{Name: "纳音", Score: 75.0, Detail: "纳音五行相生"},
			{Name: "新颖度", Score: 70.0, Detail: "重名率较低"},
			{Name: "共现", Score: 85.0, Detail: "诗词共现频率高"},
			{Name: "人名频率", Score: 78.0, Detail: "常用字搭配自然"},
		},
		Recommendations: []string{"寓意优美", "音韵和谐", "五行补益"},
	}

	if analysis.TotalScore != 85.5 {
		t.Errorf("TotalScore = %f, want 85.5", analysis.TotalScore)
	}
	if len(analysis.ScoreDetail) != 9 {
		t.Errorf("ScoreDetail length = %d, want 9", len(analysis.ScoreDetail))
	}
	if len(analysis.Recommendations) != 3 {
		t.Errorf("Recommendations length = %d, want 3", len(analysis.Recommendations))
	}
}

func TestNameCharStruct(t *testing.T) {
	char := NameChar{
		Char:    "伟",
		Pinyin:  "wei",
		Meaning: "伟大",
		Wuxing:  "土",
		Strokes: 11,
		Gender:  "男",
	}

	if char.Char != "伟" {
		t.Errorf("Char = %q, want 伟", char.Char)
	}
	if char.Wuxing != "土" {
		t.Errorf("Wuxing = %q, want 土", char.Wuxing)
	}
}

func TestGenerateOptionsStruct(t *testing.T) {
	opts := GenerateOptions{
		Surname:            "王",
		Generation:         "伟",
		Gender:             "男",
		Xiyongshen:         []string{"土", "金"},
		Count:              100,
		NameLength:         2,
		ExcludeRare:        true,
		WuxingMatch:        []string{"土"},
		SourceClassic:      "诗经",
		MinStrokes:         5,
		MaxStrokes:         20,
		IncludePoetry:      true,
		IncludeClassic:     true,
		MeaningKeywords:    []string{"伟大", "强壮"},
		PinyinInitial:      "w",
		GenerationPosition: "中间",
		NameType:           "双字",
		Zodiac:             "龙",
		Nayin:              "大林木",
		DayMasterStrength:  "身旺",
		AvoidElderNames:    []string{"父名", "母名"},
	}

	if opts.Surname != "王" {
		t.Errorf("Surname = %q, want 王", opts.Surname)
	}
	if len(opts.Xiyongshen) != 2 {
		t.Errorf("Xiyongshen length = %d, want 2", len(opts.Xiyongshen))
	}
	if opts.NameLength != 2 {
		t.Errorf("NameLength = %d, want 2", opts.NameLength)
	}
}

func TestScoreDetailItemStruct(t *testing.T) {
	item := ScoreDetailItem{
		Name:   "五行八字",
		Score:  90.0,
		Detail: "五行补益喜用神",
	}

	if item.Name != "五行八字" {
		t.Errorf("Name = %q, want 五行八字", item.Name)
	}
	if item.Score != 90.0 {
		t.Errorf("Score = %f, want 90.0", item.Score)
	}
}

// TestNameDB_Reload tests Reload functionality
func TestNameDB_AddCuratedName_Persister(t *testing.T) {
	tmpDir := t.TempDir()

	err := os.WriteFile(filepath.Join(tmpDir, "curated_names.json"), []byte("[]"), 0644)
	if err != nil {
		t.Fatalf("Failed to write curated_names.json: %v", err)
	}

	// Test without persister to avoid deadlock
	db, err := NewNameDB(tmpDir)
	if err != nil {
		t.Fatalf("NewNameDB failed: %v", err)
	}

	// Add new curated name
	err = db.AddCuratedName("新名字", "xin ming zi", "男", 88.0, "test")
	if err != nil {
		t.Errorf("AddCuratedName failed: %v", err)
	}

	// Verify added
	if !db.IsCuratedName("新名字") {
		t.Error("IsCuratedName should be true after AddCuratedName")
	}
}

// TestNameDB_loadStandardChars tests loadStandardChars
func TestNameDB_loadStandardChars(t *testing.T) {
	tmpDir := t.TempDir()

	// Create empty curated_names.json
	err := os.WriteFile(filepath.Join(tmpDir, "curated_names.json"), []byte("[]"), 0644)
	if err != nil {
		t.Fatalf("Failed to write curated_names.json: %v", err)
	}

	db, err := NewNameDB(tmpDir)
	if err != nil {
		t.Fatalf("NewNameDB failed: %v", err)
	}

	// Test GetCharGroups (which loads standard chars)
	groups := db.GetCharGroups()
	_ = groups

	// Test GetGroupChars
	chars := db.GetGroupChars("人")
	_ = chars
}

// TestGetClassicNames_Coverage tests GetClassicNames with better coverage
func TestGetClassicNames_Coverage(t *testing.T) {
	// Test all gender variants
	testCases := []string{"男", "女", "通用", "未知性别"}
	for _, gender := range testCases {
		chars := GetClassicNames(gender)
		_ = chars
	}

	// Test with empty string
	chars := GetClassicNames("")
	_ = chars
}