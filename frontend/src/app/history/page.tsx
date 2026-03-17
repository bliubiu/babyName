'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { useNameStore } from '@/lib/store';
import { getHistory, deleteHistory } from '@/lib/api';
import { GenerateResponse } from '@/types';
import { useToast } from '@/components/Toast';
import { PageLoader } from '@/components/Spinner';

export default function HistoryPage() {
  const router = useRouter();
  const { showToast } = useToast();
  const { history, setHistory, removeHistory, isLoading, setLoading } = useNameStore();

  useEffect(() => {
    loadHistory();
  }, []);

  const loadHistory = async () => {
    setLoading(true);
    try {
      const response = await getHistory();
      if (response.success && response.data) {
        setHistory(response.data);
      }
    } catch (error) {
      console.error('Error loading history:', error);
      showToast('加载历史记录失败', 'error');
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await deleteHistory(id);
      removeHistory(id);
      showToast('删除成功', 'success');
    } catch (error) {
      console.error('Error deleting history:', error);
      showToast('删除失败', 'error');
    }
  };

  const handleView = (record: typeof history[0]) => {
    const results = typeof record.results === 'string' 
      ? JSON.parse(record.results) 
      : record.results as GenerateResponse['data'];
    
    useNameStore.getState().setGenerateResult(results);
    router.push('/result');
  };

  if (isLoading) {
    return <PageLoader />;
  }

  return (
    <main className="min-h-screen py-6 md:py-8 px-3 md:px-4">
      <div className="max-w-2xl mx-auto">
        <button
          onClick={() => router.push('/')}
          className="flex items-center gap-2 text-teal hover:text-crimson mb-4 md:mb-6 transition-colors text-sm md:text-base"
        >
          ← 返回首页
        </button>

        <div className="flex justify-between items-center mb-6 md:mb-8">
          <h1 className="font-serif text-2xl md:text-3xl text-ink">📜 历史记录</h1>
          <button
            onClick={() => router.push('/favorites')}
            className="text-teal hover:text-crimson transition-colors text-sm md:text-base"
          >
            ❤️ 我的收藏
          </button>
        </div>

        {history.length === 0 ? (
          <div className="card text-center py-8 md:py-12">
            <p className="text-teal mb-4">暂无历史记录</p>
            <button
              onClick={() => router.push('/')}
              className="btn-primary"
            >
              开始起名
            </button>
          </div>
        ) : (
          <div className="space-y-3 md:space-y-4">
            {history.map((record) => {
              const results = typeof record.results === 'string' 
                ? JSON.parse(record.results) 
                : record.results as GenerateResponse['data'];
              
              return (
                <div key={record.id} className="card">
                  <div className="flex justify-between items-start mb-3 md:mb-4">
                    <div>
                      <p className="text-xs md:text-sm text-teal">
                        {new Date(record.created_at).toLocaleString('zh-CN')}
                      </p>
                      <p className="font-serif text-base md:text-lg text-ink mt-1">
                        {record.surname} · {record.gender === 'male' ? '男' : '女'} · {record.birth_date}
                      </p>
                    </div>
                    <button
                      onClick={() => handleDelete(record.id)}
                      className="text-teal hover:text-crimson text-xs md:text-sm"
                    >
                      删除
                    </button>
                  </div>

                  {results?.names && results.names.length > 0 && (
                    <div className="flex flex-wrap gap-1 md:gap-2">
                      {results.names.slice(0, 5).map((name: { surname: string; given_name: string }, idx: number) => (
                        <span
                          key={idx}
                          className="px-2 md:px-3 py-1 bg-warm-white text-ink text-xs md:text-sm rounded cursor-pointer hover:bg-paper"
                          onClick={() => handleView(record)}
                        >
                          {name.surname}{name.given_name}
                        </span>
                      ))}
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
