package fate

import (
	"testing"
)

// TestCheckBadHomophoneSelfExempt 精确匹配语义（自 P1 修复后）
//
// 新语义：仅当 selfChar == BadHomophones.Word 时才命中。
// 之前按拼音匹配会误杀所有同音好字（思 si→死、秀 xiu→朽 等）。
func TestCheckBadHomophoneSelfExempt(t *testing.T) {
	// 1. 拼音 si + 本字 "死" → 命中（本字就是谐音词"死"）
	if hit, _ := CheckBadHomophone("si", "死"); !hit {
		t.Error("CheckBadHomophone(si, 死) 应命中（本字就是谐音词），实际未命中")
	}
	// 2. 拼音 si + 本字 "思" → 不命中（思≠死，同音不视为不吉）
	if hit, _ := CheckBadHomophone("si", "思"); hit {
		t.Error("CheckBadHomophone(si, 思) 期望不命中（思≠死），实际命中")
	}
	// 3. 拼音 si + 本字 "王" → 不命中（无 si → 王 的谐音条目）
	if hit, _ := CheckBadHomophone("si", "王"); hit {
		t.Error("CheckBadHomophone(si, 王) 应不命中，实际命中")
	}
	// 4. 拼音 zang + 本字 "脏" → 命中（本字就是谐音词"脏"）
	if hit, _ := CheckBadHomophone("zang", "脏"); !hit {
		t.Error("CheckBadHomophone(zang, 脏) 应命中（本字就是谐音词），实际未命中")
	}
	// 5. 拼音 zang + 本字 "藏" → 不命中（同音好字不视为不吉）
	if hit, _ := CheckBadHomophone("zang", "藏"); hit {
		t.Error("CheckBadHomophone(zang, 藏) 期望不命中（藏≠脏），实际命中")
	}
	// 6. 兼容模式（无 selfChar）：按拼音匹配（旧行为）
	if hit, _ := CheckBadHomophone("si"); !hit {
		t.Error("CheckBadHomophone(si) 无 selfChar 应命中（旧行为兼容），实际未命中")
	}
	// 7. selfChar="" 视为未传，触发兼容模式
	if hit, _ := CheckBadHomophone("si", ""); !hit {
		t.Error("CheckBadHomophone(si, \"\") 应按拼音命中（selfChar 空），实际未命中")
	}
}

// TestCheckAllBadHomophonesPairedArgs CheckAllBadHomophones 偶数参数解析
//
// 新签名：pinyinCharPairs 偶数长度，[pinyin, char, pinyin, char, ...]
// 与 CheckBadHomophone 联动：精确匹配（仅当 char == 谐音词时命中）。
func TestCheckAllBadHomophonesPairedArgs(t *testing.T) {
	// "死 si" 拼作名字 → 命中
	hit, _ := CheckAllBadHomophones("si", "死")
	if !hit {
		t.Error("「死」应命中不吉谐音，实际未命中")
	}
	// "思 si" 拼作名字 → 不命中（思≠死）
	hit, _ = CheckAllBadHomophones("si", "思")
	if hit {
		t.Error("「思」不应命中不吉谐音，实际命中")
	}
	// "藏 zang" 拼作名字 → 不命中（藏≠脏）
	hit, _ = CheckAllBadHomophones("zang", "藏")
	if hit {
		t.Error("「藏」不应命中不吉谐音，实际命中")
	}
}

// TestNewExcellentTableWithCap 自适应容量构造
//
// 验证 NewExcellentTableWithCap(n) 创建的表能正确堆入 <=n 条，且满后按堆顶最小分替换。
func TestNewExcellentTableWithCap(t *testing.T) {
	cap := 5
	tbl := NewExcellentTableWithCap(cap)
	if tbl.cap != cap {
		t.Errorf("cap = %d, 期望 %d", tbl.cap, cap)
	}

	// 推入 cap+2 条，分数递增 → 应只保留分数最高的 cap 条
	for i := 0; i < cap+2; i++ {
		tbl.TryPush(ExcellentEntry{
			Char1:  "你",
			Char2:  string(rune('a' + i)),
			Score:  float64(i),
			Grade:  "中",
			WuXing1: "水",
			WuXing2: "木",
		})
	}
	tbl.Finalize()
	if tbl.Len() != cap {
		t.Errorf("Finalize 后 Len = %d, 期望 %d", tbl.Len(), cap)
	}
	// TopN 第 1 条应是分数最高的那条（i=cap+1）
	top := tbl.TopN(1)
	if len(top) > 0 && top[0].Score != float64(cap+1) {
		t.Errorf("Top1.Score = %f, 期望 %f", top[0].Score, float64(cap+1))
	}

	// 边界：capacity <= 0 回退默认容量
	tbl2 := NewExcellentTableWithCap(0)
	if tbl2.cap != excellentTableCapacity {
		t.Errorf("NewExcellentTableWithCap(0).cap = %d, 期望 %d (default)",
			tbl2.cap, excellentTableCapacity)
	}
	tbl3 := NewExcellentTableWithCap(-100)
	if tbl3.cap != excellentTableCapacity {
		t.Errorf("NewExcellentTableWithCap(-100).cap = %d, 期望 %d (default)",
			tbl3.cap, excellentTableCapacity)
	}
}

// TestEnsureWuxingDiversityMapDedup 去重改 map 后行为等价
//
// 之前用嵌套 for + 比对去重，O(N²)；
// 改用 map[string后bool] 后等价结果但 O(N)。
func TestEnsureWuxingDiversityMapDedup(t *testing.T) {
	entries := []ExcellentEntry{
		{Char1: "甲", Char2: "木", WuXing1: "金", WuXing2: "金", Score: 95},
		{Char1: "乙", Char2: "木", WuXing1: "金", WuXing2: "木", Score: 90},
		{Char1: "丙", Char2: "火", WuXing1: "金", WuXing2: "水", Score: 85},
		{Char1: "丁", Char2: "土", WuXing1: "木", WuXing2: "木", Score: 80},
		{Char1: "戊", Char2: "土", WuXing1: "木", WuXing2: "火", Score: 75},
		{Char1: "己", Char2: "土", WuXing1: "木", WuXing2: "土", Score: 70},
	}
	result := ensureWuxingDiversity(entries, 4)
	if len(result) != 4 {
		t.Fatalf("len(result) = %d, 期望 4", len(result))
	}
	// 第一轮按五行组合多样性：入金/金木/金水/木木 各 1 个
	wantChars := map[string]bool{"甲木": true, "乙木": true, "丙火": true, "丁土": true}
	for _, e := range result {
		if !wantChars[e.Char1+e.Char2] {
			t.Errorf("第一轮结果应包含 %s%v, 实际得到 %v", e.Char1, e.Char2, result)
		}
	}
}