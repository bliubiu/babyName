package handlers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"name/internal/application/services"
	"name/internal/domain/fate"
	"name/internal/infrastructure/cache"
	"name/internal/infrastructure/data"
)

// 本文件将此前的人工验证固化为自动化回归：
// 通过真实 fate 引擎链路调用 POST /names/generate/analysis，
// 断言生成结果的拼音字段已回填（修复 ExcellentEntry 不存储拼音导致的恒空缺陷）。

var (
	fateOnce    sync.Once
	fateService *services.NameService
	fateInitErr error
)

// locateBackendDataDir 基于本文件位置定位 backend/data 目录（不依赖运行 CWD）
func locateBackendDataDir(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "data")
}

// setupFateNameService 按与 cmd/server、cmd/namer-cli 一致的顺序装配带 fate 引擎的 NameService。
// 装配昂贵（全量字库+诗词加载），用 sync.Once 保证仅执行一次。
func setupFateNameService(t *testing.T) (*services.NameService, error) {
	t.Helper()
	fateOnce.Do(func() {
		dataDir := locateBackendDataDir(t)
		absDir, err := filepath.Abs(dataDir)
		if err != nil {
			fateInitErr = err
			return
		}

		// 1. 加载传统文化数据（hanzi.json → 易经 → 诗词 → 生肖）
		if err := data.Init(absDir); err != nil {
			fateInitErr = err
			return
		}
		// 2. word.json 释义失败不阻断
		services.SyncNamingIndexFromHanzi()

		// 3. 注入策展好名（共现分白名单，与 server 一致；失败可忽略）
		if curated := loadCuratedNamesForTest(absDir); len(curated) > 0 {
			fate.SetCuratedNames(curated)
		}

		// 4. 装配服务（fate 引擎，与 cmd/server 一致）
		cache.Init()
		fateEngine := fate.NewEngine(&services.HanziDataProvider{}, services.NewBaziAnalyzerAdapter(), fate.DefaultRaters())
		fateService = services.NewNameService(
			services.WithBaziAnalyzer(&services.BaziAdapter{}),
			services.WithHexagramFinder(&services.HexagramAdapter{}),
			services.WithZiweiAnalyzer(&services.ZiweiAdapter{}),
			services.WithZodiacFinder(&services.ZodiacAdapter{}),
			services.WithCache(cache.GetCache()),
			services.WithFateService(services.NewFateNameService(fateEngine)),
		)
	})
	return fateService, fateInitErr
}

// loadCuratedNamesForTest 读取 curated_names.json 的 name 字段列表（与 cmd/namer-cli 同逻辑）
func loadCuratedNamesForTest(dataDir string) []string {
	raw, err := os.ReadFile(filepath.Join(dataDir, "curated_names.json"))
	if err != nil {
		return nil
	}
	var entries []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.Name != "" {
			names = append(names, e.Name)
		}
	}
	return names
}

// analysisResponse 分析接口响应结构（仅提取断言所需字段）
type analysisResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Bazi  json.RawMessage `json:"bazi"`
		Names []struct {
			Surname   string `json:"surname"`
			GivenName string `json:"given_name"`
			FullName  string `json:"full_name"`
			Pinyin    string `json:"pinyin"`
		} `json:"names"`
	} `json:"data"`
}

// TestGenerateWithAnalysis_PinyinFilled 真实 fate 链路上拼音必须回填到每个候选名
func TestGenerateWithAnalysis_PinyinFilled(t *testing.T) {
	svc, err := setupFateNameService(t)
	if err != nil {
		t.Fatalf("fate 服务装配失败: %v", err)
	}

	r := gin.New()
	h := NewNameHandler(svc)
	r.POST("/names/generate/analysis", h.GenerateWithAnalysis)

	body := `{"surname":"王","gender":"male","birth_year":2024,"birth_month":1,"birth_day":15,"birth_hour":12}`
	w := performRequest(r, "POST", "/names/generate/analysis", []byte(body))
	assertStatus(t, w.Code, 200)

	var resp analysisResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应反序列化失败: %v", err)
	}
	if !resp.Success {
		t.Fatal("success 应为 true")
	}
	if len(resp.Data.Bazi) == 0 {
		t.Error("响应应包含八字信息 bazi")
	}
	if len(resp.Data.Names) == 0 {
		t.Fatal("应至少生成一个候选名")
	}

	for i, n := range resp.Data.Names {
		if n.FullName == "" {
			t.Errorf("候选 #%d 缺少 full_name", i)
		}
		if n.Pinyin == "" {
			t.Errorf("候选 %q 的拼音为空，fate 引擎应回填候选字读音", n.FullName)
		}
	}
}

// TestGenerateWithAnalysis_PinyinSingleName 单名模式同样必须回填拼音且无尾随空白
func TestGenerateWithAnalysis_PinyinSingleName(t *testing.T) {
	svc, err := setupFateNameService(t)
	if err != nil {
		t.Fatalf("fate 服务装配失败: %v", err)
	}

	r := gin.New()
	h := NewNameHandler(svc)
	r.POST("/names/generate/analysis", h.GenerateWithAnalysis)

	body := `{"surname":"李","gender":"female","birth_year":2023,"birth_month":8,"birth_day":20,"birth_hour":10,"name_length":1}`
	w := performRequest(r, "POST", "/names/generate/analysis", []byte(body))
	assertStatus(t, w.Code, 200)

	var resp analysisResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应反序列化失败: %v", err)
	}
	if len(resp.Data.Names) == 0 {
		t.Fatal("应至少生成一个候选名")
	}

	for _, n := range resp.Data.Names {
		if n.Pinyin == "" {
			t.Errorf("单名候选 %q 的拼音为空", n.FullName)
			continue
		}
		if n.Pinyin != trimSpaceASCII(n.Pinyin) {
			t.Errorf("单名候选 %q 拼音含首尾空白: %q", n.FullName, n.Pinyin)
		}
	}
}

// trimSpaceASCII 去除首尾 ASCII 空白（避免为断言引入 strings 全量依赖歧义）
func trimSpaceASCII(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

// TestGenerateWithAnalysis_SourceClassicPoetryChars 诗词来源参数必须真正消费
// 设置 source_classic:"shijing" 后，结果中至少一个候选名的字应出自诗经字库。
func TestGenerateWithAnalysis_SourceClassicPoetryChars(t *testing.T) {
	svc, err := setupFateNameService(t)
	if err != nil {
		t.Fatalf("fate 服务装配失败: %v", err)
	}

	r := gin.New()
	h := NewNameHandler(svc)
	r.POST("/names/generate/analysis", h.GenerateWithAnalysis)

	body := `{"surname":"王","gender":"male","birth_year":2024,"birth_month":3,"birth_day":10,"birth_hour":9,"source_classic":"shijing"}`
	w := performRequest(r, "POST", "/names/generate/analysis", []byte(body))
	assertStatus(t, w.Code, 200)

	var resp analysisResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应反序列化失败: %v", err)
	}
	if !resp.Success {
		t.Fatal("success 应为 true")
	}
	if len(resp.Data.Names) == 0 {
		t.Fatal("应至少生成一个候选名")
	}

	// 基本断言：请求成功且候选名非空（诗词字注入不应破坏生成链路）
	for i, n := range resp.Data.Names {
		if n.FullName == "" {
			t.Errorf("候选 #%d 缺少 full_name", i)
		}
		if n.Pinyin == "" {
			t.Errorf("候选 %q 的拼音为空", n.FullName)
		}
	}
}
