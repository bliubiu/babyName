'use client';

import { useState } from 'react';
import type { Hexagram, HexagramInterpretation } from '@/types';
import {
  TRIGRAM_GUADE,
  TRIGRAM_NAMES,
  TRIGRAM_SYMBOLS,
  TRIGRAM_WUXING,
  buildHexagramYaoLines,
  getYaoName,
  isYangYao,
} from '@/lib/hexagram';
import HexagramLines from './HexagramLines';

interface HexagramDisplayProps {
  hexagram: Hexagram | null;
  /** 六爻值（自下而上）：优先用梅花详情的本卦爻值，缺省时按上下卦推导 */
  yaoLines?: number[];
  /** 动爻位置 1-6（自下而上） */
  movingYao?: number | null;
  /** 64 卦差异化四象限解读（docs/42 P1-⑦ 接线），缺省不渲染 */
  interp?: HexagramInterpretation | null;
  /** 嵌入模式：去除外层卡片与标题，由统一折叠容器提供（结果页展示优化） */
  embedded?: boolean;
}

export default function HexagramDisplay({
  hexagram,
  yaoLines,
  movingYao,
  interp,
  embedded = false,
}: HexagramDisplayProps) {
  const [showYaoText, setShowYaoText] = useState(false);

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
    <div className={embedded ? 'animate-fade-in-up' : 'card mb-4 md:mb-6 animate-fade-in-up'}>
      {!embedded && (
        <div className="flex items-center gap-3 mb-4">
          <h3 className="section-title">易经卦象</h3>
        </div>
      )}

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

        {/* 上下卦：卦名 + 五行 + 卦德（此前误显示八卦序号数字） */}
        <div className="grid grid-cols-2 gap-3">
          <div className="text-center p-3 bg-warm-white/60 rounded-xl">
            <span className="text-xs text-jade">上卦</span>
            <p className="font-serif text-ink mt-1">
              {TRIGRAM_SYMBOLS[hexagram.upper_trigram]}{' '}
              {TRIGRAM_NAMES[hexagram.upper_trigram] ?? '?'}卦 ·{' '}
              {TRIGRAM_WUXING[hexagram.upper_trigram] ?? '?'}
              {TRIGRAM_GUADE[hexagram.upper_trigram] &&
                ` · ${TRIGRAM_GUADE[hexagram.upper_trigram]}`}
            </p>
          </div>
          <div className="text-center p-3 bg-warm-white/60 rounded-xl">
            <span className="text-xs text-jade">下卦</span>
            <p className="font-serif text-ink mt-1">
              {TRIGRAM_SYMBOLS[hexagram.lower_trigram]}{' '}
              {TRIGRAM_NAMES[hexagram.lower_trigram] ?? '?'}卦 ·{' '}
              {TRIGRAM_WUXING[hexagram.lower_trigram] ?? '?'}
              {TRIGRAM_GUADE[hexagram.lower_trigram] &&
                ` · ${TRIGRAM_GUADE[hexagram.lower_trigram]}`}
            </p>
          </div>
        </div>

        {/* 卦德 */}
        {hexagram.gua_de && (
          <div className="text-center text-xs text-jade/70">
            卦德：{hexagram.gua_de}
          </div>
        )}

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

        {/* 四象限差异化解读（docs/42 P1-⑦ 接线） */}
        {interp && (
          <div className="grid grid-cols-2 md:grid-cols-4 gap-2">
            {(
              [
                ['事业', interp.career],
                ['感情', interp.love],
                ['健康', interp.health],
                ['财运', interp.fortune],
              ] as const
            ).map(([label, text]) => (
              <div key={label} className="p-2.5 bg-warm-white/60 rounded-xl">
                <span className="text-xs text-jade font-medium">{label}</span>
                <p className="text-ink text-xs mt-1 leading-relaxed">{text}</p>
              </div>
            ))}
            {(interp.auspicious || interp.inauspicious) && (
              <div className="col-span-2 md:col-span-4 p-2.5 bg-warm-white/60 rounded-xl">
                <p className="text-xs text-ink-light/80 leading-relaxed">
                  {interp.auspicious && (
                    <>
                      <span className="text-jade font-medium">宜</span>
                      ：{interp.auspicious}
                    </>
                  )}
                  {interp.auspicious && interp.inauspicious && '　'}
                  {interp.inauspicious && (
                    <>
                      <span className="text-crimson/70 font-medium">忌</span>
                      ：{interp.inauspicious}
                    </>
                  )}
                </p>
              </div>
            )}
          </div>
        )}

        {/* 逐爻经文：爻辞 + 小象传（折叠，docs/42 P1 数据已结构化，此前无出口） */}
        {(hexagram.yao_ci?.length ?? 0) > 0 && (
          <div>
            <button
              type="button"
              onClick={() => setShowYaoText((v) => !v)}
              className="w-full flex items-center justify-center gap-1.5 text-xs text-jade/70 hover:text-jade transition-colors py-1"
              aria-expanded={showYaoText}
            >
              {showYaoText ? '收起逐爻经文' : '展开逐爻经文（爻辞 · 小象）'}
            </button>
            {showYaoText && (
              <ol className="space-y-2 mt-2">
                {hexagram.yao_ci.map((ci, idx) => {
                  const yaoName = getYaoName(idx + 1, lines[idx] ?? 0);
                  const xiaoXiang = hexagram.xiao_xiang?.[idx];
                  return (
                    <li
                      key={idx}
                      className={`p-2.5 rounded-xl ${
                        movingYao === idx + 1
                          ? 'bg-crimson/5 border border-crimson/20'
                          : 'bg-warm-white/60'
                      }`}
                    >
                      <span
                        className={`text-xs font-serif font-medium ${
                          movingYao === idx + 1 ? 'text-crimson' : 'text-jade'
                        }`}
                      >
                        {yaoName || `第${idx + 1}爻`}
                        {movingYao === idx + 1 && '（动）'}
                        {isYangYao(lines[idx] ?? 0) ? ' ⚊' : ' ⚋'}
                      </span>
                      <p className="text-ink text-sm mt-1 leading-relaxed">{ci}</p>
                      {xiaoXiang && (
                        <p className="text-ink-light/70 text-xs mt-1 leading-relaxed">
                          《象》曰：{xiaoXiang}
                        </p>
                      )}
                    </li>
                  );
                })}
              </ol>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
