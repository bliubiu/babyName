import { describe, it, expect, vi } from 'vitest';
import '@testing-library/jest-dom/vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import CompareTray from './CompareTray';

describe('CompareTray 常驻对比托盘', () => {
  it('未选中任何名字时仍常驻，并保留「开始对比」入口', () => {
    const onCompare = vi.fn();
    render(
      <CompareTray
        selectedKeys={[]}
        onRemove={vi.fn()}
        onClear={vi.fn()}
        onCompare={onCompare}
      />
    );

    expect(screen.getByTestId('compare-tray')).toBeInTheDocument();
    expect(screen.getByText('选 2–4 个名字对比')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: '开始对比' }));
    expect(onCompare).toHaveBeenCalledTimes(1);
  });

  it('已选名字可单个移除、可清空，移除按钮达到 44px 触控目标', () => {
    const onRemove = vi.fn();
    const onClear = vi.fn();
    render(
      <CompareTray
        selectedKeys={['李:沐辰', '李:知遥']}
        onRemove={onRemove}
        onClear={onClear}
        onCompare={vi.fn()}
      />
    );

    expect(screen.getByText('已选 2/4')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /开始对比 2 个/ })).toBeInTheDocument();

    const remove = screen.getByRole('button', { name: '移出对比 李沐辰' });
    expect(remove).toHaveClass('min-h-[44px]', 'min-w-[44px]');
    fireEvent.click(remove);
    expect(onRemove).toHaveBeenCalledWith('李:沐辰');

    fireEvent.click(screen.getByRole('button', { name: '清空' }));
    expect(onClear).toHaveBeenCalledTimes(1);
  });

  it('选中数量跨过 2 个时按钮文案切换为就绪态', () => {
    const props = { onRemove: vi.fn(), onClear: vi.fn(), onCompare: vi.fn() };
    const { rerender } = render(<CompareTray selectedKeys={['李:沐辰']} {...props} />);
    expect(screen.getByRole('button', { name: '开始对比' })).toBeInTheDocument();

    rerender(<CompareTray selectedKeys={['李:沐辰', '李:知遥']} {...props} />);
    expect(screen.getByRole('button', { name: '开始对比 2 个' })).toBeInTheDocument();
  });
});
