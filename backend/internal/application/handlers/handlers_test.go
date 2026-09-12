package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"name/internal/application/services"
	"name/internal/domain/hanzi"
	"name/internal/domain/namestatistics"
	"name/internal/infrastructure/cache"
	"name/internal/infrastructure/database"
	"name/internal/infrastructure/database/memory"
	"name/internal/infrastructure/logger"
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

func newTestBaziService() *services.BaziService {
	return services.NewBaziService(cache.GetCache())
}

func newTestYijingService(store *memory.Store) *services.YijingService {
	return services.NewYijingService(store, cache.GetCache())
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

func assertJSON(t *testing.T, body []byte, key string, wantVal interface{}) {
	t.Helper()
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
		return
	}
	data, ok := result["data"]
	if !ok {
		t.Fatalf("response missing 'data' field: %v", result)
	}
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		t.Fatalf("response 'data' is not an object: %v", data)
	}
	val, ok := dataMap[key]
	if !ok {
		t.Errorf("response data missing key %q, data: %v", key, dataMap)
		return
	}
	if fmt.Sprintf("%v", val) != fmt.Sprintf("%v", wantVal) {
		t.Errorf("%s: got %v, want %v", key, val, wantVal)
	}
}

func assertDataJSON(t *testing.T, body []byte, key string, wantVal interface{}) {
	t.Helper()
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
		return
	}
	data, ok := result["data"]
	if !ok {
		t.Errorf("response missing key %q", "data")
		return
	}
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		t.Errorf("data is not an object")
		return
	}
	val, ok := dataMap[key]
	if !ok {
		t.Errorf("response.data missing key %q", key)
		return
	}
	if fmt.Sprintf("%v", val) != fmt.Sprintf("%v", wantVal) {
		t.Errorf("data.%s: got %v, want %v", key, val, wantVal)
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
		{name: "Generate_SurnameWithSpaces", method: "POST", path: "/names/generate", body: `{"surname":" 王 ","gender":"male","birth_year":2024,"birth_month":1,"birth_day":15,"birth_hour":12}`, wantStatus: 200},
		{name: "GenerateWithAnalysis_ValidRequest", method: "POST", path: "/names/generate/analysis", body: `{"surname":"王","gender":"male","birth_year":2024,"birth_month":1,"birth_day":15,"birth_hour":12}`, wantStatus: 200},
		{name: "GenerateWithAnalysis_InvalidBody", method: "POST", path: "/names/generate/analysis", body: `{invalid}`, wantStatus: 400},
	}
	runHandlerTests(t, tests, func(r *gin.Engine) {
		h := NewNameHandler(svc)
		r.POST("/names/generate", h.Generate)
		r.POST("/names/generate/analysis", h.GenerateWithAnalysis)
	})
}

// --- BaziHandler Tests ---

func TestBaziHandler(t *testing.T) {
	svc := newTestBaziService()
	tests := []handlerTestCase{
		{name: "Analyze_ValidRequest", method: "POST", path: "/bazi/analyze", body: `{"year":2024,"month":1,"day":15,"hour":12}`, wantStatus: 200},
		{name: "Analyze_MissingRequired", method: "POST", path: "/bazi/analyze", body: `{"year":2024}`, wantStatus: 400},
	}
	runHandlerTests(t, tests, func(r *gin.Engine) {
		h := NewBaziHandler(svc)
		r.POST("/bazi/analyze", h.Analyze)
	})
}

// --- YijingHandler Tests ---

func TestYijingHandler(t *testing.T) {
	tests := []handlerTestCase{
		{name: "GetAllHexagrams", method: "GET", path: "/yijing/hexagram", wantStatus: 200},
		{name: "GetHexagram_Valid", method: "GET", path: "/yijing/hexagram/1", wantStatus: 200},
		{name: "GetHexagram_NotFound", method: "GET", path: "/yijing/hexagram/999", wantStatus: 404},
		{name: "GetHexagram_NonNumericID", method: "GET", path: "/yijing/hexagram/abc", wantStatus: 404},
	}
	runHandlerTestsWithStore(t, tests, func(r *gin.Engine, store *memory.Store) {
		h := NewYijingHandler(newTestYijingService(store))
		r.GET("/yijing/hexagram", h.GetAllHexagrams)
		r.GET("/yijing/hexagram/:id", h.GetHexagram)
	})
}

// --- ZodiacHandler Tests ---

func TestZodiacHandler(t *testing.T) {
	tests := []handlerTestCase{
		{name: "GetAllZodiacs", method: "GET", path: "/zodiac", wantStatus: 200},
		{name: "GetZodiac_Valid", method: "GET", path: "/zodiac/鼠", wantStatus: 200},
		{name: "GetZodiac_NotFound", method: "GET", path: "/zodiac/不存在", wantStatus: 404},
	}
	runHandlerTestsWithStore(t, tests, func(r *gin.Engine, store *memory.Store) {
		h := NewZodiacHandler(services.NewZodiacService(store))
		r.GET("/zodiac", h.GetAllZodiacs)
		r.GET("/zodiac/:animal", h.GetZodiac)
	})
}

