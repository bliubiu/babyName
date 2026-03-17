'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useNameStore } from '@/lib/store';
import { getFavorites, deleteFavorite, FavoriteData } from '@/lib/api';
import { PageLoader, InlineLoader } from '@/components/Spinner';
import { useToast } from '@/components/Toast';

export default function FavoritesPage() {
  const router = useRouter();
  const { favorites, setFavorites, removeFavorite } = useNameStore();
  const { showToast } = useToast();
  const [isLoading, setIsLoading] = useState(true);
  const [deletingId, setDeletingId] = useState<string | null>(null);

  useEffect(() => {
    loadFavorites();
  }, []);

  const loadFavorites = async () => {
    try {
      setIsLoading(true);
      const res = await getFavorites();
      if (res.success) {
        setFavorites(res.data || []);
      }
    } catch (error) {
      console.error('Error loading favorites:', error);
      showToast('加载收藏失败', 'error');
    } finally {
      setIsLoading(false);
    }
  };

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

  return (
    <main className="min-h-screen py-6 md:py-8 px-3 md:px-4">
      <div className="max-w-4xl mx-auto">
        <button
          onClick={() => router.push('/')}
          className="flex items-center gap-2 text-teal hover:text-crimson mb-4 md:mb-6 transition-colors text-sm md:text-base"
        >
          ← 返回首页
        </button>

        <div className="flex justify-between items-center mb-6 md:mb-8">
          <h1 className="font-serif text-2xl md:text-3xl text-ink">❤️ 我的收藏</h1>
          <button
            onClick={() => router.push('/history')}
            className="text-teal hover:text-crimson transition-colors text-sm md:text-base"
          >
            📜 历史记录
          </button>
        </div>

        {favorites.length === 0 ? (
          <div className="card text-center py-12 animate-fadeInUp">
            <p className="text-2xl mb-4">📭</p>
            <p className="text-teal text-sm md:text-base">还没有收藏的名字</p>
            <button
              onClick={() => router.push('/')}
              className="mt-4 px-4 py-2 bg-crimson text-white rounded-lg hover:bg-crimson/90 transition-colors text-sm md:text-base"
            >
              去生成名字
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 md:gap-4">
            {favorites.map((fav, index) => (
              <div
                key={fav.id || index}
                className="card animate-fadeInUp"
                style={{ animationDelay: `${index * 0.05}s` }}
              >
                <div className="flex justify-between items-start">
                  <div>
                    <p className="font-serif text-xl md:text-2xl text-ink">
                      {fav.surname}{fav.given_name}
                    </p>
                    <p className="text-xs md:text-sm text-teal">{fav.pinyin}</p>
                    <div className="flex gap-2 mt-2">
                      <span className="text-xs px-2 py-0.5 bg-warm-white text-teal rounded">
                        {fav.gender === 'male' ? '男' : '女'}
                      </span>
                      <span className="text-xs px-2 py-0.5 bg-gold/20 text-gold rounded">
                        {fav.score}分
                      </span>
                    </div>
                  </div>
                  <button
                    onClick={() => fav.id && handleDelete(fav.id)}
                    disabled={deletingId === fav.id}
                    className="text-teal hover:text-crimson transition-colors p-2"
                    title="删除"
                  >
                    {deletingId === fav.id ? (
                      <span className="text-xs">...</span>
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
