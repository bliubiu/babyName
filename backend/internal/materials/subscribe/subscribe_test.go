package subscribe

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"name/internal/materials/schema"
)

func TestHashBytesAndFile(t *testing.T) {
	const want = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" // 空串 sha256
	if got := HashBytes(nil); got != want {
		t.Fatalf("HashBytes(nil) = %s, 期望空串 sha256", got)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(path, []byte("abc"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := HashFile(path)
	if err != nil {
		t.Fatalf("HashFile 出错: %v", err)
	}
	if len(got) != 64 {
		t.Fatalf("sha256 长度异常: %s", got)
	}

	if _, err := HashFile(filepath.Join(dir, "missing.txt")); err == nil {
		t.Fatal("文件不存在应返回错误")
	}
}

// TestWriteFile_原子替换不留半截文件 守护 docs/34 阶段 C1：
// 覆盖写必须「临时文件 → rename」，旧内容要么全在要么全不在；
// 成功后目标目录不得残留临时文件。
func TestWriteFile_原子替换不留半截文件(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "materials.lock")

	if err := writeFile(path, []byte("第一版内容")); err != nil {
		t.Fatalf("首次写入失败: %v", err)
	}
	if err := writeFile(path, []byte("第二版内容，更长")); err != nil {
		t.Fatalf("覆盖写入失败: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取目标失败: %v", err)
	}
	if string(got) != "第二版内容，更长" {
		t.Errorf("覆盖后内容应全为新值，实得 %q", got)
	}

	// 临时文件必须已随 rename 移走，不得残留
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != filepath.Base(path) {
			t.Errorf("目标目录残留了临时文件: %s", e.Name())
		}
	}

	// 权限归一为 0644（CreateTemp 默认 0600，rename 保留权限）。
	// Windows 无 POSIX 权限位（Go Chmod 仅能切只读属性），跳过该断言。
	if runtime.GOOS != "windows" {
		fi, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if fi.Mode().Perm() != 0o644 {
			t.Errorf("文件权限期望 0644，实得 %o", fi.Mode().Perm())
		}
	}
}

// TestWriteFile_父目录缺失时报错 覆盖原子写的失败分支：
// CreateTemp 在不存在目录上失败时必须返回错误，且不留半截目标文件
//（调用方对生效物料的写入失败应当显式可见，而不是被静默吞掉）。
func TestWriteFile_父目录缺失时报错(t *testing.T) {
	path := filepath.Join(t.TempDir(), "不存在子目录", "x.json")
	if err := writeFile(path, []byte("{}")); err == nil {
		t.Fatal("父目录不存在时应返回错误")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("失败写入不得留下目标文件，stat err=%v", err)
	}
}

// TestWriteFile_目标不存在时创建 原子写在首写场景同样成立
func TestWriteFile_目标不存在时创建(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "new.json")
	if err := writeFile(path, []byte("{}")); err != nil {
		t.Fatalf("写入失败: %v", err)
	}
	if b, err := os.ReadFile(path); err != nil || string(b) != "{}" {
		t.Errorf("首写内容异常: %q err=%v", b, err)
	}
}

func TestLockSaveLoadFind(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "materials.lock")
	l := &Lock{
		Version: "1.0",
		Note:    "测试",
		Entries: []LockEntry{
			{Material: "allowlist", Source: "https://x/y", Generated: "allowlist.json", ArtifactSHA256: "abc"},
		},
	}
	if err := SaveLock(path, l); err != nil {
		t.Fatalf("SaveLock 出错: %v", err)
	}

	loaded, err := LoadLock(path)
	if err != nil {
		t.Fatalf("LoadLock 出错: %v", err)
	}
	if loaded.Version != "1.0" || len(loaded.Entries) != 1 {
		t.Fatalf("roundtrip 异常: %+v", loaded)
	}
	if e, ok := loaded.Find("allowlist"); !ok || e.Source != "https://x/y" {
		t.Fatalf("Find 命中异常: %+v ok=%v", e, ok)
	}
	if _, ok := loaded.Find("nope"); ok {
		t.Fatal("Find 未命中应返回 false")
	}

	// 文件内容以换行结尾
	b, _ := os.ReadFile(path)
	if !strings.HasSuffix(string(b), "\n") {
		t.Fatal("lock 文件应以换行结尾")
	}

	if _, err := LoadLock(filepath.Join(dir, "missing.lock")); err == nil {
		t.Fatal("缺失 lock 应返回错误")
	}
	if err := os.WriteFile(filepath.Join(dir, "bad.lock"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadLock(filepath.Join(dir, "bad.lock")); err == nil {
		t.Fatal("非法 JSON 应返回错误")
	}
}

func TestLockVerify(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "good.json")
	if err := os.WriteFile(good, []byte("[]"), 0o644); err != nil {
		t.Fatal(err)
	}
	sum, _ := HashFile(good)

	l := &Lock{Entries: []LockEntry{
		{Material: "ok", Generated: "good.json", ArtifactSHA256: sum},
		{Material: "mismatch", Generated: "good.json", ArtifactSHA256: "deadbeef"},
		{Material: "missing", Generated: "nope.json", ArtifactSHA256: "x"},
		{Material: "abs", Generated: good, ArtifactSHA256: sum},
	}}
	got := l.Verify(dir)
	if len(got) != 2 {
		t.Fatalf("应报告 2 项不一致, 实际 %d: %+v", len(got), got)
	}
	reasons := map[string]string{}
	for _, m := range got {
		reasons[m.Material] = m.Reason
	}
	if reasons["mismatch"] != "指纹不匹配" || reasons["missing"] != "读取失败" {
		t.Fatalf("不一致原因异常: %+v", reasons)
	}
}

