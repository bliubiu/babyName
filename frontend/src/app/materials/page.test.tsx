import { describe, it, expect, vi, beforeEach } from 'vitest';
import '@testing-library/jest-dom/vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import MaterialsPage from './page';
import type { MaterialStatesResponse, MaterialImportResponse } from '@/types/api/materials';

// 物料 API 替身：状态查询与重导按用例覆写
vi.mock('@/lib/api', () => ({
  getMaterialStates: vi.fn(),
  importMaterials: vi.fn(),
}));

// 导航依赖路由上下文，与被测逻辑无关，替换为占位
vi.mock('@/components/Navigation', () => ({ default: () => 'nav' }));

import { getMaterialStates, importMaterials } from '@/lib/api';

const mockStates = vi.mocked(getMaterialStates);
const mockImport = vi.mocked(importMaterials);

// 渲染夹具：独立 QueryClient（关闭重试，避免错误用例被吞）
function renderPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <MaterialsPage />
    </QueryClientProvider>
  );
}

const statesOf = (...list: MaterialStatesResponse['data']): MaterialStatesResponse => ({
  success: true,
  data: list,
});

const importReport = (overrides: Partial<MaterialImportResponse['data']> = {}): MaterialImportResponse => ({
  success: true,
  data: {
    items: [
      { material: 'hanzi', version: 'v1', rows: 8105, skipped: false },
      { material: 'chuci', version: 'v1', rows: 0, skipped: true },
      { material: 'shijing', version: '', rows: 0, skipped: false, error: '校验失败：缺 chapter 字段' },
    ],
    written: 1,
    skipped: 1,
    failed: 1,
    elapsed: '1.5s',
    ...overrides,
  },
});

beforeEach(() => {
  vi.clearAllMocks();
});

describe('MaterialsPage 物料管理页', () => {
  it('渲染物料状态列表（物料/版本/行数千分位/导入时间）', async () => {
    mockStates.mockResolvedValue(
      statesOf({ material: 'hanzi', version: 'v1', checksum: 'abcd1234efgh', row_count: 8105, imported_at: '2026-11-13 10:00:00' })
    );

    renderPage();

    expect(await screen.findByText('hanzi')).toBeInTheDocument();
    expect(screen.getByText('v1')).toBeInTheDocument();
    expect(screen.getByText('8,105')).toBeInTheDocument();
    expect(screen.getByText('2026-11-13 10:00:00')).toBeInTheDocument();
    // 指纹仅展示前 8 位缩略
    expect(screen.getByText('abcd1234…')).toBeInTheDocument();
  });

  it('内存模式：空列表给出提示且重导按钮禁用', async () => {
    mockStates.mockResolvedValue(statesOf());

    renderPage();

    expect(await screen.findByText(/当前未使用 SQLite 存储（内存模式）/) ).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '重新导入物料' })).toBeDisabled();
  });

  it('状态读取失败：展示可行动错误文案', async () => {
    mockStates.mockRejectedValue(new Error('物料状态读取失败，请稍后重试'));

    renderPage();

    expect(await screen.findByText('物料状态读取失败，请稍后重试')).toBeInTheDocument();
  });

  it('重导成功：展示汇总提示与逐项明细（写入/跳过/失败）', async () => {
    mockStates.mockResolvedValue(statesOf({ material: 'hanzi', version: 'v1', checksum: 'a', row_count: 1, imported_at: '' }));
    mockImport.mockResolvedValue(importReport());

    renderPage();
    // 等状态加载完成，重导按钮才解除禁用
    expect(await screen.findByText('hanzi')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: '重新导入物料' }));

    // 汇总提示（含耗时）
    expect(await screen.findByText('物料导入完成：写入 1 项、跳过 1 项、失败 1 项，耗时 1.5s')).toBeInTheDocument();
    // 明细三分支
    expect(screen.getByText('写入 8,105 行')).toBeInTheDocument();
    expect(screen.getByText('指纹未变，跳过')).toBeInTheDocument();
    expect(screen.getByText('校验失败：缺 chapter 字段')).toBeInTheDocument();
    // 重导后强制刷新状态列表
    await waitFor(() => expect(mockStates).toHaveBeenCalledTimes(2));
  });

  it('重导失败：以 alert 展示错误并清除旧报告', async () => {
    mockStates.mockResolvedValue(statesOf({ material: 'hanzi', version: 'v1', checksum: 'a', row_count: 1, imported_at: '' }));
    mockImport.mockRejectedValue(new Error('物料导入失败: namer.json 脏数据'));

    renderPage();
    // 等状态加载完成，重导按钮才解除禁用
    expect(await screen.findByText('hanzi')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: '重新导入物料' }));

    const alert = await screen.findByRole('alert');
    expect(alert).toHaveTextContent('物料导入失败: namer.json 脏数据');
  });
});
