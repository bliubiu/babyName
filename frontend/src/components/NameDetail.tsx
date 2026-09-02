'use client';

import { useState } from 'react';
import { Name } from '@/types';
import { IconChevronDown, IconChevronUp } from './Icons';

interface NameDetailProps {
  name: Name;
}

interface DimConfig {
  key: string;
  label: string;
  color: string;
}

export default function NameDetail({ name }: NameDetailProps) {
  const totalScore = Math.min(100, name.total_score ?? name.score ?? 0);
  const [showPoetryDetail, setShowPoetryDetail] = useState(false);

  // 优先使用后端透传的 score_detail（含依据文字），回退到旧字段
  const scoreDetails = name.score_detail?.length ? name.score_detail : [
    { name: '五行匹配', score: name.wuxing_score ?? 0, detail: name.wuxing_analysis ?? '' },
    { name: '音韵律动', score: name.yinyun_score ?? 0, detail: name.yinyun ?? '' },
    { name: '字义内涵', score: name.meaning_score ?? 0, detail: name.meaning_detail ?? name.wuxing_analysis ?? '' },
    { name: '天地人三才', score: name.sancai_score ?? 0, detail: name.sancai_analysis ?? '' },
    { name: '生肖适配', score: name.zodiac_score ?? 0, detail: name.bazi_score_detail ?? '' },
    { name: '新颖度', score: name.novelty_score ?? 0, detail: '' },
    { name: '诗词共现', score: name.bigram_score ?? 0, detail: '' },
    { name: '人名频率', score: name.frequency_score ?? 0, detail: '' },
  ].filter(d => d.score > 0 || d.detail);

  const colorMap: Record<string, string> = {
    '五行匹配': 'bg-amber-500',
    '音韵律动': 'bg-sky-500',
    '字义内涵': 'bg-emerald-500',
    '天地人三才': 'bg-violet-400',
    '生肖适配': 'bg-rose-400',
    '新颖度': 'bg-pink-400',
    '诗词共现': 'bg-cyan-400',
    '人名频率': 'bg-indigo-400',
  };

  return (
    <div className="animate-fade-in-up space-y-4">
      <h4 className="font-serif text-base text-ink">
        名字详解 - {name.full_name || `${name.surname}${name.given_name}`}
      </h4>

      <div className="space-y-3">
        {name.wuxing_analysis && (
          <div>
            <p className="text-xs text-jade mb-1">五行分析</p>
            <p className="text-ink text-sm leading-relaxed">{name.wuxing_analysis}</p>
          </div>
        )}
        {name.bazi_score_detail && (
          <div>
            <p className="text-xs text-jade mb-1">八字评分</p>
            <p className="text-ink text-sm leading-relaxed">{name.bazi_score_detail}</p>
          </div>
        )}
        {name.yinyun && (
          <div>
            <p className="text-xs text-jade mb-1">音韵意境</p>
            <p className="text-ink text-sm leading-relaxed">{name.yinyun}</p>
          </div>
        )}
        {name.poetry_source && (
          <div>
            <p className="text-xs text-jade mb-1">诗词典故</p>
            <div className="space-y-1">
              <p className="text-ink text-sm leading-relaxed cursor-pointer" onClick={() => setShowPoetryDetail(!showPoetryDetail)}>
                出自《{name.poetry_source}》
                {name.poetry_chapter && <>《{name.poetry_chapter}》</>}
                {name.poetry_sentence && <span className="text-jade">：{name.poetry_sentence}</span>}
                <span className="ml-1 text-crimson/70 text-xs">
                  {showPoetryDetail ? <IconChevronUp size={12} /> : <IconChevronDown size={12} />}
                </span>
              </p>
              {showPoetryDetail && name.poetry_full_text && (
                <div className="ml-4 mt-2 p-3 bg-paper/30 rounded-lg border border-paper/40 text-[11px] leading-relaxed">
                  <p className="font-medium text-jade mb-1">
                    {name.poetry_author && name.poetry_author !== '佚名' ? (
                      <>{name.poetry_author}｜</>
                    ) : null}
                    {name.poetry_dynasty ? <>{name.poetry_dynasty}｜</> : null}
                    {name.poetry_source}《{name.poetry_chapter}》
                  </p>
                  <p className="text-ink/80 whitespace-pre-wrap">
                    {name.poetry_full_text.split('｜').join('\n')}
                  </p>
                </div>
              )}
            </div>
          </div>
        )}
      </div>

      {/* 综合评分 */}
      <div className="pt-3 border-t border-paper/30">
        <div className="flex items-center justify-between mb-3">
          <span className="text-xs text-jade">综合评分</span>
          <div className="flex items-baseline gap-1">
            <span className="text-2xl font-bold text-gold">{totalScore.toFixed(1)}</span>
            <span className="text-xs text-jade">分</span>
          </div>
        </div>
        <div className="score-bar mb-4">
          <div
            className="score-bar-fill"
            style={{ width: `${totalScore}%` }}
          />
        </div>

        {/* 各维度评分分解（含依据文字） */}
        {scoreDetails.length > 0 && (
          <div className="space-y-2.5">
            <p className="text-[11px] text-jade/60 tracking-wider">评分分解</p>
            {scoreDetails.map(d => {
              const color = colorMap[d.name] || 'bg-gray-400';
              return (
                <div key={d.name} className="space-y-1">
                  <div className="flex items-center gap-2">
                    <span className="text-xs text-ink-light/70 w-16 flex-shrink-0">{d.name}</span>
                    <div className="flex-1 h-2 bg-paper/40 rounded-full overflow-hidden">
                      <div
                        className={`h-full rounded-full ${color} transition-all duration-700`}
                        style={{ width: `${Math.min(100, d.score)}%` }}
                      />
                    </div>
                    <span className="text-xs text-ink-light/50 w-8 text-right font-medium">{d.score.toFixed(0)}</span>
                  </div>
                  {d.detail && (
                    <p className="text-[10px] text-jade/60 ml-16 leading-relaxed">
                      {d.detail}
                    </p>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
