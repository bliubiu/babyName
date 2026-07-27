'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useNameStore } from '@/lib/store';
import { useToast } from '@/components/Toast';
import { Name } from '@/types';
import Navigation from '@/components/Navigation';

const wuxingColors: Record<string, string> = {
  '木': 'text-green-600',
  '火': 'text-red-500',
  '土': 'text-amber-600',
  '金': 'text-gray-500',
  '水': 'text-blue-500',
};

// 用 surname:given_name 作为稳定 key，避免列表渲染时 index 错位
const nameKey = (n: Name) => `${n.surname}:${n.given_name}`;

export default function ComparePage() {
  const router = useRouter();
  const { compareResult, setCompareResult, _hasHydrated } = useNameStore();
  const { showToast } = useToast();

  const metrics = [
    { key: 'score', label: '综合评分', getValue: (n: Name) => Math.min(100, n.total_score ?? n.score), unit: '分' },
    { key: 'wuxing', label: '五行', getValue: (n: Name) => n.wuxing },
    { key: 'wuxing_score', label: '五行匹配', getValue: (n: Name) => +(n.wuxing_score ?? 0).toFixed(1), unit: '分' },
    { key: 'yinyun_score', label: '音韵律动', getValue: (n: Name) => +(n.yinyun_score ?? 0).toFixed(1), unit: '分' },
    { key: 'meaning_score', label: '字义内涵', getValue: (n: Name) => +(n.meaning_score ?? 0).toFixed(1), unit: '分' },
    { key: 'sancai_score', label: '天地人三才', getValue: (n: Name) => +(n.sancai_score ?? 0).toFixed(1), unit: '分' },
    { key: 'zodiac_score', label: '生肖适配', getValue: (n: Name) => +(n.zodiac_score ?? 0).toFixed(1), unit: '分' },
    { key: 'strokes', label: '笔画数', getValue: (n: Name) => n.strokes, unit: '画' },
    { key: 'pinyin', label: '拼音', getValue: (n: Name) => n.pinyin },
    { key: 'meaning', label: '字义', getValue: (n: Name) => n.meaning },
    { key: 'gender', label: '性别', getValue: (n: Name) => n.gender === 'male' ? '男' : '女' },
  ];

  useEffect(() => {
    return () => { setCompareResult(null); };
  }, [setCompareResult]);

  // hydrate 未完成时显示等待，避免刷新后立即显示空状态
  if (!_hasHydrated) {
    return (
      <main className="min-h-screen py-6 md:py-8 px-4 md:px-6 cloud-bg">
        <div className="max-w-4xl mx-auto">
          <Navigation showBackButton={true} showHistory={true} showFavorites={true} />
          <div className="consultation-sheet text-center py-16 animate-fade-in-up corner-decor">
            <p className="text-jade text-sm tracking-wider">加载中...</p>
          </div>
        </div>
      </main>
    );
  }

  if (!compareResult || compareResult.length < 2) {
    return (
      <main className="min-h-screen py-6 md:py-8 px-4 md:px-6 cloud-bg">
        <div className="max-w-4xl mx-auto">
          <Navigation showBackButton={true} showHistory={true} showFavorites={true} />
          <div className="consultation-sheet text-center py-16 animate-fade-in-up corner-decor">
            <p className="text-jade text-sm tracking-wider">请先选择至少2个名字进行对比</p>
            <p className="text-ink-light/40 text-xs mt-2">在结果页勾选名字后点击「对比」</p>
            <button onClick={() => router.push('/')} className="mt-6 btn-primary">去起名</button>
          </div>
        </div>
      </main>
    );
  }

  const maxScore = Math.max(...compareResult.map(n => n.total_score ?? n.score));
  // 维度评分颜色映射
  const dimColors: Record<string, string> = {
    wuxing_score: 'bg-amber-500',
    yinyun_score: 'bg-sky-500',
    meaning_score: 'bg-emerald-500',
    sancai_score: 'bg-violet-400',
    zodiac_score: 'bg-rose-400',
  };

  return (
    <main className="min-h-screen py-6 md:py-8 px-4 md:px-6 cloud-bg">
      <div className="max-w-4xl mx-auto">
        <Navigation showBackButton={true} showHistory={true} showFavorites={true} />

        {/* 标题 */}
        <div className="text-center mb-6 md:mb-8 animate-fade-in-up">
          <h1 className="font-serif text-2xl md:text-3xl text-ink ink-calligraphy">名字对比</h1>
          <div className="flex items-center justify-center gap-4 mt-3">
            <div className="brush-divider max-w-[60px] flex-1" />
            <span className="text-paper-edge/40 text-xs tracking-[0.5em] font-serif">权衡</span>
            <div className="brush-divider max-w-[60px] flex-1" />
          </div>
        </div>

        {/* 对比表格 */}
        <div className="card mb-6 animate-fade-in-up overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full min-w-[500px]">
              <thead>
                <tr className="border-b border-paper/30">
                  <th className="text-left py-3 px-3 md:px-4 text-jade text-sm font-medium">对比项</th>
                  {compareResult.map((name, idx) => (
                    <th key={idx} className="text-center py-3 px-3 md:px-4">
                      <div className="font-serif text-xl md:text-2xl text-ink">
                        {name.surname}{name.given_name}
                      </div>
                      <div className="text-xs text-jade/70 mt-0.5">{name.pinyin}</div>
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {metrics.map((metric) => (
                  <tr key={metric.key} className="border-b border-paper/15 hover:bg-warm-white/30 transition-colors">
                    <td className="py-3 px-3 md:px-4 text-jade text-sm font-medium">{metric.label}</td>
                    {compareResult.map((name, idx) => (
                      <td key={nameKey(name)} className="text-center py-3 px-3 md:px-4">
                        {metric.key === 'score' || dimColors[metric.key] ? (
                          <div className="flex flex-col items-center">
                            <span className={`text-lg font-medium ${metric.key === 'score' ? 'text-gold' : 'text-ink'}`}>
                              {metric.getValue(name)}{metric.unit}
                            </span>
                            <div className="w-16 h-1.5 bg-paper/30 rounded mt-1.5 overflow-hidden">
                              <div
                                className={`h-full rounded transition-all ${dimColors[metric.key] || 'bg-gold'}`}
                                style={{ width: `${Math.min(100, metric.getValue(name) as number)}%` }}
                              />
                            </div>
                          </div>
                        ) : metric.key === 'wuxing' ? (
                          <span className={`text-lg font-medium ${wuxingColors[metric.getValue(name) as string] || 'text-ink'}`}>
                            {metric.getValue(name)}
                          </span>
                        ) : metric.key === 'meaning' ? (
                          <span className="text-sm text-ink max-w-[150px] truncate block mx-auto">
                            {metric.getValue(name)}
                          </span>
                        ) : (
                          <span className="text-ink">{metric.getValue(name)}{metric.unit}</span>
                        )}
                      </td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        {/* 综合分析 */}
        <div className="consultation-sheet animate-fade-in-up corner-decor">
          <div className="flex items-center gap-3 mb-4">
            <span className="seal-badge seal-stamp-sm">析</span>
            <div className="brush-divider flex-1" />
          </div>
          <h2 className="font-serif text-lg text-ink mb-4 ink-calligraphy text-center">综合分析</h2>
          <div className="space-y-4">
            {compareResult.map((name, idx) => (
              <div key={nameKey(name)} className="p-3 bg-warm-white/60 rounded-xl">
                <span className="font-serif text-lg text-ink">{name.surname}{name.given_name}</span>
                <div className="text-sm text-jade mt-2 space-y-1">
                  {name.wuxing_analysis && <p>· {name.wuxing_analysis}</p>}
                  {name.bazi_score_detail && <p>· {name.bazi_score_detail}</p>}
                  {name.yinyun && <p>· {name.yinyun}</p>}
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="mt-6 text-center">
          <button onClick={() => router.push('/')} className="btn-primary">重新起名</button>
        </div>
      </div>
    </main>
  );
}
