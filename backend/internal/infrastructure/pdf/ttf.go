// Package pdf 是一个只依赖标准库的最小 PDF 生成器，重点解决中文（CJK）渲染。
//
// 为什么不用第三方库：本项目的 PDF 需求集中在「可下载的中文报告」这一件事上，
// 而市面上主流的 Go PDF 库（gofpdf/fpdf/gopdf）都需要在运行时加载一个 TTF 才能输出中文。
// 这里的实现做同样的事——解析 TTF、按需**子集化**内嵌字形——但把依赖面收敛到标准库，
// 并且把「字体缺失」变成显式错误而不是静默产出空白页。
//
// 结构：
//
//	ttf.go     TrueType 解析（cmap / loca / glyf / hmtx / head / hhea / maxp）
//	subset.go  字形子集化（重编号 + 复合字形依赖重写 + 表校验和）
//	document.go PDF 对象/页面/内容流与顶层排版 API
//
// 输出为 PDF 1.7，字体使用 Type0 + CIDFontType2 + Identity-H 编码，
// CID 直接等于子集内的新字形号（因此 /CIDToGIDMap 为 Identity，无需额外映射流），
// 并写入 ToUnicode CMap 保证文本可复制、可检索。
package pdf

import (
	"encoding/binary"
	"fmt"
	"sort"
	"unicode"
	"unicode/utf16"

	"golang.org/x/text/unicode/norm"
)

// ttfFont 解析后的 TrueType 字体，仅保留渲染所需的表。
type ttfFont struct {
	raw    []byte
	tables map[string][]byte

	unitsPerEm       uint16
	numGlyphs        uint16
	numberOfHMetrics uint16
	indexToLocFormat int16
	loca             []uint32 // numGlyphs+1 个偏移

	// cmap：rune → 原始 glyphID（合并优先 Unicode 平面）
	cmap map[rune]uint16

	hmtx []byte

	// 指标（原始 em 单位）
	xMin, yMin, xMax, yMax int16
	ascender, descender    int16
	numContoursFn          func(uint16) int
}

// ttfTag 把 4 字节标签转成字符串（table 目录中的 tag）。
func ttfTag(b []byte) string { return string(b) }

