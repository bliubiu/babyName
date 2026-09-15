package services

import (
	"bytes"
	"context"
	"fmt"
	"html"
	"os"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"name/internal/infrastructure/logger"
	"name/internal/infrastructure/pdf"
)

// ReportService 报告服务
//
// 报告由「解析请求数据 → 统一报表模型 → 分别渲染 PDF / HTML」两段构成：
// 两种格式共用同一份模型，避免历史上 PDF 与 HTML 各解析一遍、字段口径不一致的问题。
type ReportService struct {
	fontOnce sync.Once
	fontData []byte
	fontErr  error
	fontPath string
}

// NewReportService 创建报告服务
func NewReportService() *ReportService {
	return &ReportService{}
}

// ---------- 报表模型 ----------

// reportName 单个推荐名字
type reportName struct {
	FullName string
	Pinyin   string
	WuXing   string
	Score    string
	Meaning  string
}

// reportModel 报表数据模型。字段名与 /report/* 既有请求体保持一致（surname/gender/…/names）。
type reportModel struct {
	Surname    string
	Gender     string
	BirthDate  string
	BirthTime  string
	Bazi       string
	WuXing     string
	XiYongShen string
	Names      []reportName
	// HasData 请求体是否提供了可用的对象数据
	HasData bool
}

// Title 报告标题
func (m reportModel) Title() string {
	if m.Surname != "" {
		return m.Surname + "氏宝宝起名报告"
	}
	return "宝宝起名报告"
}

// parseReportModel 把松散的请求体解析成报表模型。
//
// 兼容三种输入形态：map[string]interface{}（正常）、其它类型（视为未提供数据）、nil。
// 缺失字段一律留空，由渲染层决定是跳过该行还是提示「未提供」——不猜测、不编造。
func parseReportModel(data interface{}) reportModel {
	m := reportModel{}
	raw, ok := data.(map[string]interface{})
	if !ok || raw == nil {
		return m
	}
	m.HasData = true

	m.Surname = rawString(raw, "surname")
	m.Gender = rawString(raw, "gender")
	m.BirthDate = rawString(raw, "birth_date")
	m.BirthTime = rawString(raw, "birth_time")
	m.Bazi = rawString(raw, "bazi")
	m.WuXing = rawString(raw, "wuxing")
	m.XiYongShen = rawString(raw, "xiyongshen")

	switch names := raw["names"].(type) {
	case []interface{}:
		for _, n := range names {
			nm, ok := n.(map[string]interface{})
			if !ok {
				continue
			}
			m.Names = append(m.Names, reportName{
				FullName: rawString(nm, "full_name"),
				Pinyin:   rawString(nm, "pinyin"),
				WuXing:   rawString(nm, "wuxing"),
				Score:    getScoreValue(nm, "score"),
				Meaning:  rawString(nm, "meaning"),
			})
		}
	case []map[string]interface{}:
		for _, nm := range names {
			m.Names = append(m.Names, reportName{
				FullName: rawString(nm, "full_name"),
				Pinyin:   rawString(nm, "pinyin"),
				WuXing:   rawString(nm, "wuxing"),
				Score:    getScoreValue(nm, "score"),
				Meaning:  rawString(nm, "meaning"),
			})
		}
	}
	return m
}

// ---------- PDF ----------

// loadFont 载入中文字体（进程内只解析一次）。
//
// 找不到字体时直接报错：宁可让接口返回「报告生成失败」，也不能输出一份正文全是
// 空白/方框的"报告"——后者用户无从判断是数据问题还是渲染问题。
func (s *ReportService) loadFont() ([]byte, error) {
	s.fontOnce.Do(func() {
		path, err := pdf.FindSystemCJKFont()
		if err != nil {
			s.fontErr = err
			return
		}
		data, err := os.ReadFile(path)
		if err != nil {
			s.fontErr = fmt.Errorf("读取中文字体 %s 失败: %w", path, err)
			return
		}
		s.fontPath = path
		s.fontData = data
		logger.Info("报告中文字体已就绪", zap.String("font", path), zap.Int("bytes", len(data)))
	})
	return s.fontData, s.fontErr
}

