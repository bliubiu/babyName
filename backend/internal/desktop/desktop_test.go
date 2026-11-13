package desktop

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	apperrors "name/internal/application/errors"
	"name/internal/application/services"
	"name/internal/domain/namestat"
)

// --- 假实现（测试替身） ---

// fakeNameService 名字服务替身：记录入参并返回预设结果
type fakeNameService struct {
	gotGenerate *services.GenerateRequest
	gotEvaluate *services.EvaluateRequest
	gotExplore  *services.ExploreRequest

	genResp     *services.GenerateResponse
	evalResp    *services.EvaluateResponse
	exploreResp *services.ExploreResponse
	err         error
}

func (f *fakeNameService) Generate(_ context.Context, req *services.GenerateRequest) (*services.GenerateResponse, error) {
	f.gotGenerate = req
	if f.err != nil {
		return nil, f.err
	}
	return f.genResp, nil
}

func (f *fakeNameService) Evaluate(_ context.Context, req *services.EvaluateRequest) (*services.EvaluateResponse, error) {
	f.gotEvaluate = req
	if f.err != nil {
		return nil, f.err
	}
	return f.evalResp, nil
}

func (f *fakeNameService) ExploreNames(_ context.Context, req *services.ExploreRequest) (*services.ExploreResponse, error) {
	f.gotExplore = req
	if f.err != nil {
		return nil, f.err
	}
	return f.exploreResp, nil
}

// fakeTaskView 任务快照替身
type fakeTaskView struct {
	status  services.TaskStatus
	stage   string
	percent float64
	errMsg  string
	result  *services.GenerateResponse
}

func (f *fakeTaskView) View() (services.TaskStatus, string, float64, string) {
	return f.status, f.stage, f.percent, f.errMsg
}

func (f *fakeTaskView) TakeResult() *services.GenerateResponse {
	return f.result
}

// fakeTaskRunner 任务服务替身
type fakeTaskRunner struct {
	gotSubmitReq *services.GenerateRequest
	gotSubmitSID string
	gotGetID     string
	gotGetSID    string

	taskID string
	subErr error
	view   TaskView
	ok     bool
}

func (f *fakeTaskRunner) Submit(req *services.GenerateRequest, sid string) (string, error) {
	f.gotSubmitReq = req
	f.gotSubmitSID = sid
	if f.subErr != nil {
		return "", f.subErr
	}
	return f.taskID, nil
}

func (f *fakeTaskRunner) Get(id, sid string) (TaskView, bool) {
	f.gotGetID = id
	f.gotGetSID = sid
	return f.view, f.ok
}

// fakeHistoryService 历史服务替身
type fakeHistoryService struct {
	gotSave      *services.HistoryRecord
	gotPageSID   string
	gotPage      int
	gotLimit     int
	gotDeleteSID string
	gotDeleteID  string

	records []*services.HistoryRecord
	total   int
	id      string
	err     error
}

func (f *fakeHistoryService) SaveHistory(_ context.Context, record *services.HistoryRecord) (string, error) {
	f.gotSave = record
	if f.err != nil {
		return "", f.err
	}
	return f.id, nil
}

func (f *fakeHistoryService) GetHistory(context.Context, string) ([]*services.HistoryRecord, error) {
	return nil, nil
}

func (f *fakeHistoryService) GetHistoryPage(_ context.Context, sid string, page, limit int) ([]*services.HistoryRecord, int, error) {
	f.gotPageSID = sid
	f.gotPage = page
	f.gotLimit = limit
	if f.err != nil {
		return nil, 0, f.err
	}
	return f.records, f.total, nil
}

func (f *fakeHistoryService) GetHistoryRecord(context.Context, string, string) (*services.HistoryRecord, error) {
	return nil, nil
}

func (f *fakeHistoryService) DeleteHistory(_ context.Context, sid, id string) error {
	f.gotDeleteSID = sid
	f.gotDeleteID = id
	return f.err
}

// fakeFavoriteService 收藏服务替身
type fakeFavoriteService struct {
	gotSave    *services.FavoriteRecord
	gotPageSID string
	gotPage    int
	gotLimit   int
	gotDelSID  string
	gotDelID   string

	records []*services.FavoriteRecord
	total   int
	id      string
	err     error
}

