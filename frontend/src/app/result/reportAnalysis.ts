/**
 * PDF 命理详批文本生成（纯函数）。
 *
 * 复用结果页「完整报告」的解读口径，把八字格局、五行平衡、喜用神补益、
 * 生肖纳音、名字质量等变为纯文本段落下发给后端 PDF 渲染器，
 * 让 PDF 从「字段罗列」升级为「有解读的报告」。
 */
import type { GenerateResponse } from '@/types';

type ResultData = GenerateResponse['data'];

// 显式 key 元组：既做中文标签映射，又保证能安全索引 BaziAnalysis.wuxing 的字段。
const WUXING_KEYS = [
  { key: 'jin', label: '金' },
  { key: 'mu', label: '木' },
  { key: 'shui', label: '水' },
  { key: 'huo', label: '火' },
  { key: 'tu', label: '土' },
] as const;

/** 生成 PDF 命理详批段落数组（每段独立换行展示）。 */
export function buildPdfAnalysis(
  result: ResultData | null | undefined,
  surname: string,
  nayin: string,
  zodiac: string,
  hexagram?: { number?: number; name?: string }
): string[] {
  if (!result) return [];

  const paragraphs: string[] = [];
  const bazi = result.bazi;
  const wuxing = bazi?.wuxing;

  // 1. 八字格局
  if (bazi) {
    const dayGanzhi = bazi.bazi?.day || '';
    const rishou = bazi.rishou || '未知';
    const rishouWuxing = bazi.rishou_wuxing || '未知';
    const pillars = [bazi.bazi?.year, bazi.bazi?.month, bazi.bazi?.day, bazi.bazi?.hour]
      .filter(Boolean).join('、');
    paragraphs.push(
      `【八字格局】日主为${rishou}（${rishouWuxing}性），日柱${dayGanzhi}。四柱为 ${pillars}。`
    );
  }

  // 2. 五行平衡（基于真实分布）
  if (wuxing) {
    const entries = WUXING_KEYS
      .map(({ key, label }) => ({ label, value: Number(wuxing[key] ?? 0) }));
    const total = entries.reduce((s, e) => s + e.value, 0);
    if (total === 0) {
      paragraphs.push('【五行平衡】八字五行数据缺失，建议核对出生时间后重新生成。');
    } else {
      const sorted = [...entries].sort((a, b) => b.value - a.value);
      const strongest = sorted[0];
      const weakest = sorted[sorted.length - 1];
      const detail = entries.map(e => `${e.label}${e.value}`).join('、');
      const gap = ((strongest.value - weakest.value) / total) * 100;
      let balance: string;
      if (gap > 50) {
        balance = `整体偏枯，${strongest.label}最旺、${weakest.label}最弱，起名需重点补益${weakest.label}。`;
      } else if (gap > 25) {
        balance = `略有失衡，${strongest.label}较旺、${weakest.label}偏弱，起名可适度补益${weakest.label}。`;
      } else {
        balance = '五行整体较为均衡，无明显偏枯。';
      }
      paragraphs.push(`【五行平衡】分布为${detail}（共${total}个），${balance}`);
    }
  }

  // 3. 喜用神补益
  if (bazi?.xiyongshen?.length) {
    const xy = bazi.xiyongshen.join('、');
    paragraphs.push(`【喜用神补益】喜用神为${xy}，起名宜选用五行属${xy}或与之相生的字，以补益命局不足。`);
  }

  // 4. 生肖纳音 + 卦象
  const extras: string[] = [];
  if (zodiac) extras.push(`生肖属${zodiac}`);
  if (nayin) extras.push(`纳音为${nayin}`);
  if (extras.length) paragraphs.push(`【生肖纳音】${extras.join('，')}。`);
  if (hexagram?.name) {
    paragraphs.push(`【姓名卦象】得第${hexagram.number ?? '—'}卦「${hexagram.name}」，卦象仅供解读参考，不计入评分。`);
  }

  // 5. 名字质量
  const names = result.names ?? [];
  if (names.length) {
    const scores = names.map(n => n.total_score ?? n.score ?? 0);
    const avg = scores.reduce((a, b) => a + b, 0) / scores.length;
    const max = Math.max(...scores);
    const poetryCount = names.filter(n => n.poetry_source).length;
    const top = names[scores.indexOf(max)];
    const topStr = top ? `${top.surname ?? surname}${top.given_name ?? ''}` : '';
    paragraphs.push(
      `【名字质量】共推荐${names.length}个名字，平均${avg.toFixed(1)}分，最高${max.toFixed(1)}分（${topStr}），其中${poetryCount}个出自诗词典故。`
    );
  }

  return paragraphs;
}
