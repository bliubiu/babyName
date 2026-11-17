package services

import (
	"bytes"
	"context"
	"fmt"
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
	font     *pdf.Font // 已解析字体（进程内只解析一次，docs/46 P2）
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
	// Scores 各维评分（详批模式展示前若干维）
	Scores []nameScore
}

// nameScore 单个评分维度
type nameScore struct {
	Name  string
	Score float64
}

// reportModel 报表数据模型。字段名与前端 ReportPayload 保持一致。
type reportModel struct {
	Surname    string
	Gender     string
	BirthDate  string
	BirthTime  string
	Bazi       string
	WuXing     string
	XiYongShen string
	Rishou     string
	Nayin      string
	Zodiac     string
	// Analysis 前端生成的命理详批段落（八字格局/五行平衡/喜用神等）
	Analysis []string
	// WuXingDist 五行分布数值，用于条形图；键为单字 金木水火土
	WuXingDist map[string]float64
	// FullDetail 是否详批模式（含解读与分布图），默认 true
	FullDetail bool
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
	m.Rishou = rawString(raw, "rishou")
	m.Nayin = rawString(raw, "nayin")
	m.Zodiac = rawString(raw, "zodiac")
	m.FullDetail = true // 默认详批；options.detail=summary 时关闭
	if opts, ok := raw["options"].(map[string]interface{}); ok {
		if rawString(opts, "detail") == "summary" {
			m.FullDetail = false
		}
	}

	// 前端命理详批段落
	if list, ok := raw["analysis"].([]interface{}); ok {
		for _, v := range list {
			if s := strings.TrimSpace(fmt.Sprintf("%v", v)); s != "" {
				m.Analysis = append(m.Analysis, s)
			}
		}
	}

	// 五行分布（条形图数据）
	if dist, ok := raw["wuxing_dist"].(map[string]interface{}); ok {
		m.WuXingDist = make(map[string]float64, len(dist))
		for k, v := range dist {
			if f, ok := toFloat(v); ok {
				m.WuXingDist[k] = f
			}
		}
	}

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
				Scores:   parseNameScores(nm),
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
				Scores:   parseNameScores(nm),
			})
		}
	}
	return m
}

// ---------- PDF ----------

// loadFont 载入并解析中文字体（进程内只解析一次，docs/46 P2）。
//
// 字体解析开销较大（CJK 字体 10-20MB，需重建 cmap/glyf/loca 目录），故只解析一次
// 并缓存不可变的 *pdf.Font，后续每次 /report/pdf 仅按用到的 rune 子集化，不再重解析。
// 找不到字体时直接报错：宁可让接口返回「报告生成失败」，也不能输出一份正文全是
// 空白/方框的"报告"——后者用户无从判断是数据问题还是渲染问题。
func (s *ReportService) loadFont() (*pdf.Font, error) {
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
		font, err := pdf.ParseFont(data)
		if err != nil {
			s.fontErr = err
			return
		}
		s.font = font
		s.fontPath = path
		logger.Info("报告中文字体已就绪", zap.String("font", path), zap.Int("bytes", len(data)))
	})
	return s.font, s.fontErr
}

