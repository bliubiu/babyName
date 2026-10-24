package web

import (
	"bytes"
	"context"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	apperrors "name/internal/application/errors"
	"name/internal/application/services"
	"name/internal/application/validator"
	"name/internal/domain/bazi"
	"name/internal/domain/hanzi"
	"name/internal/domain/name"
	"name/internal/domain/namestat"
	"name/internal/infrastructure/logger"
)

//go:embed templates/*.html
var templateFS embed.FS

// jsonViewerHTML 离线单文件 JSON 美化查看器（/json-viewer 页的 iframe 内容）。
// 自 frontend/public/json4matter.html 复制嵌入（该文件原由 Next.js 构建复制到
// out/ 根目录并由旧静态服务放行）；SSR 全栈后不再依赖 frontend/ 目录，
// 内嵌进二进制保证单文件部署。改动此文件时同步 frontend/public 源（frontend/
// 删除后以本文件为准）。
//
//go:embed static/json4matter.html
var jsonViewerHTML []byte

// Handler 持有服务端渲染所需的服务引用与模板引擎
type Handler struct {
	name     services.NameServiceInterface
	history  services.HistoryServiceInterface
	favorite services.FavoriteServiceInterface

	// tmpls 每个页面独立一份模板集（layout + 该页 content），避免动态模板名
	tmpls map[string]*template.Template

	sessMu sync.Mutex
	// sessions 进程内会话表：sid → 本次生成结果（换一批时整体替换条目）。
	// 单实例部署；多实例需外置会话存储。
	//
	// 历史缺陷：séances map 永不过期也无容量上限，每次生成都会遗留一条含完整
	// 响应（Names 数组 + 八字/紫微/卦象）的条目，长期运行内存只增不减（OOM 面）。
	// 现加 TTL（1 小时，与 nm_sid cookie 一致）+ 写入时惰性清扫 + 容量上限。
	sessions map[string]*sessionEntry
}

// sessionEntry 会话条目：响应快照 + 过期时间。
// resp 写入后不再原地修改（Explore 换一批整体替换条目），
// 渲染 goroutine 在锁外持有引用是安全的。
type sessionEntry struct {
	resp      *services.GenerateResponse
	expiresAt time.Time
}

// sessionTTL 会话有效期，与 nm_sid cookie 的 MaxAge 一致（1 小时）。
const sessionTTL = time.Hour

// sessionMaxEntries 会话容量上限（单实例内存安全边界，超限先清过期再淘汰）。
const sessionMaxEntries = 2000

// nameFilter 结果页筛选条件（字段集与 React NameFilter 对齐）：
// 性别（male/female，引擎字面量）、五行、笔画区间、人名频率档位下限。
// 频率档位由 frequency_score 估算——与前端 result/page.tsx:91-96
// 同口径的简化映射（score 越高 tier 越高）。
type nameFilter struct {
	Gender     string
	Wuxing     string
	MinStrokes int
	MaxStrokes int
	MinTier    int
}

// parseNameFilter 从查询串解析筛选条件；无参数返回零值（不筛选）。
func parseNameFilter(c *gin.Context) nameFilter {
	atoi := func(k string) int {
		v, _ := strconv.Atoi(c.Query(k))
		if v < 0 {
			return 0
		}
		return v
	}
	return nameFilter{
		Gender:     c.Query("gender"),
		Wuxing:     c.Query("wuxing"),
		MinStrokes: atoi("min_strokes"),
		MaxStrokes: atoi("max_strokes"),
		MinTier:    atoi("min_tier"),
	}
}

// active 是否有任一筛选条件生效。
func (f nameFilter) active() bool {
	return f.Gender != "" || f.Wuxing != "" || f.MinStrokes > 0 || f.MaxStrokes > 0 || f.MinTier > 0
}

// query 序列化为查询串（供换一批/收藏表单以隐藏字段回带，保持筛选态）。
func (f nameFilter) query() string {
	if !f.active() {
		return ""
	}
	v := url.Values{}
	if f.Gender != "" {
		v.Set("gender", f.Gender)
	}
	if f.Wuxing != "" {
		v.Set("wuxing", f.Wuxing)
	}
	if f.MinStrokes > 0 {
		v.Set("min_strokes", strconv.Itoa(f.MinStrokes))
	}
	if f.MaxStrokes > 0 {
		v.Set("max_strokes", strconv.Itoa(f.MaxStrokes))
	}
	if f.MinTier > 0 {
		v.Set("min_tier", strconv.Itoa(f.MinTier))
	}
	return v.Encode()
}

// estimatedTier 由 frequency_score 估算频率档位（0-5），映射与前端一致。
func estimatedTier(score float64) int {
	switch {
	case score >= 90:
		return 5
	case score >= 70:
		return 4
	case score >= 55:
		return 3
	case score >= 35:
		return 2
	case score >= 15:
		return 1
	default:
		return 0
	}
}

// apply 按条件过滤候选名（纯函数，不改会话内原切片）。
func (f nameFilter) apply(names []name.Name) []name.Name {
	if !f.active() {
		return names
	}
	out := make([]name.Name, 0, len(names))
	for _, n := range names {
		if f.Gender != "" && n.Gender != f.Gender {
			continue
		}
		if f.Wuxing != "" && !strings.Contains(n.Wuxing, f.Wuxing) {
			continue
		}
		if f.MinStrokes > 0 && n.Strokes < f.MinStrokes {
			continue
		}
		if f.MaxStrokes > 0 && n.Strokes > f.MaxStrokes {
			continue
		}
		if f.MinTier > 0 && estimatedTier(n.FrequencyScore) < f.MinTier {
			continue
		}
		out = append(out, n)
	}
	return out
}

