package services

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"name/internal/application/validator"
	"name/internal/infrastructure/database"
)

// mockFavoriteStore 同时实现 FavoriteStore 和 CuratedStore 接口的内存 mock
type mockFavoriteStore struct {
	favorites        []*database.FavoriteRecord
	curatedNames     []database.CuratedNameEntry
	curatedCalls     int     // 记录 SaveCuratedName 调用次数
	lastCuratedScore float64 // 记录最后一次精选名评分
	saveErr          error   // 非 nil 时模拟写库失败（docs/29 A6 回归）
}

func (m *mockFavoriteStore) SaveFavorite(record *database.FavoriteRecord) (string, error) {
	if m.saveErr != nil {
		return "", m.saveErr
	}
	record.ID = "fav-" + record.Surname + record.GivenName
	m.favorites = append(m.favorites, record)
	return record.ID, nil
}

// testSessionID 服务层测试用会话标识（32 位十六进制，满足格式校验）。
const testSessionID = "0123456789abcdef0123456789abcdef"

func (m *mockFavoriteStore) GetFavorites(sessionID string) []*database.FavoriteRecord {
	var out []*database.FavoriteRecord
	for _, f := range m.favorites {
		if f.SessionID == sessionID {
			out = append(out, f)
		}
	}
	return out
}

func (m *mockFavoriteStore) GetFavoritesPage(sessionID string, page, limit int) ([]*database.FavoriteRecord, int, error) {
	out := m.GetFavorites(sessionID)
	return out, len(out), nil
}

