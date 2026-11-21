import { describe, it, expect, vi, afterEach } from 'vitest';
import '@testing-library/jest-dom/vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import Navigation from './Navigation';

function renderNav() {
  return render(
    <MemoryRouter>
      <Navigation />
    </MemoryRouter>
  );
}

describe('Navigation 物料入口开关', () => {
  afterEach(() => {
    vi.unstubAllEnvs();
  });

  it('默认隐藏物料管理入口', () => {
    vi.stubEnv('VITE_SHOW_MATERIALS', '');
    renderNav();
    expect(screen.queryByRole('button', { name: '物料管理' })).not.toBeInTheDocument();
  });

  it('VITE_SHOW_MATERIALS=true 时显示物料管理入口', () => {
    vi.stubEnv('VITE_SHOW_MATERIALS', 'true');
    renderNav();
    expect(screen.getByRole('button', { name: '物料管理' })).toBeInTheDocument();
  });
});

describe('Navigation JSON 入口开关', () => {
  afterEach(() => {
    vi.unstubAllEnvs();
  });

  it('默认隐藏 JSON 查看器入口（开发者调试工具，终端用户无意义）', () => {
    vi.stubEnv('VITE_SHOW_JSON', '');
    renderNav();
    expect(screen.queryByRole('button', { name: 'JSON 美化查看器' })).not.toBeInTheDocument();
  });

  it('VITE_SHOW_JSON=true 时显示 JSON 查看器入口', () => {
    vi.stubEnv('VITE_SHOW_JSON', 'true');
    renderNav();
    expect(screen.getByRole('button', { name: 'JSON 美化查看器' })).toBeInTheDocument();
  });
});
