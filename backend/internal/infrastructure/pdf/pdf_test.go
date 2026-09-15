package pdf

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// loadTestFont 载入系统中文测试字体；系统无可用字体时跳过用例。
func loadTestFont(t *testing.T) []byte {
	t.Helper()
	p, err := FindSystemCJKFont()
	if err != nil {
		t.Skipf("系统无可用中文 TrueType 字体，跳过：%v", err)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		t.Skipf("读取字体 %s 失败，跳过：%v", p, err)
	}
	return data
}

func TestParseTTFBasic(t *testing.T) {
	data := loadTestFont(t)
	f, err := parseTTF(data)
	if err != nil {
		t.Fatalf("parseTTF 失败: %v", err)
	}
	if f.unitsPerEm == 0 {
		t.Error("unitsPerEm 为 0")
	}
	if f.numGlyphs == 0 {
		t.Error("numGlyphs 为 0")
	}
	if len(f.loca) != int(f.numGlyphs)+1 {
		t.Errorf("loca 长度 = %d，期望 numGlyphs+1 = %d", len(f.loca), int(f.numGlyphs)+1)
	}
	if got := f.glyphID('中'); got == 0 {
		t.Error("cmap 未映射「中」")
	}
	if got := f.glyphID('A'); got == 0 {
		t.Error("cmap 未映射「A」")
	}
	if w := f.advanceWidth(f.glyphID('中')); w == 0 {
		t.Error("「中」推进量为 0")
	}
}

// TestParseTTFRejectsCFF 明确拒绝 CFF 轮廓（避免用户配了 .otf 却得到空白页）
func TestParseTTFRejectsCFF(t *testing.T) {
	head := make([]byte, 16)
	copy(head[0:4], []byte("OTTO"))
	if _, err := parseTTF(head); err == nil {
		t.Fatal("OTTO/CFF 应返回错误")
	} else if !strings.Contains(err.Error(), "CFF") {
		t.Errorf("错误信息应说明是 CFF 问题，实际：%v", err)
	}
}

// TestSubsetForKeepsCompositeDeps 子集必须收录复合字形的子字形
//
// 若漏收，「中」这类由多个部件拼装的字会渲染成残缺图形或空白。
func TestSubsetForKeepsCompositeDeps(t *testing.T) {
	data := loadTestFont(t)
	f, err := parseTTF(data)
	if err != nil {
		t.Fatalf("parseTTF 失败: %v", err)
	}

	runes := []rune("中文测试报告张明")
	sub, err := f.subsetFor(runes)
	if err != nil {
		t.Fatalf("subsetFor 失败: %v", err)
	}
	if len(sub.data) == 0 {
		t.Fatal("子集字体为空")
	}
	if binary.BigEndian.Uint32(sub.data[0:4]) != 0x00010000 {
		t.Errorf("子集 sfnt 版本非法: 0x%08X", binary.BigEndian.Uint32(sub.data[0:4]))
	}
	for _, r := range runes {
		if _, ok := sub.cidOfRune[r]; !ok {
			t.Errorf("rune %q 未获得 CID", r)
		}
	}

	// 子集必须能再次被解析，且字形数等于收录数 + 1（.notdef）
	back, err := parseTTF(sub.data)
	if err != nil {
		t.Fatalf("子集字体无法被重新解析: %v", err)
	}
	if int(back.numGlyphs) != sub.glyphCount+1 {
		t.Errorf("子集 numGlyphs = %d，期望 %d", back.numGlyphs, sub.glyphCount+1)
	}
	// 复合字形依赖：子集内每个复合字形的子字形号必须已经在新范围内
	for cid := 1; cid <= sub.glyphCount; cid++ {
		raw, err := back.glyphRange(uint16(cid))
		if err != nil {
			t.Fatalf("子集字形 %d 读取失败: %v", cid, err)
		}
		if len(raw) == 0 {
			continue
		}
		for _, dep := range compositeChildren(raw) {
			if int(dep) > sub.glyphCount {
				t.Errorf("字形 %d 引用了越界的子字形 %d（子集仅 1..%d）", cid, dep, sub.glyphCount)
			}
		}
	}
}

// compositeChildren 解析复合字形的子字形号（测试辅助，独立于实现以免自证）。
func compositeChildren(data []byte) []uint16 {
	if len(data) < 10 || int16(binary.BigEndian.Uint16(data[0:2])) >= 0 {
		return nil
	}
	var out []uint16
	off := 10
	for {
		if off+4 > len(data) {
			break
		}
		flags := binary.BigEndian.Uint16(data[off : off+2])
		out = append(out, binary.BigEndian.Uint16(data[off+2:off+4]))
		off += 4
		if flags&0x0001 != 0 {
			off += 4
		} else {
			off += 2
		}
		switch {
		case flags&0x0008 != 0:
			off += 2
		case flags&0x0040 != 0:
			off += 4
		case flags&0x0080 != 0:
			off += 8
		}
		if flags&0x0020 == 0 {
			break
		}
	}
	return out
}

