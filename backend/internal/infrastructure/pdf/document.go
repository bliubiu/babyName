package pdf

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"
)

// PageSize 页面尺寸（单位：pt，1pt = 1/72 英寸）。
type PageSize struct{ Width, Height float64 }

// 常用页面尺寸
var (
	A4     = PageSize{595.276, 841.890}
	Letter = PageSize{612, 792}
)

// Options 文档配置。
type Options struct {
	// FontData 字体文件内容（优先级最高，便于测试与嵌入式场景）
	FontData []byte
	// FontPath 字体文件路径（FontData 为空时使用）
	FontPath string
	PageSize PageSize
	// 页边距
	MarginTop, MarginBottom, MarginLeft, MarginRight float64
	// Title 写入 PDF 元数据
	Title string
}

func (o *Options) withDefaults() Options {
	out := *o
	if out.PageSize.Width <= 0 || out.PageSize.Height <= 0 {
		out.PageSize = A4
	}
	if out.MarginTop == 0 {
		out.MarginTop = 64
	}
	if out.MarginBottom == 0 {
		out.MarginBottom = 56
	}
	if out.MarginLeft == 0 {
		out.MarginLeft = 56
	}
	if out.MarginRight == 0 {
		out.MarginRight = 56
	}
	return out
}

type opKind int

const (
	opText opKind = iota
	opRect
	opLine
)

// op 一条绘制指令。文本刻意保存为字符串而非预编码字节：
// 字体子集要等所有文本收集完成后才能确定 CID，序列化统一放在 WriteTo。
type op struct {
	kind  opKind
	x, y  float64 // 左上角坐标系（y 向下）
	size  float64
	text  string
	w, h  float64
	lw    float64 // 线宽
	color rgb
}

type rgb struct{ r, g, b float64 }

// Document 一份 PDF 文档。
//
// 坐标系：对外统一使用「左上角为原点、y 向下」的排版坐标（与屏幕/HTML 一致），
// 内部写内容流时再换算成 PDF 的左下角原点。
type Document struct {
	opt   Options
	font  *ttfFont
	pages [][]op
	// curIdx 当前写入页下标（0 起）。显式记录而不是「永远写最后一页」，
	// 是为了支持写完正文后回头补页脚/页码。
	curIdx int

	// footer 在 WriteTo 前对每一页回调一次，用于绘制页码等（此时页数已确定）
	footer func(d *Document, pageIndex, pageCount int)

	// 排版光标（左上角坐标系）
	x, y    float64
	leading float64
	color   rgb

	usedRunes map[rune]bool
	missing   map[rune]bool
}

// NewDocument 读取并解析字体，创建文档。
func NewDocument(opt Options) (*Document, error) {
	opt = opt.withDefaults()

	data := opt.FontData
	if len(data) == 0 {
		if opt.FontPath == "" {
			return nil, fmt.Errorf("未指定字体：请通过 Options.FontData 或 Options.FontPath 提供一个支持中文的 TrueType 字体")
		}
		b, err := os.ReadFile(opt.FontPath)
		if err != nil {
			return nil, fmt.Errorf("读取字体 %s 失败: %w", opt.FontPath, err)
		}
		data = b
	}

	f, err := parseTTF(data)
	if err != nil {
		return nil, fmt.Errorf("解析字体失败: %w", err)
	}

	d := &Document{
		opt:       opt,
		font:      f,
		color:     rgb{0.12, 0.12, 0.12},
		usedRunes: make(map[rune]bool, 512),
		missing:   make(map[rune]bool),
	}
	d.AddPage()
	return d, nil
}

// PageSize 返回当前页面尺寸。
func (d *Document) PageSize() PageSize { return d.opt.PageSize }

// ContentWidth 返回正文可用宽度。
func (d *Document) ContentWidth() float64 {
	return d.opt.PageSize.Width - d.opt.MarginLeft - d.opt.MarginRight
}

// BottomLimit 返回正文底线（左上角坐标系下的 y）。
func (d *Document) BottomLimit() float64 {
	return d.opt.PageSize.Height - d.opt.MarginBottom
}

