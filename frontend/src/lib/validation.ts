export const validateSurname = (value: string): string | undefined => {
  if (!value || value.trim() === '') {
    return '请输入姓氏';
  }
  if (value.length > 4) {
    return '姓氏不能超过4个字符';
  }
  if (!/^[\u4e00-\u9fa5]+$/.test(value)) {
    return '姓氏只能包含中文';
  }
  return undefined;
};
