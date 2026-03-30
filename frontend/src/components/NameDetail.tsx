'use client';

import { Name } from '@/types';

interface NameDetailProps {
  name: Name;
}

interface ScoreDetail {
  label: string;
  score: number;
  maxScore: number;
  percentage: number;
  color: string;
}

export default function NameDetail({ name }: NameDetailProps) {
  const scoreDetails: ScoreDetail[] = [
    {
      label: '八字匹配度',
      score: 28,
      maxScore: 30,
      percentage: 93,
      color: 'bg-crimson'
    },
    {
      label: '纳音五行',
      score: 18,
      maxScore: 20,
      percentage: 90,
      color: 'bg-blue-500'
    },
    {
      label: '生肖适配',
      score: 14,
      maxScore: 15,
      percentage: 93,
      color: 'bg-green-500'
    },
    {
      label: '易经卦象',
      score: 16,
      maxScore: 20,
      percentage: 80,
      color: 'bg-purple-500'
    },
    {
      label: '读音韵律',
      score: 8,
      maxScore: 10,
      percentage: 80,
      color: 'bg-amber-500'
    },
    {
      label: '寓意内涵',
      score: 5,
      maxScore: 5,
      percentage: 100,
      color: 'bg-pink-500'
    }
  ];

  return (
    <div className="card animate-fadeInUp">
      <h3 className="font-serif text-lg md:text-xl text-ink mb-2 md:mb-3 lg:mb-4">
        📖 名字详解 - {name.full_name || `${name.surname}${name.given_name}`}
      </h3>
      
      <div className="space-y-2 md:space-y-3 lg:space-y-4">
        {name.wuxing_analysis && (
          <div>
            <p className="text-sm text-teal mb-1">五行分析</p>
            <p className="text-ink text-sm md:text-base">{name.wuxing_analysis}</p>
          </div>
        )}
        {name.bazi_score_detail && (
          <div>
            <p className="text-sm text-teal mb-1">八字评分</p>
            <p className="text-ink text-sm md:text-base">{name.bazi_score_detail}</p>
          </div>
        )}
        {name.yinyun && (
          <div>
            <p className="text-sm text-teal mb-1">音韵意境</p>
            <p className="text-ink text-sm md:text-base">{name.yinyun}</p>
          </div>
        )}
        {name.poetry_source && (
          <div>
            <p className="text-sm text-teal mb-1">诗词典故</p>
            <p className="text-ink text-sm md:text-base">
              出自《{name.poetry_source}》
              {name.poetry_chapter && <>《{name.poetry_chapter}》</>}
              {name.poetry_sentence && <span className="text-teal">：{name.poetry_sentence}</span>}
            </p>
          </div>
        )}

        <div className="pt-4 border-t border-warm-white">
          <h4 className="font-serif text-base md:text-lg text-ink mb-3 md:mb-4">📊 评分明细</h4>
          <div className="space-y-3">
            {scoreDetails.map((detail, index) => (
              <div key={index}>
                <div className="flex justify-between items-center mb-1">
                  <span className="text-sm text-teal">{detail.label}</span>
                  <span className="text-sm text-ink">
                    {detail.score}/{detail.maxScore} ({detail.percentage}%)
                  </span>
                </div>
                <div className="w-full h-2 bg-warm-white rounded-full overflow-hidden">
                  <div
                    className={`h-full ${detail.color} rounded-full transition-all duration-500 ease-out`}
                    style={{ width: `${detail.percentage}%` }}
                  />
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="pt-4 border-t border-warm-white">
          <div className="flex items-center justify-between">
            <span className="text-sm text-teal">综合评分</span>
            <div className="flex items-center gap-2">
              <span className="text-2xl md:text-3xl font-bold text-gold">{(typeof name.score === 'number' ? name.score : parseFloat(name.score)).toFixed(2)}</span>
              <span className="text-sm text-teal">分</span>
            </div>
          </div>
          <div className="mt-2">
            <div className="w-full h-3 bg-warm-white rounded-full overflow-hidden">
              <div
                className="h-full bg-gradient-to-r from-crimson to-gold rounded-full transition-all duration-500 ease-out"
                style={{ width: `${Math.min(100, typeof name.score === 'number' ? name.score : parseFloat(name.score))}%` }}
              />
            </div>
            <div className="flex justify-between mt-1 text-xs text-teal">
              <span>0分</span>
              <span>100分</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
