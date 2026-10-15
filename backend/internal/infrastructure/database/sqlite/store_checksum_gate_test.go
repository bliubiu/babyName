package sqlite

import (
	"testing"

	"name/internal/domain/hanzi"
)

// 守护 docs/31 ADR-014：物料指纹（checksum）是重建门控的唯一依据。
//
// 背景：实测 namer.json 的 version 是静态 "1.0"，内容变更时不更新，
// 用 version 做 diff 永不触发；且若指纹失配而静默跳过，物料修订就再也传不到库里。

func Test门控_首次写入记录指纹(t *testing.T) {
	s := newTestStore(t)
	setHanziData(t, map[string]hanzi.Hanzi{
		"永": {Char: "永", Pinyin: "yong3", PositiveScore: 10},
	})
	s.seedHanziData()

	if got := s.storedHanziChecksum(); got == "" {
		t.Fatal("重建成功后应写入物料指纹到 data_meta")
	}
}

func Test门控_指纹一致则跳过重建(t *testing.T) {
	s := newTestStore(t)
	data := map[string]hanzi.Hanzi{
		"永": {Char: "永", Pinyin: "yong3", PositiveScore: 10},
	}
	setHanziData(t, data)
	s.seedHanziData()

	// 直接改表，绕开重建：若门控生效，这次改动应被保留
	if _, err := s.writeDB.Exec("UPDATE hanzi_data SET positive_score = 999 WHERE char = '永'"); err != nil {
		t.Fatalf("构造探测标记失败: %v", err)
	}

	if !s.hanziUnchanged() {
		t.Fatal("指纹一致且表非空时应判定为未变更")
	}

	s.seedHanziData()
	if got := positiveScore(t, s, "永"); got != 999 {
		t.Errorf("指纹一致时应跳过重建（保留探测标记 999），实得 %d", got)
	}
}

func Test门控_物料变更则必须重建(t *testing.T) {
	s := newTestStore(t)
	setHanziData(t, map[string]hanzi.Hanzi{
		"永": {Char: "永", Pinyin: "yong3", PositiveScore: 10},
	})
	s.seedHanziData()

	if _, err := s.writeDB.Exec("UPDATE hanzi_data SET positive_score = 999 WHERE char = '永'"); err != nil {
		t.Fatalf("构造探测标记失败: %v", err)
	}

	// 物料内容变化 → 指纹必变 → 必须重建
	setHanziData(t, map[string]hanzi.Hanzi{
		"永": {Char: "永", Pinyin: "yong3", PositiveScore: 55},
	})

	if s.hanziUnchanged() {
		t.Fatal("物料变更后不应判定为未变更")
	}
	s.seedHanziData()
	if got := positiveScore(t, s, "永"); got != 55 {
		t.Errorf("物料变更后必须重建，实得 positive_score=%d（期望 55）", got)
	}
}

func Test门控_增字删字都会改变指纹(t *testing.T) {
	s := newTestStore(t)
	setHanziData(t, map[string]hanzi.Hanzi{
		"永": {Char: "永"},
		"水": {Char: "水"},
	})
	s.seedHanziData()
	base := currentHanziChecksum()

	// 删字
	setHanziData(t, map[string]hanzi.Hanzi{"永": {Char: "永"}})
	if currentHanziChecksum() == base {
		t.Error("删字后指纹必须变化")
	}

	// 增字
	setHanziData(t, map[string]hanzi.Hanzi{
		"永": {Char: "永"},
		"水": {Char: "水"},
		"山": {Char: "山"},
	})
	if currentHanziChecksum() == base {
		t.Error("增字后指纹必须变化")
	}
}