// parseTTF 解析 TrueType 字体数据。
//
// 支持：
//   - TTF 与「TTF 风格的 OpenType」（sfnt 版本 0x00010000 / 'true'）
//   - TrueType 集合（.ttc）：取集合中的第 0 个字体。TTC 内各表的偏移量是相对
//     整个文件起点计算的，因此只需把「表目录」的解析起点移到该字体的 sfnt 偏移处。
//
// 不支持 OTF/CFF（表 'CFF '）：CFF 字形是 Type2 CharString，需要另一套解析器。
func parseTTF(data []byte) (*ttfFont, error) {
	if len(data) < 12 {
		return nil, fmt.Errorf("字体数据过短（%d 字节）", len(data))
	}
	sfnt := binary.BigEndian.Uint32(data[0:4])
	dirOffset := 0
	switch sfnt {
	case 0x00010000, 0x74727565: // 0x00010000 / 'true'
	case 0x4F54544F: // 'OTTO'
		return nil, fmt.Errorf("不支持 CFF/OpenType 轮廓（'OTTO'），请使用 TrueType（.ttf/.ttc）字体")
	case 0x74746366: // 'ttcf'：TrueType 集合
		if len(data) < 16 {
			return nil, fmt.Errorf("TTC 头不完整")
		}
		if binary.BigEndian.Uint32(data[8:12]) == 0 {
			return nil, fmt.Errorf("TTC 中不含字体")
		}
		dirOffset = int(binary.BigEndian.Uint32(data[12:16]))
		if dirOffset <= 0 || dirOffset+12 > len(data) {
			return nil, fmt.Errorf("TTC 首个字体的 sfnt 偏移非法（%d）", dirOffset)
		}
		if binary.BigEndian.Uint32(data[dirOffset:dirOffset+4]) == 0x4F54544F {
			return nil, fmt.Errorf("TTC 首个字体为 CFF 轮廓（'OTTO'），请改用 TrueType 字体")
		}
	default:
		return nil, fmt.Errorf("未知的 sfnt 版本 0x%08X", sfnt)
	}

	numTables := int(binary.BigEndian.Uint16(data[dirOffset+4 : dirOffset+6]))
	if numTables <= 0 || len(data) < dirOffset+12+numTables*16 {
		return nil, fmt.Errorf("表目录不完整（numTables=%d）", numTables)
	}

	tables := make(map[string][]byte, numTables)
	for i := 0; i < numTables; i++ {
		base := dirOffset + 12 + i*16
		rec := data[base : base+16]
		tag := ttfTag(rec[0:4])
		off := int(binary.BigEndian.Uint32(rec[8:12]))
		length := int(binary.BigEndian.Uint32(rec[12:16]))
		if off < 0 || length < 0 || off+length > len(data) {
			// 个别字体存在越界的已弃用表，跳过而不是让整份字体不可用
			continue
		}
		tables[tag] = data[off : off+length]
	}

	f := &ttfFont{raw: data, tables: tables}

	head, ok := tables["head"]
	if !ok || len(head) < 54 {
		return nil, fmt.Errorf("缺少或损坏 head 表")
	}
	f.unitsPerEm = binary.BigEndian.Uint16(head[18:20])
	f.xMin = int16(binary.BigEndian.Uint16(head[36:38]))
	f.yMin = int16(binary.BigEndian.Uint16(head[38:40]))
	f.xMax = int16(binary.BigEndian.Uint16(head[40:42]))
	f.yMax = int16(binary.BigEndian.Uint16(head[42:44]))
	f.indexToLocFormat = int16(binary.BigEndian.Uint16(head[50:52]))
	if f.unitsPerEm == 0 {
		return nil, fmt.Errorf("head.unitsPerEm 为 0")
	}

	maxp, ok := tables["maxp"]
	if !ok || len(maxp) < 6 {
		return nil, fmt.Errorf("缺少或损坏 maxp 表")
	}
	f.numGlyphs = binary.BigEndian.Uint16(maxp[4:6])
	if f.numGlyphs == 0 {
		return nil, fmt.Errorf("字体不含字形")
	}

	hhea, ok := tables["hhea"]
	if !ok || len(hhea) < 36 {
		return nil, fmt.Errorf("缺少或损坏 hhea 表")
	}
	f.ascender = int16(binary.BigEndian.Uint16(hhea[4:6]))
	f.descender = int16(binary.BigEndian.Uint16(hhea[6:8]))
	f.numberOfHMetrics = binary.BigEndian.Uint16(hhea[34:36])
	if f.numberOfHMetrics == 0 {
		f.numberOfHMetrics = 1
	}

	hmtx, ok := tables["hmtx"]
	if !ok {
		return nil, fmt.Errorf("缺少 hmtx 表")
	}
	f.hmtx = hmtx

	loca, ok := tables["loca"]
	if !ok {
		return nil, fmt.Errorf("缺少 loca 表")
	}
	if err := f.parseLoca(loca); err != nil {
		return nil, err
	}
	if _, ok := tables["glyf"]; !ok {
		return nil, fmt.Errorf("缺少 glyf 表（不支持无轮廓字体）")
	}

	// cmap 是「文本 → 字形」映射，仅在需要按 rune 取字形时使用。
	// 子集字体刻意不携带 cmap（PDF 侧通过 CIDToGIDMap 定位字形），
	// 因此这里缺失 cmap 不算错误，只把 cmap 置空。
	if cm, ok := tables["cmap"]; ok && len(cm) >= 4 {
		if err := f.parseCmap(cm); err != nil {
			return nil, err
		}
	} else {
		f.cmap = make(map[rune]uint16)
	}
	return f, nil
}

func (f *ttfFont) parseLoca(loca []byte) error {
	n := int(f.numGlyphs) + 1
	f.loca = make([]uint32, n)
	switch f.indexToLocFormat {
	case 0: // short：偏移量以 2 字节为单位存储
		if len(loca) < n*2 {
			return fmt.Errorf("loca 表长度不足（short 格式需要 %d 字节，实际 %d）", n*2, len(loca))
		}
		for i := 0; i < n; i++ {
			f.loca[i] = uint32(binary.BigEndian.Uint16(loca[i*2:i*2+2])) * 2
		}
	case 1: // long
		if len(loca) < n*4 {
			return fmt.Errorf("loca 表长度不足（long 格式需要 %d 字节，实际 %d）", n*4, len(loca))
		}
		for i := 0; i < n; i++ {
			f.loca[i] = binary.BigEndian.Uint32(loca[i*4 : i*4+4])
		}
	default:
		return fmt.Errorf("未知的 head.indexToLocFormat=%d", f.indexToLocFormat)
	}
	return nil
}

