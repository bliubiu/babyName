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
	// ScoreValue 评分数值（用于排序高亮/摘要，Score 保留展示格式）
	ScoreValue float64
	Meaning    string
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
	// Hexagram 卦象摘要（本卦/卦辞/象辞/卦德/解读）
	Hexagram *reportHexagram
	// Ziwei 紫微斗数摘要（命宫/身宫/五行局/四化/十二宫主星）
	Ziwei *reportZiwei
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

// reportHexagram 卦象章节数据。
type reportHexagram struct {
	Number         int
	Name           string
	Symbol         string
	Upper          string
	Lower          string
	GuaCi          string
	XiangCi        string
	TuanCi         string
	GuaDe          string
	Interpretation string
}

// reportZiwei 紫微斗数章节数据。
type reportZiwei struct {
	MingGong     string
	ShenGong     string
	FiveElements string
	Soul         string
	Body         string
	NianZhu      string
	YueZhu       string
	RiZhu        string
	ShiZhu       string
	SiHua        map[string]string
	ZhuXing      map[string][]string
	Brightness   map[string]string
	Analysis     string
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
	m.Hexagram = parseReportHexagram(raw)
	m.Ziwei = parseReportZiwei(raw)
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
				FullName:   rawString(nm, "full_name"),
				Pinyin:     rawString(nm, "pinyin"),
				WuXing:     rawString(nm, "wuxing"),
				Score:      getScoreValue(nm, "score"),
				ScoreValue: getScoreFloat(nm, "score"),
				Meaning:    rawString(nm, "meaning"),
				Scores:     parseNameScores(nm),
			})
		}
	case []map[string]interface{}:
		for _, nm := range names {
			m.Names = append(m.Names, reportName{
				FullName:   rawString(nm, "full_name"),
				Pinyin:     rawString(nm, "pinyin"),
				WuXing:     rawString(nm, "wuxing"),
				Score:      getScoreValue(nm, "score"),
				ScoreValue: getScoreFloat(nm, "score"),
				Meaning:    rawString(nm, "meaning"),
				Scores:     parseNameScores(nm),
			})
		}
	}
	return m
}

// parseReportHexagram 解析前端透传的本卦摘要；无该字段或格式不符时返回 nil。
func parseReportHexagram(raw map[string]interface{}) *reportHexagram {
	hm, ok := raw["hexagram"].(map[string]interface{})
	if !ok {
		return nil
	}
	h := &reportHexagram{
		Number:         int(getScoreFloat(hm, "number")),
		Name:           rawString(hm, "name"),
		Symbol:         rawString(hm, "symbol"),
		Upper:          rawString(hm, "upper"),
		Lower:          rawString(hm, "lower"),
		GuaCi:          rawString(hm, "gua_ci"),
		XiangCi:        rawString(hm, "xiang_ci"),
		TuanCi:         rawString(hm, "tuan_ci"),
		GuaDe:          rawString(hm, "gua_de"),
		Interpretation: rawString(hm, "interpretation"),
	}
	if h.Name == "" && h.GuaCi == "" && h.Number == 0 {
		return nil
	}
	return h
}

// parseReportZiwei 解析紫微摘要；十二宫主星与生年四化都可能为空。
func parseReportZiwei(raw map[string]interface{}) *reportZiwei {
	zm, ok := raw["ziwei"].(map[string]interface{})
	if !ok {
		return nil
	}
	z := &reportZiwei{
		MingGong:     rawString(zm, "ming_gong"),
		ShenGong:     rawString(zm, "shen_gong"),
		FiveElements: rawString(zm, "five_elements"),
		Soul:         rawString(zm, "soul"),
		Body:         rawString(zm, "body"),
		NianZhu:      rawString(zm, "nian_zhu"),
		YueZhu:       rawString(zm, "yue_zhu"),
		RiZhu:        rawString(zm, "ri_zhu"),
		ShiZhu:       rawString(zm, "shi_zhu"),
		Analysis:     rawString(zm, "analysis"),
		SiHua:        rawStringMap(zm, "si_hua"),
		ZhuXing:      rawStringSliceMap(zm, "zhu_xing"),
		Brightness:   rawStringMap(zm, "brightness"),
	}
	if z.MingGong == "" && z.FiveElements == "" && len(z.ZhuXing) == 0 {
		return nil
	}
	return z
}

