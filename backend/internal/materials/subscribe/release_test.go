package subscribe

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"name/internal/materials/schema"
)

// ——— 灰度发布（docs/33 门禁层 M7 / ADR-015）——

// allowlistJSON 生成 allowlist 物料 JSON 文本
func allowlistJSON(chars, combos, blessings []string) string {
	f := allowlistFile{Version: "1.0", Chars: chars, Combos: combos, Blessings: blessings}
	b, _ := json.MarshalIndent(f, "", "  ")
	return string(b) + "\n"
}

// setupRelease 建临时 dataDir（含生效 allowlist + materials.lock），返回路径
func setupRelease(t *testing.T) (dataDir, lockPath string) {
	t.Helper()
	dataDir = t.TempDir()
	writeText(t, filepath.Join(dataDir, "allowlist.json"), allowlistJSON([]string{"嘉"}, nil, nil))
	lockPath = filepath.Join(dataDir, "materials.lock")
	lock, err := BuildLock(dataDir, "1.0", nil)
	if err != nil {
		t.Fatalf("BuildLock 出错: %v", err)
	}
	if err := SaveLock(lockPath, lock); err != nil {
		t.Fatalf("SaveLock 出错: %v", err)
	}
	return dataDir, lockPath
}

// releaseLog 读取 data/release.jsonl 的全部记录
func releaseLog(t *testing.T, dataDir string) []ReleaseRecord {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dataDir, ReleaseFile))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatal(err)
	}
	var out []ReleaseRecord
	for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var r ReleaseRecord
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			t.Fatalf("解析 release 记录失败: %v", err)
		}
		out = append(out, r)
	}
	return out
}

func TestLock_两态往返与旧格式兼容(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "materials.lock")

	l := &Lock{
		Version: "1.0",
		Entries: []LockEntry{{Material: "allowlist", Generated: "allowlist.json", ArtifactSHA256: "aaa"}},
		Staging: []LockEntry{{Material: "allowlist", Generated: "staging/allowlist.json", ArtifactSHA256: "bbb"}},
	}
	if err := SaveLock(path, l); err != nil {
		t.Fatal(err)
	}
	got, err := LoadLock(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Staging) != 1 || got.Staging[0].ArtifactSHA256 != "bbb" {
		t.Fatalf("候选态未往返: %+v", got.Staging)
	}
	if e, ok := got.FindStaging("allowlist"); !ok || e.ArtifactSHA256 != "bbb" {
		t.Fatalf("FindStaging 异常: %+v ok=%v", e, ok)
	}
	if _, ok := got.FindStaging("nope"); ok {
		t.Fatal("FindStaging 未命中应 false")
	}

	// 覆盖同物料候选 / 新增其它物料候选 / 移除
	got.SetStaging(LockEntry{Material: "allowlist", ArtifactSHA256: "ccc"})
	if len(got.Staging) != 1 || got.Staging[0].ArtifactSHA256 != "ccc" {
		t.Fatalf("SetStaging 未覆盖: %+v", got.Staging)
	}
	got.SetStaging(LockEntry{Material: "naming_gate", ArtifactSHA256: "ddd"})
	if len(got.Staging) != 2 {
		t.Fatalf("SetStaging 新增异常: %+v", got.Staging)
	}
	if !got.RemoveStaging("allowlist") || len(got.Staging) != 1 {
		t.Fatalf("RemoveStaging 异常: %+v", got.Staging)
	}
	if got.RemoveStaging("allowlist") {
		t.Fatal("重复移除应返回 false")
	}

	// UpsertEntry：更新既有（保留来源）/ 新增
	got.Entries[0].Source = "https://x"
	got.UpsertEntry(LockEntry{Material: "allowlist", ArtifactSHA256: "eee"})
	if len(got.Entries) != 1 || got.Entries[0].ArtifactSHA256 != "eee" || got.Entries[0].Source != "https://x" {
		t.Fatalf("UpsertEntry 更新异常: %+v", got.Entries[0])
	}
	got.UpsertEntry(LockEntry{Material: "compliance_redline", ArtifactSHA256: "fff"})
	if len(got.Entries) != 2 {
		t.Fatalf("UpsertEntry 新增异常: %+v", got.Entries)
	}

	// 旧格式（无 staging 字段）兼容
	legacy := filepath.Join(dir, "legacy.lock")
	writeText(t, legacy, `{"version":"1.0","entries":[{"material":"allowlist","generated":"allowlist.json","artifact_sha256":"aaa"}]}`)
	old, err := LoadLock(legacy)
	if err != nil {
		t.Fatalf("旧格式应可加载: %v", err)
	}
	if len(old.Staging) != 0 {
		t.Fatalf("旧格式候选态应为空: %+v", old.Staging)
	}
}