// pageData 模板页数据载体。
// Ok/Dark/RedirectPath 由 renderPage 统一注入（单次请求的公共视图状态），
// 各 handler 无需重复装配。
type pageData struct {
	Title        string
	Error        string
	Ok           string
	Dark         bool
	RedirectPath string
	// Path 当前请求路径（不含查询串），供导航当前页高亮
	Path       string
	Result     *services.GenerateResponse
	Favorites  []*services.FavoriteRecord
	History    []historyView
	Page       int
	TotalPages int
	PrevPage   int
	NextPage   int
	// CharGroups 偏旁选字分组（首页表单用，hanzi.GetNamerGroups 单一真源，
	// 与 /api/v1/characters/groups 同源；SSR 服务端渲染 checkbox 网格）
	CharGroups []hanzi.StandardCharGroup
	// Evaluate 测名页结果（POST /evaluate 成功后渲染）
	Evaluate *services.EvaluateResponse
	// Risks 测名风险体检展示视图（Level 已转 string，见 riskView）
	Risks []riskView
	// Huangli 黄历页数据（bazi.GetHuangli 直读域层，与 /api/v1/huangli 同源）
	Huangli *bazi.Huangli
	// MonthCal 万年历月格数据（bazi.GetLunarMonthCalendar，与 /api/v1 同源）
	MonthCal []bazi.LunarMonthDay
	// SelYear/SelMonth/SelDay 当前查询的日期（日视图=所查日，月视图=所查月）
	SelYear  int
	SelMonth int
	SelDay   int
	// MonthToday 今天（月格高亮用）
	MonthToday time.Time
	// MonthLeadingBlanks 月格首行前置空白格数（周一开头：(weekday+6)%7）
	MonthLeadingBlanks int
	// PrevDayURL/NextDayURL 黄历日期导航链接（服务端预计算，含月份进位）
	PrevDayURL string
	NextDayURL string
	// MonthViewURL 日视图→月视图入口；PrevMonthURL/NextMonthURL 月视图翻页
	MonthViewURL string
	PrevMonthURL string
	NextMonthURL string
	// Stats 姓名统计 Top-N（namestat.GetTopNames，与 /api/v1/stat 同源）
	Stats []namestat.NameStat
	// QueriedName 按名查重名率的查询词（空 = 未查询）
	QueriedName string
	// NameStat 查询结果（namestat.GetNameStats；Count=0 表示未收录）
	NameStat *namestat.NameStat
	// CompareNames 对比页的候选名字（POST /compare 从会话候选池按 given_name 取齐字段）
	CompareNames []name.Name
	// FromHistory 历史回看模式：结果页由 /history/view 渲染，隐藏换一批/对比
	// （候选池会话已不在），保留收藏。
	FromHistory bool
	// Report 完整报告页的综合解读（服务端基于同一批真实数据生成的五段文本）
	Report *reportAnalysis
	// Filter 结果页筛选条件（服务端过滤，替代 React 客户端 NameFilter）
	Filter nameFilter
	// FilterQuery 筛选条件序列化查询串（换一批/收藏表单回带用）
	FilterQuery string
	// TotalCount/FilteredCount 筛选前后的候选数（展示 filtered/total）
	TotalCount    int
	FilteredCount int
}

// riskView 风险项展示视图。
// fate.RiskLevel 是定义类型（type RiskLevel string），模板 eq 与字符串
// 字面量比较会因类型不匹配恒为 false，故在渲染层统一转成 string。
type riskView struct {
	Category string
	Level    string
	Detail   string
}

// historyView 历史记录在列表页的展示视图（含从 Results JSON 解出的摘要）
type historyView struct {
	ID        string
	Surname   string
	Gender    string
	BirthDate string
	BirthTime string
	BirthLoc  string
	CreatedAt string
	NameCount int
	TopName   string
}

// templateFuncs 模板函数表：把前端 poetryHighlight.ts 的命中字切分算法与
// 五行彩标逻辑移植到服务端，一次渲染零 JS。
var templateFuncs = template.FuncMap{
	"highlightPoetry": highlightPoetry,
	"wuxingTags":      wuxingTags,
	"genderLabel":     genderLabel,
	"makeSlice":       makeSlice,
	"isToday":         isToday,
	"max5":            max5,
	"pct":             pct,
	"hasPrefix":       strings.HasPrefix,
	"maxf":            maxf,
	"pctf":            pctf,
	"navClass":        navClass,
	"navAria":         navAria,
}

// max5 五行计数取最大值（分布条宽度归一化基准）；全 0 时返回 1 防除零。
func max5(a, b, c, d, e int) int {
	m := a
	for _, v := range []int{b, c, d, e} {
		if v > m {
			m = v
		}
	}
	if m <= 0 {
		return 1
	}
	return m
}

// pct 计数占基准的百分比（分布条宽度）。
func pct(v, max int) float64 {
	if max <= 0 {
		return 0
	}
	return float64(v) / float64(max) * 100
}

// maxf 浮点序列取最大值（占比条归一化基准）；全 0 时返回 1 防除零。
func maxf(vals ...float64) float64 {
	m := 0.0
	for _, v := range vals {
		if v > m {
			m = v
		}
	}
	if m <= 0 {
		return 1
	}
	return m
}

// pctf 浮点数占基准的百分比（统计页占比条宽度）。
func pctf(v, max float64) float64 {
	if max <= 0 {
		return 0
	}
	return v / max * 100
}

// navClass 导航当前页高亮样式：当前请求路径命中导航项（首页全等、其余前缀匹配）
// 时返回激活样式类。逻辑收在 Go 侧而非模板 {{if}}——html/template 对属性内的
// 分支条件与动态属性名有转义限制，纯字符串插值最稳。
func navClass(path, link string) string {
	if link == "/" {
		if path == "/" {
			return "nav-link active"
		}
		return "nav-link"
	}
	if strings.HasPrefix(path, link) {
		return "nav-link active"
	}
	return "nav-link"
}

// navAria 导航当前页的 aria-current 值（命中返回 "page"，否则空串）。
func navAria(path, link string) string {
	if strings.HasSuffix(navClass(path, link), " active") {
		return "page"
	}
	return ""
}

// genderLabel 引擎性别字面量（male/female）→ 中文标签。
// fate.Gender 是英文字面量，直接渲染会露出 "male"。
func genderLabel(g string) string {
	switch g {
	case "male":
		return "男"
	case "female":
		return "女"
	default:
		return g
	}
}

// makeSlice 生成 n 个空元素的切片，供月格前置空白格 {{range}} 占位
// （Go 模板无 range 整数语法）。
func makeSlice(n int) []struct{} {
	if n < 0 || n > 6 {
		return nil
	}
	return make([]struct{}, n)
}

// isToday 判断月格中的某天是否今天（高亮用）。
func isToday(today time.Time, year, month, day int) bool {
	return today.Year() == year && int(today.Month()) == month && today.Day() == day
}

// highlightPoetry 把诗词全文中命中的名字用字加粗高亮。
// 与前端 poetryHighlight.ts 同口径：逐 rune 判定是否在命中字表内
// （PoetryMatchedChars 已由引擎保证"实际出现在诗句/全诗中"），
// 命中包 <b>，其余原样转义输出；换行转 <br> 保留诗歌排版。
func highlightPoetry(text string, matched []string) template.HTML {
	if text == "" {
		return ""
	}
	esc := template.HTMLEscapeString
	if len(matched) == 0 {
		return template.HTML(strings.ReplaceAll(esc(text), "\n", "<br>"))
	}
	set := make(map[rune]bool, len(matched))
	for _, m := range matched {
		for _, r := range m {
			set[r] = true
		}
	}
	var b strings.Builder
	for _, r := range text {
		if r == '\n' {
			b.WriteString("<br>")
			continue
		}
		if set[r] {
			b.WriteString(`<b class="text-indigo-700 underline decoration-indigo-300">`)
			b.WriteString(esc(string(r)))
			b.WriteString("</b>")
			continue
		}
		b.WriteString(esc(string(r)))
	}
	return template.HTML(b.String())
}

