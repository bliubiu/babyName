'use client';

import { useEffect, useState, useMemo, lazy, Suspense } from 'react';
import { useRouter } from 'next/navigation';
import { useNameStore } from '@/lib/store';
import { Name, FavoriteData } from '@/types';
import { ResultPageSkeleton } from '@/components/Skeleton';
import { useToast } from '@/components/Toast';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { saveFavorite, deleteFavorite } from '@/lib/api';
import type { FavoritesResponse } from '@/types/api/favorites';
import Navigation from '@/components/Navigation';
import { NameFilters } from '@/components/NameFilter';

const BaziAnalysis = lazy(() => import('@/components/BaziAnalysis'));
const HexagramDisplay = lazy(() => import('@/components/HexagramDisplay'));
const NameCard = lazy(() => import('@/components/NameCard').then(module => ({
  default: module.default
})));
const NameDetail = lazy(() => import('@/components/NameDetail'));
const NameFilter = lazy(() => import('@/components/NameFilter').then(module => ({
  default: module.default
})));
const FullReport = lazy(() => import('@/components/FullReport'));

// 用 surname:given_name 作为收藏状态键，避免 filter 后 index 错位
const nameKey = (n: Name) => `${n.surname}:${n.given_name}`;

// 从缓存中提取收藏列表（缓存类型为 FavoritesResponse，需取 .data）
const getCachedFavorites = (queryClient: ReturnType<typeof useQueryClient>): FavoriteData[] => {
  const cached = queryClient.getQueryData<FavoritesResponse>(['favorites']);
  return cached?.success ? cached.data ?? [] : [];
};

