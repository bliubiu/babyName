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
	Pinyin1   string             `json:"pinyin1,omitempty"` // 首字读音（带声调，预计算自 Character.Pinyin）
	Pinyin2   string             `json:"pinyin2,omitempty"` // 次字读音（单名为空）
	Meaning1  string             `json:"meaning1,omitempty"` // 首字释义（预计算自 Character.Meaning）
	Meaning2  string             `json:"meaning2,omitempty"` // 次字释义（单名为空）
	Score     float64            `json:"score"`
	Grade     string             `json:"grade"`
	WuXing1   string             `json:"wu_xing1"`
	WuXing2   string             `json:"wu_xing2"`
	Stroke1   int                `json:"stroke1,omitempty"` // 名字笔画（filter 口径，按 StrokeMode 决定）
	Stroke2   int                `json:"stroke2,omitempty"` // 次字笔画（单名为空）
	// KangxiStroke1/2 康熙字典笔画（统一用于总笔画回显，与姓氏口径一致）
	// 0 表示未收录，输出回显时降级为 Stroke1/2。
	KangxiStroke1 int `json:"kangxi_stroke1,omitempty"`
	KangxiStroke2 int `json:"kangxi_stroke2,omitempty"`
	HasPoetry     bool               `json:"has_poetry"`
	PoetryFrom    string             `json:"poetry_from,omitempty"` // 诗词出处
	Items         map[string]float64 `json:"items,omitempty"`      // 各维度评分明细
	// Details 各维度评分依据文字（维度名 → 文字解释），随流式 Top-N 透传给 NameResult
	Details map[string]string `json:"details,omitempty"`

	// 人名频率档位（1-5，0=未收录），供前端展示频率信息
	NameFreqTier1 int `json:"name_freq_tier1,omitempty"`
	NameFreqTier2 int `json:"name_freq_tier2,omitempty"`

	// idx1/idx2 候选字在引擎候选表 infos 中的下标（-1 表示无）。
	//
	// 枚举阶段只计算总分（RateNameScore），不构造 Items/Details；
	// 进入推荐榜后再用这两个下标从 infos 还原 NameCandidate 回算完整明细
	// （详见 engine.fillEntryDetails）。不作为 JSON 输出。
	idx1 int
	idx2 int
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
	// dedup 是否按 Char1+Char2 去重。
	// 置假时 TryPush 完全跳过 seen 的「字符串拼接 + map 写入」，
	// 适用于调用方能保证组合天然唯一的场景（如双名 N² 枚举的 worker 局部表）。
	dedup bool
}

// NewExcellentTable 创建容量为 10000 的 ExcellentTable
//
// 默认容量保留旧行为（用于未指定 topCount 的场景）；
// 推荐调用方使用 NewExcellentTableWithCap(topCount*2) 显式传容量，
// 节省 N² 枚举场景下的内存开销（避免每个 worker 持 10000 容量的堆）。
func NewExcellentTable() *ExcellentTable {
	return &ExcellentTable{
		h:     make(excellentMinHeap, 0, excellentTableCapacity),
		shown: make(map[string]bool),
		seen:  make(map[string]bool),
		cap:   excellentTableCapacity,
		dedup: true,
	}
}

// NewExcellentTableWithCap 创建指定容量、按 Char1+Char2 去重的 ExcellentTable
//
// 推荐传入 topCount*2 或更大值，避免在大量枚举后堆被频繁替换（每次替换是 O(log n)）。
// capacity <= 0 时回退到默认容量 excellentTableCapacity。
func NewExcellentTableWithCap(capacity int) *ExcellentTable {
	if capacity <= 0 {
		capacity = excellentTableCapacity
	}
	return &ExcellentTable{
		h:     make(excellentMinHeap, 0, capacity),
		shown: make(map[string]bool),
		seen:  make(map[string]bool),
		cap:   capacity,
		dedup: true,
	}
}

// NewExcellentTableUnique 创建指定容量、**不做去重**的 ExcellentTable
//
// 适用场景：调用方能保证同一 Char1+Char2 不会重复推入，例如双名 N² 枚举中
// 每个 (i, j) 组合恰好访问一次、且各 worker 的 i 分片互不重叠。
//
// 相比 NewExcellentTableWithCap，每个条目省下一次「两个汉字拼接成字符串 +
// map 写入」的分配（字符串拼接在 Go 中必然分配），并使 seen map 不再随
// 枚举规模增长（原实现下每个 worker 的 seen 会膨胀到局部组合数量级）。
//
// 注意：若调用方无法保证唯一性，必须使用带去重的构造函数，否则重复条目的
// 分数虽不改变 Top-N 上界，但会稀释榜单多样性。
func NewExcellentTableUnique(capacity int) *ExcellentTable {
	t := NewExcellentTableWithCap(capacity)
	t.dedup = false
	t.seen = nil
	return t
}

// TryPush 尝试推入一个名字条目
// 如果已存在相同 Char1+Char2 则跳过
// 未达到容量直接入堆；达到容量则与堆顶（最小分）比较，若高于则替换
func (t *ExcellentTable) TryPush(entry ExcellentEntry) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// 去重（skipDedup 表跳过，省下每次 push 的字符串拼接与 map 写入）
	if t.dedup {
		key := entry.Char1 + entry.Char2
		if t.seen[key] {
			return
		}
		t.seen[key] = true
	}

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
	// 稳定排序：总分降序，同分时保持堆内既有次序。
	//
	// 为什么必须稳定：原实现用 sort.Slice（非稳定），同分条目的先后由元素的
	// 字节内容决定——连条目里是否携带 Items/Details（map 指针）都会改变结果。
	// 这会让「枚举期是否构造明细」这类与排序无关的实现细节影响到 Top-N 的
	// 边界入选（同分挤在 poolSize 截断处时尤甚），同一请求给出不同榜单。
	//
	// 堆内次序只由「得分比较结果 + 推入次序」决定（compare 只比 Score，
	// 与载荷无关），因此稳定排序后同分次序对载荷不敏感、可复现。
	sort.SliceStable(t.entries, func(i, j int) bool {
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

// MinScore 返回堆顶（最小分），用于早停判断
// 表为空时返回 0
func (t *ExcellentTable) MinScore() float64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if len(t.h) == 0 {
		return 0
	}
	return t.h[0].Score
}

// IsFull 返回表是否已满（达到容量上限）
func (t *ExcellentTable) IsFull() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.h) >= t.cap
}

// earlyStopCutoff 返回早停阈值（= 堆顶最小分 × 0.6）；表未满时返回 0 表示不早停。
//
// 无锁：仅供「单 goroutine 独占」的 worker 局部表在 N² 内层循环中调用。
// 每次调用若走 RLock 会带来两次原子操作；内层循环量级为 N²（数千万次），
// 因此这里直接读堆顶。全局表（并发 TryPush）请改用 IsFull + MinScore。
//
// 语义与「IsFull() 后用 MinScore()*0.6 判阈值」完全一致：
//   - 未满 → 返回 0，调用方以 cutoff > 0 判不早停
//   - 已满 → 返回堆顶分×0.6（堆顶分为 0 时同样退化为不早停，与原逻辑一致）
func (t *ExcellentTable) earlyStopCutoff() float64 {
	if len(t.h) < t.cap || len(t.h) == 0 {
		return 0
	}
	return t.h[0].Score * 0.6
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
