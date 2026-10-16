package subscribe

// release.go 门禁物料灰度发布（docs/33 门禁层 M7 / ADR-015）
//
// 发布流：候选（stage）→ 观察（observe）→ 启用（promote）| 回滚（abort）。
// 切分粒度按物料整体：每个门禁物料各自一份 active 版本 + 至多一个候选。
//
//   - 候选落 data/staging/<material>.json（不参与运行时装载）；
//   - promote 前把当前 active 产物存档到 data/history/<material>/<sha>.json；
//   - abort（无 --to）丢弃候选；abort --to <sha> 从存档回退 active；
//   - 每次动作追加一条 data/release.jsonl（append-only，可审计）。
//
// 复用 M5：materials.lock（扩 active/staging 两态）、HashFile、ParseLines/DiffLines。

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"name/internal/materials/schema"
)

const (
	// StagingDir 候选目录名（相对 dataDir）
	StagingDir = "staging"
	// HistoryDir 版本存档目录名（相对 dataDir）
	HistoryDir = "history"
	// ReleaseFile 发布日志文件名（相对 dataDir）
	ReleaseFile = "release.jsonl"
)

// ReleaseRecord 一次发布动作（release.jsonl 一行，append-only）
type ReleaseRecord struct {
	Time     string `json:"time"`
	Material string `json:"material"`
	Action   string `json:"action"` // stage | observe | promote | abort
	From     string `json:"from,omitempty"`
	To       string `json:"to,omitempty"`
	Operator string `json:"operator,omitempty"`
	Note     string `json:"note,omitempty"`
}

// StageDiff 候选相对生效版本的预演差分
type StageDiff struct {
	Material  string   `json:"material"`
	Current   string   `json:"current_sha256"`
	Candidate string   `json:"candidate_sha256"`
	Added     []string `json:"added"`
	Removed   []string `json:"removed"`
}

// StagingPath 物料的候选文件路径
func StagingPath(dataDir, material string) string {
	return filepath.Join(dataDir, StagingDir, material+".json")
}

// HistoryPath 物料某指纹的存档路径
func HistoryPath(dataDir, material, sha string) string {
	return filepath.Join(dataDir, HistoryDir, material, sha+".json")
}

// isGateMaterial 是否为参与灰度发布的门禁物料
func isGateMaterial(material string) bool {
	for _, m := range GateMaterials {
		if m == material {
			return true
		}
	}
	return false
}

// Stage 把候选文件登记为某物料的待发布版本
//
// 复制候选文件到 data/staging/，写入 lock.Staging 候选指纹，并追加一条 release 记录；
// 候选须可解析为该物料结构（防脏文件入库）。
func Stage(dataDir, lockPath, material, from, source, operator string) (*ReleaseRecord, error) {
	if !isGateMaterial(material) {
		return nil, fmt.Errorf("物料 %s 不参与灰度发布", material)
	}
	if from == "" {
		return nil, fmt.Errorf("必须指定候选文件路径")
	}
	b, err := os.ReadFile(from)
	if err != nil {
		return nil, fmt.Errorf("读取候选文件失败: %w", err)
	}
	if _, err := parseMaterialEntries(material, b); err != nil {
		return nil, err
	}

	lock, err := LoadLock(lockPath)
	if err != nil {
		return nil, err
	}
	sp := StagingPath(dataDir, material)
	if err := os.MkdirAll(filepath.Dir(sp), 0o755); err != nil {
		return nil, fmt.Errorf("创建候选目录失败: %w", err)
	}
	if err := writeFile(sp, b); err != nil {
		return nil, err
	}
	sum := HashBytes(b)
	lock.SetStaging(LockEntry{
		Material:       material,
		Source:         source,
		Generated:      filepath.ToSlash(filepath.Join(StagingDir, material+".json")),
		ArtifactSHA256: sum,
	})
	if err := SaveLock(lockPath, lock); err != nil {
		return nil, err
	}

	rec := ReleaseRecord{
		Time: nowStamp(), Material: material, Action: "stage",
		To: sum, Operator: operator, Note: source,
	}
	if err := appendRelease(dataDir, rec); err != nil {
		return nil, err
	}
	return &rec, nil
}

