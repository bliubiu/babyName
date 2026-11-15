// Package desktop 桌面端（Wails）绑定层
//
// 定位与 HTTP handler 同层：请求校验 + 会话归属 + 服务编排，但不依赖 gin。
// 前端经 Wails 生成的 TS 绑定直接调用本包方法，替代原 /api/v1 REST 接口。
//
// 错误约定（与 writeServiceError 同口径）：Wails 把返回的 error 以字符串
// 传给前端，因此此处必须完成「错误到用户可读文案」的映射——超时→重试提示、
// 业务 AppError→透传原消息、其余→兜底文案，内部细节绝不回传前端。
package desktop

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"
	apperrors "name/internal/application/errors"
	"name/internal/application/services"
	"name/internal/application/validator"
	"name/internal/domain/bazi"
	"name/internal/domain/hanzi"
	"name/internal/domain/namestat"
	"name/internal/infrastructure/logger"
)

// desktopSessionID 桌面端固定会话标识
//
// Web 端会话由 cookie（nm_sid）隔离；桌面为单用户应用，收藏/历史/异步任务
// 的归属统一使用该固定会话。必须跨重启稳定，否则历史与收藏查不到。
//
// 取值不是 "desktop" 字面量：服务层 NormalizeSessionID 只收 16-64 位十六进制
// （HTTP 时代遗留校验，字面量会被拒收——收藏/历史全链路报"会话标识无效"），
// 故取 "desktop" 的 SHA-256 十六进制（64 位）：语义仍是固定会话，跨重启稳定。
var desktopSessionID = func() string {
	sum := sha256.Sum256([]byte("desktop"))
	return hex.EncodeToString(sum[:])
}()

// maxPageSize 单页最大条数（与 handlers 同口径，docs/29 B6）
const maxPageSize = 100

// --- 窄接口（便于测试替身，具体类型见底部适配器） ---

// TaskView 一次异步生成任务的只读视图
type TaskView interface {
	View() (status services.TaskStatus, stage string, percent float64, errMsg string)
	TakeResult() *services.GenerateResponse
}

// TaskRunner 异步生成任务管理
type TaskRunner interface {
	Submit(req *services.GenerateRequest, sid string) (string, error)
	Get(id, sid string) (TaskView, bool)
}

// ShareStore 分享快照存取
type ShareStore interface {
	Create(ctx context.Context, payload json.RawMessage) (string, error)
	Get(ctx context.Context, id string) (json.RawMessage, error)
}

// CharGroup 标准起名用字分组（与 handlers.StandardCharGroup 同构）
type CharGroup struct {
	Radical string   `json:"radical"`
	Name    string   `json:"name"`
	Meaning string   `json:"meaning"`
	Chars   []string `json:"chars"`
}

// CharacterSource 偏旁分组数据源
type CharacterSource interface {
	CharGroups() ([]CharGroup, error)
}

// NameStatSource 名字统计数据源
type NameStatSource interface {
	NameStats(name string) *namestat.NameStat
	TopNames(limit int, kind string) []namestat.NameStat
}

// --- 物料管理（状态查看 + 手动重导，SQLite 专属） ---

// MaterialState 物料库内状态（对应 sqlite.MaterialState）
type MaterialState struct {
	Material   string `json:"material"`
	Version    string `json:"version"`
	Checksum   string `json:"checksum"`
	RowCount   int    `json:"row_count"`
	ImportedAt string `json:"imported_at"`
}

// MaterialImportItem 单项物料导入结果
type MaterialImportItem struct {
	Material string `json:"material"`
	Version  string `json:"version"`
	Rows     int    `json:"rows"`
	Skipped  bool   `json:"skipped"`
	Err      string `json:"error,omitempty"`
}

// MaterialImportReport 一次重导的汇总报告
type MaterialImportReport struct {
	Items   []MaterialImportItem `json:"items"`
	Written int                  `json:"written"`
	Skipped int                  `json:"skipped"`
	Failed  int                  `json:"failed"`
	Elapsed string               `json:"elapsed,omitempty"`
}

