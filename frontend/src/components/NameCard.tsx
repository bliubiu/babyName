'use client';

import { cn } from '@/lib/utils';
import { Name } from '@/types';
import { IconHeart, IconHeartFilled, IconChevronDown, IconChevronUp } from './Icons';
import { memo, useEffect, useRef } from 'react';
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

  // 多维度评分颜色映射（须在 dimsFromDetail 之前声明，避免 TDZ）
  const colorMap: Record<string, string> = {
    '五行匹配': 'bg-amber-500',
    '音韵律动': 'bg-sky-500',
    '字义内涵': 'bg-emerald-500',
    '天地人三才': 'bg-violet-400',
    '生肖适配': 'bg-rose-400',
    '新颖度': 'bg-pink-400',
    '诗词共现': 'bg-cyan-400',
    '人名频率': 'bg-indigo-400',
  };

  // 多维度评分配置：优先使用 score_detail，回退到旧字段
  const dimsFromDetail = name.score_detail?.map(d => ({
    key: d.name,
    label: d.name,
    value: d.score,
    color: colorMap[d.name] || 'bg-jade',
  })) || [];

  const dims = dimsFromDetail.length > 0 ? dimsFromDetail : [
    { key: 'wuxing', label: '五行匹配', value: name.wuxing_score ?? 0, color: 'bg-amber-500' },
    { key: 'yinyun', label: '音韵律动', value: name.yinyun_score ?? 0, color: 'bg-sky-500' },
    { key: 'meaning', label: '字义内涵', value: name.meaning_score ?? 0, color: 'bg-emerald-500' },
    { key: 'sancai', label: '天地人三才', value: name.sancai_score ?? 0, color: 'bg-violet-400' },
    { key: 'zodiac', label: '生肖适配', value: name.zodiac_score ?? 0, color: 'bg-rose-400' },
    { key: 'frequency', label: '人名频率', value: name.frequency_score ?? 0, color: 'bg-indigo-400' },
  ];
  const hasDimScores = dims.some(d => d.value > 0);

  return (
    // 容器仅保留鼠标点击的便捷路径；键盘/读屏用户走名字按钮（语义化选择控件），
    // 故此处有意不补键盘监听——不是遗漏（docs/30 §五 NameCard 重构）
    // eslint-disable-next-line jsx-a11y/click-events-have-key-events, jsx-a11y/no-static-element-interactions
    <div
      className={cn(
        'card cursor-pointer transition-all duration-300 animate-fade-in-up',
        isSelected && 'ring-2 ring-crimson/40 shadow-md',
        isComparing && 'ring-2 ring-gold/50 shadow-md',
      )}
      onClick={select}
    >
      {/* 头部：名字与操作 */}
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
          <p className="text-xs text-jade/80 mt-0.5 ml-6">{name.pinyin}</p>
          {/* 粤拼标注：后端始终透传（与粤语评分开关无关），无声拼数据时不渲染 */}
          {name.jyutping && (
            <p className="text-xs text-jade/70 mt-0.5 ml-6 font-mono tracking-wide" title={`粤拼：${name.jyutping}`}>
              粤 {name.jyutping}
            </p>
          )}
        </div>

        <div className="flex items-center gap-1 flex-shrink-0">
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
        {/* 粤语分徽标：仅在开启粤语评分（后端返回非零粤语分）时展示 */}
        {name.cantonese_score !== undefined && name.cantonese_score > 0 && (
          <span
            className="text-xs px-2 py-0.5 bg-sky-500/10 text-sky-700 rounded-full font-medium"
            title={`粤语分（粤语音律 + 粤语谐音）${name.cantonese_score.toFixed(1)}`}
          >
            粤语 {name.cantonese_score.toFixed(0)}
          </span>
        )}
      </div>

      {/* 寓意：设计稿里卡片的核心信息，原先只在展开详情里出现 */}
      {name.meaning && (
        <p className="text-xs text-ink-light/80 mt-3 ml-6">
          寓意：{name.meaning}
        </p>
      )}

      {/* 理由标签 */}
      {name.reasons && name.reasons.length > 0 && (
        <div className="flex flex-wrap gap-1.5 mt-3 ml-6">
          {name.reasons.map((reason, idx) => (
            <span
              key={idx}
              className="text-xs px-2 py-0.5 bg-warm-white/60 text-jade/80 rounded-full"
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
              <span className="text-xs text-jade/80 leading-none">{d.label}</span>
              <div className="w-12 h-1.5 bg-paper/40 rounded-full overflow-hidden">
                <div
                  className={`h-full rounded-full ${d.color} transition-all duration-500`}
                  style={{ width: `${Math.min(100, d.value)}%` }}
                />
              </div>
              <span className="text-xs text-ink-light/70 leading-none font-medium">{d.value.toFixed(0)}</span>
            </div>
          ))}
        </div>
      )}

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
