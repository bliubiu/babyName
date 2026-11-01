'use client';

import type { Hexagram } from '@/types';
import {
  TRIGRAM_NAMES,
  TRIGRAM_SYMBOLS,
  TRIGRAM_WUXING,
  buildHexagramYaoLines,
} from '@/lib/hexagram';
import HexagramLines from './HexagramLines';

interface HexagramDisplayProps {
  hexagram: Hexagram | null;
  /** 六爻值（自下而上）：优先用梅花详情的本卦爻值，缺省时按上下卦推导 */
  yaoLines?: number[];
  /** 动爻位置 1-6（自下而上） */
  movingYao?: number | null;
}

export default function HexagramDisplay({
  hexagram,
  yaoLines,
  movingYao,
}: HexagramDisplayProps) {
  if (!hexagram) {
    return (
      <div className="text-center py-6 text-ink-light/40 text-sm">
        暂无卦象数据
      </div>
    );
  }

  const lines =
    yaoLines && yaoLines.length === 6
      ? yaoLines
      : buildHexagramYaoLines(hexagram.upper_trigram, hexagram.lower_trigram);

  return (
    <div className="card mb-4 md:mb-6 animate-fade-in-up">
      <div className="flex items-center gap-3 mb-4">
        <h3 className="section-title">易经卦象</h3>
      </div>

      <div className="space-y-4">
        {/* 卦序、卦名与六爻画 */}
        <div className="flex items-center justify-center gap-6 py-3">
          <HexagramLines
            yaoLines={lines}
            movingYao={movingYao}
            className="w-28 md:w-32 h-auto"
          />
          <div className="text-center">
            <div className="text-xs text-jade/70 mb-1">第{hexagram.number}卦</div>
            <span className="text-lg font-serif text-crimson">{hexagram.name}</span>
            {hexagram.symbol && (
              <span className="block text-2xl font-serif text-ink/70 mt-1" aria-hidden>
                {hexagram.symbol}
              </span>
            )}
          </div>
        </div>

        {/* 上下卦：显示卦名与五行（此前误显示八卦序号数字） */}
        <div className="grid grid-cols-2 gap-3">
          <div className="text-center p-3 bg-warm-white/60 rounded-xl">
            <span className="text-xs text-jade">上卦</span>
            <p className="font-serif text-ink mt-1">
              {TRIGRAM_SYMBOLS[hexagram.upper_trigram]}{' '}
              {TRIGRAM_NAMES[hexagram.upper_trigram] ?? '?'}卦 ·{' '}
              {TRIGRAM_WUXING[hexagram.upper_trigram] ?? '?'}
            </p>
          </div>
          <div className="text-center p-3 bg-warm-white/60 rounded-xl">
            <span className="text-xs text-jade">下卦</span>
            <p className="font-serif text-ink mt-1">
              {TRIGRAM_SYMBOLS[hexagram.lower_trigram]}{' '}
              {TRIGRAM_NAMES[hexagram.lower_trigram] ?? '?'}卦 ·{' '}
              {TRIGRAM_WUXING[hexagram.lower_trigram] ?? '?'}
            </p>
          </div>
        </div>

        {/* 卦辞 */}
        {hexagram.gua_ci && (
          <div className="p-3 bg-warm-white/60 rounded-xl">
            <span className="text-xs text-jade font-medium">卦辞</span>
            <p className="text-ink text-sm mt-1.5 leading-relaxed">{hexagram.gua_ci}</p>
          </div>
        )}

        {/* 彖传 */}
        {hexagram.tuan_ci && (
          <div className="p-3 bg-warm-white/60 rounded-xl">
            <span className="text-xs text-jade font-medium">彖传</span>
            <p className="text-ink text-sm mt-1.5 leading-relaxed">{hexagram.tuan_ci}</p>
          </div>
        )}

        {/* 象辞 */}
        {hexagram.xiang_ci && (
          <div className="p-3 bg-warm-white/60 rounded-xl">
            <span className="text-xs text-jade font-medium">象辞</span>
            <p className="text-ink text-sm mt-1.5 leading-relaxed">{hexagram.xiang_ci}</p>
          </div>
        )}

        {/* 解读 */}
        {hexagram.interpretation && (
          <div className="p-3 bg-crimson/3 rounded-xl border border-crimson/8">
            <span className="text-xs text-crimson/70 font-medium">解读</span>
            <p className="text-ink text-sm mt-1.5 leading-relaxed">{hexagram.interpretation}</p>
          </div>
        )}
      </div>
    </div>
  );
}
