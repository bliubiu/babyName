'use client';

import { cn } from '@/lib/utils';
import { Name } from '@/types';
import { IconHeart, IconHeartFilled, IconChevronDown, IconChevronUp } from './Icons';
import { useState, memo } from 'react';
import NameDetail from './NameDetail';

interface NameCardProps {
  name: Name;
  index?: number;
  isSelected?: boolean;
  isComparing?: boolean;
  isFavorite?: boolean;
  onSelect?: (index: number) => void;
  onToggleFavorite?: (name: Name) => void;
  onToggleCompare?: (index: number) => void;
}

const wuxingTagClass: Record<string, string> = {
  '金': 'wuxing-tag-jin',
  '木': 'wuxing-tag-mu',
  '水': 'wuxing-tag-shui',
  '火': 'wuxing-tag-huo',
  '土': 'wuxing-tag-tu',
};

function NameCard({
  name,
  index,
  isSelected = false,
  isComparing = false,
  isFavorite = false,
  onSelect,
  onToggleFavorite,
  onToggleCompare,
}: NameCardProps) {
  const [showDetail, setShowDetail] = useState(false);

  const handleCardClick = () => {
    if (onSelect && index !== undefined) {
      onSelect(index);
    }
  };

  const handleFavoriteClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (onToggleFavorite) {
      onToggleFavorite(name);
    }
  };

  const handleCompareClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (onToggleCompare && index !== undefined) {
      onToggleCompare(index);
    }
  };

  const handleToggleDetail = (e: React.MouseEvent) => {
    e.stopPropagation();
    setShowDetail(!showDetail);
  };

  const fullName = name.full_name || `${name.surname}${name.given_name}`;
  const score = Math.min(100, name.total_score ?? name.score);

  // 多维度评分配置
  const dims = [
    { key: 'wuxing', label: '五行', value: name.wuxing_score ?? 0, color: 'bg-amber-500' },
    { key: 'yinyun', label: '音韵', value: name.yinyun_score ?? 0, color: 'bg-sky-500' },
    { key: 'meaning', label: '字义', value: name.meaning_score ?? 0, color: 'bg-emerald-500' },
    { key: 'sancai', label: '天地人三才', value: name.sancai_score ?? 0, color: 'bg-violet-400' },
    { key: 'zodiac', label: '生肖', value: name.zodiac_score ?? 0, color: 'bg-rose-400' },
    { key: 'frequency', label: '人名频率', value: name.frequency_score ?? 0, color: 'bg-indigo-400' },
  ];
  const hasDimScores = dims.some(d => d.value > 0);

  return (
    <div
      className={cn(
        'card cursor-pointer transition-all duration-300 animate-fade-in-up',
        isSelected && 'ring-2 ring-crimson/40 shadow-md',
        isComparing && 'ring-2 ring-gold/50 shadow-md',
      )}
      style={{ animationDelay: index !== undefined ? `${0.3 + index * 0.05}s` : undefined }}
      onClick={handleCardClick}
      onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); handleCardClick(); } }}
      tabIndex={0}
      role="button"
      aria-label={`${fullName}, 评分${score.toFixed(1)}分`}
    >
      {/* 头部：名字与操作 */}
      <div className="flex items-start justify-between gap-2">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            {onToggleCompare && index !== undefined && (
              <input
                type="checkbox"
                checked={isComparing}
                onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                  e.stopPropagation();
                  if (onToggleCompare && index !== undefined) {
                    onToggleCompare(index);
                  }
                }}
                className="w-4 h-4 accent-crimson cursor-pointer flex-shrink-0"
              />
            )}
            <p className="font-serif text-xl md:text-2xl text-ink truncate">
              {fullName}
            </p>
          </div>
          <p className="text-xs text-jade/70 mt-0.5 ml-6">{name.pinyin}</p>
        </div>

        <div className="flex items-center gap-1 flex-shrink-0">
          {onToggleFavorite && (
            <button
              onClick={handleFavoriteClick}
              className={`transition-all duration-200 p-1.5 rounded-full hover:scale-110 ${
                isFavorite ? 'text-seal-red' : 'text-ink-light/30 hover:text-seal-red'
              }`}
              title={isFavorite ? '取消收藏' : '收藏'}
            >
              {isFavorite ? <IconHeartFilled size={18} /> : <IconHeart size={18} />}
            </button>
          )}
          <button
            onClick={handleToggleDetail}
            className="transition-all duration-200 p-1.5 rounded-full text-ink-light/30 hover:text-crimson hover:scale-110"
            title={showDetail ? '收起详情' : '查看详情'}
          >
            {showDetail ? <IconChevronUp size={18} /> : <IconChevronDown size={18} />}
          </button>
        </div>
      </div>

      {/* 标签与评分 */}
      <div className="flex items-center gap-2 mt-3 ml-6">
        <span className="text-xs px-2 py-0.5 bg-warm-white/80 text-jade/80 rounded-full">
          {name.gender === 'male' ? '男' : '女'}
        </span>
        {name.wuxing && (
          <span className={`wuxing-tag ${wuxingTagClass[name.wuxing] || ''}`}>
            {name.wuxing}
          </span>
        )}
        <span className="text-xs px-2 py-0.5 bg-gold/10 text-gold-dark rounded-full font-medium">
          {score.toFixed(1)}分
        </span>
      </div>

      {/* 理由标签 */}
      {name.reasons && name.reasons.length > 0 && (
        <div className="flex flex-wrap gap-1.5 mt-3 ml-6">
          {name.reasons.map((reason, idx) => (
            <span
              key={idx}
              className="text-xs px-2 py-0.5 bg-warm-white/60 text-jade/70 rounded-full"
            >
              {reason}
            </span>
          ))}
        </div>
      )}

      {/* 多维度评分条（仅当后端返回了新评分字段时展示） */}
      {hasDimScores && (
        <div className="flex flex-wrap gap-x-3 gap-y-1 mt-3 ml-6">
          {dims.map(d => (
            <div key={d.key} className="flex items-center gap-1.5">
              <span className="text-[10px] text-jade/60 leading-none">{d.label}</span>
              <div className="w-12 h-1.5 bg-paper/40 rounded-full overflow-hidden">
                <div
                  className={`h-full rounded-full ${d.color} transition-all duration-500`}
                  style={{ width: `${Math.min(100, d.value)}%` }}
                />
              </div>
              <span className="text-[10px] text-ink-light/50 leading-none font-medium">{d.value.toFixed(0)}</span>
            </div>
          ))}
        </div>
      )}

      {/* 展开详情 */}
      {showDetail && (
        <div className="mt-4 pt-4 border-t border-paper/30 animate-fade-in">
          <NameDetail name={name} />
        </div>
      )}
    </div>
  );
}

export default memo(NameCard);