// wuxingClass 五行 → 国风彩标类名（wx-jin/mu/shui/huo/tu，见布局样式表）。
// 与前端 NameCard 彩标同色系，但改用语义类以便暗色模式统一切换。
var wuxingClass = map[rune]string{
	'金': "wx-jin",
	'木': "wx-mu",
	'水': "wx-shui",
	'火': "wx-huo",
	'土': "wx-tu",
}

// wuxingTags 把 "木火" 这类五行串渲染成逐字国风彩标（每字一枚 pill）。
func wuxingTags(wuxing string) template.HTML {
	if wuxing == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range wuxing {
		cls, ok := wuxingClass[r]
		if !ok {
			cls = "wx-tu"
		}
		b.WriteString(`<span class="wuxing-pill `)
		b.WriteString(cls)
		b.WriteString(`">`)
		b.WriteString(template.HTMLEscapeString(string(r)))
		b.WriteString("</span>")
	}
	return template.HTML(b.String())
}

// NewHandler 构造 web 渲染层处理器
func NewHandler(nameSvc services.NameServiceInterface, historySvc services.HistoryServiceInterface, favSvc services.FavoriteServiceInterface) (*Handler, error) {
	pages := []struct {
		name string
		file string
	}{
		{"home", "templates/home.html"},
		{"result", "templates/result.html"},
		{"favorites", "templates/favorites.html"},
		{"history", "templates/history.html"},
		{"evaluate", "templates/evaluate.html"},
		{"compare", "templates/compare.html"},
		{"huangli", "templates/huangli.html"},
		{"stat", "templates/stat.html"},
		{"report", "templates/report.html"},
		{"json-viewer", "templates/json-viewer.html"},
	}
	tmpls := make(map[string]*template.Template, len(pages))
	for _, p := range pages {
		t, err := template.New("base").Funcs(templateFuncs).ParseFS(templateFS, "templates/layout.html", p.file)
		if err != nil {
			return nil, fmt.Errorf("解析模板 %s 失败: %w", p.name, err)
		}
		tmpls[p.name] = t
	}
	return &Handler{
		name:     nameSvc,
		history:  historySvc,
		favorite: favSvc,
		tmpls:    tmpls,
		sessions: make(map[string]*sessionEntry),
	}, nil
}

// RegisterRoutes 注册页面路由（在 /api/v1 之外的页面路径）
func RegisterRoutes(router *gin.Engine, h *Handler) {
	router.GET("/", h.Home)
	router.POST("/generate", h.Generate)
	router.GET("/result", h.Result)
	router.POST("/explore", h.Explore)
	router.POST("/favorites", h.Favorite)
	router.GET("/favorites", h.Favorites)
	router.POST("/favorites/delete", h.DeleteFavorite)
	router.GET("/history", h.History)
	router.POST("/history/delete", h.DeleteHistory)
	router.GET("/evaluate", h.EvaluatePage)
	router.POST("/evaluate", h.Evaluate)
	router.GET("/compare", h.ComparePage)
	router.POST("/compare", h.Compare)
	router.GET("/huangli", h.HuangliPage)
	router.GET("/stat", h.StatPage)
	router.GET("/json-viewer", h.JSONViewerPage)
	router.GET("/json4matter.html", h.JSONViewerAsset)
	router.POST("/theme", h.Theme)
	router.GET("/report", h.ReportPage)
	router.GET("/history/view", h.HistoryView)
}

// renderPage 用指定页面的模板集渲染（base 模板装配该页 content 块）。
// 先整体渲染到 buffer 再一次性写出：模板中途报错时不会留下半页 HTML，
// 也不会触发「Headers were already written」导致状态码改不动。
// 公共视图状态（flash 文案/暗色/主题回跳路径）在此统一注入。
func (h *Handler) renderPage(c *gin.Context, status int, page string, data *pageData) {
	if data != nil {
		data.Ok = c.Query("ok")
		data.Dark = isDark(c)
		data.RedirectPath = c.Request.URL.RequestURI()
		data.Path = c.Request.URL.Path
	}
	t, ok := h.tmpls[page]
	if !ok {
		c.String(http.StatusInternalServerError, "未知页面: "+page)
		return
	}
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "base", data); err != nil {
		logger.Error("模板渲染失败", logger.ErrField(err))
		c.String(http.StatusInternalServerError, "模板渲染失败")
		return
	}
	c.Data(status, "text/html; charset=utf-8", buf.Bytes())
}

// getSession 从 cookie 取回本次生成结果；无有效会话时发一个新 sid 并返回 nil resp。
// 新 sid 不预先落表——只有 Generate 真正写入结果时才登记，避免空会话堆积。
func (h *Handler) getSession(c *gin.Context) (string, *services.GenerateResponse) {
	if sid, err := c.Cookie("nm_sid"); err == nil {
		h.sessMu.Lock()
		entry := h.sessions[sid]
		if entry != nil && time.Now().Before(entry.expiresAt) {
			resp := entry.resp
			h.sessMu.Unlock()
			return sid, resp
		}
		h.sessMu.Unlock()
	}
	sid := generateSID()
	// SameSite=Lax：SSR 全部为同站表单 POST，Lax 足够且不破坏 PRG 跳转；
	// Secure 无法固定为 true（本地 HTTP 部署会直接丢 cookie）。
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("nm_sid", sid, int(sessionTTL.Seconds()), "/", "", false, true)
	return sid, nil
}

// setSession 登记一次生成结果；顺带惰性清扫过期会话并在超出容量上限时
// 淘汰过期/最旧条目，防止 map 无限增长（历史缺陷：永不过期）。
func (h *Handler) setSession(sid string, resp *services.GenerateResponse) {
	now := time.Now()
	h.sessMu.Lock()
	defer h.sessMu.Unlock()

	// 惰性清扫：到达容量上限或概率性触发，均摊开销极小
	if len(h.sessions) >= sessionMaxEntries || len(h.sessions)%64 == 0 {
		for k, e := range h.sessions {
			if now.After(e.expiresAt) {
				delete(h.sessions, k)
			}
		}
	}
	// 仍超上限：淘汰剩余最早过期的一条，保证 map 有界
	for len(h.sessions) >= sessionMaxEntries {
		var oldestKey string
		var oldest time.Time
		first := true
		for k, e := range h.sessions {
			if first || e.expiresAt.Before(oldest) {
				oldestKey, oldest, first = k, e.expiresAt, false
			}
		}
		delete(h.sessions, oldestKey)
	}
	h.sessions[sid] = &sessionEntry{resp: resp, expiresAt: now.Add(sessionTTL)}
}