// --- HistoryHandler Tests ---

func TestHistoryHandler(t *testing.T) {
	tests := []handlerTestCase{
		{name: "GetHistory_Empty", method: "GET", path: "/history", wantStatus: 200},
		{name: "SaveHistory_InvalidBody", method: "POST", path: "/history", body: `bad`, wantStatus: 400},
		{name: "DeleteHistory", method: "DELETE", path: "/history/non-existent", wantStatus: 200},
		{name: "BatchSaveHistory", method: "POST", path: "/history/batch", body: `[{"surname":"王","given_name":"磊","gender":"male","birth_year":2024}]`, wantStatus: 200},
		{name: "BatchSaveHistory_InvalidBody", method: "POST", path: "/history/batch", body: `invalid`, wantStatus: 400},
	}
	runHandlerTestsWithStore(t, tests, func(r *gin.Engine, store *memory.Store) {
		h := NewHistoryHandler(services.NewHistoryService(store))
		r.GET("/history", h.GetHistory)
		r.POST("/history", h.SaveHistory)
		r.DELETE("/history/:id", h.DeleteHistory)
		r.POST("/history/batch", h.BatchSaveHistory)
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
	t.Run("SaveAndCheckFavorite", func(t *testing.T) {
		r, store := setupTestHandler()
		h := NewFavoriteHandler(services.NewFavoriteService(store))
		r.POST("/favorites", h.SaveFavorite)
		r.GET("/favorites/check", h.CheckFavorite)

		// save a favorite
		body := `{"surname":"王","given_name":"磊"}`
		w := performRequest(r, "POST", "/favorites", []byte(body))
		assertStatus(t, w.Code, 200)

		// check it exists
		w2 := performRequest(r, "GET", "/favorites/check?surname=王&given_name=磊", nil)
		assertStatus(t, w2.Code, 200)
		assertDataJSON(t, w2.Body.Bytes(), "is_favorite", true)

		// check non-existent
		w3 := performRequest(r, "GET", "/favorites/check?surname=李&given_name=四", nil)
		assertStatus(t, w3.Code, 200)
		assertDataJSON(t, w3.Body.Bytes(), "is_favorite", false)
	})

	// 独立无状态测试
	independentTests := []handlerTestCase{
		{name: "GetFavorites_Empty", method: "GET", path: "/favorites", wantStatus: 200},
		{name: "CheckFavorite_MissingParams", method: "GET", path: "/favorites/check?name=test", wantStatus: 400},
		{name: "DeleteFavorite", method: "DELETE", path: "/favorites/non-existent", wantStatus: 200},
		{name: "BatchSaveFavorite", method: "POST", path: "/favorites/batch", body: `[{"surname":"王","given_name":"磊"},{"surname":"李","given_name":"华"}]`, wantStatus: 200},
		{name: "BatchSaveFavorite_InvalidBody", method: "POST", path: "/favorites/batch", body: `bad json`, wantStatus: 400},
		{name: "BatchDeleteFavorite", method: "DELETE", path: "/favorites/batch", body: `{"ids":["non-existent-1","non-existent-2"]}`, wantStatus: 200},
		{name: "BatchDeleteFavorite_InvalidBody", method: "DELETE", path: "/favorites/batch", body: `bad json`, wantStatus: 400},
		{name: "SaveFavorite_InvalidBody", method: "POST", path: "/favorites", body: `bad`, wantStatus: 400},
	}
	runHandlerTestsWithStore(t, independentTests, func(r *gin.Engine, store *memory.Store) {
		h := NewFavoriteHandler(services.NewFavoriteService(store))
		r.GET("/favorites", h.GetFavorites)
		r.POST("/favorites", h.SaveFavorite)
		r.GET("/favorites/check", h.CheckFavorite)
		r.DELETE("/favorites/:id", h.DeleteFavorite)
		r.POST("/favorites/batch", h.BatchSaveFavorite)
		r.DELETE("/favorites/batch", h.BatchDeleteFavorite)
	})
}

// --- FeedbackHandler Tests ---

func TestFeedbackHandler(t *testing.T) {
	tests := []handlerTestCase{
		{name: "SaveFeedback_InvalidBody", method: "POST", path: "/feedback", body: `bad json`, wantStatus: 400},
		{name: "SaveFeedback_MissingGivenName", method: "POST", path: "/feedback", body: `{"request_id":1,"full_name":"王磊"}`, wantStatus: 400},
		{name: "SaveFeedback_Valid", method: "POST", path: "/feedback", body: `{"request_id":1,"full_name":"王磊","given_name":"磊","is_liked":true}`, wantStatus: 200},
		{name: "SaveRequest_InvalidBody", method: "POST", path: "/feedback/request", body: `bad`, wantStatus: 400},
		{name: "SaveRequest_Valid", method: "POST", path: "/feedback/request", body: `{"surname":"王","gender":"male","birth_year":2024,"birth_month":1,"birth_day":15}`, wantStatus: 200},
		{name: "GetAlgorithmPerformance", method: "GET", path: "/feedback/algorithm-performance", wantStatus: 200},
	}
	runHandlerTestsWithStore(t, tests, func(r *gin.Engine, store *memory.Store) {
		h := NewFeedbackHandler(services.NewFeedbackService(store, store))
		r.POST("/feedback", h.SaveFeedback)
		r.POST("/feedback/request", h.SaveRequest)
		r.GET("/feedback/algorithm-performance", h.GetAlgorithmPerformance)
	})
}

// --- ReportHandler Tests ---

func TestReportHandler(t *testing.T) {
	tests := []handlerTestCase{
		{name: "GeneratePDF_InvalidBody", method: "POST", path: "/report/pdf", body: `bad`, wantStatus: 400},
		{name: "GenerateHTML_InvalidBody", method: "POST", path: "/report/html", body: `bad`, wantStatus: 400},
		{name: "GenerateHTML_Valid", method: "POST", path: "/report/html", body: `{"data":{"test":"hello"}}`, wantStatus: 200},
	}
	runHandlerTests(t, tests, func(r *gin.Engine) {
		h := NewReportHandler(services.NewReportService())
		r.POST("/report/pdf", h.GeneratePDF)
		r.POST("/report/html", h.GenerateHTML)
	})
}

// --- NameStatisticsHandler Tests ---
//
// 使用临时目录的 JSON 统计文件验证 /namestats/* 端点（等价 cmd/server 装配）。

// writeNameStatTestData 在临时目录写入最小统计 JSON
func writeNameStatTestData(t *testing.T, dir string) {
	t.Helper()
	write := func(name string, v any) {
		t.Helper()
		raw, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("序列化 %s 失败: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), raw, 0644); err != nil {
			t.Fatalf("写入 %s 失败: %v", name, err)
		}
	}
	write("surname_stats.json", []database.SurnameStat{
		{Surname: "王", Count: 3, Rank: 1},
	})
	write("given_name_stats.json", []database.GivenNameStat{})
	write("full_name_stats.json", []database.FullNameStat{})
	write("name_gender_stats.json", []database.NameGenderStat{})
	write("name_frequency.json", struct {
		Meta struct {
			TotalNames int `json:"total_names"`
		} `json:"meta"`
	}{Meta: struct {
		TotalNames int `json:"total_names"`
	}{TotalNames: 1200000}})
}

// newNameStatisticsHandler 构造带 JSON 数据源的 NameStatisticsHandler
func newNameStatisticsHandler(t *testing.T, dir string) *NameStatisticsHandler {
	t.Helper()
	writeNameStatTestData(t, dir)
	store := namestatistics.NewFileNameStatStore(dir)
	return NewNameStatisticsHandler(namestatistics.NewNameStatisticsService(store))
}

func TestNameStatisticsHandler(t *testing.T) {
	dir := t.TempDir()
	h := newNameStatisticsHandler(t, dir)
	tests := []handlerTestCase{
		{name: "GetSurnameStats", method: "GET", path: "/namestats/surnames", wantStatus: 200},
		{name: "GetSurnameStat", method: "GET", path: "/namestats/surnames/王", wantStatus: 200},
		{name: "GetSurnameStat_Missing", method: "GET", path: "/namestats/surnames/赵", wantStatus: 404},
		{name: "GetGivenNameStats", method: "GET", path: "/namestats/surnames/王/given-names", wantStatus: 200},
		{name: "GetFullNameStats", method: "GET", path: "/namestats/surnames/王/full-names", wantStatus: 200},
		{name: "GetTopFullNames", method: "GET", path: "/namestats/top", wantStatus: 200},
		{name: "GetTotalNameCount", method: "GET", path: "/namestats/total", wantStatus: 200},
		{name: "GetNameGenderStats_Missing", method: "GET", path: "/namestats/names/伟/gender", wantStatus: 404},
	}
	runHandlerTests(t, tests, func(r *gin.Engine) {
		r.GET("/namestats/surnames", h.GetSurnameStats)
		r.GET("/namestats/surnames/:surname", h.GetSurnameStat)
		r.GET("/namestats/surnames/:surname/given-names", h.GetGivenNameStats)
		r.GET("/namestats/surnames/:surname/full-names", h.GetFullNameStats)
		r.GET("/namestats/full-names/:full_name", h.GetFullNameStat)
		r.GET("/namestats/names/:name/gender", h.GetNameGenderStats)
		r.GET("/namestats/top", h.GetTopFullNames)
		r.GET("/namestats/total", h.GetTotalNameCount)
	})
}

// TestNameStatisticsHandler_DataMissing 数据文件缺失时接口返回 500 / 404
func TestNameStatisticsHandler_DataMissing(t *testing.T) {
	store := namestatistics.NewFileNameStatStore(t.TempDir())
	h := NewNameStatisticsHandler(namestatistics.NewNameStatisticsService(store))
	tests := []handlerTestCase{
		{name: "GetTotalNameCount", method: "GET", path: "/namestats/total", wantStatus: 500},
	}
	runHandlerTests(t, tests, func(r *gin.Engine) {
		r.GET("/namestats/total", h.GetTotalNameCount)
	})
}
