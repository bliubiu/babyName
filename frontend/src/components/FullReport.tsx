'use client';

import { useState } from 'react';
import { useToast } from './Toast';
import { IconClose } from './Icons';
import type { GenerateResponse } from '@/types';

interface FullReportProps {
  data: GenerateResponse['data'];
  onClose: () => void;
  selectedNameIndex?: number;
}

// 五行中文名映射
const wuxingNameMap: Record<string, string> = {
  jin: '金',
  mu: '木',
  shui: '水',
  huo: '火',
  tu: '土',
};

export default function FullReport({ data, onClose, selectedNameIndex }: FullReportProps) {
  const { showToast } = useToast();
  const [activeTab, setActiveTab] = useState<'bazi' | 'names' | 'detail'>('bazi');

  const copyReport = () => {
    const reportText = generateReportText();
    navigator.clipboard.writeText(reportText).then(() => {
      showToast('报告已复制到剪贴板', 'success');
    }).catch(() => {
      showToast('复制失败', 'error');
    });
  };

  const generateReportText = () => {
    const xiyongshenText = data.bazi.xiyongshen?.length
      ? `喜用神: ${data.bazi.xiyongshen.join('、')}`
      : '';
    return [
      '=== 起名完整报告 ===',
      '',
      `八字: ${data.bazi.bazi.year} ${data.bazi.bazi.month} ${data.bazi.bazi.day} ${data.bazi.bazi.hour}`,
      `日主: ${data.bazi.rishou || '未知'} (${data.bazi.rishou_wuxing || '未知'}性)`,
      `五行: 金${data.bazi.wuxing.jin} 木${data.bazi.wuxing.mu} 水${data.bazi.wuxing.shui} 火${data.bazi.wuxing.huo} 土${data.bazi.wuxing.tu}`,
      xiyongshenText,
      `纳音: ${data.nayin || '未知'}`,
      `生肖: ${data.zodiac || '未知'}`,
      '',
      '推荐名字:',
      ...data.names.map((n, i) => `${i + 1}. ${n.surname}${n.given_name} (${(n.total_score ?? n.score).toFixed(1)}分) 五行:${n.wuxing}`),
    ].filter(Boolean).join('\n');
  };

  // 基于真实数据生成综合解读
  const generateDetailAnalysis = () => {
    const { bazi, names, nayin, zodiac } = data;
    const wuxing = bazi.wuxing;
    const wuxingEntries = Object.entries(wuxing) as [keyof typeof wuxing, number][];
    const total = wuxingEntries.reduce((sum, [, v]) => sum + v, 0);

    // 1. 八字格局分析
    const dayGanzhi = bazi.bazi.day || '';
    const rishou = bazi.rishou || '未知';
    const rishouWuxing = bazi.rishou_wuxing || '未知';
    const baziPattern = `日主为${rishou}（${rishouWuxing}性），日柱${dayGanzhi}。八字四柱为 ${bazi.bazi.year}、${bazi.bazi.month}、${bazi.bazi.day}、${bazi.bazi.hour}。`;

    // 2. 五行平衡分析（基于真实分布）
    const sorted = [...wuxingEntries].sort((a, b) => b[1] - a[1]);
    const strongest = sorted[0];
    const weakest = sorted.find(([, v]) => v < strongest[1]) ?? sorted[sorted.length - 1];
    const wuxingDetail = wuxingEntries
      .map(([k, v]) => `${wuxingNameMap[k]}${v}`)
      .join('、');

    let balanceAnalysis = '';
    if (total === 0) {
      balanceAnalysis = '八字五行数据缺失，建议核对出生时间后重新生成。';
    } else {
      const maxRatio = (strongest[1] / total) * 100;
      const minRatio = (weakest[1] / total) * 100;
      if (maxRatio - minRatio > 50) {
        balanceAnalysis = `五行分布为${wuxingDetail}（共${total}个），其中${wuxingNameMap[strongest[0]]}最旺（${strongest[1]}个，占${maxRatio.toFixed(0)}%），${wuxingNameMap[weakest[0]]}最弱（${weakest[1]}个，占${minRatio.toFixed(0)}%），整体偏枯，起名时需重点补益${wuxingNameMap[weakest[0]]}。`;
      } else if (maxRatio - minRatio > 25) {
        balanceAnalysis = `五行分布为${wuxingDetail}（共${total}个），${wuxingNameMap[strongest[0]]}较旺（${strongest[1]}个），${wuxingNameMap[weakest[0]]}偏弱（${weakest[1]}个），略有失衡，起名可适度补益${wuxingNameMap[weakest[0]]}。`;
      } else {
        balanceAnalysis = `五行分布为${wuxingDetail}（共${total}个），整体较为均衡，无明显偏枯。`;
      }
    }

    // 3. 喜用神补益分析
    let xiyongshenAnalysis = '';
    if (bazi.xiyongshen?.length) {
      const xy = bazi.xiyongshen.join('、');
      const weakestName = total > 0 ? wuxingNameMap[weakest[0]] : '';
      const matchesWeakest = weakestName && bazi.xiyongshen.includes(weakestName);
      if (matchesWeakest) {
        xiyongshenAnalysis = `喜用神为${xy}，与最弱的${weakestName}五行一致，起名时优先选用五行属${xy}的字，可直接补益命局。`;
      } else {
        xiyongshenAnalysis = `喜用神为${xy}，起名时建议选用五行属${xy}或与${xy}相生关系的字，以补益命局不足。`;
      }
    } else {
      xiyongshenAnalysis = '喜用神信息缺失，建议核对八字分析结果。';
    }

    // 4. 生肖与纳音
    const zodiacNayin = `生肖属${zodiac || '未知'}，纳音为${nayin || '未知'}。`;

    // 5. 推荐名字质量分析（基于真实评分）
    let namesAnalysis = '';
    if (names.length === 0) {
      namesAnalysis = '本次未生成推荐名字，建议调整筛选条件后重试。';
    } else {
      const scores = names.map(n => n.total_score ?? n.score);
      const avg = scores.reduce((a, b) => a + b, 0) / scores.length;
      const max = Math.max(...scores);
      const min = Math.min(...scores);
      const topName = names.find(n => (n.total_score ?? n.score) === max);
      const topNameStr = topName ? `${topName.surname}${topName.given_name}` : '';

      // 统计五行匹配情况
      const matchedNames = names.filter(n => {
        if (!n.wuxing || !bazi.xiyongshen?.length) return false;
        return n.wuxing.split('、').some(w => bazi.xiyongshen.includes(w));
      });

      // 统计诗词来源
      const poetryCount = names.filter(n => n.poetry_source).length;

      namesAnalysis = `共推荐${names.length}个名字，综合评分区间${min.toFixed(1)}-${max.toFixed(1)}分，平均${avg.toFixed(1)}分。其中${topNameStr}评分最高（${max.toFixed(1)}分），${matchedNames.length}个名字的五行与喜用神直接匹配，${poetryCount}个名字出自诗词典故。`;
    }

    return {
      baziPattern,
      balanceAnalysis,
      xiyongshenAnalysis,
      zodiacNayin,
      namesAnalysis,
    };
  };

  return (
    <div className="fixed inset-0 bg-ink/30 flex items-center justify-center z-50 overflow-y-auto">
      <div className="relative bg-warm-white-light w-full max-w-2xl mx-4 my-8 rounded-xl shadow-xl border border-paper overflow-hidden">
        <div className="sticky top-0 bg-warm-white border-b border-paper px-6 py-4 flex justify-between items-center z-10">
          <h2 className="text-xl font-serif text-ink">完整起名报告</h2>
          <div className="flex items-center gap-2">
            <button
              onClick={copyReport}
              className="px-4 py-2 border border-paper rounded-lg text-ink-light hover:bg-warm-white text-sm transition-colors"
            >
              复制报告
            </button>
            <button
              onClick={onClose}
              className="p-2 text-ink-light/40 hover:text-seal-red transition-colors"
              title="关闭"
            >
              <IconClose size={20} />
            </button>
          </div>
        </div>

        {/* Tabs */}
        <div className="flex border-b border-paper">
          {[
            { id: 'bazi', label: '八字分析' },
            { id: 'names', label: '名字详情' },
            { id: 'detail', label: '综合解读' },
          ].map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id as 'bazi' | 'names' | 'detail')}
              className={`flex-1 py-3 text-sm font-medium transition-colors ${
                activeTab === tab.id
                  ? 'text-crimson border-b-2 border-crimson'
                  : 'text-ink-light/60 hover:text-ink'
              }`}
            >
              {tab.label}
            </button>
          ))}
        </div>

        <div className="p-6 max-h-[60vh] overflow-y-auto space-y-6">
          {activeTab === 'bazi' && (
            <div>
              <h3 className="text-lg font-serif text-crimson mb-4 pb-2 border-b border-crimson/15">
                八字分析
              </h3>
              <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 mb-6">
                {[
                  { label: '年柱', value: data.bazi.bazi.year },
                  { label: '月柱', value: data.bazi.bazi.month },
                  { label: '日柱', value: data.bazi.bazi.day },
                  { label: '时柱', value: data.bazi.bazi.hour },
                ].map((pillar) => (
                  <div key={pillar.label} className="bg-warm-white p-3 rounded-lg border border-crimson/10 text-center">
                    <span className="text-xs text-jade">{pillar.label}</span>
                    <p className="font-serif text-ink mt-1">{pillar.value}</p>
                  </div>
                ))}
              </div>

              <h4 className="text-sm text-jade font-medium mb-3">五行分布</h4>
              <div className="grid grid-cols-5 gap-2 mb-4">
                {[
                  { label: '金', value: data.bazi.wuxing.jin, color: 'bg-wuxing-jin' },
                  { label: '木', value: data.bazi.wuxing.mu, color: 'bg-wuxing-mu' },
                  { label: '水', value: data.bazi.wuxing.shui, color: 'bg-wuxing-shui' },
                  { label: '火', value: data.bazi.wuxing.huo, color: 'bg-wuxing-huo' },
                  { label: '土', value: data.bazi.wuxing.tu, color: 'bg-wuxing-tu' },
                ].map((w) => (
                  <div key={w.label} className="text-center">
                    <div className="text-xs text-ink-light mb-1">{w.label}</div>
                    <div className="w-full h-2 bg-paper/60 rounded overflow-hidden">
                      <div
                        className={`h-full rounded ${w.color} transition-all duration-500`}
                        style={{ width: `${Math.min(w.value * 20, 100)}%` }}
                      />
                    </div>
                    <div className="text-xs text-ink-light mt-1">{w.value}</div>
                  </div>
                ))}
              </div>

              {data.bazi.xiyongshen?.length > 0 && (
                <div className="text-center p-4 bg-warm-white rounded-lg border border-crimson/10">
                  <span className="text-xs text-jade">喜用神</span>
                  <p className="font-serif text-lg text-crimson mt-1">
                    {data.bazi.xiyongshen.join('、')}
                  </p>
                  <p className="text-xs text-jade mt-1">此为补益八字的关键五行，起名时应优先考虑</p>
                </div>
              )}
            </div>
          )}

          {activeTab === 'names' && (
            <div>
              <h3 className="text-lg font-serif text-crimson mb-4 pb-2 border-b border-crimson/15">
                推荐名字详情
              </h3>
              {data.names.length === 0 ? (
                <div className="text-center py-12 border-2 border-dashed border-paper rounded-lg bg-warm-white">
                  <p className="text-ink-light/60">暂无推荐名字</p>
                </div>
              ) : (
                data.names.map((name, index) => (
                  <div key={index} className={`p-4 rounded-lg border ${selectedNameIndex === index ? 'border-crimson bg-warm-white' : 'border-paper hover:border-crimson/30'} transition-colors mb-3`}>
                    <div className="flex justify-between items-center mb-2">
                      <p className="font-serif text-lg text-ink">{name.surname}{name.given_name}</p>
                      <div className="flex items-center gap-2">
                        <span className="text-xs px-2 py-0.5 bg-gold/15 text-gold-dark rounded">{(name.total_score ?? name.score).toFixed(1)}分</span>
                        <span className="text-xs px-2 py-0.5 bg-warm-white border border-paper text-jade rounded">{name.wuxing}</span>
                      </div>
                    </div>
                    {(name.wuxing_score ?? 0) > 0 && (
                      <div className="flex flex-wrap gap-2 mt-2">
                        {[
                          { label: '五行', val: name.wuxing_score, color: 'bg-amber-400' },
                          { label: '音韵', val: name.yinyun_score, color: 'bg-sky-400' },
                          { label: '字义', val: name.meaning_score, color: 'bg-emerald-400' },
                          { label: '天地人三才', val: name.sancai_score, color: 'bg-violet-400' },
                          { label: '生肖', val: name.zodiac_score, color: 'bg-rose-400' },
                        ].map(dim => (dim.val ?? 0) > 0 ? (
                          <div key={dim.label} className="flex items-center gap-1.5">
                            <span className="text-[10px] text-jade/60">{dim.label}</span>
                            <div className="w-10 h-1.5 bg-paper/40 rounded-full overflow-hidden">
                              <div className={`h-full rounded-full ${dim.color}`} style={{ width: `${Math.min(100, dim.val ?? 0)}%` }} />
                            </div>
                            <span className="text-[10px] text-ink-light/50">{(dim.val ?? 0).toFixed(0)}</span>
                          </div>
                        ) : null)}
                      </div>
                    )}
                  </div>
                ))
              )}
            </div>
          )}

          {activeTab === 'detail' && (() => {
            const analysis = generateDetailAnalysis();
            return (
              <div>
                <h3 className="text-lg font-serif text-crimson mb-4 pb-2 border-b border-crimson/15">
                  综合解读
                </h3>
                <div className="prose prose-sm max-w-none text-ink-light space-y-4">
                  <div className="p-3 bg-warm-white/60 rounded-lg border border-paper">
                    <p className="text-jade font-medium mb-1">八字格局</p>
                    <p className="text-ink-light leading-relaxed">{analysis.baziPattern}</p>
                  </div>

                  <div className="p-3 bg-warm-white/60 rounded-lg border border-paper">
                    <p className="text-jade font-medium mb-1">五行平衡</p>
                    <p className="text-ink-light leading-relaxed">{analysis.balanceAnalysis}</p>
                  </div>

                  <div className="p-3 bg-gold/5 rounded-lg border border-gold/10">
                    <p className="text-jade font-medium mb-1">喜用神补益</p>
                    <p className="text-ink-light leading-relaxed">{analysis.xiyongshenAnalysis}</p>
                  </div>

                  <div className="p-3 bg-warm-white/60 rounded-lg border border-paper">
                    <p className="text-jade font-medium mb-1">生肖纳音</p>
                    <p className="text-ink-light leading-relaxed">{analysis.zodiacNayin}</p>
                  </div>

                  <div className="p-3 bg-crimson/5 rounded-lg border border-crimson/10">
                    <p className="text-jade font-medium mb-1">推荐名字质量</p>
                    <p className="text-ink-light leading-relaxed">{analysis.namesAnalysis}</p>
                  </div>
                </div>
              </div>
            );
          })()}
        </div>
      </div>
    </div>
  );
}