func rawStringMap(m map[string]interface{}, key string) map[string]string {
	src, ok := m[key].(map[string]interface{})
	if !ok {
		return nil
	}
	out := make(map[string]string, len(src))
	for k, v := range src {
		if s, ok := v.(string); ok && s != "" {
			out[k] = s
		}
	}
	return out
}

func rawStringSliceMap(m map[string]interface{}, key string) map[string][]string {
	src, ok := m[key].(map[string]interface{})
	if !ok {
		return nil
	}
	out := make(map[string][]string, len(src))
	for k, v := range src {
		list, ok := v.([]interface{})
		if !ok {
			continue
		}
		for _, item := range list {
			if s, ok := item.(string); ok && s != "" {
				out[k] = append(out[k], s)
			}
		}
	}
	return out
}

// ---------- PDF ----------

// loadFont 载入并解析中文字体（进程内只解析一次，docs/46 P2）。
//
// 字体来源优先级：
//  1. NAMER_PDF_FONT 指定的外部字体（兼容自定义场景，显式覆盖优先）；
//  2. 内置开源字体 Noto Sans SC（默认）——随二进制分发，任何机器开箱可用，
//     不依赖系统是否安装中文字体；
//  3. 平台系统字体（仅当内置字体解析失败这类异常情况才回落）。
//
// 字体解析开销较大（CJK 字体 10-20MB，需重建 cmap/glyf/loca 目录），故只解析一次
// 并缓存不可变的 *pdf.Font，后续每次 /report/pdf 仅按用到的 rune 子集化，不再重解析。
// 所有来源都失败时直接报错：宁可让接口返回「报告生成失败」，也不能输出一份正文全是
// 空白/方框的"报告"——后者用户无从判断是数据问题还是渲染问题。
func (s *ReportService) loadFont() (*pdf.Font, error) {
	s.fontOnce.Do(func() {
		// 1. 显式覆盖：调用方指定的外部字体优先
		if env := os.Getenv("NAMER_PDF_FONT"); env != "" {
			if font, path, err := loadFontFromPath(env); err == nil {
				s.font, s.fontPath = font, path
				logger.Info("报告使用外部中文字体（NAMER_PDF_FONT）", zap.String("font", path))
				return
			} else {
				logger.Warn("NAMER_PDF_FONT 指定的字体不可用，回落内置字体",
					zap.String("font", env), zap.Error(err))
			}
		}

		// 2. 默认：内置开源字体（开箱可用，零外部依赖）
		if font, err := pdf.LoadEmbeddedFont(); err == nil {
			s.font, s.fontPath = font, pdf.EmbeddedFontName
			logger.Info("报告使用内置开源中文字体", zap.String("font", pdf.EmbeddedFontName))
			return
		} else {
			logger.Warn("内置中文字体不可用，尝试系统字体", zap.Error(err))
		}

		// 3. 兜底：平台系统字体（内置字体异常的极端情况）
		path, err := pdf.FindSystemCJKFont()
		if err != nil {
			s.fontErr = fmt.Errorf("PDF 字体不可用：内置字体解析失败，且系统未找到可用中文字体：%w", err)
			return
		}
		font, path, err := loadFontFromPath(path)
		if err != nil {
			s.fontErr = err
			return
		}
		s.font, s.fontPath = font, path
		logger.Info("报告使用系统中文字体", zap.String("font", path))
	})
	return s.font, s.fontErr
}