func TestStage登记候选(t *testing.T) {
	dataDir, lockPath := setupRelease(t)
	candPath := filepath.Join(dataDir, "cand.json")
	writeText(t, candPath, allowlistJSON([]string{"嘉", "祥"}, nil, nil))

	rec, err := Stage(dataDir, lockPath, schema.AllowlistMaterial, candPath, "上游v2", "tester")
	if err != nil {
		t.Fatalf("Stage 出错: %v", err)
	}
	candBytes, _ := os.ReadFile(candPath)
	if rec.To != HashBytes(candBytes) || rec.Action != "stage" {
		t.Fatalf("候选指纹/动作异常: %+v", rec)
	}
	if _, err := os.Stat(StagingPath(dataDir, schema.AllowlistMaterial)); err != nil {
		t.Fatalf("候选文件未落地: %v", err)
	}
	lock, _ := LoadLock(lockPath)
	if st, ok := lock.FindStaging(schema.AllowlistMaterial); !ok || st.ArtifactSHA256 != rec.To {
		t.Fatalf("lock 候选未登记: %+v ok=%v", st, ok)
	}
	if logs := releaseLog(t, dataDir); len(logs) != 1 || logs[0].Action != "stage" {
		t.Fatalf("release 日志异常: %+v", logs)
	}

	// 错误路径
	if _, err := Stage(dataDir, lockPath, "nope", candPath, "", ""); err == nil {
		t.Fatal("非门禁物料应报错")
	}
	if _, err := Stage(dataDir, lockPath, schema.AllowlistMaterial, "", "", ""); err == nil {
		t.Fatal("缺候选路径应报错")
	}
	bad := filepath.Join(dataDir, "bad.json")
	writeText(t, bad, "{ not json")
	if _, err := Stage(dataDir, lockPath, schema.AllowlistMaterial, bad, "", ""); err == nil {
		t.Fatal("脏候选应报错")
	}
	if _, err := Stage(dataDir, lockPath, schema.AllowlistMaterial, filepath.Join(dataDir, "missing.json"), "", ""); err == nil {
		t.Fatal("候选文件缺失应报错")
	}
}

func TestObserve差异预演(t *testing.T) {
	dataDir, lockPath := setupRelease(t)
	if _, err := Observe(dataDir, lockPath, schema.AllowlistMaterial); err == nil {
		t.Fatal("无候选应报错")
	}
	if _, err := Observe(dataDir, lockPath, "nope"); err == nil {
		t.Fatal("非门禁物料应报错")
	}

	candPath := filepath.Join(dataDir, "cand.json")
	writeText(t, candPath, allowlistJSON([]string{"祥"}, []string{"康泰"}, nil))
	rec, err := Stage(dataDir, lockPath, schema.AllowlistMaterial, candPath, "", "")
	if err != nil {
		t.Fatal(err)
	}
	diff, err := Observe(dataDir, lockPath, schema.AllowlistMaterial)
	if err != nil {
		t.Fatalf("Observe 出错: %v", err)
	}
	if strings.Join(diff.Added, ",") != "祥,康泰" {
		t.Fatalf("新增异常: %v", diff.Added)
	}
	if strings.Join(diff.Removed, ",") != "嘉" {
		t.Fatalf("移除异常: %v", diff.Removed)
	}
	if diff.Candidate != rec.To || diff.Current == "" {
		t.Fatalf("指纹异常: %+v", diff)
	}
}

