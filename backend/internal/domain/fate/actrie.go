package fate

// actrie.go Aho-Corasick 多模式匹配（docs/33 门禁层 M8）
//
// 目的：门禁词组（合规红线等）随物料持续扩增，原「逐模式 strings.Contains」
// 为 O(模式数 × 文本长)，在大词表/长文本下退化为线性扫描。改以 Aho-Corasick
// 构建一次、按 O(文本长 + 命中数) 扫描，一次走查即得全部命中模式。
//
// 纯 Go 实现、无第三方依赖（AGENTS.md：纯 Go 无 CGO）；构建后只读，可并发 Match。

// acNode 自动机单节点
type acNode struct {
	children map[rune]int // 字符 → 子节点下标
	fail     int          // 失配指针
	outputs  []string     // 以本节点结尾的模式（含沿 fail 链继承的后缀模式）
}

// ACTrie Aho-Corasick 自动机（构建后只读）
type ACTrie struct {
	nodes    []acNode
	patterns map[string]bool // 去重后的模式集（诊断/计数）
}

// NewACTrie 由模式集构建自动机
//
// 空模式与重复模式忽略；返回的自动机只读，可并发 Match。
func NewACTrie(patterns []string) *ACTrie {
	t := &ACTrie{
		nodes:    []acNode{{children: map[rune]int{}}},
		patterns: make(map[string]bool),
	}
	for _, p := range patterns {
		if p == "" || t.patterns[p] {
			continue
		}
		t.patterns[p] = true
		node := 0
		for _, r := range p {
			next, ok := t.nodes[node].children[r]
			if !ok {
				t.nodes = append(t.nodes, acNode{children: map[rune]int{}})
				next = len(t.nodes) - 1
				t.nodes[node].children[r] = next
			}
			node = next
		}
		t.nodes[node].outputs = append(t.nodes[node].outputs, p)
	}
	t.buildFail()
	return t
}

// buildFail 广度优先构建失配指针，并把 fail 链上的模式并入 outputs
func (t *ACTrie) buildFail() {
	queue := make([]int, 0, len(t.nodes))
	for _, child := range t.nodes[0].children {
		t.nodes[child].fail = 0
		queue = append(queue, child)
	}
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		for r, child := range t.nodes[node].children {
			fail := t.nodes[node].fail
			for {
				if next, ok := t.nodes[fail].children[r]; ok {
					fail = next
					break
				}
				if fail == 0 {
					break
				}
				fail = t.nodes[fail].fail
			}
			t.nodes[child].fail = fail
			t.nodes[child].outputs = append(t.nodes[child].outputs, t.nodes[fail].outputs...)
			queue = append(queue, child)
		}
	}
}

// MatchAny 判断 text 是否命中任一模式
func (t *ACTrie) MatchAny(text string) bool {
	if t == nil || t.empty() || text == "" {
		return false
	}
	return len(t.scan(text, nil)) > 0
}

// FindAll 返回 text 中出现的全部模式（去重，保持首次命中序）
func (t *ACTrie) FindAll(text string) []string {
	if t == nil || t.empty() || text == "" {
		return nil
	}
	return t.scan(text, make(map[string]bool))
}

// scan 在 text 上跑自动机；seen 为 nil 时遇到首个命中即返回（供 MatchAny 短路）
func (t *ACTrie) scan(text string, seen map[string]bool) []string {
	var out []string
	state := 0
	for _, r := range text {
		for {
			if next, ok := t.nodes[state].children[r]; ok {
				state = next
				break
			}
			if state == 0 {
				break
			}
			state = t.nodes[state].fail
		}
		for _, p := range t.nodes[state].outputs {
			if seen == nil {
				return append(out, p)
			}
			if !seen[p] {
				seen[p] = true
				out = append(out, p)
			}
		}
	}
	return out
}

// PatternCount 返回去重后的模式数
func (t *ACTrie) PatternCount() int {
	if t == nil {
		return 0
	}
	return len(t.patterns)
}

// empty 判断自动机是否无任何模式（仅根节点）
func (t *ACTrie) empty() bool {
	return len(t.patterns) == 0
}