// loadFontFromPath 读取并解析指定路径的中文字体，返回句柄与规范化后的路径。
func loadFontFromPath(path string) (*pdf.Font, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, path, fmt.Errorf("读取中文字体 %s 失败: %w", path, err)
	}
	font, err := pdf.ParseFont(data)
	if err != nil {
		return nil, path, err
	}
	return font, path, nil
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

	szTitle      = 19.0
	szCoverTitle = 30.0
	szCoverValue = 12.0
	szSection    = 12.0
	szBody       = 10.5
	szSmall      = 9.0
)

func setColor(d *pdf.Document, c [3]float64) { d.SetColor(c[0], c[1], c[2]) }

// fillRect 以指定颜色填充矩形（FillRect 复用当前颜色，因此先设色）。
func fillRect(d *pdf.Document, x, y, w, h float64, c [3]float64) {
	setColor(d, c)
	d.FillRect(x, y, w, h)
}

// renderReport 渲染报告正文。
func renderReport(d *pdf.Document, m reportModel) {
	renderCover(d, m)
	d.AddPage()

	left := d.MarginLeft()
	contentW := d.ContentWidth()

	// ---- 封面标题块：色带 + 主标题 + 副标题/生成时间 ----
	fillRect(d, left, d.CursorY()-6, contentW, 3, clrAccent)
	d.Space(10)
	setColor(d, clrTitle)
	d.TextLine(szTitle+3, szTitle+12, m.Title())
	setColor(d, clrMuted)
	subtitle := "NameMaster · 传统命理起名详批"
	d.Text(left, d.CursorY(), szSmall+1, subtitle)
	timeText := "生成时间：" + time.Now().Format("2006-01-02 15:04")
	d.Text(d.PageSize().Width-d.MarginRight()-d.Measure(szSmall+1, timeText),
		d.CursorY(), szSmall+1, timeText)
	d.Space(szSmall + 10)
	d.ResetColor()
	d.Rule(left, d.CursorY(), contentW, 0.8)
	d.Space(12)

	if !m.HasData {
		d.Paragraph("本次请求未提供有效的报告数据（data 字段应为对象）。", szBody, 17)
		setFooter(d)
		return
	}

	// ---- 报告摘要条：一眼看到推荐数量/最高分/喜用神 ----
	summary := fmt.Sprintf("共 %d 个推荐名字", len(m.Names))
	if top := highestScore(m.Names); top > 0 {
		summary += fmt.Sprintf("　·　最高评分 %s", formatScore(top))
	}
	if m.XiYongShen != "" {
		summary += "　·　喜用神 " + m.XiYongShen
	}
	fillRect(d, left, d.CursorY()-3, contentW, 24, clrSection)
	setColor(d, clrTitle)
	d.Text(left+10, d.CursorY()+5, szBody+0.5, summary)
	d.ResetColor()
	d.Space(32)

	sec := &sectionCounter{}

	// ---- 基本信息 ----
	basicRows := [][2]string{
		{"姓氏", m.Surname},
		{"性别", genderLabel(m.Gender)},
		{"出生日期", m.BirthDate},
		{"出生时间", m.BirthTime},
	}
	if m.Zodiac != "" {
		basicRows = append(basicRows, [2]string{"生肖", m.Zodiac})
	}
	renderInfoGrid(d, sec.title("基本信息"), basicRows)

	// ---- 八字与五行 ----
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
	renderInfoGrid(d, sec.title("八字与五行"), baziRows)

	// ---- 详批模式：五行分布图 + 命理依据解读 ----
	if m.FullDetail && hasWuXingDist(m.WuXingDist) {
		renderWuXingChart(d, m.WuXingDist, sec.title("五行分布"))
	}
	if m.FullDetail && len(m.Analysis) > 0 {
		renderAnalysis(d, m.Analysis, sec.title("命理依据解读"))
	}
	if m.Hexagram != nil {
		renderHexagram(d, *m.Hexagram, sec.title("卦象"))
	}
	if m.Ziwei != nil {
		renderZiwei(d, *m.Ziwei, sec.title("紫微斗数"))
	}

	renderNames(d, m, sec.title("推荐名字"))

	d.Space(6)
	setColor(d, clrMuted)
	d.Paragraph("本报告依据传统命理与汉字文化生成，仅供取名参考，不构成任何命运断言。", szSmall, 14)
	d.ResetColor()

	setFooter(d)
}

