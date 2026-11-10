import { describe, it, expect } from 'vitest';
import { buildGenerateRequest } from './generateRequest';
import type { FormData } from '@/types';

const base: FormData = {
  surname: '李',
  gender: 'female',
  birthYear: 2024,
  birthMonth: 3,
  birthDay: 8,
  birthHour: 9,
  birthMinute: 5,
  birthLocation: '杭州',
  generation: '沐',
  generationPosition: 'middle',
  nameType: 'double',
  birthType: 'solar',
  preferences: [],
  nameLength: 2,
  sourceClassic: '',
  avoidElderNames: '李建国，王秀英',
  minFrequencyTier: 0,
  maxFrequencyTier: 0,
  enableCantonese: true,
};

describe('buildGenerateRequest', () => {
  it('完整映射生辰与表单字段', () => {
    const req = buildGenerateRequest(base);
    expect(req).toMatchObject({
      surname: '李',
      gender: 'female',
      birth_year: 2024,
      birth_month: 3,
      birth_day: 8,
      birth_hour: 9,
      birth_minute: 5,
      birth_location: '杭州',
      generation_position: 'middle',
      name_type: 'double',
      name_length: 2,
      enable_cantonese: true,
    });
  });

  it('按中英文逗号与空白切分避讳长辈与寓意关键词', () => {
    const req = buildGenerateRequest(base, {
      keywords: '温润, 澄澈 山川',
      selectedChars: ['木', '氵'],
      sourceClassic: '诗经',
    });
    expect(req.avoid_elder_names).toEqual(['李建国', '王秀英']);
    expect(req.meaning_keywords).toEqual(['温润', '澄澈', '山川']);
    expect(req.selected_chars).toEqual(['木', '氵']);
    expect(req.source_classic).toBe('诗经');
  });

  it('单名与未开启粤语时字段口径正确', () => {
    const req = buildGenerateRequest({
      ...base,
      nameType: 'single',
      enableCantonese: false,
      minFrequencyTier: 0,
      avoidElderNames: '',
    });
    expect(req.name_length).toBe(1);
    expect(req.enable_cantonese).toBeUndefined();
    expect(req.min_frequency_tier).toBeUndefined();
    expect(req.avoid_elder_names).toBeUndefined();
    expect(req.meaning_keywords).toEqual([]);
  });
});
