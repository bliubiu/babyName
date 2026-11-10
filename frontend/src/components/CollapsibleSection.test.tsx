import { describe, it, expect, vi } from 'vitest';
import '@testing-library/jest-dom/vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import CollapsibleSection from './CollapsibleSection';

const getPanel = (button: HTMLElement) =>
  document.getElementById(button.getAttribute('aria-controls') ?? '');

describe('CollapsibleSection 折叠语义与过渡', () => {
  it('默认收起：不挂载内容、显示摘要，面板退出无障碍树且高度为 0', () => {
    render(
      <CollapsibleSection title="紫微斗数" summary="火六局 · 命宫甲子">
        <div>十二宫内容</div>
      </CollapsibleSection>
    );

    const button = screen.getByRole('button', { name: /紫微斗数/ });
    expect(button).toHaveAttribute('aria-expanded', 'false');
    expect(screen.queryByText('十二宫内容')).not.toBeInTheDocument();
    expect(screen.getByText('火六局 · 命宫甲子')).toBeInTheDocument();

    const panel = getPanel(button);
    expect(panel).not.toBeNull();
    expect(panel).toHaveAttribute('data-open', 'false');
    expect(panel).toHaveAttribute('aria-hidden', 'true');
    expect(panel).toHaveStyle({ gridTemplateRows: '0fr', visibility: 'hidden' });
  });

  it('点击展开后内容挂载、摘要隐藏、面板带高度过渡', () => {
    render(
      <CollapsibleSection title="易经卦象" summary="第1卦 · 乾">
        <div>卦辞内容</div>
      </CollapsibleSection>
    );

    const button = screen.getByRole('button', { name: /易经卦象/ });
    fireEvent.click(button);

    expect(button).toHaveAttribute('aria-expanded', 'true');
    expect(screen.getByText('卦辞内容')).toBeInTheDocument();
    expect(screen.queryByText('第1卦 · 乾')).not.toBeInTheDocument();
    const panel = getPanel(button);
    expect(panel).toHaveAttribute('data-open', 'true');
    expect(panel).toHaveAttribute('aria-hidden', 'false');
    expect(panel).toHaveStyle({ gridTemplateRows: '1fr', visibility: 'visible' });
    expect(panel?.style.transition).toContain('grid-template-rows');

    fireEvent.click(button);
    expect(button).toHaveAttribute('aria-expanded', 'false');
    // 默认 unmountOnCollapse=true：收起即卸载，不留可聚焦的隐藏内容
    expect(screen.queryByText('卦辞内容')).not.toBeInTheDocument();
  });

  it('受控模式：状态由 open 决定，点击只上报不自行切换', () => {
    const onOpenChange = vi.fn();
    const { rerender } = render(
      <CollapsibleSection title="八字分析" open={false} onOpenChange={onOpenChange}>
        <div>五行分布</div>
      </CollapsibleSection>
    );

    fireEvent.click(screen.getByRole('button', { name: /八字分析/ }));
    expect(onOpenChange).toHaveBeenCalledWith(true);
    // 父组件尚未更新 open，视图必须保持收起
    expect(screen.queryByText('五行分布')).not.toBeInTheDocument();

    rerender(
      <CollapsibleSection title="八字分析" open onOpenChange={onOpenChange}>
        <div>五行分布</div>
      </CollapsibleSection>
    );
    expect(screen.getByText('五行分布')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /八字分析/ })).toHaveAttribute('aria-expanded', 'true');
  });

  it('defaultOpen 初始展开', () => {
    render(
      <CollapsibleSection title="八字分析" defaultOpen>
        <div>五行分布</div>
      </CollapsibleSection>
    );
    expect(screen.getByText('五行分布')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /八字分析/ })).toHaveAttribute('aria-expanded', 'true');
  });

  it('unmountOnCollapse=false：首次收起仍懒挂载，展开过一次后收起保留内容', () => {
    render(
      <CollapsibleSection title="八字分析" unmountOnCollapse={false}>
        <div>五行分布</div>
      </CollapsibleSection>
    );
    const button = screen.getByRole('button', { name: /八字分析/ });
    // 首屏不下载 lazy chunk
    expect(screen.queryByText('五行分布')).not.toBeInTheDocument();
    expect(getPanel(button)).toHaveAttribute('aria-hidden', 'true');

    fireEvent.click(button);
    expect(screen.getByText('五行分布')).toBeInTheDocument();

    fireEvent.click(button);
    expect(button).toHaveAttribute('aria-expanded', 'false');
    // 保留 DOM：高度过渡可见，子组件内部展开状态不丢
    expect(screen.getByText('五行分布')).toBeInTheDocument();
    expect(getPanel(button)).toHaveAttribute('aria-hidden', 'true');
    expect(getPanel(button)).toHaveStyle({ gridTemplateRows: '0fr' });
  });
});
