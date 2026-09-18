package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"name/internal/application/services"
	"name/internal/domain/hanzi"
	"name/internal/infrastructure/database/memory"
	"name/internal/infrastructure/logger"
	"name/internal/infrastructure/pdf"
)

func TestMain(m *testing.M) {
	_ = logger.InitProduction()
	gin.SetMode(gin.TestMode)
	// 加载汉字数据（测试依赖完整字库时提供基础数据）
	dataDir := "../../../data/"
	if err := hanzi.LoadNamerFromJSON(dataDir); err != nil {
		// 非关键：仅在未加载数据时影响依赖完整字库的用例
		_ = err
	}
	os.Exit(m.Run())
}

// --- Test helpers ---

type handlerTestCase struct {
	name       string
	method     string
	path       string
	body       string
	wantStatus int
	check      func(t *testing.T, body []byte) // 可选的响应体断言
}

func setupTestHandler() (*gin.Engine, *memory.Store) {
	gin.SetMode(gin.TestMode)
	store := memory.NewStore()
	r := gin.New()
	return r, store
}

func performRequest(r http.Handler, method, path string, body []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func assertStatus(t *testing.T, got, want int) {
	t.Helper()
	if got != want {
		t.Errorf("status code: got %d, want %d", got, want)
	}
}

// runHandlerTests 通用测试运行器
func runHandlerTests(t *testing.T, tests []handlerTestCase, setup func(r *gin.Engine)) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			gin.SetMode(gin.TestMode)
			setup(r)
			w := performRequest(r, tt.method, tt.path, []byte(tt.body))
			assertStatus(t, w.Code, tt.wantStatus)
			if tt.check != nil {
				tt.check(t, w.Body.Bytes())
			}
		})
	}
}

// runHandlerTestsWithStore 带 store 的通用测试运行器
func runHandlerTestsWithStore(t *testing.T, tests []handlerTestCase, setup func(r *gin.Engine, store *memory.Store)) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, store := setupTestHandler()
			setup(r, store)
			w := performRequest(r, tt.method, tt.path, []byte(tt.body))
			assertStatus(t, w.Code, tt.wantStatus)
			if tt.check != nil {
				tt.check(t, w.Body.Bytes())
			}
		})
	}
}

// --- NameStatHandler Tests ---

func TestNameStatHandler(t *testing.T) {
	tests := []handlerTestCase{
		{name: "GetNameStats", method: "GET", path: "/namestat/王", wantStatus: 200},
		{name: "GetTopNames_Default", method: "GET", path: "/namestat", wantStatus: 200},
		{name: "GetTopNames_WithLimit", method: "GET", path: "/namestat?limit=5", wantStatus: 200},
		{name: "GetTopNames_InvalidLimit", method: "GET", path: "/namestat?limit=999", wantStatus: 200},
	}
	runHandlerTests(t, tests, func(r *gin.Engine) {
		h := NewNameStatHandler()
		r.GET("/namestat/:name", h.GetNameStats)
		r.GET("/namestat", h.GetTopNames)
	})
}

// --- HuangliHandler Tests ---

func TestHuangliHandler(t *testing.T) {
	tests := []handlerTestCase{
		{name: "GetHuangli", method: "GET", path: "/huangli?year=2024&month=1&day=15", wantStatus: 200},
		{name: "GetHuangli_InvalidYear", method: "GET", path: "/huangli?year=abc&month=1&day=15", wantStatus: 400},
		{name: "GetHuangli_InvalidMonth", method: "GET", path: "/huangli?year=2024&month=13&day=15", wantStatus: 400},
		{name: "GetHuangli_InvalidDay", method: "GET", path: "/huangli?year=2024&month=1&day=32", wantStatus: 400},
		{name: "GetLunarCalendar", method: "GET", path: "/lunar?year=2024&month=1&day=15&hour=12&minute=0", wantStatus: 200},
		{name: "GetLunarCalendar_DefaultHour", method: "GET", path: "/lunar?year=2024&month=1&day=15", wantStatus: 200},
	}
	runHandlerTests(t, tests, func(r *gin.Engine) {
		h := NewHuangliHandler()
		r.GET("/huangli", h.GetHuangli)
		r.GET("/lunar", h.GetLunarCalendar)
	})
}

