package web

import (
	"bytes"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"name/internal/application/services"
	"name/internal/domain/bazi"
	"name/internal/domain/namestat"
	"name/internal/infrastructure/logger"
)

//go:embed templates/*.html
var templateFS embed.FS

// Handler 持有服务端渲染所需的服务引用与模板引擎
type Handler struct {
	name     services.NameServiceInterface
	history  services.HistoryServiceInterface
	favorite services.FavoriteServiceInterface

	// tmpls 每个页面独立一份模板集（layout + 该页 content），避免动态模板名
	tmpls map[string]*template.Template

	sessMu sync.Mutex
	// sessions 进程内会话表：sid → 本次生成结果（换一批时原地替换 Names）。
	// 单实例部署；多实例需外置会话存储。
	sessions map[string]*services.GenerateResponse
}

// pageData 模板页数据载体
type pageData struct {
	Title      string
	Error      string
	Result     *services.GenerateResponse
	Favorites  []*services.FavoriteRecord
	History    []historyView
	Page       int
	TotalPages int
	PrevPage   int
	NextPage   int
	// Evaluate 测名页结果（POST /evaluate 成功后渲染）
	Evaluate *services.EvaluateResponse
	// Risks 测名风险体检展示视图（Level 已转 string，见 riskView）
	Risks []riskView
	// Huangli 黄历页数据（bazi.GetHuangli 直读域层，与 /api/v1/huangli 同源）
	Huangli *bazi.Huangli
	// Stats 姓名统计 Top-N（namestat.GetTopNames，与 /api/v1/stat 同源）
	Stats []namestat.NameStat
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
		{"huangli", "templates/huangli.html"},
		{"stat", "templates/stat.html"},
	}
	tmpls := make(map[string]*template.Template, len(pages))
	for _, p := range pages {
		t, err := template.New("base").ParseFS(templateFS, "templates/layout.html", p.file)
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
		sessions: make(map[string]*services.GenerateResponse),
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
	router.GET("/huangli", h.HuangliPage)
	router.GET("/stat", h.StatPage)
}

// renderPage 用指定页面的模板集渲染（base 模板装配该页 content 块）。
// 先整体渲染到 buffer 再一次性写出：模板中途报错时不会留下半页 HTML，
// 也不会触发「Headers were already written」导致状态码改不动。
func (h *Handler) renderPage(c *gin.Context, status int, page string, data *pageData) {
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
		resp := h.sessions[sid]
		h.sessMu.Unlock()
		if resp != nil {
			return sid, resp
		}
	}
	sid := generateSID()
	c.SetCookie("nm_sid", sid, 3600, "/", "", false, true)
	return sid, nil
}

func generateSID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

// atoiQuery 解析分页参数（优先查询串，其次表单），缺省或非法时回退为 1
func atoiQuery(c *gin.Context, key string) int {
	v := c.Query(key)
	if v == "" {
		v = c.PostForm(key)
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return 1
	}
	return n
}

// Home 首页（生成表单）
func (h *Handler) Home(c *gin.Context) {
	h.renderPage(c, http.StatusOK, "home", &pageData{
		Title: "起名 NameMaster",
		Error: c.Query("error"),
	})
}

// Generate 接收表单 → 调用服务层 → 存会话 → 重定向结果页（PRG）
func (h *Handler) Generate(c *gin.Context) {
	req, err := parseGenerateForm(c)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/?error="+url.QueryEscape(err.Error()))
		return
	}
	resp, err := h.name.Generate(c.Request.Context(), req)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/?error="+url.QueryEscape(err.Error()))
		return
	}
	// Surname/FullName 由服务层回填（fate 引擎 NameResult 直接携带
	// Surname=input.Surname、FullName=Surname+GivenName，见
	// convertFateToNameNames / assembleEvaluatedName），此处无需再拼接。
	sid, _ := h.getSession(c)
	h.sessMu.Lock()
	h.sessions[sid] = resp
	h.sessMu.Unlock()
	h.saveHistory(c, req, resp)
	c.Redirect(http.StatusSeeOther, "/result")
}

// Result 结果展示页
func (h *Handler) Result(c *gin.Context) {
	_, resp := h.getSession(c)
	if resp == nil {
		c.Redirect(http.StatusSeeOther, "/?error="+url.QueryEscape("请先生成名字"))
		return
	}
	h.renderPage(c, http.StatusOK, "result", &pageData{
		Title:  "起名结果",
		Result: resp,
	})
}

// Explore 换一批：从同一候选表采样另一批名字，刷新会话
func (h *Handler) Explore(c *gin.Context) {
	genID := c.PostForm("generation_id")
	if genID == "" {
		c.Redirect(http.StatusSeeOther, "/result")
		return
	}
	er, err := h.name.ExploreNames(c.Request.Context(), &services.ExploreRequest{
		GenerationID: genID,
		Count:        10,
	})
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/?error="+url.QueryEscape(err.Error()))
		return
	}
	if len(er.Names) == 0 {
		c.Redirect(http.StatusSeeOther, "/result")
		return
	}
	sid, resp := h.getSession(c)
	h.sessMu.Lock()
	if resp != nil {
		resp.Names = er.Names
	} else {
		h.sessions[sid] = &services.GenerateResponse{Names: er.Names}
	}
	h.sessMu.Unlock()
	c.Redirect(http.StatusSeeOther, "/result")
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
		c.Redirect(http.StatusSeeOther, "/?error="+url.QueryEscape(err.Error()))
		return
	}
	c.Redirect(http.StatusSeeOther, "/result")
}