// Observe 计算候选相对当前生效版本的条目差分（不改变任何状态）
func Observe(dataDir, lockPath, material string) (*StageDiff, error) {
	if !isGateMaterial(material) {
		return nil, fmt.Errorf("物料 %s 不参与灰度发布", material)
	}
	lock, err := LoadLock(lockPath)
	if err != nil {
		return nil, err
	}
	st, ok := lock.FindStaging(material)
	if !ok {
		return nil, fmt.Errorf("物料 %s 无候选（请先 stage）", material)
	}
	current, err := readMaterialEntries(dataDir, material)
	if err != nil {
		return nil, err
	}
	candBytes, err := os.ReadFile(StagingPath(dataDir, material))
	if err != nil {
		return nil, fmt.Errorf("读取候选失败: %w", err)
	}
	cand, err := parseMaterialEntries(material, candBytes)
	if err != nil {
		return nil, err
	}
	d := DiffLines(current, cand)
	return &StageDiff{
		Material:  material,
		Current:   lockActiveSHA(lock, material),
		Candidate: st.ArtifactSHA256,
		Added:     d.Added,
		Removed:   d.Removed,
	}, nil
}

// lockActiveSHA 取物料生效指纹（无记录则空）
func lockActiveSHA(lock *Lock, material string) string {
	if e, ok := lock.Find(material); ok {
		return e.ArtifactSHA256
	}
	return ""
}

// Promote 把候选提升为生效版本
//
// 流程：当前生效产物存档 → 候选覆盖生效路径 → 更新 lock 指纹并清候选 → 追加 release。
func Promote(dataDir, lockPath, material, operator string) (*ReleaseRecord, error) {
	if !isGateMaterial(material) {
		return nil, fmt.Errorf("物料 %s 不参与灰度发布", material)
	}
	lock, err := LoadLock(lockPath)
	if err != nil {
		return nil, err
	}
	st, ok := lock.FindStaging(material)
	if !ok {
		return nil, fmt.Errorf("物料 %s 无候选（请先 stage）", material)
	}
	def, ok := schema.Registry[material]
	if !ok {
		return nil, fmt.Errorf("未注册物料: %s", material)
	}
	activePath := filepath.Join(dataDir, def.Source)

	// 存档当前生效版本（若存在），供 abort --to 回退
	var from string
	if cur, err := os.ReadFile(activePath); err == nil {
		from = HashBytes(cur)
		hp := HistoryPath(dataDir, material, from)
		if err := os.MkdirAll(filepath.Dir(hp), 0o755); err != nil {
			return nil, fmt.Errorf("创建存档目录失败: %w", err)
		}
		if err := writeFile(hp, cur); err != nil {
			return nil, err
		}
	}

	cand, err := os.ReadFile(StagingPath(dataDir, material))
	if err != nil {
		return nil, fmt.Errorf("读取候选失败: %w", err)
	}
	if err := writeFile(activePath, cand); err != nil {
		return nil, err
	}
	if err := os.Remove(StagingPath(dataDir, material)); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("清理候选失败: %w", err)
	}

	lock.UpsertEntry(LockEntry{Material: material, Generated: def.Source, ArtifactSHA256: st.ArtifactSHA256})
	lock.RemoveStaging(material)
	if err := SaveLock(lockPath, lock); err != nil {
		return nil, err
	}

	rec := ReleaseRecord{
		Time: nowStamp(), Material: material, Action: "promote",
		From: from, To: st.ArtifactSHA256, Operator: operator,
	}
	if err := appendRelease(dataDir, rec); err != nil {
		return nil, err
	}
	return &rec, nil
}

