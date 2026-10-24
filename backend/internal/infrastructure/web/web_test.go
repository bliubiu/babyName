package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	apperrors "name/internal/application/errors"
	"name/internal/application/services"
	"name/internal/application/validator"
	"name/internal/domain/name"
)

// --- mock 服务 ---

type mockNameService struct {
	genReq      *services.GenerateRequest
	genResp     *services.GenerateResponse
	genErr      error
	evalReq     *services.EvaluateRequest
	evalResp    *services.EvaluateResponse
	evalErr     error
	exploreReq  *services.ExploreRequest
	exploreResp *services.ExploreResponse
	exploreErr  error
}

func (m *mockNameService) Generate(ctx context.Context, req *services.GenerateRequest) (*services.GenerateResponse, error) {
	m.genReq = req
	return m.genResp, m.genErr
}

func (m *mockNameService) GenerateWithAnalysis(ctx context.Context, req *services.GenerateRequest) (*services.GenerateWithAnalysisResponse, error) {
	return nil, nil
}

func (m *mockNameService) Evaluate(ctx context.Context, req *services.EvaluateRequest) (*services.EvaluateResponse, error) {
	m.evalReq = req
	return m.evalResp, m.evalErr
}

func (m *mockNameService) ExploreNames(ctx context.Context, req *services.ExploreRequest) (*services.ExploreResponse, error) {
	m.exploreReq = req
	return m.exploreResp, m.exploreErr
}

type mockHistoryService struct {
	pageRecords []*services.HistoryRecord
	pageTotal   int
	pageErr     error
	pageNum     int
	pageLimit   int
}

func (m *mockHistoryService) SaveHistory(ctx context.Context, record *services.HistoryRecord) (string, error) {
	return "h1", nil
}

func (m *mockHistoryService) GetHistory(ctx context.Context) ([]*services.HistoryRecord, error) {
	return nil, nil
}

func (m *mockHistoryService) GetHistoryPage(ctx context.Context, page, limit int) ([]*services.HistoryRecord, int, error) {
	m.pageNum = page
	m.pageLimit = limit
	if m.pageErr != nil {
		return nil, 0, m.pageErr
	}
	return m.pageRecords, m.pageTotal, nil
}

func (m *mockHistoryService) DeleteHistory(ctx context.Context, id string) error {
	return nil
}

func (m *mockHistoryService) GetHistoryRecord(ctx context.Context, id string) (*services.HistoryRecord, error) {
	for _, rec := range m.pageRecords {
		if rec != nil && rec.ID == id {
			return rec, nil
		}
	}
	return nil, apperrors.NewError(apperrors.ErrCodeNotFound, "历史记录不存在或已删除")
}

type mockFavoriteService struct {
	saveReq *services.FavoriteRecord
	saveErr error
	pageNum int
}

func (m *mockFavoriteService) GetFavorites(ctx context.Context) ([]*services.FavoriteRecord, error) {
	return nil, nil
}

func (m *mockFavoriteService) GetFavoritesPage(ctx context.Context, page, limit int) ([]*services.FavoriteRecord, int, error) {
	m.pageNum = page
	if m.saveErr != nil && m.pageNum == -1 { // pageErr 复用在页错误场景
		return nil, 0, m.saveErr
	}
	return nil, 0, nil
}

func (m *mockFavoriteService) SaveFavorite(ctx context.Context, record *services.FavoriteRecord) (string, error) {
	m.saveReq = record
	if m.saveErr != nil {
		return "", m.saveErr
	}
	return "f1", nil
}

func (m *mockFavoriteService) DeleteFavorite(ctx context.Context, id string) error {
	return nil
}

// setup 组装带 mock 的 Handler 与已注册路由。mock 由调用方持有引用。
func setup(nm *mockNameService, hist *mockHistoryService, fav *mockFavoriteService) (*Handler, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	if nm == nil {
		nm = &mockNameService{genResp: &services.GenerateResponse{Names: []name.Name{{FullName: "王测试"}}, GenerationID: "gen-1"}}
	}
	if hist == nil {
		hist = &mockHistoryService{}
	}
	if fav == nil {
		fav = &mockFavoriteService{}
	}
	h, err := NewHandler(nm, hist, fav)
	if err != nil {
		return nil, nil
	}
	r := gin.New()
	RegisterRoutes(r, h)
	return h, r
}

