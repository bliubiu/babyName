package fate

import (
	"container/heap"
	"math/rand"
	"sort"
	"sync"
)

// ExcellentTable 容量常量
const (
	excellentTableCapacity = 10000 // 最大保留候选数
	maxShownNames          = 100   // 最大展示数
)

// ExcellentEntry 优秀名字条目
type ExcellentEntry struct {
	Char1     string             `json:"char1"`
	Char2     string             `json:"char2"`
	Score     float64            `json:"score"`
	Grade     string             `json:"grade"`
	WuXing1   string             `json:"wu_xing1"`
	WuXing2   string             `json:"wu_xing2"`
	Stroke1   int                `json:"stroke1,omitempty"`
	Stroke2   int                `json:"stroke2,omitempty"`
	HasPoetry bool               `json:"has_poetry"`
	Items     map[string]float64 `json:"items,omitempty"` // 各维度评分明细
}

// excellentMinHeap 最小堆，用于维护 Top-N
type excellentMinHeap []ExcellentEntry

func (h excellentMinHeap) Len() int            { return len(h) }
func (h excellentMinHeap) Less(i, j int) bool  { return h[i].Score < h[j].Score }
func (h excellentMinHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *excellentMinHeap) Push(x any)         { *h = append(*h, x.(ExcellentEntry)) }
func (h *excellentMinHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

// ExcellentTable 流式 Top-N 数据结构
//
// 从 fate-main 移植的核心数据结构:
// - 流式写入: 并发安全，min-heap 自动维护容量内的 Top-N
// - 去重: seen map 保证不重复
// - Finalize: 完成后排序 + 构建哈希索引
// - Explore: 随机采样，支持过滤 + 已展示去重
//
// 使用方式:
//
//	table := NewExcellentTable()
//	// 并发 goroutine 中调用:
//	table.TryPush(entry)
//	// 全部写入后:
//	table.Finalize()
//	top10 := table.TopN(10)
//	random3 := table.Explore(3, nil)
type ExcellentTable struct {
	mu      sync.RWMutex
	h       excellentMinHeap
	entries []ExcellentEntry
	index   map[string]int // Char1+Char2 → index
	shown   map[string]bool
	seen    map[string]bool
	cap     int
}

// NewExcellentTable 创建容量为 10000 的 ExcellentTable
func NewExcellentTable() *ExcellentTable {
	return &ExcellentTable{
		h:     make(excellentMinHeap, 0, excellentTableCapacity),
		shown: make(map[string]bool),
		seen:  make(map[string]bool),
		cap:   excellentTableCapacity,
	}
}

// TryPush 尝试推入一个名字条目
// 如果已存在相同 Char1+Char2 则跳过
// 未达到容量直接入堆；达到容量则与堆顶（最小分）比较，若高于则替换
func (t *ExcellentTable) TryPush(entry ExcellentEntry) {
	t.mu.Lock()
	defer t.mu.Unlock()

	key := entry.Char1 + entry.Char2
	if t.seen[key] {
		return
	}
	t.seen[key] = true

	if len(t.h) < t.cap {
		heap.Push(&t.h, entry)
		return
	}

	if entry.Score > t.h[0].Score {
		heap.Pop(&t.h)
		heap.Push(&t.h, entry)
	}
}

// Finalize 完成写入，排序并构建索引
// Finalize 后不可再调用 TryPush
func (t *ExcellentTable) Finalize() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.entries = make([]ExcellentEntry, len(t.h))
	copy(t.entries, t.h)
	sort.Slice(t.entries, func(i, j int) bool {
		return t.entries[i].Score > t.entries[j].Score
	})
	t.index = make(map[string]int, len(t.entries))
	for i, e := range t.entries {
		t.index[e.Char1+e.Char2] = i
	}
	// 释放堆和 seen 内存
	t.h = nil
	t.seen = nil
}

// HeapLen 返回当前堆中元素数量（Finalize 前调用）
func (t *ExcellentTable) HeapLen() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.h)
}

// TopN 返回评分最高的 N 个名字
func (t *ExcellentTable) TopN(n int) []ExcellentEntry {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if n > len(t.entries) {
		n = len(t.entries)
	}
	if n <= 0 {
		return nil
	}
	result := make([]ExcellentEntry, n)
	copy(result, t.entries[:n])
	return result
}

// Len 返回总条目数（Finalize 后调用）
func (t *ExcellentTable) Len() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.entries)
}

// Explore 随机采样 N 个名字，支持过滤和去重
// filter 为 nil 表示不过滤
func (t *ExcellentTable) Explore(count int, filter func(ExcellentEntry) bool) []ExcellentEntry {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.shown) >= maxShownNames {
		return nil
	}

	var eligible []int
	for i, e := range t.entries {
		key := e.Char1 + e.Char2
		if t.shown[key] {
			continue
		}
		if filter != nil && !filter(e) {
			continue
		}
		eligible = append(eligible, i)
	}

	rand.Shuffle(len(eligible), func(i, j int) {
		eligible[i], eligible[j] = eligible[j], eligible[i]
	})

	remaining := maxShownNames - len(t.shown)
	if count > remaining {
		count = remaining
	}
	if count > len(eligible) {
		count = len(eligible)
	}

	result := make([]ExcellentEntry, count)
	for i := 0; i < count; i++ {
		result[i] = t.entries[eligible[i]]
		t.shown[t.entries[eligible[i]].Char1+t.entries[eligible[i]].Char2] = true
	}
	return result
}

// ShownCount 返回已展示的条目数
func (t *ExcellentTable) ShownCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.shown)
}

// MarkShown 手动标记一个名字为已展示
func (t *ExcellentTable) MarkShown(char1, char2 string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.shown[char1+char2] = true
}

// FindEntry 按字查找条目
func (t *ExcellentTable) FindEntry(char1, char2 string) *ExcellentEntry {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if i, ok := t.index[char1+char2]; ok {
		return &t.entries[i]
	}
	return nil
}

// Entries 分页查询
func (t *ExcellentTable) Entries(offset, limit int) []ExcellentEntry {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if offset >= len(t.entries) {
		return nil
	}
	end := offset + limit
	if end > len(t.entries) {
		end = len(t.entries)
	}
	result := make([]ExcellentEntry, end-offset)
	copy(result, t.entries[offset:end])
	return result
}
