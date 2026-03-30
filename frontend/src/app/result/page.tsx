'use client';

import { useEffect, useState, useRef, lazy, Suspense } from 'react';
import { useRouter } from 'next/navigation';
import { useNameStore } from '@/lib/store';
import { Name, FavoriteData } from '@/types';
import { PageLoader } from '@/components/Spinner';
import { useToast } from '@/components/Toast';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { saveFavorite, deleteFavorite } from '@/lib/api';
import Navigation from '@/components/Navigation';
import { NameFilters } from '@/components/NameFilter';

// 懒加载大型组件
const BaziAnalysis = lazy(() => import('@/components/BaziAnalysis'));
const HexagramDisplay = lazy(() => import('@/components/HexagramDisplay'));
const NameCard = lazy(() => import('@/components/NameCard').then(module => ({
  default: module.default
})));
const NameDetail = lazy(() => import('@/components/NameDetail'));
const ShareButtons = lazy(() => import('@/components/ShareButtons'));
const NameFilter = lazy(() => import('@/components/NameFilter').then(module => ({
  default: module.default
})));
const FullReport = lazy(() => import('@/components/FullReport'));

export default function ResultPage() {
  const router = useRouter();
  const { generateResult, setGenerateResult, setCompareResult, favorites, addFavorite, removeFavorite } = useNameStore();
  const [selectedName, setSelectedName] = useState<number | null>(null);
  const [favoriteStatus, setFavoriteStatus] = useState<Record<number, boolean>>({});
  const [compareNames, setCompareNames] = useState<number[]>([]);
  const { showToast } = useToast();
  
  const queryClient = useQueryClient();

  const result = generateResult;
  const [filters, setFilters] = useState<NameFilters>({});
  const [showFullReport, setShowFullReport] = useState(false);

  const filteredNames = result?.names ? result.names.filter((name: Name, index: number) => {
    if (filters.gender && name.gender !== filters.gender) return false;
    if (filters.wuxing && name.wuxing !== filters.wuxing) return false;
    if (filters.minStrokes && name.strokes < filters.minStrokes) return false;
    if (filters.maxStrokes && name.strokes > filters.maxStrokes) return false;
    return true;
  }) : [];

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

   const toggleFavoriteMutation = useMutation({
     mutationFn: async (name: Name) => {
       const existing = favorites.find(
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
           score: name.score,
         };
         await saveFavorite(newFav);
       }
     },
     onMutate: async (name: Name) => {
       // Cancel any outgoing refetches to avoid overwriting optimistic update
       await queryClient.cancelQueries({ queryKey: ['favorites'] });
       
       // Snapshot the previous value
       const previousFavorites = queryClient.getQueryData<FavoriteData[]>(['favorites']) ?? [];
       
       // Optimistically update the cache
       const existingIndex = previousFavorites.findIndex(
         f => f.surname === name.surname && f.given_name === name.given_name
       );
       
       if (existingIndex >= 0) {
         // Remove from favorites
         queryClient.setQueryData(['favorites'], previousFavorites.filter(
           (_, index) => index !== existingIndex
         ));
       } else {
         // Add to favorites
         const newFav: FavoriteData = {
           surname: name.surname,
           given_name: name.given_name,
           pinyin: name.pinyin,
           gender: name.gender,
           score: name.score,
         };
         queryClient.setQueryData(['favorites'], [...previousFavorites, newFav]);
       }
       
       // Update local state for immediate UI feedback
       setFavoriteStatus(prev => {
         const newStatus = { ...prev };
         Object.keys(newStatus).forEach(key => {
           const index = parseInt(key);
           if (result?.names[index]?.surname === name.surname && result.names[index]?.given_name === name.given_name) {
             newStatus[index] = existingIndex >= 0 ? false : true;
           }
         });
         return newStatus;
       });
       
       // Return context with snapshot
       return { previousFavorites };
     },
     onError: (err, name, context) => {
       // Rollback to previous value on error
       if (context?.previousFavorites) {
         queryClient.setQueryData(['favorites'], context.previousFavorites);
       }
       
       // Reset local state to reflect server state
       if (result?.names) {
         setFavoriteStatus(prev => {
           const newStatus = { ...prev };
           Object.keys(newStatus).forEach(key => {
             const index = parseInt(key);
             if (result?.names[index]?.surname === name.surname && result.names[index]?.given_name === name.given_name) {
               const isFav = context?.previousFavorites.some(
                 f => f.surname === name.surname && f.given_name === name.given_name
               );
               newStatus[index] = !!isFav;
             }
           });
           return newStatus;
         });
       }
       
       showToast('操作失败，请重试', 'error');
     },
     onSuccess: () => {
       // Invalidate and refetch favorites to get latest server state
       queryClient.invalidateQueries({ queryKey: ['favorites'] });
       
       // Show success toast based on whether we added or removed
       // Note: We can't easily determine this here without tracking state, 
       // but the UI will show correct state due to optimistic update
       showToast('操作成功', 'success');
     },
     onSettled: () => {
       // Refetch whether successful or not
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

  if (!result) {
    return <PageLoader />;
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

  const exportAsPDF = async () => {
    if (!resultRef.current) return;
    
    try {
      const html2pdf = (await import('html2pdf.js')).default;
      
      const opt = {
        margin: 10,
        filename: `名字推荐_${new Date().toISOString().slice(0, 10)}.pdf`,
        image: { type: 'jpeg' as const, quality: 0.98 },
        html2canvas: {
          scale: 2,
          backgroundColor: '#FDF8F3'
        },
        jsPDF: { unit: 'mm', format: 'a4', orientation: 'portrait' as const }
      };
      
      await html2pdf().set(opt).from(resultRef.current).save();
      showToast('导出PDF成功', 'success');
    } catch (error) {
      console.error('PDF export error:', error);
      showToast('导出PDF失败', 'error');
    }
  };

  return (
    <main className="min-h-screen py-4 md:py-6 lg:py-8 px-3 md:px-4">
      <div className="max-w-4xl mx-auto" ref={resultRef}>
        <Navigation 
          showBackButton={true}
          showHistory={true}
          showFavorites={true}
          showExport={true}
          onExport={exportAsImage}
          onExportPDF={exportAsPDF}
        />

        <Suspense fallback={<div className="p-4 text-stone-600">加载八字分析...</div>}>
          <BaziAnalysis bazi={bazi} nayin={nayin} zodiac={zodiac} />
        </Suspense>

        {hexagram && (
          <Suspense fallback={<div className="p-4 text-stone-600">加载卦象...</div>}>
            <HexagramDisplay hexagram={hexagram} />
          </Suspense>
        )}

        <Suspense fallback={<div className="p-4 text-stone-600">加载筛选器...</div>}>
          <NameFilter 
            onFilterChange={setFilters}
            totalNames={names.length}
            filteredCount={filteredNames.length}
          />
        </Suspense>

        <div className="mb-4 md:mb-6 lg:mb-8">
          <div className="flex flex-wrap justify-between items-center gap-3 mb-2 md:mb-3 lg:mb-4">
            <h2 className="font-serif text-lg md:text-xl lg:text-2xl text-ink animate-fadeInUp" style={{ animationDelay: '0.2s' }}>
              ✨ 推荐名字
            </h2>
            {compareNames.length > 0 && (
              <button
                onClick={handleCompare}
                className="px-3 py-1.5 bg-crimson text-white rounded-lg text-sm hover:bg-crimson/90 transition-all duration-300 whitespace-nowrap hover-glow animate-pulse-slow"
              >
                对比 {compareNames.length} 个名字
              </button>
            )}
            <button
              onClick={() => setShowFullReport(true)}
              className="px-3 py-1.5 bg-gradient-to-r from-amber-500 to-gold text-white rounded-lg text-sm hover:from-amber-600 hover:to-gold/90 transition-all duration-300 whitespace-nowrap hover-glow"
            >
              📄 查看完整报告
            </button>
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 md:gap-4">
            {topNames.map((name: Name, index: number) => (
              <Suspense key={index} fallback={<div className="p-4 text-stone-600">加载名字卡片...</div>}>
                <NameCard
                  key={index}
                  name={name}
                  index={index}
                  isSelected={selectedName === index}
                  isComparing={compareNames.includes(index)}
                  isFavorite={favoriteStatus[index] || false}
                  onSelect={handleSelect}
                  onToggleFavorite={toggleFavorite}
                  onToggleCompare={toggleCompare}
                />
              </Suspense>
            ))}
          </div>
          <ShareButtons 
            title="宝宝起名大师 - 推荐名字" 
            text={`为${result.bazi.bazi.year}年${result.bazi.bazi.month}月${result.bazi.bazi.day}日出生的${result.names[0].gender === 'male' ? '男孩' : '女孩'}推荐的名字`} 
            url={window.location.href} 
          />
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
