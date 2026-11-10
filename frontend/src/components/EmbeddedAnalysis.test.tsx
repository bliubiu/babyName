import { describe, it, expect } from 'vitest';
import '@testing-library/jest-dom/vitest';
import { render, screen } from '@testing-library/react';
import BaziAnalysis from './BaziAnalysis';
import HexagramDisplay from './HexagramDisplay';
import HexagramDetailDisplay from './HexagramDetailDisplay';
import ZiweiDisplay from './ZiweiDisplay';
import type {
  BaziAnalysis as BaziAnalysisType,
  Hexagram,
  HexagramDetail,
  ZiweiChart,
} from '@/types';

const bazi: BaziAnalysisType = {
  bazi: {
    year: '癸卯',
    month: '乙丑',
    day: '戊子',
    hour: '戊午',
    year_ganzhi: '癸卯',
    month_ganzhi: '乙丑',
    day_ganzhi: '戊子',
    hour_ganzhi: '戊午',
  },
  wuxing: { jin: 2, mu: 2, shui: 2, huo: 2, tu: 2 },
  xiyongshen: ['水', '金'],
  rishou: '戊',
  rishou_wuxing: '土',
  nayin: '海中金',
  day_master: '戊',
};

const qian: Hexagram = {
  id: 1,
  name: '乾',
  number: 1,
  symbol: '☰☰',
  upper_trigram: 0,
  lower_trigram: 0,
  gua_ci: '元亨利贞',
  xiang_ci: '天行健，君子以自强不息',
  yao_ci: [],
  interpretation: '',
};

const detail: HexagramDetail = {
  original_hexagram: qian,
  changed_hexagram: qian,
  inter_hexagram: qian,
  upper_trigram_id: 0,
  lower_trigram_id: 0,
  upper_trigram_name: '乾',
  lower_trigram_name: '乾',
  moving_yao: 1,
  original_yao_lines: [7, 7, 7, 7, 7, 7],
  changed_yao_lines: [7, 7, 7, 7, 7, 7],
  inter_upper: 0,
  inter_lower: 0,
  interpretation: '测试综合解读',
};

const ziwei: ZiweiChart = {
  year: 2024,
  month: 1,
  day: 15,
  hour: 12,
  gender: 'male',
  nian_zhu: '甲辰',
  yue_zhu: '丙寅',
  ri_zhu: '戊子',
  shi_zhu: '戊午',
  ming_gong: '甲子',
  shen_gong: '丙寅',
  five_elements: '火六局',
  soul: '廉贞',
  body: '文昌',
  da_xian: {},
  zhu_xing: { 命宫: ['紫微'] },
  fu_xing: {},
  sui_xing: {},
  si_hua: {},
  brightness: {},
  analysis: '测试排盘分析',
};

/**
 * 结果页把四块分析收进统一折叠容器后，子组件必须以 embedded 模式输出内容：
 * 不能重复渲染卡片标题，也不能留一套各自为政的折叠按钮。
 */
describe('结果页分析组件 embedded 模式', () => {
  it('BaziAnalysis：默认有自带收起按钮，embedded 去掉按钮但内容完整', () => {
    const { rerender } = render(<BaziAnalysis bazi={bazi} nayin="海中金" zodiac="兔" />);
    expect(screen.getByRole('button', { name: '收起' })).toBeInTheDocument();

    rerender(<BaziAnalysis bazi={bazi} nayin="海中金" zodiac="兔" embedded />);
    expect(screen.queryByRole('button', { name: '收起' })).not.toBeInTheDocument();
    expect(screen.queryByText('八字分析')).not.toBeInTheDocument();
    expect(screen.getByText('五行分布')).toBeInTheDocument();
    expect(screen.getByText('水、金')).toBeInTheDocument();
  });

  it('HexagramDisplay：embedded 去掉「易经卦象」标题，卦辞仍在', () => {
    const { container } = render(<HexagramDisplay hexagram={qian} embedded />);
    expect(screen.queryByText('易经卦象')).not.toBeInTheDocument();
    expect(screen.getByText('元亨利贞')).toBeInTheDocument();
    expect(container.querySelector('.card')).toBeNull();
  });

  it('ZiweiDisplay：embedded 去掉「紫微斗数」标题，十二宫仍在', () => {
    const { container } = render(<ZiweiDisplay ziwei={ziwei} embedded />);
    expect(screen.queryByText('紫微斗数')).not.toBeInTheDocument();
    expect(screen.getByText('十二宫落星')).toBeInTheDocument();
    expect(screen.getByText('火六局')).toBeInTheDocument();
    expect(container.querySelector('.card')).toBeNull();
  });

  it('HexagramDetailDisplay：embeddedTitle 可覆盖默认小标题', () => {
    const { rerender } = render(<HexagramDetailDisplay detail={detail} embedded />);
    expect(screen.getByText('梅花卦象')).toBeInTheDocument();

    rerender(<HexagramDetailDisplay detail={detail} embedded embeddedTitle="卦象详解" />);
    expect(screen.getByText('卦象详解')).toBeInTheDocument();
    expect(screen.queryByText('梅花卦象')).not.toBeInTheDocument();
    expect(screen.getByText('测试综合解读')).toBeInTheDocument();
  });
});
