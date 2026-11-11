'use client';

import { useState, useEffect } from 'react';
import { Name } from '@/types';
import { parseJyutpingSyllables, toneClass, toneLabel, speakCantonese, cantoneseTtsStatus, cancelCantoneseSpeech } from '@/lib/jyutping';
import type { JyutpingSyllable, CantoneseTtsStatus } from '@/lib/jyutping';
import { IconVolume } from './Icons';

interface CantoneseScoreProps {
  name: Name;
}

// 粤语分两个维度的固定配色（与普通话八维评分条区分，便于一眼识别）
const cantoneseColorMap: Record<string, string> = {
  '粤语音律': 'bg-sky-500',
  '粤语谐音': 'bg-rose-400',
};

/** 粤拼音节芯片：汉字 + 粤拼 + 声调；平/仄用不同底色区分 */
function SyllableChip({ char, jp }: { char?: string; jp: JyutpingSyllable }) {
  const cls = toneClass(jp.tone);
  return (
    <div className="flex flex-col items-center gap-0.5">
      {char && <span className="text-sm text-ink font-serif">{char}</span>}
      <span className="text-xs text-jade font-mono tracking-wide">{jp.raw}</span>
      {jp.tone > 0 ? (
        <span className={`text-xs px-1.5 py-0.5 rounded-full leading-none ${
          cls === '平' ? 'bg-jade/10 text-jade' : 'bg-amber-500/10 text-amber-600'
        }`}>
          {toneLabel(jp.tone)}
        </span>
      ) : (
        <span className="text-xs text-ink-light/70">无声调</span>
      )}
    </div>
  );
}

/**
 * 粤语评分区块：粤拼标注（声调）+ 粤语试听（TTS）+ 独立粤语分明细。
 *
 * 粤拼标注与开关无关（后端始终透传 jyutping）；粤语分仅在开启评分时非零。
 * 两者都没有时不渲染，避免给未开启粤语评分的用户展示空壳。
 */