// Abort 回滚：to 为空时丢弃候选；to 非空时从存档回退生效版本到该指纹
func Abort(dataDir, lockPath, material, to, operator string) (*ReleaseRecord, error) {
	if !isGateMaterial(material) {
		return nil, fmt.Errorf("物料 %s 不参与灰度发布", material)
	}
	lock, err := LoadLock(lockPath)
	if err != nil {
		return nil, err
	}

	var rec ReleaseRecord
	if to == "" {
		if !lock.RemoveStaging(material) {
			return nil, fmt.Errorf("物料 %s 无候选可丢弃", material)
		}
		if err := SaveLock(lockPath, lock); err != nil {
			return nil, err
		}
		if err := os.Remove(StagingPath(dataDir, material)); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("清理候选失败: %w", err)
		}
		rec = ReleaseRecord{Time: nowStamp(), Material: material, Action: "abort", Operator: operator, Note: "丢弃候选"}
	} else {
		def, ok := schema.Registry[material]
		if !ok {
			return nil, fmt.Errorf("未注册物料: %s", material)
		}
		hist, err := os.ReadFile(HistoryPath(dataDir, material, to))
		if err != nil {
			return nil, fmt.Errorf("无该版本存档: %w", err)
		}
		activePath := filepath.Join(dataDir, def.Source)
		from := ""
		if cur, err := os.ReadFile(activePath); err == nil {
			from = HashBytes(cur)
		}
		if err := writeFile(activePath, hist); err != nil {
			return nil, err
		}
		lock.UpsertEntry(LockEntry{Material: material, Generated: def.Source, ArtifactSHA256: to})
		lock.RemoveStaging(material) // 回退同时丢弃候选
		if err := SaveLock(lockPath, lock); err != nil {
			return nil, err
		}
		rec = ReleaseRecord{
			Time: nowStamp(), Material: material, Action: "abort",
			From: from, To: to, Operator: operator, Note: "回退到存档版本",
		}
	}
	if err := appendRelease(dataDir, rec); err != nil {
		return nil, err
	}
	return &rec, nil
}

// parseMaterialEntries 解析物料字节为条目集合（按物料结构分派）
func parseMaterialEntries(material string, b []byte) ([]string, error) {
	switch material {
	case schema.GateMaterial, schema.ForbiddenMaterial:
		var list []string
		if err := json.Unmarshal(b, &list); err != nil {
			return nil, fmt.Errorf("解析 %s 条目失败: %w", material, err)
		}
		return normalizeEntries(list), nil
	case schema.AllowlistMaterial:
		var f allowlistFile
		if err := json.Unmarshal(b, &f); err != nil {
			return nil, fmt.Errorf("解析 %s 条目失败: %w", material, err)
		}
		return normalizeEntries(concat(f.Chars, f.Combos, f.Blessings)), nil
	case schema.ComplianceRedlineMaterial:
		var f redlineFile
		if err := json.Unmarshal(b, &f); err != nil {
			return nil, fmt.Errorf("解析 %s 条目失败: %w", material, err)
		}
		return normalizeEntries(concat(f.StateOrgs, f.Places, f.Persons)), nil
	default:
		return nil, fmt.Errorf("物料 %s 不支持条目解析", material)
	}
}

// readMaterialEntries 读取 dataDir 下物料的生效条目集合
func readMaterialEntries(dataDir, material string) ([]string, error) {
	def, ok := schema.Registry[material]
	if !ok {
		return nil, fmt.Errorf("未注册物料: %s", material)
	}
	b, err := os.ReadFile(filepath.Join(dataDir, def.Source))
	if err != nil {
		return nil, fmt.Errorf("读取物料 %s 失败: %w", material, err)
	}
	return parseMaterialEntries(material, b)
}

// normalizeEntries 去首尾空白、跳过空值、去重并保持首现顺序
func normalizeEntries(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

// concat 顺序拼接多组字符串
func concat(groups ...[]string) []string {
	total := 0
	for _, g := range groups {
		total += len(g)
	}
	out := make([]string, 0, total)
	for _, g := range groups {
		out = append(out, g...)
	}
	return out
}

// appendRelease 追加一条发布记录到 data/release.jsonl（append-only）
func appendRelease(dataDir string, rec ReleaseRecord) error {
	line, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("序列化发布记录失败: %w", err)
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("创建发布日志目录失败: %w", err)
	}
	f, err := os.OpenFile(filepath.Join(dataDir, ReleaseFile), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("打开发布日志失败: %w", err)
	}
	defer func() { _ = f.Close() }()
	if _, err := f.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("写入发布日志失败: %w", err)
	}
	return nil
}

// nowStamp 本地时间戳（秒级，不含时区）
func nowStamp() string {
	return time.Now().Format("2006-01-02 15:04:05")
}