// GeneratePDF 生成 PDF 报告
func (s *ReportService) GeneratePDF(ctx context.Context, data interface{}) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	model := parseReportModel(data)

	fontData, err := s.loadFont()
	if err != nil {
		return nil, fmt.Errorf("PDF 报告不可用：%w", err)
	}

	doc, err := pdf.NewDocument(pdf.Options{
		FontData: fontData,
		PageSize: pdf.A4,
		Title:    model.Title(),
	})
	if err != nil {
		return nil, fmt.Errorf("初始化 PDF 文档失败: %w", err)
	}
	renderReport(doc, model)

	var buf bytes.Buffer
	if _, err := doc.WriteTo(&buf); err != nil {
		return nil, fmt.Errorf("渲染 PDF 失败: %w", err)
	}
	if missing := doc.MissingRunes(); len(missing) > 0 {
		// 字体缺字已被占位符替换，这里必须留痕，否则用户只会看到一串 '?'
		logger.Warn("PDF 报告存在字体缺字，已用 ? 占位",
			zap.String("font", s.fontPath), zap.String("runes", string(missing)))
	}
	return buf.Bytes(), nil
}

// 报表配色与字号
var (
	clrTitle   = [3]float64{0.16, 0.29, 0.56}
	clrText    = [3]float64{0.13, 0.13, 0.13}
	clrMuted   = [3]float64{0.45, 0.45, 0.45}
	clrSection = [3]float64{0.94, 0.96, 0.99}

	szTitle   = 19.0
	szSection = 12.0
	szBody    = 10.5
	szSmall   = 9.0
)

func setColor(d *pdf.Document, c [3]float64) { d.SetColor(c[0], c[1], c[2]) }

// fillRect 以指定颜色填充矩形（FillRect 复用当前颜色，因此先设色）。
func fillRect(d *pdf.Document, x, y, w, h float64, c [3]float64) {
	setColor(d, c)
	d.FillRect(x, y, w, h)
}

// renderReport 渲染报告正文。
func renderReport(d *pdf.Document, m reportModel) {
	left := d.MarginLeft()
	contentW := d.ContentWidth()

	// 标题
	setColor(d, clrTitle)
	d.TextLine(szTitle, szTitle+10, m.Title())
	setColor(d, clrMuted)
	d.TextLine(szSmall, szSmall+8, "生成时间："+time.Now().Format("2006-01-02 15:04"))
	d.ResetColor()
	d.Space(2)
	d.Rule(left, d.CursorY(), contentW, 1.0)
	d.Space(14)

	if !m.HasData {
		d.Paragraph("本次请求未提供有效的报告数据（data 字段应为对象）。", szBody, 17)
		setFooter(d)
		return
	}

	renderSection(d, "一、基本信息", [][2]string{
		{"姓氏", m.Surname},
		{"性别", genderLabel(m.Gender)},
		{"出生日期", m.BirthDate},
		{"出生时间", m.BirthTime},
	})

	renderSection(d, "二、八字与五行", [][2]string{
		{"八字", m.Bazi},
		{"五行", m.WuXing},
		{"喜用神", m.XiYongShen},
	})

	renderNames(d, m)

	setFooter(d)
}

// setFooter 注册页脚：页码 + 品牌行。此时页数已确定。
func setFooter(d *pdf.Document) {
	d.SetFooter(func(doc *pdf.Document, idx, total int) {
		y := doc.PageSize().Height - doc.MarginBottom() + 22
		doc.Rule(doc.MarginLeft(), y-10, doc.ContentWidth(), 0.5)
		setColor(doc, clrMuted)
		doc.Text(doc.MarginLeft(), y, szSmall, "由 NameMaster 起名系统生成")
		page := fmt.Sprintf("第 %d / %d 页", idx+1, total)
		w := doc.Measure(szSmall, page)
		doc.Text(doc.PageSize().Width-doc.MarginRight()-w, y, szSmall, page)
		doc.ResetColor()
	})
}

// renderSection 渲染「小节标题 + 若干 label/value 行」。
func renderSection(d *pdf.Document, title string, rows [][2]string) {
	if len(rows) == 0 {
		return
	}
	d.EnsureSpace(szSection + 4*16 + 20)

	// 小节标题条
	fillRect(d, d.MarginLeft(), d.CursorY()-2, d.ContentWidth(), szSection+8, clrSection)
	setColor(d, clrTitle)
	d.Text(d.MarginLeft()+8, d.CursorY()+3, szSection, title)
	d.Space(szSection + 12)

	labelW := 74.0
	rowH := 16.0
	for _, r := range rows {
		value := r[1]
		if strings.TrimSpace(value) == "" {
			value = "—"
		}
		lines := d.Wrap(value, szBody, d.ContentWidth()-labelW)
		d.EnsureSpace(float64(len(lines)) * rowH)
		setColor(d, clrMuted)
		d.Text(d.MarginLeft()+2, d.CursorY()+3, szBody, r[0])
		setColor(d, clrText)
		for _, ln := range lines {
			d.Text(d.MarginLeft()+labelW, d.CursorY()+3, szBody, ln)
			d.Space(rowH)
		}
	}
	d.ResetColor()
	d.Space(10)
}