// MaterialManager 物料管理数据源：状态列表 + 重新导入（指纹短路幂等）。
// 内存存储模式下无物料库，由绑定方法给出明确提示而非静默空转。
type MaterialManager interface {
	MaterialStates() ([]MaterialState, error)
	Reimport() (*MaterialImportReport, error)
}

// --- 响应结构（前端 TS 绑定的数据契约） ---

// SubmitTaskResult 提交异步任务的结果
type SubmitTaskResult struct {
	TaskID string `json:"task_id"`
}

// TaskResult 异步任务状态视图（与 name_async_handler 的 gin.H 同构，收敛为类型化结构）
type TaskResult struct {
	TaskID  string                     `json:"task_id"`
	Status  services.TaskStatus        `json:"status"`
	Stage   string                     `json:"stage"`
	Percent float64                    `json:"percent"`
	Err     string                     `json:"error,omitempty"`
	Result  *services.GenerateResponse `json:"result,omitempty"`
}

// HistoryPage 历史记录分页结果
type HistoryPage struct {
	Records []*services.HistoryRecord `json:"records"`
	Total   int                       `json:"total"`
	Page    int                       `json:"page"`
	Limit   int                       `json:"limit"`
}

// FavoritePage 收藏分页结果
type FavoritePage struct {
	Records []*services.FavoriteRecord `json:"records"`
	Total   int                        `json:"total"`
	Page    int                        `json:"page"`
	Limit   int                        `json:"limit"`
}

// RecordID 写入类操作的通用返回：记录ID
type RecordID struct {
	ID string `json:"id"`
}

// Message 操作结果提示
type Message struct {
	Message string `json:"message"`
}

// ShareResult 分享快照结果
type ShareResult struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

// LunarMonthCalendar 公历月的节气与农历节日标注
type LunarMonthCalendar struct {
	Year  int                  `json:"year"`
	Month int                  `json:"month"`
	Days  []bazi.LunarMonthDay `json:"days"`
}

// --- App ---

// App Wails 绑定入口。业务方法一律 (结果, error) 返回，错误已映射为用户可读文案。
type App struct {
	runtimeCtx context.Context

	names      services.NameServiceInterface
	tasks      TaskRunner
	history    services.HistoryServiceInterface
	favorite   services.FavoriteServiceInterface
	report     services.ReportServiceInterface
	shares     ShareStore
	characters CharacterSource
	stats      NameStatSource
	materials  MaterialManager
}

// New 构造桌面端绑定层。参数即装配好的应用服务（装配见 bootstrap）。
// materials 可为 nil（内存存储模式：物料状态页空展示、重导报错提示）。
func New(
	names services.NameServiceInterface,
	tasks TaskRunner,
	history services.HistoryServiceInterface,
	favorite services.FavoriteServiceInterface,
	report services.ReportServiceInterface,
	shares ShareStore,
	characters CharacterSource,
	stats NameStatSource,
	materials MaterialManager,
) *App {
	return &App{
		names:      names,
		tasks:      tasks,
		history:    history,
		favorite:   favorite,
		report:     report,
		shares:     shares,
		characters: characters,
		stats:      stats,
		materials:  materials,
	}
}

// Startup Wails 生命周期回调：注入运行时上下文（服务调用据此传播取消信号）
func (a *App) Startup(ctx context.Context) {
	a.runtimeCtx = ctx
}

// ctx 服务调用上下文：优先运行时上下文，测试/未启动时回落 Background
func (a *App) ctx() context.Context {
	if a.runtimeCtx != nil {
		return a.runtimeCtx
	}
	return context.Background()
}

// --- 错误映射 ---

// toUserError 错误到用户可读文案的统一出口（与 writeServiceError 同口径）
func toUserError(err error, fallback string) error {
	if err == nil {
		return nil
	}
	logger.Error("桌面端调用失败", zap.Error(err))
	if errors.Is(err, context.DeadlineExceeded) {
		return errors.New("请求超时，请稍后重试")
	}
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		return errors.New(appErr.Message)
	}
	if errors.Is(err, services.ErrTooManyTasks) {
		return errors.New(err.Error())
	}
	if errors.Is(err, services.ErrShareNotFound) || errors.Is(err, services.ErrInvalidShare) {
		return errors.New(err.Error())
	}
	return errors.New(fallback)
}

