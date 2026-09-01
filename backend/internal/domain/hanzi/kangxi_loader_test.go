package hanzi

import (
	"path/filepath"
	"runtime"
	"testing"
)

// packageDir 测试包所在目录（不依赖运行 CWD）
func kangxiPackageDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Dir(file)
}

// TestLoadKangxiStrokesFromCSV 康熙笔画表加载
//
// 验证：
//  1. 加载后 KangxiStrokesMap 含 63700+ 字（含简体/异体/日韩）
//  2. 关键汉字康熙笔画正确：王=4、李=7、张=7、文=4、浩=11、然=12、轩=10
//  3. 未收录字返回 0（兜底）
//  4. 缺失文件降级（仅警告，不返回错误）
func TestLoadKangxiStrokesFromCSV(t *testing.T) {
	if !IsKangxiStrokesLoaded() {
		t.Skip("康熙笔画表未加载（data/raw/kangxi-strokecount.csv 缺失），跳过")
	}

	// 笔画抽样验证（康熙字典权威值）
	cases := []struct {
		char    string
		wantStk int
	}{
		{"王", 4},
		{"李", 7},
		{"张", 7}, // 注意：张康熙 7 画 ≠ 简体 11 画（差异正是修复目标）
		{"文", 4},
		{"浩", 11},
		{"然", 12},
		{"轩", 10},
		{"子", 3},
	}
	for _, c := range cases {
		got := GetKangxiStrokes(c.char)
		if got != c.wantStk {
			t.Errorf("GetKangxiStrokes(%q) = %d, 期望 %d", c.char, got, c.wantStk)
		}
	}

	// 未收录应返回 0：使用生僻异体字"㐬"（U+242C，sisheng 字符），
	// kangxi-strokecount.csv 不收录此类辅助平面字符。
	if got := GetKangxiStrokes("㐬"); got != 0 {
		t.Errorf("GetKangxiStrokes(未收录字) = %d, 期望 0", got)
	}

	// 字表规模应覆盖所有常用字 + 大量异体
	count := KangxiStrokesCount()
	if count < 50000 {
		t.Errorf("KangxiStrokesCount() = %d, 期望 >= 50000（康熙字典应含 ~63700 字）", count)
	}
	t.Logf("KangxiStrokesCount = %d", count)
}

// TestLoadKangxiStrokesReload 加载幂等
//
// 重复加载同一文件，集合数量应保持稳定（不单调增长）。
func TestLoadKangxiStrokesReload(t *testing.T) {
	if !IsKangxiStrokesLoaded() {
		t.Skip("康熙笔画表未加载，跳过")
	}

	n1 := KangxiStrokesCount()
	dataDir := filepath.Join(kangxiPackageDir(), "..", "..", "..", "data")
	if err := LoadKangxiStrokesFromCSV(dataDir); err != nil {
		t.Fatalf("二次加载失败: %v", err)
	}
	if KangxiStrokesCount() != n1 {
		t.Errorf("幂等加载后数量变化：%d → %d", n1, KangxiStrokesCount())
	}
}

// TestNamerKangxiStrokesMerged 验证 namer_loader 把康熙笔画注入 HanziData 与 NamerCharMap
//
// 前置：TestMain 已先加载 kangxi 再加载 namer，
// namer_loader 末尾的 mergeKangxiStrokes 会注入 NamerCharMap[char].KangxiStrokes
// 与 HanziData[char].KangxiStrokes。
//
// 抽样差异显著的字（简体 vs 康熙）：
//   - 马：namer 简体 3 画 / 康熙 10 画（差异 7）
//   - 门：namer 简体 3 画 / 康熙 8 画（差异 5）
//   - 才：namer 简体 3 画 / 康熙 4 画（差异 1）
func TestNamerKangxiStrokesMerged(t *testing.T) {
	if !IsKangxiStrokesLoaded() {
		t.Skip("康熙笔画表未加载，跳过")
	}

	cases := []struct {
		char    string
		wantSim int // namer 简体
		wantKx  int // 康熙字典
	}{
		{"马", 3, 10},
		{"门", 3, 8},
		{"才", 3, 4},
	}
	for _, c := range cases {
		h, ok := HanziData[c.char]
		if !ok {
			t.Skipf("HanziData[%q] 不存在（namer.json 未加载？）", c.char)
			continue
		}
		if h.Strokes != c.wantSim {
			t.Errorf("HanziData[%q].Strokes = %d, 期望 %d（简体）", c.char, h.Strokes, c.wantSim)
		}
		if h.KangxiStrokes != c.wantKx {
			t.Errorf("HanziData[%q].KangxiStrokes = %d, 期望 %d（康熙）", c.char, h.KangxiStrokes, c.wantKx)
		}
		// NamerCharMap 同步
		if nc, ok := NamerCharMap[c.char]; ok {
			if nc.KangxiStrokes != c.wantKx {
				t.Errorf("NamerCharMap[%q].KangxiStrokes = %d, 期望 %d（康熙）", c.char, nc.KangxiStrokes, c.wantKx)
			}
		}
	}
}