// parseCmap 解析 cmap，合并所有 Unicode 子表（后解析到的映射覆盖先前的，
// 因此把 U+0000 起步的 (0,x) 与 (3,1)/(3,10) 都纳入，尽量不丢字形）。
func (f *ttfFont) parseCmap(cm []byte) error {
	if len(cm) < 4 {
		return fmt.Errorf("cmap 表损坏（长度 %d）", len(cm))
	}
	numTables := int(binary.BigEndian.Uint16(cm[2:4]))
	f.cmap = make(map[rune]uint16, 4096)

	type record struct {
		platform, encoding uint16
		off                int
	}
	var records []record
	for i := 0; i < numTables; i++ {
		base := 4 + i*8
		if base+8 > len(cm) {
			break
		}
		records = append(records, record{
			platform: binary.BigEndian.Uint16(cm[base : base+2]),
			encoding: binary.BigEndian.Uint16(cm[base+2 : base+4]),
			off:      int(binary.BigEndian.Uint32(cm[base+4 : base+8])),
		})
	}
	// 排序：优先 Windows-BMP/全平面 与 Unicode 平台，保证后写入的是更权威的映射
	sort.SliceStable(records, func(i, j int) bool {
		pri := func(r record) int {
			switch {
			case r.platform == 3 && r.encoding == 10:
				return 0
			case r.platform == 3 && r.encoding == 1:
				return 1
			case r.platform == 0:
				return 2
			default:
				return 3
			}
		}
		return pri(records[i]) > pri(records[j])
	})

	for _, r := range records {
		if r.off <= 0 || r.off >= len(cm) {
			continue
		}
		switch binary.BigEndian.Uint16(cm[r.off : r.off+2]) {
		case 4:
			f.parseCmapFormat4(cm[r.off:])
		case 12:
			f.parseCmapFormat12(cm[r.off:])
		}
	}
	if len(f.cmap) == 0 {
		return fmt.Errorf("cmap 中未找到可用的 Unicode 子表（format 4/12）")
	}
	return nil
}

func (f *ttfFont) parseCmapFormat4(t []byte) {
	if len(t) < 14 {
		return
	}
	segCount := int(binary.BigEndian.Uint16(t[6:8])) / 2
	if segCount == 0 {
		return
	}
	endBase := 14
	startBase := endBase + segCount*2 + 2
	deltaBase := startBase + segCount*2
	rangeBase := deltaBase + segCount*2
	if rangeBase+segCount*2 > len(t) {
		return
	}
	for i := 0; i < segCount; i++ {
		end := binary.BigEndian.Uint16(t[endBase+i*2 : endBase+i*2+2])
		start := binary.BigEndian.Uint16(t[startBase+i*2 : startBase+i*2+2])
		delta := int16(binary.BigEndian.Uint16(t[deltaBase+i*2 : deltaBase+i*2+2]))
		rangeOff := int(binary.BigEndian.Uint16(t[rangeBase+i*2 : rangeBase+i*2+2]))
		if start > end {
			continue
		}
		// 分段上限保护：避免损坏数据造成超大循环
		if int(end)-int(start) > 0xFFFF {
			continue
		}
		for c := uint32(start); c <= uint32(end); c++ {
			var gid uint16
			if rangeOff == 0 {
				gid = uint16(int32(c) + int32(delta))
			} else {
				addr := rangeBase + i*2 + rangeOff + int((c-uint32(start))*2)
				if addr+2 > len(t) {
					continue
				}
				gid = binary.BigEndian.Uint16(t[addr : addr+2])
				if gid != 0 {
					gid = uint16(int32(gid) + int32(delta))
				}
			}
			if gid != 0 {
				f.cmap[rune(c)] = gid
			}
			if c == 0xFFFF {
				break
			}
		}
	}
}

func (f *ttfFont) parseCmapFormat12(t []byte) {
	if len(t) < 16 {
		return
	}
	nGroups := int(binary.BigEndian.Uint32(t[12:16]))
	for i := 0; i < nGroups; i++ {
		base := 16 + i*12
		if base+12 > len(t) {
			return
		}
		start := binary.BigEndian.Uint32(t[base : base+4])
		end := binary.BigEndian.Uint32(t[base+4 : base+8])
		startGID := binary.BigEndian.Uint32(t[base+8 : base+12])
		if start > end || end > 0x10FFFF {
			continue
		}
		for c := start; c <= end; c++ {
			gid := startGID + (c - start)
			if gid > 0xFFFF {
				break
			}
			if gid != 0 {
				f.cmap[rune(c)] = uint16(gid)
			}
		}
	}
}

