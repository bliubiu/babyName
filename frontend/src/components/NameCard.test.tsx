import { describe, it, expect, vi } from 'vitest';
import '@testing-library/jest-dom/vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import type { Name } from '@/types';
import NameCard from './NameCard';

// NameDetail 依赖大量后端字段，这里只验证「详情是否被渲染、渲染几份」
vi.mock('./NameDetail', () => ({ default: () => '名字详解内容' }));

const name = {
  surname: '李',
  given_name: '沐辰',
  full_name: '李沐辰',
  pinyin: 'lǐ mù chén',
  gender: 'male',
  wuxing: '水',
  score: 92.5,
  strokes: 20,
} as Name;

describe('NameCard 单一详情通道', () => {
  it('点击名字按钮只上报一次 onSelect（容器冒泡不得把 toggle 抵消）', () => {
    const onSelect = vi.fn();
    render(<NameCard name={name} index={0} onSelect={onSelect} />);

    fireEvent.click(screen.getByRole('button', { name: /^李沐辰，评分/ }));
    expect(onSelect).toHaveBeenCalledTimes(1);
    expect(onSelect).toHaveBeenCalledWith(0);
  });

  it('展开态完全由外部 isSelected 决定，卡片内只有一份详情', () => {
    const { rerender } = render(<NameCard name={name} index={0} onSelect={vi.fn()} isSelected={false} />);
    expect(screen.queryByText('名字详解内容')).not.toBeInTheDocument();

    rerender(<NameCard name={name} index={0} onSelect={vi.fn()} isSelected />);
    expect(screen.getAllByText('名字详解内容')).toHaveLength(1);
    expect(screen.getByRole('button', { name: '收起 李沐辰 详情' })).toHaveAttribute('aria-expanded', 'true');
  });

  it('详情箭头与卡片空白都走同一个 onSelect 通道', () => {
    const onSelect = vi.fn();
    const { container } = render(<NameCard name={name} index={0} onSelect={onSelect} />);

    fireEvent.click(screen.getByRole('button', { name: '查看 李沐辰 详情' }));
    expect(onSelect).toHaveBeenCalledTimes(1);

    fireEvent.click(container.querySelector('.card') as HTMLElement);
    expect(onSelect).toHaveBeenCalledTimes(2);
  });

  it('展开详情后把详情滚进视口（反馈不落在屏幕外）', () => {
    const scrollSpy = vi.fn();
    Object.defineProperty(Element.prototype, 'scrollIntoView', {
      configurable: true,
      value: scrollSpy,
    });

    const { rerender } = render(<NameCard name={name} index={0} onSelect={vi.fn()} isSelected={false} />);
    expect(scrollSpy).not.toHaveBeenCalled();

    rerender(<NameCard name={name} index={0} onSelect={vi.fn()} isSelected />);
    expect(scrollSpy).toHaveBeenCalledWith({ behavior: 'smooth', block: 'nearest' });
  });

  it('收藏态心形带填充回弹动效', () => {
    // 心形按钮只在传入 onToggleFavorite 时渲染，夹具必须带上
    const onToggleFavorite = vi.fn();
    const { container, rerender } = render(
      <NameCard name={name} index={0} isFavorite={false} onToggleFavorite={onToggleFavorite} />
    );
    expect(container.querySelector('.animate-heart-pop')).toBeNull();

    rerender(<NameCard name={name} index={0} isFavorite onToggleFavorite={onToggleFavorite} />);
    expect(container.querySelector('.animate-heart-pop')).not.toBeNull();
  });

  it('收藏与对比操作不会连带切换详情', () => {
    const onSelect = vi.fn();
    const onToggleFavorite = vi.fn();
    const onToggleCompare = vi.fn();
    render(
      <NameCard
        name={name}
        index={0}
        onSelect={onSelect}
        onToggleFavorite={onToggleFavorite}
        onToggleCompare={onToggleCompare}
      />
    );

    fireEvent.click(screen.getByRole('button', { name: '收藏 李沐辰' }));
    expect(onToggleFavorite).toHaveBeenCalledWith(name);
    fireEvent.click(screen.getByRole('checkbox'));
    expect(onToggleCompare).toHaveBeenCalledWith(0);
    expect(onSelect).not.toHaveBeenCalled();
  });

  it('卡片常显寓意文字（设计稿核心信息）', () => {
    render(<NameCard name={{ ...name, meaning: '沐泽承辰' }} index={0} />);
    expect(screen.getByText('寓意：沐泽承辰')).toBeInTheDocument();
  });
});