func (f *fakeFavoriteService) GetFavorites(context.Context, string) ([]*services.FavoriteRecord, error) {
	return nil, nil
}

func (f *fakeFavoriteService) GetFavoritesPage(_ context.Context, sid string, page, limit int) ([]*services.FavoriteRecord, int, error) {
	f.gotPageSID = sid
	f.gotPage = page
	f.gotLimit = limit
	return f.records, f.total, f.err
}

func (f *fakeFavoriteService) SaveFavorite(_ context.Context, record *services.FavoriteRecord) (string, error) {
	f.gotSave = record
	if f.err != nil {
		return "", f.err
	}
	return f.id, nil
}

func (f *fakeFavoriteService) DeleteFavorite(_ context.Context, sid, id string) error {
	f.gotDelSID = sid
	f.gotDelID = id
	return f.err
}

// fakeReportService 报告服务替身
type fakeReportService struct {
	gotData interface{}
	pdf     []byte
	err     error
}

func (f *fakeReportService) GeneratePDF(_ context.Context, data interface{}) ([]byte, error) {
	f.gotData = data
	return f.pdf, f.err
}

// fakeShareStore 分享服务替身
type fakeShareStore struct {
	gotCreatePayload json.RawMessage
	gotGetID         string

	id      string
	getErr  error
	payload json.RawMessage
}

func (f *fakeShareStore) Create(_ context.Context, payload json.RawMessage) (string, error) {
	f.gotCreatePayload = payload
	return f.id, nil
}

func (f *fakeShareStore) Get(_ context.Context, id string) (json.RawMessage, error) {
	f.gotGetID = id
	return f.payload, f.getErr
}

// fakeCharacterSource 偏旁分组数据源替身
type fakeCharacterSource struct {
	groups []CharGroup
	err    error
}

func (f *fakeCharacterSource) CharGroups() ([]CharGroup, error) {
	return f.groups, f.err
}

// fakeNameStatSource 名字统计源替身
type fakeNameStatSource struct {
	gotStatsName  string
	gotTopLimit   int
	gotTopKind    string
	stats         *namestat.NameStat
	top           []namestat.NameStat
}

func (f *fakeNameStatSource) NameStats(name string) *namestat.NameStat {
	f.gotStatsName = name
	return f.stats
}

func (f *fakeNameStatSource) TopNames(limit int, kind string) []namestat.NameStat {
	f.gotTopLimit = limit
	f.gotTopKind = kind
	return f.top
}

// --- 构造 App 的公共辅助 ---

// validGenerateReq 合法生成请求（通过 validateGenerateFields 校验的最小入参）
func validGenerateReq() services.GenerateRequest {
	return services.GenerateRequest{
		Surname:    "李",
		Gender:     "male",
		BirthYear:  2000,
		BirthMonth: 1,
		BirthDay:   1,
		BirthHour:  10,
		BirthMinute: 30,
	}
}

func newTestApp() (*App, *fakeNameService, *fakeTaskRunner, *fakeHistoryService, *fakeFavoriteService, *fakeReportService, *fakeShareStore) {
	names := &fakeNameService{}
	tasks := &fakeTaskRunner{}
	history := &fakeHistoryService{}
	favorite := &fakeFavoriteService{}
	report := &fakeReportService{}
	shares := &fakeShareStore{}
	app := New(names, tasks, history, favorite, report, shares,
		&fakeCharacterSource{}, &fakeNameStatSource{})
	return app, names, tasks, history, favorite, report, shares
}

// --- 名字生成 ---

func TestGenerateNames_透传服务结果并去除姓氏空格(t *testing.T) {
	app, names, _, _, _, _, _ := newTestApp()
	resp := &services.GenerateResponse{}
	names.genResp = resp

	req := validGenerateReq()
	req.Surname = " 李 "
	got, err := app.GenerateNames(req)
	if err != nil {
		t.Fatalf("生成名字失败: %v", err)
	}
	if got != resp {
		t.Fatalf("期望返回服务层结果指针，实际: %v", got)
	}
	if names.gotGenerate == nil || names.gotGenerate.Surname != "李" {
		t.Fatalf("姓氏应去除首尾空格后透传，实际: %+v", names.gotGenerate)
	}
}

