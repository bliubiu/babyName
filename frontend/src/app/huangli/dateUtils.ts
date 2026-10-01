/**
 * 黄历页的日期切换工具（docs/29 B5）
 */

/**
 * 构造指定年月的日期，并把「日」钳制到该月最后一天。
 *
 * 直接 `new Date(year, month - 1, day)` 会让 Date 自动进位，切换年/月时
 * 凭空跳过日期：
 * - 2026-01-31 切到 2 月 → `new Date(2026, 1, 31)` = 2026-03-03（跳过整个 2 月）
 * - 2024-02-29 切到 2025 年 → `new Date(2025, 1, 29)` = 2025-03-01（多出 2 天）
 *
 * @param year 完整年份，如 2026
 * @param month 月份，1-12
 * @param day 目标日，超出该月天数时取该月最后一天
 * @returns 落在目标月份内的日期
 */
export function buildClampedDate(year: number, month: number, day: number): Date {
  // month 从 1 起算，用「下个月第 0 天」取本月最后一天，天然处理闰年与大小月
  const lastDayOfMonth = new Date(year, month, 0).getDate();
  return new Date(year, month - 1, Math.min(day, lastDayOfMonth));
}