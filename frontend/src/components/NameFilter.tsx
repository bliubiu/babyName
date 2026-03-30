'use client';

import { useState } from 'react';

interface NameFilterProps {
  onFilterChange: (filters: NameFilters) => void;
  totalNames: number;
  filteredCount: number;
}

export interface NameFilters {
  gender?: 'male' | 'female';
  wuxing?: string;
  minStrokes?: number;
  maxStrokes?: number;
}

const wuxingOptions = [
  { value: '金', label: '金', color: 'text-gray-500' },
  { value: '木', label: '木', color: 'text-green-600' },
  { value: '水', label: '水', color: 'text-blue-500' },
  { value: '火', label: '火', color: 'text-red-500' },
  { value: '土', label: '土', color: 'text-amber-600' },
];

export default function NameFilter({ onFilterChange, totalNames, filteredCount }: NameFilterProps) {
  const [filters, setFilters] = useState<NameFilters>({});
  const [isExpanded, setIsExpanded] = useState(false);

  const handleGenderChange = (gender: 'male' | 'female' | undefined) => {
    const newFilters = { ...filters, gender };
    setFilters(newFilters);
    onFilterChange(newFilters);
  };

  const handleWuxingChange = (wuxing: string | undefined) => {
    const newFilters = { ...filters, wuxing };
    setFilters(newFilters);
    onFilterChange(newFilters);
  };

  const handleStrokeChange = (min?: number, max?: number) => {
    const newFilters = { ...filters, minStrokes: min, maxStrokes: max };
    setFilters(newFilters);
    onFilterChange(newFilters);
  };

  const resetFilters = () => {
    const newFilters: NameFilters = {};
    setFilters(newFilters);
    onFilterChange(newFilters);
  };

  const hasActiveFilters = filters.gender || filters.wuxing || filters.minStrokes || filters.maxStrokes;

  return (
    <div className="card mb-4 md:mb-6">
      <div className="flex items-center justify-between">
        <h3 className="font-serif text-lg md:text-xl text-ink">
          🔍 名字筛选
          {hasActiveFilters && (
            <span className="ml-2 text-sm text-teal">
              ({filteredCount}/{totalNames})
            </span>
          )}
        </h3>
        <button
          onClick={() => setIsExpanded(!isExpanded)}
          className="text-teal hover:text-crimson transition-colors text-sm"
        >
          {isExpanded ? '收起' : '展开'}
        </button>
      </div>

      {isExpanded && (
        <div className="mt-4 space-y-4">
          <div>
            <label className="block text-sm text-teal mb-2">性别</label>
            <div className="flex gap-3">
              <button
                onClick={() => handleGenderChange(filters.gender === 'male' ? undefined : 'male')}
                className={`px-4 py-2 rounded-lg transition-colors ${
                  filters.gender === 'male'
                    ? 'bg-blue-500 text-white'
                    : 'bg-warm-white text-ink hover:bg-warm-white/80'
                }`}
              >
                男
              </button>
              <button
                onClick={() => handleGenderChange(filters.gender === 'female' ? undefined : 'female')}
                className={`px-4 py-2 rounded-lg transition-colors ${
                  filters.gender === 'female'
                    ? 'bg-pink-500 text-white'
                    : 'bg-warm-white text-ink hover:bg-warm-white/80'
                }`}
              >
                女
              </button>
            </div>
          </div>

          <div>
            <label className="block text-sm text-teal mb-2">五行</label>
            <div className="flex flex-wrap gap-2">
              {wuxingOptions.map((option) => (
                <button
                  key={option.value}
                  onClick={() => handleWuxingChange(filters.wuxing === option.value ? undefined : option.value)}
                  className={`px-3 py-1.5 rounded-lg transition-colors ${
                    filters.wuxing === option.value
                      ? 'bg-crimson text-white'
                      : 'bg-warm-white text-ink hover:bg-warm-white/80'
                  }`}
                >
                  {option.label}
                </button>
              ))}
            </div>
          </div>

          <div>
            <label className="block text-sm text-teal mb-2">笔画数</label>
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
                className="w-20 px-3 py-2 border border-warm-white rounded-lg focus:outline-none focus:ring-2 focus:ring-crimson text-ink bg-white"
              />
              <span className="text-teal">-</span>
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
                className="w-20 px-3 py-2 border border-warm-white rounded-lg focus:outline-none focus:ring-2 focus:ring-crimson text-ink bg-white"
              />
            </div>
          </div>

          {hasActiveFilters && (
            <div className="pt-2 border-t border-warm-white">
              <button
                onClick={resetFilters}
                className="text-sm text-teal hover:text-crimson transition-colors"
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