// renderCover 渲染封面页。
//
// 封面独立成页，只放身份信息与报告摘要，不放表格；正文从第二页开始。
// 页脚绘制时跳过第 1 页（见 setFooter），内容页从「第 1 页」重新计数。
func renderCover(d *pdf.Document, m reportModel) {
	left := d.MarginLeft()
	contentW := d.ContentWidth()
	pageH := d.PageSize().Height

	fillRect(d, left, 92, contentW, 4, clrAccent)
	setColor(d, clrAccent)
	d.Text(left, 112, szSmall+1, "NAME REPORT  ·  起名详批")
	setColor(d, clrTitle)
	d.Text(left, 150, szCoverTitle, m.Title())

	setColor(d, clrMuted)
	d.Text(left, 192, szSmall+1, "传统命理  ·  八字五行  ·  易经卦象  ·  紫微斗数")

	// 中部信息卡：两列，放最关键的出生与命理信息
	cardY := 260.0
	cardH := 176.0
	fillRect(d, left, cardY, contentW, cardH, clrSection)
	col2 := left + contentW/2 + 8
	coverCell := func(x, y float64, label, value string) {
		setColor(d, clrMuted)
		d.Text(x, y, szSmall, label)
		setColor(d, clrTitle)
		d.Text(x, y+15, szCoverValue, value)
	}
	coverCell(left+24, cardY+26, "姓氏", fallbackDash(m.Surname))
	coverCell(col2, cardY+26, "性别", genderLabel(m.Gender))
	coverCell(left+24, cardY+72, "出生日期", fallbackDash(m.BirthDate))
	coverCell(col2, cardY+72, "出生时间", fallbackDash(m.BirthTime))
	coverCell(left+24, cardY+118, "生肖", fallbackDash(m.Zodiac))
	coverCell(col2, cardY+118, "喜用神", fallbackDash(m.XiYongShen))

	// 卡片外一条八字摘要（有数据才画）
	if m.Bazi != "" {
		setColor(d, clrMuted)
		d.Text(left, cardY+cardH+24, szSmall+1, "八字")
		setColor(d, clrTitle)
		d.Text(left+42, cardY+cardH+24, szBody+1, m.Bazi)
	}

	// 封面底部品牌与日期
	ruleY := pageH - 110
	d.Rule(left, ruleY, contentW, 0.8)
	setColor(d, clrMuted)
	d.Text(left, ruleY+16, szSmall, "NameMaster · 传统命理起名系统")
	dateText := time.Now().Format("2006-01-02")
	d.Text(d.PageSize().Width-d.MarginRight()-d.Measure(szSmall, dateText),
		ruleY+16, szSmall, dateText)
	d.ResetColor()
}

// renderHexagram 渲染卦象章节：短字段网格 + 卦辞/象辞/彖传/解读文本块。
func renderHexagram(d *pdf.Document, h reportHexagram, title string) {
	// 卦象含多条长文本，剩余空间不足时整节换页，避免「卦辞」与「象辞」被页脚截断
	ensureSectionSpace(d, 300)
	head := h.Name
	if h.Number > 0 {
		head = fmt.Sprintf("第 %d 卦 · %s", h.Number, head)
	}
	rows := [][2]string{{"本卦", head}, {"卦德", fallbackDash(h.GuaDe)}}
	if h.Upper != "" || h.Lower != "" {
		rows = append(rows, [2]string{"上下卦", fmt.Sprintf("上%s / 下%s", fallbackDash(h.Upper), fallbackDash(h.Lower))})
	}
	renderInfoGrid(d, title, rows)

	renderTextBlock(d, "卦辞", h.GuaCi)
	renderTextBlock(d, "象辞", h.XiangCi)
	renderTextBlock(d, "彖传", h.TuanCi)
	renderTextBlock(d, "解读", h.Interpretation)
}