// CursorY 当前光标 y。
func (d *Document) CursorY() float64 { return d.y }

// SetCursorY 设置光标 y（用于右对齐标题等固定位置）。
func (d *Document) SetCursorY(y float64) { d.y = y }

// SetCursorX 设置光标 x。
func (d *Document) SetCursorX(x float64) { d.x = x }

// MarginLeft 返回左边距。
func (d *Document) MarginLeft() float64 { return d.opt.MarginLeft }

// MarginRight 返回右边距。
func (d *Document) MarginRight() float64 { return d.opt.MarginRight }

// MarginTop 返回上边距。
func (d *Document) MarginTop() float64 { return d.opt.MarginTop }

// MarginBottom 返回下边距。
func (d *Document) MarginBottom() float64 { return d.opt.MarginBottom }

// AddPage 追加一页并把光标复位到页首。
func (d *Document) AddPage() {
	d.pages = append(d.pages, nil)
	d.curIdx = len(d.pages) - 1
	d.x = d.opt.MarginLeft
	d.y = d.opt.MarginTop
}

// SelectPage 切换当前写入页（0 起）；越界时忽略。
func (d *Document) SelectPage(idx int) {
	if idx < 0 || idx >= len(d.pages) {
		return
	}
	d.curIdx = idx
}

// SetFooter 设置页脚回调。WriteTo 时会为每一页调用一次，此时总页数已确定。
func (d *Document) SetFooter(fn func(d *Document, pageIndex, pageCount int)) { d.footer = fn }

// SetColor 设置后续文本/图形颜色（0-1）。
func (d *Document) SetColor(r, g, b float64) { d.color = rgb{r, g, b} }

// ResetColor 恢复默认文字色。
func (d *Document) ResetColor() { d.color = rgb{0.12, 0.12, 0.12} }

// EnsureSpace 若剩余高度不足 h 则换页，并把光标落到页首。
func (d *Document) EnsureSpace(h float64) {
	if d.y+h > d.BottomLimit() {
		d.AddPage()
	}
}

// Space 推进光标 h。
func (d *Document) Space(h float64) { d.y += h }

// Measure 计算字符串在给定字号下的宽度（pt）。
func (d *Document) Measure(size float64, s string) float64 {
	var w int
	for _, r := range s {
		if gid, _ := d.font.resolveGlyph(r); gid != 0 {
			w += int(d.font.advanceWidth(gid))
			continue
		}
		w += int(d.font.advanceWidth(d.font.cmap['?']))
	}
	return float64(scaleTo1000(w, int(d.font.unitsPerEm))) * size / 1000.0
}

// Text 在指定位置绘制一行文本（左上角坐标系，y 为基线所在行顶部）。
func (d *Document) Text(x, y float64, size float64, s string) {
	if s == "" {
		return
	}
	for _, r := range s {
		if gid, ok := d.font.cmap[r]; !ok || gid == 0 {
			if r != '\n' {
				d.missing[r] = true
			}
		}
		d.usedRunes[r] = true
	}
	p := &d.pages[d.curIdx]
	*p = append(*p, op{kind: opText, x: x, y: y, size: size, text: s, color: d.color})
}

// TextLine 在当前光标处绘制一行并推进光标（不换行、不分页）。
func (d *Document) TextLine(size, leading float64, s string) {
	d.EnsureSpace(leading)
	d.Text(d.x, d.y+size*0.85, size, s)
	d.y += leading
}

// FillRect 以当前颜色（SetColor 设置）填充矩形，坐标同 Text 的左上角坐标系。
func (d *Document) FillRect(x, y, w, h float64) {
	p := &d.pages[d.curIdx]
	*p = append(*p, op{kind: opRect, x: x, y: y, w: w, h: h, color: d.color})
}

// Rule 在指定 y 处画一条水平分隔线。
func (d *Document) Rule(x, y, w float64, lw float64) {
	p := &d.pages[d.curIdx]
	*p = append(*p, op{kind: opLine, x: x, y: y, w: w, lw: lw, color: rgb{0.78, 0.78, 0.76}})
}

