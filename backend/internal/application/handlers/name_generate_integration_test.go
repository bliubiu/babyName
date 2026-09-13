package handlers

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// generateNamesResponse /names/generate 响应（仅提取断言所需字段）
type generateNamesResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Names []struct {
			FullName       string `json:"full_name"`
			Gender         string `json:"gender"`
			PoetrySource   string `json:"poetry_source"`
			PoetryChapter  string `json:"poetry_chapter"`
			PoetrySentence string `json:"poetry_sentence"`
			PoetryAuthor   string `json:"poetry_author"`
			PoetryDynasty  string `json:"poetry_dynasty"`
			PoetryFullText string `json:"poetry_full_text"`
		} `json:"names"`
	} `json:"data"`
}

// TestGenerate_GenderFilled 生成接口返回的每个名字必须携带与请求一致的性别。
// 前端 NameCard 依据 name.gender === 'male' 渲染「男」，空串会回退为「女」，
// 导致男性生辰生成的名字全部错误显示「女」（回归：convertFateToNameNames 未映射 Gender）。
func TestGenerate_GenderFilled(t *testing.T) {
	svc, err := setupFateNameService(t)
	if err != nil {
		t.Fatalf("fate 服务装配失败: %v", err)
	}

	r := gin.New()
	h := NewNameHandler(svc)
	r.POST("/names/generate", h.Generate)

	// 男性生辰
	bodyMale := `{"surname":"王","gender":"male","birth_year":2024,"birth_month":1,"birth_day":15,"birth_hour":12}`
	w := performRequest(r, "POST", "/names/generate", []byte(bodyMale))
	assertStatus(t, w.Code, 200)

	var respMale generateNamesResponse
	if err := json.Unmarshal(w.Body.Bytes(), &respMale); err != nil {
		t.Fatalf("男性请求响应反序列化失败: %v", err)
	}
	if len(respMale.Data.Names) == 0 {
		t.Fatal("应至少生成一个候选名")
	}
	for i, n := range respMale.Data.Names {
		if n.Gender != "male" {
			t.Errorf("候选 #%d %s 性别应为 male，实际 %q", i, n.FullName, n.Gender)
		}
	}

	// 女性生辰
	bodyFemale := `{"surname":"李","gender":"female","birth_year":2023,"birth_month":8,"birth_day":20,"birth_hour":10}`
	w2 := performRequest(r, "POST", "/names/generate", []byte(bodyFemale))
	assertStatus(t, w2.Code, 200)

	var respFemale generateNamesResponse
	if err := json.Unmarshal(w2.Body.Bytes(), &respFemale); err != nil {
		t.Fatalf("女性请求响应反序列化失败: %v", err)
	}
	if len(respFemale.Data.Names) == 0 {
		t.Fatal("女性请求应至少生成一个候选名")
	}
	for i, n := range respFemale.Data.Names {
		if n.Gender != "female" {
			t.Errorf("候选 #%d %s 性别应为 female，实际 %q", i, n.FullName, n.Gender)
		}
	}
}

// TestGenerate_PoetryBackfill 生成接口（旧路径 NameService.Generate）必须同样做诗词出处结构化回填。
// 回归：此前该路径仅把引擎格式化输出（「原句」）塞入 poetry_source，前端渲染成
// 出自《「鴥彼晨风」》——出处位置必须是典籍名（诗经/楚辞等），原句走 poetry_sentence。
func TestGenerate_PoetryBackfill(t *testing.T) {
	svc, err := setupFateNameService(t)
	if err != nil {
		t.Fatalf("fate 服务装配失败: %v", err)
	}

	r := gin.New()
	h := NewNameHandler(svc)
	r.POST("/names/generate", h.Generate)

	body := `{"surname":"张","gender":"male","birth_year":2024,"birth_month":1,"birth_day":15,"birth_hour":12}`
	w := performRequest(r, "POST", "/names/generate", []byte(body))
	assertStatus(t, w.Code, 200)

	var resp generateNamesResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应反序列化失败: %v", err)
	}
	if len(resp.Data.Names) == 0 {
		t.Fatal("应至少生成一个候选名")
	}

	// 有诗词出典的名字数必须 > 0（常用字池在诗经/楚辞/唐诗中几乎必有单字出典）
	poeticCount := 0
	completeCount := 0
	for _, n := range resp.Data.Names {
		if n.PoetrySource == "" {
			continue
		}
		poeticCount++
		// 回归断言核心：poetry_source 不得是引擎「原句」格式（含「」），必须是典籍名
		if strings.Contains(n.PoetrySource, "「") || strings.Contains(n.PoetrySource, "」") {
			t.Errorf("名字 %s 的 poetry_source 仍是引擎原句格式 %q，应为典籍名", n.FullName, n.PoetrySource)
		}
		// 完整回链：chapter+sentence+author+dynasty+full_text 齐全的名字至少应存在
		if n.PoetryChapter != "" && n.PoetrySentence != "" && n.PoetryFullText != "" {
			completeCount++
		}
	}
	if poeticCount == 0 {
		t.Fatal("生成的候选名无任何诗词出典，回填链路未生效")
	}
	if completeCount == 0 {
		t.Error("无任何候选名具备完整诗词回链（Chapter/Sentence/FullText 应同时非空）")
	}
}
