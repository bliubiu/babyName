export const validateSurname = (value: string): string | undefined => {
  if (!value || value.trim() === '') {
    return '请输入姓氏';
  }
  if (value.length > 4) {
    return '姓氏不能超过4个字符';
  }
  if (!/^[一-龥]+$/.test(value)) {
    return '姓氏只能包含中文';
  }
  return undefined;
};

/** 校验出生时间（年月日时分），合法返回 undefined */
export const validateBirthTime = (input: {
  birthYear?: number;
  birthMonth?: number;
  birthDay?: number;
  birthHour?: number;
  birthMinute?: number;
}): string | undefined => {
  const { birthYear, birthMonth, birthDay, birthHour, birthMinute } = input;
  if (
    birthYear === undefined ||
    birthMonth === undefined ||
    birthDay === undefined ||
    birthHour === undefined ||
    birthMinute === undefined
  ) {
    return '请完整填写出生时间';
  }
  if (!Number.isInteger(birthYear) || birthYear < 1900 || birthYear > 2100) {
    return '出生年份需在 1900–2100 之间';
  }
  if (!Number.isInteger(birthMonth) || birthMonth < 1 || birthMonth > 12) {
    return '出生月份无效';
  }
  if (!Number.isInteger(birthDay) || birthDay < 1 || birthDay > 31) {
    return '出生日期无效';
  }
  // 真实日历校验：2 月 30 日这类日期范围检查拦不住，后端 tyme 也会拒绝
  const daysInMonth = new Date(birthYear, birthMonth, 0).getDate();
  if (birthDay > daysInMonth) {
    return '出生日期不存在（如 2 月 30 日）';
  }
  if (!Number.isInteger(birthHour) || birthHour < 0 || birthHour > 23) {
    return '出生小时需在 0–23 之间';
  }
  if (!Number.isInteger(birthMinute) || birthMinute < 0 || birthMinute > 59) {
    return '出生分钟需在 0–59 之间';
  }
  return undefined;
};
