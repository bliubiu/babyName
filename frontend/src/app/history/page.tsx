'use client';

import { useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { GenerateResponse } from '@/types';
import { getHistory, deleteHistory } from '@/lib/api';
import { useNameStore } from '@/lib/store';
import { useQueryClient } from '@tanstack/react-query';
import { useToast } from '@/components/Toast';
import { HistoryListSkeleton } from '@/components/Skeleton';
import Navigation from '@/components/Navigation';
import { IconTrash } from '@/components/Icons';

export default function HistoryPage() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const { showToast } = useToast();
  const { setGenerateResult } = useNameStore();
  const { data: historyResponse, error, isLoading } = useQuery({
    queryKey: ['history'],
    queryFn: getHistory,
  });

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const rawData: any = historyResponse?.success ? historyResponse.data : null;
  const history: Array<{
    id: string;
    surname: string;
    gender: 'male' | 'female';
    birth_date: string;
    birth_time: string;
    birth_location: string;
    results: string | GenerateResponse['data'];
    created_at: string;
  }> = Array.isArray(rawData) ? rawData : (rawData?.records ?? []);

  const handleDelete = async (id: string) => {
    try {
      await deleteHistory(id);
      queryClient.invalidateQueries({ queryKey: ['history'] });
      showToast('删除成功', 'success');
    } catch {
      showToast('删除失败', 'error');
    }
  };

  const handleView = (record: typeof history[0]) => {
    try {
      const results = typeof record.results === 'string'
        ? JSON.parse(record.results)
        : record.results as GenerateResponse['data'];
      setGenerateResult(results);
      router.push('/result');
    } catch {
      showToast('数据解析失败', 'error');
    }
  };

  if (isLoading) {
    return (
      <main className="min-h-screen py-6 md:py-8 px-4 md:px-6 cloud-bg">
        <div className="max-w-2xl mx-auto">
          <HistoryListSkeleton count={5} />
        </div>
      </main>
    );
  }

  if (error) {
    return (
      <main className="min-h-screen py-6 md:py-8 px-4 md:px-6 cloud-bg">
        <div className="max-w-2xl mx-auto">
          <Navigation showBackButton={true} showHistory={true} showFavorites={true} />
          <div className="consultation-sheet text-center py-12 corner-decor">
            <p className="text-crimson mb-4">加载历史记录失败</p>
            <button onClick={() => router.push('/')} className="btn-primary">返回首页</button>
          </div>
        </div>
      </main>
    );
  }

  return (
    <main className="min-h-screen py-6 md:py-8 px-4 md:px-6 cloud-bg">
      <div className="max-w-2xl mx-auto">
        <Navigation showBackButton={true} showHistory={true} showFavorites={true} />

        {/* 标题 */}
        <div className="text-center mb-6 md:mb-8 animate-fade-in-up">
          <h1 className="font-serif text-2xl md:text-3xl text-ink ink-calligraphy">历史记录</h1>
          <div className="flex items-center justify-center gap-4 mt-3">
            <div className="brush-divider max-w-[60px] flex-1" />
            <span className="text-paper-edge/40 text-xs tracking-[0.5em] font-serif">往昔</span>
            <div className="brush-divider max-w-[60px] flex-1" />
          </div>
        </div>

        {history.length === 0 ? (
          <div className="consultation-sheet text-center py-16 animate-fade-in-up corner-decor">
            <div className="flex justify-center mb-4">
              <svg className="w-12 h-12 text-paper-edge/40" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
            </div>
            <p className="text-jade text-sm tracking-wider">暂无历史记录</p>
            <p className="text-ink-light/40 text-xs mt-2">每次起名结果会自动保存在这里</p>
            <button onClick={() => router.push('/')} className="mt-6 btn-primary">开始起名</button>
          </div>
        ) : (
          <div className="space-y-3 md:space-y-4">
            {history.map((record, index) => {
              let results: GenerateResponse['data'] | null = null;
              try {
                results = typeof record.results === 'string'
                  ? JSON.parse(record.results)
                  : record.results as GenerateResponse['data'];
              } catch {
                results = null;
              }

              return (
                <div
                  key={record.id}
                  className="card animate-fade-in-up group cursor-pointer hover:shadow-md transition-all"
                  style={{ animationDelay: `${index * 0.05}s` }}
                  onClick={() => handleView(record)}
                >
                  <div className="flex justify-between items-start mb-3">
                    <div>
                      <p className="text-xs text-jade/70">
                        {record.created_at ? new Date(record.created_at).toLocaleString('zh-CN') : ''}
                      </p>
                      <p className="font-serif text-base md:text-lg text-ink mt-1">
                        {record.surname} · {record.gender === 'male' ? '男' : '女'} · {record.birth_date}
                      </p>
                    </div>
                    <button
                      onClick={(e) => { e.stopPropagation(); handleDelete(record.id); }}
                      className="p-1.5 rounded-full text-ink-light/30 hover:text-crimson transition-all opacity-0 group-hover:opacity-100"
                    >
                      <IconTrash size={16} />
                    </button>
                  </div>

                  {results?.names && results.names.length > 0 && (
                    <div className="flex flex-wrap gap-1.5">
                      {results.names.slice(0, 5).map((name, idx) => (
                        <span
                          key={idx}
                          className="px-2.5 py-1 bg-warm-white/80 text-ink text-xs rounded-full hover:bg-gold/10 transition-colors"
                        >
                          {name.surname}{name.given_name}
                        </span>
                      ))}
                      {results.names.length > 5 && (
                        <span className="px-2.5 py-1 text-jade/50 text-xs">
                          +{results.names.length - 5}
                        </span>
                      )}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>
    </main>
  );
}
