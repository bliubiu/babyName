package review

import (
	"fmt"

	"name/internal/domain/fate"
	"name/internal/materials/schema"
	"name/internal/materials/subscribe"
)

// ToPatch 把已落定的复审条目映射为 gate_patch 补丁条目（docs/33 门禁层 M6）：
//
//   - confirmed（确认违规）：单字 → naming_gate 增补；多字组合 → forbidden_combos 增补
//   - false_positive（误报）：单字 → allowlist(char)；多字 → allowlist(combo)
//   - border（边界）：不产生补丁（仅调阈值/权重）
//   - pending（未落定）：忽略
//
// 条目定位优先用 hit_value，缺省回退 name（收藏候选无独立命中值）。
// 生成的补丁经 subscribe.FormatPatch 写出，交由 `materials patch` + `materials import`
// 版本化合入词表；反向应用（Invert）即可回滚。
func ToPatch(items []fate.ReviewItem) ([]subscribe.PatchEntry, error) {
	entries := make([]subscribe.PatchEntry, 0, len(items))
	for _, it := range items {
		target := it.HitValue
		if target == "" {
			target = it.Name
		}
		if target == "" {
			continue
		}
		switch it.Status {
		case fate.ReviewConfirmed:
			if singleRune(target) {
				entries = append(entries, subscribe.PatchEntry{
					Material: schema.GateMaterial, Op: subscribe.OpAdd, Entry: target,
				})
			} else {
				entries = append(entries, subscribe.PatchEntry{
					Material: schema.ForbiddenMaterial, Op: subscribe.OpAdd, Entry: target,
				})
			}
		case fate.ReviewFalsePositive:
			kind := fate.AllowKindCombo
			if singleRune(target) {
				kind = fate.AllowKindChar
			}
			entries = append(entries, subscribe.PatchEntry{
				Material: schema.AllowlistMaterial, Kind: kind, Op: subscribe.OpAdd, Entry: target,
			})
		case fate.ReviewBorder, fate.ReviewPending:
			continue
		default:
			return nil, fmt.Errorf("未知复审状态: %s", it.Status)
		}
	}
	return entries, nil
}

// singleRune 判断文本是否恰为一个汉字（决定补丁落到字级还是组合级物料）
func singleRune(s string) bool {
	return len([]rune(s)) == 1
}
