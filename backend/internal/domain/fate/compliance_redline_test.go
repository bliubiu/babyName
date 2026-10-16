package fate

import (
	"os"
	"path/filepath"
	"testing"

	"name/internal/materials/schema"
)

// compliance_redline_test.go 守护 L5 合规红线（docs/33 门禁层 M2）：
// 机关单位/争议地名/负面历史人物 的装载、匹配、白名单豁免与内置兜底降级。
//
// 全局红线由本文件独占：每个用例结束清空动态集合（内置 forbiddenEntities 不受影响），
// 避免污染同包其他测试（含 TestIsForbiddenEntity）。

// resetRedline 测试结束清空动态合规红线
func resetRedline(t *testing.T) {
	t.Helper()
	t.Cleanup(func() { SetComplianceRedline(nil) })
}

func TestComplianceRedline_分类装载与计数(t *testing.T) {
	resetRedline(t)
	SetComplianceRedline([]RedlineEntry{
		{Entry: "省高院", Kind: schema.RedlineKindStateOrg},
		{Entry: "钓鱼岛", Kind: schema.RedlineKindPlace},
		{Entry: "秦桧", Kind: schema.RedlineKindPerson},
	})

	stateOrg, place, person := ComplianceRedlineCounts()
	if stateOrg != 1 || place != 1 || person != 1 {
		t.Fatalf("计数不符：state_org=%d place=%d person=%d", stateOrg, place, person)
	}
}

func TestComplianceRedline_空条目与未知kind被忽略(t *testing.T) {
	resetRedline(t)
	SetComplianceRedline([]RedlineEntry{
		{Entry: "", Kind: schema.RedlineKindStateOrg},
		{Entry: "  ", Kind: schema.RedlineKindPlace},
		{Entry: "某实体", Kind: "未知类型"},
	})
	if stateOrg, place, person := ComplianceRedlineCounts(); stateOrg != 0 || place != 0 || person != 0 {
		t.Fatalf("空条目与未知 kind 应被忽略，实得 (%d,%d,%d)", stateOrg, place, person)
	}
}

// TestComplianceRedline_命中机关单位 动态机关 + 内置兜底都必须命中
func TestComplianceRedline_命中机关单位(t *testing.T) {
	resetRedline(t)
	SetComplianceRedline([]RedlineEntry{{Entry: "省高院", Kind: schema.RedlineKindStateOrg}})

	if !IsComplianceRedline("省高院") {
		t.Error("动态机关单位「省高院」全名应命中")
	}
	if !IsComplianceRedline("李省高院") {
		t.Error("动态机关单位「省高院」名部/包含应命中")
	}
	if !IsComplianceRedline("国务院") {
		t.Error("内置兜底机关单位「国务院」应命中（物料缺失也不失效）")
	}
}

// TestComplianceRedline_命中争议地名
func TestComplianceRedline_命中争议地名(t *testing.T) {
	resetRedline(t)
	SetComplianceRedline([]RedlineEntry{{Entry: "北京", Kind: schema.RedlineKindPlace}})

	if !IsComplianceRedline("北京") || !IsComplianceRedline("李北京") {
		t.Error("争议地名「北京」应命中全名与包含")
	}
}

// TestComplianceRedline_命中负面历史人物
func TestComplianceRedline_命中负面历史人物(t *testing.T) {
	resetRedline(t)
	SetComplianceRedline([]RedlineEntry{{Entry: "秦桧", Kind: schema.RedlineKindPerson}})

	if !IsComplianceRedline("秦桧") || !IsComplianceRedline("李秦桧") {
		t.Error("负面历史人物「秦桧」应命中全名与包含")
	}
}

