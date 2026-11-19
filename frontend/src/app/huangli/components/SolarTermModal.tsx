'use client';

import { useEffect, useRef } from 'react';
import {
  SOLAR_TERM_KNOWLEDGE,
  type SolarTermKnowledge,
} from '../solarTermKnowledge';

interface SolarTermModalProps {
  /** 节气名（来自后端 lunar-month 的 solarTerm 字段） */
  term: string;
  /** 该节气对应的公历日期，用于标题展示 */
  date?: Date;
  onClose: () => void;
}

const seasonColor: Record<SolarTermKnowledge['season'], string> = {
  春: 'text-emerald-600 bg-emerald-50',
  夏: 'text-red-500 bg-red-50',
  秋: 'text-amber-600 bg-amber-50',
  冬: 'text-blue-500 bg-blue-50',
};

/**
 * 节气科普弹窗。参照 FullReport 的遮罩 + dialog 模式：
 * 独立遮罩按钮点击关闭，Esc 关闭，打开时聚焦对话框，aria 语义完整。
 */
export function SolarTermModal({ term, date, onClose }: SolarTermModalProps) {
  const dialogRef = useRef<HTMLDivElement>(null);
  const knowledge = SOLAR_TERM_KNOWLEDGE[term];

  useEffect(() => {
    dialogRef.current?.focus();
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [onClose]);

  if (!knowledge) return null;

  const dateLabel = date
    ? `${date.getFullYear()}年${date.getMonth() + 1}月${date.getDate()}日`
    : '';

  return (
    <div className="fixed inset-0 flex items-center justify-center z-50 overflow-y-auto">
      <button
        type="button"
        aria-label="关闭节气科普"
        onClick={onClose}
        className="absolute inset-0 w-full h-full bg-ink/30 cursor-default"
      />
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby="solar-term-title"
        tabIndex={-1}
        className="relative z-10 bg-warm-white-light w-full max-w-lg mx-4 my-8 rounded-xl shadow-xl border border-paper overflow-hidden outline-none"
      >
        {/* 头部：节气名 + 季节徽标 */}
        <div className="sticky top-0 bg-warm-white border-b border-paper px-6 py-4 flex justify-between items-center z-10">
          <div className="flex items-center gap-3">
            <h2 id="solar-term-title" className="text-2xl font-serif text-ink">{knowledge.name}</h2>
            <span className={`px-2 py-0.5 rounded-full text-xs ${seasonColor[knowledge.season]}`}>
              {knowledge.season}
            </span>
          </div>
          <button
            onClick={onClose}
            aria-label="关闭节气科普"
            className="p-2 text-ink-light hover:text-crimson transition-colors text-xl leading-none"
            title="关闭"
          >
            ×
          </button>
        </div>

        <div className="p-6 space-y-5 max-h-[65vh] overflow-y-auto">
          {/* 副标题：拼音 / 别称 / 日期 */}
          <div className="text-sm text-jade-dark/90 space-x-3">
            <span className="font-serif tracking-wide">{knowledge.pinyin}</span>
            {knowledge.alias && <span>· 又称{knowledge.alias}</span>}
            {dateLabel && <span>· {dateLabel}</span>}
          </div>

          {/* 气候总述 */}
          <section aria-label="气候">
            <h3 className="section-title mb-2">气候物候</h3>
            <p className="text-sm text-ink leading-relaxed">{knowledge.climate}</p>
          </section>

          {/* 三候：三张小卡横向铺开，窄屏自动换行 */}
          <section aria-label="三候">
            <h3 className="section-title mb-2">节气三候</h3>
            <div className="grid grid-cols-3 gap-2">
              {knowledge.phenology.map((p, i) => (
                <div key={p} className="rounded-lg bg-warm-white border border-paper/60 p-3 text-center">
                  <div className="text-[10px] text-jade-dark/90 mb-1">{['初候', '二候', '三候'][i]}</div>
                  <div className="text-sm text-ink font-medium">{p}</div>
                </div>
              ))}
            </div>
          </section>

          {/* 民俗 */}
          <section aria-label="习俗">
            <h3 className="section-title mb-2">传统习俗</h3>
            <ul className="flex flex-wrap gap-2">
              {knowledge.customs.map(c => (
                <li key={c} className="px-3 py-1 rounded-full bg-warm-white border border-paper/60 text-sm text-ink-light">
                  {c}
                </li>
              ))}
            </ul>
          </section>

          {/* 故事 / 来历 */}
          <section aria-label="故事">
            <h3 className="section-title mb-2">节气故事</h3>
            <p className="text-sm text-ink leading-relaxed">{knowledge.story}</p>
          </section>

          {/* 诗句卡片：复用 consultation-sheet 视觉语言 */}
          <section aria-label="诗词">
            <div className="consultation-sheet corner-decor rounded-lg p-4">
              <p className="font-serif text-ink text-center leading-relaxed">{knowledge.poem.text}</p>
              <p className="text-right text-xs text-jade-dark/90 mt-2">— {knowledge.poem.author}《{knowledge.poem.title}》</p>
            </div>
          </section>

          <p className="text-[11px] text-jade-dark/90 text-center">
            内容仅作传统文化科普，不构成气象或养生建议
          </p>
        </div>
      </div>
    </div>
  );
}