func TestPromote提升生效(t *testing.T) {
	dataDir, lockPath := setupRelease(t)
	if _, err := Promote(dataDir, lockPath, schema.AllowlistMaterial, ""); err == nil {
		t.Fatal("无候选应报错")
	}
	if _, err := Promote(dataDir, lockPath, "nope", ""); err == nil {
		t.Fatal("非门禁物料应报错")
	}

	oldBytes, _ := os.ReadFile(filepath.Join(dataDir, "allowlist.json"))
	oldSHA := HashBytes(oldBytes)

	candPath := filepath.Join(dataDir, "cand.json")
	writeText(t, candPath, allowlistJSON([]string{"嘉", "祥"}, nil, nil))
	if _, err := Stage(dataDir, lockPath, schema.AllowlistMaterial, candPath, "", ""); err != nil {
		t.Fatal(err)
	}
	rec, err := Promote(dataDir, lockPath, schema.AllowlistMaterial, "tester")
	if err != nil {
		t.Fatalf("Promote 出错: %v", err)
	}
	if rec.From != oldSHA {
		t.Fatalf("From 应为旧指纹: %s != %s", rec.From, oldSHA)
	}

	newBytes, _ := os.ReadFile(filepath.Join(dataDir, "allowlist.json"))
	if HashBytes(newBytes) != rec.To {
		t.Fatal("生效文件未替换为候选")
	}
	lock, _ := LoadLock(lockPath)
	if e, ok := lock.Find(schema.AllowlistMaterial); !ok || e.ArtifactSHA256 != rec.To {
		t.Fatalf("lock 生效指纹未更新: %+v", e)
	}
	if _, ok := lock.FindStaging(schema.AllowlistMaterial); ok {
		t.Fatal("提升后候选应清空")
	}
	if _, err := os.Stat(StagingPath(dataDir, schema.AllowlistMaterial)); !os.IsNotExist(err) {
		t.Fatal("提升后候选文件应删除")
	}
	if _, err := os.Stat(HistoryPath(dataDir, schema.AllowlistMaterial, oldSHA)); err != nil {
		t.Fatalf("旧版未存档: %v", err)
	}
	if logs := releaseLog(t, dataDir); len(logs) != 2 || logs[1].Action != "promote" {
		t.Fatalf("release 日志异常: %+v", logs)
	}
}

func TestAbort丢弃候选与回退(t *testing.T) {
	dataDir, lockPath := setupRelease(t)
	if _, err := Abort(dataDir, lockPath, schema.AllowlistMaterial, "", ""); err == nil {
		t.Fatal("无候选应报错")
	}
	if _, err := Abort(dataDir, lockPath, "nope", "", ""); err == nil {
		t.Fatal("非门禁物料应报错")
	}

	// 丢弃候选
	candPath := filepath.Join(dataDir, "cand.json")
	writeText(t, candPath, allowlistJSON([]string{"嘉", "祥"}, nil, nil))
	if _, err := Stage(dataDir, lockPath, schema.AllowlistMaterial, candPath, "", ""); err != nil {
		t.Fatal(err)
	}
	rec, err := Abort(dataDir, lockPath, schema.AllowlistMaterial, "", "tester")
	if err != nil {
		t.Fatalf("Abort 出错: %v", err)
	}
	if rec.To != "" {
		t.Fatalf("丢弃候选不应有 To: %+v", rec)
	}
	lock, _ := LoadLock(lockPath)
	if _, ok := lock.FindStaging(schema.AllowlistMaterial); ok {
		t.Fatal("候选应被清空")
	}
	if _, err := os.Stat(StagingPath(dataDir, schema.AllowlistMaterial)); !os.IsNotExist(err) {
		t.Fatal("候选文件应删除")
	}

	// 回退：v1 → v2（promote）→ v3（promote）→ 回退到 v2
	v2Path := filepath.Join(dataDir, "v2.json")
	writeText(t, v2Path, allowlistJSON([]string{"嘉", "祥"}, nil, nil))
	if _, err := Stage(dataDir, lockPath, schema.AllowlistMaterial, v2Path, "", ""); err != nil {
		t.Fatal(err)
	}
	v2rec, err := Promote(dataDir, lockPath, schema.AllowlistMaterial, "")
	if err != nil {
		t.Fatal(err)
	}

	v3Path := filepath.Join(dataDir, "v3.json")
	writeText(t, v3Path, allowlistJSON([]string{"嘉", "祥", "瑞"}, nil, nil))
	if _, err := Stage(dataDir, lockPath, schema.AllowlistMaterial, v3Path, "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := Promote(dataDir, lockPath, schema.AllowlistMaterial, ""); err != nil {
		t.Fatal(err)
	}

	if _, err := Abort(dataDir, lockPath, schema.AllowlistMaterial, "deadbeef", ""); err == nil {
		t.Fatal("不存在的存档应报错")
	}

	back, err := Abort(dataDir, lockPath, schema.AllowlistMaterial, v2rec.To, "tester")
	if err != nil {
		t.Fatalf("回退出错: %v", err)
	}
	if back.To != v2rec.To {
		t.Fatalf("回退目标异常: %+v", back)
	}
	b, _ := os.ReadFile(filepath.Join(dataDir, "allowlist.json"))
	if HashBytes(b) != v2rec.To {
		t.Fatal("生效文件未回退到 v2")
	}
	lock2, _ := LoadLock(lockPath)
	if e, ok := lock2.Find(schema.AllowlistMaterial); !ok || e.ArtifactSHA256 != v2rec.To {
		t.Fatalf("lock 未回退: %+v", e)
	}
}