// Wrap 按可用宽度做贪心折行；CJK 无空格，允许在任意字符间断行。
func (d *Document) Wrap(s string, size, maxWidth float64) []string {
	if maxWidth <= 0 {
		maxWidth = d.ContentWidth()
	}
	var lines []string
	var cur []rune
	var curW float64
	flush := func() {
		lines = append(lines, string(cur))
		cur = cur[:0]
		curW = 0
	}
	for _, r := range s {
		if r == '\n' {
			flush()
			continue
		}
		rw := d.runeWidth(r, size)
		if curW+rw > maxWidth && len(cur) > 0 {
			flush()
		}
		cur = append(cur, r)
		curW += rw
	}
	if len(cur) > 0 {
		flush()
	}
	if len(lines) == 0 {
		lines = []string{""}
	}
	return lines
}

func (d *Document) runeWidth(r rune, size float64) float64 {
	gid, _ := d.font.resolveGlyph(r)
	if gid == 0 {
		gid = d.font.cmap['?']
	}
	return float64(scaleTo1000(int(d.font.advanceWidth(gid)), int(d.font.unitsPerEm))) * size / 1000.0
}

// Paragraph 输出一段自动折行、自动分页的文本。
func (d *Document) Paragraph(s string, size, leading float64) {
	for _, line := range d.Wrap(s, size, d.ContentWidth()) {
		d.TextLine(size, leading, line)
	}
}

