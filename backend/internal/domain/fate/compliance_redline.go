package fate

// compliance_redline.go L5 合规红线（docs/33 门禁层 M2）
//
// 目的：把「机关单位 / 争议地名 / 负面历史人物」这类输出侧法务红线，从硬编码
// 抽为可独立发布 / 可回滚 / 可热更新的物料（compliance_redline.json），并保持
// 与 M1 一致的「白名单优先于黑名单」语义。
//
// 两级数据源（与 forbidden_combos / allowlist 同构）：
//   - 硬编码内置兜底 forbiddenEntities（见 sensitive_chars.go）：物料缺失时红线仍生效；
//   - 外置物料 data/compliance_redline.json：JSON 热载 / SQLite 物料
//     （kind: state_org 机关单位 / place 争议地名 / person 负面历史人物）。
//
// 匹配规则（沿用原 IsForbiddenEntity 语义）：全名精确、名部（去姓氏首字）精确、
// 或全名包含红线条目（条目长度 ≥2）即命中；命中后先查白名单豁免。

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	redlineMu       sync.RWMutex
	redlineStateOrg = map[string]bool{}
	redlinePlace    = map[string]bool{}
	redlinePerson   = map[string]bool{}
	// redlineTries 动态三集合的自动机，随 set 同步重建（docs/33 M8）
	redlineTries [3]*ACTrie
)

// entityTrie 内置兜底 forbiddenEntities 的自动机（静态，包初始化构建一次）
var entityTrie = newRedlineTrie(forbiddenEntities)

// newRedlineTrie 以集合构建自动机，跳过单字条目（沿用原 matchAnyEntity 语义）
func newRedlineTrie(set map[string]bool) *ACTrie {
	patterns := make([]string, 0, len(set))
	for entry := range set {
		if len([]rune(entry)) >= 2 {
			patterns = append(patterns, entry)
		}
	}
	return NewACTrie(patterns)
}

// RedlineEntry 一条合规红线记录（entry + kind），供物料装载路径注入
type RedlineEntry struct {
	Entry string
	Kind  string
}

// SetComplianceRedline 以物料条目重建红线集合（SQLite 物料装载路径，ADR-008）
//
// 覆盖式语义（可重复调用以热更新）；空条目与未知 kind 忽略，避免污染集合。
func SetComplianceRedline(entries []RedlineEntry) {
	stateOrg := make(map[string]bool)
	place := make(map[string]bool)
	person := make(map[string]bool)
	for _, e := range entries {
		entry := strings.TrimSpace(e.Entry)
		if entry == "" {
			continue
		}
		switch e.Kind {
		case RedlineKindStateOrg:
			stateOrg[entry] = true
		case RedlineKindPlace:
			place[entry] = true
		case RedlineKindPerson:
			person[entry] = true
		}
	}
	redlineMu.Lock()
	redlineStateOrg = stateOrg
	redlinePlace = place
	redlinePerson = person
	redlineTries = [3]*ACTrie{
		newRedlineTrie(stateOrg),
		newRedlineTrie(place),
		newRedlineTrie(person),
	}
	redlineMu.Unlock()
}

// LoadComplianceRedlineFromJSON 从 data/compliance_redline.json 加载合规红线（JSON 回落路径）
//
// 文件缺失时降级为「仅内置兜底」（清空动态集合、仅警告），不阻断启动；
// 文件存在但解析失败时返回错误以暴露数据损坏。
func LoadComplianceRedlineFromJSON(dataDir string) error {
	path := filepath.Join(dataDir, "compliance_redline.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			SetComplianceRedline(nil)
			fmt.Fprintf(os.Stderr, "警告: 合规红线表 %s 不存在，已降级为内置兜底（机关单位）\n", path)
			return nil
		}
		return fmt.Errorf("读取合规红线表失败: %w", err)
	}

	var doc struct {
		StateOrgs []string `json:"state_orgs"`
		Places    []string `json:"places"`
		Persons   []string `json:"persons"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("解析合规红线表失败: %w", err)
	}

	entries := make([]RedlineEntry, 0, len(doc.StateOrgs)+len(doc.Places)+len(doc.Persons))
	for _, e := range doc.StateOrgs {
		entries = append(entries, RedlineEntry{Entry: e, Kind: RedlineKindStateOrg})
	}
	for _, e := range doc.Places {
		entries = append(entries, RedlineEntry{Entry: e, Kind: RedlineKindPlace})
	}
	for _, e := range doc.Persons {
		entries = append(entries, RedlineEntry{Entry: e, Kind: RedlineKindPerson})
	}
	SetComplianceRedline(entries)
	return nil
}

// IsComplianceRedline 判断全名是否命中合规红线（机关单位/争议地名/负面历史人物）
//
// 匹配规则：全名精确、名部（去姓氏首字）精确、或全名包含红线条目（条目长度 ≥2）。
// 命中后先查白名单豁免（allow > deny）；内置兜底 forbiddenEntities 始终参与。
func IsComplianceRedline(fullName string) bool {
	if fullName == "" {
		return false
	}
	redlineMu.RLock()
	defer redlineMu.RUnlock()

	if matchAnyEntity(fullName, entityTrie) {
		return true
	}
	for _, trie := range redlineTries {
		if matchAnyEntity(fullName, trie) {
			return true
		}
	}
	return false
}

// matchAnyEntity 判断 fullName 是否命中集合内任一红线条目
//
// fullName 含条目（含「全名==条目」与「名部==条目」两种情形）即命中；
// 命中前置白名单豁免（IsAllowlistedText）。单字条目不参与（避免误伤）。
// 调用方须持有 redlineMu 读锁。
func matchAnyEntity(fullName string, trie *ACTrie) bool {
	if fullName == "" || trie == nil {
		return false
	}
	for _, entry := range trie.FindAll(fullName) {
		if IsAllowlistedText(entry) {
			continue // 白名单优先于黑名单
		}
		return true
	}
	return false
}

// ComplianceRedlineCounts 返回动态合规红线规模（诊断/测试用）：机关单位 / 地名 / 负面人物
func ComplianceRedlineCounts() (stateOrg, place, person int) {
	redlineMu.RLock()
	defer redlineMu.RUnlock()
	return len(redlineStateOrg), len(redlinePlace), len(redlinePerson)
}