// renderZiwei 渲染紫微斗数章节：命盘摘要 + 十二宫主星 + 排盘分析。
func renderZiwei(d *pdf.Document, z reportZiwei, title string) {
	// 命盘摘要 + 十二宫主星 + 分析通常需要 300pt 以上，空间不足时另起一页
	ensureSectionSpace(d, 340)
	rows := [][2]string{
		{"命宫", fallbackDash(z.MingGong)},
		{"身宫", fallbackDash(z.ShenGong)},
		{"五行局", fallbackDash(z.FiveElements)},
		{"命主 / 身主", strings.TrimSpace(fallbackDash(z.Soul) + " / " + fallbackDash(z.Body))},
	}
	sizhu := strings.TrimSpace(strings.Join(nonEmpty([]string{z.NianZhu, z.YueZhu, z.RiZhu, z.ShiZhu}), " "))
	if sizhu != "" {
		rows = append(rows, [2]string{"四柱", sizhu})
	}
	rows = append(rows, [2]string{"生年四化", formatSiHua(z.SiHua)})
	renderInfoGrid(d, title, rows)

	// 十二宫主星：固定宫序，缺失的宫不输出；亮度拼在星名后
	palaces := [][2]string{}
	for _, name := range ziweiPalaceOrder() {
		stars, ok := lookupPalace(z.ZhuXing, name)
		if !ok || len(stars) == 0 {
			continue
		}
		labels := make([]string, 0, len(stars))
		for _, s := range stars {
			if b := z.Brightness[s]; b != "" {
				labels = append(labels, s+"("+b+")")
			} else {
				labels = append(labels, s)
			}
		}
		palaces = append(palaces, [2]string{name, strings.Join(labels, "、")})
	}
	if len(palaces) > 0 {
		renderInfoGrid(d, "十二宫主星", palaces)
	}
	renderTextBlock(d, "排盘分析", z.Analysis)
}

func ziweiPalaceOrder() []string {
	return []string{"命宫", "兄弟", "夫妻", "子女", "财帛", "疾厄", "迁移", "交友", "官禄", "田宅", "福德", "父母"}
}

func lookupPalace(m map[string][]string, name string) ([]string, bool) {
	if stars, ok := m[name]; ok {
		return stars, true
	}
	if stars, ok := m[name+"宫"]; ok {
		return stars, true
	}
	for k, stars := range m {
		if strings.TrimSuffix(k, "宫") == name {
			return stars, true
		}
	}
	return nil, false
}

func formatSiHua(m map[string]string) string {
	if len(m) == 0 {
		return "—"
	}
	order := []string{"禄", "权", "科", "忌"}
	parts := make([]string, 0, len(order))
	for _, k := range order {
		if v := m[k]; v != "" {
			parts = append(parts, k+" "+v)
		}
	}
	return strings.Join(parts, "  ·  ")
}

func nonEmpty(list []string) []string {
	out := make([]string, 0, len(list))
	for _, s := range list {
		if strings.TrimSpace(s) != "" {
			out = append(out, s)
		}
	}
	return out
}

func fallbackDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}

// ensureSectionSpace 若当前页剩余高度不足以容纳 need，则另起一页。
func ensureSectionSpace(d *pdf.Document, need float64) {
	if d.CursorY()+need > d.BottomLimit() {
		d.AddPage()
		d.Space(6)
	}
}

