'use client';

import { useState } from 'react';
import { BaziAnalysis as BaziAnalysisType } from '@/types';

interface BaziAnalysisProps {
  bazi: BaziAnalysisType;
  nayin: string;
  zodiac: string;
}

const wuxingColors: Record<string, { bg: string; text: string; bar: string }> = {
  jin: { bg: 'bg-gray-100', text: 'text-gray-600', bar: 'bg-gray-400' },
  mu: { bg: 'bg-green-50', text: 'text-green-700', bar: 'bg-green-500' },
  shui: { bg: 'bg-blue-50', text: 'text-blue-700', bar: 'bg-blue-500' },
  huo: { bg: 'bg-red-50', text: 'text-red-700', bar: 'bg-red-500' },
  tu: { bg: 'bg-amber-50', text: 'text-amber-700', bar: 'bg-amber-500' },
};

export default function BaziAnalysis({ bazi, nayin, zodiac }: BaziAnalysisProps) {
  const [isExpanded, setIsExpanded] = useState(true);

  const getWuxingName = (key: string): string => {
    const map: Record<string, string> = { jin: '金', mu: '木', shui: '水', huo: '火', tu: '土' };
    return map[key] || key;
  };

  const maxWuxing = Math.max(...Object.values(bazi.wuxing), 1);

  return (
    <div className="card mb-4 md:mb-6 animate-fade-in-up">
      <div className="flex justify-between items-center mb-4">
        <h2 className="section-title">八字分析</h2>
        <button
          onClick={() => setIsExpanded(!isExpanded)}
          className="text-sm text-jade hover:text-crimson transition-colors px-3 py-1.5 rounded-lg hover:bg-crimson/5"
        >
          {isExpanded ? '收起' : '展开'}
        </button>
      </div>

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
              <p className="text-xs text-jade mb-1">纳音</p>
              <p className="text-sm text-ink">{nayin}</p>
              <p className="text-xs text-jade mt-1">生肖：{zodiac}</p>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