func TestParseMaterialEntries(t *testing.T) {
	if got, err := parseMaterialEntries(schema.GateMaterial, []byte(`["a","a","b"]`)); err != nil || strings.Join(got, ",") != "a,b" {
		t.Fatalf("gate 解析异常: %v %v", got, err)
	}
	if got, err := parseMaterialEntries(schema.ForbiddenMaterial, []byte(`["x"]`)); err != nil || len(got) != 1 {
		t.Fatalf("forbidden 解析异常: %v %v", got, err)
	}
	got, err := parseMaterialEntries(schema.AllowlistMaterial, []byte(allowlistJSON([]string{"嘉"}, []string{"康泰"}, []string{"去病"})))
	if err != nil || strings.Join(got, ",") != "嘉,康泰,去病" {
		t.Fatalf("allowlist 解析异常: %v %v", got, err)
	}
	got, err = parseMaterialEntries(schema.ComplianceRedlineMaterial, []byte(`{"version":"1.0","state_orgs":["A"],"places":["B"],"persons":["C"]}`))
	if err != nil || strings.Join(got, ",") != "A,B,C" {
		t.Fatalf("redline 解析异常: %v %v", got, err)
	}
	if _, err := parseMaterialEntries("nope", []byte(`[]`)); err == nil {
		t.Fatal("未支持物料应报错")
	}
	for _, bad := range [][]byte{[]byte(`{`)} {
		if _, err := parseMaterialEntries(schema.GateMaterial, bad); err == nil {
			t.Fatal("gate 脏 JSON 应报错")
		}
		if _, err := parseMaterialEntries(schema.AllowlistMaterial, bad); err == nil {
			t.Fatal("allowlist 脏 JSON 应报错")
		}
		if _, err := parseMaterialEntries(schema.ComplianceRedlineMaterial, bad); err == nil {
			t.Fatal("redline 脏 JSON 应报错")
		}
	}
}

func TestReadMaterialEntries(t *testing.T) {
	dataDir, _ := setupRelease(t)
	got, err := readMaterialEntries(dataDir, schema.AllowlistMaterial)
	if err != nil || strings.Join(got, ",") != "嘉" {
		t.Fatalf("读取生效条目异常: %v %v", got, err)
	}
	if _, err := readMaterialEntries(dataDir, "nope"); err == nil {
		t.Fatal("未注册物料应报错")
	}
	if _, err := readMaterialEntries(dataDir, schema.GateMaterial); err == nil {
		t.Fatal("物料文件缺失应报错")
	}
}

