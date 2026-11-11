'use client';

interface CompareTrayProps {
  /** 已选名字的稳定标识（surname:given_name） */
  selectedKeys: string[];
  onRemove: (key: string) => void;
  onClear: () => void;
  onCompare: () => void;
}

const MAX_COMPARE = 4;
const displayOf = (key: string) => key.split(':').join('');

/**
 * 常驻吸底对比托盘（设计稿 docs/06 §3.1）。
 * 与「选中才出现」的旧实现不同：未选中时也保留「开始对比」入口，
 * 由页面层 onCompare 给出「至少选择 2 个」的提示，避免入口凭空消失。
 * chip 移除按钮按 docs/07 §8.3 扩到 44px 触控目标。
 */
export default function CompareTray({
  selectedKeys,
  onRemove,
  onClear,
  onCompare,
}: CompareTrayProps) {
  const count = selectedKeys.length;
  const ready = count >= 2;

  return (
    <div
      data-testid="compare-tray"
      className="fixed inset-x-0 bottom-0 z-40 border-t border-paper/60 bg-warm-white-light/95 backdrop-blur-sm"
    >
      <div className="max-w-5xl xl:max-w-6xl 2xl:max-w-7xl mx-auto px-4 py-2 flex flex-wrap items-center gap-2">
        <span className="text-xs text-jade/80 mr-1">
          {count > 0 ? `已选 ${count}/${MAX_COMPARE}` : '选 2–4 个名字对比'}
        </span>

        {selectedKeys.map(key => (
          <span
            key={key}
            className="inline-flex items-center gap-1 pl-2.5 pr-0.5 py-0.5 bg-warm-white text-ink border border-paper/60 rounded-xl text-xs"
          >
            {displayOf(key)}
            <button
              type="button"
              onClick={() => onRemove(key)}
              aria-label={`移出对比 ${displayOf(key)}`}
              className="min-w-[44px] min-h-[44px] inline-flex items-center justify-center text-jade hover:text-crimson transition-colors"
            >
              ×
            </button>
          </span>
        ))}

        {count > 0 && (
          <button
            type="button"
            onClick={onClear}
            className="min-h-[44px] px-2 text-xs text-jade/80 hover:text-crimson transition-colors"
          >
            清空
          </button>
        )}

        <button
          type="button"
          onClick={onCompare}
          className={`ml-auto min-h-[44px] px-4 py-1.5 rounded-xl text-sm transition-all duration-200 ${
            ready
              ? 'bg-crimson text-white hover:bg-crimson-light'
              : 'bg-warm-white text-jade/80 border border-paper/60'
          }`}
        >
          {ready ? `开始对比 ${count} 个` : '开始对比'}
        </button>
      </div>
    </div>
  );
}
