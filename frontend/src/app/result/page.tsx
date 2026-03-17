'use client';

import { useEffect, useState, useRef } from 'react';
import { useRouter } from 'next/navigation';
import { useNameStore } from '@/lib/store';
import { Name } from '@/types';
import { PageLoader } from '@/components/Spinner';
import { useToast } from '@/components/Toast';

export default function ResultPage() {
  const router = useRouter();
  const { generateResult, setGenerateResult, setCompareResult, favorites, addFavorite, removeFavorite } = useNameStore();
  const [selectedName, setSelectedName] = useState<number | null>(null);
  const [favoriteStatus, setFavoriteStatus] = useState<Record<number, boolean>>({});
  const [compareNames, setCompareNames] = useState<number[]>([]);
  const { showToast } = useToast();

  const result = generateResult;

  useEffect(() => {
    const checkFavorites = async () => {
      if (!result?.names) return;
      
      const status: Record<number, boolean> = {};
      for (let i = 0; i < result.names.length; i++) {
        const name = result.names[i];
        const isFav = favorites.some(
          f => f.surname === name.surname && f.given_name === name.given_name
        );
        status[i] = isFav;
      }
      setFavoriteStatus(status);
    };
    
    checkFavorites();
  }, [result, favorites]);

  const toggleFavorite = (name: Name, index: number) => {
    const fullName = name.surname + name.given_name;
    const existing = favorites.find(
      f => f.surname === name.surname && f.given_name === name.given_name
    );
    
    if (existing) {
      if (existing.id) {
        removeFavorite(existing.id);
      }
      setFavoriteStatus(prev => ({ ...prev, [index]: false }));
      showToast('已取消收藏', 'info');
    } else {
      const newFav = {
        surname: name.surname,
        given_name: name.given_name,
        pinyin: name.pinyin,
        gender: name.gender,
        score: name.score,
      };
      addFavorite(newFav);
      setFavoriteStatus(prev => ({ ...prev, [index]: true }));
      showToast('收藏成功', 'success');
    }
  };

  const toggleCompare = (index: number) => {
    if (compareNames.includes(index)) {
      setCompareNames(compareNames.filter(i => i !== index));
    } else if (compareNames.length < 4) {
      setCompareNames([...compareNames, index]);
    } else {
      showToast('最多对比4个名字', 'warning');
    }
  };

  if (!result) {
    return <PageLoader />;
  }

  const { bazi, nayin, zodiac, hexagram, names } = result;
  const topNames = names.slice(0, 20);

  const handleCompare = () => {
    if (compareNames.length < 2) {
      showToast('请至少选择2个名字进行对比', 'warning');
      return;
    }
    const compareData = compareNames.map(i => topNames[i]);
    setCompareResult(compareData);
    router.push('/compare');
  };

  const resultRef = useRef<HTMLDivElement>(null);

  const exportAsImage = async () => {
    if (!resultRef.current) return;
    
    try {
      const html2canvas = (await import('html2canvas')).default;
      const canvas = await html2canvas(resultRef.current, {
        backgroundColor: '#FDF8F3',
        scale: 2,
      });
      
      const link = document.createElement('a');
      link.download = `名字推荐_${new Date().toISOString().slice(0, 10)}.png`;
      link.href = canvas.toDataURL('image/png');
      link.click();
      showToast('导出成功', 'success');
    } catch (error) {
      console.error('Export error:', error);
      showToast('导出失败', 'error');
    }
  };

  return (
    <main className="min-h-screen py-6 md:py-8 px-3 md:px-4">
      <div className="max-w-4xl mx-auto" ref={resultRef}>
        <div className="flex justify-between items-center mb-4 md:mb-6">
          <button
            onClick={() => router.push('/')}
            className="flex items-center gap-2 text-teal hover:text-crimson transition-colors text-sm md:text-base"
          >
            ← 返回首页
          </button>
          <div className="flex gap-3 text-sm md:text-base">
            <button
              onClick={() => router.push('/history')}
              className="text-teal hover:text-crimson transition-colors"
            >
              📜 历史
            </button>
            <button
              onClick={() => router.push('/favorites')}
              className="text-teal hover:text-crimson transition-colors"
            >
              ❤️ 收藏
            </button>
            <button
              onClick={() => exportAsImage()}
              className="text-teal hover:text-crimson transition-colors"
            >
              📷 导出
            </button>
          </div>
        </div>

        <div className="card mb-6 md:mb-8 animate-fadeInUp">
          <h2 className="font-serif text-xl md:text-2xl text-ink mb-3 md:mb-4">📊 八字分析</h2>
          <div className="grid md:grid-cols-2 gap-4 md:gap-6">
            <div>
              <p className="text-sm text-teal mb-1 md:mb-2">八字</p>
              <p className="font-serif text-lg md:text-xl text-ink">
                {bazi.bazi.year}年 {bazi.bazi.month}月 {bazi.bazi.day}日 {bazi.bazi.hour}时
              </p>
            </div>
            <div>
              <p className="text-sm text-teal mb-1 md:mb-2">日主</p>
              <p className="text-ink">{bazi.rishou} ({bazi.rishou_wuxing}性){bazi.rishou_wuxing === '木' ? '，身强' : '，身弱'}</p>
            </div>
            <div>
              <p className="text-sm text-teal mb-1 md:mb-2">五行分布</p>
              <div className="space-y-1 md:space-y-2">
                {Object.entries(bazi.wuxing).map(([key, value]) => (
                  <div key={key} className="flex items-center gap-2">
                    <span className="w-6 md:w-8 text-xs md:text-sm">
                      {key === 'jin' ? '金' : key === 'mu' ? '木' : key === 'shui' ? '水' : key === 'huo' ? '火' : '土'}
                    </span>
                    <div className="flex-1 h-1.5 md:h-2 bg-warm-white rounded overflow-hidden">
                      <div
                        className="h-full bg-crimson"
                        style={{ width: `${(value / 8) * 100}%` }}
                      />
                    </div>
                    <span className="w-4 text-xs md:text-sm">{value}</span>
                  </div>
                ))}
              </div>
            </div>
            <div>
              <p className="text-sm text-teal mb-1 md:mb-2">喜用神</p>
              <p className="text-base md:text-lg text-gold">{bazi.xiyongshen.join('、')}</p>
              <p className="text-xs md:text-sm text-teal mt-1 md:mt-2">纳音：{nayin}</p>
              <p className="text-xs md:text-sm text-teal">生肖：{zodiac}</p>
            </div>
          </div>
        </div>

        {hexagram && (
          <div className="card mb-6 md:mb-8 animate-fadeInUp" style={{ animationDelay: '0.1s' }}>
            <h2 className="font-serif text-xl md:text-2xl text-ink mb-3 md:mb-4">🔮 易经卦象</h2>
            <div className="flex items-start gap-4 md:gap-6">
              <div className="text-4xl md:text-6xl font-serif">{hexagram.symbol}</div>
              <div>
                <h3 className="text-lg md:text-xl text-ink mb-1 md:mb-2">{hexagram.name}卦</h3>
                <p className="text-xs md:text-sm text-teal mb-1 md:mb-2">{hexagram.gua_ci}</p>
                <p className="text-sm text-ink">{hexagram.interpretation}</p>
              </div>
            </div>
          </div>
        )}

        <div className="mb-6 md:mb-8">
          <div className="flex justify-between items-center mb-3 md:mb-4">
            <h2 className="font-serif text-xl md:text-2xl text-ink animate-fadeInUp" style={{ animationDelay: '0.2s' }}>
              ✨ 推荐名字
            </h2>
            {compareNames.length > 0 && (
              <button
                onClick={handleCompare}
                className="px-3 py-1.5 bg-crimson text-white rounded-lg text-sm hover:bg-crimson/90 transition-colors"
              >
                对比 {compareNames.length} 个名字
              </button>
            )}
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 md:gap-4">
            {topNames.map((name, index) => (
              <div
                key={index}
                className={`card cursor-pointer transition-all duration-200 hover:shadow-lg animate-fadeInUp ${
                  selectedName === index ? 'ring-2 ring-crimson' : ''
                } ${compareNames.includes(index) ? 'ring-2 ring-gold' : ''}`}
                style={{ animationDelay: `${0.3 + index * 0.05}s` }}
                onClick={() => setSelectedName(selectedName === index ? null : index)}
              >
                <div className="flex items-start gap-2 mb-2">
                  <input
                    type="checkbox"
                    checked={compareNames.includes(index)}
                    onChange={(e) => {
                      e.stopPropagation();
                      toggleCompare(index);
                    }}
                    className="mt-1 w-4 h-4 accent-crimson"
                    onClick={(e) => e.stopPropagation()}
                  />
                  <div className="flex-1 flex justify-between items-start">
                    <div>
                      <p className="font-serif text-xl md:text-2xl text-ink">
                        {name.surname}{name.given_name}
                      </p>
                      <p className="text-xs md:text-sm text-teal">{name.pinyin}</p>
                    </div>
                    <div className="flex items-center gap-2">
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          toggleFavorite(name, index);
                        }}
                        className="text-xl md:text-2xl transition-transform hover:scale-110"
                        title={favoriteStatus[index] ? '取消收藏' : '收藏'}
                      >
                        {favoriteStatus[index] ? '❤️' : '🤍'}
                      </button>
                      <div className="text-right">
                        <p className="text-lg md:text-xl text-gold font-medium">{name.score}</p>
                        <p className="text-xs text-teal">分</p>
                      </div>
                    </div>
                  </div>
                </div>
                <div className="flex flex-wrap gap-1 md:gap-2 mt-2">
                  {name.reasons?.map((reason, idx) => (
                    <span
                      key={idx}
                      className="text-xs px-2 py-0.5 bg-warm-white text-teal rounded"
                    >
                      {reason}
                    </span>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </div>

        {selectedName !== null && (
          <div className="card animate-fadeInUp">
            <h3 className="font-serif text-lg md:text-xl text-ink mb-3 md:mb-4">
              📖 名字详解 - {topNames[selectedName].surname}{topNames[selectedName].given_name}
            </h3>
            <div className="space-y-3 md:space-y-4">
              <div>
                <p className="text-sm text-teal mb-1">五行分析</p>
                <p className="text-ink text-sm md:text-base">{topNames[selectedName].wuxing_analysis}</p>
              </div>
              <div>
                <p className="text-sm text-teal mb-1">八字评分</p>
                <p className="text-ink text-sm md:text-base">{topNames[selectedName].bazi_score_detail}</p>
              </div>
              <div>
                <p className="text-sm text-teal mb-1">音韵意境</p>
                <p className="text-ink text-sm md:text-base">{topNames[selectedName].yinyun}</p>
              </div>
              {topNames[selectedName].poetry_source && (
                <div>
                  <p className="text-sm text-teal mb-1">诗词典故</p>
                  <p className="text-ink text-sm md:text-base">
                    出自《{topNames[selectedName].poetry_source}》
                    {topNames[selectedName].poetry_chapter && <>《{topNames[selectedName].poetry_chapter}》</>}
                    {topNames[selectedName].poetry_sentence && <span className="text-teal">：{topNames[selectedName].poetry_sentence}</span>}
                  </p>
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </main>
  );
}
