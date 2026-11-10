package handlers

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"name/internal/application/services"
)

func setupShareRouter() *gin.Engine {
	r, store := setupTestHandler()
	h := NewShareHandler(services.NewShareService(store))
	r.POST("/api/v1/share", h.CreateShare)
	r.GET("/api/v1/share/:id", h.GetShare)
	return r
}

type shareCreateResponse struct {
	Success bool `json:"success"`
	Data    struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	} `json:"data"`
}

func TestShareHandler_CreateThenGet(t *testing.T) {
	r := setupShareRouter()
	body := []byte(`{"data":{"surname":"李","names":[{"full_name":"李沐宸"}],"birth_location":"不应入库"}}`)
	w := performRequest(r, http.MethodPost, "/api/v1/share", body)
	assertStatus(t, w.Code, http.StatusOK)

	var created shareCreateResponse
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("创建响应解析失败: %v", err)
	}
	if len(created.Data.ID) != 32 || created.Data.URL != "/share?id="+created.Data.ID {
		t.Fatalf("分享链接不正确: %+v", created.Data)
	}

	w = performRequest(r, http.MethodGet, "/api/v1/share/"+created.Data.ID, nil)
	assertStatus(t, w.Code, http.StatusOK)
	var fetched struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &fetched); err != nil {
		t.Fatalf("读取响应解析失败: %v", err)
	}
	if _, leaked := fetched.Data["birth_location"]; leaked {
		t.Error("分享快照泄露了非白名单字段")
	}
	if string(fetched.Data["surname"]) != `"李"` {
		t.Errorf("姓氏未返回: %s", fetched.Data["surname"])
	}
}

func TestShareHandler_InvalidPayloadAndID(t *testing.T) {
	r := setupShareRouter()

	w := performRequest(r, http.MethodPost, "/api/v1/share", []byte(`{"data":{}}`))
	assertStatus(t, w.Code, http.StatusBadRequest)

	w = performRequest(r, http.MethodPost, "/api/v1/share", []byte(`not-json`))
	assertStatus(t, w.Code, http.StatusBadRequest)

	w = performRequest(r, http.MethodGet, "/api/v1/share/not-a-valid-id", nil)
	assertStatus(t, w.Code, http.StatusNotFound)
}
