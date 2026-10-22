// 粤拼（Jyutping）展示与粤语 TTS 朗读工具。
//
// 数据来源：后端 domain/yue 包（//go:embed jyutping.json），经
// NameAnalysis.jyutping 透出，形如 "wong4 zi2 man4"（姓 + 名各字粤拼，
// 空格分隔；单名时两节，双名时三节）。粤拼无声调数据时该字段为空串，
// 本模块所有函数对空输入安全降级。

/** 单个粤拼音节（含调号） */
export interface JyutpingSyllable {
  /** 原始音节（含调号数字），如 "nei5" */
  raw: string;
  /** 粤语声调 1-6；无法解析时为 0 */
  tone: number;
  /** 是否入声（-p/-t/-k 塞尾，发音短促急收） */
  isEntering: boolean;
}

/** 粤语六调名称（声调描述含调值走势，便于非粤语用户理解） */
const TONE_NAMES: Record<number, string> = {
  1: '阴平（高平）',
  2: '阴上（高升）',
  3: '阴去（中平）',
  4: '阳平（低降）',
  5: '阳上（低升）',
  6: '阳去（中平）',
};

/** 粤语声调平仄二分（音律评分基础）：1/2 为平，3/4/5/6 为仄 */
export function toneClass(tone: number): '平' | '仄' | '' {
  if (tone === 1 || tone === 2) return '平';
  if (tone >= 3 && tone <= 6) return '仄';
  return '';
}

/** 声调中文名（如 "3 阴去（中平）"）；调号缺失时返回空串 */
export function toneLabel(tone: number): string {
  if (!tone || !TONE_NAMES[tone]) return '';
  return `${tone} ${TONE_NAMES[tone]}`;
}

/**
 * 解析粤拼字符串为音节数组。
 * 输入 "wong4 zi2 man4" → [{raw:'wong4',tone:4,...}, ...]。
 * 容忍多余空白与无声调音节（如 "wong" → tone 0）。
 */
export function parseJyutpingSyllables(jyutping: string | undefined | null): JyutpingSyllable[] {
  if (!jyutping) return [];
  return jyutping
    .split(/\s+/)
    .map((token) => token.trim())
    .filter(Boolean)
    .map((raw) => {
      const match = /^([a-z]+?)([1-6])$/.exec(raw);
      const tone = match ? Number(match[2]) : 0;
      const base = match ? match[1] : raw;
      return {
        raw,
        tone,
        isEntering: /[ptk]$/.test(base),
      };
    });
}

/**
 * 生成供 TTS 朗读的纯文本：粤拼调号数字会导致朗读器按字母念，
 * 此处直接朗读汉字原名（调用方传入 fullName）。
 * 保留函数是为了把「朗读什么」的决策集中在一处。
 */
export function ttsTextFor(fullName: string): string {
  return fullName.trim();
}

// ---- 粤语 TTS（Web Speech API）检测与降级 ----

export type CantoneseTtsState = 'unsupported' | 'loading' | 'no-voice' | 'ok';

export interface CantoneseTtsStatus {
  /** 能力状态 */
  state: CantoneseTtsState;
  /** state==='ok' 时的命中嗓音 */
  voice?: SpeechSynthesisVoice;
  /** 面向用户的提示（no-voice / unsupported 时非空） */
  message?: string;
}

// 粤语（香港）相关语言标签，优先精确匹配
const CANTONESE_LANGS = ['zh-HK', 'zh-Hant-HK', 'yue-HK', 'yue', 'zh-yue'];
// 中文（含普通话）标签，作为粤语嗓音缺失时的回退
const CHINESE_LANGS = ['zh', 'zh-CN', 'zh-TW', 'zh-Hant', 'cmn', 'cmn-Hans', 'cmn-Hant'];

/**
 * 浏览器是否提供语音合成 API（同步、保守判断，不探测具体嗓音）。
 * 仅用于「连 API 都没有」的最粗粒度禁用；具体嗓音能力见 cantoneseTtsStatus()。
 */
export function cantoneseTtsSupported(): boolean {
  return typeof window !== 'undefined' && 'speechSynthesis' in window;
}

function voicesNow(): SpeechSynthesisVoice[] {
  if (!cantoneseTtsSupported()) return [];
  try {
    return window.speechSynthesis.getVoices() || [];
  } catch {
    return [];
  }
}

/**
 * 在语音列表中挑选最优粤语嗓音：优先精确匹配 zh-HK/yue，
 * 其次任意中文（普通话）嗓音作为回退。两者皆无则返回 null。
 */
