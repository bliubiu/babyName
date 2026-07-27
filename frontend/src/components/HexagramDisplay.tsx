'use client';

import type { Hexagram } from '@/types';

interface HexagramDisplayProps {
  hexagram: Hexagram | null;
}

export default function HexagramDisplay({ hexagram }: HexagramDisplayProps) {
  if (!hexagram) {
    return (
      <div className="text-center py-6 text-ink-light/40 text-sm">
        暂无卦象数据
      </div>
    );
  }

  return (
    <div className="card mb-4 md:mb-6 animate-fade-in-up">
      <div className="flex items-center gap-3 mb-4">
        <h3 className="section-title">易经卦象</h3>
      </div>

      <div className="space-y-4">
        {/* 卦名与符号 */}
        <div className="text-center py-3">
          {hexagram.symbol && (
            <span className="text-4xl font-serif text-ink block mb-2">{hexagram.symbol}</span>
          )}
          <span className="text-lg font-serif text-crimson">{hexagram.name}</span>
        </div>

        {/* 上下卦 */}
        <div className="grid grid-cols-2 gap-3">
          <div className="text-center p-3 bg-warm-white/60 rounded-xl">
            <span className="text-xs text-jade">上卦</span>
            <p className="font-serif text-ink mt-1">第{hexagram.upper_trigram}卦</p>
          </div>
          <div className="text-center p-3 bg-warm-white/60 rounded-xl">
            <span className="text-xs text-jade">下卦</span>
            <p className="font-serif text-ink mt-1">第{hexagram.lower_trigram}卦</p>
          </div>
        </div>

        {/* 卦辞 */}
        {hexagram.gua_ci && (
          <div className="p-3 bg-warm-white/60 rounded-xl">
            <span className="text-xs text-jade font-medium">卦辞</span>
            <p className="text-ink text-sm mt-1.5 leading-relaxed">{hexagram.gua_ci}</p>
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
