'use client';

import { useRef, useState } from 'react';
import { GenerateResponse } from '@/types';
import { useToast } from '@/components/Toast';
import { Spinner } from '@/components/Spinner';

interface FullReportProps {
  data: GenerateResponse['data'];
  onClose?: () => void;
  selectedNameIndex?: number;
}

export default function FullReport({ data, onClose, selectedNameIndex }: FullReportProps) {
  const reportRef = useRef<HTMLDivElement>(null);
  const { showToast } = useToast();
  const [isGenerating, setIsGenerating] = useState(false);

  const exportAsPDF = async () => {
    if (!reportRef.current) return;
    
    setIsGenerating(true);
    try {
      const html2pdf = (await import('html2pdf.js')).default;
      
      const opt = {
        margin: 10,
        filename: `宝宝起名报告_${new Date().toISOString().slice(0, 10)}.pdf`,
        image: { type: 'jpeg' as const, quality: 0.98 },
        html2canvas: {
          scale: 2,
          backgroundColor: '#FDF8F3',
          useCORS: true
        },
        jsPDF: { 
          unit: 'mm', 
          format: 'a4', 
          orientation: 'portrait' as const 
        },
        pagebreak: { mode: ['avoid-all', 'css', 'legacy'] }
      };
      
      await html2pdf().set(opt).from(reportRef.current).save();
      showToast('报告导出成功', 'success');
    } catch (error) {
      console.error('PDF export error:', error);
      showToast('报告导出失败', 'error');
    } finally {
      setIsGenerating(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 overflow-auto p-4">
      <div className="bg-white rounded-lg w-full max-w-4xl max-h-[90vh] overflow-auto">
        <div className="sticky top-0 bg-white border-b border-gray-200 px-6 py-4 flex justify-between items-center z-10">
          <h2 className="text-xl font-bold text-ink">完整起名报告</h2>
          <div className="flex gap-3">
            <button
              onClick={exportAsPDF}
              disabled={isGenerating}
              className="px-4 py-2 bg-crimson text-white rounded-lg hover:bg-crimson/90 transition-colors disabled:opacity-50 flex items-center gap-2"
            >
              {isGenerating ? (
                <>
                  <Spinner size="small" color="white" />
                  <span>生成中...</span>
                </>
              ) : (
                '📄 导出PDF'
              )}
            </button>
            {onClose && (
              <button
                onClick={onClose}
                className="px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
              >
                关闭
              </button>
            )}
          </div>
        </div>

        <div ref={reportRef} className="p-6 bg-[#FDF8F3]">
          <ReportContent data={data} selectedNameIndex={selectedNameIndex} />
        </div>
      </div>
    </div>
  );
}

function ReportContent({ data, selectedNameIndex }: { data: GenerateResponse['data']; selectedNameIndex?: number }) {
  const { bazi, nayin, zodiac, hexagram, names } = data;
  const topNames = names.slice(0, 10);
  const backupNames = names.slice(10, 50);

  return (
    <div className="space-y-8">
      <ReportCover data={data} selectedNameIndex={selectedNameIndex} />
      <ReportBaziAnalysis bazi={bazi} nayin={nayin} zodiac={zodiac} />
      {hexagram && <ReportHexagram hexagram={hexagram} />}
      <ReportTopNames names={topNames} />
      {backupNames.length > 0 && <ReportBackupNames names={backupNames} />}
      <ReportKnowledge />
    </div>
  );
}

function ReportCover({ data, selectedNameIndex = 0 }: { data: GenerateResponse['data']; selectedNameIndex?: number }) {
  const { bazi, nayin, zodiac, names } = data;
  const topName = names[selectedNameIndex];

  return (
    <div className="text-center py-12 border-2 border-amber-200 rounded-lg bg-gradient-to-br from-amber-50 to-white">
      <div className="mb-6">
        <h1 className="text-4xl md:text-5xl font-serif text-crimson mb-4">
          宝宝起名报告
        </h1>
        <div className="w-24 h-1 bg-gold mx-auto"></div>
      </div>

      {topName && (
        <div className="mb-8">
          <p className="text-3xl md:text-4xl font-serif text-ink mb-2">
            {topName.full_name || `${topName.surname}${topName.given_name}`}
          </p>
          <p className="text-lg text-teal">{topName.pinyin}</p>
        </div>
      )}

      <div className="space-y-2 text-teal">
        <p className="text-base">
          出生时间：{bazi.bazi.year}年{bazi.bazi.month}月{bazi.bazi.day}日{bazi.bazi.hour}时
        </p>
        <p className="text-base">
          生肖：{zodiac}
        </p>
        <p className="text-base">
          生成日期：{new Date().toLocaleDateString('zh-CN')}
        </p>
      </div>

      <div className="mt-8 text-sm text-teal">
        <p>本报告由宝宝起名大师生成</p>
        <p>基于传统玄学与现代大数据分析</p>
      </div>
    </div>
  );
}

function ReportBaziAnalysis({ bazi, nayin, zodiac }: { bazi: any; nayin: string; zodiac: string }) {
  const getWuxingName = (key: string): string => {
    switch (key) {
      case 'jin': return '金';
      case 'mu': return '木';
      case 'shui': return '水';
      case 'huo': return '火';
      case 'tu': return '土';
      default: return key;
    }
  };

  return (
    <div className="border-2 border-amber-200 rounded-lg p-6 bg-white">
      <h2 className="text-2xl font-serif text-ink mb-6 pb-2 border-b border-amber-200">
        八字分析
      </h2>

      <div className="grid md:grid-cols-2 gap-6 mb-6">
        <div>
          <h3 className="font-bold text-ink mb-3">四柱八字</h3>
          <div className="grid grid-cols-4 gap-2 text-center">
            <div className="bg-amber-50 p-3 rounded">
              <p className="text-sm text-teal">年柱</p>
              <p className="text-lg font-serif text-ink">{bazi.bazi.year}</p>
            </div>
            <div className="bg-amber-50 p-3 rounded">
              <p className="text-sm text-teal">月柱</p>
              <p className="text-lg font-serif text-ink">{bazi.bazi.month}</p>
            </div>
            <div className="bg-amber-50 p-3 rounded">
              <p className="text-sm text-teal">日柱</p>
              <p className="text-lg font-serif text-ink">{bazi.bazi.day}</p>
            </div>
            <div className="bg-amber-50 p-3 rounded">
              <p className="text-sm text-teal">时柱</p>
              <p className="text-lg font-serif text-ink">{bazi.bazi.hour}</p>
            </div>
          </div>
        </div>

        <div>
          <h3 className="font-bold text-ink mb-3">日主分析</h3>
          <div className="space-y-2">
            <p className="text-teal">日主：{bazi.rishou}（{bazi.rishou_wuxing}性）</p>
            <p className="text-teal">强弱：{bazi.rishou_wuxing === '木' ? '身强' : '身弱'}</p>
            <p className="text-teal">纳音：{nayin}</p>
            <p className="text-teal">生肖：{zodiac}</p>
          </div>
        </div>
      </div>

      <div className="mb-6">
        <h3 className="font-bold text-ink mb-3">五行分布</h3>
        <div className="space-y-2">
          {Object.entries(bazi.wuxing).map(([key, value]) => (
            <div key={key} className="flex items-center gap-3">
              <span className="w-12 text-center font-bold text-ink">{getWuxingName(key)}</span>
              <div className="flex-1 h-4 bg-gray-200 rounded overflow-hidden">
                <div
                  className="h-full bg-crimson rounded"
                  style={{ width: `${(value as number / 8) * 100}%` }}
                />
              </div>
              <span className="w-8 text-center text-ink">{String(value)}</span>
            </div>
          ))}
        </div>
      </div>

      <div>
        <h3 className="font-bold text-ink mb-3">喜用神</h3>
        <p className="text-lg text-gold">{bazi.xiyongshen.join('、')}</p>
        <p className="text-sm text-teal mt-2">
          根据八字分析，宝宝的命局喜用神为{bazi.xiyongshen.join('、')}，
          在起名时应优先选择包含这些五行的汉字，以达到五行平衡、运势亨通的效果。
        </p>
      </div>
    </div>
  );
}

function ReportHexagram({ hexagram }: { hexagram: any }) {
  return (
    <div className="border-2 border-amber-200 rounded-lg p-6 bg-white">
      <h2 className="text-2xl font-serif text-ink mb-6 pb-2 border-b border-amber-200">
        易经卦象
      </h2>

      <div className="flex items-start gap-6 mb-6">
        <div className="text-6xl font-serif text-ink">{hexagram.symbol}</div>
        <div className="flex-1">
          <h3 className="text-xl font-bold text-ink mb-2">{hexagram.name}卦</h3>
          <p className="text-teal mb-2">卦辞：{hexagram.gua_ci}</p>
          <p className="text-teal mb-2">象辞：{hexagram.xiang_ci}</p>
        </div>
      </div>

      <div className="mb-6">
        <h3 className="font-bold text-ink mb-3">卦象解读</h3>
        <p className="text-teal leading-relaxed">{hexagram.interpretation}</p>
      </div>

      {hexagram.yao_ci && hexagram.yao_ci.length > 0 && (
        <div>
          <h3 className="font-bold text-ink mb-3">爻辞</h3>
          <div className="space-y-2">
            {hexagram.yao_ci.map((yao: string, index: number) => (
              <p key={index} className="text-teal">
                <span className="font-bold">第{index + 1}爻：</span>{yao}
              </p>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

function ReportTopNames({ names }: { names: any[] }) {
  return (
    <div className="border-2 border-amber-200 rounded-lg p-6 bg-white">
      <h2 className="text-2xl font-serif text-ink mb-6 pb-2 border-b border-amber-200">
        推荐名字（Top 10）
      </h2>

      <div className="space-y-6">
        {names.map((name, index) => (
          <div key={index} className="border-b border-gray-200 pb-4 last:border-0">
            <div className="flex items-start gap-4">
              <div className="text-3xl font-bold text-gold w-10 flex-shrink-0">
                {index + 1}
              </div>
              <div className="flex-1">
                <div className="flex items-center gap-3 mb-2">
                  <h3 className="text-2xl font-serif text-ink">
                    {name.full_name || `${name.surname}${name.given_name}`}
                  </h3>
                  <span className="px-2 py-1 bg-gold/20 text-gold rounded text-sm">
                    {name.score}分
                  </span>
                </div>
                <p className="text-teal mb-2">{name.pinyin}</p>
                <p className="text-ink mb-2">{name.meaning}</p>
                
                {name.wuxing_analysis && (
                  <p className="text-sm text-teal mb-1">
                    <span className="font-bold">五行分析：</span>{name.wuxing_analysis}
                  </p>
                )}
                {name.bazi_score_detail && (
                  <p className="text-sm text-teal mb-1">
                    <span className="font-bold">八字评分：</span>{name.bazi_score_detail}
                  </p>
                )}
                {name.yinyun && (
                  <p className="text-sm text-teal mb-1">
                    <span className="font-bold">音韵意境：</span>{name.yinyun}
                  </p>
                )}
                {name.poetry_source && (
                  <p className="text-sm text-teal">
                    <span className="font-bold">诗词典故：</span>
                    出自《{name.poetry_source}》
                    {name.poetry_chapter && <>《{name.poetry_chapter}》</>}
                    {name.poetry_sentence && <span>：{name.poetry_sentence}</span>}
                  </p>
                )}
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

function ReportBackupNames({ names }: { names: any[] }) {
  return (
    <div className="border-2 border-amber-200 rounded-lg p-6 bg-white">
      <h2 className="text-2xl font-serif text-ink mb-6 pb-2 border-b border-amber-200">
        备选名字（Top 11-50）
      </h2>

      <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
        {names.map((name, index) => (
          <div key={index} className="bg-amber-50 p-3 rounded">
            <h3 className="text-lg font-serif text-ink mb-1">
              {name.full_name || `${name.surname}${name.given_name}`}
            </h3>
            <p className="text-sm text-teal mb-1">{name.pinyin}</p>
            <p className="text-sm text-ink">{name.meaning}</p>
            <div className="flex gap-2 mt-2">
              <span className="text-xs px-2 py-0.5 bg-white text-teal rounded">
                {name.gender === 'male' ? '男' : '女'}
              </span>
              <span className="text-xs px-2 py-0.5 bg-white text-gold rounded">
                {parseFloat(name.score).toFixed(2)}分
              </span>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

function ReportKnowledge() {
  return (
    <div className="border-2 border-amber-200 rounded-lg p-6 bg-white">
      <h2 className="text-2xl font-serif text-ink mb-6 pb-2 border-b border-amber-200">
        玄学知识科普
      </h2>

      <div className="space-y-6">
        <div>
          <h3 className="text-xl font-bold text-ink mb-3">八字基础</h3>
          <p className="text-teal leading-relaxed">
            八字，又称四柱，是中国传统命理学的重要组成部分。它根据人的出生年、月、日、时，
            分别用天干地支表示，形成四柱八字。通过分析八字中五行的生克制化关系，
            可以推断一个人的性格特点、运势走向等。
          </p>
        </div>

        <div>
          <h3 className="text-xl font-bold text-ink mb-3">五行学说</h3>
          <p className="text-teal leading-relaxed">
            五行是指金、木、水、火、土五种基本物质。五行之间存在相生相克的关系：
            金生水、水生木、木生火、火生土、土生金；金克木、木克土、土克水、水克火、火克金。
            在起名时，通过分析八字中五行的分布，选择合适的五行汉字，可以达到平衡命局的效果。
          </p>
        </div>

        <div>
          <h3 className="text-xl font-bold text-ink mb-3">生肖文化</h3>
          <p className="text-teal leading-relaxed">
            生肖是中国传统民俗文化的重要组成部分，共有12种动物，每12年循环一次。
            每个生肖都有其独特的性格特点和喜忌。在起名时，可以根据生肖的喜忌，
            选择合适的偏旁部首和汉字，以增强名字的吉祥寓意。
          </p>
        </div>

        <div>
          <h3 className="text-xl font-bold text-ink mb-3">易经入门</h3>
          <p className="text-teal leading-relaxed">
            易经是中国古代最重要的典籍之一，包含64卦，每卦由6爻组成。
            通过姓名笔画起卦，可以得到对应的卦象，从而解读名字的吉凶寓意。
            卦象分析是起名的重要参考依据之一，可以帮助父母选择更加吉祥的名字。
          </p>
        </div>
      </div>
    </div>
  );
}