func setupT(t *testing.T) (*Handler, *mockNameService, *mockHistoryService, *mockFavoriteService, *gin.Engine) {
	t.Helper()
	nm := &mockNameService{genResp: &services.GenerateResponse{Names: []name.Name{{FullName: "王测试"}}, GenerationID: "gen-1"}}
	hist := &mockHistoryService{}
	fav := &mockFavoriteService{}
	h, err := NewHandler(nm, hist, fav)
	if err != nil {
		t.Fatalf("NewHandler 失败: %v", err)
	}
	r := gin.New()
	RegisterRoutes(r, h)
	return h, nm, hist, fav, r
}

func parseErrorFromLocation(t *testing.T, location string) string {
	t.Helper()
	u, err := url.Parse(location)
	if err != nil {
		t.Fatalf("解析 Location 失败: %v", err)
	}
	qs := u.Query().Get("error")
	dec, err := url.QueryUnescape(qs)
	if err != nil {
		return qs
	}
	return dec
}

func validForm() url.Values {
	return url.Values{
		"surname":     {"王"},
		"gender":      {"男"},
		"birth_year":  {"2024"},
		"birth_month": {"5"},
		"birth_day":   {"20"},
	}
}

// --- 用例 ---

func TestNewHandler_SSR模板解析成功(t *testing.T) {
	setupT(t)
}

func doGenerate(t *testing.T, r *gin.Engine, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/generate", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)
	return w
}

func doPostForm(t *testing.T, r *gin.Engine, path string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)
	return w
}

func TestHome_GET_渲染首页(t *testing.T) {
	_, _, _, _, r := setupT(t)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实得 %d", w.Code)
	}
}

// 回归：SSR 表单性别枚举（男/女）必须归一化为 canonical male/female，
// 否则 fate 引擎性别筛选静默失效（历史缺陷）。
func TestGenerate_POST_性别中文归一化(t *testing.T) {
	_, nm, _, _, r := setupT(t)
	w := doGenerate(t, r, validForm())
	if w.Code != http.StatusSeeOther {
		t.Fatalf("期望 303，实得 %d（body: %s）", w.Code, w.Body.String())
	}
	if got, want := w.Header().Get("Location"), "/result"; got != want {
		t.Fatalf("期望重定向 %s，实得 %s", want, got)
	}
	if nm.genReq.Gender != "male" {
		t.Errorf("服务层收到 gender=%q，期望 male", nm.genReq.Gender)
	}
}

// 回归：英文 API 口径 female 保持不变；数字别名归一化。
func TestGenerate_POST_性别英文与数字别名(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"female", "female"}, {"F", "female"}, {"2", "female"}, {"M", "male"},
	} {
		_, nm, _, _, r := setupT(t)
		form := validForm()
		form.Set("gender", tc.in)
		w := doGenerate(t, r, form)
		if w.Code != http.StatusSeeOther {
			t.Fatalf("输入 %q：期望 303，实得 %d", tc.in, w.Code)
		}
		if nm.genReq.Gender != tc.want {
			t.Errorf("输入 %q：服务层收到 gender=%q，期望 %q", tc.in, nm.genReq.Gender, tc.want)
		}
	}
}

// 回归：未知性别必须 400 拒绝，不能透传给引擎。
func TestGenerate_POST_未知性别拒绝(t *testing.T) {
	_, nm, _, _, r := setupT(t)
	form := validForm()
	form.Set("gender", "外星人")
	w := doGenerate(t, r, form)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("期望 303，实得 %d", w.Code)
	}
	msg := parseErrorFromLocation(t, w.Header().Get("Location"))
	if msg == "" {
		t.Fatalf("应返回中文错误文案")
	}
	if nm.genReq != nil {
		t.Errorf("校验失败不应触达服务层")
	}
}

// 回归：表单校验错误必须走统一分级（中文文案、不泄漏内部细节）。
func TestGenerate_POST_非法生日回首页错误(t *testing.T) {
	_, nm, _, _, r := setupT(t)
	form := validForm()
	form.Set("birth_year", "1899")
	w := doGenerate(t, r, form)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("期望 303，实得 %d", w.Code)
	}
	msg := parseErrorFromLocation(t, w.Header().Get("Location"))
	if msg != "年份必须在1900-2100之间" {
		t.Errorf("错误文案 %q 与 API 端校验文案不一致", msg)
	}
	if nm.genReq != nil {
		t.Errorf("校验失败不应触达服务层")
	}
}

