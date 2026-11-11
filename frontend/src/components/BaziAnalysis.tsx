'use client';

import { useState } from 'react';
import { BaziAnalysis as BaziAnalysisType } from '@/types';

interface BaziAnalysisProps {
  bazi: BaziAnalysisType;
  nayin: string;
  zodiac: string;
  /** 嵌入模式：去除外层卡片与自带折叠按钮，由统一折叠容器控制（结果页展示优化） */
  embedded?: boolean;
}

const wuxingColors: Record<string, { bg: string; text: string; bar: string }> = {
  // 背景/文字改用随主题自适应的中性 token，五行条用语义色（bg-wuxing-*），暗色下不露白
  jin: { bg: 'bg-paper/40', text: 'text-ink-light', bar: 'bg-wuxing-jin' },
  mu: { bg: 'bg-paper/40', text: 'text-ink-light', bar: 'bg-wuxing-mu' },
  shui: { bg: 'bg-paper/40', text: 'text-ink-light', bar: 'bg-wuxing-shui' },
  huo: { bg: 'bg-paper/40', text: 'text-ink-light', bar: 'bg-wuxing-huo' },
  tu: { bg: 'bg-paper/40', text: 'text-ink-light', bar: 'bg-wuxing-tu' },
};

export default function BaziAnalysis({ bazi, nayin, zodiac, embedded = false }: BaziAnalysisProps) {
  const [expanded, setExpanded] = useState(true);
  // 嵌入模式下展开态由外层折叠容器决定，本组件始终输出完整内容
  const isExpanded = embedded || expanded;

  const getWuxingName = (key: string): string => {
    const map: Record<string, string> = { jin: '金', mu: '木', shui: '水', huo: '火', tu: '土' };
    return map[key] || key;
  };

  const maxWuxing = Math.max(...Object.values(bazi.wuxing), 1);

  return (
    <div className={embedded ? 'animate-fade-in-up' : 'card mb-4 md:mb-6 animate-fade-in-up'}>
      {!embedded && (
        <div className="flex justify-between items-center mb-4">
          <h2 className="section-title">八字分析</h2>
          <button
            onClick={() => setExpanded(!expanded)}
            className="text-sm text-jade hover:text-crimson transition-colors px-3 py-1.5 rounded-lg hover:bg-crimson/5"
          >
            {isExpanded ? '收起' : '展开'}
          </button>
        </div>
      )}

      {isExpanded && (
        <div className="animate-fade-in space-y-5">
          {/* 八字信息 */}
          <div className="flex flex-wrap gap-3">
            <div className="flex-1 min-w-[140px] p-3 bg-warm-white/60 rounded-xl">
              <p className="text-xs text-jade mb-1">八字</p>
              <p className="font-serif text-base md:text-lg text-ink">
                {bazi.bazi.year}年 {bazi.bazi.month}月 {bazi.bazi.day}日 {bazi.bazi.hour}时
              </p>
            </div>
            <div className="flex-1 min-w-[140px] p-3 bg-warm-white/60 rounded-xl">
              <p className="text-xs text-jade mb-1">日主</p>
              <p className="text-ink text-sm md:text-base">{bazi.rishou} <span className="text-jade text-xs">({bazi.rishou_wuxing}性)</span></p>
            </div>
          </div>

          {/* 五行分布 */}
          <div>
            <p className="text-xs text-jade mb-3">五行分布</p>
            <div className="space-y-2.5">
              {Object.entries(bazi.wuxing).map(([key, value]) => {
                const colors = wuxingColors[key] || wuxingColors.tu;
                return (
                  <div key={key} className="flex items-center gap-3">
                    <span className={`w-6 h-6 flex items-center justify-center rounded-md text-xs font-medium ${colors.bg} ${colors.text}`}>
                      {getWuxingName(key)}
                    </span>
                    <div className="flex-1 h-2 bg-paper/30 rounded-full overflow-hidden">
                      <div
                        className={`h-full rounded-full transition-all duration-700 ${colors.bar}`}
                        style={{ width: `${(value / maxWuxing) * 100}%` }}
                      />
                    </div>
                    <span className="w-5 text-xs text-ink-light text-right">{value}</span>
                  </div>
                );
              })}
            </div>
          </div>

          {/* 喜用神与纳音 */}
          <div className="flex flex-wrap gap-3">
            <div className="flex-1 min-w-[140px] p-3 bg-gold/5 rounded-xl border border-gold/10">
              <p className="text-xs text-jade mb-1">喜用神</p>
              <p className="text-base md:text-lg text-gold font-medium">{bazi.xiyongshen.join('、')}</p>
            </div>
            <div className="flex-1 min-w-[140px] p-3 bg-warm-white/60 rounded-xl">
              <p className="text-xs text-jade mb-1">年命纳音</p>
              <p className="text-sm text-ink">{nayin || bazi.nayin || '—'}</p>
              <p className="text-xs text-jade mt-1">生肖：{zodiac}</p>
            </div>
          </div>

          {/* 四柱纳音 */}
          {bazi.four_nayin && (
            <div className="p-3 bg-warm-white/50 rounded-xl">
              <p className="text-xs text-jade mb-2">四柱纳音</p>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-2">
                {(
                  [
                    ['年柱', bazi.four_nayin.year],
                    ['月柱', bazi.four_nayin.month],
                    ['日柱', bazi.four_nayin.day],
                    ['时柱', bazi.four_nayin.hour],
                  ] as const
                ).map(([label, item]) => (
                  <div key={label} className="text-center p-2 bg-paper/40 rounded-lg">
                    <p className="text-xs text-jade">{label}</p>
                    <p className="text-sm text-ink font-medium">{item.ganzhi}</p>
                    <p className="text-xs text-ink-light mt-0.5">{item.nayin || '—'}</p>
                    {item.nayin_wuxing && (
                      <p className="text-[10px] text-jade">（{item.nayin_wuxing}）</p>
                    )}
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
