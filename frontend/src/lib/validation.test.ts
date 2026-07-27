import { describe, it, expect } from 'vitest';
import { validateSurname } from './validation';

describe('validateSurname', () => {
  it('应返回错误信息 - 空字符串', () => {
    expect(validateSurname('')).toBe('请输入姓氏');
  });

  it('应返回错误信息 - 空白字符串', () => {
    expect(validateSurname('   ')).toBe('请输入姓氏');
  });

  it('应返回错误信息 - null/undefined', () => {
    expect(validateSurname('' as string)).toBe('请输入姓氏');
  });

  it('应返回错误信息 - 超过4个字符', () => {
    expect(validateSurname('欧阳晓晓晓')).toBe('姓氏不能超过4个字符');
  });

  it('应返回错误信息 - 包含非中文字符', () => {
    expect(validateSurname('王a')).toBe('姓氏只能包含中文');
  });

  it('应返回错误信息 - 包含数字', () => {
    expect(validateSurname('王1')).toBe('姓氏只能包含中文');
  });

  it('应返回 undefined - 单字姓氏', () => {
    expect(validateSurname('王')).toBeUndefined();
  });

  it('应返回 undefined - 双字复姓', () => {
    expect(validateSurname('欧阳')).toBeUndefined();
  });

  it('应返回 undefined - 三字复姓', () => {
    expect(validateSurname('慕容')).toBeUndefined();
  });

  it('应返回 undefined - 四字复姓', () => {
    expect(validateSurname('爱新觉罗')).toBeUndefined();
  });
});