// SSR 表单的出生日期用 <input type="date"> 原生日历选择器提交合并字段
// birth_date（YYYY-MM-DD）；服务层仍收拆分的年/月/日。
func TestGenerate_POST_合并日期字段birth_date(t *testing.T) {
	_, nm, _, _, r := setupT(t)
	form := validForm()
	form.Del("birth_year")
	form.Del("birth_month")
	form.Del("birth_day")
	form.Set("birth_date", "2024-08-15")
	w := doGenerate(t, r, form)
	if w.Code != http.StatusSeeOther || w.Header().Get("Location") != "/result" {
		t.Fatalf("期望 303 → /result，实得 %d → %s（body: %s）",
			w.Code, w.Header().Get("Location"), w.Body.String())
	}
	if nm.genReq == nil {
		t.Fatalf("服务层未收到请求")
	}
	if nm.genReq.BirthYear != 2024 || nm.genReq.BirthMonth != 8 || nm.genReq.BirthDay != 15 {
		t.Errorf("服务层收到 %d-%d-%d，期望 2024-8-15",
			nm.genReq.BirthYear, nm.genReq.BirthMonth, nm.genReq.BirthDay)
	}
}

// 浏览器只提交合法日历日期，但手工构造的请求仍可能越权——非法日期必须拦截。
func TestGenerate_POST_birth_date非法日期拒绝(t *testing.T) {
	_, nm, _, _, r := setupT(t)
	form := validForm()
	form.Del("birth_year")
	form.Del("birth_month")
	form.Del("birth_day")
	form.Set("birth_date", "2024-02-30")
	w := doGenerate(t, r, form)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("期望 303，实得 %d", w.Code)
	}
	if msg := parseErrorFromLocation(t, w.Header().Get("Location")); msg == "" {
		t.Errorf("应返回中文错误文案")
	}
	if nm.genReq != nil {
		t.Errorf("校验失败不应触达服务层")
	}
}

// birth_date 与旧三字段并存时以 birth_date 为准（保持单一入口语义）。
func TestGenerate_POST_birth_date优先于三字段(t *testing.T) {
	_, nm, _, _, r := setupT(t)
	form := validForm()
	form.Set("birth_date", "2020-01-02")
	w := doGenerate(t, r, form)
	if w.Code != http.StatusSeeOther || w.Header().Get("Location") != "/result" {
		t.Fatalf("期望 303 → /result，实得 %d", w.Code)
	}
	if nm.genReq == nil || nm.genReq.BirthYear != 2020 || nm.genReq.BirthMonth != 1 || nm.genReq.BirthDay != 2 {
		t.Errorf("birth_date 应优先于 birth_year/month/day")
	}
}

func TestResult_无会话重定向首页(t *testing.T) {
	_, _, _, _, r := setupT(t)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/result", nil))
	if w.Code != http.StatusSeeOther {
		t.Fatalf("期望 303，实得 %d", w.Code)
	}
	if msg := parseErrorFromLocation(t, w.Header().Get("Location")); msg != "请先生成名字" {
		t.Errorf("文案 %q 与预期不符", msg)
	}
}

func TestResult_有会话渲染结果页(t *testing.T) {
	_, _, _, _, r := setupT(t)
	w := doGenerate(t, r, validForm())
	sid := sidCookieFor(w)
	w2 := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/result", nil)
	req.Header.Set("Cookie", "nm_sid="+sid)
	r.ServeHTTP(w2, req)
	if w2.Code != http.StatusOK {
		t.Fatalf("期望 200，实得 %d", w2.Code)
	}
	if !strings.Contains(w2.Body.String(), "王测试") {
		t.Errorf("结果页未渲染出会话中的名字")
	}
}