func TestGenerateNames_业务错误透传原消息(t *testing.T) {
	app, names, _, _, _, _, _ := newTestApp()
	names.err = apperrors.BadRequest("无法生成符合条件的名字，请调整筛选条件")

	_, err := app.GenerateNames(validGenerateReq())
	if err == nil || err.Error() != "无法生成符合条件的名字，请调整筛选条件" {
		t.Fatalf("业务错误应透传可行动消息，实际: %v", err)
	}
}

func TestGenerateNames_超时映射为重试提示(t *testing.T) {
	app, names, _, _, _, _, _ := newTestApp()
	names.err = context.DeadlineExceeded

	_, err := app.GenerateNames(validGenerateReq())
	if err == nil || err.Error() != "请求超时，请稍后重试" {
		t.Fatalf("超时应映射为重试提示，实际: %v", err)
	}
}

func TestGenerateNames_未知错误映射为兜底文案(t *testing.T) {
	app, names, _, _, _, _, _ := newTestApp()
	names.err = errors.New("内部细节不回传")

	_, err := app.GenerateNames(validGenerateReq())
	if err == nil || err.Error() != "生成名字失败，请稍后重试" {
		t.Fatalf("未知错误应映射为兜底文案，实际: %v", err)
	}
}

func TestGenerateNames_姓氏为空返回校验错误(t *testing.T) {
	app, _, _, _, _, _, _ := newTestApp()

	if _, err := app.GenerateNames(services.GenerateRequest{Gender: "male"}); err == nil {
		t.Fatal("空姓氏应返回校验错误")
	}
}

// --- 测名 ---

func TestEvaluateName_透传并去空格(t *testing.T) {
	app, names, _, _, _, _, _ := newTestApp()
	resp := &services.EvaluateResponse{}
	names.evalResp = resp

	got, err := app.EvaluateName(services.EvaluateRequest{
		Surname:    "李",
		GivenName:  " 思远 ",
		Gender:     "male",
		BirthYear:  2000,
		BirthMonth: 1,
		BirthDay:   1,
		BirthHour:  10,
		BirthMinute: 30,
	})
	if err != nil {
		t.Fatalf("测名失败: %v", err)
	}
	if got != resp {
		t.Fatalf("期望返回服务层结果指针，实际: %v", got)
	}
	if names.gotEvaluate == nil || names.gotEvaluate.GivenName != "思远" {
		t.Fatalf("名字应去除首尾空格后透传，实际: %+v", names.gotEvaluate)
	}
}

// --- 异步任务 ---

func TestSubmitGenerateTask_返回任务ID并携带桌面会话(t *testing.T) {
	app, _, tasks, _, _, _, _ := newTestApp()
	tasks.taskID = "t1234"

	got, err := app.SubmitGenerateTask(validGenerateReq())
	if err != nil {
		t.Fatalf("提交任务失败: %v", err)
	}
	if got.TaskID != "t1234" {
		t.Fatalf("期望 task_id=t1234，实际: %s", got.TaskID)
	}
	if tasks.gotSubmitSID != desktopSessionID {
		t.Fatalf("应使用桌面端固定会话，实际: %s", tasks.gotSubmitSID)
	}
}

func TestSubmitGenerateTask_满载映射为稍后重试(t *testing.T) {
	app, _, tasks, _, _, _, _ := newTestApp()
	tasks.subErr = services.ErrTooManyTasks

	_, err := app.SubmitGenerateTask(validGenerateReq())
	if err == nil || err.Error() != "当前生成任务过多，请稍后重试" {
		t.Fatalf("满载应映射为稍后重试提示，实际: %v", err)
	}
}

func TestGetGenerateTask_成功时附完整结果(t *testing.T) {
	app, _, tasks, _, _, _, _ := newTestApp()
	result := &services.GenerateResponse{}
	tasks.view = &fakeTaskView{status: services.TaskStatusSuccess, stage: "完成", percent: 100, result: result}
	tasks.ok = true

	got, err := app.GetGenerateTask("t1234")
	if err != nil {
		t.Fatalf("查询任务失败: %v", err)
	}
	if got.Status != services.TaskStatusSuccess || got.Result != result {
		t.Fatalf("成功任务应附带完整结果，实际: %+v", got)
	}
	if tasks.gotGetID != "t1234" || tasks.gotGetSID != desktopSessionID {
		t.Fatalf("查询应携带任务ID与桌面会话，实际: id=%s sid=%s", tasks.gotGetID, tasks.gotGetSID)
	}
}

