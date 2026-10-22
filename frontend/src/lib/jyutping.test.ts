import { describe, it, expect, vi } from 'vitest';
import {
  parseJyutpingSyllables,
  toneClass,
  toneLabel,
  ttsTextFor,
  cantoneseTtsSupported,
  speakCantonese,
  cancelCantoneseSpeech,
  pickCantoneseVoice,
  resolveCantoneseVoice,
  cantoneseTtsStatus,
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

describe('粤语 TTS 检测与降级', () => {
  it('happy-dom 无 speechSynthesis 时 cantoneseTtsSupported 为 false', () => {
    expect(cantoneseTtsSupported()).toBe(false);
  });

  it('不支持时 speakCantonese 返回 false（async）', async () => {
    await expect(speakCantonese('王梓文')).resolves.toBe(false);
  });

  it('空文本不朗读', async () => {
    await expect(speakCantonese('')).resolves.toBe(false);
  });

  it('cancelCantoneseSpeech 不支持时不抛错', () => {
    expect(() => cancelCantoneseSpeech()).not.toThrow();
  });

  it('pickCantoneseVoice 优先 zh-HK，其次中文回退', () => {
    const zhOnly = [{ lang: 'en-US' }, { lang: 'zh-CN' }] as SpeechSynthesisVoice[];
    expect(pickCantoneseVoice(zhOnly)?.lang).toBe('zh-CN');
    const withHk = [{ lang: 'en-US' }, { lang: 'zh-HK' }] as SpeechSynthesisVoice[];
    expect(pickCantoneseVoice(withHk)?.lang).toBe('zh-HK');
    expect(pickCantoneseVoice([])).toBeNull();
  });

  it('resolveCantoneseVoice 在 getVoices 已有嗓音时立即返回', async () => {
    const synth = {
      getVoices: () => [{ lang: 'zh-HK' }, { lang: 'en-US' }] as SpeechSynthesisVoice[],
      addEventListener: () => undefined,
      removeEventListener: () => undefined,
      speak: () => undefined,
      cancel: () => undefined,
    };
    (window as unknown as { speechSynthesis: typeof synth }).speechSynthesis = synth;
    const v = await resolveCantoneseVoice(50);
    expect(v?.lang).toBe('zh-HK');
  });

  it('resolveCantoneseVoice 监听 voiceschanged 后异步解析', async () => {
    let current: SpeechSynthesisVoice[] = [];
    const handlers: Array<() => void> = [];
    const synth = {
      getVoices: () => current,
      addEventListener: (_e: string, cb: () => void) => handlers.push(cb),
      removeEventListener: () => undefined,
      speak: () => undefined,
      cancel: () => undefined,
    };
    (window as unknown as { speechSynthesis: typeof synth }).speechSynthesis = synth;

    const p = resolveCantoneseVoice(500);
    current = [{ lang: 'zh-HK' }] as SpeechSynthesisVoice[];
    handlers.forEach((cb) => cb());
    const v = await p;
    expect(v?.lang).toBe('zh-HK');
  });

  it('resolveCantoneseVoice 超时且无嗓音时返回 null', async () => {
    const synth = {
      getVoices: () => [] as SpeechSynthesisVoice[],
      addEventListener: () => undefined,
      removeEventListener: () => undefined,
      speak: () => undefined,
      cancel: () => undefined,
    };
    (window as unknown as { speechSynthesis: typeof synth }).speechSynthesis = synth;
    const v = await resolveCantoneseVoice(30);
    expect(v).toBeNull();
  });

  it('cantoneseTtsStatus：无 API → unsupported', async () => {
    // 清除前面测试注入的 mock，恢复“无 API”状态
    delete (window as unknown as { speechSynthesis?: unknown }).speechSynthesis;
    const status = await cantoneseTtsStatus();
    expect(status.state).toBe('unsupported');
  });

  it('cantoneseTtsStatus：有 API 但无嗓音 → no-voice', async () => {
    const synth = {
      getVoices: () => [] as SpeechSynthesisVoice[],
      addEventListener: () => undefined,
      removeEventListener: () => undefined,
      speak: () => undefined,
      cancel: () => undefined,
    };
    (window as unknown as { speechSynthesis: typeof synth }).speechSynthesis = synth;
    const status = await cantoneseTtsStatus();
    expect(status.state).toBe('no-voice');
  });

  it('cantoneseTtsStatus：命中粤语嗓音 → ok', async () => {
    const synth = {
      getVoices: () => [{ lang: 'zh-HK' }, { lang: 'en-US' }] as SpeechSynthesisVoice[],
      addEventListener: () => undefined,
      removeEventListener: () => undefined,
      speak: () => undefined,
      cancel: () => undefined,
    };
    (window as unknown as { speechSynthesis: typeof synth }).speechSynthesis = synth;
    const status = await cantoneseTtsStatus();
    expect(status.state).toBe('ok');
    expect(status.voice?.lang).toBe('zh-HK');
  });
});
