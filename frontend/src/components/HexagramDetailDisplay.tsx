'use client';

import type { HexagramDetail, TiYong } from '@/types';
import { buildHexagramYaoLines, getYaoName } from '@/lib/hexagram';
import HexagramLines from './HexagramLines';

interface HexagramDetailDisplayProps {
  detail: HexagramDetail;
  /** 体用断卦（docs/42 P1-③），缺省时不渲染该块 */
  tiyong?: TiYong | null;
  /** 嵌入模式：去除外层卡片与标题（用于名字详解内嵌，docs/42 P3-11） */
  embedded?: boolean;
}

/** 三卦小卡标题 → 辅助说明 */
const SLOT_LABELS: Record<string, string> = {
  original: '事情本体',
  inter: '发展过程',
  changed: '发展趋势',
};

/** 体用吉凶 → 徽章配色（吉/中/凶） */
function verdictBadgeClass(verdict: string): string {
  switch (verdict) {
    case '吉':
      return 'bg-jade/10 text-jade border-jade/30';
    case '凶':
      return 'bg-crimson/10 text-crimson border-crimson/30';
    default:
      return 'bg-gold/10 text-gold-dark border-gold/30';
  }
}

interface HexagramSlot {
  key: string;
  label: string;
  name: string;
  symbol: string;
  yaoLines: number[];
  highlight?: boolean;
}

/**
 * 梅花卦象详解（docs/42 P2-9/P2-10）：本/互/变三卦并排 + 动爻标记与爻辞
 * + 体用断卦结构化展示。
 */
export default function HexagramDetailDisplay({
  detail,
  tiyong,
  embedded = false,
}: HexagramDetailDisplayProps) {
  const movingName = getYaoName(
    detail.moving_yao,
    detail.original_yao_lines[detail.moving_yao - 1]
  );
  const movingYaoCi =
    detail.original_hexagram.yao_ci?.[detail.moving_yao - 1] ?? '';

  const slots: HexagramSlot[] = [
    {
      key: 'original',
      label: '本卦',
      name: detail.original_hexagram.name,
      symbol: detail.original_hexagram.symbol,
      yaoLines: detail.original_yao_lines,
      highlight: true,
    },
    {
      key: 'inter',
      label: '互卦',
      name: detail.inter_hexagram?.name ?? '',
      symbol: detail.inter_hexagram?.symbol ?? '',
      yaoLines: detail.inter_hexagram
        ? buildHexagramYaoLines(
            detail.inter_hexagram.upper_trigram,
            detail.inter_hexagram.lower_trigram
          )
        : buildHexagramYaoLines(detail.inter_upper, detail.inter_lower),
    },
    {
      key: 'changed',
      label: '变卦',
      name: detail.changed_hexagram?.name ?? '',
      symbol: detail.changed_hexagram?.symbol ?? '',
      yaoLines:
        detail.changed_yao_lines?.length === 6
          ? detail.changed_yao_lines
          : detail.changed_hexagram
            ? buildHexagramYaoLines(
                detail.changed_hexagram.upper_trigram,
                detail.changed_hexagram.lower_trigram
              )
            : [],
    },
  ];

  return (
    <div className={embedded ? 'animate-fade-in-up' : 'card mb-4 md:mb-6 animate-fade-in-up'}>
      {!embedded && (
        <div className="flex items-center gap-3 mb-4">
          <h3 className="section-title">卦象详解</h3>
          <span className="text-xs text-ink-light/60">
            本 · 互 · 变 三卦并参，第{detail.moving_yao}爻动
          </span>
        </div>
      )}
      {embedded && (
        <div className="flex items-center gap-2 mb-3">
          <span className="text-xs text-jade font-medium">梅花卦象</span>
          <span className="text-[11px] text-ink-light/50">
            本 · 互 · 变 并参，第{detail.moving_yao}爻动
          </span>
        </div>
      )}

      {/* 三卦并排 */}
      <div className="grid grid-cols-3 gap-2 md:gap-3 mb-4">
        {slots.map((slot) => (
          <div
            key={slot.key}
            className={`rounded-xl p-2.5 md:p-3 text-center ${
              slot.highlight
                ? 'bg-crimson/5 border border-crimson/20'
                : 'bg-warm-white/60'
            }`}
          >
            <div className="flex items-center justify-center gap-1.5 mb-2">
              <span
                className={`text-xs font-medium ${slot.highlight ? 'text-crimson' : 'text-jade'}`}
              >
                {slot.label}
              </span>
              <span className="text-[10px] text-ink-light/50 hidden md:inline">
                {SLOT_LABELS[slot.key]}
              </span>
            </div>
            <HexagramLines
              yaoLines={slot.yaoLines}
              movingYao={slot.highlight ? detail.moving_yao : null}
              className="w-16 md:w-20 h-auto mx-auto"
            />
            <div className="mt-2 font-serif text-sm text-ink">
              {slot.name || '—'}
              {slot.highlight && movingName && (
                <span className="ml-1 text-xs text-crimson">{movingName}动</span>
              )}
            </div>
            {slot.symbol && (
              <div className="text-xs text-ink-light/50 mt-0.5" aria-hidden>
                {slot.symbol}
              </div>
            )}
          </div>
        ))}
      </div>

      {/* 动爻爻辞 */}
      {movingName && movingYaoCi && (
        <div className="p-3 bg-warm-white/60 rounded-xl mb-3">
          <span className="text-xs text-jade font-medium">
            动爻 {movingName} 爻辞
          </span>
          <p className="text-ink text-sm mt-1.5 leading-relaxed">{movingYaoCi}</p>
        </div>
      )}

      {/* 体用断卦 */}
      {tiyong && (
        <div className="p-3 bg-warm-white/60 rounded-xl mb-3">
          <div className="flex flex-wrap items-center gap-2 mb-2">
            <span className="text-xs text-jade font-medium">体用断卦</span>
            {tiyong.verdict && (
              <span
                className={`px-2 py-0.5 rounded-full border text-xs font-medium ${verdictBadgeClass(tiyong.verdict)}`}
              >
                {tiyong.verdict}
              </span>
            )}
          </div>
          <p className="text-ink text-sm leading-relaxed">
            {`体卦「${tiyong.ti_gua}」（${tiyong.ti_wuxing}），用卦「${tiyong.yong_gua}」（${tiyong.yong_wuxing}），${tiyong.relation}${tiyong.wang_shuai ? `，体卦旺衰：${tiyong.wang_shuai}` : ''}。`}
          </p>
          {tiyong.xiyong_note && (
            <p className="text-ink-light/80 text-xs mt-1.5 leading-relaxed">
              {tiyong.xiyong_note}
            </p>
          )}
          {tiyong.text && (
            <p className="text-ink text-sm mt-2 leading-relaxed">{tiyong.text}</p>
          )}
        </div>
      )}

      {/* 综合解读 */}
      {detail.interpretation && (
        <div className="p-3 bg-crimson/3 rounded-xl border border-crimson/8">
          <span className="text-xs text-crimson/70 font-medium">综合解读</span>
          <p className="text-ink text-sm mt-1.5 leading-relaxed">
            {detail.interpretation}
          </p>
        </div>
      )}
    </div>
  );
}
