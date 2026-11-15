// Package bootstrap 桌面端全量功能 E2E（docs/47 S5）
//
// 与 frontend/e2e 的 Playwright 用例互补：这里用真实的 SQLite + 真实物料
// 走完 bootstrap.Run 装配链，逐一调用 Wails Bind 暴露的全部 22 个方法，
// 覆盖 UI 未触达的同步生成、换一批、异步任务、PDF、分享、历史/收藏 CRUD、
// 农历月历、偏旁分组与物料重导，并校验错误分支。
//
// 文件名以 z_ 开头：本用例会真实播种 63MB 物料并重设领域层全局索引，
// 放在包内最后执行，避免影响既有单元测试的全局状态。
package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"name/internal/application/services"
	"name/internal/testutil"
)

// TestAppE2E_桌面绑定全量功能 真实装配一遍桌面应用，遍历所有绑定方法。
func TestAppE2E_桌面绑定全量功能(t *testing.T) {
	if testing.Short() {
		t.Skip("-short 模式跳过真实物料装配 E2E")
	}

	_, callerFile, _, _ := runtime.Caller(0)
	dataDir := testutil.MustDataDir(t, callerFile)
	dbPath := filepath.Join(t.TempDir(), "namer-e2e.db")

	res, err := Run(Options{DBPath: dbPath, DataDir: dataDir, SeedFromJSON: true})
	if err != nil {
		t.Fatalf("真实装配失败: %v", err)
	}
	defer func() { _ = res.Close() }()

	app := res.App
	app.Startup(context.Background()) // 模拟 Wails 生命周期回调

	genReq := services.GenerateRequest{
		Surname: "李", Gender: "male",
		BirthYear: 2024, BirthMonth: 2, BirthDay: 10, BirthHour: 10, BirthMinute: 30,
	}
	var genResp *services.GenerateResponse

	t.Run("01 同步生成 GenerateNames", func(t *testing.T) {
		genResp, err = app.GenerateNames(genReq)
		if err != nil {
			t.Fatalf("同步生成失败: %v", err)
		}
		if genResp == nil || len(genResp.Names) == 0 {
			t.Fatal("应返回非空候选名列表")
		}
		if genResp.GenerationID == "" {
			t.Fatal("应返回生成会话标识（换一批依赖它）")
		}
		for _, n := range genResp.Names {
			// 服务层回填 Surname/FullName，调用方禁止自行拼接
			if !strings.HasPrefix(n.FullName, "李") {
				t.Errorf("候选名 %q 未以姓氏开头", n.FullName)
			}
			if n.TotalScore <= 0 || n.TotalScore > 100 {
				t.Errorf("候选名 %q 综合分越界: %.2f", n.FullName, n.TotalScore)
			}
		}
	})

	t.Run("02 换一批 ExploreNames 与推荐榜零交集", func(t *testing.T) {
		shown := make(map[string]bool, len(genResp.Names))
		for _, n := range genResp.Names {
			shown[n.FullName] = true
		}
		batch, err := app.ExploreNames(services.ExploreRequest{
			GenerationID: genResp.GenerationID, Count: 5,
		})
		if err != nil {
			t.Fatalf("换一批失败: %v", err)
		}
		if len(batch.Names) == 0 {
			t.Fatal("换一批应返回新候选")
		}
		for _, n := range batch.Names {
			if shown[n.FullName] {
				t.Errorf("换一批与推荐榜重复: %s", n.FullName)
			}
		}
	})

	t.Run("03 测名 EvaluateName", func(t *testing.T) {
		resp, err := app.EvaluateName(services.EvaluateRequest{
			Surname: "王", GivenName: "浩然", Gender: "male",
			BirthYear: 2024, BirthMonth: 2, BirthDay: 10, BirthHour: 10,
		})
		if err != nil {
			t.Fatalf("测名失败: %v", err)
		}
		if resp.FullName != "王浩然" {
			t.Errorf("FullName 应由服务层回填，实际 %q", resp.FullName)
		}
		if resp.Name.TotalScore <= 0 || resp.Name.TotalScore > 100 {
			t.Errorf("测名综合分越界: %.2f", resp.Name.TotalScore)
		}
	})

	t.Run("04 异步任务 SubmitGenerateTask / GetGenerateTask", func(t *testing.T) {
		submitted, err := app.SubmitGenerateTask(genReq)
		if err != nil {
			t.Fatalf("提交异步任务失败: %v", err)
		}
		if submitted.TaskID == "" {
			t.Fatal("应返回任务标识")
		}
		deadline := time.Now().Add(90 * time.Second)
		for time.Now().Before(deadline) {
			view, err := app.GetGenerateTask(submitted.TaskID)
			if err != nil {
				t.Fatalf("查询异步任务失败: %v", err)
			}
			switch view.Status {
			case services.TaskStatusSuccess:
				if view.Result == nil || len(view.Result.Names) == 0 {
					t.Fatal("任务成功但结果为空")
				}
				return
			case services.TaskStatusFailed:
				t.Fatalf("异步任务失败: %s", view.Err)
			}
			time.Sleep(200 * time.Millisecond)
		}
		t.Fatal("异步任务在 90s 内未完成")
	})

	t.Run("05 历史 SaveHistory / GetHistory / DeleteHistory", func(t *testing.T) {
		saved, err := app.SaveHistory(services.HistoryRecord{
			Surname: "李", Gender: "male", BirthDate: "2024-02-10",
			Results: json.RawMessage(`{"names":[]}`),
		})
		if err != nil {
			t.Fatalf("保存历史失败: %v", err)
		}
		page, err := app.GetHistory(1, 10)
		if err != nil {
			t.Fatalf("查询历史失败: %v", err)
		}
		found := false
		for _, r := range page.Records {
			if r.ID == saved.ID {
				found = true
			}
		}
		if !found {
			t.Fatalf("新保存的历史 %s 未出现在列表", saved.ID)
		}
		if _, err := app.DeleteHistory(saved.ID); err != nil {
			t.Fatalf("删除历史失败: %v", err)
		}
		page, err = app.GetHistory(1, 10)
		if err != nil {
			t.Fatalf("删除后查询历史失败: %v", err)
		}
		for _, r := range page.Records {
			if r.ID == saved.ID {
				t.Fatal("删除后历史记录仍然存在")
			}
		}
	})

	t.Run("06 收藏 SaveFavorite / GetFavorites / DeleteFavorite", func(t *testing.T) {
		saved, err := app.SaveFavorite(services.FavoriteRecord{
			Surname: "李", GivenName: "安定", Gender: "male", Score: 85.1,
		})
		if err != nil {
			t.Fatalf("保存收藏失败: %v", err)
		}
		page, err := app.GetFavorites(1, 10)
		if err != nil {
			t.Fatalf("查询收藏失败: %v", err)
		}
		found := false
		for _, r := range page.Records {
			if r.ID == saved.ID {
				found = true
			}
		}
		if !found {
			t.Fatalf("新保存的收藏 %s 未出现在列表", saved.ID)
		}
		if _, err := app.DeleteFavorite(saved.ID); err != nil {
			t.Fatalf("删除收藏失败: %v", err)
		}
		page, err = app.GetFavorites(1, 10)
		if err != nil {
			t.Fatalf("删除后查询收藏失败: %v", err)
		}
		for _, r := range page.Records {
			if r.ID == saved.ID {
				t.Fatal("删除后收藏仍然存在")
			}
		}
	})

	t.Run("07 PDF 报告 GenerateReportPDF 带真实数据", func(t *testing.T) {
		data := map[string]interface{}{
			"surname": "李", "gender": "male", "birth_date": "2024-02-10",
			"names": []interface{}{
				map[string]interface{}{"full_name": "李安定", "score": 85.1, "meaning": "安宁笃定"},
			},
		}
		pdfBytes, err := app.GenerateReportPDF(data)
		if err != nil {
			t.Fatalf("生成 PDF 失败: %v", err)
		}
		if !bytes.HasPrefix(pdfBytes, []byte("%PDF-")) {
			t.Fatal("PDF 应以 %PDF- 开头")
		}
		if len(pdfBytes) < 4*1024 {
			t.Errorf("PDF 体积 %d 字节，疑似未渲染真实数据", len(pdfBytes))
		}
	})

	t.Run("08 分享快照 CreateShare / GetShare 白名单与 hash 路由", func(t *testing.T) {
		payload := json.RawMessage(`{
			"title":"李氏宝宝起名结果","surname":"李",
			"names":[{"full_name":"李安定","score":85.1}],
			"birth_location":"此处应被白名单剔除"
		}`)
		share, err := app.CreateShare(payload)
		if err != nil {
			t.Fatalf("创建分享失败: %v", err)
		}
		if !strings.HasPrefix(share.URL, "/#/share?id=") {
			t.Errorf("分享链接必须走 hash 路由，实际 %q", share.URL)
		}
		raw, err := app.GetShare(share.ID)
		if err != nil {
			t.Fatalf("读取分享失败: %v", err)
		}
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(raw, &obj); err != nil {
			t.Fatalf("分享快照不是 JSON 对象: %v", err)
		}
		if _, ok := obj["surname"]; !ok {
			t.Error("白名单字段 surname 丢失")
		}
		if _, ok := obj["birth_location"]; ok {
			t.Error("非白名单字段 birth_location 未被剔除")
		}
	})

	t.Run("09 黄历农历 GetHuangli / GetLunarCalendar / GetLunarMonthCalendar", func(t *testing.T) {
		hl, err := app.GetHuangli(2024, 2, 10)
		if err != nil || hl == nil {
			t.Fatalf("黄历查询失败: %v", err)
		}
		lunar, err := app.GetLunarCalendar(2024, 2, 10, 10, 30)
		if err != nil || lunar == nil {
			t.Fatalf("农历查询失败: %v", err)
		}
		month, err := app.GetLunarMonthCalendar(2024, 2)
		if err != nil || month == nil || len(month.Days) == 0 {
			t.Fatalf("农历月历查询失败: %v", err)
		}
		if month.Year != 2024 || month.Month != 2 {
			t.Errorf("月历年月回填错误: %d-%d", month.Year, month.Month)
		}
	})

	t.Run("10 汉字统计 GetCharGroups / GetNameStats / GetTopNames", func(t *testing.T) {
		groups, err := app.GetCharGroups()
		if err != nil || len(groups) == 0 {
			t.Fatalf("偏旁分组为空: %v", err)
		}
		stat, err := app.GetNameStats("梓涵")
		if err != nil {
			t.Fatalf("名字统计失败: %v", err)
		}
		if stat == nil {
			t.Error("名字统计应返回数据")
		}
		top, err := app.GetTopNames(0, "unknown-kind") // 0 回落 20，非法 kind 回落 all
		if err != nil || len(top) == 0 {
			t.Fatalf("热门名字榜为空: %v", err)
		}
		clamped, err := app.GetTopNames(1000, "char") // 上限截断 100
		if err != nil || len(clamped) > 100 {
			t.Errorf("limit 未截断到 100: len=%d, err=%v", len(clamped), err)
		}
	})

	t.Run("11 物料 GetMaterialStates / ImportMaterials 幂等", func(t *testing.T) {
		states, err := app.GetMaterialStates()
		if err != nil {
			t.Fatalf("物料状态读取失败: %v", err)
		}
		if len(states) == 0 {
			t.Fatal("首跑播种后物料状态不应为空")
		}
		hasKangxi, hasNamer := false, false
		for _, s := range states {
			if s.Material == "kangxi" {
				hasKangxi = true
			}
			if s.Material == "namer" {
				hasNamer = true
			}
		}
		if !hasKangxi || !hasNamer {
			t.Errorf("关键物料缺失: kangxi=%v namer=%v", hasKangxi, hasNamer)
		}
		report, err := app.ImportMaterials()
		if err != nil {
			t.Fatalf("手动重导失败: %v", err)
		}
		if report.Failed != 0 {
			t.Errorf("手动重导应零失败，实际 %d 项", report.Failed)
		}
		skipped := false
		for _, item := range report.Items {
			if item.Skipped {
				skipped = true
			}
		}
		if !skipped {
			t.Error("重复导入应命中指纹短路（至少一项跳过）")
		}
	})

	t.Run("12 参数校验与错误分支", func(t *testing.T) {
		if _, err := app.GenerateNames(services.GenerateRequest{}); err == nil {
			t.Error("空姓氏应被拒绝")
		}
		if _, err := app.GetHuangli(2024, 2, 30); err == nil {
			t.Error("不存在的日期应被拒绝")
		}
		if _, err := app.GetLunarMonthCalendar(1899, 1); err == nil {
			t.Error("越界年份应被拒绝")
		}
		if _, err := app.GetLunarCalendar(2024, 2, 10, 25, 0); err == nil {
			t.Error("越界小时应被拒绝")
		}
		if _, err := app.GetHistory(-1, 10); err == nil {
			t.Error("负 page 应被拒绝")
		}
		if _, err := app.GetFavorites(1, 101); err == nil {
			t.Error("limit>100 应被拒绝")
		}
		if _, err := app.ExploreNames(services.ExploreRequest{}); err == nil {
			t.Error("空生成会话标识应被拒绝")
		}
		if _, err := app.GetGenerateTask("not-exist"); err == nil {
			t.Error("不存在的任务应被拒绝")
		}
		if _, err := app.GetShare("invalid-id"); err == nil {
			t.Error("非法分享标识应被拒绝")
		}
		if _, err := app.SaveFavorite(services.FavoriteRecord{Surname: "123", GivenName: "安定"}); err == nil {
			t.Error("非中文姓氏应被拒绝")
		}
		if _, err := app.CreateShare(json.RawMessage(`{"only_disallowed":"x"}`)); err == nil {
			t.Error("全字段被白名单剔除时应拒绝创建分享")
		}
	})
}