// --- 名字生成/测名 ---

// GenerateRequest 生成请求（services.GenerateRequest 的桌面端别名，保持前端契约单一真源）
type GenerateRequest = services.GenerateRequest

// EvaluateRequest 测名请求
type EvaluateRequest = services.EvaluateRequest

// ExploreRequest 探索模式请求
type ExploreRequest = services.ExploreRequest

// GenerateResponse 生成结果
type GenerateResponse = services.GenerateResponse

// EvaluateResponse 测名结果
type EvaluateResponse = services.EvaluateResponse

// ExploreResponse 换一批结果
type ExploreResponse = services.ExploreResponse

// GenerateNames 生成名字（同步）
func (a *App) GenerateNames(req GenerateRequest) (*GenerateResponse, error) {
	req.Surname = strings.TrimSpace(req.Surname)
	if err := validateGenerateFields(req.Surname, req.Gender, req.BirthYear, req.BirthMonth, req.BirthDay, req.BirthHour, req.BirthMinute); err != nil {
		return nil, err
	}
	result, err := a.names.Generate(a.ctx(), &req)
	if err != nil {
		return nil, toUserError(err, "生成名字失败，请稍后重试")
	}
	return result, nil
}

// EvaluateName 测名：姓名 + 生辰 → 完整评分报告 + 风险体检
func (a *App) EvaluateName(req EvaluateRequest) (*EvaluateResponse, error) {
	req.Surname = strings.TrimSpace(req.Surname)
	req.GivenName = strings.TrimSpace(req.GivenName)
	if err := validateGenerateFields(req.Surname, req.Gender, req.BirthYear, req.BirthMonth, req.BirthDay, req.BirthHour, req.BirthMinute); err != nil {
		return nil, err
	}
	result, err := a.names.Evaluate(a.ctx(), &req)
	if err != nil {
		return nil, toUserError(err, "生成名字失败，请稍后重试")
	}
	return result, nil
}

// ExploreNames 探索模式：与上次生成结果零交集换一批
func (a *App) ExploreNames(req ExploreRequest) (*ExploreResponse, error) {
	result, err := a.names.ExploreNames(a.ctx(), &req)
	if err != nil {
		return nil, toUserError(err, "换一批失败，请稍后重试")
	}
	return result, nil
}

// --- 异步生成任务 ---

// SubmitGenerateTask 提交异步生成任务，立即返回 task_id（前端轮询 GetGenerateTask）
func (a *App) SubmitGenerateTask(req GenerateRequest) (*SubmitTaskResult, error) {
	req.Surname = strings.TrimSpace(req.Surname)
	if err := validateGenerateFields(req.Surname, req.Gender, req.BirthYear, req.BirthMonth, req.BirthDay, req.BirthHour, req.BirthMinute); err != nil {
		return nil, err
	}
	taskID, err := a.tasks.Submit(&req, desktopSessionID)
	if err != nil {
		return nil, toUserError(err, "当前生成任务过多，请稍后重试")
	}
	logger.Info("异步生成：任务已提交", zap.String("task_id", taskID), zap.String("surname", req.Surname))
	return &SubmitTaskResult{TaskID: taskID}, nil
}

// GetGenerateTask 查询异步任务状态；成功时附带完整生成结果（取走即弃）
func (a *App) GetGenerateTask(id string) (*TaskResult, error) {
	view, ok := a.tasks.Get(id, desktopSessionID)
	if !ok {
		return nil, errors.New("任务不存在或已过期，请重新提交")
	}
	status, stage, percent, errMsg := view.View()
	result := &TaskResult{
		TaskID:  id,
		Status:  status,
		Stage:   stage,
		Percent: percent,
	}
	switch status {
	case services.TaskStatusFailed:
		result.Err = errMsg
	case services.TaskStatusSuccess:
		result.Result = view.TakeResult()
	}
	return result, nil
}

// --- 历史记录 ---