func TestGetGenerateTask_失败时附错误信息(t *testing.T) {
	app, _, tasks, _, _, _, _ := newTestApp()
	tasks.view = &fakeTaskView{status: services.TaskStatusFailed, errMsg: "生成失败"}
	tasks.ok = true

	got, err := app.GetGenerateTask("t1234")
	if err != nil {
		t.Fatalf("查询任务失败: %v", err)
	}
	if got.Err != "生成失败" {
		t.Fatalf("失败任务应附带错误信息，实际: %q", got.Err)
	}
}

func TestGetGenerateTask_任务不存在返回错误(t *testing.T) {
	app, _, tasks, _, _, _, _ := newTestApp()
	tasks.ok = false

	if _, err := app.GetGenerateTask("t-none"); err == nil || err.Error() != "任务不存在或已过期，请重新提交" {
		t.Fatalf("任务不存在应返回明确错误，实际: %v", err)
	}
}

// --- 探索模式 ---

func TestExploreNames_透传(t *testing.T) {
	app, names, _, _, _, _, _ := newTestApp()
	resp := &services.ExploreResponse{}
	names.exploreResp = resp

	got, err := app.ExploreNames(services.ExploreRequest{})
	if err != nil {
		t.Fatalf("换一批失败: %v", err)
	}
	if got != resp {
		t.Fatalf("期望返回服务层结果指针，实际: %v", got)
	}
}

// --- 历史记录 ---

func TestGetHistory_默认分页与会话(t *testing.T) {
	app, _, _, history, _, _, _ := newTestApp()
	history.records = []*services.HistoryRecord{{ID: "h1"}}
	history.total = 1

	got, err := app.GetHistory(0, 0)
	if err != nil {
		t.Fatalf("查询历史失败: %v", err)
	}
	if len(got.Records) != 1 || got.Total != 1 {
		t.Fatalf("期望 1 条记录，实际: %+v", got)
	}
	if got.Page != 1 || got.Limit != maxPageSize {
		t.Fatalf("缺省分页应为 page=1 limit=100，实际: page=%d limit=%d", got.Page, got.Limit)
	}
	if history.gotPageSID != desktopSessionID {
		t.Fatalf("应使用桌面端固定会话，实际: %s", history.gotPageSID)
	}
}

func TestGetHistory_分页参数非法返回错误(t *testing.T) {
	app, _, _, _, _, _, _ := newTestApp()

	if _, err := app.GetHistory(0, -1); err == nil || err.Error() != "limit 必须是正整数" {
		t.Fatalf("非法 limit 应返回明确错误，实际: %v", err)
	}
	if _, err := app.GetHistory(-1, 10); err == nil || err.Error() != "page 必须是正整数" {
		t.Fatalf("非法 page 应返回明确错误，实际: %v", err)
	}
	if _, err := app.GetHistory(1, 101); err == nil || err.Error() != "limit 不能超过 100" {
		t.Fatalf("超限 limit 应返回明确错误，实际: %v", err)
	}
}

func TestSaveHistory_归属桌面会话(t *testing.T) {
	app, _, _, history, _, _, _ := newTestApp()
	history.id = "h1"

	got, err := app.SaveHistory(services.HistoryRecord{Surname: "李"})
	if err != nil {
		t.Fatalf("保存历史失败: %v", err)
	}
	if got.ID != "h1" {
		t.Fatalf("期望返回记录ID h1，实际: %s", got.ID)
	}
	if history.gotSave.SessionID != desktopSessionID {
		t.Fatalf("归属应由服务端固定会话决定，实际: %s", history.gotSave.SessionID)
	}
}

func TestDeleteHistory_透传会话与ID(t *testing.T) {
	app, _, _, history, _, _, _ := newTestApp()

	if _, err := app.DeleteHistory("h1"); err != nil {
		t.Fatalf("删除历史失败: %v", err)
	}
	if history.gotDeleteSID != desktopSessionID || history.gotDeleteID != "h1" {
		t.Fatalf("删除应携带会话与ID，实际: sid=%s id=%s", history.gotDeleteSID, history.gotDeleteID)
	}
}

