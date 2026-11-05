// Package subscribe 实现门禁物料的外部词库订阅管线（docs/33 门禁层 M5）：
// 上游源指纹登记（materials.lock）、行差分（line-hash）、补丁生成/应用/回滚。
//
// 设计目标：
//   - 可审计：每物料登记上游源 + 上游 sha256 + 本地产物 sha256（materials.lock）。
//   - 可回滚：补丁为 append-only 的操作行（gate_patch.jsonl），Invert 即得反向补丁。
//   - 可测：核心为纯函数，IO 仅在显式 Load/Save/Apply 处。
//
// 管线：上游 pull → ParseLines/DiffLines（行差分）→ BuildPatch → gate_patch.jsonl
// → ApplyToFile 合入 data 物料 JSON → （回到）materials import 版本化入库。
package subscribe

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"name/internal/domain/fate"
	"name/internal/materials/checksum"
	"name/internal/materials/schema"
)

// ---------- 指纹 ----------
//
// sha256 统一委托 materials/checksum（docs/34 阶段 P2-4）：
// 全仓曾有两份等价实现（本包与 checksum 包），指纹口径必须单一，
// 否则「lock 登记指纹」与「导入门控指纹」可能走向不同算法细节。

// HashBytes 返回数据的 sha256 十六进制串（小写）
func HashBytes(b []byte) string {
	return checksum.Bytes(b)
}

// HashFile 计算文件 sha256；文件不存在或不可读时返回错误
func HashFile(path string) (string, error) {
	return checksum.File(path)
}

// ---------- materials.lock ----------

// LockEntry 单个物料的上游来源与产物指纹
type LockEntry struct {
	Material       string `json:"material"`          // 物料名（schema 注册名）
	Source         string `json:"source"`            // 上游来源（repo/URL）
	Commit         string `json:"commit,omitempty"`  // 上游版本 / commit
	SHA256         string `json:"sha256,omitempty"`  // 上游产物 sha256
	Generated      string `json:"generated"`         // 本地产物路径（相对 dataDir）
	ArtifactSHA256 string `json:"artifact_sha256"`   // 本地产物 sha256
	Command        string `json:"command,omitempty"` // 生成命令（可复现）
}

// Lock materials.lock 文档
//
// 两态（docs/33 门禁层 M7 / ADR-015）：
//   - Entries：当前生效版本（active）；
//   - Staging：候选版本（灰度暂存），缺省即「无候选」，兼容 M5 单态旧格式。
type Lock struct {
	Version string      `json:"version"`
	Note    string      `json:"note,omitempty"`
	Entries []LockEntry `json:"entries"`
	Staging []LockEntry `json:"staging,omitempty"`
}

// LoadLock 读取 lock 文件
func LoadLock(path string) (*Lock, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取 lock 失败: %w", err)
	}
	var l Lock
	if err := json.Unmarshal(b, &l); err != nil {
		return nil, fmt.Errorf("解析 lock 失败: %w", err)
	}
	return &l, nil
}

// SaveLock 写入 lock 文件（2 空格缩进 + 结尾换行）
func SaveLock(path string, l *Lock) error {
	b, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化 lock 失败: %w", err)
	}
	return writeFile(path, append(b, '\n'))
}

// Find 按物料名查条目
func (l *Lock) Find(material string) (LockEntry, bool) {
	for _, e := range l.Entries {
		if e.Material == material {
			return e, true
		}
	}
	return LockEntry{}, false
}

// FindStaging 按物料名查候选条目
func (l *Lock) FindStaging(material string) (LockEntry, bool) {
	for _, e := range l.Staging {
		if e.Material == material {
			return e, true
		}
	}
	return LockEntry{}, false
}

// SetStaging 登记/覆盖某物料的候选条目（同物料只保留最新候选）
func (l *Lock) SetStaging(e LockEntry) {
	for i := range l.Staging {
		if l.Staging[i].Material == e.Material {
			l.Staging[i] = e
			return
		}
	}
	l.Staging = append(l.Staging, e)
}

// RemoveStaging 移除某物料的候选，返回是否原本存在
func (l *Lock) RemoveStaging(material string) bool {
	for i := range l.Staging {
		if l.Staging[i].Material == material {
			l.Staging = append(l.Staging[:i], l.Staging[i+1:]...)
			return true
		}
	}
	return false
}

