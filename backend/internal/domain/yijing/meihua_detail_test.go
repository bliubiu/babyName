package yijing

import "testing"

// 本文件固化 docs/42 P0 修复回归：
// 1. 梅花姓名起卦口径唯一化：上卦=姓笔画%8、下卦=名总笔画%8、动爻=(姓+名)%6（余0取6）；
// 2. 应用层可获取本/变/互三卦（此前 GetHexagramByPlumBlossom 成果未接线）；
// 3. 标准梅花两数起卦取代伪随机 CastHexagramByTime；
// 4. 大衍筮法补变卦与多动爻断法（《易学启蒙》朱熹规则）。

// 金标准算例（手工推演）：
// 姓 4 画、名 3 画：上卦 4=震☳(阳阴阴)，下卦 3=离☲(阳阴阳)
// 本卦 = 震上离下 = 第 55 卦「丰」；动爻 = (4+3)%6 = 1（初爻）
// 初爻 7→8：下卦{8,8,7}=艮 → 变卦 = 震上艮下 = 第 62 卦「小过」
// 互卦 = 234爻{8,7,7}=巽 + 345爻{7,7,8}=兑 → 兑上巽下 = 第 28 卦「大过」
func TestGetMeihuaDetailByStrokes_GoldenCase(t *testing.T) {
	r := GetMeihuaDetailByStrokes(4, 3)
	if r == nil {
		t.Fatal("GetMeihuaDetailByStrokes(4,3) 返回 nil")
	}
	if r.OriginalHexagram == nil || r.OriginalHexagram.Name != "丰" || r.OriginalHexagram.Number != 55 {
		t.Fatalf("本卦应为第 55 卦「丰」，got %+v", r.OriginalHexagram)
	}
	if r.MovingYao != 1 {
		t.Fatalf("动爻应为 1（初爻），got %d", r.MovingYao)
	}
	if r.ChangedHexagram == nil || r.ChangedHexagram.Name != "小过" || r.ChangedHexagram.Number != 62 {
		t.Fatalf("变卦应为第 62 卦「小过」，got %+v", r.ChangedHexagram)
	}
	if r.InterHexagram == nil || r.InterHexagram.Name != "大过" || r.InterHexagram.Number != 28 {
		t.Fatalf("互卦应为第 28 卦「大过」，got %+v", r.InterHexagram)
	}
	// 本卦六爻：下离 {7,8,7} + 上震 {7,8,8}
	wantLines := [6]int{7, 8, 7, 7, 8, 8}
	if r.OriginalYaoLines != wantLines {
		t.Fatalf("本卦六爻 = %v，期望 %v", r.OriginalYaoLines, wantLines)
	}
	// 变卦六爻：初爻转阴 → 下艮 {8,8,7} + 上震 {7,8,8}
	wantChanged := [6]int{8, 8, 7, 7, 8, 8}
	if r.ChangedYaoLines != wantChanged {
		t.Fatalf("变卦六爻 = %v，期望 %v", r.ChangedYaoLines, wantChanged)
	}
	if r.Interpretation == "" {
		t.Fatal("综合解读不应为空")
	}
}

// TestGetMeihuaDetailByStrokes_Rule 口径规则：余 0 取 8 / 取 6
func TestGetMeihuaDetailByStrokes_Rule(t *testing.T) {
	// 姓 8 名 8：上坤 下坤 = 坤卦 #2，动爻 16%6=4
	r := GetMeihuaDetailByStrokes(8, 8)
	if r == nil || r.OriginalHexagram == nil || r.OriginalHexagram.Name != "坤" {
		t.Fatalf("姓8名8 应得「坤」，got %+v", r.OriginalHexagram)
	}
	if r.MovingYao != 4 {
		t.Fatalf("动爻应为 4，got %d", r.MovingYao)
	}

	// 姓 9 名 24：上 9%8=1 乾，下 24%8=8 坤 → 天地否 #12，动爻 33%6=3
	r2 := GetMeihuaDetailByStrokes(9, 24)
	if r2 == nil || r2.OriginalHexagram == nil || r2.OriginalHexagram.Name != "否" || r2.OriginalHexagram.Number != 12 {
		t.Fatalf("姓9名24 应得第 12 卦「否」，got %+v", r2.OriginalHexagram)
	}
	if r2.MovingYao != 3 {
		t.Fatalf("动爻应为 3，got %d", r2.MovingYao)
	}
}

// TestGetMeihuaDetailByStrokes_ConsistentWithSingle 卦体口径必须与单卦查询一致
// （GetHexagramByMeihuaName 与 GetMeihuaDetailByStrokes 同源，消除双口径漂移）
func TestGetMeihuaDetailByStrokes_ConsistentWithSingle(t *testing.T) {
	for s := 1; s <= 30; s++ {
		for g := 1; g <= 30; g++ {
			single := GetHexagramByMeihuaName(s, g)
			detail := GetMeihuaDetailByStrokes(s, g)
			if single == nil || detail == nil || detail.OriginalHexagram == nil {
				t.Fatalf("s=%d g=%d：查询返回 nil", s, g)
			}
			if single.Name != detail.OriginalHexagram.Name {
				t.Fatalf("s=%d g=%d：单卦 %s ≠ 完整梅花本卦 %s，口径漂移", s, g, single.Name, detail.OriginalHexagram.Name)
			}
		}
	}
}