// MissingRunes 返回当前字体中不存在字形的字符（调用方应据此告警）。
func (d *Document) MissingRunes() []rune {
	out := make([]rune, 0, len(d.missing))
	for r := range d.missing {
		if r != '?' {
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// ---------- 序列化 ----------

// WriteTo 渲染并写出完整 PDF。
func (d *Document) WriteTo(w io.Writer) (int64, error) {
	// 页脚必须先于字形收集执行：它会往页面里追加文本，这些字符同样需要进入字体子集。
	if d.footer != nil {
		total := len(d.pages)
		for i := range d.pages {
			d.SelectPage(i)
			d.footer(d, i, total)
		}
		d.SelectPage(len(d.pages) - 1)
	}

	// '?' 必须始终进入子集：缺字时用它兜底，否则占位符会落到 .notdef（渲染成空白）
	d.usedRunes['?'] = true

	runes := make([]rune, 0, len(d.usedRunes))
	for r := range d.usedRunes {
		runes = append(runes, r)
	}
	sort.Slice(runes, func(i, j int) bool { return runes[i] < runes[j] })

	subset, err := d.font.subsetFor(runes)
	if err != nil {
		return 0, fmt.Errorf("字体子集化失败: %w", err)
	}
	// '?' 一定需要（缺失字形时的占位符）
	placeholderCID, _ := subset.cidOfRune['?']

	var buf bytes.Buffer
	buf.WriteString("%PDF-1.7\n")
	// 二进制标记：告诉工具链本文件含二进制流
	buf.Write([]byte{'%', 0xE2, 0xE3, 0xCF, 0xD3, '\n'})

	const (
		objCatalog  = 1
		objPages    = 2
		objFont     = 3
		objCIDFont  = 4
		objDesc     = 5
		objFile2    = 6
		objToUni    = 7
		objInfo     = 8
		firstPageID = 9
	)
	pageContentID := func(k int) int { return firstPageID + 2*k }
	pageID := func(k int) int { return firstPageID + 2*k + 1 }

	offsets := map[int]int{}
	writeObj := func(id int, body []byte) {
		offsets[id] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n", id)
		buf.Write(body)
		buf.WriteString("\nendobj\n")
	}

	d.writeFontObjects(subset, writeObj, objFont, objCIDFont, objDesc, objFile2, objToUni)

	// 页面对象
	kids := make([]string, 0, len(d.pages))
	for k := range d.pages {
		kids = append(kids, fmt.Sprintf("%d 0 R", pageID(k)))
		content := d.serializePageOps(d.pages[k], subset, placeholderCID)
		writeObj(pageContentID(k), streamObject(content))

		page := fmt.Sprintf("<< /Type /Page /Parent %d 0 R /MediaBox [0 0 %.3f %.3f] "+
			"/Resources << /Font << /F1 %d 0 R >> /ProcSet [/PDF /Text] >> /Contents %d 0 R >>",
			objPages, d.opt.PageSize.Width, d.opt.PageSize.Height, objFont, pageContentID(k))
		writeObj(pageID(k), []byte(page))
	}

	writeObj(objCatalog, []byte(fmt.Sprintf("<< /Type /Catalog /Pages %d 0 R >>", objPages)))
	writeObj(objPages, []byte(fmt.Sprintf("<< /Type /Pages /Count %d /Kids [%s] >>",
		len(d.pages), strings.Join(kids, " "))))

	info := fmt.Sprintf("<< /Producer (namer) /Creator (namer) /CreationDate (D:%s)", time.Now().Format("20060102150405"))
	if d.opt.Title != "" {
		info += fmt.Sprintf(" /Title %s", pdfTextString(d.opt.Title))
	}
	info += " >>"
	writeObj(objInfo, []byte(info))

	// xref + trailer
	maxID := pageID(len(d.pages) - 1)
	xrefOff := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n", maxID+1)
	buf.WriteString("0000000000 65535 f \n")
	for id := 1; id <= maxID; id++ {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offsets[id])
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root %d 0 R /Info %d 0 R >>\nstartxref\n%d\n%%%%EOF\n",
		maxID+1, objCatalog, objInfo, xrefOff)

	n, err := w.Write(buf.Bytes())
	return int64(n), err
}

// serializePageOps 把绘制指令序列化成内容流。
func (d *Document) serializePageOps(ops []op, subset *subsetResult, placeholderCID uint16) []byte {
	var b strings.Builder
	// 页面高度换算：内容流用左下角原点
	h := d.opt.PageSize.Height
	for _, o := range ops {
		switch o.kind {
		case opText:
			cids := make([]byte, 0, len(o.text)*2)
			for _, r := range o.text {
				if r == '\n' {
					continue
				}
				cid, ok := subset.cidOfRune[r]
				if !ok {
					cid = placeholderCID
				}
				cids = append(cids, byte(cid>>8), byte(cid))
			}
			if len(cids) == 0 {
				continue
			}
			fmt.Fprintf(&b, "BT %.3f %.3f %.3f rg /F1 %.2f Tf 1 0 0 1 %.3f %.3f Tm <%X> Tj ET\n",
				o.color.r, o.color.g, o.color.b, o.size, o.x, h-o.y, cids)
		case opRect:
			fmt.Fprintf(&b, "%.3f %.3f %.3f rg %.3f %.3f %.3f %.3f re f\n",
				o.color.r, o.color.g, o.color.b, o.x, h-o.y-o.h, o.w, o.h)
		case opLine:
			fmt.Fprintf(&b, "%.3f %.3f %.3f RG %.2f w %.3f %.3f m %.3f %.3f l S\n",
				o.color.r, o.color.g, o.color.b, o.lw, o.x, h-o.y, o.x+o.w, h-o.y)
		}
	}
	return []byte(b.String())
}

// writeFontObjects 写出字体相关的 5 个对象。
func (d *Document) writeFontObjects(subset *subsetResult, writeObj func(int, []byte),
	idFont, idCID, idDesc, idFile, idToUni int) {

	psName := d.font.postScriptName()
	baseFont := subsetTag(subset.data) + "+" + psName

	// Type0
	writeObj(idFont, []byte(fmt.Sprintf(
		"<< /Type /Font /Subtype /Type0 /BaseFont /%s /Encoding /Identity-H "+
			"/DescendantFonts [%d 0 R] /ToUnicode %d 0 R >>", baseFont, idCID, idToUni)))

	// CIDFontType2：CID 即子集内新字形号，因此 CIDToGIDMap 用 Identity
	writeObj(idCID, []byte(fmt.Sprintf(
		"<< /Type /Font /Subtype /CIDFontType2 /BaseFont /%s "+
			"/CIDSystemInfo << /Registry (Adobe) /Ordering (Identity) /Supplement 0 >> "+
			"/FontDescriptor %d 0 R /DW 1000 /W [%s] /CIDToGIDMap /Identity >>",
		baseFont, idDesc, wArray(subset))))

	// FontDescriptor
	upem := int(d.font.unitsPerEm)
	desc := fmt.Sprintf("<< /Type /FontDescriptor /FontName /%s /Flags 4 "+
		"/FontBBox [%d %d %d %d] /ItalicAngle 0 /Ascent %d /Descent %d /CapHeight %d "+
		"/StemV 80 /MissingWidth 1000 /FontFile2 %d 0 R >>",
		baseFont,
		scaleTo1000(int(d.font.xMin), upem), scaleTo1000(int(d.font.yMin), upem),
		scaleTo1000(int(d.font.xMax), upem), scaleTo1000(int(d.font.yMax), upem),
		scaleTo1000(int(d.font.ascender), upem), scaleTo1000(int(d.font.descender), upem),
		scaleTo1000(int(float64(d.font.ascender)*0.7), upem),
		idFile)
	writeObj(idDesc, []byte(desc))

	// FontFile2（字体程序）
	fileHeader := fmt.Sprintf("<< /Length %d /Length1 %d >>", len(subset.data), len(subset.data))
	writeObj(idFile, streamObjectWithHeader(fileHeader, subset.data))

	// ToUnicode CMap
	writeObj(idToUni, streamObject(buildToUnicodeCMap(subset)))
}

// wArray 构造 /W 数组。子集内 CID 连续（1..N），因此只需一组。
func wArray(subset *subsetResult) string {
	if len(subset.width1000) == 0 {
		return ""
	}
	cids := make([]int, 0, len(subset.width1000))
	for cid := range subset.width1000 {
		cids = append(cids, int(cid))
	}
	sort.Ints(cids)

	var b strings.Builder
	b.WriteString(fmt.Sprintf("%d [", cids[0]))
	for i, cid := range cids {
		if i > 0 && cid != cids[i-1]+1 { // 理论上不会发生，稳妥起见分组
			b.WriteString(fmt.Sprintf("] %d [", cid))
		}
		b.WriteString(fmt.Sprintf("%d ", subset.width1000[uint16(cid)]))
	}
	b.WriteString("]")
	return b.String()
}

// buildToUnicodeCMap 生成 CID → Unicode 的 CMap，保证文本可复制/可检索。
func buildToUnicodeCMap(subset *subsetResult) []byte {
	type pair struct {
		cid uint16
		uni []byte
	}
	// 优先使用精确匹配（uniOfCID）；未被精确映射覆盖的 CID（例如折音降级出来的字形）
	// 再退回 cidOfRune 里的任意 rune，保证每个 CID 都有 ToUnicode 条目。
	runeOfCID := make(map[uint16]rune, len(subset.cidOfRune))
	for r, cid := range subset.cidOfRune {
		if _, exists := runeOfCID[cid]; !exists {
			runeOfCID[cid] = r
		}
	}
	for cid, r := range subset.uniOfCID {
		runeOfCID[cid] = r
	}
	pairs := make([]pair, 0, len(runeOfCID))
	for cid, r := range runeOfCID {
		pairs = append(pairs, pair{cid: cid, uni: utf16BE(string(r))})
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].cid < pairs[j].cid })

	var b bytes.Buffer
	b.WriteString("/CIDInit /ProcSet findresource begin\n12 dict begin\nbegincmap\n")
	b.WriteString("/CIDSystemInfo << /Registry (Adobe) /Ordering (UCS) /Supplement 0 >> def\n")
	b.WriteString("/CMapName /Adobe-Identity-UCS def\n/CMapType 2 def\n")
	b.WriteString("1 begincodespacerange\n<0000> <FFFF>\nendcodespacerange\n")
	// 规范建议每个 beginbfchar 块不超过 100 条
	for start := 0; start < len(pairs); start += 100 {
		end := start + 100
		if end > len(pairs) {
			end = len(pairs)
		}
		fmt.Fprintf(&b, "%d beginbfchar\n", end-start)
		for _, p := range pairs[start:end] {
			fmt.Fprintf(&b, "<%04X> <%X>\n", p.cid, p.uni)
		}
		b.WriteString("endbfchar\n")
	}
	b.WriteString("endcmap\nCMapName currentdict /CMap defineresource pop\nend\nend\n")
	return b.Bytes()
}

