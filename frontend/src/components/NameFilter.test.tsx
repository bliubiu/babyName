import { describe, it, expect, vi } from 'vitest';
import '@testing-library/jest-dom/vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import NameFilter from './NameFilter';

describe('NameFilter 受控筛选', () => {
  it('点击性别只上报，视图由父组件传入的 filters 决定', () => {
    const onFilterChange = vi.fn();
    const { rerender } = render(
      <NameFilter filters={{}} onFilterChange={onFilterChange} totalNames={60} filteredCount={60} />
    );

    fireEvent.click(screen.getByRole('button', { name: '展开筛选面板' }));
    fireEvent.click(screen.getByRole('button', { name: /筛选男性名字/ }));
    expect(onFilterChange).toHaveBeenCalledWith({ gender: 'male' });
    // 受控：父组件没更新前，本地不得偷偷改成选中态
    expect(screen.getByRole('button', { name: /筛选男性名字/ })).toHaveAttribute('aria-pressed', 'false');

    rerender(
      <NameFilter filters={{ gender: 'male' }} onFilterChange={onFilterChange} totalNames={60} filteredCount={30} />
    );
    expect(screen.getByRole('button', { name: /筛选男性名字（已选中）/ })).toHaveAttribute('aria-pressed', 'true');
    expect(screen.getByText('(30/60)')).toBeInTheDocument();
  });

  it('高级筛选为全宽紧凑工具条，不再使用居中窄卡片', () => {
    const { container } = render(
      <NameFilter filters={{}} onFilterChange={vi.fn()} totalNames={60} filteredCount={60} />
    );
    const root = container.firstElementChild as HTMLElement;
    expect(root).not.toHaveClass('card');
    expect(root.className).toContain('rounded-xl');
  });

  it('清除所有筛选上报空对象，快捷 chips 与高级面板共享一份条件', () => {
    const onFilterChange = vi.fn();
    render(
      <NameFilter
        filters={{ gender: 'male', wuxing: '水' }}
        onFilterChange={onFilterChange}
        totalNames={60}
        filteredCount={8}
      />
    );

    fireEvent.click(screen.getByRole('button', { name: '展开筛选面板' }));
    fireEvent.click(screen.getByRole('button', { name: '清除所有筛选条件' }));
    expect(onFilterChange).toHaveBeenCalledWith({});
  });
});
