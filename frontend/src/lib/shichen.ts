// 传统时辰：子时 23-1，其余两小时一段
const SHICHEN_NAMES = ['子', '丑', '寅', '卯', '辰', '巳', '午', '未', '申', '酉', '戌', '亥'];

/** 与后端 tyme 一致：(hour+1)/2，23 时归子 */
export function getShichenName(hour: number): string {
  const index = Math.floor((hour + 1) / 2) % 12;
  return SHICHEN_NAMES[index];
}

/** 返回时辰边界风险提示；无风险时返回 null */
export function getShichenBoundaryWarning(hour: number, minute: number): string | null {
  const current = getShichenName(hour);
  // 整点前 10 分钟（如 20:50-20:59）可能跨入下一时辰
  if (minute < 50) return null;
  if (hour === 22 || hour === 23) {
    return `当前为「${current}」时末，再往后可能跨入子时并影响日柱，请确认实际出生时辰`;
  }
  const nextHour = (hour + 1) % 24;
  const next = getShichenName(nextHour);
  if (next !== current) {
    return `已接近「${current}」/「${next}」时辰交界，请尽量选择实际出生分钟`;
  }
  return null;
}
