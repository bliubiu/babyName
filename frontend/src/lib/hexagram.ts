/**
 * 易经八卦/六爻前端口径模块。
 *
 * 与后端 backend/internal/domain/yijing 保持一致（docs/42 P2-8）：
 * - 八卦 ID 为先天序 1-8（乾1 兑2 离3 震4 巽5 坎6 艮7 坤8）；
 * - 爻值 7=少阳（阳爻）/ 8=少阴（阴爻），爻位自下而上 1-6；
 * - 三爻表须与 plum_blossom.go trigramYaoMaps 逐项一致（P0 爻画颠倒回归防线）。
 */

/** 八卦 ID → 卦名 */
export const TRIGRAM_NAMES: Record<number, string> = {
  1: '乾',
  2: '兑',
  3: '离',
  4: '震',
  5: '巽',
  6: '坎',
  7: '艮',
  8: '坤',
};

/** 八卦 ID → 三画符号 */
export const TRIGRAM_SYMBOLS: Record<number, string> = {
  1: '☰',
  2: '☱',
  3: '☲',
  4: '☳',
  5: '☴',
  6: '☵',
  7: '☶',
  8: '☷',
};

/** 八卦 ID → 卦五行（对应后端 TrigramWuxingMap） */
export const TRIGRAM_WUXING: Record<number, string> = {
  1: '金',
  2: '金',
  3: '火',
  4: '木',
  5: '木',
  6: '水',
  7: '土',
  8: '土',
};

/** 八卦 ID → 三爻值（自下而上，7=阳 / 8=阴），先天口诀：乾三连、兑上缺、离中虚、震仰盂、巽下断、坎中满、艮覆碗、坤六断 */
export const TRIGRAM_YAO: Record<number, [number, number, number]> = {
  1: [7, 7, 7],
  2: [7, 7, 8],
  3: [7, 8, 7],
  4: [7, 8, 8],
  5: [8, 7, 7],
  6: [8, 7, 8],
  7: [8, 8, 7],
  8: [8, 8, 8],
};

/** 由上/下八卦 ID 构造六爻值（自下而上）：下卦为初至三爻，上卦为四至六爻 */
export function buildHexagramYaoLines(
  upperTrigramId: number,
  lowerTrigramId: number
): number[] {
  const lower = TRIGRAM_YAO[lowerTrigramId];
  const upper = TRIGRAM_YAO[upperTrigramId];
  if (!lower || !upper) {
    return [];
  }
  return [...lower, ...upper];
}

/** 爻是否为阳爻（7 少阳 / 9 老阳） */
export function isYangYao(value: number): boolean {
  return value === 7 || value === 9;
}

/**
 * 标准爻名：初九/初六、九二/六二 … 九五/六五、上九/上六。
 * 与后端 yijing.GetYaoName 同规则；越界或非法爻值返回空串。
 */
export function getYaoName(position: number, value: number): string {
  if (position < 1 || position > 6) {
    return '';
  }
  let yinYang: string;
  switch (value) {
    case 6:
    case 8:
      yinYang = '六';
      break;
    case 7:
    case 9:
      yinYang = '九';
      break;
    default:
      return '';
  }
  switch (position) {
    case 1:
      return `初${yinYang}`;
    case 2:
      return `${yinYang}二`;
    case 3:
      return `${yinYang}三`;
    case 4:
      return `${yinYang}四`;
    case 5:
      return `${yinYang}五`;
    case 6:
      return `上${yinYang}`;
    default:
      return '';
  }
}
