'use client';

import { cn } from '@/lib/utils';
import { FavoriteData } from '@/types';
import { IconTrash } from './Icons';

interface FavoriteCardProps {
  favorite: FavoriteData;
  index?: number;
  onDelete: (id: string) => void;
  isDeleting?: boolean;
}

export default function FavoriteCard({
  favorite,
  index,
  onDelete,
  isDeleting = false,
}: FavoriteCardProps) {
  const handleDeleteClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (favorite.id) {
      onDelete(favorite.id);
    }
  };

  return (
    <div
      className={cn(
        'card animate-fade-in-up hover-lift',
      )}
      style={{ animationDelay: index !== undefined ? `${index * 0.05}s` : undefined }}
    >
      <div className="flex justify-between items-start">
        <div>
          <p className="font-serif text-xl md:text-2xl text-ink hover:text-crimson transition-colors">
            {favorite.surname}{favorite.given_name}
          </p>
          <p className="text-xs md:text-sm text-jade mt-0.5">{favorite.pinyin}</p>
          <div className="flex gap-2 mt-2">
            <span className="text-xs px-2 py-0.5 bg-paper/50 text-jade rounded">
              {favorite.gender === 'male' ? '男' : '女'}
            </span>
            <span className="text-xs px-2 py-0.5 bg-gold/15 text-gold-dark rounded">
              {favorite.score}分
            </span>
          </div>
        </div>
        <button
          onClick={handleDeleteClick}
          disabled={isDeleting}
          className="text-ink-light/40 hover:text-seal-red transition-all duration-300 p-2 hover:scale-110 hover:bg-seal-red/5 rounded-full"
          title="删除"
        >
          {isDeleting ? (
            <span className="text-xs animate-pulse">...</span>
          ) : (
            <IconTrash size={18} />
          )}
        </button>
      </div>
    </div>
  );
}