// renderTextBlock 渲染「小标签 + 折行正文」，用于卦辞/象辞/分析等长文本。
func renderTextBlock(d *pdf.Document, label, text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	lines := d.Wrap(text, szBody-0.5, d.ContentWidth()-8)
	d.EnsureSpace(16 + float64(len(lines))*15 + 8)
	setColor(d, clrAccent)
	d.Text(d.MarginLeft()+4, d.CursorY()+2, szSmall, label)
	d.Space(15)
	setColor(d, clrText)
	for _, ln := range lines {
		d.Text(d.MarginLeft()+4, d.CursorY()+3, szBody-0.5, ln)
		d.Space(15)
	}
	d.ResetColor()
	d.Space(6)
}

// sectionCounter 生成「一、二、三…」连续编号，避免因跳过某些区块而出现断号。
type sectionCounter struct{ n int }

func (s *sectionCounter) title(name string) string {
	s.n++
	return chineseNumeral(s.n) + "、" + name
}

func chineseNumeral(n int) string {
	digits := []rune("零一二三四五六七八九")
	if n <= 0 || n >= 100 {
		return fmt.Sprintf("%d", n)
	}
	if n < 10 {
		return string(digits[n])
	}
	tens, ones := n/10, n%10
	if tens == 1 {
		if ones == 0 {
			return "十"
		}
		return "十" + string(digits[ones])
	}
	if ones == 0 {
		return string(digits[tens]) + "十"
	}
	return string(digits[tens]) + "十" + string(digits[ones])
}

// highestScore 返回推荐名字中的最高评分（无有效分数时为 0）。
func highestScore(names []reportName) float64 {
	var max float64
	for _, n := range names {
		if n.ScoreValue > max {
			max = n.ScoreValue
		}
	}
	return max
}

// gradeLabel 与领域层 scoreToGrade 同阈值，保证 PDF 与结果页评级一致。
func gradeLabel(score float64) string {
	switch {
	case score >= 90:
		return "上上"
	case score >= 80:
		return "上吉"
	case score >= 70:
		return "中吉"
	case score >= 60:
		return "中平"
	case score >= 50:
		return "中下"
	default:
		return "下下"
	}
}

