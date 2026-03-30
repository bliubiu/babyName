'use client';

import { useState } from 'react';
import { BaziAnalysis as BaziAnalysisType } from '@/types';

interface BaziAnalysisProps {
  bazi: BaziAnalysisType;
  nayin: string;
  zodiac: string;
}

export default function BaziAnalysis({ bazi, nayin, zodiac }: BaziAnalysisProps) {
  const [isExpanded, setIsExpanded] = useState(true);

  const toggleExpand = () => {
    setIsExpanded(!isExpanded);
  };

  const getWuxingName = (key: string): string => {
    switch (key) {
      case 'jin': return '金';
      case 'mu': return '木';
      case 'shui': return '水';
      case 'huo': return '火';
      case 'tu': return '土';
      default: return key;
    }
  };

  return (
    <div className="card mb-4 md:mb-6 lg:mb-8 animate-fadeInUp">
      <div className="flex justify-between items-start mb-2 md:mb-3 lg:mb-4">
        <h2 className="font-serif text-lg md:text-xl lg:text-2xl text-ink">📊 八字分析</h2>
        <button
          onClick={toggleExpand}
          className="text-sm text-teal hover:text-crimson transition-colors px-3 py-1 rounded-md hover:bg-crimson/10"
        >
          {isExpanded ? '收起' : '展开'}
        </button>
      </div>

      {isExpanded && (
        <div className="animate-fadeIn">
          <div className="grid md:grid-cols-2 gap-3 md:gap-4 lg:gap-6">
            <div>
              <p className="text-sm text-teal mb-1 md:mb-2">八字</p>
              <p className="font-serif text-base md:text-lg lg:text-xl text-ink">
                {bazi.bazi.year}年 {bazi.bazi.month}月 {bazi.bazi.day}日 {bazi.bazi.hour}时
              </p>
            </div>
            <div>
              <p className="text-sm text-teal mb-1 md:mb-2">日主</p>
              <p className="text-ink text-sm md:text-base">{bazi.rishou} ({bazi.rishou_wuxing}性){bazi.rishou_wuxing === '木' ? '，身强' : '，身弱'}</p>
            </div>
            <div>
              <p className="text-sm text-teal mb-1 md:mb-2">五行分布</p>
              <div className="space-y-1 md:space-y-2">
                {Object.entries(bazi.wuxing).map(([key, value]) => (
                  <div key={key} className="flex items-center gap-2">
                    <span className="w-6 md:w-8 text-xs md:text-sm">
                      {getWuxingName(key)}
                    </span>
                    <div className="flex-1 h-1.5 md:h-2 bg-warm-white rounded overflow-hidden">
                      <div
                        className="h-full bg-crimson"
                        style={{ width: `${(value / 8) * 100}%` }}
                      />
                    </div>
                    <span className="w-4 text-xs md:text-sm">{value}</span>
                  </div>
                ))}
              </div>
            </div>
            <div>
              <p className="text-sm text-teal mb-1 md:mb-2">喜用神</p>
              <p className="text-base md:text-lg text-gold">{bazi.xiyongshen.join('、')}</p>
              <p className="text-xs md:text-sm text-teal mt-1 md:mt-2">纳音：{nayin}</p>
              <p className="text-xs md:text-sm text-teal">生肖：{zodiac}</p>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