// UpsertEntry 登记/更新某物料的生效条目：仅更新指纹与产物路径，保留既有来源信息
func (l *Lock) UpsertEntry(e LockEntry) {
	for i := range l.Entries {
		if l.Entries[i].Material == e.Material {
			if e.ArtifactSHA256 != "" {
				l.Entries[i].ArtifactSHA256 = e.ArtifactSHA256
			}
			if e.Generated != "" {
				l.Entries[i].Generated = e.Generated
			}
			if e.Source != "" {
				l.Entries[i].Source = e.Source
			}
			return
		}
	}
	l.Entries = append(l.Entries, e)
}

// VerifyMismatch 指纹校验不一致项
type VerifyMismatch struct {
	Material string `json:"material"`
	Path     string `json:"path"`
	Reason   string `json:"reason"`
	Want     string `json:"want"`
	Got      string `json:"got,omitempty"`
}

// Verify 校验 lock 登记的每个本地产物 sha256 是否与 dataDir 下实际文件一致；
// 返回不一致项列表（空表示全部一致）。文件缺失 / 指纹不匹配均算不一致。
func (l *Lock) Verify(dataDir string) []VerifyMismatch {
	var out []VerifyMismatch
	for _, e := range l.Entries {
		path := e.Generated
		if !filepath.IsAbs(path) {
			path = filepath.Join(dataDir, e.Generated)
		}
		got, err := HashFile(path)
		if err != nil {
			out = append(out, VerifyMismatch{Material: e.Material, Path: e.Generated, Reason: "读取失败", Want: e.ArtifactSHA256})
			continue
		}
		if got != e.ArtifactSHA256 {
			out = append(out, VerifyMismatch{Material: e.Material, Path: e.Generated, Reason: "指纹不匹配", Want: e.ArtifactSHA256, Got: got})
		}
	}
	return out
}

// ---------- 行差分（line-hash） ----------

// ParseLines 把文本切成归一化行：去首尾空白、跳过空行与 # 注释、去重并保持首现顺序。
func ParseLines(text string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, raw := range strings.Split(text, "\n") {
		s := strings.TrimSpace(raw)
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

// Diff 行差分结果
type Diff struct {
	Added   []string `json:"added"`
	Removed []string `json:"removed"`
}

// Empty 差分是否为空（无增无删）
func (d Diff) Empty() bool { return len(d.Added) == 0 && len(d.Removed) == 0 }

// DiffLines 计算 old → new 的增删行（按归一化内容比较，顺序稳定）。
//
// 入参应是 ParseLines 的产物（已去重）；此处仍做一次去重以防误用。
func DiffLines(oldLines, newLines []string) Diff {
	oldSet := toStringSet(oldLines)
	newSet := toStringSet(newLines)
	var d Diff
	for _, s := range newLines {
		if !oldSet[s] {
			d.Added = appendUnique(d.Added, s)
		}
	}
	for _, s := range oldLines {
		if !newSet[s] {
			d.Removed = appendUnique(d.Removed, s)
		}
	}
	return d
}

// ---------- 补丁（gate_patch.jsonl） ----------

// Op 补丁操作类型
type Op string

const (
	// OpAdd 新增条目
	OpAdd Op = "add"
	// OpRemove 移除条目
	OpRemove Op = "remove"
)

// PatchEntry 一条补丁操作（JSONL 一行）
type PatchEntry struct {
	Material string `json:"material"`
	Kind     string `json:"kind,omitempty"`
	Op       Op     `json:"op"`
	Entry    string `json:"entry"`
}

// BuildPatch 由差分生成补丁序列（先增后删，操作内部稳定有序）
func BuildPatch(material, kind string, d Diff) []PatchEntry {
	out := make([]PatchEntry, 0, len(d.Added)+len(d.Removed))
	for _, s := range d.Added {
		out = append(out, PatchEntry{Material: material, Kind: kind, Op: OpAdd, Entry: s})
	}
	for _, s := range d.Removed {
		out = append(out, PatchEntry{Material: material, Kind: kind, Op: OpRemove, Entry: s})
	}
	return out
}

// ParsePatch 解析 JSONL 补丁：空行与 # 注释跳过；操作非法或缺少关键字段即报错。
func ParsePatch(data []byte) ([]PatchEntry, error) {
	var out []PatchEntry
	for i, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		var e PatchEntry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			return nil, fmt.Errorf("第 %d 行补丁解析失败: %w", i+1, err)
		}
		if e.Op != OpAdd && e.Op != OpRemove {
			return nil, fmt.Errorf("第 %d 行补丁操作非法: %q", i+1, string(e.Op))
		}
		if e.Material == "" || e.Entry == "" {
			return nil, fmt.Errorf("第 %d 行补丁缺少 material/entry", i+1)
		}
		out = append(out, e)
	}
	return out, nil
}