// --- 收藏 ---

func TestGetFavorites_默认分页(t *testing.T) {
	app, _, _, _, favorite, _, _ := newTestApp()
	favorite.records = []*services.FavoriteRecord{{ID: "f1"}}
	favorite.total = 1

	got, err := app.GetFavorites(0, 0)
	if err != nil {
		t.Fatalf("查询收藏失败: %v", err)
	}
	if len(got.Records) != 1 || got.Total != 1 || got.Page != 1 || got.Limit != maxPageSize {
		t.Fatalf("收藏分页结果不符，实际: %+v", got)
	}
	if favorite.gotPageSID != desktopSessionID {
		t.Fatalf("应使用桌面端固定会话，实际: %s", favorite.gotPageSID)
	}
}

func TestSaveFavorite_非法姓氏与名字(t *testing.T) {
	app, _, _, _, _, _, _ := newTestApp()

	if _, err := app.SaveFavorite(services.FavoriteRecord{}); err == nil {
		t.Fatal("空姓氏应返回校验错误")
	}
	if _, err := app.SaveFavorite(services.FavoriteRecord{Surname: "李"}); err == nil {
		t.Fatal("空名字应返回校验错误")
	}
}

func TestSaveFavorite_合法记录归属桌面会话(t *testing.T) {
	app, _, _, _, favorite, _, _ := newTestApp()
	favorite.id = "f1"

	got, err := app.SaveFavorite(services.FavoriteRecord{Surname: "李", GivenName: "思远"})
	if err != nil {
		t.Fatalf("保存收藏失败: %v", err)
	}
	if got.ID != "f1" {
		t.Fatalf("期望返回记录ID f1，实际: %s", got.ID)
	}
	if favorite.gotSave.SessionID != desktopSessionID {
		t.Fatalf("归属应由服务端固定会话决定，实际: %s", favorite.gotSave.SessionID)
	}
}

func TestDeleteFavorite_透传会话与ID(t *testing.T) {
	app, _, _, _, favorite, _, _ := newTestApp()

	if _, err := app.DeleteFavorite("f1"); err != nil {
		t.Fatalf("删除收藏失败: %v", err)
	}
	if favorite.gotDelSID != desktopSessionID || favorite.gotDelID != "f1" {
		t.Fatalf("删除应携带会话与ID，实际: sid=%s id=%s", favorite.gotDelSID, favorite.gotDelID)
	}
}

// --- PDF 报告 ---

func TestGenerateReportPDF_透传数据与字节(t *testing.T) {
	app, _, _, _, _, report, _ := newTestApp()
	pdf := []byte("%PDF-1.4")
	report.pdf = pdf

	got, err := app.GenerateReportPDF(map[string]any{"name": "李思远"})
	if err != nil {
		t.Fatalf("生成PDF失败: %v", err)
	}
	if string(got) != string(pdf) {
		t.Fatalf("期望返回PDF字节，实际: %x", got)
	}
	if report.gotData == nil {
		t.Fatal("报告数据应透传")
	}
}

func TestGenerateReportPDF_失败映射兜底文案(t *testing.T) {
	app, _, _, _, _, report, _ := newTestApp()
	report.err = errors.New("内部细节")

	if _, err := app.GenerateReportPDF(nil); err == nil || err.Error() != "生成PDF失败" {
		t.Fatalf("PDF失败应映射兜底文案，实际: %v", err)
	}
}

// --- 分享 ---

func TestCreateShare_返回ID与链接(t *testing.T) {
	app, _, _, _, _, _, shares := newTestApp()
	shares.id = "abc123"

	payload := json.RawMessage(`{"surname":"李"}`)
	got, err := app.CreateShare(payload)
	if err != nil {
		t.Fatalf("创建分享失败: %v", err)
	}
	if got.ID != "abc123" {
		t.Fatalf("期望 id=abc123，实际: %s", got.ID)
	}
	if !strings.Contains(got.URL, "abc123") {
		t.Fatalf("链接应包含分享ID，实际: %s", got.URL)
	}
}