export default function CantoneseScore({ name }: CantoneseScoreProps) {
  const [speaking, setSpeaking] = useState(false);
  const [ttsStatus, setTtsStatus] = useState<CantoneseTtsStatus | null>(null);
  const fullName = name.full_name || `${name.surname}${name.given_name}`;

  // 探测本机粤语 TTS 能力（嗓音可能异步填充，需等待 voiceschanged）；
  // 必须在所有提前 return 之前调用，遵守 Hook 顺序规则
  useEffect(() => {
    let active = true;
    cantoneseTtsStatus().then((s) => {
      if (active) setTtsStatus(s);
    });
    return () => {
      active = false;
    };
  }, [fullName]);

  const syllables = parseJyutpingSyllables(name.jyutping);
  const hasJyutping = syllables.length > 0;
  const cantoneseScore = name.cantonese_score ?? 0;
  const hasCantoneseScore = cantoneseScore > 0;
  const details = name.cantonese_score_detail ?? [];

  if (!hasJyutping && !hasCantoneseScore) return null;

  // 汉字与粤拼音节按位置对齐（姓+名共 N 字 ↔ N 个音节）；
  // 对不齐（ compound 姓或数据缺失）时退化为仅展示粤拼
  const chars = Array.from(fullName);
  const aligned = hasJyutping && chars.length === syllables.length;

  const handleSpeak = async () => {
    if (speaking) {
      cancelCantoneseSpeech();
      setSpeaking(false);
      return;
    }
    const ok = await speakCantonese(fullName);
    if (!ok) return;
    setSpeaking(true);
    // Web Speech 无可靠结束事件时长预估，用文本长度粗估朗读时长后复位按钮态
    const estimate = Math.max(1200, fullName.length * 700);
    window.setTimeout(() => setSpeaking(false), estimate);
  };

  // TTS 可用态：state 为 ok / no-voice 时渲染按钮（no-voice 禁用并提示）；
  // unsupported 或尚未探测完成（null）时不渲染，避免给出无效入口
  const ttsDisabled = ttsStatus?.state === 'no-voice';

  return (
    <div className="animate-fade-in-up space-y-4">
      <div className="flex items-center justify-between">
        <h4 className="font-serif text-base text-ink">粤语评分</h4>
        <div className="flex items-center gap-2">
          {hasCantoneseScore && (
            <div className="flex items-baseline gap-1">
              <span className="text-2xl font-bold text-gold">{cantoneseScore.toFixed(1)}</span>
              <span className="text-xs text-jade">粤语分</span>
            </div>
          )}
          {ttsStatus && ttsStatus.state !== 'unsupported' && (
            <button
              type="button"
              disabled={ttsDisabled}
              onClick={handleSpeak}
              aria-label={speaking ? `停止朗读 ${fullName} 粤语读音` : `试听 ${fullName} 粤语读音`}
              title={
                speaking
                  ? '停止朗读'
                  : ttsStatus?.state === 'no-voice'
                    ? ttsStatus.message ?? '未安装粤语语音包，无法试听'
                    : '粤语试听'
              }
              className={`p-1.5 rounded-full transition-all duration-200 hover:scale-110 ${
                speaking
                  ? 'text-crimson bg-crimson/10'
                  : ttsDisabled
                    ? 'text-ink-light/20 cursor-not-allowed'
                    : 'text-ink-light/30 hover:text-crimson'
              }`}
            >
              <IconVolume size={18} />
            </button>
          )}
        </div>
      </div>

      {/* 本机未安装粤语语音包时给出明确提示，避免用户误以为功能损坏 */}
      {ttsStatus?.state === 'no-voice' && ttsStatus.message && (
        <p className="text-xs text-amber-600/80 leading-relaxed">{ttsStatus.message}</p>
      )}

      {/* 粤拼标注（逐字音节 + 声调） */}
      {hasJyutping && (
        <div className="space-y-2">
          <div className="flex flex-wrap items-end gap-4">
            {syllables.map((jp, i) => (
              <SyllableChip key={i} char={aligned ? chars[i] : undefined} jp={jp} />
            ))}
          </div>
          <p className="text-xs text-jade/70 leading-relaxed">
            粤拼标注（Jyutping）以数字标注声调：1 阴平（高平）、2 阴上（高升）、3 阴去（中平）、
            4 阳平（低降）、5 阳上（低升）、6 阳去（中平）；平声（1/2）与仄声（3-6）交替
            是粤语音律评分的基础。以 -p/-t/-k 收尾的为入声，发音短促急收。
          </p>
        </div>
      )}

      {/* 粤语分明细（粤语音律 + 粤语谐音） */}
      {hasCantoneseScore && (
        <div className="space-y-2.5 pt-3 border-t border-paper/30">
          <p className="text-xs text-jade/70 tracking-wider">粤语分分解</p>
          <div className="score-bar mb-2">
            <div className="score-bar-fill" style={{ width: `${Math.min(100, cantoneseScore)}%` }} />
          </div>
          {details.length > 0 ? (
            details.map(d => (
              <div key={d.name} className="space-y-1">
                <div className="flex items-center gap-2">
                  <span className="text-xs text-ink-light/70 w-16 flex-shrink-0">{d.name}</span>
                  <div className="flex-1 h-2 bg-paper/40 rounded-full overflow-hidden">
                    <div
                      className={`h-full rounded-full ${cantoneseColorMap[d.name] || 'bg-jade'} transition-all duration-700`}
                      style={{ width: `${Math.min(100, d.score)}%` }}
                    />
                  </div>
                  <span className="text-xs text-ink-light/70 w-8 text-right font-medium">{d.score.toFixed(0)}</span>
                </div>
                {d.detail && (
                  <p className="text-xs text-jade/70 ml-16 leading-relaxed">{d.detail}</p>
                )}
              </div>
            ))
          ) : (
            <p className="text-xs text-jade/70 ml-16">粤语音律 + 粤语谐音加权聚合（粤语分已并入综合分）</p>
          )}
        </div>
      )}
    </div>
  );
}
