package sqlite

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"name/internal/infrastructure/database"
)

// persist_bench_test.go 量化 /generate 链路的持久化写开销（常驻回归基准）。
//
// 动机（2026.10.25 性能深挖）：e2e 并发压测 16 并发 p95 5.18s（串行 0.846s
// 的 6.1×），需要定位串行段占比。本基准锁定链路上唯一的全串行段——
// SaveHistory 走单写连接（SetMaxOpenConns(1) + _txlock=immediate + WAL），
// 其耗时直接叠加在每个请求的尾延迟上。
//
// 首次实测（AMD 6800H / Windows / modernc sqlite，blob 对齐 e2e 双名基线
// 响应 293.9KB）：
//
//	SaveHistory 串行（synchronous=FULL）  9.49 ms/op
//	SaveHistory 并发（16 路）             9.83 ms/op（本就全串行，无额外退化）
//	SaveHistory blob 50/172/294/369KB   3.90/6.94/9.33/11.65 ms（≈3ms 固定 + ~22ms/MB）
//	BatchSaveHistory 16 条一批            1.55 ms/条（6.1× 便宜，一次事务一次 fsync）
//	SaveHistory synchronous=NORMAL       5.08 ms/op（−46%，WAL 推荐配置）
//
// 运行（在 backend 目录）：
//
//	go test ./internal/infrastructure/database/sqlite/ -run '^$' \
//	  -bench 'BenchmarkSaveHistory|BenchmarkBatchSaveHistory' -benchmem -benchtime 200x
//
// 注意：每次迭代都写库并落盘，-benchtime 用固定次数（如 200x）而非默认 1s，
// 避免把临时盘写满；临时库由 b.TempDir() 自动清理。

// benchBlob 生成指定大小的 results JSON blob（模拟 50 个名字的完整分析响应）
func benchBlob(sizeKB int) string {
	frag := `{"id":1,"surname":"张","given_name":"珀泽","full_name":"张珀泽","pinyin":"po ze","meaning":"指玉石的光彩与纹理，引申为珍贵、美好，古人以玉比德，故多用于形容品行高洁、温润如玉的君子","wuxing":"金水","strokes":32,"gender":"male","total_score":80.6,"wuxing_score":88,"yinyun_score":82,"meaning_score":90,"sancai_score":85,"zodiac_score":78,"poetry_source":"诗经","poetry_chapter":"卫风·淇奥","poetry_sentence":"有匪君子，如金如锡，如圭如璧","poetry_author":"佚名","poetry_dynasty":"先秦","poetry_full_text":"瞻彼淇奥，绿竹猗猗。有匪君子，如切如磋，如琢如磨。瑟兮僩兮，赫兮咺兮。有匪君子，终不可谖兮。","reasons":["五行相生","音韵和谐","寓意典雅"],"sancai_analysis":"天人地三才配置佳","hexagram":"泽天夬","hexagram_meaning":"决断而和悦"},`
	var sb strings.Builder
	sb.WriteString(`{"names":[`)
	for sb.Len() < sizeKB*1024 {
		sb.WriteString(frag)
	}
	sb.WriteString(`{}]}`)
	return sb.String()
}

func benchStore(b *testing.B) *Store {
	b.Helper()
	dir := b.TempDir()
	s, err := NewStore(filepath.Join(dir, "bench.db"), dir)
	if err != nil {
		b.Fatalf("创建测试 Store 失败: %v", err)
	}
	b.Cleanup(func() { _ = s.Close() })
	return s
}

func benchRecord(blob string) *database.HistoryRecord {
	return &database.HistoryRecord{
		Surname:       "张",
		Gender:        "male",
		BirthDate:     "2024-05-20",
		BirthTime:     "10:00",
		BirthLocation: "北京",
		Results:       blob,
	}
}

