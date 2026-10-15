package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	apperrors "name/internal/application/errors"
	"name/internal/application/services"
)

// name_handler_error_mapping_test.go — writeGenerateError 错误分级守护
//
// 背景（docs/32 模式一验证实测）：空库回落场景下 /names/generate 稳定返回
// HTTP 500，而服务端日志显示真实错误是业务层 [Code: 400]「无法生成符合条件
// 的名字，请调整筛选条件」。根因：writeGenerateError 把非超时错误一律映射
// 为 500 并替换成通用文案，业务层的 400 被吞——前端拿不到可行动提示，
// 监控上多了一批假 500。
//
// 本测试锁定三级语义：超时→503、AppError→透传 Code/Message、未知→500。

// stubNameService 只实现 Generate/Evaluate 两条被測路径，其余方法留空
type stubNameService struct {
	services.NameServiceInterface
	genErr error
	evaErr error
}

func (s *stubNameService) Generate(_ context.Context, _ *services.GenerateRequest) (*services.GenerateResponse, error) {
	if s.genErr != nil {
		return nil, s.genErr
	}
	return &services.GenerateResponse{}, nil
}

func (s *stubNameService) Evaluate(_ context.Context, _ *services.EvaluateRequest) (*services.EvaluateResponse, error) {
	if s.evaErr != nil {
		return nil, s.evaErr
	}
	return &services.EvaluateResponse{}, nil
}

// errBody 解析错误响应体
type errBody struct {
	Code    int    `json:"code"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func postGenerate(t *testing.T, svc services.NameServiceInterface, path string) *httptest.ResponseRecorder {
	t.Helper()
	r := gin.New()
	gin.SetMode(gin.TestMode)
	h := NewNameHandler(svc)
	r.POST(path, h.Generate)
	body := []byte(`{"surname":"王","gender":"male","birth_year":2024,"birth_month":1,"birth_day":15,"birth_hour":12}`)
	req := httptest.NewRequest("POST", path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestWriteGenerateError_业务错误透传400 核心场景：候选池为空时业务层返回
// [Code: 400]，HTTP 层必须透传 400 与原始提示（客户端可行动）
func TestWriteGenerateError_业务错误透传400(t *testing.T) {
	svc := &stubNameService{genErr: apperrors.NewError(apperrors.ErrCodeBadRequest, "无法生成符合条件的名字，请调整筛选条件")}
	w := postGenerate(t, svc, "/names/generate")

	if w.Code != 400 {
		t.Fatalf("业务 400 必须透传为 HTTP 400，实得 %d", w.Code)
	}
	var got errBody
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("解析响应体失败: %v", err)
	}
	if got.Code != 400 || got.Success {
		t.Errorf("响应体期望 code=400 success=false，实得 code=%d success=%v", got.Code, got.Success)
	}
	if got.Message != "无法生成符合条件的名字，请调整筛选条件" {
		t.Errorf("业务提示必须原样透传，实得 %q", got.Message)
	}
}

// TestWriteGenerateError_超时返回503 超时是可重试错误，与业务错误、服务端
// 错误语义互不掩盖
func TestWriteGenerateError_超时返回503(t *testing.T) {
	svc := &stubNameService{genErr: context.DeadlineExceeded}
	w := postGenerate(t, svc, "/names/generate")

	if w.Code != 503 {
		t.Fatalf("超时期望 HTTP 503，实得 %d", w.Code)
	}
	var got errBody
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("解析响应体失败: %v", err)
	}
	if got.Message != "请求超时，请稍后重试" {
		t.Errorf("超时提示文案不符，实得 %q", got.Message)
	}
}

// TestWriteGenerateError_未知错误仍为500 非 AppError 的未知错误按服务端
// 缺陷处理，保持 500 + 通用文案（不透传内部细节）
func TestWriteGenerateError_未知错误仍为500(t *testing.T) {
	svc := &stubNameService{genErr: context.Canceled} // 非 DeadlineExceeded、非 AppError
	w := postGenerate(t, svc, "/names/generate")

	if w.Code != 500 {
		t.Fatalf("未知错误期望 HTTP 500，实得 %d", w.Code)
	}
	var got errBody
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("解析响应体失败: %v", err)
	}
	if got.Message != "生成名字失败，请稍后重试" {
		t.Errorf("未知错误应使用通用文案，实得 %q", got.Message)
	}
}

// TestWriteGenerateError_Evaluate同源 Evaluate 与 Generate 共用
// writeGenerateError，映射行为必须一致（双口径守护）
func TestWriteGenerateError_Evaluate同源(t *testing.T) {
	svc := &stubNameService{evaErr: apperrors.NewError(apperrors.ErrCodeBadRequest, "无法生成符合条件的名字，请调整筛选条件")}
	r := gin.New()
	gin.SetMode(gin.TestMode)
	h := NewNameHandler(svc)
	r.POST("/names/evaluate", h.Evaluate)
	req := httptest.NewRequest("POST", "/names/evaluate",
		bytes.NewReader([]byte(`{"surname":"王","given_name":"浩","gender":"male","birth_year":2024,"birth_month":1,"birth_day":15,"birth_hour":12}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != 400 {
		t.Fatalf("Evaluate 的业务 400 应透传，实得 %d", rec.Code)
	}
	var got errBody
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("解析响应体失败: %v", err)
	}
	if got.Message != "无法生成符合条件的名字，请调整筛选条件" {
		t.Errorf("Evaluate 的业务提示应透传，实得 %q", got.Message)
	}
}

// TestWriteGenerateError_包裹的AppError也可识别 AppError 被 fmt.Errorf
// %w 包裹后，errors.As 仍应识别（防御未来链路加包装层）
func TestWriteGenerateError_包裹的AppError也可识别(t *testing.T) {
	wrapped := apperrors.NewError(apperrors.ErrCodeBadRequest, "无法生成符合条件的名字，请调整筛选条件")
	svc := &stubNameService{genErr: errWrap{wrapped}}
	w := postGenerate(t, svc, "/names/generate")

	if w.Code != 400 {
		t.Fatalf("被包装的业务 400 仍应透传，实得 %d", w.Code)
	}
}

// errWrap 用 %w 语义包装内部错误（手写以避免额外 import fmt 的干扰）
type errWrap struct{ inner error }

func (e errWrap) Error() string { return "包装层: " + e.inner.Error() }
func (e errWrap) Unwrap() error { return e.inner }
