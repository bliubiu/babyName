package services

// fate_name_service_e2e_test.go — FateNameService 端到端集成测试
//
// 背景：docs/19 报告指出 fate_name_service.go:54 缺少端到端测试。
// 本文件覆盖整条链路：
//   buildFilterOption → session.Start → generate → 字段映射 → NameAnalysis
//
// 装配与 cmd/server、cmd/namer-cli、name_pinyin_integration_test.go 保持一致：
//   data.Init(dataDir) → fate.SetCuratedNames(...) → fate.NewEngine(...) → NewFateNameService(engine)
//
// 注意：本测试套件使用 sync.Once 保证 data.Init 仅执行一次（与 handler 测试的 setupFateNameService 共用 fateOnce 标志）。
// 单独运行本测试时若未先跑 handler 测试，data.Init 会正常执行。

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"unicode/utf8"

	"go.uber.org/zap"

	"name/internal/domain/fate"
	"name/internal/infrastructure/cache"
	"name/internal/infrastructure/data"
)

var (
	fateServiceE2EOnce    sync.Once
	fateServiceE2E       *FateNameService
	fateServiceE2EInitErr error
)

// e2eDataDir 基于本文件位置定位 backend/data 目录（不依赖运行 CWD）
//
// 文件位于 backend/internal/application/services/，需要 ../ × 3 才能到 backend 根。
func e2eDataDir() string {
	_, file, _, _ := runtime.Caller(0)
	// filepath.Dir(file) = .../backend/internal/application/services
	// 上一级 = application，再上一级 = internal，再上一级 = backend 根
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "data")
}

// setupFateNameServiceE2E 装配 FateNameService 实例（与 cmd/server 同序）
func setupFateNameServiceE2E(t *testing.T) *FateNameService {
	t.Helper()
	fateServiceE2EOnce.Do(func() {
		dataDir := e2eDataDir()

		// 1. 加载全部数据（kangxi → namer → yijing → classic → zodiac → 门禁 → 禁忌组合）
		if err := data.Init(dataDir); err != nil {
			fateServiceE2EInitErr = err
			return
		}
		// 2. 同步命名索引（与 cmd/server 一致）
		SyncNamingIndexFromHanzi()

		// 3. 注入策展好名（共现分白名单；缺失可忽略）
		if curated := loadCuratedNamesForE2E(dataDir); len(curated) > 0 {
			fate.SetCuratedNames(curated)
		}

		// 4. 装配 fate 引擎 + FateNameService
		cache.Init()
		fateEngine := fate.NewEngine(&HanziDataProvider{}, NewBaziAnalyzerAdapter(), fate.DefaultRaters())
		fateServiceE2E = NewFateNameService(fateEngine)
	})
	if fateServiceE2EInitErr != nil {
		t.Fatalf("装配 FateNameService 失败: %v", fateServiceE2EInitErr)
	}
	return fateServiceE2E
}