// renderNames 渲染推荐名字表格。
func renderNames(d *pdf.Document, m reportModel) {
	d.EnsureSpace(szSection + 70)
	fillRect(d, d.MarginLeft(), d.CursorY()-2, d.ContentWidth(), szSection+8, clrSection)
	setColor(d, clrTitle)
	d.Text(d.MarginLeft()+8, d.CursorY()+3, szSection, "三、推荐名字")
	d.Space(szSection + 12)

	if len(m.Names) == 0 {
		setColor(d, clrMuted)
		d.Paragraph("请求中未包含推荐名字列表（names 字段）。", szBody, 17)
		d.ResetColor()
		return
	}

	left := d.MarginLeft()
	contentW := d.ContentWidth()
	// 列宽：序号 / 姓名 / 拼音 / 五行 / 评分
	const (
		wIndex = 34.0
		wName  = 76.0
		wPin   = 132.0
		wWu    = 54.0
		wScore = 52.0
	)
	colX := []float64{left, left + wIndex, left + wIndex + wName, left + wIndex + wName + wPin}
	scoreRight := left + contentW

	// 表头
	headerH := 20.0
	fillRect(d, left, d.CursorY()-4, contentW, headerH, clrSection)
	setColor(d, clrMuted)
	d.Text(colX[0]+6, d.CursorY(), szSmall, "序号")
	d.Text(colX[1], d.CursorY(), szSmall, "姓名")
	d.Text(colX[2], d.CursorY(), szSmall, "拼音")
	d.Text(colX[3], d.CursorY(), szSmall, "五行")
	scoreHeader := "评分"
	d.Text(scoreRight-d.Measure(szSmall, scoreHeader), d.CursorY(), szSmall, scoreHeader)
	d.Space(headerH)

	for i, n := range m.Names {
		// 预估整行高度：主行 + 寓意折行
		meaningLines := d.Wrap(n.Meaning, szSmall, contentW-wIndex-6)
		rowH := 18.0 + float64(len(meaningLines))*13.0 + 6
		if d.CursorY()+rowH > d.BottomLimit() {
			d.AddPage()
			d.Space(6)
		}

		setColor(d, clrMuted)
		d.Text(colX[0]+6, d.CursorY()+2, szBody, fmt.Sprintf("%d", i+1))
		setColor(d, clrTitle)
		d.Text(colX[1], d.CursorY()+2, szBody, n.FullName)
		setColor(d, clrText)
		d.Text(colX[2], d.CursorY()+2, szBody, n.Pinyin)
		d.Text(colX[3], d.CursorY()+2, szBody, n.WuXing)
		if n.Score != "" {
			d.Text(scoreRight-d.Measure(szBody, n.Score), d.CursorY()+2, szBody, n.Score)
		}
		d.Space(18)

		if n.Meaning != "" {
			setColor(d, clrMuted)
			for _, ln := range meaningLines {
				d.Text(colX[1], d.CursorY(), szSmall, ln)
				d.Space(13)
			}
		}
		d.Space(6)
		d.Rule(left, d.CursorY(), contentW, 0.4)
	}
	d.ResetColor()
	d.Space(6)
}

func genderLabel(g string) string {
	switch strings.ToLower(strings.TrimSpace(g)) {
	case "male", "男", "1":
		return "男"
	case "female", "女", "0":
		return "女"
	case "":
		return ""
	default:
		return g
	}
}

// ---------- HTML ----------

