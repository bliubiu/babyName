'use client';

import { useEffect, useState, useMemo, lazy, Suspense, type SetStateAction } from 'react';
import { useRouter } from 'next/navigation';
import { useNameStore } from '@/lib/store';
import { Name, FavoriteData } from '@/types';
import { ResultPageSkeleton, NameCardSkeleton } from '@/components/Skeleton';
import { useToast } from '@/components/Toast';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  saveFavorite, deleteFavorite, exploreNames, getFavorites, saveHistory,
  generateNamesWithProgress, exportReportPDF, createShareSnapshot,
} from '@/lib/api';
import { buildGenerateRequest } from '@/lib/generateRequest';
import type { FavoritesResponse } from '@/types/api/favorites';
import Navigation from '@/components/Navigation';
import ResultActions from '@/components/ResultActions';
import CompareTray from '@/components/CompareTray';
import type { NameFilters } from '@/components/NameFilter';
import CollapsibleSection from '@/components/CollapsibleSection';

const BaziAnalysis = lazy(() => import('@/components/BaziAnalysis'));
const HexagramDisplay = lazy(() => import('@/components/HexagramDisplay'));
const HexagramDetailDisplay = lazy(() => import('@/components/HexagramDetailDisplay'));
const ZiweiDisplay = lazy(() => import('@/components/ZiweiDisplay'));
const NameCard = lazy(() => import('@/components/NameCard').then(module => ({
  default: module.default
})));
const NameFilter = lazy(() => import('@/components/NameFilter').then(module => ({
  default: module.default
})));
const FullReport = lazy(() => import('@/components/FullReport'));

// 用 surname:given_name 作为收藏状态键，避免 filter 后 index 错位
const nameKey = (n: Name) => `${n.surname}:${n.given_name}`;

// 命理详批三个折叠区块的默认展开态。
// 结果优先：三块一律收起，折叠头仍外露关键结论；用户要看依据再按需展开。
type AnalysisKey = 'bazi' | 'hexagram' | 'ziwei';
const DEFAULT_ANALYSIS_OPEN: Record<AnalysisKey, boolean> = {
  bazi: false,
  hexagram: false,
  ziwei: false,
};

