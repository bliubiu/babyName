import { describe, it, expect } from 'vitest';
import { cn } from './utils';

describe('cn', () => {
  it('应合并多个类名', () => {
    expect(cn('a', 'b')).toBe('a b');
  });

  it('应过滤假值', () => {
    expect(cn('a', false, 'b', undefined, null, 'c')).toBe('a b c');
  });

  it('应处理条件类名（对象语法）', () => {
    expect(cn('base', { active: true, disabled: false })).toBe('base active');
  });

  it('应处理空输入', () => {
    expect(cn()).toBe('');
  });

  it('应合并 tailwind 冲突类名（twMerge 行为）', () => {
    // px-4 和 px-6 冲突，twMerge 保留最后一个
    const result = cn('px-4', 'text-red', 'px-6');
    expect(result).toContain('px-6');
    expect(result).not.toContain('px-4');
  });

  it('应处理多个条件对象', () => {
    const isActive = true;
    const isDisabled = false;
    expect(cn('btn', { 'btn-active': isActive, 'btn-disabled': isDisabled })).toBe('btn btn-active');
  });
});
