import { describe, it, expect, vi } from 'vitest';
import '@testing-library/jest-dom/vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import CollapsibleSection from './CollapsibleSection';

describe('CollapsibleSection 折叠语义', () => {
  it('默认收起：不挂载内容、显示摘要，且 aria-controls 指向隐藏面板', () => {
    render(
      <CollapsibleSection title="紫微斗数" summary="火六局 · 命宫甲子">
        <div>十二宫内容</div>
      </CollapsibleSection>
    );

    const button = screen.getByRole('button', { name: /紫微斗数/ });
    expect(button).toHaveAttribute('aria-expanded', 'false');
    expect(screen.queryByText('十二宫内容')).not.toBeInTheDocument();
    expect(screen.getByText('火六局 · 命宫甲子')).toBeInTheDocument();

    const panel = document.getElementById(button.getAttribute('aria-controls') ?? '');
    expect(panel).not.toBeNull();
    expect(panel).toHaveAttribute('hidden');
  });

  it('点击展开后内容挂载、摘要隐藏、展开态翻转', () => {
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
    const panel = document.getElementById(button.getAttribute('aria-controls') ?? '');
    expect(panel).not.toHaveAttribute('hidden');

    fireEvent.click(button);
    expect(button).toHaveAttribute('aria-expanded', 'false');
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

  it('unmountOnCollapse=false：收起时保留内容但面板隐藏', () => {
    render(
      <CollapsibleSection title="八字分析" unmountOnCollapse={false}>
        <div>五行分布</div>
      </CollapsibleSection>
    );
    const button = screen.getByRole('button', { name: /八字分析/ });
    expect(button).toHaveAttribute('aria-expanded', 'false');
    expect(screen.getByText('五行分布')).toBeInTheDocument();
    const panel = document.getElementById(button.getAttribute('aria-controls') ?? '');
    expect(panel).toHaveAttribute('hidden');
  });
});
