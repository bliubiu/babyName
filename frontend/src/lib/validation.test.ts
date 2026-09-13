import { describe, it, expect } from 'vitest';
import { validateBirthTime, validateSurname } from './validation';

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

describe('validateBirthTime', () => {
  const base = {
    birthYear: 2024,
    birthMonth: 1,
    birthDay: 15,
    birthHour: 12,
    birthMinute: 30,
  };

  it('合法时间应通过', () => {
    expect(validateBirthTime(base)).toBeUndefined();
  });

  it('缺字段应报错', () => {
    expect(validateBirthTime({ birthYear: 2024 })).toBe('请完整填写出生时间');
  });

  it('年份越界', () => {
    expect(validateBirthTime({ ...base, birthYear: 1800 })).toContain('1900');
    expect(validateBirthTime({ ...base, birthYear: 2200 })).toContain('2100');
  });

  it('月份/日期/时分越界', () => {
    expect(validateBirthTime({ ...base, birthMonth: 13 })).toBe('出生月份无效');
    expect(validateBirthTime({ ...base, birthDay: 32 })).toBe('出生日期无效');
    expect(validateBirthTime({ ...base, birthHour: 24 })).toContain('0–23');
    expect(validateBirthTime({ ...base, birthMinute: 60 })).toContain('0–59');
  });

  it('边界值应通过', () => {
    expect(
      validateBirthTime({
        birthYear: 1900,
        birthMonth: 1,
        birthDay: 1,
        birthHour: 0,
        birthMinute: 0,
      })
    ).toBeUndefined();
    expect(
      validateBirthTime({
        birthYear: 2100,
        birthMonth: 12,
        birthDay: 31,
        birthHour: 23,
        birthMinute: 59,
      })
    ).toBeUndefined();
  });
});

