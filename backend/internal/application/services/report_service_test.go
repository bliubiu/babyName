package services

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// useTestCJKFont 确保 PDF 用例走内置开源字体路径。
//
// 生产代码默认使用随二进制内置的 Noto Sans SC，
// 仓库不再依赖测试字体夹具。清除环境变量避免外部
// 设置的 NAMER_PDF_FONT 影响测试确定性；CI 干净检出无需任何额外准备。
func useTestCJKFont(t *testing.T) {
	t.Helper()
	t.Setenv("NAMER_PDF_FONT", "")
}

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
	useTestCJKFont(t)
	svc := NewReportService()
	b, err := svc.GeneratePDF(context.Background(), sampleReportPayload())
	if err != nil {
		t.Fatalf("PDF 生成失败：%v", err)
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
	useTestCJKFont(t)
	svc := NewReportService()

	for _, data := range []interface{}{nil, map[string]interface{}{}, "not-an-object"} {
		b, err := svc.GeneratePDF(context.Background(), data)
		if err != nil {
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

// TestParseReportModelFullDetail 详批扩展字段（rishou/nayin/analysis/wuxing_dist/scores/options）应正确解析。
func TestParseReportModelFullDetail(t *testing.T) {
	m := parseReportModel(map[string]interface{}{
		"rishou": "甲", "nayin": "海中金", "zodiac": "鼠",
		"analysis":    []interface{}{"【八字格局】日主为甲", "【五行平衡】整体均衡"},
		"wuxing_dist": map[string]interface{}{"金": float64(1), "木": float64(5), "水": float64(0)},
		"options":     map[string]interface{}{"detail": "full"},
		"names": []interface{}{
			map[string]interface{}{
				"full_name": "李沐", "score": 92.0,
				"scores": []interface{}{
					map[string]interface{}{"name": "五行", "score": float64(90)},
					map[string]interface{}{"name": "音律", "score": float64(88)},
				},
			},
		},
	})
	if m.Rishou != "甲" || m.Nayin != "海中金" || m.Zodiac != "鼠" {
		t.Errorf("扩展字段解析异常：%+v", m)
	}
	if !m.FullDetail {
		t.Error("默认/options=full 应为详批模式")
	}
	if len(m.Analysis) != 2 || m.Analysis[0] != "【八字格局】日主为甲" {
		t.Errorf("analysis 解析异常：%#v", m.Analysis)
	}
	if m.WuXingDist["木"] != 5 || m.WuXingDist["水"] != 0 {
		t.Errorf("wuxing_dist 解析异常：%#v", m.WuXingDist)
	}
	if len(m.Names) != 1 || len(m.Names[0].Scores) != 2 || m.Names[0].Scores[0].Score != 90 {
		t.Errorf("scores 解析异常：%+v", m.Names)
	}

	// summary 模式关闭详批
	if s := parseReportModel(map[string]interface{}{"options": map[string]interface{}{"detail": "summary"}}); s.FullDetail {
		t.Error("options.detail=summary 应关闭详批")
	}
}

// TestGeneratePDFFullDetail 详批模式 PDF（含分布图、解读、各维评分）应正常生成。
func TestGeneratePDFFullDetail(t *testing.T) {
	useTestCJKFont(t)
	payload := sampleReportPayload()
	payload["analysis"] = []interface{}{
		"【八字格局】日主为丙，日柱丙申。",
		"【五行平衡】分布为金1、木5、水1、火1、土1，整体偏枯，需补益水。",
		"【喜用神补益】喜用神为水、木，宜选属水属木之字。",
	}
	payload["wuxing_dist"] = map[string]interface{}{"金": 1, "木": 5, "水": 1, "火": 1, "土": 1}
	payload["rishou"] = "丙"
	payload["nayin"] = "山下火"
	payload["zodiac"] = "龙"
	payload["names"].([]interface{})[0].(map[string]interface{})["scores"] = []interface{}{
		map[string]interface{}{"name": "五行", "score": 92},
		map[string]interface{}{"name": "音律", "score": 88},
	}

	svc := NewReportService()
	b, err := svc.GeneratePDF(context.Background(), payload)
	if err != nil {
		t.Fatalf("详批 PDF 生成失败：%v", err)
	}
	if !bytes.HasPrefix(b, []byte("%PDF-")) || len(b) < 10*1024 {
		t.Errorf("详批 PDF 输出异常：%d 字节", len(b))
	}
	if !bytes.HasSuffix(b, []byte("%%EOF\n")) {
		t.Error("详批 PDF 缺少 EOF")
	}
}

// TestReportLayoutHelpers 章节编号连续、最高分与分数格式化。
func TestReportLayoutHelpers(t *testing.T) {
	for n, want := range []string{"", "一", "二", "三", "四", "五", "六", "七", "八", "九", "十", "十一", "十二"} {
		if n == 0 {
			continue
		}
		if got := chineseNumeral(n); got != want {
			t.Errorf("chineseNumeral(%d) = %q, 期望 %q", n, got, want)
		}
	}
	var sec sectionCounter
	if got := sec.title("基本信息"); got != "一、基本信息" {
		t.Errorf("首个章节 = %q", got)
	}
	if got := sec.title("推荐名字"); got != "二、推荐名字" {
		t.Errorf("跳号时章节应连续，实际 %q", got)
	}

	names := []reportName{{ScoreValue: 88.7}, {ScoreValue: 92.4}, {ScoreValue: 0}}
	if got := highestScore(names); got != 92.4 {
		t.Errorf("highestScore = %v, 期望 92.4", got)
	}
	if got := formatScore(92.0); got != "92" {
		t.Errorf("formatScore(92) = %q", got)
	}
	if got := formatScore(92.4); got != "92.4" {
		t.Errorf("formatScore(92.4) = %q", got)
	}
}

// TestParseReportModelScoreValue 评分字符串与数值两种形态都要能算出数值分，
// 供摘要最高分与前三名高亮使用。
func TestParseReportModelScoreValue(t *testing.T) {
	m := parseReportModel(map[string]interface{}{
		"names": []interface{}{
			map[string]interface{}{"full_name": "张明", "score": "88.6"},
			map[string]interface{}{"full_name": "张涵", "score": float64(92.4)},
		},
	})
	if len(m.Names) != 2 {
		t.Fatalf("names = %d", len(m.Names))
	}
	if m.Names[0].ScoreValue != 88.6 || m.Names[1].ScoreValue != 92.4 {
		t.Errorf("ScoreValue 解析异常：%+v", m.Names)
	}
	if got := highestScore(m.Names); got != 92.4 {
		t.Errorf("highestScore = %v, 期望 92.4", got)
	}
}
