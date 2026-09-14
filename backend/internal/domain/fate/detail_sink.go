package fate

import (
	"fmt"
	"strings"
)

// detail_sink.go 评分依据文案的懒加载累积器
//
// 背景：双名候选生成是 N² 笛卡尔积，每个组合都会依次调用 8 个 Rater 的
// Rate()。每个 Rater 都会构造一段「评分依据」文案（fmt.Sprintf 片段 +
// strings.Join 收尾拼接）。实测这两项合计占单次评分 CPU 约 22%，并带来
// 大量短命分配（GC 再吃掉约 12%）。
//
// 但枚举阶段真正需要依据文案的组合只是极少数——最终进入推荐榜并展示的
// 不超过数百条（约千分之一量级），其余组合的解释文案纯属白做。
//
// 因此引入本类型作为统一出口：
//   - SkipDetail 为真（穷举热路径）时，add/addf 全部退化为空操作；
//     关键是 addf 在调用 fmt.Sprintf 之前就返回，从根上避免格式化分配。
//   - 只有进入推荐榜的条目才用 RateName 回算一次完整明细。
//
// 用法：
//
//	ds := newDetailSink(candidate)
//	ds.add("笔画搭配匀称")
//	ds.addf("「%s」五行属%s", ch, wx)
//	...
//	return NameRating{Score: score, Detail: ds.String("默认依据")}
//
// 注意：SkipDetail 只影响文案，不改变任何维度的分值。
type detailSink struct {
	buf  []string
	skip bool
}

// newDetailSink 依候选名的懒加载开关构造 sink
//
// 返回值为值类型 + 指针接收者方法：调用处 ds 是可寻址局部变量，
// Go 会自动取址，因此不会产生堆分配。
func newDetailSink(candidate *NameCandidate) detailSink {
	return detailSink{skip: candidate != nil && candidate.skipDetail}
}

// add 追加一条已构造好的文案（skip 时直接丢弃）
func (d *detailSink) add(s string) {
	if d.skip {
		return
	}
	d.buf = append(d.buf, s)
}

// addf 按格式构造并追加一条文案
//
// skip 时在格式化之前返回，不触发 fmt.Sprintf 的任何分配——这是本类型
// 相对「先格式化再判断」的核心收益。
func (d *detailSink) addf(format string, args ...any) {
	if d.skip {
		return
	}
	d.buf = append(d.buf, fmt.Sprintf(format, args...))
}

// String 返回以「；」连接的全部文案；无文案时返回 fallback
func (d *detailSink) String(fallback string) string {
	if len(d.buf) == 0 {
		return fallback
	}
	return strings.Join(d.buf, "；")
}