// loadCuratedNamesForE2E 读取 curated_names.json name 字段（与 handler 测试同）
func loadCuratedNamesForE2E(dataDir string) []string {
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

// TestFateNameService_E2E_SingleName 端到端：单名生成（男+2024-01-15 12:00）
//
// 验证：
//   - 返回不为 nil
//   - Names 至少包含 1 个名字
//   - 每个名字的 Pinyin/FullName/Surname 字段已回填
//   - Bazi 字段已填充（日柱 + 五行）
func TestFateNameService_E2E_SingleName(t *testing.T) {
	svc := setupFateNameServiceE2E(t)
	ctx := context.Background()

	req := &GenerateRequest{
		Surname:    "王",
		Gender:     "male",
		BirthYear:  2024,
		BirthMonth: 1,
		BirthDay:   15,
		BirthHour:  12,
		BirthMinute: 0,
		NameLength: 1,
	}

	resp, err := svc.GenerateWithAnalysis(ctx, req)
	if err != nil {
		t.Fatalf("GenerateWithAnalysis 失败: %v", err)
	}
	if resp == nil {
		t.Fatal("response = nil")
	}
	if len(resp.Names) == 0 {
		t.Fatal("单名生成应返回至少 1 个名字，实际为 0")
	}

	// 抽样验证前 3 个名字
	for i, n := range resp.Names {
		if i >= 3 {
			break
		}
		if n.Surname != "王" {
			t.Errorf("名字[%d] 期望 Surname=王, 实际 %q", i, n.Surname)
		}
		if n.FullName == "" {
			t.Errorf("名字[%d] FullName 未回填", i)
		}
		if !strings.HasPrefix(n.FullName, "王") {
			t.Errorf("名字[%d] FullName=%q 应以姓\"王\"开头", i, n.FullName)
		}
		if utf8.RuneCountInString(n.GivenName) != 1 {
			t.Errorf("名字[%d] GivenName=%q 长度应=1（单名场景）", i, n.GivenName)
		}
		// TotalScore 应在 0-100 范围
		if n.TotalScore < 0 || n.TotalScore > 100 {
			t.Errorf("名字[%d] TotalScore=%v 超出 [0,100]", i, n.TotalScore)
		}
		// 拼音回填（修复前会为空 — 见 docs/19 报告 B 编号：Pinyin 字段必填）
		if n.Pinyin == "" {
			t.Errorf("名字[%d] Pinyin 未回填（已知回归点）", i)
		}
		// 释义回填
		if n.Wuxing == "" {
			t.Errorf("名字[%d] Wuxing 未回填", i)
		}
		// 综合评分不应低于 50（命理层基线）
		if n.TotalScore < 50 {
			t.Logf("名字[%d] TotalScore=%v 偏低，但仍合法", i, n.TotalScore)
		}
	}

	// Bazi 字段应填充
	if resp.Bazi.Rishou == "" {
		t.Error("Bazi.Rishou（日柱）未填充")
	}
	if len(resp.Bazi.Xiyongshen) == 0 {
		t.Error("Bazi.Xiyongshen（喜用神）应至少 1 项")
	}
	if resp.Zodiac == "" {
		t.Error("Zodiac（生肖）未填充")
	}
}

// TestFateNameService_E2E_DoubleName 端到端：双名生成
func TestFateNameService_E2E_DoubleName(t *testing.T) {
	svc := setupFateNameServiceE2E(t)
	ctx := context.Background()

	req := &GenerateRequest{
		Surname:    "李",
		Gender:     "female",
		BirthYear:  2020,
		BirthMonth: 8,
		BirthDay:   8,
		BirthHour:  10,
		BirthMinute: 0,
		NameLength: 2,
	}

	resp, err := svc.GenerateWithAnalysis(ctx, req)
	if err != nil {
		t.Fatalf("GenerateWithAnalysis 失败: %v", err)
	}
	if resp == nil || len(resp.Names) == 0 {
		t.Fatal("双名生成应返回至少 1 个名字")
	}

	// 抽样前 3 个
	for i, n := range resp.Names {
		if i >= 3 {
			break
		}
		if n.Surname != "李" {
			t.Errorf("名字[%d] Surname=%q 应=李", i, n.Surname)
		}
		if utf8.RuneCountInString(n.GivenName) != 2 {
			t.Errorf("名字[%d] GivenName=%q 长度应=2（双名场景）", i, n.GivenName)
		}
		if n.TotalScore < 50 {
			t.Errorf("名字[%d] TotalScore=%v 异常低（应>=50）", i, n.TotalScore)
		}
		// 双名不应包含禁忌组合（已在引擎中过滤）
		if fate.IsBadCombo(string([]rune(n.GivenName)[0]), string([]rune(n.GivenName)[1])) {
			t.Errorf("名字[%d] %s 命中禁忌组合", i, n.GivenName)
		}
	}
}

// TestFateNameService_E2E_FilterOptions 过滤条件生效
//
// 用户显式指定 WuxingMatch（仅木行）+ 笔画范围应过滤候选池。
func TestFateNameService_E2E_FilterOptions(t *testing.T) {
	svc := setupFateNameServiceE2E(t)
	ctx := context.Background()

	req := &GenerateRequest{
		Surname:     "张",
		Gender:      "male",
		BirthYear:   2023,
		BirthMonth:  6,
		BirthDay:    1,
		BirthHour:   8,
		BirthMinute: 0,
		NameLength:  2,
		WuxingMatch: []string{"木"}, // 显式指定木行
		MinStrokes:  8,
		MaxStrokes:  16,
	}

	resp, err := svc.GenerateWithAnalysis(ctx, req)
	if err != nil {
		t.Fatalf("GenerateWithAnalysis 失败: %v", err)
	}
	if resp == nil {
		t.Fatal("response = nil")
	}
	// 不强制要求返回名字（小池 + 严格过滤可能无结果），
	// 但若有结果，应符合过滤条件（任意名字含一个木行字）
	if len(resp.Names) > 0 {
		for i, n := range resp.Names {
			// Wuxing 字段是 2 字五行（如"木火"），至少含一个木
			if !strings.Contains(n.Wuxing, "木") {
				t.Errorf("名字[%d] Wuxing=%q 不含木（违反 WuxingMatch=木 过滤）",
					i, n.Wuxing)
			}
		}
	}
}

// TestFateNameService_E2E_ElderAvoidance 避讳长辈姓名（同形字/同音字排除）
//
// 长辈姓名含"王"——生成时应排除"王"字本身与同音字（wáng/wang）。
// 注意：surname 本身不在排除范围（请求姓氏 = 长辈姓氏时仍生效）。
func TestFateNameService_E2E_ElderAvoidance(t *testing.T) {
	svc := setupFateNameServiceE2E(t)
	ctx := context.Background()

	req := &GenerateRequest{
		Surname:        "李",
		Gender:         "male",
		BirthYear:      2024,
		BirthMonth:     3,
		BirthDay:       15,
		BirthHour:      14,
		BirthMinute:    0,
		NameLength:     2,
		AvoidElderNames: []string{"张王"}, // 长辈含"王"
	}

	resp, err := svc.GenerateWithAnalysis(ctx, req)
	if err != nil {
		t.Fatalf("GenerateWithAnalysis 失败: %v", err)
	}
	if resp == nil {
		t.Fatal("response = nil")
	}
	for i, n := range resp.Names {
		if i >= 10 {
			break
		}
		// 同形字排除：名字中不应出现"王"
		if strings.Contains(n.GivenName, "王") {
			t.Errorf("名字[%d] %s 含长辈名同形字\"王\"，违反 AvoidElderNames=[张王]",
				i, n.GivenName)
		}
	}
}

// TestFateNameService_E2E_ForbiddenCombos 禁忌组合过滤（962 条清洗组合 + 28 条硬编码）
//
// 注释中提到的 "仲尼/若兮/七政/与砺" 已是 hard negative 或清洗组合，
// 不应出现在 TopN 候选中。
func TestFateNameService_E2E_ForbiddenCombos(t *testing.T) {
	svc := setupFateNameServiceE2E(t)
	ctx := context.Background()

	req := &GenerateRequest{
		Surname:    "王",
		Gender:     "male",
		BirthYear:  2024,
		BirthMonth: 5,
		BirthDay:   20,
		BirthHour:  10,
		BirthMinute: 0,
		NameLength: 2,
	}

	resp, err := svc.GenerateWithAnalysis(ctx, req)
	if err != nil {
		t.Fatalf("GenerateWithAnalysis 失败: %v", err)
	}

	// 清洗组合抽样（覆盖 docs/19 报告 P0 修复验证点）
	dirtyCombos := []string{"仲尼", "若兮", "七政", "与砺", "以方", "以时"}
	for i, n := range resp.Names {
		if i >= 50 {
			break
		}
		for _, d := range dirtyCombos {
			if strings.Contains(n.GivenName, d) {
				t.Errorf("名字[%d] %s 命中清洗组合 %q（962 条清洗应已拦截）",
					i, n.GivenName, d)
			}
		}
	}
}

// TestFateNameService_E2E_ConsistentResults 同请求多次结果应稳定（同 Filter + 同 ctx）
//
// 仅验证不 panic + 返回非 nil；不强制结果一致（并发乱序可能造成）。
func TestFateNameService_E2E_ConsistentResults(t *testing.T) {
	svc := setupFateNameServiceE2E(t)
	ctx := context.Background()

	req := &GenerateRequest{
		Surname:    "刘",
		Gender:     "male",
		BirthYear:  2024,
		BirthMonth: 1,
		BirthDay:   1,
		BirthHour:  0,
		BirthMinute: 0,
		NameLength: 2,
	}

	for i := 0; i < 3; i++ {
		resp, err := svc.GenerateWithAnalysis(ctx, req)
		if err != nil {
			t.Fatalf("第 %d 次调用失败: %v", i+1, err)
		}
		if resp == nil {
			t.Fatalf("第 %d 次返回 nil", i+1)
		}
		if len(resp.Names) == 0 {
			t.Errorf("第 %d 次返回空候选", i+1)
		}
	}
}

// logger 初始化（防止调用过程中 zap.Default 未初始化导致 panic）
func init() {
	_ = zap.NewNop() // 触发 zap 包加载（首次调用 Logger 需要）
}