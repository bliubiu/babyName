package fate

import (
	"strings"
	"testing"
)

// ——— Aho-Corasick 多模式匹配（docs/33 门禁层 M8）——

func TestACTrie_MatchAny(t *testing.T) {
	trie := NewACTrie([]string{"he", "she", "hers", "his"})
	cases := []struct {
		text string
		want bool
	}{
		{"ushers", true},
		{"she", true},
		{"his", true},
		{"yours", false},
		{"", false},
		{"z", false},
	}
	for _, c := range cases {
		if got := trie.MatchAny(c.text); got != c.want {
			t.Errorf("MatchAny(%q) = %v, 期望 %v", c.text, got, c.want)
		}
	}
}

func TestACTrie_FindAll(t *testing.T) {
	trie := NewACTrie([]string{"he", "she", "hers", "his"})
	got := trie.FindAll("ushers")
	want := map[string]bool{"she": true, "he": true, "hers": true}
	if len(got) != len(want) {
		t.Fatalf("FindAll 命中数期望 %d，实得 %d: %v", len(want), len(got), got)
	}
	for _, p := range got {
		if !want[p] {
			t.Errorf("出现非预期命中: %q", p)
		}
	}
	if empty := trie.FindAll(""); empty != nil {
		t.Errorf("空文本应返回 nil，实得 %v", empty)
	}
}

func TestACTrie_去重与空模式(t *testing.T) {
	trie := NewACTrie([]string{"ab", "ab", "", "abc"})
	if got := trie.PatternCount(); got != 2 {
		t.Fatalf("去重后模式数期望 2，实得 %d", got)
	}
	if got := len(trie.FindAll("abc")); got != 2 {
		t.Fatalf("abc 应命中 ab 与 abc，实得 %d", got)
	}
}

func TestACTrie_中文组合(t *testing.T) {
	trie := NewACTrie([]string{"人民政府", "派出所", "白河"})
	cases := []struct {
		text string
		want bool
	}{
		{"张三", false},
		{"人民政府", true},
		{"去派出所", true},
		{"白河", true},
	}
	for _, c := range cases {
		if got := trie.MatchAny(c.text); got != c.want {
			t.Errorf("MatchAny(%q) = %v, 期望 %v", c.text, got, c.want)
		}
	}
}

func TestACTrie_边界(t *testing.T) {
	empty := NewACTrie(nil)
	if empty.PatternCount() != 0 || empty.MatchAny("abc") || empty.FindAll("abc") != nil {
		t.Error("空自动机应恒不命中")
	}
	var nilTrie *ACTrie
	if nilTrie.MatchAny("x") || nilTrie.FindAll("x") != nil || nilTrie.PatternCount() != 0 {
		t.Error("nil 自动机应安全返回零值")
	}
}

func TestMatchAnyEntity_边界(t *testing.T) {
	if matchAnyEntity("", NewACTrie([]string{"人民政府"})) {
		t.Error("空名字应不命中")
	}
	if matchAnyEntity("张三", nil) {
		t.Error("nil 自动机应不命中")
	}
}

func TestNewRedlineTrie_跳过单字(t *testing.T) {
	trie := newRedlineTrie(map[string]bool{"a": true, "ab": true, "人民政府": true})
	if got := trie.PatternCount(); got != 2 {
		t.Fatalf("单字应被跳过，模式数期望 2，实得 %d", got)
	}
	if trie.MatchAny("a") {
		t.Error("单字条目不应参与命中")
	}
	if !trie.MatchAny("ab") {
		t.Error("双字条目应参与命中")
	}
}

// buildBenchSet 生成 n 个互不相同的双字模式（性能对比用）
func buildBenchSet(n int) map[string]bool {
	set := make(map[string]bool, n)
	for i := 0; i < n; i++ {
		c := string(rune(0x4e00 + i%20000))
		set[c+c] = true
	}
	return set
}

// linearMatch 重构前的线性实现（性能对比基线）
func linearMatch(fullName string, set map[string]bool) bool {
	for entry := range set {
		if len([]rune(entry)) < 2 {
			continue
		}
		if strings.Contains(fullName, entry) {
			return true
		}
	}
	return false
}

func BenchmarkComplianceRedline_Linear(b *testing.B) {
	set := buildBenchSet(5000)
	text := "张伟安"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = linearMatch(text, set)
	}
}

func BenchmarkComplianceRedline_Trie(b *testing.B) {
	trie := newRedlineTrie(buildBenchSet(5000))
	text := "张伟安"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = trie.MatchAny(text)
	}
}
