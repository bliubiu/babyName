import { describe, it, expect, afterEach, vi } from 'vitest';
import '@testing-library/jest-dom/vitest';
import { render, screen, waitFor, act } from '@testing-library/react';
import CantoneseScore from './CantoneseScore';
import type { Name } from '@/types';

const baseName: Name = {
  surname: '王',
  given_name: '梓文',
  pinyin: 'wang zi wen',
  gender: 'male',
  score: 88,
  meaning: '',
  wuxing: '木',
  strokes: 12,
};

describe('CantoneseScore', () => {
  it('渲染粤拼标注（逐音节 + 声调）', () => {
    render(<CantoneseScore name={{ ...baseName, jyutping: 'wong4 zi2 man4' }} />);
    expect(screen.getByText('wong4')).toBeInTheDocument();
    expect(screen.getByText('zi2')).toBeInTheDocument();
    expect(screen.getByText('man4')).toBeInTheDocument();
    // 调号 4 = 阳平（低降），平仄标签为「仄」（wong4/man4 两节均命中）
    expect(screen.getAllByText('4 阳平（低降）')).toHaveLength(2);
    expect(screen.getByText('2 阴上（高升）')).toBeInTheDocument();
  });

  it('渲染粤语分与维度明细', () => {
    render(
      <CantoneseScore
        name={{
          ...baseName,
          jyutping: 'wong4 zi2 man4',
          cantonese_score: 82.5,
          cantonese_score_detail: [
            { name: '粤语音律', score: 85, detail: '平仄交替，韵母迴避' },
            { name: '粤语谐音', score: 80, detail: '未发现不雅谐音' },
          ],
        }}
      />
    );
    expect(screen.getByText('82.5')).toBeInTheDocument();
    expect(screen.getByText('粤语音律')).toBeInTheDocument();
    expect(screen.getByText('粤语谐音')).toBeInTheDocument();
    expect(screen.getByText('平仄交替，韵母迴避')).toBeInTheDocument();
  });

  it('无声拼且无粤语分时不渲染', () => {
    const { container } = render(<CantoneseScore name={baseName} />);
    expect(container).toBeEmptyDOMElement();
  });

  it('仅有粤拼（开关关闭）时展示标注但不展示粤语分', () => {
    render(<CantoneseScore name={{ ...baseName, jyutping: 'wong4 zi2 man4' }} />);
    expect(screen.getByText('粤语评分')).toBeInTheDocument();
    expect(screen.queryByText('粤语分分解')).not.toBeInTheDocument();
  });

  it('happy-dom 无 speechSynthesis 时隐藏试听按钮', async () => {
    await act(async () => {
      render(<CantoneseScore name={{ ...baseName, jyutping: 'wong4 zi2 man4', cantonese_score: 70 }} />);
    });
    expect(screen.queryByTitle('粤语试听')).not.toBeInTheDocument();
  });

  it('本机无粤语嗓音（no-voice）时按钮禁用并给出提示', async () => {
    const synth = {
      getVoices: () => [] as SpeechSynthesisVoice[],
      addEventListener: () => undefined,
      removeEventListener: () => undefined,
      speak: () => undefined,
      cancel: () => undefined,
    };
    (window as unknown as { speechSynthesis: typeof synth }).speechSynthesis = synth;

    render(<CantoneseScore name={{ ...baseName, jyutping: 'wong4 zi2 man4', cantonese_score: 70 }} />);
    const btn = await waitFor(() => screen.getByTitle(/未安装粤语语音包|无法朗读粤语/), { timeout: 4000 });
    expect(btn).toBeDisabled();
  });

  it('本机有粤语嗓音（ok）时显示可用试听按钮', async () => {
    const synth = {
      getVoices: () => [{ lang: 'zh-HK' }, { lang: 'en-US' }] as SpeechSynthesisVoice[],
      addEventListener: () => undefined,
      removeEventListener: () => undefined,
      speak: () => undefined,
      cancel: () => undefined,
    };
    (window as unknown as { speechSynthesis: typeof synth }).speechSynthesis = synth;

    render(<CantoneseScore name={{ ...baseName, jyutping: 'wong4 zi2 man4', cantonese_score: 70 }} />);
    const btn = await waitFor(() => screen.getByTitle('粤语试听'));
    expect(btn).not.toBeDisabled();
  });

  it('离线引擎可用时优先走 /api/v1/tts POST（body 含 jyutping）', async () => {
    const fetchMock = vi.fn(async (url: RequestInfo | URL, init?: RequestInit) => {
      const isPost = init?.method === 'POST';
      if (!isPost) {
        return new Response(JSON.stringify({ available: true }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        });
      }
      return new Response(new Blob([new Uint8Array([0x52, 0x49, 0x46, 0x46, 0, 0, 0, 0, 0x57, 0x41, 0x56, 0x45])], {
        type: 'audio/wav',
      }), {
        status: 200,
        headers: { 'Content-Type': 'audio/wav' },
      });
    });
    vi.stubGlobal('fetch', fetchMock);

    render(<CantoneseScore name={{ ...baseName, jyutping: 'wong4 zi2 man4', cantonese_score: 70 }} />);
    const btn = await waitFor(() => screen.getByTitle('粤语试听（离线引擎）'), { timeout: 4000 });
    await act(async () => {
      btn.click();
    });
    const postCall = fetchMock.mock.calls.find((c) => (c[1] as RequestInit | undefined)?.method === 'POST');
    expect(postCall).toBeTruthy();
    expect(JSON.parse((postCall as [RequestInfo | URL, RequestInit])[1].body as string)).toEqual({
      jyutping: 'wong4 zi2 man4',
    });
  });
});

afterEach(() => {
  delete (window as unknown as { speechSynthesis?: unknown }).speechSynthesis;
  vi.unstubAllGlobals();
});
