'use client';

import { useId, useState, type ReactNode } from 'react';
import { IconChevronDown } from './Icons';

export interface CollapsibleSectionProps {
  /** 区块标题（折叠头常显） */
  title: string;
  /** 收起时显示的一行关键摘要，避免折叠后完全黑盒 */
  summary?: ReactNode;
  /** 非受控模式的初始展开状态，默认收起 */
  defaultOpen?: boolean;
  /** 受控模式：传入后由父组件管理（如「全部展开 / 全部收起」） */
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  /**
   * 折叠时卸载内容（默认 true）。
   * 结果页的分析组件都是 lazy 加载，卸载可避免下载 chunk 与渲染长文本。
   */
  unmountOnCollapse?: boolean;
  children: ReactNode;
}

/**
 * 结果页统一折叠区块（交互原型见 docs/06 §3.1 折叠交互补充）。
 *
 * 原四块命理分析各自全量平铺，且 BaziAnalysis / HexagramDisplay 各写了一套
 * 折叠按钮，样式与语义不一致。此组件统一提供标题、摘要、展开态与无障碍属性，
 * 子组件只需以 embedded 模式输出内容。
 */
export default function CollapsibleSection({
  title,
  summary,
  defaultOpen = false,
  open,
  onOpenChange,
  unmountOnCollapse = true,
  children,
}: CollapsibleSectionProps) {
  const [internalOpen, setInternalOpen] = useState(defaultOpen);
  const isControlled = open !== undefined;
  const isOpen = isControlled ? open : internalOpen;
  const contentId = useId();

  const toggle = () => {
    const next = !isOpen;
    if (!isControlled) {
      setInternalOpen(next);
    }
    onOpenChange?.(next);
  };

  // 折叠时默认不挂载 children；内容容器保留，保证 aria-controls 始终有目标。
  const shouldRenderContent = isOpen || !unmountOnCollapse;

  return (
    <section className="card mb-4 md:mb-6 animate-fade-in-up">
      <h2 className="m-0">
        <button
          type="button"
          onClick={toggle}
          aria-expanded={isOpen}
          aria-controls={contentId}
          className="group w-full flex flex-wrap items-center gap-x-3 gap-y-1 text-left rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-crimson/30"
        >
          <span className="section-title shrink-0">{title}</span>
          {!isOpen && summary && (
            // 小屏整行显示摘要（避免窄卡片里被截没），大屏与标题同行且超长省略
            <span className="order-3 w-full sm:order-none sm:w-auto sm:flex-1 sm:min-w-0 truncate text-xs text-ink-light/60">
              {summary}
            </span>
          )}
          <span className="ml-auto shrink-0 inline-flex items-center gap-1 text-xs text-jade/70 group-hover:text-jade transition-colors">
            <span className="hidden sm:inline">{isOpen ? '收起' : '展开'}</span>
            <IconChevronDown
              size={16}
              className={`transition-transform duration-200 ${isOpen ? 'rotate-180' : ''}`}
            />
          </span>
        </button>
      </h2>
      <div
        id={contentId}
        hidden={!isOpen}
        className={isOpen ? 'mt-4 animate-fade-in' : undefined}
      >
        {shouldRenderContent && children}
      </div>
    </section>
  );
}