// GenerateHTML 生成 HTML 报告
//
// 与 PDF 共用 parseReportModel，保证两种格式的字段口径一致。
func (s *ReportService) GenerateHTML(ctx context.Context, data interface{}) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	m := parseReportModel(data)

	var b strings.Builder
	b.WriteString(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>` + html.EscapeString(m.Title()) + `</title>
	<style>
		body { font-family: "PingFang SC", "Microsoft YaHei", Arial, sans-serif; line-height: 1.7; margin: 0; padding: 20px; background: #f5f5f5; color: #232323; }
		.container { max-width: 820px; margin: 0 auto; background: #fff; padding: 40px; border-radius: 10px; box-shadow: 0 0 10px rgba(0,0,0,.1); }
		h1 { color: #294a90; text-align: center; margin: 0 0 6px; }
		.meta { text-align: center; color: #737373; font-size: 13px; margin-bottom: 28px; }
		h2 { color: #294a90; border-bottom: 2px solid #e6ebf5; padding-bottom: 8px; margin: 28px 0 14px; font-size: 17px; }
		.info-item { margin-bottom: 8px; }
		.info-label { font-weight: 600; display: inline-block; width: 88px; color: #737373; }
		table { width: 100%; border-collapse: collapse; font-size: 14px; }
		th { text-align: left; background: #f0f4fb; color: #4a4a4a; font-weight: 600; }
		th, td { padding: 8px 10px; border-bottom: 1px solid #eee; vertical-align: top; }
		.score { font-weight: 700; color: #b3261e; text-align: right; white-space: nowrap; }
		.meaning { color: #5c5c5c; font-size: 13px; }
		.empty { color: #737373; }
		footer { margin-top: 32px; padding-top: 12px; border-top: 1px solid #eee; color: #9a9a9a; font-size: 12px; text-align: center; }
	</style>
</head>
<body>
	<div class="container">
		<h1>` + html.EscapeString(m.Title()) + `</h1>
		<div class="meta">生成时间：` + time.Now().Format("2006-01-02 15:04") + `</div>`)

	if !m.HasData {
		b.WriteString(`
		<p class="empty">本次请求未提供有效的报告数据（data 字段应为对象）。</p>`)
	} else {
		b.WriteString(`
		<h2>一、基本信息</h2>`)
		b.WriteString(htmlInfoRows([][2]string{
			{"姓氏", m.Surname},
			{"性别", genderLabel(m.Gender)},
			{"出生日期", m.BirthDate},
			{"出生时间", m.BirthTime},
		}))

		b.WriteString(`
		<h2>二、八字与五行</h2>`)
		b.WriteString(htmlInfoRows([][2]string{
			{"八字", m.Bazi},
			{"五行", m.WuXing},
			{"喜用神", m.XiYongShen},
		}))

		b.WriteString(`
		<h2>三、推荐名字</h2>`)
		if len(m.Names) == 0 {
			b.WriteString(`
		<p class="empty">请求中未包含推荐名字列表（names 字段）。</p>`)
		} else {
			b.WriteString(`
		<table>
			<thead><tr><th style="width:44px">序号</th><th style="width:88px">姓名</th><th style="width:150px">拼音</th><th style="width:60px">五行</th><th style="width:60px">评分</th></tr></thead>
			<tbody>`)
			for i, n := range m.Names {
				b.WriteString(`
				<tr>
					<td>` + fmt.Sprintf("%d", i+1) + `</td>
					<td><strong>` + html.EscapeString(n.FullName) + `</strong></td>
					<td>` + html.EscapeString(n.Pinyin) + `</td>
					<td>` + html.EscapeString(n.WuXing) + `</td>
					<td class="score">` + html.EscapeString(n.Score) + `</td>
				</tr>`)
				if n.Meaning != "" {
					b.WriteString(`
				<tr><td></td><td colspan="4" class="meaning">` + html.EscapeString(n.Meaning) + `</td></tr>`)
				}
			}
			b.WriteString(`
			</tbody>
		</table>`)
		}
	}

	b.WriteString(`
		<footer>由 NameMaster 起名系统生成</footer>
	</div>
</body>
</html>`)
	return b.String(), nil
}

func htmlInfoRows(rows [][2]string) string {
	var b strings.Builder
	for _, r := range rows {
		v := r[1]
		if strings.TrimSpace(v) == "" {
			v = "—"
		}
		b.WriteString(`
			<div class="info-item"><span class="info-label">` + html.EscapeString(r[0]) +
			`：</span><span>` + html.EscapeString(v) + `</span></div>`)
	}
	return b.String()
}

// ---------- 取值工具 ----------

// rawString 取出字段的原始文本（去首尾空白）。
//
// 刻意不做 HTML 转义：报表模型同时供 PDF 与 HTML 使用，转义只能在 HTML 渲染处
// 做一次，否则 PDF 里会出现 &amp; 这类实体。
func rawString(data map[string]interface{}, key string) string {
	value, ok := data[key]
	if !ok || value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(fmt.Sprintf("%v", value))
}

// getScoreValue 取分数字段并做人类可读的格式化（92 → "92"，92.30 → "92.3"）。
func getScoreValue(data map[string]interface{}, key string) string {
	value, ok := data[key]
	if !ok || value == nil {
		return ""
	}
	switch v := value.(type) {
	case float64:
		if v == 0 {
			return ""
		}
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.1f", v), "0"), ".")
	case float32:
		return getScoreValue(map[string]interface{}{key: float64(v)}, key)
	case int:
		if v == 0 {
			return ""
		}
		return fmt.Sprintf("%d", v)
	case int64:
		if v == 0 {
			return ""
		}
		return fmt.Sprintf("%d", v)
	case string:
		return strings.TrimSpace(v)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", value))
	}
}
