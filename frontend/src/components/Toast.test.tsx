import { describe, it, expect, vi } from 'vitest';
import '@testing-library/jest-dom/vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { ToastProvider, useToast } from './Toast';

function Trigger({ action }: { action: () => void }) {
  const { showToast } = useToast();
  return (
    <button onClick={() => showToast('已收藏 李沐辰', 'success', { label: '撤销', onClick: action })}>
      触发
    </button>
  );
}

describe('Toast 可撤销反馈', () => {
  it('点击操作按钮执行动作并关闭提示，不会被整条 toast 的点击抢先关闭', () => {
    const action = vi.fn();
    render(<ToastProvider><Trigger action={action} /></ToastProvider>);

    fireEvent.click(screen.getByRole('button', { name: '触发' }));
    expect(screen.getByText('已收藏 李沐辰')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: '撤销' }));
    expect(action).toHaveBeenCalledTimes(1);
    expect(screen.queryByText('已收藏 李沐辰')).not.toBeInTheDocument();
  });
});
