'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useNameStore } from '@/lib/store';
import { useToast } from '@/components/Toast';
import { PageLoader } from '@/components/Spinner';

export default function ComparePage() {
  const router = useRouter();
  const { compareResult, setCompareResult } = useNameStore();
  const { showToast } = useToast();
  const [selectedMetrics, setSelectedMetrics] = useState<string[]>(['score', 'wuxing', 'strokes']);

  const metrics = [
    { key: 'score', label: '综合评分', getValue: (n: any) => n.score, unit: '分' },
    { key: 'wuxing', label: '五行', getValue: (n: any) => n.wuxing },
    { key: 'strokes', label: '笔画数', getValue: (n: any) => n.strokes, unit: '画' },
    { key: 'pinyin', label: '拼音', getValue: (n: any) => n.pinyin },
    { key: 'meaning', label: '字义', getValue: (n: any) => n.meaning },
    { key: 'gender', label: '性别', getValue: (n: any) => n.gender === 'male' ? '男' : '女' },
  ];

  useEffect(() => {
    return () => {
      setCompareResult(null);
    };
  }, []);

  if (!compareResult || compareResult.length < 2) {
    return (
      <main className="min-h-screen py-6 md:py-8 px-3 md:px-4">
        <div className="max-w-4xl mx-auto">
          <button
            onClick={() => router.back()}
            className="flex items-center gap-2 text-teal hover:text-crimson mb-4 md:mb-6 transition-colors text-sm md:text-base"
          >
            ← 返回
          </button>
          <div className="card text-center py-12">
            <p className="text-2xl mb-4">📊</p>
            <p className="text-teal">请先选择至少2个名字进行对比</p>
          </div>
        </div>
      </main>
    );
  }

  const maxScore = Math.max(...compareResult.map(n => n.score));

  return (
    <main className="min-h-screen py-6 md:py-8 px-3 md:px-4">
      <div className="max-w-4xl mx-auto">
        <button
          onClick={() => router.back()}
          className="flex items-center gap-2 text-teal hover:text-crimson mb-4 md:mb-6 transition-colors text-sm md:text-base"
        >
          ← 返回
        </button>

        <h1 className="font-serif text-2xl md:text-3xl text-ink mb-6 md:mb-8 text-center animate-fadeInUp">
          📊 名字对比
        </h1>

        <div className="overflow-x-auto">
          <table className="w-full min-w-[600px]">
            <thead>
              <tr className="border-b border-warm-white">
                <th className="text-left py-3 px-2 md:px-4 text-teal font-medium">对比项</th>
                {compareResult.map((name, idx) => (
                  <th key={idx} className="text-center py-3 px-2 md:px-4">
                    <div className="font-serif text-xl md:text-2xl text-ink">
                      {name.surname}{name.given_name}
                    </div>
                    <div className="text-xs text-teal">{name.pinyin}</div>
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {metrics.map((metric) => (
                <tr key={metric.key} className="border-b border-warm-white/50 hover:bg-warm-white/30">
                  <td className="py-3 px-2 md:px-4 text-teal font-medium">{metric.label}</td>
                  {compareResult.map((name, idx) => (
                    <td key={idx} className="text-center py-3 px-2 md:px-4">
                      {metric.key === 'score' ? (
                        <div className="flex flex-col items-center">
                          <span className="text-lg md:text-xl text-gold font-medium">
                            {metric.getValue(name)}{metric.unit}
                          </span>
                          <div className="w-16 h-1.5 bg-warm-white rounded mt-1 overflow-hidden">
                            <div 
                              className="h-full bg-gold rounded transition-all"
                              style={{ width: `${(metric.getValue(name) as number) / maxScore * 100}%` }}
                            />
                          </div>
                        </div>
                      ) : metric.key === 'wuxing' ? (
                        <span className={`text-lg font-medium ${
                          metric.getValue(name) === '木' ? 'text-green-600' :
                          metric.getValue(name) === '火' ? 'text-red-500' :
                          metric.getValue(name) === '土' ? 'text-amber-600' :
                          metric.getValue(name) === '金' ? 'text-gray-500' :
                          'text-blue-500'
                        }`}>
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

        <div className="mt-6 md:mt-8 card animate-fadeInUp">
          <h2 className="font-serif text-lg md:text-xl text-ink mb-3 md:mb-4">📝 综合分析</h2>
          <div className="space-y-3">
            {compareResult.map((name, idx) => (
              <div key={idx} className="flex items-start gap-3">
                <span className="font-serif text-lg text-ink">{name.surname}{name.given_name}</span>
                <div className="text-sm text-teal">
                  {name.wuxing_analysis && <p>• {name.wuxing_analysis}</p>}
                  {name.bazi_score_detail && <p>• {name.bazi_score_detail}</p>}
                  {name.yinyun && <p>• {name.yinyun}</p>}
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="mt-6 text-center">
          <button
            onClick={() => router.push('/')}
            className="px-6 py-2 bg-crimson text-white rounded-lg hover:bg-crimson/90 transition-colors text-sm md:text-base"
          >
            重新起名
          </button>
        </div>
      </div>
    </main>
  );
}