// TestDocumentProducesValidPDF 端到端：生成结构可解析的 PDF
func TestDocumentProducesValidPDF(t *testing.T) {
	fontData := loadTestFont(t)

	doc, err := NewDocument(Options{FontData: fontData, Title: "测试报告"})
	if err != nil {
		t.Fatalf("NewDocument 失败: %v", err)
	}
	doc.SetColor(0.1, 0.2, 0.5)
	doc.TextLine(20, 28, "宝宝起名报告")
	doc.ResetColor()
	doc.Space(8)
	doc.Rule(doc.MarginLeft(), doc.CursorY(), doc.ContentWidth(), 0.8)
	doc.Space(12)
	doc.Paragraph("姓名：张明｜性别：男｜出生：2024年5月20日 10:00", 11, 18)
	doc.Paragraph(strings.Repeat("中文长文本换行与分页测试。", 200), 11, 18)

	var buf bytes.Buffer
	n, err := doc.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo 失败: %v", err)
	}
	if int(n) != buf.Len() {
		t.Errorf("WriteTo 返回 %d，实际写出 %d", n, buf.Len())
	}

	out := buf.Bytes()
	if !bytes.HasPrefix(out, []byte("%PDF-1.7")) {
		t.Errorf("缺少 PDF 头（实际前 16 字节：%q）", out[:min(16, len(out))])
	}
	if !bytes.HasSuffix(out, []byte("%%EOF\n")) {
		t.Errorf("缺少 EOF 结尾（实际末 8 字节：%q）", out[max(0, len(out)-8):])
	}
	for _, want := range []string{"/Type /Catalog", "/Type /Pages", "/Subtype /CIDFontType2",
		"/CIDToGIDMap /Identity", "/FontFile2", "/ToUnicode", "startxref"} {
		if !bytes.Contains(out, []byte(want)) {
			t.Errorf("PDF 中缺少 %s", want)
		}
	}
	if doc.PageCount() < 2 {
		t.Errorf("长文本应触发分页，实际页数 = %d", doc.PageCount())
	}
	// 子集化必须显著小于原字体，否则说明没有真正做子集
	if len(out) > len(fontData)/4 {
		t.Errorf("PDF 体积 %d 字节过大（源字体 %d 字节），子集化可能未生效", len(out), len(fontData))
	}
	if len(doc.MissingRunes()) != 0 {
		t.Errorf("测试文本不应有缺字，实际缺失：%q", string(doc.MissingRunes()))
	}
}

// TestDocumentMissingGlyphFallback 字体缺字时用 '?' 兜底并登记，不得报错或产出空白
func TestDocumentMissingGlyphFallback(t *testing.T) {
	fontData := loadTestFont(t)
	doc, err := NewDocument(Options{FontData: fontData})
	if err != nil {
		t.Fatalf("NewDocument 失败: %v", err)
	}
	// U+E000 为私用区，任何正常字体都不会有字形
	doc.TextLine(12, 18, "缺字测试\uE000结束")

	var buf bytes.Buffer
	if _, err := doc.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo 失败: %v", err)
	}
	missing := doc.MissingRunes()
	if len(missing) != 1 || missing[0] != '\uE000' {
		t.Errorf("MissingRunes = %q，期望恰好包含 U+E000", string(missing))
	}
	if bytes.Contains(buf.Bytes(), []byte("\uE000")) {
		t.Error("替换后的占位符不应是原字符")
	}
}

// TestMeasureIsMonotonic 宽度测量应随内容单调增长（排版折行的基础）
func TestMeasureIsMonotonic(t *testing.T) {
	fontData := loadTestFont(t)
	doc, err := NewDocument(Options{FontData: fontData})
	if err != nil {
		t.Fatalf("NewDocument 失败: %v", err)
	}
	w1 := doc.Measure(12, "张")
	w2 := doc.Measure(12, "张明")
	w3 := doc.Measure(24, "张")
	if !(w1 > 0 && w2 > w1 && w3 > w1) {
		t.Errorf("宽度测量异常：w1=%.2f w2=%.2f w3=%.2f", w1, w2, w3)
	}
}

// TestWrapRespectsWidth 折行结果每行都不超过可用宽度
func TestWrapRespectsWidth(t *testing.T) {
	fontData := loadTestFont(t)
	doc, err := NewDocument(Options{FontData: fontData})
	if err != nil {
		t.Fatalf("NewDocument 失败: %v", err)
	}
	text := strings.Repeat("宝宝起名报告与八字五行分析", 10)
	lines := doc.Wrap(text, 11, 200)
	if len(lines) < 2 {
		t.Fatalf("长文本应折成多行，实际 %d 行", len(lines))
	}
	for i, ln := range lines {
		if w := doc.Measure(11, ln); w > 200.5 {
			t.Errorf("第 %d 行宽度 %.2f 超过上限 200（内容 %q）", i, w, ln)
		}
	}
	if got := strings.Join(lines, ""); got != text {
		t.Error("折行后拼接的内容与原文不一致（丢字或多字）")
	}
}

