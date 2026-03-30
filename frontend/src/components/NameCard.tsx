'use client';

import { cn, cardVariants, badgeVariants } from '@/lib/utils';
import { Name, FavoriteData } from '@/types';

interface NameCardProps {
  name: Name | FavoriteData;
  index?: number;
  isSelected?: boolean;
  isComparing?: boolean;
  isFavorite?: boolean;
  onSelect?: (index: number) => void;
  onToggleFavorite?: (name: Name) => void;
  onToggleCompare?: (index: number) => void;
  onDelete?: (id: string) => void;
  isDeleting?: boolean;
}

export default function NameCard({
  name,
  index,
  isSelected = false,
  isComparing = false,
  isFavorite = false,
  onSelect,
  onToggleFavorite,
  onToggleCompare,
  onDelete,
  isDeleting = false
}: NameCardProps) {
  const handleCardClick = () => {
    if (onSelect && index !== undefined) {
      onSelect(index);
    }
  };

  const handleFavoriteClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (onToggleFavorite && 'reasons' in name) {
      onToggleFavorite(name);
    }
  };

  const handleCompareClick = (e: React.MouseEvent | React.ChangeEvent<HTMLInputElement>) => {
    e.stopPropagation();
    if (onToggleCompare && index !== undefined) {
      onToggleCompare(index);
    }
  };

  const handleDeleteClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (onDelete && name.id) {
      onDelete(name.id.toString());
    }
  };

  const cardClasses = cn(
    cardVariants.base,
    isSelected && cardVariants.selected,
    isComparing && cardVariants.comparing
  );

  const genderBadgeClass = badgeVariants.gender;
  const scoreBadgeClass = badgeVariants.score;

  return (
    <div
      className={cardClasses}
      style={{ animationDelay: index !== undefined ? `${0.3 + index * 0.05}s` : undefined }}
      onClick={handleCardClick}
      onKeyPress={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); handleCardClick(); } }}
      tabIndex={0}
      role="button"
      aria-label={`${'full_name' in name && name.full_name ? name.full_name : `${name.surname}${name.given_name}`}, 评分${typeof name.score === 'number' ? name.score.toFixed(1) : parseFloat(name.score).toFixed(1)}分, ${name.gender === 'male' ? '男' : '女'}, ${isFavorite ? '已收藏' : '未收藏'}, ${isComparing ? '已加入对比' : '未加入对比'}, ${isSelected ? '当前选中' : '未选中'}`}
    >
      <div className="flex items-start gap-2 mb-2">
        {onToggleCompare && index !== undefined && (
          <input
            type="checkbox"
            checked={isComparing}
            onChange={handleCompareClick}
            className="mt-1 w-4 h-4 accent-crimson"
            onClick={handleCompareClick}
          />
        )}
        <div className="flex-1 flex justify-between items-start">
          <div>
            <p className="font-serif text-lg md:text-xl lg:text-2xl text-ink transition-all duration-200 hover:text-crimson">
              {'full_name' in name && name.full_name ? name.full_name : `${name.surname}${name.given_name}`}
            </p>
            <p className="text-xs md:text-sm text-teal">{name.pinyin}</p>
            <div className="flex gap-2 mt-2">
              <span className={genderBadgeClass}>
                {name.gender === 'male' ? '男' : '女'}
              </span>
              <span className={scoreBadgeClass}>
                {(typeof name.score === 'number' ? name.score : parseFloat(name.score)).toFixed(2)}分
              </span>
            </div>
          </div>
          <div className="flex items-center gap-2">
            {onToggleFavorite && (
              <button
                onClick={handleFavoriteClick}
                className="text-lg md:text-xl lg:text-2xl transition-transform hover:scale-110"
                title={isFavorite ? '取消收藏' : '收藏'}
              >
                {isFavorite ? '❤️' : '🤍'}
              </button>
            )}
            {onDelete && (
              <button
                onClick={handleDeleteClick}
                disabled={isDeleting}
                className="text-teal hover:text-crimson transition-all duration-300 p-2 hover:scale-110 hover:bg-crimson/10 rounded-full"
                title="删除"
              >
                {isDeleting ? (
                  <span className="text-xs animate-pulse">...</span>
                ) : (
                  '🗑️'
                )}
              </button>
            )}
          </div>
        </div>
      </div>
      {'reasons' in name && name.reasons && name.reasons.length > 0 && (
        <div className="flex flex-wrap gap-1 md:gap-2 mt-2">
          {name.reasons.map((reason, idx) => (
            <span
              key={idx}
              className="text-xs px-2 py-0.5 bg-warm-white text-teal rounded"
            >
              {reason}
            </span>
          ))}
        </div>
      )}
    </div>
  );
}
