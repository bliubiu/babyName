package sqlite

import (
	"os"
	"path/filepath"
	"testing"

	"name/internal/domain/hanzi"
)

// 本文件守护 docs/31 ADR-013：康熙笔画必须有一条「不依赖 CSV」的运行时装载路径。
//
// 修订前的状态是设计空洞：ADR-008 要求运行时零 JSON/零 CSV，
// 但康熙既不落库也无表，运行时没有来源。

// writeTestKangxiCSV 生成一份最小可解析的康熙笔画 CSV
// 结构需与真实文件一致：BOM + 4 行 MIT 注释 + header + 数据
func writeTestKangxiCSV(t *testing.T, dir string, data map[string]int) string {
	t.Helper()
	raw := dir + string(os.PathSeparator) + "raw"
	if err := os.MkdirAll(raw, 0o755); err != nil {
		t.Fatalf("创建 raw 目录失败: %v", err)
	}
	path := filepath.Join(raw, "kangxi-strokecount.csv")

	content := "\xEF\xBB\xBF" + // BOM
		"MIT License\n" +
		"Copyright (c) Kawai Lo\n" +
		"https://github.com/breezyreeds/kangxi-strokecount\n" +
		"\n" +
		"CodePoint,Value,Character,Strokes\n"
	cp := 0x4E00
	for char, strokes := range data {
		content += itoa(cp) + "," + itoa(int([]rune(char)[0])) + "," + char + "," + itoa(strokes) + "\n"
		cp++
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("写入测试康熙笔画 CSV 失败: %v", err)
	}
	return path
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		return "-" + string(b)
	}
	return string(b)
}

func Test康熙笔画_入库后可从库载入(t *testing.T) {
	dir := t.TempDir()
	want := map[string]int{"永": 5, "水": 4, "龍": 16}
	writeTestKangxiCSV(t, dir, want)

	s, err := NewStore(filepath.Join(dir, "test.db"), dir)
	if err != nil {
		t.Fatalf("创建 Store 失败: %v", err)
	}
	defer func() { _ = s.Close() }()

	// 物料写入只经由 import/seedKangxiStrokes 路径（ADR-008：运行时不再回写库）
	s.seedKangxiStrokes(dir)

	var n int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM kangxi_strokes").Scan(&n); err != nil {
		t.Fatalf("康熙笔画表查询失败: %v", err)
	}
	if n != len(want) {
		t.Fatalf("康熙笔画入库条数期望 %d，实得 %d", len(want), n)
	}

	// 清空内存集合后从库载入
	hanzi.SetKangxiStrokes(map[string]int{})
	if got := hanzi.KangxiStrokesCount(); got != 0 {
		t.Fatalf("载入前集合应为空，实得 %d 条", got)
	}

	if err := s.LoadKangxiStrokesFromDB(); err != nil {
		t.Fatalf("从库载入康熙笔画失败: %v", err)
	}

	if !hanzi.IsKangxiStrokesLoaded() {
		t.Fatal("从库载入后应标记为已加载")
	}
	for char, strokes := range want {
		if got := hanzi.GetKangxiStrokes(char); got != strokes {
			t.Errorf("字 %q 康熙笔画期望 %d，实得 %d", char, strokes, got)
		}
	}
}

func Test康熙笔画_表缺失时降级不崩(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(filepath.Join(dir, "test.db"), dir)
	if err != nil {
		t.Fatalf("创建 Store 失败: %v", err)
	}
	defer func() { _ = s.Close() }()

	// 模拟旧库：删掉康熙笔画表
	if _, err := s.writeDB.Exec("DROP TABLE IF EXISTS kangxi_strokes"); err != nil {
		t.Fatalf("删除康熙笔画表失败: %v", err)
	}

	hanzi.SetKangxiStrokes(map[string]int{"永": 5})
	if err := s.LoadKangxiStrokesFromDB(); err != nil {
		t.Fatalf("表缺失时应降级返回 nil，实得错误: %v", err)
	}
	if got := hanzi.GetKangxiStrokes("永"); got != 0 {
		t.Errorf("表缺失降级后应回落 0（按简体笔画兜底），实得 %d", got)
	}
	if !hanzi.IsKangxiStrokesLoaded() {
		t.Error("降级路径也应标记为已加载，避免上层反复触发加载")
	}
}

func Test康熙笔画_CSV缺失时不阻断启动(t *testing.T) {
	dir := t.TempDir() // 故意不创建 CSV
	s, err := NewStore(filepath.Join(dir, "test.db"), dir)
	if err != nil {
		t.Fatalf("CSV 缺失时不应导致启动失败: %v", err)
	}
	_ = s.Close()
}

func Test康熙笔画_重复入库幂等(t *testing.T) {
	dir := t.TempDir()
	want := map[string]int{"永": 5, "水": 4}
	writeTestKangxiCSV(t, dir, want)

	s, err := NewStore(filepath.Join(dir, "test.db"), dir)
	if err != nil {
		t.Fatalf("创建 Store 失败: %v", err)
	}
	defer func() { _ = s.Close() }()

	s.seedKangxiStrokes(dir)
	s.seedKangxiStrokes(dir)

	var n int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM kangxi_strokes").Scan(&n); err != nil {
		t.Fatalf("康熙笔画表查询失败: %v", err)
	}
	if n != len(want) {
		t.Fatalf("重复入库后条数期望 %d，实得 %d", len(want), n)
	}
}

func Test康熙笔画_CSV解析与运行时共用实现(t *testing.T) {
	dir := t.TempDir()
	want := map[string]int{"永": 5, "水": 4}
	path := writeTestKangxiCSV(t, dir, want)

	got, err := hanzi.ParseKangxiCSVFile(path)
	if err != nil {
		t.Fatalf("解析康熙笔画 CSV 失败: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("解析条数期望 %d，实得 %d", len(want), len(got))
	}
	for char, strokes := range want {
		if got[char] != strokes {
			t.Errorf("字 %q 笔画期望 %d，实得 %d", char, strokes, got[char])
		}
	}
}
