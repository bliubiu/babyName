package services

// golden_names_test.go — golden names 快照回归（docs/32 §6.2）
//
// 目的：防止物料侧变更（P-1.2 字段补齐、P-1.4 gender 空值归一为 neutral、
// 后续 P1 归一化上移等）造成**静默评分漂移**。三组固定生辰的 Top10 快照
// 存放在 testdata/golden_names.json，任何排名/分数变化都必须逐条解释。
//
// 用法：
//
//	go test ./internal/application/services -run TestGoldenNames       # 比对
//	UPDATE_GOLDEN=1 go test ./internal/application/services -run TestGoldenNames  # 重建基线
//
// 注意：同一次进程内连续两次生成的结果必须一致（由本文件自我一致性断言守护）；
// 若跨进程冷启动出现差异，本测试会失败并给出逐条 diff —— 这正是其存在意义。

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"name/internal/domain/classics"
	"name/internal/domain/fate"
)

// goldenTopN 快照记录的每组名字条数
const goldenTopN = 10

// goldenName 快照中的单条名字记录
type goldenName struct {
	GivenName  string  `json:"given_name"`
	FullName   string  `json:"full_name"`
	Pinyin     string  `json:"pinyin"`
	TotalScore float64 `json:"total_score"`
	// GateHit 是否命中质量门禁字或禁忌组合（正常必须全为 false：
	// 生成链在候选池阶段已硬剔除，一旦为 true 说明门禁被破坏）
	GateHit bool `json:"gate_hit"`
}

// goldenCase 一组固定生辰输入
type goldenCase struct {
	ID         string       `json:"id"`
	Surname    string       `json:"surname"`
	Gender     string       `json:"gender"`
	BirthYear  int          `json:"birth_year"`
	BirthMonth int          `json:"birth_month"`
	BirthDay   int          `json:"birth_day"`
	BirthHour  int          `json:"birth_hour"`
	NameLength int          `json:"name_length"`
	Names      []goldenName `json:"names"`
}

// goldenFile 快照文件结构
type goldenFile struct {
	Version string       `json:"version"`
	Cases   []goldenCase `json:"cases"`
}

// goldenCases 三组固定生辰：覆盖 男/女 × 单名/双名 × 不同季节（喜用神不同）
func goldenCases() []goldenCase {
	return []goldenCase{
		{ID: "male-2024-spring-single", Surname: "王", Gender: "male",
			BirthYear: 2024, BirthMonth: 1, BirthDay: 15, BirthHour: 12, NameLength: 1},
		{ID: "female-2024-summer-double", Surname: "林", Gender: "female",
			BirthYear: 2024, BirthMonth: 6, BirthDay: 20, BirthHour: 8, NameLength: 2},
		{ID: "male-2023-winter-double", Surname: "陈", Gender: "male",
			BirthYear: 2023, BirthMonth: 11, BirthDay: 2, BirthHour: 14, NameLength: 2},
	}
}

// goldenGateHit 判断名字是否命中质量门禁（门禁字或禁忌组合）
func goldenGateHit(given string) bool {
	runes := []rune(given)
	for _, r := range runes {
		if fate.IsNonNamingChar(string(r)) {
			return true
		}
	}
	for i := 0; i+1 < len(runes); i++ {
		if fate.IsBadCombo(string(runes[i]), string(runes[i+1])) {
			return true
		}
	}
	return false
}

// round2 评分保留两位小数，消除浮点显示层抖动
func round2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}

// generateGoldenCase 跑一次生成并截取 TopN
func generateGoldenCase(t *testing.T, svc *FateNameService, c goldenCase) []goldenName {
	t.Helper()
	resp, err := svc.GenerateWithAnalysis(context.Background(), &GenerateRequest{
		Surname:     c.Surname,
		Gender:      c.Gender,
		BirthYear:   c.BirthYear,
		BirthMonth:  c.BirthMonth,
		BirthDay:    c.BirthDay,
		BirthHour:   c.BirthHour,
		BirthMinute: 0,
		NameLength:  c.NameLength,
	})
	if err != nil {
		t.Fatalf("用例 %s 生成失败: %v", c.ID, err)
	}
	if resp == nil || len(resp.Names) < goldenTopN {
		t.Fatalf("用例 %s 返回名字不足 %d 个（实得 %d 个），无法构成稳定快照",
			c.ID, goldenTopN, len(resp.Names))
	}

	out := make([]goldenName, 0, goldenTopN)
	for i, n := range resp.Names {
		if i >= goldenTopN {
			break
		}
		out = append(out, goldenName{
			GivenName:  n.GivenName,
			FullName:   n.FullName,
			Pinyin:     n.Pinyin,
			TotalScore: round2(n.TotalScore),
			GateHit:    goldenGateHit(n.GivenName),
		})
	}
	return out
}

// goldenSnapshotPath 快照文件路径（基于本文件位置定位，不依赖运行 CWD）
func goldenSnapshotPath() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "testdata", "golden_names.json")
}

