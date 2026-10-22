import { describe, it, expect, vi } from 'vitest';
import {
  parseJyutpingSyllables,
  toneClass,
  toneLabel,
  ttsTextFor,
  cantoneseTtsSupported,
  speakCantonese,
  cancelCantoneseSpeech,
} from './jyutping';

describe('parseJyutpingSyllables', () => {
  it('解析双名三音节（姓+名两字）', () => {
    const syls = parseJyutpingSyllables('wong4 zi2 man4');
    expect(syls).toHaveLength(3);
    expect(syls[0]).toEqual({ raw: 'wong4', tone: 4, isEntering: false });
    expect(syls[1].tone).toBe(2);
    expect(syls[2].tone).toBe(4);
  });

  it('解析单名两音节（姓+名一字）', () => {
    const syls = parseJyutpingSyllables('zi2 man4');
    expect(syls.map(s => s.tone)).toEqual([2, 4]);
  });

  it('识别入声（-p/-t/-k 塞尾）', () => {
    const syls = parseJyutpingSyllables('sap6 gat1 zaak6');
    expect(syls.map(s => s.isEntering)).toEqual([true, true, true]);
    expect(syls[1].tone).toBe(1); // 入声也带 1/3/6 调号
  });

  it('容忍无声调音节与多余空白', () => {
    const syls = parseJyutpingSyllables('  wong   zi ');
    expect(syls).toHaveLength(2);
    expect(syls[0]).toEqual({ raw: 'wong', tone: 0, isEntering: false });
  });

  it('空输入返回空数组', () => {
    expect(parseJyutpingSyllables('')).toEqual([]);
    expect(parseJyutpingSyllables(undefined)).toEqual([]);
    expect(parseJyutpingSyllables(null)).toEqual([]);
  });

  it('gw/kw/ng 声母不被误判为调号', () => {
    const syls = parseJyutpingSyllables('gwok3 ng4');
    expect(syls[0].raw).toBe('gwok3');
    expect(syls[0].tone).toBe(3);
    expect(syls[1].raw).toBe('ng4');
  });
});

describe('toneClass / toneLabel', () => {
  it('1/2 为平，3-6 为仄', () => {
    expect(toneClass(1)).toBe('平');
    expect(toneClass(2)).toBe('平');
    expect(toneClass(3)).toBe('仄');
    expect(toneClass(6)).toBe('仄');
    expect(toneClass(0)).toBe('');
  });

  it('toneLabel 带调值与中文名', () => {
    expect(toneLabel(1)).toBe('1 阴平（高平）');
    expect(toneLabel(4)).toBe('4 阳平（低降）');
    expect(toneLabel(0)).toBe('');
  });
});

describe('ttsTextFor', () => {
  it('朗读汉字原名而非粤拼', () => {
    expect(ttsTextFor('  王梓文 ')).toBe('王梓文');
  });
});

describe('粤语 TTS（happy-dom 无 speechSynthesis）', () => {
  it('不支持时 speakCantonese 返回 false', () => {
    expect(cantoneseTtsSupported()).toBe(false);
    expect(speakCantonese('王梓文')).toBe(false);
  });

  it('cancelCantoneseSpeech 不支持时不抛错', () => {
    expect(() => cancelCantoneseSpeech()).not.toThrow();
  });

  it('空文本不朗读', () => {
    expect(speakCantonese('')).toBe(false);
  });
});
