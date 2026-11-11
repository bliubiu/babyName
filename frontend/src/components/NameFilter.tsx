'use client';

import { useState } from 'react';

interface NameFilterProps {
  /** 受控：筛选条件由结果页统一持有，快捷 chips、空状态清除与高级面板共享一份真源 */
  filters: NameFilters;
  onFilterChange: (filters: NameFilters) => void;
  totalNames: number;
  filteredCount: number;
}

export interface NameFilters {
  gender?: 'male' | 'female';
  wuxing?: string;
  minStrokes?: number;
  maxStrokes?: number;
  minFrequencyTier?: number;
  maxFrequencyTier?: number;
}

const wuxingOptions = [
  { value: '金', label: '金', color: 'text-ink-light' },
  { value: '木', label: '木', color: 'text-green-600' },
  { value: '水', label: '水', color: 'text-blue-500' },
  { value: '火', label: '火', color: 'text-red-500' },
  { value: '土', label: '土', color: 'text-amber-600' },
];

export default function NameFilter({ filters, onFilterChange, totalNames, filteredCount }: NameFilterProps) {
  const [isExpanded, setIsExpanded] = useState(false);

  const handleGenderChange = (gender: 'male' | 'female' | undefined) => {
    onFilterChange({ ...filters, gender });
  };

  const handleWuxingChange = (wuxing: string | undefined) => {
    onFilterChange({ ...filters, wuxing });
  };

  const handleStrokeChange = (min?: number, max?: number) => {
    // min ≤ max 防护：用户先填大值再调另一端时避免产生恒空的矛盾区间
    // （此前 min=20、max=5 可同时存在，筛选结果恒空且无提示）
    let safeMin = min;
    let safeMax = max;
    if (safeMin !== undefined && safeMax !== undefined && safeMin > safeMax) {
      if (min !== filters.minStrokes) {
        safeMax = safeMin; // 改动的是 min → 抬高 max
      } else {
        safeMin = safeMax; // 改动的是 max → 压低 min
      }
    }
    onFilterChange({ ...filters, minStrokes: safeMin, maxStrokes: safeMax });
  };

  const handleFrequencyTierChange = (min?: number, max?: number) => {
    onFilterChange({ ...filters, minFrequencyTier: min, maxFrequencyTier: max });
  };

  const resetFilters = () => {
    onFilterChange({});
  };

  const hasActiveFilters = filters.gender || filters.wuxing || filters.minStrokes || filters.maxStrokes || filters.minFrequencyTier || filters.maxFrequencyTier;

  // 高级筛选不再是居中窄卡片：与上方筛选工具带同宽，收起时只占一行，
  // 避免结果页中部出现「一条窄卡 + 两侧大片空白」（设计稿对齐）。
  return (
    <div className="mb-2 rounded-xl border border-paper/60 bg-warm-white/60 px-3 py-1.5">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium text-ink">
          更多筛选
          {hasActiveFilters && (
            <span className="ml-2 text-sm text-jade">
              ({filteredCount}/{totalNames})
            </span>
          )}
        </h3>
        <button
          onClick={() => setIsExpanded(!isExpanded)}
          className="min-h-[44px] px-2 text-jade hover:text-crimson transition-colors text-sm"
          aria-label={isExpanded ? '收起筛选面板' : '展开筛选面板'}
          aria-expanded={isExpanded}
        >
          {isExpanded ? '收起' : '展开'}
        </button>
      </div>

      {isExpanded && (
        <div className="mt-3 grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <p className="block text-sm text-jade mb-2">性别</p>
            <div className="flex gap-3">
              <button
                onClick={() => handleGenderChange(filters.gender === 'male' ? undefined : 'male')}
                className={`min-h-[44px] px-4 py-2 rounded-lg transition-colors ${
                  filters.gender === 'male'
                    ? 'bg-blue-500 text-white'
                    : 'bg-warm-white text-ink hover:bg-warm-white/80'
                }`}
                aria-label={`筛选男性名字${filters.gender === 'male' ? '（已选中）' : ''}`}
                aria-pressed={filters.gender === 'male'}
              >
                男
              </button>
              <button
                onClick={() => handleGenderChange(filters.gender === 'female' ? undefined : 'female')}
                className={`min-h-[44px] px-4 py-2 rounded-lg transition-colors ${
                  filters.gender === 'female'
                    ? 'bg-pink-500 text-white'
                    : 'bg-warm-white text-ink hover:bg-warm-white/80'
                }`}
                aria-label={`筛选女性名字${filters.gender === 'female' ? '（已选中）' : ''}`}
                aria-pressed={filters.gender === 'female'}
              >
                女
              </button>
            </div>
          </div>

          <div>
            <p className="block text-sm text-jade mb-2">五行</p>
            <div className="flex flex-wrap gap-2">
              {wuxingOptions.map((option) => (
                <button
                  key={option.value}
                  onClick={() => handleWuxingChange(filters.wuxing === option.value ? undefined : option.value)}
                  className={`min-h-[44px] px-3 py-1.5 rounded-lg transition-colors ${
                    filters.wuxing === option.value
                      ? 'bg-crimson text-white'
                      : 'bg-warm-white text-ink hover:bg-warm-white/80'
                  }`}
                  aria-label={`筛选${option.label}五行名字${filters.wuxing === option.value ? '（已选中）' : ''}`}
                  aria-pressed={filters.wuxing === option.value}
                >
                  {option.label}
                </button>
              ))}
            </div>
          </div>

          <div>
            <p className="block text-sm text-jade mb-2">笔画数</p>
            <div className="flex items-center gap-3">
              <input
                type="number"
                min="1"
                max="30"
                value={filters.minStrokes || ''}
                onChange={(e) => {
                  const value = e.target.value ? parseInt(e.target.value) : undefined;
                  handleStrokeChange(value, filters.maxStrokes);
                }}
                placeholder="最小"
                aria-label="最小笔画数"
                className="w-20 min-h-[44px] px-3 py-2 border border-warm-white rounded-lg focus:outline-none focus:ring-2 focus:ring-crimson text-ink bg-warm-white"
              />
              <span className="text-jade" aria-hidden="true">-</span>
              <input
                type="number"
                min="1"
                max="30"
                value={filters.maxStrokes || ''}
                onChange={(e) => {
                  const value = e.target.value ? parseInt(e.target.value) : undefined;
                  handleStrokeChange(filters.minStrokes, value);
                }}
                placeholder="最大"
                aria-label="最大笔画数"
                className="w-20 min-h-[44px] px-3 py-2 border border-warm-white rounded-lg focus:outline-none focus:ring-2 focus:ring-crimson text-ink bg-warm-white"
              />
            </div>
          </div>

          <div>
            <p className="block text-sm text-jade mb-2">人名频率</p>
            <p className="text-xs text-jade/80 mb-2">基于120万人名语料统计，筛选常用/罕见人名用字</p>
            <div className="flex flex-wrap gap-2">
              {[
                { value: 5, label: '极高频(Top1%)' },
                { value: 4, label: '高频(Top5%)' },
                { value: 3, label: '中频(Top20%)' },
                { value: 2, label: '低频(Top50%)' },
                { value: 1, label: '极低频' },
              ].map((option) => (
                <button
                  key={option.value}
                  onClick={() => {
                    if (filters.minFrequencyTier === option.value) {
                      // 取消选择
                      handleFrequencyTierChange(undefined, undefined);
                    } else {
                      // 选择该档位及以上
                      handleFrequencyTierChange(option.value, undefined);
                    }
                  }}
                  className={`min-h-[44px] px-3 py-1.5 rounded-lg transition-colors ${
                    filters.minFrequencyTier === option.value
                      ? 'bg-indigo-500 text-white'
                      : 'bg-warm-white text-ink hover:bg-warm-white/80'
                  }`}
                  aria-label={`筛选人名频率${option.label}${filters.minFrequencyTier === option.value ? '（已选中）' : ''}`}
                  aria-pressed={filters.minFrequencyTier === option.value}
                >
                  {option.label}
                </button>
              ))}
            </div>
            {(filters.minFrequencyTier || filters.maxFrequencyTier) && (
              <p className="text-xs text-jade mt-2">
                当前筛选：人名频率 ≥ {filters.minFrequencyTier || 1} 档
              </p>
            )}
          </div>

          {hasActiveFilters && (
            <div className="pt-2 border-t border-warm-white">
              <button
                onClick={resetFilters}
                className="min-h-[44px] text-sm text-jade hover:text-crimson transition-colors"
                aria-label="清除所有筛选条件"
              >
                清除所有筛选
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
