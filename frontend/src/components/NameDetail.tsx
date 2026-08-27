'use client';

import { Name } from '@/types';

interface NameDetailProps {
  name: Name;
}

interface DimConfig {
  key: string;
  label: string;
  color: string;
  getValue: (n: Name) => number;
}

const dims: DimConfig[] = [
  { key: 'wuxing', label: '五行匹配', color: 'bg-amber-500', getValue: n => n.wuxing_score ?? 0 },
  { key: 'yinyun', label: '音韵律动', color: 'bg-sky-500', getValue: n => n.yinyun_score ?? 0 },
  { key: 'meaning', label: '字义内涵', color: 'bg-emerald-500', getValue: n => n.meaning_score ?? 0 },
  { key: 'sancai', label: '天地人三才', color: 'bg-violet-400', getValue: n => n.sancai_score ?? 0 },
  { key: 'zodiac', label: '生肖适配', color: 'bg-rose-400', getValue: n => n.zodiac_score ?? 0 },
  { key: 'frequency', label: '人名频率', color: 'bg-indigo-400', getValue: n => n.frequency_score ?? 0 },
];

export default function NameDetail({ name }: NameDetailProps) {
  const totalScore = Math.min(100, name.total_score ?? name.score ?? 0);
  const hasDimScores = dims.some(d => d.getValue(name) > 0);

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
            <p className="text-ink text-sm leading-relaxed">
              出自《{name.poetry_source}》
              {name.poetry_chapter && <>《{name.poetry_chapter}》</>}
              {name.poetry_sentence && <span className="text-jade">：{name.poetry_sentence}</span>}
            </p>
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

        {/* 各维度评分分解 */}
        {hasDimScores && (
          <div className="space-y-2.5">
            <p className="text-[11px] text-jade/60 tracking-wider">评分分解</p>
            {dims.map(d => {
              const val = d.getValue(name);
              return (
                <div key={d.key} className="flex items-center gap-2">
                  <span className="text-xs text-ink-light/70 w-14 flex-shrink-0">{d.label}</span>
                  <div className="flex-1 h-2 bg-paper/40 rounded-full overflow-hidden">
                    <div
                      className={`h-full rounded-full ${d.color} transition-all duration-700`}
                      style={{ width: `${Math.min(100, val)}%` }}
                    />
                  </div>
                  <span className="text-xs text-ink-light/50 w-8 text-right font-medium">{val.toFixed(0)}</span>
                </div>
              );
            })}
            {/* 天地人三才分析详情 */}
            {name.sancai_analysis && (
              <p className="text-[10px] text-jade/50 leading-relaxed mt-1">
                天地人三才：{name.sancai_analysis}
              </p>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
