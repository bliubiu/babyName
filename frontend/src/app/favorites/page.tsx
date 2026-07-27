'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useQuery } from '@tanstack/react-query';
import { getFavorites, deleteFavorite } from '@/lib/api';
import type { FavoriteData } from '@/types';
import { useQueryClient } from '@tanstack/react-query';
import { FavoritesListSkeleton } from '@/components/Skeleton';
import { useToast } from '@/components/Toast';
import Navigation from '@/components/Navigation';
import { IconHeart, IconHeartFilled, IconTrash } from '@/components/Icons';



export default function FavoritesPage() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const { showToast } = useToast();
  const { data: favoritesResponse, error, isLoading } = useQuery({
    queryKey: ['favorites'],
    queryFn: getFavorites,
  });
  const [deletingId, setDeletingId] = useState<string | null>(null);

  const favorites: FavoriteData[] = favoritesResponse?.success ? favoritesResponse.data || [] : [];

  const handleDelete = async (id: string) => {
    try {
      setDeletingId(id);
      await deleteFavorite(id);
      queryClient.invalidateQueries({ queryKey: ['favorites'] });
      showToast('删除成功', 'success');
    } catch {
      showToast('删除失败', 'error');
    } finally {
      setDeletingId(null);
    }
  };

  if (isLoading) {
    return (
      <main className="min-h-screen py-6 md:py-8 px-4 md:px-6 cloud-bg">
        <div className="max-w-4xl mx-auto">
          <FavoritesListSkeleton count={6} />
        </div>
      </main>
    );
  }

  if (error) {
    return (
      <main className="min-h-screen py-6 md:py-8 px-4 md:px-6 cloud-bg">
        <div className="max-w-4xl mx-auto">
          <Navigation showBackButton={true} showHistory={true} showFavorites={true} />
          <div className="consultation-sheet text-center py-12 corner-decor">
            <p className="text-crimson mb-4">加载收藏失败</p>
            <button onClick={() => router.push('/')} className="btn-primary">返回首页</button>
          </div>
        </div>
      </main>
    );
  }

  return (
    <main className="min-h-screen py-6 md:py-8 px-4 md:px-6 cloud-bg">
      <div className="max-w-4xl mx-auto">
        <Navigation showBackButton={true} showHistory={true} showFavorites={true} />

        {/* 标题 */}
        <div className="text-center mb-6 md:mb-8 animate-fade-in-up">
          <h1 className="font-serif text-2xl md:text-3xl text-ink ink-calligraphy">我的收藏</h1>
          <div className="flex items-center justify-center gap-4 mt-3">
            <div className="brush-divider max-w-[60px] flex-1" />
            <span className="text-paper-edge/40 text-xs tracking-[0.5em] font-serif">珍藏</span>
            <div className="brush-divider max-w-[60px] flex-1" />
          </div>
        </div>

        {favorites.length === 0 ? (
          <div className="consultation-sheet text-center py-16 animate-fade-in-up corner-decor">
            <div className="flex justify-center mb-4">
              <IconHeart size={48} className="text-paper-edge/40" />
            </div>
            <p className="text-jade text-sm tracking-wider">尚未收藏名字</p>
            <p className="text-ink-light/40 text-xs mt-2">在结果页点击心形即可收藏</p>
            <button
              onClick={() => router.push('/')}
              className="mt-6 btn-primary"
            >
              去生成名字
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3 md:gap-4">
            {favorites.map((fav, index) => (
              <div
                key={fav.id || index}
                className="card animate-fade-in-up group"
                style={{ animationDelay: `${index * 0.05}s` }}
              >
                <div className="flex items-start justify-between gap-2">
                  <div className="flex-1 min-w-0">
                    <p className="font-serif text-xl text-ink truncate">
                      {fav.surname}{fav.given_name}
                    </p>
                    <p className="text-xs text-jade/70 mt-0.5">{fav.pinyin}</p>
                  </div>
                  <button
                    onClick={() => fav.id && handleDelete(fav.id)}
                    disabled={deletingId === fav.id}
                    className="p-1.5 rounded-full text-ink-light/30 hover:text-crimson transition-all hover:scale-110 opacity-0 group-hover:opacity-100"
                    title="删除收藏"
                  >
                    <IconTrash size={16} />
                  </button>
                </div>

                <div className="flex items-center gap-2 mt-3">
                  <span className="text-xs px-2 py-0.5 bg-warm-white/80 text-jade/80 rounded-full">
                    {fav.gender === 'male' ? '男' : '女'}
                  </span>
                  <span className="text-xs px-2 py-0.5 bg-gold/10 text-gold-dark rounded-full font-medium">
                    {typeof fav.score === 'number' ? fav.score.toFixed(1) : fav.score}分
                  </span>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </main>
  );
}
