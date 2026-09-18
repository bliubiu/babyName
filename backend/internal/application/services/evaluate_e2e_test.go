package services

// evaluate_e2e_test.go — 测名 / 探索模式 / 异步任务 端到端集成测试
//
// 装配与 fate_name_service_e2e_test.go、cmd/server 一致：
//   data.Init(dataDir) → fate.SetCuratedNames(...) → fate.NewEngine(...) → NewNameService(...)

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"name/internal/domain/fate"
	"name/internal/infrastructure/cache"
	"name/internal/infrastructure/data"
)

var (
	nameSvcE2EOnce    sync.Once
	nameSvcE2E        *NameService
	nameSvcE2EInitErr error
)

// setupNameServiceE2E 装配 NameService（与 cmd/server 同序）
func setupNameServiceE2E(t *testing.T) *NameService {
	t.Helper()
	nameSvcE2EOnce.Do(func() {
		dataDir := e2eDataDir()
		if err := data.Init(dataDir); err != nil {
			nameSvcE2EInitErr = err
			return
		}
		SyncNamingIndexFromHanzi()
		if curated := loadCuratedNamesForE2E(dataDir); len(curated) > 0 {
			fate.SetCuratedNames(curated)
		}
		cache.Init()
		fateEngine := fate.NewEngine(&HanziDataProvider{}, NewBaziAnalyzerAdapter(), fate.DefaultRaters())
		fateSvc := NewFateNameService(fateEngine,
			WithFateBaziAnalyzer(&BaziAdapter{}),
			WithFateHexagramFinder(&HexagramAdapter{}),
			WithFateZiweiAnalyzer(&ZiweiAdapter{}),
		)
		nameSvcE2E = NewNameService(
			WithBaziAnalyzer(&BaziAdapter{}),
			WithHexagramFinder(&HexagramAdapter{}),
			WithZiweiAnalyzer(&ZiweiAdapter{}),
			WithZodiacFinder(&ZodiacAdapter{}),
			WithCache(cache.GetCache()),
			WithFateService(fateSvc),
		)
	})
	if nameSvcE2EInitErr != nil {
		t.Fatalf("装配 NameService 失败: %v", nameSvcE2EInitErr)
	}
	return nameSvcE2E
}

// evalE2ERequest 标准测名请求（2024-01-15 12:00 男）
func evalE2ERequest(surname, given string) *EvaluateRequest {
	return &EvaluateRequest{
		Surname:    surname,
		GivenName:  given,
		Gender:     "male",
		BirthYear:  2024,
		BirthMonth: 1,
		BirthDay:   15,
		BirthHour:  12,
	}
}

// TestEvaluate_E2E_Basic 测名基础链路：评分 / score_detail / 风险清单齐全
func TestEvaluate_E2E_Basic(t *testing.T) {
	svc := setupNameServiceE2E(t)
	resp, err := svc.Evaluate(context.Background(), evalE2ERequest("王", "浩然"))
	if err != nil {
		t.Fatalf("Evaluate 失败: %v", err)
	}
	if resp.FullName != "王浩然" {
		t.Fatalf("全名错误: %s", resp.FullName)
	}
	if resp.Name.TotalScore <= 0 || resp.Name.TotalScore > 100 {
		t.Fatalf("总分越界: %v", resp.Name.TotalScore)
	}
	if len(resp.Name.ScoreDetail) == 0 {
		t.Fatal("score_detail 不应为空（与 /generate 同构）")
	}
	if len(resp.Risks) == 0 {
		t.Fatal("风险清单不应为空")
	}
	switch resp.RiskLevel {
	case "pass", "warn", "fail":
	default:
		t.Fatalf("风险等级非法: %s", resp.RiskLevel)
	}
	if resp.Zodiac == "" {
		t.Fatal("生肖不应为空")
	}
}

// TestEvaluate_E2E_RejectedInput 非法输入（三字名）应拒绝
func TestEvaluate_E2E_RejectedInput(t *testing.T) {
	svc := setupNameServiceE2E(t)
	_, err := svc.Evaluate(context.Background(), evalE2ERequest("王", "浩然轩"))
	if err == nil {
		t.Fatal("三字名应被拒绝")
	}
}