// Favorites 收藏列表页（分页）
func (h *Handler) Favorites(c *gin.Context) {
	page := atoiQuery(c, "page")
	limit := 20
	recs, total, err := h.favorite.GetFavoritesPage(c.Request.Context(), page, limit)
	if err != nil {
		h.renderPage(c, http.StatusInternalServerError, "favorites", &pageData{
			Title: "收藏",
			Error: err.Error(),
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
			c.Redirect(http.StatusSeeOther, "/favorites?page="+strconv.Itoa(page)+"&error="+url.QueryEscape(err.Error()))
			return
		}
	}
	c.Redirect(http.StatusSeeOther, "/favorites?page="+strconv.Itoa(page))
}

// History 历史记录列表页（分页）
func (h *Handler) History(c *gin.Context) {
	page := atoiQuery(c, "page")
	limit := 20
	recs, total, err := h.history.GetHistoryPage(c.Request.Context(), page, limit)
	if err != nil {
		h.renderPage(c, http.StatusInternalServerError, "history", &pageData{
			Title: "历史",
			Error: err.Error(),
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

// DeleteHistory 删除历史记录（PRG 重定向回列表）
func (h *Handler) DeleteHistory(c *gin.Context) {
	id := c.PostForm("id")
	page := atoiQuery(c, "page")
	if id != "" {
		if err := h.history.DeleteHistory(c.Request.Context(), id); err != nil {
			c.Redirect(http.StatusSeeOther, "/history?page="+strconv.Itoa(page)+"&error="+url.QueryEscape(err.Error()))
			return
		}
	}
	c.Redirect(http.StatusSeeOther, "/history?page="+strconv.Itoa(page))
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
		Gender:        c.PostForm("gender"),
		BirthYear:     atoi("birth_year"),
		BirthMonth:    atoi("birth_month"),
		BirthDay:      atoi("birth_day"),
		BirthHour:     atoi("birth_hour"),
		BirthMinute:   atoi("birth_minute"),
		BirthLocation: c.PostForm("birth_location"),
	}
	resp, err := h.name.Evaluate(c.Request.Context(), req)
	if err != nil {
		h.renderPage(c, http.StatusOK, "evaluate", &pageData{
			Title: "测名",
			Error: err.Error(),
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

// HuangliPage 黄历页：按年/月/日查询（缺省为今天），直读 bazi 域层，
// 与 /api/v1/huangli 同一真源。
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
	data := &pageData{Title: "黄历"}
	hl := bazi.GetHuangli(year, month, day)
	if hl == nil {
		data.Error = "该日期无法换算黄历，请检查年月日（支持 1900-2100）"
	} else {
		data.Huangli = hl
	}
	h.renderPage(c, http.StatusOK, "huangli", data)
}

// StatPage 姓名统计页：Top-N 高频名，直读 namestat 域层，与 /api/v1/stat 同源
func (h *Handler) StatPage(c *gin.Context) {
	limit := 20
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	h.renderPage(c, http.StatusOK, "stat", &pageData{
		Title: "姓名统计",
		Stats: namestat.GetTopNames(limit),
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
		if err := json.Unmarshal([]byte(r.Results), &resp); err == nil {
			v.NameCount = len(resp.Names)
			if len(resp.Names) > 0 {
				v.TopName = resp.Names[0].FullName
			}
		}
		out = append(out, v)
	}
	return out
}
func parseGenerateForm(c *gin.Context) (*services.GenerateRequest, error) {
	atoi := func(k string) int {
		v, _ := strconv.Atoi(c.PostForm(k))
		return v
	}
	req := &services.GenerateRequest{
		Surname:         strings.TrimSpace(c.PostForm("surname")),
		Gender:          c.PostForm("gender"),
		BirthYear:       atoi("birth_year"),
		BirthMonth:      atoi("birth_month"),
		BirthDay:        atoi("birth_day"),
		BirthHour:       atoi("birth_hour"),
		BirthMinute:     atoi("birth_minute"),
		BirthLocation:   c.PostForm("birth_location"),
		NameLength:      atoi("name_length"),
		ExcludeRare:     c.PostForm("exclude_rare") == "1",
		EnableCantonese: c.PostForm("enable_cantonese") == "1",
	}
	if req.Surname == "" {
		return nil, errors.New("请填写姓氏")
	}
	if req.Gender == "" {
		return nil, errors.New("请选择性别")
	}
	if req.BirthYear == 0 || req.BirthMonth == 0 || req.BirthDay == 0 {
		return nil, errors.New("请填写完整出生日期")
	}
	if req.NameLength == 0 {
		req.NameLength = 2
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