// glyphID 返回 rune 对应的原始字形号；0 表示字体不含该字形。
func (f *ttfFont) glyphID(r rune) uint16 { return f.cmap[r] }

// resolveGlyph 解析 rune 的字形号，找不到时尝试「去变音符号」的降级。
//
// 降级动机：中文字体（simhei/SimSun 等）普遍缺少拉丁扩展字符，拼音里的
// ā/ī/ō/ē/ū 会缺字。与其渲染成空白或 ?，不如退回基础字符 a/i/o/e/u——
// 少一个声调符号，但整列拼音仍然可读、可复制。
//
// 返回 (字形号, 是否为精确匹配)。
func (f *ttfFont) resolveGlyph(r rune) (uint16, bool) {
	if gid, ok := f.cmap[r]; ok && gid != 0 {
		return gid, true
	}
	if base, ok := foldRune(r); ok {
		if gid, ok := f.cmap[base]; ok && gid != 0 {
			return gid, false
		}
	}
	return 0, false
}

// foldRune 把带变音符号的拉丁字符折回基础字符（ā→a、ǘ→u、ç→c）。
func foldRune(r rune) (rune, bool) {
	if r < 0x80 {
		return 0, false
	}
	for _, d := range norm.NFD.String(string(r)) {
		if unicode.Is(unicode.Mn, d) { // 组合用记号，跳过
			continue
		}
		return d, true
	}
	return 0, false
}

// glyphRange 返回某个字形的原始字节（glyf 表内切片）；空字形返回 nil。
func (f *ttfFont) glyphRange(gid uint16) ([]byte, error) {
	if int(gid)+1 >= len(f.loca) {
		return nil, fmt.Errorf("字形号 %d 越界（numGlyphs=%d）", gid, f.numGlyphs)
	}
	start, end := f.loca[gid], f.loca[gid+1]
	glyf := f.tables["glyf"]
	if start > end || int(end) > len(glyf) {
		return nil, fmt.Errorf("字形 %d 的 loca 区间非法（%d..%d，glyf 长度 %d）", gid, start, end, len(glyf))
	}
	return glyf[start:end], nil
}

// advanceWidth 返回字形在原始 em 单位下的水平推进量。
func (f *ttfFont) advanceWidth(gid uint16) uint16 {
	if len(f.hmtx) == 0 {
		return 0
	}
	idx := int(gid)
	if idx >= int(f.numberOfHMetrics) {
		idx = int(f.numberOfHMetrics) - 1 // 超出部分复用最后一组 advance
	}
	off := idx * 4
	if off+2 > len(f.hmtx) {
		return 0
	}
	return binary.BigEndian.Uint16(f.hmtx[off : off+2])
}

// compositeDeps 返回复合字形直接引用的子字形号。
//
// 复合字形（numberOfContours < 0）由其它字形拼装而成；子集化时必须把被引用的
// 字形一并收录，否则渲染时会出现「半个字」或空白。
func (f *ttfFont) compositeDeps(gid uint16) []uint16 {
	data, err := f.glyphRange(gid)
	if err != nil || len(data) < 10 {
		return nil
	}
	if int16(binary.BigEndian.Uint16(data[0:2])) >= 0 {
		return nil // 简单字形
	}
	var deps []uint16
	off := 10
	for {
		if off+4 > len(data) {
			break
		}
		flags := binary.BigEndian.Uint16(data[off : off+2])
		child := binary.BigEndian.Uint16(data[off+2 : off+4])
		deps = append(deps, child)
		off += 4
		switch {
		case flags&0x0001 != 0: // ARG_1_AND_2_ARE_WORDS
			off += 4
		default:
			off += 2
		}
		switch {
		case flags&0x0008 != 0: // WE_HAVE_A_SCALE
			off += 2
		case flags&0x0040 != 0: // WE_HAVE_AN_X_AND_Y_SCALE
			off += 4
		case flags&0x0080 != 0: // WE_HAVE_A_TWO_BY_TWO
			off += 8
		}
		if flags&0x0020 == 0 { // MORE_COMPONENTS 未置位 → 结束
			break
		}
	}
	return deps
}

// utf16BE 把字符串编码为 UTF-16BE 字节流（ToUnicode CMap 使用）。
func utf16BE(s string) []byte {
	units := utf16.Encode([]rune(s))
	out := make([]byte, 0, len(units)*2)
	for _, u := range units {
		out = append(out, byte(u>>8), byte(u))
	}
	return out
}