func TestParseLines(t *testing.T) {
	in := "a\r\n b \n\n# 注释\nc\na\n"
	got := ParseLines(in)
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("ParseLines = %v, 期望 %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("ParseLines = %v, 期望 %v", got, want)
		}
	}
	if len(ParseLines("")) != 0 {
		t.Fatal("空文本应返回空")
	}
}

func TestDiffLines(t *testing.T) {
	d := DiffLines([]string{"a", "b"}, []string{"b", "c"})
	if len(d.Added) != 1 || d.Added[0] != "c" {
		t.Fatalf("新增异常: %+v", d)
	}
	if len(d.Removed) != 1 || d.Removed[0] != "a" {
		t.Fatalf("移除异常: %+v", d)
	}
	if !DiffLines(nil, nil).Empty() {
		t.Fatal("空差分应为 Empty")
	}
	// 入参含重复时仍应去重
	d2 := DiffLines(nil, []string{"x", "x"})
	if len(d2.Added) != 1 {
		t.Fatalf("差分应去重: %+v", d2)
	}
}

func TestBuildPatch(t *testing.T) {
	got := BuildPatch("naming_gate", "", Diff{Added: []string{"a"}, Removed: []string{"b"}})
	if len(got) != 2 {
		t.Fatalf("补丁条数异常: %+v", got)
	}
	if got[0].Op != OpAdd || got[0].Entry != "a" || got[1].Op != OpRemove || got[1].Entry != "b" {
		t.Fatalf("补丁内容异常: %+v", got)
	}
	if got[0].Material != "naming_gate" {
		t.Fatalf("物料归属异常: %+v", got[0])
	}
}

func TestParsePatch(t *testing.T) {
	data := []byte("# 注释\n{\"material\":\"naming_gate\",\"op\":\"add\",\"entry\":\"啊\"}\n\n{\"material\":\"allowlist\",\"kind\":\"char\",\"op\":\"remove\",\"entry\":\"好\"}\n")
	got, err := ParsePatch(data)
	if err != nil {
		t.Fatalf("ParsePatch 出错: %v", err)
	}
	if len(got) != 2 || got[0].Op != OpAdd || got[1].Kind != "char" {
		t.Fatalf("解析结果异常: %+v", got)
	}

	bad := map[string]string{
		"JSON 错误": "{",
		"操作非法":    `{"material":"x","op":"upsert","entry":"a"}`,
		"缺物料":     `{"op":"add","entry":"a"}`,
		"缺条目":     `{"material":"x","op":"add"}`,
	}
	for name, in := range bad {
		if _, err := ParsePatch([]byte(in)); err == nil {
			t.Fatalf("%s 应报错", name)
		}
	}

	if got, err := ParsePatch([]byte("")); err != nil || len(got) != 0 {
		t.Fatalf("空补丁应返回空且无错: %v %v", got, err)
	}
}

func TestFormatPatchRoundTrip(t *testing.T) {
	entries := []PatchEntry{
		{Material: "naming_gate", Op: OpAdd, Entry: "啊"},
		{Material: "allowlist", Kind: "blessing", Op: OpRemove, Entry: "康宁"},
	}
	b, err := FormatPatch(entries)
	if err != nil {
		t.Fatalf("FormatPatch 出错: %v", err)
	}
	if !strings.HasSuffix(string(b), "\n") {
		t.Fatal("JSONL 应以换行结尾")
	}
	back, err := ParsePatch(b)
	if err != nil {
		t.Fatalf("回读出错: %v", err)
	}
	if len(back) != 2 || back[1].Kind != "blessing" {
		t.Fatalf("roundtrip 异常: %+v", back)
	}
}