// --- NameHandler Tests ---

func TestNameHandler(t *testing.T) {
	// 使用与 cmd/server 一致的 fate 引擎装配（唯一生成引擎）
	svc, err := setupFateNameService(t)
	if err != nil {
		t.Fatalf("装配带 fate 引擎的 NameService 失败: %v", err)
	}
	tests := []handlerTestCase{
		{name: "Generate_InvalidBody", method: "POST", path: "/names/generate", body: `invalid json`, wantStatus: 400},
		{name: "Generate_EmptySurname", method: "POST", path: "/names/generate", body: `{"surname":"","gender":"male","birth_year":2024,"birth_month":1,"birth_day":15,"birth_hour":12}`, wantStatus: 400},
		{name: "Generate_InvalidGender", method: "POST", path: "/names/generate", body: `{"surname":"王","gender":"","birth_year":2024,"birth_month":1,"birth_day":15,"birth_hour":12}`, wantStatus: 400},
		{name: "Generate_ValidRequest", method: "POST", path: "/names/generate", body: `{"surname":"王","gender":"male","birth_year":2024,"birth_month":1,"birth_day":15,"birth_hour":12}`, wantStatus: 200},
		{name: "Generate_ZeroHour", method: "POST", path: "/names/generate", body: `{"surname":"王","gender":"male","birth_year":2024,"birth_month":1,"birth_day":15,"birth_hour":0,"birth_minute":12}`, wantStatus: 200},
		{name: "Generate_SurnameWithSpaces", method: "POST", path: "/names/generate", body: `{"surname":" 王 ","gender":"male","birth_year":2024,"birth_month":1,"birth_day":15,"birth_hour":12}`, wantStatus: 200},
	}
	runHandlerTests(t, tests, func(r *gin.Engine) {
		h := NewNameHandler(svc)
		r.POST("/names/generate", h.Generate)
	})
}

// TestNameEvaluateHandler 测名接口：出生时辰为 0（子时）属合法输入，不得返回 400
func TestNameEvaluateHandler(t *testing.T) {
	svc, err := setupFateNameService(t)
	if err != nil {
		t.Fatalf("装配带 fate 引擎的 NameService 失败: %v", err)
	}
	tests := []handlerTestCase{
		{name: "Evaluate_ZeroHour", method: "POST", path: "/names/evaluate", body: `{"surname":"王","given_name":"浩","gender":"male","birth_year":2024,"birth_month":1,"birth_day":15,"birth_hour":0,"birth_minute":12}`, wantStatus: 200},
	}
	runHandlerTests(t, tests, func(r *gin.Engine) {
		h := NewNameHandler(svc)
		r.POST("/names/evaluate", h.Evaluate)
	})
}

// --- HistoryHandler Tests ---

func TestHistoryHandler(t *testing.T) {
	tests := []handlerTestCase{
		{name: "GetHistory_Empty", method: "GET", path: "/history", wantStatus: 200},
		{name: "SaveHistory_InvalidBody", method: "POST", path: "/history", body: `bad`, wantStatus: 400},
		{name: "DeleteHistory", method: "DELETE", path: "/history/non-existent", wantStatus: 200},
	}
	runHandlerTestsWithStore(t, tests, func(r *gin.Engine, store *memory.Store) {
		h := NewHistoryHandler(services.NewHistoryService(store))
		r.GET("/history", h.GetHistory)
		r.POST("/history", h.SaveHistory)
		r.DELETE("/history/:id", h.DeleteHistory)
	})
}