// TestComplianceRedline_白名单优先于红线 被白名单收录的红线条目应豁免
func TestComplianceRedline_白名单优先于红线(t *testing.T) {
	resetRedline(t)
	t.Cleanup(func() { SetAllowlist(nil) })
	SetComplianceRedline([]RedlineEntry{{Entry: "北京", Kind: schema.RedlineKindPlace}})
	if !IsComplianceRedline("李北京") {
		t.Fatal("前置条件：李北京 应命中")
	}
	SetAllowlist([]AllowEntry{{Entry: "北京", Kind: schema.AllowKindCombo}})
	if IsComplianceRedline("李北京") {
		t.Error("白名单优先于黑名单：北京 被豁免后不应再命中红线")
	}
	if !IsComplianceRedline("国务院") {
		t.Error("豁免仅作用于白名单条目，其他红线仍应命中")
	}
}

// TestComplianceRedline_单字条目被忽略 单字条目不参与匹配（避免误伤）
func TestComplianceRedline_单字条目被忽略(t *testing.T) {
	resetRedline(t)
	SetComplianceRedline([]RedlineEntry{{Entry: "京", Kind: schema.RedlineKindPlace}})
	if IsComplianceRedline("京") || IsComplianceRedline("李京") {
		t.Error("单字条目不参与红线匹配")
	}
}

// TestComplianceRedline_空参不命中
func TestComplianceRedline_空参不命中(t *testing.T) {
	resetRedline(t)
	if IsComplianceRedline("") {
		t.Error("空全名不应命中")
	}
}

func TestLoadComplianceRedlineFromJSON(t *testing.T) {
	resetRedline(t)
	dir := t.TempDir()
	body := `{"version":"1.0","state_orgs":["省高院"],"places":["北京"],"persons":["秦桧"]}`
	if err := os.WriteFile(filepath.Join(dir, "compliance_redline.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := LoadComplianceRedlineFromJSON(dir); err != nil {
		t.Fatalf("加载合规红线失败: %v", err)
	}
	if !IsComplianceRedline("李省高院") || !IsComplianceRedline("李北京") || !IsComplianceRedline("李秦桧") {
		stateOrg, place, person := ComplianceRedlineCounts()
		t.Fatalf("JSON 装载后应生效：state_org=%d place=%d person=%d", stateOrg, place, person)
	}
}

// TestLoadComplianceRedlineFromJSON_文件缺失降级内置JSON 缺失只清空动态集合，内置兜底仍生效
func TestLoadComplianceRedlineFromJSON_文件缺失降级内置JSON(t *testing.T) {
	resetRedline(t)
	SetComplianceRedline([]RedlineEntry{{Entry: "北京", Kind: schema.RedlineKindPlace}})
	if err := LoadComplianceRedlineFromJSON(t.TempDir()); err != nil {
		t.Fatalf("文件缺失不应报错（降级内置兜底）: %v", err)
	}
	if stateOrg, place, person := ComplianceRedlineCounts(); stateOrg != 0 || place != 0 || person != 0 {
		t.Errorf("缺失文件应清空动态集合，实得 (%d,%d,%d)", stateOrg, place, person)
	}
	if !IsComplianceRedline("国务院") {
		t.Error("缺失文件后内置兜底机关单位仍应命中")
	}
	if IsComplianceRedline("李北京") {
		t.Error("缺失文件后动态地名应失效")
	}
}

func TestLoadComplianceRedlineFromJSON_解析失败报错(t *testing.T) {
	resetRedline(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "compliance_redline.json"), []byte("{ 不是 JSON"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := LoadComplianceRedlineFromJSON(dir); err == nil {
		t.Error("损坏的 JSON 应返回错误以暴露数据损坏")
	}
}

// TestLoadComplianceRedlineFromJSON_路径非法报错 读取失败（非「不存在」）必须返回错误
func TestLoadComplianceRedlineFromJSON_路径非法报错(t *testing.T) {
	resetRedline(t)
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "compliance_redline.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := LoadComplianceRedlineFromJSON(dir); err == nil {
		t.Error("读取失败（路径为目录）应返回错误")
	}
}