// requireClassicCorpus 语料就绪守卫（fail-fast）
//
// data/ 不入 git（.gitignore），shijing.json/chuci.json 一旦丢失或损坏，
// 评分链静默降级（loadClassicSafe 吞错），快照只会以「莫名漂移」的形式失败——
// 2026.11.01 语料清空事故中差点被误判为规则变更而重建基线。
//
// ⚠ 必须在 setupFateNameServiceE2E **之后**调用：本函数触发的诗词索引惰性构建
// （poetryIdxOnce 包级一次性）若发生在 data.Init 前，索引会以空语料固化
// （Once 不再重跑），出典分整体丢失造成假漂移。
//
// 断言口径用 GetShijingPoetryChars/GetChuciPoetryChars 直读提取数据——
// 不用 HasPoetryChar("窈") / GetPoetryCharList("诗经")：二者都会先命中
// 硬编码 PoetrySources 名句库（「窈」「诗经」key 均存在），语料丢失后恒真，测不出事故。
func requireClassicCorpus(t *testing.T) {
	t.Helper()
	if len(classics.GetShijingPoetryChars()) == 0 {
		t.Fatal("诗经语料未就绪（提取用字 0 条）：data/shijing.json 缺失或为空。" +
			"请从 chinese-poetry 上游恢复语料后重跑，禁止直接 UPDATE_GOLDEN=1 重建基线掩盖数据事故")
	}
	if len(classics.GetChuciPoetryChars()) == 0 {
		t.Fatal("楚辞语料未就绪（提取用字 0 条）：data/chuci.json 缺失或为空。" +
			"请从 chinese-poetry 上游恢复语料后重跑，禁止直接 UPDATE_GOLDEN=1 重建基线掩盖数据事故")
	}
}

// TestGoldenNames_同进程可复现 前置契约：同进程内重复生成必须逐字节一致。
// 若不成立，快照比对本身失去意义（先失败在这里而不是 diff 出一堆噪音）。
func TestGoldenNames_同进程可复现(t *testing.T) {
	svc := setupFateNameServiceE2E(t)
	requireClassicCorpus(t)
	for _, c := range goldenCases() {
		first := generateGoldenCase(t, svc, c)
		second := generateGoldenCase(t, svc, c)
		for i := range first {
			if first[i] != second[i] {
				t.Fatalf("用例 %s 第 %d 位同进程重复生成不一致：\n首次=%+v\n二次=%+v",
					c.ID, i, first[i], second[i])
			}
		}
	}
}

// TestGoldenNames_快照比对 主用例：与 testdata/golden_names.json 逐组逐条比对。
//
// 基线重建：UPDATE_GOLDEN=1 go test ./internal/application/services -run TestGoldenNames_快照比对
// 重建必须在 commit message 中逐条说明变化原因（预期行为变更 or 缺陷）。
func TestGoldenNames_快照比对(t *testing.T) {
	svc := setupFateNameServiceE2E(t)
	requireClassicCorpus(t)
	path := goldenSnapshotPath()

	actual := goldenCases()
	for i := range actual {
		actual[i].Names = generateGoldenCase(t, svc, actual[i])
	}

	// 更新模式：重写基线
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		data, err := json.MarshalIndent(goldenFile{
			Version: "2026.10.14.1",
			Cases:   actual,
		}, "", "  ")
		if err != nil {
			t.Fatalf("序列化快照失败: %v", err)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("创建 testdata 目录失败: %v", err)
		}
		if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
			t.Fatalf("写快照文件失败: %v", err)
		}
		t.Logf("已重建 golden 快照：%s（%d 组）", path, len(actual))
		return
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取快照失败（首次运行请先设 UPDATE_GOLDEN=1 生成基线）: %v", err)
	}
	var want goldenFile
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatalf("解析快照失败: %v", err)
	}

	if len(want.Cases) != len(actual) {
		t.Fatalf("快照组数不符：快照 %d 组，当前 %d 组（输入契约被修改？）",
			len(want.Cases), len(actual))
	}

	for i, c := range actual {
		w := want.Cases[i]
		if w.ID != c.ID || w.Surname != c.Surname || w.Gender != c.Gender ||
			w.BirthYear != c.BirthYear || w.BirthMonth != c.BirthMonth ||
			w.BirthDay != c.BirthDay || w.NameLength != c.NameLength {
			t.Fatalf("第 %d 组输入契约与快照不符：\n快照=%+v\n当前=%+v", i, w, c)
		}
		if len(w.Names) != len(c.Names) {
			t.Fatalf("用例 %s 名字条数不符：快照 %d，当前 %d", c.ID, len(w.Names), len(c.Names))
		}
		for j := range c.Names {
			if c.Names[j] != w.Names[j] {
				var detail strings.Builder
				limit := j + 3
				if limit > len(c.Names) {
					limit = len(c.Names)
				}
				for k := j; k < limit; k++ {
					mark := " "
					if c.Names[k] != w.Names[k] {
						mark = "*"
					}
					detail.WriteString("\n  " + mark + " 快照: " + jsonString(w.Names[k]) +
						"\n    当前: " + jsonString(c.Names[k]))
				}
				t.Fatalf("用例 %s 第 %d 位起与快照不一致（%s）%s\n"+
					"  → 若为预期行为变更（如 gender 归一、字段补齐），请设 UPDATE_GOLDEN=1 重建基线，"+
					"并在 commit message 与 CHANGELOG 中逐条说明原因",
					c.ID, j, c.ID, detail.String())
			}
		}
		// 门禁守护：快照中的 GateHit 必须与当前一致，且正常恒为 false
		for j, n := range c.Names {
			if n.GateHit != w.Names[j].GateHit {
				t.Errorf("用例 %s 第 %d 位门禁标记变化：快照 %v，当前 %v（门禁行为被修改？）",
					c.ID, j, w.Names[j].GateHit, n.GateHit)
			}
			if n.GateHit {
				t.Errorf("用例 %s 第 %d 位 %q 命中质量门禁——候选池阶段本应硬剔除",
					c.ID, j, n.FullName)
			}
		}
	}
}

// jsonString 辅助：把结构序列化为紧凑字符串用于错误报告
func jsonString(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "序列化失败"
	}
	return string(b)
}
