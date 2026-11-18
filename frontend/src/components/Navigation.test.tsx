import { describe, it, expect, vi, afterEach } from 'vitest';
import '@testing-library/jest-dom/vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import Navigation from './Navigation';

// 主题开关与被测逻辑无关，替换为占位（避免触碰 Wails runtime）
vi.mock('./ThemeToggle', () => ({ default: () => 'theme-toggle' }));

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