func TestGetShare_不存在映射提示(t *testing.T) {
	app, _, _, _, _, _, shares := newTestApp()
	shares.getErr = services.ErrShareNotFound

	if _, err := app.GetShare("nope"); err == nil || err.Error() != "分享不存在或已失效" {
		t.Fatalf("分享不存在应映射提示，实际: %v", err)
	}
}

// --- 黄历/农历（领域层直连） ---

func TestGetHuangli_非法参数(t *testing.T) {
	app, _, _, _, _, _, _ := newTestApp()

	if _, err := app.GetHuangli(2024, 13, 1); err == nil {
		t.Fatal("非法月份应返回错误")
	}
	if _, err := app.GetHuangli(2024, 2, 30); err == nil {
		t.Fatal("不存在的日期（2月30日）应返回错误")
	}
}

func TestGetHuangli_合法日期返回数据(t *testing.T) {
	app, _, _, _, _, _, _ := newTestApp()

	got, err := app.GetHuangli(2024, 2, 15)
	if err != nil {
		t.Fatalf("查询黄历失败: %v", err)
	}
	if got == nil || got.Year == "" || got.Month == "" || got.Day == "" {
		t.Fatalf("黄历数据不完整: %+v", got)
	}
}

func TestGetLunarCalendar_合法日期返回数据(t *testing.T) {
	app, _, _, _, _, _, _ := newTestApp()

	got, err := app.GetLunarCalendar(2024, 2, 15, 10, 30)
	if err != nil {
		t.Fatalf("查询农历失败: %v", err)
	}
	if got == nil || got.DayGan == "" || got.DayZhi == "" {
		t.Fatalf("农历数据不完整: %+v", got)
	}
}

func TestGetLunarMonthCalendar_年份越界返回错误(t *testing.T) {
	app, _, _, _, _, _, _ := newTestApp()

	if _, err := app.GetLunarMonthCalendar(1899, 1); err == nil {
		t.Fatal("年份早于1900应返回错误")
	}
}

// --- 偏旁分组 ---

func TestGetCharGroups_透传(t *testing.T) {
	names := &fakeNameService{}
	chars := &fakeCharacterSource{groups: []CharGroup{{Radical: "木", Name: "树木", Chars: []string{"林"}}}}
	app := New(names, &fakeTaskRunner{}, &fakeHistoryService{}, &fakeFavoriteService{},
		&fakeReportService{}, &fakeShareStore{}, chars, &fakeNameStatSource{})

	got, err := app.GetCharGroups()
	if err != nil {
		t.Fatalf("获取偏旁分组失败: %v", err)
	}
	if len(got) != 1 || got[0].Radical != "木" {
		t.Fatalf("偏旁分组透传失败: %+v", got)
	}
}

// --- 名字统计 ---

func TestGetTopNames_参数规范化(t *testing.T) {
	names := &fakeNameService{}
	stats := &fakeNameStatSource{top: []namestat.NameStat{{}}}
	app := New(names, &fakeTaskRunner{}, &fakeHistoryService{}, &fakeFavoriteService{},
		&fakeReportService{}, &fakeShareStore{}, &fakeCharacterSource{}, stats)

	// 非法 limit 回落默认 20，非法 kind 回落 all
	if _, err := app.GetTopNames(-5, "unknown"); err != nil {
		t.Fatalf("榜单查询失败: %v", err)
	}
	if stats.gotTopLimit != 20 || stats.gotTopKind != "all" {
		t.Fatalf("limit=-5 kind=unknown 应回落 20/all，实际: %d/%s", stats.gotTopLimit, stats.gotTopKind)
	}

	// 上限 100 截断
	if _, err := app.GetTopNames(500, "char"); err != nil {
		t.Fatalf("榜单查询失败: %v", err)
	}
	if stats.gotTopLimit != 100 || stats.gotTopKind != "char" {
		t.Fatalf("limit=500 应回落 100，kind=char 保留，实际: %d/%s", stats.gotTopLimit, stats.gotTopKind)
	}
}

// --- 上下文 ---

func TestStartup_注入Wails上下文(t *testing.T) {
	app, _, _, _, _, _, _ := newTestApp()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	app.Startup(ctx)
	if app.ctx() != ctx {
		t.Fatal("Startup 应注入运行时上下文")
	}
}
