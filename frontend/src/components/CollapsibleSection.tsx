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
   * 结果页传 false：首次收起仍不挂载（省 lazy chunk），但展开过一次后
   * 保留 DOM，既让收起有高度过渡，也保住子组件内部的展开状态。
   */
  unmountOnCollapse?: boolean;
  /** 外层 section 附加类；宽屏网格布局里传 '' 可去掉默认的纵向外边距 */
  className?: string;
  children: ReactNode;
}

/**
 * 结果页统一折叠区块（交互原型见 docs/06 §3.1）。
 *
 * 展开/收起用 grid-template-rows 0fr↔1fr 做高度过渡，替代原先 hidden 硬切；
 * 收起态用 visibility:hidden + aria-hidden 让内容退出 Tab 顺序与无障碍树，
 * 且延迟到过渡结束才隐藏，避免动画被瞬间裁掉。
 * 系统开启「减弱动态效果」时，globals.css 的 reduced-motion 规则会接管过渡时长。
 */
export default function CollapsibleSection({
  title,
  summary,
  defaultOpen = false,
  open,
  onOpenChange,
  unmountOnCollapse = true,
  className = 'mb-4 md:mb-6',
  children,
}: CollapsibleSectionProps) {
  const [internalOpen, setInternalOpen] = useState(defaultOpen);
  const isControlled = open !== undefined;
  const isOpen = isControlled ? open : internalOpen;
  const contentId = useId();
  // 懒挂载：首屏收起的区块不下载 lazy chunk；展开过一次后才允许常驻。
  // 用 React 官方「props 变化时调整 state」模式在渲染期同步，替代 effect 里
  // 同步 setState（react-hooks/set-state-in-effect）：prevOpen 守卫保证只在
  // open 真正翻转时触发一次重渲染。
  const [prevOpen, setPrevOpen] = useState(isOpen);
  const [hasOpened, setHasOpened] = useState(isOpen);
  if (isOpen !== prevOpen) {
    setPrevOpen(isOpen);
    if (isOpen) setHasOpened(true);
  }

  const toggle = () => {
    const next = !isOpen;
    if (!isControlled) {
      setInternalOpen(next);
    }
    onOpenChange?.(next);
  };

  const shouldRenderContent = isOpen || (!unmountOnCollapse && hasOpened);

  return (
    <section className={`card animate-fade-in-up ${className}`}>
      <h2 className="m-0">
        <button
          type="button"
          onClick={toggle}
          aria-expanded={isOpen}
          aria-controls={contentId}
          className="group w-full min-h-[44px] flex flex-wrap items-center gap-x-3 gap-y-1 text-left rounded-lg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-crimson/30"
        >
          <span className="section-title shrink-0">{title}</span>
          {!isOpen && summary && (
            // 小屏整行显示摘要（避免窄卡片里被截没），大屏与标题同行且超长省略
            <span className="order-3 w-full sm:order-none sm:w-auto sm:flex-1 sm:min-w-0 truncate text-xs text-ink-light/80">
              {summary}
            </span>
          )}
          <span className="ml-auto shrink-0 inline-flex items-center gap-1 text-xs text-jade/80 group-hover:text-jade transition-colors">
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
        data-open={isOpen ? 'true' : 'false'}
        aria-hidden={!isOpen}
        className="grid"
        style={{
          gridTemplateRows: isOpen ? '1fr' : '0fr',
          visibility: isOpen ? 'visible' : 'hidden',
          transition: isOpen
            ? 'grid-template-rows 280ms ease-out, visibility 0s'
            : 'grid-template-rows 280ms ease-out, visibility 0s 280ms',
        }}
      >
        <div className="overflow-hidden">
          <div className={isOpen ? 'mt-4 animate-fade-in' : 'mt-4'}>
            {shouldRenderContent && children}
          </div>
        </div>
      </div>
    </section>
  );
}
