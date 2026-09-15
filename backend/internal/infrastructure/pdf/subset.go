package pdf

import (
	"encoding/binary"
	"fmt"
	"sort"
)

// subsetResult 字体子集化结果。
type subsetResult struct {
	data []byte // 子集 TTF 字节（可直接作为 FontFile2 流）

	// cidOfRune rune → CID。CID 即子集内的新字形号（1..N），
	// 因此 PDF 侧可以用 /CIDToGIDMap /Identity，无需额外映射流。
	cidOfRune map[rune]uint16
	// cidOfGID 原始字形号 → CID，供复合字形去重与宽度查询使用
	cidOfGID map[uint16]uint16
	// uniOfCID CID → 原始字符，只记录**精确匹配**（不含折音降级），
	// 供 ToUnicode 使用：复制出来的文本应当尽量贴近输入。
	uniOfCID map[uint16]rune
	// width1000 CID → 1000 单位/em 下的推进量（PDF 的 W 数组使用）
	width1000 map[uint16]uint16

	glyphCount int
}

// subsetFor 为给定 rune 集合构建字体子集。
//
// 字体中不存在字形的字符会被记入 Document.MissingRunes()（见 document.go），
// 渲染时替换为 '?' 占位，避免 PDF 里出现空白。
func (f *ttfFont) subsetFor(runes []rune) (*subsetResult, error) {
	if len(f.cmap) == 0 {
		return nil, fmt.Errorf("字体缺少 cmap 表，无法建立 字符→字形 映射")
	}
	// 1. 收集需要的原始字形号（去重）
	need := make(map[uint16]bool)
	type hit struct {
		gid   uint16
		exact bool
	}
	resolved := make(map[rune]hit, len(runes))
	for _, r := range runes {
		gid, exact := f.resolveGlyph(r)
		if gid == 0 {
			continue
		}
		need[gid] = true
		resolved[r] = hit{gid: gid, exact: exact}
	}

	// 2. 展开复合字形的子字形依赖（可能需要多轮，因为子字形本身也可能是复合字形）
	queue := make([]uint16, 0, len(need))
	for gid := range need {
		queue = append(queue, gid)
	}
	for len(queue) > 0 {
		gid := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		for _, dep := range f.compositeDeps(gid) {
			if !need[dep] {
				need[dep] = true
				queue = append(queue, dep)
			}
		}
	}

	// 3. 重编号：按原始字形号升序，新号从 1 开始（0 保留给 .notdef）
	olds := make([]uint16, 0, len(need))
	for gid := range need {
		olds = append(olds, gid)
	}
	sort.Slice(olds, func(i, j int) bool { return olds[i] < olds[j] })

	oldToNew := make(map[uint16]uint16, len(olds))
	for i, gid := range olds {
		oldToNew[gid] = uint16(i + 1)
	}
	newToOld := make([]uint16, len(olds)+1) // index 0 = .notdef
	copy(newToOld[1:], olds)

	// 4. 重建 glyf / loca（长格式）
	//
	// loca 的语义是「第 k 个字形在 glyf 中的 [loca[k], loca[k+1]) 区间」。
	// 子集里第 0 号字形是空的 .notdef，因此 loca[0]=loca[1]=0，
	// 第 i 个收录字形（新号 i+1）的结束偏移写到 loca[i+2]。
	// （早期实现把结束偏移写到 loca[i+1]，整体错位一个字形，
	//  导致渲染出「形似但完全不对」的汉字。）
	glyf := make([]byte, 0, len(olds)*64)
	loca := make([]uint32, len(olds)+2)
	loca[0], loca[1] = 0, 0
	for i, oldGID := range olds {
		raw, err := f.glyphRange(oldGID)
		if err != nil {
			return nil, err
		}
		if len(raw) > 0 {
			raw, err = remapComposite(raw, oldToNew)
			if err != nil {
				return nil, err
			}
		}
		glyf = append(glyf, raw...)
		// 每个字形数据按 4 字节对齐（loca 为长格式时非强制，但对齐可兼容更多解析器）
		if pad := (4 - len(glyf)%4) % 4; pad != 0 {
			glyf = append(glyf, make([]byte, pad)...)
		}
		loca[i+2] = uint32(len(glyf))
	}

	locaBytes := make([]byte, len(loca)*4)
	for i, v := range loca {
		binary.BigEndian.PutUint32(locaBytes[i*4:i*4+4], v)
	}

	// 5. 重建 hmtx（子集内每个字形各一条记录：advanceWidth + lsb）
	newNumGlyphs := len(newToOld)
	hmtx := make([]byte, 0, newNumGlyphs*4)
	width1000 := make(map[uint16]uint16, len(olds))
	for cid := 1; cid < len(newToOld); cid++ {
		oldGID := newToOld[cid]
		adv := f.advanceWidth(oldGID)
		hw := make([]byte, 4)
		binary.BigEndian.PutUint16(hw[0:2], adv)
		binary.BigEndian.PutUint16(hw[2:4], 0) // lsb 对 PDF 渲染无影响
		hmtx = append(hmtx, hw...)
		width1000[uint16(cid)] = uint16(scaleTo1000(int(adv), int(f.unitsPerEm)))
	}

	// 6. 组装各表
	head := append([]byte(nil), f.tables["head"]...)
	binary.BigEndian.PutUint16(head[50:52], 1) // indexToLocFormat = long
	binary.BigEndian.PutUint32(head[8:12], 0)  // checkSumAdjustment 先清零，最后统一计算

	hhea := append([]byte(nil), f.tables["hhea"]...)
	binary.BigEndian.PutUint16(hhea[34:36], uint16(newNumGlyphs))

	maxp := append([]byte(nil), f.tables["maxp"]...)
	binary.BigEndian.PutUint16(maxp[4:6], uint16(newNumGlyphs))

	tables := map[string][]byte{
		"glyf": glyf,
		"head": head,
		"hhea": hhea,
		"hmtx": hmtx,
		"loca": locaBytes,
		"maxp": maxp,
	}
	// 复制（若存在）与 glyf 内指令配合的表，避免栅格化时引用缺失表
	for _, tag := range []string{"cvt ", "fpgm", "prep"} {
		if t, ok := f.tables[tag]; ok {
			tables[tag] = append([]byte(nil), t...)
		}
	}

	data, err := assembleFont(tables)
	if err != nil {
		return nil, err
	}

	cidOfRune := make(map[rune]uint16, len(resolved))
	uniOfCID := make(map[uint16]rune, len(resolved))
	for r, h := range resolved {
		cid, ok := oldToNew[h.gid]
		if !ok {
			continue
		}
		cidOfRune[r] = cid
		if h.exact {
			uniOfCID[cid] = r // 精确匹配优先作为 ToUnicode 的还原目标
		}
	}

	return &subsetResult{
		data:       data,
		cidOfRune:  cidOfRune,
		cidOfGID:   oldToNew,
		uniOfCID:   uniOfCID,
		width1000:  width1000,
		glyphCount: len(olds),
	}, nil
}