// GetHistory 分页获取历史记录（page/limit 非正数回落默认值，limit 上限 100）
func (a *App) GetHistory(page, limit int) (*HistoryPage, error) {
	page, limit, err := normalizePagination(page, limit)
	if err != nil {
		return nil, err
	}
	records, total, err := a.history.GetHistoryPage(a.ctx(), desktopSessionID, page, limit)
	if err != nil {
		logger.Error("查询历史失败", zap.Error(err))
		return nil, errors.New("获取历史记录失败")
	}
	return &HistoryPage{Records: records, Total: total, Page: page, Limit: limit}, nil
}

// SaveHistory 保存历史记录；归属由服务端固定会话决定
func (a *App) SaveHistory(req services.HistoryRecord) (*RecordID, error) {
	req.SessionID = desktopSessionID
	id, err := a.history.SaveHistory(a.ctx(), &req)
	if err != nil {
		logger.Error("保存历史失败", zap.Error(err))
		return nil, errors.New("保存历史记录失败")
	}
	return &RecordID{ID: id}, nil
}

// DeleteHistory 删除历史记录
func (a *App) DeleteHistory(id string) (*Message, error) {
	if err := a.history.DeleteHistory(a.ctx(), desktopSessionID, id); err != nil {
		logger.Error("删除历史失败", zap.Error(err))
		return nil, errors.New("删除历史记录失败")
	}
	return &Message{Message: "删除成功"}, nil
}

// --- 收藏 ---

// GetFavorites 分页获取收藏（page/limit 非正数回落默认值，limit 上限 100）
func (a *App) GetFavorites(page, limit int) (*FavoritePage, error) {
	page, limit, err := normalizePagination(page, limit)
	if err != nil {
		return nil, err
	}
	records, total, err := a.favorite.GetFavoritesPage(a.ctx(), desktopSessionID, page, limit)
	if err != nil {
		logger.Error("查询收藏失败", zap.Error(err))
		return nil, errors.New("获取收藏列表失败")
	}
	return &FavoritePage{Records: records, Total: total, Page: page, Limit: limit}, nil
}

// SaveFavorite 保存收藏；归属由服务端固定会话决定
func (a *App) SaveFavorite(req services.FavoriteRecord) (*RecordID, error) {
	if err := validator.ValidateSurname(req.Surname); err != nil {
		return nil, err
	}
	if err := validator.ValidateGivenName(req.GivenName); err != nil {
		return nil, err
	}
	req.SessionID = desktopSessionID
	id, err := a.favorite.SaveFavorite(a.ctx(), &req)
	if err != nil {
		logger.Error("保存收藏失败", zap.Error(err))
		return nil, errors.New("保存收藏失败")
	}
	return &RecordID{ID: id}, nil
}

// DeleteFavorite 删除收藏
func (a *App) DeleteFavorite(id string) (*Message, error) {
	if err := a.favorite.DeleteFavorite(a.ctx(), desktopSessionID, id); err != nil {
		logger.Error("删除收藏失败", zap.Error(err))
		return nil, errors.New("删除收藏失败")
	}
	return &Message{Message: "删除成功"}, nil
}

// --- PDF 报告 ---

// GenerateReportPDF 生成 PDF 报告，返回 PDF 文件字节
func (a *App) GenerateReportPDF(data interface{}) ([]byte, error) {
	result, err := a.report.GeneratePDF(a.ctx(), data)
	if err != nil {
		logger.Error("PDF 报告生成失败", zap.Error(err))
		return nil, errors.New("生成PDF失败")
	}
	return result, nil
}

// --- 分享快照 ---

// CreateShare 保存展示摘要并返回分享标识
func (a *App) CreateShare(data json.RawMessage) (*ShareResult, error) {
	id, err := a.shares.Create(a.ctx(), data)
	if err != nil {
		return nil, toUserError(err, "创建分享失败")
	}
	// 桌面端为 HashRouter：资产服务器对 /share 这类未知路径返回 404，
	// 分享链接必须写成 /#/share?id=...，否则外部浏览器打开即空白页。
	return &ShareResult{ID: id, URL: "/#/share?id=" + id}, nil
}

// GetShare 按 id 读取分享快照
func (a *App) GetShare(id string) (json.RawMessage, error) {
	payload, err := a.shares.Get(a.ctx(), id)
	if err != nil {
		return nil, toUserError(err, "读取分享失败")
	}
	return payload, nil
}

