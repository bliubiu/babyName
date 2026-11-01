import { describe, it, expect } from 'vitest';
import '@testing-library/jest-dom/vitest';
import { render, screen } from '@testing-library/react';
import HexagramDetailDisplay from './HexagramDetailDisplay';
import type { Hexagram, HexagramDetail, TiYong } from '@/types';

/** docs/42 P0 金标准盘：姓4名3 → 丰#55，动爻1，互卦大过#28，变卦小过#62 */
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
  interpretation: '象征丰大，如日中天，然盛极须防衰。',
};

const daguo: Hexagram = {
  id: 28,
  name: '大过',
  number: 28,
  symbol: '☱☴',
  upper_trigram: 2,
  lower_trigram: 5,
  gua_ci: '大过：栋桡，利有攸往，亨。',
  xiang_ci: '泽灭木，大过；君子以独立不惧，遁世无闷。',
  yao_ci: [],
  interpretation: '',
};

const xiaoguo: Hexagram = {
  id: 62,
  name: '小过',
  number: 62,
  symbol: '☳☶',
  upper_trigram: 4,
  lower_trigram: 7,
  gua_ci: '小过：亨，利贞，可小事，不可大事。',
  xiang_ci: '山上有雷，小过；君子以行过乎恭，丧过乎哀，用过乎俭。',
  yao_ci: [],
  interpretation: '',
};

const detail: HexagramDetail = {
  original_hexagram: feng,
  changed_hexagram: xiaoguo,
  inter_hexagram: daguo,
  upper_trigram_id: 4,
  lower_trigram_id: 3,
  upper_trigram_name: '震',
  lower_trigram_name: '离',
  moving_yao: 1,
  original_yao_lines: [7, 8, 7, 7, 8, 8],
  changed_yao_lines: [8, 8, 7, 7, 8, 8],
  inter_upper: 2,
  inter_lower: 5,
  interpretation: '体卦离火得震木相生，事可成，宜速不宜缓。',
};

const tiyong: TiYong = {
  ti_gua: '离',
  yong_gua: '震',
  ti_wuxing: '火',
  yong_wuxing: '木',
  relation: '用生体',
  verdict: '吉',
  wang_shuai: '相',
  xiyong_note: '体卦火为八字喜用神，卦象与命局同气。',
  text: '用生体，主外力相助，所谋顺遂。',
};

describe('HexagramDetailDisplay 本/互/变三卦并排', () => {
  it('渲染三卦卦名与动爻标注', () => {
    render(<HexagramDetailDisplay detail={detail} />);
    expect(screen.getByText('本卦')).toBeInTheDocument();
    expect(screen.getByText('互卦')).toBeInTheDocument();
    expect(screen.getByText('变卦')).toBeInTheDocument();
    expect(screen.getByText('初九动')).toBeInTheDocument();
    expect(screen.getByText(/第1爻动/)).toBeInTheDocument();
  });

  it('三卦爻画互不相同（本丰/互大过/变小过）', () => {
    const { container } = render(<HexagramDetailDisplay detail={detail} />);
    const svgs = container.querySelectorAll('[data-testid="hexagram-lines"]');
    expect(svgs).toHaveLength(3);
    // 丰[7,8,7,7,8,8]→3阳3阴=9矩形；大过[8,7,7,7,7,8]→4阳2阴=8矩形；小过[8,8,7,7,8,8]→2阳4阴=10矩形
    expect(svgs[0].querySelectorAll('rect')).toHaveLength(9);
    expect(svgs[1].querySelectorAll('rect')).toHaveLength(8);
    expect(svgs[2].querySelectorAll('rect')).toHaveLength(10);
  });

  it('动爻爻辞取自本卦对应爻', () => {
    render(<HexagramDetailDisplay detail={detail} />);
    expect(screen.getByText('动爻 初九 爻辞')).toBeInTheDocument();
    expect(
      screen.getByText('遇其配主，虽旬无咎，往有尚。')
    ).toBeInTheDocument();
  });

  it('渲染体用断卦结构化信息与吉凶徽章', () => {
    render(<HexagramDetailDisplay detail={detail} tiyong={tiyong} />);
    expect(screen.getByText('体用断卦')).toBeInTheDocument();
    expect(screen.getByText('吉')).toBeInTheDocument();
    expect(
      screen.getByText(/体卦「离」（火），用卦「震」（木），用生体，体卦旺衰：相/)
    ).toBeInTheDocument();
    expect(screen.getByText(tiyong.xiyong_note)).toBeInTheDocument();
    expect(screen.getByText(tiyong.text)).toBeInTheDocument();
  });

  it('无 tiyong 时不渲染体用块', () => {
    render(<HexagramDetailDisplay detail={detail} />);
    expect(screen.queryByText('体用断卦')).not.toBeInTheDocument();
  });

  it('渲染综合解读', () => {
    render(<HexagramDetailDisplay detail={detail} />);
    expect(screen.getByText('综合解读')).toBeInTheDocument();
    expect(
      screen.getByText('体卦离火得震木相生，事可成，宜速不宜缓。')
    ).toBeInTheDocument();
  });

  it('变卦/互卦缺失时显示占位符', () => {
    const sparse: HexagramDetail = {
      ...detail,
      inter_hexagram: null,
      changed_hexagram: null,
    };
    render(<HexagramDetailDisplay detail={sparse} />);
    expect(screen.getAllByText('—')).toHaveLength(2);
  });
});