func generateSID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

// clientMessage 把服务层错误转成可展示给浏览器用户的文案。
// 历史缺陷：SSR 各失败点把 err.Error() 原样塞进重定向参数渲染红条幅，
// AppError.Error() 会串联内层错误原文（含 SQLite 驱动字串、文件路径等
// 服务器内部细节），暴露给公开端访客。分级语义与 API 端
// handlers.writeGenerateError 对齐：
//   - 超时 → 可重试提示；
//   - AppError → 只透出业务 Message（不用 Error()，避免串联内层细节）；
//   - 其余未知错误 → 通用文案，细节仅记服务端日志。
func clientMessage(scope string, err error) string {
	logger.Error(scope+"失败", logger.ErrField(err))
	if stderrors.Is(err, context.DeadlineExceeded) {
		return "请求超时，请稍后重试"
	}
	var appErr *apperrors.AppError
	if stderrors.As(err, &appErr) {
		return appErr.Message
	}
	var vErr *validator.ValidationError
	if stderrors.As(err, &vErr) {
		return vErr.Message
	}
	return "操作失败，请稍后重试"
}

// atoiQuery 解析分页参数（优先查询串，其次表单），缺省或非法时回退为 1。
// 上限 1000：越界翻页会产生巨大 OFFSET 导致 SQLite 全表跳过扫描
// （历史缺陷：page=9999999 原样放行，收藏/历史变大后 IO 放大）。
func atoiQuery(c *gin.Context, key string) int {
	v := c.Query(key)
	if v == "" {
		v = c.PostForm(key)
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return 1
	}
	const maxPage = 1000
	if n > maxPage {
		return maxPage
	}
	return n
}

// isDark 读取暗色模式 cookie（theme=dark）。零 JS：切换由 POST /theme
// 表单提交写 cookie 后重定向回原页实现，服务端按 cookie 渲染对应配色。
func isDark(c *gin.Context) bool {
	v, err := c.Cookie("theme")
	return err == nil && v == "dark"
}

// Theme 切换暗色模式：POST 写 cookie 后重定向回原页。
// redirect 只接受站内路径（/ 开头），防开放重定向；缺省回首页。
func (h *Handler) Theme(c *gin.Context) {
	theme := c.PostForm("theme")
	if theme != "dark" {
		theme = "light"
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("theme", theme, 365*24*3600, "/", "", false, true)
	redirect := c.PostForm("redirect")
	if redirect == "" || !strings.HasPrefix(redirect, "/") || strings.HasPrefix(redirect, "//") {
		redirect = "/"
	}
	c.Redirect(http.StatusSeeOther, redirect)
}

// Home 首页（生成表单）
func (h *Handler) Home(c *gin.Context) {
	h.renderPage(c, http.StatusOK, "home", &pageData{
		Title:      "起名 NameMaster",
		Error:      c.Query("error"),
		CharGroups: hanzi.GetNamerGroups(),
	})
}

// Generate 接收表单 → 调用服务层 → 存会话 → 重定向结果页（PRG）
func (h *Handler) Generate(c *gin.Context) {
	req, err := parseGenerateForm(c)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/?error="+url.QueryEscape(clientMessage("生成名字", err)))
		return
	}
	resp, err := h.name.Generate(c.Request.Context(), req)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/?error="+url.QueryEscape(clientMessage("生成名字", err)))
		return
	}
	// Surname/FullName 由服务层回填（fate 引擎 NameResult 直接携带
	// Surname=input.Surname、FullName=Surname+GivenName，见
	// convertFateToNameNames / assembleEvaluatedName），此处无需再拼接。
	sid, _ := h.getSession(c)
	h.setSession(sid, resp)
	h.saveHistory(c, req, resp)
	c.Redirect(http.StatusSeeOther, "/result")
}

// Result 结果展示页（支持服务端筛选：gender/wuxing/min_strokes/max_strokes/min_tier）
func (h *Handler) Result(c *gin.Context) {
	_, resp := h.getSession(c)
	if resp == nil {
		c.Redirect(http.StatusSeeOther, "/?error="+url.QueryEscape("请先生成名字"))
		return
	}
	filter := parseNameFilter(c)
	// 浅拷贝 + 替换 Names 为筛选后视图：会话内原切片不动（换一批仍基于全量候选池），
	// 模板继续 {{range .Names}} 无需感知筛选。
	view := *resp
	view.Names = filter.apply(resp.Names)
	h.renderPage(c, http.StatusOK, "result", &pageData{
		Title:         "起名结果",
		Result:        &view,
		Filter:        filter,
		FilterQuery:   filter.query(),
		TotalCount:    len(resp.Names),
		FilteredCount: len(view.Names),
	})
}

// resultRedirect PRG 重定向回结果页；rq 为筛选查询串（隐藏字段回带），
// 保证换一批/收藏后筛选态不丢失。
func resultRedirect(c *gin.Context, rq string) {
	if rq != "" {
		c.Redirect(http.StatusSeeOther, "/result?"+rq)
		return
	}
	c.Redirect(http.StatusSeeOther, "/result")
}

// Explore 换一批：从同一候选表采样另一批名字，刷新会话
func (h *Handler) Explore(c *gin.Context) {
	rq := c.PostForm("rq")
	genID := c.PostForm("generation_id")
	if genID == "" {
		resultRedirect(c, rq)
		return
	}
	er, err := h.name.ExploreNames(c.Request.Context(), &services.ExploreRequest{
		GenerationID: genID,
		Count:        10,
	})
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/?error="+url.QueryEscape(clientMessage("换一批", err)))
		return
	}
	if len(er.Names) == 0 {
		resultRedirect(c, rq)
		return
	}
	sid, resp := h.getSession(c)
	if resp != nil {
		// 整体替换会话条目（浅拷贝 + 新 Names），不再原地改 resp.Names——
		// 原地替换与并发中的 Result 渲染存在 data race。
		// Bazi/Zodiac 等其余字段自写入后不变，浅拷贝安全。
		refresh := *resp
		refresh.Names = er.Names
		h.setSession(sid, &refresh)
	} else {
		h.setSession(sid, &services.GenerateResponse{Names: er.Names})
	}
	resultRedirect(c, rq)
}

