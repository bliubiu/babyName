'use client';

import { cn } from '@/lib/utils';
import { Name } from '@/types';
import { IconHeart, IconHeartFilled, IconChevronDown, IconChevronUp } from './Icons';
import { memo, useEffect, useRef } from 'react';
import NameDetail from './NameDetail';

interface NameCardProps {
  name: Name;
  index?: number;
  /** 排名（0 = 榜首）：用于「荐」印章角标与分数的层级强调 */
  rank?: number;
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
  rank,
  isSelected = false,
  isComparing = false,
  isFavorite = false,
  onSelect,
  onToggleFavorite,
  onToggleCompare,
}: NameCardProps) {
  // 详情改为单一通道：展开态完全由页面层 selectedName 决定。
  // 原先卡片内部另有一份 showDetail，与页面底部 <NameDetail> 并存，
  // 点名字的反馈发生在视口外，还可能同时渲染两份详解。
  const showDetail = isSelected;
  const detailRef = useRef<HTMLDivElement>(null);

  // 就地展开后把详情滚进视口：卡片在长列表中时，反馈不能发生在屏幕外。
  // 收起不滚动；测试环境（happy-dom）可能没有 scrollIntoView，做能力检测。
  useEffect(() => {
    if (!isSelected) return;
    const node = detailRef.current;
    if (node && typeof node.scrollIntoView === 'function') {
      node.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
    }
  }, [isSelected]);

  const select = () => {
    if (onSelect && index !== undefined) onSelect(index);
  };

  const handleNameClick = (e: React.MouseEvent) => {
    // 名字按钮在卡片容器内，不阻止冒泡会让容器 onClick 再触发一次，
    // 而页面层是 toggle 语义，两次调用相互抵消（点名字看似无反应）。
    e.stopPropagation();
    select();
  };

  const handleFavoriteClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (onToggleFavorite) {
      onToggleFavorite(name);
    }
  };

  const handleToggleDetail = (e: React.MouseEvent) => {
    e.stopPropagation();
    select();
  };

  const fullName = name.full_name || `${name.surname}${name.given_name}`;
  const score = Math.min(100, name.total_score ?? name.score);

  return (
    // 容器仅保留鼠标点击的便捷路径；键盘/读屏用户走名字按钮（语义化选择控件），
    // 故此处有意不补键盘监听——不是遗漏（docs/30 §五 NameCard 重构）
    // eslint-disable-next-line jsx-a11y/click-events-have-key-events, jsx-a11y/no-static-element-interactions
    <div
      className={cn(
        'card cursor-pointer transition-all duration-300 animate-fade-in-up relative overflow-hidden',
        isSelected && 'ring-2 ring-crimson/40 shadow-md',
        isComparing && 'ring-2 ring-gold/50 shadow-md',
        rank === 0 && 'border-crimson/30',
      )}
      onClick={select}
    >
      {/* 榜首「荐」印章角标：结果导向——最优解一眼可辨（docs/07 §印章元素） */}
      {rank === 0 && (
        <span
          aria-label="榜首推荐"
          className="absolute top-0 right-0 z-10 w-9 h-9 bg-crimson text-white font-serif text-base leading-none flex items-center justify-center rounded-bl-xl shadow-sm select-none"
          style={{ borderTopRightRadius: '0.75rem' }}
        >
          荐
        </span>
      )}

      {/* 头部：名字与操作。分数升为大号金棕数字——扫视列表时的第一信号 */}
      <div className="flex items-start justify-between gap-2">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            {onToggleCompare && index !== undefined && (
              <label className="relative -m-3.5 p-3.5 inline-flex items-center justify-center cursor-pointer flex-shrink-0">
                <input
                  type="checkbox"
                  checked={isComparing}
                  onClick={(e) => e.stopPropagation()}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                    e.stopPropagation();
                    if (onToggleCompare && index !== undefined) {
                      onToggleCompare(index);
                    }
                  }}
                  aria-label={isComparing ? `将 ${fullName} 移出对比` : `将 ${fullName} 加入对比`}
                  className="w-4 h-4 accent-crimson cursor-pointer"
                />
              </label>
            )}
            {/* 语义化选择控件：容器不再是 role=button（其内嵌 checkbox/按钮
                违反 WAI-ARIA 嵌套交互规则，且容器的 onKeyDown 会 preventDefault
                掉 Space，导致键盘用户无法勾选对比框——docs/30 §五） */}
            <button
              type="button"
              onClick={handleNameClick}
              aria-pressed={isSelected}
              aria-expanded={isSelected}
              aria-label={`${fullName}，评分 ${score.toFixed(1)} 分`}
              className="font-serif text-xl md:text-2xl text-ink truncate hover:text-crimson transition-colors text-left rounded focus:outline-none focus-visible:ring-2 focus-visible:ring-crimson/40"
            >
              {fullName}
            </button>
          </div>
          <p className="text-xs text-jade/80 mt-0.5 ml-6">
            {name.pinyin}
            {name.jyutping && (
              <span className="ml-2 text-jade/60 font-mono tracking-wide" title={`粤拼：${name.jyutping}`}>
                粤 {name.jyutping}
              </span>
            )}
          </p>
        </div>

        <div className="flex items-center gap-1.5 flex-shrink-0 pr-1">
          {/* 大号分数：列表页第一扫视信号（原为小 chip，权重不足） */}
          <span className="flex items-baseline gap-0.5" aria-label={`综合评分 ${score.toFixed(1)} 分`}>
            <span className="font-serif text-2xl leading-none text-gold-dark font-bold">{score.toFixed(1)}</span>
            <span className="text-[10px] text-gold-dark/70">分</span>
          </span>
          {onToggleFavorite && (
            <button
              onClick={handleFavoriteClick}
              aria-label={isFavorite ? `取消收藏 ${fullName}` : `收藏 ${fullName}`}
              className={`transition-all duration-200 p-1.5 rounded-full hover:scale-110 ${
                isFavorite ? 'text-seal-red' : 'text-ink-light/70 hover:text-seal-red'
              }`}
              title={isFavorite ? '取消收藏' : '收藏'}
            >
              <span
                key={isFavorite ? 'filled' : 'outline'}
                className={isFavorite ? 'inline-flex animate-heart-pop' : 'inline-flex'}
              >
                {isFavorite ? <IconHeartFilled size={18} /> : <IconHeart size={18} />}
              </span>
            </button>
          )}
          <button
            onClick={handleToggleDetail}
            aria-label={showDetail ? `收起 ${fullName} 详情` : `查看 ${fullName} 详情`}
            aria-expanded={showDetail}
            className="transition-all duration-200 p-1.5 rounded-full text-ink-light/70 hover:text-crimson hover:scale-110"
            title={showDetail ? '收起详情' : '查看详情'}
          >
            {showDetail ? <IconChevronUp size={18} /> : <IconChevronDown size={18} />}
          </button>
        </div>
      </div>

      {/* 标签行：性别 + 五行 + 推荐理由（最多 3 个，轻量扫视） */}
      <div className="flex items-center flex-wrap gap-1.5 mt-2.5 ml-6">
        <span className="text-xs px-2 py-0.5 bg-warm-white/80 text-jade/80 rounded-full">
          {name.gender === 'male' ? '男' : '女'}
        </span>
        {name.wuxing && (
          <span className={`wuxing-tag ${wuxingTagClass[name.wuxing] || ''}`}>
            {name.wuxing}
          </span>
        )}
        {name.cantonese_score !== undefined && name.cantonese_score > 0 && (
          <span
            className="text-xs px-2 py-0.5 bg-sky-500/10 text-sky-700 rounded-full font-medium"
            title={`粤语分（粤语音律 + 粤语谐音）${name.cantonese_score.toFixed(1)}`}
          >
            粤语 {name.cantonese_score.toFixed(0)}
          </span>
        )}
        {name.reasons?.slice(0, 3).map((reason, idx) => (
          <span
            key={idx}
            className="text-xs px-2 py-0.5 bg-warm-white/60 text-jade/80 rounded-full truncate max-w-[7.5rem]"
            title={reason}
          >
            {reason}
          </span>
        ))}
      </div>

      {/* 寓意摘要：两行截断——考据长文属于点击展开后的详情（NameDetail），
          列表页保持扫视密度（docs/44 结果导向优化） */}
      {name.meaning && (
        <p className="text-xs text-ink-light/80 mt-2.5 ml-6 leading-relaxed line-clamp-2">
          寓意：{name.meaning}
        </p>
      )}

      {/* 八维评分条已从卡片常态移除：NameDetail 展开态含完整评分分解（含依据文字），
          列表页重复渲染 8 条进度条使卡片高度近 3 倍、网格参差（docs/44 结果导向优化） */}

      {/* 就地展开详情（结果页唯一详情通道） */}
      {showDetail && (
        <div ref={detailRef} className="mt-4 pt-4 border-t border-paper/30 animate-fade-in">
          <NameDetail name={name} />
        </div>
      )}
    </div>
  );
}

export default memo(NameCard);