// TestNewDocumentWithoutFont 未提供字体时应给出明确错误而非静默产出空白
func TestNewDocumentWithoutFont(t *testing.T) {
	_, err := NewDocument(Options{FontPath: filepath.Join(t.TempDir(), "missing.ttf")})
	if err == nil {
		t.Fatal("字体缺失时应返回错误")
	}
}

// TestSubsetGlyphBytesMatchOriginal 回归：子集内每个 CID 的字形必须与原字体逐字节一致
//
// 背景：早期实现把 loca 的结束偏移写到 loca[i+1]（而非 loca[i+2]），
// 造成整体错位一个字形——PDF 能正常打开、字也是"汉字的样子"，但全是别的字
// （实测把「张氏宝宝起名报告」渲染成「德气容容辰告拼周」），极难从结构上发现。
// 因此必须逐字形比对字节，而不是只检查文件能否解析。
func TestSubsetGlyphBytesMatchOriginal(t *testing.T) {
	data := loadTestFont(t)
	f, err := parseTTF(data)
	if err != nil {
		t.Fatalf("parseTTF 失败: %v", err)
	}

	runes := []rune("张氏宝宝起名报告珀熙慧茗浩渊赟诗甲乙丙丁")
	sub, err := f.subsetFor(runes)
	if err != nil {
		t.Fatalf("subsetFor 失败: %v", err)
	}
	back, err := parseTTF(sub.data)
	if err != nil {
		t.Fatalf("子集字体无法解析: %v", err)
	}

	checked := 0
	for _, r := range runes {
		cid, ok := sub.cidOfRune[r]
		if !ok {
			continue // 字体本身没有该字形
		}
		want, err := f.glyphRange(f.glyphID(r))
		if err != nil {
			t.Fatalf("读取原始字形失败: %v", err)
		}
		got, err := back.glyphRange(cid)
		if err != nil {
			t.Fatalf("读取子集字形 CID=%d 失败: %v", cid, err)
		}
		// 子集会对每个字形做 4 字节对齐，因此允许末尾多出 0-3 个 0 字节
		if len(got) < len(want) || len(got)-len(want) > 3 {
			t.Fatalf("字形 %q：CID=%d 的子集字形长度异常（原始 %d 字节，子集 %d 字节）",
				r, cid, len(want), len(got))
		}
		if !bytes.Equal(want, got[:len(want)]) {
			t.Fatalf("字形 %q：CID=%d 的子集字形与原始不一致（原始 %d 字节，子集 %d 字节）——"+
				"loca 偏移或字形编号存在错位", r, cid, len(want), len(got))
		}
		for _, b := range got[len(want):] {
			if b != 0 {
				t.Fatalf("字形 %q：CID=%d 的对齐填充非 0", r, cid)
			}
		}
		checked++
	}
	if checked < 10 {
		t.Fatalf("仅校验了 %d 个字形，覆盖不足", checked)
	}

	// loca 必须单调不减，且首尾与 glyf 长度一致
	glyfLen := len(back.tables["glyf"])
	if got := int(back.loca[0]); got != 0 {
		t.Errorf("loca[0] = %d，期望 0", got)
	}
	if got := int(back.loca[len(back.loca)-1]); got != glyfLen {
		t.Errorf("loca 末项 = %d，期望等于 glyf 长度 %d", got, glyfLen)
	}
	for i := 1; i < len(back.loca); i++ {
		if back.loca[i] < back.loca[i-1] {
			t.Fatalf("loca[%d]=%d 小于前一项 %d（非单调）", i, back.loca[i], back.loca[i-1])
		}
	}
	// 第 0 号字形（.notdef）应为空区间
	if back.loca[0] != back.loca[1] {
		t.Errorf(".notdef 区间应为空，实际 [%d,%d)", back.loca[0], back.loca[1])
	}
}

// TestToUnicodeMapsEveryCID ToUnicode 必须覆盖所有 CID，否则文本无法复制/检索
func TestToUnicodeMapsEveryCID(t *testing.T) {
	f, err := parseTTF(loadTestFont(t))
	if err != nil {
		t.Fatalf("parseTTF 失败: %v", err)
	}
	r := '中'
	sub, err := f.subsetFor([]rune{r})
	if err != nil {
		t.Fatalf("subsetFor 失败: %v", err)
	}
	cm := string(buildToUnicodeCMap(sub))

	cid := sub.cidOfRune[r]
	want := fmt.Sprintf("<%04X> <%04X>", cid, r)
	if !strings.Contains(cm, want) {
		t.Errorf("ToUnicode CMap 缺少 %q 的映射（期望 %q）", r, want)
	}
	for _, block := range []string{"begincmap", "begincodespacerange", "/CMapType 2", "endcmap"} {
		if !strings.Contains(cm, block) {
			t.Errorf("ToUnicode CMap 缺少 %q", block)
		}
	}
}
