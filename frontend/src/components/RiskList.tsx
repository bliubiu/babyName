import { RiskItem } from '@/types';

// RiskList — 分级风险体检展示（docs/43 P2-4 前端）
//
// 生成结果页与测名页共用：数据同源（后端均由 fate.AssessNameRisks 产出），
// 展示口径统一为本组件。embedded 形态供卡片内部嵌入（无外层卡片壳）。

export const riskLevelStyle: Record<RiskItem['level'], { label: string; cls: string }> = {
  pass: { label: '通过', cls: 'bg-jade/10 text-jade border-jade/20' },
  warn: { label: '提示', cls: 'bg-gold/10 text-gold border-gold/30' },
  fail: { label: '不建议', cls: 'bg-red-50 text-crimson border-crimson/20' },
};

export const riskLevelSummary: Record<'pass' | 'warn' | 'fail', { label: string; cls: string }> = {
  pass: { label: '风险体检全部通过', cls: 'text-jade' },
  warn: { label: '有需要斟酌的提示项', cls: 'text-gold' },
  fail: { label: '存在不建议使用的风险项', cls: 'text-crimson' },
};

/** 汇总等级：有 fail → fail；否则有 warn → warn；否则 pass（与后端 OverallRiskLevel 同义） */
export function overallRiskLevel(items: RiskItem[]): RiskItem['level'] {
  if (items.some(i => i.level === 'fail')) return 'fail';
  if (items.some(i => i.level === 'warn')) return 'warn';
  return 'pass';
}

interface RiskListProps {
  risks?: RiskItem[];
  title?: string;
  /** 嵌入形态：省略外层卡片壳与标题，用于 NameDetail 等卡片内部 */
  embedded?: boolean;
}

export default function RiskList({ risks, title = '风险体检', embedded = false }: RiskListProps) {
  const items = risks ?? [];
  const level = overallRiskLevel(items);

  const list = (
    <ul className="space-y-2.5">
      {items.length === 0 ? (
        <li className="text-sm text-ink-light/60">未发现需要留意的风险项</li>
      ) : (
        items.map((item, idx) => (
          <li key={idx} className="flex items-start gap-3">
            <span className={`shrink-0 px-2 py-0.5 rounded-lg text-xs border ${riskLevelStyle[item.level].cls}`}>
              {item.category} · {riskLevelStyle[item.level].label}
            </span>
            <span className="text-sm text-ink-light leading-relaxed">{item.detail}</span>
          </li>
        ))
      )}
    </ul>
  );

  if (embedded) {
    return (
      <div className="space-y-2">
        <p className={`text-[11px] tracking-wider ${riskLevelSummary[level].cls}`}>
          {riskLevelSummary[level].label}
        </p>
        {list}
      </div>
    );
  }

  return (
    <div className="consultation-sheet corner-decor py-5 px-6">
      <h2 className="section-title mb-4">{title}</h2>
      <p className={`text-sm mb-4 ${riskLevelSummary[level].cls}`}>{riskLevelSummary[level].label}</p>
      {list}
    </div>
  );
}
