'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useNameStore } from '@/lib/store';
import { useFavorites, deleteFavorite, FavoriteData } from '@/lib/swr';
import { PageLoader } from '@/components/Spinner';
import { useToast } from '@/components/Toast';

export default function FavoritesPage() {
  const router = useRouter();
  const { removeFavorite } = useNameStore();
  const { showToast } = useToast();
  const { data: favoritesResponse, error, isLoading } = useFavorites();
  const [deletingId, setDeletingId] = useState<string | null>(null);
  
  const favorites: FavoriteData[] = favoritesResponse?.success ? favoritesResponse.data || [] : [];

  const handleDelete = async (id: string) => {
    try {
      setDeletingId(id);
      await deleteFavorite(id);
      removeFavorite(id);
      showToast('删除成功', 'success');
    } catch (error) {
      console.error('Error deleting favorite:', error);
      showToast('删除失败', 'error');
    } finally {
      setDeletingId(null);
    }
  };

  if (isLoading) {
    return <PageLoader />;
  }
  
  if (error) {
    return (
      <main className="min-h-screen py-6 md:py-8 px-3 md:px-4">
        <div className="max-w-4xl mx-auto">
          <div className="card text-center py-12">
            <p className="text-crimson mb-4">加载收藏失败</p>
            <button
              onClick={() => router.push('/')}
              className="btn-primary"
            >
              返回首页
            </button>
          </div>
        </div>
      </main>
    );
  }

  return (
    <main className="min-h-screen py-6 md:py-8 px-3 md:px-4">
      <div className="max-w-4xl mx-auto">
        <button
          onClick={() => router.push('/')}
          className="flex items-center gap-2 text-teal hover:text-crimson mb-4 md:mb-6 transition-all duration-300 text-sm md:text-base hover-lift animate-fadeInUp"
        >
          ← 返回首页
        </button>

        <div className="flex justify-between items-center mb-6 md:mb-8 animate-fadeInUp" style={{ animationDelay: '0.1s' }}>
          <h1 className="font-serif text-2xl md:text-3xl text-ink animate-slideInLeft">❤️ 我的收藏</h1>
          <button
            onClick={() => router.push('/history')}
            className="text-teal hover:text-crimson transition-all duration-300 text-sm md:text-base hover-lift animate-slideInRight"
          >
            📜 历史记录
          </button>
        </div>

        {favorites.length === 0 ? (
          <div className="card text-center py-12 animate-fadeInUp hover-glow">
            <p className="text-2xl mb-4 animate-pulse-slow">📭</p>
            <p className="text-teal text-sm md:text-base">还没有收藏的名字</p>
            <button
              onClick={() => router.push('/')}
              className="mt-4 px-4 py-2 bg-crimson text-white rounded-lg hover:bg-crimson/90 transition-all duration-300 text-sm md:text-base hover-glow animate-pulse-slow"
            >
              去生成名字
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 md:gap-4">
            {favorites.map((fav, index) => (
              <div
                key={fav.id || index}
                className="card animate-fadeInUp hover-lift hover-glow"
                style={{ animationDelay: `${index * 0.05}s` }}
              >
                <div className="flex justify-between items-start">
                  <div>
                    <p className="font-serif text-xl md:text-2xl text-ink transition-all duration-200 hover:text-crimson">
                      {fav.surname}{fav.given_name}
                    </p>
                    <p className="text-xs md:text-sm text-teal">{fav.pinyin}</p>
                    <div className="flex gap-2 mt-2">
                      <span className="text-xs px-2 py-0.5 bg-warm-white text-teal rounded hover:bg-warm-white/80 transition-colors duration-200">
                        {fav.gender === 'male' ? '男' : '女'}
                      </span>
                      <span className="text-xs px-2 py-0.5 bg-gold/20 text-gold rounded hover:bg-gold/30 transition-colors duration-200">
                        {fav.score}分
                      </span>
                    </div>
                  </div>
                  <button
                    onClick={() => fav.id && handleDelete(fav.id)}
                    disabled={deletingId === fav.id}
                    className="text-teal hover:text-crimson transition-all duration-300 p-2 hover:scale-110 hover:bg-crimson/10 rounded-full"
                    title="删除"
                  >
                    {deletingId === fav.id ? (
                      <span className="text-xs animate-pulse">...</span>
                    ) : (
                      '🗑️'
                    )}
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </main>
  );
}
