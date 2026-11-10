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
import CollapsibleSection from '@/components/CollapsibleSection';

const BaziAnalysis = lazy(() => import('@/components/BaziAnalysis'));
const HexagramDisplay = lazy(() => import('@/components/HexagramDisplay'));
const HexagramDetailDisplay = lazy(() => import('@/components/HexagramDetailDisplay'));
const ZiweiDisplay = lazy(() => import('@/components/ZiweiDisplay'));
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

// 命理详批三个折叠区块的默认展开态。
// 八字是起名依据，保留默认展开（与 docs/06 原型一致）；卦象与紫微长文本默认收起，
// 折叠头仍外露关键结论，避免四块长文把推荐名字挤出首屏。
type AnalysisKey = 'bazi' | 'hexagram' | 'ziwei';
const DEFAULT_ANALYSIS_OPEN: Record<AnalysisKey, boolean> = {
  bazi: true,
  hexagram: false,
  ziwei: false,
};

// 收藏列表的共享空数组常量，配合 useMemo 保证引用稳定（见下方 cachedFavorites）
const EMPTY_FAVORITES: FavoriteData[] = [];

// 浅比较两个收藏态映射，内容相同则视为相等。
// 只有真正变化时才 setState，从根上杜绝 effect → setState → 重渲染的自激循环。
const sameFavoriteStatus = (a: Record<string, boolean>, b: Record<string, boolean>): boolean => {
  const aKeys = Object.keys(a);
  if (aKeys.length !== Object.keys(b).length) return false;
  return aKeys.every(key => a[key] === b[key]);
};