// streamObject 包装为一个流对象。
func streamObject(content []byte) []byte {
	return streamObjectWithHeader(fmt.Sprintf("<< /Length %d >>", len(content)), content)
}

func streamObjectWithHeader(header string, content []byte) []byte {
	var b bytes.Buffer
	b.WriteString(header)
	b.WriteString("\nstream\n")
	b.Write(content)
	b.WriteString("\nendstream")
	return b.Bytes()
}

// subsetTag 依据子集内容生成 6 位大写标识（PDF 规范的 subset tag 约定）。
func subsetTag(data []byte) string {
	sum := tableChecksum(data)
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	out := make([]byte, 6)
	for i := 0; i < 6; i++ {
		out[i] = alphabet[(sum>>(uint(i)*4))%26]
	}
	return string(out)
}

// postScriptName 读取 name 表中的 PostScript 名（nameID=6）；失败时回退。
func (f *ttfFont) postScriptName() string {
	name, ok := f.tables["name"]
	if !ok || len(name) < 6 {
		return "CJKSubset"
	}
	count := int(binary.BigEndian.Uint16(name[2:4]))
	stringOffset := int(binary.BigEndian.Uint16(name[4:6]))
	for i := 0; i < count; i++ {
		base := 6 + i*12
		if base+12 > len(name) {
			break
		}
		platform := binary.BigEndian.Uint16(name[base : base+2])
		nameID := binary.BigEndian.Uint16(name[base+6 : base+8])
		length := int(binary.BigEndian.Uint16(name[base+8 : base+10]))
		off := int(binary.BigEndian.Uint16(name[base+10 : base+12]))
		if nameID != 6 {
			continue
		}
		start := stringOffset + off
		if start < 0 || start+length > len(name) {
			continue
		}
		raw := name[start : start+length]
		var s string
		if platform == 3 { // UTF-16BE
			var sb strings.Builder
			for j := 0; j+1 < len(raw); j += 2 {
				sb.WriteRune(rune(binary.BigEndian.Uint16(raw[j : j+2])))
			}
			s = sb.String()
		} else {
			s = string(raw)
		}
		s = sanitizeFontName(s)
		if s != "" {
			return s
		}
	}
	return "CJKSubset"
}