// BenchmarkSaveHistory_串行 单请求视角：每次生成后同步落库一次（生产路径）
func BenchmarkSaveHistory_串行(b *testing.B) {
	s := benchStore(b)
	blob := benchBlob(294) // 对齐 e2e 双名基线 293.9KB
	b.SetBytes(int64(len(blob)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if id := s.SaveHistory(benchRecord(blob)); id == "" {
			b.Fatal("SaveHistory 失败")
		}
	}
}

// BenchmarkSaveHistory_并发 并发视角：多请求同时落库（单写连接串行化）
func BenchmarkSaveHistory_并发(b *testing.B) {
	s := benchStore(b)
	blob := benchBlob(294)
	b.SetBytes(int64(len(blob)))
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if id := s.SaveHistory(benchRecord(blob)); id == "" {
				b.Fatal("SaveHistory 失败")
			}
		}
	})
}

// BenchmarkSaveHistory_blob尺寸 blob 大小敏感性（响应 KB → 写耗时）
func BenchmarkSaveHistory_blob尺寸(b *testing.B) {
	for _, kb := range []int{50, 172, 294, 369} {
		b.Run(fmt.Sprintf("%dKB", kb), func(b *testing.B) {
			s := benchStore(b)
			blob := benchBlob(kb)
			b.SetBytes(int64(len(blob)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if id := s.SaveHistory(benchRecord(blob)); id == "" {
					b.Fatal("SaveHistory 失败")
				}
			}
		})
	}
}

// BenchmarkBatchSaveHistory_16条一批 异步化「有界队列+批量写」方案的吞吐对照
func BenchmarkBatchSaveHistory_16条一批(b *testing.B) {
	s := benchStore(b)
	blob := benchBlob(294)
	recs := make([]*database.HistoryRecord, 16)
	for i := range recs {
		recs[i] = benchRecord(blob)
	}
	b.SetBytes(int64(len(blob) * 16))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if ids := s.BatchSaveHistory(recs); len(ids) != 16 {
			b.Fatal("BatchSaveHistory 失败")
		}
	}
}

// BenchmarkSaveHistory_synchronousNORMAL 候选改进对照：WAL 下关闭每提交 fsync
// （SQLite 对 WAL 的推荐配置；应用崩溃仍安全，仅 OS 掉电可能丢最近提交）
func BenchmarkSaveHistory_synchronousNORMAL(b *testing.B) {
	dir := b.TempDir()
	dbPath := filepath.Join(dir, "bench_normal.db")
	// 先用 NewStore 完成建表（history DDL 的唯一真源），关闭后以
	// synchronous(normal) DSN 重开，隔离「关闭每提交 fsync」这一个变量。
	if s, err := NewStore(dbPath, dir); err != nil {
		b.Fatalf("建库失败: %v", err)
	} else if err := s.Close(); err != nil {
		b.Fatalf("关闭失败: %v", err)
	}
	writeDB, err := sql.Open("sqlite", dbPath+"?_txlock=immediate&_pragma=busy_timeout(5000)&_pragma=synchronous(normal)")
	if err != nil {
		b.Fatalf("open 失败: %v", err)
	}
	b.Cleanup(func() { _ = writeDB.Close() })
	writeDB.SetMaxOpenConns(1)
	blob := benchBlob(294)
	b.SetBytes(int64(len(blob)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := writeDB.Exec(
			`INSERT INTO history (id, surname, gender, birth_date, birth_time, birth_location, results, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			fmt.Sprintf("bench-%d", i), "张", "male", "2024-05-20", "10:00", "北京", blob, time.Now(),
		); err != nil {
			b.Fatal(err)
		}
	}
}

// TestBlobSize 打印 blob 实际字节数，确认与 e2e 响应规模对齐
func TestBlobSize(t *testing.T) {
	for _, kb := range []int{50, 172, 294, 369} {
		t.Logf("benchBlob(%dKB) 实际 %d 字节", kb, len(benchBlob(kb)))
	}
	_ = time.Now
}
