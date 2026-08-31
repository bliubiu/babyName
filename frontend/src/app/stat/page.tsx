'use client';

import React, { useState, useCallback } from 'react';
import { useQuery } from '@tanstack/react-query';
import { getNameStats, getTopNames } from '@/lib/api';
import { Spinner } from '@/components/Spinner';
import Navigation from '@/components/Navigation';
import type { NameStat } from '@/types';

// 重名程度评级
// rate 为占语料库百分比数值（如 0.0523 表示 0.0523%）
function rateLevel(rate: number): { label: string; className: string } {
  if (rate >= 1) return { label: '重名度极高', className: 'bg-crimson/10 text-crimson' };
  if (rate >= 0.1) return { label: '重名度较高', className: 'bg-crimson/5 text-crimson' };
  if (rate >= 0.01) return { label: '重名度一般', className: 'bg-gold/10 text-gold' };
  if (rate > 0) return { label: '重名度较低', className: 'bg-jade/10 text-jade' };
  return { label: '未收录', className: 'bg-warm-white text-paper-edge/60' };
}

// 格式化占比：0.0523 → "0.052%"
function formatRate(rate: number): string {
  return `${rate.toFixed(3)}%`;
}

// 格式化次数：123456 → "123,456"
function formatCount(count: number): string {
  return count.toLocaleString('zh-CN');
}