export default function ResultPage() {
  const router = useRouter();
  const { generateResult, setGenerateResult, setCompareResult, _hasHydrated } = useNameStore();
  const [selectedName, setSelectedName] = useState<number | null>(null);
  const [favoriteStatus, setFavoriteStatus] = useState<Record<string, boolean>>({});
  const [compareNames, setCompareNames] = useState<number[]>([]);
  const { showToast } = useToast();

  const queryClient = useQueryClient();

  const result = generateResult;
  const [filters, setFilters] = useState<NameFilters>({});
  const [showFullReport, setShowFullReport] = useState(false);

  const filteredNames = useMemo(() => result?.names ? result.names.filter((name: Name) => {
    if (filters.gender && name.gender !== filters.gender) return false;
    if (filters.wuxing && name.wuxing !== filters.wuxing) return false;
    if (filters.minStrokes && name.strokes < filters.minStrokes) return false;
    if (filters.maxStrokes && name.strokes > filters.maxStrokes) return false;
    // 人名频率过滤（基于 frequency_score 评分，0-100 分制）
    if (filters.minFrequencyTier && name.frequency_score !== undefined) {
      // 将 frequency_score 映射回 tier（简化逻辑：分数越高 = tier 越高）
      // frequency_score 的 tier 映射：5→100, 4→80, 3→60, 2→40, 1→20, 未收录→50
      const estimatedTier = name.frequency_score >= 90 ? 5
        : name.frequency_score >= 70 ? 4
        : name.frequency_score >= 55 ? 3
        : name.frequency_score >= 35 ? 2
        : name.frequency_score >= 15 ? 1
        : 0;
      if (estimatedTier < filters.minFrequencyTier) return false;
    }
    return true;
  }) : [], [result, filters]);

  useEffect(() => {
    const checkFavorites = async () => {
      if (!result?.names) return;

      const cached = getCachedFavorites(queryClient);
      const status: Record<string, boolean> = {};
      for (const name of result.names) {
        const key = nameKey(name);
        status[key] = cached.some(
          f => f.surname === name.surname && f.given_name === name.given_name
        );
      }
      setFavoriteStatus(status);
    };

    checkFavorites();
  }, [result, queryClient]);

   const toggleFavoriteMutation = useMutation({
     mutationFn: async (name: Name) => {
       const cached = getCachedFavorites(queryClient);
       const existing = cached.find(
         f => f.surname === name.surname && f.given_name === name.given_name
       );

       if (existing) {
         if (existing.id) {
           await deleteFavorite(existing.id);
         }
       } else {
         const newFav = {
          surname: name.surname,
          given_name: name.given_name,
          pinyin: name.pinyin,
          gender: name.gender,
          score: name.total_score ?? name.score,
        };
         await saveFavorite(newFav);
       }
     },
     onMutate: async (name: Name) => {
       await queryClient.cancelQueries({ queryKey: ['favorites'] });
       const previousFavorites = getCachedFavorites(queryClient);
       const existingIndex = previousFavorites.findIndex(
         f => f.surname === name.surname && f.given_name === name.given_name
       );

       if (existingIndex >= 0) {
         // 保持缓存结构为 FavoritesResponse
         queryClient.setQueryData<FavoritesResponse>(['favorites'], (old) => {
           if (!old) return old;
           return {
             ...old,
             data: previousFavorites.filter((_, index) => index !== existingIndex),
           };
         });
       } else {
         const newFav: FavoriteData = {
          surname: name.surname,
          given_name: name.given_name,
          pinyin: name.pinyin,
          gender: name.gender,
          score: name.total_score ?? name.score,
        };
         queryClient.setQueryData<FavoritesResponse>(['favorites'], (old) => {
           if (!old) return old;
           return {
             ...old,
             data: [...previousFavorites, newFav],
           };
         });
       }

       const key = nameKey(name);
       setFavoriteStatus(prev => ({
         ...prev,
         [key]: existingIndex < 0
       }));

       return { previousFavorites };
     },
     onError: (err, name, context) => {
       if (context?.previousFavorites) {
         queryClient.setQueryData<FavoritesResponse>(['favorites'], (old) => {
           if (!old) return old;
           return {
             ...old,
             data: context.previousFavorites,
           };
         });
       }
       const key = nameKey(name);
       const isFav = context?.previousFavorites.some(
         f => f.surname === name.surname && f.given_name === name.given_name
       );
       setFavoriteStatus(prev => ({ ...prev, [key]: !!isFav }));
       showToast('操作失败，请重试', 'error');
     },
     onSuccess: () => {
       queryClient.invalidateQueries({ queryKey: ['favorites'] });
       showToast('操作成功', 'success');
     },
     onSettled: () => {
       queryClient.invalidateQueries({ queryKey: ['favorites'] });
     }
   });

  const toggleFavorite = (name: Name) => {
    toggleFavoriteMutation.mutate(name);
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

  const handleSelect = (index: number) => {
    setSelectedName(selectedName === index ? null : index);
  };

  // hydrate 未完成时显示骨架屏，避免 SSR/CSR 不一致和刷新时短暂 null
  if (!_hasHydrated || !result) {
    return (
      <main className="min-h-screen py-6 md:py-8 px-4 md:px-6">
        <div className="max-w-2xl mx-auto">
          <ResultPageSkeleton />
        </div>
      </main>
    );
  }

  const { bazi, nayin, zodiac, hexagram, names } = result;
  const topNames = filteredNames.slice(0, 20);

  const handleCompare = () => {
    if (compareNames.length < 2) {
      showToast('请至少选择2个名字进行对比', 'warning');
      return;
    }
    const compareData = compareNames.map(i => topNames[i]);
    setCompareResult(compareData);
    router.push('/compare');
  };

  return (
    <main className="min-h-screen py-4 md:py-8 px-4 md:px-6">
      <div className="max-w-4xl mx-auto">
        <Navigation
          showBackButton={true}
          showHistory={true}
          showFavorites={true}
          showExport={false}
        />

        {/* 八字分析 */}
        <Suspense fallback={<div className="p-4 text-jade/60 text-sm">加载八字分析...</div>}>
          <BaziAnalysis bazi={bazi} nayin={nayin} zodiac={zodiac} />
        </Suspense>

        {/* 卦象 */}
        {hexagram && (
          <Suspense fallback={<div className="p-4 text-jade/60 text-sm">加载卦象...</div>}>
            <HexagramDisplay hexagram={hexagram} />
          </Suspense>
        )}

        {/* 竖向名帖 */}
        {topNames.length > 0 && (
          <div className="hidden sm:block mb-6 animate-fade-in-up">
            <div className="consultation-sheet py-6 px-8 corner-decor">
              <div className="flex items-center gap-3 mb-4">
                <span className="seal-badge seal-stamp-sm">名</span>
                <div className="brush-divider flex-1" />
                <span className="text-paper-edge/40 text-xs tracking-[0.5em] font-serif">帖</span>
                <div className="brush-divider flex-1" />
                <span className="seal-badge seal-stamp-sm">帖</span>
              </div>
              <div className="flex justify-center gap-8 md:gap-12 overflow-x-auto py-4" style={{ direction: 'rtl' }}>
                {topNames.slice(0, 6).map((name: Name, idx: number) => (
                  <div
                    key={nameKey(name)}
                    className="name-scroll-wrapper flex-shrink-0"
                    style={{ animationDelay: `${idx * 0.12}s` }}
                  >
                    <div className="flex flex-col items-center">
                      <div
                        className="name-scroll text-3xl md:text-4xl text-ink px-4 py-2 name-scroll-border"
                      >
                        <span className="text-crimson">{name.surname}</span>
                        <span>{name.given_name}</span>
                      </div>
                      <div className="mt-3 text-center">
                        <div className="text-xs text-jade/70">{name.pinyin}</div>
                        <div className="text-xs text-gold mt-0.5 font-medium">{(name.total_score ?? name.score).toFixed(1)}分</div>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        )}

        {/* 筛选器 */}
        <Suspense fallback={<div className="p-4 text-jade/60 text-sm">加载筛选器...</div>}>
          <NameFilter
            onFilterChange={setFilters}
            totalNames={names.length}
            filteredCount={filteredNames.length}
          />
        </Suspense>

        {/* 名字列表 */}
        <div className="mb-6 md:mb-8">
          <div className="flex flex-wrap justify-between items-center gap-3 mb-4">
            <h2 className="section-title">推荐名字</h2>
            <div className="flex items-center gap-2">
              {compareNames.length > 0 && (
                <button
                  onClick={handleCompare}
                  className="px-3.5 py-1.5 bg-crimson text-white rounded-xl text-sm hover:bg-crimson-light transition-all duration-200"
                >
                  对比 {compareNames.length} 个
                </button>
              )}
              <button
                onClick={() => setShowFullReport(true)}
                className="px-3.5 py-1.5 bg-ink/5 text-ink rounded-xl text-sm hover:bg-ink/10 transition-all duration-200"
              >
                完整报告
              </button>
            </div>
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 md:gap-4">
            {topNames.map((name: Name, index: number) => (
              <Suspense key={nameKey(name)} fallback={<div className="p-4 text-jade/60 text-sm">加载名字卡片...</div>}>
                <NameCard
                  name={name}
                  index={index}
                  isSelected={selectedName === index}
                  isComparing={compareNames.includes(index)}
                  isFavorite={favoriteStatus[nameKey(name)] || false}
                  onSelect={handleSelect}
                  onToggleFavorite={toggleFavorite}
                  onToggleCompare={toggleCompare}
                />
              </Suspense>
            ))}
          </div>
        </div>

        {selectedName !== null && <NameDetail name={topNames[selectedName]} />}

        {showFullReport && result && (
          <FullReport
            data={result}
            onClose={() => setShowFullReport(false)}
            selectedNameIndex={selectedName !== null ? names.findIndex((n: Name) => n.surname === topNames[selectedName].surname && n.given_name === topNames[selectedName].given_name) : undefined}
          />
        )}
      </div>
    </main>
  );
}
