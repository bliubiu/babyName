import { describe, it, expect } from 'vitest';
import type { Name } from '@/types';
import {
  applyNameFilters,
  sortNames,
  scoreOf,
  wuxingGroupIndex,
} from './nameFiltering';

// 构造最小 Name：只填过滤/排序相关字段
function mk(over: Partial<Name>): Name {
  return {
    surname: '李',
    given_name: '测试',
    full_name: '李测试',
    meaning: '',
    wuxing: '金木',
    strokes: 10,
    score: 80,
    ...over,
  } as Name;
}

describe('applyNameFilters — 结果页筛选真源', () => {
  const pool: Name[] = [
    mk({ given_name: '金木', wuxing: '金木' }),
    mk({ given_name: '水木', wuxing: '水木' }),
    mk({ given_name: '土土', wuxing: '土土' }),
  ];

  it('五行 chip 单字能匹配后端两字组合串（修「点了就空列表」）', () => {
    expect(applyNameFilters(pool, { wuxing: '金' }).map(n => n.given_name)).toEqual(['金木']);
    expect(applyNameFilters(pool, { wuxing: '木' }).map(n => n.given_name)).toEqual(['金木', '水木']);
    expect(applyNameFilters(pool, { wuxing: '火' })).toHaveLength(0);
  });

  it('旧实现用全等匹配会漏掉所有双字名——锁定不再回退', () => {
    //  Pool 里没有任何 name.wuxing === '金'，但包含匹配应命中 1 个
    expect(pool.some(n => n.wuxing === '金')).toBe(false);
    expect(applyNameFilters(pool, { wuxing: '金' })).toHaveLength(1);
  });

  it('笔画区间与五行可叠加', () => {
    const p = [mk({ given_name: 'A', wuxing: '金木', strokes: 8 }), mk({ given_name: 'B', wuxing: '金木', strokes: 20 })];
    expect(applyNameFilters(p, { wuxing: '金', minStrokes: 10 }).map(n => n.given_name)).toEqual(['B']);
  });

  it('无筛选时原样返回（换一批数据源同样适用）', () => {
    expect(applyNameFilters(pool, {})).toEqual(pool);
  });
});

describe('sortNames / wuxingGroupIndex — 五行分组排序', () => {
  it('双字串按第一个命中字分组，不再恒为 -1', () => {
    expect(wuxingGroupIndex('金木')).toBe(0);
    expect(wuxingGroupIndex('水木')).toBe(2);
    expect(wuxingGroupIndex('土土')).toBe(4);
    expect(wuxingGroupIndex('')).toBe(99);
  });

  it('五行分组排序按 金木水火土 顺序，同组按分数降序', () => {
    const p = [
      mk({ given_name: '土1', wuxing: '土土', score: 99 }),
      mk({ given_name: '金1', wuxing: '金木', score: 70 }),
      mk({ given_name: '金2', wuxing: '金水', score: 90 }),
      mk({ given_name: '水1', wuxing: '水木', score: 80 }),
    ];
    expect(sortNames(p, 'wuxing').map(n => n.given_name)).toEqual(['金2', '金1', '水1', '土1']);
  });

  it('综合分排序用 total_score 优先于 score', () => {
    expect(scoreOf(mk({ score: 80, total_score: 95 }))).toBe(95);
    const p = [mk({ given_name: 'A', score: 80 }), mk({ given_name: 'B', score: 90 })];
    expect(sortNames(p, 'score').map(n => n.given_name)).toEqual(['B', 'A']);
  });
});