// TestEvaluate_E2E_SameScoreAsGenerate ★ 同名同分：生成路径产出的名字，
// 测名路径重新评分必须得到相同总分（防「结果页与测名页双口径」回归）。
func TestEvaluate_E2E_SameScoreAsGenerate(t *testing.T) {
	svc := setupNameServiceE2E(t)
	ctx := context.Background()

	genResp, err := svc.Generate(ctx, &GenerateRequest{
		Surname:     "王",
		Gender:      "male",
		BirthYear:   2024,
		BirthMonth:  1,
		BirthDay:    15,
		BirthHour:   12,
		NameLength:  2,
	})
	if err != nil {
		t.Fatalf("Generate 失败: %v", err)
	}
	if len(genResp.Names) == 0 {
		t.Fatal("生成结果为空")
	}

	checked := 0
	for _, n := range genResp.Names {
		evalResp, err := svc.Evaluate(ctx, evalE2ERequest("王", n.GivenName))
		if err != nil {
			t.Fatalf("测名 %s 失败: %v", n.FullName, err)
		}
		if evalResp.Name.TotalScore != n.TotalScore {
			t.Fatalf("同名不同分：%s 生成=%v 测名=%v",
				n.FullName, n.TotalScore, evalResp.Name.TotalScore)
		}
		checked++
		if checked >= 5 {
			break // 抽查前 5 个即可
		}
	}
	if checked == 0 {
		t.Fatal("未完成任何同分核对")
	}
}

// TestExplore_E2E_Basic 探索模式：换一批与 Top-N 零交集，重复调用不重复
func TestExplore_E2E_Basic(t *testing.T) {
	svc := setupNameServiceE2E(t)
	ctx := context.Background()

	genResp, err := svc.Generate(ctx, &GenerateRequest{
		Surname:    "王",
		Gender:     "male",
		BirthYear:  2024,
		BirthMonth: 1,
		BirthDay:   15,
		BirthHour:  12,
		NameLength: 2,
	})
	if err != nil {
		t.Fatalf("Generate 失败: %v", err)
	}
	if genResp.GenerationID == "" {
		t.Fatal("generation_id 不应为空")
	}

	topSet := make(map[string]bool, len(genResp.Names))
	for _, n := range genResp.Names {
		topSet[n.GivenName] = true
	}

	expResp, err := svc.ExploreNames(ctx, &ExploreRequest{GenerationID: genResp.GenerationID, Count: 5})
	if err != nil {
		t.Fatalf("ExploreNames 失败: %v", err)
	}
	if len(expResp.Names) == 0 {
		t.Fatal("换一批不应为空")
	}
	first := make(map[string]bool, len(expResp.Names))
	for _, n := range expResp.Names {
		if topSet[n.GivenName] {
			t.Fatalf("换一批不应与 Top-N 交集: %s", n.GivenName)
		}
		if n.TotalScore <= 0 {
			t.Fatalf("换一批的名字分数异常: %s = %v", n.GivenName, n.TotalScore)
		}
		first[n.GivenName] = true
	}

	// 第二次调用：不与第一次重复
	expResp2, err := svc.ExploreNames(ctx, &ExploreRequest{GenerationID: genResp.GenerationID, Count: 5})
	if err != nil {
		t.Fatalf("第二次 ExploreNames 失败: %v", err)
	}
	for _, n := range expResp2.Names {
		if first[n.GivenName] {
			t.Fatalf("两次换一批出现重复: %s", n.GivenName)
		}
	}
}

// TestExplore_E2E_UnknownSession 未知的 generation_id
func TestExplore_E2E_UnknownSession(t *testing.T) {
	svc := setupNameServiceE2E(t)
	_, err := svc.ExploreNames(context.Background(), &ExploreRequest{GenerationID: "not-exist"})
	if err == nil {
		t.Fatal("未知会话应报错")
	}
}

// TestTaskService_E2E_SubmitAndPoll 异步任务：提交 → 轮询 → 成功且带结果
func TestTaskService_E2E_SubmitAndPoll(t *testing.T) {
	svc := setupNameServiceE2E(t)
	ts := NewTaskService(svc)

	taskID := ts.Submit(&GenerateRequest{
		Surname:    "王",
		Gender:     "male",
		BirthYear:  2024,
		BirthMonth: 1,
		BirthDay:   15,
		BirthHour:  12,
		NameLength: 2,
	})
	if taskID == "" {
		t.Fatal("task_id 不应为空")
	}

	deadline := time.Now().Add(2 * time.Minute)
	var lastPercent float64
	for {
		task, ok := ts.Get(taskID)
		if !ok {
			t.Fatal("任务丢失")
		}
		status, stage, percent, _ := task.View()
		if percent < lastPercent {
			t.Fatalf("进度回退: %v < %v", percent, lastPercent)
		}
		lastPercent = percent
		_ = stage
		if status == TaskStatusSuccess {
			result := task.TakeResult()
			if result == nil {
				t.Fatal("成功任务应带结果")
			}
			if len(result.Names) == 0 {
				t.Fatal("结果应含名字")
			}
			if result.GenerationID == "" {
				t.Fatal("异步结果也应带 generation_id（探索模式）")
			}
			return
		}
		if status == TaskStatusFailed {
			t.Fatal("任务不应失败")
		}
		if time.Now().After(deadline) {
			t.Fatalf("任务超时未完成，最后进度 %v", lastPercent)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// guard：确保 json/os/filepath/runtime 引用（与既有 e2e 辅助共享文件时避免 import 漂移）
var _ = json.Marshal
var _ = os.Getenv
var _ = filepath.Join
var _ = runtime.Caller