type SortKey = 'score' | 'strokes' | 'wuxing';
const WUXING_ORDER = ['金', '木', '水', '火', '土'];
const PAGE_SIZE = 20;
const scoreOf = (n: Name) => n.total_score ?? n.score ?? 0;

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
  const { formData, generateResult, setGenerateResult, setCompareResult, _hasHydrated } = useNameStore();
  // 存 nameKey（surname:given_name）而不是列表下标；同时是卡片就地展开详情的唯一开关
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
  // 作为下方 effect 的依赖会让 effect 每渲染都重跑（docs/29 A7），最终
  // "Maximum update depth exceeded" 让 ErrorBoundary 整页白屏。
  // 防御：旧构建产物曾原样返回分页信封对象，导致 I.some 崩溃，加 Array.isArray 兜底。
  const cachedFavorites = useMemo<FavoriteData[]>(
    () =>
      favoritesQuery.data?.success && Array.isArray(favoritesQuery.data.data)
        ? favoritesQuery.data.data
        : EMPTY_FAVORITES,
    [favoritesQuery.data]
  );

  const result = generateResult;
  const [filters, setFilters] = useState<NameFilters>({});
  const [sortKey, setSortKey] = useState<SortKey>('score');
  const [visibleCount, setVisibleCount] = useState(PAGE_SIZE);
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

  // 排序 + 分页：原实现硬截 slice(0,20)，顶部却写「共 N 个候选」，
  // 第 21 个之后的名字永远无法到达。这里按排序键重排并按需加载更多。
  const displayList = useMemo(() => {
    const list = [...(exploreBatch ?? filteredNames)];
    if (sortKey === 'strokes') {
      return list.sort((a, b) => (a.strokes ?? 0) - (b.strokes ?? 0) || scoreOf(b) - scoreOf(a));
    }
    if (sortKey === 'wuxing') {
      return list.sort((a, b) => {
        const ai = WUXING_ORDER.indexOf(a.wuxing ?? '');
        const bi = WUXING_ORDER.indexOf(b.wuxing ?? '');
        return (ai < 0 ? 99 : ai) - (bi < 0 ? 99 : bi) || scoreOf(b) - scoreOf(a);
      });
    }
    return list.sort((a, b) => scoreOf(b) - scoreOf(a));
  }, [filteredNames, exploreBatch, sortKey]);

  const topNames = displayList.slice(0, visibleCount);

  // 筛选条件变化即回到第一页（事件驱动，不在 effect 里同步 setState）
  const updateFilters = (next: SetStateAction<NameFilters>) => {
    setFilters(next);
    setVisibleCount(PAGE_SIZE);
  };
  const changeSort = (next: SortKey) => {
    setSortKey(next);
    setVisibleCount(PAGE_SIZE);
  };

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
    // 有 sameFavoriteStatus 守卫：内容不变时返回原引用，不会产生级联渲染。
    // eslint-disable-next-line react-hooks/set-state-in-effect
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
      setVisibleCount(PAGE_SIZE);
      showToast(`已换一批新名字（${data.names.length} 个）`, 'success');
      // 滚回页面顶部会把用户送回命理区；新名字就在列表锚点处
      window.setTimeout(() => {
        document.getElementById('name-list')?.scrollIntoView({ behavior: 'smooth', block: 'start' });
      }, 0);
    },
    onError: () => {
      showToast('换一批失败，请稍后重试', 'error');
    },
  });

  // --- 动作区：导出 PDF / 分享 / 重新起名（docs/06 §3.1 接线）---
  // 报表与分享快照都只从当前结果派生；分享快照刻意不含出生地、长辈姓名等输入信息，
  // 后端 /share 还有一层字段白名单，双保险避免匿名工具的分享链接泄露表单内容。
  const buildReportPayload = () => {
    const baziInfo = result?.bazi;
    const pillars = baziInfo?.bazi
      ? [baziInfo.bazi.year_ganzhi, baziInfo.bazi.month_ganzhi,
         baziInfo.bazi.day_ganzhi, baziInfo.bazi.hour_ganzhi].filter(Boolean).join(' ')
      : '';
    const wx = baziInfo?.wuxing;
    return {
      surname: formData.surname,
      gender: formData.gender,
      birth_date: `${formData.birthYear}-${formData.birthMonth}-${formData.birthDay}`,
      birth_time: `${formData.birthHour}:${formData.birthMinute.toString().padStart(2, '0')}`,
      bazi: pillars,
      wuxing: wx ? `金${wx.jin} 木${wx.mu} 水${wx.shui} 火${wx.huo} 土${wx.tu}` : '',
      xiyongshen: baziInfo?.xiyongshen?.join('、') ?? '',
      names: (result?.names ?? []).slice(0, 30).map((n: Name) => ({
        full_name: n.full_name || `${n.surname}${n.given_name}`,
        pinyin: n.pinyin,
        wuxing: n.wuxing,
        score: scoreOf(n),
        meaning: n.meaning,
      })),
    };
  };

  const buildShareSnapshot = () => ({
    title: `${formData.surname || ''}氏宝宝起名结果`,
    surname: formData.surname,
    gender: formData.gender,
    rishou: result?.bazi?.rishou,
    xiyongshen: result?.bazi?.xiyongshen,
    zodiac: result?.zodiac,
    hexagram_name: result?.hexagram
      ? `第${result.hexagram.number}卦 ${result.hexagram.name}` : undefined,
    five_elements: result?.ziwei?.five_elements,
    total: result?.names?.length ?? 0,
    generated_at: new Date().toISOString(),
    names: (result?.names ?? []).slice(0, 20).map((n: Name) => ({
      full_name: n.full_name || `${n.surname}${n.given_name}`,
      pinyin: n.pinyin,
      wuxing: n.wuxing,
      score: scoreOf(n),
      meaning: n.meaning,
    })),
  });

  const pdfMutation = useMutation({
    mutationFn: (payload: ReturnType<typeof buildReportPayload>) => exportReportPDF(payload),
    onSuccess: (blob) => {
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = `起名报告-${formData.surname || '宝宝'}-${new Date().toISOString().slice(0, 10)}.pdf`;
      document.body.appendChild(link);
      link.click();
      link.remove();
      window.setTimeout(() => URL.revokeObjectURL(url), 0);
      showToast('PDF 报告已下载', 'success');
    },
    onError: (err) => {
      showToast(err instanceof Error ? err.message : '导出 PDF 失败，请稍后重试', 'error');
    },
  });

  const shareMutation = useMutation({
    mutationFn: (snapshot: ReturnType<typeof buildShareSnapshot>) => createShareSnapshot(snapshot),
    onSuccess: ({ url }) => {
      const absolute = `${window.location.origin}${url}`;
      const task = navigator.clipboard?.writeText(absolute);
      if (!task) {
        showToast(`请手动复制分享链接：${absolute}`, 'warning');
        return;
      }
      task.then(() => {
        showToast('分享链接已复制，发给家人朋友即可查看', 'success', {
          label: '打开',
          onClick: () => window.open(absolute, '_blank', 'noopener'),
        });
      }).catch(() => {
        showToast(`复制失败，请手动复制：${absolute}`, 'warning');
      });
    },
    onError: (err) => {
      showToast(err instanceof Error ? err.message : '创建分享失败，请稍后重试', 'error');
    },
  });

  const regenerateMutation = useMutation({
    mutationFn: () => {
      if (!formData.surname.trim()) throw new Error('缺少姓氏');
      // 与首页提交共用同一装配链，避免重新生成时漏传粤语开关/避讳等字段
      return generateNamesWithProgress(buildGenerateRequest(formData));
    },
    onSuccess: (data) => {
      if (!data?.names?.length) {
        showToast('重新生成结果为空，请稍后重试', 'error');
        return;
      }
      setGenerateResult(data);
      saveHistory({
        surname: formData.surname,
        gender: formData.gender,
        birth_date: `${formData.birthYear}-${formData.birthMonth}-${formData.birthDay}`,
        birth_time: `${formData.birthHour}:${formData.birthMinute.toString().padStart(2, '0')}`,
        birth_location: formData.birthLocation,
        results: data,
      }).catch(() => { /* 历史留痕失败不阻断生成 */ });
      setSelectedName(null);
      setCompareNames([]);
      setExploreBatch(null);
      setVisibleCount(PAGE_SIZE);
      setFilters({});
      setSortKey('score');
      showToast(`已按上次条件重新生成（${data.names.length} 个候选）`, 'success');
      window.setTimeout(() => {
        document.getElementById('name-list')?.scrollIntoView({ behavior: 'smooth', block: 'start' });
      }, 0);
    },
    onError: (err) => {
      if (err instanceof Error && err.message === '缺少姓氏') {
        router.push('/');
        return;
      }
      showToast(err instanceof Error ? err.message : '重新生成失败，请稍后重试', 'error');
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
     onSuccess: (_data, v) => {
       queryClient.invalidateQueries({ queryKey: ['favorites'] });
       const fullName = v.name.full_name || `${v.name.surname}${v.name.given_name}`;
       // 收藏/取消收藏都给可撤销的反馈，避免误点心形后只能重找名字
       showToast(v.existingId ? `已取消收藏 ${fullName}` : `已收藏 ${fullName}`, 'success', {
         label: '撤销',
         onClick: () => toggleFavorite(v.name),
       });
     },
     onSettled: () => {
       queryClient.invalidateQueries({ queryKey: ['favorites'] });
     }
   });

  // 函数声明而非 const：上面的 onSuccess 在闭包里引用它，提升后无 TDZ 风险
  function toggleFavorite(name: Name) {
    // mutate 前求值缓存快照：existingId 是判断收藏态的唯一依据（见 mutationFn 注释）
    const cached = getCachedFavorites(queryClient);
    const existing = cached.find(
      f => f.surname === name.surname && f.given_name === name.given_name
    );
    toggleFavoriteMutation.mutate({ name, existingId: existing?.id ?? null });
  }

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
        <div className="max-w-5xl xl:max-w-6xl 2xl:max-w-7xl mx-auto">
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
        <div className="max-w-5xl xl:max-w-6xl 2xl:max-w-7xl mx-auto">
          <Navigation showBackButton={true} showHistory={true} showFavorites={true} />
          <div className="text-center py-16">
            <p className="text-ink font-medium mb-2">没有可展示的起名结果</p>
            <p className="text-jade/80 text-sm mb-6">
              生成结果不落盘，刷新后需重新生成；表单信息仍保存在本机，可一键重来
            </p>
            <div className="flex flex-wrap justify-center gap-2">
              <button
                onClick={() => router.push('/')}
                className="min-h-[44px] px-5 py-2 bg-warm-white border border-paper/70 text-ink rounded-xl text-sm hover:bg-warm-white/70 transition-all duration-200"
              >
                回到首页修改条件
              </button>
              <button
                onClick={() => regenerateMutation.mutate()}
                disabled={!formData.surname || regenerateMutation.isPending}
                className="min-h-[44px] px-5 py-2 bg-crimson text-white rounded-xl text-sm hover:bg-crimson-light transition-all duration-200 disabled:opacity-50"
              >
                {regenerateMutation.isPending ? '重新生成中…' : '用上次输入重新生成'}
              </button>
            </div>
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
  const hasQuickFilters = !!(filters.gender || filters.wuxing || filters.minStrokes || filters.maxStrokes || filters.minFrequencyTier || filters.maxFrequencyTier);

  return (
    <main className="min-h-screen py-4 md:py-8 px-4 md:px-6 pb-28">
      <div className="max-w-5xl xl:max-w-6xl 2xl:max-w-7xl mx-auto">
        <Navigation
          showBackButton={true}
          showHistory={true}
          showFavorites={true}
          showExport={true}
          onExportPDF={() => pdfMutation.mutate(buildReportPayload())}
        />

        {/* 结果概览：关键结论常显，命理长文按需下钻 */}
        <div className="mb-3 flex flex-wrap items-end justify-between gap-3">
          <div className="min-w-0">
            <h1 className="font-serif text-xl md:text-2xl text-ink">起名结果</h1>
            <p className="text-xs text-ink-light/80 mt-1">
              共 {names.length} 个候选 · 已展示 {topNames.length} 个
            </p>
          </div>
          <button
            type="button"
            onClick={() => document.getElementById('analysis-sections')?.scrollIntoView({ behavior: 'smooth', block: 'start' })}
            className="shrink-0 min-h-[44px] px-3 py-1.5 bg-jade/10 text-jade rounded-xl text-xs hover:bg-jade/20 transition-all duration-200"
          >
            查看命理依据 ↓
          </button>
        </div>

        {/* 统一动作区：原型里的导出 PDF / 分享 / 重新起名，全部接前后端 */}
        <div className="mb-4 flex flex-wrap items-center justify-between gap-3 rounded-xl border border-paper/60 bg-warm-white/70 px-3 py-2.5">
          <p className="text-xs text-jade/80">保存、发给家人，或按原条件再生成一批</p>
          <ResultActions
            onExportPDF={() => pdfMutation.mutate(buildReportPayload())}
            onShare={() => shareMutation.mutate(buildShareSnapshot())}
            onRegenerate={() => regenerateMutation.mutate()}
            exporting={pdfMutation.isPending}
            sharing={shareMutation.isPending}
            regenerating={regenerateMutation.isPending}
          />
        </div>
        <div className="max-w-5xl xl:max-w-6xl 2xl:max-w-7xl mx-auto mb-4 flex flex-wrap gap-2">
          <span className="text-xs px-2.5 py-1 bg-warm-white/80 text-jade/80 rounded-full">日主 {bazi.rishou}</span>
          <span className="text-xs px-2.5 py-1 bg-warm-white/80 text-jade/80 rounded-full">喜用神 {bazi.xiyongshen.join('、')}</span>
          <span className="text-xs px-2.5 py-1 bg-warm-white/80 text-jade/80 rounded-full">{zodiac}</span>
          {hexagram && (
            <span className="text-xs px-2.5 py-1 bg-warm-white/80 text-jade/80 rounded-full">第{hexagram.number}卦 {hexagram.name}</span>
          )}
          {tiyong?.verdict && (
            <span className="text-xs px-2.5 py-1 bg-warm-white/80 text-jade/80 rounded-full">体用{tiyong.verdict}</span>
          )}
          {ziwei?.five_elements && (
            <span className="text-xs px-2.5 py-1 bg-warm-white/80 text-jade/80 rounded-full">{ziwei.five_elements}</span>
          )}
        </div>

        {/* 名字列表：结果优先，紧跟概览进入首屏 */}
        <section id="name-list" className="scroll-mt-4 mb-6 md:mb-8">
          <div className="flex flex-wrap justify-between items-center gap-3 mb-4">
            <h2 className="section-title">{exploreBatch ? '换一批' : '推荐名字'}</h2>
            <div className="flex flex-wrap items-center gap-2">
              {exploreBatch && (
                <button
                  onClick={() => { setExploreBatch(null); setSelectedName(null); setCompareNames([]); setVisibleCount(PAGE_SIZE); }}
                  className="min-h-[44px] px-3.5 py-1.5 bg-ink/5 text-ink rounded-xl text-sm hover:bg-ink/10 transition-all duration-200"
                >
                  返回推荐榜
                </button>
              )}
              {!exploreBatch && result.generation_id && (
                <button
                  onClick={() => exploreMutation.mutate()}
                  disabled={exploreMutation.isPending}
                  className="min-h-[44px] px-3.5 py-1.5 bg-jade/10 text-jade rounded-xl text-sm hover:bg-jade/20 transition-all duration-200 disabled:opacity-50"
                >
                  {exploreMutation.isPending ? '换一批中…' : '换一批'}
                </button>
              )}
              <button
                onClick={() => setShowFullReport(true)}
                className="min-h-[44px] px-3.5 py-1.5 bg-ink/5 text-ink rounded-xl text-sm hover:bg-ink/10 transition-all duration-200"
              >
                完整报告
              </button>
            </div>
          </div>

          {/* 筛选 + 排序同一条工具带：首屏少占一行，右侧排序对齐设计稿 */}
          <div className="flex flex-wrap items-center justify-between gap-2 mb-4">
            <div className="flex flex-wrap items-center gap-2">
            <span className="text-xs text-jade/80 mr-1">筛选</span>
            {(['male', 'female'] as const).map(g => (
              <button
                key={g}
                type="button"
                aria-pressed={filters.gender === g}
                onClick={() => updateFilters(prev => ({ ...prev, gender: prev.gender === g ? undefined : g }))}
                className={`min-h-[44px] px-3 py-1.5 rounded-xl text-xs transition-colors ${
                  filters.gender === g ? 'bg-crimson text-white' : 'bg-warm-white text-ink hover:bg-warm-white/80'
                }`}
              >
                {g === 'male' ? '男孩' : '女孩'}
              </button>
            ))}
            <span className="w-px h-5 bg-paper/60 mx-1" aria-hidden="true" />
            {WUXING_ORDER.map(w => (
              <button
                key={w}
                type="button"
                aria-pressed={filters.wuxing === w}
                onClick={() => updateFilters(prev => ({ ...prev, wuxing: prev.wuxing === w ? undefined : w }))}
                className={`min-h-[44px] px-3 py-1.5 rounded-xl text-xs transition-colors ${
                  filters.wuxing === w ? 'bg-crimson text-white' : 'bg-warm-white text-ink hover:bg-warm-white/80'
                }`}
              >
                {w}
              </button>
            ))}
            {hasQuickFilters && (
              <button
                type="button"
                onClick={() => updateFilters({})}
                className="min-h-[44px] px-3 py-1.5 text-xs text-jade hover:text-crimson transition-colors"
              >
                清除全部
              </button>
            )}
            </div>
            <div className="flex items-center gap-2">
              <span className="text-xs text-jade/80">排序</span>
              <select
                aria-label="排序方式"
                value={sortKey}
                onChange={(e) => changeSort(e.target.value as SortKey)}
                className="min-h-[44px] px-3 py-1.5 bg-white text-ink border border-paper/60 rounded-xl text-sm focus:outline-none focus:ring-2 focus:ring-crimson/30"
              >
                <option value="score">综合分 ↓</option>
                <option value="strokes">笔画 ↑</option>
                <option value="wuxing">五行分组</option>
              </select>
            </div>
          </div>

          <Suspense fallback={<div className="p-4 text-jade/80 text-sm">加载筛选器...</div>}>
            <NameFilter
              filters={filters}
              onFilterChange={updateFilters}
              totalNames={names.length}
              filteredCount={filteredNames.length}
            />
          </Suspense>

          {topNames.length === 0 ? (
            <div className="card text-center py-10">
              <p className="text-ink font-medium mb-1">没有符合条件的名字</p>
              <p className="text-sm text-jade/80 mb-4">试试放宽筛选条件，或换一批新名字</p>
              <button
                type="button"
                onClick={() => updateFilters({})}
                className="min-h-[44px] px-4 py-2 bg-crimson text-white rounded-xl text-sm hover:bg-crimson-light transition-all duration-200"
              >
                清除筛选
              </button>
            </div>
          ) : (
            // 列表级单一边界 + 骨架网格：原先 20 张卡各自 Suspense 且带
            // 0.3+index*0.05s 延迟，最晚 1.3s 才出现、逐张弹出造成跳动
            <Suspense fallback={
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 2xl:grid-cols-4 gap-3 md:gap-4">
                {[1, 2, 3, 4, 5, 6].map(i => <NameCardSkeleton key={i} />)}
              </div>
            }>
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 2xl:grid-cols-4 gap-3 md:gap-4">
                {topNames.map((name: Name, index: number) => (
                  <NameCard
                    key={nameKey(name)}
                    name={name}
                    index={index}
                    isSelected={selectedName === nameKey(name)}
                    isComparing={compareNames.includes(nameKey(name))}
                    isFavorite={favoriteStatus[nameKey(name)] || false}
                    onSelect={handleSelect}
                    onToggleFavorite={toggleFavorite}
                    onToggleCompare={toggleCompare}
                  />
                ))}
              </div>
            </Suspense>
          )}

          <div className="flex flex-wrap items-center justify-between gap-2 mt-5">
            {visibleCount < displayList.length ? (
              <button
                type="button"
                onClick={() => setVisibleCount(c => c + PAGE_SIZE)}
                className="min-h-[44px] px-4 py-2 bg-jade/10 text-jade rounded-xl text-sm hover:bg-jade/20 transition-all duration-200"
              >
                加载更多 {topNames.length} / {displayList.length}
              </button>
            ) : (
              <span className="text-xs text-jade/80">已展示全部 {displayList.length} 个候选</span>
            )}
            <button
              type="button"
              onClick={() => document.getElementById('analysis-sections')?.scrollIntoView({ behavior: 'smooth', block: 'start' })}
              className="min-h-[44px] px-2 text-xs text-jade/80 hover:text-crimson transition-colors"
            >
              八字分析 · 卦象详解 · 紫微斗数（默认收起）↓
            </button>
          </div>
        </section>

        {/* 命理依据：三块默认收起，结论已在概览条常显 */}
        <section id="analysis-sections" className="scroll-mt-4">
          <div className="mb-4 flex items-end justify-between gap-3">
            <div className="min-w-0">
              <h2 className="font-serif text-xl md:text-2xl text-ink">命理依据</h2>
              <p className="text-xs text-ink-light/80 mt-1">按需展开，收起时也保留关键结论</p>
            </div>
            <button
              type="button"
              onClick={toggleAllAnalysis}
              className="shrink-0 min-h-[44px] px-3 py-1.5 bg-jade/10 text-jade rounded-xl text-xs hover:bg-jade/20 transition-all duration-200"
            >
              {allAnalysisOpen ? '全部收起' : '全部展开'}
            </button>
          </div>

          {/* 宽屏双列铺开：收起时也把横向空间用起来，第三块跨两列避免半屏空白 */}
          <div className="grid grid-cols-1 xl:grid-cols-2 gap-4 items-start">
          <CollapsibleSection
            className=""
            title="八字分析"
            summary={`日主 ${bazi.rishou} · 喜用神 ${bazi.xiyongshen.join('、')} · ${zodiac}`}
            open={analysisOpen.bazi}
            onOpenChange={(v) => setAnalysisOpen((prev) => ({ ...prev, bazi: v }))}
            unmountOnCollapse={false}
          >
            <Suspense fallback={<div className="p-4 text-jade/80 text-sm">加载八字分析...</div>}>
              <BaziAnalysis bazi={bazi} nayin={nayin} zodiac={zodiac} embedded />
            </Suspense>
          </CollapsibleSection>

          {/* 易经卦象 + 卦象详解：本卦经文 / 本互变三卦 / 体用断卦（docs/42 P2-9/10） */}
          {(hexagram || hexagram_detail) && (
            <CollapsibleSection
              className=""
              title="易经卦象 · 卦象详解"
              summary={hexagramSummary}
              open={analysisOpen.hexagram}
              onOpenChange={(v) => setAnalysisOpen((prev) => ({ ...prev, hexagram: v }))}
              unmountOnCollapse={false}
            >
              {hexagram && (
                <Suspense fallback={<div className="p-4 text-jade/80 text-sm">加载卦象...</div>}>
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
                  <Suspense fallback={<div className="p-4 text-jade/80 text-sm">加载卦象详解...</div>}>
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

          {ziwei && (
            <CollapsibleSection
              className="xl:col-span-2"
              title="紫微斗数"
              summary={ziweiSummary}
              open={analysisOpen.ziwei}
              onOpenChange={(v) => setAnalysisOpen((prev) => ({ ...prev, ziwei: v }))}
              unmountOnCollapse={false}
            >
              <Suspense fallback={<div className="p-4 text-jade/80 text-sm">加载紫微斗数...</div>}>
                <ZiweiDisplay ziwei={ziwei} embedded />
              </Suspense>
            </CollapsibleSection>
          )}
          </div>
        </section>

        {/* 名帖预览：从首屏平铺降为可折叠的次级欣赏入口 */}
        {topNames.length > 0 && (
          <CollapsibleSection
            title="名帖预览"
            summary={`前 ${Math.min(topNames.length, 6)} 名竖排名帖`}
            unmountOnCollapse={false}
          >
            <div className="hidden sm:block">
              <div className="consultation-sheet py-6 px-8 corner-decor">
                <div className="flex items-center gap-3 mb-4">
                  <span className="seal-badge seal-stamp-sm">名</span>
                  <div className="brush-divider flex-1" />
                  <span className="text-paper-edge/70 text-xs tracking-[0.5em] font-serif">帖</span>
                  <div className="brush-divider flex-1" />
                  <span className="seal-badge seal-stamp-sm">帖</span>
                </div>
                <div className="flex justify-center gap-8 md:gap-12 overflow-x-auto py-4" style={{ direction: 'rtl' }}>
                  {topNames.slice(0, 6).map((name: Name) => (
                    <div key={nameKey(name)} className="name-scroll-wrapper flex-shrink-0">
                      <div className="flex flex-col items-center">
                        <div className="name-scroll text-3xl md:text-4xl text-ink px-4 py-2 name-scroll-border">
                          <span className="text-crimson">{name.surname}</span>
                          <span>{name.given_name}</span>
                        </div>
                        <div className="mt-3 text-center">
                          <div className="text-xs text-jade/80">{name.pinyin}</div>
                          <div className="text-xs text-gold mt-0.5 font-medium">{scoreOf(name).toFixed(1)}分</div>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </CollapsibleSection>
        )}

        {/* 常驻吸底对比托盘：未选中也保留「开始对比」入口（设计稿） */}
        <CompareTray
          selectedKeys={compareNames}
          onRemove={(key) => setCompareNames(prev => prev.filter(k => k !== key))}
          onClear={() => setCompareNames([])}
          onCompare={handleCompare}
        />

        {showFullReport && result && (
          <FullReport
            data={result}
            onClose={() => setShowFullReport(false)}
            selectedNameIndex={selectedEntry ? names.findIndex((n: Name) => nameKey(n) === nameKey(selectedEntry)) : undefined}
          />
        )}
      </div>
    </main>
  );
}
