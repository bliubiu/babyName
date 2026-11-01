'use client';

import { isYangYao } from '@/lib/hexagram';

interface HexagramLinesProps {
  /** 六爻值，自下而上，7=少阳（阳爻）/ 8=少阴（阴爻） */
  yaoLines: number[];
  /** 动爻位置 1-6（自下而上），非空时高亮并标记 */
  movingYao?: number | null;
  /** 是否显示动爻标记（〇/×），默认显示 */
  showMarker?: boolean;
  className?: string;
}

const BAR_W = 108;
const YIN_GAP = 16;
const SEG_W = (BAR_W - YIN_GAP) / 2;
const ROW_H = 15;
const BAR_H = 10;
const VIEW_W = BAR_W + 20;
const VIEW_H = ROW_H * 6;

/**
 * 六爻画 SVG 组件（docs/42 P2-8）：替代 Unicode 符号拼接，
 * 自下而上绘制，动爻以朱色高亮 + 脉动动画 + 传统 〇/× 标记（P2-9 动爻动画）。
 */
export default function HexagramLines({
  yaoLines,
  movingYao,
  showMarker = true,
  className,
}: HexagramLinesProps) {
  if (!yaoLines || yaoLines.length !== 6) {
    return null;
  }

  return (
    <svg
      viewBox={`0 0 ${VIEW_W} ${VIEW_H}`}
      role="img"
      aria-label={`六爻图（自下而上）：${yaoLines.map((v) => (isYangYao(v) ? '阳' : '阴')).join('、')}${movingYao ? `，第${movingYao}爻动` : ''}`}
      className={className}
      data-testid="hexagram-lines"
    >
      {yaoLines.map((value, idx) => {
        const position = idx + 1;
        const y = VIEW_H - ROW_H * position + (ROW_H - BAR_H) / 2;
        const isMoving = movingYao === position;
        const fillClass = isMoving ? 'fill-crimson' : 'fill-ink/75';
        const rects = isYangYao(value) ? (
          <rect x={0} y={y} width={BAR_W} height={BAR_H} rx={2} className={fillClass} />
        ) : (
          <>
            <rect x={0} y={y} width={SEG_W} height={BAR_H} rx={2} className={fillClass} />
            <rect
              x={SEG_W + YIN_GAP}
              y={y}
              width={SEG_W}
              height={BAR_H}
              rx={2}
              className={fillClass}
            />
          </>
        );
        return (
          <g
            key={position}
            data-testid={`yao-${position}`}
            data-moving={isMoving || undefined}
          >
            {isMoving ? <g className="animate-pulse">{rects}</g> : rects}
            {isMoving && showMarker && (
              <text
                x={BAR_W + 6}
                y={y + BAR_H - 2}
                className="fill-crimson text-[11px] font-bold"
              >
                {isYangYao(value) ? '〇' : '×'}
              </text>
            )}
          </g>
        );
      })}
    </svg>
  );
}