export function pickCantoneseVoice(voices: SpeechSynthesisVoice[]): SpeechSynthesisVoice | null {
  if (!voices || voices.length === 0) return null;
  const hk = voices.find((v) => CANTONESE_LANGS.includes(v.lang));
  if (hk) return hk;
  const zh = voices.find((v) => CHINESE_LANGS.some((l) => v.lang === l || v.lang.startsWith(l + '-')));
  return zh || null;
}

// 缓存已解析的嗓音，避免每次朗读都重新遍历 / 重新等待 voiceschanged
let cachedVoice: SpeechSynthesisVoice | null | undefined;

/**
 * 异步解析本机粤语嗓音。getVoices() 在 Chrome 等浏览器初始返回空数组，
 * 需监听 voiceschanged 事件后才填充；最多等待 timeoutMs 后取当前快照兜底。
 * 返回 null 表示未找到任何可用嗓音（粤语或中文）。
 */
export function resolveCantoneseVoice(timeoutMs = 3000): Promise<SpeechSynthesisVoice | null> {
  if (!cantoneseTtsSupported()) return Promise.resolve(null);

  const existing = pickCantoneseVoice(voicesNow());
  if (existing) {
    cachedVoice = existing;
    return Promise.resolve(existing);
  }

  return new Promise((resolve) => {
    let done = false;

    const settle = (voice: SpeechSynthesisVoice | null) => {
      if (done) return;
      done = true;
      cachedVoice = voice;
      if (timer) clearTimeout(timer);
      try {
        window.speechSynthesis.removeEventListener('voiceschanged', onChanged);
      } catch {
        /* 部分环境无 removeEventListener 重载，忽略 */
      }
      resolve(voice);
    };

    const onChanged = () => {
      const v = pickCantoneseVoice(voicesNow());
      if (v) settle(v);
    };

    try {
      window.speechSynthesis.addEventListener('voiceschanged', onChanged);
    } catch {
      /* 不支持事件监听则仅依赖超时兜底 */
    }

    const timer = setTimeout(() => settle(pickCantoneseVoice(voicesNow())), timeoutMs);
  });
}

/**
 * 探测粤语 TTS 能力并返回给用户的状态：
 * - unsupported：浏览器无 Web Speech API
 * - no-voice：有 API 但本机未安装粤语（甚至中文）语音包
 * - ok：已找到可用嗓音（优先粤语；若仅普通话则 message 提示回退）
 */
export async function cantoneseTtsStatus(): Promise<CantoneseTtsStatus> {
  if (!cantoneseTtsSupported()) {
    return { state: 'unsupported', message: '当前浏览器不支持语音合成（Web Speech API），无法试听。' };
  }
  const voice = await resolveCantoneseVoice();
  if (!voice) {
    return {
      state: 'no-voice',
      message: '未检测到粤语 / 中文语音包，无法朗读粤语；请在本机安装语音（如 macOS 在「语音」设置中添加粤语）。',
    };
  }
  const isCantonese = CANTONESE_LANGS.includes(voice.lang);
  return {
    state: 'ok',
    voice,
    message: isCantonese ? undefined : '未找到粤语嗓音，已回退到普通话嗓音朗读（非粤语发音）。',
  };
}

/**
 * 用粤语（zh-HK）朗读文本。
 * - 显式挑选本机粤语嗓音（不可用则回退中文），避免浏览器静默退化成错误的普通话嗓音。
 * - 重复调用先取消上一次朗读，避免声音叠加。
 * 返回 true 表示已发起朗读；false 表示不支持 / 无嗓音 / 参数为空。
 */
export async function speakCantonese(text: string): Promise<boolean> {
  if (!text || !cantoneseTtsSupported()) return false;
  try {
    const voice = cachedVoice ?? (await resolveCantoneseVoice());
    const utterance = new SpeechSynthesisUtterance(text);
    if (voice) {
      utterance.voice = voice;
      utterance.lang = voice.lang;
    } else {
      utterance.lang = 'zh-HK';
    }
    utterance.rate = 0.9;
    window.speechSynthesis.cancel();
    window.speechSynthesis.speak(utterance);
    return true;
  } catch {
    return false;
  }
}

/** 停止当前粤语朗读 */
export function cancelCantoneseSpeech(): void {
  if (!cantoneseTtsSupported()) return;
  try {
    window.speechSynthesis.cancel();
  } catch {
    /* 忽略 */
  }
}