// Favorite 收藏：复用后端收藏服务（含 score≥80 自学习副作用）
func (h *Handler) Favorite(c *gin.Context) {
	score, _ := strconv.ParseFloat(c.PostForm("score"), 64)
	rec := &services.FavoriteRecord{
		Surname:   c.PostForm("surname"),
		GivenName: c.PostForm("given_name"),
		Pinyin:    c.PostForm("pinyin"),
		Gender:    c.PostForm("gender"),
		Score:     score,
		Source:    c.PostForm("source"),
	}
	if _, err := h.favorite.SaveFavorite(c.Request.Context(), rec); err != nil {
		c.Redirect(http.StatusSeeOther, "/?error="+url.QueryEscape(clientMessage("收藏", err)))
		return
	}
	// rq 回带筛选查询串；ok 供结果页渲染绿色提示条
	loc := "/result?ok=" + url.QueryEscape("已加入收藏")
	if rq := c.PostForm("rq"); rq != "" {
		loc = "/result?" + rq + "&ok=" + url.QueryEscape("已加入收藏")
	}
	c.Redirect(http.StatusSeeOther, loc)
}

// Favorites 收藏列表页（分页）
func (h *Handler) Favorites(c *gin.Context) {
	page := atoiQuery(c, "page")
	limit := 20
	recs, total, err := h.favorite.GetFavoritesPage(c.Request.Context(), page, limit)
	if err != nil {
		h.renderPage(c, http.StatusInternalServerError, "favorites", &pageData{
			Title: "收藏",
			Error: clientMessage("获取收藏列表", err),
		})
		return
	}
	h.renderPage(c, http.StatusOK, "favorites", &pageData{
		Title:      "收藏",
		Error:      c.Query("error"),
		Favorites:  recs,
		Page:       page,
		TotalPages: (total + limit - 1) / limit,
		PrevPage:   page - 1,
		NextPage:   page + 1,
	})
}

// DeleteFavorite 取消收藏（PRG 重定向回列表）
func (h *Handler) DeleteFavorite(c *gin.Context) {
	id := c.PostForm("id")
	page := atoiQuery(c, "page")
	if id != "" {
		if err := h.favorite.DeleteFavorite(c.Request.Context(), id); err != nil {
			c.Redirect(http.StatusSeeOther, "/favorites?page="+strconv.Itoa(page)+"&error="+url.QueryEscape(clientMessage("取消收藏", err)))
			return
		}
	}
	c.Redirect(http.StatusSeeOther, "/favorites?page="+strconv.Itoa(page)+"&ok="+url.QueryEscape("已取消收藏"))
}

// History 历史记录列表页（分页）
func (h *Handler) History(c *gin.Context) {
	page := atoiQuery(c, "page")
	limit := 20
	recs, total, err := h.history.GetHistoryPage(c.Request.Context(), page, limit)
	if err != nil {
		h.renderPage(c, http.StatusInternalServerError, "history", &pageData{
			Title: "历史",
			Error: clientMessage("获取历史记录", err),
		})
		return
	}
	h.renderPage(c, http.StatusOK, "history", &pageData{
		Title:      "历史",
		Error:      c.Query("error"),
		History:    buildHistoryViews(recs),
		Page:       page,
		TotalPages: (total + limit - 1) / limit,
		PrevPage:   page - 1,
		NextPage:   page + 1,
	})
}

// DeleteHistory 删除历史记录（PRG 重定向回列表，带 flash 提示）
func (h *Handler) DeleteHistory(c *gin.Context) {
	id := c.PostForm("id")
	page := atoiQuery(c, "page")
	if id != "" {
		if err := h.history.DeleteHistory(c.Request.Context(), id); err != nil {
			c.Redirect(http.StatusSeeOther, "/history?page="+strconv.Itoa(page)+"&error="+url.QueryEscape(clientMessage("删除历史记录", err)))
			return
		}
	}
	c.Redirect(http.StatusSeeOther, "/history?page="+strconv.Itoa(page)+"&ok="+url.QueryEscape("已删除该条历史"))
}

// EvaluatePage 测名表单页（GET）
func (h *Handler) EvaluatePage(c *gin.Context) {
	h.renderPage(c, http.StatusOK, "evaluate", &pageData{
		Title: "测名",
		Error: c.Query("error"),
	})
}

// Evaluate 测名：POST 表单 → 服务层评分报告 → 渲染同页结果。
// 测名是幂等只读操作（无历史写入副作用），直接渲染同页而非 PRG 重定向。
func (h *Handler) Evaluate(c *gin.Context) {
	atoi := func(k string) int {
		v, _ := strconv.Atoi(c.PostForm(k))
		return v
	}
	req := &services.EvaluateRequest{
		Surname:       strings.TrimSpace(c.PostForm("surname")),
		GivenName:     strings.TrimSpace(c.PostForm("given_name")),
		Gender:        validator.NormalizeGender(c.PostForm("gender")),
		BirthYear:     atoi("birth_year"),
		BirthMonth:    atoi("birth_month"),
		BirthDay:      atoi("birth_day"),
		BirthHour:     atoi("birth_hour"),
		BirthMinute:   atoi("birth_minute"),
		BirthLocation: c.PostForm("birth_location"),
	}
	// 复用 API 同一套校验器（校验合流），错误文案分级后渲染红条幅
	if err := validateEvaluateForm(req); err != nil {
		h.renderPage(c, http.StatusOK, "evaluate", &pageData{
			Title: "测名",
			Error: clientMessage("测名", err),
		})
		return
	}
	resp, err := h.name.Evaluate(c.Request.Context(), req)
	// 服务返回错误或空结果都按失败渲染；nil resp 防御：服务实现缺陷
	// （返回 nil,nil）时不能 panic——HTTP 层兜住未知错误
	if err != nil || resp == nil {
		if err == nil {
			err = stderrors.New("测名服务返回空结果")
		}
		h.renderPage(c, http.StatusOK, "evaluate", &pageData{
			Title: "测名",
			Error: clientMessage("测名", err),
		})
		return
	}
	risks := make([]riskView, 0, len(resp.Risks))
	for _, r := range resp.Risks {
		risks = append(risks, riskView{
			Category: r.Category,
			Level:    string(r.Level),
			Detail:   r.Detail,
		})
	}
	h.renderPage(c, http.StatusOK, "evaluate", &pageData{
		Title:    "测名",
		Evaluate: resp,
		Risks:    risks,
	})
}

// ComparePage 对比页空态（直接访问 /compare 无勾选数据时给引导）
func (h *Handler) ComparePage(c *gin.Context) {
	h.renderPage(c, http.StatusOK, "compare", &pageData{
		Title: "名字对比",
		Error: c.Query("error"),
	})
}