// --- 黄历/农历（领域层直连，无服务层） ---

// GetHuangli 获取黄历信息
func (a *App) GetHuangli(year, month, day int) (*bazi.Huangli, error) {
	if err := validateYMD(year, month, day); err != nil {
		return nil, err
	}
	defer func() {
		if r := recover(); r != nil {
			logger.Error("黄历查询 panic 已恢复", zap.Any("recover", r))
		}
	}()
	huangli := bazi.GetHuangli(year, month, day)
	if huangli == nil {
		return nil, errors.New("该日期无法换算黄历，请检查年月日")
	}
	return huangli, nil
}

// GetLunarCalendar 获取农历信息
func (a *App) GetLunarCalendar(year, month, day, hour, minute int) (*bazi.LunarCalendar, error) {
	if err := validateYMD(year, month, day); err != nil {
		return nil, err
	}
	if hour < 0 || hour > 23 {
		return nil, errors.New("小时参数错误")
	}
	if minute < 0 || minute > 59 {
		return nil, errors.New("分钟参数错误")
	}
	defer func() {
		if r := recover(); r != nil {
			logger.Error("农历查询 panic 已恢复", zap.Any("recover", r))
		}
	}()
	lunar := bazi.GetLunarCalendar(year, month, day, hour, minute)
	if lunar == nil {
		return nil, errors.New("该日期无法换算农历，请检查年月日")
	}
	return lunar, nil
}

// GetLunarMonthCalendar 获取某公历年月的逐日节气与农历节日标注
func (a *App) GetLunarMonthCalendar(year, month int) (*LunarMonthCalendar, error) {
	if year < 1900 || year > 2100 {
		return nil, errors.New("年份必须在1900-2100之间")
	}
	if month < 1 || month > 12 {
		return nil, errors.New("月份参数错误")
	}
	defer func() {
		if r := recover(); r != nil {
			logger.Error("农历月历查询 panic 已恢复", zap.Any("recover", r))
		}
	}()
	days := bazi.GetLunarMonthCalendar(year, month)
	if days == nil {
		return nil, errors.New("该月份无法换算农历，请检查年月")
	}
	return &LunarMonthCalendar{Year: year, Month: month, Days: days}, nil
}

// --- 汉字/统计（数据源接口，默认实现见 adapters.go） ---

// GetCharGroups 获取所有偏旁分组
func (a *App) GetCharGroups() ([]CharGroup, error) {
	groups, err := a.characters.CharGroups()
	if err != nil {
		logger.Error("加载偏旁分组失败", zap.Error(err))
		return nil, errors.New("加载偏旁分组失败")
	}
	return groups, nil
}

// GetNameStats 获取名字统计信息
func (a *App) GetNameStats(name string) (*namestat.NameStat, error) {
	return a.stats.NameStats(name), nil
}

// GetTopNames 获取热门名字榜单（limit<=0 回落 20，>100 截断为 100；kind 仅支持 char/bigram/all）
func (a *App) GetTopNames(limit int, kind string) ([]namestat.NameStat, error) {
	if limit <= 0 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}
	if kind != "char" && kind != "bigram" {
		kind = "all"
	}
	return a.stats.TopNames(limit, kind), nil
}

// --- 适配器（把具体服务类型接入上方窄接口） ---

// taskRunnerAdapter *services.TaskService → TaskRunner
// （接口方法 Get 返回 TaskView 接口而 *TaskService 返回具体类型，需此桥接）
type taskRunnerAdapter struct{ ts *services.TaskService }

func (t taskRunnerAdapter) Submit(req *services.GenerateRequest, sid string) (string, error) {
	return t.ts.Submit(req, sid)
}

func (t taskRunnerAdapter) Get(id, sid string) (TaskView, bool) {
	task, ok := t.ts.Get(id, sid)
	return task, ok
}

// NewTaskRunner 把具体 *services.TaskService 适配为 TaskRunner
func NewTaskRunner(ts *services.TaskService) TaskRunner {
	return taskRunnerAdapter{ts: ts}
}

// hanziCharacterSource 偏旁分组默认数据源（内存数据，启动装载）
type hanziCharacterSource struct{}