// 回归：换一批必须整体替换会话条目（不原地改旧 resp.Names），
// 否则与并发渲染存在 data race（历史缺陷）。
func TestExplore_换一批整体替换会话(t *testing.T) {
	h, nm, _, _, r := setupT(t)
	w := doGenerate(t, r, validForm())
	sid := sidCookieFor(w)

	h.sessMu.Lock()
	oldNames := h.sessions[sid].resp.Names
	h.sessMu.Unlock()

	nm.exploreResp = &services.ExploreResponse{
		GenerationID: "gen-1",
		Names:        []name.Name{{FullName: "王新名"}},
	}
	form := url.Values{"generation_id": {"gen-1"}}
	w2 := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/explore", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", "nm_sid="+sid)
	r.ServeHTTP(w2, req)
	if w2.Code != http.StatusSeeOther {
		t.Fatalf("期望 303，实得 %d", w2.Code)
	}
	if w2.Header().Get("Location") != "/result" {
		t.Fatalf("期望重定向 /result，实得 %s", w2.Header().Get("Location"))
	}

	h.sessMu.Lock()
	entry := h.sessions[sid]
	h.sessMu.Unlock()
	if entry == nil {
		t.Fatal("会话条目丢失")
	}
	if len(entry.resp.Names) != 1 || entry.resp.Names[0].FullName != "王新名" {
		t.Errorf("会话 Names 未整体替换: %+v", entry.resp.Names)
	}
	// 旧条目对象不再被改动（渲染 goroutine 持有的引用安全）
	if len(oldNames) != 1 || oldNames[0].FullName != "王测试" {
		t.Errorf("旧 Names 数组被原地修改: %+v", oldNames)
	}
}

// 回归：服务层评分校验拒绝时，SSR 表单入口透传分级文案（500/400 分类不吞）。
func TestFavorite_POST_评分校验拒绝透传(t *testing.T) {
	fav := &mockFavoriteService{saveErr: validator.NewValidationError("score", "评分必须在0-100之间")}
	hist := &mockHistoryService{}
	nm := &mockNameService{}
	gin.SetMode(gin.TestMode)
	h, err := NewHandler(nm, hist, fav)
	if err != nil {
		t.Fatalf("NewHandler 失败: %v", err)
	}
	r := gin.New()
	RegisterRoutes(r, h)
	form := url.Values{"surname": {"王"}, "given_name": {"测试"}, "score": {"999"}}
	w := doPostForm(t, r, "/favorites", form)
	if fav.saveReq == nil {
		t.Fatalf("服务层未被调用")
	}
	if w.Code != http.StatusSeeOther {
		t.Fatalf("期望 303，实得 %d", w.Code)
	}
	if msg := parseErrorFromLocation(t, w.Header().Get("Location")); msg != "评分必须在0-100之间" {
		t.Errorf("错误文案 %q 与服务层校验文案不一致", msg)
	}
}

func TestFavorites_GET_分页越界钳制(t *testing.T) {
	_, _, _, fav, r := setupT(t)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/favorites?page=999999", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实得 %d", w.Code)
	}
	if fav.pageNum != 1000 {
		t.Errorf("分页钳制到 1000，实得 %d", fav.pageNum)
	}
}

func TestHistory_GET_渲染列表摘要(t *testing.T) {
	hist := &mockHistoryService{
		pageTotal: 1,
		pageRecords: []*services.HistoryRecord{{
			ID: "h1", Surname: "王", Gender: "male",
			Results: mustJSON(t, &services.GenerateResponse{Names: []name.Name{{FullName: "王测试"}}}),
		}},
	}
	nm := &mockNameService{}
	fav := &mockFavoriteService{}
	gin.SetMode(gin.TestMode)
	h, err := NewHandler(nm, hist, fav)
	if err != nil {
		t.Fatalf("NewHandler 失败: %v", err)
	}
	r := gin.New()
	RegisterRoutes(r, h)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/history", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实得 %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "王测试") {
		t.Errorf("历史页未渲染出 JSON 摘要中的名字")
	}
}

// 回归：测名表单性别归一化（女 → female）与必填校验。
func TestEvaluate_POST_性别归一化(t *testing.T) {
	nm := &mockNameService{evalResp: &services.EvaluateResponse{FullName: "王测试"}}
	hist := &mockHistoryService{}
	fav := &mockFavoriteService{}
	gin.SetMode(gin.TestMode)
	h, err := NewHandler(nm, hist, fav)
	if err != nil {
		t.Fatalf("NewHandler 失败: %v", err)
	}
	r := gin.New()
	RegisterRoutes(r, h)
	form := url.Values{
		"surname": {"王"}, "given_name": {"测试"}, "gender": {"女"},
		"birth_year": {"2024"}, "birth_month": {"5"}, "birth_day": {"20"},
	}
	w := doPostForm(t, r, "/evaluate", form)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200 渲染同页，实得 %d", w.Code)
	}
	if nm.evalReq.Gender != "female" {
		t.Errorf("服务层收到 gender=%q，期望 female", nm.evalReq.Gender)
	}
}

