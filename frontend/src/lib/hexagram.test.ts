import { describe, it, expect } from 'vitest';
import {
  TRIGRAM_GUADE,
  TRIGRAM_NAMES,
  TRIGRAM_SYMBOLS,
  TRIGRAM_WUXING,
  TRIGRAM_YAO,
  buildHexagramYaoLines,
  getHexagramGuaDe,
  getYaoName,
  isYangYao,
} from './hexagram';

describe('八卦三爻表（与后端 plum_blossom.go trigramYaoMaps 锁死）', () => {
  it('按先天口诀逐卦校验爻画', () => {
    // 乾三连、兑上缺、离中虚、震仰盂、巽下断、坎中满、艮覆碗、坤六断
    expect(TRIGRAM_YAO[1]).toEqual([7, 7, 7]);
    expect(TRIGRAM_YAO[2]).toEqual([7, 7, 8]); // 兑上缺：上爻阴
    expect(TRIGRAM_YAO[3]).toEqual([7, 8, 7]); // 离中虚：中爻阴
    expect(TRIGRAM_YAO[4]).toEqual([7, 8, 8]);
    expect(TRIGRAM_YAO[5]).toEqual([8, 7, 7]); // 巽下断：初爻阴
    expect(TRIGRAM_YAO[6]).toEqual([8, 7, 8]); // 坎中满：中爻阳
    expect(TRIGRAM_YAO[7]).toEqual([8, 8, 7]);
    expect(TRIGRAM_YAO[8]).toEqual([8, 8, 8]);
  });

  it('卦名/符号/五行表齐全且自洽（乾兑金、离火、震巽木、坎水、艮坤土）', () => {
    expect(Object.keys(TRIGRAM_NAMES)).toHaveLength(8);
    expect(Object.keys(TRIGRAM_SYMBOLS)).toHaveLength(8);
    expect(Object.keys(TRIGRAM_WUXING)).toHaveLength(8);
    expect(TRIGRAM_NAMES[1]).toBe('乾');
    expect(TRIGRAM_SYMBOLS[8]).toBe('☷');
    const expected: Record<number, string> = {
      1: '金', 2: '金', 3: '火', 4: '木',
      5: '木', 6: '水', 7: '土', 8: '土',
    };
    for (const [id, wx] of Object.entries(expected)) {
      expect(TRIGRAM_WUXING[Number(id)]).toBe(wx);
    }
  });
});

describe('buildHexagramYaoLines（金标准卦核对）', () => {
  it('水雷屯 = 坎上震下 → [阳,阴,阴,阴,阳,阴]', () => {
    // 屯卦爻辞：初九、六二、六三、六四、九五、上六
    expect(buildHexagramYaoLines(6, 4)).toEqual([7, 8, 8, 8, 7, 8]);
  });

  it('雷火丰 = 震上离下 → 初爻为阳（动爻金标准盘）', () => {
    // docs/42 P0 金标准：姓4名3 → 丰#55 动爻1；丰初爻为初九
    expect(buildHexagramYaoLines(4, 3)).toEqual([7, 8, 7, 7, 8, 8]);
  });

  it('天地泰 = 坤上乾下 → 下三爻全阳', () => {
    expect(buildHexagramYaoLines(8, 1)).toEqual([7, 7, 7, 8, 8, 8]);
  });

  it('乾坤用九用六：乾全阳、坤全阴', () => {
    expect(buildHexagramYaoLines(1, 1)).toEqual([7, 7, 7, 7, 7, 7]);
    expect(buildHexagramYaoLines(8, 8)).toEqual([8, 8, 8, 8, 8, 8]);
  });

  it('非法八卦 ID 返回空数组', () => {
    expect(buildHexagramYaoLines(0, 4)).toEqual([]);
    expect(buildHexagramYaoLines(6, 9)).toEqual([]);
  });
});

describe('getYaoName（与后端 GetYaoName 同规则）', () => {
  it('标准爻名', () => {
    expect(getYaoName(1, 7)).toBe('初九');
    expect(getYaoName(1, 8)).toBe('初六');
    expect(getYaoName(2, 8)).toBe('六二');
    expect(getYaoName(3, 7)).toBe('九三');
    expect(getYaoName(4, 8)).toBe('六四');
    expect(getYaoName(5, 7)).toBe('九五');
    expect(getYaoName(6, 8)).toBe('上六');
    expect(getYaoName(6, 7)).toBe('上九');
  });

  it('老阳/老阴（9/6）同样按阴阳前缀', () => {
    expect(getYaoName(1, 9)).toBe('初九');
    expect(getYaoName(2, 6)).toBe('六二');
  });

  it('越界或非法爻值返回空串', () => {
    expect(getYaoName(0, 7)).toBe('');
    expect(getYaoName(7, 7)).toBe('');
    expect(getYaoName(3, 5)).toBe('');
  });
});

describe('isYangYao', () => {
  it('7/9 为阳，6/8 为阴', () => {
    expect(isYangYao(7)).toBe(true);
    expect(isYangYao(9)).toBe(true);
    expect(isYangYao(8)).toBe(false);
    expect(isYangYao(6)).toBe(false);
  });
});

describe('卦德（与后端 GuaDeOf 同规则）', () => {
  it('八卦卦德表：乾健坤顺震动巽入坎陷离丽艮止兑悦', () => {
    expect(TRIGRAM_GUADE[1]).toBe('健');
    expect(TRIGRAM_GUADE[2]).toBe('悦');
    expect(TRIGRAM_GUADE[3]).toBe('丽');
    expect(TRIGRAM_GUADE[4]).toBe('动');
    expect(TRIGRAM_GUADE[5]).toBe('入');
    expect(TRIGRAM_GUADE[6]).toBe('陷');
    expect(TRIGRAM_GUADE[7]).toBe('止');
    expect(TRIGRAM_GUADE[8]).toBe('顺');
  });

  it('八纯卦单字，异卦「上X下Y」', () => {
    expect(getHexagramGuaDe(1, 1)).toBe('健');
    expect(getHexagramGuaDe(8, 8)).toBe('顺');
    expect(getHexagramGuaDe(6, 4)).toBe('上陷下动'); // 屯
    expect(getHexagramGuaDe(4, 3)).toBe('上动下丽'); // 丰
    expect(getHexagramGuaDe(8, 1)).toBe('上顺下健'); // 泰
  });

  it('非法八卦 ID 返回空串', () => {
    expect(getHexagramGuaDe(0, 4)).toBe('');
    expect(getHexagramGuaDe(6, 9)).toBe('');
  });
});