func (hanziCharacterSource) CharGroups() ([]CharGroup, error) {
	groups := hanzi.GetNamerGroups()
	result := make([]CharGroup, 0, len(groups))
	for _, g := range groups {
		result = append(result, CharGroup{
			Radical: g.Radical,
			Name:    g.Name,
			Meaning: g.Meaning,
			Chars:   g.Chars,
		})
	}
	return result, nil
}

// NewCharacterSource 默认偏旁分组数据源
func NewCharacterSource() CharacterSource { return hanziCharacterSource{} }

// nameStatSource 名字统计默认数据源（domain 直连）
type nameStatSource struct{}

func (nameStatSource) NameStats(name string) *namestat.NameStat {
	return namestat.GetNameStats(name)
}

func (nameStatSource) TopNames(limit int, kind string) []namestat.NameStat {
	return namestat.GetTopNames(limit, kind)
}

// NewNameStatSource 默认名字统计数据源
func NewNameStatSource() NameStatSource { return nameStatSource{} }

// --- 物料管理 ---

// GetMaterialStates 列出物料库内状态（物料/版本/指纹/行数/导入时间）。
// 内存存储模式返回空列表（无物料库可查，不视为错误）。
func (a *App) GetMaterialStates() ([]MaterialState, error) {
	if a.materials == nil {
		return []MaterialState{}, nil
	}
	states, err := a.materials.MaterialStates()
	if err != nil {
		// 物料状态错误来自库层，本身已是中文可行动文案，直接透传
		logger.Error("物料状态读取失败", zap.Error(err))
		return nil, errors.New(err.Error())
	}
	if states == nil {
		states = []MaterialState{}
	}
	return states, nil
}

// ImportMaterials 手动重新导入物料（从装配时的物料目录扫描，指纹短路幂等）。
// 重导只写 SQLite；运行期已装载的内存缓存（诗词索引/统计等）需重启应用后生效。
func (a *App) ImportMaterials() (*MaterialImportReport, error) {
	if a.materials == nil {
		return nil, errors.New("当前未使用 SQLite 存储，物料管理不可用")
	}
	report, err := a.materials.Reimport()
	if err != nil {
		// 物料导入错误来自库层（含校验失败明细），本身已是中文可行动文案，直接透传
		logger.Error("物料重新导入失败", zap.Error(err))
		return nil, errors.New(err.Error())
	}
	return report, nil
}

// --- 校验辅助（与 handlers/validator 同口径） ---

// validateGenerateFields 生成/测名请求的统一校验
func validateGenerateFields(surname, gender string, birthYear, birthMonth, birthDay, birthHour, birthMinute int) error {
	if err := validator.ValidateSurname(surname); err != nil {
		return err
	}
	if err := validator.ValidateGender(gender); err != nil {
		return err
	}
	if err := validator.ValidateDate(birthYear, birthMonth, birthDay); err != nil {
		return err
	}
	if err := validator.ValidateTime(birthHour, birthMinute); err != nil {
		return err
	}
	return nil
}

// validateYMD 真实日历校验（2024-02-30 这类「格式合法但不存在」的日期）
func validateYMD(year, month, day int) error {
	if month < 1 || month > 12 {
		return errors.New("月份参数错误")
	}
	if day < 1 || day > 31 {
		return errors.New("日期参数错误")
	}
	if err := validator.ValidateDate(year, month, day); err != nil {
		return errors.New("日期不合法（如 2 月 30 日）")
	}
	return nil
}

// normalizePagination 分页参数规范化：0 回落默认值 1/100；显式负值直接报错；limit 上限 100
// （与 parsePagination 同口径：非法值不静默兜底，0 视为「未传」）
func normalizePagination(page, limit int) (int, int, error) {
	if page < 0 {
		return 0, 0, errors.New("page 必须是正整数")
	}
	if page == 0 {
		page = 1
	}
	if limit < 0 {
		return 0, 0, errors.New("limit 必须是正整数")
	}
	if limit == 0 {
		limit = maxPageSize
	}
	if limit > maxPageSize {
		return 0, 0, fmt.Errorf("limit 不能超过 %d", maxPageSize)
	}
	return page, limit, nil
}