func (m *mockFavoriteStore) DeleteFavorite(sessionID, id string) error {
	for i, f := range m.favorites {
		if f.ID == id && f.SessionID == sessionID {
			m.favorites = append(m.favorites[:i], m.favorites[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *mockFavoriteStore) BatchDeleteFavorite(sessionID string, ids []string) error {
	for _, id := range ids {
		m.DeleteFavorite(sessionID, id)
	}
	return nil
}

func (m *mockFavoriteStore) GetFavoriteByName(sessionID, surname, givenName string) *database.FavoriteRecord {
	for _, f := range m.favorites {
		if f.SessionID == sessionID && f.Surname == surname && f.GivenName == givenName {
			return f
		}
	}
	return nil
}

// --- CuratedStore 实现 ---

func (m *mockFavoriteStore) SaveCuratedName(name, pinyin, gender string, score float64, source string) error {
	m.curatedCalls++
	m.lastCuratedScore = score
	m.curatedNames = append(m.curatedNames, database.CuratedNameEntry{
		Name: name, Pinyin: pinyin, Gender: gender, Score: score, Source: source,
	})
	return nil
}

func (m *mockFavoriteStore) LoadAllCuratedNames() ([]database.CuratedNameEntry, error) {
	return m.curatedNames, nil
}

func (m *mockFavoriteStore) IsCurated(name string) (bool, error) {
	for _, c := range m.curatedNames {
		if c.Name == name {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockFavoriteStore) DeleteCuratedName(name string) error {
	for i, c := range m.curatedNames {
		if c.Name == name {
			m.curatedNames = append(m.curatedNames[:i], m.curatedNames[i+1:]...)
			return nil
		}
	}
	return nil
}

// TestSaveFavorite_LowScore 评分<80 不触发自学习精选库
func TestSaveFavorite_LowScore(t *testing.T) {
	store := &mockFavoriteStore{}
	svc := NewFavoriteService(store)

	id, err := svc.SaveFavorite(context.Background(), &FavoriteRecord{
		SessionID: testSessionID,
		Surname:   "张",
		GivenName: "三",
		Pinyin:    "zhang san",
		Gender:    "male",
		Score:     70,
	})
	if err != nil {
		t.Fatalf("SaveFavorite 返回错误: %v", err)
	}
	if id == "" {
		t.Error("SaveFavorite 返回空 ID")
	}
	if store.curatedCalls != 0 {
		t.Errorf("评分70不应触发精选库保存，但调用了 %d 次", store.curatedCalls)
	}
}

// TestSaveFavorite_HighScore 评分≥80 触发自学习精选库
func TestSaveFavorite_HighScore(t *testing.T) {
	store := &mockFavoriteStore{}
	svc := NewFavoriteService(store)

	id, err := svc.SaveFavorite(context.Background(), &FavoriteRecord{
		SessionID: testSessionID,
		Surname:   "李",
		GivenName: "明轩",
		Pinyin:    "li mingxuan",
		Gender:    "male",
		Score:     85,
	})
	if err != nil {
		t.Fatalf("SaveFavorite 返回错误: %v", err)
	}
	if id == "" {
		t.Error("SaveFavorite 返回空 ID")
	}
	if store.curatedCalls != 1 {
		t.Errorf("评分85应触发1次精选库保存，实际 %d 次", store.curatedCalls)
	}
	if store.lastCuratedScore != 85 {
		t.Errorf("精选名评分应为 85，实际 %v", store.lastCuratedScore)
	}
}

// TestSaveFavorite_Existing 不重复保存已存在的收藏
func TestSaveFavorite_Existing(t *testing.T) {
	store := &mockFavoriteStore{}
	// 预置一条已存在的收藏
	store.favorites = append(store.favorites, &database.FavoriteRecord{
		ID:        "existing-id",
		SessionID: testSessionID,
		Surname:   "王",
		GivenName: "涵",
	})
	svc := NewFavoriteService(store)

	id, err := svc.SaveFavorite(context.Background(), &FavoriteRecord{
		SessionID: testSessionID,
		Surname:   "王",
		GivenName: "涵",
		Score:     90,
	})
	if err != nil {
		t.Fatalf("SaveFavorite 返回错误: %v", err)
	}
	if id != "existing-id" {
		t.Errorf("已存在收藏应返回原 ID \"existing-id\"，实际 %q", id)
	}
	if len(store.favorites) != 1 {
		t.Errorf("不应重复保存，收藏数应为 1，实际 %d", len(store.favorites))
	}
}

// TestDeleteFavorite 删除收藏
func TestDeleteFavorite(t *testing.T) {
	store := &mockFavoriteStore{}
	store.favorites = append(store.favorites, &database.FavoriteRecord{
		ID:        "del-1",
		SessionID: testSessionID,
		Surname:   "钱",
		GivenName: "睿",
	})
	svc := NewFavoriteService(store)

	if err := svc.DeleteFavorite(context.Background(), testSessionID, "del-1"); err != nil {
		t.Fatalf("DeleteFavorite 返回错误: %v", err)
	}
	if len(store.favorites) != 0 {
		t.Errorf("删除后收藏数应为 0，实际 %d", len(store.favorites))
	}
}

// TestFavoriteRecord_JSONDecimalScore 收藏请求体里的浮点评分必须能被解析。
//
// 回归护栏：FavoriteRecord.Score 曾声明为 int，前端原样提交的 total_score
// （如 92.7）会被 encoding/json 判为 UnmarshalTypeError，收藏接口一律返回
// 400「请求参数格式错误」——只有评分恰好整除时才侥幸成功。
func TestFavoriteRecord_JSONDecimalScore(t *testing.T) {
	var rec FavoriteRecord
	body := []byte(`{"surname":"张","given_name":"珀熙","pinyin":"zhang poxi","gender":"male","score":92.7}`)
	if err := json.Unmarshal(body, &rec); err != nil {
		t.Fatalf("浮点 score 解析失败（回归：int 字段会拒绝小数）: %v", err)
	}
	if rec.Score != 92.7 {
		t.Errorf("解析后评分应为 92.7，实际 %v", rec.Score)
	}
}

// TestSaveFavorite_DecimalScore 浮点评分要完整落库并传给自学习精选库
func TestSaveFavorite_DecimalScore(t *testing.T) {
	store := &mockFavoriteStore{}
	svc := NewFavoriteService(store)

	_, err := svc.SaveFavorite(context.Background(), &FavoriteRecord{
		SessionID: testSessionID,
		Surname:   "张",
		GivenName: "珀熙",
		Pinyin:    "zhang poxi",
		Gender:    "male",
		Score:     92.7,
	})
	if err != nil {
		t.Fatalf("浮点评分保存失败: %v", err)
	}
	if len(store.favorites) != 1 {
		t.Fatalf("收藏记录数应为 1，实际 %d", len(store.favorites))
	}
	if store.favorites[0].Score != 92.7 {
		t.Errorf("落库评分应保留小数 92.7，实际 %v", store.favorites[0].Score)
	}
	// ≥80 应触发自学习，且传下去的是原始浮点而不是取整后的值
	if store.curatedCalls != 1 {
		t.Errorf("评分 92.7 应触发 1 次精选库保存，实际 %d 次", store.curatedCalls)
	}
	if store.lastCuratedScore != 92.7 {
		t.Errorf("精选名评分应为 92.7，实际 %v", store.lastCuratedScore)
	}
}

// TestSaveFavorite_WriteFailurePropagates 写库失败必须上抛且不进自学习
// （docs/29 A6 回归：此前吞错仍返回 ID + nil error，收藏静默丢失、
// 精选库却被污染，形成「收藏失败但进了精选库」的不一致）
func TestSaveFavorite_WriteFailurePropagates(t *testing.T) {
	store := &mockFavoriteStore{
		saveErr: errors.New("disk I/O error"),
	}
	svc := NewFavoriteService(store)

	id, err := svc.SaveFavorite(context.Background(), &FavoriteRecord{
		SessionID: testSessionID,
		Surname:   "李",
		GivenName: "明轩",
		Pinyin:    "li mingxuan",
		Gender:    "male",
		Score:     95.0,
	})
	if err == nil {
		t.Fatal("写库失败时应返回错误，实际返回 nil")
	}
	if id != "" {
		t.Errorf("写库失败时不应返回有效 ID，实际 %q", id)
	}
	if len(store.favorites) != 0 {
		t.Errorf("失败的收藏不应进入存储，实际 %d 条", len(store.favorites))
	}
	// 关键：收藏失败不得触发自学习
	if store.curatedCalls != 0 {
		t.Errorf("收藏失败时不应调用精选库保存，实际 %d 次", store.curatedCalls)
	}
}

// mockReviewRecorder 记录门禁复审回流调用（docs/33 M6）
type mockReviewRecorder struct {
	calls [][2]string
}

func (m *mockReviewRecorder) RecordFavoriteCandidate(surname, givenName string) {
	m.calls = append(m.calls, [2]string{surname, givenName})
}

// TestSaveFavorite_触发门禁复审回流 收藏成功后应把名字登记为复审候选
func TestSaveFavorite_触发门禁复审回流(t *testing.T) {
	svc := NewFavoriteService(&mockFavoriteStore{})
	rec := &mockReviewRecorder{}
	svc.SetGateReviewRecorder(rec)

	if _, err := svc.SaveFavorite(context.Background(), &FavoriteRecord{
		SessionID: testSessionID,
		Surname:   "张", GivenName: "珀熙", Score: 90,
	}); err != nil {
		t.Fatalf("SaveFavorite 出错: %v", err)
	}
	if len(rec.calls) != 1 || rec.calls[0] != [2]string{"张", "珀熙"} {
		t.Fatalf("应回流一次收藏候选，实得 %+v", rec.calls)
	}
}

// TestSaveFavorite_未注入记录器不panic 未注入记录器时收藏应正常
func TestSaveFavorite_未注入记录器不panic(t *testing.T) {
	svc := NewFavoriteService(&mockFavoriteStore{})
	if _, err := svc.SaveFavorite(context.Background(), &FavoriteRecord{
		SessionID: testSessionID,
		Surname:   "李", GivenName: "明轩", Score: 85,
	}); err != nil {
		t.Fatalf("未注入记录器时收藏应正常: %v", err)
	}
}

// TestSaveFavorite_已存在不回流 重复收藏（返回原 ID）不应再次回流
func TestSaveFavorite_已存在不回流(t *testing.T) {
	store := &mockFavoriteStore{}
	store.favorites = append(store.favorites, &database.FavoriteRecord{
		ID: "existing-id", SessionID: testSessionID, Surname: "王", GivenName: "涵",
	})
	svc := NewFavoriteService(store)
	rec := &mockReviewRecorder{}
	svc.SetGateReviewRecorder(rec)

	id, err := svc.SaveFavorite(context.Background(), &FavoriteRecord{
		SessionID: testSessionID,
		Surname:   "王", GivenName: "涵", Score: 90,
	})
	if err != nil {
		t.Fatalf("SaveFavorite 出错: %v", err)
	}
	if id != "existing-id" {
		t.Fatalf("应返回已存在 ID，实得 %q", id)
	}
	if len(rec.calls) != 0 {
		t.Fatalf("已存在收藏不应回流，实得 %+v", rec.calls)
	}
}

// TestSaveFavorite_写库失败不回流 收藏失败不得登记复审候选
func TestSaveFavorite_写库失败不回流(t *testing.T) {
	store := &mockFavoriteStore{saveErr: errors.New("disk I/O error")}
	svc := NewFavoriteService(store)
	rec := &mockReviewRecorder{}
	svc.SetGateReviewRecorder(rec)

	if _, err := svc.SaveFavorite(context.Background(), &FavoriteRecord{
		SessionID: testSessionID,
		Surname:   "李", GivenName: "明轩", Score: 95,
	}); err == nil {
		t.Fatal("写库失败应返回错误")
	}
	if len(rec.calls) != 0 {
		t.Fatalf("收藏失败不应回流，实得 %+v", rec.calls)
	}
}

// TestValidateFavoriteEntry 收藏入库防线回归：SSR 表单曾无校验，gender/
// score/pinyin/notes 任意值直达库；score 无界会污染自学习精选库。
func TestValidateFavoriteEntry(t *testing.T) {
	valid := &FavoriteRecord{Surname: "王", GivenName: "涵", Gender: "男", Score: 88.5}
	if err := validateFavoriteEntry(valid); err != nil {
		t.Fatalf("合法记录不应报错: %v", err)
	}
	if valid.Gender != "male" {
		t.Errorf("性别应归一为 male，实得 %q", valid.Gender)
	}

	tests := []struct {
		name string
		rec  *FavoriteRecord
		want string
	}{
		{"空记录", nil, "收藏记录不能为空"},
		{"空姓氏", &FavoriteRecord{GivenName: "涵"}, "姓氏不能为空"},
		{"空名字", &FavoriteRecord{Surname: "王"}, "名字不能为空"},
		{"非法性别", &FavoriteRecord{Surname: "王", GivenName: "涵", Gender: "外星人"}, "性别必须是male或female"},
		{"评分上限", &FavoriteRecord{Surname: "王", GivenName: "涵", Score: 999}, "评分必须在0-100之间"},
		{"评分下限", &FavoriteRecord{Surname: "王", GivenName: "涵", Score: -1}, "评分必须在0-100之间"},
		{"拼音超长", &FavoriteRecord{Surname: "王", GivenName: "涵", Pinyin: strings.Repeat("p", 101)}, "拼音长度不能超过100个字符"},
		{"来源超长", &FavoriteRecord{Surname: "王", GivenName: "涵", Source: strings.Repeat("s", 65)}, "来源标记长度不能超过64个字符"},
		{"备注超长", &FavoriteRecord{Surname: "王", GivenName: "涵", Notes: strings.Repeat("备", 201)}, "备注长度不能超过200个字符"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFavoriteEntry(tt.rec)
			if err == nil {
				t.Fatalf("%s 期望报错，实际 nil", tt.name)
			}
			if ve, ok := err.(*validator.ValidationError); ok {
				if ve.Message != tt.want {
					t.Errorf("文案 %q ≠ %q", ve.Message, tt.want)
				}
			}
		})
	}
}

// TestSaveFavorite_评分越界不触发自学习 重新验证集成路径：越界评分先被
// 服务层拦截，且精选库（score≥80 触发器）前必须先入库校验。
func TestSaveFavorite_评分越界不触发自学习(t *testing.T) {
	store := &mockFavoriteStore{}
	svc := NewFavoriteService(store)
	_, err := svc.SaveFavorite(context.Background(), &FavoriteRecord{
		SessionID: testSessionID,
		Surname:   "王", GivenName: "涵", Score: 999,
	})
	if err == nil {
		t.Fatal("评分 999 应被拦截")
	}
	if store.curatedCalls != 0 {
		t.Fatalf("拦截点在精选库之前，不应触发回流，实得 %d 次", store.curatedCalls)
	}
	if len(store.favorites) != 0 {
		t.Fatalf("脏记录不应写入收藏库，实得 %d 条", len(store.favorites))
	}
}