// Compare 名字对比：POST 携带勾选的 given_name 列表（2-4 个），从会话候选池
// 按名取齐字段后渲染 11 指标对比表。
// 取代 React 的 zustand 内存态跨页携带（刷新/直入即丢）：勾选态在结果页
// 表单里、候选池在服务端会话里，任何入口行为一致。
func (h *Handler) Compare(c *gin.Context) {
	givens := c.PostFormArray("compare")
	if len(givens) < 2 {
		c.Redirect(http.StatusSeeOther, "/result?error="+url.QueryEscape("请先勾选 2-4 个名字再对比"))
		return
	}
	if len(givens) > 4 {
		c.Redirect(http.StatusSeeOther, "/result?error="+url.QueryEscape("最多同时对比 4 个名字"))
		return
	}
	_, resp := h.getSession(c)
	if resp == nil {
		c.Redirect(http.StatusSeeOther, "/?error="+url.QueryEscape("请先生成名字"))
		return
	}
	want := make(map[string]bool, len(givens))
	for _, g := range givens {
		want[g] = true
	}
	var picked []name.Name
	for _, n := range resp.Names {
		if want[n.GivenName] {
			picked = append(picked, n)
			delete(want, n.GivenName)
		}
	}
	if len(picked) < 2 {
		c.Redirect(http.StatusSeeOther, "/result?error="+url.QueryEscape("勾选的名字不在当前候选池中，请重新勾选"))
		return
	}
	h.renderPage(c, http.StatusOK, "compare", &pageData{
		Title:        "名字对比",
		CompareNames: picked,
	})
}

// HuangliPage 黄历页：按年/月/日查询（缺省为今天），直读 bazi 域层，
// 与 /api/v1/huangli 同一真源。view=month 时渲染万年历月格
// （bazi.GetLunarMonthCalendar，与 /api/v1/lunar/month 同源）。
func (h *Handler) HuangliPage(c *gin.Context) {
	now := time.Now()
	year, month, day := now.Year(), int(now.Month()), now.Day()
	if v := c.Query("year"); v != "" {
		year, _ = strconv.Atoi(v)
	}
	if v := c.Query("month"); v != "" {
		month, _ = strconv.Atoi(v)
	}
	if v := c.Query("day"); v != "" {
		day, _ = strconv.Atoi(v)
	}
	data := &pageData{Title: "黄历", SelYear: year, SelMonth: month, SelDay: day}
	dayURL := func(y int, m time.Month, d int) string {
		return fmt.Sprintf("/huangli?year=%d&month=%d&day=%d", y, int(m), d)
	}
	if c.Query("view") == "month" {
		// 万年历月格视图
		if month < 1 || month > 12 {
			data.Error = "月份需在 1-12 之间"
		} else {
			data.MonthCal = bazi.GetLunarMonthCalendar(year, month)
			data.MonthToday = now
			first := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
			data.MonthLeadingBlanks = (int(first.Weekday()) + 6) % 7 // 周一开头
			prevM := time.Date(year, time.Month(month)-1, 1, 0, 0, 0, 0, time.UTC)
			nextM := time.Date(year, time.Month(month)+1, 1, 0, 0, 0, 0, time.UTC)
			data.PrevMonthURL = fmt.Sprintf("/huangli?view=month&year=%d&month=%d", prevM.Year(), int(prevM.Month()))
			data.NextMonthURL = fmt.Sprintf("/huangli?view=month&year=%d&month=%d", nextM.Year(), int(nextM.Month()))
			data.MonthViewURL = fmt.Sprintf("/huangli?view=month&year=%d&month=%d", year, month)
		}
		h.renderPage(c, http.StatusOK, "huangli", data)
		return
	}
	hl := bazi.GetHuangli(year, month, day)
	if hl == nil {
		data.Error = "该日期无法换算黄历，请检查年月日（支持 1900-2100）"
	} else {
		data.Huangli = hl
	}
	// 日期导航：上一天/下一天（time.Date 归一化处理月份/年份进位，零 JS 链接）
	d := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	data.PrevDayURL = dayURL(d.AddDate(0, 0, -1).Year(), d.AddDate(0, 0, -1).Month(), d.AddDate(0, 0, -1).Day())
	data.NextDayURL = dayURL(d.AddDate(0, 0, 1).Year(), d.AddDate(0, 0, 1).Month(), d.AddDate(0, 0, 1).Day())
	data.MonthViewURL = fmt.Sprintf("/huangli?view=month&year=%d&month=%d", year, month)
	h.renderPage(c, http.StatusOK, "huangli", data)
}

// StatPage 姓名统计页：按名查重名率 + Top-N 高频名，直读 namestat 域层，与 /api/v1/stat 同源
func (h *Handler) StatPage(c *gin.Context) {
	limit := 20
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	data := &pageData{
		Title: "姓名统计",
		Stats: namestat.GetTopNames(limit),
	}
	// 按名查重名率：查询词非空时直查域层（未收录返回 Count=0，模板据此提示）
	if q := strings.TrimSpace(c.Query("name")); q != "" {
		data.QueriedName = q
		data.NameStat = namestat.GetNameStats(q)
	}
	h.renderPage(c, http.StatusOK, "stat", data)
}

// JSONViewerPage JSON 工具页：iframe 承载内嵌的 json4matter 查看器
// （与旧前端 /json-viewer 同源；样式隔离靠 iframe）
func (h *Handler) JSONViewerPage(c *gin.Context) {
	h.renderPage(c, http.StatusOK, "json-viewer", &pageData{
		Title: "JSON 查看器",
	})
}

// JSONViewerAsset 内嵌的离线 JSON 美化查看器单文件（/json-viewer 页 iframe 的src）
func (h *Handler) JSONViewerAsset(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", jsonViewerHTML)
}

// reportAnalysis 完整报告页的综合解读（五段文本）。
// 由前端 FullReport.tsx 的 generateDetailAnalysis 移植为服务端计算：
// SSR 零 JS，解读必须基于同一批真实数据在服务端生成，口径与旧前端一致。
type reportAnalysis struct {
	BaziPattern        string
	BalanceAnalysis    string
	XiyongshenAnalysis string
	ZodiacNayin        string
	NamesAnalysis      string
}

// wuxingCn 五行键 → 中文名（与前端 wuxingNameMap 同口径）。
var wuxingCn = map[string]string{
	"jin": "金", "mu": "木", "shui": "水", "huo": "火", "tu": "土",
}

