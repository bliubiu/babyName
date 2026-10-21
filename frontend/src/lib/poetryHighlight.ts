/**
 * 诗词原句分段：把「实际命中的名字用字」从原句中切分出来，供前端高亮。
 *
 * 背景：出典匹配是「任一字命中即算有出典」——例如「荔嘉」命中的是「嘉」，
 * 「荔」并不在这句里。若整句平铺展示，用户会误以为整个名字都出自该典籍。
 * 高亮命中字即可消除这一歧义。
 */

export interface SentenceSegment {
  text: string;
  hit: boolean;
}

/**
 * splitByMatchedChars 把原句按命中字切分为若干段。
 *
 * - 无命中字时返回单段（hit=false），调用方可退化为纯文本渲染。
 * - 多个命中字按「在原句中最先出现」的位置依次切出。
 * - 同位置命中多个字时取较短者优先（避免长字先切导致短字被拆散）。
 * - 命中字在原句中不出现时忽略（后端与前端数据不一致的兜底）。
 */
export function splitByMatchedChars(text: string, matchedChars?: string[]): SentenceSegment[] {
  if (!text) return [];
  const hits = (matchedChars ?? []).filter((c) => c && text.includes(c));
  if (hits.length === 0) {
    return [{ text, hit: false }];
  }

  // 短字优先：同位置时较短者先切
  const ordered = [...hits].sort((a, b) => a.length - b.length);

  const segments: SentenceSegment[] = [];
  let rest = text;
  let guard = 0;

  while (rest.length > 0 && guard++ < 1000) {
    let bestIdx = -1;
    let bestChar = '';
    for (const ch of ordered) {
      const idx = rest.indexOf(ch);
      if (idx >= 0 && (bestIdx === -1 || idx < bestIdx)) {
        bestIdx = idx;
        bestChar = ch;
      }
    }
    if (bestIdx === -1) {
      segments.push({ text: rest, hit: false });
      break;
    }
    if (bestIdx > 0) {
      segments.push({ text: rest.slice(0, bestIdx), hit: false });
    }
    segments.push({ text: bestChar, hit: true });
    rest = rest.slice(bestIdx + bestChar.length);
  }

  return segments;
}
