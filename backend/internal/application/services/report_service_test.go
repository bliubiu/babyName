package services

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func sampleReportPayload() map[string]interface{} {
	return map[string]interface{}{
		"surname":    "张",
		"gender":     "male",
		"birth_date": "2024年5月20日",
		"birth_time": "10:00（巳时）",
		"bazi":       "甲辰 己巳 丙申 癸巳",
		"wuxing":     "木土 土火 火金 水火",
		"xiyongshen": "水（喜神：金）",
		"names": []interface{}{
			map[string]interface{}{
				"full_name": "张珀熙", "pinyin": "zhāng pò xī", "wuxing": "水金", "score": 92.4,
				"meaning": "「珀」为琥珀，温润有光；「熙」取光明兴盛之意。",
			},
			map[string]interface{}{
				"full_name": "张慧茗", "pinyin": "zhāng huì míng", "wuxing": "水木", "score": 91,
				"meaning": "「慧」为聪慧明达，「茗」指上等茶芽。",
			},
		},
	}
}

// TestGeneratePDFContainsRealData 回归：PDF 必须由真实请求数据生成
//
// 历史实现输出的是硬编码占位 PDF（正文写死「宝宝起名报告 / 这是一个PDF报告示例」），
// 因此这里既要断言占位文案消失，也要断言请求数据确实进了文档。
// 注意正文是 CID 编码的十六进制串，不能直接 grep 汉字；改为断言 ToUnicode CMap 里
// 出现了对应字符的码位（只有真正排入版的字符才会被写入 CMap）。
func TestGeneratePDFContainsRealData(t *testing.T) {
	svc := NewReportService()
	b, err := svc.GeneratePDF(context.Background(), sampleReportPayload())
	if err != nil {
		t.Skipf("本机无可用中文字体，跳过：%v", err)
	}

	if !bytes.HasPrefix(b, []byte("%PDF-")) {
		t.Fatalf("输出不是 PDF（前 8 字节 %q）", b[:min(8, len(b))])
	}
	if !bytes.HasSuffix(b, []byte("%%EOF\n")) {
		t.Error("PDF 缺少 EOF 结尾")
	}

	// 占位文案必须彻底消失
	for _, banned := range []string{"这是一个PDF报告示例", "示例"} {
		if bytes.Contains(b, []byte(banned)) {
			t.Errorf("PDF 中仍存在占位文本 %q", banned)
		}
	}

	// 内嵌字体与 CID 结构齐备
	for _, want := range []string{"/FontFile2", "/CIDToGIDMap /Identity", "/ToUnicode", "beginbfchar"} {
		if !bytes.Contains(b, []byte(want)) {
			t.Errorf("PDF 缺少 %s", want)
		}
	}

	// 请求数据确实排入了版：姓名里的字符应出现在 ToUnicode CMap 中
	for _, r := range "张珀熙慧茗" {
		code := strings.ToUpper(utf16Hex(r))
		if !bytes.Contains(b, []byte("<"+code+">")) {
			t.Errorf("ToUnicode CMap 中缺少字符 %q（码位 %s），说明数据未进入文档", r, code)
		}
	}

	// 体积下限：占位实现约 1.5KB，真实内嵌子集字体应在数十 KB
	if len(b) < 10*1024 {
		t.Errorf("PDF 仅 %d 字节，疑似未内嵌字体或未输出正文", len(b))
	}
}

// utf16Hex 返回 rune 的 UTF-16BE 十六进制（含代理对处理）
func utf16Hex(r rune) string {
	if r > 0xFFFF {
		r -= 0x10000
		hi := 0xD800 + (r >> 10)
		lo := 0xDC00 + (r & 0x3FF)
		return hex4(hi) + hex4(lo)
	}
	return hex4(r)
}

func hex4(v rune) string {
	const digits = "0123456789ABCDEF"
	return string([]byte{
		digits[(v>>12)&0xF], digits[(v>>8)&0xF], digits[(v>>4)&0xF], digits[v&0xF],
	})
}

// TestGeneratePDFEmptyData 空数据也要产出可读报告（而非 500）
func TestGeneratePDFEmptyData(t *testing.T) {
	svc := NewReportService()

	for _, data := range []interface{}{nil, map[string]interface{}{}, "not-an-object"} {
		b, err := svc.GeneratePDF(context.Background(), data)
		if err != nil {
			if strings.Contains(err.Error(), "字体") {
				t.Skipf("本机无可用中文字体，跳过：%v", err)
			}
			t.Fatalf("data=%v 时 GeneratePDF 失败: %v", data, err)
		}
		if !bytes.HasPrefix(b, []byte("%PDF-")) {
			t.Errorf("data=%v 的输出不是 PDF", data)
		}
	}
}

// TestGeneratePDFContextCancelled ctx 已取消时应立即返回错误
func TestGeneratePDFContextCancelled(t *testing.T) {
	svc := NewReportService()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := svc.GeneratePDF(ctx, sampleReportPayload()); err == nil {
		t.Error("ctx 已取消却成功返回")
	}
}

// TestParseReportModelToleratesShapes 模型解析需兼容多种 JSON 形态
func TestParseReportModelToleratesShapes(t *testing.T) {
	m := parseReportModel(map[string]interface{}{
		"gender": "male",
		"names": []map[string]interface{}{
			{"full_name": "张明", "score": float64(88)},
		},
	})
	if !m.HasData {
		t.Error("HasData 应为 true")
	}
	if len(m.Names) != 1 || m.Names[0].Score != "88" {
		t.Errorf("名字解析异常：%+v", m.Names)
	}
	if got := genderLabel(m.Gender); got != "男" {
		t.Errorf("genderLabel(male) = %q，期望 男", got)
	}

	// 非 map 输入
	if got := parseReportModel(nil); got.HasData {
		t.Error("nil 输入时 HasData 应为 false")
	}
	if got := parseReportModel([]int{1, 2}); got.HasData {
		t.Error("非对象输入时 HasData 应为 false")
	}
}
