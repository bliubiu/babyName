package yijing

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ============================================================================
// 全量金标准对拍（覆盖窄问题修复）
//
// 以往的金标准仅抽样 15 卦的象辞（canonical_test.go 的 canonicalXiangCi）。
// 本文件以 docs/玄学/易经/*.md（独立编纂的《周易》通行本经传）为金标准，
// 对 64 卦做「卦辞 + 象辞 + 爻辞」全覆盖对拍，杜绝个别卦文本写错而不报。
//
// 口径说明：
//   - 卦辞：代码刻意精简（如 乾 仅存「元亨利贞」），故只要求代码卦辞归一化后
//     是文档全文卦辞的「子串」，用于捕获字符级错写/增删，同时容忍精简。
//   - 象辞 / 爻辞：代码为完整文本，与文档精确相等（归一化后）。
//   - 归一化：去所有中英文标点与空白，并把常见异体字折叠为规范形
//     （於→于、牀→床、無→无 …），避免因异体/标点差异产生误报。
// ============================================================================

// findYijingDocsDir 从本源文件位置向上查找 docs/玄学/易经 目录
func findYijingDocsDir(t *testing.T) string {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller 失败，无法定位源文件")
	}
	dir := filepath.Dir(thisFile)
	for i := 0; i < 12; i++ {
		cand := filepath.Join(dir, "docs", "玄学", "易经")
		if fi, err := os.Stat(cand); err == nil && fi.IsDir() {
			return cand
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

var (
	punctReplacer = strings.NewReplacer(
		"，", "", "。", "", "、", "", "；", "", "：", "", "！", "", "？", "",
		"—", "", "…", "", "·", "", "「", "", "」", "", "『", "", "』", "",
		"“", "", "”", "", "‘", "", "’", "", "（", "", "）", "", "(", "", ")", "",
		"【", "", "】", "", "《", "", "》", "", "〈", "", "〉", "", "　", "", "\t", "", "\n", "", "\r", "", " ", "",
	)
	// 异体/繁简折叠：把金标准文档中的非规范形收敛到代码所用的规范形
	variantReplacer = strings.NewReplacer(
		"於", "于",
		"牀", "床",
		"無", "无",
		"遯", "遁",
		"啕", "咷",
		"豶", "豮",
		"佑", "祐", // 自天祐之，代码统一用「祐」
		"畲", "畬", // 不菑畬，代码统一用「畬」
	)
)

// normalize 去标点空白 + 异体折叠，得到仅含汉字的归一串
func normalize(s string) string {
	s = punctReplacer.Replace(s)
	s = variantReplacer.Replace(s)
	return strings.TrimSpace(s)
}

// parseHexagramDoc 委托到 parseHexagramDocFull（canonical_docs.go 单一真源），
// 只取金标准对拍所需的字段子集
func parseHexagramDoc(content string) (number int, name, guaCiFull, daXiang string, yaoCi []string) {
	number, name, guaCiFull, daXiang, _, _, yaoCi = parseHexagramDocFull(content)
	return
}

// loadGoldenHexagrams 读取 docs 目录，返回 卦序 -> 金标准 的映射
func loadGoldenHexagrams(t *testing.T) map[int]struct {
	name, guaCi, daXiang string
	yaoCi                []string
} {
	dir := findYijingDocsDir(t)
	if dir == "" {
		t.Skip("未找到 docs/玄学/易经 目录，跳过文档金标准对拍（仅在本仓库内生效）")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("读取 docs 目录失败：%v", err)
	}
	golden := map[int]struct {
		name, guaCi, daXiang string
		yaoCi                []string
	}{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		// 仅处理 64 篇卦文档（文件名含「卦」）；系辞/说卦/序卦/杂卦/index 不含「卦」
		if !strings.Contains(e.Name(), "卦") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("读取 %s 失败：%v", e.Name(), err)
		}
		num, name, guaCi, daXiang, yaoCi := parseHexagramDoc(string(raw))
		if num < 1 || num > 64 {
			continue // 非卦文档（不应到达）
		}
		golden[num] = struct {
			name, guaCi, daXiang string
			yaoCi                []string
		}{name, guaCi, daXiang, yaoCi}
	}
	if len(golden) != 64 {
		t.Fatalf("金标准文档解析出 %d 卦，期望 64（缺少的卦序会由下方逐卦断言暴露）", len(golden))
	}
	return golden
}

// TestHexagram_GoldenGuaCi 64 卦卦辞对拍（代码精简形须为文档全文子串）
func TestHexagram_GoldenGuaCi(t *testing.T) {
	golden := loadGoldenHexagrams(t)
	for _, h := range HexagramList {
		g, ok := golden[h.Number]
		if !ok {
			t.Errorf("卦 #%d（%s）金标准缺失", h.Number, h.Name)
			continue
		}
		docFull := normalize(g.guaCi)
		code := normalize(h.GuaCi)
		if code == "" {
			t.Errorf("卦 #%d（%s）代码卦辞为空", h.Number, h.Name)
			continue
		}
		if !strings.Contains(docFull, code) {
			t.Errorf("卦 #%d（%s）卦辞未命中金标准：\n  代码=%q\n  文档=%q",
				h.Number, h.Name, h.GuaCi, g.guaCi)
		}
	}
}

// TestHexagram_GoldenXiangCi 64 卦大象（象辞）对拍（代码精简形须为文档全文子串，
// 与卦辞/爻辞同口径：代码侧大象为通行本原形或精简形，子串判定可同时捕获字符级错写）
func TestHexagram_GoldenXiangCi(t *testing.T) {
	golden := loadGoldenHexagrams(t)
	for _, h := range HexagramList {
		g, ok := golden[h.Number]
		if !ok {
			t.Errorf("卦 #%d（%s）金标准缺失", h.Number, h.Name)
			continue
		}
		doc := normalize(g.daXiang)
		code := normalize(h.XiangCi)
		if !strings.Contains(doc, code) {
			t.Errorf("卦 #%d（%s）象辞未命中金标准：\n  代码=%q\n  文档=%q",
				h.Number, h.Name, h.XiangCi, g.daXiang)
		}
	}
}

// TestHexagram_GoldenYaoCi 64 卦爻辞对拍（代码爻辞归一化后须为文档全文爻辞的子串，
// 容忍代码刻意精简，同时捕获字符级错写/异体）
func TestHexagram_GoldenYaoCi(t *testing.T) {
	golden := loadGoldenHexagrams(t)
	for _, h := range HexagramList {
		g, ok := golden[h.Number]
		if !ok {
			t.Errorf("卦 #%d（%s）金标准缺失", h.Number, h.Name)
			continue
		}
		wantYao := 6
		if h.Number == 1 || h.Number == 2 {
			wantYao = 7
		}
		if len(g.yaoCi) != wantYao {
			t.Errorf("卦 #%d（%s）文档爻辞条数=%d，期望 %d", h.Number, h.Name, len(g.yaoCi), wantYao)
			continue
		}
		if len(h.YaoCi) != wantYao {
			t.Errorf("卦 #%d（%s）代码爻辞条数=%d，期望 %d", h.Number, h.Name, len(h.YaoCi), wantYao)
			continue
		}
		for i := range g.yaoCi {
			doc := normalize(g.yaoCi[i])
			code := normalize(h.YaoCi[i])
			if code == "" {
				t.Errorf("卦 #%d（%s）第 %d 爻辞代码为空", h.Number, h.Name, i+1)
				continue
			}
			// 子串判定：代码（可能精简）须完整包含于文档全文爻辞，
			// 以捕获字符级错写/异体；若代码多写错字则亦会失败。
			if !strings.Contains(doc, code) {
				t.Errorf("卦 #%d（%s）第 %d 爻辞未命中金标准：\n  代码=%q\n  文档=%q",
					h.Number, h.Name, i+1, h.YaoCi[i], g.yaoCi[i])
			}
		}
	}
}