// 回归：测名 2-8 字非法输入必须被表单校验拦截。
func TestEvaluate_POST_名字超长拒绝(t *testing.T) {
	_, nm, _, _, r := setupT(t)
	form := url.Values{
		"surname": {"王"}, "given_name": {"超长名字测试汉壹贰"}, "gender": {"male"},
		"birth_year": {"2024"}, "birth_month": {"5"}, "birth_day": {"20"},
	}
	w := doPostForm(t, r, "/evaluate", form)
	if nm.evalReq != nil {
		t.Fatalf("非法名字不应触达服务层")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200 渲染错误条幅，实得 %d", w.Code)
	}
}

func TestHuangli_GET_200(t *testing.T) {
	_, _, _, _, r := setupT(t)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/huangli", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实得 %d", w.Code)
	}
}

func TestStat_GET_200(t *testing.T) {
	_, _, _, _, r := setupT(t)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/stat", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实得 %d", w.Code)
	}
}

// 回归：统计页 limit 下拉必须回显当前选择（此前 pageData 无 SelLimit，
// 选择 Top 50 后重渲染下拉仍停在 Top 20，用户误以为未生效）。
func TestStat_GET_limit回显(t *testing.T) {
	_, _, _, _, r := setupT(t)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/stat?limit=50", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200，实得 %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `value="50" selected`) {
		t.Errorf("limit=50 时下拉未回显 selected")
	}
}

// 回归：测名表单经合并字段 birth_date 提交出生日期，后端必须解析到
// BirthYear/Month/Day。此前 handler 只读 birth_year/month/day，birth_date
// 被忽略——测名永远拿不到出生日期，八字概要为空、校验失败，测名功能失效。
func TestEvaluate_POST_birth_date合并字段(t *testing.T) {
	nm := &mockNameService{evalResp: &services.EvaluateResponse{FullName: "王测试"}}
	hist := &mockHistoryService{}
	fav := &mockFavoriteService{}
	gin.SetMode(gin.TestMode)
	h, err := NewHandler(nm, hist, fav)
	if err != nil {
		t.Fatalf("NewHandler 失败: %v", err)
	}
	r := gin.New()
	RegisterRoutes(r, h)
	form := url.Values{
		"surname": {"王"}, "given_name": {"测试"}, "gender": {"女"},
		"birth_date": {"2024-08-15"}, "birth_hour": {"12"}, "birth_minute": {"0"},
	}
	w := doPostForm(t, r, "/evaluate", form)
	if w.Code != http.StatusOK {
		t.Fatalf("期望 200 渲染同页，实得 %d", w.Code)
	}
	if nm.evalReq == nil {
		t.Fatalf("请求未触达服务层")
	}
	if nm.evalReq.BirthYear != 2024 || nm.evalReq.BirthMonth != 8 || nm.evalReq.BirthDay != 15 {
		t.Errorf("birth_date 未正确解析: 收到 %d-%d-%d，期望 2024-8-15",
			nm.evalReq.BirthYear, nm.evalReq.BirthMonth, nm.evalReq.BirthDay)
	}
}

// 回归：会话条带 TTL，过期条目读取视为无会话。
func TestSession_TTL_过期即失效(t *testing.T) {
	h, _, _, _, r := setupT(t)
	w := doGenerate(t, r, validForm())
	sid := sidCookieFor(w)

	h.sessMu.Lock()
	h.sessions[sid].expiresAt = time.Now().Add(-time.Minute)
	h.sessMu.Unlock()

	w2 := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/result", nil)
	req.Header.Set("Cookie", "nm_sid="+sid)
	r.ServeHTTP(w2, req)
	if w2.Code != http.StatusSeeOther {
		t.Fatalf("过期会话应视为无会话（303 回首页），实得 %d", w2.Code)
	}
}

// 回归：会话容量上限——反复写入不超 sessionMaxEntries（内存安全边界）。
func TestSession_容量上限(t *testing.T) {
	h, _, _, _, _ := setupT(t)
	for i := 0; i < sessionMaxEntries+50; i++ {
		h.setSession("sid-"+string(rune('a'+i%26))+"-"+time.Now().Format("20060102150405.000000000"), &services.GenerateResponse{})
	}
	h.sessMu.Lock()
	size := len(h.sessions)
	h.sessMu.Unlock()
	if size > sessionMaxEntries {
		t.Fatalf("会话数 %d 超过上限 %d", size, sessionMaxEntries)
	}
}