// 金标准算例（数字起卦）：n1=7 n2=3
// 上卦 7=艮☶(阴阴阳)，下卦 3=离☲(阳阴阳) → 本卦 = 山火贲 #22
// 动爻 = 10%6 = 4（四爻）；四爻 8→7：上卦{7,8,7}=离 → 变卦 = 离上离下 = 离为火 #30
// 互卦 = 234爻{8,7,8}=坎 + 345爻{7,8,8}=震 → 震上坎下 = 雷水解 #40
func TestCastMeihuaByNumbers_GoldenCase(t *testing.T) {
	r := CastMeihuaByNumbers(7, 3)
	if r == nil || r.Hexagram == nil {
		t.Fatal("CastMeihuaByNumbers(7,3) 返回 nil")
	}
	if r.Hexagram.Name != "贲" || r.Hexagram.Number != 22 {
		t.Fatalf("本卦应为第 22 卦「贲」，got %+v", r.Hexagram)
	}
	if r.ChangeYao != 4 {
		t.Fatalf("动爻应为 4，got %d", r.ChangeYao)
	}
	if r.ChangedHexagram == nil || r.ChangedHexagram.Name != "离" || r.ChangedHexagram.Number != 30 {
		t.Fatalf("变卦应为第 30 卦「离」，got %+v", r.ChangedHexagram)
	}
	if r.InterHexagram == nil || r.InterHexagram.Name != "解" || r.InterHexagram.Number != 40 {
		t.Fatalf("互卦应为第 40 卦「解」，got %+v", r.InterHexagram)
	}
}

// TestCastMeihuaByNumbers_Deterministic 同参数必须同结果（确定性，无隐藏随机）
func TestCastMeihuaByNumbers_Deterministic(t *testing.T) {
	a := CastMeihuaByNumbers(5, 12)
	b := CastMeihuaByNumbers(5, 12)
	if a == nil || b == nil || a.Hexagram == nil || b.Hexagram == nil {
		t.Fatal("返回 nil")
	}
	if a.Hexagram.Name != b.Hexagram.Name || a.ChangeYao != b.ChangeYao {
		t.Fatalf("同参数两次起卦结果不同：%s/%d vs %s/%d",
			a.Hexagram.Name, a.ChangeYao, b.Hexagram.Name, b.ChangeYao)
	}
}

// TestDecideYaoFromChanges 多动爻断法（《易学启蒙》朱熹规则）：
// 0 动 → 0（用本卦卦辞）；1 动 → 动爻；2 动 → 上动爻；3 动 → 中动爻；
// 4 动 → 变卦二静爻之下者；5 动 → 变卦静爻；6 动 → 0（乾坤用九用六，余以变卦卦辞）
func TestDecideYaoFromChanges(t *testing.T) {
	tests := []struct {
		changes []int
		want    int
	}{
		{nil, 0},
		{[]int{3}, 3},
		{[]int{2, 5}, 5},
		{[]int{1, 3, 5}, 3},
		{[]int{1, 2, 4, 5}, 3},    // 静爻 3,6，取下者 3
		{[]int{2, 3, 4, 5, 6}, 1}, // 静爻 1
		{[]int{1, 2, 3, 4, 5, 6}, 0},
	}
	for _, tc := range tests {
		got := decideYaoFromChanges(tc.changes)
		if got != tc.want {
			t.Errorf("decideYaoFromChanges(%v) = %d, want %d", tc.changes, got, tc.want)
		}
	}
}

// TestCastHexagramByDayan_ChangedHexagram 大衍结果必须给出变卦与互卦，
// 且变卦与六爻翻转后的阴阳严格一致
func TestCastHexagramByDayan_ChangedHexagram(t *testing.T) {
	for i := 0; i < 50; i++ {
		r := CastHexagramByDayan()
		if r == nil || r.Hexagram == nil {
			t.Fatal("CastHexagramByDayan 返回 nil")
		}
		// 互卦恒存在（任何卦 234/345 爻必成卦）
		if r.InterHexagram == nil {
			t.Fatal("互卦不应为空")
		}
		wantInterUpper := trigramFromYaoLines(r.YaoLines[2:5])
		wantInterLower := trigramFromYaoLines(r.YaoLines[1:4])
		if r.InterHexagram.UpperTrigram != wantInterUpper || r.InterHexagram.LowerTrigram != wantInterLower {
			t.Fatalf("互卦上下卦 %d/%d，应由 345/234 爻排定的 %d/%d",
				r.InterHexagram.UpperTrigram, r.InterHexagram.LowerTrigram, wantInterUpper, wantInterLower)
		}

		if r.ChangeYao == 0 {
			if r.ChangedHexagram != nil {
				t.Fatalf("无动爻时变卦应为 nil，got %+v", r.ChangedHexagram)
			}
			continue
		}
		if r.ChangedHexagram == nil {
			t.Fatal("有动爻时变卦不应为空")
		}
		// 变卦六爻 = 本卦翻转动爻阴阳
		changed := r.YaoLines
		for _, pos := range r.ChangeYaos {
			if changed[pos-1] == 9 {
				changed[pos-1] = 8
			} else if changed[pos-1] == 6 {
				changed[pos-1] = 7
			}
		}
		wantUpper := trigramFromYaoLines(changed[3:6])
		wantLower := trigramFromYaoLines(changed[0:3])
		if r.ChangedHexagram.UpperTrigram != wantUpper || r.ChangedHexagram.LowerTrigram != wantLower {
			t.Fatalf("变卦 %s 上下卦 %d/%d，与爻翻转排定的 %d/%d 不符",
				r.ChangedHexagram.Name, r.ChangedHexagram.UpperTrigram, r.ChangedHexagram.LowerTrigram, wantUpper, wantLower)
		}
		// ChangeYao 必须是断法选定的爻（动爻或静爻位），且动爻记录完整
		if len(r.ChangeYaos) == 0 {
			t.Fatal("有动爻时 ChangeYaos 不应为空")
		}
	}
}
