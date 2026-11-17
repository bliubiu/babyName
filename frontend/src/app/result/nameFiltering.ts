import type { Name } from '@/types';
import type { NameFilters } from '@/components/NameFilter';

export const WUXING_ORDER = ['金', '木', '水', '火', '土'] as const;
export type SortKey = 'score' | 'strokes' | 'wuxing';

export const scoreOf = (n: Name): number => n.total_score ?? n.score ?? 0;

// 后端 wuxing 字段是两字的组合串（如「金木」），筛选 chip 是单字。
// 分组排序取串首字（名的第一个五行）作为组键；找不到排最后。
// 注意不能用 findIndex(x => w.includes(x))——那会按五行表顺序取到「木」而非串首「水」。
export function wuxingGroupIndex(w?: string): number {
  if (!w) return 99;
  const idx = WUXING_ORDER.findIndex(x => x === w.charAt(0));
  return idx < 0 ? 99 : idx;
}

/**
 * 对当前数据源套用筛选条件。
 * - 五行：chip 单字 vs 后端两字组合串，按「包含」匹配（修「点了就空列表」缺陷）。
 * - 性别不参与：首页已定性别，一次生成的所有候选同源，结果页性别选项已移除。
 * - 同样作用于「换一批」数据，保证换批后筛选不失效。
 */
export function applyNameFilters(names: Name[], filters: NameFilters): Name[] {
  return names.filter((name: Name) => {
    if (filters.wuxing && !name.wuxing?.includes(filters.wuxing)) return false;
    if (filters.minStrokes && name.strokes < filters.minStrokes) return false;
    if (filters.maxStrokes && name.strokes > filters.maxStrokes) return false;
    if (filters.minFrequencyTier && name.frequency_score !== undefined) {
      const estimatedTier = name.frequency_score >= 90 ? 5
        : name.frequency_score >= 70 ? 4
        : name.frequency_score >= 55 ? 3
        : name.frequency_score >= 35 ? 2
        : name.frequency_score >= 15 ? 1
        : 0;
      if (estimatedTier < filters.minFrequencyTier) return false;
    }
    return true;
  });
}

/** 按排序键重排（返回新数组，不改动入参）。 */
export function sortNames(names: Name[], sortKey: SortKey): Name[] {
  const list = [...names];
  if (sortKey === 'strokes') {
    return list.sort((a, b) => (a.strokes ?? 0) - (b.strokes ?? 0) || scoreOf(b) - scoreOf(a));
  }
  if (sortKey === 'wuxing') {
    return list.sort((a, b) => wuxingGroupIndex(a.wuxing) - wuxingGroupIndex(b.wuxing) || scoreOf(b) - scoreOf(a));
  }
  return list.sort((a, b) => scoreOf(b) - scoreOf(a));
}
