import { describe, it, expect } from 'vitest';
import '@testing-library/jest-dom/vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import HexagramDisplay from './HexagramDisplay';
import type { Hexagram, HexagramInterpretation } from '@/types';

// 丰 #55：震上离下，初爻动；爻辞 6 条 + 小象 6 条
const feng: Hexagram = {
  id: 55,
  name: '丰',
  number: 55,
  symbol: '☳☲',
  upper_trigram: 4,
  lower_trigram: 3,
  gua_ci: '丰：亨，王假之，勿忧，宜日中。',
  xiang_ci: '雷电皆至，丰；君子以折狱致刑。',
  yao_ci: [
    '遇其配主，虽旬无咎，往有尚。',
    '丰其蔀，日中见斗，往得疑疾，有孚发若，吉。',
    '丰其沛，日中见沬，折其右肱，无咎。',
    '丰其蔀，日中见斗，遇其夷主，吉。',
    '来章，有庆誉，吉。',
    '丰其屋，蔀其家，窥其户，阒其无人，三岁不觌，凶。',
  ],
  xiao_xiang: [
    '虽旬无咎，过旬灾也。',
    '有孚发若，信以发志也。',
    '丰其沛，不可大事也。',
    '遇其夷主，吉行也。',
    '六五之吉，有庆也。',
    '丰其屋，天际翔也。',
  ],
  gua_de: '上动下丽',
  interpretation: '象征丰大，如日中天，然盛极须防衰。',
};

const interp: HexagramInterpretation = {
  overall: feng.interpretation,
  career: '丰大如日中天，宜明断守成防盛极',
  love: '情浓意盛，防盛极生隙',
  health: '精力旺盛，防亢奋过耗',
  fortune: '财运丰隆，趁势而须守正',
  auspicious: '祭祀、祈福、嫁娶',
  inauspicious: '动土、破土',
};

describe('HexagramDisplay 卦序卦名与上下卦', () => {
  it('显示第N卦·卦名与上下卦卦名/五行/卦德（不再显示序号数字）', () => {
    render(<HexagramDisplay hexagram={feng} />);
    expect(screen.getByText('第55卦')).toBeInTheDocument();
    expect(screen.getByText('丰')).toBeInTheDocument();
    expect(screen.getByText(/☳ 震卦 · 木 · 动/)).toBeInTheDocument();
    expect(screen.getByText(/☲ 离卦 · 火 · 丽/)).toBeInTheDocument();
    expect(screen.getByText('卦德：上动下丽')).toBeInTheDocument();
  });

  it('显示彖传（tuan_ci 有值时）', () => {
    const withTuan = { ...feng, tuan_ci: '丰，大也。明以动，故丰。' };
    render(<HexagramDisplay hexagram={withTuan} />);
    expect(screen.getByText('彖传')).toBeInTheDocument();
    expect(screen.getByText('丰，大也。明以动，故丰。')).toBeInTheDocument();
  });
});

describe('HexagramDisplay 四象限差异化解读', () => {
  it('渲染事业/感情/健康/财运四卡与宜忌', () => {
    render(<HexagramDisplay hexagram={feng} interp={interp} />);
    expect(screen.getByText('事业')).toBeInTheDocument();
    expect(screen.getByText('感情')).toBeInTheDocument();
    expect(screen.getByText('健康')).toBeInTheDocument();
    expect(screen.getByText('财运')).toBeInTheDocument();
    expect(screen.getByText(interp.career)).toBeInTheDocument();
    // 宜/忌与正文分属不同元素，按元素分别断言
    expect(screen.getByText('宜')).toBeInTheDocument();
    expect(screen.getByText(/祭祀、祈福、嫁娶/)).toBeInTheDocument();
    expect(screen.getByText('忌')).toBeInTheDocument();
    expect(screen.getByText(/动土、破土/)).toBeInTheDocument();
  });

  it('无 interp 时不渲染四象限', () => {
    render(<HexagramDisplay hexagram={feng} />);
    expect(screen.queryByText('事业')).not.toBeInTheDocument();
  });
});

describe('HexagramDisplay 逐爻经文折叠', () => {
  it('默认折叠，展开后逐爻显示爻名/爻辞/小象', () => {
    render(<HexagramDisplay hexagram={feng} movingYao={1} />);
    expect(screen.queryByText('遇其配主，虽旬无咎，往有尚。')).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /展开逐爻经文/ }));
    expect(screen.getByText('遇其配主，虽旬无咎，往有尚。')).toBeInTheDocument();
    // 丰初九为阳爻：爻名「初九」，动爻标注
    expect(screen.getByText(/初九（动）/)).toBeInTheDocument();
    // 小象传
    expect(screen.getByText('《象》曰：虽旬无咎，过旬灾也。')).toBeInTheDocument();
    // 六爻共 6 条小象
    expect(screen.getAllByText(/《象》曰：/)).toHaveLength(6);

    fireEvent.click(screen.getByRole('button', { name: /收起逐爻经文/ }));
    expect(screen.queryByText('遇其配主，虽旬无咎，往有尚。')).not.toBeInTheDocument();
  });
});