// remapComposite 把复合字形内部引用的子字形号改写为子集内的新号。
// 简单字形（numberOfContours >= 0）原样返回。
func remapComposite(data []byte, oldToNew map[uint16]uint16) ([]byte, error) {
	if len(data) < 10 {
		return data, nil
	}
	if int16(binary.BigEndian.Uint16(data[0:2])) >= 0 {
		return data, nil
	}
	out := append([]byte(nil), data...)
	off := 10
	for {
		if off+4 > len(out) {
			return nil, fmt.Errorf("复合字形数据在 flags 处提前结束（off=%d, len=%d）", off, len(out))
		}
		flags := binary.BigEndian.Uint16(out[off : off+2])
		child := binary.BigEndian.Uint16(out[off+2 : off+4])
		newChild, ok := oldToNew[child]
		if !ok {
			return nil, fmt.Errorf("复合字形引用了未被收录的子字形 %d", child)
		}
		binary.BigEndian.PutUint16(out[off+2:off+4], newChild)

		off += 4
		if flags&0x0001 != 0 { // ARG_1_AND_2_ARE_WORDS
			off += 4
		} else {
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
		if flags&0x0020 == 0 { // MORE_COMPONENTS 未置位
			break
		}
	}
	return out, nil
}

// assembleFont 按 sfnt 规范把表组装成字体文件，并写入校验和。
func assembleFont(tables map[string][]byte) ([]byte, error) {
	tags := make([]string, 0, len(tables))
	for tag := range tables {
		if len(tag) != 4 {
			return nil, fmt.Errorf("非法的表标签 %q（必须为 4 字节）", tag)
		}
		tags = append(tags, tag)
	}
	sort.Strings(tags)

	numTables := len(tags)
	headerSize := 12 + numTables*16

	// 计算各表偏移（4 字节对齐）
	total := headerSize
	offsets := make(map[string]int, numTables)
	for _, tag := range tags {
		total = align4(total)
		offsets[tag] = total
		total += len(tables[tag])
	}
	total = align4(total)

	out := make([]byte, total)
	binary.BigEndian.PutUint32(out[0:4], 0x00010000)
	binary.BigEndian.PutUint16(out[4:6], uint16(numTables))

	// 二分查找参数
	entrySelector := 0
	for (1 << (entrySelector + 1)) <= numTables {
		entrySelector++
	}
	searchRange := (1 << entrySelector) * 16
	binary.BigEndian.PutUint16(out[6:8], uint16(searchRange))
	binary.BigEndian.PutUint16(out[8:10], uint16(entrySelector))
	binary.BigEndian.PutUint16(out[10:12], uint16(numTables*16-searchRange))

	for i, tag := range tags {
		rec := out[12+i*16 : 12+i*16+16]
		copy(rec[0:4], tag)
		binary.BigEndian.PutUint32(rec[4:8], tableChecksum(tables[tag]))
		binary.BigEndian.PutUint32(rec[8:12], uint32(offsets[tag]))
		binary.BigEndian.PutUint32(rec[12:16], uint32(len(tables[tag])))
		copy(out[offsets[tag]:], tables[tag])
	}

	// head.checkSumAdjustment = 0xB1B0AFBA - 整份字体的校验和
	headOff, ok := offsets["head"]
	if !ok {
		return nil, fmt.Errorf("缺少 head 表")
	}
	binary.BigEndian.PutUint32(out[headOff+8:headOff+12], 0)
	adj := uint32(0xB1B0AFBA) - tableChecksum(out)
	binary.BigEndian.PutUint32(out[headOff+8:headOff+12], adj)
	return out, nil
}

// tableChecksum 计算 sfnt 校验和：按 4 字节大端求和（末尾不足处补 0）。
func tableChecksum(b []byte) uint32 {
	var sum uint32
	for i := 0; i < len(b); i += 4 {
		var word uint32
		for j := 0; j < 4; j++ {
			word <<= 8
			if i+j < len(b) {
				word |= uint32(b[i+j])
			}
		}
		sum += word
	}
	return sum
}

func align4(n int) int { return (n + 3) &^ 3 }

// scaleTo1000 把原始 em 单位换算为 PDF 的 1000 单位/em。
func scaleTo1000(v, unitsPerEm int) int {
	if unitsPerEm <= 0 {
		return v
	}
	return (v*1000 + unitsPerEm/2) / unitsPerEm
}
