import { describe, it, expect } from 'vitest';
import '@testing-library/jest-dom/vitest';
import { render, screen } from '@testing-library/react';
import HexagramLines from './HexagramLines';

const QIAN = [7, 7, 7, 7, 7, 7];
const KUN = [8, 8, 8, 8, 8, 8];
const ZHUN = [7, 8, 8, 8, 7, 8]; // 水雷屯

describe('HexagramLines SVG 爻画组件', () => {
  it('渲染 6 行爻位', () => {
    render(<HexagramLines yaoLines={QIAN} />);
    for (let p = 1; p <= 6; p += 1) {
      expect(screen.getByTestId(`yao-${p}`)).toBeInTheDocument();
    }
  });

  it('阳爻单横、阴爻双段（乾 6 矩形 / 坤 12 矩形）', () => {
    const qian = render(<HexagramLines yaoLines={QIAN} />);
    expect(qian.container.querySelectorAll('rect')).toHaveLength(6);
    qian.unmount();
    const kun = render(<HexagramLines yaoLines={KUN} />);
    expect(kun.container.querySelectorAll('rect')).toHaveLength(12);
  });

  it('水雷屯：两阳四阴共 10 个矩形', () => {
    const { container } = render(<HexagramLines yaoLines={ZHUN} />);
    expect(container.querySelectorAll('rect')).toHaveLength(10);
  });

  it('动爻高亮：朱色脉动动画 + 阳爻〇标记', () => {
    render(<HexagramLines yaoLines={ZHUN} movingYao={1} />);
    const moving = screen.getByTestId('yao-1');
    expect(moving).toHaveAttribute('data-moving');
    expect(moving.querySelector('g.animate-pulse')).not.toBeNull();
    expect(moving.querySelector('rect')?.classList.contains('fill-crimson')).toBe(true);
    expect(moving.textContent).toContain('〇');
    // 静爻无标记
    expect(screen.getByTestId('yao-5')).not.toHaveAttribute('data-moving');
  });

  it('阴爻动爻用 × 标记', () => {
    render(<HexagramLines yaoLines={ZHUN} movingYao={2} />);
    expect(screen.getByTestId('yao-2').textContent).toContain('×');
  });

  it('无动爻时不产生标记', () => {
    render(<HexagramLines yaoLines={ZHUN} />);
    expect(screen.queryByText('〇')).not.toBeInTheDocument();
    expect(screen.queryByText('×')).not.toBeInTheDocument();
  });

  it('爻值数量不为 6 时不渲染', () => {
    const { container } = render(<HexagramLines yaoLines={[7, 7, 7]} />);
    expect(container).toBeEmptyDOMElement();
  });

  it('可关闭动爻标记（showMarker=false）', () => {
    render(<HexagramLines yaoLines={ZHUN} movingYao={1} showMarker={false} />);
    expect(screen.queryByText('〇')).not.toBeInTheDocument();
    expect(screen.getByTestId('yao-1')).toHaveAttribute('data-moving');
  });
});
