package fate

// allowlist.go 显式白名单（docs/33 门禁层 M1）
//
// 门禁层原则「白名单优先于黑名单」：被显式 allowlist 收录的「字 / 组合」直接
// 豁免各级黑名单（naming_gate 字级、forbidden_combos 组合级、历史人物组合级），
// 并承载祈福组合（blessing）——祈福组合既豁免软性消极惩罚，又参与评分加分。
//
// 数据来源（两级，与 forbidden_combos 同构）：
//   - 硬编码内置默认（blessingCombos，见 semantic_filter.go）：物料缺失时的兜底；
//   - 外置物料 data/allowlist.json：JSON 热载 / SQLite 物料（kind: char/combo/blessing）。
//
// 语义：内置默认与动态物料取并集；动态物料为空时行为与既有完全一致（零回归）。

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"name/internal/materials/schema"
)

var (
	allowlistMu   sync.RWMutex
	allowChars    = map[string]bool{}
	allowCombos   = map[string]bool{}
	blessingExtra = map[string]bool{} // 动态祈福组合（与内置 blessingCombos 取并集）
)

// AllowEntry 一条白名单记录（entry + kind），供物料装载路径注入
type AllowEntry struct {
	Entry string
	Kind  string
}

// SetAllowlist 以物料条目重建白名单集合（SQLite 物料装载路径，ADR-008）
//
// 覆盖式语义（可重复调用以热更新）；空条目与未知 kind 忽略，避免污染集合。
func SetAllowlist(entries []AllowEntry) {
	chars := make(map[string]bool)
	combos := make(map[string]bool)
	blessings := make(map[string]bool)
	for _, e := range entries {
		entry := strings.TrimSpace(e.Entry)
		if entry == "" {
			continue
		}
		switch e.Kind {
		case schema.AllowKindChar:
			chars[entry] = true
		case schema.AllowKindCombo:
			combos[entry] = true
		case schema.AllowKindBlessing:
			blessings[entry] = true
		}
	}
	allowlistMu.Lock()
	allowChars = chars
	allowCombos = combos
	blessingExtra = blessings
	allowlistMu.Unlock()
}

// LoadAllowlistFromJSON 从 data/allowlist.json 加载白名单（JSON 回落路径）
//
// 文件缺失时降级为「仅内置默认」（清空动态集合、仅警告），不阻断启动；
// 文件存在但解析失败时返回错误以暴露数据损坏。
func LoadAllowlistFromJSON(dataDir string) error {
	path := filepath.Join(dataDir, "allowlist.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			SetAllowlist(nil)
			fmt.Fprintf(os.Stderr, "警告: 白名单表 %s 不存在，已降级为内置默认（仅祈福组合）\n", path)
			return nil
		}
		return fmt.Errorf("读取白名单表失败: %w", err)
	}

	var doc struct {
		Chars     []string `json:"chars"`
		Combos    []string `json:"combos"`
		Blessings []string `json:"blessings"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("解析白名单表失败: %w", err)
	}

	entries := make([]AllowEntry, 0, len(doc.Chars)+len(doc.Combos)+len(doc.Blessings))
	for _, c := range doc.Chars {
		entries = append(entries, AllowEntry{Entry: c, Kind: schema.AllowKindChar})
	}
	for _, c := range doc.Combos {
		entries = append(entries, AllowEntry{Entry: c, Kind: schema.AllowKindCombo})
	}
	for _, c := range doc.Blessings {
		entries = append(entries, AllowEntry{Entry: c, Kind: schema.AllowKindBlessing})
	}
	SetAllowlist(entries)
	return nil
}

// IsAllowlistedChar 判断单字是否被白名单豁免（豁免字级黑名单）
func IsAllowlistedChar(char string) bool {
	if char == "" {
		return false
	}
	allowlistMu.RLock()
	defer allowlistMu.RUnlock()
	return allowChars[char]
}

// IsAllowlistedCombo 判断双字组合是否被白名单豁免（正反序均查）
func IsAllowlistedCombo(c1, c2 string) bool {
	if c1 == "" || c2 == "" {
		return false
	}
	allowlistMu.RLock()
	defer allowlistMu.RUnlock()
	return allowCombos[c1+c2] || allowCombos[c2+c1]
}

// isBlessingExtra 动态祈福组合判定（包内使用，正反序均查）
func isBlessingExtra(c1, c2 string) bool {
	if c1 == "" || c2 == "" {
		return false
	}
	allowlistMu.RLock()
	defer allowlistMu.RUnlock()
	return blessingExtra[c1+c2] || blessingExtra[c2+c1]
}

// AllowlistCounts 返回白名单规模（诊断/测试用）：单字 / 组合 / 祈福
func AllowlistCounts() (chars, combos, blessings int) {
	allowlistMu.RLock()
	defer allowlistMu.RUnlock()
	return len(allowChars), len(allowCombos), len(blessingExtra)
}
