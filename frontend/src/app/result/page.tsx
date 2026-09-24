'use client';

import { useEffect, useState, useMemo, lazy, Suspense } from 'react';
import { useRouter } from 'next/navigation';
import { useNameStore } from '@/lib/store';
import { Name, FavoriteData } from '@/types';
import { ResultPageSkeleton } from '@/components/Skeleton';
import { useToast } from '@/components/Toast';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { saveFavorite, deleteFavorite, exploreNames, getFavorites } from '@/lib/api';
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
  // 存 nameKey（surname:given_name）而不是列表下标
  const [selectedName, setSelectedName] = useState<string | null>(null);
  const [favoriteStatus, setFavoriteStatus] = useState<Record<string, boolean>>({});
  const [compareNames, setCompareNames] = useState<string[]>([]);
  // 探索模式（换一批）：非空时列表展示换上的这一批（与推荐榜零交集）
  const [exploreBatch, setExploreBatch] = useState<Name[] | null>(null);
  const { showToast } = useToast();

  const queryClient = useQueryClient();

  // 结果页必须自己拉一次收藏列表：收藏态图标原先只读 ['favorites'] 缓存，
  // 而缓存只有访问过收藏页才会被填充，导致没访问过的人心形图标一律显示未收藏。
  const favoritesQuery = useQuery({ queryKey: ['favorites'], queryFn: getFavorites });
  const cachedFavorites = favoritesQuery.data?.success ? favoritesQuery.data.data ?? [] : [];

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

  // 探索模式下展示换上的一批；否则展示筛选后的推荐榜
  const displayList = exploreBatch ?? filteredNames;
  const topNames = displayList.slice(0, 20);

  // 选中/对比一律按 nameKey 记录身份，不用数组下标：
  // 筛选或换一批后列表会变短，下标会错位甚至越界成 undefined。
  const nameByKey = (key: string) => displayList.find(n => nameKey(n) === key);

  useEffect(() => {
    const checkFavorites = async () => {
      if (!result?.names) return;

      if (favoritesQuery.isPending) return;
      const status: Record<string, boolean> = {};
      for (const name of result.names) {
        const key = nameKey(name);
        status[key] = cachedFavorites.some(
          f => f.surname === name.surname && f.given_name === name.given_name
        );
      }
      setFavoriteStatus(status);
    };

    checkFavorites();
  }, [result, cachedFavorites, favoritesQuery.isPending]);

  // 探索模式（换一批）：从同一生成会话的候选表中取与榜单零交集的新一批候选
  const exploreMutation = useMutation({
    mutationFn: async () => {
      if (!result?.generation_id) throw new Error('缺少生成会话 ID');
      const data = await exploreNames(result.generation_id, 10);
      if (!data) throw new Error('探索结果为空');
      return data;
    },
    onSuccess: (data) => {
      setExploreBatch(data.names as Name[]);
      setSelectedName(null);
      setCompareNames([]);
      showToast(`已换一批新名字（${data.names.length} 个）`, 'success');
      window.scrollTo({ top: 0, behavior: 'smooth' });
    },
    onError: () => {
      showToast('换一批失败，请稍后重试', 'error');
    },
  });

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

  // 入参仍是 NameCard 给的下标，落库前先换成稳定的 nameKey
  const toggleCompare = (index: number) => {
    const key = topNames[index] ? nameKey(topNames[index]) : null;
    if (!key) return;
    if (compareNames.includes(key)) {
      setCompareNames(compareNames.filter(k => k !== key));
    } else if (compareNames.length < 4) {
      setCompareNames([...compareNames, key]);
    } else {
      showToast('最多对比4个名字', 'warning');
    }
  };

  const handleSelect = (index: number) => {
    const key = topNames[index] ? nameKey(topNames[index]) : null;
    if (!key) return;
    setSelectedName(selectedName === key ? null : key);
  };

  // hydrate 未完成时显示骨架屏，避免 SSR/CSR 不一致
  if (!_hasHydrated) {
    return (
      <main className="min-h-screen py-6 md:py-8 px-4 md:px-6">
        <div className="max-w-2xl mx-auto">
          <ResultPageSkeleton />
        </div>
      </main>
    );
  }

  // 刷新后走这里：store 的 partialize 只持久化了 formData，生成结果不落盘，
  // 所以必须给一个带导航栏和出口的空状态——否则用户被永久困在骨架屏里。
  if (!result) {
    return (
      <main className="min-h-screen py-6 md:py-8 px-4 md:px-6">
        <div className="max-w-2xl mx-auto">
          <Navigation showBackButton={true} showHistory={true} showFavorites={true} />
          <div className="text-center py-16">
            <p className="text-ink font-medium mb-2">没有可展示的起名结果</p>
            <p className="text-jade/60 text-sm mb-6">
              生成结果不会被保存，刷新页面后需要重新生成一次
            </p>
            <button
              onClick={() => router.push('/')}
              className="px-5 py-2 bg-crimson text-white rounded-xl text-sm hover:bg-crimson-light transition-all duration-200"
            >
              回到首页重新起名
            </button>
          </div>
        </div>
      </main>
    );
  }

  const { bazi, nayin, zodiac, hexagram, names } = result;

  const handleCompare = () => {
    if (compareNames.length < 2) {
      showToast('请至少选择2个名字进行对比', 'warning');
      return;
    }
    const compareData = compareNames
      .map(nameByKey)
      .filter((n): n is Name => !!n);
    if (compareData.length < 2) {
      showToast('所选名字已不在当前列表，请重新选择', 'warning');
      return;
    }
    setCompareResult(compareData);
    router.push('/compare');
  };

  // 列表变化后按 key 回查；查不到（已被筛掉）时不渲染详情而不是崩溃
  const selectedEntry = selectedName ? nameByKey(selectedName) : undefined;

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
            <h2 className="section-title">{exploreBatch ? '换一批' : '推荐名字'}</h2>
            <div className="flex items-center gap-2">
              {compareNames.length > 0 && (
                <button
                  onClick={handleCompare}
                  className="px-3.5 py-1.5 bg-crimson text-white rounded-xl text-sm hover:bg-crimson-light transition-all duration-200"
                >
                  对比 {compareNames.length} 个
                </button>
              )}
              {exploreBatch && (
                <button
                  onClick={() => { setExploreBatch(null); setSelectedName(null); setCompareNames([]); }}
                  className="px-3.5 py-1.5 bg-ink/5 text-ink rounded-xl text-sm hover:bg-ink/10 transition-all duration-200"
                >
                  返回推荐榜
                </button>
              )}
              {!exploreBatch && result.generation_id && (
                <button
                  onClick={() => exploreMutation.mutate()}
                  disabled={exploreMutation.isPending}
                  className="px-3.5 py-1.5 bg-jade/10 text-jade rounded-xl text-sm hover:bg-jade/20 transition-all duration-200 disabled:opacity-50"
                >
                  {exploreMutation.isPending ? '换一批中…' : '换一批'}
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
                  isSelected={selectedName === nameKey(name)}
                  isComparing={compareNames.includes(nameKey(name))}
                  isFavorite={favoriteStatus[nameKey(name)] || false}
                  onSelect={handleSelect}
                  onToggleFavorite={toggleFavorite}
                  onToggleCompare={toggleCompare}
                />
              </Suspense>
            ))}
          </div>
        </div>

        {selectedEntry && <NameDetail name={selectedEntry} />}

        {showFullReport && result && (
          <FullReport
            data={result}
            onClose={() => setShowFullReport(false)}
            selectedNameIndex={selectedEntry ? names.findIndex((n: Name) => n.surname === selectedEntry.surname && n.given_name === selectedEntry.given_name) : undefined}
          />
        )}
      </div>
    </main>
  );
}