const StatPage = () => {
  const [input, setInput] = useState('');
  const [submittedName, setSubmittedName] = useState('');

  // 名字重名率查询：仅在提交后触发
  const nameQuery = useQuery({
    queryKey: ['namestat', submittedName],
    queryFn: () => getNameStats(submittedName),
    enabled: submittedName !== '',
  });

  // 热门名字榜单
  const topQuery = useQuery({
    queryKey: ['namestat-top'],
    queryFn: () => getTopNames(10),
  });

  const handleSubmit = useCallback(() => {
    const name = input.trim();
    if (!name) return;
    setSubmittedName(name);
  }, [input]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLInputElement>) => {
      if (e.key === 'Enter') {
        handleSubmit();
      }
    },
    [handleSubmit],
  );

  const stats: NameStat | null =
    nameQuery.data && nameQuery.data.count > 0 ? nameQuery.data : submittedName ? { name: submittedName, count: 0, rate: 0, rank: 0 } : null;
  const level = stats ? rateLevel(stats.rate) : null;
  const topNames = topQuery.data ?? [];

  return (
    <main className="min-h-screen py-6 md:py-8 px-4 md:px-6 cloud-bg">
      <div className="max-w-4xl mx-auto">
        <Navigation showBackButton={true} showHistory={true} showFavorites={true} />

        {/* 标题 */}
        <div className="text-center mb-6 md:mb-8 animate-fade-in-up">
          <h1 className="font-serif text-2xl md:text-3xl text-ink ink-calligraphy">重名率查询</h1>
          <div className="flex items-center justify-center gap-4 mt-3">
            <div className="brush-divider max-w-[60px] flex-1" />
            <span className="text-paper-edge/40 text-xs tracking-[0.5em] font-serif">同名之缘</span>
            <div className="brush-divider max-w-[60px] flex-1" />
          </div>
        </div>

        {/* 查询区块 */}
        <div className="card mb-6 animate-fade-in-up">
          <div className="flex flex-col sm:flex-row gap-3">
            <input
              type="text"
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="请输入名字，如：梓涵"
              maxLength={4}
              className="flex-1 bg-warm-white/60 text-ink border border-warm-white focus:border-crimson/40 rounded-lg px-4 py-2.5 text-sm focus:outline-none transition-colors"
              aria-label="输入名字查询重名率"
            />
            <button
              onClick={handleSubmit}
              disabled={!input.trim()}
              className="btn-primary disabled:opacity-40 disabled:cursor-not-allowed whitespace-nowrap"
            >
              查询
            </button>
          </div>

          {nameQuery.isLoading && (
            <div className="flex justify-center items-center py-10">
              <Spinner size="large" />
            </div>
          )}
          {nameQuery.isError && (
            <div className="text-center py-8">
              <p className="text-crimson mb-3">查询失败，请重试</p>
              <button onClick={() => nameQuery.refetch()} className="btn-primary">
                重试
              </button>
            </div>
          )}
          {stats && !nameQuery.isLoading && !nameQuery.isError && (
            <div className="mt-6 pt-6 border-t border-warm-white">
              <div className="flex flex-col sm:flex-row items-center gap-4 sm:gap-6 justify-center">
                {/* 名字与评级 */}
                <div className="text-center">
                  <div className="font-serif text-3xl md:text-4xl text-ink ink-calligraphy">{stats.name}</div>
                  {level && (
                    <span className={`inline-block mt-2 px-3 py-1 rounded-full text-xs ${level.className}`}>
                      {level.label}
                    </span>
                  )}
                </div>

                {/* 统计数字 */}
                <div className="grid grid-cols-3 gap-4 text-center">
                  <div>
                    <div className="text-xl md:text-2xl text-jade font-medium">{formatCount(stats.count)}</div>
                    <div className="text-xs text-paper-edge/50 mt-1">重名人数</div>
                  </div>
                  <div>
                    <div className="text-xl md:text-2xl text-jade font-medium">{formatRate(stats.rate)}</div>
                    <div className="text-xs text-paper-edge/50 mt-1">占比</div>
                  </div>
                  <div>
                    <div className="text-xl md:text-2xl text-jade font-medium">{stats.rank > 0 ? `#${stats.rank}` : '—'}</div>
                    <div className="text-xs text-paper-edge/50 mt-1">频率排名</div>
                  </div>
                </div>
              </div>

              {stats.count === 0 && (
                <p className="text-center text-xs text-paper-edge/60 mt-4">
                  该名字暂未收录于统计数据中，换个写法试试？
                </p>
              )}
            </div>
          )}
        </div>

        {/* 热门榜区块 */}
        <div className="card animate-fade-in-up" style={{ animationDelay: '0.1s' }}>
          <div className="flex items-center justify-between mb-4">
            <h2 className="font-serif text-lg text-ink ink-calligraphy">热门名字榜</h2>
            <button
              onClick={() => topQuery.refetch()}
              className="px-3 py-1.5 bg-warm-white/60 text-jade rounded-lg text-sm hover:bg-warm-white transition-all"
              aria-label="刷新热门榜单"
            >
              刷新
            </button>
          </div>

          {topQuery.isLoading && (
            <div className="flex justify-center items-center py-10">
              <Spinner size="large" />
            </div>
          )}
          {topQuery.isError && (
            <div className="text-center py-8">
              <p className="text-crimson mb-3">榜单加载失败，请重试</p>
              <button onClick={() => topQuery.refetch()} className="btn-primary">
                重试
              </button>
            </div>
          )}
          {!topQuery.isLoading && !topQuery.isError && topNames.length === 0 && (
            <p className="text-center text-paper-edge/60 py-8 text-sm">暂无数据</p>
          )}
          {!topQuery.isLoading && !topQuery.isError && topNames.length > 0 && (
            <ul className="space-y-2">
              {topNames.map((item, idx) => (
                <li
                  key={item.name}
                  className={`flex items-center gap-3 px-3 py-2.5 rounded-lg ${
                    idx < 3 ? 'bg-crimson/5' : 'bg-warm-white/40'
                  }`}
                >
                  <span
                    className={`w-7 h-7 flex items-center justify-center rounded-full text-sm font-medium shrink-0 ${
                      idx < 3 ? 'bg-crimson text-white' : 'bg-warm-white text-jade'
                    }`}
                  >
                    {item.rank}
                  </span>
                  <span className="font-serif text-lg text-ink flex-1">{item.name}</span>
                  <span className="text-sm text-jade">{formatCount(item.count)}人</span>
                  <span className="text-xs text-paper-edge/50 w-20 text-right">{formatRate(item.rate)}</span>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </main>
  );
};

export default StatPage;