func Test门控_字段修订会改变指纹(t *testing.T) {
	base := currentHanziDataChecksumOf(map[string]hanzi.Hanzi{"永": {Char: "永", PositiveScore: 10}})

	for name, mutate := range map[string]func(hanzi.Hanzi) hanzi.Hanzi{
		"拼音":   func(h hanzi.Hanzi) hanzi.Hanzi { h.Pinyin = "yong4"; return h },
		"笔画":   func(h hanzi.Hanzi) hanzi.Hanzi { h.Strokes = 9; return h },
		"康熙笔画": func(h hanzi.Hanzi) hanzi.Hanzi { h.KangxiStrokes = 9; return h },
		"部首":   func(h hanzi.Hanzi) hanzi.Hanzi { h.Radical = "水"; return h },
		"五行":   func(h hanzi.Hanzi) hanzi.Hanzi { h.Wuxing = "金"; return h },
		"性别":   func(h hanzi.Hanzi) hanzi.Hanzi { h.Gender = "male"; return h },
		"等级":   func(h hanzi.Hanzi) hanzi.Hanzi { h.Level = 3; return h },
		"声调":   func(h hanzi.Hanzi) hanzi.Hanzi { h.Tone = 4; return h },
		"频率档":  func(h hanzi.Hanzi) hanzi.Hanzi { h.NameFreqTier = 5; return h },
		"多音字":  func(h hanzi.Hanzi) hanzi.Hanzi { h.IsPolyphonic = true; return h },
		"生僻":   func(h hanzi.Hanzi) hanzi.Hanzi { h.IsRare = true; return h },
		"消极":   func(h hanzi.Hanzi) hanzi.Hanzi { h.IsNegative = true; return h },
		"寓意评分": func(h hanzi.Hanzi) hanzi.Hanzi { h.PositiveScore = 11; return h },
		"起名分类": func(h hanzi.Hanzi) hanzi.Hanzi { h.NamingCategories = []string{"品德"}; return h },
	} {
		m := map[string]hanzi.Hanzi{"永": mutate(hanzi.Hanzi{Char: "永", PositiveScore: 10})}
		if currentHanziDataChecksumOf(m) == base {
			t.Errorf("修订「%s」后指纹必须变化", name)
		}
	}
}

func Test门控_指纹与键序无关(t *testing.T) {
	// map 遍历顺序随机，指纹必须稳定，否则每次启动都会误判为「已变更」
	m := map[string]hanzi.Hanzi{
		"永": {Char: "永", PositiveScore: 10},
		"水": {Char: "水", PositiveScore: 20},
		"山": {Char: "山", PositiveScore: 30},
	}
	first := currentHanziDataChecksumOf(m)
	for i := 0; i < 20; i++ {
		if got := currentHanziDataChecksumOf(m); got != first {
			t.Fatalf("指纹不稳定（第 %d 次不一致）：%s != %s", i, got, first)
		}
	}
}

func Test门控_表为空时不跳过(t *testing.T) {
	s := newTestStore(t)
	setHanziData(t, map[string]hanzi.Hanzi{"永": {Char: "永"}})
	s.seedHanziData()

	if _, err := s.writeDB.Exec("DELETE FROM hanzi_data"); err != nil {
		t.Fatalf("清空表失败: %v", err)
	}
	if s.hanziUnchanged() {
		t.Fatal("表为空时不得判定为未变更，否则物料永远灌不进库")
	}
	s.seedHanziData()
	if got := rowCount(t, s); got != 1 {
		t.Fatalf("空表后应重建出 1 行，实得 %d", got)
	}
}

func Test门控_无指纹记录时不跳过(t *testing.T) {
	s := newTestStore(t)
	setHanziData(t, map[string]hanzi.Hanzi{"永": {Char: "永"}})
	s.seedHanziData()

	if _, err := s.writeDB.Exec("DELETE FROM data_meta WHERE key = ?", metaKeyHanziChecksum); err != nil {
		t.Fatalf("清除指纹记录失败: %v", err)
	}
	if s.hanziUnchanged() {
		t.Fatal("无指纹记录时必须重建")
	}
}

// currentHanziDataChecksumOf 对给定字集计算指纹（不依赖全局态，供测试隔离使用）
func currentHanziDataChecksumOf(m map[string]hanzi.Hanzi) string {
	old := hanzi.HanziData
	hanzi.HanziData = m
	defer func() { hanzi.HanziData = old }()
	return currentHanziChecksum()
}
