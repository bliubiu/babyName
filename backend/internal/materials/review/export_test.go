package review

import (
	"testing"

	"name/internal/domain/fate"
	"name/internal/materials/schema"
	"name/internal/materials/subscribe"
)

// ——— bad-case 沉淀映射（docs/33 门禁层 M6）——
// 已落定复审条目 → gate_patch 补丁：confirmed 增黑名单，false_positive 增白名单，
// border/pending 不产生补丁。

func TestToPatch_确认违规(t *testing.T) {
	items := []fate.ReviewItem{
		// 单字命中 → naming_gate
		{Status: fate.ReviewConfirmed, HitValue: "某"},
		// 多字组合 → forbidden_combos
		{Status: fate.ReviewConfirmed, HitValue: "七云"},
	}
	got, err := ToPatch(items)
	if err != nil {
		t.Fatalf("ToPatch 出错: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("应生成 2 条补丁，实得 %d", len(got))
	}
	if got[0].Material != schema.GateMaterial || got[0].Op != subscribe.OpAdd || got[0].Entry != "某" {
		t.Fatalf("单字确认应落到 naming_gate: %+v", got[0])
	}
	if got[1].Material != schema.ForbiddenMaterial || got[1].Op != subscribe.OpAdd || got[1].Entry != "七云" {
		t.Fatalf("组合确认应落到 forbidden_combos: %+v", got[1])
	}
}

func TestToPatch_误报进白名单(t *testing.T) {
	items := []fate.ReviewItem{
		{Status: fate.ReviewFalsePositive, HitValue: "伙"},
		{Status: fate.ReviewFalsePositive, HitValue: "康泰"},
	}
	got, err := ToPatch(items)
	if err != nil {
		t.Fatalf("ToPatch 出错: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("应生成 2 条补丁，实得 %d", len(got))
	}
	if got[0].Material != schema.AllowlistMaterial || got[0].Kind != schema.AllowKindChar || got[0].Entry != "伙" {
		t.Fatalf("单字误报应为 allowlist(char): %+v", got[0])
	}
	if got[1].Material != schema.AllowlistMaterial || got[1].Kind != schema.AllowKindCombo || got[1].Entry != "康泰" {
		t.Fatalf("组合误报应为 allowlist(combo): %+v", got[1])
	}
}

func TestToPatch_忽略未落定与边界(t *testing.T) {
	items := []fate.ReviewItem{
		{Status: fate.ReviewPending, HitValue: "待审"},
		{Status: fate.ReviewBorder, HitValue: "边界"},
	}
	got, err := ToPatch(items)
	if err != nil {
		t.Fatalf("ToPatch 出错: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("pending/border 不应产生补丁，实得 %+v", got)
	}
}

func TestToPatch_命中值回退与空值(t *testing.T) {
	items := []fate.ReviewItem{
		// HitValue 为空 → 回退 Name（收藏候选场景）
		{Status: fate.ReviewFalsePositive, Name: "张珀熙"},
		// 两者皆空 → 跳过
		{Status: fate.ReviewConfirmed},
		// 优先使用 HitValue
		{Status: fate.ReviewConfirmed, Name: "王涵", HitValue: "涵"},
	}
	got, err := ToPatch(items)
	if err != nil {
		t.Fatalf("ToPatch 出错: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("应生成 2 条补丁（跳过空值），实得 %d: %+v", len(got), got)
	}
	if got[0].Material != schema.AllowlistMaterial || got[0].Kind != schema.AllowKindCombo || got[0].Entry != "张珀熙" {
		t.Fatalf("应回退 Name 且视为组合: %+v", got[0])
	}
	if got[1].Material != schema.GateMaterial || got[1].Entry != "涵" {
		t.Fatalf("应优先使用 HitValue 且视为单字: %+v", got[1])
	}
}

func TestToPatch_未知状态报错(t *testing.T) {
	if _, err := ToPatch([]fate.ReviewItem{{Status: "bogus", HitValue: "x"}}); err == nil {
		t.Fatal("未知状态应报错")
	}
}

func TestToPatch_空输入(t *testing.T) {
	got, err := ToPatch(nil)
	if err != nil || len(got) != 0 {
		t.Fatalf("空输入应返回空补丁: %+v err=%v", got, err)
	}
}
