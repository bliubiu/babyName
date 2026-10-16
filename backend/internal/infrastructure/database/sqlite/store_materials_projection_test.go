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
//
// name_freq_tier 的归一化点已按 docs/34 阶段 A3 收敛到 Import() 末尾的
// syncNameFreqTier（以 name_frequency 表为唯一真源），不再由 importNamer
// 回读 + importCharFreq UPDATE 两处维护。

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

// TestImport_频率档位落库 守护 name_freq_tier 由频率物料派生进 hanzi_data
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
// 回归背景：频率档位此前分散在 importNamer 回读与 importCharFreq 内 UPDATE
// hanzi_data 两处。若仅 namer.json 内容变更（频率源不变），任一处失效都会让
// name_freq_tier 被静默清零。修复后归一化收敛到 Import() 末尾的
// syncNameFreqTier，以库内 name_frequency 表为唯一真源，与导入顺序、
// 是否被指纹短路无关。
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

// TestImport_全部短路时仍归一化档位 守护 A3 的「末态归一化」语义
//
// 归一化点收敛到 Import() 末尾后，它与「本次是否有物料真正落库」解耦：
// 即便所有物料都因指纹未变被短路，也要以 name_frequency 表为唯一真源
// 重算派生列。修订前指纹全匹配即整体 return，库内的历史脏档位会永久残留。
func TestImport_全部短路时仍归一化档位(t *testing.T) {
	dir := writeMinimalMaterials(t)
	s := newTestStore(t)
	importMinimal(t, s, dir, ImportOptions{})

	// 人为破坏派生列，模拟历史脏数据
	if _, err := s.writeDB.Exec("UPDATE hanzi_data SET name_freq_tier = 3 WHERE char = '和'"); err != nil {
		t.Fatalf("构造脏档位失败: %v", err)
	}

	report := importMinimal(t, s, dir, ImportOptions{})
	if report.Written != 0 {
		t.Errorf("二次导入应全部短路，实得写入 %d 项", report.Written)
	}

	if got := queryInt(t, s, "SELECT name_freq_tier FROM hanzi_data WHERE char = '和'"); got != 0 {
		t.Errorf("全部短路时末态归一化也应把「和」修回 0，实得 %d", got)
	}
	if got := queryInt(t, s, "SELECT name_freq_tier FROM hanzi_data WHERE char = '永'"); got != 5 {
		t.Errorf("「永」档位应保持 5，实得 %d", got)
	}
}

// TestImport_频率档下调即清零 守护 A3 的双向传导
//
// 单一真源派生列必须能双向传导：字频档位上调要回填，下调（低于 Tier ≥ 2
// 口径）或移出频率表要归零，否则 hanzi_data 会残留过期档位。
func TestImport_频率档下调即清零(t *testing.T) {
	dir := writeMinimalMaterials(t)
	s := newTestStore(t)
	importMinimal(t, s, dir, ImportOptions{})

	// 频率源变更：「永」tier 5 → 1，「水」tier 4 → 3，「和」保持 1
	freq := `{"meta":{"total_names":100,"built_at":"2026-01-02"},"frequency":[
      {"char":"永","count":10,"rank":9,"tier":1},
      {"char":"水","count":200,"rank":2,"tier":3},
      {"char":"和","count":10,"rank":9,"tier":1}
    ]}`
	if err := os.WriteFile(filepath.Join(dir, "name_frequency.json"), []byte(freq), 0o644); err != nil {
		t.Fatal(err)
	}

	report := importMinimal(t, s, dir, ImportOptions{})
	if resultOf(report, "name_frequency").Skipped {
		t.Fatal("频率源已变更，不应被短路跳过")
	}

	if got := queryInt(t, s, "SELECT name_freq_tier FROM hanzi_data WHERE char = '永'"); got != 0 {
		t.Errorf("「永」档位下调到 Tier 1 后应归零，实得 %d", got)
	}
	if got := queryInt(t, s, "SELECT name_freq_tier FROM hanzi_data WHERE char = '水'"); got != 3 {
		t.Errorf("「水」档位应更新为 3，实得 %d", got)
	}
}