func TestReleasePaths(t *testing.T) {
	if StagingPath("d", "allowlist") != filepath.Join("d", StagingDir, "allowlist.json") {
		t.Fatal("StagingPath 异常")
	}
	if HistoryPath("d", "allowlist", "abc") != filepath.Join("d", HistoryDir, "allowlist", "abc.json") {
		t.Fatal("HistoryPath 异常")
	}
}

// ——— 边界与防御路径 ———

func TestObserve_无生效指纹与候选缺失(t *testing.T) {
	dataDir := t.TempDir()
	writeText(t, filepath.Join(dataDir, "allowlist.json"), allowlistJSON([]string{"嘉"}, nil, nil))
	lockPath := filepath.Join(dataDir, "materials.lock")
	if err := SaveLock(lockPath, &Lock{
		Version: "1.0",
		Staging: []LockEntry{{Material: schema.AllowlistMaterial, ArtifactSHA256: "abc"}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(StagingPath(dataDir, schema.AllowlistMaterial)), 0o755); err != nil {
		t.Fatal(err)
	}
	writeText(t, StagingPath(dataDir, schema.AllowlistMaterial), allowlistJSON([]string{"祥"}, nil, nil))

	diff, err := Observe(dataDir, lockPath, schema.AllowlistMaterial)
	if err != nil {
		t.Fatalf("Observe 出错: %v", err)
	}
	if diff.Current != "" || diff.Candidate != "abc" {
		t.Fatalf("无生效记录时应只有候选指纹: %+v", diff)
	}

	if err := os.Remove(StagingPath(dataDir, schema.AllowlistMaterial)); err != nil {
		t.Fatal(err)
	}
	if _, err := Observe(dataDir, lockPath, schema.AllowlistMaterial); err == nil {
		t.Fatal("候选文件缺失应报错")
	}
	// lock 缺失
	if _, err := Observe(dataDir, filepath.Join(dataDir, "nope.lock"), schema.AllowlistMaterial); err == nil {
		t.Fatal("lock 缺失应报错")
	}
}

func TestPromote_首次发布与候选缺失(t *testing.T) {
	dataDir, lockPath := setupRelease(t)
	if _, err := Promote(dataDir, filepath.Join(dataDir, "nope.lock"), schema.AllowlistMaterial, ""); err == nil {
		t.Fatal("lock 缺失应报错")
	}
	// 首次发布：生效文件不存在 → From 为空、无旧版存档
	if err := os.Remove(filepath.Join(dataDir, "allowlist.json")); err != nil {
		t.Fatal(err)
	}
	candPath := filepath.Join(dataDir, "cand.json")
	writeText(t, candPath, allowlistJSON([]string{"嘉"}, nil, nil))
	if _, err := Stage(dataDir, lockPath, schema.AllowlistMaterial, candPath, "", ""); err != nil {
		t.Fatal(err)
	}
	rec, err := Promote(dataDir, lockPath, schema.AllowlistMaterial, "")
	if err != nil {
		t.Fatalf("首次发布 Promote 出错: %v", err)
	}
	if rec.From != "" {
		t.Fatalf("首次发布 From 应为空: %q", rec.From)
	}
	if _, err := os.Stat(HistoryPath(dataDir, schema.AllowlistMaterial, "")); !os.IsNotExist(err) {
		t.Fatal("首次发布不应生成空指纹存档")
	}

	// 候选文件缺失
	cand2 := filepath.Join(dataDir, "cand2.json")
	writeText(t, cand2, allowlistJSON([]string{"祥"}, nil, nil))
	if _, err := Stage(dataDir, lockPath, schema.AllowlistMaterial, cand2, "", ""); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(StagingPath(dataDir, schema.AllowlistMaterial)); err != nil {
		t.Fatal(err)
	}
	if _, err := Promote(dataDir, lockPath, schema.AllowlistMaterial, ""); err == nil {
		t.Fatal("候选文件缺失应报错")
	}
}

func TestPromote_生效路径为目录(t *testing.T) {
	dataDir, lockPath := setupRelease(t)
	cand := filepath.Join(dataDir, "cand.json")
	writeText(t, cand, allowlistJSON([]string{"嘉"}, nil, nil))
	if _, err := Stage(dataDir, lockPath, schema.AllowlistMaterial, cand, "", ""); err != nil {
		t.Fatal(err)
	}
	ap := filepath.Join(dataDir, "allowlist.json")
	if err := os.Remove(ap); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(ap, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Promote(dataDir, lockPath, schema.AllowlistMaterial, ""); err == nil {
		t.Fatal("生效路径为目录应报错")
	}
}

func TestAbort回退_生效文件缺失(t *testing.T) {
	dataDir, lockPath := setupRelease(t)
	v2 := filepath.Join(dataDir, "v2.json")
	writeText(t, v2, allowlistJSON([]string{"嘉", "祥"}, nil, nil))
	if _, err := Stage(dataDir, lockPath, schema.AllowlistMaterial, v2, "", ""); err != nil {
		t.Fatal(err)
	}
	first, err := Promote(dataDir, lockPath, schema.AllowlistMaterial, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dataDir, "allowlist.json")); err != nil {
		t.Fatal(err)
	}
	rec, err := Abort(dataDir, lockPath, schema.AllowlistMaterial, first.From, "")
	if err != nil {
		t.Fatalf("回退出错: %v", err)
	}
	if rec.From != "" {
		t.Fatalf("生效文件缺失时 From 应为空: %q", rec.From)
	}
}

func TestAbort_候选清理失败与lock缺失(t *testing.T) {
	dataDir := t.TempDir()
	lockPath := filepath.Join(dataDir, "materials.lock")
	if err := SaveLock(lockPath, &Lock{
		Version: "1.0",
		Staging: []LockEntry{{Material: schema.AllowlistMaterial, ArtifactSHA256: "x"}},
	}); err != nil {
		t.Fatal(err)
	}
	// 候选路径为非空目录 → 删除失败
	sp := StagingPath(dataDir, schema.AllowlistMaterial)
	if err := os.MkdirAll(sp, 0o755); err != nil {
		t.Fatal(err)
	}
	writeText(t, filepath.Join(sp, "keep.txt"), "x")
	if _, err := Abort(dataDir, lockPath, schema.AllowlistMaterial, "", ""); err == nil {
		t.Fatal("候选为非空目录应报错")
	}
	// lock 缺失
	if _, err := Abort(dataDir, filepath.Join(dataDir, "nope.lock"), schema.AllowlistMaterial, "", ""); err == nil {
		t.Fatal("lock 缺失应报错")
	}
}

func TestAbort回退_生效路径为目录(t *testing.T) {
	dataDir := t.TempDir()
	lockPath := filepath.Join(dataDir, "materials.lock")
	if err := SaveLock(lockPath, &Lock{
		Version: "1.0",
		Entries: []LockEntry{{Material: schema.AllowlistMaterial, Generated: "allowlist.json", ArtifactSHA256: "old"}},
	}); err != nil {
		t.Fatal(err)
	}
	hp := HistoryPath(dataDir, schema.AllowlistMaterial, "sha1")
	if err := os.MkdirAll(filepath.Dir(hp), 0o755); err != nil {
		t.Fatal(err)
	}
	writeText(t, hp, allowlistJSON([]string{"嘉"}, nil, nil))
	if err := os.MkdirAll(filepath.Join(dataDir, "allowlist.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Abort(dataDir, lockPath, schema.AllowlistMaterial, "sha1", ""); err == nil {
		t.Fatal("生效路径为目录应报错")
	}
}

func TestStage_目录占用防御(t *testing.T) {
	dataDir, lockPath := setupRelease(t)
	cand := filepath.Join(dataDir, "cand.json")
	writeText(t, cand, allowlistJSON([]string{"嘉"}, nil, nil))

	// 候选路径被占为目录 → 写入失败
	sp := StagingPath(dataDir, schema.AllowlistMaterial)
	if err := os.MkdirAll(sp, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Stage(dataDir, lockPath, schema.AllowlistMaterial, cand, "", ""); err == nil {
		t.Fatal("候选路径为目录应报错")
	}
	if err := os.Remove(sp); err != nil {
		t.Fatal(err)
	}

	// dataDir 为文件 → 创建候选目录失败
	fileDir := t.TempDir()
	afile := filepath.Join(fileDir, "afile")
	writeText(t, afile, "x")
	if _, err := Stage(afile, lockPath, schema.AllowlistMaterial, cand, "", ""); err == nil {
		t.Fatal("dataDir 为文件应报错")
	}

	// 发布日志为目录 → 追加记录失败
	logDir := filepath.Join(dataDir, ReleaseFile)
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Stage(dataDir, lockPath, schema.AllowlistMaterial, cand, "", ""); err == nil {
		t.Fatal("发布日志为目录应报错")
	}
}

func TestAppendRelease失败(t *testing.T) {
	fileDir := t.TempDir()
	afile := filepath.Join(fileDir, "afile")
	writeText(t, afile, "x")
	if err := appendRelease(afile, ReleaseRecord{Action: "stage"}); err == nil {
		t.Fatal("dataDir 为文件应报错")
	}

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ReleaseFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := appendRelease(dir, ReleaseRecord{Action: "stage"}); err == nil {
		t.Fatal("日志路径为目录应报错")
	}
}

func TestStage_lock缺失(t *testing.T) {
	dataDir, _ := setupRelease(t)
	cand := filepath.Join(dataDir, "cand.json")
	writeText(t, cand, allowlistJSON([]string{"嘉"}, nil, nil))
	if _, err := Stage(dataDir, filepath.Join(dataDir, "nope.lock"), schema.AllowlistMaterial, cand, "", ""); err == nil {
		t.Fatal("lock 缺失应报错")
	}
}

func TestObserve_物料与候选脏读(t *testing.T) {
	dataDir, lockPath := setupRelease(t)
	cand := filepath.Join(dataDir, "cand.json")
	writeText(t, cand, allowlistJSON([]string{"嘉"}, nil, nil))
	if _, err := Stage(dataDir, lockPath, schema.AllowlistMaterial, cand, "", ""); err != nil {
		t.Fatal(err)
	}

	// 生效物料文件缺失 → 读取失败
	ap := filepath.Join(dataDir, "allowlist.json")
	if err := os.Rename(ap, ap+".bak"); err != nil {
		t.Fatal(err)
	}
	if _, err := Observe(dataDir, lockPath, schema.AllowlistMaterial); err == nil {
		t.Fatal("生效物料缺失应报错")
	}
	if err := os.Rename(ap+".bak", ap); err != nil {
		t.Fatal(err)
	}

	// 候选文件脏 → 解析失败
	writeText(t, StagingPath(dataDir, schema.AllowlistMaterial), "{ not json")
	if _, err := Observe(dataDir, lockPath, schema.AllowlistMaterial); err == nil {
		t.Fatal("脏候选应报错")
	}
}

func TestPromote_存档失败(t *testing.T) {
	// 存档目录被文件占用 → 创建目录失败
	dataDir, lockPath := setupRelease(t)
	cand := filepath.Join(dataDir, "cand.json")
	writeText(t, cand, allowlistJSON([]string{"嘉"}, nil, nil))
	if _, err := Stage(dataDir, lockPath, schema.AllowlistMaterial, cand, "", ""); err != nil {
		t.Fatal(err)
	}
	writeText(t, filepath.Join(dataDir, HistoryDir), "x")
	if _, err := Promote(dataDir, lockPath, schema.AllowlistMaterial, ""); err == nil {
		t.Fatal("存档目录被文件占用应报错")
	}

	// 存档文件路径被目录占用 → 写入失败
	dataDir2, lockPath2 := setupRelease(t)
	cand2 := filepath.Join(dataDir2, "cand.json")
	writeText(t, cand2, allowlistJSON([]string{"嘉", "祥"}, nil, nil))
	if _, err := Stage(dataDir2, lockPath2, schema.AllowlistMaterial, cand2, "", ""); err != nil {
		t.Fatal(err)
	}
	oldBytes, _ := os.ReadFile(filepath.Join(dataDir2, "allowlist.json"))
	hp := HistoryPath(dataDir2, schema.AllowlistMaterial, HashBytes(oldBytes))
	if err := os.MkdirAll(hp, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Promote(dataDir2, lockPath2, schema.AllowlistMaterial, ""); err == nil {
		t.Fatal("存档文件被目录占用应报错")
	}
}

func TestPromote_发布日志为目录(t *testing.T) {
	dataDir, lockPath := setupRelease(t)
	cand := filepath.Join(dataDir, "cand.json")
	writeText(t, cand, allowlistJSON([]string{"嘉"}, nil, nil))
	if _, err := Stage(dataDir, lockPath, schema.AllowlistMaterial, cand, "", ""); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dataDir, ReleaseFile)); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dataDir, ReleaseFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Promote(dataDir, lockPath, schema.AllowlistMaterial, ""); err == nil {
		t.Fatal("发布日志为目录应报错")
	}
}

func TestAbort_发布日志为目录(t *testing.T) {
	dataDir, lockPath := setupRelease(t)
	cand := filepath.Join(dataDir, "cand.json")
	writeText(t, cand, allowlistJSON([]string{"嘉"}, nil, nil))
	if _, err := Stage(dataDir, lockPath, schema.AllowlistMaterial, cand, "", ""); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dataDir, ReleaseFile)); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dataDir, ReleaseFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Abort(dataDir, lockPath, schema.AllowlistMaterial, "", ""); err == nil {
		t.Fatal("发布日志为目录应报错")
	}
}

func TestRelease_只读lock写入失败(t *testing.T) {
	makeStaged := func(t *testing.T) (string, string) {
		t.Helper()
		dataDir, lockPath := setupRelease(t)
		cand := filepath.Join(dataDir, "cand.json")
		writeText(t, cand, allowlistJSON([]string{"嘉", "祥"}, nil, nil))
		if _, err := Stage(dataDir, lockPath, schema.AllowlistMaterial, cand, "", ""); err != nil {
			t.Fatalf("预置候选失败: %v", err)
		}
		return dataDir, lockPath
	}

	t.Run("Stage", func(t *testing.T) {
		dataDir, lockPath := setupRelease(t)
		cand := filepath.Join(dataDir, "cand.json")
		writeText(t, cand, allowlistJSON([]string{"嘉"}, nil, nil))
		if err := os.Chmod(lockPath, 0o444); err != nil {
			t.Fatal(err)
		}
		if _, err := Stage(dataDir, lockPath, schema.AllowlistMaterial, cand, "", ""); err == nil {
			t.Fatal("只读 lock 应导致 Stage 失败")
		}
	})

	t.Run("Promote", func(t *testing.T) {
		dataDir, lockPath := makeStaged(t)
		if err := os.Chmod(lockPath, 0o444); err != nil {
			t.Fatal(err)
		}
		if _, err := Promote(dataDir, lockPath, schema.AllowlistMaterial, ""); err == nil {
			t.Fatal("只读 lock 应导致 Promote 失败")
		}
	})

	t.Run("Abort丢弃", func(t *testing.T) {
		dataDir, lockPath := makeStaged(t)
		if err := os.Chmod(lockPath, 0o444); err != nil {
			t.Fatal(err)
		}
		if _, err := Abort(dataDir, lockPath, schema.AllowlistMaterial, "", ""); err == nil {
			t.Fatal("只读 lock 应导致 Abort 失败")
		}
	})

	t.Run("Abort回退", func(t *testing.T) {
		dataDir, lockPath := makeStaged(t)
		rec, err := Promote(dataDir, lockPath, schema.AllowlistMaterial, "")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(lockPath, 0o444); err != nil {
			t.Fatal(err)
		}
		if _, err := Abort(dataDir, lockPath, schema.AllowlistMaterial, rec.From, ""); err == nil {
			t.Fatal("只读 lock 应导致回退失败")
		}
	})
}
