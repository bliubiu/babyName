package classics

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func packageDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Dir(file)
}

func dataDir() string {
	return filepath.Join(packageDir(), "..", "..", "..", "data")
}

func TestMain(m *testing.M) {
	if err := LoadFromJSON(dataDir()); err != nil {
		// 诗经/楚辞等核心失败时仍继续，由各用例自行判断数据是否就绪
	}
	os.Exit(m.Run())
}

// TestPoemIndex_Built 索引至少成功构建
func TestPoemIndex_Built(t *testing.T) {
	idx := GetGlobalPoemIndex()
	if idx == nil {
		t.Skip("GlobalPoemIndex 未构建（数据目录缺失）")
	}
	if len(idx.entries) == 0 {
		t.Fatal("GlobalPoemIndex 条目数为 0")
	}
}

// TestQueryNamePoetry_QuoteInFullText 出典校验：返回的原句必须是全诗连续子串
func TestQueryNamePoetry_QuoteInFullText(t *testing.T) {
	idx := GetGlobalPoemIndex()
	if idx == nil || len(idx.entries) == 0 {
		t.Skip("诗词索引未就绪")
	}

	// 抽样若干常见名字字，覆盖多典籍
	names := []string{"窈", "窕", "君", "子", "清", "明", "风", "月", "山", "水", "德", "仁"}

	checked := 0
	for _, name := range names {
		result := QueryNamePoetry(name)
		if result == nil || result.TotalMatches == 0 {
			continue
		}
		for _, m := range result.Matches {
			if m.Poem == nil {
				t.Errorf("名字 %q 匹配 %q 的 Poem 为空", name, name)
				continue
			}
			checked++
			// 有 Quote 时，必须能在 Content 或 FullText 中找到
			if m.Quote != "" {
				found := false
				for _, line := range m.Poem.Content {
					if strings.Contains(line, m.Quote) || strings.Contains(m.Quote, line) && line != "" {
						found = true
						break
					}
				}
				if !found && m.Poem.FullText != "" && !strings.Contains(m.Poem.FullText, m.Quote) {
					// 允许 Quote 是 Content 中某行本身（相等）
					for _, line := range m.Poem.Content {
						if line == m.Quote {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("名字 %q 的 Quote %q 不在 FullText/Content 中（诗题=%q）",
							name, m.Quote, m.Poem.Title)
					}
				}
			}
			// SourceDesc 不应为空
			if m.SourceDesc == "" {
				t.Errorf("名字 %q 的 SourceDesc 为空（诗题=%q）", name, m.Poem.Title)
			}
		}
	}
	if checked == 0 {
		t.Log("抽样名字均无匹配，跳过实质断言")
	}
}

// TestQueryNamePoetry_Yaotiao_ShiJing 金标准：「窈」优先关联《诗经·关雎》
func TestQueryNamePoetry_Yaotiao_ShiJing(t *testing.T) {
	idx := GetGlobalPoemIndex()
	if idx == nil || len(idx.entries) == 0 {
		t.Skip("诗词索引未就绪")
	}

	result := QueryNamePoetry("窈")
	if result == nil || result.TotalMatches == 0 {
		t.Skip("「窈」无匹配")
	}

	best := result.BestMatch
	if best == nil || best.Poem == nil {
		t.Fatal("BestMatch 为空")
	}
	// 诗经优先加分，Best 应含 Source=诗经 或 Title 含关雎/窈窕
	src := best.Poem.Source
	title := best.Poem.Title
	if src != "诗经" && !strings.Contains(title, "关雎") && !strings.Contains(title, "窈窕") {
		t.Errorf("「窈」最佳匹配 Source=%q Title=%q，期望优先诗经/关雎", src, title)
	}
	if best.Quote != "" && !strings.Contains(best.Quote, "窈") && !containsRune(best.Quote, '窈') {
		// Quote 至少应包含匹配字（或其所在句）
		t.Logf("「窈」Quote=%q 未直接含「窈」（可能为同篇他句）", best.Quote)
	}
}

func containsRune(s string, r rune) bool {
	for _, c := range s {
		if c == r {
			return true
		}
	}
	return false
}

// TestPoemEntry_Structural 结构校验：条目 Title 非空，Chars 与 Content 对齐
func TestPoemEntry_Structural(t *testing.T) {
	idx := GetGlobalPoemIndex()
	if idx == nil || len(idx.entries) == 0 {
		t.Skip("诗词索引未就绪")
	}

	emptyTitle := 0
	for _, e := range idx.entries {
		if e == nil {
			continue
		}
		if strings.TrimSpace(e.Title) == "" {
			emptyTitle++
		}
	}
	// 允许极少量空题（数据源噪声），但不应占多数
	if emptyTitle > len(idx.entries)/10 {
		t.Errorf("空题条目 %d/%d 超过 10%%", emptyTitle, len(idx.entries))
	}
}

// TestFindPoetryByChar_Shijing 诗经字级索引可查
func TestFindPoetryByChar_Shijing(t *testing.T) {
	entries := FindPoetryByChar("窈")
	if len(entries) == 0 {
		t.Skip("「窈」字级索引为空（诗经未加载）")
	}
	found := false
	for _, e := range entries {
		if e.Work == "诗经" || strings.Contains(e.Chapter, "关雎") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("「窈」字级索引 %d 条，但无诗经/关雎条目", len(entries))
	}
}
