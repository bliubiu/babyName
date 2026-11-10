package services

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"name/internal/infrastructure/database"
)

// fakeShareStore 只实现 ShareStore，避免服务层测试牵扯整套 database.Store。
type fakeShareStore struct {
	records map[string]*database.ShareRecord
}

func (f *fakeShareStore) SaveShare(record *database.ShareRecord) error {
	if _, exists := f.records[record.ID]; exists {
		return nil
	}
	copied := *record
	f.records[record.ID] = &copied
	return nil
}

func (f *fakeShareStore) GetShare(id string) (*database.ShareRecord, error) {
	record := f.records[id]
	if record == nil {
		return nil, nil
	}
	copied := *record
	return &copied, nil
}

func newTestShareService() *ShareService {
	return NewShareService(&fakeShareStore{records: make(map[string]*database.ShareRecord)})
}

func TestShareService_CreateAndGet(t *testing.T) {
	svc := newTestShareService()
	payload := json.RawMessage(`{"surname":"李","names":[{"full_name":"李沐宸"}],"total":60}`)
	id, err := svc.Create(context.Background(), payload)
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}
	if !shareIDPattern.MatchString(id) {
		t.Fatalf("分享 id 不是 32 位随机 hex: %q", id)
	}
	got, err := svc.Get(context.Background(), id)
	if err != nil {
		t.Fatalf("Get 失败: %v", err)
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(got, &obj); err != nil {
		t.Fatalf("快照不是 JSON 对象: %v", err)
	}
	if string(obj["surname"]) != `"李"` {
		t.Errorf("姓氏未保存: %s", obj["surname"])
	}
	if _, ok := obj["names"]; !ok {
		t.Error("名字列表未保存")
	}
}

func TestShareService_DropsSensitiveFields(t *testing.T) {
	svc := newTestShareService()
	// 客户端可能把整份表单/完整结果直接塞进来；服务端必须按白名单裁剪。
	payload := json.RawMessage(`{
		"surname":"王",
		"birth_location":"北京市朝阳区",
		"avoid_elder_names":["王建国"],
		"bazi":{"year_ganzhi":"丙午"},
		"names":[{"full_name":"王泽"}]
	}`)
	id, err := svc.Create(context.Background(), payload)
	if err != nil {
		t.Fatalf("Create 失败: %v", err)
	}
	got, err := svc.Get(context.Background(), id)
	if err != nil {
		t.Fatalf("Get 失败: %v", err)
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(got, &obj); err != nil {
		t.Fatalf("快照解析失败: %v", err)
	}
	for _, key := range []string{"birth_location", "avoid_elder_names", "bazi"} {
		if _, leaked := obj[key]; leaked {
			t.Errorf("敏感/非白名单字段 %q 被写入分享快照", key)
		}
	}
	if _, ok := obj["surname"]; !ok || obj["names"] == nil {
		t.Error("白名单字段被误删")
	}
}

func TestShareService_RejectsInvalidPayload(t *testing.T) {
	svc := newTestShareService()
	cases := []struct {
		name    string
		payload string
	}{
		{"空内容", ``},
		{"非对象", `[1,2,3]`},
		{"非法 JSON", `{"surname":`},
		{"只有非白名单字段", `{"birth_location":"某地址"}`},
		{"超出大小上限", `{"surname":"` + strings.Repeat("x", maxSharePayloadBytes+1) + `"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Create(context.Background(), json.RawMessage(tc.payload))
			if !errors.Is(err, ErrInvalidShare) {
				t.Errorf("期望 ErrInvalidShare，实际 %v", err)
			}
		})
	}
}

func TestShareService_GetRejectsBadID(t *testing.T) {
	svc := newTestShareService()
	for _, id := range []string{"", "abc", "../../etc/passwd", strings.Repeat("a", 33)} {
		if _, err := svc.Get(context.Background(), id); !errors.Is(err, ErrShareNotFound) {
			t.Errorf("id=%q 期望 ErrShareNotFound，实际 %v", id, err)
		}
	}
	if _, err := svc.Get(context.Background(), strings.Repeat("a", 32)); !errors.Is(err, ErrShareNotFound) {
		t.Errorf("不存在的合法 id 期望 ErrShareNotFound，实际 %v", err)
	}
}
