import { describe, it, expect } from 'vitest';
import type { GenerateResponse } from '@/types';
import { buildPdfAnalysis } from './reportAnalysis';

type ResultData = GenerateResponse['data'];

const baseResult = {
  bazi: {
    rishou: '甲', rishou_wuxing: '木',
    bazi: { year: '甲子', month: '乙丑', day: '丙寅', hour: '丁卯' },
    wuxing: { jin: 1, mu: 8, shui: 1, huo: 1, tu: 1 },
    xiyongshen: ['水', '木'],
  },
  nayin: '海中金',
  zodiac: '鼠',
  hexagram: { number: 1, name: '乾' },
  names: [
    { surname: '李', given_name: '沐', total_score: 92, poetry_source: '诗经' },
    { surname: '李', given_name: '林', total_score: 80 },
  ],
} as unknown as ResultData;

describe('buildPdfAnalysis — PDF 命理详批文本', () => {
  it('完整数据：生成 5 段解读，含八字/五行/喜用神/生肖纳音/名字质量', () => {
    const out = buildPdfAnalysis(baseResult, '李', '海中金', '鼠', { number: 1, name: '乾' });
    expect(out).toHaveLength(6); // 八字+五行+喜用神+生肖纳音+卦象+名字质量
    expect(out.some(s => s.includes('【八字格局】') && s.includes('甲'))).toBe(true);
    expect(out.some(s => s.includes('【五行平衡】') && s.includes('最旺'))).toBe(true);
    expect(out.some(s => s.includes('【喜用神补益】') && s.includes('水、木'))).toBe(true);
    expect(out.some(s => s.includes('属鼠') && s.includes('海中金'))).toBe(true);
    expect(out.some(s => s.includes('姓名卦象') && s.includes('乾'))).toBe(true);
    expect(out.some(s => s.includes('平均86.0分') && s.includes('1个出自诗词'))).toBe(true);
  });

  it('偏枯判定：max-min>50% 提示重点补益最弱五行', () => {
    const out = buildPdfAnalysis(baseResult, '李', '', '', undefined);
    const balance = out.find(s => s.includes('【五行平衡】'))!;
    expect(balance).toContain('偏枯');
    expect(balance).toContain('重点补益');
  });

  it('无结果时返回空数组', () => {
    expect(buildPdfAnalysis(null, '李', '', '', undefined)).toEqual([]);
  });
});