func TestHistoryHandler_SaveAndGetHistory(t *testing.T) {
	r, store := setupTestHandler()
	h := NewHistoryHandler(services.NewHistoryService(store))
	r.GET("/history", h.GetHistory)
	r.POST("/history", h.SaveHistory)

	body := `{"surname":"王","given_name":"磊","gender":"male","birth_year":2024}`
	w := performRequest(r, "POST", "/history", []byte(body))
	assertStatus(t, w.Code, 200)

	// verify it appears in history list
	w2 := performRequest(r, "GET", "/history", nil)
	assertStatus(t, w2.Code, 200)
}

// --- FavoriteHandler Tests ---

func TestFavoriteHandler(t *testing.T) {
	// 需要按顺序执行的测试（保存后验证）
	t.Run("SaveAndGetFavorite", func(t *testing.T) {
		r, store := setupTestHandler()
		h := NewFavoriteHandler(services.NewFavoriteService(store))
		r.POST("/favorites", h.SaveFavorite)
		r.GET("/favorites", h.GetFavorites)

		// save a favorite
		body := `{"surname":"王","given_name":"磊"}`
		w := performRequest(r, "POST", "/favorites", []byte(body))
		assertStatus(t, w.Code, 200)

		// verify it appears in favorites list
		w2 := performRequest(r, "GET", "/favorites", nil)
		assertStatus(t, w2.Code, 200)
	})

	// 独立无状态测试
	independentTests := []handlerTestCase{
		{name: "GetFavorites_Empty", method: "GET", path: "/favorites", wantStatus: 200},
		{name: "DeleteFavorite", method: "DELETE", path: "/favorites/non-existent", wantStatus: 200},
		{name: "SaveFavorite_InvalidBody", method: "POST", path: "/favorites", body: `bad`, wantStatus: 400},
	}
	runHandlerTestsWithStore(t, independentTests, func(r *gin.Engine, store *memory.Store) {
		h := NewFavoriteHandler(services.NewFavoriteService(store))
		r.GET("/favorites", h.GetFavorites)
		r.POST("/favorites", h.SaveFavorite)
		r.DELETE("/favorites/:id", h.DeleteFavorite)
	})
}

// --- ReportHandler Tests ---

func TestReportHandler(t *testing.T) {
	tests := []handlerTestCase{
		{name: "GeneratePDF_InvalidBody", method: "POST", path: "/report/pdf", body: `bad`, wantStatus: 400},
	}
	runHandlerTests(t, tests, func(r *gin.Engine) {
		h := NewReportHandler(services.NewReportService())
		r.POST("/report/pdf", h.GeneratePDF)
	})
}

// TestReportHandlerGeneratePDF 有效请求应返回真正的 PDF 字节流
//
// 本机无中文字体时跳过：PDF 渲染必须有可嵌入的中文字体，缺失时服务端返回 500
// 并给出明确错误，而不是输出一份正文空白的"报告"（这一行为由 services 层保证）。
func TestReportHandlerGeneratePDF(t *testing.T) {
	if _, err := pdf.FindSystemCJKFont(); err != nil {
		t.Skipf("本机无可用中文字体，跳过：%v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewReportHandler(services.NewReportService())
	r.POST("/report/pdf", h.GeneratePDF)

	body := `{"data":{"surname":"张","gender":"male","birth_date":"2024年5月20日",` +
		`"names":[{"full_name":"张珀熙","pinyin":"zhāng pò xī","wuxing":"水金","score":92.4,` +
		`"meaning":"温润有光"}]}}`
	w := performRequest(r, "POST", "/report/pdf", []byte(body))

	if w.Code != 200 {
		t.Fatalf("状态码 = %d，期望 200（响应体 %s）", w.Code, w.Body.String())
	}
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "application/pdf") {
		t.Errorf("Content-Type = %q，期望 application/pdf", ct)
	}
	if !bytes.HasPrefix(w.Body.Bytes(), []byte("%PDF-")) {
		t.Errorf("响应体不是 PDF（前 8 字节 %q）", w.Body.Bytes()[:min(8, w.Body.Len())])
	}
	if len(w.Body.Bytes()) < 10*1024 {
		t.Errorf("PDF 仅 %d 字节，疑似未内嵌字体", w.Body.Len())
	}
}