// FormatPatch 把补丁序列化为 JSONL（每行一条，结尾换行）
func FormatPatch(entries []PatchEntry) ([]byte, error) {
	var b strings.Builder
	for _, e := range entries {
		line, err := json.Marshal(e)
		if err != nil {
			return nil, fmt.Errorf("补丁序列化失败: %w", err)
		}
		b.Write(line)
		b.WriteByte('\n')
	}
	return []byte(b.String()), nil
}

// Invert 生成反补丁（add ↔ remove），用于回滚
func Invert(entries []PatchEntry) []PatchEntry {
	out := make([]PatchEntry, len(entries))
	for i, e := range entries {
		n := e
		if e.Op == OpAdd {
			n.Op = OpRemove
		} else {
			n.Op = OpAdd
		}
		out[i] = n
	}
	return out
}

// ApplyToSet 把补丁应用到字符串集合：新增追加（去重），移除下线；重复/缺失操作幂等。
func ApplyToSet(current []string, entries []PatchEntry) []string {
	set := make(map[string]bool, len(current)+len(entries))
	order := make([]string, 0, len(current)+len(entries))
	for _, s := range current {
		if !set[s] {
			set[s] = true
			order = append(order, s)
		}
	}
	for _, e := range entries {
		switch e.Op {
		case OpAdd:
			if !set[e.Entry] {
				set[e.Entry] = true
				order = append(order, e.Entry)
			}
		case OpRemove:
			set[e.Entry] = false
		}
	}
	out := make([]string, 0, len(order))
	for _, s := range order {
		if set[s] {
			out = append(out, s)
		}
	}
	return out
}

// ---------- 应用到物料文件 ----------

// allowlistFile allowlist.json 结构（docs/33 M1）
type allowlistFile struct {
	Version   string   `json:"version"`
	Note      string   `json:"note,omitempty"`
	Chars     []string `json:"chars"`
	Combos    []string `json:"combos"`
	Blessings []string `json:"blessings"`
}

// redlineFile compliance_redline.json 结构（docs/33 M2）
type redlineFile struct {
	Version   string   `json:"version"`
	Note      string   `json:"note,omitempty"`
	StateOrgs []string `json:"state_orgs"`
	Places    []string `json:"places"`
	Persons   []string `json:"persons"`
}

// ApplyToFile 把补丁应用到 dataDir 下对应物料的 JSON 文件（仅应用本物料的条目）。
//
// 支持：
//   - naming_gate / forbidden_combos：字符串数组；
//   - allowlist：按 kind（char/combo/blessing）写入 chars/combos/blessings；
//   - compliance_redline：按 kind（state_org/place/person）写入三集合。
//
// 未知物料或非法 kind 返回错误；不改动其他物料的条目。
func ApplyToFile(dataDir, material string, entries []PatchEntry) error {
	def, ok := schema.Registry[material]
	if !ok {
		return fmt.Errorf("未注册物料: %s", material)
	}
	path := filepath.Join(dataDir, def.Source)

	owned := make([]PatchEntry, 0, len(entries))
	for _, e := range entries {
		if e.Material == material {
			owned = append(owned, e)
		}
	}

	switch material {
	case schema.GateMaterial, schema.ForbiddenMaterial:
		var list []string
		if err := readJSON(path, &list); err != nil {
			return err
		}
		return writeJSON(path, ApplyToSet(list, owned))
	case schema.AllowlistMaterial:
		var f allowlistFile
		if err := readJSON(path, &f); err != nil {
			return err
		}
		byKind, err := groupByKind(owned, []string{fate.AllowKindChar, fate.AllowKindCombo, fate.AllowKindBlessing})
		if err != nil {
			return err
		}
		f.Chars = ApplyToSet(f.Chars, byKind[fate.AllowKindChar])
		f.Combos = ApplyToSet(f.Combos, byKind[fate.AllowKindCombo])
		f.Blessings = ApplyToSet(f.Blessings, byKind[fate.AllowKindBlessing])
		return writeJSON(path, f)
	case schema.ComplianceRedlineMaterial:
		var f redlineFile
		if err := readJSON(path, &f); err != nil {
			return err
		}
		byKind, err := groupByKind(owned, []string{fate.RedlineKindStateOrg, fate.RedlineKindPlace, fate.RedlineKindPerson})
		if err != nil {
			return err
		}
		f.StateOrgs = ApplyToSet(f.StateOrgs, byKind[fate.RedlineKindStateOrg])
		f.Places = ApplyToSet(f.Places, byKind[fate.RedlineKindPlace])
		f.Persons = ApplyToSet(f.Persons, byKind[fate.RedlineKindPerson])
		return writeJSON(path, f)
	default:
		return fmt.Errorf("物料 %s 暂不支持补丁应用", material)
	}
}