// formatScore 统一评分的展示精度：整数不带小数，其余保留一位。
func formatScore(v float64) string {
	if v == float64(int64(v)) {
		return fmt.Sprintf("%d", int64(v))
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.1f", v), "0"), ".")
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
func renderWuXingChart(d *pdf.Document, dist map[string]float64, title string) {
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
	d.Text(d.MarginLeft()+8, d.CursorY()+3, szSection, title)
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
func renderAnalysis(d *pdf.Document, paragraphs []string, title string) {
	d.EnsureSpace(szSection + 30)
	fillRect(d, d.MarginLeft(), d.CursorY()-2, d.ContentWidth(), szSection+8, clrSection)
	setColor(d, clrTitle)
	d.Text(d.MarginLeft()+8, d.CursorY()+3, szSection, title)
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
		// 封面不显示页脚，正文页从「第 1 页」重新计数
		if idx == 0 {
			return
		}
		y := doc.PageSize().Height - doc.MarginBottom() + 22
		doc.Rule(doc.MarginLeft(), y-10, doc.ContentWidth(), 0.5)
		setColor(doc, clrMuted)
		doc.Text(doc.MarginLeft(), y, szSmall, "由 NameMaster 起名系统生成")
		page := fmt.Sprintf("第 %d / %d 页", idx, total-1)
		w := doc.Measure(szSmall, page)
		doc.Text(doc.PageSize().Width-doc.MarginRight()-w, y, szSmall, page)
		doc.ResetColor()
	})
}

// renderInfoGrid 渲染「小节标题 + 双列 label/value 网格」。
//
// 基本信息/八字字段都较短，单列排布会在右半边留下大片空白；改为每行放两组，
// 字段从 5 行压到 3 行，版面更紧凑。值过长时仍在本单元格内折行，不会跨列。
func renderInfoGrid(d *pdf.Document, title string, rows [][2]string) {
	if len(rows) == 0 {
		return
	}

	left := d.MarginLeft()
	contentW := d.ContentWidth()
	gutter := 20.0
	cellW := (contentW - gutter) / 2
	labelW := 52.0
	valueW := cellW - labelW
	lineH := 15.0

	// 先算总高，尽量让整节落在同一页，避免标题与字段被拆开
	totalH := szSection + 12
	for i := 0; i < len(rows); i += 2 {
		h := infoRowHeight(d, rows, i, valueW, lineH)
		totalH += h
	}
	d.EnsureSpace(totalH + 10)

	// 小节标题条
	fillRect(d, left, d.CursorY()-2, contentW, szSection+8, clrSection)
	setColor(d, clrTitle)
	d.Text(left+8, d.CursorY()+3, szSection, title)
	d.Space(szSection + 12)

	for i := 0; i < len(rows); i += 2 {
		rowH := infoRowHeight(d, rows, i, valueW, lineH)
		d.EnsureSpace(rowH)
		drawInfoCell(d, rows[i], left, d.CursorY(), labelW, valueW, lineH)
		if i+1 < len(rows) {
			drawInfoCell(d, rows[i+1], left+cellW+gutter, d.CursorY(), labelW, valueW, lineH)
		}
		d.Space(rowH)
	}
	d.ResetColor()
	d.Space(10)
}

// infoRowHeight 计算网格中一行（最多两个字段）的高度。
func infoRowHeight(d *pdf.Document, rows [][2]string, i int, valueW, lineH float64) float64 {
	maxLines := len(d.Wrap(infoValue(rows[i]), szBody, valueW))
	if i+1 < len(rows) {
		if n := len(d.Wrap(infoValue(rows[i+1]), szBody, valueW)); n > maxLines {
			maxLines = n
		}
	}
	return float64(maxLines)*lineH + 6
}

func infoValue(row [2]string) string {
	if strings.TrimSpace(row[1]) == "" {
		return "—"
	}
	return row[1]
}

// drawInfoCell 画一个 label/value 单元格。
func drawInfoCell(d *pdf.Document, row [2]string, x, y, labelW, valueW, lineH float64) {
	setColor(d, clrMuted)
	d.Text(x, y+3, szBody, row[0])
	setColor(d, clrText)
	for _, ln := range d.Wrap(infoValue(row), szBody, valueW) {
		d.Text(x+labelW, y+3, szBody, ln)
		y += lineH
	}
}

// nameTableLayout 名字表格列布局，按内容宽度比例分配，消除中段大空白。
type nameTableLayout struct {
	left       float64
	contentW   float64
	colX       [4]float64 // 序号 / 姓名 / 拼音 / 五行
	gradeX     float64    // 评级
	scoreRight float64
}

func newNameTableLayout(d *pdf.Document) nameTableLayout {
	left := d.MarginLeft()
	contentW := d.ContentWidth()
	// 相对权重：序号 7% / 姓名 21% / 拼音 31% / 五行 14%，评级 70% 起，评分右对齐
	wIndex := contentW * 0.07
	wName := contentW * 0.21
	wPin := contentW * 0.31
	wWuxing := contentW * 0.14
	return nameTableLayout{
		left:     left,
		contentW: contentW,
		colX: [4]float64{
			left,
			left + wIndex,
			left + wIndex + wName,
			left + wIndex + wName + wPin + wWuxing*0.35,
		},
		gradeX:     left + contentW*0.70,
		scoreRight: left + contentW,
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
	d.Text(l.gradeX, d.CursorY(), szSmall, "评级")
	scoreHeader := "评分"
	d.Text(l.scoreRight-d.Measure(szSmall, scoreHeader)-4, d.CursorY(), szSmall, scoreHeader)
	d.Space(headerH)
}

// renderNames 渲染推荐名字表格。
//
// 排版要点：
//   - 前 3 名用左侧色条标记，评分按档位着色；
//   - 每行主行只放结构化短字段，寓意/维度分在下一行起折行，避免大段文字挤进列内；
//   - 换页时重画表头，保证续页仍可读。
func renderNames(d *pdf.Document, m reportModel, title string) {
	d.EnsureSpace(szSection + 72)
	fillRect(d, d.MarginLeft(), d.CursorY()-2, d.ContentWidth(), szSection+8, clrSection)
	setColor(d, clrTitle)
	d.Text(d.MarginLeft()+8, d.CursorY()+3, szSection, title)
	d.Space(szSection + 12)

	if len(m.Names) == 0 {
		setColor(d, clrMuted)
		d.Paragraph("请求中未包含推荐名字列表（names 字段）。", szBody, 17)
		d.ResetColor()
		return
	}

	l := newNameTableLayout(d)
	drawNameTableHeader(d, l)

	rankColors := [3][3]float64{
		{0.78, 0.24, 0.24}, // 第 1 名：朱砂
		{0.82, 0.62, 0.22}, // 第 2 名：赭金
		{0.55, 0.58, 0.62}, // 第 3 名：青灰
	}

	for i, n := range m.Names {
		meaning := ""
		if n.Meaning != "" {
			meaning = "寓意：" + n.Meaning
		}
		meaningW := l.contentW - (l.colX[1] - l.left) - 4
		meaningLines := d.Wrap(meaning, szSmall, meaningW)
		scoreLine := ""
		if m.FullDetail && len(n.Scores) > 0 {
			scoreLine = formatNameScores(n.Scores)
		}
		scoreLines := 0
		if scoreLine != "" {
			scoreLines = len(d.Wrap(scoreLine, szSmall, meaningW))
		}
		rowH := 19.0 + float64(len(meaningLines))*13.0 + float64(scoreLines)*13.0 + 8
		if d.CursorY()+rowH > d.BottomLimit() {
			d.AddPage()
			d.Space(6)
			drawNameTableHeader(d, l) // 续页重画表头
		}
		rowTop := d.CursorY()

		// 不用整行底色：底色与文字基线存在视觉错位，长寓意行尤其明显。
		// 行间仅保留细分隔线，前三名另加左侧色条。
		if i < 3 {
			fillRect(d, l.left, rowTop-2, 3, rowH-6, rankColors[i])
		}

		setColor(d, clrMuted)
		d.Text(l.colX[0]+8, rowTop+2, szBody, fmt.Sprintf("%d", i+1))
		setColor(d, clrTitle)
		d.Text(l.colX[1], rowTop+2, szBody, n.FullName)
		setColor(d, clrText)
		d.Text(l.colX[2], rowTop+2, szBody, n.Pinyin)
		d.Text(l.colX[3], rowTop+2, szBody, n.WuXing)

		grade := gradeLabel(n.ScoreValue)
		if n.Score == "" {
			grade = "—"
		}
		setColor(d, clrMuted)
		d.Text(l.gradeX, rowTop+2, szBody, grade)

		if n.Score != "" {
			scoreColor := clrText
			switch {
			case n.ScoreValue >= 90:
				scoreColor = clrAccent
			case n.ScoreValue >= 85:
				scoreColor = clrTitle
			}
			setColor(d, scoreColor)
			d.Text(l.scoreRight-d.Measure(szBody, n.Score)-4, rowTop+2, szBody, n.Score)
		}
		d.Space(19)

		if meaning != "" {
			setColor(d, clrMuted)
			for _, ln := range meaningLines {
				d.Text(l.colX[1], d.CursorY(), szSmall, ln)
				d.Space(13)
			}
		}
		if scoreLine != "" {
			setColor(d, clrAccent)
			for _, ln := range d.Wrap(scoreLine, szSmall, meaningW) {
				d.Text(l.colX[1], d.CursorY(), szSmall-0.5, ln)
				d.Space(13)
			}
		}
		d.Space(8)
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

// getScoreFloat 取评分数值；字符串数字、整数与浮点统一转为 float64。
func getScoreFloat(data map[string]interface{}, key string) float64 {
	if v, ok := toFloat(data[key]); ok {
		return v
	}
	return 0
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