// 从缓存中提取收藏列表（缓存类型为 FavoritesResponse，需取 .data）
const getCachedFavorites = (queryClient: ReturnType<typeof useQueryClient>): FavoriteData[] => {
  const cached = queryClient.getQueryData<FavoritesResponse>(['favorites']);
  return cached?.success && Array.isArray(cached.data) ? cached.data : [];
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
  // 收藏列表必须固定引用。原先写成 `? data ?? []` 会在每次渲染新建空数组，
  // 作为下方 effect 的依赖会让 effect 每渲染都重跑（docs/29 A7）：
  // 收藏查询失败（isError、data 为 undefined）且页面已有结果时形成
  // 「重渲染 → 新 [] → effect 重跑 → setState → 再重渲染」死循环，
  // 最终 "Maximum update depth exceeded" 让 ErrorBoundary 整页白屏。
  // 防御：收藏接口理论上恒返回数组（getFavorites 已归一化），但旧构建产物曾
// 原样返回分页信封 {data,records,total,...} 对象，导致此处 I.some 崩溃
// （I.some is not a function）。加 Array.isArray 兜底，杜绝该静默崩溃。
const cachedFavorites = useMemo<FavoriteData[]>(
    () =>
      favoritesQuery.data?.success && Array.isArray(favoritesQuery.data.data)
        ? favoritesQuery.data.data
        : EMPTY_FAVORITES,
    [favoritesQuery.data]
  );

  const result = generateResult;
  const [filters, setFilters] = useState<NameFilters>({});
  const [showFullReport, setShowFullReport] = useState(false);
  // 折叠态提到页面层集中管理，才能提供「全部展开 / 全部收起」
  const [analysisOpen, setAnalysisOpen] = useState<Record<AnalysisKey, boolean>>(DEFAULT_ANALYSIS_OPEN);
  const allAnalysisOpen = Object.values(analysisOpen).every(Boolean);
  const toggleAllAnalysis = () => {
    const next = !allAnalysisOpen;
    setAnalysisOpen({ bazi: next, hexagram: next, ziwei: next });
  };

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

  // 收藏态由「当前展示的列表 + 收藏缓存」推导。
  // 覆盖 displayList（而非仅 result.names）是为了修 docs/29 B16：点「换一批」
  // 后展示的是 exploreBatch，新名字不在 result.names 里，用 result.names 推导
  // 会让所有心形一律显示未收藏。
  useEffect(() => {
    if (displayList.length === 0) return;

    const status: Record<string, boolean> = {};
    for (const name of displayList) {
      const key = nameKey(name);
      status[key] = cachedFavorites.some(
        f => f.surname === name.surname && f.given_name === name.given_name
      );
    }
    setFavoriteStatus(prev => (sameFavoriteStatus(prev, status) ? prev : status));
  }, [displayList, cachedFavorites]);

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
     // variables 在 mutate 调用前求值：existingId 取自乐观更新前的缓存快照。
     // 此前 mutationFn 内部重读缓存判断 existing，但 onMutate 的乐观更新已先写入
     // 无 id 的乐观记录，导致 existing.id 为空而静默 no-op——首次收藏的
     // POST /favorites 永远不会发出（心形变红、toast 报成功，收藏页却为空）。
     mutationFn: async (v: { name: Name; existingId: string | null }) => {
       const name = v.name;
       if (v.existingId) {
         await deleteFavorite(v.existingId);
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
     onMutate: async (v: { name: Name; existingId: string | null }) => {
       const name = v.name;
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
       const key = nameKey(name.name);
       const isFav = context?.previousFavorites.some(
         f => f.surname === name.name.surname && f.given_name === name.name.given_name
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
    // mutate 前求值缓存快照：existingId 是判断收藏态的唯一依据（见 mutationFn 注释）
    const cached = getCachedFavorites(queryClient);
    const existing = cached.find(
      f => f.surname === name.surname && f.given_name === name.given_name
    );
    toggleFavoriteMutation.mutate({ name, existingId: existing?.id ?? null });
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

  const { bazi, nayin, zodiac, hexagram, hexagram_detail, hexagram_interpretation, tiyong, ziwei, names } = result;

  // 折叠头摘要：把最关键结论外露，收起状态也能一眼扫到
  const hexagramSummary = [
    hexagram ? `第${hexagram.number}卦 ${hexagram.name}` : '',
    hexagram_detail ? `第${hexagram_detail.moving_yao}爻动` : '',
    tiyong?.verdict ? `体用${tiyong.verdict}` : '',
  ].filter(Boolean).join(' · ');
  const ziweiSummary = ziwei
    ? [
        ziwei.five_elements,
        ziwei.ming_gong ? `命宫 ${ziwei.ming_gong}` : '',
        ziwei.soul ? `命主${ziwei.soul}` : '',
      ].filter(Boolean).join(' · ')
    : '';

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

        {/* 结果概览 + 命理详批全局开关（结果页展示优化） */}
        <div className="max-w-lg mx-auto mb-4 flex items-end justify-between gap-3">
          <div className="min-w-0">
            <h1 className="font-serif text-xl md:text-2xl text-ink">起名结果</h1>
            <p className="text-xs text-ink-light/60 mt-1">
              共 {names.length} 个候选 · 下方命理详批可按需展开
            </p>
          </div>
          <button
            type="button"
            onClick={toggleAllAnalysis}
            className="shrink-0 px-3 py-1.5 bg-jade/10 text-jade rounded-xl text-xs hover:bg-jade/20 transition-all duration-200"
          >
            {allAnalysisOpen ? '全部收起' : '全部展开'}
          </button>
        </div>

        {/* 八字分析 */}
        <CollapsibleSection
          title="八字分析"
          summary={`日主 ${bazi.rishou} · 喜用神 ${bazi.xiyongshen.join('、')} · ${zodiac}`}
          open={analysisOpen.bazi}
          onOpenChange={(v) => setAnalysisOpen((prev) => ({ ...prev, bazi: v }))}
        >
          <Suspense fallback={<div className="p-4 text-jade/60 text-sm">加载八字分析...</div>}>
            <BaziAnalysis bazi={bazi} nayin={nayin} zodiac={zodiac} embedded />
          </Suspense>
        </CollapsibleSection>

        {/* 易经卦象 + 卦象详解：本卦经文 / 本互变三卦 / 体用断卦（docs/42 P2-9/10） */}
        {(hexagram || hexagram_detail) && (
          <CollapsibleSection
            title="易经卦象 · 卦象详解"
            summary={hexagramSummary}
            open={analysisOpen.hexagram}
            onOpenChange={(v) => setAnalysisOpen((prev) => ({ ...prev, hexagram: v }))}
          >
            {hexagram && (
              <Suspense fallback={<div className="p-4 text-jade/60 text-sm">加载卦象...</div>}>
                <HexagramDisplay
                  hexagram={hexagram}
                  yaoLines={hexagram_detail?.original_yao_lines}
                  movingYao={hexagram_detail?.moving_yao}
                  interp={hexagram_interpretation}
                  embedded
                />
              </Suspense>
            )}
            {hexagram_detail && (
              <div className="mt-5 border-t border-paper/40 pt-4">
                <Suspense fallback={<div className="p-4 text-jade/60 text-sm">加载卦象详解...</div>}>
                  <HexagramDetailDisplay
                    detail={hexagram_detail}
                    tiyong={tiyong}
                    embedded
                    embeddedTitle="卦象详解"
                  />
                </Suspense>
              </div>
            )}
          </CollapsibleSection>
        )}

        {/* 紫微斗数 */}
        {ziwei && (
          <CollapsibleSection
            title="紫微斗数"
            summary={ziweiSummary}
            open={analysisOpen.ziwei}
            onOpenChange={(v) => setAnalysisOpen((prev) => ({ ...prev, ziwei: v }))}
          >
            <Suspense fallback={<div className="p-4 text-jade/60 text-sm">加载紫微斗数...</div>}>
              <ZiweiDisplay ziwei={ziwei} embedded />
            </Suspense>
          </CollapsibleSection>
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
