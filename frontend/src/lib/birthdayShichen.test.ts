import { describe, expect, it } from 'vitest';
import { getShichenBoundaryWarning, getShichenName } from './shichen';

describe('getShichenName', () => {
  it('按 (hour+1)/2 定时支，与后端 tyme 一致', () => {
    expect(getShichenName(0)).toBe('子');
    expect(getShichenName(1)).toBe('丑');
    expect(getShichenName(12)).toBe('午');
    expect(getShichenName(19)).toBe('戌');
    expect(getShichenName(20)).toBe('戌');
    expect(getShichenName(21)).toBe('亥');
    expect(getShichenName(22)).toBe('亥');
    expect(getShichenName(23)).toBe('子'); // 晚子时
  });
});

describe('getShichenBoundaryWarning', () => {
  it('非边界分钟不提示', () => {
    expect(getShichenBoundaryWarning(19, 0)).toBeNull();
    expect(getShichenBoundaryWarning(19, 45)).toBeNull();
    expect(getShichenBoundaryWarning(12, 30)).toBeNull();
  });

  it('奇数整点前 10 分钟提示跨时辰', () => {
    const w = getShichenBoundaryWarning(20, 55);
    expect(w).not.toBeNull();
    expect(w).toContain('戌');
  });

  it('22:50 后提示可能跨日柱', () => {
    const w = getShichenBoundaryWarning(22, 50);
    expect(w).not.toBeNull();
    expect(w).toContain('日柱');
  });

  it('偶数整点末尾同属当前时辰，仅在跨支时提示', () => {
    // 19:55 与 20:00 同为戌时，不应误报跨支
    expect(getShichenBoundaryWarning(19, 55)).toBeNull();
  });
});
