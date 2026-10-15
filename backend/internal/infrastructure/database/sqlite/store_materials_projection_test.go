package sqlite

import (
	"os"
	"path/filepath"
	"testing"
)

// 本文件守护 docs/31 §11.9 第 11/12 项的落库语义：
//   - name_penalty / is_negative 必须从 namer.json 真值落进 hanzi_data（此前 SQL 路径恒零）；
//   - name_freq_tier 由 name_frequency 表派生，且「仅 namer.json 变更」时不得被静默清零。
//
// 这三列都会参与 fate 评分（NamePenalty/IsNegative/NameFreqTier），一旦在写库或
// 读库任一侧丢失，生成链路（走 SQL）与测名链路（走内存）就会算出不同总分，
// 违反「测名与生成同源同分」契约（adapters_p2_test.go 守护 SQL 读侧，本文件守护写侧）。

// queryInt 读取单列整型（用于落库断言）
func queryInt(t *testing.T, s *Store, query string) int {
	t.Helper()
	var v int
	if err := s.db.QueryRow(query).Scan(&v); err != nil {
		t.Fatalf("查询失败 %q: %v", query, err)
	}
	return v
}

// TestImport_策展评分字段落库 守护 name_penalty / is_negative 写库
func TestImport_策展评分字段落库(t *testing.T) {
	dir := writeMinimalMaterials(t)
	// 覆盖 namer.json：给「和」注入 namePenalty / isNegative 真值
	namer := `{
  "version": "1.0",
  "generated": "2026-01-02",
  "totalChars": 3,
  "chars": [
    {"char":"永","pinyin":"yong3","strokes":5,"radical":"水","wuxing":"水","level":1,"gender":"男"},
    {"char":"水","pinyin":"shui3","strokes":4,"radical":"水","wuxing":"水","level":1},
    {"char":"和","pinyin":"he2","strokes":8,"radical":"禾","wuxing":"","level":2,"isNegative":true,"namePenalty":30}
  ],
  "charGroups":[{"radical":"水","name":"水字旁","meaning":"","chars":["永","水"]}]
}`
	if err := os.WriteFile(filepath.Join(dir, "namer.json"), []byte(namer), 0o644); err != nil {
		t.Fatal(err)
	}

	s := newTestStore(t)
	importMinimal(t, s, dir, ImportOptions{})

	if got := queryInt(t, s, "SELECT is_negative FROM hanzi_data WHERE char = '和'"); got != 1 {
		t.Errorf("is_negative 期望 1（源真值 true），实得 %d", got)
	}
	if got := queryInt(t, s, "SELECT name_penalty FROM hanzi_data WHERE char = '和'"); got != 30 {
		t.Errorf("name_penalty 期望 30（源真值），实得 %d", got)
	}
	// 无真值字应为 0，不得被其它字段误填
	if got := queryInt(t, s, "SELECT name_penalty FROM hanzi_data WHERE char = '永'"); got != 0 {
		t.Errorf("「永」无 namePenalty 真值，期望 0，实得 %d", got)
	}
}

// TestImport_频率档位落库 守护 name_freq_tier 由频率物料合并进 hanzi_data
func TestImport_频率档位落库(t *testing.T) {
	dir := writeMinimalMaterials(t)
	s := newTestStore(t)
	importMinimal(t, s, dir, ImportOptions{})

	// 最小频率集：永 tier5 / 水 tier4 / 和 tier1（<2 不过滤）
	if got := queryInt(t, s, "SELECT name_freq_tier FROM hanzi_data WHERE char = '永'"); got != 5 {
		t.Errorf("「永」频率档期望 5，实得 %d", got)
	}
	if got := queryInt(t, s, "SELECT name_freq_tier FROM hanzi_data WHERE char = '水'"); got != 4 {
		t.Errorf("「水」频率档期望 4，实得 %d", got)
	}
	if got := queryInt(t, s, "SELECT name_freq_tier FROM hanzi_data WHERE char = '和'"); got != 0 {
		t.Errorf("Tier<2 不应合并（期望 0），实得 %d", got)
	}
}

// TestImport_仅namer变更时保留频率档位 守护「部分重导不丢派生列」
//
// 回归背景：频率档位此前只在 importCharFreq 内 UPDATE hanzi_data。若仅 namer.json
// 内容变更（频率源不变），importNamer 会重写全表 name_freq_tier=0，而 importCharFreq
// 因指纹未变被短路跳过 → 档位静默清零。修复后 importNamer 从 name_frequency 表回读合并。
func TestImport_仅namer变更时保留频率档位(t *testing.T) {
	dir := writeMinimalMaterials(t)
	s := newTestStore(t)
	importMinimal(t, s, dir, ImportOptions{})

	if got := queryInt(t, s, "SELECT name_freq_tier FROM hanzi_data WHERE char = '永'"); got != 5 {
		t.Fatalf("首轮「永」频率档期望 5，实得 %d", got)
	}

	// 仅改 namer.json 内容（generated + 永的 meaning 变化），频率源文件不动
	namer := `{
  "version": "1.0",
  "generated": "2026-01-03",
  "totalChars": 3,
  "chars": [
    {"char":"永","pinyin":"yong3","strokes":5,"radical":"水","meaning":"水流长远","wuxing":"水","level":1,"gender":"男"},
    {"char":"水","pinyin":"shui3","strokes":4,"radical":"水","wuxing":"水","level":1,"gender":"female"},
    {"char":"和","pinyin":"he2","strokes":8,"radical":"禾","wuxing":"","level":2}
  ],
  "charGroups":[{"radical":"水","name":"水字旁","meaning":"与水有关","chars":["永","水"]}]
}`
	if err := os.WriteFile(filepath.Join(dir, "namer.json"), []byte(namer), 0o644); err != nil {
		t.Fatal(err)
	}

	report := importMinimal(t, s, dir, ImportOptions{})
	if resultOf(report, "namer").Skipped {
		t.Fatal("namer 源已变更，不应被短路跳过")
	}
	if !resultOf(report, "name_frequency").Skipped {
		t.Fatal("频率源未变更，应按指纹短路跳过")
	}

	if got := queryInt(t, s, "SELECT name_freq_tier FROM hanzi_data WHERE char = '永'"); got != 5 {
		t.Errorf("仅 namer 变更后「永」频率档必须保留 5，实得 %d（派生列被静默清零）", got)
	}
}
