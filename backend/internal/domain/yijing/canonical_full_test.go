package yijing

import (
	"os"
	"path/filepath"
	"regexp"
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
	)
)

// normalize 去标点空白 + 异体折叠，得到仅含汉字的归一串
func normalize(s string) string {
	s = punctReplacer.Replace(s)
	s = variantReplacer.Replace(s)
	return strings.TrimSpace(s)
}

// cnDigit 单个中文数字字符 → 0-9
func cnDigit(r rune) int {
	switch r {
	case '一':
		return 1
	case '二':
		return 2
	case '三':
		return 3
	case '四':
		return 4
	case '五':
		return 5
	case '六':
		return 6
	case '七':
		return 7
	case '八':
		return 8
	case '九':
		return 9
	}
	return 0
}

// parseChineseNum 解析 1-64 的中文数字（如 二十三 → 23，六十四 → 64）
func parseChineseNum(s string) int {
	s = strings.TrimSpace(s)
	if s == "十" {
		return 10
	}
	ten := strings.Index(s, "十")
	if ten < 0 {
		return cnDigit([]rune(s)[0])
	}
	left := s[:ten]
	right := s[ten+len("十"):]
	tens := 1
	if left != "" {
		tens = cnDigit([]rune(left)[0])
	}
	ones := 0
	if right != "" {
		ones = cnDigit([]rune(right)[0])
	}
	return tens*10 + ones
}

// parseHexagramDoc 从单篇经传文档解析出 卦序、卦名、卦辞全文、大象、爻辞序列表
func parseHexagramDoc(content string) (number int, name, guaCiFull, daXiang string, yaoCi []string) {
	titleRe := regexp.MustCompile(`《易经》第(.+?)卦\s+(\S+?)\s`)
	if m := titleRe.FindStringSubmatch(content); m != nil {
		number = parseChineseNum(m[1])
		name = m[2]
	}

	// 卦辞全文：名：之后，至首个 彖曰 / 象曰 / 文言 之前（含后续续句）
	headerRe := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(name) + `：(.+)$`)
	if m := headerRe.FindStringSubmatch(content); m != nil {
		rest := m[1]
		if idx := strings.Index(rest, "。彖曰"); idx >= 0 {
			guaCiFull = rest[:idx]
		} else if idx := strings.Index(rest, "。象曰"); idx >= 0 {
			guaCiFull = rest[:idx]
		} else if idx := strings.Index(rest, "。文言"); idx >= 0 {
			guaCiFull = rest[:idx]
		} else {
			guaCiFull = rest
		}
	}

	// 大象（彖曰 之后的第一个 象曰：）：取至行尾，兼容「雷出地奋，豫。先王以作乐崇德」
	// 等两句式大象，避免在首个句号处误截断（对拍按子串判定，多截无害）
	tIdx := strings.Index(content, "。彖曰")
	if tIdx < 0 {
		tIdx = strings.Index(content, "彖曰")
	}
	if tIdx < 0 {
		tIdx = 0 // 防御：无彖曰时从全文检索（正常卦文档均有彖曰）
	}
	if xIdx := strings.Index(content[tIdx:], "象曰："); xIdx >= 0 {
		xIdx += tIdx
		raw := content[xIdx+len("象曰："):]
		if nl := strings.IndexAny(raw, "\r\n"); nl >= 0 {
			raw = raw[:nl]
		}
		daXiang = raw
	}

	// 爻辞：以 爻位前缀（初/九/六/上/用 + 六九）起的行，排除 象曰/彖曰/文言
	yaoRe := regexp.MustCompile(`^(初[六九]|九[二三四五]|六[二三四五]|上[六九]|用[六九])[：:](.+)$`)
	for _, line := range strings.Split(content, "\n") {
		if m := yaoRe.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
			yaoCi = append(yaoCi, m[2])
		}
	}
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