// GeneratePDF 生成 PDF 报告
func (s *ReportService) GeneratePDF(ctx context.Context, data interface{}) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	model := parseReportModel(data)

	font, err := s.loadFont()
	if err != nil {
		return nil, fmt.Errorf("PDF 报告不可用：%w", err)
	}

	doc, err := pdf.NewDocumentWithFont(font, pdf.Options{
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
	clrAccent  = [3]float64{0.78, 0.24, 0.24} // 朱砂红：封面色带与条形图主色

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

	// ---- 封面标题块：色带 + 主标题 + 副标题 + 生成时间 ----
	fillRect(d, left, d.CursorY()-6, contentW, 3, clrAccent)
	d.Space(10)
	setColor(d, clrTitle)
	d.TextLine(szTitle+3, szTitle+12, m.Title())
	setColor(d, clrMuted)
	d.TextLine(szSmall+1, szSmall+8, "NameMaster · 传统命理起名详批")
	d.TextLine(szSmall, szSmall+7, "生成时间："+time.Now().Format("2006-01-02 15:04"))
	d.ResetColor()
	d.Space(4)
	d.Rule(left, d.CursorY(), contentW, 1.0)
	d.Space(14)

	if !m.HasData {
		d.Paragraph("本次请求未提供有效的报告数据（data 字段应为对象）。", szBody, 17)
		setFooter(d)
		return
	}

	// ---- 一、基本信息 ----
	basicRows := [][2]string{
		{"姓氏", m.Surname},
		{"性别", genderLabel(m.Gender)},
		{"出生日期", m.BirthDate},
		{"出生时间", m.BirthTime},
	}
	if m.Zodiac != "" {
		basicRows = append(basicRows, [2]string{"生肖", m.Zodiac})
	}
	renderSection(d, "一、基本信息", basicRows)

	// ---- 二、八字与五行 ----
	baziRows := [][2]string{
		{"八字", m.Bazi},
		{"五行", m.WuXing},
		{"喜用神", m.XiYongShen},
	}
	if m.Rishou != "" {
		baziRows = append(baziRows, [2]string{"日主", m.Rishou})
	}
	if m.Nayin != "" {
		baziRows = append(baziRows, [2]string{"纳音", m.Nayin})
	}
	renderSection(d, "二、八字与五行", baziRows)

	// ---- 详批模式：五行分布图 + 命理依据解读 ----
	if m.FullDetail {
		if hasWuXingDist(m.WuXingDist) {
			renderWuXingChart(d, m.WuXingDist)
		}
		if len(m.Analysis) > 0 {
			renderAnalysis(d, m.Analysis)
		}
	}

	renderNames(d, m)

	d.Space(6)
	setColor(d, clrMuted)
	d.Paragraph("本报告依据传统命理与汉字文化生成，仅供取名参考，不构成任何命运断言。", szSmall, 14)
	d.ResetColor()

	setFooter(d)
}

// hasWuXingDist 五行分布是否有非零数据。
func hasWuXingDist(dist map[string]float64) bool {
	for _, v := range dist {
		if v > 0 {
			return true
		}
	}
	return false
}

// renderWuXingChart 画五行分布横向条形图（复用 FillRect / Text）。
func renderWuXingChart(d *pdf.Document, dist map[string]float64) {
	order := []string{"金", "木", "水", "火", "土"}
	var max float64
	for _, k := range order {
		if v := dist[k]; v > max {
			max = v
		}
	}
	if max <= 0 {
		return
	}
	rowH := 18.0
	d.EnsureSpace(szSection + rowH*5 + 24)
	fillRect(d, d.MarginLeft(), d.CursorY()-2, d.ContentWidth(), szSection+8, clrSection)
	setColor(d, clrTitle)
	d.Text(d.MarginLeft()+8, d.CursorY()+3, szSection, "三、五行分布")
	d.Space(szSection + 12)

	left := d.MarginLeft()
	labelW := 30.0
	barMaxW := d.ContentWidth() - labelW - 46
	for _, k := range order {
		v := dist[k]
		setColor(d, clrText)
		d.Text(left, d.CursorY()+3, szBody, k)
		// 条形底槽
		fillRect(d, left+labelW, d.CursorY(), barMaxW, 10, clrSection)
		if v > 0 {
			fillRect(d, left+labelW, d.CursorY(), barMaxW*(v/max), 10, clrAccent)
		}
		setColor(d, clrMuted)
		d.Text(left+labelW+barMaxW+8, d.CursorY()+3, szSmall, fmt.Sprintf("%d", int(v)))
		d.Space(rowH)
	}
	d.ResetColor()
	d.Space(8)
}

// renderAnalysis 渲染命理详批段落（每段独立成行，自动折行与分页）。
func renderAnalysis(d *pdf.Document, paragraphs []string) {
	d.EnsureSpace(szSection + 30)
	fillRect(d, d.MarginLeft(), d.CursorY()-2, d.ContentWidth(), szSection+8, clrSection)
	setColor(d, clrTitle)
	d.Text(d.MarginLeft()+8, d.CursorY()+3, szSection, "四、命理依据解读")
	d.Space(szSection + 12)

	for _, p := range paragraphs {
		lines := d.Wrap(p, szBody-0.5, d.ContentWidth()-8)
		d.EnsureSpace(float64(len(lines))*15 + 6)
		setColor(d, clrText)
		for _, ln := range lines {
			d.Text(d.MarginLeft()+4, d.CursorY()+3, szBody-0.5, ln)
			d.Space(15)
		}
		d.Space(6)
	}
	d.ResetColor()
	d.Space(4)
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

// nameTableLayout 名字表格列布局，按内容宽度比例分配，消除中段大空白。
type nameTableLayout struct {
	left       float64
	contentW   float64
	colX       [4]float64 // 序号 / 姓名 / 拼音 / 五行
	scoreRight float64
}

func newNameTableLayout(d *pdf.Document) nameTableLayout {
	left := d.MarginLeft()
	contentW := d.ContentWidth()
	// 用相对权重分配：序号 7% / 姓名 18% / 拼音 42% / 五行 18% / 评分 15%
	wIndex := contentW * 0.07
	wName := contentW * 0.18
	wPin := contentW * 0.42
	scoreW := contentW * 0.15
	return nameTableLayout{
		left:       left,
		contentW:   contentW,
		colX:       [4]float64{left, left + wIndex, left + wIndex + wName, left + wIndex + wName + wPin},
		scoreRight: left + contentW - scoreW/2,
	}
}

// drawNameTableHeader 画表头；换页后必须重画，否则续页列标题丢失。
func drawNameTableHeader(d *pdf.Document, l nameTableLayout) {
	headerH := 20.0
	fillRect(d, l.left, d.CursorY()-4, l.contentW, headerH, clrSection)
	setColor(d, clrMuted)
	d.Text(l.colX[0]+6, d.CursorY(), szSmall, "序号")
	d.Text(l.colX[1], d.CursorY(), szSmall, "姓名")
	d.Text(l.colX[2], d.CursorY(), szSmall, "拼音")
	d.Text(l.colX[3], d.CursorY(), szSmall, "五行")
	scoreHeader := "评分"
	d.Text(l.scoreRight-d.Measure(szSmall, scoreHeader), d.CursorY(), szSmall, scoreHeader)
	d.Space(headerH)
}

// renderNames 渲染推荐名字表格。
func renderNames(d *pdf.Document, m reportModel) {
	d.EnsureSpace(szSection + 70)
	fillRect(d, d.MarginLeft(), d.CursorY()-2, d.ContentWidth(), szSection+8, clrSection)
	setColor(d, clrTitle)
	d.Text(d.MarginLeft()+8, d.CursorY()+3, szSection, "五、推荐名字")
	d.Space(szSection + 12)

	if len(m.Names) == 0 {
		setColor(d, clrMuted)
		d.Paragraph("请求中未包含推荐名字列表（names 字段）。", szBody, 17)
		d.ResetColor()
		return
	}

	l := newNameTableLayout(d)
	drawNameTableHeader(d, l)

	for i, n := range m.Names {
		// 预估整行高度：主行 + 寓意折行 + 详批模式下各维评分一行
		meaningLines := d.Wrap(n.Meaning, szSmall, l.contentW-(l.colX[1]-l.left)-6)
		scoreLine := ""
		if m.FullDetail && len(n.Scores) > 0 {
			scoreLine = formatNameScores(n.Scores)
		}
		scoreLines := 0
		if scoreLine != "" {
			scoreLines = len(d.Wrap(scoreLine, szSmall, l.contentW-8))
		}
		rowH := 18.0 + float64(len(meaningLines))*13.0 + float64(scoreLines)*13.0 + 6
		if d.CursorY()+rowH > d.BottomLimit() {
			d.AddPage()
			d.Space(6)
			drawNameTableHeader(d, l) // 续页重画表头
		}

		// 斑马纹：偶数行浅底，提升可读性
		if i%2 == 1 {
			fillRect(d, l.left, d.CursorY()-3, l.contentW, 18, clrSection)
		}
		setColor(d, clrMuted)
		d.Text(l.colX[0]+6, d.CursorY()+2, szBody, fmt.Sprintf("%d", i+1))
		setColor(d, clrTitle)
		d.Text(l.colX[1], d.CursorY()+2, szBody, n.FullName)
		setColor(d, clrText)
		d.Text(l.colX[2], d.CursorY()+2, szBody, n.Pinyin)
		d.Text(l.colX[3], d.CursorY()+2, szBody, n.WuXing)
		if n.Score != "" {
			d.Text(l.scoreRight-d.Measure(szBody, n.Score), d.CursorY()+2, szBody, n.Score)
		}
		d.Space(18)

		if n.Meaning != "" {
			setColor(d, clrMuted)
			for _, ln := range meaningLines {
				d.Text(l.colX[1], d.CursorY(), szSmall, ln)
				d.Space(13)
			}
		}
		if scoreLine != "" {
			setColor(d, clrAccent)
			for _, ln := range d.Wrap(scoreLine, szSmall, l.contentW-8) {
				d.Text(l.colX[1], d.CursorY(), szSmall-0.5, ln)
				d.Space(13)
			}
		}
		d.Space(6)
		d.Rule(l.left, d.CursorY(), l.contentW, 0.4)
	}
	d.ResetColor()
	d.Space(6)
}

// formatNameScores 把各维评分压成一行可读文本，如「五行 92 · 音律 88 · 字义 90」。
func formatNameScores(scores []nameScore) string {
	parts := make([]string, 0, len(scores))
	for _, s := range scores {
		if s.Name == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s %.0f", s.Name, s.Score))
	}
	return strings.Join(parts, " · ")
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

// parseNameScores 解析单名字的各维评分数组（前端 score_detail 投影）。
func parseNameScores(data map[string]interface{}) []nameScore {
	raw, ok := data["scores"].([]interface{})
	if !ok {
		return nil
	}
	var out []nameScore
	for _, v := range raw {
		m, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		score, ok := toFloat(m["score"])
		if !ok {
			continue
		}
		out = append(out, nameScore{Name: rawString(m, "name"), Score: score})
	}
	return out
}

// toFloat 把 JSON 解码后的数值（float64/int/字符串数字）统一转为 float64。
func toFloat(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case string:
		var f float64
		if _, err := fmt.Sscanf(strings.TrimSpace(n), "%f", &f); err == nil {
			return f, true
		}
	}
	return 0, false
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