func TestInvert(t *testing.T) {
	in := []PatchEntry{
		{Material: "m", Op: OpAdd, Entry: "a"},
		{Material: "m", Op: OpRemove, Entry: "b"},
	}
	got := Invert(in)
	if got[0].Op != OpRemove || got[1].Op != OpAdd {
		t.Fatalf("反向补丁异常: %+v", got)
	}
	if in[0].Op != OpAdd {
		t.Fatal("Invert 不应修改原切片")
	}
}

func TestApplyToSet(t *testing.T) {
	cur := []string{"a", "b", "b"}
	entries := []PatchEntry{
		{Op: OpAdd, Entry: "c"},
		{Op: OpAdd, Entry: "a"}, // 已存在：幂等
		{Op: OpRemove, Entry: "b"},
		{Op: OpRemove, Entry: "z"}, // 不存在：忽略
	}
	got := ApplyToSet(cur, entries)
	if len(got) != 2 || got[0] != "a" || got[1] != "c" {
		t.Fatalf("应用后集合异常: %v", got)
	}
}

func writeText(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestApplyToFileStringList(t *testing.T) {
	dir := t.TempDir()
	writeText(t, filepath.Join(dir, "naming_quality.json"), `["啊","吧"]`)
	writeText(t, filepath.Join(dir, "forbidden_combos.json"), `["七云"]`)

	entries := []PatchEntry{
		{Material: schema.GateMaterial, Op: OpAdd, Entry: "吗"},
		{Material: schema.GateMaterial, Op: OpRemove, Entry: "吧"},
		{Material: schema.ForbiddenMaterial, Op: OpAdd, Entry: "八云"},
		{Material: "other", Op: OpAdd, Entry: "无关"}, // 非本物料：忽略
	}
	if err := ApplyToFile(dir, schema.GateMaterial, entries); err != nil {
		t.Fatalf("应用 naming_gate 出错: %v", err)
	}
	if err := ApplyToFile(dir, schema.ForbiddenMaterial, entries); err != nil {
		t.Fatalf("应用 forbidden 出错: %v", err)
	}

	gateB, _ := os.ReadFile(filepath.Join(dir, "naming_quality.json"))
	if !strings.Contains(string(gateB), "吗") || strings.Contains(string(gateB), "吧") {
		t.Fatalf("naming_gate 应用异常: %s", gateB)
	}
	fbB, _ := os.ReadFile(filepath.Join(dir, "forbidden_combos.json"))
	if !strings.Contains(string(fbB), "八云") {
		t.Fatalf("forbidden 应用异常: %s", fbB)
	}
}

func TestApplyToFileAllowlist(t *testing.T) {
	dir := t.TempDir()
	writeText(t, filepath.Join(dir, "allowlist.json"), `{"version":"1.0","chars":[],"combos":[],"blessings":["去病"]}`)

	entries := []PatchEntry{
		{Material: schema.AllowlistMaterial, Kind: schema.AllowKindChar, Op: OpAdd, Entry: "好"},
		{Material: schema.AllowlistMaterial, Kind: schema.AllowKindCombo, Op: OpAdd, Entry: "去病"},
		{Material: schema.AllowlistMaterial, Kind: schema.AllowKindBlessing, Op: OpRemove, Entry: "去病"},
	}
	if err := ApplyToFile(dir, schema.AllowlistMaterial, entries); err != nil {
		t.Fatalf("应用 allowlist 出错: %v", err)
	}
	var f allowlistFile
	if err := readJSON(filepath.Join(dir, "allowlist.json"), &f); err != nil {
		t.Fatal(err)
	}
	if len(f.Chars) != 1 || f.Chars[0] != "好" {
		t.Fatalf("chars 异常: %+v", f.Chars)
	}
	if len(f.Combos) != 1 || f.Combos[0] != "去病" {
		t.Fatalf("combos 异常: %+v", f.Combos)
	}
	if len(f.Blessings) != 0 {
		t.Fatalf("blessings 应被移除: %+v", f.Blessings)
	}

	// 非法 kind
	bad := []PatchEntry{{Material: schema.AllowlistMaterial, Kind: "nope", Op: OpAdd, Entry: "x"}}
	if err := ApplyToFile(dir, schema.AllowlistMaterial, bad); err == nil {
		t.Fatal("非法 kind 应报错")
	}
}

func TestApplyToFileRedline(t *testing.T) {
	dir := t.TempDir()
	writeText(t, filepath.Join(dir, "compliance_redline.json"), `{"version":"1.0","state_orgs":["国务院"],"places":[],"persons":[]}`)

	entries := []PatchEntry{
		{Material: schema.ComplianceRedlineMaterial, Kind: schema.RedlineKindStateOrg, Op: OpAdd, Entry: "省政府"},
		{Material: schema.ComplianceRedlineMaterial, Kind: schema.RedlineKindPlace, Op: OpAdd, Entry: "北京"},
		{Material: schema.ComplianceRedlineMaterial, Kind: schema.RedlineKindPerson, Op: OpAdd, Entry: "秦桧"},
		{Material: schema.ComplianceRedlineMaterial, Kind: schema.RedlineKindStateOrg, Op: OpRemove, Entry: "国务院"},
	}
	if err := ApplyToFile(dir, schema.ComplianceRedlineMaterial, entries); err != nil {
		t.Fatalf("应用 redline 出错: %v", err)
	}
	var f redlineFile
	if err := readJSON(filepath.Join(dir, "compliance_redline.json"), &f); err != nil {
		t.Fatal(err)
	}
	if len(f.StateOrgs) != 1 || f.StateOrgs[0] != "省政府" {
		t.Fatalf("state_orgs 异常: %+v", f.StateOrgs)
	}
	if len(f.Places) != 1 || f.Places[0] != "北京" {
		t.Fatalf("places 异常: %+v", f.Places)
	}
	if len(f.Persons) != 1 || f.Persons[0] != "秦桧" {
		t.Fatalf("persons 异常: %+v", f.Persons)
	}

	bad := []PatchEntry{{Material: schema.ComplianceRedlineMaterial, Kind: "nope", Op: OpAdd, Entry: "x"}}
	if err := ApplyToFile(dir, schema.ComplianceRedlineMaterial, bad); err == nil {
		t.Fatal("非法 kind 应报错")
	}
}

func TestApplyToFileErrors(t *testing.T) {
	dir := t.TempDir()

	// 未注册物料
	if err := ApplyToFile(dir, "no_such_material", nil); err == nil {
		t.Fatal("未注册物料应报错")
	}
	// 已注册但不支持补丁的物料
	if err := ApplyToFile(dir, schema.YijingMaterial, nil); err == nil {
		t.Fatal("不支持补丁的物料应报错")
	}
	// 文件缺失
	if err := ApplyToFile(dir, schema.GateMaterial, nil); err == nil {
		t.Fatal("源文件缺失应报错")
	}
	// 文件 JSON 非法
	writeText(t, filepath.Join(dir, "naming_quality.json"), "{")
	if err := ApplyToFile(dir, schema.GateMaterial, nil); err == nil {
		t.Fatal("JSON 非法应报错")
	}
}

// TestWriteFileErrors 覆盖写盘失败（只读文件）的防御分支
func TestWriteFileErrors(t *testing.T) {
	dir := t.TempDir()

	lockPath := filepath.Join(dir, "materials.lock")
	writeText(t, lockPath, "{}")
	if err := os.Chmod(lockPath, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(lockPath, 0o644) })
	if err := SaveLock(lockPath, &Lock{Version: "1"}); err == nil {
		t.Fatal("只读文件 SaveLock 应失败")
	}

	gatePath := filepath.Join(dir, "naming_quality.json")
	writeText(t, gatePath, `["啊"]`)
	if err := os.Chmod(gatePath, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(gatePath, 0o644) })
	if err := ApplyToFile(dir, schema.GateMaterial, nil); err == nil {
		t.Fatal("只读物料文件写入应失败")
	}
}

