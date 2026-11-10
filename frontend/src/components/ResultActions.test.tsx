import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import ResultActions from './ResultActions';

describe('ResultActions', () => {
  it('三个动作按钮均触发各自回调', () => {
    const onExportPDF = vi.fn();
    const onShare = vi.fn();
    const onRegenerate = vi.fn();
    render(
      <ResultActions onExportPDF={onExportPDF} onShare={onShare} onRegenerate={onRegenerate} />
    );
    fireEvent.click(screen.getByRole('button', { name: /导出 PDF/ }));
    fireEvent.click(screen.getByRole('button', { name: /分享/ }));
    fireEvent.click(screen.getByRole('button', { name: /重新起名/ }));
    expect(onExportPDF).toHaveBeenCalledTimes(1);
    expect(onShare).toHaveBeenCalledTimes(1);
    expect(onRegenerate).toHaveBeenCalledTimes(1);
  });

  it('忙碌态禁用按钮并展示进度文案', () => {
    const onExportPDF = vi.fn();
    render(
      <ResultActions
        onExportPDF={onExportPDF}
        onShare={vi.fn()}
        onRegenerate={vi.fn()}
        exporting
        sharing
        regenerating
      />
    );
    expect(screen.getByText('导出中…')).toBeTruthy();
    expect(screen.getByText('生成链接…')).toBeTruthy();
    expect(screen.getByText('重新生成中…')).toBeTruthy();
    expect(screen.getAllByRole('button').every(b => (b as HTMLButtonElement).disabled)).toBe(true);
  });
});