// buildReportAnalysis 综合解读生成：八字格局/五行平衡/喜用神补益/生肖纳音/名字质量。
// 全部取自 resp 真实字段，不虚构数据；数据缺失时给出可操作的提示文案。
func buildReportAnalysis(resp *services.GenerateResponse) *reportAnalysis {
	a := &reportAnalysis{}
	if resp == nil {
		return a
	}
	// 1. 八字格局（Bazi 为值类型；四柱全空视为数据缺失，如损坏的历史行）
	b := &resp.Bazi
	if b.Bazi.YearGanzhi == "" && b.Bazi.DayGanzhi == "" {
		a.BaziPattern = "八字数据缺失，建议核对出生时间后重新生成。"
		a.BalanceAnalysis = "八字五行数据缺失，建议核对出生时间后重新生成。"
		a.XiyongshenAnalysis = "喜用神信息缺失，建议核对八字分析结果。"
	} else {
		rishou := b.Rishou
		if rishou == "" {
			rishou = b.DayMaster
		}
		if rishou == "" {
			rishou = "未知"
		}
		rishouWx := b.RishouWuxing
		if rishouWx == "" {
			rishouWx = "未知"
		}
		a.BaziPattern = fmt.Sprintf("日主为%s（%s性），日柱%s。八字四柱为 %s、%s、%s、%s。",
			rishou, rishouWx, b.Bazi.DayGanzhi, b.Bazi.YearGanzhi, b.Bazi.MonthGanzhi, b.Bazi.DayGanzhi, b.Bazi.HourGanzhi)

		// 2. 五行平衡（按数量降序找最旺/最弱）
		type wxEntry struct {
			name string
			cnt  int
		}
		entries := []wxEntry{
			{"金", b.Wuxing.Jin}, {"木", b.Wuxing.Mu}, {"水", b.Wuxing.Shui}, {"火", b.Wuxing.Huo}, {"土", b.Wuxing.Tu},
		}
		total := 0
		for _, e := range entries {
			total += e.cnt
		}
		sorted := append([]wxEntry(nil), entries...)
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].cnt > sorted[j].cnt })
		strongest := sorted[0]
		weakest := sorted[len(sorted)-1]
		for _, e := range sorted {
			if e.cnt < strongest.cnt {
				weakest = e
				break
			}
		}
		detail := fmt.Sprintf("金%d、木%d、水%d、火%d、土%d", b.Wuxing.Jin, b.Wuxing.Mu, b.Wuxing.Shui, b.Wuxing.Huo, b.Wuxing.Tu)
		if total == 0 {
			a.BalanceAnalysis = "八字五行数据缺失，建议核对出生时间后重新生成。"
		} else {
			maxRatio := float64(strongest.cnt) / float64(total) * 100
			minRatio := float64(weakest.cnt) / float64(total) * 100
			switch {
			case maxRatio-minRatio > 50:
				a.BalanceAnalysis = fmt.Sprintf("五行分布为%s（共%d个），其中%s最旺（%d个，占%.0f%%），%s最弱（%d个，占%.0f%%），整体偏枯，起名时需重点补益%s。",
					detail, total, strongest.name, strongest.cnt, maxRatio, weakest.name, weakest.cnt, minRatio, weakest.name)
			case maxRatio-minRatio > 25:
				a.BalanceAnalysis = fmt.Sprintf("五行分布为%s（共%d个），%s较旺（%d个），%s偏弱（%d个），略有失衡，起名可适度补益%s。",
					detail, total, strongest.name, strongest.cnt, weakest.name, weakest.cnt, weakest.name)
			default:
				a.BalanceAnalysis = fmt.Sprintf("五行分布为%s（共%d个），整体较为均衡，无明显偏枯。", detail, total)
			}
		}

		// 3. 喜用神补益
		if len(b.Xiyongshen) > 0 {
			xy := strings.Join(b.Xiyongshen, "、")
			if total > 0 && containsStr(b.Xiyongshen, weakest.name) {
				a.XiyongshenAnalysis = fmt.Sprintf("喜用神为%s，与最弱的%s五行一致，起名时优先选用五行属%s的字，可直接补益命局。", xy, weakest.name, xy)
			} else {
				a.XiyongshenAnalysis = fmt.Sprintf("喜用神为%s，起名时建议选用五行属%s或与%s相生关系的字，以补益命局不足。", xy, xy, xy)
			}
		} else {
			a.XiyongshenAnalysis = "喜用神信息缺失，建议核对八字分析结果。"
		}
	}

	// 4. 生肖与纳音
	a.ZodiacNayin = fmt.Sprintf("生肖属%s，纳音为%s。", orUnknown(resp.Zodiac), orUnknown(resp.Nayin))

	// 5. 推荐名字质量（基于真实评分）
	if len(resp.Names) == 0 {
		a.NamesAnalysis = "本次未生成推荐名字，建议调整筛选条件后重试。"
	} else {
		var sum, maxS, minS float64
		topName := ""
		for i, n := range resp.Names {
			s := n.TotalScore
			sum += s
			if i == 0 || s > maxS {
				maxS = s
				topName = n.FullName
			}
			if i == 0 || s < minS {
				minS = s
			}
		}
		avg := sum / float64(len(resp.Names))
		matched, poetry := 0, 0
		xySet := map[string]bool{}
		for _, x := range resp.Bazi.Xiyongshen {
			xySet[x] = true
		}
		for _, n := range resp.Names {
			if n.PoetrySource != "" {
				poetry++
			}
			for _, r := range n.Wuxing {
				if xySet[string(r)] {
					matched++
					break
				}
			}
		}
		a.NamesAnalysis = fmt.Sprintf("共推荐%d个名字，综合评分区间%.1f-%.1f分，平均%.1f分。其中%s评分最高（%.1f分），%d个名字的五行与喜用神直接匹配，%d个名字出自诗词典故。",
			len(resp.Names), minS, maxS, avg, topName, maxS, matched, poetry)
	}
	return a
}

func orUnknown(s string) string {
	if s == "" {
		return "未知"
	}
	return s
}

func containsStr(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// HistoryView 历史回看：按 ID 取历史记录，解析 Results JSON 渲染结果页。
// 取代 React「点历史卡片→store 携带→跳 /result」的客户端态方案：
// 回看是服务端直查，刷新/分享链接行为一致。
func (h *Handler) HistoryView(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.Redirect(http.StatusSeeOther, "/history")
		return
	}
	rec, err := h.history.GetHistoryRecord(c.Request.Context(), id)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/history?error="+url.QueryEscape(clientMessage("查看历史记录", err)))
		return
	}
	var resp services.GenerateResponse
	if err := json.Unmarshal([]byte(rec.Results), &resp); err != nil {
		c.Redirect(http.StatusSeeOther, "/history?error="+url.QueryEscape("该条历史数据已损坏，无法回看"))
		return
	}
	h.renderPage(c, http.StatusOK, "result", &pageData{
		Title:       "起名结果（历史回看）",
		Result:      &resp,
		FromHistory: true,
	})
}