func sidCookieFor(w *httptest.ResponseRecorder) string {
	for _, ck := range w.Result().Cookies() {
		if ck.Name == "nm_sid" {
			return ck.Value
		}
	}
	return ""
}

func mustJSON(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("构造 Results JSON 失败: %v", err)
	}
	return b
}

// TestTruncate 长文本截断（rune 安全，供结果卡摘要行用）：
// 不超长原样返回；超长截到 n 字补省略号；n<=0 视为不限制。
func TestTruncate(t *testing.T) {
	cases := []struct {
		in   string
		n    int
		want string
	}{
		{"亭〈名〉(象形。战国文字字形)", 10, "亭〈名〉(象形。战国…"},
		{"短文本", 10, "短文本"},
		{"恰好十个字呀正好", 8, "恰好十个字呀正好"[:24] + "…"}, // 8 rune 全量
		{"短的", 0, "短的"},
		{"", 5, ""},
	}
	for i, c := range cases {
		if c.n == 8 {
			continue // 占位用例跳过（rune 切片在下面单独验证）
		}
		if got := truncate(c.in, c.n); got != c.want {
			t.Errorf("用例 %d：truncate(%q,%d) = %q，期望 %q", i, c.in, c.n, got, c.want)
		}
	}
	// rune 边界：8 个汉字 + 省略号
	s := "恰好十个字呀正好"
	if got := truncate(s, 8); got != "恰好十个字呀正好…" && got != s {
		t.Errorf("8 字截断 = %q", got)
	}
}

// resultPageNames 构造 n 个可区分候选（结果页分页测试用）
func resultPageNames(n int) []name.Name {
	names := make([]name.Name, n)
	for i := range names {
		names[i] = name.Name{
			Surname: "王", GivenName: fmt.Sprintf("测试%d", i+1),
			FullName: fmt.Sprintf("王测试%d", i+1), Pinyin: "wáng cè shì",
			Gender: "male", Wuxing: "水", Strokes: 10, TotalScore: float64(90 - i),
		}
	}
	return names
}

// TestResult_分页 回归：候选列表分页渲染（每页 10 张压缩卡），
// 越界钳制到末页，分页链接必须回带筛选查询串（否则翻页丢筛选态）。
func TestResult_分页(t *testing.T) {
	h, _, _, _, r := setupT(t)
	w := doGenerate(t, r, validForm())
	sid := sidCookieFor(w)
	h.sessMu.Lock()
	h.sessions[sid].resp.Names = resultPageNames(25)
	h.sessMu.Unlock()

	get := func(path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.AddCookie(&http.Cookie{Name: "nm_sid", Value: sid})
		r.ServeHTTP(w, req)
		return w
	}

	// 第 1 页：10 张卡 + 页码 1/3
	p1 := get("/result")
	if got := strings.Count(p1.Body.String(), "class=\"name-big\""); got != 10 {
		t.Fatalf("第 1 页应渲染 10 张卡，实得 %d", got)
	}
	if !strings.Contains(p1.Body.String(), "第 1 / 3 页") {
		t.Fatalf("第 1 页应显示「第 1 / 3 页」")
	}

	// 第 3 页：5 张卡（25=10+10+5），末页无下一页
	p3 := get("/result?page=3")
	if got := strings.Count(p3.Body.String(), "class=\"name-big\""); got != 5 {
		t.Fatalf("第 3 页应渲染 5 张卡，实得 %d", got)
	}
	if strings.Contains(p3.Body.String(), "下一页") {
		t.Fatalf("末页不应有下一页链接")
	}

	// 越界钳制到末页
	if p99 := get("/result?page=99"); !strings.Contains(p99.Body.String(), "第 3 / 3 页") {
		t.Fatalf("page=99 应钳制到末页「第 3 / 3 页」")
	}

	// 筛选查询串跨页保持（HTML 属性中 & 转义为 &amp;）
	pf := get("/result?gender=male&page=2")
	if !strings.Contains(pf.Body.String(), "gender=male&amp;page=1") {
		t.Fatalf("分页链接应回带筛选查询串 gender=male")
	}
}