// groupByKind 按 kind 分组，并校验 kind 属于允许集合
func groupByKind(entries []PatchEntry, allowed []string) (map[string][]PatchEntry, error) {
	ok := make(map[string]bool, len(allowed))
	for _, k := range allowed {
		ok[k] = true
	}
	out := make(map[string][]PatchEntry, len(allowed))
	for _, e := range entries {
		if !ok[e.Kind] {
			return nil, fmt.Errorf("物料条目类别非法: %q（允许 %s）", e.Kind, strings.Join(allowed, "/"))
		}
		out[e.Kind] = append(out[e.Kind], e)
	}
	return out, nil
}

// ---------- IO 辅助 ----------

func readJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取 %s 失败: %w", path, err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("解析 %s 失败: %w", path, err)
	}
	return nil
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化 %s 失败: %w", path, err)
	}
	return writeFile(path, append(b, '\n'))
}

// writeFile 原子写入：临时文件 → fsync → rename 覆盖目标
//
// 直接 WriteFile 覆盖在进程中断（kill / 掉电）时会留下半截文件——
// promote/abort 改的正是运行时生效物料与 lock，瞬时损坏窗口不可接受
// （docs/34 阶段 C1）。rename 由 OS 保证原子替换；临时文件与目标同目录，
// 确保同一文件系统内 rename 不跨盘。
func writeFile(path string, b []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".wb-tmp-*")
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }() // 失败路径清理；成功后 rename 已移走

	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("写入临时文件失败: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("同步临时文件失败: %w", err)
	}
	// CreateTemp 默认 0600，归一为 0644 与历史行为一致（rename 保留权限）
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("设置临时文件权限失败: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("关闭临时文件失败: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("替换 %s 失败: %w", path, err)
	}
	return nil
}

func toStringSet(in []string) map[string]bool {
	m := make(map[string]bool, len(in))
	for _, s := range in {
		m[s] = true
	}
	return m
}

func appendUnique(dst []string, seen ...string) []string {
	exists := make(map[string]bool, len(dst))
	for _, s := range dst {
		exists[s] = true
	}
	for _, s := range seen {
		if !exists[s] {
			exists[s] = true
			dst = append(dst, s)
		}
	}
	return dst
}

// ---------- 订阅清单生成 ----------

// GateMaterials 参与订阅/补丁管线的门禁类物料（docs/33 M5）
var GateMaterials = []string{
	schema.GateMaterial,
	schema.ForbiddenMaterial,
	schema.AllowlistMaterial,
	schema.ComplianceRedlineMaterial,
}

// BuildLock 生成 materials.lock：登记每个门禁物料的源文件路径与本地产物 sha256。
//
// sources 为「物料名 → 上游来源（repo/URL）」映射，未提供的物料 Source 留空。
// 源文件缺失的物料跳过（可选物料允许缺失）；全部缺失时返回的 Entries 为空。
func BuildLock(dataDir, version string, sources map[string]string) (*Lock, error) {
	l := &Lock{
		Version: version,
		Note:    "门禁物料上游来源与产物指纹（docs/33 门禁层 M5）：可审计、可回滚。",
		Entries: make([]LockEntry, 0, len(GateMaterials)),
	}
	for _, m := range GateMaterials {
		def, ok := schema.Registry[m]
		if !ok {
			continue
		}
		sum, err := HashFile(filepath.Join(dataDir, def.Source))
		if err != nil {
			continue
		}
		l.Entries = append(l.Entries, LockEntry{
			Material:       m,
			Source:         sources[m],
			Generated:      def.Source,
			ArtifactSHA256: sum,
		})
	}
	return l, nil
}
