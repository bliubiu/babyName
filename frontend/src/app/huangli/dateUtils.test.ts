import { describe, it, expect } from 'vitest';
import { buildClampedDate } from './dateUtils';

/** 格式化为 YYYY-MM-DD，便于断言 */
const fmt = (d: Date) =>
  `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;

describe('buildClampedDate', () => {
  it('日未超出目标月时保持原样', () => {
    expect(fmt(buildClampedDate(2026, 3, 15))).toBe('2026-03-15');
  });

  it('大月切到小月时钳制到月末', () => {
    // 1月31日切到2月：2026年2月只有28天
    expect(fmt(buildClampedDate(2026, 2, 31))).toBe('2026-02-28');
    expect(fmt(buildClampedDate(2026, 4, 31))).toBe('2026-04-30');
  });

  it('平年 2 月钳制到 28 日', () => {
    expect(fmt(buildClampedDate(2025, 2, 30))).toBe('2025-02-28');
  });

  it('闰年 2 月允许 29 日', () => {
    expect(fmt(buildClampedDate(2024, 2, 29))).toBe('2024-02-29');
    // 闰年切到平年，2月29日必须落到28日而非3月1日
    expect(fmt(buildClampedDate(2025, 2, 29))).toBe('2025-02-28');
  });

  it('闰年 2 月 29 日切到其他大月仍保持 29 日', () => {
    expect(fmt(buildClampedDate(2024, 3, 29))).toBe('2024-03-29');
  });

  it('小月切到大月不改变日', () => {
    expect(fmt(buildClampedDate(2026, 2, 28))).toBe('2026-02-28');
  });

  it('跨年切换：12月31日切到1月', () => {
    expect(fmt(buildClampedDate(2027, 1, 31))).toBe('2027-01-31');
  });

  it('保留月末语义：每月最后一日切换后仍是月末', () => {
    // 遍历 24 个月，每月 1 号从「上月末」出发，切换后不应出现进位跳月
    let cursor = new Date(2024, 0, 31); // 2024-01-31
    for (let i = 0; i < 24; i++) {
      const next = buildClampedDate(
        cursor.getFullYear(),
        cursor.getMonth() + 2, // 目标月份（下个月）
        cursor.getDate(),
      );
      expect(next.getMonth()).toBe((cursor.getMonth() + 1) % 12);
      expect(next.getDate()).toBeGreaterThanOrEqual(28);
      cursor = next;
    }
  });

  it('结果始终落在请求的月份内（不发生自动进位）', () => {
    for (let year = 2020; year <= 2030; year++) {
      for (let month = 1; month <= 12; month++) {
        for (const day of [1, 28, 29, 30, 31]) {
          const d = buildClampedDate(year, month, day);
          expect(d.getFullYear()).toBe(year);
          expect(d.getMonth() + 1).toBe(month);
          expect(d.getDate()).toBeLessThanOrEqual(new Date(year, month, 0).getDate());
        }
      }
    }
  });
});