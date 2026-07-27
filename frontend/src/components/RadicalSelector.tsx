'use client';

import { useState, useEffect, useCallback } from 'react';
import type { StandardCharGroup } from '@/types';
import { getCharGroups } from '@/lib/api';

interface RadicalSelectorProps {
  onSelectChars: (chars: string[]) => void;
  selectedChars: string[];
  disabled?: boolean;
}

export function RadicalSelector({ onSelectChars, selectedChars, disabled }: RadicalSelectorProps) {
  const [groups, setGroups] = useState<StandardCharGroup[]>([]);
  const [activeRadical, setActiveRadical] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let mounted = true;
    setLoading(true);
    getCharGroups()
      .then((data) => {
        if (mounted) {
          setGroups(data);
          setLoading(false);
        }
      })
      .catch((err) => {
        if (mounted) {
          setError(err instanceof Error ? err.message : '加载偏旁数据失败');
          setLoading(false);
        }
      });
    return () => { mounted = false; };
  }, []);

  const activeGroup = groups.find((g) => g.radical === activeRadical);

  const toggleChar = useCallback(
    (char: string) => {
      if (disabled) return;
      const isSelected = selectedChars.includes(char);
      const newSelected = isSelected
        ? selectedChars.filter((c) => c !== char)
        : [...selectedChars, char];
      onSelectChars(newSelected);
    },
    [selectedChars, onSelectChars, disabled]
  );

  if (loading) {
    return (
      <div className="card p-4">
        <div className="animate-pulse space-y-3">
          <div className="h-4 bg-gray-200 rounded w-1/3" />
          <div className="h-8 bg-gray-200 rounded" />
          <div className="h-8 bg-gray-200 rounded w-2/3" />
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="card p-4 text-crimson text-sm">
        偏旁数据加载失败：{error}
      </div>
    );
  }

  return (
    <div className="card p-4 space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="font-serif text-lg text-ink">按偏旁选字</h3>
        {selectedChars.length > 0 && (
          <span className="text-sm text-jade">
            已选 {selectedChars.length} 字
          </span>
        )}
      </div>

      {/* 偏旁标签栏 */}
      <div className="flex flex-wrap gap-1.5" role="tablist" aria-label="偏旁部首列表">
        {groups.map((g) => (
          <button
            key={g.radical}
            onClick={() => setActiveRadical(activeRadical === g.radical ? null : g.radical)}
            className={`px-2.5 py-1 text-sm rounded-lg transition-colors ${
              activeRadical === g.radical
                ? 'bg-crimson text-white shadow-sm'
                : 'bg-warm-white text-ink hover:bg-warm-white/80'
            }`}
            role="tab"
            aria-selected={activeRadical === g.radical}
            title={g.name}
          >
            {g.radical}
          </button>
        ))}
      </div>

      {/* 选中偏旁的字表 */}
      {activeGroup && (
        <div className="space-y-2">
          <div className="flex items-center gap-2 text-sm text-jade">
            <span className="font-medium">{activeGroup.name}</span>
            <span className="text-jade/60">·</span>
            <span className="text-jade/60">{activeGroup.meaning}</span>
          </div>
          <div
            className="grid grid-cols-6 sm:grid-cols-8 md:grid-cols-10 gap-1.5"
            role="tabpanel"
            aria-label={`${activeGroup.name}的常用字`}
          >
            {activeGroup.chars.map((char) => {
              const isSelected = selectedChars.includes(char);
              return (
                <button
                  key={char}
                  onClick={() => toggleChar(char)}
                  disabled={disabled}
                  className={`aspect-square flex items-center justify-center text-base rounded-lg transition-all ${
                    isSelected
                      ? 'bg-jade text-white shadow-sm scale-105'
                      : disabled
                        ? 'bg-gray-100 text-gray-400 cursor-not-allowed'
                        : 'bg-warm-white text-ink hover:bg-warm-white/80 hover:shadow-sm active:scale-95'
                  }`}
                  title={char}
                  aria-label={`${char}${isSelected ? '（已选）' : ''}`}
                  aria-pressed={isSelected}
                >
                  {char}
                </button>
              );
            })}
          </div>
          {selectedChars.length > 0 && (
            <div className="flex items-center gap-2 pt-2 border-t border-warm-white">
              <span className="text-sm text-jade">已选：</span>
              <div className="flex flex-wrap gap-1">
                {selectedChars.map((char) => (
                  <span
                    key={char}
                    className="inline-flex items-center gap-1 px-2 py-0.5 bg-jade/10 text-jade text-sm rounded"
                  >
                    {char}
                    <button
                      onClick={() => toggleChar(char)}
                      disabled={disabled}
                      className="hover:text-crimson transition-colors leading-none"
                      aria-label={`移除 ${char}`}
                    >
                      &times;
                    </button>
                  </span>
                ))}
              </div>
            </div>
          )}
        </div>
      )}

      {/* 提示 */}
      {!activeRadical && (
        <p className="text-sm text-jade/60 text-center py-2">
          点击上方偏旁标签浏览对应汉字，点击汉字即可选中
        </p>
      )}
    </div>
  );
}