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

/** 浏览器是否支持粤语语音合成（Web Speech API） */
export function cantoneseTtsSupported(): boolean {
  return typeof window !== 'undefined' && 'speechSynthesis' in window;
}

/**
 * 用粤语（zh-HK）朗读文本。
 * 返回 false 表示浏览器不支持或参数为空（调用方据此禁用试听按钮）。
 * 重复调用会先取消上一次朗读，避免声音叠加。
 */
export function speakCantonese(text: string): boolean {
  if (!text || !cantoneseTtsSupported()) return false;
  const utterance = new SpeechSynthesisUtterance(text);
  utterance.lang = 'zh-HK';
  utterance.rate = 0.9;
  window.speechSynthesis.cancel();
  window.speechSynthesis.speak(utterance);
  return true;
}

/** 停止当前粤语朗读 */
export function cancelCantoneseSpeech(): void {
  if (!cantoneseTtsSupported()) return;
  window.speechSynthesis.cancel();
}