// TestBuildLock 覆盖订阅清单生成（存在登记 / 缺失跳过 / 全缺失）
func TestBuildLock(t *testing.T) {
	dir := t.TempDir()
	writeText(t, filepath.Join(dir, "naming_quality.json"), `["啊"]`)
	writeText(t, filepath.Join(dir, "allowlist.json"), `{"version":"1.0","chars":[],"combos":[],"blessings":[]}`)
	// forbidden_combos.json / compliance_redline.json 故意缺失

	l, err := BuildLock(dir, "2026.10.16", map[string]string{schema.GateMaterial: "https://x/gate"})
	if err != nil {
		t.Fatalf("BuildLock 出错: %v", err)
	}
	if l.Version != "2026.10.16" || len(l.Entries) != 2 {
		t.Fatalf("清单异常: %+v", l)
	}
	gate, ok := l.Find(schema.GateMaterial)
	if !ok || gate.Source != "https://x/gate" || len(gate.ArtifactSHA256) != 64 {
		t.Fatalf("gate 条目异常: %+v ok=%v", gate, ok)
	}
	if _, ok := l.Find(schema.ComplianceRedlineMaterial); ok {
		t.Fatal("缺失物料应被跳过")
	}

	empty, err := BuildLock(t.TempDir(), "1.0", nil)
	if err != nil || len(empty.Entries) != 0 {
		t.Fatalf("全缺失应得到空清单: %+v err=%v", empty, err)
	}
}