// ReportPage 完整报告页：八字分析 + 全量名字 + 综合解读。
// 数据来自当前会话的生成结果（与结果页同源）；无会话时引导回首页。
func (h *Handler) ReportPage(c *gin.Context) {
	_, resp := h.getSession(c)
	if resp == nil {
		c.Redirect(http.StatusSeeOther, "/?error="+url.QueryEscape("请先生成名字"))
		return
	}
	h.renderPage(c, http.StatusOK, "report", &pageData{
		Title:  "完整报告",
		Result: resp,
		Report: buildReportAnalysis(resp),
	})
}

// buildHistoryViews 把原始历史记录转成列表展示视图，并从 Results JSON 解出摘要
func buildHistoryViews(recs []*services.HistoryRecord) []historyView {
	out := make([]historyView, 0, len(recs))
	for _, r := range recs {
		v := historyView{
			ID:        r.ID,
			Surname:   r.Surname,
			Gender:    r.Gender,
			BirthDate: r.BirthDate,
			BirthTime: r.BirthTime,
			BirthLoc:  r.BirthLocation,
			CreatedAt: r.CreatedAt.Format("2006-01-02 15:04"),
		}
		var resp services.GenerateResponse
		if err := json.Unmarshal([]byte(r.Results), &resp); err != nil {
			// 语义损坏的历史行只记日志：列表页显示"共 0 个名字"，
			// 同时留下排查线索（历史缺陷：错误被静默吞掉）
			logger.Warn("历史记录 Results JSON 解析失败", logger.ErrField(err), logger.String("history_id", r.ID))
		} else {
			v.NameCount = len(resp.Names)
			if len(resp.Names) > 0 {
				v.TopName = resp.Names[0].FullName
			}
		}
		out = append(out, v)
	}
	return out
}

// validateEvaluateForm 测名表单校验：与 API 端 NameHandler.Evaluate 同一套
// 校验器与顺序（surname → given_name → gender → date → time），保证 SSR 与
// JSON 两条链路校验口径合流，不各写一套。given_name 在此只做字符级校验
// （非空 / ≤8 汉字），更具体的「1-2 汉字」由服务层 Evaluate 负责，不重复。
func validateEvaluateForm(req *services.EvaluateRequest) error {
	if err := validator.ValidateSurname(req.Surname); err != nil {
		return err
	}
	if err := validator.ValidateGivenName(req.GivenName); err != nil {
		return err
	}
	if err := validator.ValidateGender(req.Gender); err != nil {
		return err
	}
	if err := validator.ValidateDate(req.BirthYear, req.BirthMonth, req.BirthDay); err != nil {
		return err
	}
	return validator.ValidateTime(req.BirthHour, req.BirthMinute)
}

// parseGenerateForm 接收 /generate 表单
func parseGenerateForm(c *gin.Context) (*services.GenerateRequest, error) {
	atoi := func(k string) int {
		v, _ := strconv.Atoi(c.PostForm(k))
		return v
	}
	req := &services.GenerateRequest{
		Surname:            strings.TrimSpace(c.PostForm("surname")),
		Generation:         strings.TrimSpace(c.PostForm("generation")),
		GenerationPosition: c.PostForm("generation_position"),
		Gender:             validator.NormalizeGender(c.PostForm("gender")),
		BirthYear:          atoi("birth_year"),
		BirthMonth:         atoi("birth_month"),
		BirthDay:           atoi("birth_day"),
		BirthHour:          atoi("birth_hour"),
		BirthMinute:        atoi("birth_minute"),
		BirthLocation:      c.PostForm("birth_location"),
		NameLength:         atoi("name_length"),
		ExcludeRare:        c.PostForm("exclude_rare") == "1",
		SourceClassic:      c.PostForm("source_classic"),
		EnableCantonese:    c.PostForm("enable_cantonese") == "1",
	}
	// 寓意关键词：与 React 版同口径——逗号/空格切分（lib/api.ts 组装逻辑）
	for _, kw := range strings.FieldsFunc(c.PostForm("meaning_keywords"), func(r rune) bool {
		return r == ',' || r == '，' || r == ' '
	}) {
		if kw = strings.TrimSpace(kw); kw != "" {
			req.MeaningKeywords = append(req.MeaningKeywords, kw)
		}
	}
	// 偏旁选字：checkbox 同名多值（structured 通道，不混入 meaning_keywords）
	if chars := c.PostFormArray("selected_chars"); len(chars) > 0 {
		req.SelectedChars = append(req.SelectedChars, chars...)
	}
	if req.Surname == "" {
		return nil, stderrors.New("请填写姓氏")
	}
	if req.Gender == "" {
		return nil, stderrors.New("请选择性别")
	}
	if req.BirthYear == 0 || req.BirthMonth == 0 || req.BirthDay == 0 {
		return nil, stderrors.New("请填写完整出生日期")
	}
	if req.NameLength == 0 {
		req.NameLength = 2
	}

	// 复用 API 同一套校验器（校验合流）：年出生日期/时刻/姓氏非法时，
	// 与 JSON API 返回同款中文错误文案，避免任意数值直入库与命理计算。
	if err := validator.ValidateSurname(req.Surname); err != nil {
		return nil, err
	}
	if err := validator.ValidateGender(req.Gender); err != nil {
		return nil, err
	}
	if err := validator.ValidateDate(req.BirthYear, req.BirthMonth, req.BirthDay); err != nil {
		return nil, err
	}
	if err := validator.ValidateTime(req.BirthHour, req.BirthMinute); err != nil {
		return nil, err
	}

	if v := strings.TrimSpace(c.PostForm("avoid_elder_names")); v != "" {
		for _, item := range strings.Split(v, ",") {
			if t := strings.TrimSpace(item); t != "" {
				req.AvoidElderNames = append(req.AvoidElderNames, t)
			}
		}
	}
	return req, nil
}

// saveHistory 将本次生成结果自动写入历史（后端持久化）
func (h *Handler) saveHistory(c *gin.Context, req *services.GenerateRequest, resp *services.GenerateResponse) {
	if h.history == nil {
		return
	}
	data, err := json.Marshal(resp)
	if err != nil {
		return
	}
	rec := &services.HistoryRecord{
		Surname:       req.Surname,
		Gender:        req.Gender,
		BirthDate:     fmt.Sprintf("%04d-%02d-%02d", req.BirthYear, req.BirthMonth, req.BirthDay),
		BirthTime:     fmt.Sprintf("%02d:%02d", req.BirthHour, req.BirthMinute),
		BirthLocation: req.BirthLocation,
		Results:       data,
	}
	if _, err := h.history.SaveHistory(c.Request.Context(), rec); err != nil {
		logger.Warn("自动保存历史失败", logger.ErrField(err))
	}
}