// sanitizeFontName 让字体名可用于 PDF 名称对象：只保留字母数字与 - _ +。
func sanitizeFontName(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '+':
			b.WriteRune(r)
		}
	}
	return b.String()
}

// pdfTextString 把 Go 字符串编码为 PDF 文本串。
//
// 纯 ASCII 用字面串 (…)，非 ASCII 必须用带 BOM 的 UTF-16BE 十六进制串 <FEFF…>：
// PDF 的 Info 字典只认 PDFDocEncoding 或带 BOM 的 UTF-16BE，把 UTF-8 字节直接
// 塞进字面串会得到乱码标题（pdfinfo 显示为一串怪字符）。
func pdfTextString(s string) string {
	ascii := true
	for _, r := range s {
		if r < 0x20 || r > 0x7E {
			ascii = false
			break
		}
	}
	if ascii {
		var b strings.Builder
		b.WriteByte('(')
		for _, r := range s {
			switch r {
			case '(', ')', '\\':
				b.WriteByte('\\')
			case '\n':
				b.WriteString("\\n")
				continue
			case '\r':
				b.WriteString("\\r")
				continue
			}
			b.WriteRune(r)
		}
		b.WriteByte(')')
		return b.String()
	}

	raw := append([]byte{0xFE, 0xFF}, utf16BE(s)...)
	var b strings.Builder
	b.WriteByte('<')
	for _, by := range raw {
		fmt.Fprintf(&b, "%02X", by)
	}
	b.WriteByte('>')
	return b.String()
}

// PageCount 返回页数。
func (d *Document) PageCount() int { return len(d.pages) }
