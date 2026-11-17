import { describe, it, expect } from 'vitest';
import {
  SOLAR_TERM_KNOWLEDGE,
  hasSolarTermKnowledge,
} from './solarTermKnowledge';

const ALL_TERMS = [
  '立春', '雨水', '惊蛰', '春分', '清明', '谷雨',
  '立夏', '小满', '芒种', '夏至', '小暑', '大暑',
  '立秋', '处暑', '白露', '秋分', '寒露', '霜降',
  '立冬', '小雪', '大雪', '冬至', '小寒', '大寒',
];

describe('二十四节气科普知识库', () => {
  it('24 节气齐全，无遗漏无多余', () => {
    expect(Object.keys(SOLAR_TERM_KNOWLEDGE).sort()).toEqual([...ALL_TERMS].sort());
    expect(Object.keys(SOLAR_TERM_KNOWLEDGE)).toHaveLength(24);
  });

  it('每条科普字段完整：三候恰好 3 条、习俗非空、诗句齐全', () => {
    for (const term of ALL_TERMS) {
      const k = SOLAR_TERM_KNOWLEDGE[term];
      expect(k.name, `${term} name`).toBe(term);
      expect(k.pinyin, `${term} pinyin`).toBeTruthy();
      expect(k.climate.length, `${term} climate`).toBeGreaterThan(10);
      expect(k.phenology, `${term} 三候`).toHaveLength(3);
      expect(k.phenology.every(Boolean), `${term} 三候非空`).toBe(true);
      expect(k.customs.length, `${term} 习俗`).toBeGreaterThan(0);
      expect(k.story.length, `${term} 故事`).toBeGreaterThan(10);
      expect(k.poem.text && k.poem.author && k.poem.title, `${term} 诗句`).toBeTruthy();
    }
  });

  it('hasSolarTermKnowledge 正确识别已知/未知节气', () => {
    expect(hasSolarTermKnowledge('清明')).toBe(true);
    expect(hasSolarTermKnowledge('')).toBe(false);
    expect(hasSolarTermKnowledge('不存在')).toBe(false);
  });
});
