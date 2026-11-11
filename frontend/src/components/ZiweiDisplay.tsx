'use client';

import type { ZiweiChart } from '@/types';

interface ZiweiDisplayProps {
  ziwei: ZiweiChart | null;
  /** 嵌入模式：去除外层卡片与标题，由统一折叠容器提供（结果页展示优化） */
  embedded?: boolean;
}

// 十二宫固定顺序（与后端 Gongs 一致，自命宫逆行）
const PALACES = [
  '命宫', '兄弟宫', '夫妻宫', '子女宫', '财帛宫', '疾厄宫',
  '迁移宫', '交友宫', '官禄宫', '田宅宫', '福德宫', '父母宫',
];

// 亮度 → 配色（庙旺得利为吉，平不陷为平/凶）
const BRIGHTNESS_STYLE: Record<string, string> = {
  庙: 'bg-crimson/15 text-crimson',
  旺: 'bg-amber-500/15 text-amber-600',
  得: 'bg-emerald-500/15 text-emerald-600',
  利: 'bg-teal-500/15 text-teal-600',
  平: 'bg-sky-500/15 text-sky-600',
  // 凶格（不/陷）改用中性自适应色，暗色下不露白
  不: 'bg-ink-light/10 text-ink-light/70',
  陷: 'bg-ink-light/15 text-ink-light/80',
};

function brightnessTag(b: string | undefined) {
  if (!b) return null;
  const cls = BRIGHTNESS_STYLE[b] ?? 'bg-ink-light/10 text-ink-light/70';
  return (
    <span className={`ml-1 px-1.5 py-0.5 rounded text-xs leading-none ${cls}`}>{b}</span>
  );
}

export default function ZiweiDisplay({ ziwei, embedded = false }: ZiweiDisplayProps) {
  if (!ziwei) {
    return (
      <div className="text-center py-6 text-ink-light/70 text-sm">
        暂无紫微斗数排盘数据
      </div>
    );
  }

  // 命宫/身宫地支提示
  const mingGong = ziwei.ming_gong ?? '';
  const shenGong = ziwei.shen_gong ?? '';

  // 四化
  const siHua = ziwei.si_hua ?? {};
  const siHuaItems = (['禄', '权', '科', '忌'] as const)
    .map((k) => ({ k, v: siHua[k] }))
    .filter((x) => x.v);

  // 大限
  const daXian = ziwei.da_xian ?? {};

  // 亮度（按主星名索引）
  const brightness = ziwei.brightness ?? {};

  return (
    <div className={embedded ? 'animate-fade-in-up' : 'card mb-4 md:mb-6 animate-fade-in-up'}>
      {!embedded && (
        <div className="flex items-center gap-3 mb-4">
          <h3 className="section-title">紫微斗数</h3>
        </div>
      )}

      <div className="space-y-4">
        {/* 概要 */}
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-2 text-sm">
          <SummaryItem label="五行局" value={ziwei.five_elements} />
          <SummaryItem label="命主" value={ziwei.soul} />
          <SummaryItem label="身主" value={ziwei.body} />
          <SummaryItem label="命宫/身宫" value={`${mingGong} / ${shenGong}`} />
        </div>

        {/* 四柱 */}
        <div className="p-3 bg-warm-white/60 rounded-xl">
          <span className="text-xs text-jade font-medium">四柱</span>
          <p className="text-ink text-sm mt-1.5 tracking-wide">
            {ziwei.nian_zhu}（年） · {ziwei.yue_zhu}（月） · {ziwei.ri_zhu}（日） · {ziwei.shi_zhu}（时）
          </p>
        </div>

        {/* 生年四化 */}
        {siHuaItems.length > 0 && (
          <div className="p-3 bg-warm-white/60 rounded-xl">
            <span className="text-xs text-jade font-medium">生年四化</span>
            <div className="flex flex-wrap gap-2 mt-1.5">
              {siHuaItems.map(({ k, v }) => (
                <span
                  key={k}
                  className="px-2 py-1 rounded-lg bg-crimson/8 text-crimson text-xs font-medium"
                >
                  {k} → {v}
                </span>
              ))}
            </div>
          </div>
        )}

        {/* 十二宫落星 */}
        <div>
          <span className="text-xs text-jade font-medium">十二宫落星</span>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 mt-2">
            {PALACES.map((gong) => {
              const zhu = ziwei.zhu_xing?.[gong] ?? [];
              const fu = ziwei.fu_xing?.[gong] ?? [];
              const sui = ziwei.sui_xing?.[gong] ?? [];
              const hasContent = zhu.length > 0 || fu.length > 0 || sui.length > 0;
              return (
                <div
                  key={gong}
                  className="p-3 bg-warm-white/50 rounded-xl border border-crimson/5"
                >
                  <div className="text-xs text-jade/80 mb-1.5">{gong}</div>
                  {hasContent ? (
                    <div className="space-y-1">
                      {zhu.length > 0 && (
                        <div className="text-ink text-sm font-medium flex flex-wrap items-center">
                          {zhu.map((s) => (
                            <span key={s} className="inline-flex items-center">
                              {s}
                              {brightnessTag(brightness[s])}
                            </span>
                          ))}
                        </div>
                      )}
                      {fu.length > 0 && (
                        <div className="text-ink-light/70 text-xs">
                          辅：{fu.join('、')}
                        </div>
                      )}
                      {sui.length > 0 && (
                        <div className="text-ink-light/70 text-xs">
                          曜：{sui.join('、')}
                        </div>
                      )}
                    </div>
                  ) : (
                    <div className="text-ink-light/30 text-xs">—</div>
                  )}
                </div>
              );
            })}
          </div>
        </div>

        {/* 大限 */}
        {Object.keys(daXian).length > 0 && (
          <div>
            <span className="text-xs text-jade font-medium">大限</span>
            <div className="flex flex-wrap gap-2 mt-2">
              {PALACES.filter((g) => daXian[g]).map((g) => {
                const d = daXian[g];
                return (
                  <span
                    key={g}
                    className="px-2 py-1 rounded-lg bg-jade/8 text-jade text-xs"
                  >
                    {g} {d.range[0]}-{d.range[1]}岁（{d.heavenly_stem}{d.earthly_branch}）
                  </span>
                );
              })}
            </div>
          </div>
        )}

        {/* 分析文本 */}
        {ziwei.analysis && (
          <div className="p-3 bg-crimson/3 rounded-xl border border-crimson/8">
            <span className="text-xs text-crimson/70 font-medium">排盘分析</span>
            <p className="text-ink text-sm mt-1.5 leading-relaxed whitespace-pre-line">
              {ziwei.analysis}
            </p>
          </div>
        )}
      </div>
    </div>
  );
}

function SummaryItem({ label, value }: { label: string; value?: string }) {
  return (
    <div className="p-2.5 bg-warm-white/60 rounded-xl">
      <span className="text-xs text-jade/70 block">{label}</span>
      <p className="text-ink text-sm mt-0.5 font-medium">{value || '—'}</p>
    </div>
  );
}
